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
	"math/big"
	"strings"

	"github.com/dexon-foundation/dexon/accounts/abi"
	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/core/types"
	"github.com/dexon-foundation/dexon/crypto"
	"github.com/dexon-foundation/dexon/params"
	"github.com/dexon-foundation/dexon/rlp"

	coreCommon "github.com/dexon-foundation/dexon-consensus/common"
	"github.com/dexon-foundation/dexon-consensus/core"
	coreCrypto "github.com/dexon-foundation/dexon-consensus/core/crypto"
	"github.com/dexon-foundation/dexon-consensus/core/crypto/ecdsa"
	coreTypes "github.com/dexon-foundation/dexon-consensus/core/types"
	dkgTypes "github.com/dexon-foundation/dexon-consensus/core/types/dkg"
)

var GovernanceContractAddress = common.HexToAddress("5765692d4e696e6720536f6e696320426f6a6965")

const GovernanceABIJSON = `
[
  {
    "constant": true,
    "inputs": [
      {
        "name": "",
        "type": "address"
      },
      {
        "name": "",
        "type": "address"
      }
    ],
    "name": "delegatorsOffset",
    "outputs": [
      {
        "name": "",
        "type": "int256"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "blockReward",
    "outputs": [
      {
        "name": "",
        "type": "uint256"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [
      {
        "name": "",
        "type": "uint256"
      },
      {
        "name": "",
        "type": "uint256"
      }
    ],
    "name": "dkgComplaints",
    "outputs": [
      {
        "name": "",
        "type": "bytes"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "notarySetSize",
    "outputs": [
      {
        "name": "",
        "type": "uint256"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "dkgSetSize",
    "outputs": [
      {
        "name": "",
        "type": "uint256"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [
      {
        "name": "",
        "type": "uint256"
      }
    ],
    "name": "nodes",
    "outputs": [
      {
        "name": "owner",
        "type": "address"
      },
      {
        "name": "publicKey",
        "type": "bytes"
      },
      {
        "name": "staked",
        "type": "uint256"
      },
      {
        "name": "name",
        "type": "string"
      },
      {
        "name": "email",
        "type": "string"
      },
      {
        "name": "location",
        "type": "string"
      },
      {
        "name": "url",
        "type": "string"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "lambdaBA",
    "outputs": [
      {
        "name": "",
        "type": "uint256"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "minStake",
    "outputs": [
      {
        "name": "",
        "type": "uint256"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [
      {
        "name": "",
        "type": "uint256"
      }
    ],
    "name": "crs",
    "outputs": [
      {
        "name": "",
        "type": "bytes32"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "phiRatio",
    "outputs": [
      {
        "name": "",
        "type": "uint256"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [
      {
        "name": "",
        "type": "address"
      },
      {
        "name": "",
        "type": "uint256"
      }
    ],
    "name": "delegators",
    "outputs": [
      {
        "name": "owner",
        "type": "address"
      },
      {
        "name": "value",
        "type": "uint256"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "blockGasLimit",
    "outputs": [
      {
        "name": "",
        "type": "uint256"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "roundInterval",
    "outputs": [
      {
        "name": "",
        "type": "uint256"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "owner",
    "outputs": [
      {
        "name": "",
        "type": "address"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [
      {
        "name": "",
        "type": "address"
      }
    ],
    "name": "nodesOffset",
    "outputs": [
      {
        "name": "",
        "type": "int256"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "lambdaDKG",
    "outputs": [
      {
        "name": "",
        "type": "uint256"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [
      {
        "name": "",
        "type": "uint256"
      }
    ],
    "name": "roundHeight",
    "outputs": [
      {
        "name": "",
        "type": "uint256"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "minBlockInterval",
    "outputs": [
      {
        "name": "",
        "type": "uint256"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "k",
    "outputs": [
      {
        "name": "",
        "type": "uint256"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [
      {
        "name": "",
        "type": "uint256"
      },
      {
        "name": "",
        "type": "uint256"
      }
    ],
    "name": "dkgMasterPublicKeys",
    "outputs": [
      {
        "name": "",
        "type": "bytes"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [
      {
        "name": "",
        "type": "uint256"
      },
      {
        "name": "",
        "type": "address"
      }
    ],
    "name": "dkgFinalizeds",
    "outputs": [
      {
        "name": "",
        "type": "bool"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "numChains",
    "outputs": [
      {
        "name": "",
        "type": "uint256"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [
      {
        "name": "",
        "type": "uint256"
      }
    ],
    "name": "dkgFinalizedsCount",
    "outputs": [
      {
        "name": "",
        "type": "uint256"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "anonymous": false,
    "inputs": [],
    "name": "ConfigurationChanged",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": true,
        "name": "Round",
        "type": "uint256"
      },
      {
        "indexed": false,
        "name": "CRS",
        "type": "bytes32"
      }
    ],
    "name": "CRSProposed",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": true,
        "name": "NodeAddress",
        "type": "address"
      }
    ],
    "name": "Staked",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": true,
        "name": "NodeAddress",
        "type": "address"
      }
    ],
    "name": "Unstaked",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": true,
        "name": "NodeAddress",
        "type": "address"
      },
      {
        "indexed": true,
        "name": "DelegatorAddress",
        "type": "address"
      },
      {
        "indexed": false,
        "name": "Amount",
        "type": "uint256"
      }
    ],
    "name": "Delegated",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": true,
        "name": "NodeAddress",
        "type": "address"
      },
      {
        "indexed": true,
        "name": "DelegatorAddress",
        "type": "address"
      }
    ],
    "name": "Undelegated",
    "type": "event"
  },
  {
    "constant": false,
    "inputs": [
      {
        "name": "newOwner",
        "type": "address"
      }
    ],
    "name": "transferOwnership",
    "outputs": [],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "constant": false,
    "inputs": [
      {
        "name": "MinStake",
        "type": "uint256"
      },
      {
        "name": "BlockReward",
        "type": "uint256"
      },
      {
        "name": "BlockGasLimit",
        "type": "uint256"
      },
      {
        "name": "NumChains",
        "type": "uint256"
      },
      {
        "name": "LambdaBA",
        "type": "uint256"
      },
      {
        "name": "LambdaDKG",
        "type": "uint256"
      },
      {
        "name": "K",
        "type": "uint256"
      },
      {
        "name": "PhiRatio",
        "type": "uint256"
      },
      {
        "name": "NotarySetSize",
        "type": "uint256"
      },
      {
        "name": "DKGSetSize",
        "type": "uint256"
      },
      {
        "name": "RoundInterval",
        "type": "uint256"
      },
      {
        "name": "MinBlockInterval",
        "type": "uint256"
      }
    ],
    "name": "updateConfiguration",
    "outputs": [],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [],
    "name": "nodesLength",
    "outputs": [
      {
        "name": "",
        "type": "uint256"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": true,
    "inputs": [
      {
        "name": "NodeAddress",
        "type": "address"
      }
    ],
    "name": "delegatorsLength",
    "outputs": [
      {
        "name": "",
        "type": "uint256"
      }
    ],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
  },
  {
    "constant": false,
    "inputs": [
      {
        "name": "Round",
        "type": "uint256"
      },
      {
        "name": "Height",
        "type": "uint256"
      }
    ],
    "name": "snapshotRound",
    "outputs": [],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "constant": false,
    "inputs": [
      {
        "name": "Round",
        "type": "uint256"
      },
      {
        "name": "SignedCRS",
        "type": "bytes"
      }
    ],
    "name": "proposeCRS",
    "outputs": [],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "constant": false,
    "inputs": [
      {
        "name": "Round",
        "type": "uint256"
      },
      {
        "name": "Complaint",
        "type": "bytes"
      }
    ],
    "name": "addDKGComplaint",
    "outputs": [],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "constant": false,
    "inputs": [
      {
        "name": "Round",
        "type": "uint256"
      },
      {
        "name": "PublicKey",
        "type": "bytes"
      }
    ],
    "name": "addDKGMasterPublicKey",
    "outputs": [],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "constant": false,
    "inputs": [
      {
        "name": "Round",
        "type": "uint256"
      },
      {
        "name": "Finalize",
        "type": "bytes"
      }
    ],
    "name": "addDKGFinalize",
    "outputs": [],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "constant": false,
    "inputs": [
      {
        "name": "PublicKey",
        "type": "bytes"
      },
      {
        "name": "Name",
        "type": "string"
      },
      {
        "name": "Email",
        "type": "string"
      },
      {
        "name": "Location",
        "type": "string"
      },
      {
        "name": "Url",
        "type": "string"
      }
    ],
    "name": "stake",
    "outputs": [],
    "payable": true,
    "stateMutability": "payable",
    "type": "function"
  },
  {
    "constant": false,
    "inputs": [],
    "name": "unstake",
    "outputs": [],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "constant": false,
    "inputs": [
      {
        "name": "NodeAddress",
        "type": "address"
      }
    ],
    "name": "delegate",
    "outputs": [],
    "payable": true,
    "stateMutability": "payable",
    "type": "function"
  },
  {
    "constant": false,
    "inputs": [
      {
        "name": "NodeAddress",
        "type": "address"
      }
    ],
    "name": "undelegate",
    "outputs": [],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
  }
]
`

