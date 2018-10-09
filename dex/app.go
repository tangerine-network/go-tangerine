// Copyright 2018 The dexon-consensus-core Authors
// This file is part of the dexon-consensus-core library.
//
// The dexon-consensus-core library is free software: you can redistribute it
// and/or modify it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The dexon-consensus-core library is distributed in the hope that it will be
// useful, but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the dexon-consensus-core library. If not, see
// <http://www.gnu.org/licenses/>.

package dex

import (
	"bytes"
	"fmt"
	"math/big"
	"sync"
	"time"

	coreCommon "github.com/dexon-foundation/dexon-consensus-core/common"
	coreTypes "github.com/dexon-foundation/dexon-consensus-core/core/types"

	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/core"
	"github.com/dexon-foundation/dexon/core/state"
	"github.com/dexon-foundation/dexon/core/types"
	"github.com/dexon-foundation/dexon/core/vm"
	"github.com/dexon-foundation/dexon/ethdb"
	"github.com/dexon-foundation/dexon/log"
	"github.com/dexon-foundation/dexon/rlp"
)

// DexconApp implementes the DEXON consensus core application interface.
type DexconApp struct {
	txPool     *core.TxPool
	blockchain *core.BlockChain
	gov        *DexconGovernance
	chainDB    ethdb.Database
	config     *Config
	vmConfig   vm.Config

	notifyChan map[uint64]*notify
	notifyMu   *sync.Mutex

	lastPendingHeight uint64
	insertMu          sync.Mutex

	chainLocksInitMu *sync.Mutex
	chainLocks       map[uint32]*sync.Mutex
}

type notify struct {
	results []chan uint64
}

type witnessData struct {
	Root        common.Hash
	TxHash      common.Hash
	ReceiptHash common.Hash
}

func NewDexconApp(txPool *core.TxPool, blockchain *core.BlockChain, gov *DexconGovernance, chainDB ethdb.Database, config *Config, vmConfig vm.Config) *DexconApp {
	return &DexconApp{
		txPool:           txPool,
		blockchain:       blockchain,
		gov:              gov,
		chainDB:          chainDB,
		config:           config,
		vmConfig:         vmConfig,
		notifyChan:       make(map[uint64]*notify),
		notifyMu:         &sync.Mutex{},
		chainLocksInitMu: &sync.Mutex{},
		chainLocks:       make(map[uint32]*sync.Mutex),
	}
}

func (d *DexconApp) addNotify(height uint64) <-chan uint64 {
	d.notifyMu.Lock()
	defer d.notifyMu.Unlock()
	result := make(chan uint64)
	if n, exist := d.notifyChan[height]; exist {
		n.results = append(n.results, result)
	} else {
		d.notifyChan[height] = &notify{}
		d.notifyChan[height].results = append(d.notifyChan[height].results, result)
	}
	return result
}

func (d *DexconApp) notify(height uint64) {
	d.notifyMu.Lock()
	defer d.notifyMu.Unlock()
	for h, n := range d.notifyChan {
		if height >= h {
			for _, ch := range n.results {
				ch <- height
			}
			delete(d.notifyChan, h)
		}
	}
	d.lastPendingHeight = height
}

func (d *DexconApp) checkChain(address common.Address, chainSize, chainID *big.Int) bool {
	addrModChainSize := new(big.Int)
	return addrModChainSize.Mod(address.Big(), chainSize).Cmp(chainID) == 0
}

