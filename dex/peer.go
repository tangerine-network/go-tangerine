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

package dex

import (
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	mapset "github.com/deckarep/golang-set"
	coreTypes "github.com/dexon-foundation/dexon-consensus-core/core/types"

	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/core/types"
	"github.com/dexon-foundation/dexon/log"
	"github.com/dexon-foundation/dexon/p2p"
	"github.com/dexon-foundation/dexon/p2p/enode"
	"github.com/dexon-foundation/dexon/p2p/enr"
	"github.com/dexon-foundation/dexon/rlp"
)

var (
	errClosed            = errors.New("peer set is closed")
	errAlreadyRegistered = errors.New("peer is already registered")
	errNotRegistered     = errors.New("peer is not registered")
)

const (
	maxKnownTxs    = 32768 // Maximum transactions hashes to keep in the known list (prevent DOS)
	maxKnownMetas  = 32768 // Maximum metas hashes to keep in the known list (prevent DOS)
	maxKnownBlocks = 1024  // Maximum block hashes to keep in the known list (prevent DOS)

	maxKnownLatticeBLocks       = 2048
	maxKnownVotes               = 2048
	maxKnownAgreements          = 10240
	maxKnownRandomnesses        = 10240
	maxKnownDKGPrivateShare     = 1024 // this related to DKG Size
	maxKnownDKGPartialSignature = 1024 // this related to DKG Size

	// maxQueuedTxs is the maximum number of transaction lists to queue up before
	// dropping broadcasts. This is a sensitive number as a transaction list might
	// contain a single transaction, or thousands.
	maxQueuedTxs = 128

	maxQueuedMetas = 512

	// maxQueuedProps is the maximum number of block propagations to queue up before
	// dropping broadcasts. There's not much point in queueing stale blocks, so a few
	// that might cover uncles should be enough.
	maxQueuedProps = 4

	// maxQueuedAnns is the maximum number of block announcements to queue up before
	// dropping broadcasts. Similarly to block propagations, there's no point to queue
	// above some healthy uncle limit, so use that.
	maxQueuedAnns = 4

	maxQueuedLatticeBlocks        = 16
	maxQueuedVotes                = 128
	maxQueuedAgreements           = 16
	maxQueuedRandomnesses         = 16
	maxQueuedDKGPrivateShare      = 16
	maxQueuedDKGParitialSignature = 16

	handshakeTimeout = 5 * time.Second

	groupNodeNum = 3
)

// PeerInfo represents a short summary of the Ethereum sub-protocol metadata known
// about a connected peer.
type PeerInfo struct {
	Version    int      `json:"version"`    // Ethereum protocol version negotiated
	Difficulty *big.Int `json:"difficulty"` // Total difficulty of the peer's blockchain
	Head       string   `json:"head"`       // SHA3 hash of the peer's best owned block
}

// propEvent is a block propagation, waiting for its turn in the broadcast queue.
type propEvent struct {
	block *types.Block
	td    *big.Int
}

type setType uint32

const (
	dkgset = iota
	notaryset
)

type peerLabel struct {
	set     setType
	chainID uint32
	round   uint64
}

type peer struct {
	id string

	*p2p.Peer
	rw p2p.MsgReadWriter

	version int // Protocol version negotiated

	head common.Hash
	td   *big.Int
	lock sync.RWMutex

	knownTxs                   mapset.Set // Set of transaction hashes known to be known by this peer
	knownMetas                 mapset.Set // Set of node metas known to be known by this peer
	knownBlocks                mapset.Set // Set of block hashes known to be known by this peer
	knownLatticeBlocks         mapset.Set
	knownVotes                 mapset.Set
	knownAgreements            mapset.Set
	knownRandomnesses          mapset.Set
	knownDKGPrivateShares      mapset.Set
	knownDKGPartialSignatures  mapset.Set
	queuedTxs                  chan []*types.Transaction // Queue of transactions to broadcast to the peer
	queuedMetas                chan []*NodeMeta          // Queue of node metas to broadcast to the peer
	queuedProps                chan *propEvent           // Queue of blocks to broadcast to the peer
	queuedAnns                 chan *types.Block         // Queue of blocks to announce to the peer
	queuedLatticeBlocks        chan *coreTypes.Block
	queuedVotes                chan *coreTypes.Vote
	queuedAgreements           chan *coreTypes.AgreementResult
	queuedRandomnesses         chan *coreTypes.BlockRandomnessResult
	queuedDKGPrivateShares     chan *coreTypes.DKGPrivateShare
	queuedDKGPartialSignatures chan *coreTypes.DKGPartialSignature
	term                       chan struct{} // Termination channel to stop the broadcaster
}

