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

package dex

import (
	"fmt"
	"time"

	"github.com/tangerine-network/go-tangerine/accounts"
	"github.com/tangerine-network/go-tangerine/consensus"
	"github.com/tangerine-network/go-tangerine/consensus/dexcon"
	"github.com/tangerine-network/go-tangerine/core"
	"github.com/tangerine-network/go-tangerine/core/bloombits"
	"github.com/tangerine-network/go-tangerine/core/rawdb"
	"github.com/tangerine-network/go-tangerine/core/vm"
	"github.com/tangerine-network/go-tangerine/dex/downloader"
	"github.com/tangerine-network/go-tangerine/eth/filters"
	"github.com/tangerine-network/go-tangerine/eth/gasprice"
	"github.com/tangerine-network/go-tangerine/ethdb"
	"github.com/tangerine-network/go-tangerine/event"
	"github.com/tangerine-network/go-tangerine/indexer"
	"github.com/tangerine-network/go-tangerine/internal/ethapi"
	"github.com/tangerine-network/go-tangerine/log"
	"github.com/tangerine-network/go-tangerine/node"
	"github.com/tangerine-network/go-tangerine/p2p"
	"github.com/tangerine-network/go-tangerine/params"
	"github.com/tangerine-network/go-tangerine/rpc"
	"github.com/tangerine-network/tangerine-consensus/core/syncer"
)

// Tangerine implements the DEXON fullnode service.
type Tangerine struct {
	config      *Config
	chainConfig *params.ChainConfig

	// Channel for shutting down the service
	shutdownChan chan bool // Channel for shutting down the Ethereum

	// Handlers
	txPool          *core.TxPool
	blockchain      *core.BlockChain
	protocolManager *ProtocolManager

	// DB interfaces
	chainDb ethdb.Database // Block chain database

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	bloomRequests chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer  *core.ChainIndexer             // Bloom indexer operating during block imports

	APIBackend *DexAPIBackend

	// Tangerine consensus.
	app        *DexconApp
	governance *DexconGovernance
	network    *DexconNetwork

	bp *blockProposer

	networkID     uint64
	netRPCService *ethapi.PublicNetAPI

	indexer indexer.Indexer
}

func New(ctx *node.ServiceContext, config *Config) (*Tangerine, error) {
	// Consensus.
	chainDb, err := CreateDB(ctx, config, "chaindata")
	if err != nil {
		return nil, err
	}
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb,
		config.Genesis)
	if _, ok := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !ok {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	if !config.SkipBcVersionCheck {
		bcVersion := rawdb.ReadDatabaseVersion(chainDb)
		if bcVersion != nil && *bcVersion != core.BlockChainVersion {
			return nil, fmt.Errorf("Blockchain DB version mismatch (%d / %d).\n",
				bcVersion, core.BlockChainVersion)
		}
		rawdb.WriteDatabaseVersion(chainDb, core.BlockChainVersion)
	}
	engine := dexcon.New()

	dex := &Tangerine{
		config:         config,
		chainDb:        chainDb,
		chainConfig:    chainConfig,
		eventMux:       ctx.EventMux,
		accountManager: ctx.AccountManager,
		shutdownChan:   make(chan bool),
		networkID:      config.NetworkId,
		bloomRequests:  make(chan chan *bloombits.Retrieval),
		bloomIndexer:   NewBloomIndexer(chainDb, params.BloomBitsBlocks, params.BloomConfirms),
		engine:         engine,
	}

	var (
		vmConfig = vm.Config{
			EnablePreimageRecording: config.EnablePreimageRecording,
			EWASMInterpreter:        config.EWASMInterpreter,
			EVMInterpreter:          config.EVMInterpreter,
			IsBlockProposer:         config.BlockProposerEnabled,
		}
		cacheConfig = &core.CacheConfig{Disabled: config.NoPruning, TrieCleanLimit: config.TrieCleanCache, TrieDirtyLimit: config.TrieDirtyCache, TrieTimeLimit: config.TrieTimeout}
	)
	dex.blockchain, err = core.NewBlockChain(chainDb, cacheConfig, dex.chainConfig, dex.engine, vmConfig, nil)

	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		dex.blockchain.SetHead(compat.RewindTo)
		rawdb.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}
	dex.bloomIndexer.Start(dex.blockchain)

	if config.Indexer.Enable {
		dex.indexer = indexer.NewIndexerFromConfig(
			indexer.NewROBlockChain(dex.blockchain),
			config.Indexer,
		)
		dex.indexer.Start()
	}

	if config.TxPool.Journal != "" {
		config.TxPool.Journal = ctx.ResolvePath(config.TxPool.Journal)
	}
	dex.txPool = core.NewTxPool(config.TxPool, dex.chainConfig, dex.blockchain)

	dex.APIBackend = &DexAPIBackend{dex, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.DefaultGasPrice
	}
	dex.APIBackend.gpo = gasprice.NewOracle(dex.APIBackend, gpoParams)

	// Dexcon related objects.
	dex.governance = NewDexconGovernance(dex.APIBackend, dex.chainConfig, config.PrivateKey)
	dex.app = NewDexconApp(dex.txPool, dex.blockchain, dex.governance, chainDb, config)

	// Set config fetcher so engine can fetch current system configuration from state.
	engine.SetGovStateFetcher(dex.governance)

	dMoment := time.Unix(int64(chainConfig.DMoment), 0)
	log.Info("Consensus DMoment", "dMoment", dMoment)

	// Force starting with full sync mode if this node is a bootstrap proposer.
	if config.BlockProposerEnabled && dMoment.After(time.Now()) {
		config.SyncMode = downloader.FullSync
	}

	pm, err := NewProtocolManager(dex.chainConfig, config.SyncMode,
		config.NetworkId, dex.eventMux, dex.txPool, dex.engine, dex.blockchain,
		chainDb, config.Whitelist, config.BlockProposerEnabled, dex.governance, dex.app)
	if err != nil {
		return nil, err
	}

	dex.protocolManager = pm
	dex.network = NewDexconNetwork(pm)

	recovery := NewRecovery(chainConfig.Recovery, config.RecoveryNetworkRPC,
		dex.governance, config.PrivateKey)
	watchCat := syncer.NewWatchCat(recovery, dex.governance, 10*time.Second,
		time.Duration(chainConfig.Recovery.Timeout)*time.Second, log.Root())

	dex.bp = NewBlockProposer(dex, watchCat, dMoment)
	return dex, nil
}

