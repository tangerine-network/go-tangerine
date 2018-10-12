package dex

import (
	coreCommon "github.com/dexon-foundation/dexon-consensus-core/common"
	"github.com/dexon-foundation/dexon-consensus-core/core/crypto"
	"github.com/dexon-foundation/dexon-consensus-core/core/types"
	"github.com/dexon-foundation/dexon/core/vm"
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

// Configuration return the total ordering K constant.
func (d *DexconGovernance) Configuration(round uint64) *types.Config {
	state, _, err := d.b.StateAndHeaderByNumber(ctx, blockNr)
	if state == nil || err != nil {
		return nil, err
	}

	s := vm.GovernanceStateHelper{state}

	return &types.Config{}
}

// CRS returns the CRS for a given round.
func (d *DexconGovernance) CRS(round uint64) coreCommon.Hash {
	return coreCommon.Hash{}
}

// ProposeCRS send proposals of a new CRS
func (d *DexconGovernance) ProposeCRS(signedCRS []byte) {
}

// NodeSet returns the current notary set.
func (d *DexconGovernance) NodeSet(round uint64) []crypto.PublicKey {
	return nil
}

// AddDKGComplaint adds a DKGComplaint.
func (d *DexconGovernance) AddDKGComplaint(round uint64, complaint *types.DKGComplaint) {
}

// DKGComplaints gets all the DKGComplaints of round.
func (d *DexconGovernance) DKGComplaints(round uint64) []*types.DKGComplaint {
	return nil
}

// AddDKGMasterPublicKey adds a DKGMasterPublicKey.
func (d *DexconGovernance) AddDKGMasterPublicKey(round uint64, masterPublicKey *types.DKGMasterPublicKey) {
}

// DKGMasterPublicKeys gets all the DKGMasterPublicKey of round.
func (d *DexconGovernance) DKGMasterPublicKeys(round uint64) []*types.DKGMasterPublicKey {
	return nil
}

// AddDKGFinalize adds a DKG finalize message.
func (d *DexconGovernance) AddDKGFinalize(round uint64, final *types.DKGFinalize) {
}

// IsDKGFinal checks if DKG is final.
func (d *DexconGovernance) IsDKGFinal(round uint64) bool {
	return false
}
