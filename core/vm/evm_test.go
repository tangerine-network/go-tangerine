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
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/dexon-foundation/dexon/accounts/abi"
	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/core/state"
	"github.com/dexon-foundation/dexon/crypto"
	"github.com/dexon-foundation/dexon/ethdb"
	"github.com/dexon-foundation/dexon/params"
)

var cobContract = string("608060405234801561001057600080fd5b5033600360006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055506123ba806100616000396000f300608060405260043610610154576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806306fdde0314610160578063078fd9ea146101f0578063095ea7b31461021b5780630b97bc861461028057806318160ddd146102ab57806323b872dd146102d6578063313ce5671461035b5780634042b66f1461038c5780634bb278f3146103b7578063521eb273146103ce578063661884631461042557806368428a1b1461048a57806370a08231146104b9578063715018a6146105105780638da5cb5b1461052757806395d89b411461057e578063a77c61f21461060e578063a9059cbb1461066d578063b52e0dc8146106d2578063b753a98c14610713578063c24a0f8b14610760578063c3262dfd1461078b578063d73dd623146107bc578063dd62ed3e14610821578063f2fde38b14610898578063f92ad219146108db575b61015e3334610946565b005b34801561016c57600080fd5b50610175610b97565b6040518080602001828103825283818151815260200191508051906020019080838360005b838110156101b557808201518184015260208101905061019a565b50505050905090810190601f1680156101e25780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b3480156101fc57600080fd5b50610205610bd0565b6040518082815260200191505060405180910390f35b34801561022757600080fd5b50610266600480360381019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080359060200190929190505050610bd6565b604051808215151515815260200191505060405180910390f35b34801561028c57600080fd5b50610295610cc8565b6040518082815260200191505060405180910390f35b3480156102b757600080fd5b506102c0610cce565b6040518082815260200191505060405180910390f35b3480156102e257600080fd5b50610341600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080359060200190929190505050610cd8565b604051808215151515815260200191505060405180910390f35b34801561036757600080fd5b50610370611093565b604051808260ff1660ff16815260200191505060405180910390f35b34801561039857600080fd5b506103a1611098565b6040518082815260200191505060405180910390f35b3480156103c357600080fd5b506103cc61109e565b005b3480156103da57600080fd5b506103e361123f565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b34801561043157600080fd5b50610470600480360381019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080359060200190929190505050611265565b604051808215151515815260200191505060405180910390f35b34801561049657600080fd5b5061049f6114f7565b604051808215151515815260200191505060405180910390f35b3480156104c557600080fd5b506104fa600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050611532565b6040518082815260200191505060405180910390f35b34801561051c57600080fd5b5061052561157a565b005b34801561053357600080fd5b5061053c61167f565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b34801561058a57600080fd5b506105936116a5565b6040518080602001828103825283818151815260200191508051906020019080838360005b838110156105d35780820151818401526020810190506105b8565b50505050905090810190601f1680156106005780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b34801561061a57600080fd5b5061064f600480360381019080803573ffffffffffffffffffffffffffffffffffffffff1690602001909291905050506116de565b60405180826000191660001916815260200191505060405180910390f35b34801561067957600080fd5b506106b8600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803590602001909291905050506116f6565b604051808215151515815260200191505060405180910390f35b3480156106de57600080fd5b506106fd60048036038101908080359060200190929190505050611916565b6040518082815260200191505060405180910390f35b34801561071f57600080fd5b5061075e600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803590602001909291905050506119ae565b005b34801561076c57600080fd5b50610775611c35565b6040518082815260200191505060405180910390f35b34801561079757600080fd5b506107ba6004803603810190808035600019169060200190929190505050611c3b565b005b3480156107c857600080fd5b50610807600480360381019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080359060200190929190505050611cf9565b604051808215151515815260200191505060405180910390f35b34801561082d57600080fd5b50610882600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050611ef5565b6040518082815260200191505060405180910390f35b3480156108a457600080fd5b506108d9600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050611f7c565b005b3480156108e757600080fd5b50610944600480360381019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080359060200190929190803590602001909291908035906020019092919080359060200190929190505050611fe4565b005b6000806000806109546114f7565b151561095f57600080fd5b67016345785d8a0000851015151561097657600080fd5b84935061098e846009546121ec90919063ffffffff16565b92506109a061099b612208565b611916565b91506109b5828561221090919063ffffffff16565b9050806109c0612248565b101515156109cd57600080fd5b610a098160008060b173ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205461227990919063ffffffff16565b60008060b173ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002081905550610a87816000808973ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020546121ec90919063ffffffff16565b6000808873ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508573ffffffffffffffffffffffffffffffffffffffff167fcd60aa75dea3072fbc07ae6d7d856b5dc5f4eee88854f5b4abf7b680ef8bc50f8583604051808381526020018281526020019250505060405180910390a282600981905550600860009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166108fc349081150290604051600060405180830381858888f19350505050158015610b8e573d6000803e3d6000fd5b50505050505050565b6040805190810160405280600f81526020017f436f62696e686f6f6420546f6b656e000000000000000000000000000000000081525081565b60075481565b600081600260003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508273ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff167f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925846040518082815260200191505060405180910390a36001905092915050565b60055481565b6000600454905090565b60008060008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020548211151515610d2757600080fd5b600260008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020548211151515610db257600080fd5b600073ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff1614151515610dee57600080fd5b610e3f826000808773ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205461227990919063ffffffff16565b6000808673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002081905550610ed2826000808673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020546121ec90919063ffffffff16565b6000808573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002081905550610fa382600260008773ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205461227990919063ffffffff16565b600260008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508273ffffffffffffffffffffffffffffffffffffffff168473ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef846040518082815260200191505060405180910390a3600190509392505050565b601281565b60095481565b600360009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff161415156110fa57600080fd5b6111026114f7565b15151561110e57600080fd5b6111aa60008060b173ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054600080600860009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020546121ec90919063ffffffff16565b600080600860009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002081905550600080600060b173ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002081905550565b600860009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b600080600260003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205490508083101515611377576000600260003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000208190555061140b565b61138a838261227990919063ffffffff16565b600260003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055505b8373ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff167f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925600260003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008873ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020546040518082815260200191505060405180910390a3600191505092915050565b6000600554611504612208565b1015801561151a5750600654611518612208565b105b801561152d5750600061152b612248565b115b905090565b60008060008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020549050919050565b600360009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff161415156115d657600080fd5b600360009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff167ff8df31144d9c2f0f6b59d69b8b98abd5459d07f2742c4df920b25aae33c6482060405160405180910390a26000600360006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550565b600360009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b6040805190810160405280600381526020017f434f42000000000000000000000000000000000000000000000000000000000081525081565b600a6020528060005260406000206000915090505481565b60008060003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054821115151561174557600080fd5b600073ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff161415151561178157600080fd5b6117d2826000803373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205461227990919063ffffffff16565b6000803373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002081905550611865826000808673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020546121ec90919063ffffffff16565b6000808573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508273ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef846040518082815260200191505060405180910390a36001905092915050565b600060055482101561192b57600090506119a9565b62093a8060055401821015611944576115e090506119a9565b621275006005540182101561195d5761145090506119a9565b621baf8060055401821015611976576112c090506119a9565b6224ea006005540182101561198f5761113090506119a9565b600654821115156119a457610fa090506119a9565b600090505b919050565b600360009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16141515611a0a57600080fd5b80600080600860009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205410151515611a7957600080fd5b611aec81600080600860009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205461227990919063ffffffff16565b600080600860009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002081905550611ba1816000808573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020546121ec90919063ffffffff16565b6000808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508173ffffffffffffffffffffffffffffffffffffffff167fdb2d10a559cb6e14fee5a7a2d8c216314e11c22404e85a4f9af45f07c87192bb826040518082815260200191505060405180910390a25050565b60065481565b80600a60003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002081600019169055507fa1a3e4c7b21b6004c77b4fe18bdd0d1bd1be31dbb88112463524daa9abacb8363382604051808373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200182600019166000191681526020019250505060405180910390a150565b6000611d8a82600260003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020546121ec90919063ffffffff16565b600260003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508273ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff167f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925600260003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008773ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020546040518082815260200191505060405180910390a36001905092915050565b6000600260008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054905092915050565b600360009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16141515611fd857600080fd5b611fe181612292565b50565b600360009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614151561204057600080fd5b6000600860009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1614151561208757600080fd5b61208f612208565b841015151561209d57600080fd5b82841015156120ab57600080fd5b60008573ffffffffffffffffffffffffffffffffffffffff16141515156120d157600080fd5b81811115156120df57600080fd5b83600581905550826006819055508160078190555084600860006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550806004819055506121516007548261227990919063ffffffff16565b600080600860009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000208190555060075460008060b173ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055505050505050565b600081830190508281101515156121ff57fe5b80905092915050565b600042905090565b6000808314156122235760009050612242565b818302905081838281151561223457fe5b0414151561223e57fe5b8090505b92915050565b600080600060b173ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054905090565b600082821115151561228757fe5b818303905092915050565b600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff16141515156122ce57600080fd5b8073ffffffffffffffffffffffffffffffffffffffff16600360009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e060405160405180910390a380600360006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550505600a165627a7a723058209accf1a69ff5978107f5013b9b8716cc08726d534484a57fc04c94d084be8cf80029")
var cobABIJSON = `
[{"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"saleCap","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"_spender","type":"address"},{"name":"_value","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"startDate","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"_from","type":"address"},{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transferFrom","outputs":[{"name":"","type":"bool"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"weiRaised","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":false,"inputs":[],"name":"finalize","outputs":[],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"wallet","outputs":[{"name":"","type":"address"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"saleActive","outputs":[{"name":"","type":"bool"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"balance","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"owner","outputs":[{"name":"","type":"address"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"","type":"address"}],"name":"cobinhoodUserIDs","outputs":[{"name":"","type":"bytes32"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transfer","outputs":[{"name":"","type":"bool"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"at","type":"uint256"}],"name":"getRateAt","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"buyer","type":"address"},{"name":"amount","type":"uint256"}],"name":"push","outputs":[],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"endDate","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"user_id","type":"bytes32"}],"name":"setUserID","outputs":[],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"_owner","type":"address"},{"name":"_spender","type":"address"}],"name":"allowance","outputs":[{"name":"remaining","type":"uint256"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"newOwner","type":"address"}],"name":"transferOwnership","outputs":[],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"_wallet","type":"address"},{"name":"_start","type":"uint256"},{"name":"_end","type":"uint256"},{"name":"_saleCap","type":"uint256"},{"name":"_totalSupply","type":"uint256"}],"name":"initialize","outputs":[],"payable":false,"type":"function"},{"inputs":[],"payable":false,"type":"constructor"},{"payable":true,"type":"fallback"},{"anonymous":false,"inputs":[{"indexed":true,"name":"purchaser","type":"address"},{"indexed":false,"name":"value","type":"uint256"},{"indexed":false,"name":"amount","type":"uint256"}],"name":"TokenPurchase","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"buyer","type":"address"},{"indexed":false,"name":"amount","type":"uint256"}],"name":"PreICOTokenPushed","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"name":"owner","type":"address"},{"indexed":false,"name":"user_id","type":"bytes32"}],"name":"UserIDChanged","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"owner","type":"address"},{"indexed":true,"name":"spender","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Approval","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"}]
`
var cobABI abi.ABI
var cobSaleCap *big.Int
var cobTotalSupply *big.Int
var betContract = string("60806040523480156200001157600080fd5b5060405160208062000ee083398101806040528101908080519060200190929190505050336000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055506000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16600073ffffffffffffffffffffffffffffffffffffffff167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e060405160405180910390a36200010b8162000112640100000000026401000000009004565b50620003f5565b60006200011e62000364565b6000620001396200029f640100000000026401000000009004565b15156200014557600080fd5b606484101515620001e4576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260248152602001807f4578706563746174696f6e2073686f756c64206265206c657373207468616e2081526020017f3130302e0000000000000000000000000000000000000000000000000000000081525060400191505060405180910390fd5b836001819055506200021061271085620002f664010000000002620008c4179091906401000000009004565b925060008260006064811015156200022457fe5b602002018181525050600190505b606481101562000285576200025f8184620003386401000000000262000902179091906401000000009004565b82826064811015156200026e57fe5b602002018181525050808060010191505062000232565b8160029060646200029892919062000388565b5050505050565b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614905090565b60008060008414156200030d576000915062000331565b82840290508284828115156200031f57fe5b041415156200032d57600080fd5b8091505b5092915050565b6000806000831115156200034b57600080fd5b82848115156200035757fe5b0490508091505092915050565b610c8060405190810160405280606490602082028038833980820191505090505090565b8260648101928215620003ba579160200282015b82811115620003b95782518255916020019190600101906200039c565b5b509050620003c99190620003cd565b5090565b620003f291905b80821115620003ee576000816000905550600101620003d4565b5090565b90565b610adb80620004056000396000f3006080604052600436106100a4576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680630c60e0c3146100a9578063379607f5146100d6578063715018a6146101035780637365870b1461011a5780638da5cb5b1461013a5780638f32d59b14610191578063e1152343146101c0578063ed88c68e14610201578063f2fde38b1461020b578063fc1c39dc1461024e575b600080fd5b3480156100b557600080fd5b506100d460048036038101908080359060200190929190505050610279565b005b3480156100e257600080fd5b50610101600480360381019080803590602001909291905050506103cb565b005b34801561010f57600080fd5b506101186104be565b005b61013860048036038101908080359060200190929190505050610590565b005b34801561014657600080fd5b5061014f610803565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b34801561019d57600080fd5b506101a661082c565b604051808215151515815260200191505060405180910390f35b3480156101cc57600080fd5b506101eb60048036038101908080359060200190929190505050610883565b6040518082815260200191505060405180910390f35b61020961089d565b005b34801561021757600080fd5b5061024c600480360381019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919050505061089f565b005b34801561025a57600080fd5b506102636108be565b6040518082815260200191505060405180910390f35b6000610283610a26565b600061028d61082c565b151561029857600080fd5b606484101515610336576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260248152602001807f4578706563746174696f6e2073686f756c64206265206c657373207468616e2081526020017f3130302e0000000000000000000000000000000000000000000000000000000081525060400191505060405180910390fd5b83600181905550610352612710856108c490919063ffffffff16565b9250600082600060648110151561036557fe5b602002018181525050600190505b60648110156103b35761038f818461090290919063ffffffff16565b828260648110151561039d57fe5b6020020181815250508080600101915050610373565b8160029060646103c4929190610a4a565b5050505050565b6103d361082c565b15156103de57600080fd5b3073ffffffffffffffffffffffffffffffffffffffff1631811115151561046d576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040180806020018281038252601c8152602001807f4e6f20656e6f756768206f6620746f6b656e20746f20636c61696d2e0000000081525060200191505060405180910390fd5b610475610803565b73ffffffffffffffffffffffffffffffffffffffff166108fc829081150290604051600060405180830381858888f193505050501580156104ba573d6000803e3d6000fd5b5050565b6104c661082c565b15156104d157600080fd5b600073ffffffffffffffffffffffffffffffffffffffff166000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e060405160405180910390a360008060006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550565b60008060018311151561060b576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260188152602001807f5461726765742073686f756c64206265206269676765722e000000000000000081525060200191505060405180910390fd5b6001548311151515610685576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260198152602001807f5461726765742073686f756c6420626520736d616c6c65722e0000000000000081525060200191505060405180910390fd5b612710341115156106fe576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260158152602001807f4d696e696d756d206265742069732031303030302e000000000000000000000081525060200191505060405180910390fd5b600160642f81151561070c57fe5b0601915060009050828210156107a05761075661271061074860026001870360648110151561073757fe5b0154346108c490919063ffffffff16565b61090290919063ffffffff16565b90503373ffffffffffffffffffffffffffffffffffffffff166108fc829081150290604051600060405180830381858888f1935050505015801561079e573d6000803e3d6000fd5b505b3373ffffffffffffffffffffffffffffffffffffffff167f97371a3349bea11f577edf6e64350a3dfb9de665d1154c7e6d08eb0805aa043084348460405180848152602001838152602001828152602001935050505060405180910390a2505050565b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905090565b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614905090565b60028160648110151561089257fe5b016000915090505481565b565b6108a761082c565b15156108b257600080fd5b6108bb8161092c565b50565b60015481565b60008060008414156108d957600091506108fb565b82840290508284828115156108ea57fe5b041415156108f757600080fd5b8091505b5092915050565b60008060008311151561091457600080fd5b828481151561091f57fe5b0490508091505092915050565b600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff161415151561096857600080fd5b8073ffffffffffffffffffffffffffffffffffffffff166000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e060405160405180910390a3806000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555050565b610c8060405190810160405280606490602082028038833980820191505090505090565b8260648101928215610a79579160200282015b82811115610a78578251825591602001919060010190610a5d565b5b509050610a869190610a8a565b5090565b610aac91905b80821115610aa8576000816000905550600101610a90565b5090565b905600a165627a7a7230582087d19571ab3ae7207fb8f6182b6b891f2414f00e1904790341a82c957044b5330029")
var betConstructor = []string{"62"}
var betABIJSON = `
[ { "constant": false, "inputs": [], "name": "renounceOwnership", "outputs": [], "payable": false, "stateMutability": "nonpayable", "type": "function" }, { "constant": true, "inputs": [], "name": "owner", "outputs": [ { "name": "", "type": "address" } ], "payable": false, "stateMutability": "view", "type": "function" }, { "constant": true, "inputs": [], "name": "isOwner", "outputs": [ { "name": "", "type": "bool" } ], "payable": false, "stateMutability": "view", "type": "function" }, { "constant": true, "inputs": [ { "name": "", "type": "uint256" } ], "name": "payout", "outputs": [ { "name": "", "type": "uint256" } ], "payable": false, "stateMutability": "view", "type": "function" }, { "constant": false, "inputs": [ { "name": "newOwner", "type": "address" } ], "name": "transferOwnership", "outputs": [], "payable": false, "stateMutability": "nonpayable", "type": "function" }, { "constant": true, "inputs": [], "name": "expectation", "outputs": [ { "name": "", "type": "uint256" } ], "payable": false, "stateMutability": "view", "type": "function" }, { "inputs": [ { "name": "_expectation", "type": "uint256" } ], "payable": false, "stateMutability": "nonpayable", "type": "constructor" }, { "anonymous": false, "inputs": [ { "indexed": true, "name": "_addr", "type": "address" }, { "indexed": false, "name": "_target", "type": "uint256" }, { "indexed": false, "name": "_value", "type": "uint256" }, { "indexed": false, "name": "_pay", "type": "uint256" } ], "name": "Bet", "type": "event" }, { "anonymous": false, "inputs": [ { "indexed": true, "name": "previousOwner", "type": "address" }, { "indexed": true, "name": "newOwner", "type": "address" } ], "name": "OwnershipTransferred", "type": "event" }, { "constant": false, "inputs": [ { "name": "_expectation", "type": "uint256" } ], "name": "updateExpectation", "outputs": [], "payable": false, "stateMutability": "nonpayable", "type": "function" }, { "constant": false, "inputs": [ { "name": "_target", "type": "uint256" } ], "name": "bet", "outputs": [], "payable": true, "stateMutability": "payable", "type": "function" }, { "constant": false, "inputs": [ { "name": "_amount", "type": "uint256" } ], "name": "claim", "outputs": [], "payable": false, "stateMutability": "nonpayable", "type": "function" }, { "constant": false, "inputs": [], "name": "donate", "outputs": [], "payable": true, "stateMutability": "payable", "type": "function" } ]
`
var betABI abi.ABI

