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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	coreCommon "github.com/dexon-foundation/dexon-consensus/common"
	dexCore "github.com/dexon-foundation/dexon-consensus/core"
	coreCrypto "github.com/dexon-foundation/dexon-consensus/core/crypto"
	coreTypes "github.com/dexon-foundation/dexon-consensus/core/types"
	dkgTypes "github.com/dexon-foundation/dexon-consensus/core/types/dkg"

	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/consensus"
	"github.com/dexon-foundation/dexon/core"
	"github.com/dexon-foundation/dexon/core/types"
	"github.com/dexon-foundation/dexon/crypto"
	dexDB "github.com/dexon-foundation/dexon/dex/db"
	"github.com/dexon-foundation/dexon/dex/downloader"
	"github.com/dexon-foundation/dexon/dex/fetcher"
	"github.com/dexon-foundation/dexon/ethdb"
	"github.com/dexon-foundation/dexon/event"
	"github.com/dexon-foundation/dexon/log"
	"github.com/dexon-foundation/dexon/metrics"
	"github.com/dexon-foundation/dexon/p2p"
	"github.com/dexon-foundation/dexon/p2p/enode"
	"github.com/dexon-foundation/dexon/p2p/enr"
	"github.com/dexon-foundation/dexon/params"
	"github.com/dexon-foundation/dexon/rlp"
)

const (
	softResponseLimit = 2 * 1024 * 1024 // Target maximum size of returned blocks, headers or node data.
	estHeaderRlpSize  = 500             // Approximate size of an RLP encoded block header

	// txChanSize is the size of channel listening to NewTxsEvent.
	// The number is referenced from the size of tx pool.
	txChanSize = 4096

	finalizedBlockChanSize = 128

	recordChanSize = 10240

	maxPullPeers     = 3
	maxPullVotePeers = 1

	pullVoteRateLimit = 10 * time.Second
)

// errIncompatibleConfig is returned if the requested protocols and configs are
// not compatible (low protocol version restrictions and high requirements).
var errIncompatibleConfig = errors.New("incompatible configuration")

func errResp(code errCode, format string, v ...interface{}) error {
	return fmt.Errorf("%v - %v", code, fmt.Sprintf(format, v...))
}

type ProtocolManager struct {
	networkID uint64

	fastSync  uint32 // Flag whether fast sync is enabled (gets disabled if we already have blocks)
	acceptTxs uint32 // Flag whether we're considered synchronised (enables transaction processing)

	txpool       txPool
	nodeTable    *nodeTable
	gov          governance
	blockchain   *core.BlockChain
	chainconfig  *params.ChainConfig
	cache        *cache
	nextPullVote *sync.Map
	maxPeers     int

	downloader *downloader.Downloader
	fetcher    *fetcher.Fetcher
	peers      *peerSet

	SubProtocols []p2p.Protocol

	eventMux   *event.TypeMux
	txsCh      chan core.NewTxsEvent
	txsSub     event.Subscription
	recordsCh  chan newRecordsEvent
	recordsSub event.Subscription

	// channels for fetcher, syncer, txsyncLoop
	newPeerCh    chan *peer
	txsyncCh     chan *txsync
	recordsyncCh chan *recordsync
	quitSync     chan struct{}
	noMorePeers  chan struct{}

	// channels for peerSetLoop
	chainHeadCh  chan core.ChainHeadEvent
	chainHeadSub event.Subscription

	// channels for dexon consensus core
	receiveCh chan interface{}

	srvr p2pServer

	// wait group is used for graceful shutdowns during downloading
	// and processing
	wg sync.WaitGroup

	// Dexcon
	isBlockProposer bool
	app             dexconApp

	finalizedBlockCh  chan core.NewFinalizedBlockEvent
	finalizedBlockSub event.Subscription

	// metrics
	blockNumberGauge metrics.Gauge
}

