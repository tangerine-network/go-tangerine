package dex

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"testing"
	"time"

	coreCommon "github.com/dexon-foundation/dexon-consensus/common"
	coreTypes "github.com/dexon-foundation/dexon-consensus/core/types"

	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/consensus/dexcon"
	"github.com/dexon-foundation/dexon/core"
	"github.com/dexon-foundation/dexon/core/types"
	"github.com/dexon-foundation/dexon/core/vm"
	"github.com/dexon-foundation/dexon/crypto"
	"github.com/dexon-foundation/dexon/ethdb"
	"github.com/dexon-foundation/dexon/params"
	"github.com/dexon-foundation/dexon/rlp"
)

func TestPreparePayload(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Errorf("hex to ecdsa error: %v", err)
	}

	dex, err := newTestDexonWithGenesis(key)
	if err != nil {
		t.Errorf("new test dexon error: %v", err)
	}

	signer := types.NewEIP155Signer(dex.chainConfig.ChainID)

	var expectTx types.Transactions
	for i := 0; i < 5; i++ {
		tx, err := addTx(dex, i, signer, key)
		if err != nil {
			t.Errorf("add tx error: %v", err)
		}
		expectTx = append(expectTx, tx)
	}

	// This transaction will not be included.
	_, err = addTx(dex, 100, signer, key)
	if err != nil {
		t.Errorf("add tx error: %v", err)
	}

	chainID := new(big.Int).Mod(crypto.PubkeyToAddress(key.PublicKey).Big(),
		big.NewInt(int64(dex.chainConfig.Dexcon.NumChains)))
	var chainNum uint32
	for chainNum = 0; chainNum < dex.chainConfig.Dexcon.NumChains; chainNum++ {

		payload, err := dex.app.PreparePayload(coreTypes.Position{ChainID: chainNum})
		if err != nil {
			t.Errorf("prepare payload error: %v", err)
		}

		var transactions types.Transactions
		err = rlp.DecodeBytes(payload, &transactions)
		if err != nil {
			t.Errorf("rlp decode error: %v", err)
		}

		// Only one chain id allow prepare transactions.
		if chainID.Uint64() == uint64(chainNum) && len(transactions) != 5 {
			t.Errorf("incorrect transaction num expect %v but %v", 5, len(transactions))
		} else if chainID.Uint64() != uint64(chainNum) && len(transactions) != 0 {
			t.Errorf("expect empty slice but %v", len(transactions))
		}

		for i, tx := range transactions {
			if expectTx[i].Gas() != tx.Gas() {
				t.Errorf("unexpected gas expect %v but %v", expectTx[i].Gas(), tx.Gas())
			}

			if expectTx[i].Hash() != tx.Hash() {
				t.Errorf("unexpected hash expect %v but %v", expectTx[i].Hash(), tx.Hash())
			}

			if expectTx[i].Nonce() != tx.Nonce() {
				t.Errorf("unexpected nonce expect %v but %v", expectTx[i].Nonce(), tx.Nonce())
			}

			if expectTx[i].GasPrice().Uint64() != tx.GasPrice().Uint64() {
				t.Errorf("unexpected gas price expect %v but %v",
					expectTx[i].GasPrice().Uint64(), tx.GasPrice().Uint64())
			}
		}
	}
}

func TestPrepareWitness(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Errorf("hex to ecdsa error: %v", err)
	}

	dex, err := newTestDexonWithGenesis(key)
	if err != nil {
		t.Errorf("new test dexon error: %v", err)
	}

	currentBlock := dex.blockchain.CurrentBlock()

	witness, err := dex.app.PrepareWitness(0)
	if err != nil {
		t.Errorf("prepare witness error: %v", err)
	}

	if witness.Height != currentBlock.NumberU64() {
		t.Errorf("unexpeted witness height %v", witness.Height)
	}

	var witnessData types.WitnessData
	err = rlp.DecodeBytes(witness.Data, &witnessData)
	if err != nil {
		t.Errorf("rlp decode error: %v", err)
	}

	if witnessData.Root != currentBlock.Root() {
		t.Errorf("expect root %v but %v", currentBlock.Root(), witnessData.Root)
	}

	if witnessData.ReceiptHash != currentBlock.ReceiptHash() {
		t.Errorf("expect receipt hash %v but %v", currentBlock.ReceiptHash(), witnessData.ReceiptHash)
	}

	if _, err := dex.app.PrepareWitness(999); err == nil {
		t.Errorf("it must be get error from prepare")
	} else {
		t.Logf("Nice error: %v", err)
	}
}

