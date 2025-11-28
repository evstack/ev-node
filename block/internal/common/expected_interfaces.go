package common

import (
	"context"

	"github.com/evstack/ev-node/types"
	pubsub "github.com/libp2p/go-libp2p-pubsub"

	"github.com/celestiaorg/go-header"
)

type (
	HeaderP2PBroadcaster = Broadcaster[*types.SignedHeader]
	DataP2PBroadcaster   = Broadcaster[*types.Data]
)

// Broadcaster interface for P2P broadcasting
type Broadcaster[H header.Header[H]] interface {
	WriteToStoreAndBroadcast(ctx context.Context, payload H, opts ...pubsub.PubOpt) error
	AppendDAHint(ctx context.Context, daHeight uint64, hashes ...types.Hash) error
	GetByHeight(ctx context.Context, height uint64) (H, uint64, error)
}

//
//// Decorator to access the payload type without the container
//type Decorator[H header.Header[H]] struct {
//	nested Broadcaster[*sync.DAHeightHintContainer[H]]
//}
//
//func NewDecorator[H header.Header[H]](nested Broadcaster[*sync.DAHeightHintContainer[H]]) Decorator[H] {
//	return Decorator[H]{nested: nested}
//}
//
//func (d Decorator[H]) WriteToStoreAndBroadcast(ctx context.Context, payload H, opts ...pubsub.PubOpt) error {
//	return d.nested.WriteToStoreAndBroadcast(ctx, &sync.DAHeightHintContainer[H]{Entry: payload}, opts...)
//}
//
//func (d Decorator[H]) AppendDAHint(ctx context.Context, daHeight uint64, hashes ...types.Hash) error {
//	return d.nested.AppendDAHint(ctx, daHeight, hashes...)
//}
//func (d Decorator[H]) GetByHeight(ctx context.Context, height uint64) (H, error) {
//	panic("not implemented")
//}
