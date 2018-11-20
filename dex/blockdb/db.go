package blockdb

import (
	coreCommon "github.com/dexon-foundation/dexon-consensus/common"
	coreBlockdb "github.com/dexon-foundation/dexon-consensus/core/blockdb"
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

func (d *BlockDB) Has(hash coreCommon.Hash) bool {
	return rawdb.HasCoreBlock(d.db, common.Hash(hash))
}

func (d *BlockDB) Get(hash coreCommon.Hash) (coreTypes.Block, error) {
	block := rawdb.ReadCoreBlock(d.db, common.Hash(hash))
	if block == nil {
		return coreTypes.Block{}, coreBlockdb.ErrBlockDoesNotExist
	}
	return *block, nil
}

func (d *BlockDB) GetAll() (coreBlockdb.BlockIterator, error) {
	return nil, coreBlockdb.ErrNotImplemented
}

func (d *BlockDB) Update(block coreTypes.Block) error {
	if !d.Has(block.Hash) {
		return coreBlockdb.ErrBlockDoesNotExist
	}
	rawdb.WriteCoreBlock(d.db, common.Hash(block.Hash), &block)
	return nil
}

func (d *BlockDB) Put(block coreTypes.Block) error {
	if d.Has(block.Hash) {
		return coreBlockdb.ErrBlockExists
	}
	rawdb.WriteCoreBlock(d.db, common.Hash(block.Hash), &block)
	return nil
}

func (d *BlockDB) Close() error { return nil }