// NewProtocolManager returns a new Ethereum sub protocol manager. The Ethereum sub protocol manages peers capable
// with the Ethereum network.
func NewProtocolManager(
	config *params.ChainConfig, mode downloader.SyncMode, networkID uint64,
	mux *event.TypeMux, txpool txPool, engine consensus.Engine,
	blockchain *core.BlockChain, chaindb ethdb.Database,
	isBlockProposer bool, gov governance, app dexconApp) (*ProtocolManager, error) {
	tab := newNodeTable()
	// Create the protocol manager with the base fields
	manager := &ProtocolManager{
		networkID:        networkID,
		eventMux:         mux,
		txpool:           txpool,
		nodeTable:        tab,
		gov:              gov,
		blockchain:       blockchain,
		cache:            newCache(5120, dexDB.NewDatabase(chaindb)),
		nextPullVote:     &sync.Map{},
		chainconfig:      config,
		newPeerCh:        make(chan *peer),
		noMorePeers:      make(chan struct{}),
		txsyncCh:         make(chan *txsync),
		recordsyncCh:     make(chan *recordsync),
		quitSync:         make(chan struct{}),
		receiveCh:        make(chan interface{}, 1024),
		isBlockProposer:  isBlockProposer,
		app:              app,
		blockNumberGauge: metrics.GetOrRegisterGauge("dex/blocknumber", nil),
	}

	// Figure out whether to allow fast sync or not
	if mode == downloader.FastSync && blockchain.CurrentBlock().NumberU64() > 0 {
		log.Warn("Blockchain not empty, fast sync disabled")
		mode = downloader.FullSync
	}
	if mode == downloader.FastSync {
		manager.fastSync = uint32(1)
	}
	// Initiate a sub-protocol for every implemented version we can handle
	manager.SubProtocols = make([]p2p.Protocol, 0, len(ProtocolVersions))
	for i, version := range ProtocolVersions {
		version := version // Closure for the run
		manager.SubProtocols = append(manager.SubProtocols, p2p.Protocol{
			Name:    ProtocolName,
			Version: version,
			Length:  ProtocolLengths[i],
			Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
				peer := manager.newPeer(int(version), p, rw)
				select {
				case manager.newPeerCh <- peer:
					manager.wg.Add(1)
					defer manager.wg.Done()
					return manager.handle(peer)
				case <-manager.quitSync:
					return p2p.DiscQuitting
				}
			},
			NodeInfo: func() interface{} {
				return manager.NodeInfo()
			},
			PeerInfo: func(id enode.ID) interface{} {
				if p := manager.peers.Peer(id.String()); p != nil {
					return p.Info()
				}
				return nil
			},
		})
	}
	if len(manager.SubProtocols) == 0 {
		return nil, errIncompatibleConfig
	}
	// Construct the different synchronisation mechanisms
	manager.downloader = downloader.New(mode, chaindb, manager.eventMux, blockchain, nil, manager.removePeer)

	validator := func(header *types.Header) error {
		return blockchain.VerifyDexonHeader(header)
	}
	heighter := func() uint64 {
		return blockchain.CurrentBlock().NumberU64()
	}
	inserter := func(blocks types.Blocks) (int, error) {
		// If fast sync is running, deny importing weird blocks
		if atomic.LoadUint32(&manager.fastSync) == 1 {
			log.Warn("Discarded bad propagated block", "number", blocks[0].Number(), "hash", blocks[0].Hash())
			return 0, nil
		}
		atomic.StoreUint32(&manager.acceptTxs, 1) // Mark initial sync done on any fetcher import
		return manager.blockchain.InsertDexonChain(blocks)
	}
	manager.fetcher = fetcher.New(blockchain.GetBlockByHash, validator, manager.BroadcastBlock, heighter, inserter, manager.removePeer)

	return manager, nil
}

func (pm *ProtocolManager) removePeer(id string) {
	// Short circuit if the peer was already removed
	peer := pm.peers.Peer(id)
	if peer == nil {
		return
	}
	log.Debug("Removing Ethereum peer", "peer", id)

	pm.nextPullVote.Delete(peer.ID())

	// Unregister the peer from the downloader and Ethereum peer set
	pm.downloader.UnregisterPeer(id)
	if err := pm.peers.Unregister(id); err != nil {
		log.Error("Peer removal failed", "peer", id, "err", err)
	}
	// Hard disconnect at the networking layer
	if peer != nil {
		peer.Peer.Disconnect(p2p.DiscUselessPeer)
	}
}

func (pm *ProtocolManager) Start(srvr p2pServer, maxPeers int) {
	pm.maxPeers = maxPeers
	pm.srvr = srvr
	pm.peers = newPeerSet(pm.gov, pm.srvr, pm.nodeTable)

	// broadcast transactions
	pm.txsCh = make(chan core.NewTxsEvent, txChanSize)
	pm.txsSub = pm.txpool.SubscribeNewTxsEvent(pm.txsCh)
	go pm.txBroadcastLoop()

	if pm.isBlockProposer {
		// broadcast finalized blocks
		pm.finalizedBlockCh = make(chan core.NewFinalizedBlockEvent,
			finalizedBlockChanSize)
		pm.finalizedBlockSub = pm.app.SubscribeNewFinalizedBlockEvent(
			pm.finalizedBlockCh)
		go pm.finalizedBlockBroadcastLoop()
	}

	// broadcast node records
	pm.recordsCh = make(chan newRecordsEvent, recordChanSize)
	pm.recordsSub = pm.nodeTable.SubscribeNewRecordsEvent(pm.recordsCh)
	go pm.recordBroadcastLoop()

	// run the peer set loop
	pm.chainHeadCh = make(chan core.ChainHeadEvent)
	pm.chainHeadSub = pm.blockchain.SubscribeChainHeadEvent(pm.chainHeadCh)
	go pm.peerSetLoop()

	// start sync handlers
	go pm.syncer()
	go pm.txsyncLoop()
	go pm.recordsyncLoop()

}

func (pm *ProtocolManager) Stop() {
	log.Info("Stopping DEXON protocol")

	pm.txsSub.Unsubscribe() // quits txBroadcastLoop
	pm.chainHeadSub.Unsubscribe()

	if pm.isBlockProposer {
		pm.finalizedBlockSub.Unsubscribe()
	}

	// Quit the sync loop.
	// After this send has completed, no new peers will be accepted.
	pm.noMorePeers <- struct{}{}

	// Quit fetcher, txsyncLoop.
	close(pm.quitSync)

	// Disconnect existing sessions.
	// This also closes the gate for any new registrations on the peer set.
	// sessions which are already established but not added to pm.peers yet
	// will exit when they try to register.
	pm.peers.Close()

	// Wait for all peer handler goroutines and the loops to come down.
	pm.wg.Wait()

	log.Info("DEXON protocol stopped")
}

