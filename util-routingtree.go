package dht

// DHT routing using a binary tree and no buckets.
//
// Nodes have 20 byte IDs. When looking up an infohash for itself or for a
// remote host, the nodes have to look in its routing table for the closest
// nodes and return them.
//
// The distance between a node and an infohash is the XOR of the respective
// strings. This means that 'sorting' nodes only makes sense with an infohash
// as the pivot. You can't pre-sort nodes in any meaningful way.
//
// Most BitTorrent/Kademlia DHT implementations use a mix of bit-by-bit
// comparison with the usage of buckets. That works very well. But I wanted to
// try something different, that doesn't use buckets. Buckets have a single id
// and one calculates the distance based on that, speeding up lookups.
//
// I decided to lay out the routing table in a binary tree instead, which is
// more intuitive. At the moment, the implementation is a real tree, not a
// free-list, but it's performing well.
//
// All nodes are inserted in the binary tree, with a fixed height of 160 (20
// bytes). To lookup an infohash, I do an inorder traversal using the infohash
// bit for each level.
//
// In most cases the lookup reaches the bottom of the tree without hitting the
// target infohash, since in the vast majority of the cases it's not in my
// routing table. Then I simply continue the in-order traversal (but then to
// the 'left') and return after I collect the 8 closest nodes.
//
// To speed things up, I keep the tree as short as possible. The path to each
// node is compressed and later uncompressed if a collision happens when
// inserting another node.
//
// I don't know how slow the overall algorithm is compared to a implementation
// that uses buckets, but for what is worth, the routing table lookups don't
// even show on the CPU profiling anymore.
type routingTree struct {
	Zero, One *routingTree
	Node      *node
}

func (rt *routingTree) Insert(p *node) {
	rt.put(p, 0)
}

const kNodes = 8

func (rt *routingTree) put(n *node, i int) {
	if i >= NodeIDBits {
		// Replaces the existing value, if any.
		rt.Node = n
		return
	}

	if rt.Node != nil {
		if rt.Node.NodeID == n.NodeID {
			// Replace existing compressed value.
			rt.Node = n
			return
		}

		// Compression collision. Branch them out.
		oldNode := rt.Node
		rt.Node = nil
		rt.branchOut(n, oldNode, i)
	}

	if idBitSet(n.NodeID, i) == false {
		if rt.Zero == nil {
			rt.Zero = &routingTree{
				Node: n,
			}
			return
		}

		rt.Zero.put(n, i+1)
	} else {
		if rt.One == nil {
			rt.One = &routingTree{
				Node: n,
			}
			return
		}

		rt.One.put(n, i+1)
	}
}

func (rt *routingTree) branchOut(x, y *node, i int) {
	// Since they are branching out it's guaranteed that no other nodes
	// exist below this branch currently, so just create the respective
	// nodes until their respective bits are different.
	bitX := idBitSet(x.NodeID, i)
	bitY := idBitSet(y.NodeID, i)

	if bitX != bitY {
		rt.put(x, i)
		rt.put(y, i)
		return
	}

	if bitX == false {
		rt.Zero = &routingTree{}
		rt.Zero.branchOut(x, y, i+1)
	} else {
		rt.One = &routingTree{}
		rt.One.branchOut(x, y, i+1)
	}
}

func idBitSet(nodeID NodeID, i int) bool {
	return byte(nodeID[i/8])<<uint8(i%8)&128 != 0
}

func ihBitSet(infoHash InfoHash, i int) bool {
	return byte(infoHash[i/8])<<uint8(i%8)&128 != 0
}

func alwaysYes(infoHash InfoHash, n *node) bool {
	return true
}

// Find the nodes nearest to the given infohash.
func (rt *routingTree) Lookup(infoHash InfoHash) []*node {
	return rt.LookupFiltered(infoHash, alwaysYes)
}

// Find the nodes nearest to the given infohash, filtered by filterFunc.
func (rt *routingTree) LookupFiltered(infoHash InfoHash, filterFunc func(infoHash InfoHash, n *node) bool) []*node {
	r := make([]*node, 0, kNodes)
	if rt == nil || infoHash == "" {
		return nil
	}

	return rt.traverse(infoHash, 0, r, filterFunc)
}

func (rt *routingTree) traverse(infoHash InfoHash, i int, r []*node, filterFunc func(infoHash InfoHash, n *node) bool) []*node {
	if rt == nil {
		return r
	}

	if rt.Node != nil {
		if filterFunc(infoHash, rt.Node) {
			return append(r, rt.Node)
		}
	}

	if i >= len(infoHash)*8 || len(r) >= kNodes {
		return r
	}

	var L, R *routingTree
	if ihBitSet(infoHash, i) == false {
		L, R = rt.Zero, rt.One
	} else {
		R, L = rt.Zero, rt.One
	}

	r = L.traverse(infoHash, i+1, r, filterFunc)
	if len(r) >= kNodes {
		return r
	}
	return R.traverse(infoHash, i+1, r, filterFunc)
}

func (rt *routingTree) Cut(infoHash InfoHash) {
	rt.cut(infoHash, 0)
}

func (rt *routingTree) cut(infoHash InfoHash, i int) (cutMe bool) {
	if rt == nil || i >= len(infoHash)*8 {
		return true
	}

	if ihBitSet(infoHash, i) == false {
		if rt.Zero.cut(infoHash, i+1) {
			rt.Zero = nil
			if rt.One == nil {
				return true
			}
		}
	} else {
		if rt.One.cut(infoHash, i+1) {
			rt.One = nil
			if rt.Zero == nil {
				return true
			}
		}
	}

	return false
}
