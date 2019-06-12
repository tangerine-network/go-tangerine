# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: gtan android ios gtan-cross swarm evm all test clean
.PHONY: gtan-linux gtan-linux-386 gtan-linux-amd64 gtan-linux-mips64 gtan-linux-mips64le
.PHONY: gtan-linux-arm gtan-linux-arm-5 gtan-linux-arm-6 gtan-linux-arm-7 gtan-linux-arm64
.PHONY: gtan-darwin gtan-darwin-386 gtan-darwin-amd64
.PHONY: gtan-windows gtan-windows-386 gtan-windows-amd64

GOBIN = $(shell pwd)/build/bin
GO ?= latest

gtan: libbls
	build/env.sh go run build/ci.go install ./cmd/gtan
	@echo "Done building."
	@echo "Run \"$(GOBIN)/gtan\" to launch gtan."

swarm: libbls
	build/env.sh go run build/ci.go install ./cmd/swarm
	@echo "Done building."
	@echo "Run \"$(GOBIN)/swarm\" to launch swarm."

all: libbls
	build/env.sh go run build/ci.go install

android:
	build/env.sh go run build/ci.go aar --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/gtan.aar\" to use the library."

ios:
	build/env.sh go run build/ci.go xcode --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/Geth.framework\" to use the library."

test: all libbls
	build/env.sh go run build/ci.go test

lint: ## Run linters.
	build/env.sh go run build/ci.go lint

libbls:
	make -C vendor/github.com/byzantine-lab/bls MCL_USE_OPENSSL=0 lib/libbls384.a

clean-cgo:
	make -C vendor/github.com/byzantine-lab/bls clean
	make -C vendor/github.com/byzantine-lab/mcl clean

clean: clean-cgo
	./build/clean_go_build_cache.sh
	rm -fr build/_workspace/pkg/ $(GOBIN)/*

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

devtools:
	env GOBIN= go get -u golang.org/x/tools/cmd/stringer
	env GOBIN= go get -u github.com/kevinburke/go-bindata/go-bindata
	env GOBIN= go get -u github.com/fjl/gencodec
	env GOBIN= go get -u github.com/golang/protobuf/protoc-gen-go
	env GOBIN= go install ./cmd/abigen
	@type "npm" 2> /dev/null || echo 'Please install node.js and npm'
	@type "solc" 2> /dev/null || echo 'Please install solc'
	@type "protoc" 2> /dev/null || echo 'Please install protoc'

swarm-devtools:
	env GOBIN= go install ./cmd/swarm/mimegen

# Cross Compilation Targets (xgo)

gtan-cross: gtan-linux gtan-darwin gtan-windows gtan-android gtan-ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/gtan-*

gtan-linux: gtan-linux-386 gtan-linux-amd64 gtan-linux-arm gtan-linux-mips64 gtan-linux-mips64le
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/gtan-linux-*

gtan-linux-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/386 -v ./cmd/gtan
	@echo "Linux 386 cross compilation done:"
	@ls -ld $(GOBIN)/gtan-linux-* | grep 386

gtan-linux-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./cmd/gtan
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gtan-linux-* | grep amd64

gtan-linux-arm: gtan-linux-arm-5 gtan-linux-arm-6 gtan-linux-arm-7 gtan-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/gtan-linux-* | grep arm

gtan-linux-arm-5:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-5 -v ./cmd/gtan
	@echo "Linux ARMv5 cross compilation done:"
	@ls -ld $(GOBIN)/gtan-linux-* | grep arm-5

gtan-linux-arm-6:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-6 -v ./cmd/gtan
	@echo "Linux ARMv6 cross compilation done:"
	@ls -ld $(GOBIN)/gtan-linux-* | grep arm-6

gtan-linux-arm-7:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-7 -v ./cmd/gtan
	@echo "Linux ARMv7 cross compilation done:"
	@ls -ld $(GOBIN)/gtan-linux-* | grep arm-7

gtan-linux-arm64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 -v ./cmd/gtan
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/gtan-linux-* | grep arm64

gtan-linux-mips:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./cmd/gtan
	@echo "Linux MIPS cross compilation done:"
	@ls -ld $(GOBIN)/gtan-linux-* | grep mips

gtan-linux-mipsle:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./cmd/gtan
	@echo "Linux MIPSle cross compilation done:"
	@ls -ld $(GOBIN)/gtan-linux-* | grep mipsle

gtan-linux-mips64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./cmd/gtan
	@echo "Linux MIPS64 cross compilation done:"
	@ls -ld $(GOBIN)/gtan-linux-* | grep mips64

gtan-linux-mips64le:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./cmd/gtan
	@echo "Linux MIPS64le cross compilation done:"
	@ls -ld $(GOBIN)/gtan-linux-* | grep mips64le

gtan-darwin: gtan-darwin-386 gtan-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/gtan-darwin-*

gtan-darwin-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./cmd/gtan
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/gtan-darwin-* | grep 386

gtan-darwin-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./cmd/gtan
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gtan-darwin-* | grep amd64

gtan-windows: gtan-windows-386 gtan-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/gtan-windows-*

gtan-windows-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/386 -v ./cmd/gtan
	@echo "Windows 386 cross compilation done:"
	@ls -ld $(GOBIN)/gtan-windows-* | grep 386

gtan-windows-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 -v ./cmd/gtan
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gtan-windows-* | grep amd64
