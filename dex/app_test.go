package dex

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"math/rand"
	"reflect"
	"sync"
	"testing"
	"time"

	coreCommon "github.com/dexon-foundation/dexon-consensus/common"
	coreCrypto "github.com/dexon-foundation/dexon-consensus/core/crypto"
	coreEcdsa "github.com/dexon-foundation/dexon-consensus/core/crypto/ecdsa"
	coreTypes "github.com/dexon-foundation/dexon-consensus/core/types"
	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/common/math"
	"github.com/dexon-foundation/dexon/consensus/dexcon"
	"github.com/dexon-foundation/dexon/core"
	"github.com/dexon-foundation/dexon/core/rawdb"
	"github.com/dexon-foundation/dexon/core/types"
	"github.com/dexon-foundation/dexon/core/vm"
	"github.com/dexon-foundation/dexon/crypto"
	"github.com/dexon-foundation/dexon/ethdb"
	"github.com/dexon-foundation/dexon/event"
	"github.com/dexon-foundation/dexon/rlp"
)

type singnal int

const (
	runFail singnal = iota
	runSuccess
)

type App interface {
	PreparePayload(position coreTypes.Position) (payload []byte, err error)
	PrepareWitness(height uint64) (witness coreTypes.Witness, err error)
	VerifyBlock(block *coreTypes.Block) coreTypes.BlockVerifyStatus
	BlockConfirmed(block coreTypes.Block)
	BlockDelivered(blockHash coreCommon.Hash, position coreTypes.Position, rand []byte)
	SubscribeNewFinalizedBlockEvent(ch chan<- core.NewFinalizedBlockEvent) event.Subscription
	Stop()
}

type Product interface{}

type Tester interface {
	// Name the name of tester
	Name() string

	// ViewAndRecord view the data and record then start when requirement is ready.
	ViewAndRecord(product Product)

	// ReadyToTest check tester is ready or not.
	ReadyToTest() bool

	// InputsForTest return the inputs which we want to test, it will be called only when it is get ready.
	InputsForTest(product Product) []reflect.Value

	// ValidateResults validate the results what we expected.
	ValidateResults(results []reflect.Value) error

	// Done return true when tester finish it job.
	Done() bool

	// StopTime lock all working jobs for test and rollback data if necessary.
	StopTime() bool

	// Rollback rollback data in the final.
	Rollback() error
}

type baseTester struct {
	App

	ready bool

	testTimer    *time.Timer
	testInterval time.Duration

	counter   int
	threshold int

	self interface{}
}

func (t baseTester) Name() string {
	return reflect.TypeOf(t.self).Name()
}

func (t baseTester) ReadyToTest() bool {
	return t.ready
}

func (t baseTester) Done() bool {
	return t.counter >= t.threshold
}

func (t baseTester) StopTime() bool {
	return false
}

func (t *baseTester) Rollback() error {
	return nil
}

func (t *baseTester) ViewAndRecord(product Product) {
	panic("need to implement")
}

func (t baseTester) InputsForTest(product Product) []reflect.Value {
	panic("need to implement")
}

func (t *baseTester) ValidateResults(results []reflect.Value) error {
	panic("need to implement")
}

type takerName string

type makerName string

type ProductCenter struct {
	takerChan map[takerName]chan Product

	takerList map[makerName]map[takerName]struct{}
}

// RequestProduct make a blocking request the product from maker.
func (center *ProductCenter) RequestProduct(tName takerName) Product {
	p := <-center.takerChan[tName]
	return p
}

// DeliverProduct deliver product for takers.
func (center *ProductCenter) DeliverProduct(mName makerName, product Product) {
	for tName := range center.takerList[mName] {
		center.takerChan[tName] <- product
	}
}

// Register build the connection between taker and maker.
func (center *ProductCenter) Register(tName takerName, mName ...makerName) {
	center.takerChan[tName] = make(chan Product, 1000)

	for _, n := range mName {
		if _, exist := center.takerList[n]; !exist {
			center.takerList[n] = make(map[takerName]struct{})
			center.takerList[n][tName] = struct{}{}
		} else {
			center.takerList[n][tName] = struct{}{}
		}
	}
}

func (center ProductCenter) New() *ProductCenter {
	center.takerChan = map[takerName]chan Product{}
	center.takerList = map[makerName]map[takerName]struct{}{}
	return &center
}

type FactoryBase struct {
	App

	targetFunc interface{}

	name string

	center *ProductCenter

	testers []Tester

	status chan map[singnal]interface{}

	stopTimeMu *sync.RWMutex
}

func (base *FactoryBase) testerDoWork(product Product) error {
	for _, t := range base.testers {
		if t.Done() {
			continue
		}

		if err := func() (tErr error) {
			var returns []reflect.Value
			defer func() {
				r := recover()
				if r != nil {
					returns = append(returns, reflect.ValueOf(fmt.Errorf("%v", r)))
				}

				if t.ReadyToTest() {
					err := t.ValidateResults(returns)
					if err != nil {
						tErr = err
						return
					}

					err = t.Rollback()
					if err != nil {
						tErr = fmt.Errorf("recover fail: %v", tErr)
						return
					}
				} else if r != nil {
					tErr = fmt.Errorf("%v", r)
				}

				if t.StopTime() {
					base.stopTimeMu.Unlock()
				} else {
					base.stopTimeMu.RUnlock()
				}
			}()

			if t.StopTime() {
				base.stopTimeMu.Lock()
			} else {
				base.stopTimeMu.RLock()
			}
			t.ViewAndRecord(product)
			if t.ReadyToTest() {
				inputs := t.InputsForTest(product)
				returns = reflect.ValueOf(base.targetFunc).Call(inputs)
			}
			return
		}(); err != nil {
			return fmt.Errorf("%s: %v", t.Name(), err)
		}
	}

	return nil
}

func (base *FactoryBase) testerAllDone() bool {
	for _, t := range base.testers {
		if !t.Done() {
			return false
		}
	}

	return true
}

func (base *FactoryBase) notifySuccess() {
	base.status <- map[singnal]interface{}{runSuccess: nil}
}

func (base *FactoryBase) notifyFail(msg interface{}) {
	base.status <- map[singnal]interface{}{runFail: msg}
}

type ConfigFactory struct {
	FactoryBase

	initialized bool

	sleepTime time.Duration

	masterKey *ecdsa.PrivateKey
}

func (f *ConfigFactory) Run() {
	for {
		if !f.initialized {
			// Initial block for first round.
			go f.center.DeliverProduct(makerName(f.name),
				&PositionProduct{position: coreTypes.Position{
					Round:  0,
					Height: coreTypes.GenesisHeight,
				}})

			f.initialized = true
			continue
		}

		time.Sleep(f.sleepTime)

		product := f.center.RequestProduct(takerName(f.name))
		position := f.covertProduct(product)
		position.Height++

		if f.roundStartAt(position.Round+1) == position.Height {
			position.Round = position.Round + 1
		}

		go f.center.DeliverProduct(makerName(f.name), &PositionProduct{
			position: position,
		})
	}
}

func (f ConfigFactory) covertProduct(product interface{}) coreTypes.Position {
	var position coreTypes.Position
	switch product.(type) {
	case *BlockConfirmedProduct:
		position = product.(*BlockConfirmedProduct).block.Position
	default:
		panic(fmt.Errorf("unexpected type %T", product))
	}

	return position
}

func (f *ConfigFactory) roundStartAt(round uint64) uint64 {
	dexonApp := f.App.(*DexconApp)
	start := uint64(0)
	for i := uint64(0); i < round; i++ {
		start += dexonApp.gov.Configuration(i).RoundLength
	}

	return start - 1
}

func (f ConfigFactory) New(app App, center *ProductCenter, stopTimeMu *sync.RWMutex, masterKey *ecdsa.PrivateKey) *ConfigFactory {
	f.FactoryBase = FactoryBase{
		App:        app,
		name:       reflect.TypeOf(f).Name(),
		center:     center,
		stopTimeMu: stopTimeMu,
	}
	f.sleepTime = 250 * time.Millisecond
	f.masterKey = masterKey
	f.center.Register(takerName(f.name), makerName(reflect.TypeOf(BlockConfirmedFactory{}).Name()))
	return &f
}

type PositionProduct struct {
	position coreTypes.Position
}

type PreparePayloadFactory struct {
	FactoryBase
}

func (f *PreparePayloadFactory) Run() {
	defer func() {
		if r := recover(); r != nil {
			f.notifyFail(r)
		}
	}()

	for {
		product := f.center.RequestProduct(takerName(f.name))

		if len(f.testers) > 0 && f.testerAllDone() {
			f.notifySuccess()
			f.testers = nil
		} else if err := f.testerDoWork(product); err != nil {
			panic(fmt.Errorf("test fail: %v", err))
		}

		go func() {
			defer func() {
				if r := recover(); r != nil {
					f.notifyFail(r)
				}
			}()

			position := f.covertProduct(product)
			f.stopTimeMu.RLock()
			payload, err := f.App.PreparePayload(position)
			if err != nil {
				panic(err)
			}
			f.stopTimeMu.RUnlock()

			go f.center.DeliverProduct(makerName(f.name), &PreparePayloadProduct{
				position: position,
				payload:  payload,
			})
		}()
	}
}