func TestVerifyBlock(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Errorf("hex to ecdsa error: %v", err)
	}

	dex, err := newTestDexonWithGenesis(key)
	if err != nil {
		t.Errorf("new test dexon error: %v", err)
	}

	// Prepare first confirmed block.
	prepareConfirmedBlocks(dex, []*ecdsa.PrivateKey{key}, 0)

	chainID := new(big.Int).Mod(crypto.PubkeyToAddress(key.PublicKey).Big(),
		big.NewInt(int64(dex.chainConfig.Dexcon.NumChains)))

	// Prepare normal block.
	block := &coreTypes.Block{}
	block.Hash = coreCommon.NewRandomHash()
	block.Position.ChainID = uint32(chainID.Uint64())
	block.Position.Height = 1
	block.Payload, block.Witness, err = prepareDataWithoutTxPool(dex, key, 0, 100)
	if err != nil {
		t.Errorf("prepare data error: %v", err)
	}

	// Expect ok.
	status := dex.app.VerifyBlock(block)
	if status != coreTypes.VerifyOK {
		t.Errorf("verify fail: %v", status)
	}

	// Prepare invalid nonce tx.
	block = &coreTypes.Block{}
	block.Hash = coreCommon.NewRandomHash()
	block.Position.ChainID = uint32(chainID.Uint64())
	block.Position.Height = 1
	block.Payload, block.Witness, err = prepareDataWithoutTxPool(dex, key, 1, 100)
	if err != nil {
		t.Errorf("prepare data error: %v", err)
	}

	// Expect invalid block.
	status = dex.app.VerifyBlock(block)
	if status != coreTypes.VerifyInvalidBlock {
		t.Errorf("verify fail: %v", status)
	}

	// Prepare invalid block height.
	block = &coreTypes.Block{}
	block.Hash = coreCommon.NewRandomHash()
	block.Position.ChainID = uint32(chainID.Uint64())
	block.Position.Height = 2
	block.Payload, block.Witness, err = prepareDataWithoutTxPool(dex, key, 0, 100)
	if err != nil {
		t.Errorf("prepare data error: %v", err)
	}

	// Expect retry later.
	status = dex.app.VerifyBlock(block)
	if status != coreTypes.VerifyRetryLater {
		t.Errorf("verify fail expect retry later but get %v", status)
	}

	// Prepare reach block limit.
	block = &coreTypes.Block{}
	block.Hash = coreCommon.NewRandomHash()
	block.Position.ChainID = uint32(chainID.Uint64())
	block.Position.Height = 1
	block.Payload, block.Witness, err = prepareDataWithoutTxPool(dex, key, 0, 10000)
	if err != nil {
		t.Errorf("prepare data error: %v", err)
	}

	// Expect invalid block.
	status = dex.app.VerifyBlock(block)
	if status != coreTypes.VerifyInvalidBlock {
		t.Errorf("verify fail expect invalid block but get %v", status)
	}

	// Prepare insufficient funds.
	block = &coreTypes.Block{}
	block.Hash = coreCommon.NewRandomHash()
	block.Position.ChainID = uint32(chainID.Uint64())
	block.Position.Height = 1
	_, block.Witness, err = prepareDataWithoutTxPool(dex, key, 0, 0)
	if err != nil {
		t.Errorf("prepare data error: %v", err)
	}

	signer := types.NewEIP155Signer(dex.chainConfig.ChainID)
	tx := types.NewTransaction(
		0,
		common.BytesToAddress([]byte{9}),
		big.NewInt(50000000000000001),
		params.TxGas,
		big.NewInt(1),
		nil)
	tx, err = types.SignTx(tx, signer, key)
	if err != nil {
		return
	}

	block.Payload, err = rlp.EncodeToBytes(types.Transactions{tx})
	if err != nil {
		return
	}

	// expect invalid block
	status = dex.app.VerifyBlock(block)
	if status != coreTypes.VerifyInvalidBlock {
		t.Errorf("verify fail expect invalid block but get %v", status)
	}

	// Prepare invalid intrinsic gas.
	block = &coreTypes.Block{}
	block.Hash = coreCommon.NewRandomHash()
	block.Position.ChainID = uint32(chainID.Uint64())
	block.Position.Height = 1
	_, block.Witness, err = prepareDataWithoutTxPool(dex, key, 0, 0)
	if err != nil {
		t.Errorf("prepare data error: %v", err)
	}

	signer = types.NewEIP155Signer(dex.chainConfig.ChainID)
	tx = types.NewTransaction(
		0,
		common.BytesToAddress([]byte{9}),
		big.NewInt(1),
		params.TxGas,
		big.NewInt(1),
		make([]byte, 1))
	tx, err = types.SignTx(tx, signer, key)
	if err != nil {
		return
	}

	block.Payload, err = rlp.EncodeToBytes(types.Transactions{tx})
	if err != nil {
		return
	}

	// Expect invalid block.
	status = dex.app.VerifyBlock(block)
	if status != coreTypes.VerifyInvalidBlock {
		t.Errorf("verify fail expect invalid block but get %v", status)
	}

	// Prepare invalid transactions with nonce.
	block = &coreTypes.Block{}
	block.Hash = coreCommon.NewRandomHash()
	block.Position.ChainID = uint32(chainID.Uint64())
	block.Position.Height = 1
	_, block.Witness, err = prepareDataWithoutTxPool(dex, key, 0, 0)
	if err != nil {
		t.Errorf("prepare data error: %v", err)
	}

	signer = types.NewEIP155Signer(dex.chainConfig.ChainID)
	tx1 := types.NewTransaction(
		0,
		common.BytesToAddress([]byte{9}),
		big.NewInt(1),
		params.TxGas,
		big.NewInt(1),
		make([]byte, 1))
	tx1, err = types.SignTx(tx, signer, key)
	if err != nil {
		return
	}

	// Invalid nonce.
	tx2 := types.NewTransaction(
		2,
		common.BytesToAddress([]byte{9}),
		big.NewInt(1),
		params.TxGas,
		big.NewInt(1),
		make([]byte, 1))
	tx2, err = types.SignTx(tx, signer, key)
	if err != nil {
		return
	}

	block.Payload, err = rlp.EncodeToBytes(types.Transactions{tx1, tx2})
	if err != nil {
		return
	}

	// Expect invalid block.
	status = dex.app.VerifyBlock(block)
	if status != coreTypes.VerifyInvalidBlock {
		t.Errorf("verify fail expect invalid block but get %v", status)
	}
}

