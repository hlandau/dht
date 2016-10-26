package krpc

import (
	"fmt"
	"github.com/hlandauf/bencode"
	"net"
	"testing"
)

func TestEndpoint(t *testing.T) {
	endpoints := []Endpoint{
		Endpoint{IP: net.ParseIP("1.2.3.4"), Port: 2345},
		Endpoint{IP: net.ParseIP("5.6.7.8"), Port: 8877},
	}

	b, err := bencode.EncodeBytes(endpoints)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if fmt.Sprintf("%x", b) != "6c363a010203040929363a0506070822ad65" {
		t.Fatalf("mismatch: %x", b)
	}
}
