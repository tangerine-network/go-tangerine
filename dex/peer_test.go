package dex

import (
	"fmt"
	"math/big"
	"testing"

	mapset "github.com/deckarep/golang-set"
	"github.com/dexon-foundation/dexon/crypto"
	"github.com/dexon-foundation/dexon/p2p/discover"
	"github.com/dexon-foundation/dexon/p2p/enode"
)

func TestPeerSetBuildAndForgetNotaryConn(t *testing.T) {
	self := discover.Node{ID: nodeID(0)}
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	server := newTestP2PServer(&self, key)
	table := newNodeTable()

	gov := &testGovernance{
		numChainsFunc: func(uint64) uint32 {
			return 3
		},
	}

	round10 := [][]enode.ID{
		[]enode.ID{nodeID(0), nodeID(1), nodeID(2)},
		[]enode.ID{nodeID(1), nodeID(3)},
		[]enode.ID{nodeID(2), nodeID(4)},
	}
	round11 := [][]enode.ID{
		[]enode.ID{nodeID(0), nodeID(1), nodeID(5)},
		[]enode.ID{nodeID(5), nodeID(6)},
		[]enode.ID{nodeID(0), nodeID(2), nodeID(4)},
	}
	round12 := [][]enode.ID{
		[]enode.ID{nodeID(0), nodeID(3), nodeID(5)},
		[]enode.ID{nodeID(0), nodeID(7), nodeID(8)},
		[]enode.ID{nodeID(0), nodeID(2), nodeID(6)},
	}

	gov.notarySetFunc = func(
		round uint64, cid uint32) (map[string]struct{}, error) {
		m := map[uint64][][]enode.ID{
			10: round10,
			11: round11,
			12: round12,
		}
		return newTestNodeSet(m[round][cid]), nil
	}

	ps := newPeerSet(gov, server, table)
	peer1 := newDummyPeer(nodeID(1))
	peer2 := newDummyPeer(nodeID(2))
	err = ps.Register(peer1)
	if err != nil {
		t.Error(err)
	}
	err = ps.Register(peer2)
	if err != nil {
		t.Error(err)
	}

	// build round 10
	ps.BuildNotaryConn(10)

	err = checkLabels(peer1, []peerLabel{
		peerLabel{notaryset, 0, 10},
	})
	if err != nil {
		t.Error(err)
	}
	err = checkLabels(peer2, []peerLabel{
		peerLabel{notaryset, 0, 10},
	})
	if err != nil {
		t.Error(err)
	}
	err = checkPeer2Labels(ps, map[string][]peerLabel{
		nodeID(1).String(): []peerLabel{
			peerLabel{notaryset, 0, 10},
		},
		nodeID(2).String(): []peerLabel{
			peerLabel{notaryset, 0, 10},
		},
	})
	if err != nil {
		t.Error(err)
	}
	err = checkPeerSetHistory(ps, []uint64{10}, notaryset)
	if err != nil {
		t.Error(err)
	}
	err = checkDirectPeer(server, []enode.ID{
		nodeID(1), nodeID(2),
	})
	if err != nil {
		t.Error(err)
	}
	err = checkGroup(server, []string{
		notarySetName(1, 10),
		notarySetName(2, 10),
	})
	if err != nil {
		t.Error(err)
	}

	// build round 11
	ps.BuildNotaryConn(11)

	err = checkLabels(peer1, []peerLabel{
		peerLabel{notaryset, 0, 10},
		peerLabel{notaryset, 0, 11},
	})
	if err != nil {
		t.Error(err)
	}
	err = checkLabels(peer2, []peerLabel{
		peerLabel{notaryset, 0, 10},
		peerLabel{notaryset, 2, 11},
	})
	if err != nil {
		t.Error(err)
	}
	err = checkPeer2Labels(ps, map[string][]peerLabel{
		nodeID(1).String(): []peerLabel{
			peerLabel{notaryset, 0, 10},
			peerLabel{notaryset, 0, 11},
		},
		nodeID(2).String(): []peerLabel{
			peerLabel{notaryset, 0, 10},
			peerLabel{notaryset, 2, 11},
		},
		nodeID(4).String(): []peerLabel{
			peerLabel{notaryset, 2, 11},
		},
		nodeID(5).String(): []peerLabel{
			peerLabel{notaryset, 0, 11},
		},
	})
	if err != nil {
		t.Error(err)
	}
	err = checkPeerSetHistory(ps, []uint64{10, 11}, notaryset)
	if err != nil {
		t.Error(err)
	}
	err = checkDirectPeer(server, []enode.ID{
		nodeID(1), nodeID(2), nodeID(4), nodeID(5),
	})
	if err != nil {
		t.Error(err)
	}
	err = checkGroup(server, []string{
		notarySetName(1, 10),
		notarySetName(2, 10),
		notarySetName(1, 11),
	})
	if err != nil {
		t.Error(err)
	}

	// build round 12
	ps.BuildNotaryConn(12)

	err = checkLabels(peer1, []peerLabel{
		peerLabel{notaryset, 0, 10},
		peerLabel{notaryset, 0, 11},
	})
	if err != nil {
		t.Error(err)
	}
	err = checkLabels(peer2, []peerLabel{
		peerLabel{notaryset, 0, 10},
		peerLabel{notaryset, 2, 11},
		peerLabel{notaryset, 2, 12},
	})
	if err != nil {
		t.Error(err)
	}
	err = checkPeer2Labels(ps, map[string][]peerLabel{
		nodeID(1).String(): []peerLabel{
			peerLabel{notaryset, 0, 10},
			peerLabel{notaryset, 0, 11},
		},
		nodeID(2).String(): []peerLabel{
			peerLabel{notaryset, 0, 10},
			peerLabel{notaryset, 2, 11},
			peerLabel{notaryset, 2, 12},
		},
		nodeID(3).String(): []peerLabel{
			peerLabel{notaryset, 0, 12},
		},
		nodeID(4).String(): []peerLabel{
			peerLabel{notaryset, 2, 11},
		},
		nodeID(5).String(): []peerLabel{
			peerLabel{notaryset, 0, 11},
			peerLabel{notaryset, 0, 12},
		},
		nodeID(6).String(): []peerLabel{
			peerLabel{notaryset, 2, 12},
		},
		nodeID(7).String(): []peerLabel{
			peerLabel{notaryset, 1, 12},
		},
		nodeID(8).String(): []peerLabel{
			peerLabel{notaryset, 1, 12},
		},
	})
	if err != nil {
		t.Error(err)
	}
	err = checkPeerSetHistory(ps, []uint64{10, 11, 12}, notaryset)
	if err != nil {
		t.Error(err)
	}
	err = checkDirectPeer(server, []enode.ID{
		nodeID(1), nodeID(2), nodeID(3), nodeID(4),
		nodeID(5), nodeID(6), nodeID(7), nodeID(8),
	})
	if err != nil {
		t.Error(err)
	}
	err = checkGroup(server, []string{
		notarySetName(1, 10),
		notarySetName(2, 10),
		notarySetName(1, 11),
	})
	if err != nil {
		t.Error(err)
	}

	// forget round 11
	ps.ForgetNotaryConn(11)

	err = checkLabels(peer1, []peerLabel{})
	if err != nil {
		t.Error(err)
	}
	err = checkLabels(peer2, []peerLabel{
		peerLabel{notaryset, 2, 12},
	})
	if err != nil {
		t.Error(err)
	}
	err = checkPeer2Labels(ps, map[string][]peerLabel{
		nodeID(2).String(): []peerLabel{
			peerLabel{notaryset, 2, 12},
		},
		nodeID(3).String(): []peerLabel{
			peerLabel{notaryset, 0, 12},
		},
		nodeID(5).String(): []peerLabel{
			peerLabel{notaryset, 0, 12},
		},
		nodeID(6).String(): []peerLabel{
			peerLabel{notaryset, 2, 12},
		},
		nodeID(7).String(): []peerLabel{
			peerLabel{notaryset, 1, 12},
		},
		nodeID(8).String(): []peerLabel{
			peerLabel{notaryset, 1, 12},
		},
	})
	if err != nil {
		t.Error(err)
	}
	err = checkPeerSetHistory(ps, []uint64{12}, notaryset)
	if err != nil {
		t.Error(err)
	}
	err = checkDirectPeer(server, []enode.ID{
		nodeID(2), nodeID(3),
		nodeID(5), nodeID(6), nodeID(7), nodeID(8),
	})
	if err != nil {
		t.Error(err)
	}
	err = checkGroup(server, []string{})
	if err != nil {
		t.Error(err)
	}

	// forget round 12
	ps.ForgetNotaryConn(12)
	err = checkLabels(peer1, []peerLabel{})
	if err != nil {
		t.Error(err)
	}
	err = checkLabels(peer2, []peerLabel{})
	if err != nil {
		t.Error(err)
	}
	err = checkPeer2Labels(ps, map[string][]peerLabel{})
	if err != nil {
		t.Error(err)
	}
	err = checkPeerSetHistory(ps, []uint64{}, notaryset)
	if err != nil {
		t.Error(err)
	}
	err = checkDirectPeer(server, []enode.ID{})
	if err != nil {
		t.Error(err)
	}
	err = checkGroup(server, []string{})
	if err != nil {
		t.Error(err)
	}

}

