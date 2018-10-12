package dex

import (
	"context"
	"math/big"
	"time"

	coreCommon "github.com/dexon-foundation/dexon-consensus-core/common"
	coreCrypto "github.com/dexon-foundation/dexon-consensus-core/core/crypto"
	"github.com/dexon-foundation/dexon-consensus-core/core/crypto/ecdsa"
	coreTypes "github.com/dexon-foundation/dexon-consensus-core/core/types"
	"github.com/dexon-foundation/dexon/core/vm"
	"github.com/dexon-foundation/dexon/rlp"
	"github.com/dexon-foundation/dexon/rpc"
)

type DexconGovernance struct {
	b *DexAPIBackend
}

// NewDexconGovernance retruns a governance implementation of the DEXON
// consensus governance interface.
func NewDexconGovernance(backend *DexAPIBackend) *DexconGovernance {
	return &DexconGovernance{
		b: backend,
	}
}

func (d *DexconGovernance) getRoundHeight(ctx context.Context, round uint64) (uint64, error) {
	state, _, err := d.b.StateAndHeaderByNumber(ctx, rpc.LatestBlockNumber)
	if state == nil || err != nil {
		return 0, err
	}
	s := vm.GovernanceStateHelper{state}
	return s.RoundHeight(big.NewInt(int64(round))).Uint64(), nil
}

func (d *DexconGovernance) getGovState() *vm.GovernanceStateHelper {
	ctx := context.Background()
	state, _, err := d.b.StateAndHeaderByNumber(ctx, rpc.LatestBlockNumber)
	if state == nil || err != nil {
		return nil
	}

	return &vm.GovernanceStateHelper{state}
}

func (d *DexconGovernance) getGovStateAtRound(round uint64) *vm.GovernanceStateHelper {
	ctx := context.Background()
	blockHeight, err := d.getRoundHeight(ctx, round)
	if err != nil {
		return nil
	}

	state, _, err := d.b.StateAndHeaderByNumber(ctx, rpc.BlockNumber(blockHeight))
	if state == nil || err != nil {
		return nil
	}

	return &vm.GovernanceStateHelper{state}
}

// Configuration return the total ordering K constant.
func (d *DexconGovernance) Configuration(round uint64) *coreTypes.Config {
	s := d.getGovStateAtRound(round)
	c := s.Configuration()

	return &coreTypes.Config{
		NumChains:        c.NumChains,
		LambdaBA:         time.Duration(c.LambdaBA) * time.Millisecond,
		LambdaDKG:        time.Duration(c.LambdaDKG) * time.Millisecond,
		K:                c.K,
		PhiRatio:         c.PhiRatio,
		NotarySetSize:    c.NotarySetSize,
		DKGSetSize:       c.DKGSetSize,
		RoundInterval:    time.Duration(c.RoundInterval) * time.Millisecond,
		MinBlockInterval: time.Duration(c.MinBlockInterval) * time.Millisecond,
		MaxBlockInterval: time.Duration(c.MaxBlockInterval) * time.Millisecond,
	}
}

// CRS returns the CRS for a given round.
func (d *DexconGovernance) CRS(round uint64) coreCommon.Hash {
	s := d.getGovStateAtRound(round)
	return coreCommon.Hash(s.CRS(big.NewInt(int64(round))))
}

// ProposeCRS send proposals of a new CRS
func (d *DexconGovernance) ProposeCRS(signedCRS []byte) {
}

// NodeSet returns the current notary set.
func (d *DexconGovernance) NodeSet(round uint64) []coreCrypto.PublicKey {
	s := d.getGovStateAtRound(round)
	var pks []coreCrypto.PublicKey

	for _, n := range s.Nodes() {
		pks = append(pks, ecdsa.NewPublicKeyFromByteSlice(n.PublicKey))
	}
	return pks
}

// AddDKGComplaint adds a DKGComplaint.
func (d *DexconGovernance) AddDKGComplaint(round uint64, complaint *coreTypes.DKGComplaint) {
}

// DKGComplaints gets all the DKGComplaints of round.
func (d *DexconGovernance) DKGComplaints(round uint64) []*coreTypes.DKGComplaint {
	s := d.getGovState()
	var dkgComplaints []*coreTypes.DKGComplaint
	for _, pk := range s.DKGMasterPublicKeys(big.NewInt(int64(round))) {
		x := new(coreTypes.DKGComplaint)
		if err := rlp.DecodeBytes(pk, x); err != nil {
			panic(err)
		}
		dkgComplaints = append(dkgComplaints, x)
	}
	return dkgComplaints
}

// AddDKGMasterPublicKey adds a DKGMasterPublicKey.
func (d *DexconGovernance) AddDKGMasterPublicKey(round uint64, masterPublicKey *coreTypes.DKGMasterPublicKey) {
}

// DKGMasterPublicKeys gets all the DKGMasterPublicKey of round.
func (d *DexconGovernance) DKGMasterPublicKeys(round uint64) []*coreTypes.DKGMasterPublicKey {
	s := d.getGovState()
	var dkgMasterPKs []*coreTypes.DKGMasterPublicKey
	for _, pk := range s.DKGMasterPublicKeys(big.NewInt(int64(round))) {
		x := new(coreTypes.DKGMasterPublicKey)
		if err := rlp.DecodeBytes(pk, x); err != nil {
			panic(err)
		}
		dkgMasterPKs = append(dkgMasterPKs, x)
	}
	return dkgMasterPKs
}

// AddDKGFinalize adds a DKG finalize message.
func (d *DexconGovernance) AddDKGFinalize(round uint64, final *coreTypes.DKGFinalize) {
}

// IsDKGFinal checks if DKG is final.
func (d *DexconGovernance) IsDKGFinal(round uint64) bool {
	s := d.getGovStateAtRound(round)
	threshold := 2*s.DKGSetSize().Uint64()/3 + 1
	count := s.DKGFinalizedsCount(big.NewInt(int64(round))).Uint64()
	return count >= threshold
}
