package dexcon

import (
	"crypto/ecdsa"
	"math/big"

	coreCommon "github.com/dexon-foundation/dexon-consensus/common"
	dexCore "github.com/dexon-foundation/dexon-consensus/core"
	coreCrypto "github.com/dexon-foundation/dexon-consensus/core/crypto"
	coreDKG "github.com/dexon-foundation/dexon-consensus/core/crypto/dkg"
	coreEcdsa "github.com/dexon-foundation/dexon-consensus/core/crypto/ecdsa"
	coreTypes "github.com/dexon-foundation/dexon-consensus/core/types"
	coreTypesDKG "github.com/dexon-foundation/dexon-consensus/core/types/dkg"
	coreUtils "github.com/dexon-foundation/dexon-consensus/core/utils"

	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/consensus"
	"github.com/dexon-foundation/dexon/core/types"
	"github.com/dexon-foundation/dexon/core/vm"
	"github.com/dexon-foundation/dexon/crypto"
	"github.com/dexon-foundation/dexon/rlp"
)

type FakeDexcon struct {
	*Dexcon
	nodes *NodeSet
}

func NewFaker(nodes *NodeSet) *FakeDexcon {
	return &FakeDexcon{
		Dexcon: New(),
		nodes:  nodes,
	}
}

func (f *FakeDexcon) Prepare(chain consensus.ChainReader, header *types.Header) error {
	var coreBlock coreTypes.Block
	if err := rlp.DecodeBytes(header.DexconMeta, &coreBlock); err != nil {
		return err
	}

	blockHash, err := coreUtils.HashBlock(&coreBlock)
	if err != nil {
		return err
	}

	parentHeader := chain.GetHeaderByNumber(header.Number.Uint64() - 1)
	var parentCoreBlock coreTypes.Block
	if parentHeader.Number.Uint64() != 0 {
		if err := rlp.DecodeBytes(
			parentHeader.DexconMeta, &parentCoreBlock); err != nil {
			return err
		}
	}

	randomness := f.nodes.Randomness(header.Round, common.Hash(blockHash))
	coreBlock.Randomness = randomness

	dexconMeta, err := rlp.EncodeToBytes(&coreBlock)
	if err != nil {
		return err
	}
	header.DexconMeta = dexconMeta
	return nil
}

type Node struct {
	cryptoKey coreCrypto.PrivateKey
	ecdsaKey  *ecdsa.PrivateKey

	id                  coreTypes.NodeID
	dkgid               coreDKG.ID
	address             common.Address
	prvShares           *coreDKG.PrivateKeyShares
	pubShares           *coreDKG.PublicKeyShares
	receivedPrvShares   *coreDKG.PrivateKeyShares
	recoveredPrivateKey *coreDKG.PrivateKey
	signer              *coreUtils.Signer
	txSigner            types.Signer

	mpk *coreTypesDKG.MasterPublicKey
}

func newNode(privkey *ecdsa.PrivateKey, txSigner types.Signer) *Node {
	k := coreEcdsa.NewPrivateKeyFromECDSA(privkey)
	id := coreTypes.NewNodeID(k.PublicKey())
	return &Node{
		cryptoKey: k,
		ecdsaKey:  privkey,
		id:        id,
		dkgid:     coreDKG.NewID(id.Bytes()),
		address:   crypto.PubkeyToAddress(privkey.PublicKey),
		signer:    coreUtils.NewSigner(k),
		txSigner:  txSigner,
	}
}

func (n *Node) ID() coreTypes.NodeID    { return n.id }
func (n *Node) DKGID() coreDKG.ID       { return n.dkgid }
func (n *Node) Address() common.Address { return n.address }

func (n *Node) MasterPublicKey(round uint64) *coreTypesDKG.MasterPublicKey {
	mpk := &coreTypesDKG.MasterPublicKey{
		ProposerID:      n.ID(),
		Round:           round,
		DKGID:           n.DKGID(),
		PublicKeyShares: *n.pubShares,
	}

	if err := n.signer.SignDKGMasterPublicKey(mpk); err != nil {
		panic(err)
	}
	return mpk
}

func (n *Node) DKGMPKReady(round uint64) *coreTypesDKG.MPKReady {
	ready := &coreTypesDKG.MPKReady{
		ProposerID: n.ID(),
		Round:      round,
	}

	if err := n.signer.SignDKGMPKReady(ready); err != nil {
		panic(err)
	}
	return ready
}

func (n *Node) DKGFinalize(round uint64) *coreTypesDKG.Finalize {
	final := &coreTypesDKG.Finalize{
		ProposerID: n.ID(),
		Round:      round,
	}

	if err := n.signer.SignDKGFinalize(final); err != nil {
		panic(err)
	}
	return final
}

func (n *Node) CreateGovTx(nonce uint64, data []byte) *types.Transaction {
	tx, err := types.SignTx(types.NewTransaction(
		nonce,
		vm.GovernanceContractAddress,
		big.NewInt(0),
		uint64(2000000),
		big.NewInt(1e10),
		data), n.txSigner, n.ecdsaKey)
	if err != nil {
		panic(err)
	}
	return tx
}

type NodeSet struct {
	signer    types.Signer
	privkeys  []*ecdsa.PrivateKey
	nodes     map[uint64][]*Node
	crs       map[uint64]common.Hash
	signedCRS map[uint64][]byte
}

func NewNodeSet(round uint64, signedCRS []byte, signer types.Signer,
	privkeys []*ecdsa.PrivateKey) *NodeSet {
	n := &NodeSet{
		signer:    signer,
		privkeys:  privkeys,
		nodes:     make(map[uint64][]*Node),
		crs:       make(map[uint64]common.Hash),
		signedCRS: make(map[uint64][]byte),
	}
	n.signedCRS[round] = signedCRS
	n.crs[round] = crypto.Keccak256Hash(signedCRS)
	return n
}

func (n *NodeSet) Nodes(round uint64) []*Node {
	if nodes, ok := n.nodes[round]; ok {
		return nodes
	}
	panic("dkg not ready")
}

func (n *NodeSet) CRS(round uint64) common.Hash {
	if c, ok := n.crs[round]; ok {
		return c
	}
	panic("crs not exist")
}

func (n *NodeSet) SignedCRS(round uint64) []byte {
	if c, ok := n.signedCRS[round]; ok {
		return c
	}
	panic("signedCRS not exist")
}

// Assume All nodes in NodeSet are in DKG Set too.
func (n *NodeSet) RunDKG(round uint64, threshold int) {
	var ids coreDKG.IDs
	var nodes []*Node
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
	return n.TSig(round, hash)
}

func (n *NodeSet) SignCRS(round uint64) {
	var signedCRS []byte
	if round < dexCore.DKGDelayRound {
		signedCRS = crypto.Keccak256(n.signedCRS[round])
	} else {
		signedCRS = n.TSig(round, n.crs[round])
	}
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
