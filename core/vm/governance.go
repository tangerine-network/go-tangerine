package vm

import (
	"math/big"
	"strings"

	"github.com/dexon-foundation/dexon/accounts/abi"
	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/crypto"
	"github.com/dexon-foundation/dexon/rlp"

	coreCommon "github.com/dexon-foundation/dexon-consensus-core/common"
	"github.com/dexon-foundation/dexon-consensus-core/core"
	coreCrypto "github.com/dexon-foundation/dexon-consensus-core/core/crypto"
	"github.com/dexon-foundation/dexon-consensus-core/core/types"
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
		args := struct {
			Round     *big.Int
			SignedCRS []byte
		}{}
		if err := method.Inputs.Unpack(&args, argument); err != nil {
			return nil, errExecutionReverted
		}
		return g.proposeCRS(args.Round, args.SignedCRS)
	case "addDKGMasterPublicKey":
		args := struct {
			Round              *big.Int
			DKGMasterPublicKey []byte
		}{}
		if err := method.Inputs.Unpack(&args, argument); err != nil {
			return nil, errExecutionReverted
		}
		return g.addDKGMasterPublicKey(args.Round, args.DKGMasterPublicKey)
	case "addDKGComplaint":
		args := struct {
			Round        *big.Int
			DKGComplaint []byte
		}{}
		if err := method.Inputs.Unpack(&args, argument); err != nil {
			return nil, errExecutionReverted
		}
		return g.addDKGComplaint(args.Round, args.DKGComplaint)
	}
	return nil, nil
}

// State manipulation helper fro the governance contract.
type StateHelper struct {
	StateDB StateDB
}

// struct Node {
//     address owner;
//     bytes publicKey;
//     uint256 staked;
// }
//
// 0: Node[] nodes;

type nodeInfo struct {
	owner     common.Address
	publicKey []byte
	staked    *big.Int
}

func (s *StateHelper) nodesLength() *big.Int {
	return nil
}
func (s *StateHelper) node(index *big.Int) *nodeInfo {
	return nil
}
func (s *StateHelper) pushNode(n *nodeInfo) {
}
func (s *StateHelper) updateNode(index *big.Int, n *nodeInfo) {
}

// 1: mapping(address => uint256) public offset;
func (s *StateHelper) offset(addr common.Address) *big.Int {
	return nil
}
func (s *StateHelper) putOffset(addr common.Address, offset *big.Int) {
}
func (s *StateHelper) deleteOffset(addr common.Address) {
}

// 2: mapping(uint256 => bytes32) public crs;
func (s *StateHelper) crs(round *big.Int) common.Hash {
	return common.Hash{}
}
func (s *StateHelper) putCRS(round *big.Int, crs []byte) common.Hash {
	return common.Hash{}
}

// 3: mapping(uint256 => bytes[]) public DKGMasterPublicKeys;
func (s *StateHelper) dkgMasterPublicKeys(round *big.Int) [][]byte {
	return nil
}
func (s *StateHelper) pushDKGMasterPublicKey(round *big.Int, pk []byte) {
}

// 4: mapping(uint256 => bytes[]) public DKGComplaints;
func (s *StateHelper) dkgComplaints(round *big.Int) [][]byte {
	return nil
}
func (s *StateHelper) addDKGComplaint(round *big.Int, complaint []byte) {
}

// 5: uint256 public round;
func (s *StateHelper) round() *big.Int {
	return nil
}
func (s *StateHelper) incRound() *big.Int {
	return nil
}

// 6: address public governanceMultisig;
func (s *StateHelper) governanceMultisig() common.Address {
	return common.Address{}
}

// 7: uint256 public numChains;
func (s *StateHelper) numChains() *big.Int {
	return nil
}

// 8: uint256 public lambdaBA;
func (s *StateHelper) lambdaBA() *big.Int {
	return nil
}

// 9: uint256 public lambdaDKG;
func (s *StateHelper) lambdaDKG() *big.Int {
	return nil
}

// 10: uint256 public k;
func (s *StateHelper) k() *big.Int {
	return nil
}

// 11: uint256 public phiRatio;  // stored as PhiRatio * 10^6
func (s *StateHelper) phiRatio() *big.Int {
	return nil
}

// 12: uint256 public numNotarySet;
func (s *StateHelper) numNotarySet() *big.Int {
	return nil
}

// 13: uint256 public numDKGSet;
func (s *StateHelper) numDKGSet() *big.Int {
	return nil
}

