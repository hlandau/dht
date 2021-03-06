package dht

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/hlandau/dht/krpc"
	"github.com/hlandauf/bencode"
	"net"
	"strings"
)

func init() {
	krpc.RegisterQuery("ping", krPing{})
	krpc.RegisterResponse("ping", krPing{})
	krpc.RegisterQuery("find_node", krFindNodeReq{})
	krpc.RegisterResponse("find_node", krFindNodeRes{})
	krpc.RegisterQuery("get_peers", krGetPeersReq{})
	krpc.RegisterResponse("get_peers", krGetPeersRes{})
	krpc.RegisterQuery("announce_peer", krAnnouncePeerReq{})
	krpc.RegisterResponse("announce_peer", krAnnouncePeerRes{})
	krpc.RegisterQuery("get", krGetReq{})
	krpc.RegisterResponse("get", krGetRes{})
	krpc.RegisterQuery("put", krPutReq{})
	krpc.RegisterResponse("put", krPutRes{})
}

func getNodeID(msg *krpc.Message) NodeID {
	r := msg.Response
	if msg.Type == "q" {
		r = msg.Args
	}

	hni, ok := r.(interface {
		GetNodeID() NodeID
	})
	if !ok {
		log.Errorf("warning: %T doesn't support GetNodeID", msg.Response)
		return ""
	}
	return hni.GetNodeID()
}

// KRPC "ping" request/response.
type krPing struct {
	ID NodeID `bencode:"id"`
}

func (m *krPing) String() string {
	return fmt.Sprintf("ping(%v)", m.ID.ShortString())
}

func (m *krPing) GetNodeID() NodeID {
	return m.ID
}

// KRPC "find_node" request.
type krFindNodeReq struct {
	ID     NodeID   `bencode:"id"`
	Target NodeID   `bencode:"target"`
	Want   []string `bencode:"want,omitempty"` // "n4", "n6"
}

func (m *krFindNodeReq) String() string {
	return fmt.Sprintf("findNode?(%v ->%v %v)", m.ID.ShortString(), m.Target.ShortString(), m.Want)
}

func (m *krFindNodeReq) GetNodeID() NodeID {
	return m.ID
}

// KRPC "find_node" response.
type krFindNodeRes struct {
	ID     NodeID      `bencode:"id"`
	Nodes  krNodesIPv4 `bencode:"nodes,omitempty"`
	Nodes6 krNodesIPv6 `bencode:"nodes6,omitempty"`
}

func (m *krFindNodeRes) String() string {
	return fmt.Sprintf("findNode.(%v %v %v)", m.ID.ShortString(), m.Nodes, m.Nodes6)
}

func (m *krFindNodeRes) GetNodeID() NodeID {
	return m.ID
}

// KRPC "get_peers" request.
type krGetPeersReq struct {
	ID       NodeID   `bencode:"id"`
	InfoHash InfoHash `bencode:"info_hash"`
	Want     []string `bencode:"want,omitempty"` // "n4", "n6"
}

func (m *krGetPeersReq) String() string {
	return fmt.Sprintf("getPeers?(%v ->%v %v)", m.ID.ShortString(), m.InfoHash.ShortString(), m.Want)
}

func (m *krGetPeersReq) GetNodeID() NodeID {
	return m.ID
}

// KRPC "get_peers" response.
type krGetPeersRes struct {
	ID        NodeID          `bencode:"id"`
	Token     []byte          `bencode:"token"`
	Nodes     krNodesIPv4     `bencode:"nodes,omitempty"`
	Nodes6    krNodesIPv6     `bencode:"nodes6,omitempty"`
	Endpoints []krpc.Endpoint `bencode:"values"`
}

func (m *krGetPeersRes) String() string {
	return fmt.Sprintf("getPeers.(%v tok=%x %v %v %v)", m.ID.ShortString(), string(m.Token), m.Nodes, m.Nodes6, m.Endpoints)
}

func (m *krGetPeersRes) GetNodeID() NodeID {
	return m.ID
}

// KRPC "announce_peer" request.
type krAnnouncePeerReq struct {
	ID          NodeID   `bencode:"id"`
	ImpliedPort int      `bencode:"implied_port"`
	InfoHash    InfoHash `bencode:"info_hash"`
	Port        int      `bencode:"port"`
	Token       []byte   `bencode:"token"`
}

