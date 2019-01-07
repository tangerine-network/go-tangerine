package dex

import (
	"crypto/ecdsa"
	"net"
	"testing"
	"time"

	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/crypto"
	"github.com/dexon-foundation/dexon/p2p/enode"
	"github.com/dexon-foundation/dexon/p2p/enr"
)

func TestNodeTable(t *testing.T) {
	table := newNodeTable()
	ch := make(chan newRecordsEvent)
	table.SubscribeNewRecordsEvent(ch)

	records1 := []*enr.Record{
		randomNode().Record(),
		randomNode().Record(),
	}

	records2 := []*enr.Record{
		randomNode().Record(),
		randomNode().Record(),
	}

	go table.AddRecords(records1)

	select {
	case newRecords := <-ch:
		m := map[common.Hash]struct{}{}
		for _, record := range newRecords.Records {
			m[rlpHash(record)] = struct{}{}
		}

		if len(m) != len(records1) {
			t.Errorf("len mismatch: got %d, want: %d",
				len(m), len(records1))
		}

		for _, record := range records1 {
			if _, ok := m[rlpHash(record)]; !ok {
				t.Errorf("expected record (%s) not exists", rlpHash(record))
			}
		}
	case <-time.After(1 * time.Second):
		t.Error("did not receive new records event within one second")
	}

	go table.AddRecords(records2)
	select {
	case newRecords := <-ch:
		m := map[common.Hash]struct{}{}
		for _, record := range newRecords.Records {
			m[rlpHash(record)] = struct{}{}
		}

		if len(m) != len(records2) {
			t.Errorf("len mismatch: got %d, want: %d",
				len(m), len(records2))
		}

		for _, record := range records2 {
			if _, ok := m[rlpHash(record)]; !ok {
				t.Errorf("expected record (%s) not exists", rlpHash(record))
			}
		}
	case <-time.After(1 * time.Second):
		t.Error("did not receive new records event within one second")
	}

	var records []*enr.Record
	records = append(records, records1...)
	records = append(records, records2...)
	allRecords := table.Records()
	if len(allRecords) != len(records) {
		t.Errorf("all metas num mismatch: got %d, want %d",
			len(records), len(allRecords))
	}

	for _, r := range records {
		n, err := enode.New(enode.V4ID{}, r)
		if err != nil {
			t.Errorf(err.Error())
		}
		if rlpHash(r) != rlpHash(table.GetNode(n.ID()).Record()) {
			t.Errorf("record (%s) mismatch", n.ID().String())
		}
	}
}

func randomNode() *enode.Node {
	var err error
	var privkey *ecdsa.PrivateKey
	for {
		privkey, err = crypto.GenerateKey()
		if err == nil {
			break
		}
	}
	var r enr.Record
	r.Set(enr.IP(net.IP{}))
	r.Set(enr.UDP(0))
	r.Set(enr.TCP(0))
	if err := enode.SignV4(&r, privkey); err != nil {
		panic(err)
	}
	node, err := enode.New(enode.V4ID{}, &r)
	if err != nil {
		panic(err)

	}
	return node
}

func randomID() enode.ID {
	return randomNode().ID()
}