// 14: uint256 public roundInterval
func (s *StateHelper) roundInterval() *big.Int {
	return nil
}

// 15: uint256 public minBlockInterval
func (s *StateHelper) minBlockInterval() *big.Int {
	return nil
}

// 16: uint256 public maxBlockInterval
func (s *StateHelper) maxBlockInterval() *big.Int {
	return nil
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
	caller := g.contract.Caller()
	offset := g.state.offset(caller)

	// Can not stake if already staked.
	if offset != nil {
		return nil, errExecutionReverted
	}

	// TODO(w): check of pk belongs to the address.
	offset = g.state.nodesLength()
	g.state.pushNode(&nodeInfo{
		owner:     caller,
		publicKey: publicKey,
		staked:    g.contract.Value(),
	})
	g.state.putOffset(caller, offset)
	return nil, nil
}

func (g *GovernanceContract) unstake() ([]byte, error) {
	caller := g.contract.Caller()
	offset := g.state.offset(caller)
	if offset == nil {
		return nil, errExecutionReverted
	}

	node := g.state.node(offset)
	length := g.state.nodesLength()
	lastIndex := new(big.Int).Sub(length, big.NewInt(1))

	// Delete the node.
	if offset != lastIndex {
		lastNode := g.state.node(lastIndex)
		g.state.updateNode(offset, lastNode)
		g.state.putOffset(lastNode.owner, offset)
		g.state.deleteOffset(caller)
	}

	// Return the staked fund.
	g.evm.Transfer(g.evm.StateDB, GovernanceContractAddress, caller, node.staked)
	return nil, nil
}

func (g *GovernanceContract) proposeCRS(round *big.Int, signedCRS []byte) ([]byte, error) {
	crs := g.state.crs(round)

	// Revert if CRS for that round already exists.
	if crs != (common.Hash{}) {
		return nil, errExecutionReverted
	}

	prevRound := g.state.round()
	prevCRS := g.state.crs(prevRound)

	// round should be the next round number, abort otherwise.
	if new(big.Int).Add(prevRound, big.NewInt(1)).Cmp(round) != 0 {
		return nil, errExecutionReverted
	}

	// Prepare DKGMasterPublicKeys.
	// TODO(w): make sure DKGMasterPKs are unique.
	var dkgMasterPKs []*types.DKGMasterPublicKey
	for _, pk := range g.state.dkgMasterPublicKeys(round) {
		x := new(types.DKGMasterPublicKey)
		if err := rlp.DecodeBytes(pk, x); err != nil {
			panic(err)
		}
		dkgMasterPKs = append(dkgMasterPKs, x)
	}

	// Prepare DKGComplaints.
	var dkgComplaints []*types.DKGComplaint
	for _, comp := range g.state.dkgComplaints(round) {
		x := new(types.DKGComplaint)
		if err := rlp.DecodeBytes(comp, x); err != nil {
			panic(err)
		}
		dkgComplaints = append(dkgComplaints, x)
	}

	threshold := int(g.state.numDKGSet().Uint64() / 3)

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
		return nil, errExecutionReverted
	}

	// Save new CRS into state and increase round.
	newCRS := crypto.Keccak256(signedCRS)
	g.state.incRound()
	g.state.putCRS(round, newCRS)

	return nil, nil
}

func (g *GovernanceContract) addDKGMasterPublicKey(round *big.Int, pk []byte) ([]byte, error) {
	var dkgMasterPK types.DKGMasterPublicKey
	if err := rlp.DecodeBytes(pk, &dkgMasterPK); err != nil {
		return nil, errExecutionReverted
	}
	verified, _ := core.VerifyDKGMasterPublicKeySignature(&dkgMasterPK)
	if !verified {
		return nil, errExecutionReverted
	}

	g.state.pushDKGMasterPublicKey(round, pk)
	return nil, nil
}

func (g *GovernanceContract) addDKGComplaint(round *big.Int, comp []byte) ([]byte, error) {
	var dkgComplaint types.DKGComplaint
	if err := rlp.DecodeBytes(comp, &dkgComplaint); err != nil {
		return nil, errExecutionReverted
	}
	verified, _ := core.VerifyDKGComplaintSignature(&dkgComplaint)
	if !verified {
		return nil, errExecutionReverted
	}

	g.state.addDKGComplaint(round, comp)
	return nil, nil
}
