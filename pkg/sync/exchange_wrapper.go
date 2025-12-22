package sync

import (
	"context"

	"github.com/celestiaorg/go-header"
	"github.com/evstack/ev-node/pkg/store"
)

type storeGetter[H header.Header[H]] func(context.Context, store.Store, header.Hash) (H, error)
type storeGetterByHeight[H header.Header[H]] func(context.Context, store.Store, uint64) (H, error)

// P2PExchange defines the interface for the underlying P2P exchange.
type P2PExchange[H header.Header[H]] interface {
	header.Exchange[H]
	Start(context.Context) error
	Stop(context.Context) error
}

type exchangeWrapper[H header.Header[H]] struct {
	p2pExchange    P2PExchange[H]
	daStore        store.Store
	getter         storeGetter[H]
	getterByHeight storeGetterByHeight[H]
}

func (ew *exchangeWrapper[H]) Get(ctx context.Context, hash header.Hash) (H, error) {
	// Check DA store first
	if ew.daStore != nil && ew.getter != nil {
		if h, err := ew.getter(ctx, ew.daStore, hash); err == nil && !h.IsZero() {
			return h, nil
		}
	}

	// Fallback to network exchange
	return ew.p2pExchange.Get(ctx, hash)
}

func (ew *exchangeWrapper[H]) GetByHeight(ctx context.Context, height uint64) (H, error) {
	// Check DA store first
	if ew.daStore != nil && ew.getterByHeight != nil {
		if h, err := ew.getterByHeight(ctx, ew.daStore, height); err == nil && !h.IsZero() {
			return h, nil
		}
	}

	// Fallback to network exchange
	return ew.p2pExchange.GetByHeight(ctx, height)
}

func (ew *exchangeWrapper[H]) Head(ctx context.Context, opts ...header.HeadOption[H]) (H, error) {
	return ew.p2pExchange.Head(ctx, opts...)
}

func (ew *exchangeWrapper[H]) GetRangeByHeight(ctx context.Context, from H, to uint64) ([]H, error) {
	return ew.p2pExchange.GetRangeByHeight(ctx, from, to)
}

func (ew *exchangeWrapper[H]) Start(ctx context.Context) error {
	return ew.p2pExchange.Start(ctx)
}

func (ew *exchangeWrapper[H]) Stop(ctx context.Context) error {
	return ew.p2pExchange.Stop(ctx)
}
