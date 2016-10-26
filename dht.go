// Package dht implements a BitTorrent Mainline DHT node.
//
// Implements BEP-0005 and BEP-0032.
package dht

import (
	denet "github.com/hlandau/degoutils/net"
	"github.com/hlandau/goutils/clock"
	"github.com/hlandau/xlog"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

var log, Log = xlog.New("dht")

// DHT structure, setup and teardown. {{{1

type DHT struct {
	cfg      Config
	wantList []string

	// Requests from the client.
	addNodeChan               chan addNodeInfo
	requestPeersChan          chan requestPeersInfo
	requestReachableNodesChan chan chan<- []NodeInfo

	// Channels to return information to the client.
	peersChan chan PeerResult

	// Network traffic channels.
	rxChan              chan packet
	addrUnreachableChan chan net.UDPAddr

	// Internally generated requests.
	recurseNodeChan chan NodeID
	requestPingChan chan *node

	// Stopping.
	stopChan chan struct{}
	stopOnce sync.Once
	stopping uint32

	// UDP TX/RX socket.
	conn denet.UDPConn

	// State.
	neighbourhood     *neighbourhood
	peerStore         *peerStore
	tokenStore        *tokenStore
	locallyOriginated map[InfoHash]struct{}
	locallyInterested map[InfoHash]struct{}
}

// Create a new DHT node and start it.
func New(cfg *Config) (*DHT, error) {
	cfg.setDefaults()

	dht := &DHT{
		cfg: *cfg,

		// Requests from the client.
		stopChan:                  make(chan struct{}),
		addNodeChan:               make(chan addNodeInfo, 10),
		requestPeersChan:          make(chan requestPeersInfo, 10),
		requestReachableNodesChan: make(chan chan<- []NodeInfo, 10),

		// Channels to return information to the client.
		peersChan: make(chan PeerResult, 10),

		// Network traffic channels.
		rxChan:              make(chan packet, 10),
		addrUnreachableChan: make(chan net.UDPAddr, 10),

		// Internally generated requests.
		recurseNodeChan: make(chan NodeID, 10),
		requestPingChan: make(chan *node, 10),

		// State.
		neighbourhood:     newNeighbourhood(cfg.NodeID),
		peerStore:         newPeerStore(cfg.MaxInfoHashes, cfg.MaxInfoHashPeers),
		tokenStore:        newTokenStore(),
		locallyOriginated: map[InfoHash]struct{}{},
		locallyInterested: map[InfoHash]struct{}{},
	}

	if dht.cfg.AnyPeerAF {
		dht.wantList = []string{"n4", "n6"}
	}

	if dht.cfg.Clock == nil {
		dht.cfg.Clock = clock.Real
	}

	// Create UDP socket.
	var err error
	if dht.cfg.ListenFunc != nil {
		dht.conn, err = dht.cfg.ListenFunc(cfg)
	} else {
		var addr *net.UDPAddr
		addr, err = net.ResolveUDPAddr("udp", dht.cfg.Address)
		if err != nil {
			return nil, err
		}

		dht.conn, err = net.ListenUDP("udp", addr)
	}
	if err != nil {
		return nil, err
	}

	// Start loops.
	log.Debugf("(%v) starting", dht.cfg.NodeID.ShortString())
	go dht.readLoop()
	go dht.controlLoop()

	return dht, nil
}

// Main loops. {{{1

type packet struct {
	Data []byte
	Addr net.UDPAddr
}

// Reads datagrams from the connection and queues them for processing on the
// l-goroutine.
func (dht *DHT) readLoop() {
	for {
		b, addr, err := denet.ReadDatagramFromUDP(dht.conn)

		switch {
		// Successful receive.
		case err == nil:
			dht.rxChan <- packet{
				Data: b,
				Addr: *addr,
			}

			// An address was unreachable.
		case denet.ErrorIsPortUnreachable(err):
			dht.addrUnreachableChan <- *addr

			// Unless we are stopping, other errors should not occur.
		case atomic.LoadUint32(&dht.stopping) == 0:
			log.Errore(err, "unexpected error while receiving")

			// We are stopping.
		default:
			return
		}
	}
}

// Main loop. Methods which are only to be run from this goroutine are named
// "lFoo".
func (dht *DHT) controlLoop() {
	defer close(dht.peersChan) // notifies client that no more peers are forthcoming
	defer dht.conn.Close()     // ensures the readLoop dies

	// Ticker for the cleanup operation.
	cleanupTicker := dht.cfg.Clock.NewTicker(dht.cfg.CleanupPeriod)
	defer cleanupTicker.Stop()

	// Ticker for the token secret rotation operation.
	tokenRotateTicker := dht.cfg.Clock.NewTicker(dht.cfg.TokenRotatePeriod)
	defer tokenRotateTicker.Stop()

	// Service requests.
	for {
		select {
		case <-dht.stopChan:
			log.Debugf("cl(%p) stopping", dht)
			return

			// Servicing client requests.
		case ani := <-dht.addNodeChan:
			log.Debugf("cl(%v) addNode %v", dht.cfg.NodeID.ShortString(), ani)
			dht.lAddNode(ani.Addr, ani.NodeID, ani.ForceAdd)

		case rpi := <-dht.requestPeersChan:
			log.Debugf("cl(%v) requestPeers %v", dht.cfg.NodeID.ShortString(), rpi)
			dht.lRequestPeers(rpi.InfoHash, rpi.Announce)

		case ch := <-dht.requestReachableNodesChan:
			r := dht.lListReachableNodes()
			log.Debugf("cl(%p) requestReachableNodes result=%v", dht, r)
			ch <- r

			// Network traffic.
		case pkt := <-dht.rxChan:
			err := dht.lRxPacket(pkt.Data, pkt.Addr)
			log.Errore(err, "rx packet")
			//log.Tracef("cl(%p) rxPacket %v", dht, pkt)

		case addr := <-dht.addrUnreachableChan:
			log.Debugf("cl addrUnreachable %v", addr)
			dht.lAddrUnreachable(addr)

			// These are generated internally when requests result in further work.
		case nodeID := <-dht.recurseNodeChan:
			log.Debugf("cl(%v)   recurseNode %v", dht.cfg.NodeID.ShortString(), nodeID)
			dht.lProcRecurseNode(nodeID)

		case n := <-dht.requestPingChan:
			log.Debugf("cl requestPing %v", n)
			dht.lTxPing(n)

			// Periodically run cleanup and periodic ping operations.
		case <-cleanupTicker.C():
			log.Debugf("cl cleanupTicker")
			dht.lCleanup()

			// Periodically rotate token secret.
		case <-tokenRotateTicker.C():
			log.Debugf("cl tokenRotateTicker")
			dht.tokenStore.Cycle()

			// Rate limiting...
		}
	}
}

// l: Addition of nodes. {{{1

// Add the node and ping it if it was not already known. NodeID is optional.
func (dht *DHT) lAddNode(addr net.UDPAddr, nodeID NodeID, forceAdd bool) error {
	n := dht.neighbourhood.routingTable.FindByAddress(addr)
	if n != nil {
		// already known
		return nil
	}

	if dht.acceptMoreNodes() || forceAdd {
		err := dht.lTxPingAddr(addr, nodeID)
		if err != nil {
			return err
		}
	}

	return nil
}

// l: Node listing. {{{1

func (dht *DHT) lListReachableNodes() []NodeInfo {
	var nodeInfo []NodeInfo

	dht.neighbourhood.routingTable.Visit(func(n *node) error {
		if !n.NodeID.Valid() || !n.IsReachable() {
			return nil
		}

		nodeInfo = append(nodeInfo, NodeInfo{
			NodeLocator: NodeLocator{
				NodeID: n.NodeID,
				Addr:   n.Addr,
			},
		})
		return nil
	})

	return nodeInfo
}

// l: Peer searching. {{{1

// Called via channel from client.
func (dht *DHT) lRequestPeers(infoHash InfoHash, announce bool) error {
	if announce {
		dht.lSetLocallyOriginated(infoHash, announce)
	}

	dht.lSetLocallyInterested(infoHash, true)

	if dht.needMorePeers(infoHash) {
		dht.lRequestPeersActual(infoHash)
	}

	return nil
}

// Called when more peers are needed for an infohash. We already know we don't
// have the maximum number.
func (dht *DHT) lRequestPeersActual(infoHash InfoHash) error {
	closest := dht.neighbourhood.routingTable.routingTree.LookupFiltered(infoHash, dht.lFilterPredicate)

	for _, n := range closest {
		dht.lRequestPeersFrom(n, infoHash)
	}

	return nil
}

func (dht *DHT) lRequestPeersFrom(n *node, infoHash InfoHash) {
	dht.lTxGetPeers(n, infoHash)
	n.MarkContacted(dht.cfg.Clock, infoHash)
}

func (dht *DHT) lFilterPredicate(infoHash InfoHash, n *node) bool {
	return n.NodeID.Valid() && n.NumPendingQueries() < dht.cfg.MaxPendingQueries && !n.WasContactedRecently(infoHash, dht.cfg.SearchRetryPeriod)
}

// l: Node searching. {{{1

// Called via channel to do further searchinng based on a node. Generated
// internally from other work.
func (dht *DHT) lProcRecurseNode(nodeID NodeID) error {
	closest := dht.neighbourhood.routingTable.routingTree.LookupFiltered(InfoHash(nodeID), dht.lFilterPredicate)
	for _, n := range closest {
		dht.lTxFindNode(n, nodeID)
		n.MarkContacted(dht.cfg.Clock, InfoHash(nodeID))
	}

	return nil
}

// Called from RX when we are informed of new nodes.
func (dht *DHT) lReceivedNodes(nodes []NodeLocator, originAddr net.UDPAddr) error {
	for _, locator := range nodes {
		if locator.NodeID == dht.cfg.NodeID {
			// Skip references to ourself.
			continue
		}

		if locator.Addr.String() == originAddr.String() {
			// Ignore self-promotion.
			continue
		}

		n, wasInserted := dht.neighbourhood.routingTable.Node(locator.NodeID, locator.Addr)
		if !wasInserted {
			// Duplicate.
			continue
		}

		if dht.needMoreNodes() {
			select {
			case dht.recurseNodeChan <- locator.NodeID:
			default:
				// Too many find_node commands queued. The peer has already been added
				// to the routing table, so we're not losing any information.
			}
		}

		dht.lRequestMorePeers(n) // TODO: check this
	}

	return nil
}

func (dht *DHT) lRequestMorePeers(n *node) {
	for infoHash := range dht.locallyInterested {
		if !dht.needMorePeers(infoHash) {
			continue
		}

		dht.lRequestPeersFrom(n, infoHash)
	}
}

// l: Cleanup. {{{1

// Run cleanup operations. Called periodically.
func (dht *DHT) lCleanup() {
	nodesToBePinged := dht.neighbourhood.Cleanup(dht.cfg.CleanupPeriod)
	go dht.slowPingLoop(nodesToBePinged)
}

// Runs in its own goroutine.
func (dht *DHT) slowPingLoop(nodes []*node) {
	duration := dht.cfg.CleanupPeriod - 1*time.Minute
	perPingWait := duration / time.Duration(len(nodes))
	startTime := dht.cfg.Clock.Now()

	for i, n := range nodes {
		dht.requestPingChan <- n

		waitUntil := startTime.Add(perPingWait * time.Duration(i+1))

		select {
		case <-timerAt(dht.cfg.Clock, waitUntil):
		case <-dht.stopChan:
			return
		}
	}
}