var abiObject abi.ABI
var GovernanceContractName2Method map[string]abi.Method
var sig2Method map[string]abi.Method
var events map[string]abi.Event

func init() {
	var err error

	// Parse governance contract ABI.
	abiObject, err = abi.JSON(strings.NewReader(GovernanceABIJSON))
	if err != nil {
		panic(err)
	}

	sig2Method = make(map[string]abi.Method)
	GovernanceContractName2Method = make(map[string]abi.Method)

	// Construct dispatch table.
	for _, method := range abiObject.Methods {
		sig2Method[string(method.Id())] = method
		GovernanceContractName2Method[method.Name] = method
	}

	events = make(map[string]abi.Event)

	// Event cache.
	for _, event := range abiObject.Events {
		events[event.Name] = event
	}
}

// RunGovernanceContract executes governance contract.
func RunGovernanceContract(evm *EVM, input []byte, contract *Contract) (ret []byte, err error) {
	if len(input) < 4 {
		return nil, nil
	}

	// Parse input.
	method, exists := sig2Method[string(input[:4])]
	if !exists {
		return nil, errExecutionReverted
	}

	// Dispatch method call.
	g := newGovernanceContract(evm, contract)
	arguments := input[4:]

	switch method.Name {
	case "addDKGComplaint":
		args := struct {
			Round     *big.Int
			Complaint []byte
		}{}
		if err := method.Inputs.Unpack(&args, arguments); err != nil {
			return nil, errExecutionReverted
		}
		return g.addDKGComplaint(args.Round, args.Complaint)
	case "addDKGMasterPublicKey":
		args := struct {
			Round     *big.Int
			PublicKey []byte
		}{}
		if err := method.Inputs.Unpack(&args, arguments); err != nil {
			return nil, errExecutionReverted
		}
		return g.addDKGMasterPublicKey(args.Round, args.PublicKey)
	case "addDKGFinalize":
		args := struct {
			Round    *big.Int
			Finalize []byte
		}{}
		if err := method.Inputs.Unpack(&args, arguments); err != nil {
			return nil, errExecutionReverted
		}
		return g.addDKGFinalize(args.Round, args.Finalize)
	case "delegate":
		address := common.Address{}
		if err := method.Inputs.Unpack(&address, arguments); err != nil {
			return nil, errExecutionReverted
		}
		return g.delegate(address)
	case "delegatorsLength":
		address := common.Address{}
		if err := method.Inputs.Unpack(&address, arguments); err != nil {
			return nil, errExecutionReverted
		}
		res, err := method.Outputs.Pack(g.state.LenDelegators(address))
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "nodesLength":
		res, err := method.Outputs.Pack(g.state.LenNodes())
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "proposeCRS":
		args := struct {
			Round     *big.Int
			SignedCRS []byte
		}{}
		if err := method.Inputs.Unpack(&args, arguments); err != nil {
			return nil, errExecutionReverted
		}
		return g.proposeCRS(args.Round, args.SignedCRS)
	case "stake":
		args := struct {
			PublicKey []byte
			Name      string
			Email     string
			Location  string
			Url       string
		}{}
		if err := method.Inputs.Unpack(&args, arguments); err != nil {
			return nil, errExecutionReverted
		}
		return g.stake(args.PublicKey, args.Name, args.Email, args.Location, args.Url)
	case "snapshotRound":
		args := struct {
			Round  *big.Int
			Height *big.Int
		}{}
		if err := method.Inputs.Unpack(&args, arguments); err != nil {
			return nil, errExecutionReverted
		}
		return g.snapshotRound(args.Round, args.Height)
	case "transferOwnership":
		var newOwner common.Address
		if err := method.Inputs.Unpack(&newOwner, arguments); err != nil {
			return nil, errExecutionReverted
		}
		return g.transferOwnership(newOwner)
	case "undelegate":
		address := common.Address{}
		if err := method.Inputs.Unpack(&address, arguments); err != nil {
			return nil, errExecutionReverted
		}
		return g.undelegate(address)
	case "unstake":
		return g.unstake()
	case "updateConfiguration":
		var cfg rawConfigStruct
		if err := method.Inputs.Unpack(&cfg, arguments); err != nil {
			return nil, errExecutionReverted
		}
		return g.updateConfiguration(&cfg)

	// --------------------------------
	// Solidity auto generated methods.
	// --------------------------------

	case "blockGasLimit":
		res, err := method.Outputs.Pack(g.state.BlockGasLimit())
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "blockReward":
		res, err := method.Outputs.Pack(g.state.BlockReward())
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "crs":
		round := new(big.Int)
		if err := method.Inputs.Unpack(&round, arguments); err != nil {
			return nil, errExecutionReverted
		}
		res, err := method.Outputs.Pack(g.state.CRS(round))
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "delegators":
		nodeAddr, index := common.Address{}, new(big.Int)
		args := []interface{}{&nodeAddr, &index}
		if err := method.Inputs.Unpack(&args, arguments); err != nil {
			return nil, errExecutionReverted
		}
		delegator := g.state.Delegator(nodeAddr, index)
		res, err := method.Outputs.Pack(delegator.Owner, delegator.Value)
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "delegatorsOffset":
		nodeAddr, delegatorAddr := common.Address{}, common.Address{}
		args := []interface{}{&nodeAddr, &delegatorAddr}
		if err := method.Inputs.Unpack(&args, arguments); err != nil {
			return nil, errExecutionReverted
		}
		res, err := method.Outputs.Pack(g.state.DelegatorsOffset(nodeAddr, delegatorAddr))
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "dkgComplaints":
		round, index := new(big.Int), new(big.Int)
		args := []interface{}{&round, &index}
		if err := method.Inputs.Unpack(&args, arguments); err != nil {
			return nil, errExecutionReverted
		}
		complaints := g.state.DKGComplaints(round)
		if int(index.Uint64()) >= len(complaints) {
			return nil, errExecutionReverted
		}
		complaint := complaints[index.Uint64()]
		res, err := method.Outputs.Pack(complaint)
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "dkgFinalizeds":
		round, addr := new(big.Int), common.Address{}
		args := []interface{}{&round, &addr}
		if err := method.Inputs.Unpack(&args, arguments); err != nil {
			return nil, errExecutionReverted
		}
		finalized := g.state.DKGFinalized(round, addr)
		res, err := method.Outputs.Pack(finalized)
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "dkgFinalizedsCount":
		round := new(big.Int)
		if err := method.Inputs.Unpack(&round, arguments); err != nil {
			return nil, errExecutionReverted
		}
		count := g.state.DKGFinalizedsCount(round)
		res, err := method.Outputs.Pack(count)
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "dkgMasterPublicKeys":
		round, index := new(big.Int), new(big.Int)
		args := []interface{}{&round, &index}
		if err := method.Inputs.Unpack(&args, arguments); err != nil {
			return nil, errExecutionReverted
		}
		mpks := g.state.DKGMasterPublicKeys(round)
		if int(index.Uint64()) >= len(mpks) {
			return nil, errExecutionReverted
		}
		mpk := mpks[index.Uint64()]
		res, err := method.Outputs.Pack(mpk)
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "dkgSetSize":
		res, err := method.Outputs.Pack(g.state.DKGSetSize())
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "k":
		res, err := method.Outputs.Pack(g.state.K())
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "lambdaBA":
		res, err := method.Outputs.Pack(g.state.LambdaBA())
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "lambdaDKG":
		res, err := method.Outputs.Pack(g.state.LambdaDKG())
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "minBlockInterval":
		res, err := method.Outputs.Pack(g.state.MinBlockInterval())
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "minStake":
		res, err := method.Outputs.Pack(g.state.MinStake())
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "numChains":
		res, err := method.Outputs.Pack(g.state.NumChains())
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "nodes":
		index := new(big.Int)
		if err := method.Inputs.Unpack(&index, arguments); err != nil {
			return nil, errExecutionReverted
		}
		info := g.state.Node(index)
		res, err := method.Outputs.Pack(
			info.Owner, info.PublicKey, info.Staked,
			info.Name, info.Email, info.Location, info.Url)
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "nodesOffset":
		address := common.Address{}
		if err := method.Inputs.Unpack(&address, arguments); err != nil {
			return nil, errExecutionReverted
		}
		res, err := method.Outputs.Pack(g.state.NodesOffset(address))
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "notarySetSize":
		res, err := method.Outputs.Pack(g.state.NotarySetSize())
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "owner":
		res, err := method.Outputs.Pack(g.state.Owner())
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "phiRatio":
		res, err := method.Outputs.Pack(g.state.PhiRatio())
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "roundHeight":
		round := new(big.Int)
		if err := method.Inputs.Unpack(&round, arguments); err != nil {
			return nil, errExecutionReverted
		}
		res, err := method.Outputs.Pack(g.state.RoundHeight(round))
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "roundInterval":
		res, err := method.Outputs.Pack(g.state.RoundInterval())
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	}
	return nil, nil
}

