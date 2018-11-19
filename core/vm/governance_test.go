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
	"testing"
	"time"

	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/core/state"
	"github.com/dexon-foundation/dexon/crypto"
	"github.com/dexon-foundation/dexon/ethdb"
	"github.com/dexon-foundation/dexon/params"
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

type GovernanceContractTestSuite struct {
	suite.Suite

	config  *params.DexconConfig
	memDB   *ethdb.MemDatabase
	stateDB *state.StateDB
	s       *GovernanceStateHelper
}

func (g *GovernanceContractTestSuite) SetupTest() {
	memDB := ethdb.NewMemDatabase()
	stateDB, err := state.New(common.Hash{}, state.NewDatabase(memDB))
	if err != nil {
		panic(err)
	}
	g.memDB = memDB
	g.stateDB = stateDB
	g.s = &GovernanceStateHelper{stateDB}

	config := params.TestnetChainConfig.Dexcon
	g.config = config

	// Give governance contract balance so it will not be deleted because of being an empty state object.
	stateDB.AddBalance(GovernanceContractAddress, big.NewInt(1))

	// Genesis CRS.
	crs := crypto.Keccak256Hash([]byte(config.GenesisCRSText))
	g.s.PushCRS(crs)

	// Round 0 height.
	g.s.PushRoundHeight(big.NewInt(0))

	// Owner.
	g.s.SetOwner(g.config.Owner)

	// Governance configuration.
	g.s.UpdateConfiguration(config)

	g.stateDB.Commit(true)
}

func (g *GovernanceContractTestSuite) newPrefundAccount() (*ecdsa.PrivateKey, common.Address) {
	privKey, err := crypto.GenerateKey()
	if err != nil {
		panic(err)
	}
	address := crypto.PubkeyToAddress(privKey.PublicKey)

	g.stateDB.AddBalance(address, new(big.Int).Mul(big.NewInt(1e18), big.NewInt(2e5)))
	return privKey, address
}

func (g *GovernanceContractTestSuite) call(caller common.Address, input []byte, value *big.Int) ([]byte, error) {
	context := Context{
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
		Time:        big.NewInt(time.Now().UnixNano() / 1000000000),
		BlockNumber: big.NewInt(0),
	}

	evm := NewEVM(context, g.stateDB, params.TestChainConfig, Config{IsBlockProposer: true})
	ret, _, err := evm.Call(AccountRef(caller), GovernanceContractAddress, input, 10000000, value)
	return ret, err
}

func (g *GovernanceContractTestSuite) TestTransferOwnership() {
	_, addr := g.newPrefundAccount()

	input, err := abiObject.Pack("transferOwnership", addr)
	g.Require().Nil(err)

	// Call with non-owner.
	_, err = g.call(addr, input, big.NewInt(0))
	g.Require().NotNil(err)

	// Call with owner.
	_, err = g.call(g.config.Owner, input, big.NewInt(0))
	g.Require().Nil(err)
	g.Require().Equal(addr, g.s.Owner())
}

