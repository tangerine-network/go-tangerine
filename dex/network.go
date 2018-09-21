package dex

import "github.com/dexon-foundation/dexon-consensus-core/core/types"

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

// BroadcastWitnessAck broadcasts witnessAck to all nodes in DEXON network.
func (n *DexconNetwork) BroadcastWitnessAck(witnessAck *types.WitnessAck) {
}

// SendDKGPrivateShare sends PrivateShare to a DKG participant.
func (n *DexconNetwork) SendDKGPrivateShare(
	recv types.NodeID, prvShare *types.DKGPrivateShare) {
}

// BroadcastDKGPrivateShare broadcasts PrivateShare to all DKG participants.
func (n *DexconNetwork) BroadcastDKGPrivateShare(
	prvShare *types.DKGPrivateShare)  {
}

// ReceiveChan returns a channel to receive messages from DEXON network.
func (n *DexconNetwork) ReceiveChan() <-chan interface{} {
	return n.receiveChan
}
