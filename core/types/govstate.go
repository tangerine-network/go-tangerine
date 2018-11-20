package types

import (
	"math/big"

	"github.com/dexon-foundation/dexon/common"
)

type GovState struct {
	BlockHash common.Hash
	Number    *big.Int
	Root      common.Hash
	Proof     [][]byte
	Storage   [][2][]byte
}

type HeaderWithGovState struct {
	*Header
	GovState *GovState `rlp:"nil"`
}