// PreparePayload is called when consensus core is preparing payload for block.
func (d *DexconApp) PreparePayload(position coreTypes.Position) (payload []byte, err error) {
	d.chainLock(position.ChainID)
	defer d.chainUnlock(position.ChainID)

	if position.Height != 0 {
		// check if chain block height is sequential
		chainLastHeight := d.blockchain.GetChainLastConfirmedHeight(position.ChainID)
		if chainLastHeight != position.Height-1 {
			log.Error("Check confirmed block height fail", "chain", position.ChainID, "height", position.Height-1, "cache height", chainLastHeight)
			return nil, fmt.Errorf("check confirmed block height fail")
		}
	}

	// set state to the pending height
	var latestState *state.StateDB
	if d.lastPendingHeight == 0 {
		latestState, err = d.blockchain.State()
		if err != nil {
			log.Error("Get current state", "error", err)
			return nil, fmt.Errorf("get current state error %v", err)
		}
	} else {
		latestState, err = d.blockchain.StateAt(d.blockchain.GetPendingBlockByHeight(d.lastPendingHeight).Root())
		if err != nil {
			log.Error("Get pending state", "error", err)
			return nil, fmt.Errorf("get pending state error: %v", err)
		}
	}

	txsMap, err := d.txPool.Pending()
	if err != nil {
		return
	}

	chainID := new(big.Int).SetUint64(uint64(position.ChainID))
	chainNums := new(big.Int).SetUint64(uint64(d.gov.GetNumChains(position.Round)))
	blockGasLimit := new(big.Int).SetUint64(core.CalcGasLimit(d.blockchain.CurrentBlock(), d.config.GasFloor, d.config.GasCeil))
	blockGasUsed := new(big.Int)
	var allTxs types.Transactions

addressMap:
	for address, txs := range txsMap {
		// every address's transactions will appear in fixed chain
		if !d.checkChain(address, chainNums, chainID) {
			continue
		}

		var expectNonce uint64
		// get last nonce from confirmed blocks
		lastConfirmedNonce, exist := d.blockchain.GetLastNonceInConfirmedBlocks(address)
		if !exist {
			// get expect nonce from latest state when confirmed block is empty
			expectNonce = latestState.GetNonce(address)
		} else {
			expectNonce = lastConfirmedNonce + 1
		}

		if expectNonce != txs[0].Nonce() {
			log.Debug("Nonce check error", "expect", expectNonce, "nonce", txs[0].Nonce())
			continue
		}

		balance := latestState.GetBalance(address)
		cost, exist := d.blockchain.GetCostInConfirmedBlocks(address)
		if exist {
			balance = new(big.Int).Sub(balance, cost)
		}

		for _, tx := range txs {
			maxGasUsed := new(big.Int).Mul(new(big.Int).SetUint64(tx.Gas()), tx.GasPrice())
			intrinsicGas, err := core.IntrinsicGas(tx.Data(), tx.To() == nil, true)
			if err != nil {
				log.Error("Calculate intrinsic gas fail", "err", err)
				return nil, fmt.Errorf("calculate intrinsic gas error: %v", err)
			}
			if big.NewInt(int64(intrinsicGas)).Cmp(maxGasUsed) > 0 {
				log.Warn("Intrinsic gas is larger than (gas limit * gas price)", "intrinsic", intrinsicGas, "maxGasUsed", maxGasUsed)
				break
			}

			balance = new(big.Int).Sub(balance, tx.Cost())
			if balance.Cmp(big.NewInt(0)) < 0 {
				log.Error("Tx fail", "reason", "not enough balance")
				break
			}

			blockGasUsed = new(big.Int).Add(blockGasUsed, new(big.Int).SetUint64(tx.Gas()))
			if blockGasLimit.Cmp(blockGasUsed) < 0 {
				break addressMap
			}

			allTxs = append(allTxs, tx)
		}
	}
	payload, err = rlp.EncodeToBytes(&allTxs)
	if err != nil {
		return
	}

	return
}

