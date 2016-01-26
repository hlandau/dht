package dhtstorage

import (
	"encoding/json"
	"fmt"
	"github.com/hlandau/dht"
	"io/ioutil"
	"os"
)

type doc struct {
	Nodes  []node     `json:"nodes"`
	NodeID dht.NodeID `json:"id"`
}

type node struct {
	NodeID string `json:"n,omitempty"`
	Addr   string `json:"a"`
}

// Save the current set of reachable DHT nodes to disk.
func Save(filename string, dht *dht.DHT) error {
	d := doc{
		NodeID: dht.NodeID(),
	}

	nodes := dht.ListReachableNodes()
	for i := range nodes {
		d.Nodes = append(d.Nodes, node{
			NodeID: nodes[i].NodeID.String(),
			Addr:   nodes[i].Addr.String(),
		})
	}

	b, err := json.Marshal(&d)
	if err != nil {
		return err
	}

	tfilename := filename + ".tmp"

	defer os.Remove(tfilename)

	err = ioutil.WriteFile(tfilename, b, 0644)
	if err != nil {
		return err
	}

	return os.Rename(tfilename, filename)
}

func loadDoc(filename string) (*doc, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	d := doc{}
	dec := json.NewDecoder(f)
	err = dec.Decode(&d)
	if err != nil {
		return nil, err
	}

	return &d, nil
}

func GetNodeID(filename string) (dht.NodeID, error) {
	doc, err := loadDoc(filename)
	if err != nil {
		return "", err
	}

	return doc.NodeID, nil
}

// Load the current set of reachable DHT nodes to disk.
func Load(filename string, dh *dht.DHT) error {
	d, err := loadDoc(filename)
	if err != nil {
		return err
	}

	if len(d.Nodes) == 0 {
		return fmt.Errorf("no nodes found")
	}

	for i := range d.Nodes {
		var nodeID dht.NodeID
		nodeID.UnmarshalString(d.Nodes[i].NodeID) // ignore error
		dht.AddHost(dh, d.Nodes[i].Addr, nodeID)  // ignore error
	}

	return nil
}
