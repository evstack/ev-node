package common

import (
	"context"

	"github.com/evstack/ev-node/types"
	pubsub "github.com/libp2p/go-libp2p-pubsub"

	"github.com/celestiaorg/go-header"
)

type (
	HeaderP2PBroadcaster = Broadcaster[*types.P2PSignedHeader]
	DataP2PBroadcaster   = Broadcaster[*types.P2PData]
)

// Broadcaster interface for P2P broadcasting
type Broadcaster[H header.Header[H]] interface {
	WriteToStoreAndBroadcast(ctx context.Context, payload H, opts ...pubsub.PubOpt) error
	AppendDAHint(ctx context.Context, daHeight uint64, hashes ...types.Hash) error
	GetByHeight(ctx context.Context, height uint64) (H, uint64, error)
}
