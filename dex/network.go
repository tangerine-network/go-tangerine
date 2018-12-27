// Copyright 2018 The dexon-consensus Authors
// This file is part of the dexon-consensus library.
//
// The dexon-consensus library is free software: you can redistribute it
// and/or modify it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The dexon-consensus library is distributed in the hope that it will be
// useful, but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the dexon-consensus library. If not, see
// <http://www.gnu.org/licenses/>.

package dex

import (
	coreCommon "github.com/dexon-foundation/dexon-consensus/common"
	"github.com/dexon-foundation/dexon-consensus/core/crypto"
	"github.com/dexon-foundation/dexon-consensus/core/types"
	dkgTypes "github.com/dexon-foundation/dexon-consensus/core/types/dkg"
)

type DexconNetwork struct {
	pm *ProtocolManager
}

func NewDexconNetwork(pm *ProtocolManager) *DexconNetwork {
	return &DexconNetwork{pm: pm}
}

// PullBlocks tries to pull blocks from the DEXON network.
func (n *DexconNetwork) PullBlocks(hashes coreCommon.Hashes) {
	if len(hashes) == 0 {
		return
	}
	n.pm.BroadcastPullBlocks(hashes)
}

// PullVotes tries to pull votes from the DEXON network.
func (n *DexconNetwork) PullVotes(pos types.Position) {
	n.pm.BroadcastPullVotes(pos)
}

// PullRandomness tries to pull randomness result from the DEXON network.
func (n *DexconNetwork) PullRandomness(hashes coreCommon.Hashes) {
	if len(hashes) == 0 {
		return
	}
	n.pm.BroadcastPullRandomness(hashes)
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
