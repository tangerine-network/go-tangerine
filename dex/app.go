// Copyright 2018 The dexon-consensus Authors
// This file is part of the dexon-consensus library.
//
// The dexon-consensus library is free software: you can redistribute it
// and/or modify it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The dexon-consensus library is distributed in the hope that it will be
// useful, but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the dexon-consensus library. If not, see
// <http://www.gnu.org/licenses/>.

package dex

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	coreCommon "github.com/dexon-foundation/dexon-consensus/common"
	coreTypes "github.com/dexon-foundation/dexon-consensus/core/types"

	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/core"
	"github.com/dexon-foundation/dexon/core/types"
	"github.com/dexon-foundation/dexon/ethdb"
	"github.com/dexon-foundation/dexon/event"
	"github.com/dexon-foundation/dexon/log"
	"github.com/dexon-foundation/dexon/rlp"
)

// DexconApp implements the DEXON consensus core application interface.
type DexconApp struct {
	txPool     *core.TxPool
	blockchain *core.BlockChain
	gov        *DexconGovernance
	chainDB    ethdb.Database
	config     *Config

	finalizedBlockFeed event.Feed
	scope              event.SubscriptionScope

	appMu sync.RWMutex

	confirmedBlocks map[coreCommon.Hash]*blockInfo
	addressNonce    map[common.Address]uint64
	addressCost     map[common.Address]*big.Int
	addressCounter  map[common.Address]uint64
	undeliveredNum  uint64
	deliveredHeight uint64
}

func NewDexconApp(txPool *core.TxPool, blockchain *core.BlockChain, gov *DexconGovernance,
	chainDB ethdb.Database, config *Config) *DexconApp {
	return &DexconApp{
		txPool:          txPool,
		blockchain:      blockchain,
		gov:             gov,
		chainDB:         chainDB,
		config:          config,
		confirmedBlocks: map[coreCommon.Hash]*blockInfo{},
		addressNonce:    map[common.Address]uint64{},
		addressCost:     map[common.Address]*big.Int{},
		addressCounter:  map[common.Address]uint64{},
		deliveredHeight: blockchain.CurrentBlock().NumberU64(),
	}
}

// validateNonce check if nonce is in order and return first nonce of every address.
func (d *DexconApp) validateNonce(txs types.Transactions) (map[common.Address]uint64, error) {
	addressFirstNonce := map[common.Address]uint64{}
	addressNonce := map[common.Address]uint64{}

	for _, tx := range txs {
		msg, err := tx.AsMessage(types.MakeSigner(d.blockchain.Config(), new(big.Int)))
		if err != nil {
			return nil, err
		}

		if _, exist := addressFirstNonce[msg.From()]; exist {
			if addressNonce[msg.From()]+1 != msg.Nonce() {
				return nil, fmt.Errorf("address nonce check error: expect %v actual %v",
					addressNonce[msg.From()]+1, msg.Nonce())
			}
			addressNonce[msg.From()] = msg.Nonce()
		} else {
			addressNonce[msg.From()] = msg.Nonce()
			addressFirstNonce[msg.From()] = msg.Nonce()
		}
	}
	return addressFirstNonce, nil
}

// validateGasPrice checks if no gas price is lower than minGasPrice defined in
// governance contract.
func (d *DexconApp) validateGasPrice(txs types.Transactions, round uint64) bool {
	minGasPrice := d.gov.MinGasPrice(round)
	for _, tx := range txs {
		if minGasPrice.Cmp(tx.GasPrice()) > 0 {
			return false
		}
	}
	return true
}