func (g *GovernanceContractTestSuite) TestStakeUnstakeWithoutDelegators() {
	privKey, addr := g.newPrefundAccount()
	pk := crypto.FromECDSAPub(&privKey.PublicKey)

	// Stake.
	amount := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(5e4))
	balanceBeforeStake := g.stateDB.GetBalance(addr)
	input, err := abiObject.Pack("stake", pk, "Test1", "test1@dexon.org", "Taipei, Taiwan", "https://dexon.org")
	g.Require().Nil(err)
	_, err = g.call(addr, input, amount)
	g.Require().Nil(err)

	// Node staked but staked fund < MinStake so is still unqualified.
	g.Require().Equal(1, int(g.s.LenNodes().Uint64()))
	g.Require().Equal(0, len(g.s.QualifiedNodes()))
	g.Require().Equal("Test1", g.s.Node(big.NewInt(0)).Name)

	// Check balance.
	g.Require().Equal(new(big.Int).Sub(balanceBeforeStake, amount), g.stateDB.GetBalance(addr))
	g.Require().Equal(new(big.Int).Add(big.NewInt(1), amount), g.stateDB.GetBalance(GovernanceContractAddress))

	// Staking again should fail.
	_, err = g.call(addr, input, amount)
	g.Require().NotNil(err)

	// Unstake.
	input, err = abiObject.Pack("unstake")
	g.Require().Nil(err)
	_, err = g.call(addr, input, big.NewInt(0))
	g.Require().Nil(err)
	g.Require().Equal(0, int(g.s.LenNodes().Uint64()))

	// Stake 2 nodes, and unstake the first then the second.

	// 2nd node Stake.
	privKey2, addr2 := g.newPrefundAccount()
	pk2 := crypto.FromECDSAPub(&privKey2.PublicKey)
	input, err = abiObject.Pack("stake", pk2, "Test2", "test2@dexon.org", "Taipei, Taiwan", "https://dexon.org")
	g.Require().Nil(err)
	_, err = g.call(addr2, input, amount)
	g.Require().Nil(err)
	g.Require().Equal("Test2", g.s.Node(big.NewInt(0)).Name)
	g.Require().Equal(0, int(g.s.NodesOffset(addr2).Int64()))

	// 1st node Stake.
	input, err = abiObject.Pack("stake", pk, "Test1", "test1@dexon.org", "Taipei, Taiwan", "https://dexon.org")
	g.Require().Nil(err)
	_, err = g.call(addr, input, amount)
	g.Require().Nil(err)

	// 2nd node Unstake.
	input, err = abiObject.Pack("unstake")
	g.Require().Nil(err)
	_, err = g.call(addr2, input, big.NewInt(0))
	g.Require().Nil(err)
	g.Require().Equal(1, int(g.s.LenNodes().Uint64()))
	g.Require().Equal("Test1", g.s.Node(big.NewInt(0)).Name)
	g.Require().Equal(-1, int(g.s.NodesOffset(addr2).Int64()))

	// 1st node Unstake.
	input, err = abiObject.Pack("unstake")
	g.Require().Nil(err)
	_, err = g.call(addr, input, big.NewInt(0))
	g.Require().Nil(err)
	g.Require().Equal(0, int(g.s.LenNodes().Uint64()))
	g.Require().Equal(-1, int(g.s.NodesOffset(addr).Int64()))
	g.Require().Equal(-1, int(g.s.DelegatorsOffset(addr, addr).Int64()))

	// Check balance.
	g.Require().Equal(balanceBeforeStake, g.stateDB.GetBalance(addr))
	g.Require().Equal(balanceBeforeStake, g.stateDB.GetBalance(addr2))
	g.Require().Equal(big.NewInt(1), g.stateDB.GetBalance(GovernanceContractAddress))
}

