// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"fmt"
	"math/big"
	"math/rand"
	"time"

	coreTypes "github.com/byzantine-lab/dexon-consensus/core/types"

	"github.com/tangerine-network/go-tangerine/common"
	"github.com/tangerine-network/go-tangerine/consensus"
	"github.com/tangerine-network/go-tangerine/core/state"
	"github.com/tangerine-network/go-tangerine/core/types"
	"github.com/tangerine-network/go-tangerine/core/vm"
	"github.com/tangerine-network/go-tangerine/ethdb"
	"github.com/tangerine-network/go-tangerine/params"
	"github.com/tangerine-network/go-tangerine/rlp"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

// TangerineBlockGen creates blocks for testing.
// See GenerateChain for a detailed explanation.
type TangerineBlockGen struct {
	i       int
	parent  *types.Block
	chain   []*types.Block
	header  *types.Header
	statedb *state.StateDB

	dirtyNonce map[common.Address]uint64

	position coreTypes.Position

	gasPool  *GasPool
	txs      []*types.Transaction
	receipts []*types.Receipt

	config *params.ChainConfig
	engine consensus.Engine
}

// SetCoinbase sets the coinbase of the generated block.
// It can be called at most once.
func (b *TangerineBlockGen) SetCoinbase(addr common.Address) {
	if b.gasPool != nil {
		if len(b.txs) > 0 {
			panic("coinbase must be set before adding transactions")
		}
		panic("coinbase can only be set once")
	}
	b.header.Coinbase = addr
	b.gasPool = new(GasPool).AddGas(b.header.GasLimit)
}

// SetExtra sets the extra data field of the generated block.
func (b *TangerineBlockGen) SetExtra(data []byte) {
	b.header.Extra = data
}

// SetNonce sets the nonce field of the generated block.
func (b *TangerineBlockGen) SetNonce(nonce types.BlockNonce) {
	b.header.Nonce = nonce
}

func (b *TangerineBlockGen) SetPosition(position coreTypes.Position) {
	b.position = position
}

// AddTx adds a transaction to the generated block. If no coinbase has
// been set, the block's coinbase is set to the zero address.
//
// AddTx panics if the transaction cannot be executed. In addition to
// the protocol-imposed limitations (gas limit, etc.), there are some
// further limitations on the content of transactions that can be
// added. If contract code relies on the BLOCKHASH instruction,
// the block in chain will be returned.
func (b *TangerineBlockGen) AddTx(tx *types.Transaction) {
	// TODO: check nonce and maintain nonce.
	b.txs = append(b.txs, tx)
}

func (b *TangerineBlockGen) PreparePayload() []byte {
	return nil
}

func (b *TangerineBlockGen) ProcessTransactions(c ChainContext) {
	if b.gasPool == nil {
		b.SetCoinbase(common.Address{})
	}
	for _, tx := range b.txs {
		b.statedb.Prepare(tx.Hash(), common.Hash{}, len(b.txs))
		// TODO: fix the chain context parameter
		receipt, _, err := ApplyTransaction(b.config, c, &b.header.Coinbase, b.gasPool, b.statedb, b.header, tx, &b.header.GasUsed, vm.Config{})
		if err != nil {
			panic(err)
		}
		b.receipts = append(b.receipts, receipt)
	}
}

// Number returns the block number of the block being generated.
func (b *TangerineBlockGen) Number() *big.Int {
	return new(big.Int).Set(b.header.Number)
}

// AddUncheckedReceipt forcefully adds a receipts to the block without a
// backing transaction.
//
// AddUncheckedReceipt will cause consensus failures when used during real
// chain processing. This is best used in conjunction with raw block insertion.
func (b *TangerineBlockGen) AddUncheckedReceipt(receipt *types.Receipt) {
	b.receipts = append(b.receipts, receipt)
}

// TxNonce returns the next valid transaction nonce for the
// account at addr. It panics if the account does not exist.
func (b *TangerineBlockGen) TxNonce(addr common.Address) uint64 {
	if !b.statedb.Exist(addr) {
		panic("account does not exist")
	}
	if nonce, exists := b.dirtyNonce[addr]; exists {
		return nonce
	}
	return b.statedb.GetNonce(addr)
}

// PrevBlock returns a previously generated block by number. It panics if
// num is greater or equal to the number of the block being generated.
// For index -1, PrevBlock returns the parent block given to GenerateChain.
func (b *TangerineBlockGen) PrevBlock(index int) *types.Block {
	if index >= b.i {
		panic(fmt.Errorf("block index %d out of range (%d,%d)", index, -1, b.i))
	}
	if index == -1 {
		return b.parent
	}
	return b.chain[index]
}