func (pm *ProtocolManager) ReceiveChan() <-chan interface{} {
	return pm.receiveCh
}

func (pm *ProtocolManager) newPeer(pv int, p *p2p.Peer, rw p2p.MsgReadWriter) *peer {
	return newPeer(pv, p, newMeteredMsgWriter(rw))
}

// handle is the callback invoked to manage the life cycle of an eth peer. When
// this function terminates, the peer is disconnected.
func (pm *ProtocolManager) handle(p *peer) error {
	// Ignore maxPeers if this is a trusted peer
	if pm.peers.Len() >= pm.maxPeers && !p.Peer.Info().Network.Trusted {
		return p2p.DiscTooManyPeers
	}
	p.Log().Debug("Ethereum peer connected", "name", p.Name())

	// Execute the Ethereum handshake
	var (
		genesis = pm.blockchain.Genesis()
		head    = pm.blockchain.CurrentHeader()
		hash    = head.Hash()
		number  = head.Number.Uint64()
	)
	if err := p.Handshake(pm.networkID, number, hash, genesis.Hash()); err != nil {
		p.Log().Debug("Ethereum handshake failed", "err", err)
		return err
	}
	if rw, ok := p.rw.(*meteredMsgReadWriter); ok {
		rw.Init(p.version)
	}
	// Register the peer locally
	if err := pm.peers.Register(p); err != nil {
		p.Log().Error("Ethereum peer registration failed", "err", err)
		return err
	}
	defer pm.removePeer(p.id)

	// Register the peer in the downloader. If the downloader considers it banned, we disconnect
	if err := pm.downloader.RegisterPeer(p.id, p.version, p); err != nil {
		return err
	}
	// Propagate existing transactions. new transactions appearing
	// after this will be sent via broadcasts.
	pm.syncTransactions(p)
	pm.syncNodeRecords(p)

	// main loop. handle incoming messages.
	for {
		if err := pm.handleMsg(p); err != nil {
			p.Log().Debug("Ethereum message handling failed", "err", err)
			return err
		}
	}
}

