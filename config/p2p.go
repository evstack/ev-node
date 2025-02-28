package config

// P2PConfig stores configuration related to peer-to-peer networking.
type P2PConfig struct {
	ListenAddress   string // Address to listen for incoming connections
	ExternalAddress string // External address to advertise to peers
	Seeds           string // Comma separated list of seed nodes to connect to
	BlockedPeers    string // Comma separated list of nodes to ignore
	AllowedPeers    string // Comma separated list of nodes to whitelist
}