func (g *GovernanceContractTestSuite) TestDelegateUndelegate() {
	privKey, addr := g.newPrefundAccount()
	pk := crypto.FromECDSAPub(&privKey.PublicKey)

	// Stake.
	input, err := abiObject.Pack("stake", pk, "Test1", "test1@dexon.org", "Taipei, Taiwan", "https://dexon.org")
	g.Require().Nil(err)
	ownerStaked := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(5e4))
	_, err = g.call(addr, input, ownerStaked)
	g.Require().Nil(err)
	g.Require().Equal(0, len(g.s.QualifiedNodes()))
	g.Require().Equal(addr, g.s.Delegator(addr, big.NewInt(0)).Owner)
	g.Require().Equal(ownerStaked, g.s.Node(big.NewInt(0)).Staked)

	// 1st delegator delegate to 1st node.
	_, addrDelegator := g.newPrefundAccount()

	balanceBeforeDelegate := g.stateDB.GetBalance(addrDelegator)
	amount := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(3e4))
	input, err = abiObject.Pack("delegate", addr)
	g.Require().Nil(err)

	_, err = g.call(addrDelegator, input, amount)
	g.Require().Nil(err)
	g.Require().Equal(new(big.Int).Sub(balanceBeforeDelegate, amount), g.stateDB.GetBalance(addrDelegator))
	g.Require().Equal(addrDelegator, g.s.Delegator(addr, big.NewInt(1)).Owner)
	g.Require().Equal(new(big.Int).Add(amount, ownerStaked), g.s.Node(big.NewInt(0)).Staked)
	g.Require().Equal(1, int(g.s.DelegatorsOffset(addr, addrDelegator).Int64()))

	// Same person delegate the 2nd time should fail.
	_, err = g.call(addrDelegator, input, big.NewInt(1e18))
	g.Require().NotNil(err)

	// Not yet qualified.
	g.Require().Equal(0, len(g.s.QualifiedNodes()))

	// 2nd delegator delegate to 1st node.
	_, addrDelegator2 := g.newPrefundAccount()
	_, err = g.call(addrDelegator2, input, amount)
	g.Require().Nil(err)
	g.Require().Equal(new(big.Int).Sub(balanceBeforeDelegate, amount), g.stateDB.GetBalance(addrDelegator2))
	g.Require().Equal(addrDelegator2, g.s.Delegator(addr, big.NewInt(2)).Owner)
	g.Require().Equal(new(big.Int).Add(ownerStaked, new(big.Int).Mul(amount, big.NewInt(2))),
		g.s.Node(big.NewInt(0)).Staked)
	g.Require().Equal(2, int(g.s.DelegatorsOffset(addr, addrDelegator2).Int64()))

	// Qualified.
	g.Require().Equal(1, len(g.s.QualifiedNodes()))

	// Undelegate addrDelegator.
	balanceBeforeUnDelegate := g.stateDB.GetBalance(addrDelegator)
	input, err = abiObject.Pack("undelegate", addr)
	g.Require().Nil(err)
	_, err = g.call(addrDelegator, input, big.NewInt(0))
	g.Require().Nil(err)
	g.Require().Equal(2, int(g.s.LenDelegators(addr).Uint64()))
	g.Require().Equal(new(big.Int).Add(balanceBeforeUnDelegate, amount), g.stateDB.GetBalance(addrDelegator))
	g.Require().Equal(-1, int(g.s.DelegatorsOffset(addr, addrDelegator).Int64()))

	// Undelegate addrDelegator2.
	balanceBeforeUnDelegate = g.stateDB.GetBalance(addrDelegator2)
	input, err = abiObject.Pack("undelegate", addr)
	g.Require().Nil(err)
	_, err = g.call(addrDelegator2, input, big.NewInt(0))
	g.Require().Nil(err)
	g.Require().Equal(1, int(g.s.LenDelegators(addr).Uint64()))
	g.Require().Equal(new(big.Int).Add(balanceBeforeUnDelegate, amount), g.stateDB.GetBalance(addrDelegator2))
	g.Require().Equal(-1, int(g.s.DelegatorsOffset(addr, addrDelegator2).Int64()))

	// Unqualified
	g.Require().Equal(0, len(g.s.QualifiedNodes()))
}

