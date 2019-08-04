package core

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/hashicorp/golang-lru/simplelru"
	coreCommon "github.com/tangerine-network/tangerine-consensus/common"
	dexCore "github.com/tangerine-network/tangerine-consensus/core"
	coreCrypto "github.com/tangerine-network/tangerine-consensus/core/crypto"
	coreEcdsa "github.com/tangerine-network/tangerine-consensus/core/crypto/ecdsa"
	coreTypes "github.com/tangerine-network/tangerine-consensus/core/types"
	dkgTypes "github.com/tangerine-network/tangerine-consensus/core/types/dkg"
	coreUtils "github.com/tangerine-network/tangerine-consensus/core/utils"

	"github.com/tangerine-network/go-tangerine/common"
	"github.com/tangerine-network/go-tangerine/core/state"
	"github.com/tangerine-network/go-tangerine/core/vm"
	"github.com/tangerine-network/go-tangerine/log"
)

const dkgCacheSize = 5

type GovernanceStateDB interface {
	State() (*state.StateDB, error)
	StateAt(height uint64) (*state.StateDB, error)
}

func NewGovernanceStateDB(bc *BlockChain) GovernanceStateDB {
	return &governanceStateDB{bc: bc}
}

type governanceStateDB struct {
	bc *BlockChain
}

func (g *governanceStateDB) State() (*state.StateDB, error) {
	return g.bc.State()
}

func (g *governanceStateDB) StateAt(height uint64) (*state.StateDB, error) {
	header := g.bc.GetHeaderByNumber(height)
	if header == nil {
		return nil, fmt.Errorf("header at %d not exists", height)
	}
	return g.bc.StateAt(header.Root)
}

type dkgCacheItem struct {
	Round               uint64
	Reset               uint64
	MasterPublicKeysLen uint64
	MasterPublicKeys    []*dkgTypes.MasterPublicKey
	ComplaintsLen       uint64
	Complaints          []*dkgTypes.Complaint
}

type Governance struct {
	db           GovernanceStateDB
	nodeSetCache *dexCore.NodeSetCache
	dkgCache     *simplelru.LRU
	dkgCacheMu   sync.RWMutex
	util         vm.GovUtil
}

func NewGovernance(db GovernanceStateDB) *Governance {
	cache, err := simplelru.NewLRU(dkgCacheSize, nil)
	if err != nil {
		log.Error("Failed to initialize DKG cache", "error", err)
		return nil
	}
	g := &Governance{
		db:       db,
		dkgCache: cache,
	}
	g.nodeSetCache = dexCore.NewNodeSetCache(g)
	g.util = vm.GovUtil{g}
	return g
}

func (g *Governance) GetHeadGovState() (*vm.GovernanceState, error) {
	headState, err := g.db.State()
	if err != nil {
		log.Error("Governance head state not ready", "err", err)
		return nil, err
	}
	return &vm.GovernanceState{StateDB: headState}, nil
}

func (g *Governance) StateAt(height uint64) (*state.StateDB, error) {
	return g.db.StateAt(height)
}

func (g *Governance) GetConfigState(round uint64) (*vm.GovernanceState, error) {
	return g.util.GetConfigState(round)
}

func (g *Governance) GetStateForDKGAtRound(round uint64) (*vm.GovernanceState, error) {
	gs, err := g.GetHeadGovState()
	if err != nil {
		return nil, err
	}
	dkgRound := gs.DKGRound().Uint64()
	if round > dkgRound {
		return nil, fmt.Errorf("invalid round input: %d, dkgRound: %d", round, dkgRound)
	}
	if round == dkgRound {
		return gs, nil
	}
	return g.util.GetRoundState(round)
}

func (g *Governance) CRSRound() uint64 {
	gs, err := g.GetHeadGovState()
	if err != nil {
		log.Error("Failed to get head governance state", "err", err)
		return 0
	}
	return gs.CRSRound().Uint64()
}

// CRS returns the CRS for a given round.
func (g *Governance) CRS(round uint64) coreCommon.Hash {
	return coreCommon.Hash(g.util.CRS(round))
}

func (g *Governance) GetRoundHeight(round uint64) uint64 {
	return g.util.GetRoundHeight(round)
}

func (g *Governance) Configuration(round uint64) *coreTypes.Config {
	s, err := g.util.GetConfigState(round)
	if err != nil {
		panic(err)
	}
	c := s.Configuration()
	return &coreTypes.Config{
		LambdaBA:         time.Duration(c.LambdaBA) * time.Millisecond,
		LambdaDKG:        time.Duration(c.LambdaDKG) * time.Millisecond,
		NotarySetSize:    uint32(s.NotarySetSize().Uint64()),
		RoundLength:      c.RoundLength,
		MinBlockInterval: time.Duration(c.MinBlockInterval) * time.Millisecond,
	}
}

