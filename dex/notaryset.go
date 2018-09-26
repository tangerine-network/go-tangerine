package dex

import (
	"fmt"
	"sync"

	"github.com/dexon-foundation/dexon/p2p/discover"
)

type nodeInfo struct {
	info  *notaryNodeInfo
	added bool
}

func (n *nodeInfo) NewNode() *discover.Node {
	return discover.NewNode(n.info.ID, n.info.IP, n.info.UDP, n.info.TCP)
}

type notarySet struct {
	round uint64
	m     map[string]*nodeInfo
	lock  sync.RWMutex
}

func newNotarySet(round uint64, s map[string]struct{}) *notarySet {
	m := make(map[string]*nodeInfo)
	for nodeID := range s {
		m[nodeID] = &nodeInfo{}
	}

	return &notarySet{
		round: round,
		m:     m,
	}
}

// Call this function when the notaryNodeInfoMsg is received.
func (n *notarySet) AddInfo(info *notaryNodeInfo) error {
	n.lock.Lock()
	defer n.lock.Unlock()

	// check round
	if info.Round != n.round {
		return fmt.Errorf("invalid round")
	}

	nInfo, ok := n.m[info.ID.String()]
	if !ok {
		return fmt.Errorf("not in notary set")
	}

	// if the info exists check timstamp
	if nInfo.info != nil {
		if nInfo.info.Timestamp > info.Timestamp {
			return fmt.Errorf("old msg")
		}
	}

	nInfo.info = info
	return nil
}

// MarkAdded mark the notary node as added
// to prevent duplcate addition in the future.
func (n *notarySet) MarkAdded(nodeID string) {
	if info, ok := n.m[nodeID]; ok {
		info.added = true
	}
}

// Return all nodes
func (n *notarySet) Nodes() []*discover.Node {
	n.lock.RLock()
	defer n.lock.RUnlock()

	list := make([]*discover.Node, 0, len(n.m))
	for _, info := range n.m {
		list = append(list, info.NewNode())
	}
	return list
}

// Return nodes that need to be added to p2p server as notary node.
func (n *notarySet) NodesToAdd() []*discover.Node {
	n.lock.RLock()
	defer n.lock.RUnlock()

	var list []*discover.Node
	for _, info := range n.m {
		// craete a new discover.Node
		if !info.added {
			list = append(list, info.NewNode())
		}
	}
	return list
}

type notarySetManager struct {
	m                   map[uint64]*notarySet
	lock                sync.RWMutex
	queued              map[uint64]map[string]*notaryNodeInfo
	round               uint64 // biggest round of managed notary sets
	newNotaryNodeInfoCh chan *notaryNodeInfo
}

func newNotarySetManager(
	newNotaryNodeInfoCh chan *notaryNodeInfo) *notarySetManager {
	return &notarySetManager{
		m:                   make(map[uint64]*notarySet),
		queued:              make(map[uint64]map[string]*notaryNodeInfo),
		newNotaryNodeInfoCh: newNotaryNodeInfoCh,
	}
}

// Register injects a new notary set into the manager and
// processes the queued info.
func (n *notarySetManager) Register(r uint64, s *notarySet) {
	n.lock.Lock()
	defer n.lock.Unlock()
	if r > n.round {
		n.round = r
	}
	n.m[r] = s
	n.processQueuedInfo()
}

// Unregister removes the notary set of the given round.
func (n *notarySetManager) Unregister(r uint64) {
	n.lock.Lock()
	defer n.lock.Unlock()
	delete(n.m, r)
}

// Round returns the notary set of the given round.
func (n *notarySetManager) Round(r uint64) (*notarySet, bool) {
	n.lock.RLock()
	defer n.lock.RUnlock()
	s, ok := n.m[r]
	return s, ok
}

// Before returns all the notary sets that before the given round.
func (n *notarySetManager) Before(r uint64) []*notarySet {
	n.lock.RLock()
	defer n.lock.RUnlock()
	var list []*notarySet
	for round, s := range n.m {
		if round < r {
			list = append(list, s)
		}
	}
	return list
}

// TryAddInfo associates the given info to the notary set if the notary set is
// managed by the manager.
// If the notary node info is belong to future notary set, queue the info.
func (n *notarySetManager) TryAddInfo(info *notaryNodeInfo) {
	n.lock.Lock()
	defer n.lock.Unlock()
	n.tryAddInfo(info)
}

// This function is extract for calling without lock.
// Make sure the caller already accquired the lock.
func (n *notarySetManager) tryAddInfo(info *notaryNodeInfo) {
	if info.Round > n.round {
		if q, ok := n.queued[info.Round]; ok {
			q[info.Hash().String()] = info
			return
		}
		n.queued[info.Round] = map[string]*notaryNodeInfo{
			info.Hash().String(): info,
		}
		return
	}

	s, ok := n.Round(info.Round)
	if !ok {
		return
	}
	s.AddInfo(info)

	// TODO(sonic): handle timeout
	n.newNotaryNodeInfoCh <- info
}

func (n *notarySetManager) processQueuedInfo() {
	n.lock.Lock()
	defer n.lock.Unlock()
	if q, ok := n.queued[n.round]; ok {
		for _, info := range q {
			n.tryAddInfo(info)
		}
	}

	// Clear queue infos before current round.
	for round := range n.queued {
		if round <= n.round {
			delete(n.queued, round)
		}
	}
}
