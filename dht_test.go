package dht

import (
	"fmt"
	"github.com/hlandau/degoutils/clock"
	denet "github.com/hlandau/degoutils/net"
	"github.com/hlandau/degoutils/net/mocknet"
	"net"
	"testing"
	"time"
)

func mustResolve(a string) *net.UDPAddr {
	ua, err := net.ResolveUDPAddr("udp", a)
	if err != nil {
		panic(err)
	}
	return ua
}

func createDHT(inet *mocknet.Internet, cfg *Config) (*DHT, error) {
	cfg2 := *cfg
	cfg2.ListenFunc = func(cfg *Config) (denet.UDPConn, error) {
		return inet.ListenUDP("udp", mustResolve(cfg.Address))
	}
	cfg2.Clock = clock.Real

	return New(&cfg2)
}

// Makes n DHTs.
func makeDHTs(inet *mocknet.Internet, n int) (dhts []*DHT, addrs []string, err error) {
	for i := 0; i < n; i++ {
		a := fmt.Sprintf("1.2.3.%d:5555", i+1)
		d, err := createDHT(inet, &Config{
			Address: a,
		})
		if err != nil {
			return nil, nil, err
		}

		dhts = append(dhts, d)
		addrs = append(addrs, a)
	}

	return
}

func stopDHTs(dhts []*DHT) {
	for _, d := range dhts {
		d.Stop()
	}
}

func TestDHT(t *testing.T) {
	inet := mocknet.NewInternet(nil)

	// Construct DHTs.
	dhts, addrs, err := makeDHTs(inet, 10)
	if err != nil {
		t.Fatal()
	}
	defer stopDHTs(dhts)

	// Connect the DHTs in a chain a->b->c->d->e to test propagation.
	for i := 0; i < len(dhts)-1; i++ {
		dhts[i].AddNode(NodeLocator{
			Addr: *mustResolve(addrs[i+1]),
		})
	}

	// Announce the infohash at the last DHT.
	var ih1 = MustParseInfoHash("e2231dfe1d791ebfe619ec7f87ae1ca103b84239")
	announcer, err := NewSearch(dhts[len(dhts)-1], ih1, true)
	if err != nil {
		t.Fatal()
	}
	defer announcer.Stop()

	// Try and find it at the first.
	searcher, err := NewSearch(dhts[0], ih1, false)
	if err != nil {
		t.Fatal()
	}
	defer searcher.Stop()

	select {
	case p := <-dhts[0].PeersChan():
		t.Logf("peer %v", p)
	case <-time.After(10 * time.Second):
		t.Fatalf("no result after 10s")
	}
}