func TestBlockConfirmed(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Errorf("hex to ecdsa error: %v", err)
	}

	dex, err := newTestDexonWithGenesis(key)
	if err != nil {
		t.Errorf("new test dexon error: %v", err)
	}

	chainID := new(big.Int).Mod(crypto.PubkeyToAddress(key.PublicKey).Big(),
		big.NewInt(int64(dex.chainConfig.Dexcon.NumChains)))

	var (
		expectCost    big.Int
		expectNonce   uint64
		expectCounter uint64
	)
	for i := 0; i < 10; i++ {
		var startNonce int
		if expectNonce != 0 {
			startNonce = int(expectNonce) + 1
		}
		payload, witness, cost, nonce, err := prepareData(dex, key, startNonce, 50)
		if err != nil {
			t.Errorf("prepare data error: %v", err)
		}
		expectCost.Add(&expectCost, &cost)
		expectNonce = nonce

		block := &coreTypes.Block{}
		block.Hash = coreCommon.NewRandomHash()
		block.Witness = witness
		block.Payload = payload
		block.Position.ChainID = uint32(chainID.Uint64())

		dex.app.BlockConfirmed(*block)
		expectCounter++
	}

	info := dex.app.blockchain.GetAddressInfo(uint32(chainID.Uint64()),
		crypto.PubkeyToAddress(key.PublicKey))

	if info.Counter != expectCounter {
		t.Errorf("expect address counter is %v but %v", expectCounter, info.Counter)
	}

	if info.Cost.Cmp(&expectCost) != 0 {
		t.Errorf("expect address cost is %v but %v", expectCost.Uint64(), info.Cost.Uint64())
	}

	if info.Nonce != expectNonce {
		t.Errorf("expect address nonce is %v but %v", expectNonce, info.Nonce)
	}
}

