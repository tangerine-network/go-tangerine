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
	"fmt"
	"math/big"
	"sync"
	"time"

	coreCommon "github.com/dexon-foundation/dexon-consensus/common"
	coreTypes "github.com/dexon-foundation/dexon-consensus/core/types"

	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/core"
	"github.com/dexon-foundation/dexon/core/types"
	"github.com/dexon-foundation/dexon/core/vm"
	"github.com/dexon-foundation/dexon/ethdb"
	"github.com/dexon-foundation/dexon/log"
	"github.com/dexon-foundation/dexon/rlp"
)

const (
	verifyBlockMaxRetries = 4
)

// DexconApp implementes the DEXON consensus core application interface.
type DexconApp struct {
	txPool     *core.TxPool
	blockchain *core.BlockChain
	gov        *DexconGovernance
	chainDB    ethdb.Database
	config     *Config
	vmConfig   vm.Config

	chainLocksInitMu sync.Mutex
	chainLocks       map[uint32]*sync.RWMutex

	notifyMu        sync.Mutex
	notifyChan      sync.Map
	chainLatestRoot sync.Map
}

type notify struct {
	results []chan uint64
}

type witnessData struct {
	Root        common.Hash
	TxHash      common.Hash
	ReceiptHash common.Hash
}

func NewDexconApp(txPool *core.TxPool, blockchain *core.BlockChain, gov *DexconGovernance,
	chainDB ethdb.Database, config *Config, vmConfig vm.Config) *DexconApp {
	return &DexconApp{
		txPool:     txPool,
		blockchain: blockchain,
		gov:        gov,
		chainDB:    chainDB,
		config:     config,
		vmConfig:   vmConfig,
		chainLocks: make(map[uint32]*sync.RWMutex),
	}
}

func (d *DexconApp) addrBelongsToChain(address common.Address, chainSize, chainID *big.Int) bool {
	return new(big.Int).Mod(address.Big(), chainSize).Cmp(chainID) == 0
}

func (d *DexconApp) chainLockInit(chainID uint32) {
	d.chainLocksInitMu.Lock()
	defer d.chainLocksInitMu.Unlock()

	_, exist := d.chainLocks[chainID]
	if !exist {
		d.chainLocks[chainID] = &sync.RWMutex{}
	}
}

func (d *DexconApp) chainLock(chainID uint32) {
	d.chainLockInit(chainID)
	d.chainLocks[chainID].Lock()
}

func (d *DexconApp) chainUnlock(chainID uint32) {
	d.chainLocks[chainID].Unlock()
}

func (d *DexconApp) chainRLock(chainID uint32) {
	d.chainLockInit(chainID)
	d.chainLocks[chainID].RLock()
}

