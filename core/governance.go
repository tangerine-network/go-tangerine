package core

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	coreCommon "github.com/dexon-foundation/dexon-consensus/common"
	dexCore "github.com/dexon-foundation/dexon-consensus/core"
	coreCrypto "github.com/dexon-foundation/dexon-consensus/core/crypto"
	coreEcdsa "github.com/dexon-foundation/dexon-consensus/core/crypto/ecdsa"
	coreTypes "github.com/dexon-foundation/dexon-consensus/core/types"
	dkgTypes "github.com/dexon-foundation/dexon-consensus/core/types/dkg"

	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/core/state"
	"github.com/dexon-foundation/dexon/core/vm"
	"github.com/dexon-foundation/dexon/crypto"
	"github.com/dexon-foundation/dexon/log"
	"github.com/dexon-foundation/dexon/rlp"
)

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

type Governance struct {
	db           GovernanceStateDB
	nodeSetCache *dexCore.NodeSetCache
}

func NewGovernance(db GovernanceStateDB) *Governance {
	g := &Governance{db: db}
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

func (d *Governance) CRSRound() uint64 {
	return d.GetHeadState().CRSRound().Uint64()
}

// CRS returns the CRS for a given round.
func (d *Governance) CRS(round uint64) coreCommon.Hash {
	if round <= dexCore.DKGDelayRound {
		s := d.GetStateAtRound(0)
		crs := s.CRS()
		for i := uint64(0); i < round; i++ {
			crs = crypto.Keccak256Hash(crs[:])
		}
		return coreCommon.Hash(crs)
	}
	if round > d.CRSRound() {
		return coreCommon.Hash{}
	}
	var s *vm.GovernanceState
	if round == d.CRSRound() {
		s = d.GetHeadState()
	} else {
		s = d.GetStateAtRound(round)
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
		DKGSetSize:       c.DKGSetSize,
		RoundLength:      c.RoundLength,
		MinBlockInterval: time.Duration(c.MinBlockInterval) * time.Millisecond,
	}
}

func (g *Governance) GetRoundHeight(round uint64) uint64 {
	return g.GetHeadState().RoundHeight(big.NewInt(int64(round))).Uint64()
}

// NodeSet returns the current node set.
func (d *Governance) NodeSet(round uint64) []coreCrypto.PublicKey {
	s := d.GetStateForConfigAtRound(round)
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

func (d *Governance) NotarySet(round uint64) (map[string]struct{}, error) {
	notarySet, err := d.nodeSetCache.GetNotarySet(round)
	if err != nil {
		return nil, err
	}

	r := make(map[string]struct{}, len(notarySet))
	for id := range notarySet {
		if key, exists := d.nodeSetCache.GetPublicKey(id); exists {
			r[hex.EncodeToString(key.Bytes())] = struct{}{}
		}
	}
	return r, nil
}

func (d *Governance) NotarySetNodeKeyAddresses(round uint64) (map[common.Address]struct{}, error) {
	notarySet, err := d.nodeSetCache.GetNotarySet(round)
	if err != nil {
		return nil, err
	}

	r := make(map[common.Address]struct{}, len(notarySet))
	for id := range notarySet {
		r[vm.IdToAddress(id)] = struct{}{}
	}
	return r, nil
}

func (d *Governance) DKGSet(round uint64) (map[string]struct{}, error) {
	dkgSet, err := d.nodeSetCache.GetDKGSet(round)
	if err != nil {
		return nil, err
	}

	r := make(map[string]struct{}, len(dkgSet))
	for id := range dkgSet {
		if key, exists := d.nodeSetCache.GetPublicKey(id); exists {
			r[hex.EncodeToString(key.Bytes())] = struct{}{}
		}
	}
	return r, nil
}

func (g *Governance) DKGComplaints(round uint64) []*dkgTypes.Complaint {
	s := g.GetStateForDKGAtRound(round)
	if s == nil {
		return nil
	}

	var dkgComplaints []*dkgTypes.Complaint
	for _, pk := range s.DKGComplaints() {
		x := new(dkgTypes.Complaint)
		if err := rlp.DecodeBytes(pk, x); err != nil {
			panic(err)
		}
		dkgComplaints = append(dkgComplaints, x)
	}
	return dkgComplaints
}

func (g *Governance) DKGMasterPublicKeys(round uint64) []*dkgTypes.MasterPublicKey {
	s := g.GetStateForDKGAtRound(round)
	if s == nil {
		return nil
	}
	return s.UniqueDKGMasterPublicKeys()
}

func (g *Governance) IsDKGMPKReady(round uint64) bool {
	s := g.GetStateForDKGAtRound(round)
	if s == nil {
		return false
	}
	config := g.Configuration(round)
	threshold := 2*uint64(config.DKGSetSize)/3 + 1
	count := s.DKGMPKReadysCount().Uint64()
	return count >= threshold
}

func (g *Governance) IsDKGFinal(round uint64) bool {
	s := g.GetStateForDKGAtRound(round)
	if s == nil {
		return false
	}
	config := g.Configuration(round)
	threshold := 2*uint64(config.DKGSetSize)/3 + 1
	count := s.DKGFinalizedsCount().Uint64()
	return count >= threshold
}

func (g *Governance) MinGasPrice(round uint64) *big.Int {
	return g.GetStateForConfigAtRound(round).MinGasPrice()
}

func (g *Governance) DKGResetCount(round uint64) uint64 {
	return g.GetHeadState().DKGResetCount(big.NewInt(int64(round))).Uint64()
}
