package rawdb

import (
	"bytes"

	"github.com/tangerine-network/go-tangerine/log"
	"github.com/tangerine-network/go-tangerine/rlp"
	coreDb "github.com/tangerine-network/tangerine-consensus/core/db"
)

func ReadCoreDKGProtocolRLP(db DatabaseReader) rlp.RawValue {
	data, _ := db.Get(coreDKGProtocolKey)
	return data
}

func WriteCoreDKGProtocolRLP(db DatabaseWriter, rlp rlp.RawValue) error {
	err := db.Put(coreDKGProtocolKey, rlp)
	if err != nil {
		log.Crit("Failed to store core DKG private key", "err", err)
	}
	return err
}

func ReadCoreDKGProtocol(db DatabaseReader) *coreDb.DKGProtocolInfo {
	data := ReadCoreDKGProtocolRLP(db)
	if len(data) == 0 {
		return nil
	}
	protocol := new(coreDb.DKGProtocolInfo)
	if err := rlp.Decode(bytes.NewReader(data), protocol); err != nil {
		log.Error("Invalid core DKG protocol RLP", "err", err)
		return nil
	}
	return protocol
}

func WriteCoreDKGProtocol(db DatabaseWriter, protocol *coreDb.DKGProtocolInfo) error {
	data, err := rlp.EncodeToBytes(protocol)
	if err != nil {
		log.Crit("Failed to RLP encode core DKG protocol", "err", err)
		return err
	}
	return WriteCoreDKGProtocolRLP(db, data)
}
