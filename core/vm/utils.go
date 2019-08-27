package vm

import (
	"errors"
	"math/big"

	"github.com/tangerine-network/go-tangerine/common"
	"github.com/tangerine-network/go-tangerine/core/state"
	"github.com/tangerine-network/go-tangerine/crypto"
	"github.com/tangerine-network/go-tangerine/log"
	dexCore "github.com/tangerine-network/tangerine-consensus/core"
)

type GovUtilInterface interface {
	GetHeadGovState() (*GovernanceState, error)
	StateAt(height uint64) (*state.StateDB, error)
}

type GovUtil struct {
	Intf GovUtilInterface
}

func (g GovUtil) GetRoundHeight(round uint64) uint64 {
	gs, err := g.Intf.GetHeadGovState()
	if err != nil {
		return 0
	}
	return gs.RoundHeight(big.NewInt(int64(round))).Uint64()
}

func (g GovUtil) GetStateAtRound(round uint64) (*GovernanceState, error) {
	height := g.GetRoundHeight(round)

	if round != 0 && height == 0 {
		log.Error("Governance state incorrect", "round", round, "got height", height)
		return nil, errors.New("incorrect governance state")
	}

	s, err := g.Intf.StateAt(height)
	if err != nil {
		return nil, err
	}
	return &GovernanceState{StateDB: s}, nil
}

func (g GovUtil) GetConfigState(round uint64) (*GovernanceState, error) {
	if round < dexCore.ConfigRoundShift {
		round = 0
	} else {
		round -= dexCore.ConfigRoundShift
	}
	return g.GetStateAtRound(round)
}

func (g *GovUtil) CRSRound() uint64 {
	gs, err := g.Intf.GetHeadGovState()
	if err != nil {
		return 0
	}
	return gs.CRSRound().Uint64()
}

func (g GovUtil) CRS(round uint64) common.Hash {
	if round <= dexCore.DKGDelayRound {
		s, err := g.GetStateAtRound(0)
		if err != nil {
			return common.Hash{}
		}
		crs := s.CRS()
		for i := uint64(0); i < round; i++ {
			crs = crypto.Keccak256Hash(crs[:])
		}
		return crs
	}
	if round > g.CRSRound() {
		return common.Hash{}
	}
	var s *GovernanceState
	var err error
	if round == g.CRSRound() {
		s, err = g.Intf.GetHeadGovState()
		if err != nil {
			return common.Hash{}
		}
	} else {
		s, err = g.GetStateAtRound(round)
		if err != nil {
			return common.Hash{}
		}
	}
	return s.CRS()
}
