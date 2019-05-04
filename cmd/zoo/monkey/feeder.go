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

package monkey

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"strings"
	"time"

	"github.com/dexon-foundation/dexon/accounts/abi"
	"github.com/dexon-foundation/dexon/cmd/zoo/client"
	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/crypto"
)

var bananaABI abi.ABI

func init() {
	var err error
	bananaABI, err = abi.JSON(strings.NewReader(bananaABIJSON))
	if err != nil {
		panic(err)
	}
}

func (m *Monkey) DistributeBanana(contract common.Address) {
	fmt.Println("Distributing Banana to random accounts ...")
	address := crypto.PubkeyToAddress(m.source.PublicKey)
	nonce, err := m.PendingNonceAt(context.Background(), address)
	if err != nil {
		panic(err)
	}

	ctxs := make([]*client.TransferContext, len(m.keys))
	amount := new(big.Int)
	amount.SetString("10000000000000000", 10)
	for i, key := range m.keys {
		address := crypto.PubkeyToAddress(key.PublicKey)
		input, err := bananaABI.Pack("transfer", address, amount)
		if err != nil {
			panic(err)
		}
		ctxs[i] = &client.TransferContext{
			Key:       m.source,
			ToAddress: contract,
			Data:      input,
			Nonce:     nonce,
			Gas:       100000,
		}
		nonce++
	}
	m.BatchTransfer(ctxs)
	time.Sleep(20 * time.Second)
}

func (m *Monkey) Feed() uint64 {
	fmt.Println("Deploying contract ...")
	contract := m.Deploy(m.source, bananaContract, nil, new(big.Int), math.MaxUint64)
	fmt.Println("  Contract deployed: ", contract.String())
	m.DistributeBanana(contract)

	time.Sleep(5 * time.Second)

	nonce := uint64(0)
loop:
	for {
		fmt.Println("nonce", nonce)
		ctxs := make([]*client.TransferContext, len(m.keys))
		for i, key := range m.keys {
			to := crypto.PubkeyToAddress(m.keys[rand.Int()%len(m.keys)].PublicKey)
			input, err := bananaABI.Pack("transfer", to, big.NewInt(rand.Int63n(100)+1))
			if err != nil {
				panic(err)
			}

			ctx := &client.TransferContext{
				Key:       key,
				ToAddress: contract,
				Data:      input,
				Nonce:     nonce,
				Gas:       42000,
			}
			if config.Batch {
				ctxs[i] = ctx
			} else {
				m.Transfer(ctx)
			}
		}
		if config.Batch {
			m.BatchTransfer(ctxs)
		}

		if m.timer != nil {
			select {
			case <-m.timer:
				break loop
			default:
			}
		}

		nonce++
		time.Sleep(time.Duration(config.Sleep) * time.Millisecond)
	}

	return nonce
}