func (d *DexconApp) chainRUnlock(chainID uint32) {
	d.chainLocks[chainID].RUnlock()
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

// PreparePayload is called when consensus core is preparing payload for block.
func (d *DexconApp) PreparePayload(position coreTypes.Position) (payload []byte, err error) {
	d.chainRLock(position.ChainID)
	defer d.chainRUnlock(position.ChainID)

	if position.Height != 0 {
		// Check if chain block height is strictly increamental.
		chainLastHeight := d.blockchain.GetChainLastConfirmedHeight(position.ChainID)
		if chainLastHeight != position.Height-1 {
			log.Error("Check confirmed block height fail",
				"chain", position.ChainID, "height", position.Height-1, "cache height", chainLastHeight)
			return nil, fmt.Errorf("check confirmed block height fail")
		}
	}

	var root *common.Hash
	value, ok := d.chainLatestRoot.Load(position.ChainID)
	if ok {
		root = value.(*common.Hash)
	} else {
		currentRoot := d.blockchain.CurrentBlock().Root()
		root = &currentRoot
	}

	// Set state to the chain latest height.
	latestState, err := d.blockchain.StateAt(*root)
	if err != nil {
		log.Error("Get pending state", "error", err)
		return nil, fmt.Errorf("get pending state error: %v", err)
	}

	txsMap, err := d.txPool.Pending()
	if err != nil {
		return
	}

	chainID := new(big.Int).SetUint64(uint64(position.ChainID))
	chainNums := new(big.Int).SetUint64(uint64(d.gov.GetNumChains(position.Round)))
	blockGasLimit := new(big.Int).SetUint64(d.blockchain.CurrentBlock().GasLimit())
	blockGasUsed := new(big.Int)
	allTxs := make([]*types.Transaction, 0, 3000)

addressMap:
	for address, txs := range txsMap {
		// TX hash need to be slot to the given chain in order to be included in the block.
		if !d.addrBelongsToChain(address, chainNums, chainID) {
			continue
		}

		balance := latestState.GetBalance(address)
		cost, exist := d.blockchain.GetCostInConfirmedBlocks(position.ChainID, address)
		if exist {
			balance = new(big.Int).Sub(balance, cost)
		}

		var expectNonce uint64
		lastConfirmedNonce, exist := d.blockchain.GetLastNonceInConfirmedBlocks(position.ChainID, address)
		if !exist {
			expectNonce = latestState.GetNonce(address)
		} else {
			expectNonce = lastConfirmedNonce + 1
		}

		if len(txs) == 0 {
			continue
		}

		firstNonce := txs[0].Nonce()
		startIndex := int(expectNonce - firstNonce)

		for i := startIndex; i < len(txs); i++ {
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
	lastPendingHeight := d.blockchain.GetPendingHeight()
	if lastPendingHeight == 0 && consensusHeight == 0 {
		witnessBlock = d.blockchain.CurrentBlock()
	} else if lastPendingHeight >= consensusHeight {
		witnessBlock = d.blockchain.PendingBlock()
	} else {
		log.Error("last pending height too low", "lastPendingHeight", lastPendingHeight,
			"consensusHeight", consensusHeight)
		return witness, fmt.Errorf("last pending height < consensus height")
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
	err := rlp.DecodeBytes(block.Witness.Data, &witnessData)
	if err != nil {
		log.Error("failed to RLP decode witness data", "error", err)
		return coreTypes.VerifyInvalidBlock
	}

	// Wait until the witnessed root is seen on our local chain.
	for i := 0; i < verifyBlockMaxRetries && err != nil; i++ {
		_, err = d.blockchain.StateAt(witnessData.Root)
		if err != nil {
			log.Debug("Witness root not found, retry in 500ms", "error", err)
			time.Sleep(500 * time.Millisecond)
		}
	}

	if err != nil {
		log.Error("Expected witness root not in stateDB", "err", err)
		return coreTypes.VerifyRetryLater
	}

	d.chainRLock(block.Position.ChainID)
	defer d.chainRUnlock(block.Position.ChainID)

	if block.Position.Height != 0 {
		// Check if target block is the next height to be verified, we can only
		// verify the next block in a given chain.
		chainLastHeight := d.blockchain.GetChainLastConfirmedHeight(block.Position.ChainID)
		if chainLastHeight != block.Position.Height-1 {
			log.Error("Check confirmed block height fail", "chain", block.Position.ChainID,
				"height", block.Position.Height-1, "cache height", chainLastHeight)
			return coreTypes.VerifyRetryLater
		}
	}

	// Find the latest state for the given block height.
	var root *common.Hash
	value, ok := d.chainLatestRoot.Load(block.Position.ChainID)
	if ok {
		root = value.(*common.Hash)
	} else {
		currentRoot := d.blockchain.CurrentBlock().Root()
		root = &currentRoot
	}

	latestState, err := d.blockchain.StateAt(*root)
	if err != nil {
		log.Error("Failed to get pending state", "error", err)
		return coreTypes.VerifyInvalidBlock
	}

	var transactions types.Transactions
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

	// Check if nonce is strictly increasing for every address.
	chainID := big.NewInt(int64(block.Position.ChainID))
	chainNums := big.NewInt(int64(d.gov.GetNumChains(block.Position.Round)))

	for address, firstNonce := range addressNonce {
		if !d.addrBelongsToChain(address, chainNums, chainID) {
			log.Error("Address does not belong to given chain ID", "address", address, "chainD", chainID)
			return coreTypes.VerifyInvalidBlock
		}

		var expectNonce uint64
		lastConfirmedNonce, exist := d.blockchain.GetLastNonceInConfirmedBlocks(block.Position.ChainID, address)
		if exist {
			expectNonce = lastConfirmedNonce + 1
		} else {
			expectNonce = latestState.GetNonce(address)
		}

		if expectNonce != firstNonce {
			log.Error("Nonce check error", "expect", expectNonce, "firstNonce", firstNonce)
			return coreTypes.VerifyInvalidBlock
		}
	}

	// Calculate balance in last state (including pending state).
	addressesBalance := map[common.Address]*big.Int{}
	for address := range addressNonce {
		cost, exist := d.blockchain.GetCostInConfirmedBlocks(block.Position.ChainID, address)
		if exist {
			addressesBalance[address] = new(big.Int).Sub(latestState.GetBalance(address), cost)
		} else {
			addressesBalance[address] = latestState.GetBalance(address)
		}
	}

	// Validate if balance is enough for TXs in this block.
	blockGasLimit := new(big.Int).SetUint64(d.blockchain.CurrentBlock().GasLimit())
	blockGasUsed := new(big.Int)

	for _, tx := range transactions {
		msg, err := tx.AsMessage(types.MakeSigner(d.blockchain.Config(), new(big.Int)))
		if err != nil {
			log.Error("Failed to convert tx to message", "error", err)
			return coreTypes.VerifyInvalidBlock
		}
		balance, _ := addressesBalance[msg.From()]
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

	chainID := blockPosition.ChainID
	d.chainLock(chainID)
	defer d.chainUnlock(chainID)

	block, txs := d.blockchain.GetConfirmedBlockByHash(chainID, blockHash)
	if block == nil {
		panic("Can not get confirmed block")
	}

	block.Payload = nil
	dexconMeta, err := rlp.EncodeToBytes(block)
	if err != nil {
		panic(err)
	}

	newBlock := types.NewBlock(&types.Header{
		Number:     new(big.Int).SetUint64(result.Height),
		Time:       big.NewInt(result.Timestamp.UnixNano() / 1000000),
		Coinbase:   common.BytesToAddress(block.ProposerID.Bytes()),
		GasLimit:   d.gov.DexconConfiguration(block.Position.Round).BlockGasLimit,
		Difficulty: big.NewInt(1),
		Round:      block.Position.Round,
		DexconMeta: dexconMeta,
		Randomness: result.Randomness,
	}, txs, nil, nil)

	root, err := d.blockchain.ProcessPendingBlock(newBlock, &block.Witness)
	if err != nil {
		log.Error("Failed to process pending block", "error", err)
		panic(err)
	}
	d.chainLatestRoot.Store(block.Position.ChainID, root)

	d.blockchain.RemoveConfirmedBlock(chainID, blockHash)
}

// BlockConfirmed is called when a block is confirmed and added to lattice.
func (d *DexconApp) BlockConfirmed(block coreTypes.Block) {
	d.chainLock(block.Position.ChainID)
	defer d.chainUnlock(block.Position.ChainID)

	d.blockchain.AddConfirmedBlock(&block)
}
