package downloader

import (
	"fmt"
	"sync"

	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/core"
	"github.com/dexon-foundation/dexon/core/state"
	"github.com/dexon-foundation/dexon/core/types"
	"github.com/dexon-foundation/dexon/crypto"
	"github.com/dexon-foundation/dexon/ethdb"
	"github.com/dexon-foundation/dexon/log"
	"github.com/dexon-foundation/dexon/trie"
)

// governanceDB is backed by memory db for fast sync.
// it implements core.GovernanceStateDB
type governanceStateDB struct {
	db          ethdb.Database
	headRoot    common.Hash
	headHeight  uint64
	height2Root map[uint64]common.Hash
	mu          sync.Mutex
}

func (g *governanceStateDB) State() (*state.StateDB, error) {
	return state.New(g.headRoot, state.NewDatabase(g.db))
}

func (g *governanceStateDB) StateAt(height uint64) (*state.StateDB, error) {
	root, exists := g.height2Root[height]
	if !exists {
		return nil, fmt.Errorf("Governance state not ready, height: %d", height)
	}
	return state.New(root, state.NewDatabase(g.db))
}

func (g *governanceStateDB) StoreState(s *types.GovState) {
	g.mu.Lock()
	defer g.mu.Unlock()
	log.Debug("Store state", "height", s.Number.Uint64())

	// Store the height -> root mapping.
	g.height2Root[s.Number.Uint64()] = s.Root

	// Store the account.
	for _, node := range s.Proof {
		g.db.Put(crypto.Keccak256(node), node)
	}

	// Store the storage.
	triedb := trie.NewDatabase(g.db)
	t, err := trie.New(common.Hash{}, triedb)
	if err != nil {
		panic(err)
	}
	for _, kv := range s.Storage {
		t.TryUpdate(kv[0], kv[1])
	}
	t.Commit(nil)
	triedb.Commit(t.Hash(), false)

	if s.Number.Uint64() > g.headHeight {
		log.Debug("Governance head root changed", "number", s.Number.Uint64())
		g.headRoot = s.Root
		g.headHeight = s.Number.Uint64()
	}
}

// This is a governance for fast sync
type governance struct {
	*core.Governance
	db *governanceStateDB
}

func newGovernance(headState *types.GovState) *governance {
	db := ethdb.NewMemDatabase()
	govStateDB := &governanceStateDB{
		db:          db,
		headRoot:    headState.Root,
		headHeight:  headState.Number.Uint64(),
		height2Root: make(map[uint64]common.Hash),
	}
	govStateDB.StoreState(headState)
	return &governance{
		Governance: core.NewGovernance(govStateDB),
		db:         govStateDB,
	}
}

func (g *governance) StoreState(s *types.GovState) {
	g.db.StoreState(s)
}