func (f PreparePayloadFactory) covertProduct(product interface{}) coreTypes.Position {
	var position coreTypes.Position
	switch product.(type) {
	case *PositionProduct:
		position = product.(*PositionProduct).position
	default:
		panic(fmt.Errorf("unexpected type %T", product))
	}

	return position
}

func (f PreparePayloadFactory) New(app App, center *ProductCenter, stopTimeMu *sync.RWMutex) *PreparePayloadFactory {
	f.FactoryBase = FactoryBase{
		App:        app,
		name:       reflect.TypeOf(f).Name(),
		center:     center,
		targetFunc: app.PreparePayload,
		status:     make(chan map[singnal]interface{}, 1),
		stopTimeMu: stopTimeMu,
	}
	f.center.Register(takerName(f.name), makerName(reflect.TypeOf(ConfigFactory{}).Name()))
	return &f
}

func (f PreparePayloadFactory) NewWithTester(app App, center *ProductCenter, stopTimeMu *sync.RWMutex) *PreparePayloadFactory {
	factory := f.New(app, center, stopTimeMu)
	factory.testers = []Tester{
		ppBlockLimitTester{}.New(app, 10, 3, 3),
		ppBlockHeightTester{}.New(app, 20, 3, 3),
	}
	return factory
}

type PreparePayloadProduct struct {
	position coreTypes.Position
	payload  []byte
}

type ppBlockLimitTester struct {
	baseTester

	round uint64
}

func (t ppBlockLimitTester) New(app App, startAt, interval, threshold int) *ppBlockLimitTester {
	t.baseTester = baseTester{
		App:          app,
		testTimer:    time.NewTimer(time.Duration(startAt) * time.Second),
		testInterval: time.Duration(interval) * time.Second,
		threshold:    threshold,
		self:         t,
	}
	return &t
}

func (t *ppBlockLimitTester) ViewAndRecord(product Product) {
	select {
	case <-t.testTimer.C:
		switch product.(type) {
		case *PositionProduct:
			t.round = product.(*PositionProduct).position.Round
			t.ready = true
		}

		t.testTimer.Reset(t.testInterval)
	default:
	}
}

func (t ppBlockLimitTester) InputsForTest(product Product) []reflect.Value {
	return []reflect.Value{reflect.ValueOf(product.(*PositionProduct).position)}
}

func (t *ppBlockLimitTester) ValidateResults(results []reflect.Value) error {
	if len(results) > 2 {
		return fmt.Errorf("unexpected return values: %v", results)
	}

	switch results[1].Interface().(type) {
	case nil:
	case error:
		return fmt.Errorf("result[1] must nil: %v", results[1].Interface())
	default:
		return fmt.Errorf("unexpect results[1] return type %T", results[1].Interface())
	}

	switch results[0].Interface().(type) {
	case []byte:
		if results[0].Bytes() != nil {
			var txs []*types.Transaction
			err := rlp.DecodeBytes(results[0].Bytes(), &txs)
			if err != nil {
				return fmt.Errorf("rlp decode error: %v", err)
			}

			app := t.App.(*DexconApp)
			blockLimit := app.gov.DexconConfiguration(t.round).BlockGasLimit
			totalGas := uint64(0)
			for _, tx := range txs {
				totalGas += tx.Gas()
			}

			if blockLimit < totalGas {
				return fmt.Errorf("total cost larger than block limit %d < %d", blockLimit, totalGas)
			}

			t.counter++
		}
	default:
		return fmt.Errorf("unexpect results[0] return type %T", results[0].Interface())
	}

	t.ready = false
	return nil
}

type ppBlockHeightTester struct {
	baseTester

	height uint64
}

func (t ppBlockHeightTester) New(app App, startAt, interval, threshold int) *ppBlockHeightTester {
	t.baseTester = baseTester{
		App:          app,
		testTimer:    time.NewTimer(time.Duration(startAt) * time.Second),
		testInterval: time.Duration(interval) * time.Second,
		threshold:    threshold,
		self:         t,
	}
	return &t
}

func (t *ppBlockHeightTester) ViewAndRecord(product Product) {
	select {
	case <-t.testTimer.C:
		switch product.(type) {
		case *PositionProduct:
			t.height = product.(*PositionProduct).position.Height
			t.ready = true
		}
		t.testTimer.Reset(t.testInterval)
	default:
	}
}

func (t ppBlockHeightTester) InputsForTest(product Product) []reflect.Value {
	position := product.(*PositionProduct).position
	position.Height--
	return []reflect.Value{reflect.ValueOf(position)}
}

func (t *ppBlockHeightTester) ValidateResults(results []reflect.Value) error {
	if len(results) > 2 {
		return fmt.Errorf("unexpected return values: %v", results)
	}

	switch results[1].Interface().(type) {
	case error:
		expectErr := fmt.Sprintf("expected height %d but get %d", t.height, t.height-1)
		if results[1].Interface().(error).Error() != expectErr {
			return fmt.Errorf("unexpected error msg: %v", results[1].Interface())
		}
	default:
		return fmt.Errorf("unexpect results[1] return type %T", results[1].Interface())
	}

	switch results[0].Interface().(type) {
	case []byte:
		if results[0].Bytes() != nil {
			return fmt.Errorf("payload should be nil")
		}
	default:
		return fmt.Errorf("unexpect results[0] return type %T", results[0].Interface())
	}

	t.ready = false
	t.counter++
	return nil
}

type PrepareWitnessFactory struct {
	FactoryBase
}

func (f *PrepareWitnessFactory) Run() {
	defer func() {
		if r := recover(); r != nil {
			f.notifyFail(r)
		}
	}()

	for {
		product := f.center.RequestProduct(takerName(f.name))

		if len(f.testers) > 0 && f.testerAllDone() {
			f.notifySuccess()
			f.testers = nil
		} else if err := f.testerDoWork(product); err != nil {
			panic(fmt.Errorf("test fail: %v", err))
		}

		go func() {
			defer func() {
				if r := recover(); r != nil {
					f.notifyFail(r)
				}
			}()

			f.stopTimeMu.RLock()
			witness, err := f.App.PrepareWitness(f.App.(*DexconApp).blockchain.CurrentBlock().NumberU64())
			if err != nil {
				panic(err)
			}
			f.stopTimeMu.RUnlock()

			position, payload := f.convertProduct(product)
			go f.center.DeliverProduct(makerName(f.name), &PrepareWitnessProduct{
				block: coreTypes.Block{
					Hash:        coreCommon.NewRandomHash(),
					ProposerID:  coreTypes.NodeID{coreCommon.Hash{1, 2, 3}},
					Position:    position,
					Witness:     witness,
					Payload:     payload,
					PayloadHash: coreCrypto.Keccak256Hash(payload),
				},
			})
		}()
	}
}

func (f PrepareWitnessFactory) convertProduct(product Product) (coreTypes.Position, []byte) {
	var (
		position coreTypes.Position
		payload  []byte
	)
	switch product.(type) {
	case *PreparePayloadProduct:
		realProduct := product.(*PreparePayloadProduct)
		position = realProduct.position
		payload = realProduct.payload
	default:
		panic(fmt.Errorf("unexpected type %T", product))
	}

	return position, payload
}

func (f PrepareWitnessFactory) New(app App, center *ProductCenter, stopTimeMu *sync.RWMutex) *PrepareWitnessFactory {
	f.FactoryBase = FactoryBase{
		App:        app,
		name:       reflect.TypeOf(f).Name(),
		center:     center,
		targetFunc: app.PrepareWitness,
		status:     make(chan map[singnal]interface{}, 1),
		stopTimeMu: stopTimeMu,
	}
	f.center.Register(takerName(f.name), makerName(reflect.TypeOf(PreparePayloadFactory{}).Name()))
	return &f
}

func (f PrepareWitnessFactory) NewWithTester(app App, center *ProductCenter, stopTimeMu *sync.RWMutex) *PrepareWitnessFactory {
	factory := f.New(app, center, stopTimeMu)
	factory.testers = []Tester{
		pwConsensusHeightTester{}.New(app, 10, 10, 3),
	}

	return factory
}

type PrepareWitnessProduct struct {
	block coreTypes.Block
}

type pwConsensusHeightTester struct {
	baseTester
}

func (t pwConsensusHeightTester) New(app App, startAt, interval, threshold int) *pwConsensusHeightTester {
	t.baseTester = baseTester{
		App:          app,
		testTimer:    time.NewTimer(time.Duration(startAt) * time.Second),
		testInterval: time.Duration(interval) * time.Second,
		threshold:    threshold,
		self:         t,
	}
	return &t
}

