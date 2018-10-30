// Copyright 2017 The DEXON Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package dexcon

import (
	"math/big"

	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/consensus"
	"github.com/dexon-foundation/dexon/core/state"
	"github.com/dexon-foundation/dexon/core/types"
	"github.com/dexon-foundation/dexon/params"
	"github.com/dexon-foundation/dexon/rpc"
)

type ConfigurationFetcher interface {
	DexconConfiguration(round uint64) *params.DexconConfig
}

// Dexcon is a delegated proof-of-stake consensus engine.
type Dexcon struct {
	configFetcher ConfigurationFetcher
}

// New creates a Clique proof-of-authority consensus engine with the initial
// signers set to the ones provided by the user.
func New() *Dexcon {
	return &Dexcon{}
}

// SetConfigFetcher sets the config fetcher for Dexcon. The reason this is not
// passed in the New() method is to bypass cycle dependencies when initializing
// dex backend.
func (d *Dexcon) SetConfigFetcher(fetcher ConfigurationFetcher) {
	d.configFetcher = fetcher
}

// Author implements consensus.Engine, returning the Ethereum address recovered
// from the signature in the header's extra-data section.
func (d *Dexcon) Author(header *types.Header) (common.Address, error) {
	return common.Address{}, nil
}

// VerifyHeader checks whether a header conforms to the consensus rules.
func (d *Dexcon) VerifyHeader(chain consensus.ChainReader, header *types.Header, seal bool) error {
	return nil
}

// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers. The
// method returns a quit channel to abort the operations and a results channel to
// retrieve the async verifications (the order is that of the input slice).
func (d *Dexcon) VerifyHeaders(chain consensus.ChainReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {
	abort, results := make(chan struct{}), make(chan error)
	go func() {
		for range headers {
			results <- nil
		}
	}()
	return abort, results
}

// verifyHeader checks whether a header conforms to the consensus rules.The
// caller may optionally pass in a batch of parents (ascending order) to avoid
// looking those up from the database. This is useful for concurrently verifying
// a batch of new headers.
func (d *Dexcon) verifyHeader(chain consensus.ChainReader, header *types.Header, parents []*types.Header) error {
	return nil
}

// verifyCascadingFields verifies all the header fields that are not standalone,
// rather depend on a batch of previous headers. The caller may optionally pass
// in a batch of parents (ascending order) to avoid looking those up from the
// database. This is useful for concurrently verifying a batch of new headers.
func (d *Dexcon) verifyCascadingFields(chain consensus.ChainReader, header *types.Header, parents []*types.Header) error {
	return nil
}

// VerifyUncles implements consensus.Engine, always returning an error for any
// uncles as this consensus mechanism doesn't permit uncles.
func (d *Dexcon) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	return nil
}

// VerifySeal implements consensus.Engine, checking whether the signature contained
// in the header satisfies the consensus protocol requirements.
func (d *Dexcon) VerifySeal(chain consensus.ChainReader, header *types.Header) error {
	return nil
}

// Prepare implements consensus.Engine, preparing all the consensus fields of the
// header for running the transactions on top.
func (d *Dexcon) Prepare(chain consensus.ChainReader, header *types.Header) error {
	return nil
}

// Finalize implements consensus.Engine, ensuring no uncles are set, nor block
// rewards given, and returns the final block.
func (d *Dexcon) Finalize(chain consensus.ChainReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {
	config := d.configFetcher.DexconConfiguration(header.Round)
	reward := new(big.Int).Div(config.BlockReward, big.NewInt(int64(config.NumChains)))
	state.AddBalance(header.Coinbase, reward)

	header.Reward = reward
	header.Root = state.IntermediateRoot(true)
	return types.NewBlock(header, txs, uncles, receipts), nil
}

// Seal implements consensus.Engine, attempting to create a sealed block using
// the local signing credentials.
func (d *Dexcon) Seal(chain consensus.ChainReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
	return nil
}

// SealHash returns the hash of a block prior to it being sealed.
func (d *Dexcon) SealHash(header *types.Header) (hash common.Hash) {
	return common.Hash{}
}

// CalcDifficulty is the difficulty adjustment algorithm. It returns the difficulty
// that a new block should have based on the previous blocks in the chain and the
// current signer.
func (d *Dexcon) CalcDifficulty(chain consensus.ChainReader, time uint64, parent *types.Header) *big.Int {
	return big.NewInt(0)
}

// Close implements consensus.Engine. It's a noop for clique as there is are no background threads.
func (d *Dexcon) Close() error {
	return nil
}

// APIs implements consensus.Engine, returning the user facing RPC API to allow
// controlling the signer voting.
func (d *Dexcon) APIs(chain consensus.ChainReader) []rpc.API {
	return []rpc.API{{
		Namespace: "dexcon",
		Version:   "1.0",
		Service:   &API{dexcon: d},
		Public:    false,
	}}
}
