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

package vm

import (
	"bytes"
	"crypto/ecdsa"
	"math/big"
	"math/rand"
	"sort"
	"testing"
	"time"

	coreCommon "github.com/byzantine-lab/dexon-consensus/common"
	dexCore "github.com/byzantine-lab/dexon-consensus/core"
	coreCrypto "github.com/byzantine-lab/dexon-consensus/core/crypto"
	coreEcdsa "github.com/byzantine-lab/dexon-consensus/core/crypto/ecdsa"
	coreTypes "github.com/byzantine-lab/dexon-consensus/core/types"
	dkgTypes "github.com/byzantine-lab/dexon-consensus/core/types/dkg"
	coreUtils "github.com/byzantine-lab/dexon-consensus/core/utils"

	"github.com/stretchr/testify/suite"
	"github.com/tangerine-network/go-tangerine/common"
	"github.com/tangerine-network/go-tangerine/core/state"
	"github.com/tangerine-network/go-tangerine/crypto"
	"github.com/tangerine-network/go-tangerine/ethdb"
	"github.com/tangerine-network/go-tangerine/params"
	"github.com/tangerine-network/go-tangerine/rlp"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func randomBytes(minLength, maxLength int32) []byte {
	length := minLength
	if maxLength != minLength {
		length += rand.Int31() % (maxLength - minLength)
	}
	b := make([]byte, length)
	for i := range b {
		b[i] = byte(65 + rand.Int31()%60)
	}
	return b
}

func newPrefundAccount(s *state.StateDB) (*ecdsa.PrivateKey, common.Address) {
	privKey, err := crypto.GenerateKey()
	if err != nil {
		panic(err)
	}
	address := crypto.PubkeyToAddress(privKey.PublicKey)

	s.AddBalance(address, new(big.Int).Mul(big.NewInt(1e18), big.NewInt(2e6)))
	return privKey, address
}

type GovernanceStateTestSuite struct {
	suite.Suite

	stateDB *state.StateDB
	s       *GovernanceState
}

func (g *GovernanceStateTestSuite) SetupTest() {
	db := state.NewDatabase(ethdb.NewMemDatabase())
	statedb, err := state.New(common.Hash{}, db)
	if err != nil {
		panic(err)
	}
	g.stateDB = statedb
	g.s = &GovernanceState{statedb}

	config := params.TestnetChainConfig.Dexcon
	g.s.Initialize(config, new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e7)))

	statedb.AddBalance(GovernanceContractAddress, big.NewInt(1))
	statedb.Commit(true)
}

func (g *GovernanceStateTestSuite) TestReadWriteEraseBytes() {
	for i := 0; i < 100; i++ {
		// Short bytes.
		loc := big.NewInt(rand.Int63())
		data := randomBytes(3, 32)
		g.s.writeBytes(loc, data)
		read := g.s.readBytes(loc)
		g.Require().Equal(0, bytes.Compare(data, read))
		g.s.eraseBytes(loc)
		read = g.s.readBytes(loc)
		g.Require().Len(read, 0)

		// long bytes.
		loc = big.NewInt(rand.Int63())
		data = randomBytes(33, 2560)
		g.s.writeBytes(loc, data)
		read = g.s.readBytes(loc)
		g.Require().Equal(0, bytes.Compare(data, read))
		g.s.eraseBytes(loc)
		read = g.s.readBytes(loc)
		g.Require().Len(read, 0)
	}
}

func (g *GovernanceStateTestSuite) TestReadWriteErase1DArray() {
	emptyOffset := 100
	for j := 0; j < 50; j++ {
		idx := big.NewInt(int64(j + emptyOffset))
		data := make([][]byte, 30)
		for key := range data {
			data[key] = randomBytes(3, 32)
			g.s.appendTo1DByteArray(idx, data[key])
		}
		read := g.s.read1DByteArray(idx)
		g.Require().Len(read, len(data))
		for key := range data {
			g.Require().Equal(0, bytes.Compare(data[key], read[key]))
		}
		g.s.erase1DByteArray(idx)
		read = g.s.read1DByteArray(idx)
		g.Require().Len(read, 0)
	}
}

func (g *GovernanceStateTestSuite) TestDisqualify() {
	privKey, addr := newPrefundAccount(g.stateDB)
	pk := crypto.FromECDSAPub(&privKey.PublicKey)

	g.s.Register(addr, pk, "Test", "test@dexon.org", "Taipei", "https://test.com", g.s.MinStake())

	node := g.s.Node(big.NewInt(0))
	g.Require().Equal(uint64(0), node.Fined.Uint64())

	// Disqualify
	g.s.Disqualify(node)
	node = g.s.Node(big.NewInt(0))
	g.Require().Equal(uint64(0xd78ebc5ac6200000), node.Fined.Uint64())

	// Disqualify none exist node should return error.
	privKey2, _ := newPrefundAccount(g.stateDB)
	node.PublicKey = crypto.FromECDSAPub(&privKey2.PublicKey)
	g.Require().Error(g.s.Disqualify(node))
}

func TestGovernanceState(t *testing.T) {
	suite.Run(t, new(GovernanceStateTestSuite))
}

type OracleContractsTestSuite struct {
	suite.Suite

	context Context
	config  *params.DexconConfig
	memDB   *ethdb.MemDatabase
	stateDB *state.StateDB
	s       *GovernanceState
}

