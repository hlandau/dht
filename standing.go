package dht

import (
	"sync"
	"time"
)

// Represents a standing search request.
type Search interface {
	// Call to stop the search request. Results may still be returned.  May be
	// called multiple times without consequence.
	Stop()
}

type search struct {
	dht      *DHT
	stopChan chan struct{}
	stopOnce sync.Once

	infoHash InfoHash
	announce bool
}

func (s *search) loop() {
	for {
		s.dht.RequestPeers(s.infoHash, s.announce)

		select {
		case <-time.After(20 * time.Second):
		case <-s.stopChan:
			return
		}
	}
}

func (s *search) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopChan)
	})
}

// Creates a new standing peer request. The request will be repeated as
// appropriate until the desired number of peers has been found. Cancel the
// request by calling Stop on the returned interface.
func NewSearch(dht *DHT, infoHash InfoHash, announce bool) (Search, error) {
	s := &search{
		dht:      dht,
		stopChan: make(chan struct{}),

		infoHash: infoHash,
		announce: announce,
	}
	go s.loop()
	return s, nil
}
