package main

import (
	"bytes"
	"context"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	dexCore "github.com/tangerine-network/tangerine-consensus/core"

	ethereum "github.com/tangerine-network/go-tangerine"
	"github.com/tangerine-network/go-tangerine/accounts/abi"
	"github.com/tangerine-network/go-tangerine/cmd/zoo/monkey"
	"github.com/tangerine-network/go-tangerine/core/types"
	"github.com/tangerine-network/go-tangerine/core/vm"
	"github.com/tangerine-network/go-tangerine/crypto"
	"github.com/tangerine-network/go-tangerine/ethclient"
	"github.com/tangerine-network/go-tangerine/internal/build"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("need subcommand as first argument")
	}

	switch os.Args[1] {
	case "verifyGovCRS":
		doVerifyGovCRS(os.Args[2:])
	case "verifyGovMPK":
		doVerifyGovMPK(os.Args[2:])
	case "monkeyTest":
		doMonkeyTest(os.Args[2:])
	case "waitForRecovery":
		doWaitForRecovery(os.Args[2:])
	case "upload":
		doUpload(os.Args[2:])
	}
}

func getBlockNumber(client *ethclient.Client, round int) *big.Int {
	if round == 0 {
		return big.NewInt(0)
	}
	abiObject, err := abi.JSON(strings.NewReader(vm.GovernanceABIJSON))
	if err != nil {
		log.Fatalf("read abi fail: %v", err)
	}

	input, err := abiObject.Pack("roundHeight", big.NewInt(int64(round)))
	if err != nil {
		log.Fatalf("pack input fail: %v", err)
	}

	result, err := client.CallContract(context.Background(), ethereum.CallMsg{
		To:   &vm.GovernanceContractAddress,
		Data: input,
	}, nil)
	if err != nil {
		log.Fatalf("call contract fail: %v", err)
	}

	if bytes.Equal(make([]byte, 32), result) {
		log.Fatalf("round %d height not found", round)
	}

	roundHeight := new(big.Int)
	if err := abiObject.Unpack(&roundHeight, "roundHeight", result); err != nil {
		log.Fatalf("unpack output fail: %v", err)
	}
	return roundHeight
}

func doVerifyGovCRS(args []string) {
	if len(args) < 2 {
		log.Fatal("arg length is not enough")
	}

	abiObject, err := abi.JSON(strings.NewReader(vm.GovernanceABIJSON))
	if err != nil {
		log.Fatalf("read abi fail: %v", err)
	}

	round, err := strconv.Atoi(args[1])
	if err != nil {
		log.Fatalf("pasre round from arg 2 fail: %v", err)
	}

	client, err := ethclient.Dial(args[0])
	if err != nil {
		log.Fatalf("new ethclient fail: %v", err)
	}

	blockNumber := getBlockNumber(client, round)

	input, err := abiObject.Pack("crs")
	if err != nil {
		log.Fatalf("pack input fail: %v", err)
	}

	result, err := client.CallContract(context.Background(), ethereum.CallMsg{
		To:   &vm.GovernanceContractAddress,
		Data: input,
	}, blockNumber)
	if err != nil {
		log.Fatalf("call contract fail: %v", err)
	}

	if bytes.Equal(make([]byte, 32), result) {
		log.Fatalf("round %s crs not found", args[1])
	}

	log.Printf("get round %s crs %x", args[1], result)
}

func doVerifyGovMPK(args []string) {
	if len(args) < 3 {
		log.Fatal("arg length is not enough")
	}
	abiObject, err := abi.JSON(strings.NewReader(vm.GovernanceABIJSON))
	if err != nil {
		log.Fatalf("read abi fail: %v", err)
	}

	round, err := strconv.Atoi(args[1])
	if err != nil {
		log.Fatalf("pasre round from arg 2 fail: %v", err)
	}

	if uint64(round) < dexCore.DKGDelayRound {
		return
	}

	client, err := ethclient.Dial(args[0])
	if err != nil {
		log.Fatalf("new ethclient fail: %v", err)
	}

	blockNumber := getBlockNumber(client, round)

	index, err := strconv.Atoi(args[2])
	if err != nil {
		log.Fatalf("pasre index from arg 2 fail: %v", err)
	}

	input, err := abiObject.Pack("dkgMasterPublicKeys", big.NewInt(int64(index)))
	if err != nil {
		log.Fatalf("pack input fail: %v", err)
	}

	result, err := client.CallContract(context.Background(), ethereum.CallMsg{
		To:   &vm.GovernanceContractAddress,
		Data: input,
	}, blockNumber)
	if err != nil {
		log.Fatalf("call contract fail: %v", err)
	}

	if bytes.Equal(make([]byte, 0), result) {
		log.Fatalf("round %s index %s mpk not found", args[1], args[2])
	}

	log.Printf("get round %s index %s master public key %x", args[1], args[2], result)
}

func doUpload(args []string) {
	auth := build.GCPOption{
		CredentialPath: os.Getenv("GCP_CREDENTIAL_PATH"),
	}

	if err := build.GCPFileUpload(args[0], args[1], filepath.Base(args[0]), auth); err != nil {
		log.Fatalf("upload fail: %v", err)
	}
}

func doMonkeyTest(args []string) {
	if len(args) < 1 {
		log.Fatal("arg length is not enough")
	}

	client, err := ethclient.Dial(args[0])
	if err != nil {
		log.Fatalf("new ethclient fail: %v", err)
	}

	monkey.Init(&monkey.MonkeyConfig{
		Key:      "test/keystore/monkey.key",
		Endpoint: args[0],
		N:        30,
		Sleep:    3000,
		Timeout:  30,
	})
	m, nonce := monkey.Exec()

	time.Sleep(10 * time.Second)

	for _, key := range m.Keys() {
		currentNonce, err := client.NonceAt(context.Background(), crypto.PubkeyToAddress(key.PublicKey), nil)
		if err != nil {
			log.Fatalf("get address nonce fail: %v", err)
		}

		if currentNonce != nonce+1 {
			log.Fatalf("expect nonce %v but %v", nonce, currentNonce)
		}
	}
}

func doWaitForRecovery(args []string) {
	if len(args) < 2 {
		log.Fatal("arg length is not enough")
	}

	client, err := ethclient.Dial(args[0])
	if err != nil {
		log.Fatalf("new ethclient fail: %v", err)
	}

	lastBlock, err := client.BlockByNumber(context.Background(), nil)
	if err != nil {
		log.Fatalf("get last block fail %v", err)
	}
	t, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		log.Fatalf("timeout args invalid, %s", args[1])
	}

	sleep := time.Duration(t) * time.Second

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		for ctx.Err() == nil {
			select {
			case <-time.After(1 * time.Minute):
				log.Printf("tick")
			case <-ctx.Done():
				return
			}
		}
	}()

	log.Printf("Sleep %s", sleep)
	time.Sleep(sleep)
	cancel()

	// Check block height is increasing 5 time for safe.
	var block *types.Block
	for i := 0; i < 5; i++ {
		block, err = client.BlockByNumber(context.Background(), nil)
		if err != nil {
			log.Fatalf("get current block fail, err=%v, i=%d", err, i)
		}
		if block.NumberU64() <= lastBlock.NumberU64() {
			log.Fatalf("recovery fail, last=%d, current=%d",
				lastBlock.NumberU64(), block.NumberU64())
		}
		log.Printf("last=%d, current=%d, ok, sleep a while", lastBlock.NumberU64(), block.NumberU64())
		lastBlock = block
		time.Sleep(2 * time.Second)
	}
}
