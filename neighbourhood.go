package dht

import "time"

type neighbourhood struct {
	routingTable *routingTable

	nodeID       NodeID
	boundaryNode *node
	proximity    int // How many prefix bits are shared between boundaryNode and nodeID.
}

func newNeighbourhood(nodeID NodeID) *neighbourhood {
	return &neighbourhood{
		routingTable: newRoutingTable(),
		nodeID:       nodeID,
	}
}

// Remove peer from routing table.
func (nh *neighbourhood) Remove(n *node) {
	nh.routingTable.Remove(n)

	if n.NodeID == nh.boundaryNode.NodeID {
		nh.resetBoundary()
	}
}

func (nh *neighbourhood) resetBoundary() {
	nh.proximity = 0
	neighbours := nh.routingTable.routingTree.Lookup(InfoHash(nh.nodeID))
	if len(neighbours) > 0 {
		nh.boundaryNode = neighbours[len(neighbours)-1]
		nh.proximity = commonBits([]byte(nh.nodeID), []byte(nh.boundaryNode.NodeID))
	}
}

// Update the routing table if the peer p is closer than the eight nodes in our
// neighbourhood, by replacing the most distant one (boundaryNode).
func (nh *neighbourhood) Upkeep(n *node) {
	if nh.boundaryNode == nil {
		nh.addNewNeighbour(n, false)
		return
	}

	if nh.routingTable.Size() < kNodes {
		nh.addNewNeighbour(n, false)
		return
	}

	cmp := commonBits([]byte(nh.nodeID), []byte(n.NodeID))
	if cmp == 0 {
		// Not significantly better.
		return
	}

	if cmp > nh.proximity {
		nh.addNewNeighbour(n, true)
		return
	}
}

func (nh *neighbourhood) addNewNeighbour(n *node, displaceBoundary bool) {
	if displaceBoundary && nh.boundaryNode != nil {
		nh.Remove(nh.boundaryNode)
	} else {
		nh.resetBoundary()
	}
}

func (nh *neighbourhood) Cleanup(period time.Duration) (nodesToBePinged []*node) {
	nh.routingTable.Visit(func(n *node) error {
		if n.IsExpired(period) {
			nh.Remove(n)
		} else if n.NeedsPing(period) {
			nodesToBePinged = append(nodesToBePinged, n)
		}

		return nil
	})

	return
}

func (nh *neighbourhood) ReachableNodes(peerChan chan<- *node) {
	nh.routingTable.Visit(func(n *node) error {
		if n.IsReachable() && n.NodeID.Valid() {
			peerChan <- n
		}

		return nil
	})
}
