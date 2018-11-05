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
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"
	"math/big"
	"math/rand"
	"time"

	coreCommon "github.com/dexon-foundation/dexon-consensus/common"
	coreCrypto "github.com/dexon-foundation/dexon-consensus/core/crypto"
	coreDKG "github.com/dexon-foundation/dexon-consensus/core/crypto/dkg"
	coreEcdsa "github.com/dexon-foundation/dexon-consensus/core/crypto/ecdsa"
	coreTypes "github.com/dexon-foundation/dexon-consensus/core/types"
	coreTypesDKG "github.com/dexon-foundation/dexon-consensus/core/types/dkg"

	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/consensus"
	"github.com/dexon-foundation/dexon/consensus/misc"
	"github.com/dexon-foundation/dexon/core/state"
	"github.com/dexon-foundation/dexon/core/types"
	"github.com/dexon-foundation/dexon/core/vm"
	"github.com/dexon-foundation/dexon/crypto"
	"github.com/dexon-foundation/dexon/ethdb"
	"github.com/dexon-foundation/dexon/params"
	"github.com/dexon-foundation/dexon/rlp"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

// BlockGen creates blocks for testing.
// See GenerateChain for a detailed explanation.
type BlockGen struct {
	i       int
	parent  *types.Block
	chain   []*types.Block
	header  *types.Header
	statedb *state.StateDB

	gasPool  *GasPool
	txs      []*types.Transaction
	receipts []*types.Receipt
	uncles   []*types.Header

	config *params.ChainConfig
	engine consensus.Engine
}

