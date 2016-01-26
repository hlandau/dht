package dht

import (
	"hash/crc32"
	"net"
)

var crc32c = crc32.MakeTable(crc32.Castagnoli)

func nodeIDIsAllowed(ip net.IP, nodeID NodeID) bool {
	return conformNodeID(ip, nodeID) == nodeID
}

func conformNodeID(ip net.IP, nodeID NodeID) NodeID {
	var b [20]byte
	copy(b[:], nodeID)

	// r is supposed to be random but let's make this deterministic and get it
	// from the last byte of the original NodeID.
	r := byte(b[19]) & 0x07

	ip4 := ip.To4()
	var y uint32
	if ip4 != nil {
		var x [4]byte
		copy(x[:], ip4)
		x[0] &= 0x03
		x[1] &= 0x0F
		x[2] &= 0x3F
		x[3] &= 0xFF
		x[0] |= (r << 5)
		y = crc32.Checksum(x[:], crc32c)
	} else {
		var x [16]byte
		copy(x[:], ip)
		x[0] &= 0x01
		x[1] &= 0x03
		x[2] &= 0x07
		x[3] &= 0x0f
		x[4] &= 0x1f
		x[5] &= 0x3f
		x[6] &= 0x7f
		x[7] &= 0xff
		x[0] |= (r << 5)
		y = crc32.Checksum(x[:], crc32c)
	}

	b[0] = byte((y >> 24) & 0xFF)
	b[1] = byte((y >> 16) & 0xFF)
	b[2] = byte((y>>8)&0xF8) | (b[2] & 0x07)
	b[19] = (b[19] & 0xF8) | r

	return NodeID(string(b[:]))
}