var oneDEX *big.Int

func init() {
	oneDEX = new(big.Int).Exp(big.NewInt(10), big.NewInt(19), nil)
	var err error
	cobABI, err = abi.JSON(strings.NewReader(cobABIJSON))
	if err != nil {
		panic(err)
	}
	var ok bool
	cobSaleCap, ok = new(big.Int).SetString("19d971e4fe8401e74000000", 16)
	if !ok {
		panic(fmt.Errorf("error setting cobSaleCap"))
	}
	cobTotalSupply, ok = new(big.Int).SetString("33b2e3c9fd0803ce8000000", 16)
	if !ok {
		panic(fmt.Errorf("error setting cobTotalSupply"))
	}
	betABI, err = abi.JSON(strings.NewReader(betABIJSON))
	if err != nil {
		panic(err)
	}
}

type testVM struct {
	evm         *EVM
	interpreter Interpreter
}

func newTestVM() *testVM {
	memDB := ethdb.NewMemDatabase()
	stateDB, err := state.New(common.Hash{}, state.NewDatabase(memDB))
	if err != nil {
		panic(err)
	}

	context := Context{
		CanTransfer: func(StateDB, common.Address, *big.Int) bool { return true },
		Transfer:    func(StateDB, common.Address, common.Address, *big.Int) {},
		Time:        big.NewInt(time.Now().UnixNano() / 1000000000),
		BlockNumber: big.NewInt(0),
	}

	env := NewEVM(context, stateDB, params.TestChainConfig, Config{})
	evmInterpreter := NewEVMInterpreter(env, env.vmConfig)

	env.interpreter = evmInterpreter

	return &testVM{
		evm:         env,
		interpreter: evmInterpreter,
	}
}