func TestPeerSetBuildDKGConn(t *testing.T) {
	self := discover.Node{ID: nodeID(0)}
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	server := newTestP2PServer(&self, key)
	table := newNodeTable()

	gov := &testGovernance{}

	gov.dkgSetFunc = func(round uint64) (map[string]struct{}, error) {
		m := map[uint64][]enode.ID{
			10: []enode.ID{nodeID(0), nodeID(1), nodeID(2)},
			11: []enode.ID{nodeID(1), nodeID(2), nodeID(5)},
			12: []enode.ID{nodeID(0), nodeID(3), nodeID(5)},
		}
		return newTestNodeSet(m[round]), nil
	}

	ps := newPeerSet(gov, server, table)
	peer1 := newDummyPeer(nodeID(1))
	peer2 := newDummyPeer(nodeID(2))
	err = ps.Register(peer1)
	if err != nil {
		t.Error(err)
	}
	err = ps.Register(peer2)
	if err != nil {
		t.Error(err)
	}

	// build round 10
	ps.BuildDKGConn(10)

	err = checkLabels(peer1, []peerLabel{
		peerLabel{dkgset, 0, 10},
	})
	if err != nil {
		t.Error(err)
	}
	err = checkLabels(peer2, []peerLabel{
		peerLabel{dkgset, 0, 10},
	})
	if err != nil {
		t.Error(err)
	}
	err = checkPeer2Labels(ps, map[string][]peerLabel{
		nodeID(1).String(): []peerLabel{
			peerLabel{dkgset, 0, 10},
		},
		nodeID(2).String(): []peerLabel{
			peerLabel{dkgset, 0, 10},
		},
	})
	if err != nil {
		t.Error(err)
	}
	err = checkPeerSetHistory(ps, []uint64{10}, dkgset)
	if err != nil {
		t.Error(err)
	}
	err = checkDirectPeer(server, []enode.ID{
		nodeID(1), nodeID(2),
	})
	if err != nil {
		t.Error(err)
	}

	// build round 11
	ps.BuildDKGConn(11)

	err = checkLabels(peer1, []peerLabel{
		peerLabel{dkgset, 0, 10},
	})
	if err != nil {
		t.Error(err)
	}
	err = checkLabels(peer2, []peerLabel{
		peerLabel{dkgset, 0, 10},
	})
	if err != nil {
		t.Error(err)
	}
	err = checkPeer2Labels(ps, map[string][]peerLabel{
		nodeID(1).String(): []peerLabel{
			peerLabel{dkgset, 0, 10},
		},
		nodeID(2).String(): []peerLabel{
			peerLabel{dkgset, 0, 10},
		},
	})
	if err != nil {
		t.Error(err)
	}
	err = checkPeerSetHistory(ps, []uint64{10}, dkgset)
	if err != nil {
		t.Error(err)
	}
	err = checkDirectPeer(server, []enode.ID{
		nodeID(1), nodeID(2),
	})
	if err != nil {
		t.Error(err)
	}

	// build round 12
	ps.BuildDKGConn(12)

	err = checkLabels(peer1, []peerLabel{
		peerLabel{dkgset, 0, 10},
	})
	if err != nil {
		t.Error(err)
	}
	err = checkLabels(peer2, []peerLabel{
		peerLabel{dkgset, 0, 10},
	})
	if err != nil {
		t.Error(err)
	}
	err = checkPeer2Labels(ps, map[string][]peerLabel{
		nodeID(1).String(): []peerLabel{
			peerLabel{dkgset, 0, 10},
		},
		nodeID(2).String(): []peerLabel{
			peerLabel{dkgset, 0, 10},
		},
		nodeID(3).String(): []peerLabel{
			peerLabel{dkgset, 0, 12},
		},
		nodeID(5).String(): []peerLabel{
			peerLabel{dkgset, 0, 12},
		},
	})
	if err != nil {
		t.Error(err)
	}
	err = checkPeerSetHistory(ps, []uint64{10, 12}, dkgset)
	if err != nil {
		t.Error(err)
	}
	err = checkDirectPeer(server, []enode.ID{
		nodeID(1), nodeID(2), nodeID(3), nodeID(5),
	})
	if err != nil {
		t.Error(err)
	}

	// forget round 11
	ps.ForgetDKGConn(11)

	err = checkLabels(peer1, []peerLabel{})
	if err != nil {
		t.Error(err)
	}
	err = checkLabels(peer2, []peerLabel{})
	if err != nil {
		t.Error(err)
	}
	err = checkPeer2Labels(ps, map[string][]peerLabel{
		nodeID(3).String(): []peerLabel{
			peerLabel{dkgset, 0, 12},
		},
		nodeID(5).String(): []peerLabel{
			peerLabel{dkgset, 0, 12},
		},
	})
	if err != nil {
		t.Error(err)
	}
	err = checkPeerSetHistory(ps, []uint64{12}, dkgset)
	if err != nil {
		t.Error(err)
	}
	err = checkDirectPeer(server, []enode.ID{
		nodeID(3), nodeID(5),
	})
	if err != nil {
		t.Error(err)
	}

	// forget round 12
	ps.ForgetDKGConn(12)
	err = checkLabels(peer1, []peerLabel{})
	if err != nil {
		t.Error(err)
	}
	err = checkLabels(peer2, []peerLabel{})
	if err != nil {
		t.Error(err)
	}
	err = checkPeer2Labels(ps, map[string][]peerLabel{})
	if err != nil {
		t.Error(err)
	}
	err = checkPeerSetHistory(ps, []uint64{}, dkgset)
	if err != nil {
		t.Error(err)
	}
	err = checkDirectPeer(server, []enode.ID{})
	if err != nil {
		t.Error(err)
	}
}

