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
	"github.com/dexon-foundation/dexon/p2p"
	"github.com/dexon-foundation/dexon/p2p/discover"
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
	labels  mapset.Set

	head common.Hash
	td   *big.Int
	lock sync.RWMutex

	knownTxs                  mapset.Set // Set of transaction hashes known to be known by this peer
	knownMetas                mapset.Set // Set of node metas known to be known by this peer
	knownBlocks               mapset.Set // Set of block hashes known to be known by this peer
	knownVotes                mapset.Set
	queuedTxs                 chan []*types.Transaction // Queue of transactions to broadcast to the peer
	queuedMetas               chan []*NodeMeta          // Queue of node metas to broadcast to the peer
	queuedProps               chan *propEvent           // Queue of blocks to broadcast to the peer
	queuedAnns                chan *types.Block         // Queue of blocks to announce to the peer
	queuedLatticeBlock        chan *coreTypes.Block
	queuedVote                chan *coreTypes.Vote
	queuedAgreement           chan *coreTypes.AgreementResult
	queuedRandomness          chan *coreTypes.BlockRandomnessResult
	queuedDKGPrivateShare     chan *coreTypes.DKGPrivateShare
	queuedDKGPartialSignature chan *coreTypes.DKGPartialSignature
	term                      chan struct{} // Termination channel to stop the broadcaster
}

