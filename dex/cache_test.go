// Copyright 2018 The dexon-consensus Authors
// This file is part of the dexon-consensus library.
//
// The dexon-consensus library is free software: you can redistribute it
// and/or modify it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The dexon-consensus library is distributed in the hope that it will be
// useful, but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the dexon-consensus library. If not, see
// <http://www.gnu.org/licenses/>.

package dex

import (
	"sort"
	"strings"
	"testing"

	coreCommon "github.com/dexon-foundation/dexon-consensus/common"
	coreBlockdb "github.com/dexon-foundation/dexon-consensus/core/blockdb"
	coreTypes "github.com/dexon-foundation/dexon-consensus/core/types"
)

type byHash []*coreTypes.Vote

func (v byHash) Len() int {
	return len(v)
}

func (v byHash) Less(i int, j int) bool {
	return strings.Compare(v[i].BlockHash.String(), v[j].BlockHash.String()) < 0
}

func (v byHash) Swap(i int, j int) {
	v[i], v[j] = v[j], v[i]
}

func TestCacheVote(t *testing.T) {
	db, err := coreBlockdb.NewMemBackedBlockDB()
	if err != nil {
		panic(err)
	}
	cache := newCache(3, db)
	pos0 := coreTypes.Position{
		Height: uint64(0),
	}
	pos1 := coreTypes.Position{
		Height: uint64(1),
	}
	vote1 := &coreTypes.Vote{
		VoteHeader: coreTypes.VoteHeader{
			BlockHash: coreCommon.NewRandomHash(),
			Position:  pos0,
		},
	}
	vote2 := &coreTypes.Vote{
		VoteHeader: coreTypes.VoteHeader{
			BlockHash: coreCommon.NewRandomHash(),
			Position:  pos0,
		},
	}
	vote3 := &coreTypes.Vote{
		VoteHeader: coreTypes.VoteHeader{
			BlockHash: coreCommon.NewRandomHash(),
			Position:  pos1,
		},
	}
	vote4 := &coreTypes.Vote{
		VoteHeader: coreTypes.VoteHeader{
			BlockHash: coreCommon.NewRandomHash(),
			Position:  pos1,
		},
	}
	cache.addVote(vote1)
	cache.addVote(vote2)
	cache.addVote(vote3)

	votes := cache.votes(pos0)
	sort.Sort(byHash(votes))

	resultVotes := []*coreTypes.Vote{vote1, vote2}
	sort.Sort(byHash(resultVotes))

	if len(votes) != 2 {
		t.Errorf("fail to get votes: have %d, want 2", len(votes))
	}
	if !votes[0].BlockHash.Equal(resultVotes[0].BlockHash) {
		t.Errorf("get wrong vote: have %s, want %s", votes[0], resultVotes[0])
	}
	if !votes[1].BlockHash.Equal(resultVotes[1].BlockHash) {
		t.Errorf("get wrong vote: have %s, want %s", votes[1], resultVotes[1])
	}
	votes = cache.votes(pos1)
	sort.Sort(byHash(votes))
	if len(votes) != 1 {
		t.Errorf("fail to get votes: have %d, want 1", len(votes))
	}
	if !votes[0].BlockHash.Equal(vote3.BlockHash) {
		t.Errorf("get wrong vote: have %s, want %s", votes[0], vote3)
	}

	cache.addVote(vote4)

	votes = cache.votes(pos0)
	sort.Sort(byHash(votes))

	if len(votes) != 0 {
		t.Errorf("fail to get votes: have %d, want 0", len(votes))
	}
	votes = cache.votes(pos1)
	sort.Sort(byHash(votes))

	resultVotes = []*coreTypes.Vote{vote3, vote4}
	sort.Sort(byHash(resultVotes))

	if len(votes) != 2 {
		t.Errorf("fail to get votes: have %d, want 1", len(votes))
	}
	if !votes[0].BlockHash.Equal(resultVotes[0].BlockHash) {
		t.Errorf("get wrong vote: have %s, want %s", votes[0], resultVotes[0])
	}
	if !votes[1].BlockHash.Equal(resultVotes[1].BlockHash) {
		t.Errorf("get wrong vote: have %s, want %s", votes[1], resultVotes[1])
	}
}

func TestCacheBlock(t *testing.T) {
	db, err := coreBlockdb.NewMemBackedBlockDB()
	if err != nil {
		panic(err)
	}
	cache := newCache(3, db)
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
		block1.Hash: {},
		block2.Hash: {},
		block3.Hash: {},
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

	block5 := &coreTypes.Block{
		Hash: coreCommon.NewRandomHash(),
	}
	if err := db.Put(*block5); err != nil {
		panic(err)
	}
	blocks = cache.blocks(coreCommon.Hashes{block5.Hash})
	if len(blocks) != 1 {
		t.Errorf("fail to get blocks: have %d, want 1", len(blocks))
	} else {
		if !blocks[0].Hash.Equal(block5.Hash) {
			t.Errorf("get wrong block: have %s, want %s", blocks[0], block5)
		}
	}
}
