package rawdb

import (
	"bytes"

	coreTypes "github.com/dexon-foundation/dexon-consensus/core/types"

	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/log"
	"github.com/dexon-foundation/dexon/rlp"
)

func ReadCoreBlockRLP(db DatabaseReader, hash common.Hash) rlp.RawValue {
	data, _ := db.Get(coreBlockKey(hash))
	return data
}

func WriteCoreBlockRLP(db DatabaseWriter, hash common.Hash, rlp rlp.RawValue) {
	if err := db.Put(coreBlockKey(hash), rlp); err != nil {
		log.Crit("Failed to store core block", "err", err)
	}
}

func HasCoreBlock(db DatabaseReader, hash common.Hash) bool {
	if has, err := db.Has(coreBlockKey(hash)); !has || err != nil {
		return false
	}
	return true
}

func ReadCoreBlock(db DatabaseReader, hash common.Hash) *coreTypes.Block {
	data := ReadCoreBlockRLP(db, hash)
	if len(data) == 0 {
		return nil
	}

	block := new(coreTypes.Block)
	if err := rlp.Decode(bytes.NewReader(data), block); err != nil {
		log.Error("Invalid core block RLP", "hash", hash, "err", err)
		return nil
	}
	return block
}

func WriteCoreBlock(db DatabaseWriter, hash common.Hash, block *coreTypes.Block) {
	data, err := rlp.EncodeToBytes(block)
	if err != nil {
		log.Crit("Failed to RLP encode core block", "err", err)
	}
	WriteCoreBlockRLP(db, hash, data)
}