func TestBlockDelivered(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Errorf("hex to ecdsa error: %v", err)
	}

	dex, err := newTestDexonWithGenesis(key)
	if err != nil {
		t.Errorf("new test dexon error: %v", err)
	}

	address := crypto.PubkeyToAddress(key.PublicKey)
	firstBlocksInfo, err := prepareConfirmedBlocks(dex, []*ecdsa.PrivateKey{key}, 50)
	if err != nil {
		t.Errorf("preapare confirmed block error: %v", err)
	}

	dex.app.BlockDelivered(firstBlocksInfo[0].Block.Hash, firstBlocksInfo[0].Block.Position,
		coreTypes.FinalizationResult{
			Timestamp: time.Now(),
			Height:    1,
		})

	_, pendingState := dex.blockchain.GetPending()
	currentBlock := dex.blockchain.CurrentBlock()
	if currentBlock.NumberU64() != 0 {
		t.Errorf("unexpected current block number %v", currentBlock.NumberU64())
	}

	pendingNonce := pendingState.GetNonce(address)
	if pendingNonce != firstBlocksInfo[0].Nonce+1 {
		t.Errorf("unexpected pending state nonce %v", pendingNonce)
	}

	balance := pendingState.GetBalance(address)
	if new(big.Int).Add(balance, &firstBlocksInfo[0].Cost).Cmp(big.NewInt(50000000000000000)) != 0 {
		t.Errorf("unexpected pending state balance %v", balance)
	}

	// prepare second block to witness first block
	secondBlocksInfo, err := prepareConfirmedBlocks(dex, []*ecdsa.PrivateKey{key}, 25)
	if err != nil {
		t.Errorf("preapare confirmed block error: %v", err)
	}

	dex.app.BlockDelivered(secondBlocksInfo[0].Block.Hash, secondBlocksInfo[0].Block.Position,
		coreTypes.FinalizationResult{
			Timestamp: time.Now(),
			Height:    2,
		})

	// second block witness first block, so current block number should be 1
	currentBlock = dex.blockchain.CurrentBlock()
	if currentBlock.NumberU64() != 1 {
		t.Errorf("unexpected current block number %v", currentBlock.NumberU64())
	}

	currentState, err := dex.blockchain.State()
	if err != nil {
		t.Errorf("current state error: %v", err)
	}

	currentNonce := currentState.GetNonce(address)
	if currentNonce != firstBlocksInfo[0].Nonce+1 {
		t.Errorf("unexpected current state nonce %v", currentNonce)
	}

	balance = currentState.GetBalance(address)
	if new(big.Int).Add(balance, &firstBlocksInfo[0].Cost).Cmp(big.NewInt(50000000000000000)) != 0 {
		t.Errorf("unexpected current state balance %v", balance)
	}
}

func BenchmarkBlockDeliveredFlow(b *testing.B) {
	key, err := crypto.GenerateKey()
	if err != nil {
		b.Errorf("hex to ecdsa error: %v", err)
		return
	}

	dex, err := newTestDexonWithGenesis(key)
	if err != nil {
		b.Errorf("new test dexon error: %v", err)
	}

	b.ResetTimer()
	for i := 1; i <= b.N; i++ {
		blocksInfo, err := prepareConfirmedBlocks(dex, []*ecdsa.PrivateKey{key}, 100)
		if err != nil {
			b.Errorf("preapare confirmed block error: %v", err)
			return
		}

		dex.app.BlockDelivered(blocksInfo[0].Block.Hash, blocksInfo[0].Block.Position,
			coreTypes.FinalizationResult{
				Timestamp: time.Now(),
				Height:    uint64(i),
			})
	}
}

func newTestDexonWithGenesis(allocKey *ecdsa.PrivateKey) (*Dexon, error) {
	db := ethdb.NewMemDatabase()

	testBankAddress := crypto.PubkeyToAddress(allocKey.PublicKey)
	genesis := core.DefaultTestnetGenesisBlock()
	genesis.Alloc = core.GenesisAlloc{
		testBankAddress: {
			Balance: big.NewInt(100000000000000000),
			Staked:  big.NewInt(50000000000000000),
		},
	}
	chainConfig, _, err := core.SetupGenesisBlock(db, genesis)
	if err != nil {
		return nil, err
	}

	key, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}

	config := Config{PrivateKey: key}
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
		return nil, err
	}

	txPoolConfig := core.DefaultTxPoolConfig
	dex.txPool = core.NewTxPool(txPoolConfig, chainConfig, dex.blockchain, true)

	dex.APIBackend = &DexAPIBackend{dex, nil}
	dex.governance = NewDexconGovernance(dex.APIBackend, dex.chainConfig, config.PrivateKey)
	engine.SetConfigFetcher(dex.governance)
	dex.app = NewDexconApp(dex.txPool, dex.blockchain, dex.governance, db, &config)

	return dex, nil
}