func (g *OracleContractsTestSuite) SetupTest() {
	memDB := ethdb.NewMemDatabase()
	stateDB, err := state.New(common.Hash{}, state.NewDatabase(memDB))
	if err != nil {
		panic(err)
	}
	g.memDB = memDB
	g.stateDB = stateDB
	g.s = &GovernanceState{stateDB}

	config := params.TestnetChainConfig.Dexcon
	config.LockupPeriod = 1000
	config.NextHalvingSupply = new(big.Int).Mul(big.NewInt(1e18), big.NewInt(2.5e9))
	config.LastHalvedAmount = new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1.5e9))
	config.MiningVelocity = 0.1875

	g.config = config

	// Give governance contract balance so it will not be deleted because of being an empty state object.
	stateDB.AddBalance(GovernanceContractAddress, big.NewInt(1))

	// Genesis CRS.
	crs := crypto.Keccak256Hash([]byte(config.GenesisCRSText))
	g.s.SetCRS(crs)

	// Round 0 height.
	g.s.PushRoundHeight(big.NewInt(0))

	// Owner.
	g.s.SetOwner(g.config.Owner)

	// Governance configuration.
	g.s.UpdateConfiguration(config)

	g.stateDB.Commit(true)

	g.context = Context{
		CanTransfer: func(db StateDB, addr common.Address, amount *big.Int) bool {
			return db.GetBalance(addr).Cmp(amount) >= 0
		},
		Transfer: func(db StateDB, sender common.Address, recipient common.Address, amount *big.Int) {
			db.SubBalance(sender, amount)
			db.AddBalance(recipient, amount)
		},
		GetRoundHeight: func(round uint64) (uint64, bool) {
			switch round {
			case 0:
				return 0, true
			case 1:
				return 1000, true
			case 2:
				return 2000, true
			}
			return 0, false
		},
		StateAtNumber: func(n uint64) (*state.StateDB, error) {
			return g.stateDB, nil
		},
		BlockNumber: big.NewInt(0),
	}

}

func (g *OracleContractsTestSuite) TearDownTest() {
	OracleContracts[GovernanceContractAddress] = func() OracleContract {
		return &GovernanceContract{
			coreDKGUtils: &defaultCoreDKGUtils{},
		}
	}
}

func (g *OracleContractsTestSuite) call(
	contractAddr common.Address, caller common.Address, input []byte, value *big.Int) ([]byte, error) {

	g.context.Time = big.NewInt(time.Now().UnixNano() / 1000000)

	evm := NewEVM(g.context, g.stateDB, params.TestChainConfig, Config{IsBlockProposer: true})
	ret, _, err := evm.Call(AccountRef(caller), contractAddr, input, 10000000, value)
	return ret, err
}

func (g *OracleContractsTestSuite) TestTransferOwnership() {
	input, err := GovernanceABI.ABI.Pack("transferOwnership", common.Address{})
	g.Require().NoError(err)
	// Call with owner but invalid new owner.
	_, err = g.call(GovernanceContractAddress, g.config.Owner, input, big.NewInt(0))
	g.Require().NotNil(err)

	_, addr := newPrefundAccount(g.stateDB)

	input, err = GovernanceABI.ABI.Pack("transferOwnership", addr)
	g.Require().NoError(err)

	// Call with non-owner.
	_, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NotNil(err)

	// Call with owner.
	_, err = g.call(GovernanceContractAddress, g.config.Owner, input, big.NewInt(0))
	g.Require().NoError(err)
	g.Require().Equal(addr, g.s.Owner())
}

func (g *OracleContractsTestSuite) TestTransferNodeOwnership() {
	privKey, addr := newPrefundAccount(g.stateDB)
	pk := crypto.FromECDSAPub(&privKey.PublicKey)

	amount := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e6))
	input, err := GovernanceABI.ABI.Pack("register", pk, "Test1", "test1@dexon.org", "Taipei", "https://dexon.org")
	g.Require().NoError(err)
	_, err = g.call(GovernanceContractAddress, addr, input, amount)
	g.Require().NoError(err)

	// Call with not valid new owner.
	input, err = GovernanceABI.ABI.Pack("transferNodeOwnership", common.Address{})
	g.Require().NoError(err)
	_, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NotNil(err)

	_, newAddr := newPrefundAccount(g.stateDB)

	input, err = GovernanceABI.ABI.Pack("transferNodeOwnership", newAddr)
	g.Require().NoError(err)

	// Call with non-owner.
	_, noneOwner := newPrefundAccount(g.stateDB)
	_, err = g.call(GovernanceContractAddress, noneOwner, input, big.NewInt(0))
	g.Require().Error(err)

	// Call with owner.
	_, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NoError(err)
	g.Require().Equal(-1, int(g.s.NodesOffsetByAddress(addr).Int64()))
	g.Require().Equal(0, int(g.s.NodesOffsetByNodeKeyAddress(addr).Int64()))
	g.Require().Equal(0, int(g.s.NodesOffsetByAddress(newAddr).Int64()))

	// New node for duplication test.
	privKey2, addr2 := newPrefundAccount(g.stateDB)
	pk2 := crypto.FromECDSAPub(&privKey2.PublicKey)
	input, err = GovernanceABI.ABI.Pack("register", pk2, "Test2", "test1@dexon.org", "Taipei", "https://dexon.org")
	g.Require().NoError(err)
	_, err = g.call(GovernanceContractAddress, addr2, input, amount)
	g.Require().NoError(err)

	// Transfer to duplicate owner address.
	input, err = GovernanceABI.ABI.Pack("transferNodeOwnership", addr2)
	g.Require().NoError(err)
	_, err = g.call(GovernanceContractAddress, newAddr, input, amount)
	g.Require().Error(err)
}

func (g *OracleContractsTestSuite) TestTransferNodeOwnershipByFoundation() {
	privKey, addr := newPrefundAccount(g.stateDB)
	pk := crypto.FromECDSAPub(&privKey.PublicKey)

	amount := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e6))
	input, err := GovernanceABI.ABI.Pack("register", pk, "Test1", "test1@dexon.org", "Taipei", "https://dexon.org")
	g.Require().NoError(err)
	_, err = g.call(GovernanceContractAddress, addr, input, amount)
	g.Require().NoError(err)

	_, newAddr := newPrefundAccount(g.stateDB)

	// Call with not valid new owner.
	input, err = GovernanceABI.ABI.Pack("transferNodeOwnershipByFoundation", common.Address{}, newAddr)
	g.Require().NoError(err)
	_, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NotNil(err)

	input, err = GovernanceABI.ABI.Pack("transferNodeOwnershipByFoundation", addr, newAddr)
	g.Require().NoError(err)

	// Call with gov owner.
	_, noneOwner := newPrefundAccount(g.stateDB)
	_, err = g.call(GovernanceContractAddress, noneOwner, input, big.NewInt(0))
	g.Require().Error(err)

	// Call with gov owner.
	_, err = g.call(GovernanceContractAddress, g.config.Owner, input, big.NewInt(0))
	g.Require().NoError(err)
	g.Require().Equal(-1, int(g.s.NodesOffsetByAddress(addr).Int64()))
	g.Require().Equal(0, int(g.s.NodesOffsetByNodeKeyAddress(addr).Int64()))
	g.Require().Equal(0, int(g.s.NodesOffsetByAddress(newAddr).Int64()))
}

