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

package db

import (
	coreCommon "github.com/dexon-foundation/dexon-consensus/common"
	coreDKG "github.com/dexon-foundation/dexon-consensus/core/crypto/dkg"
	coreDb "github.com/dexon-foundation/dexon-consensus/core/db"
	coreTypes "github.com/dexon-foundation/dexon-consensus/core/types"

	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/core/rawdb"
	"github.com/dexon-foundation/dexon/ethdb"
)

// DB implement dexon-consensus BlockDatabase interface.
type DB struct {
	db ethdb.Database
}

func NewDatabase(db ethdb.Database) *DB {
	return &DB{db}
}

func (d *DB) HasBlock(hash coreCommon.Hash) bool {
	return rawdb.HasCoreBlock(d.db, common.Hash(hash))
}

func (d *DB) GetBlock(hash coreCommon.Hash) (coreTypes.Block, error) {
	block := rawdb.ReadCoreBlock(d.db, common.Hash(hash))
	if block == nil {
		return coreTypes.Block{}, coreDb.ErrBlockDoesNotExist
	}
	return *block, nil
}

func (d *DB) GetAllBlocks() (coreDb.BlockIterator, error) {
	return nil, coreDb.ErrNotImplemented
}

func (d *DB) UpdateBlock(block coreTypes.Block) error {
	if !d.HasBlock(block.Hash) {
		return coreDb.ErrBlockDoesNotExist
	}
	rawdb.WriteCoreBlock(d.db, common.Hash(block.Hash), &block)
	return nil
}

func (d *DB) PutBlock(block coreTypes.Block) error {
	if d.HasBlock(block.Hash) {
		return coreDb.ErrBlockExists
	}
	rawdb.WriteCoreBlock(d.db, common.Hash(block.Hash), &block)
	return nil
}

func (d *DB) HasDKGPrivateKey(round uint64) (bool, error) {
	return rawdb.HasCoreDKGPrivateKey(d.db, round)
}

func (d *DB) GetDKGPrivateKey(round uint64) (coreDKG.PrivateKey, error) {
	key := rawdb.ReadCoreDKGPrivateKey(d.db, round)
	if key == nil {
		return coreDKG.PrivateKey{}, coreDb.ErrDKGPrivateKeyDoesNotExist
	}
	return *key, nil
}

func (d *DB) PutDKGPrivateKey(round uint64, key coreDKG.PrivateKey) error {
	has, err := d.HasDKGPrivateKey(round)
	if err != nil {
		return err
	}
	if has {
		return coreDb.ErrDKGPrivateKeyExists
	}
	return rawdb.WriteCoreDKGPrivateKey(d.db, round, &key)
}

func (d *DB) PutCompactionChainTipInfo(hash coreCommon.Hash, height uint64) error {
	_, currentHeight := d.GetCompactionChainTipInfo()
	if height <= currentHeight {
		return coreDb.ErrInvalidCompactionChainTipHeight
	}
	return rawdb.WriteCoreCompactionChainTip(d.db, hash, height)
}

func (d *DB) GetCompactionChainTipInfo() (hash coreCommon.Hash, height uint64) {
	return rawdb.ReadCoreCompactionChainTip(d.db)
}

func (d *DB) PutOrUpdateDKGMasterPrivateShares(
	round uint64, shares coreDKG.PrivateKeyShares) error {
	return rawdb.WriteCoreDKGMasterPrivateShares(d.db, round, &shares)
}

func (d *DB) GetDKGMasterPrivateShares(round uint64) (
	shares coreDKG.PrivateKeyShares, err error) {
	mpk := rawdb.ReadCoreDKGMasterPrivateShares(d.db, round)
	if mpk == nil {
		return coreDKG.PrivateKeyShares{}, coreDb.ErrDKGMasterPrivateSharesDoesNotExist
	}
	return *mpk, nil
}

func (d *DB) Close() error { return nil }
