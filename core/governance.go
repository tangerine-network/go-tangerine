package core

import (
	"math/big"
	"time"

	dexCore "github.com/dexon-foundation/dexon-consensus/core"
	coreTypes "github.com/dexon-foundation/dexon-consensus/core/types"
	dkgTypes "github.com/dexon-foundation/dexon-consensus/core/types/dkg"

	"github.com/dexon-foundation/dexon/core/vm"
	"github.com/dexon-foundation/dexon/log"
	"github.com/dexon-foundation/dexon/rlp"
)

type Governance struct {
	bc *BlockChain
}

func NewGovernance(bc *BlockChain) *Governance {
	return &Governance{bc: bc}
}

func (g *Governance) getHeadHelper() *vm.GovernanceStateHelper {
	headState, err := g.bc.State()
	if err != nil {
		log.Error("get head state fail", "err", err)
		panic(err)
	}
	return &vm.GovernanceStateHelper{headState}
}

func (g *Governance) getConfigHelper(round uint64) *vm.GovernanceStateHelper {
	if round < dexCore.ConfigRoundShift {
		round = 0
	} else {
		round -= dexCore.ConfigRoundShift
	}
	return g.getHelperAtRound(round)
}

func (g *Governance) getHelperAtRound(round uint64) *vm.GovernanceStateHelper {
	headHelper := g.getHeadHelper()
	height := headHelper.RoundHeight(big.NewInt(int64(round))).Uint64()
	header := g.bc.GetHeaderByNumber(height)
	s, err := g.bc.StateAt(header.Root)
	if err != nil {
		log.Error("get state fail", "err", err)
		panic(err)
	}
	return &vm.GovernanceStateHelper{s}
}

func (g *Governance) Configuration(round uint64) *coreTypes.Config {
	helper := g.getConfigHelper(round)
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

func (g *Governance) DKGComplaints(round uint64) []*dkgTypes.Complaint {
	headHelper := g.getHeadHelper()
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
	headHelper := g.getHeadHelper()
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
	headHelper := g.getHeadHelper()
	configHelper := g.getConfigHelper(round)
	threshold := 2*configHelper.DKGSetSize().Uint64()/3 + 1
	count := headHelper.DKGFinalizedsCount(big.NewInt(int64(round))).Uint64()
	return count >= threshold
}
