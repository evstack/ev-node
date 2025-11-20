package sync

import (
	"context"

	"github.com/celestiaorg/go-header"
	"github.com/evstack/ev-node/pkg/store"
)

type storeGetter[H header.Header[H]] func(context.Context, store.Store, header.Hash) (H, error)
type storeGetterByHeight[H header.Header[H]] func(context.Context, store.Store, uint64) (H, error)

type exchangeWrapper[H header.Header[H]] struct {
	header.Exchange[H]
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
	return ew.Exchange.Get(ctx, hash)
}

func (ew *exchangeWrapper[H]) GetByHeight(ctx context.Context, height uint64) (H, error) {
	// Check DA store first
	if ew.daStore != nil && ew.getterByHeight != nil {
		if h, err := ew.getterByHeight(ctx, ew.daStore, height); err == nil && !h.IsZero() {
			return h, nil
		}
	}

	// Fallback to network exchange
	return ew.Exchange.GetByHeight(ctx, height)
}
