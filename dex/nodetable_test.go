package dex

import (
	"crypto/ecdsa"
	"net"
	"testing"
	"time"

	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/crypto"
	"github.com/dexon-foundation/dexon/p2p/enode"
)

func TestNodeTable(t *testing.T) {
	table := newNodeTable()
	ch := make(chan newMetasEvent)
	table.SubscribeNewMetasEvent(ch)

	metas1 := []*NodeMeta{
		{ID: randomID()},
		{ID: randomID()},
	}

	metas2 := []*NodeMeta{
		{ID: randomID()},
		{ID: randomID()},
	}

	go table.Add(metas1)

	select {
	case newMetas := <-ch:
		m := map[common.Hash]struct{}{}
		for _, meta := range newMetas.Metas {
			m[meta.Hash()] = struct{}{}
		}

		if len(m) != len(metas1) {
			t.Errorf("len mismatch: got %d, want: %d",
				len(m), len(metas1))
		}

		for _, meta := range metas1 {
			if _, ok := m[meta.Hash()]; !ok {
				t.Errorf("expected meta (%s) not exists", meta.Hash())
			}
		}
	case <-time.After(1 * time.Second):
		t.Error("did not receive new metas event within one second")
	}

	go table.Add(metas2)
	select {
	case newMetas := <-ch:
		m := map[common.Hash]struct{}{}
		for _, meta := range newMetas.Metas {
			m[meta.Hash()] = struct{}{}
		}

		if len(m) != len(metas1) {
			t.Errorf("len mismatch: got %d, want: %d",
				len(m), len(metas2))
		}

		for _, meta := range metas2 {
			if _, ok := m[meta.Hash()]; !ok {
				t.Errorf("expected meta (%s) not exists", meta.Hash())
			}
		}
	case <-time.After(1 * time.Second):
		t.Error("did not receive new metas event within one second")
	}

	var metas []*NodeMeta
	metas = append(metas, metas1...)
	metas = append(metas, metas2...)
	allMetas := table.Metas()
	if len(allMetas) != len(metas) {
		t.Errorf("all metas num mismatch: got %d, want %d",
			len(metas), len(allMetas))
	}

	for _, m := range metas {
		if m.Hash() != table.Get(m.ID).Hash() {
			t.Errorf("meta (%s) mismatch", m.ID.String())
		}
	}
}

func randomEnode() *enode.Node {
	var err error
	var privkey *ecdsa.PrivateKey
	for {
		privkey, err = crypto.GenerateKey()
		if err == nil {
			break
		}
	}
	return enode.NewV4(&privkey.PublicKey, net.IP{}, 0, 0)
}

func randomID() enode.ID {
	return randomEnode().ID()
}