func (vm *testVM) create(caller string, code []byte, value *big.Int) (
	ret []byte, contractAddr common.Address, err error) {
	callerAddr := common.HexToAddress(caller)
	contractAddr = crypto.CreateAddress(callerAddr, uint64(0))
	contract := NewContract(AccountRef(callerAddr),
		AccountRef(contractAddr), value, math.MaxUint64)
	contract.SetCodeOptionalHash(&callerAddr, &codeAndHash{code: code})
	ret, err = vm.interpreter.Run(contract, nil, false)
	if err != nil {
		contractAddr = common.Address{}
		return
	}
	vm.evm.StateDB.SetCode(contractAddr, ret)
	return
}

func (vm *testVM) call(
	caller string, addr common.Address, input []byte, value *big.Int) (
	ret []byte, err error) {
	contract := vm.createContract(caller, addr, value)
	return vm.callContract(contract, input)
}

func (vm *testVM) createContract(
	caller string, addr common.Address, value *big.Int) *Contract {
	callerAddr := common.HexToAddress(caller)
	vm.evm.StateDB.CreateAccount(callerAddr)
	contract := NewContract(AccountRef(callerAddr),
		AccountRef(addr), value, math.MaxUint64)
	contract.SetCallCode(
		&addr, vm.evm.StateDB.GetCodeHash(addr), vm.evm.StateDB.GetCode(addr))
	return contract
}