func (g *OracleContractsTestSuite) TestReplaceNodePublicKey() {
	privKey, addr := newPrefundAccount(g.stateDB)
	pk := crypto.FromECDSAPub(&privKey.PublicKey)

	amount := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e6))
	input, err := GovernanceABI.ABI.Pack("register", pk, "Test1", "test1@dexon.org", "Taipei", "https://dexon.org")
	g.Require().NoError(err)
	_, err = g.call(GovernanceContractAddress, addr, input, amount)
	g.Require().NoError(err)

	privKey2, addr2 := newPrefundAccount(g.stateDB)
	pk2 := crypto.FromECDSAPub(&privKey2.PublicKey)

	input, err = GovernanceABI.ABI.Pack("replaceNodePublicKey", pk2)
	g.Require().NoError(err)

	// Call with non-owner.
	_, noneOwner := newPrefundAccount(g.stateDB)
	_, err = g.call(GovernanceContractAddress, noneOwner, input, big.NewInt(0))
	g.Require().Error(err)

	// Call with owner.
	_, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NoError(err)

	g.Require().Equal(-1, int(g.s.NodesOffsetByNodeKeyAddress(addr).Int64()))
	g.Require().Equal(0, int(g.s.NodesOffsetByAddress(addr).Int64()))
	g.Require().Equal(0, int(g.s.NodesOffsetByNodeKeyAddress(addr2).Int64()))

	// Duplicate NodeKey.
	_, addr3 := newPrefundAccount(g.stateDB)
	pk3 := crypto.FromECDSAPub(&privKey.PublicKey)
	input, err = GovernanceABI.ABI.Pack("register", pk3, "Test2", "test2@dexon.org", "Taipei", "https://dexon.org")
	g.Require().NoError(err)

	_, err = g.call(GovernanceContractAddress, addr3, input, amount)
	g.Require().NoError(err)

	input, err = GovernanceABI.ABI.Pack("replaceNodePublicKey", pk2)
	g.Require().NoError(err)

	// Duplicate nodekey
	_, err = g.call(GovernanceContractAddress, addr3, input, big.NewInt(0))
	g.Require().Error(err)
}

