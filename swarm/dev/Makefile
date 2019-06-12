.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    github.com/tangerine-network/go-tangerine/cmd/swarm
	go build -o bin/geth     github.com/tangerine-network/go-tangerine/cmd/geth
	go build -o bin/bootnode github.com/tangerine-network/go-tangerine/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v github.com/tangerine-network/go-tangerine/swarm/...
