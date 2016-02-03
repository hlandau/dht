package dht

import (
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestNodeDistance(t *testing.T) {
	nodes := make([]*node, 10)
	for i := 0; i < len(nodes); i++ {
		id := make([]byte, 20)
		id[19] = byte(i)
		nodes[i] = &node{
			NodeID: NodeID(id),
		}
	}

	tree := &routingTree{}
	for _, r := range nodes {
		r.LastRxTime = time.Now()
		tree.Insert(r)
	}

	var tests = []struct {
		Query InfoHash
		Want  int
	}{
		{MustParseInfoHash("0400000000000000000000000000000000000000"), 8},
		{MustParseInfoHash("0700000000000000000000000000000000000000"), 8},
	}

	for _, tst := range tests {
		distances := make([]string, 0, len(tests))
		neighbours := tree.Lookup(tst.Query)
		if len(neighbours) != tst.Want {
			t.Fatal()
		}

		for _, n := range neighbours {
			dist := hashDistance(tst.Query, InfoHash(n.NodeID))
			var b []string
			for _, c := range dist {
				if c != 0 {
					b = append(b, fmt.Sprintf("%08b", c))
				} else {
					b = append(b, "00000000")
				}
			}
			dist = strings.Join(b, ".")
			distances = append(distances, dist)
		}

		if !sort.StringsAreSorted(distances) {
			t.Fatal()
		}
	}
}
