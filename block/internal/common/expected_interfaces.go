package common

import (
	"context"

	goheader "github.com/celestiaorg/go-header"
)

// Broadcaster interface for handling P2P stores and broadcasting
type Broadcaster[H goheader.Header[H]] interface {
	WriteToStoreAndBroadcast(ctx context.Context, payload H) error
	Store() goheader.Store[H]
}
