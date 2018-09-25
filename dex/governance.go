package dex

import (
	"github.com/dexon-foundation/dexon-consensus-core/core/types"
)

type DexconGovernance struct {
}

// NewDexconGovernance retruns a governance implementation of the DEXON
// consensus governance interface.
func NewDexconGovernance() *DexconGovernance {
	return &DexconGovernance{}
}

// GetValidatorSet returns the current notary set.
func (d *DexconGovernance) GetNotarySet(
	blockHeight uint64) map[types.NodeID]struct{} {
	return make(map[types.NodeID]struct{})
}

// GetTotalOrderingK return the total ordering K constant.
func (d *DexconGovernance) GetConfiguration(blockHeight uint64) *types.Config {
	return &types.Config{}
}

// AddDKGComplaint adds a DKGComplaint.
func (d *DexconGovernance) AddDKGComplaint(complaint *types.DKGComplaint) {
}

// GetDKGComplaints gets all the DKGComplaints of round.
func (d *DexconGovernance) DKGComplaints(round uint64) []*types.DKGComplaint {
	return nil
}

// AddDKGMasterPublicKey adds a DKGMasterPublicKey.
func (d *DexconGovernance) AddDKGMasterPublicKey(masterPublicKey *types.DKGMasterPublicKey) {
}

// DKGMasterPublicKeys gets all the DKGMasterPublicKey of round.
func (d *DexconGovernance) DKGMasterPublicKeys(round uint64) []*types.DKGMasterPublicKey {
	return nil
}