func newPeer(version int, p *p2p.Peer, rw p2p.MsgReadWriter) *peer {
	return &peer{
		Peer:                       p,
		rw:                         rw,
		version:                    version,
		id:                         p.ID().String(),
		knownTxs:                   mapset.NewSet(),
		knownMetas:                 mapset.NewSet(),
		knownBlocks:                mapset.NewSet(),
		knownLatticeBlocks:         mapset.NewSet(),
		knownVotes:                 mapset.NewSet(),
		knownAgreements:            mapset.NewSet(),
		knownRandomnesses:          mapset.NewSet(),
		knownDKGPrivateShares:      mapset.NewSet(),
		knownDKGPartialSignatures:  mapset.NewSet(),
		queuedTxs:                  make(chan []*types.Transaction, maxQueuedTxs),
		queuedMetas:                make(chan []*NodeMeta, maxQueuedMetas),
		queuedProps:                make(chan *propEvent, maxQueuedProps),
		queuedAnns:                 make(chan *types.Block, maxQueuedAnns),
		queuedLatticeBlocks:        make(chan *coreTypes.Block, maxQueuedLatticeBlocks),
		queuedVotes:                make(chan *coreTypes.Vote, maxQueuedVotes),
		queuedAgreements:           make(chan *coreTypes.AgreementResult, maxQueuedAgreements),
		queuedRandomnesses:         make(chan *coreTypes.BlockRandomnessResult, maxQueuedRandomnesses),
		queuedDKGPrivateShares:     make(chan *coreTypes.DKGPrivateShare, maxQueuedDKGPrivateShare),
		queuedDKGPartialSignatures: make(chan *coreTypes.DKGPartialSignature, maxQueuedDKGParitialSignature),
		term: make(chan struct{}),
	}
}

// broadcast is a write loop that multiplexes block propagations, announcements,
// transaction and notary node metas broadcasts into the remote peer.
// The goal is to have an async writer that does not lock up node internals.
func (p *peer) broadcast() {
	for {
		select {
		case txs := <-p.queuedTxs:
			if err := p.SendTransactions(txs); err != nil {
				return
			}
			p.Log().Trace("Broadcast transactions", "count", len(txs))

		case metas := <-p.queuedMetas:
			if err := p.SendNodeMetas(metas); err != nil {
				return
			}
			p.Log().Trace("Broadcast node metas", "count", len(metas))

		case prop := <-p.queuedProps:
			if err := p.SendNewBlock(prop.block, prop.td); err != nil {
				return
			}
			p.Log().Trace("Propagated block", "number", prop.block.Number(), "hash", prop.block.Hash(), "td", prop.td)

		case block := <-p.queuedAnns:
			if err := p.SendNewBlockHashes([]common.Hash{block.Hash()}, []uint64{block.NumberU64()}); err != nil {
				return
			}
			p.Log().Trace("Announced block", "number", block.Number(), "hash", block.Hash())
		case block := <-p.queuedLatticeBlocks:
			if err := p.SendLatticeBlock(block); err != nil {
				return
			}
			p.Log().Trace("Broadcast lattice block")
		case vote := <-p.queuedVotes:
			if err := p.SendVote(vote); err != nil {
				return
			}
			p.Log().Trace("Broadcast vote", "vote", vote.String(), "hash", rlpHash(vote))
		case agreement := <-p.queuedAgreements:
			if err := p.SendAgreement(agreement); err != nil {
				return
			}
			p.Log().Trace("Broadcast agreement")
		case randomness := <-p.queuedRandomnesses:
			if err := p.SendRandomness(randomness); err != nil {
				return
			}
			p.Log().Trace("Broadcast randomness")
		case privateShare := <-p.queuedDKGPrivateShares:
			if err := p.SendDKGPrivateShare(privateShare); err != nil {
				return
			}
			p.Log().Trace("Broadcast DKG private share")
		case psig := <-p.queuedDKGPartialSignatures:
			if err := p.SendDKGPartialSignature(psig); err != nil {
				return
			}
			p.Log().Trace("Broadcast DKG partial signature")
		case <-p.term:
			return
		}
	}
}

// close signals the broadcast goroutine to terminate.
func (p *peer) close() {
	close(p.term)
}

// Info gathers and returns a collection of metadata known about a peer.
func (p *peer) Info() *PeerInfo {
	hash, td := p.Head()

	return &PeerInfo{
		Version:    p.version,
		Difficulty: td,
		Head:       hash.Hex(),
	}
}

// Head retrieves a copy of the current head hash and total difficulty of the
// peer.
func (p *peer) Head() (hash common.Hash, td *big.Int) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	copy(hash[:], p.head[:])
	return hash, new(big.Int).Set(p.td)
}

// SetHead updates the head hash and total difficulty of the peer.
func (p *peer) SetHead(hash common.Hash, td *big.Int) {
	p.lock.Lock()
	defer p.lock.Unlock()

	copy(p.head[:], hash[:])
	p.td.Set(td)
}

