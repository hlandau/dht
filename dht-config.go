package dht

import (
	"github.com/hlandau/degoutils/clock"
	denet "github.com/hlandau/degoutils/net"
	"time"
)

// DHT configuration.
type Config struct {
	// IP address to listen on. If blank, a port is chosen randomly.
	Address string `usage:"Address to bind on"`

	// Number of peers that DHT will try to find for every infohash being searched.
	// Default: 50.
	NumTargetPeers int `usage:"Maximum number of peers to retrieve for an infohash"`

	// Minimum number of nodes. Default: 16.
	MinNodes int `usage:"Minimum number of nodes"`

	// Maximum nodes to store in routing table. Default: 100.
	MaxNodes int `usage:"Maximum number of nodes to store in the routing table"`

	// How often to ping nodes in the network to see if they are reachable. Default: 15 minutes.
	CleanupPeriod time.Duration `usage:"How often to ping nodes to see if they are reachable"`

	// How often to rotate announce_peer tokens. Default: 10 minutes.
	TokenRotatePeriod time.Duration `usage:"How often to rotate announce_peer tokens"`

	// ...
	SearchRetryPeriod time.Duration `usage:"Search retry period"`

	// Maximum packets per second to be processed. If negative, no limit is imposed. Default: 100.
	RateLimit int64 `usage:"Maximum packets per second to be processed"`

	// The maximum number of infohashes for whicha a peer list should be
	// maintained. Default: 2048.
	MaxInfoHashes int `usage:"Maximum number of infohashes to maintain a peer list for"`

	// The maximum number of peers to track for each infohash. Default: 256.
	MaxInfoHashPeers int `usage:"Maximum number of values to store for a given infohash"`

	// The maximum number of pending queries before a node is considered unreachable.
	MaxPendingQueries int `usage:"Maximum number of pending queries before a node is considered unreachable"`

	// Node ID. A random Node ID is generated if this is left blank.
	NodeID NodeID `usage:"Node ID"`

	// If not set, request peers only of the address family (IPv4 or IPv6) used to make
	// requests. If set, request peers of all supported address families (IPv4, IPv6).
	AnyPeerAF bool `usage:"Return peers of all address families"`

	// If set, this is used to get a listener instead of net.Listen.
	ListenFunc func(cfg *Config) (denet.UDPConn, error)

	// If set, use this clock. Else use a realtime clock.
	Clock clock.Clock
}

func (cfg *Config) setDefaults() {
	if cfg.NumTargetPeers == 0 {
		cfg.NumTargetPeers = 50
	}

	if cfg.MinNodes == 0 {
		cfg.MinNodes = 16
	}

	if cfg.MaxNodes == 0 {
		cfg.MaxNodes = 500
	}

	if cfg.CleanupPeriod == 0 {
		cfg.CleanupPeriod = 15 * time.Minute
	}

	if cfg.TokenRotatePeriod == 0 {
		cfg.TokenRotatePeriod = 5 * time.Minute
	}

	if cfg.SearchRetryPeriod == 0 {
		cfg.SearchRetryPeriod = 15 * time.Second
	}

	if cfg.RateLimit == 0 {
		cfg.RateLimit = 100
	}

	if cfg.MaxInfoHashes == 0 {
		cfg.MaxInfoHashes = 2048
	}

	if cfg.MaxInfoHashPeers == 0 {
		cfg.MaxInfoHashPeers = 256
	}

	if cfg.MaxPendingQueries == 0 {
		cfg.MaxPendingQueries = 5
	}

	if !cfg.NodeID.Valid() {
		cfg.NodeID = GenerateNodeID()
	}
}