func (g *OracleContractsTestSuite) TestStakingMechanism() {
	privKey, addr := newPrefundAccount(g.stateDB)
	pk := crypto.FromECDSAPub(&privKey.PublicKey)

	// Register with some stake.
	amount := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(5e5))
	balanceBeforeStake := g.stateDB.GetBalance(addr)
	input, err := GovernanceABI.ABI.Pack("register", pk, "Test1", "test1@dexon.org", "Taipei", "https://dexon.org")
	g.Require().NoError(err)
	_, err = g.call(GovernanceContractAddress, addr, input, amount)
	g.Require().NoError(err)

	g.Require().Equal(1, int(g.s.LenNodes().Uint64()))
	g.Require().Equal(0, len(g.s.QualifiedNodes()))
	g.Require().Equal("Test1", g.s.Node(big.NewInt(0)).Name)
	g.Require().Equal(amount.String(), g.s.TotalStaked().String())

	// Check balance.
	g.Require().Equal(new(big.Int).Sub(balanceBeforeStake, amount), g.stateDB.GetBalance(addr))
	g.Require().Equal(new(big.Int).Add(big.NewInt(1), amount), g.stateDB.GetBalance(GovernanceContractAddress))

	// Registering again should fail.
	_, err = g.call(GovernanceContractAddress, addr, input, amount)
	g.Require().Error(err)

	// Duplicate public key should fail
	_, addrDup := newPrefundAccount(g.stateDB)
	input, err = GovernanceABI.ABI.Pack("register", pk, "Test1", "test1@dexon.org", "Taipei", "https://dexon.org")
	g.Require().NoError(err)
	_, err = g.call(GovernanceContractAddress, addrDup, input, amount)
	g.Require().Error(err)

	// Stake more to qualify.
	input, err = GovernanceABI.ABI.Pack("stake")
	g.Require().NoError(err)
	_, err = g.call(GovernanceContractAddress, addr, input, amount)
	g.Require().NoError(err)
	g.Require().Equal(1, len(g.s.QualifiedNodes()))
	g.Require().Equal(new(big.Int).Add(amount, amount).String(), g.s.TotalStaked().String())

	// Unstake more then staked should fail.
	unstakeAmount := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(5e6))
	input, err = GovernanceABI.ABI.Pack("unstake", unstakeAmount)
	g.Require().NoError(err)
	_, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().Error(err)

	// Unstake.
	input, err = GovernanceABI.ABI.Pack("unstake", amount)
	g.Require().NoError(err)
	_, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NoError(err)
	g.Require().Equal(0, len(g.s.QualifiedNodes()))
	g.Require().Equal(amount.String(), g.s.TotalStaked().String())

	var ok bool
	// Withdraw immediately should fail.
	input, err = GovernanceABI.ABI.Pack("withdrawable")
	g.Require().NoError(err)
	output, err := g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NoError(err)
	GovernanceABI.ABI.Unpack(&ok, "withdrawable", output)
	g.Require().False(ok)
	input, err = GovernanceABI.ABI.Pack("withdraw")
	g.Require().NoError(err)
	_, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().Error(err)

	// Wait for lockup time than withdraw.
	input, err = GovernanceABI.ABI.Pack("withdrawable")
	g.Require().NoError(err)
	output, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NoError(err)
	GovernanceABI.ABI.Unpack(&ok, "withdrawable", output)
	g.Require().False(ok)
	time.Sleep(time.Second * 2)
	input, err = GovernanceABI.ABI.Pack("withdraw")
	g.Require().NoError(err)
	_, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NoError(err)

	g.Require().Equal(0, len(g.s.QualifiedNodes()))
	g.Require().Equal(1, int(g.s.LenNodes().Uint64()))

	// Unstake all to remove node.
	input, err = GovernanceABI.ABI.Pack("unstake", amount)
	g.Require().NoError(err)
	_, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NoError(err)

	time.Sleep(time.Second * 2)
	input, err = GovernanceABI.ABI.Pack("withdrawable")
	g.Require().NoError(err)
	output, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NoError(err)
	GovernanceABI.ABI.Unpack(&ok, "withdrawable", output)
	g.Require().True(ok)
	input, err = GovernanceABI.ABI.Pack("withdraw")
	g.Require().NoError(err)
	_, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NoError(err)

	g.Require().Equal(0, len(g.s.QualifiedNodes()))
	g.Require().Equal(0, int(g.s.LenNodes().Uint64()))

	node := g.s.Node(big.NewInt(0))
	g.Require().Equal(big.NewInt(0).String(), node.Staked.String())
	g.Require().Equal(big.NewInt(0).String(), g.s.TotalStaked().String())

	// Stake 2 nodes, and unstake the first then the second.
	amount = new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e6))

	// 2nd node Stake.
	privKey2, addr2 := newPrefundAccount(g.stateDB)
	pk2 := crypto.FromECDSAPub(&privKey2.PublicKey)
	input, err = GovernanceABI.ABI.Pack("register", pk2, "Test2", "test2@dexon.org", "Taipei", "https://dexon.org")
	g.Require().NoError(err)
	_, err = g.call(GovernanceContractAddress, addr2, input, amount)
	g.Require().NoError(err)
	g.Require().Equal("Test2", g.s.Node(big.NewInt(0)).Name)
	g.Require().Equal(0, int(g.s.NodesOffsetByAddress(addr2).Int64()))

	// 1st node Stake.
	input, err = GovernanceABI.ABI.Pack("register", pk, "Test1", "test1@dexon.org", "Taipei", "https://dexon.org")
	g.Require().NoError(err)
	_, err = g.call(GovernanceContractAddress, addr, input, amount)
	g.Require().NoError(err)

	g.Require().Equal(2, int(g.s.LenNodes().Uint64()))
	g.Require().Equal(2, len(g.s.QualifiedNodes()))
	g.Require().Equal(new(big.Int).Mul(amount, big.NewInt(2)).String(), g.s.TotalStaked().String())

	// 2nd node Unstake.
	input, err = GovernanceABI.ABI.Pack("unstake", amount)
	g.Require().NoError(err)
	_, err = g.call(GovernanceContractAddress, addr2, input, big.NewInt(0))
	g.Require().NoError(err)

	node = g.s.Node(big.NewInt(0))
	g.Require().Equal("Test2", node.Name)
	g.Require().Equal(big.NewInt(0).String(), node.Staked.String())
	g.Require().Equal(1, len(g.s.QualifiedNodes()))
	g.Require().Equal(amount.String(), g.s.TotalStaked().String())

	time.Sleep(time.Second * 2)
	input, err = GovernanceABI.ABI.Pack("withdraw")
	g.Require().NoError(err)
	_, err = g.call(GovernanceContractAddress, addr2, input, big.NewInt(0))
	g.Require().NoError(err)

	g.Require().Equal(1, len(g.s.QualifiedNodes()))
	g.Require().Equal(1, int(g.s.LenNodes().Uint64()))
	g.Require().Equal("Test1", g.s.Node(big.NewInt(0)).Name)
	g.Require().Equal(-1, int(g.s.NodesOffsetByAddress(addr2).Int64()))

	// 1st node Unstake.
	input, err = GovernanceABI.ABI.Pack("unstake", amount)
	g.Require().NoError(err)
	_, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NoError(err)
	g.Require().Equal(0, len(g.s.QualifiedNodes()))
	g.Require().Equal(big.NewInt(0).String(), g.s.TotalStaked().String())

	time.Sleep(time.Second * 2)
	input, err = GovernanceABI.ABI.Pack("withdraw")
	g.Require().NoError(err)
	_, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NoError(err)

	g.Require().Equal(0, int(g.s.LenNodes().Uint64()))
	g.Require().Equal(-1, int(g.s.NodesOffsetByAddress(addr).Int64()))

	// Check balance.
	g.Require().Equal(balanceBeforeStake, g.stateDB.GetBalance(addr))
	g.Require().Equal(balanceBeforeStake, g.stateDB.GetBalance(addr2))
	g.Require().Equal(big.NewInt(1), g.stateDB.GetBalance(GovernanceContractAddress))
}

func (g *OracleContractsTestSuite) TestFine() {
	privKey, addr := newPrefundAccount(g.stateDB)
	pk := crypto.FromECDSAPub(&privKey.PublicKey)

	// Stake.
	input, err := GovernanceABI.ABI.Pack("register", pk, "Test1", "test1@dexon.org", "Taipei", "https://dexon.org")
	g.Require().NoError(err)
	ownerStaked := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e6))
	_, err = g.call(GovernanceContractAddress, addr, input, ownerStaked)
	g.Require().NoError(err)
	g.Require().Equal(1, len(g.s.QualifiedNodes()))
	g.Require().Equal(ownerStaked, g.s.Node(big.NewInt(0)).Staked)

	_, finePayer := newPrefundAccount(g.stateDB)
	amount := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(5e5))

	// Paying to node without fine should fail.
	input, err = GovernanceABI.ABI.Pack("payFine", addr)
	g.Require().NoError(err)
	_, err = g.call(GovernanceContractAddress, finePayer, input, amount)
	g.Require().NotNil(err)

	// Fined.
	offset := g.s.NodesOffsetByAddress(addr)
	g.Require().True(offset.Cmp(big.NewInt(0)) >= 0)
	node := g.s.Node(offset)
	node.Fined = new(big.Int).Set(amount)
	g.s.UpdateNode(offset, node)
	node = g.s.Node(offset)
	g.Require().Equal(0, node.Fined.Cmp(amount))

	// Not qualified after fined.
	g.Require().Equal(0, len(g.s.QualifiedNodes()))

	// Cannot unstake before fines are paied.
	input, err = GovernanceABI.ABI.Pack("unstake", big.NewInt(10))
	g.Require().NoError(err)
	_, err = g.call(GovernanceContractAddress, finePayer, input, big.NewInt(0))
	g.Require().NotNil(err)

	// Paying more than fine should fail.
	payAmount := new(big.Int).Add(amount, amount)
	input, err = GovernanceABI.ABI.Pack("payFine", addr)
	g.Require().NoError(err)
	_, err = g.call(GovernanceContractAddress, finePayer, input, payAmount)
	g.Require().NotNil(err)

	// Pay the fine.
	govBalance := g.stateDB.GetBalance(GovernanceContractAddress)
	ownerBalance := g.stateDB.GetBalance(g.config.Owner)

	input, err = GovernanceABI.ABI.Pack("payFine", addr)
	g.Require().NoError(err)
	_, err = g.call(GovernanceContractAddress, finePayer, input, amount)
	g.Require().NoError(err)

	g.Require().Equal(g.stateDB.GetBalance(GovernanceContractAddress).String(),
		govBalance.String())
	g.Require().Equal(g.stateDB.GetBalance(g.config.Owner).String(),
		new(big.Int).Add(ownerBalance, amount).String())

	// Qualified.
	g.Require().Equal(1, len(g.s.QualifiedNodes()))
}

