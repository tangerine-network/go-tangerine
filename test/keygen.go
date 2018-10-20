package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"strconv"

	"github.com/dexon-foundation/dexon/crypto"
)

func main() {
	count, err := strconv.Atoi(os.Args[1])
	if err != nil {
		panic(err)
	}
	for i := 0; i < count; i++ {
		privKey, err := crypto.GenerateKey()
		if err != nil {
			panic(err)
		}
		address := crypto.PubkeyToAddress(privKey.PublicKey).String()
		pk := hex.EncodeToString(crypto.FromECDSAPub(&privKey.PublicKey))

		fmt.Printf(`
    "%s": {
      "balance": "1000000000000000000000",
      "staked": "500000000000000000000",
      "publicKey": "0x%s"
    },`, address, pk)

		crypto.SaveECDSA(fmt.Sprintf("test%d.nodekey", i), privKey)
	}
}
