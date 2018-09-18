package dex

import (
	"github.com/dexon-foundation/dexon-consensus-core/core/types"
)

type DexconGovernance struct {
}

// GetValidatorSet returns the current notary set.
func (d *DexconGovernance) GetNotarySet() map[types.ValidatorID]struct{} {
}

// GetTotalOrderingK return the total ordering K constant.
func (d *DexconGovernance) GetCCP() types.CCP {
}

// AddDKGComplaint adds a DKGComplaint.
func (d *DexconGovernance) AddDKGComplaint(complaint *types.DKGComplaint) {
}

// GetDKGComplaints gets all the DKGComplaints of round.
func (d *DexconGovernance) DKGComplaints(round uint64) []*types.DKGComplaint {
}

// AddDKGMasterPublicKey adds a DKGMasterPublicKey.
func (d *DexconGovernance) AddDKGMasterPublicKey(masterPublicKey *types.DKGMasterPublicKey) {
}

// DKGMasterPublicKeys gets all the DKGMasterPublicKey of round.
func (d *DexconGovernance) DKGMasterPublicKeys(round uint64) []*types.DKGMasterPublicKey {
}
