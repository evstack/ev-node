package common

import (
	"context"

	"github.com/celestiaorg/go-header"
)

// broadcaster interface for P2P broadcasting
type Broadcaster[H header.Header[H]] interface {
	WriteToStoreAndBroadcast(ctx context.Context, payload H) error
	Store() header.Store[H]
}
