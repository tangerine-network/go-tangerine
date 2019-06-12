package indexer

import (
	"math/big"

	"github.com/tangerine-network/go-tangerine/common"
	"github.com/tangerine-network/go-tangerine/consensus"
	"github.com/tangerine-network/go-tangerine/core"
	"github.com/tangerine-network/go-tangerine/core/state"
	"github.com/tangerine-network/go-tangerine/core/types"
	"github.com/tangerine-network/go-tangerine/core/vm"
	"github.com/tangerine-network/go-tangerine/event"
	"github.com/tangerine-network/go-tangerine/params"
	"github.com/tangerine-network/go-tangerine/rlp"
)

// ReadOnlyBlockChain defines safe reading blockchain interface by removing write
// methods of core.BlockChain struct.
type ReadOnlyBlockChain interface {
	BadBlocks() []*types.Block
	Config() *params.ChainConfig
	CurrentBlock() *types.Block
	CurrentFastBlock() *types.Block
	CurrentHeader() *types.Header
	Engine() consensus.Engine
	GasLimit() uint64
	Genesis() *types.Block
	GetAncestor(common.Hash, uint64, uint64, *uint64) (common.Hash, uint64)
	GetBlock(common.Hash, uint64) *types.Block
	GetBlockByHash(common.Hash) *types.Block
	GetBlockByNumber(uint64) *types.Block
	GetBlockHashesFromHash(common.Hash, uint64) []common.Hash
	GetBlocksFromHash(common.Hash, int) (blocks []*types.Block)
	GetBody(common.Hash) *types.Body
	GetBodyRLP(common.Hash) rlp.RawValue
	GetGovStateByHash(common.Hash) (*types.GovState, error)
	GetGovStateByNumber(uint64) (*types.GovState, error)
	GetHeader(common.Hash, uint64) *types.Header
	GetHeaderByHash(common.Hash) *types.Header
	GetHeaderByNumber(number uint64) *types.Header
	GetReceiptsByHash(common.Hash) types.Receipts
	GetRoundHeight(uint64) (uint64, bool)
	GetTd(common.Hash, uint64) *big.Int
	GetTdByHash(common.Hash) *big.Int
	GetUnclesInChain(*types.Block, int) []*types.Header
	GetVMConfig() *vm.Config
	HasBlock(common.Hash, uint64) bool
	HasBlockAndState(common.Hash, uint64) bool
	HasHeader(common.Hash, uint64) bool
	HasState(common.Hash) bool
	Processor() core.Processor
	State() (*state.StateDB, error)
	StateAt(root common.Hash) (*state.StateDB, error)
	StateCache() state.Database
	SubscribeChainEvent(chan<- core.ChainEvent) event.Subscription
	SubscribeChainHeadEvent(chan<- core.ChainHeadEvent) event.Subscription
	SubscribeChainSideEvent(chan<- core.ChainSideEvent) event.Subscription
	SubscribeLogsEvent(chan<- []*types.Log) event.Subscription
	SubscribeRemovedLogsEvent(chan<- core.RemovedLogsEvent) event.Subscription
}

// access protection
type ro interface {
	ReadOnlyBlockChain
}

// ROBlockChain struct for safe read.
type ROBlockChain struct {
	ro
}

// NewROBlockChain converts original block chain to readonly interface.
func NewROBlockChain(bc *core.BlockChain) ReadOnlyBlockChain {
	return &ROBlockChain{ro: bc}
}