// SetCoinbase sets the coinbase of the generated block.
// It can be called at most once.
func (b *BlockGen) SetCoinbase(addr common.Address) {
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
func (b *BlockGen) SetExtra(data []byte) {
	b.header.Extra = data
}

// SetNonce sets the nonce field of the generated block.
func (b *BlockGen) SetNonce(nonce types.BlockNonce) {
	b.header.Nonce = nonce
}

// AddTx adds a transaction to the generated block. If no coinbase has
// been set, the block's coinbase is set to the zero address.
//
// AddTx panics if the transaction cannot be executed. In addition to
// the protocol-imposed limitations (gas limit, etc.), there are some
// further limitations on the content of transactions that can be
// added. Notably, contract code relying on the BLOCKHASH instruction
// will panic during execution.
func (b *BlockGen) AddTx(tx *types.Transaction) {
	b.AddTxWithChain(nil, tx)
}

// AddTxWithChain adds a transaction to the generated block. If no coinbase has
// been set, the block's coinbase is set to the zero address.
//
// AddTxWithChain panics if the transaction cannot be executed. In addition to
// the protocol-imposed limitations (gas limit, etc.), there are some
// further limitations on the content of transactions that can be
// added. If contract code relies on the BLOCKHASH instruction,
// the block in chain will be returned.
func (b *BlockGen) AddTxWithChain(bc *BlockChain, tx *types.Transaction) {
	if b.gasPool == nil {
		b.SetCoinbase(common.Address{})
	}
	b.statedb.Prepare(tx.Hash(), common.Hash{}, len(b.txs))
	receipt, _, err := ApplyTransaction(b.config, bc, &b.header.Coinbase, b.gasPool, b.statedb, b.header, tx, &b.header.GasUsed, vm.Config{})
	if err != nil {
		panic(err)
	}
	b.txs = append(b.txs, tx)
	b.receipts = append(b.receipts, receipt)
}

// Number returns the block number of the block being generated.
func (b *BlockGen) Number() *big.Int {
	return new(big.Int).Set(b.header.Number)
}

// AddUncheckedReceipt forcefully adds a receipts to the block without a
// backing transaction.
//
// AddUncheckedReceipt will cause consensus failures when used during real
// chain processing. This is best used in conjunction with raw block insertion.
func (b *BlockGen) AddUncheckedReceipt(receipt *types.Receipt) {
	b.receipts = append(b.receipts, receipt)
}

// TxNonce returns the next valid transaction nonce for the
// account at addr. It panics if the account does not exist.
func (b *BlockGen) TxNonce(addr common.Address) uint64 {
	if !b.statedb.Exist(addr) {
		panic("account does not exist")
	}
	return b.statedb.GetNonce(addr)
}

// AddUncle adds an uncle header to the generated block.
func (b *BlockGen) AddUncle(h *types.Header) {
	b.uncles = append(b.uncles, h)
}

// PrevBlock returns a previously generated block by number. It panics if
// num is greater or equal to the number of the block being generated.
// For index -1, PrevBlock returns the parent block given to GenerateChain.
func (b *BlockGen) PrevBlock(index int) *types.Block {
	if index >= b.i {
		panic(fmt.Errorf("block index %d out of range (%d,%d)", index, -1, b.i))
	}
	if index == -1 {
		return b.parent
	}
	return b.chain[index]
}

// OffsetTime modifies the time instance of a block, implicitly changing its
// associated difficulty. It's useful to test scenarios where forking is not
// tied to chain length directly.
func (b *BlockGen) OffsetTime(seconds int64) {
	b.header.Time += uint64(seconds)
	if b.header.Time <= b.parent.Header().Time {
		panic("block time out of range")
	}
	chainreader := &fakeChainReader{config: b.config}
	b.header.Difficulty = b.engine.CalcDifficulty(chainreader, b.header.Time, b.parent.Header())
}

// GenerateChain creates a chain of n blocks. The first block's
// parent will be the provided parent. db is used to store
// intermediate states and should contain the parent's state trie.
//
// The generator function is called with a new block generator for
// every block. Any transactions and uncles added to the generator
// become part of the block. If gen is nil, the blocks will be empty
// and their coinbase will be the zero address.
//
// Blocks created by GenerateChain do not contain valid proof of work
// values. Inserting them into BlockChain requires use of FakePow or
// a similar non-validating proof of work implementation.
func GenerateChain(config *params.ChainConfig, parent *types.Block, engine consensus.Engine, db ethdb.Database, n int, gen func(int, *BlockGen)) ([]*types.Block, []types.Receipts) {
	if config == nil {
		config = params.TestChainConfig
	}
	blocks, receipts := make(types.Blocks, n), make([]types.Receipts, n)
	chainreader := &fakeChainReader{config: config}
	genblock := func(i int, parent *types.Block, statedb *state.StateDB) (*types.Block, types.Receipts) {
		b := &BlockGen{i: i, chain: blocks, parent: parent, statedb: statedb, config: config, engine: engine}
		b.header = makeHeader(chainreader, parent, statedb, b.engine)

		// Mutate the state and block according to any hard-fork specs
		if daoBlock := config.DAOForkBlock; daoBlock != nil {
			limit := new(big.Int).Add(daoBlock, params.DAOForkExtraRange)
			if b.header.Number.Cmp(daoBlock) >= 0 && b.header.Number.Cmp(limit) < 0 {
				if config.DAOForkSupport {
					b.header.Extra = common.CopyBytes(params.DAOForkBlockExtra)
				}
			}
		}
		if config.DAOForkSupport && config.DAOForkBlock != nil && config.DAOForkBlock.Cmp(b.header.Number) == 0 {
			misc.ApplyDAOHardFork(statedb)
		}
		// Execute any user modifications to the block
		if gen != nil {
			gen(i, b)
		}
		if b.engine != nil {
			// Finalize and seal the block
			block, _ := b.engine.Finalize(chainreader, b.header, statedb, b.txs, b.uncles, b.receipts)

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
		return nil, nil
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
	}
	return blocks, receipts
}

func makeHeader(chain consensus.ChainReader, parent *types.Block, state *state.StateDB, engine consensus.Engine) *types.Header {
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
		Difficulty: engine.CalcDifficulty(chain, time, &types.Header{
			Number:     parent.Number(),
			Time:       time - 10,
			Difficulty: parent.Difficulty(),
			UncleHash:  parent.UncleHash(),
		}),
		GasLimit: CalcGasLimit(parent, parent.GasLimit(), parent.GasLimit()),
		Number:   new(big.Int).Add(parent.Number(), common.Big1),
		Time:     time,
	}
}

func GenerateChainWithRoundChange(config *params.ChainConfig, parent *types.Block,
	engine consensus.Engine, db ethdb.Database, n int, gen func(int, *BlockGen),
	nodeSet *NodeSet, roundInterval int) ([]*types.Block, []types.Receipts) {
	if config == nil {
		config = params.TestChainConfig
	}

	round := parent.Header().Round

	blocks, receipts := make(types.Blocks, n), make([]types.Receipts, n)
	chainreader := &fakeChainReader{config: config}
	genblock := func(i int, parent *types.Block, statedb *state.StateDB) (*types.Block, types.Receipts) {
		b := &BlockGen{i: i, parent: parent, chain: blocks, statedb: statedb, config: config, engine: engine}
		b.header = makeHeader(chainreader, parent, statedb, b.engine)
		b.header.DexconMeta = makeDexconMeta(round, parent, nodeSet)

		switch i % roundInterval {
		case 0:
			// First block of this round, notify round height
			tx := nodeSet.NotifyRoundHeightTx(round, b.header.Number.Uint64(), b)
			b.AddTx(tx)
		case roundInterval / 2:
			// Run DKG for next round part 1, AddMasterPublicKey
			nodeSet.RunDKG(round, 2)
			for _, node := range nodeSet.nodes[round] {
				tx := node.MasterPublicKeyTx(round, b.TxNonce(node.address))
				b.AddTx(tx)
			}
		case (roundInterval / 2) + 1:
			// Run DKG for next round part 2, DKG finalize
			for _, node := range nodeSet.nodes[round] {
				tx := node.DKGFinalizeTx(round, b.TxNonce(node.address))
				b.AddTx(tx)
			}
		case (roundInterval / 2) + 2:
			// Current DKG set create signed CRS for next round and propose it
			nodeSet.SignedCRS(round)
			tx := nodeSet.CRSTx(round+1, b)
			b.AddTx(tx)
		case roundInterval - 1:
			// Round change
			round++
		}

		// Execute any user modifications to the block and finalize it
		if gen != nil {
			gen(i, b)
		}

		if b.engine != nil {
			block, _ := b.engine.Finalize(chainreader, b.header, statedb, b.txs, b.uncles, b.receipts)
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
		return nil, nil
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
	}
	return blocks, receipts
}

type witnessData struct {
	Root        common.Hash
	TxHash      common.Hash
	ReceiptHash common.Hash
}

func makeDexconMeta(round uint64, parent *types.Block, nodeSet *NodeSet) []byte {
	data, err := rlp.EncodeToBytes(&witnessData{
		Root:        parent.Root(),
		TxHash:      parent.TxHash(),
		ReceiptHash: parent.ReceiptHash(),
	})
	if err != nil {
		panic(err)
	}

	// only put required data, ignore information for BA, ex: acks, votes
	coreBlock := coreTypes.Block{
		Witness: coreTypes.Witness{
			Height: parent.Number().Uint64(),
			Data:   data,
		},
	}

	blockHash, err := hashBlock(&coreBlock)
	if err != nil {
		panic(err)
	}

	var parentCoreBlock coreTypes.Block

	if parent.Number().Uint64() != 0 {
		if err := rlp.DecodeBytes(
			parent.Header().DexconMeta, &parentCoreBlock); err != nil {
			panic(err)
		}
	}

	parentCoreBlockHash, err := hashBlock(&parentCoreBlock)
	if err != nil {
		panic(err)
	}
	randomness := nodeSet.Randomness(round, blockHash)
	coreBlock.Finalization.ParentHash = coreCommon.Hash(parentCoreBlockHash)
	coreBlock.Finalization.Randomness = randomness
	coreBlock.Finalization.Timestamp = time.Now().UTC()
	coreBlock.Finalization.Height = parent.Number().Uint64()

	dexconMeta, err := rlp.EncodeToBytes(&coreBlock)
	if err != nil {
		panic(err)
	}
	return dexconMeta
}

// makeHeaderChain creates a deterministic chain of headers rooted at parent.
func makeHeaderChain(parent *types.Header, n int, engine consensus.Engine, db ethdb.Database, seed int) []*types.Header {
	blocks := makeBlockChain(types.NewBlockWithHeader(parent), n, engine, db, seed)
	headers := make([]*types.Header, len(blocks))
	for i, block := range blocks {
		headers[i] = block.Header()
	}
	return headers
}

// makeBlockChain creates a deterministic chain of blocks rooted at parent.
func makeBlockChain(parent *types.Block, n int, engine consensus.Engine, db ethdb.Database, seed int) []*types.Block {
	blocks, _ := GenerateChain(params.TestChainConfig, parent, engine, db, n, func(i int, b *BlockGen) {
		b.SetCoinbase(common.Address{0: byte(seed), 19: byte(i)})
	})
	return blocks
}

type fakeChainReader struct {
	config  *params.ChainConfig
	genesis *types.Block
}

// Config returns the chain configuration.
func (cr *fakeChainReader) Config() *params.ChainConfig {
	return cr.config
}

func (cr *fakeChainReader) CurrentHeader() *types.Header                            { return nil }
func (cr *fakeChainReader) GetHeaderByNumber(number uint64) *types.Header           { return nil }
func (cr *fakeChainReader) GetHeaderByHash(hash common.Hash) *types.Header          { return nil }
func (cr *fakeChainReader) GetHeader(hash common.Hash, number uint64) *types.Header { return nil }
func (cr *fakeChainReader) GetBlock(hash common.Hash, number uint64) *types.Block   { return nil }

type node struct {
	cryptoKey coreCrypto.PrivateKey
	ecdsaKey  *ecdsa.PrivateKey

	id                  coreTypes.NodeID
	dkgid               coreDKG.ID
	address             common.Address
	prvShares           *coreDKG.PrivateKeyShares
	pubShares           *coreDKG.PublicKeyShares
	receivedPrvShares   *coreDKG.PrivateKeyShares
	recoveredPrivateKey *coreDKG.PrivateKey
	signer              types.Signer

	mpk *coreTypesDKG.MasterPublicKey
}

func newNode(privkey *ecdsa.PrivateKey, signer types.Signer) *node {
	k := coreEcdsa.NewPrivateKeyFromECDSA(privkey)
	id := coreTypes.NewNodeID(k.PublicKey())
	return &node{
		cryptoKey: k,
		ecdsaKey:  privkey,
		id:        id,
		dkgid:     coreDKG.NewID(id.Bytes()),
		address:   crypto.PubkeyToAddress(privkey.PublicKey),
		signer:    signer,
	}
}

func (n *node) ID() coreTypes.NodeID {
	return n.id
}

func (n *node) DKGID() coreDKG.ID {
	return n.dkgid
}

// return signed dkg master public key
func (n *node) MasterPublicKeyTx(round uint64, nonce uint64) *types.Transaction {
	mpk := &coreTypesDKG.MasterPublicKey{
		ProposerID:      n.ID(),
		Round:           round,
		DKGID:           n.DKGID(),
		PublicKeyShares: *n.pubShares,
	}

	binaryRound := make([]byte, 8)
	binary.LittleEndian.PutUint64(binaryRound, mpk.Round)

	hash := crypto.Keccak256Hash(
		mpk.ProposerID.Hash[:],
		mpk.DKGID.GetLittleEndian(),
		mpk.PublicKeyShares.MasterKeyBytes(),
		binaryRound,
	)

	var err error
	mpk.Signature, err = n.cryptoKey.Sign(coreCommon.Hash(hash))
	if err != nil {
		panic(err)
	}

	method := vm.GovernanceContractName2Method["addDKGMasterPublicKey"]
	encoded, err := rlp.EncodeToBytes(mpk)
	if err != nil {
		panic(err)
	}

	res, err := method.Inputs.Pack(big.NewInt(int64(round)), encoded)
	if err != nil {
		panic(err)
	}
	data := append(method.Id(), res...)
	return n.CreateGovTx(nonce, data)
}

func (n *node) DKGFinalizeTx(round uint64, nonce uint64) *types.Transaction {
	final := coreTypesDKG.Finalize{
		ProposerID: n.ID(),
		Round:      round,
	}
	binaryRound := make([]byte, 8)
	binary.LittleEndian.PutUint64(binaryRound, final.Round)
	hash := crypto.Keccak256Hash(
		final.ProposerID.Hash[:],
		binaryRound,
	)

	var err error
	final.Signature, err = n.cryptoKey.Sign(coreCommon.Hash(hash))
	if err != nil {
		panic(err)
	}

	method := vm.GovernanceContractName2Method["addDKGFinalize"]

	encoded, err := rlp.EncodeToBytes(final)
	if err != nil {
		panic(err)
	}

	res, err := method.Inputs.Pack(big.NewInt(int64(round)), encoded)
	if err != nil {
		panic(err)
	}

	data := append(method.Id(), res...)
	return n.CreateGovTx(nonce, data)
}

func (n *node) CreateGovTx(nonce uint64, data []byte) *types.Transaction {
	tx, err := types.SignTx(types.NewTransaction(
		nonce,
		vm.GovernanceContractAddress,
		big.NewInt(0),
		uint64(2000000),
		big.NewInt(1e10),
		data), n.signer, n.ecdsaKey)
	if err != nil {
		panic(err)
	}
	return tx
}

type NodeSet struct {
	signer    types.Signer
	privkeys  []*ecdsa.PrivateKey
	nodes     map[uint64][]*node
	crs       map[uint64]common.Hash
	signedCRS map[uint64][]byte
}

func NewNodeSet(round uint64, crs common.Hash, signer types.Signer,
	privkeys []*ecdsa.PrivateKey) *NodeSet {
	n := &NodeSet{
		signer:    signer,
		privkeys:  privkeys,
		nodes:     make(map[uint64][]*node),
		crs:       make(map[uint64]common.Hash),
		signedCRS: make(map[uint64][]byte),
	}
	n.crs[round] = crs
	n.RunDKG(round, 2)
	return n
}

func (n *NodeSet) CRS(round uint64) common.Hash {
	if c, ok := n.crs[round]; ok {
		return c
	}
	panic("crs not exist")
}

// Assume All nodes in NodeSet are in DKG Set too.
func (n *NodeSet) RunDKG(round uint64, threshold int) {
	var ids coreDKG.IDs
	var nodes []*node
	for _, key := range n.privkeys {
		node := newNode(key, n.signer)
		nodes = append(nodes, node)
		ids = append(ids, node.DKGID())
	}

	for _, node := range nodes {
		node.prvShares, node.pubShares = coreDKG.NewPrivateKeyShares(threshold)
		node.prvShares.SetParticipants(ids)
		node.receivedPrvShares = coreDKG.NewEmptyPrivateKeyShares()
	}

	// exchange keys
	for _, sender := range nodes {
		for _, receiver := range nodes {
			// no need to verify
			prvShare, ok := sender.prvShares.Share(receiver.DKGID())
			if !ok {
				panic("not ok")
			}
			receiver.receivedPrvShares.AddShare(sender.DKGID(), prvShare)
		}
	}

	// recover private key
	for _, node := range nodes {
		privKey, err := node.receivedPrvShares.RecoverPrivateKey(ids)
		if err != nil {
			panic(err)
		}
		node.recoveredPrivateKey = privKey
	}

	// store these nodes
	n.nodes[round] = nodes
}

func (n *NodeSet) Randomness(round uint64, hash common.Hash) []byte {
	if round == 0 {
		return []byte{}
	}
	return n.TSig(round-1, hash)
}

func (n *NodeSet) SignedCRS(round uint64) {
	signedCRS := n.TSig(round, n.crs[round])
	n.signedCRS[round+1] = signedCRS
	n.crs[round+1] = crypto.Keccak256Hash(signedCRS)
}

func (n *NodeSet) TSig(round uint64, hash common.Hash) []byte {
	var ids coreDKG.IDs
	var psigs []coreDKG.PartialSignature
	for _, node := range n.nodes[round] {
		ids = append(ids, node.DKGID())
	}
	for _, node := range n.nodes[round] {
		sig, err := node.recoveredPrivateKey.Sign(coreCommon.Hash(hash))
		if err != nil {
			panic(err)
		}
		psigs = append(psigs, coreDKG.PartialSignature(sig))
		// ids = append(ids, node.DKGID())

		// FIXME: Debug verify signature
		pk := coreDKG.NewEmptyPublicKeyShares()
		for _, nnode := range n.nodes[round] {
			p, err := nnode.pubShares.Share(node.DKGID())
			if err != nil {
				panic(err)
			}
			err = pk.AddShare(nnode.DKGID(), p)
			if err != nil {
				panic(err)
			}
		}

		recovered, err := pk.RecoverPublicKey(ids)
		if err != nil {
			panic(err)
		}

		if !recovered.VerifySignature(coreCommon.Hash(hash), sig) {
			panic("##########can not verify signature")
		}
	}

	sig, err := coreDKG.RecoverSignature(psigs, ids)
	if err != nil {
		panic(err)
	}
	return sig.Signature
}

func (n *NodeSet) CRSTx(round uint64, b *BlockGen) *types.Transaction {
	method := vm.GovernanceContractName2Method["proposeCRS"]
	res, err := method.Inputs.Pack(big.NewInt(int64(round)), n.signedCRS[round])
	if err != nil {
		panic(err)
	}
	data := append(method.Id(), res...)

	node := n.nodes[round-1][0]
	return node.CreateGovTx(b.TxNonce(node.address), data)
}

func (n *NodeSet) NotifyRoundHeightTx(round, height uint64,
	b *BlockGen) *types.Transaction {
	method := vm.GovernanceContractName2Method["snapshotRound"]
	res, err := method.Inputs.Pack(
		big.NewInt(int64(round)), big.NewInt(int64(height)))
	if err != nil {
		panic(err)
	}
	data := append(method.Id(), res...)

	var r uint64
	if round < 1 {
		r = 0
	} else {
		r = round - 1
	}
	node := n.nodes[r][0]
	return node.CreateGovTx(b.TxNonce(node.address), data)
}

// Copy from dexon consensus core
// TODO(sonic): polish this
func hashBlock(block *coreTypes.Block) (common.Hash, error) {
	hashPosition := hashPosition(block.Position)
	// Handling Block.Acks.
	binaryAcks := make([][]byte, len(block.Acks))
	for idx, ack := range block.Acks {
		binaryAcks[idx] = ack[:]
	}
	hashAcks := crypto.Keccak256Hash(binaryAcks...)
	binaryTimestamp, err := block.Timestamp.UTC().MarshalBinary()
	if err != nil {
		return common.Hash{}, err
	}
	binaryWitness, err := hashWitness(&block.Witness)
	if err != nil {
		return common.Hash{}, err
	}

	hash := crypto.Keccak256Hash(
		block.ProposerID.Hash[:],
		block.ParentHash[:],
		hashPosition[:],
		hashAcks[:],
		binaryTimestamp[:],
		block.PayloadHash[:],
		binaryWitness[:])
	return hash, nil
}

func hashPosition(position coreTypes.Position) common.Hash {
	binaryChainID := make([]byte, 4)
	binary.LittleEndian.PutUint32(binaryChainID, position.ChainID)

	binaryHeight := make([]byte, 8)
	binary.LittleEndian.PutUint64(binaryHeight, position.Height)

	return crypto.Keccak256Hash(
		binaryChainID,
		binaryHeight,
	)
}

func hashWitness(witness *coreTypes.Witness) (common.Hash, error) {
	binaryHeight := make([]byte, 8)
	binary.LittleEndian.PutUint64(binaryHeight, witness.Height)
	return crypto.Keccak256Hash(
		binaryHeight,
		witness.Data), nil
}