func newPeer(version int, p *p2p.Peer, rw p2p.MsgReadWriter) *peer {
	return &peer{
		Peer:                      p,
		rw:                        rw,
		version:                   version,
		labels:                    mapset.NewSet(),
		id:                        p.ID().String(),
		knownTxs:                  mapset.NewSet(),
		knownMetas:                mapset.NewSet(),
		knownBlocks:               mapset.NewSet(),
		knownVotes:                mapset.NewSet(),
		queuedTxs:                 make(chan []*types.Transaction, maxQueuedTxs),
		queuedMetas:               make(chan []*NodeMeta, maxQueuedMetas),
		queuedProps:               make(chan *propEvent, maxQueuedProps),
		queuedAnns:                make(chan *types.Block, maxQueuedAnns),
		queuedLatticeBlock:        make(chan *coreTypes.Block, 16),
		queuedVote:                make(chan *coreTypes.Vote, 16),
		queuedAgreement:           make(chan *coreTypes.AgreementResult, 16),
		queuedRandomness:          make(chan *coreTypes.BlockRandomnessResult, 16),
		queuedDKGPrivateShare:     make(chan *coreTypes.DKGPrivateShare, 16),
		queuedDKGPartialSignature: make(chan *coreTypes.DKGPartialSignature, 16),
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
		case block := <-p.queuedLatticeBlock:
			if err := p.SendLatticeBlock(block); err != nil {
				return
			}
			p.Log().Trace("Broadcast lattice block")
		case vote := <-p.queuedVote:
			if err := p.SendVote(vote); err != nil {
				return
			}
			p.Log().Trace("Broadcast vote", "vote", vote.String(), "hash", rlpHash(vote))
		case agreement := <-p.queuedAgreement:
			if err := p.SendAgreement(agreement); err != nil {
				return
			}
			p.Log().Trace("Broadcast agreement")
		case randomness := <-p.queuedRandomness:
			if err := p.SendRandomness(randomness); err != nil {
				return
			}
			p.Log().Trace("Broadcast randomness")
		case privateShare := <-p.queuedDKGPrivateShare:
			if err := p.SendDKGPrivateShare(privateShare); err != nil {
				return
			}
			p.Log().Trace("Broadcast DKG private share")
		case psig := <-p.queuedDKGPartialSignature:
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

func (p *peer) addLabel(label peerLabel) {
	p.labels.Add(label)
}

func (p *peer) removeLabel(label peerLabel) {
	p.labels.Remove(label)
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
	return p2p.Send(p.rw, LatticeBlockMsg, toRLPLatticeBlock(block))
}

func (p *peer) AsyncSendLatticeBlock(block *coreTypes.Block) {
	select {
	case p.queuedLatticeBlock <- block:
	default:
		p.Log().Debug("Dropping lattice block propagation")
	}
}

func (p *peer) SendVote(vote *coreTypes.Vote) error {
	return p2p.Send(p.rw, VoteMsg, vote)
}

func (p *peer) AsyncSendVote(vote *coreTypes.Vote) {
	select {
	case p.queuedVote <- vote:
	default:
		p.Log().Debug("Dropping vote propagation")
	}
}

func (p *peer) SendAgreement(agreement *coreTypes.AgreementResult) error {
	return p2p.Send(p.rw, AgreementMsg, agreement)
}

func (p *peer) AsyncSendAgreement(agreement *coreTypes.AgreementResult) {
	select {
	case p.queuedAgreement <- agreement:
	default:
		p.Log().Debug("Dropping agreement result")
	}
}

func (p *peer) SendRandomness(randomness *coreTypes.BlockRandomnessResult) error {
	return p2p.Send(p.rw, RandomnessMsg, randomness)
}

func (p *peer) AsyncSendRandomness(randomness *coreTypes.BlockRandomnessResult) {
	select {
	case p.queuedRandomness <- randomness:
	default:
		p.Log().Debug("Dropping randomness result")
	}
}

func (p *peer) SendDKGPrivateShare(privateShare *coreTypes.DKGPrivateShare) error {
	return p2p.Send(p.rw, DKGPrivateShareMsg, toRLPDKGPrivateShare(privateShare))
}

func (p *peer) AsyncSendDKGPrivateShare(privateShare *coreTypes.DKGPrivateShare) {
	select {
	case p.queuedDKGPrivateShare <- privateShare:
	default:
		p.Log().Debug("Dropping DKG private share")
	}
}

func (p *peer) SendDKGPartialSignature(psig *coreTypes.DKGPartialSignature) error {
	return p2p.Send(p.rw, DKGPartialSignatureMsg, psig)
}

func (p *peer) AsyncSendDKGPartialSignature(psig *coreTypes.DKGPartialSignature) {
	select {
	case p.queuedDKGPartialSignature <- psig:
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

// TODO(sonic): finish the following dummy function.
func (ps *peerSet) PeersWithoutAgreement(hash common.Hash) []*peer {
	return ps.allPeers()
}

func (ps *peerSet) PeersWithoutRandomness(hash common.Hash) []*peer {
	return ps.allPeers()
}

func (ps *peerSet) PeersWithoutDKGPartialSignature(hash common.Hash) []*peer {
	return ps.allPeers()
}

func (ps *peerSet) PeersWithoutLatticeBlock(hash common.Hash) []*peer {
	return ps.allPeers()
}

func (ps *peerSet) allPeers() []*peer {
	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		list = append(list, p)
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

func (ps *peerSet) BuildNotaryConn(round uint64) {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if _, ok := ps.notaryHistory[round]; ok {
		return
	}

	ps.notaryHistory[round] = struct{}{}

	selfID := ps.srvr.Self().ID.String()
	for chainID := uint32(0); chainID < ps.gov.GetNumChains(round); chainID++ {
		s := ps.gov.NotarySet(chainID, round)

		// not in notary set, add group
		if _, ok := s[selfID]; !ok {
			var nodes []*discover.Node
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

func (ps *peerSet) ForgetNotaryConn(round uint64) {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	// forget all the rounds before the given round
	for r := range ps.notaryHistory {
		if r <= round {
			ps.forgetNotaryConn(r)
			delete(ps.notaryHistory, r)
		}
	}
}

func (ps *peerSet) forgetNotaryConn(round uint64) {
	selfID := ps.srvr.Self().ID.String()
	for chainID := uint32(0); chainID < ps.gov.GetNumChains(round); chainID++ {
		s := ps.gov.NotarySet(chainID, round)
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

func (ps *peerSet) BuildDKGConn(round uint64) {
	ps.lock.Lock()
	defer ps.lock.Unlock()
	selfID := ps.srvr.Self().ID.String()
	s := ps.gov.DKGSet(round)
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

	// forget all the rounds before the given round
	for r := range ps.dkgHistory {
		if r <= round {
			ps.forgetDKGConn(r)
			delete(ps.dkgHistory, r)
		}
	}
}

func (ps *peerSet) forgetDKGConn(round uint64) {
	selfID := ps.srvr.Self().ID.String()
	s := ps.gov.DKGSet(round)
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

// make sure the ps.lock is hold
func (ps *peerSet) addDirectPeer(id string, label peerLabel) {
	// if the peer exists add the label
	if p, ok := ps.peers[id]; ok {
		p.addLabel(label)
	}

	if _, ok := ps.peer2Labels[id]; !ok {
		ps.peer2Labels[id] = make(map[peerLabel]struct{})
	}

	if _, ok := ps.label2Peers[label]; !ok {
		ps.label2Peers[label] = make(map[string]struct{})
	}

	ps.peer2Labels[id][label] = struct{}{}
	ps.label2Peers[label][id] = struct{}{}
	ps.srvr.AddDirectPeer(ps.newNode(id))
}

// make sure the ps.lock is hold
func (ps *peerSet) removeDirectPeer(id string, label peerLabel) {
	if p, ok := ps.peers[id]; ok {
		p.removeLabel(label)
	}

	delete(ps.peer2Labels[id], label)

	if len(ps.peer2Labels[id]) == 0 {
		ps.srvr.RemoveDirectPeer(ps.newNode(id))
		delete(ps.peer2Labels, id)
	}

	if _, ok := ps.label2Peers[label]; ok {
		delete(ps.label2Peers[label], id)
		if len(ps.label2Peers[label]) == 0 {
			delete(ps.label2Peers, label)
		}
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