// Storage position enums.
const (
	roundHeightLoc = iota
	nodesLoc
	nodesOffsetLoc
	delegatorsLoc
	delegatorsOffsetLoc
	crsLoc
	dkgMasterPublicKeysLoc
	dkgComplaintsLoc
	dkgFinalizedLoc
	dkgFinalizedsCountLoc
	ownerLoc
	minStakeLoc
	blockRewardLoc
	blockGasLimitLoc
	numChainsLoc
	lambdaBALoc
	lambdaDKGLoc
	kLoc
	phiRatioLoc
	notarySetSizeLoc
	dkgSetSizeLoc
	roundIntervalLoc
	minBlockIntervalLoc
)

// State manipulation helper fro the governance contract.
type GovernanceStateHelper struct {
	StateDB StateDB
}

func (s *GovernanceStateHelper) getState(loc common.Hash) common.Hash {
	return s.StateDB.GetState(GovernanceContractAddress, loc)
}

func (s *GovernanceStateHelper) setState(loc common.Hash, val common.Hash) {
	s.StateDB.SetState(GovernanceContractAddress, loc, val)
}

func (s *GovernanceStateHelper) getStateBigInt(loc *big.Int) *big.Int {
	res := s.StateDB.GetState(GovernanceContractAddress, common.BigToHash(loc))
	return new(big.Int).SetBytes(res.Bytes())
}

func (s *GovernanceStateHelper) setStateBigInt(loc *big.Int, val *big.Int) {
	s.setState(common.BigToHash(loc), common.BigToHash(val))
}

func (s *GovernanceStateHelper) getSlotLoc(loc *big.Int) *big.Int {
	return new(big.Int).SetBytes(crypto.Keccak256(common.BigToHash(loc).Bytes()))
}

func (s *GovernanceStateHelper) getMapLoc(pos *big.Int, key []byte) *big.Int {
	return new(big.Int).SetBytes(crypto.Keccak256(key, common.BigToHash(pos).Bytes()))
}

func (s *GovernanceStateHelper) readBytes(loc *big.Int) []byte {
	// Length of the dynamic array (bytes).
	rawLength := s.getStateBigInt(loc)
	lengthByte := new(big.Int).Mod(rawLength, big.NewInt(256))

	// Bytes length <= 31, lengthByte % 2 == 0
	// return the high 31 bytes.
	if new(big.Int).Mod(lengthByte, big.NewInt(2)).Cmp(big.NewInt(0)) == 0 {
		length := new(big.Int).Div(lengthByte, big.NewInt(2)).Uint64()
		return rawLength.Bytes()[:length]
	}

	// Actual length = (rawLength - 1) / 2
	length := new(big.Int).Div(new(big.Int).Sub(rawLength, big.NewInt(1)), big.NewInt(2)).Uint64()

	// Data address.
	dataLoc := s.getSlotLoc(loc)

	// Read continuously for length bytes.
	carry := int64(0)
	if length%32 > 0 {
		carry = 1
	}
	chunks := int64(length/32) + carry
	var data []byte
	for i := int64(0); i < chunks; i++ {
		loc = new(big.Int).Add(dataLoc, big.NewInt(i))
		data = append(data, s.getState(common.BigToHash(loc)).Bytes()...)
	}
	data = data[:length]
	return data
}

func (s *GovernanceStateHelper) writeBytes(loc *big.Int, data []byte) {
	length := int64(len(data))

	if length == 0 {
		s.setState(common.BigToHash(loc), common.Hash{})
		return
	}

	// Short bytes (length <= 31).
	if length < 32 {
		data2 := append([]byte(nil), data...)
		// Right pad with zeros
		for len(data2) < 31 {
			data2 = append(data2, byte(0))
		}
		data2 = append(data2, byte(length*2))
		s.setState(common.BigToHash(loc), common.BytesToHash(data2))
		return
	}

	// Write 2 * length + 1.
	storedLength := new(big.Int).Add(new(big.Int).Mul(
		big.NewInt(length), big.NewInt(2)), big.NewInt(1))
	s.setStateBigInt(loc, storedLength)
	// Write data chunck.
	dataLoc := s.getSlotLoc(loc)
	carry := int64(0)
	if length%32 > 0 {
		carry = 1
	}
	chunks := length/32 + carry
	for i := int64(0); i < chunks; i++ {
		loc = new(big.Int).Add(dataLoc, big.NewInt(i))
		maxLoc := (i + 1) * 32
		if maxLoc > length {
			maxLoc = length
		}
		data2 := data[i*32 : maxLoc]
		// Right pad with zeros.
		for len(data2) < 32 {
			data2 = append(data2, byte(0))
		}
		s.setState(common.BigToHash(loc), common.BytesToHash(data2))
	}
}

func (s *GovernanceStateHelper) read2DByteArray(pos, index *big.Int) [][]byte {
	baseLoc := s.getSlotLoc(pos)
	loc := new(big.Int).Add(baseLoc, index)

	arrayLength := s.getStateBigInt(loc)
	dataLoc := s.getSlotLoc(loc)

	data := [][]byte{}
	for i := int64(0); i < int64(arrayLength.Uint64()); i++ {
		elementLoc := new(big.Int).Add(dataLoc, big.NewInt(i))
		data = append(data, s.readBytes(elementLoc))
	}

	return data
}
func (s *GovernanceStateHelper) appendTo2DByteArray(pos, index *big.Int, data []byte) {
	// Find the loc of the last element.
	baseLoc := s.getSlotLoc(pos)
	loc := new(big.Int).Add(baseLoc, index)

	// Increase length by 1.
	arrayLength := s.getStateBigInt(loc)
	s.setStateBigInt(loc, new(big.Int).Add(arrayLength, big.NewInt(1)))

	// Write element.
	dataLoc := s.getSlotLoc(loc)
	elementLoc := new(big.Int).Add(dataLoc, arrayLength)
	s.writeBytes(elementLoc, data)
}

