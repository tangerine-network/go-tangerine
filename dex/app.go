// Copyright 2018 The dexon-consensus-core Authors
// This file is part of the dexon-consensus-core library.
//
// The dexon-consensus-core library is free software: you can redistribute it
// and/or modify it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The dexon-consensus-core library is distributed in the hope that it will be
// useful, but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the dexon-consensus-core library. If not, see
// <http://www.gnu.org/licenses/>.

package dex

import (
	"time"

	"github.com/dexon-foundation/dexon-consensus-core/common"
	"github.com/dexon-foundation/dexon-consensus-core/core/types"
)

// DexconApp implementes the DEXON consensus core application interface.
type DexconApp struct {
}

// PreparePayload is called when consensus core is preparing a block.
func (d *DexconApp) PreparePayloads(position types.Position) [][]byte {
}

// VerifyPayloads verifies if the payloads are valid.
func (d *DexconApp) VerifyPayloads(payloads [][]byte) bool {
}

// BlockConfirmed is called when a block is confirmed and added to lattice.
func (d *DexconApp) BlockConfirmed(block *types.Block) {
}

// StronglyAcked is called when a block is strongly acked.
func (d *DexconApp) StronglyAcked(blockHash common.Hash) {
}

// TotalOrderingDeliver is called when the total ordering algorithm deliver
// a set of block.
func (d *DexconApp) TotalOrderingDeliver(blockHashes common.Hashes, early bool) {
}

// DeliverBlock is called when a block is add to the compaction chain.
func (d *DexconApp) DeliverBlock(blockHash common.Hash, timestamp time.Time) {
}

// NotaryAckDeliver is called when a notary ack is created.
func (d *DexconApp) NotaryAckDeliver(notaryAck *types.NotaryAck) {
}
