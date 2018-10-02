package vm

import (
	"math/big"
	"strings"

	"github.com/dexon-foundation/dexon/accounts/abi"
	"github.com/dexon-foundation/dexon/common"
)

var GovernanceContractAddress = common.BytesToAddress([]byte{0XED}) // Reverse of DEX0
var multisigAddress = common.BytesToAddress([]byte("0x....."))
var minStake = big.NewInt(10000000000000)

const abiJSON = `
[
  {
    "constant": true,
    "inputs": [],
    "name": "round",
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
    "name": "lambdaBA",
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
        "type": "int256"
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
    "name": "lambdaDKG",
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
    "name": "k",
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
    "name": "numChains",
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
    "name": "DKGMasterPublicKeys",
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
        "type": "uint256"
      }
    ],
    "name": "DKGComplaints",
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
    "name": "governanceMultisig",
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
    "constant": false,
    "inputs": [
      {
        "name": "",
        "type": "int256"
      },
      {
        "name": "",
        "type": "int256"
      },
      {
        "name": "",
        "type": "int256"
      },
      {
        "name": "",
        "type": "int256"
      },
      {
        "name": "",
        "type": "int256"
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
        "name": "",
        "type": "uint256"
      },
      {
        "name": "",
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
        "name": "",
        "type": "uint256"
      },
      {
        "name": "",
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
        "name": "",
        "type": "uint256"
      },
      {
        "name": "",
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
        "name": "",
        "type": "bytes"
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
  }
]
`

var abiObject abi.ABI
var sig2Method map[string]abi.Method

func init() {
	// Parse governance contract ABI.
	abiObject, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		panic(err)
	}

	sig2Method = make(map[string]abi.Method)

	// Construct dispatch table.
	for _, method := range abiObject.Methods {
		sig2Method[method.Sig()] = method
	}
}

// RunGovernanceContract executes governance contract.
func RunGovernanceContract(evm *EVM, input []byte, contract *Contract) (
	ret []byte, err error) {
	// Parse input.
	method, exists := sig2Method[string(input[:4])]
	if !exists {
		return nil, errExecutionReverted
	}

	// Dispatch method call.
	g := NewGovernanceContract(evm, contract)
	argument := input[4:]

	switch method.Name {
	case "stake":
		var publicKey []byte
		if err := method.Inputs.Unpack(&publicKey, argument); err != nil {
			return nil, errExecutionReverted
		}
		return g.stake(publicKey)
	case "unstake":
		return g.unstake()
	case "proposeCRS":
		return g.proposeCRS()
	case "addDKGMasterPublicKey":
		return g.addDKGMasterPublicKey()
	case "addDKGComplaint":
		return g.addDKGComplaint()
	}
	return nil, nil
}

// State manipulation helper fro the governance contract.
type StateHelper struct {
	StateDB StateDB
}

// 0: address public governanceMultisig;
func (s *StateHelper) governanceMultisig() common.Address {
	return common.Address{}
}

// 1: int256 public numChains;
func (s *StateHelper) numChains() *big.Int {
	return nil
}

// 2: int256 public lambdaBA;
func (s *StateHelper) lambdaBA() *big.Int {
	return nil
}

// 3: int256 public lambdaDKG;
func (s *StateHelper) lambdaDKG() *big.Int {
	return nil
}

// 4: int256 public k;
func (s *StateHelper) k() *big.Int {
	return nil
}

// 5: int256 public phiRatio;  // stored as PhiRatio * 10^6
func (s *StateHelper) phiRatio() *big.Int {
	return nil
}

// struct Node {
//     address owner;
//     bytes publicKey;
//     uint256 staked;
// }
//
// 6: Node[] nodes;

type nodeInfo struct {
	owner     common.Address
	publicKey []byte
	staked    *big.Int
}

func (s *StateHelper) nodesLength() *big.Int {
	return nil
}
func (s *StateHelper) node(offset *big.Int) *nodeInfo {
	return nil
}
func (s *StateHelper) pushNode(n *nodeInfo) {
}

// 7: mapping(address => uint256) public offset;
func (s *StateHelper) offset(addr common.Address) *big.Int {
	return nil
}
func (s *StateHelper) putOffset(addr common.Address, offset *big.Int) {
}
func (s *StateHelper) deleteOffset(addr common.Address) {
}

// 8: uint256 public round;
func (s *StateHelper) round() *big.Int {
	return nil
}
func (s *StateHelper) incRound() *big.Int {
	return nil
}

// 9: mapping(uint256 => bytes32) public crs;
func (s *StateHelper) crs(round *big.Int) common.Hash {
	return common.Hash{}
}
func (s *StateHelper) pushCRS(round *big.Int, crs []byte) common.Hash {
	return common.Hash{}
}

// 10: mapping(uint256 => bytes[]) public DKGMasterPublicKeys;
func (s *StateHelper) dkgMasterPublicKey(round *big.Int) [][]byte {
	return nil
}
func (s *StateHelper) addDKGMasterPublicKey(round *big.Int, pk []byte) {
}

// 11: mapping(uint256 => bytes[]) public DKGComplaints;
func (s *StateHelper) dkgComplaint(round *big.Int) [][]byte {
	return nil
}
func (s *StateHelper) addDKGComplaint(round *big.Int, complaint []byte) {
}

type GovernanceContract struct {
	evm      *EVM
	state    StateHelper
	contract *Contract
}

func NewGovernanceContract(evm *EVM, contract *Contract) *GovernanceContract {
	return &GovernanceContract{
		evm:      evm,
		state:    StateHelper{evm.StateDB},
		contract: contract,
	}
}

func (G *GovernanceContract) updateConfiguration() ([]byte, error) {
	return nil, nil
}

func (g *GovernanceContract) stake(publicKey []byte) ([]byte, error) {
	return nil, nil
}

func (g *GovernanceContract) unstake() ([]byte, error) {
	return nil, nil
}

func (g *GovernanceContract) proposeCRS() ([]byte, error) {
	return nil, nil
}

func (g *GovernanceContract) addDKGMasterPublicKey() ([]byte, error) {
	return nil, nil
}

func (g *GovernanceContract) addDKGComplaint() ([]byte, error) {
	return nil, nil
}
