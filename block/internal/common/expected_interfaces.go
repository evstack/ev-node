package common

import (
	"context"

	"github.com/celestiaorg/go-header"
	pubsub "github.com/libp2p/go-libp2p-pubsub"

	syncnotifier "github.com/evstack/ev-node/pkg/sync/notifier"
)

// broadcaster interface for P2P broadcasting
type Broadcaster[H header.Header[H]] interface {
	WriteToStoreAndBroadcast(ctx context.Context, payload H, opts ...pubsub.PubOpt) error
	Store() header.Store[H]
	Notifier() *syncnotifier.Notifier
}
