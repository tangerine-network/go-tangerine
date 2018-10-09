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
	"github.com/dexon-foundation/dexon/common/math"
	"github.com/dexon-foundation/dexon/core"
	"github.com/dexon-foundation/dexon/core/rawdb"
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
}

type notify struct {
	results []chan bool
}

type witnessData struct {
	Root        common.Hash
	TxHash      common.Hash
	ReceiptHash common.Hash
}

func NewDexconApp(txPool *core.TxPool, blockchain *core.BlockChain, gov *DexconGovernance, chainDB ethdb.Database, config *Config, vmConfig vm.Config) *DexconApp {
	return &DexconApp{
		txPool:     txPool,
		blockchain: blockchain,
		gov:        gov,
		chainDB:    chainDB,
		config:     config,
		vmConfig:   vmConfig,
		notifyChan: make(map[uint64]*notify),
		mutex:      &sync.Mutex{},
	}
}

func (d *DexconApp) addNotify(height uint64) <-chan bool {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	result := make(chan bool)
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
				ch <- true
			}
			delete(d.notifyChan, h)
		}
	}
}

func (d *DexconApp) checkChain(address common.Address, chainSize, chainID *big.Int) bool {
	addrModChainSize := new(big.Int)
	return addrModChainSize.Mod(address.Big(), chainSize).Cmp(chainID) == 0
}