func (vm *testVM) callContract(contract *Contract, input []byte) (
	ret []byte, err error) {
	if len(contract.Code) == 0 {
		panic(fmt.Errorf("no code"))
	}

	if !vm.interpreter.CanRun(contract.Code) {
		panic(fmt.Errorf("cannot run"))
	}
	return vm.interpreter.Run(contract, input, false)
}

func BenchmarkContractCOBTransfer(bench *testing.B) {
	vm := newTestVM()
	owner := "1"
	wallet := "2"
	addresses := []common.Address{}
	for i := 0; i < 100; i++ {
		addr := crypto.CreateAddress(common.HexToAddress("0x123"), uint64(i))
		addresses = append(addresses, addr)
	}
	_, contractAddr, err := vm.create(owner, common.Hex2Bytes(cobContract), new(big.Int))
	if err != nil {
		bench.Error(err)
		return
	}
	start := time.Now().Add(86400*time.Second).UnixNano() / 1000000000
	end := time.Now().Add(86400*2*time.Second).UnixNano() / 1000000000
	input, err := cobABI.Pack("initialize", common.HexToAddress(wallet), big.NewInt(start), big.NewInt(end), cobSaleCap, cobTotalSupply)
	if err != nil {
		bench.Error(err)
		return
	}
	_, err = vm.call(owner, contractAddr, input, new(big.Int))
	if err != nil {
		bench.Error(err)
		return
	}

	transferContract := vm.createContract(wallet, contractAddr, new(big.Int))
	transferCalls := make([][]byte, len(addresses))
	for i, addr := range addresses {
		transferCalls[i], err = cobABI.Pack("transfer", addr, big.NewInt(1))
		if err != nil {
			bench.Error(err)
			return
		}
	}

	bench.ResetTimer()
	for i := 0; i < bench.N; i++ {
		transferCall := transferCalls[rand.Intn(len(transferCalls))]
		_, err = vm.callContract(transferContract, transferCall)
	}
	bench.StopTimer()
	//Check if it is correct
	if err != nil {
		bench.Error(err)
		return
	}
}

func contractConstructor(ctor []string) (output string) {
	for _, input := range ctor {
		output += fmt.Sprintf("%064s", input)
	}
	return
}

func BenchmarkContractDEXONBet(bench *testing.B) {
	vm := newTestVM()
	owner := "1"
	gambler := "2"
	_, contractAddr, err := vm.create(owner, common.Hex2Bytes(betContract+contractConstructor(betConstructor)), new(big.Int))
	if err != nil {
		bench.Error(err)
		return
	}
	input, err := betABI.Pack("donate")
	if err != nil {
		bench.Error(err)
		return
	}
	_, err = vm.call(owner, contractAddr, input, new(big.Int).Mul(big.NewInt(100), oneDEX))
	if err != nil {
		bench.Error(err)
		return
	}

	betContract := vm.createContract(gambler, contractAddr, new(big.Int).Set(oneDEX))
	betCall, err := betABI.Pack("bet", big.NewInt(50))
	if err != nil {
		bench.Error(err)
		return
	}

	bench.ResetTimer()
	for i := 0; i < bench.N; i++ {
		_, err = vm.callContract(betContract, betCall)
		if err != nil {
			bench.Error(err)
			return
		}
	}
	bench.StopTimer()
}
