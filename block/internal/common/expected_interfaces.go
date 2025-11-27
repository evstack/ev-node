package common

import (
	"context"

	"github.com/evstack/ev-node/types"
	pubsub "github.com/libp2p/go-libp2p-pubsub"

	"github.com/celestiaorg/go-header"
)

type (
	HeaderP2PBroadcaster = Decorator[*types.SignedHeader]
	DataP2PBroadcaster   = Decorator[*types.Data]
)

// Broadcaster interface for P2P broadcasting
type Broadcaster[H header.Header[H]] interface {
	WriteToStoreAndBroadcast(ctx context.Context, payload H, opts ...pubsub.PubOpt) error
	Store() header.Store[H]
	AppendDAHint(ctx context.Context, daHeight uint64, hashes ...types.Hash) error
}

// Decorator to access the the payload type without the container
type Decorator[H header.Header[H]] struct {
	nested Broadcaster[*types.DAHeightHintContainer[H]]
}

func NewDecorator[H header.Header[H]](nested Broadcaster[*types.DAHeightHintContainer[H]]) Decorator[H] {
	return Decorator[H]{nested: nested}
}

func (d Decorator[H]) WriteToStoreAndBroadcast(ctx context.Context, payload H, opts ...pubsub.PubOpt) error {
	return d.nested.WriteToStoreAndBroadcast(ctx, &types.DAHeightHintContainer[H]{Entry: payload}, opts...)
}

func (d Decorator[H]) Store() HeightStore[H] {
	return HeightStoreImpl[H]{store: d.nested.Store()}
}
func (d Decorator[H]) XStore() header.Store[*types.DAHeightHintContainer[H]] {
	return d.nested.Store()
}

func (d Decorator[H]) AppendDAHint(ctx context.Context, daHeight uint64, hashes ...types.Hash) error {
	return d.nested.AppendDAHint(ctx, daHeight, hashes...)
}

// HeightStore is a subset of goheader.Store
type HeightStore[H header.Header[H]] interface {
	GetByHeight(context.Context, uint64) (H, error)
}

type HeightStoreImpl[H header.Header[H]] struct {
	store header.Store[*types.DAHeightHintContainer[H]]
}

func (s HeightStoreImpl[H]) GetByHeight(ctx context.Context, height uint64) (H, error) {
	var zero H
	v, err := s.store.GetByHeight(ctx, height)
	if err != nil {
		return zero, err
	}
	return v.Entry, nil

}