// handleMsg is invoked whenever an inbound message is received from a remote
// peer. The remote connection is torn down upon returning any error.
func (pm *ProtocolManager) handleMsg(p *peer) error {
	// Read the next message from the remote peer, and ensure it's fully consumed
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return err
	}
	if msg.Size > ProtocolMaxMsgSize {
		return errResp(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}
	defer msg.Discard()

	// Handle the message depending on its contents
	switch {
	case msg.Code == StatusMsg:
		// Status messages should never arrive after the handshake
		return errResp(ErrExtraStatusMsg, "uncontrolled status message")

	// Block header query, collect the requested headers and reply
	case msg.Code == GetBlockHeadersMsg:
		// Decode the complex header query
		var query getBlockHeadersData
		if err := msg.Decode(&query); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}
		hashMode := query.Origin.Hash != (common.Hash{})
		first := true
		maxNonCanonical := uint64(100)

		round := map[uint64]uint64{}
		// Gather headers until the fetch or network limits is reached
		var (
			bytes   common.StorageSize
			headers []*types.HeaderWithGovState
			unknown bool
		)
		for !unknown && len(headers) < int(query.Amount) && bytes < softResponseLimit && len(headers) < downloader.MaxHeaderFetch {
			// Retrieve the next header satisfying the query
			var origin *types.Header
			if hashMode {
				if first {
					first = false
					origin = pm.blockchain.GetHeaderByHash(query.Origin.Hash)
					if origin != nil {
						query.Origin.Number = origin.Number.Uint64()
					}
				} else {
					origin = pm.blockchain.GetHeader(query.Origin.Hash, query.Origin.Number)
				}
			} else {
				origin = pm.blockchain.GetHeaderByNumber(query.Origin.Number)
			}
			if origin == nil {
				break
			}
			headers = append(headers, &types.HeaderWithGovState{Header: origin})
			if round[origin.Round] == 0 {
				round[origin.Round] = origin.Number.Uint64()
			}
			bytes += estHeaderRlpSize

			// Advance to the next header of the query
			switch {
			case hashMode && query.Reverse:
				// Hash based traversal towards the genesis block
				ancestor := query.Skip + 1
				if ancestor == 0 {
					unknown = true
				} else {
					query.Origin.Hash, query.Origin.Number = pm.blockchain.GetAncestor(query.Origin.Hash, query.Origin.Number, ancestor, &maxNonCanonical)
					unknown = (query.Origin.Hash == common.Hash{})
				}
			case hashMode && !query.Reverse:
				// Hash based traversal towards the leaf block
				var (
					current = origin.Number.Uint64()
					next    = current + query.Skip + 1
				)
				if next <= current {
					infos, _ := json.MarshalIndent(p.Peer.Info(), "", "  ")
					p.Log().Warn("GetBlockHeaders skip overflow attack", "current", current, "skip", query.Skip, "next", next, "attacker", infos)
					unknown = true
				} else {
					if header := pm.blockchain.GetHeaderByNumber(next); header != nil {
						nextHash := header.Hash()
						expOldHash, _ := pm.blockchain.GetAncestor(nextHash, next, query.Skip+1, &maxNonCanonical)
						if expOldHash == query.Origin.Hash {
							query.Origin.Hash, query.Origin.Number = nextHash, next
						} else {
							unknown = true
						}
					} else {
						unknown = true
					}
				}
			case query.Reverse:
				// Number based traversal towards the genesis block
				if query.Origin.Number >= query.Skip+1 {
					query.Origin.Number -= query.Skip + 1
				} else {
					unknown = true
				}

			case !query.Reverse:
				// Number based traversal towards the leaf block
				query.Origin.Number += query.Skip + 1
			}
		}

		if query.WithGov && len(headers) > 0 {
			last := headers[len(headers)-1]
			currentBlock := pm.blockchain.CurrentBlock()

			// Do not reply if we don't have current gov state
			if currentBlock.NumberU64() < last.Number.Uint64() {
				log.Debug("Current block < last request",
					"current", currentBlock.NumberU64(), "last", last.Number.Uint64())
				return p.SendBlockHeaders(query.Flag, []*types.HeaderWithGovState{})
			}

			snapshotHeight := map[uint64]struct{}{}
			for r, height := range round {
				log.Trace("#Include round", "round", r)
				if r == 0 {
					continue
				}
				h := pm.gov.GetRoundHeight(r)
				log.Trace("#Snapshot height", "height", h)
				if h == 0 {
					h = height
				}
				snapshotHeight[h] = struct{}{}
			}

			for _, header := range headers {
				if _, exist := snapshotHeight[header.Number.Uint64()]; exist {
					s, err := pm.blockchain.GetGovStateByHash(header.Hash())
					if err != nil {
						log.Warn("Get gov state by hash fail", "number", header.Number.Uint64(), "err", err)
						return p.SendBlockHeaders(query.Flag, []*types.HeaderWithGovState{})
					}
					header.GovState = s
				}
				log.Trace("Send header", "round", header.Round, "number", header.Number.Uint64(), "gov state == nil", header.GovState == nil)
			}
		}
		return p.SendBlockHeaders(query.Flag, headers)

	case msg.Code == BlockHeadersMsg:
		// A batch of headers arrived to one of our previous requests
		var data headersData
		if err := msg.Decode(&data); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}

		switch data.Flag {
		case fetcherReq:
			if len(data.Headers) > 0 {
				pm.fetcher.FilterHeaders(p.id, []*types.Header{data.Headers[0].Header}, time.Now())
			}
		case downloaderReq:
			err := pm.downloader.DeliverHeaders(p.id, data.Headers)
			if err != nil {
				log.Debug("Failed to deliver headers", "err", err)
			}
		default:
			log.Debug("Got headers with unexpected flag", "flag", data.Flag)
		}

	case msg.Code == GetBlockBodiesMsg:
		// Decode the retrieval message
		var query getBlockBodiesData
		if err := msg.Decode(&query); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}

		msgStream := rlp.NewStream(bytes.NewBuffer(query.Hashes), uint64(len(query.Hashes)))
		if _, err := msgStream.List(); err != nil {
			return err
		}
		// Gather blocks until the fetch or network limits is reached
		var (
			hash   common.Hash
			bytes  int
			bodies []rlp.RawValue
		)
		for bytes < softResponseLimit && len(bodies) < downloader.MaxBlockFetch {
			// Retrieve the hash of the next block
			if err := msgStream.Decode(&hash); err == rlp.EOL {
				break
			} else if err != nil {
				return errResp(ErrDecode, "msg %v: %v", msg, err)
			}
			// Retrieve the requested block body, stopping if enough was found
			if data := pm.blockchain.GetBodyRLP(hash); len(data) != 0 {
				bodies = append(bodies, data)
				bytes += len(data)
			}
		}
		return p.SendBlockBodiesRLP(query.Flag, bodies)

	case msg.Code == BlockBodiesMsg:
		// A batch of block bodies arrived to one of our previous requests
		var request blockBodiesData
		if err := msg.Decode(&request); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		// Deliver them all to the downloader for queuing
		transactions := make([][]*types.Transaction, len(request.Bodies))
		uncles := make([][]*types.Header, len(request.Bodies))

		for i, body := range request.Bodies {
			transactions[i] = body.Transactions
			uncles[i] = body.Uncles
		}

		switch request.Flag {
		case fetcherReq:
			if len(transactions) > 0 || len(uncles) > 0 {
				pm.fetcher.FilterBodies(p.id, transactions, uncles, time.Now())
			}
		case downloaderReq:
			err := pm.downloader.DeliverBodies(p.id, transactions, uncles)
			if err != nil {
				log.Debug("Failed to deliver bodies", "err", err)
			}
		default:
			log.Debug("Got bodies with unexpected flag", "flag", request.Flag)
		}

	case msg.Code == GetNodeDataMsg:
		// Decode the retrieval message
		msgStream := rlp.NewStream(msg.Payload, uint64(msg.Size))
		if _, err := msgStream.List(); err != nil {
			return err
		}
		// Gather state data until the fetch or network limits is reached
		var (
			hash  common.Hash
			bytes int
			data  [][]byte
		)
		for bytes < softResponseLimit && len(data) < downloader.MaxStateFetch {
			// Retrieve the hash of the next state entry
			if err := msgStream.Decode(&hash); err == rlp.EOL {
				break
			} else if err != nil {
				return errResp(ErrDecode, "msg %v: %v", msg, err)
			}
			// Retrieve the requested state entry, stopping if enough was found
			if entry, err := pm.blockchain.TrieNode(hash); err == nil {
				data = append(data, entry)
				bytes += len(entry)
			}
		}
		return p.SendNodeData(data)

	case msg.Code == NodeDataMsg:
		// A batch of node state data arrived to one of our previous requests
		var data [][]byte
		if err := msg.Decode(&data); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		// Deliver all to the downloader
		if err := pm.downloader.DeliverNodeData(p.id, data); err != nil {
			log.Debug("Failed to deliver node state data", "err", err)
		}

	case msg.Code == GetReceiptsMsg:
		// Decode the retrieval message
		msgStream := rlp.NewStream(msg.Payload, uint64(msg.Size))
		if _, err := msgStream.List(); err != nil {
			return err
		}
		// Gather state data until the fetch or network limits is reached
		var (
			hash     common.Hash
			bytes    int
			receipts []rlp.RawValue
		)
		for bytes < softResponseLimit && len(receipts) < downloader.MaxReceiptFetch {
			// Retrieve the hash of the next block
			if err := msgStream.Decode(&hash); err == rlp.EOL {
				break
			} else if err != nil {
				return errResp(ErrDecode, "msg %v: %v", msg, err)
			}
			// Retrieve the requested block's receipts, skipping if unknown to us
			results := pm.blockchain.GetReceiptsByHash(hash)
			if results == nil {
				if header := pm.blockchain.GetHeaderByHash(hash); header == nil || header.ReceiptHash != types.EmptyRootHash {
					continue
				}
			}
			// If known, encode and queue for response packet
			if encoded, err := rlp.EncodeToBytes(results); err != nil {
				log.Error("Failed to encode receipt", "err", err)
			} else {
				receipts = append(receipts, encoded)
				bytes += len(encoded)
			}
		}
		return p.SendReceiptsRLP(receipts)

	case msg.Code == ReceiptsMsg:
		// A batch of receipts arrived to one of our previous requests
		var receipts [][]*types.Receipt
		if err := msg.Decode(&receipts); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		// Deliver all to the downloader
		if err := pm.downloader.DeliverReceipts(p.id, receipts); err != nil {
			log.Debug("Failed to deliver receipts", "err", err)
		}

	case msg.Code == NewBlockHashesMsg:
		var announces newBlockHashesData
		if err := msg.Decode(&announces); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}
		// Mark the hashes as present at the remote node
		for _, block := range announces {
			p.MarkBlock(block.Hash)
		}
		// Schedule all the unknown hashes for retrieval
		unknown := make(newBlockHashesData, 0, len(announces))
		for _, block := range announces {
			if !pm.blockchain.HasBlock(block.Hash, block.Number) {
				unknown = append(unknown, block)
			}
		}
		for _, block := range unknown {
			pm.fetcher.Notify(p.id, block.Hash, block.Number, time.Now(), p.RequestOneHeader, p.FetchBodies)
		}

	case msg.Code == NewBlockMsg:
		// Retrieve and decode the propagated block
		var block types.Block
		if err := msg.Decode(&block); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}
		block.ReceivedAt = msg.ReceivedAt
		block.ReceivedFrom = p

		// Mark the peer as owning the block and schedule it for import
		p.MarkBlock(block.Hash())
		pm.fetcher.Enqueue(p.id, &block)

		// Assuming the block is importable by the peer, but possibly not yet done so,
		// calculate the head hash and number that the peer truly must have.
		var (
			trueHead   = block.ParentHash()
			trueNumber = block.NumberU64() - 1
		)
		// Update the peers number if better than the previous
		if _, number := p.Head(); trueNumber > number {
			p.SetHead(trueHead, trueNumber)

			// Schedule a sync if above ours. Note, this will not fire a sync for a gap of
			// a singe block (as the true number is below the propagated block), however this
			// scenario should easily be covered by the fetcher.
			currentBlock := pm.blockchain.CurrentBlock()
			if trueNumber > currentBlock.NumberU64() {
				go pm.synchronise(p)
			}
		}

	case msg.Code == TxMsg:
		// Transactions arrived, make sure we have a valid and fresh chain to handle them
		if atomic.LoadUint32(&pm.acceptTxs) == 0 {
			break
		}
		// Transactions can be processed, parse all of them and deliver to the pool
		var txs []*types.Transaction
		if err := msg.Decode(&txs); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		for i, tx := range txs {
			// Validate and mark the remote transaction
			if tx == nil {
				return errResp(ErrDecode, "transaction %d is nil", i)
			}
			p.MarkTransaction(tx.Hash())
		}
		types.GlobalSigCache.Add(types.NewEIP155Signer(pm.blockchain.Config().ChainID), txs)
		pm.txpool.AddRemotes(txs)

	case msg.Code == RecordMsg:
		var records []*enr.Record
		if err := msg.Decode(&records); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		for i, record := range records {
			if record == nil {
				return errResp(ErrDecode, "node record %d is nil", i)
			}
			p.MarkNodeRecord(rlpHash(record))
		}
		pm.nodeTable.AddRecords(records)

	// Block proposer-only messages.

	case msg.Code == CoreBlockMsg:
		if !pm.isBlockProposer {
			break
		}
		var blocks []*coreTypes.Block
		if err := msg.Decode(&blocks); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		for _, block := range blocks {
			pm.cache.addBlock(block)
			pm.receiveCh <- block
		}
	case msg.Code == VoteMsg:
		if !pm.isBlockProposer {
			break
		}
		var votes []*coreTypes.Vote
		if err := msg.Decode(&votes); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		for _, vote := range votes {
			if vote.Type >= coreTypes.VotePreCom {
				pm.cache.addVote(vote)
			}
			pm.receiveCh <- vote
		}
	case msg.Code == AgreementMsg:
		if !pm.isBlockProposer {
			break
		}
		// DKG set is receiver
		var agreement coreTypes.AgreementResult
		if err := msg.Decode(&agreement); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		p.MarkAgreement(rlpHash(agreement))
		pm.receiveCh <- &agreement
	case msg.Code == RandomnessMsg:
		if !pm.isBlockProposer {
			break
		}
		// Broadcast this to all peer
		var randomnesses []*coreTypes.BlockRandomnessResult
		if err := msg.Decode(&randomnesses); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		for _, randomness := range randomnesses {
			p.MarkRandomness(rlpHash(randomness))
			pm.receiveCh <- randomness
		}
	case msg.Code == DKGPrivateShareMsg:
		if !pm.isBlockProposer {
			break
		}
		// Do not relay this msg
		var ps dkgTypes.PrivateShare
		if err := msg.Decode(&ps); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		p.MarkDKGPrivateShares(rlpHash(ps))
		pm.receiveCh <- &ps
	case msg.Code == DKGPartialSignatureMsg:
		if !pm.isBlockProposer {
			break
		}
		// broadcast in DKG set
		var psig dkgTypes.PartialSignature
		if err := msg.Decode(&psig); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		pm.receiveCh <- &psig
	case msg.Code == PullBlocksMsg:
		if !pm.isBlockProposer {
			break
		}
		var hashes coreCommon.Hashes
		if err := msg.Decode(&hashes); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		blocks := pm.cache.blocks(hashes)
		log.Debug("Push blocks", "blocks", blocks)
		return p.SendCoreBlocks(blocks)
	case msg.Code == PullVotesMsg:
		if !pm.isBlockProposer {
			break
		}
		next, ok := pm.nextPullVote.Load(p.ID())
		if ok {
			nextTime := next.(time.Time)
			if nextTime.After(time.Now()) {
				break
			}
		}
		pm.nextPullVote.Store(p.ID(), time.Now().Add(pullVoteRateLimit))
		var pos coreTypes.Position
		if err := msg.Decode(&pos); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		votes := pm.cache.votes(pos)
		log.Debug("Push votes", "votes", votes)
		return p.SendVotes(votes)
	case msg.Code == PullRandomnessMsg:
		if !pm.isBlockProposer {
			break
		}
		var hashes coreCommon.Hashes
		if err := msg.Decode(&hashes); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		randomnesses := pm.cache.randomness(hashes)
		log.Debug("Push randomness", "randomness", randomnesses)
		return p.SendRandomnesses(randomnesses)
	case msg.Code == GetGovStateMsg:
		var hash common.Hash
		if err := msg.Decode(&hash); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		govState, err := pm.blockchain.GetGovStateByHash(hash)
		if err != nil {
			panic(err)
		}
		return p.SendGovState(govState)
	case msg.Code == GovStateMsg:
		var govState types.GovState
		if err := msg.Decode(&govState); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		if err := pm.downloader.DeliverGovState(p.id, &govState); err != nil {
			log.Debug("Failed to deliver govstates", "err", err)
		}
	default:
		return errResp(ErrInvalidMsgCode, "%v", msg.Code)
	}
	return nil
}

