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

// A simple monkey that sends random transactions into the network.

package monkey

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"time"

	dexon "github.com/dexon-foundation/dexon"
	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/core/types"
	"github.com/dexon-foundation/dexon/crypto"
	"github.com/dexon-foundation/dexon/ethclient"
)

var config *MonkeyConfig

type MonkeyConfig struct {
	Key      string
	Endpoint string
	N        int
	Gambler  bool
	Feeder   bool
	Batch    bool
	Sleep    int
	Timeout  int
}

func Init(cfg *MonkeyConfig) {
	config = cfg
}

type Monkey struct {
	client    *ethclient.Client
	source    *ecdsa.PrivateKey
	keys      []*ecdsa.PrivateKey
	networkID *big.Int
	timer     <-chan time.Time
}

func New(ep string, source *ecdsa.PrivateKey, num int, timeout time.Duration) *Monkey {
	client, err := ethclient.Dial(ep)
	if err != nil {
		panic(err)
	}

	var keys []*ecdsa.PrivateKey

	for i := 0; i < num; i++ {
		key, err := crypto.GenerateKey()
		if err != nil {
			panic(err)
		}
		keys = append(keys, key)
	}

	networkID, err := client.NetworkID(context.Background())
	if err != nil {
		panic(err)
	}

	monkey := &Monkey{
		client:    client,
		source:    source,
		keys:      keys,
		networkID: networkID,
	}

	if timeout > 0 {
		monkey.timer = time.After(timeout * time.Second)
	}

	return monkey
}

type transferContext struct {
	Key       *ecdsa.PrivateKey
	ToAddress common.Address
	Amount    *big.Int
	Data      []byte
	Nonce     uint64
	Gas       uint64
}

func (m *Monkey) prepareTx(ctx *transferContext) *types.Transaction {
	if ctx.Nonce == math.MaxUint64 {
		var err error
		address := crypto.PubkeyToAddress(ctx.Key.PublicKey)
		ctx.Nonce, err = m.client.PendingNonceAt(context.Background(), address)
		if err != nil {
			panic(err)
		}
	}

	if ctx.Gas == uint64(0) {
		var err error
		ctx.Gas, err = m.client.EstimateGas(context.Background(), dexon.CallMsg{
			Data: ctx.Data,
		})
		if err != nil {
			panic(err)
		}
	}

	tx := types.NewTransaction(
		ctx.Nonce,
		ctx.ToAddress,
		ctx.Amount,
		ctx.Gas,
		big.NewInt(1e9),
		ctx.Data)

	signer := types.NewEIP155Signer(m.networkID)
	tx, err := types.SignTx(tx, signer, ctx.Key)
	if err != nil {
		panic(err)
	}

	return tx
}

func (m *Monkey) transfer(ctx *transferContext) {
	tx := m.prepareTx(ctx)

	err := m.client.SendTransaction(context.Background(), tx)
	if err != nil {
		panic(err)
	}
}

func (m *Monkey) batchTransfer(ctxs []*transferContext) {
	txs := make([]*types.Transaction, len(ctxs))
	for i, ctx := range ctxs {
		txs[i] = m.prepareTx(ctx)
	}

	err := m.client.SendTransactions(context.Background(), txs)
	if err != nil {
		panic(err)
	}
}

func (m *Monkey) deploy(
	key *ecdsa.PrivateKey, code string, ctors []string, amount *big.Int, nonce uint64) common.Address {

	address := crypto.PubkeyToAddress(key.PublicKey)
	if nonce == math.MaxUint64 {
		var err error
		nonce, err = m.client.PendingNonceAt(context.Background(), address)
		if err != nil {
			panic(err)
		}
	}

	var input string
	for _, ctor := range ctors {
		input += fmt.Sprintf("%064s", ctor)
	}
	data := common.Hex2Bytes(code + input)

	gas, err := m.client.EstimateGas(context.Background(), dexon.CallMsg{
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

	signer := types.NewEIP155Signer(m.networkID)
	tx, err = types.SignTx(tx, signer, key)
	if err != nil {
		panic(err)
	}

	fmt.Println("Sending TX", "fullhash", tx.Hash().String())

	err = m.client.SendTransaction(context.Background(), tx)
	if err != nil {
		panic(err)
	}

	for {
		time.Sleep(500 * time.Millisecond)
		recp, err := m.client.TransactionReceipt(context.Background(), tx.Hash())
		if err != nil {
			if err == dexon.NotFound {
				continue
			}
			panic(err)
		}
		return recp.ContractAddress
	}
}

func (m *Monkey) Distribute() {
	fmt.Println("Distributing DEX to random accounts ...")
	address := crypto.PubkeyToAddress(m.source.PublicKey)
	nonce, err := m.client.PendingNonceAt(context.Background(), address)
	if err != nil {
		panic(err)
	}

	ctxs := make([]*transferContext, len(m.keys))
	for i, key := range m.keys {
		address := crypto.PubkeyToAddress(key.PublicKey)
		amount := new(big.Int)
		amount.SetString("1000000000000000000", 10)
		ctxs[i] = &transferContext{
			Key:       m.source,
			ToAddress: address,
			Amount:    amount,
			Nonce:     nonce,
			Gas:       21000,
		}
		nonce += 1
	}
	m.batchTransfer(ctxs)
	time.Sleep(20 * time.Second)
}

func (m *Monkey) Crazy() uint64 {
	fmt.Println("Performing random transfers ...")
	nonce := uint64(0)
loop:
	for {
		ctxs := make([]*transferContext, len(m.keys))
		for i, key := range m.keys {
			to := crypto.PubkeyToAddress(m.keys[rand.Int()%len(m.keys)].PublicKey)
			amount := new(big.Int)
			amount.SetString(fmt.Sprintf("%d0000000000000", rand.Intn(10)+1), 10)
			ctx := &transferContext{
				Key:       key,
				ToAddress: to,
				Amount:    amount,
				Nonce:     nonce,
				Gas:       21000,
			}
			if config.Batch {
				ctxs[i] = ctx
			} else {
				m.transfer(ctx)
			}
		}
		if config.Batch {
			m.batchTransfer(ctxs)
		}
		fmt.Printf("Sent %d transactions, nonce = %d\n", len(m.keys), nonce)

		if m.timer != nil {
			select {
			case <-m.timer:
				break loop
			default:
			}
		}

		nonce += 1
		time.Sleep(time.Duration(config.Sleep) * time.Millisecond)
	}

	return nonce
}

func (m *Monkey) Keys() []*ecdsa.PrivateKey {
	return m.keys
}

func Exec() (*Monkey, uint64) {
	privKey, err := crypto.LoadECDSA(config.Key)
	if err != nil {
		panic(err)
	}

	m := New(config.Endpoint, privKey, config.N, time.Duration(config.Timeout))
	m.Distribute()
	var finalNonce uint64
	if config.Gambler {
		finalNonce = m.Gamble()
	} else if config.Feeder {
		finalNonce = m.Feed()
	} else {
		finalNonce = m.Crazy()
	}

	return m, finalNonce
}