func (g *OracleContractsTestSuite) TestUpdateConfiguration() {
	_, addr := newPrefundAccount(g.stateDB)

	input, err := GovernanceABI.ABI.Pack("updateConfiguration",
		new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e6)),
		big.NewInt(1000),
		big.NewInt(2e9),
		big.NewInt(8000000),
		big.NewInt(250),
		big.NewInt(2500),
		big.NewInt(int64(70.5*decimalMultiplier)),
		big.NewInt(264*decimalMultiplier),
		big.NewInt(600),
		big.NewInt(900),
		[]*big.Int{big.NewInt(1), big.NewInt(1), big.NewInt(1), big.NewInt(1), big.NewInt(1)})
	g.Require().NoError(err)

	// Call with non-owner.
	_, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NotNil(err)

	// Call with owner.
	_, err = g.call(GovernanceContractAddress, g.config.Owner, input, big.NewInt(0))
	g.Require().NoError(err)
}

func (g *OracleContractsTestSuite) TestConfigurationReading() {
	_, addr := newPrefundAccount(g.stateDB)

	// CRS.
	input, err := GovernanceABI.ABI.Pack("crs")
	g.Require().NoError(err)
	res, err := g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NoError(err)
	var crs0 [32]byte
	err = GovernanceABI.ABI.Unpack(&crs0, "crs", res)
	g.Require().NoError(err)
	g.Require().Equal(crypto.Keccak256Hash([]byte(g.config.GenesisCRSText)),
		common.BytesToHash(crs0[:]))

	// CRSRound.
	input, err = GovernanceABI.ABI.Pack("crsRound")
	g.Require().NoError(err)
	res, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NoError(err)

	// Owner.
	input, err = GovernanceABI.ABI.Pack("owner")
	g.Require().NoError(err)
	res, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NoError(err)
	var owner common.Address
	err = GovernanceABI.ABI.Unpack(&owner, "owner", res)
	g.Require().NoError(err)
	g.Require().Equal(g.config.Owner, owner)

	// MinStake.
	input, err = GovernanceABI.ABI.Pack("minStake")
	g.Require().NoError(err)
	res, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NoError(err)
	var value *big.Int
	err = GovernanceABI.ABI.Unpack(&value, "minStake", res)
	g.Require().NoError(err)
	g.Require().Equal(g.config.MinStake.String(), value.String())

	// BlockReward.
	input, err = GovernanceABI.ABI.Pack("miningVelocity")
	g.Require().NoError(err)
	res, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NoError(err)
	err = GovernanceABI.ABI.Unpack(&value, "miningVelocity", res)
	g.Require().NoError(err)
	g.Require().Equal(g.config.MiningVelocity, float32(value.Uint64())/decimalMultiplier)

	// BlockGasLimit.
	input, err = GovernanceABI.ABI.Pack("blockGasLimit")
	g.Require().NoError(err)
	res, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NoError(err)
	err = GovernanceABI.ABI.Unpack(&value, "blockGasLimit", res)
	g.Require().NoError(err)
	g.Require().Equal(g.config.BlockGasLimit, value.Uint64())

	// LambdaBA.
	input, err = GovernanceABI.ABI.Pack("lambdaBA")
	g.Require().NoError(err)
	res, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NoError(err)
	err = GovernanceABI.ABI.Unpack(&value, "lambdaBA", res)
	g.Require().NoError(err)
	g.Require().Equal(g.config.LambdaBA, value.Uint64())

	// LambdaDKG.
	input, err = GovernanceABI.ABI.Pack("lambdaDKG")
	g.Require().NoError(err)
	res, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NoError(err)
	err = GovernanceABI.ABI.Unpack(&value, "lambdaDKG", res)
	g.Require().NoError(err)
	g.Require().Equal(g.config.LambdaDKG, value.Uint64())

	// NotarySetSize.
	input, err = GovernanceABI.ABI.Pack("notarySetSize")
	g.Require().NoError(err)
	res, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NoError(err)
	err = GovernanceABI.ABI.Unpack(&value, "notarySetSize", res)
	g.Require().NoError(err)
	g.Require().True(uint32(value.Uint64()) > 0)

	// DKGRound.
	input, err = GovernanceABI.ABI.Pack("dkgRound")
	g.Require().NoError(err)
	res, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NoError(err)

	// DKGResetCount.
	input, err = GovernanceABI.ABI.Pack("dkgResetCount", big.NewInt(3))
	g.Require().NoError(err)
	res, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NoError(err)

	// RoundLength.
	input, err = GovernanceABI.ABI.Pack("roundLength")
	g.Require().NoError(err)
	res, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NoError(err)
	err = GovernanceABI.ABI.Unpack(&value, "roundLength", res)
	g.Require().NoError(err)
	g.Require().Equal(g.config.RoundLength, value.Uint64())

	// MinBlockInterval.
	input, err = GovernanceABI.ABI.Pack("minBlockInterval")
	g.Require().NoError(err)
	res, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NoError(err)
	err = GovernanceABI.ABI.Unpack(&value, "minBlockInterval", res)
	g.Require().NoError(err)
	g.Require().Equal(g.config.MinBlockInterval, value.Uint64())

	// MinGasPrice.
	input, err = GovernanceABI.ABI.Pack("minGasPrice")
	g.Require().NoError(err)
	res, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NoError(err)
	err = GovernanceABI.ABI.Unpack(&value, "minGasPrice", res)
	g.Require().NoError(err)
	g.Require().Equal(g.config.MinGasPrice, value)
}