func (g *GovernanceContractTestSuite) TestUnstakeWithDelegators() {
	privKey, addr := g.newPrefundAccount()
	pk := crypto.FromECDSAPub(&privKey.PublicKey)

	// Stake.
	amount := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(5e4))
	input, err := abiObject.Pack("stake", pk, "Test1", "test1@dexon.org", "Taipei, Taiwan", "https://dexon.org")
	g.Require().Nil(err)
	_, err = g.call(addr, input, amount)
	g.Require().Nil(err)

	// 1st delegator delegate to 1st node.
	_, addrDelegator := g.newPrefundAccount()

	balanceBeforeDelegate := g.stateDB.GetBalance(addrDelegator)
	amount = new(big.Int).Mul(big.NewInt(1e18), big.NewInt(3e4))
	input, err = abiObject.Pack("delegate", addr)
	g.Require().Nil(err)

	_, err = g.call(addrDelegator, input, amount)
	g.Require().Nil(err)
	g.Require().Equal(new(big.Int).Sub(balanceBeforeDelegate, amount), g.stateDB.GetBalance(addrDelegator))
	g.Require().Equal(addrDelegator, g.s.Delegator(addr, big.NewInt(1)).Owner)
	g.Require().Equal(0, len(g.s.QualifiedNodes()))

	// 2st delegator delegate to 1st node.
	_, addrDelegator2 := g.newPrefundAccount()

	balanceBeforeDelegate = g.stateDB.GetBalance(addrDelegator2)
	input, err = abiObject.Pack("delegate", addr)
	g.Require().Nil(err)

	_, err = g.call(addrDelegator2, input, amount)
	g.Require().Nil(err)
	g.Require().Equal(new(big.Int).Sub(balanceBeforeDelegate, amount), g.stateDB.GetBalance(addrDelegator2))
	g.Require().Equal(addrDelegator2, g.s.Delegator(addr, big.NewInt(2)).Owner)

	// Node is now qualified.
	g.Require().Equal(1, len(g.s.QualifiedNodes()))

	// Unstake.
	input, err = abiObject.Pack("unstake")
	g.Require().Nil(err)
	_, err = g.call(addr, input, big.NewInt(0))
	g.Require().Nil(err)
	g.Require().Equal(0, int(g.s.LenDelegators(addr).Uint64()))
	g.Require().Equal(0, int(g.s.LenNodes().Uint64()))

	// Check balance.
	g.Require().Equal(balanceBeforeDelegate, g.stateDB.GetBalance(addr))
	g.Require().Equal(balanceBeforeDelegate, g.stateDB.GetBalance(addrDelegator))
	g.Require().Equal(balanceBeforeDelegate, g.stateDB.GetBalance(addrDelegator2))
	g.Require().Equal(big.NewInt(1), g.stateDB.GetBalance(GovernanceContractAddress))
}

func (g *GovernanceContractTestSuite) TestUpdateConfiguration() {
	_, addr := g.newPrefundAccount()

	input, err := abiObject.Pack("updateConfiguration",
		new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e5)),
		big.NewInt(1e18), big.NewInt(8000000), big.NewInt(6), big.NewInt(250), big.NewInt(2500),
		big.NewInt(0), big.NewInt(667000), big.NewInt(4), big.NewInt(4), big.NewInt(600000), big.NewInt(900))
	g.Require().Nil(err)

	// Call with non-owner.
	_, err = g.call(addr, input, big.NewInt(0))
	g.Require().NotNil(err)

	// Call with owner.
	_, err = g.call(g.config.Owner, input, big.NewInt(0))
	g.Require().Nil(err)
}

func (g *GovernanceContractTestSuite) TestSnapshotRound() {
	_, addr := g.newPrefundAccount()

	// Wrong height.
	input, err := abiObject.Pack("snapshotRound", big.NewInt(1), big.NewInt(666))
	g.Require().Nil(err)
	_, err = g.call(addr, input, big.NewInt(0))
	g.Require().NotNil(err)

	// Invalid round.
	input, err = abiObject.Pack("snapshotRound", big.NewInt(2), big.NewInt(2000))
	g.Require().Nil(err)
	_, err = g.call(addr, input, big.NewInt(0))
	g.Require().NotNil(err)

	// Correct.
	input, err = abiObject.Pack("snapshotRound", big.NewInt(1), big.NewInt(1000))
	g.Require().Nil(err)
	_, err = g.call(addr, input, big.NewInt(0))
	g.Require().Nil(err)

	// Duplicate round.
	input, err = abiObject.Pack("snapshotRound", big.NewInt(1), big.NewInt(1000))
	g.Require().Nil(err)
	_, err = g.call(addr, input, big.NewInt(0))
	g.Require().NotNil(err)

	// Invalid round.
	input, err = abiObject.Pack("snapshotRound", big.NewInt(3), big.NewInt(3000))
	g.Require().Nil(err)
	_, err = g.call(addr, input, big.NewInt(0))
	g.Require().NotNil(err)

	// Correct.
	input, err = abiObject.Pack("snapshotRound", big.NewInt(2), big.NewInt(2000))
	g.Require().Nil(err)
	_, err = g.call(addr, input, big.NewInt(0))
	g.Require().Nil(err)
}