func (t *pwConsensusHeightTester) ViewAndRecord(product Product) {
	select {
	case <-t.testTimer.C:
		t.ready = true
		t.testTimer.Reset(t.testInterval)
	default:
	}
}

func (t pwConsensusHeightTester) InputsForTest(product Product) []reflect.Value {
	return []reflect.Value{reflect.ValueOf(uint64(99999))}
}

func (t *pwConsensusHeightTester) ValidateResults(results []reflect.Value) error {
	if len(results) > 2 {
		return fmt.Errorf("unexpected return values: %v", results)
	}

	switch results[1].Interface().(type) {
	case nil:
		return fmt.Errorf("results[1] must not nil")
	case error:
		if results[1].Interface().(error).Error() != "current height < consensus height" {
			return fmt.Errorf("unexpected error: %v", results[1].Interface())
		}
	default:
		return fmt.Errorf("unexpect results[1] return type %T", results[1].Interface())
	}

	switch results[0].Interface().(type) {
	case coreTypes.Witness:
		witness := results[0].Interface().(coreTypes.Witness)
		if witness.Height != 0 || len(witness.Data) > 0 {
			return fmt.Errorf("unexpected results[1] return %+v", results[0].Interface())
		}
	default:
		return fmt.Errorf("unexpect results[0] return type %T", results[0].Interface())
	}

	t.counter++
	t.ready = false
	return nil
}

type VerifyBlockFactory struct {
	FactoryBase
}

func (f *VerifyBlockFactory) Run() {
	defer func() {
		if r := recover(); r != nil {
			f.notifyFail(r)
		}
	}()

	for {
		product := f.center.RequestProduct(takerName(f.name))

		if len(f.testers) > 0 && f.testerAllDone() {
			f.notifySuccess()
			f.testers = nil
		} else if err := f.testerDoWork(product); err != nil {
			panic(fmt.Errorf("test fail: %v", err))
		}

		go func() {
			defer func() {
				if r := recover(); r != nil {
					f.notifyFail(r)
				}
			}()

			block := f.convertProduct(product)

			f.stopTimeMu.RLock()
			if status := f.App.VerifyBlock(&block); status != coreTypes.VerifyOK {
				panic(fmt.Errorf("verify block fail: status %v", status))
			}
			f.stopTimeMu.RUnlock()

			go f.center.DeliverProduct(makerName(f.name), &VerifyBlockProduct{
				block: block,
			})
		}()
	}
}

func (f VerifyBlockFactory) convertProduct(product Product) coreTypes.Block {
	var block coreTypes.Block
	switch product.(type) {
	case *PrepareWitnessProduct:
		block = product.(*PrepareWitnessProduct).block
	default:
		panic(fmt.Errorf("unexpected type %T", product))
	}

	return block
}

func (f VerifyBlockFactory) New(app App, center *ProductCenter, stopTimeMu *sync.RWMutex) *VerifyBlockFactory {
	f.FactoryBase = FactoryBase{
		App:        app,
		name:       reflect.TypeOf(f).Name(),
		center:     center,
		targetFunc: app.VerifyBlock,
		status:     make(chan map[singnal]interface{}, 1),
		stopTimeMu: stopTimeMu,
	}
	f.center.Register(takerName(f.name), makerName(reflect.TypeOf(PrepareWitnessFactory{}).Name()))
	return &f
}

func (f VerifyBlockFactory) NewWithTester(app App, center *ProductCenter, masterKey *ecdsa.PrivateKey,
	stopTimeMu *sync.RWMutex) *VerifyBlockFactory {
	factory := f.New(app, center, stopTimeMu)
	factory.testers = []Tester{
		vbWitnessDataDecodeTester{}.New(app, 10, 5, 3),
		vbWitnessHeightTester{}.New(app, 20, 5, 3),
		vbWitnessDataTester{}.New(app, 30, 5, 3),
		vbBlockHeightTester{}.New(app, 40, 3, 3),
		vbPayloadDecodeTester{}.New(app, 50, 5, 3),
		vbTxNonceSequenceTester{}.New(app, masterKey, 60, 5, 3),
		vbTxNonceIncrementTester{}.New(app, masterKey, 70, 5, 3),
		vbTxIntrinsicGasTester{}.New(app, masterKey, 80, 5, 3),
		vbTxGasTooLowTester{}.New(app, masterKey, 90, 5, 3),
		vbTxInvalidGasPriceTester{}.New(app, masterKey, 100, 5, 3),
		vbInsufficientFundsTester{}.New(app, 110, 5, 3),
		vbBlockLimitTester{}.New(app, 120, 5, 3),
	}

	return factory
}

type VerifyBlockProduct struct {
	block coreTypes.Block
}

type vbWitnessDataDecodeTester struct {
	baseTester
}

func (t vbWitnessDataDecodeTester) New(app App, startAt, interval, threshold int) *vbWitnessDataDecodeTester {
	t.baseTester = baseTester{
		App:          app,
		testTimer:    time.NewTimer(time.Duration(startAt) * time.Second),
		testInterval: time.Duration(interval) * time.Second,
		threshold:    threshold,
		self:         t,
	}
	return &t
}

func (t *vbWitnessDataDecodeTester) ViewAndRecord(product Product) {
	select {
	case <-t.testTimer.C:
		switch product.(type) {
		case *PrepareWitnessProduct:
			t.ready = true
		}
		t.testTimer.Reset(t.testInterval)
	default:
	}
}

func (t vbWitnessDataDecodeTester) InputsForTest(product Product) []reflect.Value {
	block := product.(*PrepareWitnessProduct).block
	block.Witness.Data = make([]byte, 100)
	return []reflect.Value{reflect.ValueOf(&block)}
}

func (t *vbWitnessDataDecodeTester) ValidateResults(results []reflect.Value) error {
	if len(results) > 1 {
		return fmt.Errorf("unexpected return values: %v", results)
	}

	switch results[0].Interface().(type) {
	case coreTypes.BlockVerifyStatus:
		if results[0].Interface().(coreTypes.BlockVerifyStatus) != coreTypes.VerifyInvalidBlock {
			return fmt.Errorf("unexpected status %v", results[0].Interface())
		}
	default:
		return fmt.Errorf("unexpect results[0] return type %T", results[0].Interface())
	}

	t.counter++
	t.ready = false
	return nil
}

type vbWitnessHeightTester struct {
	baseTester
}

func (t vbWitnessHeightTester) New(app App, startAt, interval, threshold int) *vbWitnessHeightTester {
	t.baseTester = baseTester{
		App:          app,
		testTimer:    time.NewTimer(time.Duration(startAt) * time.Second),
		testInterval: time.Duration(interval) * time.Second,
		threshold:    threshold,
		self:         t,
	}
	return &t
}

func (t *vbWitnessHeightTester) ViewAndRecord(product Product) {
	select {
	case <-t.testTimer.C:
		switch product.(type) {
		case *PrepareWitnessProduct:
			t.ready = true
		}
		t.testTimer.Reset(t.testInterval)
	default:
	}
}

func (t vbWitnessHeightTester) InputsForTest(product Product) []reflect.Value {
	block := product.(*PrepareWitnessProduct).block
	block.Witness.Height += uint64(rand.New(rand.NewSource(time.Now().UnixNano())).Intn(10) + 1)
	return []reflect.Value{reflect.ValueOf(&block)}
}

func (t *vbWitnessHeightTester) ValidateResults(results []reflect.Value) error {
	if len(results) > 1 {
		return fmt.Errorf("unexpected return values: %v", results)
	}

	switch results[0].Interface().(type) {
	case coreTypes.BlockVerifyStatus:
		if results[0].Interface().(coreTypes.BlockVerifyStatus) != coreTypes.VerifyRetryLater {
			return fmt.Errorf("unexpected status %v", results[0].Interface())
		}
	default:
		return fmt.Errorf("unexpect results[0] return type %T", results[0].Interface())
	}

	t.counter++
	t.ready = false
	return nil
}

type vbWitnessDataTester struct {
	baseTester
}

func (t vbWitnessDataTester) New(app App, startAt, interval, threshold int) *vbWitnessDataTester {
	t.baseTester = baseTester{
		App:          app,
		testTimer:    time.NewTimer(time.Duration(startAt) * time.Second),
		testInterval: time.Duration(interval) * time.Second,
		threshold:    threshold,
		self:         t,
	}
	return &t
}

func (t *vbWitnessDataTester) ViewAndRecord(product Product) {
	select {
	case <-t.testTimer.C:
		switch product.(type) {
		case *PrepareWitnessProduct:
			t.ready = true
		}
		t.testTimer.Reset(t.testInterval)
	default:
	}
}

