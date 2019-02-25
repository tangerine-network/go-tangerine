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

func (g *Governance) GetRoundHeight(round uint64) uint64 {
	return g.GetHeadState().RoundHeight(big.NewInt(int64(round))).Uint64()
}

func (g *Governance) Configuration(round uint64) *coreTypes.Config {
	configHelper := g.GetStateForConfigAtRound(round)
	c := configHelper.Configuration()
	return &coreTypes.Config{
		LambdaBA:         time.Duration(c.LambdaBA) * time.Millisecond,
		LambdaDKG:        time.Duration(c.LambdaDKG) * time.Millisecond,
		NotarySetSize:    c.NotarySetSize,
		DKGSetSize:       c.DKGSetSize,
		RoundLength:      c.RoundLength,
		MinBlockInterval: time.Duration(c.MinBlockInterval) * time.Millisecond,
	}
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
