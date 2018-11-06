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
var minStake = big.NewInt(10000000000000)

const abiJSON = `
[
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
      }
    ],
    "name": "offset",
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
        "name": "round",
        "type": "uint256"
      },
      {
        "indexed": false,
        "name": "crs",
        "type": "bytes32"
      }
    ],
    "name": "CRSProposed",
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
  }
]
`

var abiObject abi.ABI
var GovernanceContractName2Method map[string]abi.Method
var sig2Method map[string]abi.Method
var events map[string]abi.Event

func init() {
	// Parse governance contract ABI.
	abiObject, err := abi.JSON(strings.NewReader(abiJSON))
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
func RunGovernanceContract(evm *EVM, input []byte, contract *Contract) (
	ret []byte, err error) {
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
	case "nodesLength":
		res, err := method.Outputs.Pack(g.state.NodesLength())
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
	case "unstake":
		return g.unstake()
	case "updateConfiguration":
		var cfg params.DexconConfig
		if err := method.Inputs.Unpack(&cfg, arguments); err != nil {
			return nil, errExecutionReverted
		}
		g.updateConfiguration(&cfg)

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
	case "notarySetSize":
		res, err := method.Outputs.Pack(g.state.NotarySetSize())
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "offset":
		addr := common.Address{}
		if err := method.Inputs.Unpack(&addr, arguments); err != nil {
			return nil, errExecutionReverted
		}
		res, err := method.Outputs.Pack(g.state.Offset(addr))
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
	case "RoundHeight":
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
	RoundHeightLoc = iota
	nodesLoc
	offsetLoc
	crsLoc
	dkgMasterPublicKeysLoc
	dkgComplaintsLoc
	dkgFinalizedLoc
	dkgFinalizedsCountLoc
	ownerLoc
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
	return new(big.Int).SetBytes(crypto.Keccak256(
		key, common.BigToHash(pos).Bytes()))
}

func (s *GovernanceStateHelper) readBytes(loc *big.Int) []byte {
	// length of the dynamic array (bytes).
	rawLength := s.getStateBigInt(loc)
	lengthByte := new(big.Int).Mod(rawLength, big.NewInt(256))

	// bytes length <= 31, lengthByte % 2 == 0
	// return the high 31 bytes.
	if new(big.Int).Mod(lengthByte, big.NewInt(2)).Cmp(big.NewInt(0)) == 0 {
		length := new(big.Int).Div(lengthByte, big.NewInt(2)).Uint64()
		return rawLength.Bytes()[:length]
	}

	// actual length = (rawLength - 1) / 2
	length := new(big.Int).Div(new(big.Int).Sub(rawLength, big.NewInt(1)), big.NewInt(2)).Uint64()

	// data address.
	dataLoc := s.getSlotLoc(loc)

	// read continiously for length bytes.
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

	// short bytes (length <= 31)
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

	// write 2 * length + 1
	storedLength := new(big.Int).Add(new(big.Int).Mul(
		big.NewInt(length), big.NewInt(2)), big.NewInt(1))
	s.setStateBigInt(loc, storedLength)
	// write data chunck.
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
		// Right pad with zeros
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

	// increase length by 1.
	arrayLength := s.getStateBigInt(loc)
	s.setStateBigInt(loc, new(big.Int).Add(arrayLength, big.NewInt(1)))

	// write element.
	dataLoc := s.getSlotLoc(loc)
	elementLoc := new(big.Int).Add(dataLoc, arrayLength)
	s.writeBytes(elementLoc, data)
}

// uint256[] public RoundHeight;
func (s *GovernanceStateHelper) LenRoundHeight() *big.Int {
	return s.getStateBigInt(big.NewInt(RoundHeightLoc))
}
func (s *GovernanceStateHelper) RoundHeight(round *big.Int) *big.Int {
	baseLoc := s.getSlotLoc(big.NewInt(RoundHeightLoc))
	loc := new(big.Int).Add(baseLoc, round)
	return s.getStateBigInt(loc)
}
func (s *GovernanceStateHelper) PushRoundHeight(height *big.Int) {
	// increase length by 1.
	length := s.getStateBigInt(big.NewInt(RoundHeightLoc))
	s.setStateBigInt(big.NewInt(RoundHeightLoc), new(big.Int).Add(length, big.NewInt(1)))

	baseLoc := s.getSlotLoc(big.NewInt(RoundHeightLoc))
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

const nodesInfoSize = 7

func (s *GovernanceStateHelper) NodesLength() *big.Int {
	return s.getStateBigInt(big.NewInt(nodesLoc))
}
func (s *GovernanceStateHelper) Node(index *big.Int) *nodeInfo {
	node := new(nodeInfo)

	arrayBaseLoc := s.getSlotLoc(big.NewInt(nodesLoc))
	elementBaseLoc := new(big.Int).Add(arrayBaseLoc,
		new(big.Int).Mul(index, big.NewInt(nodesInfoSize)))

	// owner.
	loc := elementBaseLoc
	node.Owner = common.BytesToAddress(s.getState(common.BigToHash(elementBaseLoc)).Bytes())

	// publicKey.
	loc = new(big.Int).Add(elementBaseLoc, big.NewInt(1))
	node.PublicKey = s.readBytes(loc)

	// staked.
	loc = new(big.Int).Add(elementBaseLoc, big.NewInt(2))
	node.Staked = s.getStateBigInt(loc)

	// name.
	loc = new(big.Int).Add(elementBaseLoc, big.NewInt(3))
	node.Name = string(s.readBytes(loc))

	// email.
	loc = new(big.Int).Add(elementBaseLoc, big.NewInt(4))
	node.Email = string(s.readBytes(loc))

	// location.
	loc = new(big.Int).Add(elementBaseLoc, big.NewInt(5))
	node.Location = string(s.readBytes(loc))

	// url.
	loc = new(big.Int).Add(elementBaseLoc, big.NewInt(6))
	node.Url = string(s.readBytes(loc))

	return node
}
func (s *GovernanceStateHelper) PushNode(n *nodeInfo) {
	// increase length by 1
	arrayLength := s.NodesLength()
	s.setStateBigInt(big.NewInt(nodesLoc), new(big.Int).Add(arrayLength, big.NewInt(1)))

	s.UpdateNode(arrayLength, n)
}
func (s *GovernanceStateHelper) UpdateNode(index *big.Int, n *nodeInfo) {
	arrayBaseLoc := s.getSlotLoc(big.NewInt(nodesLoc))
	elementBaseLoc := new(big.Int).Add(arrayBaseLoc,
		new(big.Int).Mul(index, big.NewInt(nodesInfoSize)))

	// owner.
	loc := elementBaseLoc
	s.setState(common.BigToHash(loc), n.Owner.Hash())

	// publicKey.
	loc = new(big.Int).Add(elementBaseLoc, big.NewInt(1))
	s.writeBytes(loc, n.PublicKey)

	// staked.
	loc = new(big.Int).Add(elementBaseLoc, big.NewInt(2))
	s.setStateBigInt(loc, n.Staked)

	// name.
	loc = new(big.Int).Add(elementBaseLoc, big.NewInt(3))
	s.writeBytes(loc, []byte(n.Name))

	// email.
	loc = new(big.Int).Add(elementBaseLoc, big.NewInt(4))
	s.writeBytes(loc, []byte(n.Email))

	// location.
	loc = new(big.Int).Add(elementBaseLoc, big.NewInt(5))
	s.writeBytes(loc, []byte(n.Location))

	// url.
	loc = new(big.Int).Add(elementBaseLoc, big.NewInt(6))
	s.writeBytes(loc, []byte(n.Url))
}
func (s *GovernanceStateHelper) Nodes() []*nodeInfo {
	var nodes []*nodeInfo
	for i := int64(0); i < int64(s.NodesLength().Uint64()); i++ {
		nodes = append(nodes, s.Node(big.NewInt(i)))
	}
	return nodes
}

// mapping(address => uint256) public offset;
func (s *GovernanceStateHelper) Offset(addr common.Address) *big.Int {
	loc := s.getMapLoc(big.NewInt(offsetLoc), addr.Bytes())
	return new(big.Int).Sub(s.getStateBigInt(loc), big.NewInt(1))
}
func (s *GovernanceStateHelper) PutOffset(addr common.Address, offset *big.Int) {
	loc := s.getMapLoc(big.NewInt(offsetLoc), addr.Bytes())
	s.setStateBigInt(loc, new(big.Int).Add(offset, big.NewInt(1)))
}
func (s *GovernanceStateHelper) DeleteOffset(addr common.Address) {
	loc := s.getMapLoc(big.NewInt(offsetLoc), addr.Bytes())
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
	offset := s.NodesLength()
	s.PushNode(&nodeInfo{
		Owner:     addr,
		PublicKey: publicKey,
		Staked:    staked,
		Name:      name,
		Email:     email,
		Location:  location,
		Url:       url,
	})
	s.PutOffset(addr, offset)
}

// Configuration returns the current configuration.
func (s *GovernanceStateHelper) Configuration() *params.DexconConfig {
	return &params.DexconConfig{
		BlockReward:      s.getStateBigInt(big.NewInt(blockRewardLoc)),
		BlockGasLimit:    uint64(s.getStateBigInt(big.NewInt(blockGasLimitLoc)).Uint64()),
		NumChains:        uint32(s.getStateBigInt(big.NewInt(numChainsLoc)).Uint64()),
		LambdaBA:         s.getStateBigInt(big.NewInt(lambdaBALoc)).Uint64(),
		LambdaDKG:        s.getStateBigInt(big.NewInt(lambdaDKGLoc)).Uint64(),
		K:                int(s.getStateBigInt(big.NewInt(kLoc)).Uint64()),
		PhiRatio:         float32(s.getStateBigInt(big.NewInt(phiRatioLoc)).Uint64()) / 1000000.0,
		NotarySetSize:    uint32(s.getStateBigInt(big.NewInt(notarySetSizeLoc)).Uint64()),
		DKGSetSize:       uint32(s.getStateBigInt(big.NewInt(dkgSetSizeLoc)).Uint64()),
		RoundInterval:    s.getStateBigInt(big.NewInt(roundIntervalLoc)).Uint64(),
		MinBlockInterval: s.getStateBigInt(big.NewInt(minBlockIntervalLoc)).Uint64(),
	}
}

// UpdateConfiguration updates system configuration.
func (s *GovernanceStateHelper) UpdateConfiguration(cfg *params.DexconConfig) {
	s.setStateBigInt(big.NewInt(blockRewardLoc), cfg.BlockReward)
	s.setStateBigInt(big.NewInt(blockGasLimitLoc), big.NewInt(int64(cfg.BlockGasLimit)))
	s.setStateBigInt(big.NewInt(numChainsLoc), big.NewInt(int64(cfg.NumChains)))
	s.setStateBigInt(big.NewInt(lambdaBALoc), big.NewInt(int64(cfg.LambdaBA)))
	s.setStateBigInt(big.NewInt(lambdaDKGLoc), big.NewInt(int64(cfg.LambdaDKG)))
	s.setStateBigInt(big.NewInt(kLoc), big.NewInt(int64(cfg.K)))
	s.setStateBigInt(big.NewInt(phiRatioLoc), big.NewInt(int64(cfg.PhiRatio)))
	s.setStateBigInt(big.NewInt(notarySetSizeLoc), big.NewInt(int64(cfg.NotarySetSize)))
	s.setStateBigInt(big.NewInt(dkgSetSizeLoc), big.NewInt(int64(cfg.DKGSetSize)))
	s.setStateBigInt(big.NewInt(roundIntervalLoc), big.NewInt(int64(cfg.RoundInterval)))
	s.setStateBigInt(big.NewInt(minBlockIntervalLoc), big.NewInt(int64(cfg.MinBlockInterval)))
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

func (g *GovernanceContract) penalize() {
	g.contract.UseGas(g.contract.Gas)
}

func (g *GovernanceContract) inDKGSet(nodeID coreTypes.NodeID) bool {
	target := coreTypes.NewDKGSetTarget(coreCommon.Hash(g.state.CurrentCRS()))
	ns := coreTypes.NewNodeSet()

	for _, x := range g.state.Nodes() {
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
		g.penalize()
		return nil, errExecutionReverted
	}

	caller := g.contract.Caller()

	// Finalized caller is not allowed to propose complaint.
	if g.state.DKGFinalized(round, caller) {
		g.penalize()
		return nil, errExecutionReverted
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
		g.penalize()
		return nil, errExecutionReverted
	}

	// DKGComplaint must belongs to someone in DKG set.
	if !g.inDKGSet(dkgComplaint.ProposerID) {
		g.penalize()
		return nil, errExecutionReverted
	}

	verified, _ := core.VerifyDKGComplaintSignature(&dkgComplaint)
	if !verified {
		g.penalize()
		return nil, errExecutionReverted
	}

	g.state.PushDKGComplaint(round, comp)

	// Set this to relatively high to prevent spamming
	g.contract.UseGas(10000000)
	return nil, nil
}

func (g *GovernanceContract) addDKGMasterPublicKey(round *big.Int, mpk []byte) ([]byte, error) {
	// Can only add DKG master public key of current and next round.
	if round.Cmp(new(big.Int).Add(g.state.Round(), big.NewInt(1))) > 0 {
		g.penalize()
		return nil, errExecutionReverted
	}

	caller := g.contract.Caller()
	offset := g.state.Offset(caller)

	// Can not add dkg mpk if not staked.
	if offset.Cmp(big.NewInt(0)) < 0 {
		return nil, errExecutionReverted
	}

	var dkgMasterPK dkgTypes.MasterPublicKey
	if err := rlp.DecodeBytes(mpk, &dkgMasterPK); err != nil {
		g.penalize()
		return nil, errExecutionReverted
	}

	// DKGMasterPublicKey must belongs to someone in DKG set.
	if !g.inDKGSet(dkgMasterPK.ProposerID) {
		g.penalize()
		return nil, errExecutionReverted
	}

	verified, _ := core.VerifyDKGMasterPublicKeySignature(&dkgMasterPK)
	if !verified {
		g.penalize()
		return nil, errExecutionReverted
	}

	g.state.PushDKGMasterPublicKey(round, mpk)

	// DKG operation is expensive.
	g.contract.UseGas(100000)
	return nil, nil
}

func (g *GovernanceContract) addDKGFinalize(round *big.Int, finalize []byte) ([]byte, error) {
	if round.Cmp(g.state.Round()) != 0 {
		g.penalize()
		return nil, errExecutionReverted
	}

	caller := g.contract.Caller()

	var dkgFinalize dkgTypes.Finalize
	if err := rlp.DecodeBytes(finalize, &dkgFinalize); err != nil {
		g.penalize()
		return nil, errExecutionReverted
	}

	// DKGFInalize must belongs to someone in DKG set.
	if !g.inDKGSet(dkgFinalize.ProposerID) {
		g.penalize()
		return nil, errExecutionReverted
	}

	verified, _ := core.VerifyDKGFinalizeSignature(&dkgFinalize)
	if !verified {
		g.penalize()
		return nil, errExecutionReverted
	}

	if !g.state.DKGFinalized(round, caller) {
		g.state.PutDKGFinalized(round, caller, true)
		g.state.IncDKGFinalizedsCount(round)
	}

	// DKG operation is expensive.
	g.contract.UseGas(100000)
	return nil, nil
}

func (g *GovernanceContract) updateConfiguration(config *params.DexconConfig) ([]byte, error) {
	// Only owner can update configuration.
	if g.contract.Caller() != g.state.Owner() {
		return nil, errExecutionReverted
	}

	g.state.UpdateConfiguration(config)
	g.state.emitConfigurationChangedEvent()
	return nil, nil
}

func (g *GovernanceContract) stake(
	publicKey []byte, name, email, location, url string) ([]byte, error) {

	// Reject invalid inputs.
	if len(name) >= 32 || len(email) >= 32 || len(location) >= 32 || len(url) >= 128 {
		g.penalize()
		return nil, errExecutionReverted
	}

	caller := g.contract.Caller()
	offset := g.state.Offset(caller)

	// Can not stake if already staked.
	if offset.Cmp(big.NewInt(0)) >= 0 {
		g.penalize()
		return nil, errExecutionReverted
	}

	offset = g.state.NodesLength()
	g.state.PushNode(&nodeInfo{
		Owner:     caller,
		PublicKey: publicKey,
		Staked:    g.contract.Value(),
		Name:      name,
		Email:     email,
		Location:  location,
		Url:       url,
	})
	g.state.PutOffset(caller, offset)

	g.contract.UseGas(21000)
	return nil, nil
}

func (g *GovernanceContract) unstake() ([]byte, error) {
	caller := g.contract.Caller()
	offset := g.state.Offset(caller)
	if offset.Cmp(big.NewInt(0)) < 0 {
		return nil, errExecutionReverted
	}

	node := g.state.Node(offset)
	length := g.state.NodesLength()
	lastIndex := new(big.Int).Sub(length, big.NewInt(1))

	// Delete the node.
	if offset != lastIndex {
		lastNode := g.state.Node(lastIndex)
		g.state.UpdateNode(offset, lastNode)
		g.state.PutOffset(lastNode.Owner, offset)
		g.state.DeleteOffset(caller)
	}

	// Return the staked fund.
	// TODO(w): add this to debug trace so it shows up as internal transaction.
	g.evm.Transfer(g.evm.StateDB, GovernanceContractAddress, caller, node.Staked)

	g.contract.UseGas(21000)
	return nil, nil
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
		g.penalize()
		return nil, errExecutionReverted
	}

	// Save new CRS into state and increase round.
	newCRS := crypto.Keccak256(signedCRS)
	crs := common.BytesToHash(newCRS)

	g.state.PushCRS(crs)
	g.state.emitCRSProposed(g.state.LenCRS(), crs)

	// To encourage DKG set to propose the correct value, correctly submitting
	// this should cause nothing.
	g.contract.UseGas(0)
	return nil, nil
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
	// Validate if this mapping is correct.
	rawHeight, ok := g.evm.Context.RoundHeight.Load(round)
	if !ok {
		g.penalize()
		return nil, errExecutionReverted
	}

	realHeight := rawHeight.(uint64)
	if height.Cmp(new(big.Int).SetUint64(realHeight)) != 0 {
		g.penalize()
		return nil, errExecutionReverted
	}

	// Only allow updating the next round.
	nextRound := g.state.LenRoundHeight()
	if round.Cmp(nextRound) != 0 {
		g.penalize()
		return nil, errExecutionReverted
	}

	g.state.PushRoundHeight(height)
	return nil, nil
}
