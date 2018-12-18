package rawdb

import (
	"bytes"

	coreCommon "github.com/dexon-foundation/dexon-consensus/common"
	"github.com/dexon-foundation/dexon/log"
	"github.com/dexon-foundation/dexon/rlp"
)

func ReadCoreCompactionChainTipRLP(db DatabaseReader) (rlp.RawValue, error) {
	return db.Get(coreCompactionChainTipKey)
}

func WriteCoreCompactionChainTipRLP(db DatabaseWriter, rlp rlp.RawValue) error {
	if err := db.Put(coreCompactionChainTipKey, rlp); err != nil {
		log.Crit("Failed to store core compaction chain tip")
		return err
	}
	return nil
}

func ReadCoreCompactionChainTip(db DatabaseReader) (coreCommon.Hash, uint64) {
	data, err := ReadCoreCompactionChainTipRLP(db)
	if err != nil {
		return coreCommon.Hash{}, 0
	}
	v := struct {
		Height uint64
		Hash   coreCommon.Hash
	}{}
	if err := rlp.Decode(bytes.NewReader(data), &v); err != nil {
		log.Error("Invalid core compaction chain tip RLP", "err", err)
		return coreCommon.Hash{}, 0
	}
	return v.Hash, v.Height
}

func WriteCoreCompactionChainTip(db DatabaseWriter, hash coreCommon.Hash, height uint64) error {
	data, err := rlp.EncodeToBytes(&struct {
		Height uint64
		Hash   coreCommon.Hash
	}{height, hash})
	if err != nil {
		log.Crit("Failed to RLP encode core compaction chain tip", "err", err)
		return err
	}
	return WriteCoreCompactionChainTipRLP(db, data)
}
