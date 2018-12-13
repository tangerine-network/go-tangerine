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

package blockdb

import (
	coreCommon "github.com/dexon-foundation/dexon-consensus/common"
	coreDb "github.com/dexon-foundation/dexon-consensus/core/db"
	coreTypes "github.com/dexon-foundation/dexon-consensus/core/types"

	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/core/rawdb"
	"github.com/dexon-foundation/dexon/ethdb"
)

// BlockDB implement dexon-consensus BlockDatabase interface.
type BlockDB struct {
	db ethdb.Database
}

func NewDatabase(db ethdb.Database) *BlockDB {
	return &BlockDB{db}
}

func (d *BlockDB) HasBlock(hash coreCommon.Hash) bool {
	return rawdb.HasCoreBlock(d.db, common.Hash(hash))
}

func (d *BlockDB) GetBlock(hash coreCommon.Hash) (coreTypes.Block, error) {
	block := rawdb.ReadCoreBlock(d.db, common.Hash(hash))
	if block == nil {
		return coreTypes.Block{}, coreDb.ErrBlockDoesNotExist
	}
	return *block, nil
}

func (d *BlockDB) GetAllBlocks() (coreDb.BlockIterator, error) {
	return nil, coreDb.ErrNotImplemented
}

func (d *BlockDB) UpdateBlock(block coreTypes.Block) error {
	if !d.HasBlock(block.Hash) {
		return coreDb.ErrBlockDoesNotExist
	}
	rawdb.WriteCoreBlock(d.db, common.Hash(block.Hash), &block)
	return nil
}

func (d *BlockDB) PutBlock(block coreTypes.Block) error {
	if d.HasBlock(block.Hash) {
		return coreDb.ErrBlockExists
	}
	rawdb.WriteCoreBlock(d.db, common.Hash(block.Hash), &block)
	return nil
}

func (d *BlockDB) Close() error { return nil }