// PreparePayload is called when consensus core is preparing payload for block.
func (d *DexconApp) PreparePayload(position coreTypes.Position) (payload []byte, err error) {
	// softLimit limits the runtime of inner call to preparePayload.
	// hardLimit limits the runtime of outer PreparePayload.
	// If hardLimit is hit, it is possible that no payload is prepared.
	softLimit := 100 * time.Millisecond
	hardLimit := 150 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), hardLimit)
	defer cancel()
	payloadCh := make(chan []byte, 1)
	errCh := make(chan error, 1)
	doneCh := make(chan struct{}, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), softLimit)
		defer cancel()
		payload, err := d.preparePayload(ctx, position)
		if err != nil {
			errCh <- err
		}
		payloadCh <- payload
		doneCh <- struct{}{}
	}()
	select {
	case <-ctx.Done():
	case <-doneCh:
	}
	select {
	case err = <-errCh:
		if err != nil {
			return
		}
	default:
	}
	select {
	case payload = <-payloadCh:
	default:
	}
	return
}

func (d *DexconApp) preparePayload(ctx context.Context, position coreTypes.Position) (
	payload []byte, err error) {
	d.appMu.RLock()
	defer d.appMu.RUnlock()
	select {
	// This case will hit if previous RLock took too much time.
	case <-ctx.Done():
		return
	default:
	}

	// deliver height = position height + 1
	if d.deliveredHeight+d.undeliveredNum != position.Height {
		return nil, fmt.Errorf("expected height %d but get %d", d.deliveredHeight+d.undeliveredNum, position.Height)
	}

	deliveredBlock := d.blockchain.GetBlockByNumber(d.deliveredHeight)
	state, err := d.blockchain.StateAt(deliveredBlock.Root())
	if err != nil {
		return nil, fmt.Errorf("get state by root %v error: %v", deliveredBlock.Root(), err)
	}

	log.Debug("Prepare payload", "height", position.Height)

	txsMap, err := d.txPool.Pending()
	if err != nil {
		return
	}

	blockGasLimit := new(big.Int).SetUint64(d.gov.DexconConfiguration(position.Round).BlockGasLimit)
	blockGasUsed := new(big.Int)
	allTxs := make([]*types.Transaction, 0, 10000)

addressMap:
	for address, txs := range txsMap {
		select {
		case <-ctx.Done():
			break addressMap
		default:
		}

		balance := state.GetBalance(address)
		cost, exist := d.addressCost[address]
		if exist {
			balance = new(big.Int).Sub(balance, cost)
		}

		var expectNonce uint64
		lastConfirmedNonce, exist := d.addressNonce[address]
		if !exist {
			expectNonce = state.GetNonce(address)
		} else {
			expectNonce = lastConfirmedNonce + 1
		}

		if len(txs) == 0 {
			continue
		}

		firstNonce := txs[0].Nonce()
		startIndex := int(expectNonce - firstNonce)

		// Warning: the pending tx will also affect by syncing, so startIndex maybe negative
		for i := startIndex; i >= 0 && i < len(txs); i++ {
			tx := txs[i]
			intrGas, err := core.IntrinsicGas(tx.Data(), tx.To() == nil, true)
			if err != nil {
				log.Error("Failed to calculate intrinsic gas", "error", err)
				return nil, fmt.Errorf("calculate intrinsic gas error: %v", err)
			}
			if tx.Gas() < intrGas {
				log.Error("Intrinsic gas too low", "txHash", tx.Hash().String())
				break
			}

			balance = new(big.Int).Sub(balance, tx.Cost())
			if balance.Cmp(big.NewInt(0)) < 0 {
				log.Warn("Insufficient funds for gas * price + value", "txHash", tx.Hash().String())
				break
			}

			blockGasUsed = new(big.Int).Add(blockGasUsed, big.NewInt(int64(tx.Gas())))
			if blockGasUsed.Cmp(blockGasLimit) > 0 {
				break addressMap
			}

			allTxs = append(allTxs, tx)
		}
	}

	return rlp.EncodeToBytes(&allTxs)
}

