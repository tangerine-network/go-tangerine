package dex

import (
	"github.com/dexon-foundation/dexon-consensus-core/core/crypto"
	"github.com/dexon-foundation/dexon-consensus-core/core/types"
	dkgTypes "github.com/dexon-foundation/dexon-consensus-core/core/types/dkg"
)

type DexconNetwork struct {
	pm *ProtocolManager
}

func NewDexconNetwork(pm *ProtocolManager) *DexconNetwork {
	return &DexconNetwork{pm: pm}
}

// BroadcastVote broadcasts vote to all nodes in DEXON network.
func (n *DexconNetwork) BroadcastVote(vote *types.Vote) {
	n.pm.BroadcastVote(vote)
}

// BroadcastBlock broadcasts block to all nodes in DEXON network.
func (n *DexconNetwork) BroadcastBlock(block *types.Block) {
	n.pm.BroadcastLatticeBlock(block)
}

// SendDKGPrivateShare sends PrivateShare to a DKG participant.
func (n *DexconNetwork) SendDKGPrivateShare(
	pub crypto.PublicKey, prvShare *dkgTypes.PrivateShare) {
	n.pm.SendDKGPrivateShare(pub, prvShare)
}

// BroadcastDKGPrivateShare broadcasts PrivateShare to all DKG participants.
func (n *DexconNetwork) BroadcastDKGPrivateShare(
	prvShare *dkgTypes.PrivateShare) {
	n.pm.BroadcastDKGPrivateShare(prvShare)
}

// BroadcastDKGPartialSignature broadcasts partialSignature to all
// DKG participants.
func (n *DexconNetwork) BroadcastDKGPartialSignature(
	psig *dkgTypes.PartialSignature) {
	n.pm.BroadcastDKGPartialSignature(psig)
}

// BroadcastAgreementResult broadcasts rand request to DKG set.
func (n *DexconNetwork) BroadcastAgreementResult(randRequest *types.AgreementResult) {
	n.pm.BroadcastAgreementResult(randRequest)
}

// BroadcastRandomnessResult broadcasts rand request to Notary set.
func (n *DexconNetwork) BroadcastRandomnessResult(randResult *types.BlockRandomnessResult) {
	n.pm.BroadcastRandomnessResult(randResult)
}

// ReceiveChan returns a channel to receive messages from DEXON network.
func (n *DexconNetwork) ReceiveChan() <-chan interface{} {
	return n.pm.ReceiveChan()
}
