package core

import (
	"fmt"
	"math/big"
	"time"

	dexCore "github.com/dexon-foundation/dexon-consensus/core"
	coreTypes "github.com/dexon-foundation/dexon-consensus/core/types"
	dkgTypes "github.com/dexon-foundation/dexon-consensus/core/types/dkg"

	"github.com/dexon-foundation/dexon/core/state"
	"github.com/dexon-foundation/dexon/core/vm"
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
	db GovernanceStateDB
}

func NewGovernance(db GovernanceStateDB) *Governance {
	return &Governance{db: db}
}

func (g *Governance) GetHeadHelper() *vm.GovernanceStateHelper {
	headState, err := g.db.State()
	if err != nil {
		log.Error("Governance head state not ready", "err", err)
		panic(err)
	}
	return &vm.GovernanceStateHelper{headState}
}

func (g *Governance) getHelperAtRound(round uint64) *vm.GovernanceStateHelper {
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
	return &vm.GovernanceStateHelper{s}
}

func (g *Governance) GetConfigHelper(round uint64) *vm.GovernanceStateHelper {
	if round < dexCore.ConfigRoundShift {
		round = 0
	} else {
		round -= dexCore.ConfigRoundShift
	}
	return g.getHelperAtRound(round)
}

func (g *Governance) GetRoundHeight(round uint64) uint64 {
	return g.GetHeadHelper().RoundHeight(big.NewInt(int64(round))).Uint64()
}

func (g *Governance) Configuration(round uint64) *coreTypes.Config {
	configHelper := g.GetConfigHelper(round)
	c := configHelper.Configuration()
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

func (g *Governance) DKGComplaints(round uint64) []*dkgTypes.Complaint {
	headHelper := g.GetHeadHelper()
	var dkgComplaints []*dkgTypes.Complaint
	for _, pk := range headHelper.DKGComplaints(big.NewInt(int64(round))) {
		x := new(dkgTypes.Complaint)
		if err := rlp.DecodeBytes(pk, x); err != nil {
			panic(err)
		}
		dkgComplaints = append(dkgComplaints, x)
	}
	return dkgComplaints
}

func (g *Governance) DKGMasterPublicKeys(round uint64) []*dkgTypes.MasterPublicKey {
	headHelper := g.GetHeadHelper()
	var dkgMasterPKs []*dkgTypes.MasterPublicKey
	for _, pk := range headHelper.DKGMasterPublicKeys(big.NewInt(int64(round))) {
		x := new(dkgTypes.MasterPublicKey)
		if err := rlp.DecodeBytes(pk, x); err != nil {
			panic(err)
		}
		dkgMasterPKs = append(dkgMasterPKs, x)
	}
	return dkgMasterPKs
}

func (g *Governance) IsDKGFinal(round uint64) bool {
	headHelper := g.GetHeadHelper()
	config := g.Configuration(round)
	threshold := 2*uint64(config.DKGSetSize)/3 + 1
	count := headHelper.DKGFinalizedsCount(big.NewInt(int64(round))).Uint64()
	return count >= threshold
}
