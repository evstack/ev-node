package celestianodefiber

import (
	"github.com/celestiaorg/celestia-node/api/client"
)

// Config configures the celestia-node-backed Fibre adapter.
type Config struct {
	// Client is the full celestia-node api/client.Config. See that package
	// for field semantics (ReadConfig.BridgeDAAddr, SubmitConfig.DefaultKeyName,
	// SubmitConfig.CoreGRPCConfig, SubmitConfig.Fibre, etc.).
	Client client.Config

	// ListenChannelSize bounds the buffered BlobEvent channel returned by
	// Listen. 0 selects the default (16), matching the upstream
	// blob.Subscribe buffer so backpressure behaves consistently.
	ListenChannelSize int
}
