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
	"github.com/dexon-foundation/dexon/core/rawdb"
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

// PreparePayload is called when consensus core is preparing payload for block.
func (d *DexconApp) PreparePayload(position coreTypes.Position) (payload []byte) {
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
	var gasUsed uint64
	for addr, txs := range txsMap {
		addrModChainSize := new(big.Int)
		if addrModChainSize.Mod(addr.Big(), chainSize).Cmp(chainID) != 0 {
			continue
		}

		var nonce uint64
		for i, tx := range txs {
			// validate start nonce
			// check undelivered block first
			// or else check compaction chain state
			if i == 0 {
				nonce = tx.Nonce()
				msg, err := tx.AsMessage(types.MakeSigner(nil, currentBlock.Header().Number))
				if err != nil {
					return
				}
				undeliveredNonce, exist, err := d.blockchain.GetNonceInConfirmedBlock(position.ChainID, msg.From())
				if err != nil {
					return
				} else if exist {
					if msg.Nonce() != undeliveredNonce+1 {
						break
					}
				} else {
					stateDB, err := d.blockchain.State()
					if err != nil {
						return
					}
					if msg.Nonce() != stateDB.GetNonce(msg.From())+1 {
						break
					}
				}
			} else if tx.Nonce() != nonce+1 { // check if nonce is sequential
				break
			}

			core.ApplyTransaction(d.blockchain.Config(), d.blockchain, nil, gp, stateDB, currentBlock.Header(), tx, &gasUsed, d.vmConfig)
			if gasUsed > gasLimit {
				break
			}
			allTxs = append(allTxs, tx)
			nonce = tx.Nonce()
		}
	}
	payload, err = rlp.EncodeToBytes(&allTxs)
	if err != nil {
		// do something
		return
	}

	return
}

type WitnessData struct {
	Root        common.Hash
	TxHash      common.Hash
	ReceiptHash common.Hash
}

func (d *DexconApp) PrepareWitness(consensusHeight uint64) (witness coreTypes.Witness) {
	var currentBlock *types.Block
	currentBlock = d.blockchain.CurrentBlock()
	if currentBlock.NumberU64() < consensusHeight {
		// wait notification
		if <-d.addNotify(consensusHeight) {
			currentBlock = d.blockchain.CurrentBlock()
		} else {
			// do something if notify fail
		}
	}

	witnessData, err := rlp.EncodeToBytes(&WitnessData{
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
	}
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

	// verify transactions
	addressNonce := map[common.Address]*struct {
		Min uint64
		Max uint64
	}{}
	for _, transaction := range transactions {
		if d.txPool.ValidateTx(transaction, false) != nil {
			return false
		}

		msg, err := transaction.AsMessage(types.MakeSigner(nil, currentBlock.Header().Number))
		if err != nil {
			return false
		}
		nonce, exist := addressNonce[msg.From()]
		if !exist {
			addressNonce[msg.From()] = &struct {
				Min uint64
				Max uint64
			}{Min: msg.Nonce(), Max: msg.Nonce()}
		} else if msg.Nonce() != nonce.Max+1 {
			// address nonce is not sequential
			return false
		}
		nonce.Max++
	}

	for address, nonce := range addressNonce {
		undeliveredNonce, exist, err := d.blockchain.GetNonceInConfirmedBlock(block.Position.ChainID, address)
		if err != nil {
			return false
		} else if exist {
			if nonce.Min != undeliveredNonce+1 {
				return false
			}
		} else {
			stateDB, err := d.blockchain.State()
			if err != nil {
				return false
			}
			if nonce.Min != stateDB.GetNonce(address)+1 {
				return false
			}
		}
	}

	gasLimit := core.CalcGasLimit(currentBlock, d.config.GasFloor, d.config.GasCeil)
	gp := new(core.GasPool).AddGas(gasLimit)

	stateDB, err := state.New(currentBlock.Root(), state.NewDatabase(d.chainDB))
	if err != nil {
		return false
	}

	var gasUsed uint64
	for _, tx := range transactions {
		core.ApplyTransaction(d.blockchain.Config(), d.blockchain, nil, gp, stateDB, currentBlock.Header(), tx, &gasUsed, d.vmConfig)
	}

	if gasUsed > gasLimit+d.config.GasLimitTolerance {
		return false
	}

	witnessData := WitnessData{}
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
		return
	}

	var transactions types.Transactions
	err := rlp.Decode(bytes.NewReader(block.Payload), &transactions)
	if err != nil {
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
		// do something
		return
	}

	d.blockchain.RemoveConfirmedBlock(blockHash)
	d.notify(result.Height)
}

// BlockConfirmed is called when a block is confirmed and added to lattice.
func (d *DexconApp) BlockConfirmed(block coreTypes.Block) {
	d.blockchain.AddConfirmedBlock(&block)
}
