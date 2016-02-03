package dht

import "testing"

func TestPeerStorage(t *testing.T) {
	ih := MustParseInfoHash("d1c5676ae7ac98e8b19f63565905105e3c4c37a2")

	p := newPeerStore(1, 2)

	if p.Count(ih) != 0 {
		t.Fatal()
	}

	ok := p.Add(ih, *mustResolve("1.2.3.4:1234"))
	if !ok {
		t.Fatal()
	}

	if p.Count(ih) != 1 {
		t.Fatal()
	}

	p.Add(ih, *mustResolve("2.3.4.5:2345"))
	if p.Count(ih) != 2 {
		t.Fatal()
	}

	p.Add(ih, *mustResolve("2.3.4.5:2345"))
	if p.Count(ih) != 2 {
		t.Fatal()
	}

	p.Add(ih, *mustResolve("3.4.5.6:3456"))
	if p.Count(ih) != 2 {
		t.Fatal()
	}

	ih2 := MustParseInfoHash("deca7a89a1dbdc4b213de1c0d5351e92582f31fb")
	if p.Count(ih2) != 0 {
		t.Fatal()
	}

	p.Add(ih2, *mustResolve("2.3.4.5:2345"))
	if p.Count(ih) != 0 {
		t.Fatal()
	}
	if p.Count(ih2) != 1 {
		t.Fatal()
	}
}
