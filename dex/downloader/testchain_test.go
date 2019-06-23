// Copyright 2018 The go-ethereum Authors
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

package downloader

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"

	dexCore "github.com/tangerine-network/tangerine-consensus/core"
	coreTypes "github.com/tangerine-network/tangerine-consensus/core/types"

	"github.com/tangerine-network/go-tangerine/common"
	"github.com/tangerine-network/go-tangerine/consensus/dexcon"
	"github.com/tangerine-network/go-tangerine/core"
	"github.com/tangerine-network/go-tangerine/core/state"
	"github.com/tangerine-network/go-tangerine/core/types"
	"github.com/tangerine-network/go-tangerine/core/vm"
	"github.com/tangerine-network/go-tangerine/crypto"
	"github.com/tangerine-network/go-tangerine/ethdb"
	"github.com/tangerine-network/go-tangerine/params"
)

// Test chain parameters.
var (
	testKey, _             = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	testAddress            = crypto.PubkeyToAddress(testKey.PublicKey)
	testDB                 = ethdb.NewMemDatabase()
	testGenesis, testNodes = genesisBlockForTesting(testDB, testAddress, big.NewInt(1000000000))
)

// The common prefix of all test chains:
var testChainBase = newTestChain(blockCacheItems+200, testGenesis, testNodes)

func genesisBlockForTesting(db ethdb.Database,
	addr common.Address, balance *big.Int) (*types.Block, *dexcon.NodeSet) {
	var (
		// genesis node set
		nodekey1, _ = crypto.HexToECDSA("3cf5bdee098cc34536a7b0e80d85e07a380efca76fc12136299b9e5ba24193c8")
		nodekey2, _ = crypto.HexToECDSA("96c9f1435d53577db18d45411326311529a0e8affb19218e27f65769a482c0fb")
		nodekey3, _ = crypto.HexToECDSA("b25e955e30dd87cbaec83287beea6ec9c4c72498bc66905590756bf48da5d1fc")
		nodekey4, _ = crypto.HexToECDSA("35577f65312f4a5e0b5391f5385043a6bc7b51fa4851a579e845b5fea33efded")
		nodeaddr1   = crypto.PubkeyToAddress(nodekey1.PublicKey)
		nodeaddr2   = crypto.PubkeyToAddress(nodekey2.PublicKey)
		nodeaddr3   = crypto.PubkeyToAddress(nodekey3.PublicKey)
		nodeaddr4   = crypto.PubkeyToAddress(nodekey4.PublicKey)
	)

	ether := big.NewInt(1e18)
	gspec := core.Genesis{
		Config: params.TestnetChainConfig,
		Alloc: core.GenesisAlloc{
			addr: {
				Balance: balance,
				Staked:  new(big.Int),
			},
			nodeaddr1: {
				Balance:   new(big.Int).Mul(big.NewInt(2e6), ether),
				Staked:    new(big.Int).Mul(big.NewInt(1e6), ether),
				PublicKey: crypto.FromECDSAPub(&nodekey1.PublicKey),
			},
			nodeaddr2: {
				Balance:   new(big.Int).Mul(big.NewInt(2e6), ether),
				Staked:    new(big.Int).Mul(big.NewInt(1e6), ether),
				PublicKey: crypto.FromECDSAPub(&nodekey2.PublicKey),
			},
			nodeaddr3: {
				Balance:   new(big.Int).Mul(big.NewInt(2e6), ether),
				Staked:    new(big.Int).Mul(big.NewInt(1e6), ether),
				PublicKey: crypto.FromECDSAPub(&nodekey3.PublicKey),
			},
			nodeaddr4: {
				Balance:   new(big.Int).Mul(big.NewInt(2e6), ether),
				Staked:    new(big.Int).Mul(big.NewInt(1e6), ether),
				PublicKey: crypto.FromECDSAPub(&nodekey4.PublicKey),
			},
		},
	}
	genesis := gspec.MustCommit(db)

	signedCRS := []byte(gspec.Config.Dexcon.GenesisCRSText)
	signer := types.NewEIP155Signer(gspec.Config.ChainID)
	nodeSet := dexcon.NewNodeSet(uint64(0), signedCRS, signer,
		[]*ecdsa.PrivateKey{nodekey1, nodekey2, nodekey3, nodekey4})
	return genesis, nodeSet
}