func (g *OracleContractsTestSuite) TestReportForkVote() {
	key, addr := newPrefundAccount(g.stateDB)
	pkBytes := crypto.FromECDSAPub(&key.PublicKey)

	// Stake.
	amount := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(5e5))
	input, err := GovernanceABI.ABI.Pack("register", pkBytes, "Test1", "test1@dexon.org", "Taipei", "https://dexon.org")
	g.Require().NoError(err)
	_, err = g.call(GovernanceContractAddress, addr, input, amount)
	g.Require().NoError(err)

	pubKey := coreEcdsa.NewPublicKeyFromECDSA(&key.PublicKey)
	privKey := coreEcdsa.NewPrivateKeyFromECDSA(key)
	vote1 := coreTypes.NewVote(coreTypes.VoteCom, coreCommon.NewRandomHash(), uint64(0))
	vote1.ProposerID = coreTypes.NewNodeID(pubKey)

	vote2 := vote1.Clone()
	for vote2.BlockHash == vote1.BlockHash {
		vote2.BlockHash = coreCommon.NewRandomHash()
	}
	vote1.Signature, err = privKey.Sign(coreUtils.HashVote(vote1))
	g.Require().NoError(err)
	vote2.Signature, err = privKey.Sign(coreUtils.HashVote(vote2))
	g.Require().NoError(err)

	vote1Bytes, err := rlp.EncodeToBytes(vote1)
	g.Require().NoError(err)
	vote2Bytes, err := rlp.EncodeToBytes(vote2)
	g.Require().NoError(err)

	// Report wrong type (fork block)
	input, err = GovernanceABI.ABI.Pack("report", big.NewInt(FineTypeForkBlock), vote1Bytes, vote2Bytes)
	g.Require().NoError(err)
	_, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().Error(err)

	input, err = GovernanceABI.ABI.Pack("report", big.NewInt(FineTypeForkVote), vote1Bytes, vote2Bytes)
	g.Require().NoError(err)
	_, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NoError(err)

	node := g.s.Node(big.NewInt(0))
	g.Require().Equal(node.Fined, g.s.FineValue(big.NewInt(FineTypeForkVote)))

	// Duplicate report should fail.
	input, err = GovernanceABI.ABI.Pack("report", big.NewInt(FineTypeForkVote), vote1Bytes, vote2Bytes)
	g.Require().NoError(err)
	_, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().Error(err)

	// Check if finedRecords is set.
	payloads := [][]byte{vote1Bytes, vote2Bytes}
	sort.Sort(sortBytes(payloads))

	hash := Bytes32(crypto.Keccak256Hash(payloads...))
	input, err = GovernanceABI.ABI.Pack("finedRecords", hash)
	g.Require().NoError(err)
	res, err := g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NoError(err)

	var value bool
	err = GovernanceABI.ABI.Unpack(&value, "finedRecords", res)
	g.Require().NoError(err)
	g.Require().True(value)
}

func (g *OracleContractsTestSuite) TestReportForkBlock() {
	key, addr := newPrefundAccount(g.stateDB)
	pkBytes := crypto.FromECDSAPub(&key.PublicKey)

	// Stake.
	amount := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(5e5))
	input, err := GovernanceABI.ABI.Pack("register", pkBytes, "Test1", "test1@dexon.org", "Taipei, Taiwan", "https://dexon.org")
	g.Require().NoError(err)
	_, err = g.call(GovernanceContractAddress, addr, input, amount)
	g.Require().NoError(err)

	privKey := coreEcdsa.NewPrivateKeyFromECDSA(key)
	block1 := &coreTypes.Block{
		ProposerID: coreTypes.NewNodeID(privKey.PublicKey()),
		ParentHash: coreCommon.NewRandomHash(),
		Timestamp:  time.Now(),
	}

	block2 := block1.Clone()
	for block2.ParentHash == block1.ParentHash {
		block2.ParentHash = coreCommon.NewRandomHash()
	}

	hashBlock := func(block *coreTypes.Block) coreCommon.Hash {
		block.PayloadHash = coreCrypto.Keccak256Hash(block.Payload)
		var err error
		block.Hash, err = coreUtils.HashBlock(block)
		g.Require().NoError(err)
		return block.Hash
	}

	block1.Signature, err = privKey.Sign(hashBlock(block1))
	g.Require().NoError(err)
	block2.Signature, err = privKey.Sign(hashBlock(block2))
	g.Require().NoError(err)

	block1Bytes, err := rlp.EncodeToBytes(block1)
	g.Require().NoError(err)
	block2Bytes, err := rlp.EncodeToBytes(block2)
	g.Require().NoError(err)

	// Report wrong type (fork vote)
	input, err = GovernanceABI.ABI.Pack("report", big.NewInt(FineTypeForkVote), block1Bytes, block2Bytes)
	g.Require().NoError(err)
	_, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().Error(err)

	input, err = GovernanceABI.ABI.Pack("report", big.NewInt(FineTypeForkBlock), block1Bytes, block2Bytes)
	g.Require().NoError(err)
	_, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NoError(err)

	node := g.s.Node(big.NewInt(0))
	g.Require().Equal(node.Fined, g.s.FineValue(big.NewInt(FineTypeForkBlock)))

	// Duplicate report should fail.
	input, err = GovernanceABI.ABI.Pack("report", big.NewInt(FineTypeForkBlock), block1Bytes, block2Bytes)
	g.Require().NoError(err)
	_, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().Error(err)

	// Check if finedRecords is set.
	payloads := [][]byte{block1Bytes, block2Bytes}
	sort.Sort(sortBytes(payloads))

	hash := Bytes32(crypto.Keccak256Hash(payloads...))
	input, err = GovernanceABI.ABI.Pack("finedRecords", hash)
	g.Require().NoError(err)
	res, err := g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NoError(err)

	var value bool
	err = GovernanceABI.ABI.Unpack(&value, "finedRecords", res)
	g.Require().NoError(err)
	g.Require().True(value)
}

