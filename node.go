package dht

import (
	"github.com/hlandau/dht/krpc"
	"net"
	"time"
)

// Represents a known node in the DHT.
type node struct {
	Addr   net.UDPAddr // The address of the peer.
	NodeID NodeID      // May be invalid if not yet known.

	// Outgoing queries for which we are awaiting a response.
	PendingQueries map[string]*krpc.Message

	// Time of last incoming message from this peer.
	LastRxTime time.Time

	// Used to determine which infohashes have already been requested from this node.
	PastQueries map[InfoHash]time.Time
}

func newNode(addr net.UDPAddr, nodeID NodeID) *node {
	return &node{
		Addr:           addr,
		NodeID:         nodeID,
		PendingQueries: make(map[string]*krpc.Message),
		PastQueries:    make(map[InfoHash]time.Time),
	}
}

func (p *node) IsReachable() bool {
	return !p.LastRxTime.IsZero()
}

func (p *node) NumPendingQueries() int {
	return len(p.PendingQueries)
}

// Returns true iff the node is due for expiry because of unanswered queries or
// because it has not been heard from.
func (n *node) IsExpired(cleanupPeriod time.Duration) bool {
	if !n.IsReachable() && n.NumPendingQueries() > 2 {
		return true
	}

	timeSince := time.Since(n.LastRxTime)
	if timeSince > (2*cleanupPeriod + 1*time.Minute) {
		return true
	}

	return false
}

// Returns true iff the node is due for ping. It is assumed this function will
// only be called after checking that IsExpired() is false.
func (n *node) NeedsPing(cleanupPeriod time.Duration) bool {
	if !n.IsReachable() || n.NumPendingQueries() == 0 {
		return true
	}

	timeSince := time.Since(n.LastRxTime)
	return timeSince >= cleanupPeriod/2
}

// Returns true if a node was contacted recently in relation to some infohash.
func (n *node) WasContactedRecently(infoHash InfoHash, searchRetryPeriod time.Duration) bool {
	t, ok := n.PastQueries[infoHash]
	return ok && time.Since(t) > searchRetryPeriod
}

func (n *node) MarkContacted(infoHash InfoHash) {
	n.PastQueries[infoHash] = time.Now()
}