// PrepareWitness will return the witness data no lower than consensusHeight.
func (d *DexconApp) PrepareWitness(consensusHeight uint64) (witness coreTypes.Witness, err error) {
	var witnessBlock *types.Block
	if d.blockchain.CurrentBlock().NumberU64() >= consensusHeight {
		witnessBlock = d.blockchain.CurrentBlock()
	} else {
		log.Error("Current height too low", "lastPendingHeight", d.blockchain.CurrentBlock().NumberU64(),
			"consensusHeight", consensusHeight)
		return witness, fmt.Errorf("current height < consensus height")
	}

	witnessData, err := rlp.EncodeToBytes(witnessBlock.Hash())
	if err != nil {
		return
	}

	return coreTypes.Witness{
		Height: witnessBlock.NumberU64(),
		Data:   witnessData,
	}, nil
}

// VerifyBlock verifies if the payloads are valid.
func (d *DexconApp) VerifyBlock(block *coreTypes.Block) coreTypes.BlockVerifyStatus {
	var witnessBlockHash common.Hash
	err := rlp.DecodeBytes(block.Witness.Data, &witnessBlockHash)
	if err != nil {
		log.Error("Failed to RLP decode witness data", "error", err)
		return coreTypes.VerifyInvalidBlock
	}

	// Validate witness height.
	if d.blockchain.CurrentBlock().NumberU64() < block.Witness.Height {
		log.Debug("Current height < witness height")
		return coreTypes.VerifyRetryLater
	}

	b := d.blockchain.GetBlockByNumber(block.Witness.Height)
	if b == nil {
		log.Error("Can not get block by height", "height", block.Witness.Height)
		return coreTypes.VerifyInvalidBlock
	}

	if b.Hash() != witnessBlockHash {
		log.Error("Witness block hash not match",
			"expect", b.Hash().String(), "got", witnessBlockHash.String())
		return coreTypes.VerifyInvalidBlock
	}

	_, err = d.blockchain.StateAt(b.Root())
	if err != nil {
		log.Error("Get state by root %v error: %v", b.Root(), err)
		return coreTypes.VerifyInvalidBlock
	}

	d.appMu.RLock()
	defer d.appMu.RUnlock()

	// deliver height = position height + 1
	if d.deliveredHeight+d.undeliveredNum != block.Position.Height {
		return coreTypes.VerifyRetryLater
	}

	var transactions types.Transactions
	if len(block.Payload) == 0 {
		return coreTypes.VerifyOK
	}

	deliveredBlock := d.blockchain.GetBlockByNumber(d.deliveredHeight)
	state, err := d.blockchain.StateAt(deliveredBlock.Root())
	if err != nil {
		return coreTypes.VerifyInvalidBlock
	}

	err = rlp.DecodeBytes(block.Payload, &transactions)
	if err != nil {
		log.Error("Payload rlp decode", "error", err)
		return coreTypes.VerifyInvalidBlock
	}

	_, err = types.GlobalSigCache.Add(types.NewEIP155Signer(d.blockchain.Config().ChainID), transactions)
	if err != nil {
		log.Error("Failed to calculate sender", "error", err)
		return coreTypes.VerifyInvalidBlock
	}

	addressNonce, err := d.validateNonce(transactions)
	if err != nil {
		log.Error("Validate nonce failed", "error", err)
		return coreTypes.VerifyInvalidBlock
	}
	if !d.validateGasPrice(transactions, block.Position.Round) {
		log.Error("Validate gas price failed")
		return coreTypes.VerifyInvalidBlock
	}

	for address, firstNonce := range addressNonce {
		var expectNonce uint64
		nonce, exist := d.addressNonce[address]
		if exist {
			expectNonce = nonce + 1
		} else {
			expectNonce = state.GetNonce(address)
		}

		if expectNonce != firstNonce {
			log.Error("Nonce check error", "expect", expectNonce, "firstNonce", firstNonce)
			return coreTypes.VerifyInvalidBlock
		}
	}

	// Calculate balance in last state (including pending state).
	addressesBalance := map[common.Address]*big.Int{}
	for address := range addressNonce {
		cost, exist := d.addressCost[address]
		if exist {
			addressesBalance[address] = new(big.Int).Sub(state.GetBalance(address), cost)
		} else {
			addressesBalance[address] = state.GetBalance(address)
		}
	}

	// Validate if balance is enough for TXs in this block.
	blockGasLimit := new(big.Int).SetUint64(d.gov.DexconConfiguration(block.Position.Round).BlockGasLimit)
	blockGasUsed := new(big.Int)

	for _, tx := range transactions {
		msg, err := tx.AsMessage(types.MakeSigner(d.blockchain.Config(), new(big.Int)))
		if err != nil {
			log.Error("Failed to convert tx to message", "error", err)
			return coreTypes.VerifyInvalidBlock
		}
		balance := addressesBalance[msg.From()]
		intrGas, err := core.IntrinsicGas(msg.Data(), msg.To() == nil, true)
		if err != nil {
			log.Error("Failed to calculate intrinsic gas", "err", err)
			return coreTypes.VerifyInvalidBlock
		}
		if tx.Gas() < intrGas {
			log.Error("Intrinsic gas too low", "txHash", tx.Hash().String(), "intrinsic", intrGas, "gas", tx.Gas())
			return coreTypes.VerifyInvalidBlock
		}

		balance = new(big.Int).Sub(balance, tx.Cost())
		if balance.Cmp(big.NewInt(0)) < 0 {
			log.Error("Insufficient funds for gas * price + value", "txHash", tx.Hash().String())
			return coreTypes.VerifyInvalidBlock
		}

		blockGasUsed = new(big.Int).Add(blockGasUsed, new(big.Int).SetUint64(tx.Gas()))
		if blockGasUsed.Cmp(blockGasLimit) > 0 {
			log.Error("Reach block gas limit", "gasUsed", blockGasUsed)
			return coreTypes.VerifyInvalidBlock
		}
		addressesBalance[msg.From()] = balance
	}

	return coreTypes.VerifyOK
}