// MarkBlock marks a block as known for the peer, ensuring that the block will
// never be propagated to this particular peer.
func (p *peer) MarkBlock(hash common.Hash) {
	// If we reached the memory allowance, drop a previously known block hash
	for p.knownBlocks.Cardinality() >= maxKnownBlocks {
		p.knownBlocks.Pop()
	}
	p.knownBlocks.Add(hash)
}

// MarkTransaction marks a transaction as known for the peer, ensuring that it
// will never be propagated to this particular peer.
func (p *peer) MarkTransaction(hash common.Hash) {
	// If we reached the memory allowance, drop a previously known transaction hash
	for p.knownTxs.Cardinality() >= maxKnownTxs {
		p.knownTxs.Pop()
	}
	p.knownTxs.Add(hash)
}

func (p *peer) MarkNodeMeta(hash common.Hash) {
	for p.knownMetas.Cardinality() >= maxKnownMetas {
		p.knownMetas.Pop()
	}
	p.knownMetas.Add(hash)
}

// SendTransactions sends transactions to the peer and includes the hashes
// in its transaction hash set for future reference.
func (p *peer) SendTransactions(txs types.Transactions) error {
	for _, tx := range txs {
		p.knownTxs.Add(tx.Hash())
	}
	return p2p.Send(p.rw, TxMsg, txs)
}

// AsyncSendTransactions queues list of transactions propagation to a remote
// peer. If the peer's broadcast queue is full, the event is silently dropped.
func (p *peer) AsyncSendTransactions(txs []*types.Transaction) {
	select {
	case p.queuedTxs <- txs:
		for _, tx := range txs {
			p.knownTxs.Add(tx.Hash())
		}
	default:
		p.Log().Debug("Dropping transaction propagation", "count", len(txs))
	}
}

// SendNodeMetas sends the metas to the peer and includes the hashes
// in its metas hash set for future reference.
func (p *peer) SendNodeMetas(metas []*NodeMeta) error {
	for _, meta := range metas {
		p.knownMetas.Add(meta.Hash())
	}
	return p2p.Send(p.rw, MetaMsg, metas)
}

// AsyncSendNodeMeta queues list of notary node meta propagation to a
// remote peer. If the peer's broadcast queue is full, the event is silently
// dropped.
func (p *peer) AsyncSendNodeMetas(metas []*NodeMeta) {
	select {
	case p.queuedMetas <- metas:
		for _, meta := range metas {
			p.knownMetas.Add(meta.Hash())
		}
	default:
		p.Log().Debug("Dropping node meta propagation", "count", len(metas))
	}
}

// SendNewBlockHashes announces the availability of a number of blocks through
// a hash notification.
func (p *peer) SendNewBlockHashes(hashes []common.Hash, numbers []uint64) error {
	for _, hash := range hashes {
		p.knownBlocks.Add(hash)
	}
	request := make(newBlockHashesData, len(hashes))
	for i := 0; i < len(hashes); i++ {
		request[i].Hash = hashes[i]
		request[i].Number = numbers[i]
	}
	return p2p.Send(p.rw, NewBlockHashesMsg, request)
}

// AsyncSendNewBlockHash queues the availability of a block for propagation to a
// remote peer. If the peer's broadcast queue is full, the event is silently
// dropped.
func (p *peer) AsyncSendNewBlockHash(block *types.Block) {
	select {
	case p.queuedAnns <- block:
		p.knownBlocks.Add(block.Hash())
	default:
		p.Log().Debug("Dropping block announcement", "number", block.NumberU64(), "hash", block.Hash())
	}
}

// SendNewBlock propagates an entire block to a remote peer.
func (p *peer) SendNewBlock(block *types.Block, td *big.Int) error {
	p.knownBlocks.Add(block.Hash())
	return p2p.Send(p.rw, NewBlockMsg, []interface{}{block, td})
}

// AsyncSendNewBlock queues an entire block for propagation to a remote peer. If
// the peer's broadcast queue is full, the event is silently dropped.
func (p *peer) AsyncSendNewBlock(block *types.Block, td *big.Int) {
	select {
	case p.queuedProps <- &propEvent{block: block, td: td}:
		p.knownBlocks.Add(block.Hash())
	default:
		p.Log().Debug("Dropping block propagation", "number", block.NumberU64(), "hash", block.Hash())
	}
}

func (p *peer) SendLatticeBlock(block *coreTypes.Block) error {
	p.knownLatticeBlocks.Add(rlpHash(block))
	return p2p.Send(p.rw, LatticeBlockMsg, block)
}

func (p *peer) AsyncSendLatticeBlock(block *coreTypes.Block) {
	select {
	case p.queuedLatticeBlocks <- block:
		p.knownLatticeBlocks.Add(rlpHash(block))
	default:
		p.Log().Debug("Dropping lattice block propagation")
	}
}

func (p *peer) SendVote(vote *coreTypes.Vote) error {
	p.knownVotes.Add(rlpHash(vote))
	return p2p.Send(p.rw, VoteMsg, vote)
}

