package dex

import (
	"sync"

	"github.com/dexon-foundation/dexon/event"
	"github.com/dexon-foundation/dexon/log"
	"github.com/dexon-foundation/dexon/p2p/enode"
	"github.com/dexon-foundation/dexon/p2p/enr"
)

type newRecordsEvent struct{ Records []*enr.Record }

type nodeTable struct {
	mu    sync.RWMutex
	entry map[enode.ID]*enode.Node
	feed  event.Feed
}

func newNodeTable() *nodeTable {
	return &nodeTable{
		entry: make(map[enode.ID]*enode.Node),
	}
}

func (t *nodeTable) GetNode(id enode.ID) *enode.Node {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.entry[id]
}

func (t *nodeTable) AddRecords(records []*enr.Record) {
	t.mu.Lock()
	defer t.mu.Unlock()

	var newRecords []*enr.Record
	for _, record := range records {
		node, err := enode.New(enode.ValidSchemes, record)
		if err != nil {
			log.Error("invalid node record", "err", err)
			return
		}

		if n, ok := t.entry[node.ID()]; ok && n.Seq() >= node.Seq() {
			log.Debug("Ignore new record, already exists", "id", node.ID().String(),
				"ip", node.IP().String(), "udp", node.UDP(), "tcp", node.TCP())
			continue
		}

		t.entry[node.ID()] = node
		newRecords = append(newRecords, record)
		log.Debug("Add new record to node table", "id", node.ID().String(),
			"ip", node.IP().String(), "udp", node.UDP(), "tcp", node.TCP())
	}
	t.feed.Send(newRecordsEvent{newRecords})
}

func (t *nodeTable) Records() []*enr.Record {
	t.mu.RLock()
	defer t.mu.RUnlock()
	records := make([]*enr.Record, 0, len(t.entry))
	for _, node := range t.entry {
		records = append(records, node.Record())
	}
	return records
}

func (t *nodeTable) SubscribeNewRecordsEvent(
	ch chan<- newRecordsEvent) event.Subscription {
	return t.feed.Subscribe(ch)
}
