package dht

import (
	denet "github.com/hlandau/degoutils/net"
	"github.com/hlandau/dht/krpc"
	"net"
)

// Ping an address. Node ID is optional.
func (dht *DHT) lTxPingAddr(addr net.UDPAddr, nodeID NodeID) error {
	n, _ := dht.neighbourhood.routingTable.Node(nodeID, addr)
	return dht.lTxPing(n)
}

// Ping a node. NodeID may be unknown.
func (dht *DHT) lTxPing(n *node) error {
	return dht.lTxQuery(n, "ping", &krPing{
		ID: dht.cfg.NodeID,
	})
}

func (dht *DHT) lTxFindNode(n *node, target NodeID) error {
	return dht.lTxQuery(n, "find_node", &krFindNodeReq{
		ID:     dht.cfg.NodeID,
		Target: target,
	})
}

// Send a get_peers command to a node.
func (dht *DHT) lTxGetPeers(n *node, infoHash InfoHash) error {
	return dht.lTxQuery(n, "get_peers", &krGetPeersReq{
		ID:       dht.cfg.NodeID,
		InfoHash: infoHash,
	})
}

// Send an announce_peer command to a node.
func (dht *DHT) lTxAnnouncePeer(n *node, infoHash InfoHash, token []byte) error {
	return dht.lTxQuery(n, "announce_peer", &krAnnouncePeerReq{
		ID:          dht.cfg.NodeID,
		InfoHash:    infoHash,
		Token:       token,
		ImpliedPort: 1,
		Port:        0,
	})
}

// Message writing.

func (dht *DHT) lTxQuery(n *node, method string, args interface{}) error {
	q, err := krpc.MakeQuery(method, args)
	if err != nil {
		return err
	}

	n.PendingQueries[q.TxID] = q
	err = krpc.Write(dht.conn, n.Addr, q)
	if err != nil && denet.ErrorIsPortUnreachable(err) {
		dht.lNodeUnreachable(n)
	}

	return nil
}

// Respond to a given query message.
func (dht *DHT) lTxResponse(addr net.UDPAddr, q *krpc.Message, response interface{}) error {
	return krpc.WriteResponse(dht.conn, addr, q, response)
}

// Called when an address is deemed to be unreachable.
func (dht *DHT) lAddrUnreachable(addr net.UDPAddr) {
	n := dht.neighbourhood.routingTable.FindByAddress(addr)
	if n == nil {
		return
	}

	dht.lNodeUnreachable(n)
}

// Called when a node is deemed to be unreachable.
func (dht *DHT) lNodeUnreachable(n *node) {
	// TODO
}