func (t vbWitnessDataTester) InputsForTest(product Product) []reflect.Value {
	block := product.(*PrepareWitnessProduct).block
	randNum := big.NewInt(rand.New(rand.NewSource(time.Now().UnixNano())).Int63())
	var err error
	block.Witness.Data, err = rlp.EncodeToBytes(common.BigToHash(randNum))
	if err != nil {
		panic(err)
	}

	return []reflect.Value{reflect.ValueOf(&block)}
}

func (t *vbWitnessDataTester) ValidateResults(results []reflect.Value) error {
	if len(results) > 1 {
		return fmt.Errorf("unexpected return values: %v", results)
	}

	switch results[0].Interface().(type) {
	case coreTypes.BlockVerifyStatus:
		if results[0].Interface().(coreTypes.BlockVerifyStatus) != coreTypes.VerifyInvalidBlock {
			return fmt.Errorf("unexpected status %v", results[0].Interface())
		}
	default:
		return fmt.Errorf("unexpect results[0] return type %T", results[0].Interface())
	}

	t.counter++
	t.ready = false
	return nil
}

type vbBlockHeightTester struct {
	baseTester
}

func (t vbBlockHeightTester) New(app App, startAt, interval, threshold int) *vbBlockHeightTester {
	t.baseTester = baseTester{
		App:          app,
		testTimer:    time.NewTimer(time.Duration(startAt) * time.Second),
		testInterval: time.Duration(interval) * time.Second,
		threshold:    threshold,
		self:         t,
	}
	return &t
}

func (t *vbBlockHeightTester) ViewAndRecord(product Product) {
	select {
	case <-t.testTimer.C:
		switch product.(type) {
		case *PrepareWitnessProduct:
			t.ready = true
		}
		t.testTimer.Reset(t.testInterval)
	default:
	}
}

func (t vbBlockHeightTester) InputsForTest(product Product) []reflect.Value {
	block := product.(*PrepareWitnessProduct).block
	block.Position.Height--
	return []reflect.Value{reflect.ValueOf(&block)}
}

func (t *vbBlockHeightTester) ValidateResults(results []reflect.Value) error {
	if len(results) > 1 {
		return fmt.Errorf("unexpected return values: %v", results)
	}

	switch results[0].Interface().(type) {
	case coreTypes.BlockVerifyStatus:
		if results[0].Interface().(coreTypes.BlockVerifyStatus) != coreTypes.VerifyRetryLater {
			return fmt.Errorf("unexpected status %v", results[0].Interface())
		}
	default:
		return fmt.Errorf("unexpect results[0] return type %T", results[0].Interface())
	}

	t.counter++
	t.ready = false
	return nil
}

type vbPayloadDecodeTester struct {
	baseTester
}

func (t vbPayloadDecodeTester) New(app App, startAt, interval, threshold int) *vbPayloadDecodeTester {
	t.baseTester = baseTester{
		App:          app,
		testTimer:    time.NewTimer(time.Duration(startAt) * time.Second),
		testInterval: time.Duration(interval) * time.Second,
		threshold:    threshold,
		self:         t,
	}
	return &t
}

func (t *vbPayloadDecodeTester) ViewAndRecord(product Product) {
	select {
	case <-t.testTimer.C:
		switch product.(type) {
		case *PrepareWitnessProduct:
			t.ready = true
		}
		t.testTimer.Reset(t.testInterval)
	default:
	}
}

func (t vbPayloadDecodeTester) InputsForTest(product Product) []reflect.Value {
	block := product.(*PrepareWitnessProduct).block
	block.Payload = []byte{0x00}
	return []reflect.Value{reflect.ValueOf(&block)}
}

func (t *vbPayloadDecodeTester) ValidateResults(results []reflect.Value) error {
	if len(results) > 1 {
		return fmt.Errorf("unexpected return values: %v", results)
	}

	switch results[0].Interface().(type) {
	case coreTypes.BlockVerifyStatus:
		status := results[0].Interface().(coreTypes.BlockVerifyStatus)
		if status != coreTypes.VerifyInvalidBlock {
			return fmt.Errorf("unexpect status %v", status)
		}
	default:
		return fmt.Errorf("unexpect results[0] return type %T", results[0].Interface())
	}

	t.counter++
	t.ready = false
	return nil
}

type vbTxNonceSequenceTester struct {
	baseTester

	key *ecdsa.PrivateKey
}

func (t vbTxNonceSequenceTester) New(app App, key *ecdsa.PrivateKey, startAt, interval,
	threshold int) *vbTxNonceSequenceTester {
	t.baseTester = baseTester{
		App:          app,
		testTimer:    time.NewTimer(time.Duration(startAt) * time.Second),
		testInterval: time.Duration(interval) * time.Second,
		threshold:    threshold,
		self:         t,
	}
	t.key = key
	return &t
}

func (t *vbTxNonceSequenceTester) ViewAndRecord(product Product) {
	select {
	case <-t.testTimer.C:
		switch product.(type) {
		case *PrepareWitnessProduct:
			t.ready = true
		}
		t.testTimer.Reset(t.testInterval)
	default:
	}
}

func (t vbTxNonceSequenceTester) InputsForTest(product Product) []reflect.Value {
	app := t.App.(*DexconApp)
	block := product.(*PrepareWitnessProduct).block
	var err error

	blockchain := app.blockchain
	signer := types.NewEIP155Signer(blockchain.Config().ChainID)
	var txs []*types.Transaction
	for i := uint64(0); i < 3; i++ {
		if i == 1 {
			continue
		}

		tx, err := types.SignTx(
			types.NewTransaction(i, common.Address{}, nil, 21000, new(big.Int).SetInt64(1e9), nil), signer, t.key)
		if err != nil {
			panic(err)
		}
		txs = append(txs, tx)
	}

	block.Payload, err = rlp.EncodeToBytes(txs)
	if err != nil {
		panic(err)
	}

	return []reflect.Value{reflect.ValueOf(&block)}
}

func (t *vbTxNonceSequenceTester) ValidateResults(results []reflect.Value) error {
	if len(results) > 1 {
		return fmt.Errorf("unexpected return values: %v", results)
	}

	switch results[0].Interface().(type) {
	case coreTypes.BlockVerifyStatus:
		status := results[0].Interface().(coreTypes.BlockVerifyStatus)
		if status != coreTypes.VerifyInvalidBlock {
			return fmt.Errorf("unexpect status %v", status)
		}
	default:
		return fmt.Errorf("unexpect results[0] return type %T", results[0].Interface())
	}

	t.counter++
	t.ready = false
	return nil
}

type vbTxNonceIncrementTester struct {
	baseTester

	key *ecdsa.PrivateKey
}

func (t vbTxNonceIncrementTester) New(app App, key *ecdsa.PrivateKey, startAt, interval,
	threshold int) *vbTxNonceIncrementTester {
	t.baseTester = baseTester{
		App:          app,
		testTimer:    time.NewTimer(time.Duration(startAt) * time.Second),
		testInterval: time.Duration(interval) * time.Second,
		threshold:    threshold,
		self:         t,
	}
	t.key = key
	return &t
}

func (t *vbTxNonceIncrementTester) ViewAndRecord(product Product) {
	select {
	case <-t.testTimer.C:
		switch product.(type) {
		case *PrepareWitnessProduct:
			t.ready = true
		}
		t.testTimer.Reset(t.testInterval)
	default:
	}
}

func (t vbTxNonceIncrementTester) InputsForTest(product Product) []reflect.Value {
	app := t.App.(*DexconApp)
	block := product.(*PrepareWitnessProduct).block
	var err error

	blockchain := app.blockchain
	signer := types.NewEIP155Signer(blockchain.Config().ChainID)
	var txs []*types.Transaction
	for i := uint64(1); i < 4; i++ {
		tx, err := types.SignTx(
			types.NewTransaction(i, common.Address{}, nil, 21000, new(big.Int).SetInt64(1e9), nil), signer, t.key)
		if err != nil {
			panic(err)
		}
		txs = append(txs, tx)
	}

	block.Payload, err = rlp.EncodeToBytes(txs)
	if err != nil {
		panic(err)
	}

	return []reflect.Value{reflect.ValueOf(&block)}
}

func (t *vbTxNonceIncrementTester) ValidateResults(results []reflect.Value) error {
	if len(results) > 1 {
		return fmt.Errorf("unexpected return values: %v", results)
	}

	switch results[0].Interface().(type) {
	case coreTypes.BlockVerifyStatus:
		status := results[0].Interface().(coreTypes.BlockVerifyStatus)
		if status != coreTypes.VerifyInvalidBlock {
			return fmt.Errorf("unexpect status %v", status)
		}
	default:
		return fmt.Errorf("unexpect results[0] return type %T", results[0].Interface())
	}

	t.counter++
	t.ready = false
	return nil
}

type vbTxIntrinsicGasTester struct {
	baseTester

	key *ecdsa.PrivateKey
}