// BroadcastBlock will either propagate a block to a subset of it's peers, or
// will only announce it's availability (depending what's requested).
func (pm *ProtocolManager) BroadcastBlock(block *types.Block, propagate bool) {
	hash := block.Hash()
	peers := pm.peers.PeersWithoutBlock(hash)

	// If propagation is requested, send to a subset of the peer
	if propagate {
		if parent := pm.blockchain.GetBlock(block.ParentHash(), block.NumberU64()-1); parent == nil {
			log.Error("Propagating dangling block", "number", block.Number(), "hash", hash)
			return
		}
		// Send the block to a subset of our peers
		transfer := peers[:int(math.Sqrt(float64(len(peers))))]
		for _, peer := range transfer {
			peer.AsyncSendNewBlock(block)
		}
		log.Trace("Propagated block", "hash", hash, "recipients", len(transfer), "duration", common.PrettyDuration(time.Since(block.ReceivedAt)))
		return
	}
	// Otherwise if the block is indeed in out own chain, announce it
	if pm.blockchain.HasBlock(hash, block.NumberU64()) {
		for _, peer := range peers {
			peer.AsyncSendNewBlockHash(block)
		}
		log.Trace("Announced block", "hash", hash, "recipients", len(peers), "duration", common.PrettyDuration(time.Since(block.ReceivedAt)))
	}
}

