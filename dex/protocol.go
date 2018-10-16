// Copyright 2014 The go-ethereum Authors
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
	"crypto/ecdsa"
	"fmt"
	"io"
	"math/big"
	"time"

	coreCommon "github.com/dexon-foundation/dexon-consensus-core/common"
	"github.com/dexon-foundation/dexon-consensus-core/core/crypto"
	coreTypes "github.com/dexon-foundation/dexon-consensus-core/core/types"
	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/core"
	"github.com/dexon-foundation/dexon/core/types"
	"github.com/dexon-foundation/dexon/event"
	"github.com/dexon-foundation/dexon/p2p/enode"
	"github.com/dexon-foundation/dexon/rlp"
	"golang.org/x/crypto/sha3"
)

// Constants to match up protocol versions and messages
const (
	dex64 = 64
)

// ProtocolName is the official short name of the protocol used during capability negotiation.
var ProtocolName = "dex"

// ProtocolVersions are the upported versions of the eth protocol (first is primary).
var ProtocolVersions = []uint{dex64}

// ProtocolLengths are the number of implemented message corresponding to different protocol versions.
var ProtocolLengths = []uint64{38}

const ProtocolMaxMsgSize = 10 * 1024 * 1024 // Maximum cap on the size of a protocol message

// eth protocol message codes
const (
	// Protocol messages belonging to eth/62
	StatusMsg          = 0x00
	NewBlockHashesMsg  = 0x01
	TxMsg              = 0x02
	GetBlockHeadersMsg = 0x03
	BlockHeadersMsg    = 0x04
	GetBlockBodiesMsg  = 0x05
	BlockBodiesMsg     = 0x06
	NewBlockMsg        = 0x07

	// Protocol messages belonging to eth/63
	GetNodeDataMsg = 0x0d
	NodeDataMsg    = 0x0e
	GetReceiptsMsg = 0x0f
	ReceiptsMsg    = 0x10

	// Protocol messages belonging to dex/64
	MetaMsg = 0x11

	LatticeBlockMsg        = 0x20
	VoteMsg                = 0x21
	AgreementMsg           = 0x22
	RandomnessMsg          = 0x23
	DKGPrivateShareMsg     = 0x24
	DKGPartialSignatureMsg = 0x25
)

type errCode int

const (
	ErrMsgTooLarge = iota
	ErrDecode
	ErrInvalidMsgCode
	ErrProtocolVersionMismatch
	ErrNetworkIdMismatch
	ErrGenesisBlockMismatch
	ErrNoStatusMsg
	ErrExtraStatusMsg
	ErrSuspendedPeer
)

func (e errCode) String() string {
	return errorToString[int(e)]
}

// XXX change once legacy code is out
var errorToString = map[int]string{
	ErrMsgTooLarge:             "Message too long",
	ErrDecode:                  "Invalid message",
	ErrInvalidMsgCode:          "Invalid message code",
	ErrProtocolVersionMismatch: "Protocol version mismatch",
	ErrNetworkIdMismatch:       "NetworkId mismatch",
	ErrGenesisBlockMismatch:    "Genesis block mismatch",
	ErrNoStatusMsg:             "No status message",
	ErrExtraStatusMsg:          "Extra status message",
	ErrSuspendedPeer:           "Suspended peer",
}

type txPool interface {
	// AddRemotes should add the given transactions to the pool.
	AddRemotes([]*types.Transaction) []error

	// Pending should return pending transactions.
	// The slice should be modifiable by the caller.
	Pending() (map[common.Address]types.Transactions, error)

	// SubscribeNewTxsEvent should return an event subscription of
	// NewTxsEvent and send events to the given channel.
	SubscribeNewTxsEvent(chan<- core.NewTxsEvent) event.Subscription
}

type governance interface {
	GetNumChains(uint64) uint32

	LenCRS() uint64

	NotarySet(uint64, uint32) (map[string]struct{}, error)

	DKGSet(uint64) (map[string]struct{}, error)
}

type p2pServer interface {
	Self() *enode.Node

	GetPrivateKey() *ecdsa.PrivateKey

	AddDirectPeer(*enode.Node)

	RemoveDirectPeer(*enode.Node)

	AddGroup(string, []*enode.Node, uint64)

	RemoveGroup(string)
}

// statusData is the network packet for the status message.
type statusData struct {
	ProtocolVersion uint32
	NetworkId       uint64
	TD              *big.Int
	CurrentBlock    common.Hash
	GenesisBlock    common.Hash
}

// newBlockHashesData is the network packet for the block announcements.
type newBlockHashesData []struct {
	Hash   common.Hash // Hash of one particular block being announced
	Number uint64      // Number of one particular block being announced
}

// getBlockHeadersData represents a block header query.
type getBlockHeadersData struct {
	Origin  hashOrNumber // Block from which to retrieve headers
	Amount  uint64       // Maximum number of headers to retrieve
	Skip    uint64       // Blocks to skip between consecutive headers
	Reverse bool         // Query direction (false = rising towards latest, true = falling towards genesis)
}

// hashOrNumber is a combined field for specifying an origin block.
type hashOrNumber struct {
	Hash   common.Hash // Block hash from which to retrieve headers (excludes Number)
	Number uint64      // Block hash from which to retrieve headers (excludes Hash)
}