func (g *GovernanceContractTestSuite) TestConfigurationReading() {
	_, addr := g.newPrefundAccount()

	// CRS.
	input, err := abiObject.Pack("crs", big.NewInt(0))
	g.Require().Nil(err)
	res, err := g.call(addr, input, big.NewInt(0))
	g.Require().Nil(err)
	var crs0 [32]byte
	err = abiObject.Unpack(&crs0, "crs", res)
	g.Require().Nil(err)
	g.Require().Equal(crypto.Keccak256Hash([]byte(g.config.GenesisCRSText)), common.BytesToHash(crs0[:]))

	// Owner.
	input, err = abiObject.Pack("owner")
	g.Require().Nil(err)
	res, err = g.call(addr, input, big.NewInt(0))
	g.Require().Nil(err)
	var owner common.Address
	err = abiObject.Unpack(&owner, "owner", res)
	g.Require().Nil(err)
	g.Require().Equal(g.config.Owner, owner)

	// MinStake.
	input, err = abiObject.Pack("minStake")
	g.Require().Nil(err)
	res, err = g.call(addr, input, big.NewInt(0))
	g.Require().Nil(err)
	var value *big.Int
	err = abiObject.Unpack(&value, "minStake", res)
	g.Require().Nil(err)
	g.Require().Equal(g.config.MinStake.String(), value.String())

	// BlockReward.
	input, err = abiObject.Pack("blockReward")
	g.Require().Nil(err)
	res, err = g.call(addr, input, big.NewInt(0))
	g.Require().Nil(err)
	err = abiObject.Unpack(&value, "blockReward", res)
	g.Require().Nil(err)
	g.Require().Equal(g.config.BlockReward.String(), value.String())

	// BlockGasLimit.
	input, err = abiObject.Pack("blockGasLimit")
	g.Require().Nil(err)
	res, err = g.call(addr, input, big.NewInt(0))
	g.Require().Nil(err)
	err = abiObject.Unpack(&value, "blockGasLimit", res)
	g.Require().Nil(err)
	g.Require().Equal(g.config.BlockGasLimit, value.Uint64())

	// NumChains.
	input, err = abiObject.Pack("numChains")
	g.Require().Nil(err)
	res, err = g.call(addr, input, big.NewInt(0))
	g.Require().Nil(err)
	err = abiObject.Unpack(&value, "numChains", res)
	g.Require().Nil(err)
	g.Require().Equal(g.config.NumChains, uint32(value.Uint64()))

	// LambdaBA.
	input, err = abiObject.Pack("lambdaBA")
	g.Require().Nil(err)
	res, err = g.call(addr, input, big.NewInt(0))
	g.Require().Nil(err)
	err = abiObject.Unpack(&value, "lambdaBA", res)
	g.Require().Nil(err)
	g.Require().Equal(g.config.LambdaBA, value.Uint64())

	// LambdaDKG.
	input, err = abiObject.Pack("lambdaDKG")
	g.Require().Nil(err)
	res, err = g.call(addr, input, big.NewInt(0))
	g.Require().Nil(err)
	err = abiObject.Unpack(&value, "lambdaDKG", res)
	g.Require().Nil(err)
	g.Require().Equal(g.config.LambdaDKG, value.Uint64())

	// K.
	input, err = abiObject.Pack("k")
	g.Require().Nil(err)
	res, err = g.call(addr, input, big.NewInt(0))
	g.Require().Nil(err)
	err = abiObject.Unpack(&value, "k", res)
	g.Require().Nil(err)
	g.Require().Equal(g.config.K, uint32(value.Uint64()))

	// PhiRatio.
	input, err = abiObject.Pack("phiRatio")
	g.Require().Nil(err)
	res, err = g.call(addr, input, big.NewInt(0))
	g.Require().Nil(err)
	err = abiObject.Unpack(&value, "phiRatio", res)
	g.Require().Nil(err)
	g.Require().Equal(g.config.PhiRatio, float32(value.Uint64())/phiRatioMultiplier)

	// NotarySetSize.
	input, err = abiObject.Pack("notarySetSize")
	g.Require().Nil(err)
	res, err = g.call(addr, input, big.NewInt(0))
	g.Require().Nil(err)
	err = abiObject.Unpack(&value, "notarySetSize", res)
	g.Require().Nil(err)
	g.Require().Equal(g.config.NotarySetSize, uint32(value.Uint64()))

	// DKGSetSize.
	input, err = abiObject.Pack("dkgSetSize")
	g.Require().Nil(err)
	res, err = g.call(addr, input, big.NewInt(0))
	g.Require().Nil(err)
	err = abiObject.Unpack(&value, "dkgSetSize", res)
	g.Require().Nil(err)
	g.Require().Equal(g.config.DKGSetSize, uint32(value.Uint64()))

	// RoundInterval.
	input, err = abiObject.Pack("roundInterval")
	g.Require().Nil(err)
	res, err = g.call(addr, input, big.NewInt(0))
	g.Require().Nil(err)
	err = abiObject.Unpack(&value, "roundInterval", res)
	g.Require().Nil(err)
	g.Require().Equal(g.config.RoundInterval, value.Uint64())

	// MinBlockInterval.
	input, err = abiObject.Pack("minBlockInterval")
	g.Require().Nil(err)
	res, err = g.call(addr, input, big.NewInt(0))
	g.Require().Nil(err)
	err = abiObject.Unpack(&value, "minBlockInterval", res)
	g.Require().Nil(err)
	g.Require().Equal(g.config.MinBlockInterval, value.Uint64())
}

