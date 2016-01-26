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

func TestSecurityKAT(t *testing.T) {
	var tests = []struct {
		IP     net.IP
		NodeID NodeID
	}{
		{
			IP:     net.ParseIP("124.31.75.21"),
			NodeID: MustParseNodeID("5fbfbff10c5d6a4ec8a88e4c6ab4c28b95eee401"),
		},
		{
			IP:     net.ParseIP("21.75.31.124"),
			NodeID: MustParseNodeID("5a3ce9c14e7a08645677bbd1cfe7d8f956d53256"),
		},
		{
			IP:     net.ParseIP("65.23.51.170"),
			NodeID: MustParseNodeID("a5d43220bc8f112a3d426c84764f8c2a1150e616"),
		},
		{
			IP:     net.ParseIP("84.124.73.14"),
			NodeID: MustParseNodeID("1b0321dd1bb1fe518101ceef99462b947a01ff41"),
		},
		{
			IP:     net.ParseIP("43.213.53.83"),
			NodeID: MustParseNodeID("e56f6cbf5b7c4be0237986d5243b87aa6d51305a"),
		},
	}

	for _, tst := range tests {
		if !nodeIDIsAllowed(tst.IP, tst.NodeID) {
			t.Fatalf("node ID should be allowed")
		}

		tst.IP[2] ^= 1
		if nodeIDIsAllowed(tst.IP, tst.NodeID) {
			t.Fatalf("node ID should not be allowed")
		}
	}
}
