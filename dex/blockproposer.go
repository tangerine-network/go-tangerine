package dex

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	dexCore "github.com/tangerine-network/tangerine-consensus/core"
	coreEcdsa "github.com/tangerine-network/tangerine-consensus/core/crypto/ecdsa"
	"github.com/tangerine-network/tangerine-consensus/core/syncer"
	coreTypes "github.com/tangerine-network/tangerine-consensus/core/types"

	"github.com/tangerine-network/go-tangerine/core"
	"github.com/tangerine-network/go-tangerine/dex/db"
	"github.com/tangerine-network/go-tangerine/log"
	"github.com/tangerine-network/go-tangerine/node"
	"github.com/tangerine-network/go-tangerine/rlp"
)

var (
	forceSyncTimeout = 20 * time.Second
)

type blockProposer struct {
	mu        sync.Mutex
	running   int32
	syncing   int32
	proposing int32
	dex       *Tangerine
	watchCat  *syncer.WatchCat
	dMoment   time.Time

	wg     sync.WaitGroup
	stopCh chan struct{}
}

func NewBlockProposer(dex *Tangerine, watchCat *syncer.WatchCat, dMoment time.Time) *blockProposer {
	return &blockProposer{
		dex:      dex,
		watchCat: watchCat,
		dMoment:  dMoment,
	}
}

func (b *blockProposer) Start(svc node.Service) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !atomic.CompareAndSwapInt32(&b.running, 0, 1) {
		return fmt.Errorf("block proposer is already running")
	}
	log.Info("Started block proposer")

	b.stopCh = make(chan struct{})
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		defer atomic.StoreInt32(&b.running, 0)

		var err error
		var c *dexCore.Consensus
		if b.dMoment.After(time.Now()) {
			// Start receiving core messages.
			b.dex.protocolManager.SetReceiveCoreMessage(true)

			c = b.initConsensus()
		} else {
			c, err = b.syncConsensus()
		}

		if err != nil {
			log.Error("Block proposer stopped, before start running", "err", err)
			return
		}

		log.Info("Start running consensus core")
		go c.Run(b.stopCh)
		atomic.StoreInt32(&b.proposing, 1)

		<-b.stopCh
		log.Debug("Block proposer receive stop signal")

		log.Info("Block proposer successfully stopped")
		go func() {
			svc.Stop()
			os.Exit(1)
		}()
	}()
	return nil
}

func (b *blockProposer) Stop() {
	log.Info("Stopping block proposer")
	b.mu.Lock()
	defer b.mu.Unlock()

	if atomic.LoadInt32(&b.running) == 1 {
		b.dex.protocolManager.SetReceiveCoreMessage(false)
		close(b.stopCh)
		b.wg.Wait()
		atomic.StoreInt32(&b.proposing, 0)
	}
	log.Info("Block proposer stopped")
}

func (b *blockProposer) IsCoreSyncing() bool {
	return atomic.LoadInt32(&b.syncing) == 1
}

func (b *blockProposer) IsProposing() bool {
	return atomic.LoadInt32(&b.proposing) == 1
}

func (b *blockProposer) initConsensus() *dexCore.Consensus {
	db := db.NewDatabase(b.dex.chainDb)
	privkey := coreEcdsa.NewPrivateKeyFromECDSA(b.dex.config.PrivateKey)
	return dexCore.NewConsensus(b.dMoment,
		b.dex.app, b.dex.governance, db, b.dex.network, privkey, log.Root())
}