func (g *GovernanceContractTestSuite) TestMiscVariableReading() {
	privKey, addr := g.newPrefundAccount()
	pk := crypto.FromECDSAPub(&privKey.PublicKey)

	// Stake.
	amount := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(5e4))
	input, err := abiObject.Pack("stake", pk, "Test1", "test1@dexon.org", "Taipei, Taiwan", "https://dexon.org")
	g.Require().Nil(err)
	_, err = g.call(addr, input, amount)
	g.Require().Nil(err)

	// 1st delegator delegate to 1st node.
	_, addrDelegator := g.newPrefundAccount()
	amount = new(big.Int).Mul(big.NewInt(1e18), big.NewInt(3e4))
	input, err = abiObject.Pack("delegate", addr)
	g.Require().Nil(err)
	_, err = g.call(addrDelegator, input, amount)
	g.Require().Nil(err)

	// 2st delegator delegate to 1st node.
	_, addrDelegator2 := g.newPrefundAccount()
	input, err = abiObject.Pack("delegate", addr)
	g.Require().Nil(err)
	_, err = g.call(addrDelegator2, input, amount)
	g.Require().Nil(err)

	input, err = abiObject.Pack("nodesLength")
	g.Require().Nil(err)
	res, err := g.call(addr, input, big.NewInt(0))
	g.Require().Nil(err)
	var value *big.Int
	err = abiObject.Unpack(&value, "nodesLength", res)
	g.Require().Nil(err)
	g.Require().Equal(1, int(value.Uint64()))

	input, err = abiObject.Pack("nodesOffset", addr)
	g.Require().Nil(err)
	res, err = g.call(addr, input, big.NewInt(0))
	g.Require().Nil(err)
	err = abiObject.Unpack(&value, "nodesOffset", res)
	g.Require().Nil(err)
	g.Require().Equal(0, int(value.Uint64()))

	input, err = abiObject.Pack("delegatorsLength", addr)
	g.Require().Nil(err)
	res, err = g.call(addr, input, big.NewInt(0))
	g.Require().Nil(err)
	err = abiObject.Unpack(&value, "delegatorsLength", res)
	g.Require().Nil(err)
	g.Require().Equal(3, int(value.Uint64()))

	input, err = abiObject.Pack("delegatorsOffset", addr, addrDelegator2)
	g.Require().Nil(err)
	res, err = g.call(addr, input, big.NewInt(0))
	g.Require().Nil(err)
	err = abiObject.Unpack(&value, "delegatorsOffset", res)
	g.Require().Nil(err)
	g.Require().Equal(2, int(value.Uint64()))
}

func TestGovernanceContract(t *testing.T) {
	suite.Run(t, new(GovernanceContractTestSuite))
}
