// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package dex

import (
	"github.com/dexon-foundation/dexon/metrics"
	"github.com/dexon-foundation/dexon/p2p"
)

var (
	propTxnInPacketsMeter                  = metrics.NewRegisteredMeter("dex/prop/txns/in/packets", nil)
	propTxnInTrafficMeter                  = metrics.NewRegisteredMeter("dex/prop/txns/in/traffic", nil)
	propTxnOutPacketsMeter                 = metrics.NewRegisteredMeter("dex/prop/txns/out/packets", nil)
	propTxnOutTrafficMeter                 = metrics.NewRegisteredMeter("dex/prop/txns/out/traffic", nil)
	propHashInPacketsMeter                 = metrics.NewRegisteredMeter("dex/prop/hashes/in/packets", nil)
	propHashInTrafficMeter                 = metrics.NewRegisteredMeter("dex/prop/hashes/in/traffic", nil)
	propHashOutPacketsMeter                = metrics.NewRegisteredMeter("dex/prop/hashes/out/packets", nil)
	propHashOutTrafficMeter                = metrics.NewRegisteredMeter("dex/prop/hashes/out/traffic", nil)
	propBlockInPacketsMeter                = metrics.NewRegisteredMeter("dex/prop/blocks/in/packets", nil)
	propBlockInTrafficMeter                = metrics.NewRegisteredMeter("dex/prop/blocks/in/traffic", nil)
	propBlockOutPacketsMeter               = metrics.NewRegisteredMeter("dex/prop/blocks/out/packets", nil)
	propBlockOutTrafficMeter               = metrics.NewRegisteredMeter("dex/prop/blocks/out/traffic", nil)
	propLatticeBlockInPacketsMeter         = metrics.NewRegisteredMeter("dex/prop/latticeblocks/in/packets", nil)
	propLatticeBlockInTrafficMeter         = metrics.NewRegisteredMeter("dex/prop/latticeblocks/in/traffic", nil)
	propLatticeBlockOutPacketsMeter        = metrics.NewRegisteredMeter("dex/prop/latticeblocks/out/packets", nil)
	propLatticeBlockOutTrafficMeter        = metrics.NewRegisteredMeter("dex/prop/latticeblocks/out/traffic", nil)
	propVoteInPacketsMeter                 = metrics.NewRegisteredMeter("dex/prop/votes/in/packets", nil)
	propVoteInTrafficMeter                 = metrics.NewRegisteredMeter("dex/prop/votes/in/traffic", nil)
	propVoteOutPacketsMeter                = metrics.NewRegisteredMeter("dex/prop/votes/out/packets", nil)
	propVoteOutTrafficMeter                = metrics.NewRegisteredMeter("dex/prop/votes/out/traffic", nil)
	propDKGPartialSignatureInPacketsMeter  = metrics.NewRegisteredMeter("dex/prop/dkgpartialsignatures/in/packets", nil)
	propDKGPartialSignatureInTrafficMeter  = metrics.NewRegisteredMeter("dex/prop/dkgpartialsignatures/in/traffic", nil)
	propDKGPartialSignatureOutPacketsMeter = metrics.NewRegisteredMeter("dex/prop/dkgpartialsignatures/out/packets", nil)
	propDKGPartialSignatureOutTrafficMeter = metrics.NewRegisteredMeter("dex/prop/dkgpartialsignatures/out/traffic", nil)
	reqHeaderInPacketsMeter                = metrics.NewRegisteredMeter("dex/req/headers/in/packets", nil)
	reqHeaderInTrafficMeter                = metrics.NewRegisteredMeter("dex/req/headers/in/traffic", nil)
	reqHeaderOutPacketsMeter               = metrics.NewRegisteredMeter("dex/req/headers/out/packets", nil)
	reqHeaderOutTrafficMeter               = metrics.NewRegisteredMeter("dex/req/headers/out/traffic", nil)
	reqBodyInPacketsMeter                  = metrics.NewRegisteredMeter("dex/req/bodies/in/packets", nil)
	reqBodyInTrafficMeter                  = metrics.NewRegisteredMeter("dex/req/bodies/in/traffic", nil)
	reqBodyOutPacketsMeter                 = metrics.NewRegisteredMeter("dex/req/bodies/out/packets", nil)
	reqBodyOutTrafficMeter                 = metrics.NewRegisteredMeter("dex/req/bodies/out/traffic", nil)
	reqStateInPacketsMeter                 = metrics.NewRegisteredMeter("dex/req/states/in/packets", nil)
	reqStateInTrafficMeter                 = metrics.NewRegisteredMeter("dex/req/states/in/traffic", nil)
	reqStateOutPacketsMeter                = metrics.NewRegisteredMeter("dex/req/states/out/packets", nil)
	reqStateOutTrafficMeter                = metrics.NewRegisteredMeter("dex/req/states/out/traffic", nil)
	reqReceiptInPacketsMeter               = metrics.NewRegisteredMeter("dex/req/receipts/in/packets", nil)
	reqReceiptInTrafficMeter               = metrics.NewRegisteredMeter("dex/req/receipts/in/traffic", nil)
	reqReceiptOutPacketsMeter              = metrics.NewRegisteredMeter("dex/req/receipts/out/packets", nil)
	reqReceiptOutTrafficMeter              = metrics.NewRegisteredMeter("dex/req/receipts/out/traffic", nil)
	reqLatticeBlockInPacketsMeter          = metrics.NewRegisteredMeter("dex/req/latticeblocks/in/packets", nil)
	reqLatticeBlockInTrafficMeter          = metrics.NewRegisteredMeter("dex/req/latticeblocks/in/traffic", nil)
	reqLatticeBlockOutPacketsMeter         = metrics.NewRegisteredMeter("dex/req/latticeblocks/out/packets", nil)
	reqLatticeBlockOutTrafficMeter         = metrics.NewRegisteredMeter("dex/req/latticeblocks/out/traffic", nil)
	reqVoteInPacketsMeter                  = metrics.NewRegisteredMeter("dex/req/votes/in/packets", nil)
	reqVoteInTrafficMeter                  = metrics.NewRegisteredMeter("dex/req/votes/in/traffic", nil)
	reqVoteOutPacketsMeter                 = metrics.NewRegisteredMeter("dex/req/votes/out/packets", nil)
	reqVoteOutTrafficMeter                 = metrics.NewRegisteredMeter("dex/req/votes/out/traffic", nil)
	miscInPacketsMeter                     = metrics.NewRegisteredMeter("dex/misc/in/packets", nil)
	miscInTrafficMeter                     = metrics.NewRegisteredMeter("dex/misc/in/traffic", nil)
	miscOutPacketsMeter                    = metrics.NewRegisteredMeter("dex/misc/out/packets", nil)
	miscOutTrafficMeter                    = metrics.NewRegisteredMeter("dex/misc/out/traffic", nil)
)

