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
	"github.com/dexon-foundation/dexon-consensus-core/core/types"

	"github.com/dexon-foundation/dexon/core"
)

// DexconApp implementes the DEXON consensus core application interface.
type DexconApp struct {
	txPool *core.TxPool

	witnessResultChan chan types.WitnessResult
}

func NewDexconApp(txPool *core.TxPool) *DexconApp {
	return &DexconApp{
		txPool:            txPool,
		witnessResultChan: make(chan types.WitnessResult),
	}
}

// PreparePayload is called when consensus core is preparing a block.
func (d *DexconApp) PreparePayload(position types.Position) []byte {
	return nil
}

// VerifyPayload verifies if the payloads are valid.
func (d *DexconApp) VerifyPayload(payload []byte) bool {
	return true
}

// BlockDelivered is called when a block is add to the compaction chain.
func (d *DexconApp) BlockDelivered(block types.Block) {
}

// BlockProcessedChan returns a channel to receive the block hashes that have
// finished processing by the application.
func (d *DexconApp) BlockProcessedChan() <-chan types.WitnessResult {
	return d.witnessResultChan
}

// WitnessAckDeliver is called when a notary ack is created.
func (d *DexconApp) WitnessAckDelivered(notaryAck *types.WitnessAck) {
}
