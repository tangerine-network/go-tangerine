package main

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"os"

	coreTypes "github.com/dexon-foundation/dexon-consensus/core/types"
	dkgTypes "github.com/dexon-foundation/dexon-consensus/core/types/dkg"
	"github.com/dexon-foundation/dexon/cmd/utils"
	"github.com/dexon-foundation/dexon/core/vm"
	"github.com/dexon-foundation/dexon/rlp"
	"gopkg.in/urfave/cli.v1"
)

// Git SHA1 commit hash of the release (set via linker flags)
var gitCommit = ""

var app *cli.App

func init() {
	app = utils.NewApp(gitCommit, "DEXON governance tool")
	app.Commands = []cli.Command{
		commandDecodeInput,
	}
}

func decodeInput(ctx *cli.Context) error {
	inputHex := ctx.Args().First()
	if inputHex == "" {
		utils.Fatalf("no input specified")
	}

	if inputHex[:2] == "0x" {
		inputHex = inputHex[2:]
	}

	input, err := hex.DecodeString(inputHex)
	if err != nil {
		utils.Fatalf("failed to decode input")
	}

	// Parse input.
	method, exists := vm.GovernanceABI.Sig2Method[string(input[:4])]
	if !exists {
		utils.Fatalf("invalid method")
	}

	arguments := input[4:]

	switch method.Name {
	case "addDKGComplaint":
		var Complaint []byte
		if err := method.Inputs.Unpack(&Complaint, arguments); err != nil {
			utils.Fatalf("%s", err)
		}
		var comp dkgTypes.Complaint
		if err := rlp.DecodeBytes(Complaint, &comp); err != nil {
			utils.Fatalf("%s", err)
		}
		fmt.Printf("Complaint: %+v\n", comp)
	case "addDKGMasterPublicKey":
		var PublicKey []byte
		if err := method.Inputs.Unpack(&PublicKey, arguments); err != nil {
			utils.Fatalf("%s", err)
		}
		var mpk dkgTypes.MasterPublicKey
		if err := rlp.DecodeBytes(PublicKey, &mpk); err != nil {
			utils.Fatalf("%s", err)
		}
		fmt.Printf("MasterPublicKey: %+v\n", mpk)
	case "addDKGMPKReady":
		var MPKReady []byte
		if err := method.Inputs.Unpack(&MPKReady, arguments); err != nil {
			utils.Fatalf("%s", err)
		}
		var ready dkgTypes.MPKReady
		if err := rlp.DecodeBytes(MPKReady, &ready); err != nil {
			utils.Fatalf("%s", err)
		}
		fmt.Printf("MPKReady: %+v\n", ready)
	case "addDKGFinalize":
		var Finalize []byte
		if err := method.Inputs.Unpack(&Finalize, arguments); err != nil {
			utils.Fatalf("%s", err)
		}
		var finalize dkgTypes.Finalize
		if err := rlp.DecodeBytes(Finalize, &finalize); err != nil {
			utils.Fatalf("%s", err)
		}
		fmt.Println(finalize)
	case "report":
		args := struct {
			Type *big.Int
			Arg1 []byte
			Arg2 []byte
		}{}
		if err := method.Inputs.Unpack(&args, arguments); err != nil {
			utils.Fatalf("%s", err)
		}
		switch args.Type.Uint64() {
		case vm.FineTypeForkVote:
			vote1 := new(coreTypes.Vote)
			if err := rlp.DecodeBytes(args.Arg1, vote1); err != nil {
				utils.Fatalf("%s", err)
			}
			vote2 := new(coreTypes.Vote)
			if err := rlp.DecodeBytes(args.Arg2, vote2); err != nil {
				utils.Fatalf("%s", err)
			}
			fmt.Printf("Vote1: %+v\n", vote1)
			fmt.Printf("Vote2: %+v\n", vote2)
		case vm.FineTypeForkBlock:
			block1 := new(coreTypes.Block)
			if err := rlp.DecodeBytes(args.Arg1, block1); err != nil {
				utils.Fatalf("%s", err)
			}
			block2 := new(coreTypes.Block)
			if err := rlp.DecodeBytes(args.Arg2, block2); err != nil {
				utils.Fatalf("%s", err)
			}
			fmt.Printf("Block1: %+v\n", block1)
			fmt.Printf("Block2: %+v\n", block2)
		}
	default:
		fmt.Printf("Unsupported method: %s\n", method.Name)
	}
	return nil
}

var commandDecodeInput = cli.Command{
	Name:        "decode-input",
	Usage:       "decode governance input",
	ArgsUsage:   "[ <hex-data> ]",
	Description: `decode governance tx input`,
	Action:      decodeInput,
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
