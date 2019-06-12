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
	"testing"

	"github.com/tangerine-network/go-tangerine/common"
	"github.com/tangerine-network/go-tangerine/crypto"
	"github.com/tangerine-network/go-tangerine/params"
)

func TestRecoveryVoteTxGeneration(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate keypair: %v", err)
	}

	r := NewRecovery(&params.RecoveryConfig{
		Contract:     common.HexToAddress("f675c0e9bf4b949f50dcec5b224a70f0361d4680"),
		Timeout:      30,
		Confirmation: 1,
	}, "https://rinkeby.infura.io", nil, key)
	_, err = r.genVoteForSkipBlockTx(0)
	if err != nil {
		t.Fatalf("failed to generate voteForSkipBlock tx: %v", err)
	}
}
