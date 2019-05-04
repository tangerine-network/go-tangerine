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
	"encoding/hex"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"time"

	"github.com/dexon-foundation/dexon/cmd/zoo/client"
	"github.com/dexon-foundation/dexon/crypto"
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
	client.Client

	source *ecdsa.PrivateKey
	keys   []*ecdsa.PrivateKey
	timer  <-chan time.Time
}

func New(ep string, source *ecdsa.PrivateKey, num int, timeout time.Duration) *Monkey {
	client, err := client.New(ep)
	if err != nil {
		panic(err)
	}

	file := func() *os.File {
		for i := 0; i < 100; i++ {
			name := fmt.Sprintf("zoo-%d.keys", i)
			file, err := os.OpenFile(name,
				os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
			if err != nil {
				continue
			}
			fmt.Printf("Save keys to file %s\n", name)
			return file
		}
		return nil
	}()
	if file == nil {
		panic(fmt.Errorf("Failed to create file for zoo keys"))
	}
	defer file.Close()

	var keys []*ecdsa.PrivateKey

	for i := 0; i < num; i++ {
		key, err := crypto.GenerateKey()
		if err != nil {
			panic(err)
		}
		_, err = file.Write([]byte(hex.EncodeToString(crypto.FromECDSA(key)) + "\n"))
		if err != nil {
			panic(err)
		}
		keys = append(keys, key)
	}

	monkey := &Monkey{
		Client: *client,
		source: source,
		keys:   keys,
	}

	if timeout > 0 {
		monkey.timer = time.After(timeout * time.Second)
	}

	return monkey
}

func (m *Monkey) Distribute() {
	fmt.Println("Distributing DEX to random accounts ...")
	address := crypto.PubkeyToAddress(m.source.PublicKey)
	nonce, err := m.PendingNonceAt(context.Background(), address)
	if err != nil {
		panic(err)
	}

	ctxs := make([]*client.TransferContext, len(m.keys))
	for i, key := range m.keys {
		address := crypto.PubkeyToAddress(key.PublicKey)
		amount := new(big.Int)
		amount.SetString("1000000000000000000", 10)
		ctxs[i] = &client.TransferContext{
			Key:       m.source,
			ToAddress: address,
			Amount:    amount,
			Nonce:     nonce,
			Gas:       21000,
		}
		nonce++
	}
	m.BatchTransfer(ctxs)
	time.Sleep(20 * time.Second)
}

func (m *Monkey) Crazy() uint64 {
	fmt.Println("Performing random transfers ...")
	nonce := uint64(0)
loop:
	for {
		ctxs := make([]*client.TransferContext, len(m.keys))
		for i, key := range m.keys {
			to := crypto.PubkeyToAddress(m.keys[rand.Int()%len(m.keys)].PublicKey)
			amount := new(big.Int)
			amount.SetString(fmt.Sprintf("%d0000000000000", rand.Intn(10)+1), 10)
			ctx := &client.TransferContext{
				Key:       key,
				ToAddress: to,
				Amount:    amount,
				Nonce:     nonce,
				Gas:       21000,
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
		fmt.Printf("Sent %d transactions, nonce = %d\n", len(m.keys), nonce)

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