// NodeSet returns the current node set.
func (g *Governance) NodeSet(round uint64) []coreCrypto.PublicKey {
	configState, err := g.util.GetConfigState(round)
	if err != nil {
		panic(err)
	}

	var pks []coreCrypto.PublicKey
	for _, n := range configState.QualifiedNodes() {
		pk, err := coreEcdsa.NewPublicKeyFromByteSlice(n.PublicKey)
		if err != nil {
			panic(err)
		}
		pks = append(pks, pk)
	}
	return pks
}

func (g *Governance) PurgeNotarySet(round uint64) {
	g.nodeSetCache.Purge(round)
}

func (g *Governance) NotarySet(round uint64) (map[string]struct{}, error) {
	notarySet, err := g.nodeSetCache.GetNotarySet(round)
	if err != nil {
		return nil, err
	}

	r := make(map[string]struct{}, len(notarySet))
	for id := range notarySet {
		if key, exists := g.nodeSetCache.GetPublicKey(id); exists {
			r[hex.EncodeToString(key.Bytes())] = struct{}{}
		}
	}
	return r, nil
}

func (g *Governance) DKGSetNodeKeyAddresses(round uint64) (map[common.Address]struct{}, error) {
	config := g.Configuration(round)

	mpks := g.DKGMasterPublicKeys(round)
	complaints := g.DKGComplaints(round)
	threshold := coreUtils.GetDKGThreshold(&coreTypes.Config{
		NotarySetSize: config.NotarySetSize})

	_, ids, err := dkgTypes.CalcQualifyNodes(mpks, complaints, threshold)
	if err != nil {
		return nil, err
	}

	r := make(map[common.Address]struct{})
	for id := range ids {
		r[vm.IdToAddress(id)] = struct{}{}
	}
	return r, nil
}

func (g *Governance) getOrUpdateDKGCache(round uint64) *dkgCacheItem {
	s, err := g.GetStateForDKGAtRound(round)
	if err != nil {
		log.Error("Failed to get DKG state", "round", round, "err", err)
		return nil
	}
	reset := s.DKGResetCount(new(big.Int).SetUint64(round))
	mpksLen := s.LenDKGMasterPublicKeys().Uint64()
	compsLen := s.LenDKGComplaints().Uint64()

	var cache *dkgCacheItem

	g.dkgCacheMu.RLock()
	if v, ok := g.dkgCache.Get(round); ok {
		cache = v.(*dkgCacheItem)
	}
	g.dkgCacheMu.RUnlock()

	if cache != nil && cache.Reset == reset.Uint64() &&
		cache.MasterPublicKeysLen == mpksLen &&
		cache.ComplaintsLen == compsLen {
		return cache
	}

	g.dkgCacheMu.Lock()
	defer g.dkgCacheMu.Unlock()

	cache = &dkgCacheItem{
		Round: round,
		Reset: reset.Uint64(),
	}

	if cache == nil || cache.MasterPublicKeysLen != mpksLen {
		cache.MasterPublicKeys = s.DKGMasterPublicKeyItems()
		cache.MasterPublicKeysLen = uint64(len(cache.MasterPublicKeys))
	}

	if cache == nil || cache.ComplaintsLen != compsLen {
		cache.Complaints = s.DKGComplaintItems()
		cache.ComplaintsLen = uint64(len(cache.Complaints))
	}

	g.dkgCache.Add(round, cache)
	return cache
}

func (g *Governance) DKGComplaints(round uint64) []*dkgTypes.Complaint {
	cache := g.getOrUpdateDKGCache(round)
	return cache.Complaints
}

func (g *Governance) DKGMasterPublicKeys(round uint64) []*dkgTypes.MasterPublicKey {
	cache := g.getOrUpdateDKGCache(round)
	return cache.MasterPublicKeys
}

func (g *Governance) IsDKGMPKReady(round uint64) bool {
	s, err := g.GetStateForDKGAtRound(round)
	if err != nil {
		log.Error("Failed to get state for DKG", "round", round, "err", err)
		return false
	}
	config := g.Configuration(round)
	threshold := 2*uint64(config.NotarySetSize)/3 + 1
	count := s.DKGMPKReadysCount().Uint64()
	return count >= threshold
}

func (g *Governance) IsDKGFinal(round uint64) bool {
	s, err := g.GetStateForDKGAtRound(round)
	if err != nil {
		log.Error("Failed to get state for DKG", "round", round, "err", err)
		return false
	}
	config := g.Configuration(round)
	threshold := 2*uint64(config.NotarySetSize)/3 + 1
	count := s.DKGFinalizedsCount().Uint64()
	return count >= threshold
}

func (g *Governance) IsDKGSuccess(round uint64) bool {
	s, err := g.GetStateForDKGAtRound(round)
	if err != nil {
		log.Error("Failed to get state for DKG", "round", round, "err", err)
		return false
	}
	return s.DKGSuccessesCount().Uint64() >=
		uint64(coreUtils.GetDKGValidThreshold(g.Configuration(round)))
}

func (g *Governance) DKGResetCount(round uint64) uint64 {
	gs, err := g.GetHeadGovState()
	if err != nil {
		log.Error("Failed to get head governance state", "err", err)
	}
	return gs.DKGResetCount(big.NewInt(int64(round))).Uint64()
}