func GenerateTangerineChain(config *params.ChainConfig, parent *types.Block, engine consensus.Engine, db ethdb.Database, n int, gen func(int, *TangerineBlockGen)) ([]*types.Block, []types.Receipts) {
	if config == nil {
		config = params.TestChainConfig
	}
	if engine == nil {
		panic("engine is nil")
	}
	blocks, receipts := make(types.Blocks, n), make([]types.Receipts, n)
	chain := &fakeTangerineChain{
		config:          config,
		engine:          engine,
		genesis:         parent,
		db:              db,
		headersByNumber: make(map[uint64]*types.Header),
		roundHeight:     make(map[uint64]uint64),
	}
	genblock := func(i int, parent *types.Block, statedb *state.StateDB) (*types.Block, types.Receipts) {
		b := &TangerineBlockGen{i: i, chain: blocks, parent: parent, statedb: statedb, config: config, engine: engine}
		b.header = makeTangerineHeader(chain, parent, statedb, b.engine)

		// Execute any user modifications to the block
		if gen != nil {
			gen(i, b)
		}

		b.header.DexconMeta = makeDexconMeta(b, parent)
		b.header.Round = b.position.Round

		// sign tsig to create dexcon, prepare witness
		b.engine.Prepare(chain, b.header)

		// Run EVM and finalize the block.
		b.ProcessTransactions(chain)

		// Finalize and seal the block
		block, _ := b.engine.Finalize(chain, b.header, statedb, b.txs, nil, b.receipts)

		// Write state changes to db
		root, err := statedb.Commit(config.IsEIP158(b.header.Number))
		if err != nil {
			panic(fmt.Sprintf("state write error: %v", err))
		}
		if err := statedb.Database().TrieDB().Commit(root, false); err != nil {
			panic(fmt.Sprintf("trie write error: %v", err))
		}
		return block, b.receipts
	}
	for i := 0; i < n; i++ {
		statedb, err := state.New(parent.Root(), state.NewDatabase(db))
		if err != nil {
			panic(err)
		}
		block, receipt := genblock(i, parent, statedb)
		blocks[i] = block
		receipts[i] = receipt
		parent = block
		chain.InsertBlock(block)
	}
	return blocks, receipts
}

func makeTangerineHeader(chain consensus.ChainReader, parent *types.Block, state *state.StateDB, engine consensus.Engine) *types.Header {
	var time uint64
	if parent.Time() == 0 {
		time = 10
	} else {
		time = parent.Time() + 10 // block time is fixed at 10 seconds
	}

	return &types.Header{
		Root:       state.IntermediateRoot(chain.Config().IsEIP158(parent.Number())),
		ParentHash: parent.Hash(),
		Coinbase:   parent.Coinbase(),
		Difficulty: new(big.Int).Add(parent.Difficulty(), common.Big1),
		GasLimit:   chain.Config().Dexcon.BlockGasLimit,
		Number:     new(big.Int).Add(parent.Number(), common.Big1),
		Time:       time,
	}
}

// Setup DexconMeata skeleton
func makeDexconMeta(b *TangerineBlockGen, parent *types.Block) []byte {
	// Setup witness
	h := parent.Number().Int64() - rand.Int63n(6)
	if h < 0 {
		h = 0
	}
	witnessedBlock := b.chain[h]
	if witnessedBlock == nil {
		witnessedBlock = parent
	}
	witnessedBlockHash := witnessedBlock.Hash()
	data, err := rlp.EncodeToBytes(&witnessedBlockHash)
	if err != nil {
		panic(err)
	}

	block := coreTypes.Block{
		Position: b.position,
		Witness: coreTypes.Witness{
			Height: witnessedBlock.NumberU64(),
			Data:   data,
		},
	}

	dexconMeta, err := rlp.EncodeToBytes(&block)
	if err != nil {
		panic(err)
	}
	return dexconMeta
}

type fakeTangerineChain struct {
	config          *params.ChainConfig
	engine          consensus.Engine
	db              ethdb.Database
	genesis         *types.Block
	headersByNumber map[uint64]*types.Header
	roundHeight     map[uint64]uint64
}

func (f *fakeTangerineChain) InsertBlock(block *types.Block) {
	f.headersByNumber[block.NumberU64()] = block.Header()
	if _, exists := f.roundHeight[block.Round()]; !exists {
		f.roundHeight[block.Round()] = block.NumberU64()
	}
}

// Config returns the chain configuration.
func (f *fakeTangerineChain) Config() *params.ChainConfig                             { return f.config }
func (f *fakeTangerineChain) Engine() consensus.Engine                                { return f.engine }
func (f *fakeTangerineChain) CurrentHeader() *types.Header                            { return nil }
func (f *fakeTangerineChain) GetHeaderByHash(hash common.Hash) *types.Header          { return nil }
func (f *fakeTangerineChain) GetHeader(hash common.Hash, number uint64) *types.Header { return nil }
func (f *fakeTangerineChain) GetBlock(hash common.Hash, number uint64) *types.Block   { return nil }

func (f *fakeTangerineChain) GetHeaderByNumber(number uint64) *types.Header {
	if number == 0 {
		return f.genesis.Header()
	}
	return f.headersByNumber[number]
}

func (f *fakeTangerineChain) StateAt(hash common.Hash) (*state.StateDB, error) {
	return state.New(hash, state.NewDatabase(f.db))
}

func (f *fakeTangerineChain) GetRoundHeight(round uint64) (uint64, bool) {
	if round == 0 {
		return 0, true
	}
	height, ok := f.roundHeight[round]
	return height, ok
}
