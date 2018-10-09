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
	mutex      *sync.Mutex

	lastPendingHeight uint64
	insertMu          sync.Mutex

	chainHeight map[uint32]uint64
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
		txPool:      txPool,
		blockchain:  blockchain,
		gov:         gov,
		chainDB:     chainDB,
		config:      config,
		vmConfig:    vmConfig,
		notifyChan:  make(map[uint64]*notify),
		mutex:       &sync.Mutex{},
		chainHeight: make(map[uint32]uint64),
	}
}

func (d *DexconApp) addNotify(height uint64) <-chan uint64 {
	d.mutex.Lock()
	defer d.mutex.Unlock()
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
	d.mutex.Lock()
	defer d.mutex.Unlock()
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
	d.insertMu.Lock()
	defer d.insertMu.Unlock()
	txsMap, err := d.txPool.Pending()
	if err != nil {
		return
	}

	chainID := new(big.Int).SetUint64(uint64(position.ChainID))
	chainNums := new(big.Int).SetUint64(uint64(d.gov.GetNumChains(position.Round)))
	var allTxs types.Transactions
	for addr, txs := range txsMap {
		// every address's transactions will appear in fixed chain
		if !d.checkChain(addr, chainNums, chainID) {
			continue
		}

		var stateDB *state.StateDB
		if d.lastPendingHeight > 0 {
			stateDB, err = d.blockchain.StateAt(d.blockchain.GetPendingBlockByHeight(d.lastPendingHeight).Root())
			if err != nil {
				return nil, fmt.Errorf("PreparePayload d.blockchain.StateAt err %v", err)
			}
		} else {
			stateDB, err = d.blockchain.State()
			if err != nil {
				return nil, fmt.Errorf("PreparePayload d.blockchain.State err %v", err)
			}
		}

		for _, tx := range txs {
			if tx.Nonce() != stateDB.GetNonce(addr) {
				log.Debug("Break transaction", "tx.hash", tx.Hash(), "nonce", tx.Nonce(), "expect", stateDB.GetNonce(addr))
				break
			}
			log.Debug("Receive transaction", "tx.hash", tx.Hash(), "nonce", tx.Nonce(), "amount", tx.Value())
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
		witnessBlock = d.blockchain.GetPendingBlockByHeight(d.lastPendingHeight)
	} else if h := <-d.addNotify(consensusHeight); h >= consensusHeight {
		witnessBlock = d.blockchain.GetPendingBlockByHeight(h)
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
		Timestamp: time.Unix(witnessBlock.Time().Int64(), 0),
		Height:    witnessBlock.NumberU64(),
		Data:      witnessData,
	}, nil
}

// VerifyBlock verifies if the payloads are valid.
func (d *DexconApp) VerifyBlock(block *coreTypes.Block) bool {
	d.insertMu.Lock()
	defer d.insertMu.Unlock()

	var witnessData witnessData
	err := rlp.Decode(bytes.NewReader(block.Witness.Data), &witnessData)
	if err != nil {
		log.Error("Witness rlp decode", "error", err)
		return false
	}

	// check witness root exist
	_, err = d.blockchain.StateAt(witnessData.Root)
	if err != nil {
		log.Error("Get state root error", "err", err)
		return false
	}

	if block.Position.Height != 0 {
		chainLastHeight, empty := d.blockchain.GetChainLastConfirmedHeight(block.Position.ChainID)
		if empty {
			var exist bool
			chainLastHeight, exist = d.chainHeight[block.Position.ChainID]
			if !exist {
				log.Error("Something wrong")
				return false
			}
		}

		// check if chain block height is sequential
		if chainLastHeight != block.Position.Height-1 {
			log.Error("Check confirmed block height fail", "chain", block.Position.ChainID, "height", block.Position.Height-1)
			return false
		}
	}

	// set state to the pending height
	var latestState *state.StateDB
	if d.lastPendingHeight == 0 {
		latestState, err = d.blockchain.State()
		if err != nil {
			log.Error("Get current state", "error", err)
			return false
		}
	} else {
		latestState, err = d.blockchain.StateAt(d.blockchain.GetPendingBlockByHeight(d.lastPendingHeight).Root())
		if err != nil {
			log.Error("Get pending state", "error", err)
			return false
		}
	}

	var transactions types.Transactions
	err = rlp.Decode(bytes.NewReader(block.Payload), &transactions)
	if err != nil {
		log.Error("Payload rlp decode", "error", err)
		return false
	}

	// check if nonce is sequential and return first nonce of every address
	addresses, err := d.validateNonce(transactions)
	if err != nil {
		log.Error("Get address nonce", "error", err)
		return false
	}

	// check all address nonce
	chainID := big.NewInt(int64(block.Position.ChainID))
	chainNums := new(big.Int).SetUint64(uint64(d.gov.GetNumChains(block.Position.Round)))
	for address, firstNonce := range addresses {
		if !d.checkChain(address, chainNums, chainID) {
			log.Error("check chain fail", "address", address)
			return false
		}

		var expectNonce uint64
		// get last nonce from confirmed blocks
		lastConfirmedNonce, empty, err := d.blockchain.GetLastNonceFromConfirmedBlocks(block.Position.ChainID, address)
		if err != nil {
			log.Error("Get last nonce from confirmed blocks", "error", err)
			return false
		} else if empty {
			// get expect nonce from latest state when confirmed block is empty
			expectNonce = latestState.GetNonce(address)
		} else {
			expectNonce = lastConfirmedNonce + 1
		}

		if expectNonce != firstNonce {
			log.Error("Nonce check error", "expect", expectNonce, "firstNonce", firstNonce)
			return false
		}
	}

	// get balance from state
	addressesBalance := map[common.Address]*big.Int{}
	for address := range addresses {
		addressesBalance[address] = latestState.GetBalance(address)
	}

	// replay confirmed block tx to correct balance
	confirmedBlocks := d.blockchain.GetConfirmedBlocksByChainID(block.Position.ChainID)
	for _, block := range confirmedBlocks {
		var txs types.Transactions
		err := rlp.Decode(bytes.NewReader(block.Payload), &txs)
		if err != nil {
			log.Error("Decode confirmed block", "error", err)
			return false
		}

		for _, tx := range txs {
			msg, err := tx.AsMessage(types.MakeSigner(d.blockchain.Config(), new(big.Int)))
			if err != nil {
				log.Error("Tx to message", "error", err)
				return false
			}

			balance, exist := addressesBalance[msg.From()]
			if exist {
				maxGasUsed := new(big.Int).Mul(new(big.Int).SetUint64(msg.Gas()), msg.GasPrice())
				balance = new(big.Int).Sub(balance, maxGasUsed)
				if balance.Cmp(big.NewInt(0)) <= 0 {
					log.Error("Replay confirmed tx fail", "reason", "not enough balance")
					return false
				}
				addressesBalance[msg.From()] = balance
			}
		}
	}

	// validate tx to check available balance
	for _, tx := range transactions {
		msg, err := tx.AsMessage(types.MakeSigner(d.blockchain.Config(), new(big.Int)))
		if err != nil {
			log.Error("Tx to message", "error", err)
			return false
		}
		balance, _ := addressesBalance[msg.From()]
		maxGasUsed := new(big.Int).Mul(new(big.Int).SetUint64(msg.Gas()), msg.GasPrice())
		balance = new(big.Int).Sub(balance, maxGasUsed)
		if balance.Cmp(big.NewInt(0)) <= 0 {
			log.Error("Tx fail", "reason", "not enough balance")
			return false
		}
		addressesBalance[msg.From()] = balance
	}

	return true
}

// BlockDelivered is called when a block is add to the compaction chain.
func (d *DexconApp) BlockDelivered(blockHash coreCommon.Hash, result coreTypes.FinalizationResult) {
	d.insertMu.Lock()
	defer d.insertMu.Unlock()

	block := d.blockchain.GetConfirmedBlockByHash(blockHash)
	if block == nil {
		log.Error("Can not get confirmed block")
		return
	}

	var transactions types.Transactions
	err := rlp.Decode(bytes.NewReader(block.Payload), &transactions)
	if err != nil {
		log.Error("Payload rlp decode failed", "error", err)
		return
	}

	var witnessData witnessData
	err = rlp.Decode(bytes.NewReader(block.Witness.Data), &witnessData)
	if err != nil {
		log.Error("Witness rlp decode failed", "error", err)
		return
	}

	log.Debug("Block proposer id", "hash", block.ProposerID)
	newBlock := types.NewBlock(&types.Header{
		Number:             new(big.Int).SetUint64(result.Height),
		Time:               big.NewInt(result.Timestamp.Unix()),
		Coinbase:           common.BytesToAddress(block.ProposerID.Bytes()),
		Position:           block.Position,
		WitnessHeight:      block.Witness.Height,
		WitnessRoot:        witnessData.Root,
		WitnessReceiptHash: witnessData.ReceiptHash,
		ChainID:            block.Position.ChainID,
		ChainBlockHeight:   block.Position.Height,
		// TODO(bojie): fix it
		GasLimit:   8000000,
		Difficulty: big.NewInt(1),
	}, transactions, nil, nil)

	_, err = d.blockchain.InsertPendingBlocks([]*types.Block{newBlock})
	if err != nil {
		log.Error("Insert chain", "error", err)
		return
	}

	log.Debug("Insert pending block success", "height", result.Height)
	d.chainHeight[block.Position.ChainID] = block.Position.Height
	d.blockchain.RemoveConfirmedBlock(blockHash)
	d.notify(result.Height)
}

// BlockConfirmed is called when a block is confirmed and added to lattice.
func (d *DexconApp) BlockConfirmed(block coreTypes.Block) {
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
