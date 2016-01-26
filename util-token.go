package dht

import "io"
import "net"
import "crypto/rand"
import "crypto/hmac"
import "crypto/sha256"
import "crypto/subtle"

// Used to generate and verify tokens.
type tokenStore struct {
	secrets [][]byte
}

func newTokenStore() *tokenStore {
	ts := &tokenStore{
		secrets: [][]byte{newTokenSecret()},
	}
	return ts
}

// Generate a token to provide to a node at the given address for it to return
// to us for verification.
func (ts *tokenStore) Generate(addr net.UDPAddr) []byte {
	return computeToken(ts.secrets[0], addr)
}

// Verify whether a token provided by a node at the given address is valid.
func (ts *tokenStore) Verify(token []byte, addr net.UDPAddr) bool {
	ok := false

	for _, secret := range ts.secrets {
		ok = ok || checkToken(secret, token, addr)
	}

	return ok
}

// Cycle token key. To be called periodically.
func (ts *tokenStore) Cycle() {
	ts.secrets = [][]byte{newTokenSecret(), ts.secrets[0]}
}

func newTokenSecret() []byte {
	b := make([]byte, 32)
	rand.Read(b)
	return b
}

func checkToken(secret []byte, token []byte, addr net.UDPAddr) bool {
	correctToken := computeToken(secret, addr)
	return subtle.ConstantTimeCompare(correctToken, token) == 1
}

func computeToken(secret []byte, addr net.UDPAddr) []byte {
	h := hmac.New(sha256.New, secret)
	io.WriteString(h, addr.String())
	return h.Sum(nil)
}
