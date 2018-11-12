package dex

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/dexon-foundation/dexon/crypto"
	"github.com/dexon-foundation/dexon/p2p/enode"
)

func TestPeerSetBuildAndForgetNotaryConn(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	server := newTestP2PServer(key)
	self := server.Self()
	table := newNodeTable()

	gov := &testGovernance{
		numChainsFunc: func(uint64) uint32 {
			return 3
		},
	}

	var nodes []*enode.Node
	for i := 0; i < 9; i++ {
		nodes = append(nodes, randomEnode())
	}

	round10 := [][]*enode.Node{
		[]*enode.Node{self, nodes[1], nodes[2]},
		[]*enode.Node{nodes[1], nodes[3]},
		[]*enode.Node{nodes[2], nodes[4]},
	}
	round11 := [][]*enode.Node{
		[]*enode.Node{self, nodes[1], nodes[5]},
		[]*enode.Node{nodes[5], nodes[6]},
		[]*enode.Node{self, nodes[2], nodes[4]},
	}
	round12 := [][]*enode.Node{
		[]*enode.Node{self, nodes[3], nodes[5]},
		[]*enode.Node{self, nodes[7], nodes[8]},
		[]*enode.Node{self, nodes[2], nodes[6]},
	}

	gov.notarySetFunc = func(
		round uint64, cid uint32) (map[string]struct{}, error) {
		m := map[uint64][][]*enode.Node{
			10: round10,
			11: round11,
			12: round12,
		}
		return newTestNodeSet(m[round][cid]), nil
	}

	ps := newPeerSet(gov, server, table)
	peer1 := newDummyPeer(nodes[1])
	peer2 := newDummyPeer(nodes[2])
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

	err = checkPeer2Labels(ps, map[string][]peerLabel{
		nodes[1].ID().String(): []peerLabel{
			peerLabel{notaryset, 0, 10},
		},
		nodes[2].ID().String(): []peerLabel{
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
		nodes[1].ID(), nodes[2].ID(),
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

	err = checkPeer2Labels(ps, map[string][]peerLabel{
		nodes[1].ID().String(): []peerLabel{
			peerLabel{notaryset, 0, 10},
			peerLabel{notaryset, 0, 11},
		},
		nodes[2].ID().String(): []peerLabel{
			peerLabel{notaryset, 0, 10},
			peerLabel{notaryset, 2, 11},
		},
		nodes[4].ID().String(): []peerLabel{
			peerLabel{notaryset, 2, 11},
		},
		nodes[5].ID().String(): []peerLabel{
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
		nodes[1].ID(), nodes[2].ID(), nodes[4].ID(), nodes[5].ID(),
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

	err = checkPeer2Labels(ps, map[string][]peerLabel{
		nodes[1].ID().String(): []peerLabel{
			peerLabel{notaryset, 0, 10},
			peerLabel{notaryset, 0, 11},
		},
		nodes[2].ID().String(): []peerLabel{
			peerLabel{notaryset, 0, 10},
			peerLabel{notaryset, 2, 11},
			peerLabel{notaryset, 2, 12},
		},
		nodes[3].ID().String(): []peerLabel{
			peerLabel{notaryset, 0, 12},
		},
		nodes[4].ID().String(): []peerLabel{
			peerLabel{notaryset, 2, 11},
		},
		nodes[5].ID().String(): []peerLabel{
			peerLabel{notaryset, 0, 11},
			peerLabel{notaryset, 0, 12},
		},
		nodes[6].ID().String(): []peerLabel{
			peerLabel{notaryset, 2, 12},
		},
		nodes[7].ID().String(): []peerLabel{
			peerLabel{notaryset, 1, 12},
		},
		nodes[8].ID().String(): []peerLabel{
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
		nodes[1].ID(), nodes[2].ID(), nodes[3].ID(), nodes[4].ID(),
		nodes[5].ID(), nodes[6].ID(), nodes[7].ID(), nodes[8].ID(),
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

	err = checkPeer2Labels(ps, map[string][]peerLabel{
		nodes[2].ID().String(): []peerLabel{
			peerLabel{notaryset, 2, 12},
		},
		nodes[3].ID().String(): []peerLabel{
			peerLabel{notaryset, 0, 12},
		},
		nodes[5].ID().String(): []peerLabel{
			peerLabel{notaryset, 0, 12},
		},
		nodes[6].ID().String(): []peerLabel{
			peerLabel{notaryset, 2, 12},
		},
		nodes[7].ID().String(): []peerLabel{
			peerLabel{notaryset, 1, 12},
		},
		nodes[8].ID().String(): []peerLabel{
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
		nodes[2].ID(), nodes[3].ID(),
		nodes[5].ID(), nodes[6].ID(), nodes[7].ID(), nodes[8].ID(),
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
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	server := newTestP2PServer(key)
	self := server.Self()
	table := newNodeTable()

	var nodes []*enode.Node
	for i := 0; i < 6; i++ {
		nodes = append(nodes, randomEnode())
	}

	gov := &testGovernance{}

	gov.dkgSetFunc = func(round uint64) (map[string]struct{}, error) {
		m := map[uint64][]*enode.Node{
			10: []*enode.Node{self, nodes[1], nodes[2]},
			11: []*enode.Node{nodes[1], nodes[2], nodes[5]},
			12: []*enode.Node{self, nodes[3], nodes[5]},
		}
		return newTestNodeSet(m[round]), nil
	}

	ps := newPeerSet(gov, server, table)
	peer1 := newDummyPeer(nodes[1])
	peer2 := newDummyPeer(nodes[2])
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

	err = checkPeer2Labels(ps, map[string][]peerLabel{
		nodes[1].ID().String(): []peerLabel{
			peerLabel{dkgset, 0, 10},
		},
		nodes[2].ID().String(): []peerLabel{
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
		nodes[1].ID(), nodes[2].ID(),
	})
	if err != nil {
		t.Error(err)
	}

	// build round 11
	ps.BuildDKGConn(11)

	err = checkPeer2Labels(ps, map[string][]peerLabel{
		nodes[1].ID().String(): []peerLabel{
			peerLabel{dkgset, 0, 10},
		},
		nodes[2].ID().String(): []peerLabel{
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
		nodes[1].ID(), nodes[2].ID(),
	})
	if err != nil {
		t.Error(err)
	}

	// build round 12
	ps.BuildDKGConn(12)

	err = checkPeer2Labels(ps, map[string][]peerLabel{
		nodes[1].ID().String(): []peerLabel{
			peerLabel{dkgset, 0, 10},
		},
		nodes[2].ID().String(): []peerLabel{
			peerLabel{dkgset, 0, 10},
		},
		nodes[3].ID().String(): []peerLabel{
			peerLabel{dkgset, 0, 12},
		},
		nodes[5].ID().String(): []peerLabel{
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
		nodes[1].ID(), nodes[2].ID(), nodes[3].ID(), nodes[5].ID(),
	})
	if err != nil {
		t.Error(err)
	}

	// forget round 11
	ps.ForgetDKGConn(11)

	err = checkPeer2Labels(ps, map[string][]peerLabel{
		nodes[3].ID().String(): []peerLabel{
			peerLabel{dkgset, 0, 12},
		},
		nodes[5].ID().String(): []peerLabel{
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
		nodes[3].ID(), nodes[5].ID(),
	})
	if err != nil {
		t.Error(err)
	}

	// forget round 12
	ps.ForgetDKGConn(12)
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

func newTestNodeSet(nodes []*enode.Node) map[string]struct{} {
	m := make(map[string]struct{})
	for _, node := range nodes {
		b := crypto.FromECDSAPub(node.Pubkey())
		m[hex.EncodeToString(b)] = struct{}{}
	}
	return m
}

func newDummyPeer(node *enode.Node) *peer {
	return &peer{id: node.ID().String()}
}