func (p *peer) AsyncSendVote(vote *coreTypes.Vote) {
	select {
	case p.queuedVotes <- vote:
		p.knownVotes.Add(rlpHash(vote))
	default:
		p.Log().Debug("Dropping vote propagation")
	}
}

func (p *peer) SendAgreement(agreement *coreTypes.AgreementResult) error {
	p.knownAgreements.Add(rlpHash(agreement))
	return p2p.Send(p.rw, AgreementMsg, agreement)
}

func (p *peer) AsyncSendAgreement(agreement *coreTypes.AgreementResult) {
	select {
	case p.queuedAgreements <- agreement:
		p.knownAgreements.Add(rlpHash(agreement))
	default:
		p.Log().Debug("Dropping agreement result")
	}
}

func (p *peer) SendRandomness(randomness *coreTypes.BlockRandomnessResult) error {
	p.knownRandomnesses.Add(rlpHash(randomness))
	return p2p.Send(p.rw, RandomnessMsg, randomness)
}

func (p *peer) AsyncSendRandomness(randomness *coreTypes.BlockRandomnessResult) {
	select {
	case p.queuedRandomnesses <- randomness:
		p.knownRandomnesses.Add(rlpHash(randomness))
	default:
		p.Log().Debug("Dropping randomness result")
	}
}

func (p *peer) SendDKGPrivateShare(privateShare *coreTypes.DKGPrivateShare) error {
	p.knownDKGPrivateShares.Add(rlpHash(privateShare))
	return p2p.Send(p.rw, DKGPrivateShareMsg, privateShare)
}

func (p *peer) AsyncSendDKGPrivateShare(privateShare *coreTypes.DKGPrivateShare) {
	select {
	case p.queuedDKGPrivateShares <- privateShare:
		p.knownDKGPrivateShares.Add(rlpHash(privateShare))
	default:
		p.Log().Debug("Dropping DKG private share")
	}
}

func (p *peer) SendDKGPartialSignature(psig *coreTypes.DKGPartialSignature) error {
	p.knownDKGPartialSignatures.Add(rlpHash(psig))
	return p2p.Send(p.rw, DKGPartialSignatureMsg, psig)
}

func (p *peer) AsyncSendDKGPartialSignature(psig *coreTypes.DKGPartialSignature) {
	select {
	case p.queuedDKGPartialSignatures <- psig:
		p.knownDKGPartialSignatures.Add(rlpHash(psig))
	default:
		p.Log().Debug("Dropping DKG partial signature")
	}
}

// SendBlockHeaders sends a batch of block headers to the remote peer.
func (p *peer) SendBlockHeaders(headers []*types.Header) error {
	return p2p.Send(p.rw, BlockHeadersMsg, headers)
}

// SendBlockBodies sends a batch of block contents to the remote peer.
func (p *peer) SendBlockBodies(bodies []*blockBody) error {
	return p2p.Send(p.rw, BlockBodiesMsg, blockBodiesData(bodies))
}

// SendBlockBodiesRLP sends a batch of block contents to the remote peer from
// an already RLP encoded format.
func (p *peer) SendBlockBodiesRLP(bodies []rlp.RawValue) error {
	return p2p.Send(p.rw, BlockBodiesMsg, bodies)
}

// SendNodeDataRLP sends a batch of arbitrary internal data, corresponding to the
// hashes requested.
func (p *peer) SendNodeData(data [][]byte) error {
	return p2p.Send(p.rw, NodeDataMsg, data)
}

// SendReceiptsRLP sends a batch of transaction receipts, corresponding to the
// ones requested from an already RLP encoded format.
func (p *peer) SendReceiptsRLP(receipts []rlp.RawValue) error {
	return p2p.Send(p.rw, ReceiptsMsg, receipts)
}

// RequestOneHeader is a wrapper around the header query functions to fetch a
// single header. It is used solely by the fetcher.
func (p *peer) RequestOneHeader(hash common.Hash) error {
	p.Log().Debug("Fetching single header", "hash", hash)
	return p2p.Send(p.rw, GetBlockHeadersMsg, &getBlockHeadersData{Origin: hashOrNumber{Hash: hash}, Amount: uint64(1), Skip: uint64(0), Reverse: false})
}

// RequestHeadersByHash fetches a batch of blocks' headers corresponding to the
// specified header query, based on the hash of an origin block.
func (p *peer) RequestHeadersByHash(origin common.Hash, amount int, skip int, reverse bool) error {
	p.Log().Debug("Fetching batch of headers", "count", amount, "fromhash", origin, "skip", skip, "reverse", reverse)
	return p2p.Send(p.rw, GetBlockHeadersMsg, &getBlockHeadersData{Origin: hashOrNumber{Hash: origin}, Amount: uint64(amount), Skip: uint64(skip), Reverse: reverse})
}