type testChain struct {
	genesis  *types.Block
	nodes    *dexcon.NodeSet
	chain    []common.Hash
	headerm  map[common.Hash]*types.Header
	blockm   map[common.Hash]*types.Block
	receiptm map[common.Hash][]*types.Receipt
}

// newTestChain creates a blockchain of the given length.
func newTestChain(length int, genesis *types.Block, nodes *dexcon.NodeSet) *testChain {
	tc := new(testChain).copy(length)
	tc.genesis = genesis
	tc.chain = append(tc.chain, genesis.Hash())
	tc.headerm[tc.genesis.Hash()] = tc.genesis.Header()
	tc.blockm[tc.genesis.Hash()] = tc.genesis
	tc.generate(length-1, 0, genesis, nodes, false)
	return tc
}

// shorten creates a copy of the chain with the given length. It panics if the
// length is longer than the number of available blocks.
func (tc *testChain) shorten(length int) *testChain {
	if length > tc.len() {
		panic(fmt.Errorf("can't shorten test chain to %d blocks, it's only %d blocks long", length, tc.len()))
	}
	return tc.copy(length)
}

func (tc *testChain) copy(newlen int) *testChain {
	cpy := &testChain{
		genesis:  tc.genesis,
		headerm:  make(map[common.Hash]*types.Header, newlen),
		blockm:   make(map[common.Hash]*types.Block, newlen),
		receiptm: make(map[common.Hash][]*types.Receipt, newlen),
	}
	for i := 0; i < len(tc.chain) && i < newlen; i++ {
		hash := tc.chain[i]
		cpy.chain = append(cpy.chain, tc.chain[i])
		cpy.blockm[hash] = tc.blockm[hash]
		cpy.headerm[hash] = tc.headerm[hash]
		cpy.receiptm[hash] = tc.receiptm[hash]
	}
	return cpy
}

// generate creates a chain of n blocks starting at and including parent.
// the returned hash chain is ordered head->parent. In addition, every 22th block
// contains a transaction and every 5th an uncle to allow testing correct block
// reassembly.
func (tc *testChain) generate(n int, seed byte, parent *types.Block, nodes *dexcon.NodeSet, heavy bool) {
	// start := time.Now()
	// defer func() { fmt.Printf("test chain generated in %v\n", time.Since(start)) }()

	engine := dexcon.NewFaker(testNodes)
	govFetcher := newGovStateFetcher(state.NewDatabase(testDB))
	govFetcher.SnapshotRound(0, tc.genesis.Root())
	engine.SetGovStateFetcher(govFetcher)

	round := uint64(0)
	roundInterval := int(100)

	addTx := func(block *core.TangerineBlockGen, node *dexcon.Node, data []byte) {
		nonce := block.TxNonce(node.Address())
		tx := node.CreateGovTx(nonce, data)
		block.AddTx(tx)
	}

	blocks, receipts := core.GenerateTangerineChain(params.TestnetChainConfig, parent, engine, testDB, n, func(i int, block *core.TangerineBlockGen) {
		block.SetCoinbase(common.Address{seed})
		block.SetPosition(coreTypes.Position{
			Round:  round,
			Height: uint64(i),
		})
		half := roundInterval / 2
		switch i % roundInterval {
		case half:
			if round >= dexCore.DKGDelayRound {
				// Sign current CRS to geneate the next round CRS and propose it.
				testNodes.SignCRS(round)
				node := testNodes.Nodes(round)[0]
				data, err := vm.PackProposeCRS(round, testNodes.SignedCRS(round+1))
				if err != nil {
					panic(err)
				}
				addTx(block, node, data)
			}
		case half + 1:
			// Run the DKG for next round.
			testNodes.RunDKG(round+1, 2)

			// Add DKG MasterPublicKeys
			for _, node := range testNodes.Nodes(round + 1) {
				data, err := vm.PackAddDKGMasterPublicKey(node.MasterPublicKey(round + 1))
				if err != nil {
					panic(err)
				}
				addTx(block, node, data)
			}
		case half + 2:
			// Add DKG MPKReady
			for _, node := range testNodes.Nodes(round + 1) {
				data, err := vm.PackAddDKGMPKReady(node.DKGMPKReady(round + 1))
				if err != nil {
					panic(err)
				}
				addTx(block, node, data)
			}
		case half + 3:
			// Add DKG Finalize
			for _, node := range testNodes.Nodes(round + 1) {
				data, err := vm.PackAddDKGFinalize(node.DKGFinalize(round + 1))
				if err != nil {
					panic(err)
				}
				addTx(block, node, data)
			}
		case roundInterval - 1:
			round++
		}

		// Include transactions to the miner to make blocks more interesting.
		if parent == tc.genesis && i%22 == 0 {
			signer := types.MakeSigner(params.TestnetChainConfig, block.Number())
			tx, err := types.SignTx(types.NewTransaction(block.TxNonce(testAddress), common.Address{seed}, big.NewInt(1000), params.TxGas, nil, nil), signer, testKey)
			if err != nil {
				panic(err)
			}
			block.AddTx(tx)
		}
	})

	// Convert the block-chain into a hash-chain and header/block maps
	for i, b := range blocks {
		hash := b.Hash()
		tc.chain = append(tc.chain, hash)
		tc.blockm[hash] = b
		tc.headerm[hash] = b.Header()
		tc.receiptm[hash] = receipts[i]
	}
}

