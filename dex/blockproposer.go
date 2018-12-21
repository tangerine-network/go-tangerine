package dex

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	dexCore "github.com/dexon-foundation/dexon-consensus/core"
	coreEcdsa "github.com/dexon-foundation/dexon-consensus/core/crypto/ecdsa"
	"github.com/dexon-foundation/dexon-consensus/core/syncer"
	coreTypes "github.com/dexon-foundation/dexon-consensus/core/types"

	"github.com/dexon-foundation/dexon/core"
	"github.com/dexon-foundation/dexon/dex/db"
	"github.com/dexon-foundation/dexon/log"
	"github.com/dexon-foundation/dexon/rlp"
)

type blockProposer struct {
	mu        sync.Mutex
	running   int32
	syncing   int32
	proposing int32
	dex       *Dexon
	dMoment   time.Time

	wg     sync.WaitGroup
	stopCh chan struct{}
}

func NewBlockProposer(dex *Dexon, dMoment time.Time) *blockProposer {
	return &blockProposer{
		dex:     dex,
		dMoment: dMoment,
	}
}

func (b *blockProposer) Start() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !atomic.CompareAndSwapInt32(&b.running, 0, 1) {
		return fmt.Errorf("block proposer is already running")
	}
	log.Info("Block proposer started")

	b.stopCh = make(chan struct{})
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		defer atomic.StoreInt32(&b.running, 0)

		var err error
		var c *dexCore.Consensus

		if b.dMoment.After(time.Now()) {
			c = b.initConsensus()
		} else {
			c, err = b.syncConsensus()
		}

		if err != nil {
			log.Error("Block proposer stopped, before start running", "err", err)
			return
		}

		b.run(c)
		log.Info("Block proposer successfully stopped")
	}()
	return nil
}

func (b *blockProposer) run(c *dexCore.Consensus) {
	log.Info("Start running consensus core")
	go c.Run()
	atomic.StoreInt32(&b.proposing, 1)
	<-b.stopCh
	log.Debug("Block proposer receive stop signal")
	c.Stop()
}

func (b *blockProposer) Stop() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if atomic.LoadInt32(&b.running) == 1 {
		b.dex.protocolManager.isBlockProposer = false
		close(b.stopCh)
		b.wg.Wait()
		atomic.StoreInt32(&b.proposing, 0)
	}
}

func (b *blockProposer) IsLatticeSyncing() bool {
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

	db := db.NewDatabase(b.dex.chainDb)
	privkey := coreEcdsa.NewPrivateKeyFromECDSA(b.dex.config.PrivateKey)
	consensusSync := syncer.NewConsensus(b.dMoment, b.dex.app, b.dex.governance,
		db, b.dex.network, privkey, log.Root())

	blocksToSync := func(coreHeight, height uint64) []*coreTypes.Block {
		var blocks []*coreTypes.Block
		for len(blocks) < 1024 && coreHeight < height {
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
			break Loop
		}

		log.Debug("Filling compaction chain", "num", len(blocks),
			"first", blocks[0].Finalization.Height,
			"last", blocks[len(blocks)-1].Finalization.Height)
		if _, err := consensusSync.SyncBlocks(blocks, false); err != nil {
			return nil, err
		}
		coreHeight = blocks[len(blocks)-1].Finalization.Height

		select {
		case <-b.stopCh:
			return nil, errors.New("early stop")
		default:
		}
	}

	// Enable isBlockProposer flag to start receiving msg.
	b.dex.protocolManager.isBlockProposer = true

	ch := make(chan core.ChainHeadEvent)
	sub := b.dex.blockchain.SubscribeChainHeadEvent(ch)
	defer sub.Unsubscribe()

	// Listen chain head event until synced.
ListenLoop:
	for {
		select {
		case ev := <-ch:
			blocks := blocksToSync(coreHeight, ev.Block.NumberU64())
			if len(blocks) > 0 {
				log.Debug("Filling compaction chain", "num", len(blocks),
					"first", blocks[0].Finalization.Height,
					"last", blocks[len(blocks)-1].Finalization.Height)
				synced, err := consensusSync.SyncBlocks(blocks, true)
				if err != nil {
					log.Error("SyncBlocks fail", "err", err)
					return nil, err
				}
				if synced {
					log.Debug("Consensus core synced")
					break ListenLoop
				}
				coreHeight = blocks[len(blocks)-1].Finalization.Height
			}
		case <-sub.Err():
			log.Debug("System stopped when syncing consensus core")
			return nil, errors.New("system stop")
		case <-b.stopCh:
			log.Debug("Early stop, before consensus core can run")
			return nil, errors.New("early stop")
		}
	}

	return consensusSync.GetSyncedConsensus()
}
