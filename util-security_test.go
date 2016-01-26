package dht

import (
	"crypto/rand"
	"net"
	"testing"
)

func TestSecurity(t *testing.T) {
	var ips = []string{"127.0.0.1", "dead:beef::deca:fbad"}

	for _, ipstr := range ips {
		var seed [20]byte
		rand.Read(seed[:])

		nodeID := NodeID(string(seed[:]))
		ip := net.ParseIP(ipstr)

		if nodeIDIsAllowed(ip, nodeID) {
			t.Fatalf("allowed before conforming")
		}

		cNodeID := conformNodeID(ip, nodeID)
		if !nodeIDIsAllowed(ip, cNodeID) {
			t.Fatalf("supposed to be allowed after conforming")
		}
	}
}