// len returns the total number of blocks in the chain.
func (tc *testChain) len() int {
	return len(tc.chain)
}

// headBlock returns the head of the chain.
func (tc *testChain) headBlock() *types.Block {
	return tc.blockm[tc.chain[len(tc.chain)-1]]
}

// headersByHash returns headers in ascending order from the given hash.
func (tc *testChain) headersByHash(origin common.Hash, amount int, skip int) []*types.HeaderWithGovState {
	num, _ := tc.hashToNumber(origin)
	return tc.headersByNumber(num, amount, skip)
}

// headersByNumber returns headers in ascending order from the given number.
func (tc *testChain) headersByNumber(origin uint64, amount int, skip int) []*types.HeaderWithGovState {
	result := make([]*types.HeaderWithGovState, 0, amount)
	for num := origin; num < uint64(len(tc.chain)) && len(result) < amount; num += uint64(skip) + 1 {
		if header, ok := tc.headerm[tc.chain[int(num)]]; ok {
			result = append(result, &types.HeaderWithGovState{Header: header})
		}
	}
	return result
}

func (tc *testChain) govStateByHash(hash common.Hash) *types.GovState {
	header := tc.headersByHash(hash, 1, 0)[0]
	statedb, err := state.New(header.Root, state.NewDatabase(testDB))
	if err != nil {
		panic(err)
	}

	govState, err := state.GetGovState(statedb, header.Header,
		vm.GovernanceContractAddress)
	if err != nil {
		panic(err)
	}
	return govState
}

// receipts returns the receipts of the given block hashes.
func (tc *testChain) receipts(hashes []common.Hash) [][]*types.Receipt {
	results := make([][]*types.Receipt, 0, len(hashes))
	for _, hash := range hashes {
		if receipt, ok := tc.receiptm[hash]; ok {
			results = append(results, receipt)
		}
	}
	return results
}

// bodies returns the block bodies of the given block hashes.
func (tc *testChain) bodies(hashes []common.Hash) ([][]*types.Transaction, [][]*types.Header) {
	transactions := make([][]*types.Transaction, 0, len(hashes))
	uncles := make([][]*types.Header, 0, len(hashes))
	for _, hash := range hashes {
		if block, ok := tc.blockm[hash]; ok {
			transactions = append(transactions, block.Transactions())
			uncles = append(uncles, block.Uncles())
		}
	}
	return transactions, uncles
}

func (tc *testChain) hashToNumber(target common.Hash) (uint64, bool) {
	for num, hash := range tc.chain {
		if hash == target {
			return uint64(num), true
		}
	}
	return 0, false
}

type govStateFetcher struct {
	db          state.Database
	rootByRound map[uint64]common.Hash
}

func newGovStateFetcher(db state.Database) *govStateFetcher {
	return &govStateFetcher{
		db:          db,
		rootByRound: make(map[uint64]common.Hash),
	}
}

func (g *govStateFetcher) SnapshotRound(round uint64, root common.Hash) {
	g.rootByRound[round] = root
}

func (g *govStateFetcher) GetStateForConfigAtRound(round uint64) *vm.GovernanceState {
	if root, ok := g.rootByRound[round]; ok {
		s, err := state.New(root, g.db)
		if err != nil {
			panic(err)
		}
		return &vm.GovernanceState{s}
	}
	return nil
}

func (g *govStateFetcher) DKGSetNodeKeyAddresses(round uint64) (map[common.Address]struct{}, error) {
	return make(map[common.Address]struct{}), nil
}