// PrepareWitness will return the witness data no lower than consensusHeight.
func (d *DexconApp) PrepareWitness(consensusHeight uint64) (witness coreTypes.Witness, err error) {
	var witnessBlock *types.Block
	if d.lastPendingHeight == 0 && consensusHeight == 0 {
		witnessBlock = d.blockchain.CurrentBlock()
	} else if d.lastPendingHeight >= consensusHeight {
		d.insertMu.Lock()
		witnessBlock = d.blockchain.GetPendingBlockByHeight(d.lastPendingHeight)
		d.insertMu.Unlock()
	} else if h := <-d.addNotify(consensusHeight); h >= consensusHeight {
		d.insertMu.Lock()
		witnessBlock = d.blockchain.GetPendingBlockByHeight(h)
		d.insertMu.Unlock()
	} else {
		log.Error("need pending block")
		return witness, fmt.Errorf("need pending block")
	}

	witnessData, err := rlp.EncodeToBytes(&witnessData{
		Root:        witnessBlock.Root(),
		TxHash:      witnessBlock.TxHash(),
		ReceiptHash: witnessBlock.ReceiptHash(),
	})
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
	var witnessData witnessData
	err := rlp.Decode(bytes.NewReader(block.Witness.Data), &witnessData)
	if err != nil {
		log.Error("Witness rlp decode", "error", err)
		return coreTypes.VerifyInvalidBlock
	}

	for i := 0; i < 6 && err != nil; i++ {
		// check witness root exist
		err = nil
		_, err = d.blockchain.StateAt(witnessData.Root)
		if err != nil {
			log.Debug("Sleep 0.5 seconds and try again", "error", err)
			time.Sleep(500 * time.Millisecond)
		}
	}
	if err != nil {
		log.Error("Expect witness root not in stateDB", "err", err)
		return coreTypes.VerifyRetryLater
	}

	d.chainLock(block.Position.ChainID)
	defer d.chainUnlock(block.Position.ChainID)

	if block.Position.Height != 0 {
		// check if chain block height is sequential
		chainLastHeight := d.blockchain.GetChainLastConfirmedHeight(block.Position.ChainID)
		if chainLastHeight != block.Position.Height-1 {
			log.Error("Check confirmed block height fail", "chain", block.Position.ChainID, "height", block.Position.Height-1, "cache height", chainLastHeight)
			return coreTypes.VerifyRetryLater
		}
	}

	// set state to the pending height
	var latestState *state.StateDB
	if d.lastPendingHeight == 0 {
		latestState, err = d.blockchain.State()
		if err != nil {
			log.Error("Get current state", "error", err)
			return coreTypes.VerifyInvalidBlock
		}
	} else {
		latestState, err = d.blockchain.StateAt(d.blockchain.GetPendingBlockByHeight(d.lastPendingHeight).Root())
		if err != nil {
			log.Error("Get pending state", "error", err)
			return coreTypes.VerifyInvalidBlock
		}
	}

	var transactions types.Transactions
	err = rlp.Decode(bytes.NewReader(block.Payload), &transactions)
	if err != nil {
		log.Error("Payload rlp decode", "error", err)
		return coreTypes.VerifyInvalidBlock
	}

	// check if nonce is sequential and return first nonce of every address
	addresses, err := d.validateNonce(transactions)
	if err != nil {
		log.Error("Get address nonce", "error", err)
		return coreTypes.VerifyInvalidBlock
	}

	// check all address nonce
	chainID := big.NewInt(int64(block.Position.ChainID))
	chainNums := new(big.Int).SetUint64(uint64(d.gov.GetNumChains(block.Position.Round)))
	for address, firstNonce := range addresses {
		if !d.checkChain(address, chainNums, chainID) {
			log.Error("check chain fail", "address", address)
			return coreTypes.VerifyInvalidBlock
		}

		var expectNonce uint64
		// get last nonce from confirmed blocks
		lastConfirmedNonce, exist := d.blockchain.GetLastNonceInConfirmedBlocks(address)
		if !exist {
			// get expect nonce from latest state when confirmed block is empty
			expectNonce = latestState.GetNonce(address)
		} else {
			expectNonce = lastConfirmedNonce + 1
		}

		if expectNonce != firstNonce {
			log.Error("Nonce check error", "expect", expectNonce, "firstNonce", firstNonce)
			return coreTypes.VerifyInvalidBlock
		}
	}

	// get balance from state
	addressesBalance := map[common.Address]*big.Int{}
	for address := range addresses {
		// replay confirmed block tx to correct balance
		cost, exist := d.blockchain.GetCostInConfirmedBlocks(address)
		if exist {
			addressesBalance[address] = new(big.Int).Sub(latestState.GetBalance(address), cost)
		} else {
			addressesBalance[address] = latestState.GetBalance(address)
		}
	}

	// validate tx to check available balance
	blockGasLimit := new(big.Int).SetUint64(core.CalcGasLimit(d.blockchain.CurrentBlock(), d.config.GasFloor, d.config.GasCeil))
	blockGasUsed := new(big.Int)
	for _, tx := range transactions {
		msg, err := tx.AsMessage(types.MakeSigner(d.blockchain.Config(), new(big.Int)))
		if err != nil {
			log.Error("Tx to message", "error", err)
			return coreTypes.VerifyInvalidBlock
		}
		balance, _ := addressesBalance[msg.From()]
		maxGasUsed := new(big.Int).Mul(new(big.Int).SetUint64(msg.Gas()), msg.GasPrice())
		intrinsicGas, err := core.IntrinsicGas(msg.Data(), msg.To() == nil, true)
		if err != nil {
			log.Error("Calculate intrinsic gas fail", "err", err)
			return coreTypes.VerifyInvalidBlock
		}
		if big.NewInt(int64(intrinsicGas)).Cmp(maxGasUsed) > 0 {
			log.Error("Intrinsic gas is larger than (gas limit * gas price)", "intrinsic", intrinsicGas, "maxGasUsed", maxGasUsed)
			return coreTypes.VerifyInvalidBlock
		}

		balance = new(big.Int).Sub(balance, tx.Cost())
		if balance.Cmp(big.NewInt(0)) < 0 {
			log.Error("Tx fail", "reason", "not enough balance")
			return coreTypes.VerifyInvalidBlock
		}

		blockGasUsed = new(big.Int).Add(blockGasUsed, new(big.Int).SetUint64(tx.Gas()))
		if blockGasLimit.Cmp(blockGasUsed) < 0 {
			log.Error("Reach block gas limit", "limit", blockGasLimit)
			return coreTypes.VerifyInvalidBlock
		}

		addressesBalance[msg.From()] = balance
	}

	return coreTypes.VerifyOK
}

