// Copyright 2019 The dexon-consensus Authors
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
	"strings"

	"github.com/dexon-foundation/dexon/accounts/abi"
	"github.com/dexon-foundation/dexon/common"
)

var GovernanceContractAddress = common.HexToAddress("63751838d6485578b23e8b051d40861ecc416794")
var NodeInfoOracleAddress = common.HexToAddress("58a7c88ad1f32e7252bebba54def98d3e7b3df11")

var GovernanceABI *OracleContractABI
var NodeInfoOracleABI *OracleContractABI

func init() {
	GovernanceABI = NewOracleContractABI(GovernanceABIJSON)
	NodeInfoOracleABI = NewOracleContractABI(NodeInfoOracleABIJSON)
}

// OracleContract represent special system contracts written in Go.
type OracleContract interface {
	Run(evm *EVM, input []byte, contract *Contract) (ret []byte, err error)
}

// A map representing available system oracle contracts.
var OracleContracts = map[common.Address]OracleContract{
	GovernanceContractAddress: &GovernanceContract{
		coreDKGUtils: &defaultCoreDKGUtils{},
	},
	NodeInfoOracleAddress: &NodeInfoOracleContract{},
}

// Run oracle contract.
func RunOracleContract(oracle OracleContract, evm *EVM, input []byte, contract *Contract) (ret []byte, err error) {
	return oracle.Run(evm, input, contract)
}

// OracleContractABI represents ABI information for a given contract.
type OracleContractABI struct {
	ABI         abi.ABI
	Name2Method map[string]abi.Method
	Sig2Method  map[string]abi.Method
	Events      map[string]abi.Event
}

// NewOracleContractABI parse the ABI.
func NewOracleContractABI(abiDefinition string) *OracleContractABI {
	abiObject, err := abi.JSON(strings.NewReader(abiDefinition))
	if err != nil {
		panic(err)
	}

	sig2Method := make(map[string]abi.Method)
	name2Method := make(map[string]abi.Method)

	for _, method := range abiObject.Methods {
		sig2Method[string(method.Id())] = method
		name2Method[method.Name] = method
	}

	events := make(map[string]abi.Event)
	for _, event := range abiObject.Events {
		events[event.Name] = event
	}

	return &OracleContractABI{
		ABI:         abiObject,
		Name2Method: name2Method,
		Sig2Method:  sig2Method,
		Events:      events,
	}
}