func (g *OracleContractsTestSuite) TestMiscVariableReading() {
	privKey, addr := newPrefundAccount(g.stateDB)
	pk := crypto.FromECDSAPub(&privKey.PublicKey)

	input, err := GovernanceABI.ABI.Pack("totalSupply")
	g.Require().NoError(err)
	res, err := g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NoError(err)

	input, err = GovernanceABI.ABI.Pack("totalStaked")
	g.Require().NoError(err)
	res, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NoError(err)

	// Stake.
	amount := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(5e5))
	input, err = GovernanceABI.ABI.Pack("register", pk, "Test1", "test1@dexon.org", "Taipei, Taiwan", "https://dexon.org")
	g.Require().NoError(err)
	_, err = g.call(GovernanceContractAddress, addr, input, amount)
	g.Require().NoError(err)

	input, err = GovernanceABI.ABI.Pack("nodes", big.NewInt(0))
	g.Require().NoError(err)
	res, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NoError(err)

	input, err = GovernanceABI.ABI.Pack("nodesLength")
	g.Require().NoError(err)
	res, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NoError(err)
	var value *big.Int
	err = GovernanceABI.ABI.Unpack(&value, "nodesLength", res)
	g.Require().NoError(err)
	g.Require().Equal(1, int(value.Uint64()))

	input, err = GovernanceABI.ABI.Pack("nodesOffsetByAddress", addr)
	g.Require().NoError(err)
	res, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NoError(err)
	err = GovernanceABI.ABI.Unpack(&value, "nodesOffsetByAddress", res)
	g.Require().NoError(err)
	g.Require().Equal(0, int(value.Uint64()))

	addr, err = publicKeyToNodeKeyAddress(pk)
	g.Require().NoError(err)
	input, err = GovernanceABI.ABI.Pack("nodesOffsetByNodeKeyAddress", addr)
	g.Require().NoError(err)
	res, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NoError(err)
	err = GovernanceABI.ABI.Unpack(&value, "nodesOffsetByNodeKeyAddress", res)
	g.Require().NoError(err)
	g.Require().Equal(0, int(value.Uint64()))

	input, err = GovernanceABI.ABI.Pack("fineValues", big.NewInt(0))
	g.Require().NoError(err)
	res, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
	g.Require().NoError(err)
}

func (g *OracleContractsTestSuite) TestHalvingCondition() {
	// TotalSupply 2.5B reached
	g.s.MiningHalved()
	g.Require().Equal(new(big.Int).Mul(big.NewInt(1e18), big.NewInt(3.25e9)).String(),
		g.s.NextHalvingSupply().String())
	g.Require().Equal(new(big.Int).Mul(big.NewInt(1e18), big.NewInt(0.75e9)).String(),
		g.s.LastHalvedAmount().String())

	// TotalSupply 3.25B reached
	g.s.MiningHalved()
	g.Require().Equal(new(big.Int).Mul(big.NewInt(1e18), big.NewInt(3.625e9)).String(),
		g.s.NextHalvingSupply().String())
	g.Require().Equal(new(big.Int).Mul(big.NewInt(1e18), big.NewInt(0.375e9)).String(),
		g.s.LastHalvedAmount().String())
}

type testCoreMock struct {
	newDKGGPKError error
	tsigReturn     bool
}

func (m *testCoreMock) NewGroupPublicKey(*GovernanceState, *big.Int, int) (tsigVerifierIntf, error) {
	if m.newDKGGPKError != nil {
		return nil, m.newDKGGPKError
	}
	return &testTSigVerifierMock{m.tsigReturn}, nil
}

type testTSigVerifierMock struct {
	ret bool
}

func (v *testTSigVerifierMock) VerifySignature(coreCommon.Hash, coreCrypto.Signature) bool {
	return v.ret
}