// BroadcastTxs will propagate a batch of transactions to all peers which are not known to
// already have the given transaction.
func (pm *ProtocolManager) BroadcastTxs(txs types.Transactions) {
	var txset = make(map[*peer]types.Transactions)

	// Broadcast transactions to a batch of peers not knowing about it
	for _, tx := range txs {
		peers := pm.peers.PeersWithoutTx(tx.Hash())
		for _, peer := range peers {
			txset[peer] = append(txset[peer], tx)
		}
		log.Trace("Broadcast transaction", "hash", tx.Hash(), "recipients", len(peers))
	}
	// FIXME include this again: peers = peers[:int(math.Sqrt(float64(len(peers))))]
	for peer, txs := range txset {
		peer.AsyncSendTransactions(txs)
	}
}

// BroadcastRecords will propagate node records to its peers.
func (pm *ProtocolManager) BroadcastRecords(records []*enr.Record) {
	var recordset = make(map[*peer][]*enr.Record)

	for _, record := range records {
		peers := pm.peers.PeersWithoutNodeRecord(rlpHash(record))
		for _, peer := range peers {
			recordset[peer] = append(recordset[peer], record)
		}
		log.Trace("Broadcast record", "recipients", len(peers))
	}

	for peer, records := range recordset {
		peer.AsyncSendNodeRecords(records)
	}
}

