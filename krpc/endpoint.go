package krpc

import (
	"encoding/binary"
	"fmt"
	"github.com/zeebo/bencode"
	"net"
)

// An endpoint is encoded as a 6-byte string (IPv4 IP:port) or 18 byte string
// (IPv6 IP:port). The length is used to disambiguate.
type Endpoint net.UDPAddr

func (n Endpoint) MarshalBencode() ([]byte, error) {
	var b [18]byte

	if n.IP == nil {
		return nil, nil
	}

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

func (n Endpoint) IsEmptyBencode() bool {
	return n.IP == nil
}

func (n *Endpoint) UnmarshalBencode(b []byte) error {
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

	*n = Endpoint{
		IP:   ip,
		Port: int(port),
	}

	return nil
}