func (m *krAnnouncePeerReq) String() string {
	port := "implied"
	if m.ImpliedPort == 0 {
		port = fmt.Sprintf("%d", m.Port)
	}
	return fmt.Sprintf("announcePeer?(%v ->%v %v tok=%x)", m.ID.ShortString(), m.InfoHash.ShortString(), port, m.Token)
}

func (m *krAnnouncePeerReq) GetNodeID() NodeID {
	return m.ID
}

// KRPC "announce_peer" response.
type krAnnouncePeerRes struct {
	ID NodeID `bencode:"id"`
}

func (m *krAnnouncePeerRes) String() string {
	return fmt.Sprintf("announcePeer.(%v)", m.ID.ShortString())
}

func (m *krAnnouncePeerRes) GetNodeID() NodeID {
	return m.ID
}

// KRPC "get" request.
type krGetReq struct {
	ID     NodeID   `bencode:"id"`
	Target InfoHash `bencode:"target"`

	Seq *uint64 `bencode:"seq,omitempty"` // mutable values only
}

func (m *krGetReq) GetNodeID() NodeID {
	return m.ID
}

// KRPC "get" response.
type krGetRes struct {
	ID     NodeID      `bencode:"id"`
	Nodes  krNodesIPv4 `bencode:"nodes,omitempty"`
	Nodes6 krNodesIPv6 `bencode:"nodes6,omitempty"`
	Token  []byte      `bencode:"token"`
	Value  string      `bencode:"v"`

	// For mutable values only.
	Key        krPublicKey `bencode:"k,omitempty"`   // 32 bytes
	Signature  krSignature `bencode:"sig,omitempty"` // 64 byte Ed25519 signature
	SequenceNo *uint64     `bencode:"seq,omitempty"`
}

func (m *krGetRes) GetNodeID() NodeID {
	return m.ID
}

// KRPC "put" request.
type krPutReq struct {
	ID    NodeID `bencode:"id"`
	Token []byte `bencode:"token"`
	Value string `bencode:"v"`

	// For mutable values only.
	Key        krPublicKey `bencode:"k,omitempty"`    // 32-byte Ed25519 public key
	Signature  krSignature `bencode:"sig,omitempty"`  // 64 byte Ed25519 signature
	Salt       []byte      `bencode:"salt,omitempty"` // Optional salt
	SequenceNo *uint64     `bencode:"seq,omitempty"`
	CAS        *uint64     `bencode:"cas,omitempty"` // Optional compare-and-set sequence number
}

func (m *krPutReq) GetNodeID() NodeID {
	return m.ID
}

// KRPC "put" response.
type krPutRes struct {
	ID NodeID `bencode:"id"`
}

func (m *krPutRes) GetNodeID() NodeID {
	return m.ID
}

// A NodeLocator provides the NodeID and UDP address of a node.
type NodeLocator struct {
	NodeID NodeID      // Node ID
	Addr   net.UDPAddr // DHT Node UDP IP:Port
}

func (nl *NodeLocator) String() string {
	return fmt.Sprintf("%v@%s", nl.NodeID.ShortString(), &nl.Addr)
}

func isValidAddress(addr net.UDPAddr) bool {
	return !addr.IP.IsUnspecified() && addr.Port != 0
}

// An IPv4 node list is a string which is the concatenation of 26-byte
// node descriptors (NodeID, IPv4 Address, Port).
type krNodesIPv4 []NodeLocator

func (n krNodesIPv4) String() string {
	var s []string
	for _, x := range n {
		s = append(s, x.String())
	}
	return fmt.Sprintf("n4(%v)", strings.Join(s, "; "))
}

func (n krNodesIPv4) MarshalBencode() ([]byte, error) {
	b := bytes.Buffer{}

	var bb [26]byte

	for i := range n {
		v4 := n[i].Addr.IP.To4()
		if v4 == nil {
			return nil, fmt.Errorf("IPv6 address in IPv4 nodes list")
		}

		copy(bb[:], n[i].NodeID)
		copy(bb[20:], v4)
		binary.BigEndian.PutUint16(bb[24:26], uint16(n[i].Addr.Port))

		_, err := b.Write(bb[:])
		if err != nil {
			return nil, err
		}
	}

	return bencode.EncodeBytes(b.Bytes())
}