// BroadcastCoreBlock broadcasts the core block to all its peers.
func (pm *ProtocolManager) BroadcastCoreBlock(block *coreTypes.Block) {
	pm.cache.addBlock(block)
	for _, peer := range pm.peers.Peers() {
		peer.AsyncSendCoreBlocks([]*coreTypes.Block{block})
	}
}

// BroadcastVote broadcasts the given vote to all peers in same notary set
func (pm *ProtocolManager) BroadcastVote(vote *coreTypes.Vote) {
	if vote.Type >= coreTypes.VotePreCom {
		pm.cache.addVote(vote)
	}
	label := peerLabel{
		set:   notaryset,
		round: vote.Position.Round,
	}
	for _, peer := range pm.peers.PeersWithLabel(label) {
		peer.AsyncSendVotes([]*coreTypes.Vote{vote})
	}
}

func (pm *ProtocolManager) BroadcastAgreementResult(
	agreement *coreTypes.AgreementResult) {
	// send to dkg nodes first (direct)
	label := peerLabel{
		set:   dkgset,
		round: agreement.Position.Round,
	}
	for _, peer := range pm.peers.PeersWithLabel(label) {
		if !peer.knownAgreements.Contains(rlpHash(agreement)) {
			peer.AsyncSendAgreement(agreement)
		}
	}

	for _, peer := range pm.peers.PeersWithoutAgreement(rlpHash(agreement)) {
		peer.AsyncSendAgreement(agreement)
	}
}

func (pm *ProtocolManager) BroadcastRandomnessResult(
	randomness *coreTypes.BlockRandomnessResult) {
	pm.cache.addRandomness(randomness)
	// send to notary nodes first (direct)
	label := peerLabel{
		set:   notaryset,
		round: randomness.Position.Round,
	}
	randomnesses := []*coreTypes.BlockRandomnessResult{randomness}
	for _, peer := range pm.peers.PeersWithLabel(label) {
		if !peer.knownRandomnesses.Contains(rlpHash(randomness)) {
			peer.AsyncSendRandomnesses(randomnesses)
		}
	}

	for _, peer := range pm.peers.PeersWithoutRandomness(rlpHash(randomness)) {
		peer.AsyncSendRandomnesses(randomnesses)
	}
}

func (pm *ProtocolManager) SendDKGPrivateShare(
	pub coreCrypto.PublicKey, privateShare *dkgTypes.PrivateShare) {

	pk, err := crypto.UnmarshalPubkey(pub.Bytes())
	if err != nil {
		panic(err)
	}

	id := enode.PubkeyToIDV4(pk)

	if p := pm.peers.Peer(id.String()); p != nil {
		p.AsyncSendDKGPrivateShare(privateShare)
	} else {
		log.Error("Failed to send DKG private share", "publicKey", id.String())
	}
}

func (pm *ProtocolManager) BroadcastDKGPrivateShare(
	privateShare *dkgTypes.PrivateShare) {
	label := peerLabel{set: dkgset, round: privateShare.Round}
	for _, peer := range pm.peers.PeersWithLabel(label) {
		if !peer.knownDKGPrivateShares.Contains(rlpHash(privateShare)) {
			peer.AsyncSendDKGPrivateShare(privateShare)
		}
	}
}

func (pm *ProtocolManager) BroadcastDKGPartialSignature(
	psig *dkgTypes.PartialSignature) {
	label := peerLabel{set: dkgset, round: psig.Round}
	for _, peer := range pm.peers.PeersWithLabel(label) {
		peer.AsyncSendDKGPartialSignature(psig)
	}
}

func (pm *ProtocolManager) BroadcastPullBlocks(
	hashes coreCommon.Hashes) {
	// TODO(jimmy-dexon): pull from notary set only.
	for idx, peer := range pm.peers.Peers() {
		if idx >= maxPullPeers {
			break
		}
		peer.AsyncSendPullBlocks(hashes)
	}
}