func checkLabels(p *peer, want []peerLabel) error {
	if p.labels.Cardinality() != len(want) {
		return fmt.Errorf("num of labels mismatch: got %d, want %d",
			p.labels.Cardinality(), len(want))
	}

	for _, label := range want {
		if !p.labels.Contains(label) {
			return fmt.Errorf("label %+v not exist", label)
		}
	}
	return nil
}

func checkPeer2Labels(ps *peerSet, want map[string][]peerLabel) error {
	if len(ps.peer2Labels) != len(want) {
		return fmt.Errorf("peer num mismatch: got %d, want %d",
			len(ps.peer2Labels), len(want))
	}

	for peerID, gotLabels := range ps.peer2Labels {
		wantLabels, ok := want[peerID]
		if !ok {
			return fmt.Errorf("peer id %s not exists", peerID)
		}

		if len(gotLabels) != len(wantLabels) {
			return fmt.Errorf(
				"num of labels of peer id %s mismatch: got %d, want %d",
				peerID, len(gotLabels), len(wantLabels))
		}

		for _, label := range wantLabels {
			if _, ok := gotLabels[label]; !ok {
				fmt.Errorf("label: %+v not exists", label)
			}
		}
	}
	return nil
}

func checkPeerSetHistory(ps *peerSet, want []uint64, set setType) error {
	var history map[uint64]struct{}
	switch set {
	case notaryset:
		history = ps.notaryHistory
	case dkgset:
		history = ps.dkgHistory
	default:
		return fmt.Errorf("invalid set: %d", set)
	}

	if len(history) != len(want) {
		return fmt.Errorf("num of history mismatch: got %d, want %d",
			len(history), len(want))
	}

	for _, r := range want {
		if _, ok := history[r]; !ok {
			return fmt.Errorf("round %d not exists", r)
		}
	}
	return nil
}