func (t vbTxIntrinsicGasTester) New(app App, key *ecdsa.PrivateKey, startAt, interval,
	threshold int) *vbTxIntrinsicGasTester {
	t.baseTester = baseTester{
		App:          app,
		testTimer:    time.NewTimer(time.Duration(startAt) * time.Second),
		testInterval: time.Duration(interval) * time.Second,
		threshold:    threshold,
		self:         t,
	}
	t.key = key
	return &t
}

func (t *vbTxIntrinsicGasTester) ViewAndRecord(product Product) {
	select {
	case <-t.testTimer.C:
		switch product.(type) {
		case *PrepareWitnessProduct:
			t.ready = true
		}
		t.testTimer.Reset(t.testInterval)
	default:
	}
}

func (t vbTxIntrinsicGasTester) InputsForTest(product Product) []reflect.Value {
	app := t.App.(*DexconApp)
	block := product.(*PrepareWitnessProduct).block
	var err error

	blockchain := app.blockchain
	signer := types.NewEIP155Signer(blockchain.Config().ChainID)
	var txs []*types.Transaction
	for i := uint64(0); i < 3; i++ {
		tx, err := types.SignTx(types.NewTransaction(i, common.Address{}, nil, 10000, new(big.Int).SetInt64(1e9), nil),
			signer, t.key)
		if err != nil {
			panic(err)
		}
		txs = append(txs, tx)
	}

	block.Payload, err = rlp.EncodeToBytes(txs)
	if err != nil {
		panic(err)
	}

	return []reflect.Value{reflect.ValueOf(&block)}
}

func (t *vbTxIntrinsicGasTester) ValidateResults(results []reflect.Value) error {
	if len(results) > 1 {
		return fmt.Errorf("unexpected return values: %v", results)
	}

	switch results[0].Interface().(type) {
	case coreTypes.BlockVerifyStatus:
		status := results[0].Interface().(coreTypes.BlockVerifyStatus)
		if status != coreTypes.VerifyInvalidBlock {
			return fmt.Errorf("unexpect status %v", status)
		}
	default:
		return fmt.Errorf("unexpect results[0] return type %T", results[0].Interface())
	}

	t.counter++
	t.ready = false
	return nil
}

type vbTxGasTooLowTester struct {
	baseTester

	key *ecdsa.PrivateKey
}

func (t vbTxGasTooLowTester) New(app App, key *ecdsa.PrivateKey, startAt, interval,
	threshold int) *vbTxGasTooLowTester {
	t.baseTester = baseTester{
		App:          app,
		testTimer:    time.NewTimer(time.Duration(startAt) * time.Second),
		testInterval: time.Duration(interval) * time.Second,
		threshold:    threshold,
		self:         t,
	}
	t.key = key
	return &t
}

func (t *vbTxGasTooLowTester) ViewAndRecord(product Product) {
	select {
	case <-t.testTimer.C:
		switch product.(type) {
		case *PrepareWitnessProduct:
			t.ready = true
		}
		t.testTimer.Reset(t.testInterval)
	default:
	}
}

func (t vbTxGasTooLowTester) InputsForTest(product Product) []reflect.Value {
	app := t.App.(*DexconApp)
	block := product.(*PrepareWitnessProduct).block
	var err error

	blockchain := app.blockchain
	signer := types.NewEIP155Signer(blockchain.Config().ChainID)
	var txs []*types.Transaction
	for i := uint64(0); i < 3; i++ {
		tx, err := types.SignTx(
			types.NewTransaction(i, common.Address{}, nil, 21000, new(big.Int).SetInt64(1e9), []byte{0x00}), signer, t.key)
		if err != nil {
			panic(err)
		}
		txs = append(txs, tx)
	}

	block.Payload, err = rlp.EncodeToBytes(txs)
	if err != nil {
		panic(err)
	}

	return []reflect.Value{reflect.ValueOf(&block)}
}

func (t *vbTxGasTooLowTester) ValidateResults(results []reflect.Value) error {
	if len(results) > 1 {
		return fmt.Errorf("unexpected return values: %v", results)
	}

	switch results[0].Interface().(type) {
	case coreTypes.BlockVerifyStatus:
		status := results[0].Interface().(coreTypes.BlockVerifyStatus)
		if status != coreTypes.VerifyInvalidBlock {
			return fmt.Errorf("unexpect status %v", status)
		}
	default:
		return fmt.Errorf("unexpect results[0] return type %T", results[0].Interface())
	}

	t.counter++
	t.ready = false
	return nil
}

type vbTxInvalidGasPriceTester struct {
	baseTester

	key *ecdsa.PrivateKey
}

func (t vbTxInvalidGasPriceTester) New(app App, key *ecdsa.PrivateKey, startAt, interval,
	threshold int) *vbTxInvalidGasPriceTester {
	t.baseTester = baseTester{
		App:          app,
		testTimer:    time.NewTimer(time.Duration(startAt) * time.Second),
		testInterval: time.Duration(interval) * time.Second,
		threshold:    threshold,
		self:         t,
	}
	t.key = key
	return &t
}

func (t *vbTxInvalidGasPriceTester) ViewAndRecord(product Product) {
	select {
	case <-t.testTimer.C:
		switch product.(type) {
		case *PrepareWitnessProduct:
			t.ready = true
		}
		t.testTimer.Reset(t.testInterval)
	default:
	}
}

func (t vbTxInvalidGasPriceTester) InputsForTest(product Product) []reflect.Value {
	app := t.App.(*DexconApp)
	block := product.(*PrepareWitnessProduct).block
	var err error

	blockchain := app.blockchain
	signer := types.NewEIP155Signer(blockchain.Config().ChainID)
	var txs []*types.Transaction
	for i := uint64(0); i < 3; i++ {
		tx, err := types.SignTx(
			types.NewTransaction(i, common.Address{}, nil, 21000, new(big.Int).SetInt64(1e8), nil), signer, t.key)
		if err != nil {
			panic(err)
		}
		txs = append(txs, tx)
	}

	block.Payload, err = rlp.EncodeToBytes(txs)
	if err != nil {
		panic(err)
	}

	return []reflect.Value{reflect.ValueOf(&block)}
}

func (t *vbTxInvalidGasPriceTester) ValidateResults(results []reflect.Value) error {
	if len(results) > 1 {
		return fmt.Errorf("unexpected return values: %v", results)
	}

	switch results[0].Interface().(type) {
	case coreTypes.BlockVerifyStatus:
		status := results[0].Interface().(coreTypes.BlockVerifyStatus)
		if status != coreTypes.VerifyInvalidBlock {
			return fmt.Errorf("unexpect status %v", status)
		}
	default:
		return fmt.Errorf("unexpect results[0] return type %T", results[0].Interface())
	}

	t.counter++
	t.ready = false
	return nil
}

type vbInsufficientFundsTester struct {
	baseTester
}

func (t vbInsufficientFundsTester) New(app App, startAt, interval, threshold int) *vbInsufficientFundsTester {
	t.baseTester = baseTester{
		App:          app,
		testTimer:    time.NewTimer(time.Duration(startAt) * time.Second),
		testInterval: time.Duration(interval) * time.Second,
		threshold:    threshold,
		self:         t,
	}
	return &t
}

func (t *vbInsufficientFundsTester) ViewAndRecord(product Product) {
	select {
	case <-t.testTimer.C:
		switch product.(type) {
		case *PrepareWitnessProduct:
			t.ready = true
		}
		t.testTimer.Reset(t.testInterval)
	default:
	}
}

func (t vbInsufficientFundsTester) InputsForTest(product Product) []reflect.Value {
	app := t.App.(*DexconApp)
	block := product.(*PrepareWitnessProduct).block
	key, err := crypto.GenerateKey()
	if err != nil {
		panic(err)
	}

	blockchain := app.blockchain
	signer := types.NewEIP155Signer(blockchain.Config().ChainID)
	var txs []*types.Transaction
	for i := uint64(0); i < 3; i++ {
		tx, err := types.SignTx(
			types.NewTransaction(i, common.Address{}, big.NewInt(1), 21000, new(big.Int).SetInt64(1e9), nil), signer, key)
		if err != nil {
			panic(err)
		}
		txs = append(txs, tx)
	}

	block.Payload, err = rlp.EncodeToBytes(txs)
	if err != nil {
		panic(err)
	}

	return []reflect.Value{reflect.ValueOf(&block)}
}

func (t *vbInsufficientFundsTester) ValidateResults(results []reflect.Value) error {
	if len(results) > 1 {
		return fmt.Errorf("unexpected return values: %v", results)
	}

	switch results[0].Interface().(type) {
	case coreTypes.BlockVerifyStatus:
		status := results[0].Interface().(coreTypes.BlockVerifyStatus)
		if status != coreTypes.VerifyInvalidBlock {
			return fmt.Errorf("unexpect status %v", status)
		}
	default:
		return fmt.Errorf("unexpect results[0] return type %T", results[0].Interface())
	}

	t.counter++
	t.ready = false
	return nil
}

type vbBlockLimitTester struct {
	baseTester
}

