package dht

import (
	"github.com/hlandau/degoutils/clock"
	"github.com/hlandau/dht/krpc"
	"net"
	"time"
)

// Given a list of nodes, a wants string, and the address the request was
// received from, return a list of IPv4 and IPv6 node locators. The lists may
// be empty if that address family was not requested.
func formNodeList(nodes []*node, ws []string, addr net.UDPAddr) (nodes4 krNodesIPv4, nodes6 krNodesIPv6) {
	v4, v6 := wants(ws, addr)

	for _, n := range nodes {
		nl := NodeLocator{
			NodeID: n.NodeID,
			Addr:   n.Addr,
		}

		is4 := nl.Addr.IP.To4() != nil
		if is4 && v4 {
			nodes4 = append(nodes4, nl)
		} else if !is4 && v6 {
			nodes6 = append(nodes6, nl)
		}
	}

	return
}

// Given a list of endpoints, a wants string, and the address the request was
// received from, return a list of IPv4 and IPv6 peer addresses.
func formPeerList(peerAddrs []net.UDPAddr, ws []string, addr net.UDPAddr) []krpc.Endpoint {
	var endpoints []krpc.Endpoint
	v4, v6 := wants(ws, addr)

	for i := range peerAddrs {
		is4 := peerAddrs[i].IP.To4() != nil
		if (is4 && v4) || (!is4 && v6) {
			endpoints = append(endpoints, krpc.Endpoint(peerAddrs[i]))
		}
	}

	return endpoints
}

// Determine which address families a node is requesting or is assumed to
// support.
func wants(w []string, addr net.UDPAddr) (v4, v6 bool) {
	v4 = hasItem(w, "n4")
	v6 = hasItem(w, "n6")
	if !v4 && !v6 {
		v4 = (addr.IP.To4() != nil)
		v6 = !v4
	}

	return
}

// Returns true if the list of strings contains the given item.
func hasItem(w []string, v string) bool {
	if len(w) > 10 {
		return false
	}

	for _, x := range w {
		if x == v {
			return true
		}
	}

	return false
}

func timerAt(c clock.Clock, t time.Time) <-chan time.Time {
	return c.After(t.Sub(c.Now()))
}

// Set whether an infohash is one we announce for.
func (dht *DHT) lSetLocallyOriginated(infoHash InfoHash, announce bool) {
	if announce {
		dht.locallyOriginated[infoHash] = struct{}{}
	} else {
		delete(dht.locallyOriginated, infoHash)
	}
}

func (dht *DHT) lSetLocallyInterested(infoHash InfoHash, interested bool) {
	if interested {
		dht.locallyInterested[infoHash] = struct{}{}
	} else {
		delete(dht.locallyInterested, infoHash)
	}
}

func (dht *DHT) needMoreNodes() bool {
	numNodes := dht.neighbourhood.routingTable.Size()
	return numNodes < dht.cfg.MinNodes || numNodes*2 < dht.cfg.MaxNodes
}

func (dht *DHT) acceptMoreNodes() bool {
	numNodes := dht.neighbourhood.routingTable.Size()
	return numNodes < dht.cfg.MaxNodes
}

func (dht *DHT) needMorePeers(infoHash InfoHash) bool {
	return dht.peerStore.Count(infoHash) < dht.cfg.NumTargetPeers
}

func (dht *DHT) peersFor(infoHash InfoHash) []net.UDPAddr {
	return dht.peerStore.Values(infoHash)
}