func checkDirectPeer(srvr *testP2PServer, want []enode.ID) error {
	if len(srvr.direct) != len(want) {
		return fmt.Errorf("num of direct peer mismatch: got %d, want %d",
			len(srvr.direct), len(want))
	}

	for _, id := range want {
		if _, ok := srvr.direct[id]; !ok {
			return fmt.Errorf("direct peer %s not exists", id.String())
		}
	}
	return nil
}
func checkGroup(srvr *testP2PServer, want []string) error {
	if len(srvr.group) != len(want) {
		return fmt.Errorf("num of group mismatch: got %d, want %d",
			len(srvr.group), len(want))
	}

	for _, name := range want {
		if _, ok := srvr.group[name]; !ok {
			return fmt.Errorf("group %s not exists", name)
		}
	}
	return nil
}

func nodeID(n int64) enode.ID {
	b := big.NewInt(n).Bytes()
	var id enode.ID
	copy(id[len(id)-len(b):], b)
	return id
}

func newTestNodeSet(nodes []enode.ID) map[string]struct{} {
	m := make(map[string]struct{})
	for _, node := range nodes {
		m[node.String()] = struct{}{}
	}
	return m
}

func newDummyPeer(id enode.ID) *peer {
	return &peer{
		labels: mapset.NewSet(),
		id:     id.String(),
	}
}