// Add tx into tx pool.
func addTx(dex *Dexon, nonce int, signer types.Signer, key *ecdsa.PrivateKey) (
	*types.Transaction, error) {
	tx := types.NewTransaction(
		uint64(nonce),
		common.BytesToAddress([]byte{9}),
		big.NewInt(int64(nonce*1)),
		params.TxGas,
		big.NewInt(1),
		nil)
	tx, err := types.SignTx(tx, signer, key)
	if err != nil {
		return nil, err
	}

	if err := dex.txPool.AddRemote(tx); err != nil {
		return nil, err
	}

	return tx, nil
}

// Prepare data with given transaction number and start nonce.
func prepareData(dex *Dexon, key *ecdsa.PrivateKey, startNonce, txNum int) (
	payload []byte, witness coreTypes.Witness, cost big.Int, nonce uint64, err error) {
	signer := types.NewEIP155Signer(dex.chainConfig.ChainID)
	chainID := new(big.Int).Mod(crypto.PubkeyToAddress(key.PublicKey).Big(),
		big.NewInt(int64(dex.chainConfig.Dexcon.NumChains)))

	for n := startNonce; n < startNonce+txNum; n++ {
		var tx *types.Transaction
		tx, err = addTx(dex, n, signer, key)
		if err != nil {
			return
		}

		cost.Add(&cost, tx.Cost())
		nonce = uint64(n)
	}

	payload, err = dex.app.PreparePayload(coreTypes.Position{ChainID: uint32(chainID.Uint64())})
	if err != nil {
		return
	}

	witness, err = dex.app.PrepareWitness(0)
	if err != nil {
		return
	}

	return
}

func prepareDataWithoutTxPool(dex *Dexon, key *ecdsa.PrivateKey, startNonce, txNum int) (
	payload []byte, witness coreTypes.Witness, err error) {
	signer := types.NewEIP155Signer(dex.chainConfig.ChainID)

	var transactions types.Transactions
	for n := startNonce; n < startNonce+txNum; n++ {
		tx := types.NewTransaction(
			uint64(n),
			common.BytesToAddress([]byte{9}),
			big.NewInt(int64(n*1)),
			params.TxGas,
			big.NewInt(1),
			nil)
		tx, err = types.SignTx(tx, signer, key)
		if err != nil {
			return
		}
		transactions = append(transactions, tx)
	}

	payload, err = rlp.EncodeToBytes(&transactions)
	if err != nil {
		return
	}

	witness, err = dex.app.PrepareWitness(0)
	if err != nil {
		return
	}

	return
}

func prepareConfirmedBlocks(dex *Dexon, keys []*ecdsa.PrivateKey, txNum int) (blocksInfo []struct {
	Block *coreTypes.Block
	Cost  big.Int
	Nonce uint64
}, err error) {
	for _, key := range keys {
		address := crypto.PubkeyToAddress(key.PublicKey)
		chainID := new(big.Int).Mod(address.Big(),
			big.NewInt(int64(dex.chainConfig.Dexcon.NumChains)))

		// Prepare one block for pending.
		var (
			payload []byte
			witness coreTypes.Witness
			cost    big.Int
			nonce   uint64
		)
		startNonce := dex.txPool.State().GetNonce(address)
		payload, witness, cost, nonce, err = prepareData(dex, key, int(startNonce), txNum)
		if err != nil {
			err = fmt.Errorf("prepare data error: %v", err)
			return
		}

		block := &coreTypes.Block{}
		block.Hash = coreCommon.NewRandomHash()
		block.Witness = witness
		block.Payload = payload
		block.Position.ChainID = uint32(chainID.Uint64())

		status := dex.app.VerifyBlock(block)
		if status != coreTypes.VerifyOK {
			err = fmt.Errorf("verify fail: %v", status)
			return
		}

		dex.app.BlockConfirmed(*block)

		blocksInfo = append(blocksInfo, struct {
			Block *coreTypes.Block
			Cost  big.Int
			Nonce uint64
		}{Block: block, Cost: cost, Nonce: nonce})
	}

	return
}