// meteredMsgReadWriter is a wrapper around a p2p.MsgReadWriter, capable of
// accumulating the above defined metrics based on the data stream contents.
type meteredMsgReadWriter struct {
	p2p.MsgReadWriter     // Wrapped message stream to meter
	version           int // Protocol version to select correct meters
}

// newMeteredMsgWriter wraps a p2p MsgReadWriter with metering support. If the
// metrics system is disabled, this function returns the original object.
func newMeteredMsgWriter(rw p2p.MsgReadWriter) p2p.MsgReadWriter {
	if !metrics.Enabled {
		return rw
	}
	return &meteredMsgReadWriter{MsgReadWriter: rw}
}

// Init sets the protocol version used by the stream to know which meters to
// increment in case of overlapping message ids between protocol versions.
func (rw *meteredMsgReadWriter) Init(version int) {
	rw.version = version
}

func (rw *meteredMsgReadWriter) ReadMsg() (p2p.Msg, error) {
	// Read the message and short circuit in case of an error
	msg, err := rw.MsgReadWriter.ReadMsg()
	if err != nil {
		return msg, err
	}
	// Account for the data traffic
	packets, traffic := miscInPacketsMeter, miscInTrafficMeter
	switch {
	case msg.Code == BlockHeadersMsg:
		packets, traffic = reqHeaderInPacketsMeter, reqHeaderInTrafficMeter
	case msg.Code == BlockBodiesMsg:
		packets, traffic = reqBodyInPacketsMeter, reqBodyInTrafficMeter

	case msg.Code == NodeDataMsg:
		packets, traffic = reqStateInPacketsMeter, reqStateInTrafficMeter
	case msg.Code == ReceiptsMsg:
		packets, traffic = reqReceiptInPacketsMeter, reqReceiptInTrafficMeter

	case msg.Code == NewBlockHashesMsg:
		packets, traffic = propHashInPacketsMeter, propHashInTrafficMeter
	case msg.Code == NewBlockMsg:
		packets, traffic = propBlockInPacketsMeter, propBlockInTrafficMeter
	case msg.Code == TxMsg:
		packets, traffic = propTxnInPacketsMeter, propTxnInTrafficMeter

	case msg.Code == LatticeBlockMsg:
		packets = propLatticeBlockInPacketsMeter
		traffic = propLatticeBlockInTrafficMeter
	case msg.Code == VoteMsg:
		packets, traffic = propVoteInPacketsMeter, propVoteInTrafficMeter

	case msg.Code == PullBlocksMsg:
		packets = reqLatticeBlockInPacketsMeter
		traffic = reqLatticeBlockInTrafficMeter
	case msg.Code == PullVotesMsg:
		packets, traffic = reqVoteInPacketsMeter, reqVoteInTrafficMeter

	case msg.Code == DKGPartialSignatureMsg:
		packets = propDKGPartialSignatureInPacketsMeter
		traffic = propDKGPartialSignatureInTrafficMeter
	}
	packets.Mark(1)
	traffic.Mark(int64(msg.Size))

	return msg, err
}

