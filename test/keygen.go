package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"strconv"

	"github.com/tangerine-network/go-tangerine/common"
	"github.com/tangerine-network/go-tangerine/core"
	"github.com/tangerine-network/go-tangerine/crypto"
)

const genesisFile = "genesis.json"

var preFundAmount *big.Int = new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e8))

var preFundAddresss = []string{
	"0x0D54AF942d6bF13870F5CA65D470954f21D3cBE5", // Owner
	"0xAA2fe8D6024682aF0540F8BA776b549bB50251ab", // Monkey
}

func main() {
	genesis := core.DefaultGenesisBlock()

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
			Balance:   new(big.Int).Mul(big.NewInt(1e18), big.NewInt(2e6)),
			Staked:    new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e6)),
			PublicKey: crypto.FromECDSAPub(&privKey.PublicKey),
			NodeInfo: core.NodeInfo{
				Name:     fmt.Sprintf("Tangerine Test Node %d", i),
				Email:    fmt.Sprintf("tangerine%dtangerine-network.io", i),
				Location: "Taipei, Taiwan",
				Url:      "https://tangerine-network.io",
			},
		}
		fmt.Printf("Created account %s\n", address.String())
	}

	data, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		panic(err)
	}

	if err := ioutil.WriteFile(genesisFile, data, 0644); err != nil {
		panic(err)
	}
	fmt.Println("Done.")
}
