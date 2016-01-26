package dht

import (
	"fmt"
	"github.com/hlandau/dht/krpc"
	"net"
	"time"
)

// l: Handle a raw incoming packet. {{{2
func (dht *DHT) lRxPacket(data []byte, addr net.UDPAddr) error {
	msg, err := krpc.Decode(data)
	if err != nil {
		log.Noticee(err, "rx ignore (cannot decode)")
		return err
	}

	switch msg.Type {
	case "q":
		log.Debugf("cl lRxQuery %v %v", msg, addr)
		err = dht.lRxQuery(msg, addr)
	case "r":
		log.Tracef("cl lRxResponse %v %v", msg, addr)
		err = dht.lRxResponse(msg, addr)
	case "e":
		log.Debugf("cl lRxError %v %v", msg, addr)
		err = dht.lRxError(msg, addr)
	default:
		log.Warnf("unknown packet type received: %#v (from %v)", msg.Type, addr)
		return nil
	}

	log.Noticee(err, "rx processing error")
	return err
}

// Handle an incoming query-type packet.
func (dht *DHT) lRxQuery(msg *krpc.Message, addr net.UDPAddr) error {
	nodeID, err := dht.lRxCheckNodeID(msg)
	if err != nil {
		log.Errore(err, "cl lRxCheckNodeID ", addr)
		return err
	}

	// Make sure the peer exists so we can track responses.
	n := dht.neighbourhood.routingTable.FindByAddress(addr)
	if n == nil && dht.acceptMorePeers() {
		dht.lTxPingAddr(addr, nodeID)
	}

	switch v := msg.Args.(type) {
	case *krPing:
		log.Debugf("cl lRxPingReq %v %v", msg, addr)
		return dht.lRxPingReq(v, msg, addr)
	case *krGetPeersReq:
		log.Debugf("cl lRxGetPeersReq %v %v", msg, addr)
		return dht.lRxGetPeersReq(v, msg, addr)
	case *krFindNodeReq:
		log.Debugf("cl lRxFindNodeReq %v %v", msg, addr)
		return dht.lRxFindNodeReq(v, msg, addr)
	case *krAnnouncePeerReq:
		log.Debugf("cl lRxAnnouncePeerReq %v %v", msg, addr)
		return dht.lRxAnnouncePeerReq(v, msg, addr)
	default:
		log.Warnf("unknown query type received: %#v", msg.Method)
		return nil
	}
}

// Handle an incoming ping query. Respond with another ping.
func (dht *DHT) lRxPingReq(v *krPing, msg *krpc.Message, addr net.UDPAddr) error {
	dht.lTxResponse(addr, msg, krPing{
		ID: dht.cfg.NodeID,
	})

	return nil
}

// Handle an incoming get_peers query.
func (dht *DHT) lRxGetPeersReq(v *krGetPeersReq, msg *krpc.Message, addr net.UDPAddr) error {
	res := &krGetPeersRes{
		ID:        dht.cfg.NodeID,
		Token:     dht.tokenStore.Generate(addr),
		Endpoints: nil,
	}

	peers := dht.peersFor(v.InfoHash)
	if len(peers) > 0 {
		res.Endpoints = formPeerList(peers, v.Want, addr)
	} else {
		neighbours := dht.neighbourhood.routingTable.routingTree.Lookup(v.InfoHash)
		res.Nodes, res.Nodes6 = formNodeList(neighbours, v.Want, addr)
	}

	dht.lTxResponse(addr, msg, res)
	return nil
}

// Handle an incoming find_nodes query.
func (dht *DHT) lRxFindNodeReq(v *krFindNodeReq, msg *krpc.Message, addr net.UDPAddr) error {
	ihTarget := InfoHash(v.Target)

	res := &krFindNodeRes{
		ID: dht.cfg.NodeID,
	}

	neighbours := dht.neighbourhood.routingTable.routingTree.Lookup(ihTarget)
	res.Nodes, res.Nodes6 = formNodeList(neighbours, v.Want, addr)

	dht.lTxResponse(addr, msg, res)
	return nil
}

// Handle an incoming announce_peer query.
func (dht *DHT) lRxAnnouncePeerReq(v *krAnnouncePeerReq, msg *krpc.Message, addr net.UDPAddr) error {
	if dht.tokenStore.Verify(v.Token, addr) {
		announceAddr := addr
		if v.ImpliedPort == 0 {
			announceAddr.Port = v.Port
		}

		dht.peerStore.Add(v.InfoHash, announceAddr)

		n, _ := dht.neighbourhood.routingTable.Node(v.ID, addr)
		n.LastRxTime = time.Now().Add(-dht.cfg.SearchRetryPeriod)
	}

	// "Always reply positively. jech says this is to avoid 'backtracking', not
	// sure what that means."
	dht.lTxResponse(addr, msg, &krAnnouncePeerRes{
		ID: dht.cfg.NodeID,
	})
	return nil
}

// l: Rx response. {{{2

func (dht *DHT) lRxCheckNodeID(msg *krpc.Message) (NodeID, error) {
	// Make sure we have a NodeID in the response. Have to do this after calling
	// ResponseAsMethod.
	nodeID := getNodeID(msg)
	if !nodeID.Valid() || nodeID == dht.cfg.NodeID {
		// Ignore messages without a valid node ID or which appear to be from this
		// node.
		log.Noticef("rx ignore (no nodeID: %v)", nodeID)
		return "", fmt.Errorf("no nodeID")
	}

	return nodeID, nil
}

