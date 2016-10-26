package dht

import (
	"crypto/sha1"
	"fmt"
	"github.com/hlandau/dht/krpc"
	"github.com/hlandau/eddsa"
	"net"
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
		//log.Debugf("cl(%v) lRxQuery %v %v", dht.cfg.NodeID.ShortString(), msg, &addr)
		err = dht.lRxQuery(msg, addr)
	case "r":
		//log.Tracef("cl(%v) lRxResponse %v %v", dht.cfg.NodeID.ShortString(), msg, &addr)
		err = dht.lRxResponse(msg, addr)
	case "e":
		log.Debugf("cl(%v) lRxError %v %v", dht.cfg.NodeID.ShortString(), msg, &addr)
		err = dht.lRxError(msg, addr)
	default:
		log.Warnf("unknown packet type received: %#v (from %v)", msg.Type, &addr)
		return nil
	}

	log.Noticee(err, "rx processing error")
	return err
}

// Handle an incoming query-type packet.
func (dht *DHT) lRxQuery(msg *krpc.Message, addr net.UDPAddr) error {
	nodeID, err := dht.lRxCheckNodeID(msg)
	if err != nil {
		log.Errore(err, "cl lRxCheckNodeID ", &addr)
		return err
	}

	// Make sure the peer exists so we can track responses.
	n := dht.neighbourhood.routingTable.FindByAddress(addr)
	if n == nil && dht.acceptMoreNodes() {
		dht.lTxPingAddr(addr, nodeID)
	}

	switch v := msg.Args.(type) {
	case *krPing:
		log.Debugf("cl(%v) lRxPingReq %v %v", dht.cfg.NodeID.ShortString(), msg, &addr)
		return dht.lRxPingReq(v, msg, addr)
	case *krGetPeersReq:
		log.Debugf("cl(%v) lRxGetPeersReq %v %v", dht.cfg.NodeID.ShortString(), msg, &addr)
		return dht.lRxGetPeersReq(v, msg, addr)
	case *krFindNodeReq:
		log.Debugf("cl(%v) lRxFindNodeReq %v %v", dht.cfg.NodeID.ShortString(), msg, &addr)
		return dht.lRxFindNodeReq(v, msg, addr)
	case *krGetReq:
		log.Debugf("cl(%v), lRxGetReq %v %v", dht.cfg.NodeID.ShortString(), msg, &addr)
		return dht.lRxGetReq(v, msg, addr)
	case *krAnnouncePeerReq:
		log.Debugf("cl(%v) lRxAnnouncePeerReq %v %v", dht.cfg.NodeID.ShortString(), msg, &addr)
		return dht.lRxAnnouncePeerReq(v, msg, addr)
	case *krPutReq:
		log.Debugf("cl(%v) lRxPutReq %v %v", dht.cfg.NodeID.ShortString(), msg, &addr)
		return dht.lRxPutReq(v, msg, addr)
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

var wantAll = []string{"n4", "n6"}

// Handle an incoming get query.
func (dht *DHT) lRxGetReq(v *krGetReq, msg *krpc.Message, addr net.UDPAddr) error {
	res := &krGetRes{
		ID:    dht.cfg.NodeID,
		Token: dht.tokenStore.Generate(addr),
	}

	neighbours := dht.neighbourhood.routingTable.routingTree.Lookup(v.Target)
	res.Nodes, res.Nodes6 = formNodeList(neighbours, wantAll, addr)

	datum := dht.peerStore.Datum(v.Target)
	if datum != nil {
		res.Value = datum.Value
	}

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
		n.LastRxTime = dht.cfg.Clock.Now().Add(-dht.cfg.SearchRetryPeriod) // TODO: check this

		if _, ok := dht.locallyInterested[v.InfoHash]; ok {
			dht.peersChan <- PeerResult{
				InfoHash: v.InfoHash,
				Addr:     addr,
			}
		}
	}

	// "Always reply positively. jech says this is to avoid 'backtracking', not
	// sure what that means."
	dht.lTxResponse(addr, msg, &krAnnouncePeerRes{
		ID: dht.cfg.NodeID,
	})
	return nil
}

const maxPutLen = 1000

// Handle an incoming put query.
func (dht *DHT) lRxPutReq(v *krPutReq, msg *krpc.Message, addr net.UDPAddr) error {
	if !dht.tokenStore.Verify(v.Token, addr) {
		dht.lTxError(addr, msg, 203, "bad token")
		return nil
	}

	datum := &Datum{
		Value:     v.Value,
		Key:       v.Key,
		Signature: v.Signature,
		Salt:      v.Salt,
	}

	if len(datum.Value) > maxPutLen {
		dht.lTxError(addr, msg, 205, "value too large")
		return nil
	}

	if len(datum.Salt) > 64 {
		dht.lTxError(addr, msg, 207, "salt too large")
		return nil
	}

	if !datum.IsMutable() {
		// The immutable case is simple.
		h := sha1.New()
		h.Write([]byte(fmt.Sprintf("%d:", len(datum.Value)) + datum.Value))
		target := InfoHash(string(h.Sum(nil)))

		dht.peerStore.AddDatum(target, datum)
	} else {
		// Mutable case.

		if v.SequenceNo != nil {
			datum.SequenceNo = *v.SequenceNo
		}

		// Yes, it really is the case that mutable keys are hashed using the raw
		// Ed25519 public key+salt whereas immutable keys are hashed using a
		// bencoded value.
		h := sha1.New()
		h.Write([]byte(datum.Key))
		h.Write(datum.Salt)
		keyTarget := InfoHash(string(h.Sum(nil)))

		oldDatum := dht.peerStore.Datum(keyTarget)
		if oldDatum != nil {
			if oldDatum.SequenceNo >= datum.SequenceNo {
				dht.lTxError(addr, msg, 302, "sequence number rollback not permitted")
				return nil
			}

			if v.CAS != nil && *v.CAS != oldDatum.SequenceNo {
				dht.lTxError(addr, msg, 301, "CAS mismatch")
				return nil
			}
		}

		tbs := fmt.Sprintf("3:seqi%de1:v%d:", datum.SequenceNo, len(v.Value)) + v.Value
		if len(v.Salt) > 0 {
			tbs = fmt.Sprintf("4:salt%d:", len(v.Salt)) + string(v.Salt) + tbs
		}

		if len(datum.Signature) != 64 || len(datum.Key) != 32 {
			dht.lTxError(addr, msg, 206, "bad signature")
			return nil
		}

		publicKey := eddsa.PublicKey{
			Curve: eddsa.Ed25519(),
			X:     []byte(datum.Key),
		}
		if !publicKey.Verify([]byte(tbs), []byte(datum.Signature)) {
			dht.lTxError(addr, msg, 206, "bad signature")
			return nil
		}

		dht.peerStore.AddDatum(keyTarget, datum)
	}

	n, _ := dht.neighbourhood.routingTable.Node(v.ID, addr)
	n.LastRxTime = dht.cfg.Clock.Now().Add(-dht.cfg.SearchRetryPeriod) // TODO: check this

	dht.lTxResponse(addr, msg, &krPutRes{
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

	n.LastRxTime = dht.cfg.Clock.Now()

	dht.neighbourhood.Upkeep(n)
	if dht.needMoreNodes() {
		select {
		case dht.recurseNodeChan <- n.NodeID:
		default:
		}
		//dht.findNode(nodeID)
	}

	// Type-specific dispatch.
	switch v := msg.Response.(type) {
	case *krPing:
		log.Debugf("cl(%v) lRxPingRes %v %v", dht.cfg.NodeID.ShortString(), msg, &addr)
		err = dht.lRxPingRes(v, msg, addr)
	case *krGetPeersRes:
		log.Debugf("cl(%v) lRxGetPeersRes %v %v", dht.cfg.NodeID.ShortString(), msg, &addr)
		err = dht.lRxGetPeersRes(v, msg, addr)
	case *krFindNodeRes:
		log.Debugf("cl(%v) lRxFindNodeRes %v %v", dht.cfg.NodeID.ShortString(), msg, &addr)
		err = dht.lRxFindNodeRes(v, msg, addr)
	case *krGetRes:
		log.Debugf("cl(%v) lRxGetRes %v %v", dht.cfg.NodeID.ShortString(), msg, &addr)
		err = dht.lRxGetRes(v, msg, addr)
	case *krAnnouncePeerRes:
		log.Debugf("cl(%v) lRxAnnouncePeerRes %v %v", dht.cfg.NodeID.ShortString(), msg, &addr)
		err = dht.lRxAnnouncePeerRes(v, msg, addr)
	case *krPutRes:
		log.Debugf("cl(%v) lRxPutRes %v %v", dht.cfg.NodeID.ShortString(), msg, &addr)
		err = dht.lRxPutRes(v, msg, addr)
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
	//log.Debugf("cl(%v) lRxGetPeersRes %#v", dht.cfg.NodeID.ShortString(), n.PendingQueries[msg.TxID])
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

	return nil
}

// Handle an incoming find_node response.
func (dht *DHT) lRxFindNodeRes(v *krFindNodeRes, msg *krpc.Message, addr net.UDPAddr) error {
	dht.neighbourhood.routingTable.Node(v.ID, addr)

	dht.lReceivedNodes(v.Nodes, addr)
	dht.lReceivedNodes(v.Nodes6, addr)

	return nil
}

func (dht *DHT) lRxGetRes(v *krGetRes, msg *krpc.Message, addr net.UDPAddr) error {
	// TODO
	return nil
}

// Handle an incoming announce_peer response. No-op.
func (dht *DHT) lRxAnnouncePeerRes(v *krAnnouncePeerRes, msg *krpc.Message, addr net.UDPAddr) error {
	// Nothing to do.
	return nil
}

// Handle an incoming put response. No-op.
func (dht *DHT) lRxPutRes(v *krPutRes, msg *krpc.Message, addr net.UDPAddr) error {
	// Nothing to do.
	return nil
}

// Rx error. {{{2

// Handle an incoming error. No-op.
func (dht *DHT) lRxError(msg *krpc.Message, addr net.UDPAddr) error {
	// Nothing to do.
	return nil
}
