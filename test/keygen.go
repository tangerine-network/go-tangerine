package main

import (
	"encoding/hex"
	"fmt"

	"github.com/dexon-foundation/dexon/crypto"
)

func main() {
	for i := 0; i < 4; i++ {
		privKey, err := crypto.GenerateKey()
		if err != nil {
			panic(err)
		}
		address := crypto.PubkeyToAddress(privKey.PublicKey).String()
		pk := hex.EncodeToString(crypto.CompressPubkey(&privKey.PublicKey))

		fmt.Printf(`
    "%s": {
      "balance": "1000000000000000000000",
      "staked": "500000000000000000000",
      "publicKey": "0x%s"
    },`, address, pk)

		crypto.SaveECDSA(fmt.Sprintf("test%d.nodekey", i), privKey)
	}
}