// RequestHeadersByNumber fetches a batch of blocks' headers corresponding to the
// specified header query, based on the number of an origin block.
func (p *peer) RequestHeadersByNumber(origin uint64, amount int, skip int, reverse bool) error {
	p.Log().Debug("Fetching batch of headers", "count", amount, "fromnum", origin, "skip", skip, "reverse", reverse)
	return p2p.Send(p.rw, GetBlockHeadersMsg, &getBlockHeadersData{Origin: hashOrNumber{Number: origin}, Amount: uint64(amount), Skip: uint64(skip), Reverse: reverse})
}

// RequestBodies fetches a batch of blocks' bodies corresponding to the hashes
// specified.
func (p *peer) RequestBodies(hashes []common.Hash) error {
	p.Log().Debug("Fetching batch of block bodies", "count", len(hashes))
	return p2p.Send(p.rw, GetBlockBodiesMsg, hashes)
}

// RequestNodeData fetches a batch of arbitrary data from a node's known state
// data, corresponding to the specified hashes.
func (p *peer) RequestNodeData(hashes []common.Hash) error {
	p.Log().Debug("Fetching batch of state data", "count", len(hashes))
	return p2p.Send(p.rw, GetNodeDataMsg, hashes)
}

// RequestReceipts fetches a batch of transaction receipts from a remote node.
func (p *peer) RequestReceipts(hashes []common.Hash) error {
	p.Log().Debug("Fetching batch of receipts", "count", len(hashes))
	return p2p.Send(p.rw, GetReceiptsMsg, hashes)
}

// Handshake executes the eth protocol handshake, negotiating version number,
// network IDs, difficulties, head and genesis blocks.
func (p *peer) Handshake(network uint64, td *big.Int, head common.Hash, genesis common.Hash) error {
	// Send out own handshake in a new thread
	errc := make(chan error, 2)
	var status statusData // safe to read after two values have been received from errc

	go func() {
		errc <- p2p.Send(p.rw, StatusMsg, &statusData{
			ProtocolVersion: uint32(p.version),
			NetworkId:       network,
			TD:              td,
			CurrentBlock:    head,
			GenesisBlock:    genesis,
		})
	}()
	go func() {
		errc <- p.readStatus(network, &status, genesis)
	}()
	timeout := time.NewTimer(handshakeTimeout)
	defer timeout.Stop()
	for i := 0; i < 2; i++ {
		select {
		case err := <-errc:
			if err != nil {
				return err
			}
		case <-timeout.C:
			return p2p.DiscReadTimeout
		}
	}
	p.td, p.head = status.TD, status.CurrentBlock
	return nil
}

func (p *peer) readStatus(network uint64, status *statusData, genesis common.Hash) (err error) {
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return err
	}
	if msg.Code != StatusMsg {
		return errResp(ErrNoStatusMsg, "first msg has code %x (!= %x)", msg.Code, StatusMsg)
	}
	if msg.Size > ProtocolMaxMsgSize {
		return errResp(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}
	// Decode the handshake and make sure everything matches
	if err := msg.Decode(&status); err != nil {
		return errResp(ErrDecode, "msg %v: %v", msg, err)
	}
	if status.GenesisBlock != genesis {
		return errResp(ErrGenesisBlockMismatch, "%x (!= %x)", status.GenesisBlock[:8], genesis[:8])
	}
	if status.NetworkId != network {
		return errResp(ErrNetworkIdMismatch, "%d (!= %d)", status.NetworkId, network)
	}
	if int(status.ProtocolVersion) != p.version {
		return errResp(ErrProtocolVersionMismatch, "%d (!= %d)", status.ProtocolVersion, p.version)
	}
	return nil
}

// String implements fmt.Stringer.
func (p *peer) String() string {
	return fmt.Sprintf("Peer %s [%s]", p.id,
		fmt.Sprintf("dex/%2d", p.version),
	)
}

// peerSet represents the collection of active peers currently participating in
// the Ethereum sub-protocol.
type peerSet struct {
	peers  map[string]*peer
	lock   sync.RWMutex
	closed bool
	tab    *nodeTable

	srvr          p2pServer
	gov           governance
	peer2Labels   map[string]map[peerLabel]struct{}
	label2Peers   map[peerLabel]map[string]struct{}
	history       map[uint64]struct{}
	notaryHistory map[uint64]struct{}
	dkgHistory    map[uint64]struct{}
}

// newPeerSet creates a new peer set to track the active participants.
func newPeerSet(gov governance, srvr p2pServer, tab *nodeTable) *peerSet {
	return &peerSet{
		peers:         make(map[string]*peer),
		gov:           gov,
		srvr:          srvr,
		tab:           tab,
		peer2Labels:   make(map[string]map[peerLabel]struct{}),
		label2Peers:   make(map[peerLabel]map[string]struct{}),
		history:       make(map[uint64]struct{}),
		notaryHistory: make(map[uint64]struct{}),
		dkgHistory:    make(map[uint64]struct{}),
	}
}

