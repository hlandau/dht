package dht

import (
	"encoding/hex"
	"fmt"
)

// Info hash. Binary form.
type InfoHash string

// An infohash is a 20-byte SHA-1 value.
const (
	InfoHashBytes = 20
	InfoHashBits  = NodeIDBytes * 8
)

// Parse a hexadecimal infohash.
func ParseInfoHash(infoHash string) (InfoHash, error) {
	var n InfoHash

	err := n.UnmarshalString(infoHash)
	if err != nil {
		return "", err
	}

	return n, nil
}

// Unmarshal from a hexadecimal infohash string.
func (infoHash *InfoHash) UnmarshalString(s string) error {
	b, err := hex.DecodeString(s)
	if err != nil {
		return err
	}

	if len(b) != 20 {
		return fmt.Errorf("info hash must be 20 bytes")
	}

	*infoHash = InfoHash(b)
	return nil
}

// Returns the infohash in hexadecimal form.
func (infoHash InfoHash) String() string {
	return hex.EncodeToString([]byte(infoHash))
}

// True iff the infohash is the right length.
func (infoHash InfoHash) Valid() bool {
	return len(infoHash) == InfoHashBytes
}

func commonBits(x []byte, y []byte) int {
	// byte
	i := 0
	for ; i < 20; i++ {
		if x[i] != y[i] {
			break
		}
	}

	if i == 20 {
		return 160
	}

	xor := x[i] ^ y[i]

	// bit
	j := 0
	for (xor & 0x80) == 0 {
		xor <<= 1
		j++
	}

	return 8*i + j
}
