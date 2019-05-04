package main

import (
	"flag"

	"github.com/dexon-foundation/dexon/cmd/zoo/monkey"
	"github.com/dexon-foundation/dexon/cmd/zoo/utils"
)

var key = flag.String("key", "", "private key path")
var endpoint = flag.String("endpoint", "http://127.0.0.1:8545", "JSON RPC endpoint")
var n = flag.Int("n", 100, "number of random accounts")
var gambler = flag.Bool("gambler", false, "make this monkey a gambler")
var batch = flag.Bool("batch", false, "monkeys will send transaction in batch")
var sleep = flag.Int("sleep", 500, "time in millisecond that monkeys sleep between each transaction")
var feeder = flag.Bool("feeder", false, "make this monkey a feeder")
var timeout = flag.Int("timeout", 0, "execution time limit after start")
var shutdown = flag.String("shutdown", "", "shutdown the previously opened zoo")

func main() {
	flag.Parse()

	if *shutdown != "" {
		utils.Shutdown(&utils.ShutdownConfig{
			Key:      *key,
			Endpoint: *endpoint,
			File:     *shutdown,
			Batch:    *batch,
		})
		return
	}

	monkey.Init(&monkey.MonkeyConfig{
		Key:      *key,
		Endpoint: *endpoint,
		N:        *n,
		Gambler:  *gambler,
		Feeder:   *feeder,
		Batch:    *batch,
		Sleep:    *sleep,
		Timeout:  *timeout,
	})
	monkey.Exec()
}