// BlockDelivered is called when a block is add to the compaction chain.
func (d *DexconApp) BlockDelivered(
	blockHash coreCommon.Hash,
	blockPosition coreTypes.Position,
	result coreTypes.FinalizationResult) {

	log.Debug("DexconApp block deliver", "height", result.Height, "hash", blockHash, "position", blockPosition.String())
	defer log.Debug("DexconApp block delivered", "height", result.Height, "hash", blockHash, "position", blockPosition.String())

	d.appMu.Lock()
	defer d.appMu.Unlock()

	block, txs := d.getConfirmedBlockByHash(blockHash)
	if block == nil {
		panic("Can not get confirmed block")
	}

	block.Payload = nil
	block.Finalization = result
	dexconMeta, err := rlp.EncodeToBytes(block)
	if err != nil {
		panic(err)
	}

	var owner common.Address
	if !block.IsEmpty() {
		gs := d.gov.GetStateForConfigAtRound(block.Position.Round)
		node, err := gs.GetNodeByID(block.ProposerID)
		if err != nil {
			panic(err)
		}
		owner = node.Owner
	}

	newBlock := types.NewBlock(&types.Header{
		Number:     new(big.Int).SetUint64(result.Height),
		Time:       big.NewInt(result.Timestamp.UnixNano() / 1000000),
		Coinbase:   owner,
		GasLimit:   d.gov.DexconConfiguration(block.Position.Round).BlockGasLimit,
		Difficulty: big.NewInt(1),
		Round:      block.Position.Round,
		DexconMeta: dexconMeta,
		Randomness: result.Randomness,
	}, txs, nil, nil)

	if block.IsEmpty() {
		_, err = d.blockchain.ProcessEmptyBlock(newBlock)
		if err != nil {
			log.Error("Failed to process empty block", "error", err)
			panic(err)
		}
	} else {
		_, err = d.blockchain.ProcessBlock(newBlock, &block.Witness)
		if err != nil {
			log.Error("Failed to process pending block", "error", err)
			panic(err)
		}
	}

	d.removeConfirmedBlock(blockHash)
	d.deliveredHeight = result.Height

	// New blocks are finalized, notify other components.
	go d.finalizedBlockFeed.Send(core.NewFinalizedBlockEvent{Block: d.blockchain.CurrentBlock()})
}

