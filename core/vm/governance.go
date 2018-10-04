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
    "name": "numNotarySet",
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
    "inputs": [],
    "name": "numDKGSet",
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
    "constant": false,
    "inputs": [
      {
        "name": "numChains",
        "type": "int256"
      },
      {
        "name": "lambdaBA",
        "type": "int256"
      },
      {
        "name": "lambdaDKG",
        "type": "int256"
      },
      {
        "name": "K",
        "type": "int256"
      },
      {
        "name": "PhiRatio",
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
        "name": "round",
        "type": "uint256"
      },
      {
        "name": "publicKey",
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
        "name": "round",
        "type": "uint256"
      },
      {
        "name": "complaint",
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
        "name": "publicKey",
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
		sig2Method[string(method.Id())] = method
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
			Round     *big.Int
			PublicKey []byte
		}{}
		if err := method.Inputs.Unpack(&args, argument); err != nil {
			return nil, errExecutionReverted
		}
		return g.addDKGMasterPublicKey(args.Round, args.PublicKey)
	case "addDKGComplaint":
		args := struct {
			Round     *big.Int
			Complaint []byte
		}{}
		if err := method.Inputs.Unpack(&args, argument); err != nil {
			return nil, errExecutionReverted
		}
		return g.addDKGComplaint(args.Round, args.Complaint)
	}
	return nil, nil
}

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
	return s.getStateBigInt(big.NewInt(0))
}
func (s *StateHelper) node(index *big.Int) *nodeInfo {
	node := new(nodeInfo)

	arrayBaseLoc := s.getSlotLoc(big.NewInt(0))
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
	s.setStateBigInt(big.NewInt(0), new(big.Int).Add(arrayLength, big.NewInt(1)))

	s.updateNode(arrayLength, n)
}
func (s *StateHelper) updateNode(index *big.Int, n *nodeInfo) {
	arrayBaseLoc := s.getSlotLoc(big.NewInt(0))
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

// 1: mapping(address => uint256) public offset;
func (s *StateHelper) offset(addr common.Address) *big.Int {
	loc := s.getMapLoc(big.NewInt(1), addr.Bytes())
	return new(big.Int).Sub(s.getStateBigInt(loc), big.NewInt(1))
}
func (s *StateHelper) putOffset(addr common.Address, offset *big.Int) {
	loc := s.getMapLoc(big.NewInt(1), addr.Bytes())
	s.setStateBigInt(loc, new(big.Int).Add(offset, big.NewInt(1)))
}
func (s *StateHelper) deleteOffset(addr common.Address) {
	loc := s.getMapLoc(big.NewInt(1), addr.Bytes())
	s.setStateBigInt(loc, big.NewInt(0))
}

// 2: mapping(uint256 => bytes32) public crs;
func (s *StateHelper) crs(round *big.Int) common.Hash {
	loc := s.getMapLoc(big.NewInt(2), common.BigToHash(round).Bytes())
	return s.getState(common.BigToHash(loc))
}
func (s *StateHelper) putCRS(round *big.Int, crs common.Hash) {
	loc := s.getMapLoc(big.NewInt(2), common.BigToHash(round).Bytes())
	s.setState(common.BigToHash(loc), crs)
}

// 3: mapping(uint256 => bytes[]) public DKGMasterPublicKeys;
func (s *StateHelper) dkgMasterPublicKeys(round *big.Int) [][]byte {
	loc := s.getMapLoc(big.NewInt(3), common.BigToHash(round).Bytes())

	arrayLength := s.getStateBigInt(loc)
	dataLoc := s.getSlotLoc(loc)

	data := make([][]byte, arrayLength.Uint64())
	for i := int64(0); i < int64(arrayLength.Uint64()); i++ {
		elementLoc := new(big.Int).Add(dataLoc, big.NewInt(i))
		data = append(data, s.readBytes(elementLoc))
	}
	return data
}
func (s *StateHelper) pushDKGMasterPublicKey(round *big.Int, pk []byte) {
	loc := s.getMapLoc(big.NewInt(3), common.BigToHash(round).Bytes())

	// increase length by 1.
	arrayLength := s.getStateBigInt(loc)
	s.setStateBigInt(loc, new(big.Int).Add(arrayLength, big.NewInt(1)))

	// write element.
	dataLoc := s.getSlotLoc(loc)
	elementLoc := new(big.Int).Add(dataLoc, arrayLength)
	s.writeBytes(elementLoc, pk)
}

// 4: mapping(uint256 => bytes[]) public DKGComplaints;
func (s *StateHelper) dkgComplaints(round *big.Int) [][]byte {
	loc := s.getMapLoc(big.NewInt(4), common.BigToHash(round).Bytes())

	arrayLength := s.getStateBigInt(loc)
	dataLoc := s.getSlotLoc(loc)

	data := make([][]byte, arrayLength.Uint64())
	for i := int64(0); i < int64(arrayLength.Uint64()); i++ {
		elementLoc := new(big.Int).Add(dataLoc, big.NewInt(i))
		data = append(data, s.readBytes(elementLoc))
	}
	return data
}
func (s *StateHelper) addDKGComplaint(round *big.Int, complaint []byte) {
	loc := s.getMapLoc(big.NewInt(3), common.BigToHash(round).Bytes())

	// increase length by 1.
	arrayLength := s.getStateBigInt(loc)
	s.setStateBigInt(loc, new(big.Int).Add(arrayLength, big.NewInt(1)))

	// write element.
	dataLoc := s.getSlotLoc(loc)
	elementLoc := new(big.Int).Add(dataLoc, arrayLength)
	s.writeBytes(elementLoc, complaint)
}

// 5: uint256 public round;
func (s *StateHelper) round() *big.Int {
	return s.getStateBigInt(big.NewInt(5))
}
func (s *StateHelper) incRound() *big.Int {
	newRound := new(big.Int).Add(s.round(), big.NewInt(1))
	s.setStateBigInt(big.NewInt(5), newRound)
	return newRound
}

// 6: address public governanceMultisig;
func (s *StateHelper) governanceMultisig() common.Address {
	val := s.getState(common.BigToHash(big.NewInt(6)))
	return common.BytesToAddress(val.Bytes())
}

// 7: uint256 public numChains;
func (s *StateHelper) numChains() *big.Int {
	return s.getStateBigInt(big.NewInt(7))
}

// 8: uint256 public lambdaBA;
func (s *StateHelper) lambdaBA() *big.Int {
	return s.getStateBigInt(big.NewInt(8))
}

// 9: uint256 public lambdaDKG;
func (s *StateHelper) lambdaDKG() *big.Int {
	return s.getStateBigInt(big.NewInt(9))
}

// 10: uint256 public k;
func (s *StateHelper) k() *big.Int {
	return s.getStateBigInt(big.NewInt(10))
}

// 11: uint256 public phiRatio;  // stored as PhiRatio * 10^6
func (s *StateHelper) phiRatio() *big.Int {
	return s.getStateBigInt(big.NewInt(11))
}

// 12: uint256 public numNotarySet;
func (s *StateHelper) numNotarySet() *big.Int {
	return s.getStateBigInt(big.NewInt(12))
}

// 13: uint256 public numDKGSet;
func (s *StateHelper) numDKGSet() *big.Int {
	return s.getStateBigInt(big.NewInt(13))
}

// 14: uint256 public roundInterval
func (s *StateHelper) roundInterval() *big.Int {
	return s.getStateBigInt(big.NewInt(14))
}

// 15: uint256 public minBlockInterval
func (s *StateHelper) minBlockInterval() *big.Int {
	return s.getStateBigInt(big.NewInt(15))
}

// 16: uint256 public maxBlockInterval
func (s *StateHelper) maxBlockInterval() *big.Int {
	return s.getStateBigInt(big.NewInt(16))
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

func (g *GovernanceContract) updateConfiguration() ([]byte, error) {
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

	// TODO(w): check of pk belongs to the address.
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
		g.penalize()
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
		g.penalize()
		return nil, errExecutionReverted
	}

	// Save new CRS into state and increase round.
	newCRS := crypto.Keccak256(signedCRS)
	g.state.incRound()
	g.state.putCRS(round, common.BytesToHash(newCRS))

	// To encourage DKG set to propose the correct value, correctly submitting
	// this should cause nothing.
	g.contract.UseGas(0)
	return nil, nil
}

func (g *GovernanceContract) addDKGMasterPublicKey(round *big.Int, pk []byte) ([]byte, error) {
	var dkgMasterPK types.DKGMasterPublicKey
	if err := rlp.DecodeBytes(pk, &dkgMasterPK); err != nil {
		g.penalize()
		return nil, errExecutionReverted
	}
	verified, _ := core.VerifyDKGMasterPublicKeySignature(&dkgMasterPK)
	if !verified {
		g.penalize()
		return nil, errExecutionReverted
	}

	g.state.pushDKGMasterPublicKey(round, pk)

	// DKG operation is expensive.
	g.contract.UseGas(100000)
	return nil, nil
}

func (g *GovernanceContract) addDKGComplaint(round *big.Int, comp []byte) ([]byte, error) {
	var dkgComplaint types.DKGComplaint
	if err := rlp.DecodeBytes(comp, &dkgComplaint); err != nil {
		return nil, errExecutionReverted
	}
	verified, _ := core.VerifyDKGComplaintSignature(&dkgComplaint)
	if !verified {
		g.penalize()
		return nil, errExecutionReverted
	}

	g.state.addDKGComplaint(round, comp)

	// Set this to relatively high to prevent spamming
	g.contract.UseGas(10000000)
	return nil, nil
}