// uint256[] public roundHeight;
func (s *GovernanceStateHelper) LenRoundHeight() *big.Int {
	return s.getStateBigInt(big.NewInt(roundHeightLoc))
}
func (s *GovernanceStateHelper) RoundHeight(round *big.Int) *big.Int {
	baseLoc := s.getSlotLoc(big.NewInt(roundHeightLoc))
	loc := new(big.Int).Add(baseLoc, round)
	return s.getStateBigInt(loc)
}
func (s *GovernanceStateHelper) PushRoundHeight(height *big.Int) {
	// Increase length by 1.
	length := s.getStateBigInt(big.NewInt(roundHeightLoc))
	s.setStateBigInt(big.NewInt(roundHeightLoc), new(big.Int).Add(length, big.NewInt(1)))

	baseLoc := s.getSlotLoc(big.NewInt(roundHeightLoc))
	loc := new(big.Int).Add(baseLoc, length)

	s.setStateBigInt(loc, height)
}

// struct Node {
//     address owner;
//     bytes publicKey;
//     uint256 staked;
//     string name;
//     string email;
//     string location;
//     string url;
// }
//
// Node[] nodes;

type nodeInfo struct {
	Owner     common.Address
	PublicKey []byte
	Staked    *big.Int
	Name      string
	Email     string
	Location  string
	Url       string
}

const nodeStructSize = 7