// BlockDelivered is called when a block is add to the compaction chain.
func (d *DexconApp) BlockDelivered(blockHash coreCommon.Hash, result coreTypes.FinalizationResult) {
	d.insertMu.Lock()
	defer d.insertMu.Unlock()

	block := d.blockchain.GetConfirmedBlockByHash(blockHash)
	if block == nil {
		panic("Can not get confirmed block")
	}

	d.chainLock(block.Position.ChainID)
	defer d.chainUnlock(block.Position.ChainID)

	var transactions types.Transactions
	err := rlp.Decode(bytes.NewReader(block.Payload), &transactions)
	if err != nil {
		log.Error("Payload rlp decode failed", "error", err)
		panic(err)
	}

	block.Payload = nil
	dexconMeta, err := rlp.EncodeToBytes(block)
	if err != nil {
		panic(err)
	}

	newBlock := types.NewBlock(&types.Header{
		Number:   new(big.Int).SetUint64(result.Height),
		Time:     big.NewInt(result.Timestamp.Unix()),
		Coinbase: common.BytesToAddress(block.ProposerID.Bytes()),
		// TODO(bojie): fix it
		GasLimit:   8000000,
		Difficulty: big.NewInt(1),
		Round:      block.Position.Round,
		DexconMeta: dexconMeta,
		Randomness: result.Randomness,
	}, transactions, nil, nil)

	_, err = d.blockchain.ProcessPendingBlock(newBlock, &block.Witness)
	if err != nil {
		log.Error("Insert chain", "error", err)
		panic(err)
	}

	log.Info("Insert pending block success", "height", result.Height)
	d.blockchain.RemoveConfirmedBlock(blockHash)
	d.notify(result.Height)
}

// BlockConfirmed is called when a block is confirmed and added to lattice.
func (d *DexconApp) BlockConfirmed(block coreTypes.Block) {
	d.chainLock(block.Position.ChainID)
	defer d.chainUnlock(block.Position.ChainID)

	d.blockchain.AddConfirmedBlock(&block)
}

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
				return nil, fmt.Errorf("address nonce check error: expect %v actual %v", addressNonce[msg.From()]+1, msg.Nonce())
			}
			addressNonce[msg.From()] = msg.Nonce()
			continue
		} else {
			addressNonce[msg.From()] = msg.Nonce()
			addressFirstNonce[msg.From()] = msg.Nonce()
		}
	}

	return addressFirstNonce, nil
}

func (d *DexconApp) chainLock(chainID uint32) {
	d.chainLocksInitMu.Lock()
	_, exist := d.chainLocks[chainID]
	if !exist {
		d.chainLocks[chainID] = &sync.Mutex{}
	}
	d.chainLocksInitMu.Unlock()

	d.chainLocks[chainID].Lock()
}

func (d *DexconApp) chainUnlock(chainID uint32) {
	d.chainLocks[chainID].Unlock()
}
