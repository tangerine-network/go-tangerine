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

package utils

import (
	"bufio"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"math/big"
	"os"

	"github.com/dexon-foundation/dexon"
	"github.com/dexon-foundation/dexon/cmd/zoo/client"
	"github.com/dexon-foundation/dexon/crypto"
)

type ShutdownConfig struct {
	Key      string
	Endpoint string
	File     string
	Batch    bool
}

func Shutdown(config *ShutdownConfig) {
	privKey, err := crypto.LoadECDSA(config.Key)
	if err != nil {
		panic(err)
	}
	addr := crypto.PubkeyToAddress(privKey.PublicKey)

	cl, err := client.New(config.Endpoint)
	if err != nil {
		panic(err)
	}

	file, err := os.Open(config.File)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	ctxs := make([]*client.TransferContext, 0)
	reader := bufio.NewReader(file)
	for {
		buf, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		buf = buf[:len(buf)-1]
		key, err := hex.DecodeString(buf)
		if err != nil {
			panic(err)
		}
		prvKey, err := crypto.ToECDSA(key)
		if err != nil {
			panic(err)
		}
		balance, err := cl.BalanceAt(context.Background(),
			crypto.PubkeyToAddress(prvKey.PublicKey), nil)
		if err != nil {
			panic(err)
		}

		gasLimit, err := cl.EstimateGas(context.Background(), dexon.CallMsg{})
		if err != nil {
			panic(err)
		}
		gasPrice, err := cl.SuggestGasPrice(context.Background())
		if err != nil {
			panic(err)
		}
		gas := new(big.Int).Mul(gasPrice, big.NewInt(int64(gasLimit)))
		if gas.Cmp(balance) >= 0 {
			fmt.Println("Skipping account: not enough of DXN", crypto.PubkeyToAddress(prvKey.PublicKey))
			continue
		}

		ctx := &client.TransferContext{
			Key:       prvKey,
			ToAddress: addr,
			Amount:    new(big.Int).Sub(balance, gas),
			Nonce:     math.MaxUint64,
		}
		if !config.Batch {
			cl.Transfer(ctx)
		} else {
			ctxs = append(ctxs, ctx)
		}
	}
	if config.Batch {
		cl.BatchTransfer(ctxs)
	}
}
