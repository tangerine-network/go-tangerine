package rawdb

import (
	"bytes"

	coreDKG "github.com/dexon-foundation/dexon-consensus/core/crypto/dkg"
	"github.com/dexon-foundation/dexon/log"
	"github.com/dexon-foundation/dexon/rlp"
)

func ReadCoreDKGPrivateKeyRLP(db DatabaseReader, round uint64) rlp.RawValue {
	data, _ := db.Get(coreDKGPrivateKeyKey(round))
	return data
}

func WriteCoreDKGPrivateKeyRLP(db DatabaseWriter, round uint64, rlp rlp.RawValue) error {
	err := db.Put(coreDKGPrivateKeyKey(round), rlp)
	if err != nil {
		log.Crit("Failed to store core DKG private key", "err", err, "round", round)
	}
	return err
}

func HasCoreDKGPrivateKey(db DatabaseReader, round uint64) (bool, error) {
	return db.Has(coreDKGPrivateKeyKey(round))
}

func ReadCoreDKGPrivateKey(db DatabaseReader, round uint64) *coreDKG.PrivateKey {
	data := ReadCoreDKGPrivateKeyRLP(db, round)
	if len(data) == 0 {
		return nil
	}
	key := new(coreDKG.PrivateKey)
	if err := rlp.Decode(bytes.NewReader(data), key); err != nil {
		log.Error("Invalid core DKG private key RLP", "round", round, "err", err)
		return nil
	}
	return key
}

func WriteCoreDKGPrivateKey(db DatabaseWriter, round uint64, key *coreDKG.PrivateKey) error {
	data, err := rlp.EncodeToBytes(key)
	if err != nil {
		log.Crit("Failed to RLP encode core DKG private key", "round", round, "err", err)
		return err
	}
	return WriteCoreDKGPrivateKeyRLP(db, round, data)
}
