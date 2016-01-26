package dht

import (
	"github.com/hlandau/dht/krpc"
	"github.com/hlandauf/bencode"
	"reflect"
	"testing"
)

type test struct {
	B      string
	Method string
}

var tests = []test{
	// ping q/r
	{
		B: `d1:q4:ping1:ad2:id20:....................e1:t4:abcd1:y1:qe`,
	},
	{
		B:      `d1:rd2:id20:....................e1:t4:abcd1:y1:re`,
		Method: "ping",
	},
	// find_nodes q/r
	{
		B: `d1:q10:find_nodes1:ad4:wantl2:n42:n6e2:id20:....................6:target20:,,,,,,,,,,,,,,,,,,,,e1:t4:abcd1:y1:qe`,
	},
	{
		B:      `d1:rd2:id20:....................5:nodes52:!!!!!!!!!!!!!!!!!!!!<<<<>>!!!!!!!!!!!!!!!!!!!!<<<<>>e1:t4:abcd1:y1:re`,
		Method: "find_nodes",
	},
	// get_peers q/r
	{
		B: `d1:q9:get_peers1:ad4:wantl2:n42:n6e2:id20:....................9:info_hash20:,,,,,,,,,,,,,,,,,,,,e1:t4:abcd1:y1:qe`,
	},
	{
		B:      `d1:rd2:id20:....................5:token8:@@@@@@@@6:valuesl6:<<<<>>6:<<<<>>6:<<<<>>ee1:t4:abcd1:y1:re`,
		Method: "get_peers",
	},
	{
		B:      `d1:rd2:id20:....................5:token8:@@@@@@@@6:valuesl18:<<<<<<<<<<<<<<<<>>6:<<<<>>6:<<<<>>ee1:t4:abcd1:y1:re`,
		Method: "get_peers",
	},
	{
		B:      `d1:rd2:id20:....................5:token8:@@@@@@@@5:nodes26:,,,,,,,,,,,,,,,,,,,,<<<<>>e1:t4:abcd1:y1:re`,
		Method: "get_peers",
	},
	{
		B:      `d1:rd2:id20:....................5:token8:@@@@@@@@6:nodes638:,,,,,,,,,,,,,,,,,,,,<<<<<<<<<<<<<<<<>>e1:t4:abcd1:y1:re`,
		Method: "get_peers",
	},
	// announce_peer q/r
	{
		B: `d1:q13:announce_peer1:ad2:id20:....................4:porti65321e5:token8:@@@@@@@@9:info_hash20:,,,,,,,,,,,,,,,,,,,,12:implied_porti0ee1:t4:abcd1:y1:qe`,
	},
	{
		B:      `d1:rd2:id20:....................e1:t4:abcd1:y1:re`,
		Method: "announce_peer",
	},
}

func TestKRPC(t *testing.T) {
	for _, tt := range tests {
		msg, err := krpc.Decode([]byte(tt.B))
		if err != nil {
			t.Fatalf("cannot decode: %v", err)
		}

		if tt.Method != "" {
			err = msg.ResponseAsMethod(tt.Method)
			if err != nil {
				t.Fatalf("cannot decode response part: %v", err)
			}
		}

		b, err := bencode.EncodeBytes(msg)
		if err != nil {
			t.Fatalf("couldn't encode: %v", err)
		}

		msg2, err := krpc.Decode(b)
		if err != nil {
			t.Fatalf("cannot decode: %v: %v %v", err, string(b), tt.B)
		}

		if tt.Method != "" {
			err = msg2.ResponseAsMethod(tt.Method)
			if err != nil {
				t.Fatalf("cannot decode response part: %v", err)
			}
		}

		if !reflect.DeepEqual(msg, msg2) {
			t.Logf("not equal after reserialization: %#v != %#v", msg, msg2)
		}
	}
}
