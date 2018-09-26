// Copyright 2018 The dexon-consensus-core Authors
// This file is part of the dexon-consensus-core library.
//
// The dexon-consensus-core library is free software: you can redistribute it
// and/or modify it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The dexon-consensus-core library is distributed in the hope that it will be
// useful, but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the dexon-consensus-core library. If not, see
// <http://www.gnu.org/licenses/>.

package dex

import (
	dexCore "github.com/dexon-foundation/dexon-consensus-core/core"
	"github.com/dexon-foundation/dexon-consensus-core/core/blockdb"
	ethCrypto "github.com/dexon-foundation/dexon-consensus-core/crypto/eth"

	"github.com/dexon-foundation/dexon/internal/ethapi"
	"github.com/dexon-foundation/dexon/accounts"
	"github.com/dexon-foundation/dexon/consensus"
	"github.com/dexon-foundation/dexon/core"
	"github.com/dexon-foundation/dexon/core/bloombits"
	"github.com/dexon-foundation/dexon/ethdb"
	"github.com/dexon-foundation/dexon/event"
	"github.com/dexon-foundation/dexon/node"
	"github.com/dexon-foundation/dexon/p2p"
	"github.com/dexon-foundation/dexon/params"
	"github.com/dexon-foundation/dexon/rpc"
)

// Dexon implementes the DEXON fullnode service.
type Dexon struct {
	config      *Config
	chainConfig *params.ChainConfig

	// Channel for shutting down the service
	shutdownChan chan bool // Channel for shutting down the Ethereum
	txPool       *core.TxPool
	blockchain   *core.BlockChain

	// DB interfaces
	chainDb ethdb.Database // Block chain database

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	bloomRequests chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer  *core.ChainIndexer             // Bloom indexer operating during block imports

	// Dexon consensus.
	app        *DexconApp
	governance *DexconGovernance
	network    *DexconNetwork
	blockdb    blockdb.BlockDatabase
	consensus  *dexCore.Consensus

	networkID     uint64
	netRPCService *ethapi.PublicNetAPI
}

func New(ctx *node.ServiceContext, config *Config) (*Dexon, error) {
	// Consensus.
	db, err := blockdb.NewLevelDBBackedBlockDB("main.blockdb")
	if err != nil {
		panic(err)
	}
	app := NewDexconApp(nil)
	gov := NewDexconGovernance()
	network := NewDexconNetwork()

	// TODO(w): replace this with node key.
	privKey, err := ethCrypto.NewPrivateKey()
	if err != nil {
		panic(err)
	}
	consensus := dexCore.NewConsensus(
		app, gov, db, network, privKey, ethCrypto.SigToPub)

	dex := &Dexon{
		config:         config,
		eventMux:       ctx.EventMux,
		accountManager: ctx.AccountManager,
		shutdownChan:   make(chan bool),
		networkID:      config.NetworkId,
		bloomRequests:  make(chan chan *bloombits.Retrieval),
		app:            app,
		governance:     gov,
		network:        network,
		blockdb:        db,
		consensus:      consensus,
	}
	return dex, nil
}

func (s *Dexon) Protocols() []p2p.Protocol {
	return nil
}

func (s *Dexon) APIs() []rpc.API {
	return nil
}

func (s *Dexon) Start(server *p2p.Server) error {
	return nil
}

func (s *Dexon) Stop() error {
	return nil
}
