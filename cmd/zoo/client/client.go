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

// A simple monkey that sends random transactions into the network.

package client

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math"
	"math/big"
	"time"

	dexon "github.com/dexon-foundation/dexon"
	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/core/types"
	"github.com/dexon-foundation/dexon/crypto"
	"github.com/dexon-foundation/dexon/ethclient"
)

type Client struct {
	ethclient.Client

	source    *ecdsa.PrivateKey
	networkID *big.Int
}

func New(ep string) (*Client, error) {
	client, err := ethclient.Dial(ep)
	if err != nil {
		return nil, err
	}

	networkID, err := client.NetworkID(context.Background())
	if err != nil {
		return nil, err
	}

	return &Client{
		Client:    *client,
		networkID: networkID,
	}, nil
}

type TransferContext struct {
	Key       *ecdsa.PrivateKey
	ToAddress common.Address
	Amount    *big.Int
	Data      []byte
	Nonce     uint64
	Gas       uint64
}

func (c *Client) PrepareTx(ctx *TransferContext) *types.Transaction {
	if ctx.Nonce == math.MaxUint64 {
		var err error
		address := crypto.PubkeyToAddress(ctx.Key.PublicKey)
		ctx.Nonce, err = c.PendingNonceAt(context.Background(), address)
		if err != nil {
			panic(err)
		}
	}

	if ctx.Gas == uint64(0) {
		var err error
		ctx.Gas, err = c.EstimateGas(context.Background(), dexon.CallMsg{
			Data: ctx.Data,
		})
		if err != nil {
			panic(err)
		}
	}

	gasPrice, err := c.SuggestGasPrice(context.Background())
	if err != nil {
		panic(err)
	}

	tx := types.NewTransaction(
		ctx.Nonce,
		ctx.ToAddress,
		ctx.Amount,
		ctx.Gas,
		gasPrice,
		ctx.Data)

	signer := types.NewEIP155Signer(c.networkID)
	tx, err = types.SignTx(tx, signer, ctx.Key)
	if err != nil {
		panic(err)
	}

	return tx
}

func (c *Client) Transfer(ctx *TransferContext) {
	tx := c.PrepareTx(ctx)

	err := c.SendTransaction(context.Background(), tx)
	if err != nil {
		panic(err)
	}
}

func (c *Client) BatchTransfer(ctxs []*TransferContext) {
	txs := make([]*types.Transaction, len(ctxs))
	for i, ctx := range ctxs {
		txs[i] = c.PrepareTx(ctx)
	}

	err := c.SendTransactions(context.Background(), txs)
	if err != nil {
		panic(err)
	}
}

func (c *Client) Deploy(
	key *ecdsa.PrivateKey, code string, ctors []string, amount *big.Int, nonce uint64) common.Address {

	address := crypto.PubkeyToAddress(key.PublicKey)
	if nonce == math.MaxUint64 {
		var err error
		nonce, err = c.PendingNonceAt(context.Background(), address)
		if err != nil {
			panic(err)
		}
	}

	var input string
	for _, ctor := range ctors {
		input += fmt.Sprintf("%064s", ctor)
	}
	data := common.Hex2Bytes(code + input)

	gas, err := c.EstimateGas(context.Background(), dexon.CallMsg{
		From: address,
		Data: data,
	})
	if err != nil {
		panic(err)
	}

	tx := types.NewContractCreation(
		nonce,
		amount,
		gas,
		big.NewInt(1e9),
		data)

	signer := types.NewEIP155Signer(c.networkID)
	tx, err = types.SignTx(tx, signer, key)
	if err != nil {
		panic(err)
	}

	fmt.Println("Sending TX", "fullhash", tx.Hash().String())

	err = c.SendTransaction(context.Background(), tx)
	if err != nil {
		panic(err)
	}

	for {
		time.Sleep(500 * time.Millisecond)
		recp, err := c.TransactionReceipt(context.Background(), tx.Hash())
		if err != nil {
			if err == dexon.NotFound {
				continue
			}
			panic(err)
		}
		return recp.ContractAddress
	}
}