func (g *OracleContractsTestSuite) TestResetDKG() {
	for i := 0; i < 7; i++ {
		privKey, addr := newPrefundAccount(g.stateDB)
		pk := crypto.FromECDSAPub(&privKey.PublicKey)

		// Stake.
		amount := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e6))
		input, err := GovernanceABI.ABI.Pack("register", pk, "Test", "test1@dexon.org", "Taipei", "https://dexon.org")
		g.Require().NoError(err)
		_, err = g.call(GovernanceContractAddress, addr, input, amount)
		g.Require().NoError(err)
	}
	g.Require().Len(g.s.QualifiedNodes(), int(g.s.NotarySetSize().Uint64()))

	addrs := make(map[int][]common.Address)
	dkgSets := make(map[int]map[coreTypes.NodeID]struct{})
	addDKG := func(round int, final, proposeCRS bool) {
		if proposeCRS && uint64(round) > dexCore.DKGDelayRound {
			// ProposeCRS and clear DKG state.
			input, err := GovernanceABI.ABI.Pack(
				"proposeCRS", big.NewInt(int64(round)), randomBytes(32, 32))
			g.Require().NoError(err)
			_, err = g.call(GovernanceContractAddress, addrs[round-1][0], input, big.NewInt(0))
			g.Require().NoError(err)

			// Clear DKG states for next round.
			dkgSet := dkgSets[round-1]
			g.s.ClearDKGMasterPublicKeyOffset()
			g.s.ClearDKGMasterPublicKeys()
			g.s.ClearDKGComplaintProposed()
			g.s.ClearDKGComplaints()
			g.s.ClearDKGMPKReadys(dkgSet)
			g.s.ResetDKGMPKReadysCount()
			g.s.ClearDKGFinalizeds(dkgSet)
			g.s.ResetDKGFinalizedsCount()
			g.s.ClearDKGSuccesses(dkgSet)
			g.s.ResetDKGSuccessesCount()
			g.s.SetDKGRound(big.NewInt(int64(round)))
		}

		addrs[round] = []common.Address{}
		target := coreTypes.NewNotarySetTarget(coreCommon.Hash(g.s.CRS()))
		ns := coreTypes.NewNodeSet()

		for _, x := range g.s.QualifiedNodes() {
			mpk, err := coreEcdsa.NewPublicKeyFromByteSlice(x.PublicKey)
			if err != nil {
				panic(err)
			}
			ns.Add(coreTypes.NewNodeID(mpk))
		}
		dkgSet := ns.GetSubSet(int(g.s.NotarySetSize().Uint64()), target)
		g.Require().Len(dkgSet, int(g.s.NotarySetSize().Uint64()))
		dkgSets[round] = dkgSet

		i := 0
		for id := range dkgSet {
			offset := g.s.NodesOffsetByNodeKeyAddress(IdToAddress(id))
			if offset.Cmp(big.NewInt(0)) < 0 {
				panic("DKG node does not exist")
			}
			node := g.s.Node(offset)
			// Prepare MPK.
			x := dkgTypes.MasterPublicKey{}
			b, err := rlp.EncodeToBytes(&x)
			if err != nil {
				panic(err)
			}
			g.s.PushDKGMasterPublicKey(b)
			g.s.PutDKGMasterPublicKeyOffset(Bytes32(id.Hash), big.NewInt(int64(i)))
			// Prepare Complaint.
			y := dkgTypes.Complaint{}
			b, err = rlp.EncodeToBytes(&y)
			if err != nil {
				panic(err)
			}
			g.s.PushDKGComplaint(b)
			addr := node.Owner
			addrs[round] = append(addrs[round], addr)
			// Prepare MPK Ready.
			g.s.PutDKGMPKReady(addr, true)
			g.s.IncDKGMPKReadysCount()
			if final {
				// Prepare Finalized.
				g.s.PutDKGFinalized(addr, true)
				g.s.IncDKGFinalizedsCount()
				// Prepare Success.
				g.s.PutDKGSuccess(addr, true)
				g.s.IncDKGSuccessesCount()
			}
			i += 1
		}
		dkgSetSize := len(dkgSet)
		g.Require().Len(g.s.DKGMasterPublicKeys(), dkgSetSize)
		g.Require().Len(g.s.DKGComplaints(), dkgSetSize)
		g.Require().Equal(int64(dkgSetSize), g.s.DKGMPKReadysCount().Int64())
		for _, addr := range addrs[round] {
			g.Require().True(g.s.DKGMPKReady(addr))
		}

		if final {
			g.Require().Equal(int64(dkgSetSize), g.s.DKGFinalizedsCount().Int64())
			for _, addr := range addrs[round] {
				g.Require().True(g.s.DKGFinalized(addr))
			}
			g.Require().Equal(int64(dkgSetSize), g.s.DKGSuccessesCount().Int64())
			for _, addr := range addrs[round] {
				g.Require().True(g.s.DKGSuccess(addr))
			}
		}

	}

	mock := &testCoreMock{
		tsigReturn: true,
	}
	OracleContracts[GovernanceContractAddress] = func() OracleContract {
		return &GovernanceContract{
			coreDKGUtils: mock,
		}
	}

	// Fill data for previous rounds.
	roundHeight := int64(g.config.RoundLength)
	round := int(dexCore.DKGDelayRound) + 3
	for i := 0; i <= round; i++ {
		g.context.Round = big.NewInt(int64(i))

		// Prepare Round Height
		if i != 0 {
			g.s.PushRoundHeight(big.NewInt(int64(i) * roundHeight))
		}

		addDKG(i+1, true, true)
	}

	round++
	g.s.PushRoundHeight(big.NewInt(int64(round) * roundHeight))
	g.context.Round = big.NewInt(int64(round))
	addDKG(round+1, false, true)
	repeat := 3
	for r := 0; r < repeat; r++ {
		// Add one finalized for test.
		roundPlusOne := big.NewInt(int64(round + 1))
		g.s.PutDKGFinalized(addrs[round+1][0], true)
		g.s.IncDKGFinalizedsCount()

		g.context.BlockNumber = big.NewInt(
			roundHeight*int64(round) + roundHeight*int64(r) + roundHeight*85/100)
		_, addr := newPrefundAccount(g.stateDB)
		newCRS := randomBytes(common.HashLength, common.HashLength)
		input, err := GovernanceABI.ABI.Pack("resetDKG", newCRS)
		g.Require().NoError(err)
		_, err = g.call(GovernanceContractAddress, addr, input, big.NewInt(0))
		g.Require().NoError(err)

		// Test if CRS is reset.
		newCRSHash := crypto.Keccak256Hash(newCRS)
		g.Require().Equal(newCRSHash, g.s.CRS())

		// Test if MPK is purged.
		g.Require().Len(g.s.DKGMasterPublicKeys(), 0)
		// Test if MPKReady is purged.
		g.Require().Equal(int64(0), g.s.DKGMPKReadysCount().Int64())
		for _, addr := range addrs[round+1] {
			g.Require().False(g.s.DKGMPKReady(addr))
		}
		// Test if Complaint is purged.
		g.Require().Len(g.s.DKGComplaints(), 0)
		// Test if Finalized is purged.
		g.Require().Equal(int64(0), g.s.DKGFinalizedsCount().Int64())
		for _, addr := range addrs[round+1] {
			g.Require().False(g.s.DKGFinalized(addr))
		}
		// Test if Success is purged.
		g.Require().Equal(int64(0), g.s.DKGSuccessesCount().Int64())
		for _, addr := range addrs[round+1] {
			g.Require().False(g.s.DKGSuccess(addr))
		}

		g.Require().Equal(int64(r+1), g.s.DKGResetCount(roundPlusOne).Int64())

		addDKG(round+1, false, false)
	}
}

func TestOracleContracts(t *testing.T) {
	suite.Run(t, new(OracleContractsTestSuite))
}