func (t vbBlockLimitTester) New(app App, startAt, interval, threshold int) *vbBlockLimitTester {
	t.baseTester = baseTester{
		App:          app,
		testTimer:    time.NewTimer(time.Duration(startAt) * time.Second),
		testInterval: time.Duration(interval) * time.Second,
		threshold:    threshold,
		self:         t,
	}
	return &t
}

func (t *vbBlockLimitTester) ViewAndRecord(product Product) {
	select {
	case <-t.testTimer.C:
		switch product.(type) {
		case *PrepareWitnessProduct:
			t.ready = true
		}
		t.testTimer.Reset(t.testInterval)
	default:
	}
}

func (t vbBlockLimitTester) InputsForTest(product Product) []reflect.Value {
	app := t.App.(*DexconApp)
	block := product.(*PrepareWitnessProduct).block
	key, err := crypto.GenerateKey()
	if err != nil {
		panic(err)
	}

	blockchain := app.blockchain
	signer := types.NewEIP155Signer(blockchain.Config().ChainID)
	var txs []*types.Transaction
	for i := uint64(0); i < 3; i++ {
		tx, err := types.SignTx(types.NewTransaction(i, common.Address{}, nil, 10e10, new(big.Int).SetInt64(1e9), nil),
			signer, key)
		if err != nil {
			panic(err)
		}
		txs = append(txs, tx)
	}

	block.Payload, err = rlp.EncodeToBytes(txs)
	if err != nil {
		panic(err)
	}

	return []reflect.Value{reflect.ValueOf(&block)}
}

func (t *vbBlockLimitTester) ValidateResults(results []reflect.Value) error {
	if len(results) > 1 {
		return fmt.Errorf("unexpected return values: %v", results)
	}

	switch results[0].Interface().(type) {
	case coreTypes.BlockVerifyStatus:
		status := results[0].Interface().(coreTypes.BlockVerifyStatus)
		if status != coreTypes.VerifyInvalidBlock {
			return fmt.Errorf("unexpect status %v", status)
		}
	default:
		return fmt.Errorf("unexpect results[0] return type %T", results[0].Interface())
	}

	t.counter++
	t.ready = false
	return nil
}

type BlockConfirmedFactory struct {
	FactoryBase

	masterKey *coreEcdsa.PrivateKey
}

func (f *BlockConfirmedFactory) Run() {
	defer func() {
		if r := recover(); r != nil {
			f.notifyFail(r)
		}
	}()

	for {
		product := f.center.RequestProduct(takerName(f.name))

		if len(f.testers) > 0 && f.testerAllDone() {
			f.notifySuccess()
			f.testers = nil
		} else if err := f.testerDoWork(product); err != nil {
			panic(fmt.Errorf("test fail: %v", err))
		}

		go func() {
			defer func() {
				if r := recover(); r != nil {
					f.notifyFail(r)
				}
			}()

			block := f.convertProduct(product)
			block.ProposerID = coreTypes.NewNodeID(f.masterKey.PublicKey())
			block.Timestamp = time.Now()
			f.stopTimeMu.RLock()
			f.App.BlockConfirmed(block)
			f.stopTimeMu.RUnlock()

			f.center.DeliverProduct(makerName(f.name), &BlockConfirmedProduct{
				block: block,
			})
		}()
	}
}

func (f BlockConfirmedFactory) convertProduct(product Product) coreTypes.Block {
	var block coreTypes.Block
	switch product.(type) {
	case *VerifyBlockProduct:
		block = product.(*VerifyBlockProduct).block
	default:
		panic(fmt.Errorf("unexpected type %T", product))
	}

	return block
}

func (f BlockConfirmedFactory) New(app App, center *ProductCenter, stopTimeMu *sync.RWMutex,
	masterKey *ecdsa.PrivateKey) *BlockConfirmedFactory {
	f.FactoryBase = FactoryBase{
		App:        app,
		name:       reflect.TypeOf(f).Name(),
		center:     center,
		targetFunc: app.BlockConfirmed,
		status:     make(chan map[singnal]interface{}, 1),
		stopTimeMu: stopTimeMu,
	}
	f.masterKey = coreEcdsa.NewPrivateKeyFromECDSA(masterKey)
	f.center.Register(takerName(f.name), makerName(reflect.TypeOf(VerifyBlockFactory{}).Name()))
	return &f
}

func (f BlockConfirmedFactory) NewWithTester(app App, center *ProductCenter, stopTimeMu *sync.RWMutex,
	masterKey *ecdsa.PrivateKey) *BlockConfirmedFactory {
	factory := f.New(app, center, stopTimeMu, masterKey)
	factory.testers = []Tester{
		bcBlockConfirmedTester{}.New(app, 30, 5, 3),
	}

	return factory
}

type BlockConfirmedProduct struct {
	block coreTypes.Block
}

type addInfo struct {
	nonce   *uint64
	cost    *big.Int
	counter *uint64
}

type bcBlockConfirmedTester struct {
	baseTester

	block          coreTypes.Block
	originAddrInfo map[common.Address]addInfo
}

func (t bcBlockConfirmedTester) New(app App, startAt, interval, threshold int) *bcBlockConfirmedTester {
	t.baseTester = baseTester{
		App:          app,
		testTimer:    time.NewTimer(time.Duration(startAt) * time.Second),
		testInterval: time.Duration(interval) * time.Second,
		threshold:    threshold,
		self:         t,
	}
	t.originAddrInfo = map[common.Address]addInfo{}
	return &t
}

func (t *bcBlockConfirmedTester) ViewAndRecord(product Product) {
	select {
	case <-t.testTimer.C:
		switch product.(type) {
		case *VerifyBlockProduct:
			t.block = product.(*VerifyBlockProduct).block
			var txs []*types.Transaction
			err := rlp.DecodeBytes(t.block.Payload, &txs)
			if err != nil {
				panic(err)
			} else if len(txs) > 0 {
				app := t.App.(*DexconApp)
				blockchain := app.blockchain
				for _, tx := range txs {
					msg, err := tx.AsMessage(types.MakeSigner(blockchain.Config(), new(big.Int)))
					if err != nil {
						panic(err)
					}

					if _, exist := t.originAddrInfo[msg.From()]; !exist {
						info := addInfo{}

						nonce, exist := app.addressNonce[msg.From()]
						if !exist {
							info.nonce = nil
						} else {
							info.nonce = &nonce
						}

						cost, exist := app.addressCost[msg.From()]
						if !exist {
							info.cost = nil
						} else {
							info.cost = cost
						}

						counter, exist := app.addressCounter[msg.From()]
						if !exist {
							info.counter = nil
						} else {
							info.counter = &counter
						}

						t.originAddrInfo[msg.From()] = info
					}
				}
				t.ready = true
			}
		}
		t.testTimer.Reset(t.testInterval)
	default:
	}
}

func (t bcBlockConfirmedTester) InputsForTest(product Product) []reflect.Value {
	return []reflect.Value{reflect.ValueOf(product.(*VerifyBlockProduct).block)}
}

func (t *bcBlockConfirmedTester) ValidateResults(results []reflect.Value) error {
	if len(results) > 0 {
		return fmt.Errorf("unexpected return values: %v", results)
	}

	var expectTxs []*types.Transaction
	err := rlp.DecodeBytes(t.block.Payload, &expectTxs)
	if err != nil {
		return fmt.Errorf("rlp decode error: %v", err)
	}

	app := t.App.(*DexconApp)
	blockchain := app.blockchain
	block, cachedTxs := app.getConfirmedBlockByHash(t.block.Hash)
	if block == nil {
		return fmt.Errorf("block can not be nil")
	}

	if t.block.Hash != block.Hash {
		return fmt.Errorf("block hash not equal %v vs %v", t.block.Hash, block.Hash)
	}

	addrInfo := map[common.Address]*addInfo{}
	for i, tx := range expectTxs {
		if tx.Hash() != cachedTxs[i].Hash() {
			return fmt.Errorf("incorrect tx %+v vs %+v", tx, cachedTxs[i])
		}

		msg, err := tx.AsMessage(types.MakeSigner(blockchain.Config(), new(big.Int)))
		if err != nil {
			panic(err)
		}

		nonce := tx.Nonce()
		if info, exist := addrInfo[msg.From()]; !exist {
			counter := uint64(1)
			addrInfo[msg.From()] = &addInfo{nonce: &nonce, cost: tx.Cost(), counter: &counter}
		} else {
			info.nonce = &nonce
			info.cost = new(big.Int).Add(info.cost, tx.Cost())
		}
	}

	for addr, info := range addrInfo {

		var expectCost *big.Int
		var expectNonce uint64
		var expectCounter uint64
		if t.originAddrInfo[addr].cost == nil {
			expectCost = info.cost
		} else {
			expectCost = new(big.Int).Add(t.originAddrInfo[addr].cost, info.cost)
		}

		expectNonce = *info.nonce

		if t.originAddrInfo[addr].counter == nil {
			expectCounter = *info.counter
		} else {
			expectCounter = *t.originAddrInfo[addr].counter + *info.counter
		}

		cost, exist := app.addressCost[addr]
		counter, exist := app.addressCounter[addr]
		nonce, exist := app.addressNonce[addr]
		if !exist {
			return fmt.Errorf("cache in confirmed block is empty %v %v %v", cost, counter, nonce)
		}

		if cost.Cmp(expectCost) != 0 {
			return fmt.Errorf("incorrect cost expect %v but %v", expectCost, cost)
		}

		if counter != expectCounter {
			return fmt.Errorf("incorrect counter expect %v but %v", expectCounter, counter)
		}

		if nonce != expectNonce {
			return fmt.Errorf("incorrect nonce expect %v but %v", expectNonce, nonce)
		}
	}

	t.counter++
	t.ready = false
	return nil
}

