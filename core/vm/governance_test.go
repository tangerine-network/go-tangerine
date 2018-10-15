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

package vm

import (
	"bytes"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/core/state"
	"github.com/dexon-foundation/dexon/ethdb"
	"github.com/stretchr/testify/suite"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func randomBytes(minLength, maxLength int32) []byte {
	length := rand.Int31()%(maxLength-minLength) + minLength
	b := make([]byte, length)
	for i := range b {
		b[i] = byte(65 + rand.Int31()%60)
	}
	return b
}

type GovernanceStateHelperTestSuite struct {
	suite.Suite

	s *GovernanceStateHelper
}

func (g *GovernanceStateHelperTestSuite) SetupTest() {
	db := state.NewDatabase(ethdb.NewMemDatabase())
	statedb, err := state.New(common.Hash{}, db)
	if err != nil {
		panic(err)
	}
	g.s = &GovernanceStateHelper{statedb}
}

func (g *GovernanceStateHelperTestSuite) TestReadWriteBytes() {
	for i := 0; i < 100; i++ {
		// Short bytes.
		loc := big.NewInt(rand.Int63())
		data := randomBytes(3, 32)
		g.s.writeBytes(loc, data)
		read := g.s.readBytes(loc)
		g.Require().Equal(0, bytes.Compare(data, read))

		// long bytes.
		loc = big.NewInt(rand.Int63())
		data = randomBytes(33, 2560)
		g.s.writeBytes(loc, data)
		read = g.s.readBytes(loc)
		g.Require().Equal(0, bytes.Compare(data, read))
	}
}

func TestGovernanceStateHelper(t *testing.T) {
	suite.Run(t, new(GovernanceStateHelperTestSuite))
}