func (n *krNodesIPv4) UnmarshalBencode(b []byte) error {
	var bb []byte
	err := bencode.DecodeBytes(b, &bb)
	if err != nil {
		return err
	}

	if len(bb)%26 != 0 {
		return fmt.Errorf("not divisible by 26")
	}

	*n = nil
	for len(bb) > 0 {
		nodeID := NodeID(bb[0:20])
		ip := net.IP(bb[20:24])
		port := binary.BigEndian.Uint16(bb[24:26])

		*n = append(*n, NodeLocator{
			NodeID: nodeID,
			Addr: net.UDPAddr{
				IP:   ip,
				Port: int(port),
			},
		})

		bb = bb[26:]
	}

	return nil
}

// An IPv6 node list is a string which is the concatenation of 38-byte
// node descriptors (NodeID, IPv6 Address, Port).
type krNodesIPv6 []NodeLocator

func (n krNodesIPv6) String() string {
	var s []string
	for _, x := range n {
		s = append(s, x.String())
	}
	return fmt.Sprintf("n6(%v)", strings.Join(s, "; "))
}

func (n krNodesIPv6) MarshalBencode() ([]byte, error) {
	b := bytes.Buffer{}

	var bb [38]byte

	for i := range n {
		v4 := n[i].Addr.IP.To4()
		if v4 != nil {
			return nil, fmt.Errorf("IPv4 address in IPv6 nodes list")
		}

		copy(bb[:], n[i].NodeID)
		copy(bb[20:], n[i].Addr.IP.To16())
		binary.BigEndian.PutUint16(bb[36:38], uint16(n[i].Addr.Port))

		_, err := b.Write(bb[:])
		if err != nil {
			return nil, err
		}
	}

	return bencode.EncodeBytes(b.Bytes())
}

func (n *krNodesIPv6) UnmarshalBencode(b []byte) error {
	var bb []byte
	err := bencode.DecodeBytes(b, &bb)
	if err != nil {
		return err
	}

	if len(bb)%38 != 0 {
		return fmt.Errorf("not divisible by 38")
	}

	*n = nil
	for len(bb) > 0 {
		nodeID := NodeID(bb[0:20])
		ip := net.IP(bb[20:36])
		port := binary.BigEndian.Uint16(bb[36:38])

		*n = append(*n, NodeLocator{
			NodeID: nodeID,
			Addr: net.UDPAddr{
				IP:   ip,
				Port: int(port),
			},
		})

		bb = bb[38:]
	}

	return nil
}

// Ed25519 public key.
type krPublicKey string

// Is the key a well-formatted Ed25519 public key?
func (k krPublicKey) IsWellFormed() bool {
	return len(k) == 32
}

// Ed25519 signature.
type krSignature string

// Is the key a well-formatted (i.e. 64-byte) signature?
func (k krSignature) IsWellFormed() bool {
	return len(k) == 64
}

// An endpoint is encoded as a 6-byte string (IPv4 IP:port) or 18 byte string
// (IPv6 IP:port). The length is used to disambiguate.
type xkrEndpoint net.UDPAddr

/*
func (n krEndpoint) MarshalBencode() ([]byte, error) {
	var b [18]byte

	v4 := n.IP.To4()
	if v4 != nil {
		copy(b[:], v4)
		binary.BigEndian.PutUint16(b[4:6], uint16(n.Port))
		return b[0:6], nil
	}

	copy(b[0:16], n.IP.To16())
	binary.BigEndian.PutUint16(b[16:18], uint16(n.Port))
	return b[:], nil
}

func (n *krEndpoint) UnmarshalBencode(b []byte) error {
	var bb []byte
	err := bencode.DecodeBytes(b, &bb)
	if err != nil {
		return err
	}

	if len(bb) != 6 && len(bb) != 18 {
		return fmt.Errorf("endpoint not 6 or 18 bytes in length")
	}

	ip := net.IP(bb[0 : len(bb)-2])
	port := binary.BigEndian.Uint16(bb[len(bb)-2:])

	*n = krEndpoint{
		IP:   ip,
		Port: int(port),
	}

	return nil
}*/
