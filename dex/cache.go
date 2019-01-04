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
	"sync"

	coreCommon "github.com/dexon-foundation/dexon-consensus/common"
	coreDb "github.com/dexon-foundation/dexon-consensus/core/db"
	coreTypes "github.com/dexon-foundation/dexon-consensus/core/types"
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
	lock            sync.RWMutex
	blockCache      map[coreCommon.Hash]*coreTypes.Block
	voteCache       map[coreTypes.Position]map[voteKey]*coreTypes.Vote
	randomnessCache map[coreCommon.Hash]*coreTypes.BlockRandomnessResult
	votePosition    []coreTypes.Position
	db              coreDb.Database
	voteSize        int
	size            int
}

func newCache(size int, db coreDb.Database) *cache {
	return &cache{
		blockCache:      make(map[coreCommon.Hash]*coreTypes.Block),
		voteCache:       make(map[coreTypes.Position]map[voteKey]*coreTypes.Vote),
		randomnessCache: make(map[coreCommon.Hash]*coreTypes.BlockRandomnessResult),
		db:              db,
		size:            size,
	}
}

func (c *cache) addVote(vote *coreTypes.Vote) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.voteSize >= c.size {
		pos := c.votePosition[0]
		c.voteSize -= len(c.voteCache[pos])
		delete(c.voteCache, pos)
		c.votePosition = c.votePosition[1:]
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
	block = block.Clone()
	block.Finalization.Height = 0
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
		} else {
			block, err := c.db.GetBlock(hash)
			if err != nil {
				continue
			}
			// Blocks request from the cache do not need the finalization info.
			block.Finalization = coreTypes.FinalizationResult{}
			cacheBlocks = append(cacheBlocks, &block)
		}
	}
	return cacheBlocks
}

func (c *cache) addRandomness(rand *coreTypes.BlockRandomnessResult) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if len(c.randomnessCache) >= c.size {
		// Randomly delete one entry.
		for k := range c.randomnessCache {
			delete(c.randomnessCache, k)
			break
		}
	}
	c.randomnessCache[rand.BlockHash] = rand
}

func (c *cache) randomness(hashes coreCommon.Hashes) []*coreTypes.BlockRandomnessResult {
	c.lock.RLock()
	defer c.lock.RUnlock()
	cacheRandomnesss := make([]*coreTypes.BlockRandomnessResult, 0, len(hashes))
	for _, hash := range hashes {
		if block, exist := c.randomnessCache[hash]; exist {
			cacheRandomnesss = append(cacheRandomnesss, block)
		} else {
			block, err := c.db.GetBlock(hash)
			if err != nil {
				continue
			}
			if len(block.Finalization.Randomness) == 0 {
				continue
			}
			cacheRandomnesss = append(cacheRandomnesss, &coreTypes.BlockRandomnessResult{
				BlockHash:  block.Hash,
				Position:   block.Position,
				Randomness: block.Finalization.Randomness,
			})
		}
	}
	return cacheRandomnesss
}
