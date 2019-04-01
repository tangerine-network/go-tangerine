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

package dex

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/dexon-foundation/dexon/accounts/abi"
	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/core/types"
	"github.com/dexon-foundation/dexon/crypto"
	"github.com/dexon-foundation/dexon/log"
	"github.com/dexon-foundation/dexon/params"
	"github.com/dexon-foundation/dexon/rlp"
	"github.com/onrik/ethrpc"
)

const numConfirmation = 1

const recoveryABI = `
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
        "type": "address"
      }
    ],
    "name": "voted",
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
    "constant": false,
    "inputs": [],
    "name": "renounceOwnership",
    "outputs": [],
    "payable": false,
    "stateMutability": "nonpayable",
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
    "name": "isOwner",
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
    "name": "depositValue",
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
    "name": "votes",
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
    "name": "withdrawable",
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
    "anonymous": false,
    "inputs": [
      {
        "indexed": true,
        "name": "height",
        "type": "uint256"
      },
      {
        "indexed": false,
        "name": "voter",
        "type": "address"
      }
    ],
    "name": "VotedForRecovery",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": true,
        "name": "owner",
        "type": "address"
      },
      {
        "indexed": false,
        "name": "amount",
        "type": "uint256"
      }
    ],
    "name": "Withdrawn",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": true,
        "name": "previousOwner",
        "type": "address"
      },
      {
        "indexed": true,
        "name": "newOwner",
        "type": "address"
      }
    ],
    "name": "OwnershipTransferred",
    "type": "event"
  },
  {
    "constant": false,
    "inputs": [
      {
        "name": "DepositValue",
        "type": "uint256"
      }
    ],
    "name": "setDeposit",
    "outputs": [],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "constant": false,
    "inputs": [
      {
        "name": "height",
        "type": "uint256"
      },
      {
        "name": "value",
        "type": "uint256"
      }
    ],
    "name": "refund",
    "outputs": [],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "constant": false,
    "inputs": [],
    "name": "withdraw",
    "outputs": [],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "constant": false,
    "inputs": [
      {
        "name": "height",
        "type": "uint256"
      }
    ],
    "name": "voteForSkipBlock",
    "outputs": [],
    "payable": true,
    "stateMutability": "payable",
    "type": "function"
  },
  {
    "constant": false,
    "inputs": [
      {
        "name": "height",
        "type": "uint256"
      }
    ],
    "name": "numVotes",
    "outputs": [
      {
        "name": "",
        "type": "uint256"
      }
    ],
    "payable": false,
    "stateMutability": "nonpayable",
    "type": "function"
  }
]
`

var errAlreadyVoted = errors.New("already voted for recovery")

var abiObject abi.ABI

func init() {
	var err error
	abiObject, err = abi.JSON(strings.NewReader(recoveryABI))
	if err != nil {
		panic(err)
	}
}

type Recovery struct {
	gov          *DexconGovernance
	contract     common.Address
	confirmation int
	publicKey    string
	privateKey   *ecdsa.PrivateKey
	nodeAddress  common.Address
	client       *ethrpc.EthRPC
}

func NewRecovery(config *params.RecoveryConfig, networkRPC string,
	gov *DexconGovernance, privKey *ecdsa.PrivateKey) *Recovery {
	client := ethrpc.New(networkRPC)
	return &Recovery{
		gov:          gov,
		contract:     config.Contract,
		confirmation: config.Confirmation,
		publicKey:    hex.EncodeToString(crypto.FromECDSAPub(&privKey.PublicKey)),
		privateKey:   privKey,
		nodeAddress:  crypto.PubkeyToAddress(privKey.PublicKey),
		client:       client,
	}
}

func (r *Recovery) callRPC(data []byte, tag string) ([]byte, error) {
	res, err := r.client.EthCall(ethrpc.T{
		From: r.nodeAddress.String(),
		To:   r.contract.String(),
		Data: "0x" + hex.EncodeToString(data),
	}, tag)
	if err != nil {
		return nil, err
	}

	resBytes, err := hex.DecodeString(res[2:])
	if err != nil {
		return nil, err
	}
	return resBytes, nil
}