// EncodeRLP is a specialized encoder for hashOrNumber to encode only one of the
// two contained union fields.
func (hn *hashOrNumber) EncodeRLP(w io.Writer) error {
	if hn.Hash == (common.Hash{}) {
		return rlp.Encode(w, hn.Number)
	}
	if hn.Number != 0 {
		return fmt.Errorf("both origin hash (%x) and number (%d) provided", hn.Hash, hn.Number)
	}
	return rlp.Encode(w, hn.Hash)
}

// DecodeRLP is a specialized decoder for hashOrNumber to decode the contents
// into either a block hash or a block number.
func (hn *hashOrNumber) DecodeRLP(s *rlp.Stream) error {
	_, size, _ := s.Kind()
	origin, err := s.Raw()
	if err == nil {
		switch {
		case size == 32:
			err = rlp.DecodeBytes(origin, &hn.Hash)
		case size <= 8:
			err = rlp.DecodeBytes(origin, &hn.Number)
		default:
			err = fmt.Errorf("invalid input size %d for origin", size)
		}
	}
	return err
}

// newBlockData is the network packet for the block propagation message.
type newBlockData struct {
	Block *types.Block
	TD    *big.Int
}

// blockBody represents the data content of a single block.
type blockBody struct {
	Transactions []*types.Transaction // Transactions contained within a block
	Uncles       []*types.Header      // Uncles contained within a block
}

// blockBodiesData is the network packet for block content distribution.
type blockBodiesData []*blockBody

func rlpHash(x interface{}) (h common.Hash) {
	hw := sha3.NewLegacyKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}

type rlpDKGPrivateShare struct {
	ProposerID   coreTypes.NodeID
	ReceiverID   coreTypes.NodeID
	Round        uint64
	PrivateShare []byte
	Signature    crypto.Signature
}

func toRLPDKGPrivateShare(ps *coreTypes.DKGPrivateShare) *rlpDKGPrivateShare {
	return &rlpDKGPrivateShare{
		ProposerID:   ps.ProposerID,
		ReceiverID:   ps.ReceiverID,
		Round:        ps.Round,
		PrivateShare: ps.PrivateShare.Bytes(),
		Signature:    ps.Signature,
	}
}

func fromRLPDKGPrivateShare(rps *rlpDKGPrivateShare) *coreTypes.DKGPrivateShare {
	ps := &coreTypes.DKGPrivateShare{
		ProposerID: rps.ProposerID,
		ReceiverID: rps.ReceiverID,
		Round:      rps.Round,
		Signature:  rps.Signature,
	}
	ps.PrivateShare.SetBytes(rps.PrivateShare)
	return ps
}

type rlpWitness struct {
	Timestamp uint64
	Height    uint64
	Data      []byte
}

type rlpFinalizeResult struct {
	Randomness []byte
	Timestamp  uint64
	Height     uint64
}

type rlpLatticeBlock struct {
	ProposerID   coreTypes.NodeID        `json:"proposer_id"`
	ParentHash   coreCommon.Hash         `json:"parent_hash"`
	Hash         coreCommon.Hash         `json:"hash"`
	Position     coreTypes.Position      `json:"position"`
	Timestamp    uint64                  `json:"timestamps"`
	Acks         coreCommon.SortedHashes `json:"acks"`
	Payload      []byte                  `json:"payload"`
	Witness      rlpWitness
	Finalization rlpFinalizeResult
	Signature    crypto.Signature `json:"signature"`
	CRSSignature crypto.Signature `json:"crs_signature"`
}

func toRLPLatticeBlock(b *coreTypes.Block) *rlpLatticeBlock {
	return &rlpLatticeBlock{
		ProposerID: b.ProposerID,
		ParentHash: b.ParentHash,
		Hash:       b.Hash,
		Position:   b.Position,
		Timestamp:  toMillisecond(b.Timestamp),
		Acks:       b.Acks,
		Payload:    b.Payload,
		Witness: rlpWitness{
			Timestamp: toMillisecond(b.Witness.Timestamp),
			Height:    b.Witness.Height,
			Data:      b.Witness.Data,
		},
		Finalization: rlpFinalizeResult{
			Randomness: b.Finalization.Randomness,
			Timestamp:  toMillisecond(b.Finalization.Timestamp),
			Height:     b.Finalization.Height,
		},
		Signature:    b.Signature,
		CRSSignature: b.CRSSignature,
	}
}

func fromRLPLatticeBlock(rb *rlpLatticeBlock) *coreTypes.Block {
	return &coreTypes.Block{
		ProposerID: rb.ProposerID,
		ParentHash: rb.ParentHash,
		Hash:       rb.Hash,
		Position:   rb.Position,
		Timestamp:  fromMillisecond(rb.Timestamp),
		Acks:       rb.Acks,
		Payload:    rb.Payload,
		Witness: coreTypes.Witness{
			Timestamp: fromMillisecond(rb.Witness.Timestamp),
			Height:    rb.Witness.Height,
			Data:      rb.Witness.Data,
		},
		Finalization: coreTypes.FinalizationResult{
			Randomness: rb.Finalization.Randomness,
			Timestamp:  fromMillisecond(rb.Finalization.Timestamp),
			Height:     rb.Finalization.Height,
		},
		Signature:    rb.Signature,
		CRSSignature: rb.CRSSignature,
	}
}

func fromMillisecond(s uint64) time.Time {
	sec := int64(s / 1000)
	nsec := int64((s % 1000) * 1000000)
	return time.Unix(sec, nsec)
}

func toMillisecond(t time.Time) uint64 {
	return uint64(t.UnixNano() / 1000000)
}
