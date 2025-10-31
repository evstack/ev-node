package common

import (
	"context"

	pubsub "github.com/libp2p/go-libp2p-pubsub"

	"github.com/celestiaorg/go-header"

	syncnotifier "github.com/evstack/ev-node/pkg/sync/notifier"
)

// broadcaster interface for P2P broadcasting
type Broadcaster[H header.Header[H]] interface {
	WriteToStoreAndBroadcast(ctx context.Context, payload H, opts ...pubsub.PubOpt) error
	Store() header.Store[H]
	Notifier() *syncnotifier.Notifier
}
