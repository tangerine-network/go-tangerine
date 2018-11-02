// Copyright 2018 The dexon-consensus-core Authors
// This file is part of the dexon-consensus-core library.
//
// The dexon-consensus-core library is free software: you can redistribute it
// and/or modify it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The dexon-consensus-core library is distributed in the hope that it will be
// useful, but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the dexon-consensus-core library. If not, see
// <http://www.gnu.org/licenses/>.

package dex

import (
	"testing"

	coreCommon "github.com/dexon-foundation/dexon-consensus-core/common"
	coreTypes "github.com/dexon-foundation/dexon-consensus-core/core/types"
)

func TestCacheVote(t *testing.T) {
	cache := newCache(3)
	pos0 := coreTypes.Position{
		Height: uint64(0),
	}
	pos1 := coreTypes.Position{
		Height: uint64(1),
	}
	vote1 := &coreTypes.Vote{
		BlockHash: coreCommon.NewRandomHash(),
		Position:  pos0,
	}
	vote2 := &coreTypes.Vote{
		BlockHash: coreCommon.NewRandomHash(),
		Position:  pos0,
	}
	vote3 := &coreTypes.Vote{
		BlockHash: coreCommon.NewRandomHash(),
		Position:  pos1,
	}
	vote4 := &coreTypes.Vote{
		BlockHash: coreCommon.NewRandomHash(),
		Position:  pos1,
	}
	cache.addVote(vote1)
	cache.addVote(vote2)
	cache.addVote(vote3)

	votes := cache.votes(pos0)
	if len(votes) != 2 {
		t.Errorf("fail to get votes: have %d, want 2", len(votes))
	}
	if !votes[0].BlockHash.Equal(vote1.BlockHash) {
		t.Errorf("get wrong vote: have %s, want %s", votes[0], vote1)
	}
	if !votes[1].BlockHash.Equal(vote2.BlockHash) {
		t.Errorf("get wrong vote: have %s, want %s", votes[1], vote2)
	}
	votes = cache.votes(pos1)
	if len(votes) != 1 {
		t.Errorf("fail to get votes: have %d, want 1", len(votes))
	}
	if !votes[0].BlockHash.Equal(vote3.BlockHash) {
		t.Errorf("get wrong vote: have %s, want %s", votes[0], vote3)
	}

	cache.addVote(vote4)

	votes = cache.votes(pos0)
	if len(votes) != 0 {
		t.Errorf("fail to get votes: have %d, want 0", len(votes))
	}
	votes = cache.votes(pos1)
	if len(votes) != 2 {
		t.Errorf("fail to get votes: have %d, want 1", len(votes))
	}
	if !votes[0].BlockHash.Equal(vote3.BlockHash) {
		t.Errorf("get wrong vote: have %s, want %s", votes[0], vote3)
	}
	if !votes[1].BlockHash.Equal(vote4.BlockHash) {
		t.Errorf("get wrong vote: have %s, want %s", votes[1], vote4)
	}
}

func TestCacheBlock(t *testing.T) {
	cache := newCache(3)
	block1 := &coreTypes.Block{
		Hash: coreCommon.NewRandomHash(),
	}
	block2 := &coreTypes.Block{
		Hash: coreCommon.NewRandomHash(),
	}
	block3 := &coreTypes.Block{
		Hash: coreCommon.NewRandomHash(),
	}
	block4 := &coreTypes.Block{
		Hash: coreCommon.NewRandomHash(),
	}
	cache.addBlock(block1)
	cache.addBlock(block2)
	cache.addBlock(block3)

	hashes := coreCommon.Hashes{block1.Hash, block2.Hash, block3.Hash, block4.Hash}
	hashMap := map[coreCommon.Hash]struct{}{
		block1.Hash: struct{}{},
		block2.Hash: struct{}{},
		block3.Hash: struct{}{},
	}
	blocks := cache.blocks(hashes)
	if len(blocks) != 3 {
		t.Errorf("fail to get blocks: have %d, want 3", len(blocks))
	}
	for _, block := range blocks {
		if _, exist := hashMap[block.Hash]; !exist {
			t.Errorf("get wrong block: have %s, want %v", block, hashMap)
		}
	}

	cache.addBlock(block4)

	blocks = cache.blocks(hashes)
	hashMap[block4.Hash] = struct{}{}
	if len(blocks) != 3 {
		t.Errorf("fail to get blocks: have %d, want 3", len(blocks))
	}
	hasNewBlock := false
	for _, block := range blocks {
		if _, exist := hashMap[block.Hash]; !exist {
			t.Errorf("get wrong block: have %s, want %v", block, hashMap)
		}
		if block.Hash.Equal(block4.Hash) {
			hasNewBlock = true
		}
	}
	if !hasNewBlock {
		t.Errorf("expect block %s in cache, have %v", block4, blocks)
	}
}