func (rw *meteredMsgReadWriter) WriteMsg(msg p2p.Msg) error {
	// Account for the data traffic
	packets, traffic := miscOutPacketsMeter, miscOutTrafficMeter
	switch {
	case msg.Code == BlockHeadersMsg:
		packets, traffic = reqHeaderOutPacketsMeter, reqHeaderOutTrafficMeter
	case msg.Code == BlockBodiesMsg:
		packets, traffic = reqBodyOutPacketsMeter, reqBodyOutTrafficMeter

	case msg.Code == NodeDataMsg:
		packets, traffic = reqStateOutPacketsMeter, reqStateOutTrafficMeter
	case msg.Code == ReceiptsMsg:
		packets, traffic = reqReceiptOutPacketsMeter, reqReceiptOutTrafficMeter

	case msg.Code == NewBlockHashesMsg:
		packets, traffic = propHashOutPacketsMeter, propHashOutTrafficMeter
	case msg.Code == NewBlockMsg:
		packets, traffic = propBlockOutPacketsMeter, propBlockOutTrafficMeter
	case msg.Code == TxMsg:
		packets, traffic = propTxnOutPacketsMeter, propTxnOutTrafficMeter

	case msg.Code == LatticeBlockMsg:
		packets = propLatticeBlockOutPacketsMeter
		traffic = propLatticeBlockOutTrafficMeter
	case msg.Code == VoteMsg:
		packets, traffic = propVoteOutPacketsMeter, propVoteOutTrafficMeter

	case msg.Code == PullBlocksMsg:
		packets = reqLatticeBlockOutPacketsMeter
		traffic = reqLatticeBlockOutTrafficMeter
	case msg.Code == PullVotesMsg:
		packets, traffic = reqVoteOutPacketsMeter, reqVoteOutTrafficMeter

	case msg.Code == DKGPartialSignatureMsg:
		packets = propDKGPartialSignatureOutPacketsMeter
		traffic = propDKGPartialSignatureOutTrafficMeter
	}
	packets.Mark(1)
	traffic.Mark(int64(msg.Size))

	// Send the packet to the p2p layer
	return rw.MsgReadWriter.WriteMsg(msg)
}
