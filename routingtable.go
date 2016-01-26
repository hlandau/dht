package dht

import "net"

type routingTable struct {
	routingTree *routingTree

	// The keys are of the format "IP:port", representing UDP addresses.
	// The hostname must be an IP, not a name.
	addresses map[string]*node
}

func newRoutingTable() *routingTable {
	return &routingTable{
		routingTree: &routingTree{},
		addresses:   make(map[string]*node),
	}
}

// Looks up a peer by IP:port. Returns nil if no such peer is found.
func (rt *routingTable) FindByAddress(addr net.UDPAddr) *node {
	n, _ := rt.addresses[addr.String()]
	return n
}

// Number of addresses in the routing table.
func (rt *routingTable) Size() int {
	return len(rt.addresses)
}

// Insert peer.
func (rt *routingTable) Insert(n *node) {
	if !isValidAddress(n.Addr) {
		panic("routingTable.Insert with nil peer")
	}

	nn := rt.FindByAddress(n.Addr)
	if nn != nil {
		return //err
	}

	rt.addresses[n.Addr.String()] = n

	if n.NodeID.Valid() {
		rt.routingTree.Insert(n)
	}
}

func (rt *routingTable) Update(n *node) {
	nn := rt.FindByAddress(n.Addr)
	if nn == nil {
		return //fmt.Errorf("peer not present in routing table: %v", p.Addr)
	}

	if n.NodeID.Valid() {
		rt.routingTree.Insert(n)
		//rt.addresses[n.Addr.String()].NodeID = n.NodeID
	}
}

// Get or create peer by address.
func (rt *routingTable) Node(nodeID NodeID, addr net.UDPAddr) (n *node, wasInserted bool) {
	n = rt.FindByAddress(addr)
	if n != nil {
		return n, false
	}

	n = newNode(addr, nodeID)

	rt.Insert(n)
	return n, true
}

func (rt *routingTable) Remove(n *node) {
	delete(rt.addresses, n.Addr.String())
	rt.routingTree.Cut(InfoHash(n.NodeID))
}

func (rt *routingTable) Visit(f func(n *node) error) error {
	for addr, n := range rt.addresses {
		if addr != n.Addr.String() {
			panic("consistency error: " + n.Addr.String())
		}

		if addr == "" {
			panic("invalid address key")
		}

		err := f(n)
		if err != nil {
			return err
		}
	}

	return nil
}
