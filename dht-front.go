package dht

// This file contains the public methods which send commands to the control
// goroutine.

import (
	"net"
	"sync/atomic"
)

// Information about a given node.
type NodeInfo struct {
	NodeLocator
}

// Represents a peer address identified for a given infohash.
type PeerResult struct {
	// The infohash to which the result pertains.
	InfoHash InfoHash

	// The IP and port for the peer. (Note that this may not match the port used
	// by the advertising DHT node, as a different port can be nominated.)
	Addr net.UDPAddr
}

type addNodeInfo struct {
	NodeLocator
	ForceAdd bool
}

type requestPeersInfo struct {
	InfoHash InfoHash
	Announce bool
}

// Peer search results will be returned on this channel. It is closed when the
// node is stopped.
func (dht *DHT) PeersChan() <-chan PeerResult {
	return dht.peersChan
}

// Stop the DHT instance. Do not call any other method after calling this.
func (dht *DHT) Stop() error {
	dht.stopOnce.Do(func() {
		atomic.StoreUint32(&dht.stopping, 1)
		close(dht.stopChan)
	})
	return nil
}

// Add a node with the given hostname and unknown NodeID.
func AddHost(dht *DHT, hostname string, nodeID NodeID) error {
	addr, err := net.ResolveUDPAddr("udp", hostname)
	if err != nil {
		return err
	}

	dht.AddNode(NodeLocator{
		Addr:   *addr,
		NodeID: nodeID,
	})
	return nil
}

// Soft-adds a node to the DHT. You must call this with at least one node to
// bootstrap the DHT. Address is in format "IP:port". The node ID is optional.
func (dht *DHT) AddNode(nodeLocator NodeLocator) {
	dht.addNode(nodeLocator, false)
}

// Hard-adds a node to the DHT. Adds the node even if there are already enough
// nodes added.
func (dht *DHT) ForceAddNode(nodeLocator NodeLocator) {
	dht.addNode(nodeLocator, true)
}

// Get the l-goroutine to handle the node addition.
func (dht *DHT) addNode(nodeLocator NodeLocator, forceAdd bool) {
	dht.addNodeChan <- addNodeInfo{nodeLocator, forceAdd}
}

// Request peers for an infohash. If announce is set to true, this node will be
// signed up as a peer.
func (dht *DHT) RequestPeers(infoHash InfoHash, announce bool) error {
	dht.requestPeersChan <- requestPeersInfo{
		InfoHash: infoHash,
		Announce: announce,
	}
	return nil
}

// TODO
func (dht *DHT) RequestDatum(infoHash InfoHash) error {
	return nil
}

// TODO
func (dht *DHT) PutDatum(datum *Datum) error {
	return nil
}

// Returns information on all known reachable nodes. Useful for saving the node
// database to persistent storage.
func (dht *DHT) ListReachableNodes() []NodeInfo {
	ch := make(chan []NodeInfo, 1)
	dht.requestReachableNodesChan <- ch
	return <-ch
}

// Return the node ID.
func (dht *DHT) NodeID() NodeID {
	return dht.cfg.NodeID
}