// Register injects a new peer into the working set, or returns an error if the
// peer is already known. If a new peer it registered, its broadcast loop is also
// started.
func (ps *peerSet) Register(p *peer) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if ps.closed {
		return errClosed
	}
	if _, ok := ps.peers[p.id]; ok {
		return errAlreadyRegistered
	}
	ps.peers[p.id] = p
	go p.broadcast()

	return nil
}

// Unregister removes a remote peer from the active set, disabling any further
// actions to/from that particular entity.
func (ps *peerSet) Unregister(id string) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	p, ok := ps.peers[id]
	if !ok {
		return errNotRegistered
	}
	delete(ps.peers, id)
	p.close()

	return nil
}

// Peer retrieves the registered peer with the given id.
func (ps *peerSet) Peer(id string) *peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return ps.peers[id]
}

// Len returns if the current number of peers in the set.
func (ps *peerSet) Len() int {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return len(ps.peers)
}

// PeersWithoutBlock retrieves a list of peers that do not have a given block in
// their set of known hashes.
func (ps *peerSet) PeersWithoutBlock(hash common.Hash) []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		if !p.knownBlocks.Contains(hash) {
			list = append(list, p)
		}
	}
	return list
}

// PeersWithoutTx retrieves a list of peers that do not have a given transaction
// in their set of known hashes.
func (ps *peerSet) PeersWithoutTx(hash common.Hash) []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		if !p.knownTxs.Contains(hash) {
			list = append(list, p)
		}
	}
	return list
}

func (ps *peerSet) PeersWithLabel(label peerLabel) []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()
	list := make([]*peer, 0, len(ps.label2Peers[label]))
	for id := range ps.label2Peers[label] {
		if p, ok := ps.peers[id]; ok {
			list = append(list, p)
		}
	}
	return list
}

func (ps *peerSet) PeersWithoutVote(hash common.Hash, label peerLabel) []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*peer, 0, len(ps.label2Peers[label]))
	for id := range ps.label2Peers[label] {
		if p, ok := ps.peers[id]; ok {
			if !p.knownVotes.Contains(hash) {
				list = append(list, p)
			}
		}
	}
	return list
}

// PeersWithoutNodeMeta retrieves a list of peers that do not have a
// given meta in their set of known hashes.
func (ps *peerSet) PeersWithoutNodeMeta(hash common.Hash) []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()
	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		if !p.knownMetas.Contains(hash) {
			list = append(list, p)
		}
	}
	return list
}

func (ps *peerSet) PeersWithoutLatticeBlock(hash common.Hash) []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()
	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		if !p.knownLatticeBlocks.Contains(hash) {
			list = append(list, p)
		}
	}
	return list
}

func (ps *peerSet) PeersWithoutAgreement(hash common.Hash) []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()
	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		if !p.knownAgreements.Contains(hash) {
			list = append(list, p)
		}
	}
	return list
}

func (ps *peerSet) PeersWithoutRandomness(hash common.Hash) []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()
	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		if !p.knownRandomnesses.Contains(hash) {
			list = append(list, p)
		}
	}
	return list
}

func (ps *peerSet) PeersWithoutDKGPartialSignature(hash common.Hash) []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()
	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		if !p.knownDKGPartialSignatures.Contains(hash) {
			list = append(list, p)
		}
	}
	return list
}

// BestPeer retrieves the known peer with the currently highest total difficulty.
func (ps *peerSet) BestPeer() *peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	var (
		bestPeer *peer
		bestTd   *big.Int
	)
	for _, p := range ps.peers {
		if _, td := p.Head(); bestPeer == nil || td.Cmp(bestTd) > 0 {
			bestPeer, bestTd = p, td
		}
	}
	return bestPeer
}

// Close disconnects all peers.
// No new peers can be registered after Close has returned.
func (ps *peerSet) Close() {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	for _, p := range ps.peers {
		p.Disconnect(p2p.DiscQuitting)
	}
	ps.closed = true
}

