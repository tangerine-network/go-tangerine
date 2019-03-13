package rawdb

import (
	"bytes"

	coreDKG "github.com/dexon-foundation/dexon-consensus/core/crypto/dkg"
	"github.com/dexon-foundation/dexon/log"
	"github.com/dexon-foundation/dexon/rlp"
)

func ReadCoreDKGMasterPrivateSharesRLP(db DatabaseReader, round uint64) rlp.RawValue {
	data, _ := db.Get(coreDKGMasterPrivateSharesKey(round))
	return data
}

func WriteCoreDKGMasterPrivateSharesRLP(db DatabaseWriter, round uint64, rlp rlp.RawValue) error {
	err := db.Put(coreDKGMasterPrivateSharesKey(round), rlp)
	if err != nil {
		log.Crit("Failed to store core DKG private key", "err", err, "round", round)
	}
	return err
}

func ReadCoreDKGMasterPrivateShares(db DatabaseReader, round uint64) *coreDKG.PrivateKeyShares {
	data := ReadCoreDKGMasterPrivateSharesRLP(db, round)
	if len(data) == 0 {
		return nil
	}
	shares := new(coreDKG.PrivateKeyShares)
	if err := rlp.Decode(bytes.NewReader(data), shares); err != nil {
		log.Error("Invalid core DKG master private shares RLP", "round", round, "err", err)
		return nil
	}
	return shares
}

func WriteCoreDKGMasterPrivateShares(db DatabaseWriter, round uint64, shares *coreDKG.PrivateKeyShares) error {
	data, err := rlp.EncodeToBytes(shares)
	if err != nil {
		log.Crit("Failed to RLP encode core DKG master private shares", "round", round, "err", err)
		return err
	}
	return WriteCoreDKGMasterPrivateSharesRLP(db, round, data)
}
