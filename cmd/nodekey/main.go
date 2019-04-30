package main

import (
	"encoding/hex"
	"fmt"
	"os"

	"github.com/dexon-foundation/dexon/cmd/utils"
	"github.com/dexon-foundation/dexon/crypto"

	"gopkg.in/urfave/cli.v1"
)

const (
	defaultKeyfileName = "node.key"
)

// Git SHA1 commit hash of the release (set via linker flags)
var gitCommit = ""

var app *cli.App

func init() {
	app = utils.NewApp(gitCommit, "DEXON node key manager")
	app.Commands = []cli.Command{
		commandGenerate,
		commandInspect,
		commandPK2Addr,
	}
}

var commandGenerate = cli.Command{
	Name:        "generate",
	Usage:       "generate new keyfile",
	ArgsUsage:   "[ <keyfile> ]",
	Description: `Generate a new node key.`,
	Action: func(ctx *cli.Context) error {
		// Check if keyfile path given and make sure it doesn't already exist.
		keyfilepath := ctx.Args().First()
		if keyfilepath == "" {
			keyfilepath = defaultKeyfileName
		}
		if _, err := os.Stat(keyfilepath); err == nil {
			utils.Fatalf("Keyfile already exists at %s.", keyfilepath)
		} else if !os.IsNotExist(err) {
			utils.Fatalf("Error checking if keyfile exists: %v", err)
		}

		privKey, err := crypto.GenerateKey()
		if err != nil {
			utils.Fatalf("Failed to generate random private key: %v", err)
		}

		address := crypto.PubkeyToAddress(privKey.PublicKey)
		err = crypto.SaveECDSA(keyfilepath, privKey)
		if err != nil {
			utils.Fatalf("Failed to save keyfile: %v", err)
		}

		fmt.Printf("Node Address: %s\n", address.String())
		fmt.Printf("Public Key: 0x%s\n",
			hex.EncodeToString(crypto.FromECDSAPub(&privKey.PublicKey)))
		return nil
	},
}

var commandInspect = cli.Command{
	Name:        "inspect",
	Usage:       "inspect a keyfile",
	ArgsUsage:   "[ <keyfile> ]",
	Description: `Generate a new node key.`,
	Action: func(ctx *cli.Context) error {
		keyfilepath := ctx.Args().First()

		privKey, err := crypto.LoadECDSA(keyfilepath)
		if err != nil {
			utils.Fatalf("Failed to read key file: %v", err)
		}

		address := crypto.PubkeyToAddress(privKey.PublicKey)

		fmt.Printf("Node Address: %s\n", address.String())
		fmt.Printf("Public Key: 0x%s\n",
			hex.EncodeToString(crypto.FromECDSAPub(&privKey.PublicKey)))
		return nil
	},
}

var commandPK2Addr = cli.Command{
	Name:        "pk2addr",
	Usage:       "public key to address",
	ArgsUsage:   "publickey",
	Description: `Convert public key to address`,
	Action: func(ctx *cli.Context) error {
		pk := ctx.Args().First()
		if pk[1] == 'x' {
			pk = pk[2:]
		}
		pkhex, err := hex.DecodeString(pk)
		if err != nil {
			panic(err)
		}
		key, err := crypto.UnmarshalPubkey(pkhex)
		if err != nil {
			panic(err)
		}

		address := crypto.PubkeyToAddress(*key)

		fmt.Printf("Node Address: %s\n", address.String())
		return nil
	},
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