func (ps *peerSet) BuildConnection(round uint64) {
	ps.lock.Lock()
	defer ps.lock.Unlock()
	defer ps.dumpPeerLabel(fmt.Sprintf("BuildConnection: %d", round))

	ps.history[round] = struct{}{}

	selfID := ps.srvr.Self().ID().String()

	dkgIDs, err := ps.gov.DKGSet(round)
	if err != nil {
		log.Error("get dkg set fail", "round", round, "err", err)
	}

	// build dkg connection
	_, inDKGSet := dkgIDs[selfID]
	if inDKGSet {
		delete(dkgIDs, selfID)
		dkgLabel := peerLabel{set: dkgset, round: round}
		for id := range dkgIDs {
			ps.addDirectPeer(id, dkgLabel)
		}
	}

	var inOneNotarySet bool
	for cid := uint32(0); cid < ps.gov.GetNumChains(round); cid++ {
		notaryIDs, err := ps.gov.NotarySet(round, cid)
		if err != nil {
			log.Error("get notary set fail",
				"round", round, "chain id", cid, "err", err)
			continue
		}

		label := peerLabel{set: notaryset, chainID: cid, round: round}
		// not in notary set, add group
		if _, ok := notaryIDs[selfID]; !ok {
			var nodes []*enode.Node
			for id := range notaryIDs {
				nodes = append(nodes, ps.newNode(id))
				ps.addLabel(id, label)
			}
			ps.srvr.AddGroup(notarySetName(cid, round), nodes, groupNodeNum)
			continue
		}

		delete(notaryIDs, selfID)
		for id := range notaryIDs {
			ps.addDirectPeer(id, label)
		}
		inOneNotarySet = true
	}

	// build some connections to DKG nodes
	if !inDKGSet && inOneNotarySet {
		var nodes []*enode.Node
		label := peerLabel{set: dkgset, round: round}
		for id := range dkgIDs {
			nodes = append(nodes, ps.newNode(id))
			ps.addLabel(id, label)
		}
		ps.srvr.AddGroup(dkgSetName(round), nodes, groupNodeNum)
	}
}

func (ps *peerSet) ForgetConnection(round uint64) {
	ps.lock.Lock()
	defer ps.lock.Unlock()
	defer ps.dumpPeerLabel(fmt.Sprintf("ForgetConnection: %d", round))

	for r := range ps.history {
		if r <= round {
			ps.forgetConnection(round)
			delete(ps.history, r)
		}
	}
}

func (ps *peerSet) forgetConnection(round uint64) {
	selfID := ps.srvr.Self().ID().String()
	dkgIDs, err := ps.gov.DKGSet(round)
	if err != nil {
		log.Error("get dkg set fail", "round", round, "err", err)
	}

	_, inDKGSet := dkgIDs[selfID]
	if inDKGSet {
		delete(dkgIDs, selfID)
		label := peerLabel{set: dkgset, round: round}
		for id := range dkgIDs {
			ps.removeDirectPeer(id, label)
		}
	}

	var inOneNotarySet bool
	for cid := uint32(0); cid < ps.gov.GetNumChains(round); cid++ {
		notaryIDs, err := ps.gov.NotarySet(round, cid)
		if err != nil {
			log.Error("get notary set fail",
				"round", round, "chain id", cid, "err", err)
			continue
		}

		label := peerLabel{set: notaryset, chainID: cid, round: round}

		// not in notary set, add group
		if _, ok := notaryIDs[selfID]; !ok {
			var nodes []*enode.Node
			for id := range notaryIDs {
				nodes = append(nodes, ps.newNode(id))
				ps.removeLabel(id, label)
			}
			ps.srvr.RemoveGroup(notarySetName(cid, round))
			continue
		}

		delete(notaryIDs, selfID)
		for id := range notaryIDs {
			ps.removeDirectPeer(id, label)
		}
		inOneNotarySet = true
	}

	// build some connections to DKG nodes
	if !inDKGSet && inOneNotarySet {
		var nodes []*enode.Node
		label := peerLabel{set: dkgset, round: round}
		for id := range dkgIDs {
			nodes = append(nodes, ps.newNode(id))
			ps.removeLabel(id, label)
		}
		ps.srvr.RemoveGroup(dkgSetName(round))
	}
}

func (ps *peerSet) BuildNotaryConn(round uint64) {
	ps.lock.Lock()
	defer ps.lock.Unlock()
	defer ps.dumpPeerLabel(fmt.Sprintf("BuildNotaryConn: %d", round))

	if _, ok := ps.notaryHistory[round]; ok {
		return
	}

	ps.notaryHistory[round] = struct{}{}

	selfID := ps.srvr.Self().ID().String()
	for chainID := uint32(0); chainID < ps.gov.GetNumChains(round); chainID++ {
		s, err := ps.gov.NotarySet(round, chainID)
		if err != nil {
			log.Error("get notary set fail",
				"round", round, "chain id", chainID, "err", err)
			continue
		}

		// not in notary set, add group
		if _, ok := s[selfID]; !ok {
			var nodes []*enode.Node
			for id := range s {
				nodes = append(nodes, ps.newNode(id))
			}
			ps.srvr.AddGroup(notarySetName(chainID, round), nodes, groupNodeNum)
			continue
		}

		label := peerLabel{
			set:     notaryset,
			chainID: chainID,
			round:   round,
		}
		delete(s, selfID)
		for id := range s {
			ps.addDirectPeer(id, label)
		}
	}
}

func (ps *peerSet) dumpPeerLabel(s string) {
	log.Trace(s, "peer num", len(ps.peers))
	for id, labels := range ps.peer2Labels {
		_, ok := ps.peers[id]
		for label := range labels {
			log.Trace(s, "connected", ok, "id", id[:16],
				"round", label.round, "cid", label.chainID, "set", label.set)
		}
	}
}