// BlockConfirmed is called when a block is confirmed.
func (d *DexconApp) BlockConfirmed(block coreTypes.Block) {
	d.appMu.Lock()
	defer d.appMu.Unlock()

	log.Debug("DexconApp block confirmed", "block", block.String())
	if err := d.addConfirmedBlock(&block); err != nil {
		panic(err)
	}
}

type addressInfo struct {
	cost *big.Int
}

type blockInfo struct {
	addresses map[common.Address]*addressInfo
	block     *coreTypes.Block
	txs       types.Transactions
}

func (d *DexconApp) addConfirmedBlock(block *coreTypes.Block) error {
	var transactions types.Transactions
	if len(block.Payload) != 0 {
		err := rlp.Decode(bytes.NewReader(block.Payload), &transactions)
		if err != nil {
			return err
		}
		_, err = types.GlobalSigCache.Add(types.NewEIP155Signer(d.blockchain.Config().ChainID), transactions)
		if err != nil {
			return err
		}
	}

	addressMap := map[common.Address]*addressInfo{}
	for _, tx := range transactions {
		msg, err := tx.AsMessage(types.MakeSigner(d.blockchain.Config(), new(big.Int)))
		if err != nil {
			return err
		}

		if addrInfo, exist := addressMap[msg.From()]; !exist {
			addressMap[msg.From()] = &addressInfo{cost: tx.Cost()}
		} else {
			addrInfo.cost = new(big.Int).Add(addrInfo.cost, tx.Cost())
		}

		// get latest nonce in block
		if nonce, exist := d.addressNonce[msg.From()]; !exist || nonce < msg.Nonce() {
			d.addressNonce[msg.From()] = msg.Nonce()
		} else {
			return fmt.Errorf("address %v nonce incorrect cached(%d) >= tx(%d)", msg.From(), nonce, msg.Nonce())
		}

		// calculate max cost in confirmed blocks
		if d.addressCost[msg.From()] == nil {
			d.addressCost[msg.From()] = big.NewInt(0)
		}
		d.addressCost[msg.From()] = new(big.Int).Add(d.addressCost[msg.From()], tx.Cost())
	}

	for addr := range addressMap {
		d.addressCounter[addr]++
	}

	d.confirmedBlocks[block.Hash] = &blockInfo{
		addresses: addressMap,
		block:     block,
		txs:       transactions,
	}

	d.undeliveredNum++
	return nil
}

func (d *DexconApp) removeConfirmedBlock(hash coreCommon.Hash) {
	blockInfo := d.confirmedBlocks[hash]
	for addr, info := range blockInfo.addresses {
		d.addressCounter[addr]--
		d.addressCost[addr] = new(big.Int).Sub(d.addressCost[addr], info.cost)
		if d.addressCounter[addr] == 0 {
			delete(d.addressCounter, addr)
			delete(d.addressCost, addr)
			delete(d.addressNonce, addr)
		}
	}

	delete(d.confirmedBlocks, hash)
	d.undeliveredNum--
}

func (d *DexconApp) getConfirmedBlockByHash(hash coreCommon.Hash) (*coreTypes.Block, types.Transactions) {
	info, exist := d.confirmedBlocks[hash]
	if !exist {
		return nil, nil
	}

	return info.block, info.txs
}

func (d *DexconApp) SubscribeNewFinalizedBlockEvent(
	ch chan<- core.NewFinalizedBlockEvent) event.Subscription {
	return d.scope.Track(d.finalizedBlockFeed.Subscribe(ch))
}

func (d *DexconApp) Stop() {
	d.scope.Close()
}