func (pm *ProtocolManager) BroadcastPullVotes(
	pos coreTypes.Position) {
	label := peerLabel{
		set:   notaryset,
		round: pos.Round,
	}
	for idx, peer := range pm.peers.PeersWithLabel(label) {
		if idx >= maxPullVotePeers {
			break
		}
		peer.AsyncSendPullVotes(pos)
	}
}

func (pm *ProtocolManager) BroadcastPullRandomness(
	hashes coreCommon.Hashes) {
	// TODO(jimmy-dexon): pull from dkg set only.
	for idx, peer := range pm.peers.Peers() {
		if idx >= maxPullPeers {
			break
		}
		peer.AsyncSendPullRandomness(hashes)
	}
}

func (pm *ProtocolManager) txBroadcastLoop() {
	queueSizeMax := common.StorageSize(100 * 1024) // 100 KB
	currentSize := common.StorageSize(0)
	txs := make(types.Transactions, 0)
	for {
		select {
		case <-time.After(100 * time.Millisecond):
			pm.BroadcastTxs(txs)
			txs = txs[:0]
			currentSize = 0
		case event := <-pm.txsCh:
			txs = append(txs, event.Txs...)
			for _, tx := range event.Txs {
				currentSize += tx.Size()
			}
			if currentSize >= queueSizeMax {
				pm.BroadcastTxs(txs)
				txs = txs[:0]
				currentSize = 0
			}

		// Err() channel will be closed when unsubscribing.
		case <-pm.txsSub.Err():
			return
		}
	}
}

func (pm *ProtocolManager) finalizedBlockBroadcastLoop() {
	for {
		select {
		case event := <-pm.finalizedBlockCh:
			pm.BroadcastBlock(event.Block, true)
			pm.BroadcastBlock(event.Block, false)

		// Err() channel will be closed when unsubscribing.
		case <-pm.finalizedBlockSub.Err():
			return
		}
	}
}

func (pm *ProtocolManager) recordBroadcastLoop() {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	t := time.NewTimer(0)
	defer t.Stop()

	for {
		select {
		case event := <-pm.recordsCh:
			pm.BroadcastRecords(event.Records)
			pm.peers.Refresh()

		case <-t.C:
			record := pm.srvr.Self().Record()
			log.Debug("refresh our node record", "seq", record.Seq())
			pm.nodeTable.AddRecords([]*enr.Record{record})

			// Log current peers connection status.
			pm.peers.Status()

			// Reset timer.
			d := 1*time.Minute + time.Duration(r.Int63n(60))*time.Second
			t.Reset(d)

		// Err() channel will be closed when unsubscribing.
		case <-pm.recordsSub.Err():
			return
		}
	}
}

// a loop keep building and maintaining peers in notary set.
// TODO: finish this
func (pm *ProtocolManager) peerSetLoop() {
	log.Debug("ProtocolManager: started peer set loop")

	round := pm.gov.Round()
	log.Trace("ProtocolManager: startup round", "round", round)

	if round < dexCore.DKGDelayRound {
		for i := round; i <= dexCore.DKGDelayRound; i++ {
			pm.peers.BuildConnection(i)
		}
	} else {
		pm.peers.BuildConnection(round)
	}

	for {
		select {
		case event := <-pm.chainHeadCh:
			pm.blockNumberGauge.Update(int64(event.Block.NumberU64()))

			if !pm.isBlockProposer {
				break
			}

			newRound := pm.gov.CRSRound()
			if newRound == 0 {
				break
			}

			log.Debug("ProtocolManager: new round", "round", newRound)
			if newRound == round {
				break
			}

			if newRound == round+1 {
				pm.peers.BuildConnection(newRound)
				if round >= 1 {
					pm.peers.ForgetConnection(round - 1)
				}
			} else {
				// just forget all network connection and rebuild.
				pm.peers.ForgetConnection(round)

				if newRound >= 1 {
					pm.peers.BuildConnection(newRound - 1)
				}
				pm.peers.BuildConnection(newRound)
			}
			round = newRound
		case <-pm.chainHeadSub.Err():
			return
		}
	}
}

// NodeInfo represents a short summary of the Ethereum sub-protocol metadata
// known about the host peer.
type NodeInfo struct {
	Network uint64              `json:"network"` // DEXON network ID (237=Mainnet, 238=Taiwan, 239=Taipei, 240=Yilan)
	Number  uint64              `json:"number"`  // Total difficulty of the host's blockchain
	Genesis common.Hash         `json:"genesis"` // SHA3 hash of the host's genesis block
	Config  *params.ChainConfig `json:"config"`  // Chain configuration for the fork rules
	Head    common.Hash         `json:"head"`    // SHA3 hash of the host's best owned block
}

// NodeInfo retrieves some protocol metadata about the running host node.
func (pm *ProtocolManager) NodeInfo() *NodeInfo {
	currentBlock := pm.blockchain.CurrentBlock()
	return &NodeInfo{
		Network: pm.networkID,
		Number:  currentBlock.NumberU64(),
		Genesis: pm.blockchain.Genesis().Hash(),
		Config:  pm.blockchain.Config(),
		Head:    currentBlock.Hash(),
	}
}
