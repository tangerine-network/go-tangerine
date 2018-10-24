// Copyright 2018 The dexon-consensus-core Authors
// This file is part of the dexon-consensus-core library.
//
// The dexon-consensus-core library is free software: you can redistribute it
// and/or modify it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The dexon-consensus-core library is distributed in the hope that it will be
// useful, but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the dexon-consensus-core library. If not, see
// <http://www.gnu.org/licenses/>.

// A simple monkey that sends random transactions into the network.

package main

import (
	"context"
	"flag"
	"fmt"
	"math/big"
	"math/rand"
	"time"

	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/core/types"
	"github.com/dexon-foundation/dexon/crypto"
	"github.com/dexon-foundation/dexon/ethclient"
)

var key = flag.String("key", "", "private key path")
var endpoint = flag.String("endpoint", "http://localhost:8545", "JSON RPC endpoint")
var tps = flag.Int("tps", 10, "transactions per second")

func randomAddress() common.Address {
	x := common.Address{}
	for i := 0; i < 20; i++ {
		x[i] = byte(rand.Int31() % 16)
	}
	return x
}

func main() {
	flag.Parse()

	privKey, err := crypto.LoadECDSA(*key)
	if err != nil {
		panic(err)
	}

	client, err := ethclient.Dial(*endpoint)
	if err != nil {
		panic(err)
	}

	address := crypto.PubkeyToAddress(privKey.PublicKey)

	nonce, err := client.PendingNonceAt(context.Background(), address)
	if err != nil {
		panic(err)
	}

	for {
		tx := types.NewTransaction(
			nonce,
			randomAddress(),
			big.NewInt(10),
			uint64(21000),
			big.NewInt(1e9),
			nil)

		signer := types.NewEIP155Signer(big.NewInt(237))
		tx, err = types.SignTx(tx, signer, privKey)
		if err != nil {
			panic(err)
		}

		err := client.SendTransaction(context.Background(), tx)
		if err != nil {
			panic(err)
		}

		fmt.Println("Sending TX", "fullhash", tx.Hash().String())

		nonce++
		time.Sleep(time.Duration(float32(time.Second) / float32(*tps)))
	}
}
