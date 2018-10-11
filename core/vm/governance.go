package vm

import (
	"math/big"
	"strings"

	"github.com/dexon-foundation/dexon/accounts/abi"
	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/core/types"
	"github.com/dexon-foundation/dexon/crypto"
	"github.com/dexon-foundation/dexon/rlp"

	coreCommon "github.com/dexon-foundation/dexon-consensus-core/common"
	"github.com/dexon-foundation/dexon-consensus-core/core"
	coreCrypto "github.com/dexon-foundation/dexon-consensus-core/core/crypto"
	coreTypes "github.com/dexon-foundation/dexon-consensus-core/core/types"
)

var GovernanceContractAddress = common.BytesToAddress([]byte{0XED}) // Reverse of DEX0
var minStake = big.NewInt(10000000000000)

const abiJSON = `
[
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
    "inputs": [],
    "name": "maxBlockInterval",
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
      },
      {
        "name": "MaxBlockInterval",
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
var events map[string]abi.Event

func init() {
	// Parse governance contract ABI.
	abiObject, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		panic(err)
	}

	sig2Method = make(map[string]abi.Method)

	// Construct dispatch table.
	for _, method := range abiObject.Methods {
		sig2Method[string(method.Id())] = method
	}

	events = make(map[string]abi.Event)

	// Event cache.
	for _, event := range abiObject.Events {
		events[event.Name] = event
	}
}

func nodeIDToAddress(nodeID coreTypes.NodeID) common.Address {
	return common.BytesToAddress(nodeID.Bytes()[12:])
}

type configParams struct {
	NumChains        uint32
	LambdaBA         uint64
	LambdaDKG        uint64
	K                int
	PhiRatio         float32
	NotarySetSize    uint32
	DKGSetSize       uint32
	RoundInterval    uint64
	MinBlockInterval uint64
	MaxBlockInterval uint64
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
	case "updateConfiguration":
		var config configParams
		if err := method.Inputs.Unpack(&config, arguments); err != nil {
			return nil, errExecutionReverted
		}
		g.updateConfiguration(&config)
	case "stake":
		var publicKey []byte
		if err := method.Inputs.Unpack(&publicKey, arguments); err != nil {
			return nil, errExecutionReverted
		}
		return g.stake(publicKey)
	case "unstake":
		return g.unstake()
	case "proposeCRS":
		var signedCRS []byte
		if err := method.Inputs.Unpack(&signedCRS, arguments); err != nil {
			return nil, errExecutionReverted
		}
		return g.proposeCRS(signedCRS)
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

	// --------------------------------
	// Solidity auto generated methods.
	// --------------------------------

	case "crs":
		round := new(big.Int)
		if err := method.Inputs.Unpack(&round, arguments); err != nil {
			return nil, errExecutionReverted
		}
		res, err := method.Outputs.Pack(g.state.crs(round))
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
		complaints := g.state.dkgComplaints(round)
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
		finalized := g.state.dkgFinalized(round, addr)
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
		count := g.state.dkgFinalizedsCount(round)
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
		pks := g.state.dkgMasterPublicKeys(round)
		if int(index.Uint64()) >= len(pks) {
			return nil, errExecutionReverted
		}
		pk := pks[index.Uint64()]
		res, err := method.Outputs.Pack(pk)
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "dkgSetSize":
		res, err := method.Outputs.Pack(g.state.dkgSetSize())
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "k":
		res, err := method.Outputs.Pack(g.state.k())
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "lambdaBA":
		res, err := method.Outputs.Pack(g.state.lambdaBA())
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "lambdaDKG":
		res, err := method.Outputs.Pack(g.state.lambdaDKG())
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "maxBlockInterval":
		res, err := method.Outputs.Pack(g.state.maxBlockInterval())
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "minBlockInterval":
		res, err := method.Outputs.Pack(g.state.minBlockInterval())
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "numChains":
		res, err := method.Outputs.Pack(g.state.numChains())
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "nodes":
		index := new(big.Int)
		if err := method.Inputs.Unpack(&index, arguments); err != nil {
			return nil, errExecutionReverted
		}
		info := g.state.node(index)
		res, err := method.Outputs.Pack(info.owner, info.publicKey, info.staked)
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "notarySetSize":
		res, err := method.Outputs.Pack(g.state.notarySetSize())
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "offset":
		addr := common.Address{}
		if err := method.Inputs.Unpack(&addr, arguments); err != nil {
			return nil, errExecutionReverted
		}
		res, err := method.Outputs.Pack(g.state.offset(addr))
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "owner":
		res, err := method.Outputs.Pack(g.state.owner())
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "phiRatio":
		res, err := method.Outputs.Pack(g.state.phiRatio())
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	case "roundInterval":
		res, err := method.Outputs.Pack(g.state.roundInterval())
		if err != nil {
			return nil, errExecutionReverted
		}
		return res, nil
	}
	return nil, nil
}

// Storage position enums.
const (
	nodesLoc = iota
	offsetLoc
	crsLoc
	dkgMasterPublicKeysLoc
	dkgComplaintsLoc
	dkgFinailizedLoc
	dkgFinalizedsCountLoc
	ownerLoc
	numChainsLoc
	lambdaBALoc
	lambdaDKGLoc
	kLoc
	phiRatioLoc
	notarySetSizeLoc
	dkgSetSizeLoc
	roundIntervalLoc
	minBlockIntervalLoc
	maxBlockIntervalLoc
)

// State manipulation helper fro the governance contract.
type StateHelper struct {
	Address common.Address
	StateDB StateDB
}

func (s *StateHelper) getState(loc common.Hash) common.Hash {
	return s.StateDB.GetState(s.Address, loc)
}

func (s *StateHelper) setState(loc common.Hash, val common.Hash) {
	s.StateDB.SetState(s.Address, loc, val)
}

func (s *StateHelper) getStateBigInt(loc *big.Int) *big.Int {
	res := s.StateDB.GetState(s.Address, common.BigToHash(loc))
	return new(big.Int).SetBytes(res.Bytes())
}

func (s *StateHelper) setStateBigInt(loc *big.Int, val *big.Int) {
	s.setState(common.BigToHash(loc), common.BigToHash(val))
}

func (s *StateHelper) getSlotLoc(loc *big.Int) *big.Int {
	return new(big.Int).SetBytes(crypto.Keccak256(common.BigToHash(loc).Bytes()))
}

func (s *StateHelper) getMapLoc(pos *big.Int, key []byte) *big.Int {
	return new(big.Int).SetBytes(crypto.Keccak256(
		key, common.BigToHash(pos).Bytes()))
}

func (s *StateHelper) readBytes(loc *big.Int) []byte {
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

func (s *StateHelper) writeBytes(loc *big.Int, data []byte) {
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

func (s *StateHelper) read2DByteArray(pos, index *big.Int) [][]byte {
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
func (s *StateHelper) appendTo2DByteArray(pos, index *big.Int, data []byte) {
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

// struct Node {
//     address owner;
//     bytes publicKey;
//     uint256 staked;
// }
//
// Node[] nodes;

type nodeInfo struct {
	owner     common.Address
	publicKey []byte
	staked    *big.Int
}

func (s *StateHelper) nodesLength() *big.Int {
	return s.getStateBigInt(big.NewInt(nodesLoc))
}
func (s *StateHelper) node(index *big.Int) *nodeInfo {
	node := new(nodeInfo)

	arrayBaseLoc := s.getSlotLoc(big.NewInt(nodesLoc))
	elementBaseLoc := new(big.Int).Add(arrayBaseLoc, new(big.Int).Mul(index, big.NewInt(3)))

	// owner.
	loc := elementBaseLoc
	node.owner = common.BytesToAddress(s.getState(common.BigToHash(elementBaseLoc)).Bytes())

	// publicKey.
	loc = new(big.Int).Add(elementBaseLoc, big.NewInt(1))
	node.publicKey = s.readBytes(loc)

	// staked.
	loc = new(big.Int).Add(elementBaseLoc, big.NewInt(2))
	node.staked = s.getStateBigInt(loc)

	return nil
}
func (s *StateHelper) pushNode(n *nodeInfo) {
	// increase length by 1
	arrayLength := s.nodesLength()
	s.setStateBigInt(big.NewInt(nodesLoc), new(big.Int).Add(arrayLength, big.NewInt(1)))

	s.updateNode(arrayLength, n)
}
func (s *StateHelper) updateNode(index *big.Int, n *nodeInfo) {
	arrayBaseLoc := s.getSlotLoc(big.NewInt(nodesLoc))
	elementBaseLoc := new(big.Int).Add(arrayBaseLoc, new(big.Int).Mul(index, big.NewInt(3)))

	// owner.
	loc := elementBaseLoc
	s.setState(common.BigToHash(loc), n.owner.Hash())

	// publicKey.
	loc = new(big.Int).Add(elementBaseLoc, big.NewInt(1))
	s.writeBytes(loc, n.publicKey)

	// staked.
	loc = new(big.Int).Add(elementBaseLoc, big.NewInt(2))
	s.setStateBigInt(loc, n.staked)
}

// mapping(address => uint256) public offset;
func (s *StateHelper) offset(addr common.Address) *big.Int {
	loc := s.getMapLoc(big.NewInt(offsetLoc), addr.Bytes())
	return new(big.Int).Sub(s.getStateBigInt(loc), big.NewInt(1))
}
func (s *StateHelper) putOffset(addr common.Address, offset *big.Int) {
	loc := s.getMapLoc(big.NewInt(offsetLoc), addr.Bytes())
	s.setStateBigInt(loc, new(big.Int).Add(offset, big.NewInt(1)))
}
func (s *StateHelper) deleteOffset(addr common.Address) {
	loc := s.getMapLoc(big.NewInt(offsetLoc), addr.Bytes())
	s.setStateBigInt(loc, big.NewInt(0))
}

// bytes32[] public crs;
func (s *StateHelper) lenCRS() *big.Int {
	return s.getStateBigInt(big.NewInt(crsLoc))
}
func (s *StateHelper) crs(index *big.Int) common.Hash {
	baseLoc := s.getSlotLoc(big.NewInt(crsLoc))
	loc := new(big.Int).Add(baseLoc, index)
	return s.getState(common.BigToHash(loc))
}
func (s *StateHelper) pushCRS(crs common.Hash) {
	// increase length by 1.
	length := s.getStateBigInt(big.NewInt(crsLoc))
	s.setStateBigInt(big.NewInt(crsLoc), new(big.Int).Add(length, big.NewInt(1)))

	baseLoc := s.getSlotLoc(big.NewInt(crsLoc))
	loc := new(big.Int).Add(baseLoc, length)

	s.setState(common.BigToHash(loc), crs)
}

// bytes[][] public dkgMasterPublicKeys;
func (s *StateHelper) dkgMasterPublicKeys(round *big.Int) [][]byte {
	return s.read2DByteArray(big.NewInt(dkgMasterPublicKeysLoc), round)
}
func (s *StateHelper) pushDKGMasterPublicKey(round *big.Int, pk []byte) {
	s.appendTo2DByteArray(big.NewInt(dkgMasterPublicKeysLoc), round, pk)
}

// bytes[][] public dkgComplaints;
func (s *StateHelper) dkgComplaints(round *big.Int) [][]byte {
	return s.read2DByteArray(big.NewInt(dkgComplaintsLoc), round)
}
func (s *StateHelper) pushDKGComplaint(round *big.Int, complaint []byte) {
	s.appendTo2DByteArray(big.NewInt(dkgComplaintsLoc), round, complaint)
}

// mapping(address => bool)[] public dkgFinalized;
func (s *StateHelper) dkgFinalized(round *big.Int, addr common.Address) bool {
	baseLoc := new(big.Int).Add(s.getSlotLoc(big.NewInt(dkgFinailizedLoc)), round)
	mapLoc := s.getMapLoc(baseLoc, addr.Bytes())
	return s.getStateBigInt(mapLoc).Cmp(big.NewInt(0)) != 0
}
func (s *StateHelper) putDKGFinalized(round *big.Int, addr common.Address, finalized bool) {
	baseLoc := new(big.Int).Add(s.getSlotLoc(big.NewInt(dkgFinailizedLoc)), round)
	mapLoc := s.getMapLoc(baseLoc, addr.Bytes())
	res := big.NewInt(0)
	if finalized {
		res = big.NewInt(1)
	}
	s.setStateBigInt(mapLoc, res)
}

// uint256[] public dkgFinalizedsCount;
func (s *StateHelper) dkgFinalizedsCount(round *big.Int) *big.Int {
	loc := new(big.Int).Add(s.getSlotLoc(big.NewInt(dkgFinalizedsCountLoc)), round)
	return s.getStateBigInt(loc)
}
func (s *StateHelper) incDKGFinalizedsCount(round *big.Int) {
	loc := new(big.Int).Add(s.getSlotLoc(big.NewInt(dkgFinalizedsCountLoc)), round)
	count := s.getStateBigInt(loc)
	s.setStateBigInt(loc, new(big.Int).Add(count, big.NewInt(1)))
}

// address public owner;
func (s *StateHelper) owner() common.Address {
	val := s.getState(common.BigToHash(big.NewInt(ownerLoc)))
	return common.BytesToAddress(val.Bytes())
}

// uint256 public numChains;
func (s *StateHelper) numChains() *big.Int {
	return s.getStateBigInt(big.NewInt(numChainsLoc))
}

// uint256 public lambdaBA;
func (s *StateHelper) lambdaBA() *big.Int {
	return s.getStateBigInt(big.NewInt(lambdaBALoc))
}

// uint256 public lambdaDKG;
func (s *StateHelper) lambdaDKG() *big.Int {
	return s.getStateBigInt(big.NewInt(lambdaDKGLoc))
}

// uint256 public k;
func (s *StateHelper) k() *big.Int {
	return s.getStateBigInt(big.NewInt(kLoc))
}

// uint256 public phiRatio;  // stored as PhiRatio * 10^6
func (s *StateHelper) phiRatio() *big.Int {
	return s.getStateBigInt(big.NewInt(phiRatioLoc))
}

// uint256 public notarySetSize;
func (s *StateHelper) notarySetSize() *big.Int {
	return s.getStateBigInt(big.NewInt(notarySetSizeLoc))
}

// uint256 public dkgSetSize;
func (s *StateHelper) dkgSetSize() *big.Int {
	return s.getStateBigInt(big.NewInt(dkgSetSizeLoc))
}

// uint256 public roundInterval
func (s *StateHelper) roundInterval() *big.Int {
	return s.getStateBigInt(big.NewInt(roundIntervalLoc))
}

// uint256 public minBlockInterval
func (s *StateHelper) minBlockInterval() *big.Int {
	return s.getStateBigInt(big.NewInt(minBlockIntervalLoc))
}

// uint256 public maxBlockInterval
func (s *StateHelper) maxBlockInterval() *big.Int {
	return s.getStateBigInt(big.NewInt(maxBlockIntervalLoc))
}

func (s *StateHelper) emitConfigurationChangedEvent() {
	s.StateDB.AddLog(&types.Log{
		Address: s.Address,
		Topics:  []common.Hash{events["ConfigurationChanged"].Id()},
		Data:    []byte{},
	})
}

func (s *StateHelper) emitCRSProposed(round *big.Int, crs common.Hash) {
	s.StateDB.AddLog(&types.Log{
		Address: s.Address,
		Topics:  []common.Hash{events["CRSProposed"].Id(), common.BigToHash(round)},
		Data:    crs.Bytes(),
	})
}

// GovernanceContract represents the governance contract of DEXCON.
type GovernanceContract struct {
	evm      *EVM
	state    StateHelper
	contract *Contract
}

func newGovernanceContract(evm *EVM, contract *Contract) *GovernanceContract {
	return &GovernanceContract{
		evm:      evm,
		state:    StateHelper{contract.Address(), evm.StateDB},
		contract: contract,
	}
}

func (g *GovernanceContract) penalize() {
	g.contract.UseGas(g.contract.Gas)
}

func (g *GovernanceContract) updateConfiguration(config *configParams) ([]byte, error) {
	// Only owner can update configuration.
	if g.contract.Caller() != g.state.owner() {
		return nil, errExecutionReverted
	}

	g.state.setStateBigInt(big.NewInt(numChainsLoc), big.NewInt(int64(config.NumChains)))
	g.state.setStateBigInt(big.NewInt(lambdaBALoc), big.NewInt(int64(config.LambdaBA)))
	g.state.setStateBigInt(big.NewInt(lambdaDKGLoc), big.NewInt(int64(config.LambdaDKG)))
	g.state.setStateBigInt(big.NewInt(kLoc), big.NewInt(int64(config.K)))
	g.state.setStateBigInt(big.NewInt(phiRatioLoc), big.NewInt(int64(config.PhiRatio)))
	g.state.setStateBigInt(big.NewInt(notarySetSizeLoc), big.NewInt(int64(config.NotarySetSize)))
	g.state.setStateBigInt(big.NewInt(dkgSetSizeLoc), big.NewInt(int64(config.DKGSetSize)))
	g.state.setStateBigInt(big.NewInt(roundIntervalLoc), big.NewInt(int64(config.RoundInterval)))
	g.state.setStateBigInt(big.NewInt(minBlockIntervalLoc), big.NewInt(int64(config.MinBlockInterval)))
	g.state.setStateBigInt(big.NewInt(maxBlockIntervalLoc), big.NewInt(int64(config.MaxBlockInterval)))

	g.state.emitConfigurationChangedEvent()
	return nil, nil
}

func (g *GovernanceContract) stake(publicKey []byte) ([]byte, error) {
	caller := g.contract.Caller()
	offset := g.state.offset(caller)

	// Can not stake if already staked.
	if offset.Cmp(big.NewInt(0)) >= 0 {
		g.penalize()
		return nil, errExecutionReverted
	}

	pk, err := crypto.DecompressPubkey(publicKey)
	if err != nil {
		g.penalize()
		return nil, errExecutionReverted
	}

	// Make sure the public key belongs to the caller.
	if crypto.PubkeyToAddress(*pk) != caller {
		g.penalize()
		return nil, errExecutionReverted
	}

	offset = g.state.nodesLength()
	g.state.pushNode(&nodeInfo{
		owner:     caller,
		publicKey: publicKey,
		staked:    g.contract.Value(),
	})
	g.state.putOffset(caller, offset)

	g.contract.UseGas(21000)
	return nil, nil
}

func (g *GovernanceContract) unstake() ([]byte, error) {
	caller := g.contract.Caller()
	offset := g.state.offset(caller)
	if offset.Cmp(big.NewInt(0)) < 0 {
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
	// TODO(w): use OP_CALL so this show up is internal transaction.
	g.evm.Transfer(g.evm.StateDB, GovernanceContractAddress, caller, node.staked)

	g.contract.UseGas(21000)
	return nil, nil
}

func (g *GovernanceContract) proposeCRS(signedCRS []byte) ([]byte, error) {
	round := g.state.lenCRS()
	prevCRS := g.state.crs(round)

	// round should be the next round number, abort otherwise.
	if new(big.Int).Add(round, big.NewInt(1)).Cmp(round) != 0 {
		g.penalize()
		return nil, errExecutionReverted
	}

	// Prepare DKGMasterPublicKeys.
	// TODO(w): make sure DKGMasterPKs are unique.
	var dkgMasterPKs []*coreTypes.DKGMasterPublicKey
	for _, pk := range g.state.dkgMasterPublicKeys(round) {
		x := new(coreTypes.DKGMasterPublicKey)
		if err := rlp.DecodeBytes(pk, x); err != nil {
			panic(err)
		}
		dkgMasterPKs = append(dkgMasterPKs, x)
	}

	// Prepare DKGComplaints.
	var dkgComplaints []*coreTypes.DKGComplaint
	for _, comp := range g.state.dkgComplaints(round) {
		x := new(coreTypes.DKGComplaint)
		if err := rlp.DecodeBytes(comp, x); err != nil {
			panic(err)
		}
		dkgComplaints = append(dkgComplaints, x)
	}

	threshold := int(g.state.dkgSetSize().Uint64() / 3)

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

	g.state.pushCRS(crs)
	g.state.emitCRSProposed(g.state.lenCRS(), crs)

	// To encourage DKG set to propose the correct value, correctly submitting
	// this should cause nothing.
	g.contract.UseGas(0)
	return nil, nil
}

func (g *GovernanceContract) addDKGComplaint(round *big.Int, comp []byte) ([]byte, error) {
	caller := g.contract.Caller()

	// Finalized caller is not allowed to propose complaint.
	if g.state.dkgFinalized(round, caller) {
		g.penalize()
		return nil, errExecutionReverted
	}

	// Calculate 2f
	threshold := new(big.Int).Mul(
		big.NewInt(2),
		new(big.Int).Div(g.state.dkgSetSize(), big.NewInt(3)))

	// If 2f + 1 of DKG set is finalized, one can not propose complaint anymore.
	if g.state.dkgFinalizedsCount(round).Cmp(threshold) > 0 {
		return nil, errExecutionReverted
	}

	var dkgComplaint coreTypes.DKGComplaint
	if err := rlp.DecodeBytes(comp, &dkgComplaint); err != nil {
		g.penalize()
		return nil, errExecutionReverted
	}
	verified, _ := core.VerifyDKGComplaintSignature(&dkgComplaint)
	if !verified {
		g.penalize()
		return nil, errExecutionReverted
	}

	// Verify that the message is sent from the caller.
	signer := nodeIDToAddress(dkgComplaint.ProposerID)
	if signer != caller {
		g.penalize()
		return nil, errExecutionReverted
	}

	g.state.pushDKGComplaint(round, comp)

	// Set this to relatively high to prevent spamming
	g.contract.UseGas(10000000)
	return nil, nil
}

func (g *GovernanceContract) addDKGMasterPublicKey(round *big.Int, pk []byte) ([]byte, error) {
	caller := g.contract.Caller()
	offset := g.state.offset(caller)

	// Can not add dkg mpk if not staked.
	if offset.Cmp(big.NewInt(0)) < 0 {
		return nil, errExecutionReverted
	}

	var dkgMasterPK coreTypes.DKGMasterPublicKey
	if err := rlp.DecodeBytes(pk, &dkgMasterPK); err != nil {
		g.penalize()
		return nil, errExecutionReverted
	}
	verified, _ := core.VerifyDKGMasterPublicKeySignature(&dkgMasterPK)
	if !verified {
		g.penalize()
		return nil, errExecutionReverted
	}

	// Verify that the message is sent from the caller.
	signer := nodeIDToAddress(dkgMasterPK.ProposerID)
	if signer != caller {
		g.penalize()
		return nil, errExecutionReverted
	}

	g.state.pushDKGMasterPublicKey(round, pk)

	// DKG operation is expensive.
	g.contract.UseGas(100000)
	return nil, nil
}

func (g *GovernanceContract) addDKGFinalize(round *big.Int, finalize []byte) ([]byte, error) {
	caller := g.contract.Caller()

	var dkgFinalize coreTypes.DKGFinalize
	if err := rlp.DecodeBytes(finalize, &dkgFinalize); err != nil {
		g.penalize()
		return nil, errExecutionReverted
	}
	verified, _ := core.VerifyDKGFinalizeSignature(&dkgFinalize)
	if !verified {
		g.penalize()
		return nil, errExecutionReverted
	}

	// Verify that the message is sent from the caller.
	signer := nodeIDToAddress(dkgFinalize.ProposerID)
	if signer != caller {
		g.penalize()
		return nil, errExecutionReverted
	}

	if !g.state.dkgFinalized(round, caller) {
		g.state.putDKGFinalized(round, caller, true)
		g.state.incDKGFinalizedsCount(round)
	}

	// DKG operation is expensive.
	g.contract.UseGas(100000)
	return nil, nil
}
