package state

import (
	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/core/types"
	"github.com/dexon-foundation/dexon/trie"
)

func GetGovState(statedb *StateDB, header *types.Header,
	addr common.Address) (*types.GovState, error) {
	proof, err := statedb.GetProof(addr)
	if err != nil {
		return nil, err
	}

	govState := &types.GovState{
		BlockHash: header.Hash(),
		Number:    header.Number,
		Root:      header.Root,
		Proof:     proof,
	}

	if t := statedb.StorageTrie(addr); t != nil {
		it := trie.NewIterator(t.NodeIterator(nil))
		for it.Next() {
			govState.Storage = append(govState.Storage,
				[2][]byte{it.Key, it.Value})
		}
	}
	return govState, nil
}