// Handle an incoming response-type query.
func (dht *DHT) lRxResponse(msg *krpc.Message, addr net.UDPAddr) error {
	var err error

	n := dht.neighbourhood.routingTable.FindByAddress(addr)
	if n == nil {
		// This can't be a valid response if we don't even know about the node.
		// Ping the node.
		err := dht.lTxPingAddr(addr, "") // ignore errors
		log.Errore(err, "cannot ping unknown node")

		return nil
	}

	// Ensure this is a response to a query we issued.
	q, ok := n.PendingQueries[msg.TxID]
	if !ok {
		// Unknown query.
		return nil
	}

	// Don't accept duplicate responses.
	defer delete(n.PendingQueries, msg.TxID)

	// Interpret method-specific response information.
	err = msg.ResponseAsMethod(q.Method)
	if err != nil {
		return err
	}

	nodeID, err := dht.lRxCheckNodeID(msg)
	if err != nil {
		return err
	}

	if !n.NodeID.Valid() {
		// We didn't already have the NodeID, set it.
		n.NodeID = nodeID
		dht.neighbourhood.routingTable.Update(n)
	} else if n.NodeID != nodeID {
		// Changed ID. TODO
	}

	n.LastRxTime = time.Now()

	dht.neighbourhood.Upkeep(n)
	if dht.needMorePeers() {
		select {
		case dht.recurseNodeChan <- n.NodeID:
		default:
		}
		//dht.findNode(nodeID)
	}

	// Type-specific dispatch.
	switch v := msg.Response.(type) {
	case *krPing:
		log.Debugf("cl lRxPingRes %v %v", msg, addr)
		err = dht.lRxPingRes(v, msg, addr)
	case *krGetPeersRes:
		log.Debugf("cl lRxGetPeersRes %v %v", msg, addr)
		err = dht.lRxGetPeersRes(v, msg, addr)
	case *krFindNodeRes:
		log.Debugf("cl lRxFindNodeRes %v %v", msg, addr)
		err = dht.lRxFindNodeRes(v, msg, addr)
	case *krAnnouncePeerRes:
		log.Debugf("cl lRxAnnouncePeerRes %v %v", msg, addr)
		err = dht.lRxAnnouncePeerRes(v, msg, addr)
	default:
		log.Warnf("unknown response type received: %#v", msg.Method)
	}

	return err
}

// Handle an incoming ping response. No-op.
func (dht *DHT) lRxPingRes(v *krPing, msg *krpc.Message, addr net.UDPAddr) error {
	// Nothing to do.
	return nil
}

// Process another node's response to a get_peers query. If the response contains peers,
// return them to the client. If it contains closer nodes, query them. Announce ourselves
// as a peer if applicable.
func (dht *DHT) lRxGetPeersRes(v *krGetPeersRes, msg *krpc.Message, addr net.UDPAddr) error {
	n, _ := dht.neighbourhood.routingTable.Node("", addr)
	log.Debugf("cl lRxGetPeersRes %#v", n.PendingQueries[msg.TxID])
	q := n.PendingQueries[msg.TxID].Args.(*krGetPeersReq)
	// We know p and q exist because these were checked earlier.

	infoHash := q.InfoHash
	if _, ok := dht.locallyOriginated[infoHash]; ok {
		dht.lTxAnnouncePeer(n, q.InfoHash, v.Token)
	}

	if v.Endpoints != nil {
		var newEndpoints []net.UDPAddr
		for _, endpoint := range v.Endpoints {
			inserted := dht.peerStore.Add(infoHash, net.UDPAddr(endpoint))
			if inserted {
				newEndpoints = append(newEndpoints, net.UDPAddr(endpoint))
			}
		}

		for _, e := range newEndpoints {
			// Put results on channel.
			dht.peersChan <- PeerResult{
				InfoHash: infoHash,
				Addr:     e,
			}
		}
	}

	var nodes []NodeLocator
	nodes = append(nodes, []NodeLocator(v.Nodes)...)
	nodes = append(nodes, []NodeLocator(v.Nodes6)...)
	if len(nodes) == 0 {
		return nil
	}

	for _, nodeLocator := range nodes {
		if nodeLocator.NodeID == dht.cfg.NodeID {
			// Ignore self.
			continue
		}

		// Ignore items already in the routing table.
		n2 := dht.neighbourhood.routingTable.FindByAddress(nodeLocator.Addr)
		if n2 != nil {
			continue
		}

		// Ignore references to ourself.
		if nodeLocator.Addr.String() == dht.cfg.Address {
			continue
		}

		n2, _ = dht.neighbourhood.routingTable.Node(nodeLocator.NodeID, nodeLocator.Addr)

		// Re-add the request to the queue.
		//
		// Announce has already been recorded via PeerStore.MarkLocallyOriginated,
		// so it can be set to false here.
		select {
		case dht.requestPeersChan <- requestPeersInfo{
			InfoHash: infoHash,
			Announce: false,
		}:
		default:
			// Channel full. The peer was already added to the routing table and will be
			// used the next time RequestPeers is called if it is close enough to the
			// infohash.
		}
	}

	return nil
}

// Handle an incoming find_node response.
func (dht *DHT) lRxFindNodeRes(v *krFindNodeRes, msg *krpc.Message, addr net.UDPAddr) error {
	dht.neighbourhood.routingTable.Node(v.ID, addr)

	dht.lReceivedNodes(v.Nodes, addr)
	dht.lReceivedNodes(v.Nodes6, addr)

	return nil
}

// Handle an incoming announce_peer response. No-op.
func (dht *DHT) lRxAnnouncePeerRes(v *krAnnouncePeerRes, msg *krpc.Message, addr net.UDPAddr) error {
	// Nothing to do.
	return nil
}

// Rx error. {{{2

// Handle an incoming error. No-op.
func (dht *DHT) lRxError(msg *krpc.Message, addr net.UDPAddr) error {
	// Nothing to do.
	return nil
}