func (s *GovernanceStateHelper) LenNodes() *big.Int {
	return s.getStateBigInt(big.NewInt(nodesLoc))
}
func (s *GovernanceStateHelper) Node(index *big.Int) *nodeInfo {
	node := new(nodeInfo)

	arrayBaseLoc := s.getSlotLoc(big.NewInt(nodesLoc))
	elementBaseLoc := new(big.Int).Add(arrayBaseLoc,
		new(big.Int).Mul(index, big.NewInt(nodeStructSize)))

	// Owner.
	loc := elementBaseLoc
	node.Owner = common.BytesToAddress(s.getState(common.BigToHash(elementBaseLoc)).Bytes())

	// PublicKey.
	loc = new(big.Int).Add(elementBaseLoc, big.NewInt(1))
	node.PublicKey = s.readBytes(loc)

	// Staked.
	loc = new(big.Int).Add(elementBaseLoc, big.NewInt(2))
	node.Staked = s.getStateBigInt(loc)

	// Name.
	loc = new(big.Int).Add(elementBaseLoc, big.NewInt(3))
	node.Name = string(s.readBytes(loc))

	// Email.
	loc = new(big.Int).Add(elementBaseLoc, big.NewInt(4))
	node.Email = string(s.readBytes(loc))

	// Location.
	loc = new(big.Int).Add(elementBaseLoc, big.NewInt(5))
	node.Location = string(s.readBytes(loc))

	// Url.
	loc = new(big.Int).Add(elementBaseLoc, big.NewInt(6))
	node.Url = string(s.readBytes(loc))

	return node
}
func (s *GovernanceStateHelper) PushNode(n *nodeInfo) {
	// Increase length by 1.
	arrayLength := s.LenNodes()
	s.setStateBigInt(big.NewInt(nodesLoc), new(big.Int).Add(arrayLength, big.NewInt(1)))

	s.UpdateNode(arrayLength, n)
}
func (s *GovernanceStateHelper) UpdateNode(index *big.Int, n *nodeInfo) {
	arrayBaseLoc := s.getSlotLoc(big.NewInt(nodesLoc))
	elementBaseLoc := new(big.Int).Add(arrayBaseLoc, new(big.Int).Mul(index, big.NewInt(nodeStructSize)))

	// Owner.
	loc := elementBaseLoc
	s.setState(common.BigToHash(loc), n.Owner.Hash())

	// PublicKey.
	loc = new(big.Int).Add(elementBaseLoc, big.NewInt(1))
	s.writeBytes(loc, n.PublicKey)

	// Staked.
	loc = new(big.Int).Add(elementBaseLoc, big.NewInt(2))
	s.setStateBigInt(loc, n.Staked)

	// Name.
	loc = new(big.Int).Add(elementBaseLoc, big.NewInt(3))
	s.writeBytes(loc, []byte(n.Name))

	// Email.
	loc = new(big.Int).Add(elementBaseLoc, big.NewInt(4))
	s.writeBytes(loc, []byte(n.Email))

	// Location.
	loc = new(big.Int).Add(elementBaseLoc, big.NewInt(5))
	s.writeBytes(loc, []byte(n.Location))

	// Url.
	loc = new(big.Int).Add(elementBaseLoc, big.NewInt(6))
	s.writeBytes(loc, []byte(n.Url))
}
func (s *GovernanceStateHelper) PopLastNode() {
	// Decrease length by 1.
	arrayLength := s.LenNodes()
	newArrayLength := new(big.Int).Sub(arrayLength, big.NewInt(1))
	s.setStateBigInt(big.NewInt(nodesLoc), newArrayLength)

	s.UpdateNode(newArrayLength, &nodeInfo{Staked: big.NewInt(0)})
}
func (s *GovernanceStateHelper) Nodes() []*nodeInfo {
	var nodes []*nodeInfo
	for i := int64(0); i < int64(s.LenNodes().Uint64()); i++ {
		nodes = append(nodes, s.Node(big.NewInt(i)))
	}
	return nodes
}
func (s *GovernanceStateHelper) QualifiedNodes() []*nodeInfo {
	var nodes []*nodeInfo
	for i := int64(0); i < int64(s.LenNodes().Uint64()); i++ {
		node := s.Node(big.NewInt(i))
		if node.Staked.Cmp(s.MinStake()) >= 0 {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// mapping(address => uint256) public nodeOffset;
func (s *GovernanceStateHelper) NodesOffset(addr common.Address) *big.Int {
	loc := s.getMapLoc(big.NewInt(nodesOffsetLoc), addr.Bytes())
	return new(big.Int).Sub(s.getStateBigInt(loc), big.NewInt(1))
}
func (s *GovernanceStateHelper) PutNodesOffset(addr common.Address, offset *big.Int) {
	loc := s.getMapLoc(big.NewInt(nodesOffsetLoc), addr.Bytes())
	s.setStateBigInt(loc, new(big.Int).Add(offset, big.NewInt(1)))
}
func (s *GovernanceStateHelper) DeleteNodesOffset(addr common.Address) {
	loc := s.getMapLoc(big.NewInt(nodesOffsetLoc), addr.Bytes())
	s.setStateBigInt(loc, big.NewInt(0))
}

// struct Delegator {
//     address node;
//     address owner;
//     uint256 value;
// }

type delegatorInfo struct {
	Owner common.Address
	Value *big.Int
}

const delegatorStructSize = 2

// mapping(address => Delegator[]) public delegators;
func (s *GovernanceStateHelper) LenDelegators(nodeAddr common.Address) *big.Int {
	loc := s.getMapLoc(big.NewInt(delegatorsLoc), nodeAddr.Bytes())
	return s.getStateBigInt(loc)
}
func (s *GovernanceStateHelper) Delegator(nodeAddr common.Address, offset *big.Int) *delegatorInfo {
	delegator := new(delegatorInfo)

	loc := s.getMapLoc(big.NewInt(delegatorsLoc), nodeAddr.Bytes())
	arrayBaseLoc := s.getSlotLoc(loc)
	elementBaseLoc := new(big.Int).Add(arrayBaseLoc, new(big.Int).Mul(big.NewInt(delegatorStructSize), offset))

	// Owner.
	loc = elementBaseLoc
	delegator.Owner = common.BytesToAddress(s.getState(common.BigToHash(elementBaseLoc)).Bytes())

	// Value.
	loc = new(big.Int).Add(elementBaseLoc, big.NewInt(1))
	delegator.Value = s.getStateBigInt(loc)

	return delegator
}
func (s *GovernanceStateHelper) PushDelegator(nodeAddr common.Address, delegator *delegatorInfo) {
	// Increase length by 1.
	arrayLength := s.LenDelegators(nodeAddr)
	loc := s.getMapLoc(big.NewInt(delegatorsLoc), nodeAddr.Bytes())
	s.setStateBigInt(loc, new(big.Int).Add(arrayLength, big.NewInt(1)))

	s.UpdateDelegator(nodeAddr, arrayLength, delegator)
}
func (s *GovernanceStateHelper) UpdateDelegator(nodeAddr common.Address, offset *big.Int, delegator *delegatorInfo) {
	loc := s.getMapLoc(big.NewInt(delegatorsLoc), nodeAddr.Bytes())
	arrayBaseLoc := s.getSlotLoc(loc)
	elementBaseLoc := new(big.Int).Add(arrayBaseLoc, new(big.Int).Mul(big.NewInt(delegatorStructSize), offset))

	// Owner.
	loc = elementBaseLoc
	s.setState(common.BigToHash(loc), delegator.Owner.Hash())

	// Value.
	loc = new(big.Int).Add(elementBaseLoc, big.NewInt(1))
	s.setStateBigInt(loc, delegator.Value)
}
func (s *GovernanceStateHelper) PopLastDelegator(nodeAddr common.Address) {
	// Decrease length by 1.
	arrayLength := s.LenDelegators(nodeAddr)
	newArrayLength := new(big.Int).Sub(arrayLength, big.NewInt(1))
	loc := s.getMapLoc(big.NewInt(delegatorsLoc), nodeAddr.Bytes())
	s.setStateBigInt(loc, newArrayLength)

	s.UpdateDelegator(nodeAddr, newArrayLength, &delegatorInfo{Value: big.NewInt(0)})
}

// mapping(address => mapping(address => uint256)) delegatorsOffset;
func (s *GovernanceStateHelper) DelegatorsOffset(nodeAddr, delegatorAddr common.Address) *big.Int {
	loc := s.getMapLoc(s.getMapLoc(big.NewInt(delegatorsOffsetLoc), nodeAddr.Bytes()), delegatorAddr.Bytes())
	return new(big.Int).Sub(s.getStateBigInt(loc), big.NewInt(1))
}
func (s *GovernanceStateHelper) PutDelegatorOffset(nodeAddr, delegatorAddr common.Address, offset *big.Int) {
	loc := s.getMapLoc(s.getMapLoc(big.NewInt(delegatorsOffsetLoc), nodeAddr.Bytes()), delegatorAddr.Bytes())
	s.setStateBigInt(loc, new(big.Int).Add(offset, big.NewInt(1)))
}
func (s *GovernanceStateHelper) DeleteDelegatorsOffset(nodeAddr, delegatorAddr common.Address) {
	loc := s.getMapLoc(s.getMapLoc(big.NewInt(delegatorsOffsetLoc), nodeAddr.Bytes()), delegatorAddr.Bytes())
	s.setStateBigInt(loc, big.NewInt(0))
}

// bytes32[] public crs;
func (s *GovernanceStateHelper) LenCRS() *big.Int {
	return s.getStateBigInt(big.NewInt(crsLoc))
}
func (s *GovernanceStateHelper) CRS(index *big.Int) common.Hash {
	baseLoc := s.getSlotLoc(big.NewInt(crsLoc))
	loc := new(big.Int).Add(baseLoc, index)
	return s.getState(common.BigToHash(loc))
}
func (s *GovernanceStateHelper) CurrentCRS() common.Hash {
	return s.CRS(new(big.Int).Sub(s.LenCRS(), big.NewInt(1)))
}
func (s *GovernanceStateHelper) PushCRS(crs common.Hash) {
	// increase length by 1.
	length := s.getStateBigInt(big.NewInt(crsLoc))
	s.setStateBigInt(big.NewInt(crsLoc), new(big.Int).Add(length, big.NewInt(1)))

	baseLoc := s.getSlotLoc(big.NewInt(crsLoc))
	loc := new(big.Int).Add(baseLoc, length)

	s.setState(common.BigToHash(loc), crs)
}
func (s *GovernanceStateHelper) Round() *big.Int {
	return new(big.Int).Sub(s.getStateBigInt(big.NewInt(crsLoc)), big.NewInt(1))
}

// bytes[][] public dkgMasterPublicKeys;
func (s *GovernanceStateHelper) DKGMasterPublicKeys(round *big.Int) [][]byte {
	return s.read2DByteArray(big.NewInt(dkgMasterPublicKeysLoc), round)
}
func (s *GovernanceStateHelper) PushDKGMasterPublicKey(round *big.Int, mpk []byte) {
	s.appendTo2DByteArray(big.NewInt(dkgMasterPublicKeysLoc), round, mpk)
}

// bytes[][] public dkgComplaints;
func (s *GovernanceStateHelper) DKGComplaints(round *big.Int) [][]byte {
	return s.read2DByteArray(big.NewInt(dkgComplaintsLoc), round)
}
func (s *GovernanceStateHelper) PushDKGComplaint(round *big.Int, complaint []byte) {
	s.appendTo2DByteArray(big.NewInt(dkgComplaintsLoc), round, complaint)
}

// mapping(address => bool)[] public dkgFinalized;
func (s *GovernanceStateHelper) DKGFinalized(round *big.Int, addr common.Address) bool {
	baseLoc := new(big.Int).Add(s.getSlotLoc(big.NewInt(dkgFinalizedLoc)), round)
	mapLoc := s.getMapLoc(baseLoc, addr.Bytes())
	return s.getStateBigInt(mapLoc).Cmp(big.NewInt(0)) != 0
}
func (s *GovernanceStateHelper) PutDKGFinalized(round *big.Int, addr common.Address, finalized bool) {
	baseLoc := new(big.Int).Add(s.getSlotLoc(big.NewInt(dkgFinalizedLoc)), round)
	mapLoc := s.getMapLoc(baseLoc, addr.Bytes())
	res := big.NewInt(0)
	if finalized {
		res = big.NewInt(1)
	}
	s.setStateBigInt(mapLoc, res)
}

// uint256[] public dkgFinalizedsCount;
func (s *GovernanceStateHelper) DKGFinalizedsCount(round *big.Int) *big.Int {
	loc := new(big.Int).Add(s.getSlotLoc(big.NewInt(dkgFinalizedsCountLoc)), round)
	return s.getStateBigInt(loc)
}
func (s *GovernanceStateHelper) IncDKGFinalizedsCount(round *big.Int) {
	loc := new(big.Int).Add(s.getSlotLoc(big.NewInt(dkgFinalizedsCountLoc)), round)
	count := s.getStateBigInt(loc)
	s.setStateBigInt(loc, new(big.Int).Add(count, big.NewInt(1)))
}

// address public owner;
func (s *GovernanceStateHelper) Owner() common.Address {
	val := s.getState(common.BigToHash(big.NewInt(ownerLoc)))
	return common.BytesToAddress(val.Bytes())
}
func (s *GovernanceStateHelper) SetOwner(newOwner common.Address) {
	s.setState(common.BigToHash(big.NewInt(ownerLoc)), newOwner.Hash())
}

// uint256 public minStake;
func (s *GovernanceStateHelper) MinStake() *big.Int {
	return s.getStateBigInt(big.NewInt(minStakeLoc))
}
func (s *GovernanceStateHelper) SetMinStake(stake *big.Int) {
	s.setStateBigInt(big.NewInt(minStakeLoc), stake)
}

// uint256 public blockReward;
func (s *GovernanceStateHelper) BlockReward() *big.Int {
	return s.getStateBigInt(big.NewInt(blockRewardLoc))
}
func (s *GovernanceStateHelper) SetBlockReward(reward *big.Int) {
	s.setStateBigInt(big.NewInt(blockRewardLoc), reward)
}

// uint256 public blockGasLimit;
func (s *GovernanceStateHelper) BlockGasLimit() *big.Int {
	return s.getStateBigInt(big.NewInt(blockGasLimitLoc))
}
func (s *GovernanceStateHelper) SetBlockGasLimit(reward *big.Int) {
	s.setStateBigInt(big.NewInt(blockGasLimitLoc), reward)
}

// uint256 public numChains;
func (s *GovernanceStateHelper) NumChains() *big.Int {
	return s.getStateBigInt(big.NewInt(numChainsLoc))
}

// uint256 public lambdaBA;
func (s *GovernanceStateHelper) LambdaBA() *big.Int {
	return s.getStateBigInt(big.NewInt(lambdaBALoc))
}

// uint256 public lambdaDKG;
func (s *GovernanceStateHelper) LambdaDKG() *big.Int {
	return s.getStateBigInt(big.NewInt(lambdaDKGLoc))
}

// uint256 public k;
func (s *GovernanceStateHelper) K() *big.Int {
	return s.getStateBigInt(big.NewInt(kLoc))
}

// uint256 public phiRatio;  // stored as PhiRatio * 10^6
func (s *GovernanceStateHelper) PhiRatio() *big.Int {
	return s.getStateBigInt(big.NewInt(phiRatioLoc))
}

// uint256 public notarySetSize;
func (s *GovernanceStateHelper) NotarySetSize() *big.Int {
	return s.getStateBigInt(big.NewInt(notarySetSizeLoc))
}

// uint256 public dkgSetSize;
func (s *GovernanceStateHelper) DKGSetSize() *big.Int {
	return s.getStateBigInt(big.NewInt(dkgSetSizeLoc))
}

// uint256 public roundInterval;
func (s *GovernanceStateHelper) RoundInterval() *big.Int {
	return s.getStateBigInt(big.NewInt(roundIntervalLoc))
}

// uint256 public minBlockInterval;
func (s *GovernanceStateHelper) MinBlockInterval() *big.Int {
	return s.getStateBigInt(big.NewInt(minBlockIntervalLoc))
}

// Stake is a helper function for creating genesis state.
func (s *GovernanceStateHelper) Stake(
	addr common.Address, publicKey []byte, staked *big.Int,
	name, email, location, url string) {
	offset := s.LenNodes()
	s.PushNode(&nodeInfo{
		Owner:     addr,
		PublicKey: publicKey,
		Staked:    staked,
		Name:      name,
		Email:     email,
		Location:  location,
		Url:       url,
	})
	s.PutNodesOffset(addr, offset)
}

const phiRatioMultiplier = 1000000.0

// Configuration returns the current configuration.
func (s *GovernanceStateHelper) Configuration() *params.DexconConfig {
	return &params.DexconConfig{
		MinStake:         s.getStateBigInt(big.NewInt(minStakeLoc)),
		BlockReward:      s.getStateBigInt(big.NewInt(blockRewardLoc)),
		BlockGasLimit:    s.getStateBigInt(big.NewInt(blockGasLimitLoc)).Uint64(),
		NumChains:        uint32(s.getStateBigInt(big.NewInt(numChainsLoc)).Uint64()),
		LambdaBA:         s.getStateBigInt(big.NewInt(lambdaBALoc)).Uint64(),
		LambdaDKG:        s.getStateBigInt(big.NewInt(lambdaDKGLoc)).Uint64(),
		K:                uint32(s.getStateBigInt(big.NewInt(kLoc)).Uint64()),
		PhiRatio:         float32(s.getStateBigInt(big.NewInt(phiRatioLoc)).Uint64()) / phiRatioMultiplier,
		NotarySetSize:    uint32(s.getStateBigInt(big.NewInt(notarySetSizeLoc)).Uint64()),
		DKGSetSize:       uint32(s.getStateBigInt(big.NewInt(dkgSetSizeLoc)).Uint64()),
		RoundInterval:    s.getStateBigInt(big.NewInt(roundIntervalLoc)).Uint64(),
		MinBlockInterval: s.getStateBigInt(big.NewInt(minBlockIntervalLoc)).Uint64(),
	}
}

// UpdateConfiguration updates system configuration.
func (s *GovernanceStateHelper) UpdateConfiguration(cfg *params.DexconConfig) {
	s.setStateBigInt(big.NewInt(minStakeLoc), cfg.MinStake)
	s.setStateBigInt(big.NewInt(blockRewardLoc), cfg.BlockReward)
	s.setStateBigInt(big.NewInt(blockGasLimitLoc), big.NewInt(int64(cfg.BlockGasLimit)))
	s.setStateBigInt(big.NewInt(numChainsLoc), big.NewInt(int64(cfg.NumChains)))
	s.setStateBigInt(big.NewInt(lambdaBALoc), big.NewInt(int64(cfg.LambdaBA)))
	s.setStateBigInt(big.NewInt(lambdaDKGLoc), big.NewInt(int64(cfg.LambdaDKG)))
	s.setStateBigInt(big.NewInt(kLoc), big.NewInt(int64(cfg.K)))
	s.setStateBigInt(big.NewInt(phiRatioLoc), big.NewInt(int64(cfg.PhiRatio*phiRatioMultiplier)))
	s.setStateBigInt(big.NewInt(notarySetSizeLoc), big.NewInt(int64(cfg.NotarySetSize)))
	s.setStateBigInt(big.NewInt(dkgSetSizeLoc), big.NewInt(int64(cfg.DKGSetSize)))
	s.setStateBigInt(big.NewInt(roundIntervalLoc), big.NewInt(int64(cfg.RoundInterval)))
	s.setStateBigInt(big.NewInt(minBlockIntervalLoc), big.NewInt(int64(cfg.MinBlockInterval)))
}

type rawConfigStruct struct {
	MinStake         *big.Int
	BlockReward      *big.Int
	BlockGasLimit    *big.Int
	NumChains        *big.Int
	LambdaBA         *big.Int
	LambdaDKG        *big.Int
	K                *big.Int
	PhiRatio         *big.Int
	NotarySetSize    *big.Int
	DKGSetSize       *big.Int
	RoundInterval    *big.Int
	MinBlockInterval *big.Int
}

// UpdateConfigurationRaw updates system configuration.
func (s *GovernanceStateHelper) UpdateConfigurationRaw(cfg *rawConfigStruct) {
	s.setStateBigInt(big.NewInt(minStakeLoc), cfg.MinStake)
	s.setStateBigInt(big.NewInt(blockRewardLoc), cfg.BlockReward)
	s.setStateBigInt(big.NewInt(blockGasLimitLoc), cfg.BlockGasLimit)
	s.setStateBigInt(big.NewInt(numChainsLoc), cfg.NumChains)
	s.setStateBigInt(big.NewInt(lambdaBALoc), cfg.LambdaBA)
	s.setStateBigInt(big.NewInt(lambdaDKGLoc), cfg.LambdaDKG)
	s.setStateBigInt(big.NewInt(kLoc), cfg.K)
	s.setStateBigInt(big.NewInt(phiRatioLoc), cfg.PhiRatio)
	s.setStateBigInt(big.NewInt(notarySetSizeLoc), cfg.NotarySetSize)
	s.setStateBigInt(big.NewInt(dkgSetSizeLoc), cfg.DKGSetSize)
	s.setStateBigInt(big.NewInt(roundIntervalLoc), cfg.RoundInterval)
	s.setStateBigInt(big.NewInt(minBlockIntervalLoc), cfg.MinBlockInterval)
}

// event ConfigurationChanged();
func (s *GovernanceStateHelper) emitConfigurationChangedEvent() {
	s.StateDB.AddLog(&types.Log{
		Address: GovernanceContractAddress,
		Topics:  []common.Hash{events["ConfigurationChanged"].Id()},
		Data:    []byte{},
	})
}

// event CRSProposed(uint256 round, bytes32 crs);
func (s *GovernanceStateHelper) emitCRSProposed(round *big.Int, crs common.Hash) {
	s.StateDB.AddLog(&types.Log{
		Address: GovernanceContractAddress,
		Topics:  []common.Hash{events["CRSProposed"].Id(), common.BigToHash(round)},
		Data:    crs.Bytes(),
	})
}

// event Staked(address indexed NodeAddress, uint256 Amount);
func (s *GovernanceStateHelper) emitStaked(nodeAddr common.Address) {
	s.StateDB.AddLog(&types.Log{
		Address: GovernanceContractAddress,
		Topics:  []common.Hash{events["Staked"].Id(), nodeAddr.Hash()},
		Data:    []byte{},
	})
}

// event Unstaked(address indexed NodeAddress);
func (s *GovernanceStateHelper) emitUnstaked(nodeAddr common.Address) {
	s.StateDB.AddLog(&types.Log{
		Address: GovernanceContractAddress,
		Topics:  []common.Hash{events["Unstaked"].Id(), nodeAddr.Hash()},
		Data:    []byte{},
	})
}

// event Delegated(address indexed NodeAddress, address indexed DelegatorAddress, uint256 Amount);
func (s *GovernanceStateHelper) emitDelegated(nodeAddr, delegatorAddr common.Address, amount *big.Int) {
	s.StateDB.AddLog(&types.Log{
		Address: GovernanceContractAddress,
		Topics:  []common.Hash{events["Delegated"].Id(), nodeAddr.Hash(), delegatorAddr.Hash()},
		Data:    common.BigToHash(amount).Bytes(),
	})
}

// event Undelegated(address indexed NodeAddress, address indexed DelegatorAddress);
func (s *GovernanceStateHelper) emitUndelegated(nodeAddr, delegatorAddr common.Address) {
	s.StateDB.AddLog(&types.Log{
		Address: GovernanceContractAddress,
		Topics:  []common.Hash{events["Undelegated"].Id(), nodeAddr.Hash(), delegatorAddr.Hash()},
		Data:    []byte{},
	})
}

// GovernanceContract represents the governance contract of DEXCON.
type GovernanceContract struct {
	evm      *EVM
	state    GovernanceStateHelper
	contract *Contract
}

func newGovernanceContract(evm *EVM, contract *Contract) *GovernanceContract {
	return &GovernanceContract{
		evm:      evm,
		state:    GovernanceStateHelper{evm.StateDB},
		contract: contract,
	}
}

func (g *GovernanceContract) Address() common.Address {
	return GovernanceContractAddress
}

func (g *GovernanceContract) transfer(from, to common.Address, amount *big.Int) bool {
	// TODO(w): add this to debug trace so it shows up as internal transaction.
	if g.evm.CanTransfer(g.evm.StateDB, from, amount) {
		g.evm.Transfer(g.evm.StateDB, from, to, amount)
		return true
	}
	return false
}

func (g *GovernanceContract) useGas(gas uint64) ([]byte, error) {
	if !g.contract.UseGas(gas) {
		return nil, ErrOutOfGas
	}
	return nil, nil
}

func (g *GovernanceContract) penalize() ([]byte, error) {
	g.useGas(g.contract.Gas)
	return nil, errExecutionReverted
}

func (g *GovernanceContract) inDKGSet(round *big.Int, nodeID coreTypes.NodeID) bool {
	target := coreTypes.NewDKGSetTarget(coreCommon.Hash(g.state.CurrentCRS()))
	ns := coreTypes.NewNodeSet()

	configRound := big.NewInt(0) // If round < core.ConfigRoundShift, use 0.
	if round.Uint64() >= core.ConfigRoundShift {
		configRound = new(big.Int).Sub(round, big.NewInt(int64(core.ConfigRoundShift)))
	}

	statedb, err := g.evm.StateAtNumber(g.state.RoundHeight(configRound).Uint64())
	if err != nil {
		panic(err)
	}

	state := GovernanceStateHelper{statedb}
	for _, x := range state.QualifiedNodes() {
		mpk, err := ecdsa.NewPublicKeyFromByteSlice(x.PublicKey)
		if err != nil {
			panic(err)
		}
		ns.Add(coreTypes.NewNodeID(mpk))
	}

	dkgSet := ns.GetSubSet(int(g.state.DKGSetSize().Uint64()), target)
	_, ok := dkgSet[nodeID]
	return ok
}

func (g *GovernanceContract) addDKGComplaint(round *big.Int, comp []byte) ([]byte, error) {
	if round.Cmp(g.state.Round()) != 0 {
		return g.penalize()
	}

	caller := g.contract.Caller()

	// Finalized caller is not allowed to propose complaint.
	if g.state.DKGFinalized(round, caller) {
		return g.penalize()
	}

	// Calculate 2f
	threshold := new(big.Int).Mul(
		big.NewInt(2),
		new(big.Int).Div(g.state.DKGSetSize(), big.NewInt(3)))

	// If 2f + 1 of DKG set is finalized, one can not propose complaint anymore.
	if g.state.DKGFinalizedsCount(round).Cmp(threshold) > 0 {
		return nil, errExecutionReverted
	}

	var dkgComplaint dkgTypes.Complaint
	if err := rlp.DecodeBytes(comp, &dkgComplaint); err != nil {
		return g.penalize()
	}

	// DKGComplaint must belongs to someone in DKG set.
	if !g.inDKGSet(round, dkgComplaint.ProposerID) {
		return g.penalize()
	}

	verified, _ := core.VerifyDKGComplaintSignature(&dkgComplaint)
	if !verified {
		return g.penalize()
	}

	g.state.PushDKGComplaint(round, comp)

	// Set this to relatively high to prevent spamming
	return g.useGas(5000000)
}

func (g *GovernanceContract) addDKGMasterPublicKey(round *big.Int, mpk []byte) ([]byte, error) {
	// Can only add DKG master public key of current and next round.
	if round.Cmp(new(big.Int).Add(g.state.Round(), big.NewInt(1))) > 0 {
		return g.penalize()
	}

	caller := g.contract.Caller()
	offset := g.state.NodesOffset(caller)

	// Can not add dkg mpk if not staked.
	if offset.Cmp(big.NewInt(0)) < 0 {
		return nil, errExecutionReverted
	}

	var dkgMasterPK dkgTypes.MasterPublicKey
	if err := rlp.DecodeBytes(mpk, &dkgMasterPK); err != nil {
		return g.penalize()
	}

	// DKGMasterPublicKey must belongs to someone in DKG set.
	if !g.inDKGSet(round, dkgMasterPK.ProposerID) {
		return g.penalize()
	}

	verified, _ := core.VerifyDKGMasterPublicKeySignature(&dkgMasterPK)
	if !verified {
		return g.penalize()
	}

	g.state.PushDKGMasterPublicKey(round, mpk)

	return g.useGas(100000)
}

func (g *GovernanceContract) addDKGFinalize(round *big.Int, finalize []byte) ([]byte, error) {
	if round.Cmp(g.state.Round()) != 0 {
		return g.penalize()
	}

	caller := g.contract.Caller()

	var dkgFinalize dkgTypes.Finalize
	if err := rlp.DecodeBytes(finalize, &dkgFinalize); err != nil {
		return g.penalize()
	}

	// DKGFInalize must belongs to someone in DKG set.
	if !g.inDKGSet(round, dkgFinalize.ProposerID) {
		return g.penalize()
	}

	verified, _ := core.VerifyDKGFinalizeSignature(&dkgFinalize)
	if !verified {
		return g.penalize()
	}

	if !g.state.DKGFinalized(round, caller) {
		g.state.PutDKGFinalized(round, caller, true)
		g.state.IncDKGFinalizedsCount(round)
	}

	return g.useGas(100000)
}

func (g *GovernanceContract) delegate(nodeAddr common.Address) ([]byte, error) {
	offset := g.state.NodesOffset(nodeAddr)
	if offset.Cmp(big.NewInt(0)) < 0 {
		return nil, errExecutionReverted
	}

	caller := g.contract.Caller()
	value := g.contract.Value()

	// Can not delegate if no fund was sent.
	if value.Cmp(big.NewInt(0)) == 0 {
		return nil, errExecutionReverted
	}

	// Can not delegate if already delegated.
	delegatorOffset := g.state.DelegatorsOffset(nodeAddr, caller)
	if delegatorOffset.Cmp(big.NewInt(0)) >= 0 {
		return nil, errExecutionReverted
	}

	// Add to the total staked of node.
	node := g.state.Node(offset)
	node.Staked = new(big.Int).Add(node.Staked, g.contract.Value())
	g.state.UpdateNode(offset, node)

	// Push delegator record.
	offset = g.state.LenDelegators(nodeAddr)
	g.state.PushDelegator(nodeAddr, &delegatorInfo{
		Owner: caller,
		Value: value,
	})
	g.state.PutDelegatorOffset(nodeAddr, caller, offset)
	g.state.emitDelegated(nodeAddr, caller, value)

	return g.useGas(200000)
}

func (g *GovernanceContract) updateConfiguration(cfg *rawConfigStruct) ([]byte, error) {
	// Only owner can update configuration.
	if g.contract.Caller() != g.state.Owner() {
		return nil, errExecutionReverted
	}

	g.state.UpdateConfigurationRaw(cfg)
	g.state.emitConfigurationChangedEvent()
	return nil, nil
}

func (g *GovernanceContract) stake(
	publicKey []byte, name, email, location, url string) ([]byte, error) {

	// Reject invalid inputs.
	if len(name) >= 32 || len(email) >= 32 || len(location) >= 32 || len(url) >= 128 {
		return g.penalize()
	}

	caller := g.contract.Caller()
	offset := g.state.NodesOffset(caller)

	// Check if public key is valid.
	if _, err := crypto.UnmarshalPubkey(publicKey); err != nil {
		return g.penalize()
	}

	// Can not stake if already staked.
	if offset.Cmp(big.NewInt(0)) >= 0 {
		return nil, errExecutionReverted
	}

	offset = g.state.LenNodes()
	g.state.PushNode(&nodeInfo{
		Owner:     caller,
		PublicKey: publicKey,
		Staked:    big.NewInt(0),
		Name:      name,
		Email:     email,
		Location:  location,
		Url:       url,
	})
	g.state.PutNodesOffset(caller, offset)

	// Delegate fund to itself.
	if g.contract.Value().Cmp(big.NewInt(0)) > 0 {
		if ret, err := g.delegate(caller); err != nil {
			return ret, err
		}
	}

	g.state.emitStaked(caller)
	return g.useGas(100000)
}

func (g *GovernanceContract) undelegateHelper(nodeAddr, owner common.Address) ([]byte, error) {
	nodeOffset := g.state.NodesOffset(nodeAddr)
	if nodeOffset.Cmp(big.NewInt(0)) < 0 {
		return nil, errExecutionReverted
	}

	offset := g.state.DelegatorsOffset(nodeAddr, owner)
	if offset.Cmp(big.NewInt(0)) < 0 {
		return nil, errExecutionReverted
	}

	delegator := g.state.Delegator(nodeAddr, offset)
	length := g.state.LenDelegators(nodeAddr)
	lastIndex := new(big.Int).Sub(length, big.NewInt(1))

	// Delete the delegator.
	if offset.Cmp(lastIndex) != 0 {
		lastNode := g.state.Delegator(nodeAddr, lastIndex)
		g.state.UpdateDelegator(nodeAddr, offset, lastNode)
		g.state.PutDelegatorOffset(nodeAddr, lastNode.Owner, offset)
	}
	g.state.DeleteDelegatorsOffset(nodeAddr, owner)
	g.state.PopLastDelegator(nodeAddr)

	// Subtract from the total staked of node.
	node := g.state.Node(nodeOffset)
	node.Staked = new(big.Int).Sub(node.Staked, delegator.Value)
	g.state.UpdateNode(nodeOffset, node)

	// Return the staked fund.
	if !g.transfer(GovernanceContractAddress, delegator.Owner, delegator.Value) {
		return nil, errExecutionReverted
	}

	g.state.emitUndelegated(nodeAddr, owner)
	return g.useGas(100000)
}

func (g *GovernanceContract) undelegate(nodeAddr common.Address) ([]byte, error) {
	return g.undelegateHelper(nodeAddr, g.contract.Caller())
}

func (g *GovernanceContract) unstake() ([]byte, error) {
	caller := g.contract.Caller()
	offset := g.state.NodesOffset(caller)
	if offset.Cmp(big.NewInt(0)) < 0 {
		return nil, errExecutionReverted
	}

	// Undelegate all delegators.
	lenDelegators := g.state.LenDelegators(caller)
	i := new(big.Int).Sub(lenDelegators, big.NewInt(1))
	for i.Cmp(big.NewInt(0)) >= 0 {
		delegator := g.state.Delegator(caller, i)
		if ret, err := g.undelegateHelper(caller, delegator.Owner); err != nil {
			return ret, err
		}
		i = i.Sub(i, big.NewInt(1))
	}

	length := g.state.LenNodes()
	lastIndex := new(big.Int).Sub(length, big.NewInt(1))

	// Delete the node.
	if offset.Cmp(lastIndex) != 0 {
		lastNode := g.state.Node(lastIndex)
		g.state.UpdateNode(offset, lastNode)
		g.state.PutNodesOffset(lastNode.Owner, offset)
	}
	g.state.DeleteNodesOffset(caller)
	g.state.PopLastNode()

	g.state.emitUnstaked(caller)

	return g.useGas(100000)
}

func (g *GovernanceContract) proposeCRS(nextRound *big.Int, signedCRS []byte) ([]byte, error) {
	round := g.state.Round()

	if nextRound.Cmp(round) <= 0 {
		return nil, errExecutionReverted
	}

	prevCRS := g.state.CRS(round)

	// Prepare DKGMasterPublicKeys.
	var dkgMasterPKs []*dkgTypes.MasterPublicKey
	existence := make(map[coreTypes.NodeID]struct{})
	for _, mpk := range g.state.DKGMasterPublicKeys(round) {
		x := new(dkgTypes.MasterPublicKey)
		if err := rlp.DecodeBytes(mpk, x); err != nil {
			panic(err)
		}

		// Only the first DKG MPK submission is valid.
		if _, exists := existence[x.ProposerID]; exists {
			continue
		}
		existence[x.ProposerID] = struct{}{}
		dkgMasterPKs = append(dkgMasterPKs, x)
	}

	// Prepare DKGComplaints.
	var dkgComplaints []*dkgTypes.Complaint
	for _, comp := range g.state.DKGComplaints(round) {
		x := new(dkgTypes.Complaint)
		if err := rlp.DecodeBytes(comp, x); err != nil {
			panic(err)
		}
		dkgComplaints = append(dkgComplaints, x)
	}

	threshold := int(g.state.DKGSetSize().Uint64()/3 + 1)

	dkgGPK, err := core.NewDKGGroupPublicKey(
		round.Uint64(), dkgMasterPKs, dkgComplaints, threshold)
	if err != nil {
		return nil, errExecutionReverted
	}
	signature := coreCrypto.Signature{
		Type:      "bls",
		Signature: signedCRS,
	}
	if !dkgGPK.VerifySignature(coreCommon.Hash(prevCRS), signature) {
		return g.penalize()
	}

	// Save new CRS into state and increase round.
	newCRS := crypto.Keccak256(signedCRS)
	crs := common.BytesToHash(newCRS)

	g.state.PushCRS(crs)
	g.state.emitCRSProposed(nextRound, crs)

	// To encourage DKG set to propose the correct value, correctly submitting
	// this should cause nothing.
	return g.useGas(0)
}

func (g *GovernanceContract) transferOwnership(newOwner common.Address) ([]byte, error) {
	// Only owner can update configuration.
	if g.contract.Caller() != g.state.Owner() {
		return nil, errExecutionReverted
	}
	g.state.SetOwner(newOwner)
	return nil, nil
}

func (g *GovernanceContract) snapshotRound(round, height *big.Int) ([]byte, error) {
	// Validate if this mapping is correct. Only block proposer need to verify this.
	if g.evm.IsBlockProposer() {
		realHeight, ok := g.evm.GetRoundHeight(round.Uint64())
		if !ok {
			return g.penalize()
		}

		if height.Cmp(new(big.Int).SetUint64(realHeight)) != 0 {
			return g.penalize()
		}
	}

	// Only allow updating the next round.
	nextRound := g.state.LenRoundHeight()
	if round.Cmp(nextRound) != 0 {
		// No need to penalize, since the only possibility at this point is the
		// round height is already snapshoted.
		return nil, errExecutionReverted
	}

	g.state.PushRoundHeight(height)
	return nil, nil
}
