package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"strconv"

	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/core"
	"github.com/dexon-foundation/dexon/crypto"
)

const genesisFile = "genesis.json"

var preFundAmount *big.Int = new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e8))

var preFundAddresss = []string{
	"0x2a9D669e4791845EED01D4c0ffF3B927cC94A884",
	"0x1245A8672FA881Cf858eF01D34c42B55D9b263fF",
	"0xe0F859340353854693F537ea337106a33E9FeAB0",
}

func main() {
	data, err := ioutil.ReadFile(genesisFile)
	if err != nil {
		panic(err)
	}

	genesis := new(core.Genesis)
	if err := json.Unmarshal(data, &genesis); err != nil {
		panic(err)
	}

	// Clear previous allocation.
	genesis.Alloc = make(map[common.Address]core.GenesisAccount)

	count, err := strconv.Atoi(os.Args[1])
	if err != nil {
		panic(err)
	}
	for _, addr := range preFundAddresss {
		address := common.HexToAddress(addr)
		genesis.Alloc[address] = core.GenesisAccount{
			Balance: preFundAmount,
			Staked:  big.NewInt(0),
		}
		fmt.Printf("Created account %s\n", address.String())
	}
	for i := 0; i < count; i++ {
		privKey, err := crypto.GenerateKey()
		if err != nil {
			panic(err)
		}
		crypto.SaveECDSA(fmt.Sprintf("keystore/test%d.key", i), privKey)

		address := crypto.PubkeyToAddress(privKey.PublicKey)
		genesis.Alloc[address] = core.GenesisAccount{
			Balance:   new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e6)),
			Staked:    new(big.Int).Mul(big.NewInt(1e18), big.NewInt(5e5)),
			PublicKey: crypto.FromECDSAPub(&privKey.PublicKey),
			NodeInfo: core.NodeInfo{
				Name:     fmt.Sprintf("DEXON Test Node %d", i),
				Email:    fmt.Sprintf("dexon%d@dexon.org", i),
				Location: "Taipei, Taiwan",
				Url:      "https://dexon.org",
			},
		}
		fmt.Printf("Created account %s\n", address.String())
	}

	data, err = json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		panic(err)
	}

	if err := ioutil.WriteFile(genesisFile, data, 0644); err != nil {
		panic(err)
	}
	fmt.Println("Done.")
}
