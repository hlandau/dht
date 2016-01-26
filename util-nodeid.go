package dht

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// Node ID. Binary form.
type NodeID string

// A node ID is a 20-byte random value.
const (
	NodeIDBytes = 20
	NodeIDBits  = NodeIDBytes * 8
)

// Parse a hexadecimal node ID.
func ParseNodeID(nodeID string) (NodeID, error) {
	var n NodeID

	err := n.UnmarshalString(nodeID)
	if err != nil {
		return "", err
	}

	return n, nil
}

// Unmarshal from a hexadecimal node ID string.
func (nid *NodeID) UnmarshalString(s string) error {
	b, err := hex.DecodeString(s)
	if err != nil {
		return err
	}

	if len(b) != 20 {
		return fmt.Errorf("nodeID must be 20 bytes")
	}

	*nid = NodeID(b)
	return nil
}

// Returns the node ID in hexadecimal form.
func (nid NodeID) String() string {
	return hex.EncodeToString([]byte(nid))
}

// True iff the node ID is the right length.
func (nid NodeID) Valid() bool {
	return len(nid) == NodeIDBytes
}

func (nid NodeID) MarshalJSON() ([]byte, error) {
	return json.Marshal(nid.String())
}

func (nid *NodeID) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	nodeID, err := ParseNodeID(s)
	if err != nil {
		return err
	}

	*nid = nodeID
	return nil
}

// Generate a random node ID.
func GenerateNodeID() NodeID {
	var b [20]byte
	rand.Read(b[:])
	return NodeID(b[:])
}
