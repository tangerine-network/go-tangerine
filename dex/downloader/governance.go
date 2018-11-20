package downloader

import (
	"math/big"
	"sync"
	"time"

	dexCore "github.com/dexon-foundation/dexon-consensus/core"
	coreTypes "github.com/dexon-foundation/dexon-consensus/core/types"
	dkgTypes "github.com/dexon-foundation/dexon-consensus/core/types/dkg"

	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/core/state"
	"github.com/dexon-foundation/dexon/core/types"
	"github.com/dexon-foundation/dexon/core/vm"
	"github.com/dexon-foundation/dexon/crypto"
	"github.com/dexon-foundation/dexon/ethdb"
	"github.com/dexon-foundation/dexon/log"
	"github.com/dexon-foundation/dexon/rlp"
	"github.com/dexon-foundation/dexon/trie"
)

// This is a goverance for fast sync
type governance struct {
	db          ethdb.Database
	headRoot    common.Hash
	headHeight  uint64
	height2Root map[uint64]common.Hash
	mu          sync.Mutex
}

func newGovernance(headState *types.GovState) *governance {
	db := ethdb.NewMemDatabase()
	g := &governance{
		db:          db,
		headRoot:    headState.Root,
		headHeight:  headState.Number.Uint64(),
		height2Root: make(map[uint64]common.Hash),
	}
	g.StoreState(headState)

	statedb, err := state.New(headState.Root, state.NewDatabase(g.db))
	if statedb == nil || err != nil {
		log.Error("New governance fail", "statedb == nil", statedb == nil, "err", err)
	}
	return g
}

func (g *governance) getHeadHelper() *vm.GovernanceStateHelper {
	return g.getHelper(g.headRoot)
}

func (g *governance) getHelperByRound(round uint64) *vm.GovernanceStateHelper {
	height := g.GetRoundHeight(round)
	root, exists := g.height2Root[height]
	if !exists {
		log.Debug("Gov get helper by round", "round", round, "exists", exists)
		return nil
	}
	return g.getHelper(root)
}

func (g *governance) getHelper(root common.Hash) *vm.GovernanceStateHelper {
	statedb, err := state.New(root, state.NewDatabase(g.db))
	if statedb == nil || err != nil {
		return nil
	}
	return &vm.GovernanceStateHelper{statedb}
}

func (g *governance) GetRoundHeight(round uint64) uint64 {
	var h uint64
	if helper := g.getHeadHelper(); helper != nil {
		h = helper.RoundHeight(big.NewInt(int64(round))).Uint64()
	}
	return h
}

func (g *governance) StoreState(s *types.GovState) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// store the hight -> root mapping
	g.height2Root[s.Number.Uint64()] = s.Root

	// store the account
	for _, node := range s.Proof {
		g.db.Put(crypto.Keccak256(node), node)
	}

	// store the storage
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
		log.Debug("Gov head root changed", "number", s.Number.Uint64())
		g.headRoot = s.Root
		g.headHeight = s.Number.Uint64()
	}
}

// Return the genesis configuration if round == 0.
func (g *governance) Configuration(round uint64) *coreTypes.Config {
	if round < dexCore.ConfigRoundShift {
		round = 0
	} else {
		round -= dexCore.ConfigRoundShift
	}
	helper := g.getHelperByRound(round)
	if helper == nil {
		log.Warn("Get config helper fail", "round - round shift", round)
		return nil
	}
	c := helper.Configuration()
	return &coreTypes.Config{
		NumChains:        c.NumChains,
		LambdaBA:         time.Duration(c.LambdaBA) * time.Millisecond,
		LambdaDKG:        time.Duration(c.LambdaDKG) * time.Millisecond,
		K:                int(c.K),
		PhiRatio:         c.PhiRatio,
		NotarySetSize:    c.NotarySetSize,
		DKGSetSize:       c.DKGSetSize,
		RoundInterval:    time.Duration(c.RoundInterval) * time.Millisecond,
		MinBlockInterval: time.Duration(c.MinBlockInterval) * time.Millisecond,
	}
}

// DKGComplaints gets all the DKGComplaints of round.
func (g *governance) DKGComplaints(round uint64) []*dkgTypes.Complaint {
	helper := g.getHeadHelper()
	var dkgComplaints []*dkgTypes.Complaint
	for _, pk := range helper.DKGComplaints(big.NewInt(int64(round))) {
		x := new(dkgTypes.Complaint)
		if err := rlp.DecodeBytes(pk, x); err != nil {
			panic(err)
		}
		dkgComplaints = append(dkgComplaints, x)
	}
	return dkgComplaints
}

// DKGMasterPublicKeys gets all the DKGMasterPublicKey of round.
func (g *governance) DKGMasterPublicKeys(round uint64) []*dkgTypes.MasterPublicKey {
	helper := g.getHeadHelper()
	var dkgMasterPKs []*dkgTypes.MasterPublicKey
	for _, pk := range helper.DKGMasterPublicKeys(big.NewInt(int64(round))) {
		x := new(dkgTypes.MasterPublicKey)
		if err := rlp.DecodeBytes(pk, x); err != nil {
			panic(err)
		}
		dkgMasterPKs = append(dkgMasterPKs, x)
	}
	return dkgMasterPKs
}

// IsDKGFinal checks if DKG is final.
func (g *governance) IsDKGFinal(round uint64) bool {
	helper := g.getHeadHelper()
	threshold := 2*uint64(g.Configuration(round).DKGSetSize)/3 + 1
	count := helper.DKGFinalizedsCount(big.NewInt(int64(round))).Uint64()
	return count >= threshold
}