func (ps *peerSet) ForgetNotaryConn(round uint64) {
	ps.lock.Lock()
	defer ps.lock.Unlock()
	defer ps.dumpPeerLabel(fmt.Sprintf("ForgetNotaryConn: %d", round))

	// forget all the rounds before the given round
	for r := range ps.notaryHistory {
		if r <= round {
			ps.forgetNotaryConn(r)
			delete(ps.notaryHistory, r)
		}
	}
}

func (ps *peerSet) forgetNotaryConn(round uint64) {
	selfID := ps.srvr.Self().ID().String()
	for chainID := uint32(0); chainID < ps.gov.GetNumChains(round); chainID++ {
		s, err := ps.gov.NotarySet(round, chainID)
		if err != nil {
			log.Error("get notary set fail",
				"round", round, "chain id", chainID, "err", err)
			continue
		}
		if _, ok := s[selfID]; !ok {
			ps.srvr.RemoveGroup(notarySetName(chainID, round))
			continue
		}

		label := peerLabel{
			set:     notaryset,
			chainID: chainID,
			round:   round,
		}
		delete(s, selfID)
		for id := range s {
			ps.removeDirectPeer(id, label)
		}
	}
}

func notarySetName(chainID uint32, round uint64) string {
	return fmt.Sprintf("%d-%d-notaryset", chainID, round)
}

func dkgSetName(round uint64) string {
	return fmt.Sprintf("%d-dkgset", round)
}

func (ps *peerSet) BuildDKGConn(round uint64) {
	ps.lock.Lock()
	defer ps.lock.Unlock()
	defer ps.dumpPeerLabel(fmt.Sprintf("BuildDKGConn: %d", round))
	selfID := ps.srvr.Self().ID().String()
	s, err := ps.gov.DKGSet(round)
	if err != nil {
		log.Error("get dkg set fail", "round", round)
		return
	}

	if _, ok := s[selfID]; !ok {
		return
	}
	ps.dkgHistory[round] = struct{}{}

	delete(s, selfID)
	for id := range s {
		ps.addDirectPeer(id, peerLabel{
			set:   dkgset,
			round: round,
		})
	}
}

func (ps *peerSet) ForgetDKGConn(round uint64) {
	ps.lock.Lock()
	defer ps.lock.Unlock()
	defer ps.dumpPeerLabel(fmt.Sprintf("ForgetDKGConn: %d", round))

	// forget all the rounds before the given round
	for r := range ps.dkgHistory {
		if r <= round {
			ps.forgetDKGConn(r)
			delete(ps.dkgHistory, r)
		}
	}
}

func (ps *peerSet) forgetDKGConn(round uint64) {
	selfID := ps.srvr.Self().ID().String()
	s, err := ps.gov.DKGSet(round)
	if err != nil {
		log.Error("get dkg set fail", "round", round)
		return
	}
	if _, ok := s[selfID]; !ok {
		return
	}

	delete(s, selfID)
	label := peerLabel{
		set:   dkgset,
		round: round,
	}
	for id := range s {
		ps.removeDirectPeer(id, label)
	}
}

// make sure the ps.lock is held
func (ps *peerSet) addDirectPeer(id string, label peerLabel) {
	ps.addLabel(id, label)
	ps.srvr.AddDirectPeer(ps.newNode(id))
}

// make sure the ps.lock is held
func (ps *peerSet) removeDirectPeer(id string, label peerLabel) {
	ps.removeLabel(id, label)
	if len(ps.peer2Labels[id]) == 0 {
		ps.srvr.RemoveDirectPeer(ps.newNode(id))
	}
}

// make sure the ps.lock is held
func (ps *peerSet) addLabel(id string, label peerLabel) {
	if _, ok := ps.peer2Labels[id]; !ok {
		ps.peer2Labels[id] = make(map[peerLabel]struct{})
	}
	if _, ok := ps.label2Peers[label]; !ok {
		ps.label2Peers[label] = make(map[string]struct{})
	}
	ps.peer2Labels[id][label] = struct{}{}
	ps.label2Peers[label][id] = struct{}{}
}

// make sure the ps.lock is held
func (ps *peerSet) removeLabel(id string, label peerLabel) {
	delete(ps.peer2Labels[id], label)
	delete(ps.label2Peers[label], id)
	if len(ps.peer2Labels[id]) == 0 {
		delete(ps.peer2Labels, id)
	}
	if len(ps.label2Peers[label]) == 0 {
		delete(ps.label2Peers, label)
	}
}

func (ps *peerSet) newNode(id string) *enode.Node {
	nodeID := enode.HexID(id)
	meta := ps.tab.Get(enode.HexID(id))

	var r enr.Record
	r.Set(enr.ID(nodeID.String()))
	r.Set(enr.IP(meta.IP))
	r.Set(enr.TCP(meta.TCP))
	r.Set(enr.UDP(meta.UDP))

	n, err := enode.New(enode.ValidSchemes, &r)
	if err != nil {
		panic(err)
	}
	return n
}