func (b *blockProposer) syncConsensus() (*dexCore.Consensus, error) {
	atomic.StoreInt32(&b.syncing, 1)
	defer atomic.StoreInt32(&b.syncing, 0)

	cb := b.dex.blockchain.CurrentBlock()

	db := db.NewDatabase(b.dex.chainDb)
	privkey := coreEcdsa.NewPrivateKeyFromECDSA(b.dex.config.PrivateKey)
	consensusSync := syncer.NewConsensus(cb.NumberU64(), b.dMoment, b.dex.app,
		b.dex.governance, db, b.dex.network, privkey, log.Root())

	// Start the watchCat.
	b.watchCat.Start()
	defer b.watchCat.Stop()
	log.Info("Started sync watchCat")

	// Feed the current block we have in local blockchain.
	if cb.NumberU64() > 0 {
		var block coreTypes.Block
		if err := rlp.DecodeBytes(cb.Header().DexconMeta, &block); err != nil {
			panic(err)
		}
		b.watchCat.Feed(block.Position)
	}

	blocksToSync := func(coreHeight, height uint64) []*coreTypes.Block {
		var blocks []*coreTypes.Block
		for coreHeight < height {
			var block coreTypes.Block
			b := b.dex.blockchain.GetBlockByNumber(coreHeight + 1)
			if err := rlp.DecodeBytes(b.Header().DexconMeta, &block); err != nil {
				panic(err)
			}
			blocks = append(blocks, &block)
			coreHeight = coreHeight + 1
		}
		return blocks
	}

	// Sync all blocks in compaction chain to core.
	_, coreHeight := db.GetCompactionChainTipInfo()

Loop:
	for {
		currentBlock := b.dex.blockchain.CurrentBlock()
		log.Debug("Syncing compaction chain", "core height", coreHeight,
			"height", currentBlock.NumberU64())
		blocks := blocksToSync(coreHeight, currentBlock.NumberU64())

		if len(blocks) == 0 {
			log.Debug("No new block to sync", "current", currentBlock.NumberU64())
			break Loop
		}
		b.watchCat.Feed(blocks[len(blocks)-1].Position)

		log.Debug("Filling compaction chain", "num", len(blocks),
			"first", blocks[0].Position.Height,
			"last", blocks[len(blocks)-1].Position.Height)
		if _, err := consensusSync.SyncBlocks(blocks, false); err != nil {
			log.Debug("SyncBlocks fail", "err", err)
			return nil, err
		}
		coreHeight = blocks[len(blocks)-1].Position.Height

		select {
		case <-b.stopCh:
			return nil, errors.New("early stop")
		default:
		}
	}

	ch := make(chan core.ChainHeadEvent)
	sub := b.dex.blockchain.SubscribeChainHeadEvent(ch)
	defer sub.Unsubscribe()

	log.Debug("Listen chain head event until synced")

	// Listen chain head event until synced.
ListenLoop:
	for {
		select {
		case ev := <-ch:
			blocks := blocksToSync(coreHeight, ev.Block.NumberU64())

			if len(blocks) > 0 {
				b.watchCat.Feed(blocks[len(blocks)-1].Position)
				log.Debug("Filling compaction chain", "num", len(blocks),
					"first", blocks[0].Position.Height,
					"last", blocks[len(blocks)-1].Position.Height)
				synced, err := consensusSync.SyncBlocks(blocks, true)
				if err != nil {
					log.Error("SyncBlocks fail", "err", err)
					return nil, err
				}
				b.dex.protocolManager.SetReceiveCoreMessage(true)
				if synced {
					log.Debug("Consensus core synced")
					break ListenLoop
				}
				coreHeight = blocks[len(blocks)-1].Position.Height
			}
		case <-sub.Err():
			log.Debug("System stopped when syncing consensus core")
			return nil, errors.New("system stop")
		case <-b.stopCh:
			log.Debug("Early stop, before consensus core can run")
			return nil, errors.New("early stop")
		case <-time.After(forceSyncTimeout):
			log.Debug("no new chain head for a while")
			if p := b.dex.protocolManager.peers.BestPeer(); p != nil {
				log.Debug("try force sync with peer", "id", p.id)
				go b.dex.protocolManager.synchronise(p, true)
			} else {
				log.Debug("no peer to sync")
			}
		case <-b.watchCat.Meow():
			log.Info("WatchCat signaled to stop syncing")

			b.dex.protocolManager.SetReceiveCoreMessage(true)
			consensusSync.ForceSync(b.watchCat.LastPosition(), true)
			break ListenLoop
		}
	}

	con, err := consensusSync.GetSyncedConsensus()
	return con, err
}
