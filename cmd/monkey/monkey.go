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

package main

import (
	"context"
	"crypto/ecdsa"
	"flag"
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

var key = flag.String("key", "", "private key path")
var endpoint = flag.String("endpoint", "http://127.0.0.1:8545", "JSON RPC endpoint")
var n = flag.Int("n", 100, "number of random accounts")
var gambler = flag.Bool("gambler", false, "make this monkey a gambler")

type Monkey struct {
	client    *ethclient.Client
	source    *ecdsa.PrivateKey
	keys      []*ecdsa.PrivateKey
	networkID *big.Int
}

func New(ep string, source *ecdsa.PrivateKey, num int) *Monkey {
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

	return &Monkey{
		client:    client,
		source:    source,
		keys:      keys,
		networkID: networkID,
	}
}

func (m *Monkey) transfer(
	key *ecdsa.PrivateKey, toAddress common.Address, amount *big.Int, nonce uint64) {

	if nonce == math.MaxUint64 {
		var err error
		address := crypto.PubkeyToAddress(key.PublicKey)
		nonce, err = m.client.PendingNonceAt(context.Background(), address)
		if err != nil {
			panic(err)
		}
	}

	tx := types.NewTransaction(
		uint64(nonce),
		toAddress,
		amount,
		uint64(21000),
		big.NewInt(1e9),
		nil)

	signer := types.NewEIP155Signer(m.networkID)
	tx, err := types.SignTx(tx, signer, key)
	if err != nil {
		panic(err)
	}

	fmt.Println("Sending TX", "fullhash", tx.Hash().String())

	err = m.client.SendTransaction(context.Background(), tx)
	if err != nil {
		panic(err)
	}
}

func (m *Monkey) deploy(
	key *ecdsa.PrivateKey, code string, ctors []string, amount *big.Int, nonce uint64) common.Address {

	if nonce == math.MaxUint64 {
		var err error
		address := crypto.PubkeyToAddress(key.PublicKey)
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
		Data: data,
	})
	if err != nil {
		panic(err)
	}

	tx := types.NewContractCreation(
		uint64(nonce),
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

func (m *Monkey) call(
	key *ecdsa.PrivateKey, contract common.Address, input []byte, amount *big.Int, gas uint64, nonce uint64) {

	address := crypto.PubkeyToAddress(key.PublicKey)
	if nonce == math.MaxUint64 {
		var err error
		nonce, err = m.client.PendingNonceAt(context.Background(), address)
		if err != nil {
			panic(err)
		}
	}

	if gas == uint64(0) {
		var err error
		gas, err = m.client.EstimateGas(context.Background(), dexon.CallMsg{
			From:  address,
			To:    &contract,
			Value: amount,
			Data:  input,
		})
		if err != nil {
			panic(err)
		}
	}

	tx := types.NewTransaction(
		uint64(nonce),
		contract,
		amount,
		gas,
		big.NewInt(1e9),
		input)

	signer := types.NewEIP155Signer(m.networkID)
	tx, err := types.SignTx(tx, signer, key)
	if err != nil {
		panic(err)
	}

	fmt.Println("Sending TX", "fullhash", tx.Hash().String())

	err = m.client.SendTransaction(context.Background(), tx)
	if err != nil {
		panic(err)
	}
}

func (m *Monkey) Distribute() {
	fmt.Println("Distributing DEX to random accounts ...")
	address := crypto.PubkeyToAddress(m.source.PublicKey)
	nonce, err := m.client.PendingNonceAt(context.Background(), address)
	if err != nil {
		panic(err)
	}

	for _, key := range m.keys {
		address := crypto.PubkeyToAddress(key.PublicKey)
		amount := new(big.Int)
		amount.SetString("1000000000000000000", 10)
		m.transfer(m.source, address, amount, nonce)
		nonce += 1
	}
	time.Sleep(20 * time.Second)
}

func (m *Monkey) Crazy() {
	fmt.Println("Performing random transfers ...")
	nonce := uint64(0)
	for {
		fmt.Println("nonce", nonce)
		for _, key := range m.keys {
			to := crypto.PubkeyToAddress(m.keys[rand.Int()%len(m.keys)].PublicKey)
			m.transfer(key, to, big.NewInt(1), nonce)
		}
		nonce += 1
		time.Sleep(500 * time.Millisecond)
	}
}

func main() {
	flag.Parse()

	privKey, err := crypto.LoadECDSA(*key)
	if err != nil {
		panic(err)
	}

	m := New(*endpoint, privKey, *n)
	m.Distribute()
	if *gambler {
		m.Gamble()
	} else {
		m.Crazy()
	}
}
