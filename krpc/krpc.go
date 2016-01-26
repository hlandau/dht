package krpc

import "net"
import "crypto/rand"
import "sync/atomic"
import "github.com/zeebo/bencode"
import denet "github.com/hlandau/degoutils/net"
import "encoding/binary"
import "bytes"
import "reflect"

// KRPC message.
type Message struct {
	TxID string `bencode:"t"` // For correlation.
	Type string `bencode:"y"` // Query, response or error? [qre]

	Method string             `bencode:"q,omitempty"` // Queries: method name.
	Args   interface{}        `bencode:"-"`           // Queries: Query arguments.
	Args_  bencode.RawMessage `bencode:"a,omitempty"` // (Internal use only.)

	Response  interface{}        `bencode:"-"`           // Responses: Response value.
	Response_ bencode.RawMessage `bencode:"r,omitempty"` // (Internal use only.)

	Error []interface{} `bencode:"e,omitempty"` // Error responses: error information.
}

// Send a query to a host.
func MakeQuery(method string, args interface{}) (*Message, error) {
	var txIDb [4]byte
	txIDi := newTxID()
	binary.LittleEndian.PutUint32(txIDb[:], txIDi)

	argsb, err := bencode.EncodeBytes(args)
	if err != nil {
		return nil, err
	}

	txID := string(txIDb[:])

	msg := Message{
		TxID:   txID,
		Type:   "q",
		Method: method,
		Args:   args,
		Args_:  argsb,
	}

	return &msg, nil
}

func WriteResponse(conn *net.UDPConn, remoteAddr net.UDPAddr, q *Message, response interface{}) error {
	responseb, err := bencode.EncodeBytes(response)
	if err != nil {
		return err
	}

	msg := Message{
		TxID:      q.TxID,
		Type:      "r",
		Response_: responseb,
	}

	return Write(conn, remoteAddr, &msg)
}

func WriteError(conn *net.UDPConn, remoteAddr net.UDPAddr, q *Message, errorCode int, errorMessage string) error {
	msg := Message{
		TxID: q.TxID,
		Type: "e",
		Error: []interface{}{
			errorCode, errorMessage,
		},
	}

	return Write(conn, remoteAddr, &msg)
}

// Write a message to a host.
func Write(conn *net.UDPConn, remoteAddr net.UDPAddr, msg *Message) error {
	b := bytes.Buffer{}
	err := bencode.NewEncoder(&b).Encode(msg)
	if err != nil {
		return err
	}

	_, err = conn.WriteToUDP(b.Bytes(), &remoteAddr)
	return err
}

func Decode(b []byte) (msg *Message, err error) {
	err = bencode.DecodeBytes(b, &msg)
	if err != nil {
		return
	}

	if msg.Type == "q" {
		msg.Args, err = decodeByType(msg.Args_, queryTypes[msg.Method])
	}
	// msg.Type == "r": Must decode later using ResponseAsMethod.

	return
}

func decodeByType(b bencode.RawMessage, valueType reflect.Type) (interface{}, error) {
	if valueType != nil {
		v := reflect.New(valueType).Interface()
		err := bencode.DecodeBytes([]byte(b), v)
		if err != nil {
			return nil, err
		}

		return v, nil
	}

	var generic interface{}
	err := bencode.DecodeBytes([]byte(b), &generic)
	if err != nil {
		return nil, err
	}

	return generic, nil
}

// Read a message from the connection.
func Read(conn *net.UDPConn) (msg *Message, remoteAddr *net.UDPAddr, err error) {
	buf, remoteAddr, err := denet.ReadDatagramFromUDP(conn)
	if err != nil {
		return
	}

	msg, err = Decode(buf)
	return
}

func (msg *Message) ResponseAsMethod(method string) error {
	if msg.Type != "r" {
		return nil
	}

	var err error
	msg.Response, err = decodeByType(msg.Response_, responseTypes[method])
	return err
}

var queryTypes = map[string]reflect.Type{}
var responseTypes = map[string]reflect.Type{}

func RegisterQuery(methodName string, methodType interface{}) {
	queryTypes[methodName] = reflect.TypeOf(methodType)
}

func RegisterResponse(methodName string, methodType interface{}) {
	responseTypes[methodName] = reflect.TypeOf(methodType)
}

// TxID counter.
var curTxID uint32 = 0

func init() {
	var b [4]byte
	rand.Read(b[:])
	curTxID = binary.LittleEndian.Uint32(b[:])
}

func newTxID() uint32 {
	return atomic.AddUint32(&curTxID, 1)
}
