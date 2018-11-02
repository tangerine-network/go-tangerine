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
	"sync"

	coreCommon "github.com/dexon-foundation/dexon-consensus-core/common"
	coreTypes "github.com/dexon-foundation/dexon-consensus-core/core/types"
)

type voteKey struct {
	ProposerID coreTypes.NodeID
	Type       coreTypes.VoteType
	BlockHash  coreCommon.Hash
	Period     uint64
	Position   coreTypes.Position
}

func voteToKey(vote *coreTypes.Vote) voteKey {
	return voteKey{
		ProposerID: vote.ProposerID,
		Type:       vote.Type,
		BlockHash:  vote.BlockHash,
		Period:     vote.Period,
		Position:   vote.Position,
	}
}

type cache struct {
	lock         sync.RWMutex
	blockCache   map[coreCommon.Hash]*coreTypes.Block
	voteCache    map[coreTypes.Position]map[voteKey]*coreTypes.Vote
	votePosition []coreTypes.Position
	voteSize     int
	size         int
}

func newCache(size int) *cache {
	return &cache{
		blockCache: make(map[coreCommon.Hash]*coreTypes.Block),
		voteCache:  make(map[coreTypes.Position]map[voteKey]*coreTypes.Vote),
		size:       size,
	}
}

func (c *cache) addVote(vote *coreTypes.Vote) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.voteSize >= c.size {
		pos := c.votePosition[0]
		c.voteSize -= len(c.voteCache[pos])
		delete(c.voteCache, pos)
	}
	if _, exist := c.voteCache[vote.Position]; !exist {
		c.votePosition = append(c.votePosition, vote.Position)
		c.voteCache[vote.Position] = make(map[voteKey]*coreTypes.Vote)
	}
	key := voteToKey(vote)
	if _, exist := c.voteCache[vote.Position][key]; exist {
		return
	}
	c.voteCache[vote.Position][key] = vote
	c.voteSize++
}

func (c *cache) votes(pos coreTypes.Position) []*coreTypes.Vote {
	c.lock.RLock()
	defer c.lock.RUnlock()
	votes := make([]*coreTypes.Vote, 0, len(c.voteCache[pos]))
	for _, vote := range c.voteCache[pos] {
		votes = append(votes, vote)
	}
	return votes
}

func (c *cache) addBlock(block *coreTypes.Block) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if len(c.blockCache) >= c.size {
		// Randomly delete one entry.
		for k := range c.blockCache {
			delete(c.blockCache, k)
			break
		}
	}
	c.blockCache[block.Hash] = block
}

func (c *cache) blocks(hashes coreCommon.Hashes) []*coreTypes.Block {
	c.lock.RLock()
	defer c.lock.RUnlock()
	cacheBlocks := make([]*coreTypes.Block, 0, len(hashes))
	for _, hash := range hashes {
		if block, exist := c.blockCache[hash]; exist {
			cacheBlocks = append(cacheBlocks, block)
		}
	}
	return cacheBlocks
}
