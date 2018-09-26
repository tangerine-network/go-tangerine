.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    github.com/dexon-foundation/dexon/cmd/swarm
	go build -o bin/geth     github.com/dexon-foundation/dexon/cmd/geth
	go build -o bin/bootnode github.com/dexon-foundation/dexon/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v github.com/dexon-foundation/dexon/swarm/...
