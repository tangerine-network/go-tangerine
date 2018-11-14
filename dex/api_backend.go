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
	"context"
	"math/big"

	"github.com/dexon-foundation/dexon/accounts"
	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/common/math"
	"github.com/dexon-foundation/dexon/core"
	"github.com/dexon-foundation/dexon/core/bloombits"
	"github.com/dexon-foundation/dexon/core/rawdb"
	"github.com/dexon-foundation/dexon/core/state"
	"github.com/dexon-foundation/dexon/core/types"
	"github.com/dexon-foundation/dexon/core/vm"
	"github.com/dexon-foundation/dexon/eth/downloader"
	"github.com/dexon-foundation/dexon/eth/gasprice"

	"github.com/dexon-foundation/dexon/ethdb"
	"github.com/dexon-foundation/dexon/event"
	"github.com/dexon-foundation/dexon/params"
	"github.com/dexon-foundation/dexon/rpc"
)

// DexAPIBackend implements ethapi.Backend for full nodes
type DexAPIBackend struct {
	dex *Dexon
	gpo *gasprice.Oracle
}

// ChainConfig returns the active chain configuration.
func (b *DexAPIBackend) ChainConfig() *params.ChainConfig {
	return b.dex.chainConfig
}

func (b *DexAPIBackend) CurrentBlock() *types.Block {
	return b.dex.blockchain.CurrentBlock()
}

func (b *DexAPIBackend) SetHead(number uint64) {
	b.dex.protocolManager.downloader.Cancel()
	b.dex.blockchain.SetHead(number)
}

func (b *DexAPIBackend) HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error) {
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.dex.blockchain.CurrentBlock().Header(), nil
	}
	return b.dex.blockchain.GetHeaderByNumber(uint64(blockNr)), nil
}

func (b *DexAPIBackend) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return b.dex.blockchain.GetHeaderByHash(hash), nil
}

func (b *DexAPIBackend) BlockByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Block, error) {
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.dex.blockchain.CurrentBlock(), nil
	}
	return b.dex.blockchain.GetBlockByNumber(uint64(blockNr)), nil
}

func (b *DexAPIBackend) StateAndHeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	if blockNr == rpc.PendingBlockNumber {
		block, state := b.dex.BlockChain().GetPending()
		return state, block.Header(), nil
	}
	header, err := b.HeaderByNumber(ctx, blockNr)
	if header == nil || err != nil {
		return nil, nil, err
	}
	stateDb, err := b.dex.BlockChain().StateAt(header.Root)
	return stateDb, header, err
}

func (b *DexAPIBackend) GetBlock(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return b.dex.blockchain.GetBlockByHash(hash), nil
}

func (b *DexAPIBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	if number := rawdb.ReadHeaderNumber(b.dex.chainDb, hash); number != nil {
		return rawdb.ReadReceipts(b.dex.chainDb, hash, *number), nil
	}
	return nil, nil
}

func (b *DexAPIBackend) GetLogs(ctx context.Context, hash common.Hash) ([][]*types.Log, error) {
	number := rawdb.ReadHeaderNumber(b.dex.chainDb, hash)
	if number == nil {
		return nil, nil
	}
	receipts := rawdb.ReadReceipts(b.dex.chainDb, hash, *number)
	if receipts == nil {
		return nil, nil
	}
	logs := make([][]*types.Log, len(receipts))
	for i, receipt := range receipts {
		logs[i] = receipt.Logs
	}
	return logs, nil
}

func (b *DexAPIBackend) GetTd(blockHash common.Hash) *big.Int {
	return b.dex.blockchain.GetTdByHash(blockHash)
}

func (b *DexAPIBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header) (*vm.EVM, func() error, error) {
	state.SetBalance(msg.From(), math.MaxBig256)
	vmError := func() error { return nil }

	context := core.NewEVMContext(msg, header, b.dex.BlockChain(), nil)
	return vm.NewEVM(context, state, b.dex.chainConfig, *b.dex.blockchain.GetVMConfig()), vmError, nil
}

func (b *DexAPIBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.dex.BlockChain().SubscribeRemovedLogsEvent(ch)
}

func (b *DexAPIBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.dex.BlockChain().SubscribeChainEvent(ch)
}

func (b *DexAPIBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.dex.BlockChain().SubscribeChainHeadEvent(ch)
}

func (b *DexAPIBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return b.dex.BlockChain().SubscribeChainSideEvent(ch)
}

func (b *DexAPIBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.dex.BlockChain().SubscribeLogsEvent(ch)
}

func (b *DexAPIBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.dex.txPool.AddLocal(signedTx)
}

func (b *DexAPIBackend) GetPoolTransactions() (types.Transactions, error) {
	pending, err := b.dex.txPool.Pending()
	if err != nil {
		return nil, err
	}
	var txs types.Transactions
	for _, batch := range pending {
		txs = append(txs, batch...)
	}
	return txs, nil
}

func (b *DexAPIBackend) GetPoolTransaction(hash common.Hash) *types.Transaction {
	return b.dex.txPool.Get(hash)
}

func (b *DexAPIBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.dex.txPool.State().GetNonce(addr), nil
}

func (b *DexAPIBackend) Stats() (pending int, queued int) {
	return b.dex.txPool.Stats()
}

func (b *DexAPIBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return b.dex.TxPool().Content()
}

func (b *DexAPIBackend) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
	return b.dex.TxPool().SubscribeNewTxsEvent(ch)
}

func (b *DexAPIBackend) Downloader() *downloader.Downloader {
	return b.dex.Downloader()
}

func (b *DexAPIBackend) ProtocolVersion() int {
	return b.dex.DexVersion()
}

func (b *DexAPIBackend) SuggestPrice(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestPrice(ctx)
}

func (b *DexAPIBackend) ChainDb() ethdb.Database {
	return b.dex.ChainDb()
}

func (b *DexAPIBackend) EventMux() *event.TypeMux {
	return b.dex.EventMux()
}

func (b *DexAPIBackend) AccountManager() *accounts.Manager {
	return b.dex.AccountManager()
}

func (b *DexAPIBackend) RPCGasCap() *big.Int {
	return b.dex.config.RPCGasCap
}

func (b *DexAPIBackend) BloomStatus() (uint64, uint64) {
	sections, _, _ := b.dex.bloomIndexer.Sections()
	return params.BloomBitsBlocks, sections
}

func (b *DexAPIBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.dex.bloomRequests)
	}
}