func (t bcBlockConfirmedTester) StopTime() bool {
	return true
}

func (t *bcBlockConfirmedTester) Rollback() error {
	app := t.App.(*DexconApp)
	delete(app.confirmedBlocks, t.block.Hash)
	app.undeliveredNum--
	for addr, info := range t.originAddrInfo {
		if info.nonce == nil {
			delete(app.addressNonce, addr)
		} else {
			app.addressNonce[addr] = *info.nonce
		}

		if info.cost == nil {
			delete(app.addressCost, addr)
		} else {
			app.addressCost[addr] = info.cost
		}

		if info.cost == nil {
			delete(app.addressCounter, addr)
		} else {
			app.addressCounter[addr] = *info.counter
		}
	}

	t.originAddrInfo = map[common.Address]addInfo{}
	return nil
}

type BlockDeliveredFactory struct {
	FactoryBase
}

func (f *BlockDeliveredFactory) Run() {
	defer func() {
		if r := recover(); r != nil {
			f.notifyFail(r)
		}
	}()

	for {
		product := f.center.RequestProduct(takerName(f.name))

		if len(f.testers) > 0 && f.testerAllDone() {
			f.notifySuccess()
			f.testers = nil
		} else if err := f.testerDoWork(product); err != nil {
			panic(fmt.Errorf("test fail: %v", err))
		}

		block := f.convertProduct(product)
		f.stopTimeMu.RLock()
		f.App.BlockDelivered(block.Hash, block.Position, block.Randomness)
		f.stopTimeMu.RUnlock()
	}
}

func (f BlockDeliveredFactory) convertProduct(product Product) *coreTypes.Block {
	var block *coreTypes.Block
	switch product.(type) {
	case *BlockConfirmedProduct:
		block = &product.(*BlockConfirmedProduct).block
	default:
		panic(fmt.Errorf("unexpected type %T", product))
	}

	return block
}

func (f BlockDeliveredFactory) New(app App, center *ProductCenter, stopTimeMu *sync.RWMutex) *BlockDeliveredFactory {
	f.FactoryBase = FactoryBase{
		App:        app,
		name:       reflect.TypeOf(f).Name(),
		center:     center,
		targetFunc: app.BlockDelivered,
		status:     make(chan map[singnal]interface{}, 1),
		stopTimeMu: stopTimeMu,
	}
	f.center.Register(takerName(f.name), makerName(reflect.TypeOf(BlockConfirmedFactory{}).Name()))
	return &f
}

func (f BlockDeliveredFactory) NewWithTester(app App, center *ProductCenter,
	stopTimeMu *sync.RWMutex) *BlockDeliveredFactory {
	factory := f.New(app, center, stopTimeMu)
	factory.testers = []Tester{
		bdBlockHashTester{}.New(app, 30, 5, 3),
		bdBlockDeliveredTester{}.New(app, 60, 5, 3),
	}

	return factory
}

type bdBlockHashTester struct {
	baseTester
}

func (t bdBlockHashTester) New(app App, startAt, interval, threshold int) *bdBlockHashTester {
	t.baseTester = baseTester{
		App:          app,
		testTimer:    time.NewTimer(time.Duration(startAt) * time.Second),
		testInterval: time.Duration(interval) * time.Second,
		threshold:    threshold,
		self:         t,
	}
	return &t
}

func (t *bdBlockHashTester) ViewAndRecord(product Product) {
	select {
	case <-t.testTimer.C:
		switch product.(type) {
		case *BlockConfirmedProduct:
			t.ready = true
		}
		t.testTimer.Reset(t.testInterval)
	default:
	}
}

func (t bdBlockHashTester) InputsForTest(product Product) []reflect.Value {
	block := product.(*BlockConfirmedProduct).block
	return []reflect.Value{reflect.ValueOf(coreCommon.Hash{}), reflect.ValueOf(block.Position),
		reflect.ValueOf(block.Randomness)}
}

func (t *bdBlockHashTester) ValidateResults(results []reflect.Value) error {
	if len(results) != 1 {
		return fmt.Errorf("unexpected return values: %v", results)
	}

	switch results[0].Interface().(type) {
	case error:
		if results[0].Interface().(error).Error() != "Can not get confirmed block" {
			return fmt.Errorf("unexpected error: %v", results[0].Interface().(error))
		}
	default:
		return fmt.Errorf("unexpect results[0] return type %T", results[0].Interface())
	}

	t.counter++
	t.ready = false
	return nil
}

type originalCache struct {
	confirmedBlocks map[coreCommon.Hash]*blockInfo
	addressNonce    map[common.Address]uint64
	addressCost     map[common.Address]*big.Int
	addressCounter  map[common.Address]uint64
}

type bdBlockDeliveredTester struct {
	baseTester

	expectHeight  uint64
	originalCache originalCache
	blockInfo     *blockInfo
}

func (t bdBlockDeliveredTester) New(app App, startAt, interval, threshold int) *bdBlockDeliveredTester {
	t.baseTester = baseTester{
		App:          app,
		testTimer:    time.NewTimer(time.Duration(startAt) * time.Second),
		testInterval: time.Duration(interval) * time.Second,
		threshold:    threshold,
		self:         t,
	}
	return &t
}

func (t *bdBlockDeliveredTester) ViewAndRecord(product Product) {
	select {
	case <-t.testTimer.C:
		switch product.(type) {
		case *BlockConfirmedProduct:
			app := t.App.(*DexconApp)
			block := product.(*BlockConfirmedProduct).block
			t.expectHeight = block.Position.Height
			var txs []*types.Transaction
			_, txs = app.getConfirmedBlockByHash(block.Hash)

			if len(txs) > 0 {
				t.originalCache.confirmedBlocks = map[coreCommon.Hash]*blockInfo{}
				for k, v := range app.confirmedBlocks {
					t.originalCache.confirmedBlocks[k] = v
				}

				t.originalCache.addressNonce = map[common.Address]uint64{}
				for k, v := range app.addressNonce {
					t.originalCache.addressNonce[k] = v
				}

				t.originalCache.addressCounter = map[common.Address]uint64{}
				for k, v := range app.addressCounter {
					t.originalCache.addressCounter[k] = v
				}

				t.originalCache.addressCost = map[common.Address]*big.Int{}
				for k, v := range app.addressCost {
					t.originalCache.addressCost[k] = v
				}

				t.blockInfo = app.confirmedBlocks[block.Hash]
				t.ready = true
			}
		}
		t.testTimer.Reset(t.testInterval)
	default:
	}
}

func (t bdBlockDeliveredTester) InputsForTest(product Product) []reflect.Value {
	block := product.(*BlockConfirmedProduct).block
	return []reflect.Value{reflect.ValueOf(block.Hash), reflect.ValueOf(block.Position),
		reflect.ValueOf(block.Randomness)}
}

func (t *bdBlockDeliveredTester) ValidateResults(results []reflect.Value) error {
	if len(results) != 0 {
		return fmt.Errorf("unexpected return values: %v", results)
	}

	app := t.App.(*DexconApp)
	if app.deliveredHeight != t.expectHeight {
		return fmt.Errorf("unexpected delivered height: expect %d but %d", t.expectHeight, app.deliveredHeight)
	}

	for addr, info := range t.blockInfo.addresses {
		if t.originalCache.addressCounter[addr] == 1 {
			_, exist := app.addressNonce[addr]
			if exist {
				return fmt.Errorf("nonce cache %v should not exist", addr)
			}

			_, exist = app.addressCost[addr]
			if exist {
				return fmt.Errorf("cost cache %v should not exist", addr)
			}

			_, exist = app.addressCounter[addr]
			if exist {
				return fmt.Errorf("counter cache %v should not exist", addr)
			}
			continue
		}

		if app.addressNonce[addr] != t.originalCache.addressNonce[addr] {
			return fmt.Errorf("nonce should not be affected")
		}

		expectCost := new(big.Int).Sub(t.originalCache.addressCost[addr], info.cost)
		if expectCost.Cmp(app.addressCost[addr]) != 0 {
			return fmt.Errorf("unexpected cost %v %v vs %v", addr, expectCost, app.addressCost[addr])
		}

		if app.addressCounter[addr]+1 != t.originalCache.addressCounter[addr] {
			return fmt.Errorf("unexpected counter %v vs %v", app.addressCounter[addr]+1, t.originalCache.addressCounter[addr])
		}
	}

	t.counter++
	t.ready = false
	return nil
}

