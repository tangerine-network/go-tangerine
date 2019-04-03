package core

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"sync"
	"time"

	coreCommon "github.com/dexon-foundation/dexon-consensus/common"
	dexCore "github.com/dexon-foundation/dexon-consensus/core"
	coreCrypto "github.com/dexon-foundation/dexon-consensus/core/crypto"
	coreEcdsa "github.com/dexon-foundation/dexon-consensus/core/crypto/ecdsa"
	coreTypes "github.com/dexon-foundation/dexon-consensus/core/types"
	dkgTypes "github.com/dexon-foundation/dexon-consensus/core/types/dkg"
	coreUtils "github.com/dexon-foundation/dexon-consensus/core/utils"
	"github.com/hashicorp/golang-lru/simplelru"

	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/core/state"
	"github.com/dexon-foundation/dexon/core/vm"
	"github.com/dexon-foundation/dexon/crypto"
	"github.com/dexon-foundation/dexon/log"
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
	return g
}

func (g *Governance) GetHeadState() *vm.GovernanceState {
	headState, err := g.db.State()
	if err != nil {
		log.Error("Governance head state not ready", "err", err)
		panic(err)
	}
	return &vm.GovernanceState{StateDB: headState}
}

func (g *Governance) getHelperAtRound(round uint64) *vm.GovernanceState {
	height := g.GetRoundHeight(round)

	// Sanity check
	if round != 0 && height == 0 {
		log.Error("Governance state incorrect", "round", round, "got height", height)
		panic("round != 0 but height == 0")
	}

	s, err := g.db.StateAt(height)
	if err != nil {
		log.Error("Governance state not ready", "round", round, "height", height, "err", err)
		panic(err)
	}
	return &vm.GovernanceState{StateDB: s}
}

func (g *Governance) GetStateForConfigAtRound(round uint64) *vm.GovernanceState {
	if round < dexCore.ConfigRoundShift {
		round = 0
	} else {
		round -= dexCore.ConfigRoundShift
	}
	return g.getHelperAtRound(round)
}

func (g *Governance) GetStateAtRound(round uint64) *vm.GovernanceState {
	height := g.GetRoundHeight(round)
	s, err := g.db.StateAt(height)
	if err != nil {
		log.Error("Governance state not ready", "round", round, "height", height, "err", err)
		panic(err)
	}
	return &vm.GovernanceState{StateDB: s}
}

func (g *Governance) GetStateForDKGAtRound(round uint64) *vm.GovernanceState {
	dkgRound := g.GetHeadState().DKGRound().Uint64()
	if round > dkgRound {
		return nil
	}
	if round == dkgRound {
		return g.GetHeadState()
	}
	return g.GetStateAtRound(round)
}

func (g *Governance) CRSRound() uint64 {
	return g.GetHeadState().CRSRound().Uint64()
}

// CRS returns the CRS for a given round.
func (g *Governance) CRS(round uint64) coreCommon.Hash {
	if round <= dexCore.DKGDelayRound {
		s := g.GetStateAtRound(0)
		crs := s.CRS()
		for i := uint64(0); i < round; i++ {
			crs = crypto.Keccak256Hash(crs[:])
		}
		return coreCommon.Hash(crs)
	}
	if round > g.CRSRound() {
		return coreCommon.Hash{}
	}
	var s *vm.GovernanceState
	if round == g.CRSRound() {
		s = g.GetHeadState()
	} else {
		s = g.GetStateAtRound(round)
	}
	return coreCommon.Hash(s.CRS())
}

func (g *Governance) Configuration(round uint64) *coreTypes.Config {
	configHelper := g.GetStateForConfigAtRound(round)
	c := configHelper.Configuration()
	return &coreTypes.Config{
		LambdaBA:         time.Duration(c.LambdaBA) * time.Millisecond,
		LambdaDKG:        time.Duration(c.LambdaDKG) * time.Millisecond,
		NotarySetSize:    uint32(configHelper.NotarySetSize().Uint64()),
		RoundLength:      c.RoundLength,
		MinBlockInterval: time.Duration(c.MinBlockInterval) * time.Millisecond,
	}
}

func (g *Governance) GetRoundHeight(round uint64) uint64 {
	return g.GetHeadState().RoundHeight(big.NewInt(int64(round))).Uint64()
}

// NodeSet returns the current node set.
func (g *Governance) NodeSet(round uint64) []coreCrypto.PublicKey {
	s := g.GetStateForConfigAtRound(round)
	var pks []coreCrypto.PublicKey

	for _, n := range s.QualifiedNodes() {
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
	s := g.GetStateForDKGAtRound(round)
	if s == nil {
		log.Error("Failed to get DKG state", "round", round)
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
	s := g.GetStateForDKGAtRound(round)
	if s == nil {
		return false
	}
	config := g.Configuration(round)
	threshold := 2*uint64(config.NotarySetSize)/3 + 1
	count := s.DKGMPKReadysCount().Uint64()
	return count >= threshold
}

func (g *Governance) IsDKGFinal(round uint64) bool {
	s := g.GetStateForDKGAtRound(round)
	if s == nil {
		return false
	}
	config := g.Configuration(round)
	threshold := 2*uint64(config.NotarySetSize)/3 + 1
	count := s.DKGFinalizedsCount().Uint64()
	return count >= threshold
}

func (g *Governance) MinGasPrice(round uint64) *big.Int {
	return g.GetStateForConfigAtRound(round).MinGasPrice()
}

func (g *Governance) DKGResetCount(round uint64) uint64 {
	return g.GetHeadState().DKGResetCount(big.NewInt(int64(round))).Uint64()
}