func (r *Recovery) genVoteForSkipBlockTx(height uint64) (*types.Transaction, error) {
	netVersion, err := r.client.NetVersion()
	if err != nil {
		return nil, err
	}

	networkID, err := strconv.Atoi(netVersion)
	if err != nil {
		return nil, err
	}

	data, err := abiObject.Pack("voted", big.NewInt(int64(height)), r.nodeAddress)
	if err != nil {
		return nil, err
	}

	resBytes, err := r.callRPC(data, "latest")
	if err != nil {
		return nil, err
	}

	var voted bool
	err = abiObject.Unpack(&voted, "voted", resBytes)
	if err != nil {
		return nil, err
	}

	if voted {
		log.Info("Already voted for skip block", "height", height)
		return nil, errAlreadyVoted
	}

	data, err = abiObject.Pack("depositValue")
	if err != nil {
		return nil, err
	}

	resBytes, err = r.callRPC(data, "latest")
	if err != nil {
		return nil, err
	}

	var depositValue *big.Int
	err = abiObject.Unpack(&depositValue, "depositValue", resBytes)
	if err != nil {
		return nil, err
	}

	data, err = abiObject.Pack("voteForSkipBlock", new(big.Int).SetUint64(height))
	if err != nil {
		return nil, err
	}

	gasPrice, err := r.client.EthGasPrice()
	if err != nil {
		return nil, err
	}

	nonce, err := r.client.EthGetTransactionCount(r.nodeAddress.String(), "pending")
	if err != nil {
		return nil, err
	}

	// Increase gasPrice to 3 times of suggested gas price to make sure it will
	// be included in time.
	useGasPrice := new(big.Int).Mul(&gasPrice, big.NewInt(3))

	tx := types.NewTransaction(
		uint64(nonce),
		r.contract,
		depositValue,
		uint64(100000),
		useGasPrice,
		data)

	signer := types.NewEIP155Signer(big.NewInt(int64(networkID)))
	return types.SignTx(tx, signer, r.privateKey)
}

func (r *Recovery) ProposeSkipBlock(height uint64) error {
	notarySet, err := r.gov.NotarySet(r.gov.Round())
	if err != nil {
		return err
	}
	if _, ok := notarySet[r.publicKey]; !ok {
		return errors.New("not in notary set")
	}

	tx, err := r.genVoteForSkipBlockTx(height)
	if err == errAlreadyVoted {
		return nil
	}
	if err != nil {
		return err
	}

	txData, err := rlp.EncodeToBytes(tx)
	if err != nil {
		return err
	}
	_, err = r.client.EthSendRawTransaction("0x" + hex.EncodeToString(txData))
	return err
}

func (r *Recovery) Votes(height uint64) (uint64, error) {
	data, err := abiObject.Pack("numVotes", new(big.Int).SetUint64(height))
	if err != nil {
		return 0, err
	}

	bn, err := r.client.EthBlockNumber()
	if err != nil {
		return 0, err
	}

	snapshotHeight := bn - numConfirmation

	resBytes, err := r.callRPC(data, fmt.Sprintf("0x%x", snapshotHeight))
	if err != nil {
		return 0, err
	}

	votes := new(big.Int)
	err = abiObject.Unpack(&votes, "numVotes", resBytes)
	if err != nil {
		return 0, err
	}

	notarySet, err := r.gov.DKGSetNodeKeyAddresses(r.gov.Round())
	if err != nil {
		return 0, err
	}

	count := uint64(0)

	for i := uint64(0); i < votes.Uint64(); i++ {
		data, err = abiObject.Pack(
			"votes", new(big.Int).SetUint64(height), new(big.Int).SetUint64(i))
		if err != nil {
			return 0, err
		}

		resBytes, err := r.callRPC(data, fmt.Sprintf("0x%x", snapshotHeight))
		if err != nil {
			return 0, err
		}

		var addr common.Address
		err = abiObject.Unpack(&addr, "votes", resBytes)
		if err != nil {
			return 0, err
		}

		if _, ok := notarySet[addr]; ok {
			count += 1
		}
	}
	return count, nil
}
