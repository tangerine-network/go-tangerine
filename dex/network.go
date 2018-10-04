package dex

import (
	"github.com/dexon-foundation/dexon-consensus-core/core/crypto"
	"github.com/dexon-foundation/dexon-consensus-core/core/types"
)

type DexconNetwork struct {
	receiveChan chan interface{}
}

func NewDexconNetwork() *DexconNetwork {
	return &DexconNetwork{
		receiveChan: make(chan interface{}),
	}
}

// BroadcastVote broadcasts vote to all nodes in DEXON network.
func (n *DexconNetwork) BroadcastVote(vote *types.Vote) {
}

// BroadcastBlock broadcasts block to all nodes in DEXON network.
func (n *DexconNetwork) BroadcastBlock(block *types.Block) {
}

// BroadcastRandomnessRequest broadcasts rand request to DKG set.
func (n *DexconNetwork) BroadcastRandomnessRequest(randRequest *types.AgreementResult) {
}

// BroadcastRandomnessResult broadcasts rand request to Notary set.
func (n *DexconNetwork) BroadcastRandomnessResult(randResult *types.BlockRandomnessResult) {
}

// SendDKGPrivateShare sends PrivateShare to a DKG participant.
func (n *DexconNetwork) SendDKGPrivateShare(
	pub crypto.PublicKey, prvShare *types.DKGPrivateShare) {
}

// BroadcastDKGPrivateShare broadcasts PrivateShare to all DKG participants.
func (n *DexconNetwork) BroadcastDKGPrivateShare(
	prvShare *types.DKGPrivateShare) {
}

// BroadcastDKGPartialSignature broadcasts partialSignature to all
// DKG participants.
func (n *DexconNetwork) BroadcastDKGPartialSignature(
	psig *types.DKGPartialSignature) {
}

// ReceiveChan returns a channel to receive messages from DEXON network.
func (n *DexconNetwork) ReceiveChan() <-chan interface{} {
	return n.receiveChan
}