// PreparePayload is called when consensus core is preparing payload for block.
func (d *DexconApp) PreparePayload(position coreTypes.Position) (payload []byte, err error) {
	txsMap, err := d.txPool.Pending()
	if err != nil {
		return
	}

	currentBlock := d.blockchain.CurrentBlock()
	gasLimit := core.CalcGasLimit(currentBlock, d.config.GasFloor, d.config.GasCeil)
	gp := new(core.GasPool).AddGas(gasLimit)
	stateDB, err := state.New(currentBlock.Root(), state.NewDatabase(d.chainDB))
	if err != nil {
		return
	}

	chainID := new(big.Int).SetUint64(uint64(position.ChainID))
	chainSize := new(big.Int).SetUint64(uint64(d.gov.Configuration(position.Round).NumChains))
	var allTxs types.Transactions
	var totalGasUsed uint64
	for addr, txs := range txsMap {
		// every address's transactions will appear in fixed chain
		if !d.checkChain(addr, chainSize, chainID) {
			continue
		}

		undeliveredTxs, err := d.blockchain.GetConfirmedTxsByAddress(position.ChainID, addr)
		if err != nil {
			return nil, err
		}

		for _, tx := range undeliveredTxs {
			// confirmed txs must apply successfully
			var gasUsed uint64
			gp := new(core.GasPool).AddGas(math.MaxUint64)
			_, _, err = core.ApplyTransaction(d.blockchain.Config(), d.blockchain, nil, gp, stateDB, currentBlock.Header(), tx, &gasUsed, d.vmConfig)
			if err != nil {
				log.Error("apply confirmed transaction error: %v", err)
				return nil, err
			}
		}

		for _, tx := range txs {
			// apply transaction to calculate total gas used, validate nonce and check available balance
			_, _, err = core.ApplyTransaction(d.blockchain.Config(), d.blockchain, nil, gp, stateDB, currentBlock.Header(), tx, &totalGasUsed, d.vmConfig)
			if err != nil || totalGasUsed > gasLimit {
				log.Debug("apply transaction fail error: %v, totalGasUsed: %d, gasLimit: %d", err, totalGasUsed, gasLimit)
				break
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
	var currentBlock *types.Block
	currentBlock = d.blockchain.CurrentBlock()
	if currentBlock.NumberU64() < consensusHeight {
		// wait notification
		if <-d.addNotify(consensusHeight) {
			currentBlock = d.blockchain.CurrentBlock()
		} else {
			err = fmt.Errorf("fail to wait notification")
			return
		}
	}

	witnessData, err := rlp.EncodeToBytes(&witnessData{
		Root:        currentBlock.Root(),
		TxHash:      currentBlock.TxHash(),
		ReceiptHash: currentBlock.ReceiptHash(),
	})
	if err != nil {
		return
	}

	return coreTypes.Witness{
		Timestamp: time.Unix(currentBlock.Time().Int64(), 0),
		Height:    currentBlock.NumberU64(),
		Data:      witnessData,
	}, nil
}

// VerifyBlock verifies if the payloads are valid.
func (d *DexconApp) VerifyBlock(block *coreTypes.Block) bool {
	// decode payload to transactions
	var transactions types.Transactions
	err := rlp.Decode(bytes.NewReader(block.Payload), &transactions)
	if err != nil {
		return false
	}

	currentBlock := d.blockchain.CurrentBlock()
	chainID := new(big.Int).SetUint64(uint64(block.Position.ChainID))
	chainSize := new(big.Int).SetUint64(uint64(d.gov.Configuration(block.Position.Round).NumChains))
	stateDB, err := state.New(currentBlock.Root(), state.NewDatabase(d.chainDB))
	if err != nil {
		return false
	}

	// verify transactions
	addressMap := map[common.Address]interface{}{}
	gasLimit := core.CalcGasLimit(currentBlock, d.config.GasFloor, d.config.GasCeil)
	gp := new(core.GasPool).AddGas(gasLimit)
	var totalGasUsed uint64
	for _, transaction := range transactions {
		msg, err := transaction.AsMessage(types.MakeSigner(d.blockchain.Config(), currentBlock.Header().Number))
		if err != nil {
			return false
		}

		if !d.checkChain(msg.From(), chainSize, chainID) {
			log.Error("%s can not be delivered on chain %d", msg.From().String(), chainID)
			return false
		}

		_, exist := addressMap[msg.From()]
		if !exist {
			txs, err := d.blockchain.GetConfirmedTxsByAddress(block.Position.ChainID, msg.From())
			if err != nil {
				log.Error("get confirmed txs by address error: %v", err)
				return false
			}

			gp := new(core.GasPool).AddGas(math.MaxUint64)
			for _, tx := range txs {
				var gasUsed uint64
				// confirmed txs must apply successfully
				_, _, err := core.ApplyTransaction(d.blockchain.Config(), d.blockchain, nil, gp, stateDB, currentBlock.Header(), tx, &gasUsed, d.vmConfig)
				if err != nil {
					log.Error("apply confirmed transaction error: %v", err)
					return false
				}
			}
			addressMap[msg.From()] = nil
		}

		_, _, err = core.ApplyTransaction(d.blockchain.Config(), d.blockchain, nil, gp, stateDB, currentBlock.Header(), transaction, &totalGasUsed, d.vmConfig)
		if err != nil {
			log.Error("apply block transaction error: %v", err)
			return false
		}
	}

	if totalGasUsed > gasLimit+d.config.GasLimitTolerance {
		return false
	}

	witnessData := witnessData{}
	err = rlp.Decode(bytes.NewReader(block.Witness.Data), &witnessData)
	if err != nil {
		return false
	}

	witnessBlock := d.blockchain.GetBlockByNumber(block.Witness.Height)
	if witnessBlock == nil {
		return false
	} else if witnessBlock.Root() != witnessData.Root {
		// invalid state root of witness data
		return false
	} else if witnessBlock.ReceiptHash() != witnessData.ReceiptHash {
		// invalid receipt root of witness data
		return false
	} else if witnessBlock.TxHash() != witnessData.TxHash {
		// invalid tx root of witness data
		return false
	}

	for _, transaction := range witnessBlock.Transactions() {
		tx, _, _, _ := rawdb.ReadTransaction(d.chainDB, transaction.Hash())
		if tx == nil {
			return false
		}
	}

	return true
}

// BlockDelivered is called when a block is add to the compaction chain.
func (d *DexconApp) BlockDelivered(blockHash coreCommon.Hash, result coreTypes.FinalizationResult) {
	block := d.blockchain.GetConfirmedBlockByHash(blockHash)
	if block == nil {
		// do something
		log.Error("can not get confirmed block")
		return
	}

	var transactions types.Transactions
	err := rlp.Decode(bytes.NewReader(block.Payload), &transactions)
	if err != nil {
		log.Error("payload rlp decode error: %v", err)
		return
	}

	_, err = d.blockchain.InsertChain(
		[]*types.Block{types.NewBlock(&types.Header{
			ParentHash: common.Hash(block.ParentHash),
			Number:     new(big.Int).SetUint64(result.Height),
			Time:       new(big.Int).SetInt64(result.Timestamp.Unix()),
			TxHash:     types.DeriveSha(transactions),
			Coinbase:   common.BytesToAddress(block.ProposerID.Hash[:]),
		}, transactions, nil, nil)})
	if err != nil {
		log.Error("insert chain error: %v", err)
		return
	}

	d.blockchain.RemoveConfirmedBlock(blockHash)
	d.notify(result.Height)
}

// BlockConfirmed is called when a block is confirmed and added to lattice.
func (d *DexconApp) BlockConfirmed(block coreTypes.Block) {
	d.blockchain.AddConfirmedBlock(&block)
}