func (t bdBlockDeliveredTester) StopTime() bool {
	return true
}

func (t *bdBlockDeliveredTester) Rollback() error {
	app := t.App.(*DexconApp)
	block := app.blockchain.CurrentBlock()
	app.blockchain.Rollback([]common.Hash{app.blockchain.CurrentBlock().Hash()})
	rawdb.DeleteCanonicalHash(t.App.(*DexconApp).chainDB, block.NumberU64())
	time.Sleep(100 * time.Millisecond)
	app.txPool.Reset(app.blockchain.CurrentBlock().Header())

	app.confirmedBlocks = t.originalCache.confirmedBlocks
	app.addressNonce = t.originalCache.addressNonce
	app.addressCost = t.originalCache.addressCost
	app.addressCounter = t.originalCache.addressCounter
	app.undeliveredNum++
	app.deliveredHeight--
	return nil
}

type TxFactory struct {
	FactoryBase

	keys []*ecdsa.PrivateKey

	sendInterval time.Duration

	nonce uint64
}

func (f *TxFactory) Run() {
	blockchain := f.App.(*DexconApp).blockchain
	txPool := f.App.(*DexconApp).txPool
	for {
		for i, key := range f.keys {
			go func(at int, nonce uint64, key *ecdsa.PrivateKey) {
				f.stopTimeMu.RLock()
				for i := 0; i < len(f.keys); i++ {
					if i == at {
						continue
					}

					tx := types.NewTransaction(
						nonce,
						crypto.PubkeyToAddress(f.keys[i].PublicKey),
						big.NewInt(1),
						21000,
						big.NewInt(1e9),
						[]byte{})

					signer := types.NewEIP155Signer(blockchain.Config().ChainID)

					tx, err := types.SignTx(tx, signer, key)
					if err != nil {
						panic(err)
					}

					err = txPool.AddLocal(tx)
					if err != nil {
						panic(err)
					}
					nonce++
				}
				f.stopTimeMu.RUnlock()
			}(i, f.nonce, key)
		}

		f.nonce += uint64(len(f.keys)) - 1

		time.Sleep(f.sendInterval)
	}
}

func (f TxFactory) New(app App, center *ProductCenter, stopTimeMu *sync.RWMutex, keys []*ecdsa.PrivateKey) *TxFactory {
	f.FactoryBase = FactoryBase{
		App:        app,
		name:       reflect.TypeOf(f).Name(),
		center:     center,
		stopTimeMu: stopTimeMu,
	}
	f.keys = keys
	f.sendInterval = 1000 * time.Millisecond
	return &f
}

func TestDexonApp(t *testing.T) {
	masterKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Generate key fail: %v", err)
	}

	dex, keys, err := newDexon(masterKey, 15)
	if err != nil {
		t.Fatalf("New dexon fail: %v", err)
	}

	stopTimeMu := &sync.RWMutex{}

	center := ProductCenter{}.New()
	configFactory := ConfigFactory{}.New(dex.app, center, stopTimeMu, masterKey)
	preparePayloadFactory := PreparePayloadFactory{}.NewWithTester(dex.app, center, stopTimeMu)
	prepareWitnessFactory := PrepareWitnessFactory{}.NewWithTester(dex.app, center, stopTimeMu)
	verifyBlockFactory := VerifyBlockFactory{}.NewWithTester(dex.app, center, masterKey, stopTimeMu)
	blockConfirmedFactory := BlockConfirmedFactory{}.NewWithTester(dex.app, center, stopTimeMu, masterKey)
	blockDeliveredFactory := BlockDeliveredFactory{}.NewWithTester(dex.app, center, stopTimeMu)
	txFactory := TxFactory{}.New(dex.app, center, stopTimeMu, keys)

	go configFactory.Run()
	go preparePayloadFactory.Run()
	go prepareWitnessFactory.Run()
	go verifyBlockFactory.Run()
	go blockConfirmedFactory.Run()
	go blockDeliveredFactory.Run()
	go txFactory.Run()

	timer := time.NewTimer(300 * time.Second)
	successRecord := make(map[string]struct{})
	for {
		select {
		case sig := <-preparePayloadFactory.status:
			if _, exist := sig[runSuccess]; exist {
				successRecord[reflect.TypeOf(*preparePayloadFactory).Name()] = struct{}{}
			} else if msg, exist := sig[runFail]; exist {
				t.Fatalf("preparePayloadFactory error: %v", msg)
			}
		case sig := <-prepareWitnessFactory.status:
			if _, exist := sig[runSuccess]; exist {
				successRecord[reflect.TypeOf(*prepareWitnessFactory).Name()] = struct{}{}
			} else if msg, exist := sig[runFail]; exist {
				t.Fatalf("prepareWitnessFactory error: %v", msg)
			}
		case sig := <-verifyBlockFactory.status:
			if _, exist := sig[runSuccess]; exist {
				successRecord[reflect.TypeOf(*verifyBlockFactory).Name()] = struct{}{}
			} else if msg, exist := sig[runFail]; exist {
				t.Fatalf("verifyBlockFactory error: %v", msg)
			}
		case sig := <-blockConfirmedFactory.status:
			if _, exist := sig[runSuccess]; exist {
				successRecord[reflect.TypeOf(*blockConfirmedFactory).Name()] = struct{}{}
			} else if msg, exist := sig[runFail]; exist {
				t.Fatalf("blockConfirmedFactory error: %v", msg)
			}
		case sig := <-blockDeliveredFactory.status:
			if _, exist := sig[runSuccess]; exist {
				successRecord[reflect.TypeOf(*blockDeliveredFactory).Name()] = struct{}{}
			} else if msg, exist := sig[runFail]; exist {
				t.Fatalf("blockDeliveredFactory error: %v", msg)
			}
		case <-timer.C:
			t.Fatalf("time's up and all test is not finish yet: %v", successRecord)
		}

		leftTesterCount := len(preparePayloadFactory.testers) + len(prepareWitnessFactory.testers) +
			len(verifyBlockFactory.testers) + len(blockConfirmedFactory.testers) + len(blockDeliveredFactory.testers)
		if leftTesterCount == 0 {
			t.Logf("tests all pass")
			break
		}

		time.Sleep(1 * time.Second)
	}
}

func newDexon(masterKey *ecdsa.PrivateKey, accountNum int) (*Dexon, []*ecdsa.PrivateKey, error) {
	db := ethdb.NewMemDatabase()

	genesis := core.DefaultTestnetGenesisBlock()
	genesis.Alloc[crypto.PubkeyToAddress(masterKey.PublicKey)] = core.GenesisAccount{
		Balance:   big.NewInt(100000000000000000),
		Staked:    big.NewInt(50000000000000000),
		PublicKey: crypto.FromECDSAPub(&masterKey.PublicKey),
	}

	var accounts []*ecdsa.PrivateKey
	for i := 0; i < accountNum; i++ {
		key, err := crypto.GenerateKey()
		if err != nil {
			panic(err)
		}

		genesis.Alloc[crypto.PubkeyToAddress(key.PublicKey)] = core.GenesisAccount{
			Balance: math.BigPow(10, 18),
			Staked:  big.NewInt(0),
		}
		accounts = append(accounts, key)
	}

	genesis.Config.Dexcon.BlockGasLimit = 2000000
	genesis.Config.Dexcon.RoundLength = 600
	genesis.Config.Dexcon.Owner = crypto.PubkeyToAddress(masterKey.PublicKey)

	chainConfig, _, err := core.SetupGenesisBlock(db, genesis)
	if err != nil {
		return nil, nil, err
	}

	config := Config{PrivateKey: masterKey}
	vmConfig := vm.Config{IsBlockProposer: true}

	engine := dexcon.New()

	dex := &Dexon{
		chainDb:     db,
		chainConfig: chainConfig,
		networkID:   config.NetworkId,
		engine:      engine,
	}

	dex.blockchain, err = core.NewBlockChain(db, nil, chainConfig, engine, vmConfig, nil)
	if err != nil {
		return nil, nil, err
	}

	txPoolConfig := core.DefaultTxPoolConfig
	dex.txPool = core.NewTxPool(txPoolConfig, chainConfig, dex.blockchain)

	dex.APIBackend = &DexAPIBackend{dex, nil}
	dex.governance = NewDexconGovernance(dex.APIBackend, dex.chainConfig, config.PrivateKey)
	engine.SetGovStateFetcher(dex.governance)
	dex.app = NewDexconApp(dex.txPool, dex.blockchain, dex.governance, db, &config)

	return dex, accounts, nil
}
