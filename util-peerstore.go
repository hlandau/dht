package dht

import (
	"container/ring"
	"github.com/golang/groupcache/lru"
	"net"
)

// Stores values. Specific to an infohash.
type peerSet struct {
	// The set of values.
	values map[string]struct{}

	// Immutable/mutable datum, if present.
	datum *Datum

	// Needed to ensure different peers are returned each time.
	ring *ring.Ring
}

func newValueSet() *peerSet {
	return &peerSet{
		values: map[string]struct{}{},
	}
}

// Returns up to eight contacts, if available. Further calls will return a
// different set of contacts, if possible.
func (ps *peerSet) Next() []net.UDPAddr {
	count := 8
	if count > len(ps.values) {
		count = len(ps.values)
	}

	xs := make([]net.UDPAddr, 0, count+1)
	var next *ring.Ring
	for i := 0; i < count; i++ {
		next = ps.ring.Next()
		ps.ring = next

		xs = append(xs, next.Value.(net.UDPAddr))
	}

	return xs
}

func (ps *peerSet) Datum() *Datum {
	return ps.datum
}

func (ps *peerSet) PutDatum(datum *Datum) bool {
	ps.datum = datum
	return true
}

// Add an address to the value set. Returns true if the address was not
// already in the value set.
func (ps *peerSet) Put(addr net.UDPAddr) bool {
	s := addr.String()

	if _, ok := ps.values[s]; ok {
		return false
	}

	ps.values[s] = struct{}{}

	r := &ring.Ring{
		Value: addr,
	}

	if ps.ring == nil {
		ps.ring = r
	} else {
		ps.ring.Link(r)
	}

	return true
}

func (ps *peerSet) Size() int {
	return len(ps.values)
}

// Stores values for infohashes.
type peerStore struct {
	// Caches values for infohashes. Each key is an infohash and each value is a
	// valueSet.
	values *lru.Cache

	maxInfoHashes    int
	maxInfoHashPeers int
}

func newPeerStore(maxInfoHashes, maxInfoHashPeers int) *peerStore {
	return &peerStore{
		values:           lru.New(maxInfoHashes),
		maxInfoHashes:    maxInfoHashes,
		maxInfoHashPeers: maxInfoHashPeers,
	}
}

func (ps *peerStore) Set(infoHash InfoHash) *peerSet {
	set, ok := ps.values.Get(string(infoHash))
	if !ok {
		return nil
	}

	return set.(*peerSet)
}

// Get number of known peer values for the given infohash.
func (ps *peerStore) Count(infoHash InfoHash) int {
	set := ps.Set(infoHash)
	if set == nil {
		return 0
	}

	return set.Size()
}

// Returns a random set of approximately eight values for the given infohash.
func (ps *peerStore) Values(infoHash InfoHash) []net.UDPAddr {
	set := ps.Set(infoHash)
	if set == nil {
		return nil
	}

	return set.Next()
}

func (ps *peerStore) Datum(infoHash InfoHash) *Datum {
	set := ps.Set(infoHash)
	if set == nil {
		return nil
	}

	return set.Datum()
}

// Add the given address as a value for the provided infohash.
// Returns true if the address was added.
func (ps *peerStore) Add(infoHash InfoHash, addr net.UDPAddr) bool {
	set := ps.Set(infoHash)
	if set == nil {
		set = newValueSet()
	}

	if set.Size() >= ps.maxInfoHashPeers {
		// Already have too many values.
		// TODO: use a circular buffer and discard other contacts.
		return false
	}

	// Add/touch set in LRU cache and add address to set.
	ps.values.Add(string(infoHash), set)
	return set.Put(addr)
}

// Add the given datum as a value for the provided infohash.
// Returns true if the datum was added.
func (ps *peerStore) AddDatum(infoHash InfoHash, datum *Datum) bool {
	set := ps.Set(infoHash)
	if set == nil {
		set = newValueSet()
	}

	return set.PutDatum(datum)
}
