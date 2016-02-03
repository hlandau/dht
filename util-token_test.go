package dht

import (
	"net"
	"testing"
)

var (
	testAddr1 = net.UDPAddr{
		IP:   net.ParseIP("192.0.2.1"),
		Port: 1234,
	}
	testAddr2 = net.UDPAddr{
		IP:   net.ParseIP("192.0.2.2"),
		Port: 1234,
	}
)

func TestTokenStore(t *testing.T) {
	ts := newTokenStore()
	b := ts.Generate(testAddr1)
	var b2 []byte
	for i := 0; i < 2; i++ {
		if !ts.Verify(b, testAddr1) {
			t.Fatal()
		}

		b[4] ^= 1
		if ts.Verify(b, testAddr1) {
			t.Fatal()
		}

		b[4] ^= 1
		if ts.Verify(b, testAddr2) {
			t.Fatal()
		}

		ts.Cycle()
		if i == 0 {
			b2 = ts.Generate(testAddr2)
		}
	}

	if ts.Verify(b, testAddr1) {
		t.Fatal()
	}
	if !ts.Verify(b2, testAddr2) {
		t.Fatal()
	}
}
