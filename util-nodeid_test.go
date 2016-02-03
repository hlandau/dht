package dht

import (
	"strings"
	"testing"
)

func TestNodeID(t *testing.T) {
	tests := []struct {
		Hex  string
		Fail bool
	}{
		{
			Hex: "393de2e25380e5c7d48dde8acd2732c6592342cb",
		},
		{
			Hex: "393DE2E25380E5C7D48DDE8ACD2732C6592342CB",
		},
		{
			Hex:  "",
			Fail: true,
		},
		{
			Hex:  "a",
			Fail: true,
		},
		{
			Hex:  "aa",
			Fail: true,
		},
		{
			Hex:  "393E2E25380E5C7D48DDE8ACD2732C6592342CB",
			Fail: true,
		},
		{
			Hex:  "0393E2E25380E5C7D48DDE8ACD2732C65 2342CB",
			Fail: true,
		},
	}

	for _, tst := range tests {
		if tst.Fail {
			_, err := ParseNodeID(tst.Hex)
			if err == nil {
				t.Fatal()
			}
			continue
		}

		nid := MustParseNodeID(tst.Hex)

		if !nid.Valid() {
			t.Fatal()
		}

		s := nid.String()
		if strings.ToLower(tst.Hex) != s {
			t.Fatal()
		}
	}
}

func TestGenerateNodeID(t *testing.T) {
	nid := GenerateNodeID()
	if !nid.Valid() {
		t.Fatal()
	}
	nid2 := GenerateNodeID()
	if nid == nid2 {
		t.Fatal()
	}
}