func (s *Tangerine) Protocols() []p2p.Protocol {
	return s.protocolManager.SubProtocols
}

func (s *Tangerine) APIs() []rpc.API {
	apis := ethapi.GetAPIs(s.APIBackend)

	// Append any APIs exposed explicitly by the consensus engine
	apis = append(apis, s.engine.APIs(s.BlockChain())...)

	// Append all the local APIs and return
	return append(apis, []rpc.API{
		{
			Namespace: "eth",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.APIBackend, false),
			Public:    true,
		}, {
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPrivateAdminAPI(s),
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPublicDebugAPI(s),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPrivateDebugAPI(s.chainConfig, s),
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *Tangerine) Start(srvr *p2p.Server) error {
	// Start the bloom bits servicing goroutines
	s.startBloomHandlers(params.BloomBitsBlocks)

	// Start the RPC service
	s.netRPCService = ethapi.NewPublicNetAPI(srvr, s.NetVersion())

	// Figure out a max peers count based on the server limits
	maxPeers := srvr.MaxPeers
	if s.config.LightServ > 0 {
		if s.config.LightPeers >= srvr.MaxPeers {
			return fmt.Errorf("invalid peer config: light peer count (%d) >= total peer count (%d)", s.config.LightPeers, srvr.MaxPeers)
		}
		maxPeers -= s.config.LightPeers
	}
	// Start the networking layer and the light server if requested
	s.protocolManager.Start(srvr, maxPeers)

	if s.config.BlockProposerEnabled {
		go func() {
			// Since we might be in fast sync mode when started. wait for
			// ChainHeadEvent before starting blockproposer, or else we will trigger
			// watchcat.
			if s.config.SyncMode == downloader.FastSync &&
				s.blockchain.CurrentBlock().NumberU64() == 0 {
				ch := make(chan core.ChainHeadEvent)
				sub := s.blockchain.SubscribeChainHeadEvent(ch)
				defer sub.Unsubscribe()

				<-ch
			}
			s.bp.Start(s)
		}()
	}
	return nil
}

func (s *Tangerine) Stop() error {
	s.bloomIndexer.Close()
	s.blockchain.Stop()
	s.engine.Close()
	s.protocolManager.Stop()
	s.txPool.Stop()
	s.eventMux.Stop()
	s.bp.Stop()
	s.app.Stop()
	if s.indexer != nil {
		s.indexer.Stop()
	}
	s.chainDb.Close()
	close(s.shutdownChan)
	return nil
}

func (s *Tangerine) IsCoreSyncing() bool {
	return s.bp.IsCoreSyncing()
}

func (s *Tangerine) IsProposing() bool {
	return s.bp.IsProposing()
}

// CreateDB creates the chain database.
func CreateDB(ctx *node.ServiceContext, config *Config, name string) (ethdb.Database, error) {
	db, err := ctx.OpenDatabase(name, config.DatabaseCache, config.DatabaseHandles)
	if err != nil {
		return nil, err
	}
	if db, ok := db.(*ethdb.LDBDatabase); ok {
		db.Meter("eth/db/chaindata/")
	}
	return db, nil
}

func (d *Tangerine) AccountManager() *accounts.Manager { return d.accountManager }
func (d *Tangerine) BlockChain() *core.BlockChain      { return d.blockchain }
func (d *Tangerine) TxPool() *core.TxPool              { return d.txPool }
func (d *Tangerine) DexVersion() int                   { return int(d.protocolManager.SubProtocols[0].Version) }
func (d *Tangerine) EventMux() *event.TypeMux          { return d.eventMux }
func (d *Tangerine) Engine() consensus.Engine          { return d.engine }
func (d *Tangerine) ChainDb() ethdb.Database           { return d.chainDb }
func (d *Tangerine) Downloader() ethapi.Downloader     { return d.protocolManager.downloader }
func (d *Tangerine) NetVersion() uint64                { return d.networkID }
