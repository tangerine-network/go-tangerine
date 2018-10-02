package dex

import (
	coreCommon "github.com/dexon-foundation/dexon-consensus-core/common"
	"github.com/dexon-foundation/dexon-consensus-core/core/crypto"
	"github.com/dexon-foundation/dexon-consensus-core/core/types"
)

type DexconGovernance struct {
}

// NewDexconGovernance retruns a governance implementation of the DEXON
// consensus governance interface.
func NewDexconGovernance() *DexconGovernance {
	return &DexconGovernance{}
}

// Configuration return the total ordering K constant.
func (d *DexconGovernance) Configuration(round uint64) *types.Config {
	return &types.Config{}
}

// CRS returns the CRS for a given round.
func (d *DexconGovernance) CRS(round uint64) coreCommon.Hash {
	return coreCommon.Hash{}
}

// ProposeCRS send proposals of a new CRS
func (d *DexconGovernance) ProposeCRS(round uint64, signedCRS []byte) {
}

// NodeSet returns the current notary set.
func (d *DexconGovernance) NodeSet(round uint64) []crypto.PublicKey {
	return nil
}

// AddDKGComplaint adds a DKGComplaint.
func (d *DexconGovernance) AddDKGComplaint(complaint *types.DKGComplaint) {
}

// DKGComplaints gets all the DKGComplaints of round.
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
