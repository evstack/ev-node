package sync

import (
	"context"

	"github.com/celestiaorg/go-header"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

type exchangeWrapper[H header.Header[H]] struct {
	header.Exchange[H]
	daStore store.Store
}

func (ew *exchangeWrapper[H]) Get(ctx context.Context, hash header.Hash) (H, error) {
	// Check DA store first
	var zero H
	if ew.daStore != nil {
		switch any(zero).(type) {
		case *types.SignedHeader:
			h, _, err := ew.daStore.GetBlockByHash(ctx, hash)
			if err == nil && h != nil {
				return any(h).(H), nil
			}
		case *types.Data:
			_, d, err := ew.daStore.GetBlockByHash(ctx, hash)
			if err == nil && d != nil {
				return any(d).(H), nil
			}
		}
	}

	// Fallback to network exchange
	return ew.Exchange.Get(ctx, hash)
}

func (ew *exchangeWrapper[H]) GetByHeight(ctx context.Context, height uint64) (H, error) {
	// Check DA store first
	var zero H
	if ew.daStore != nil {
		switch any(zero).(type) {
		case *types.SignedHeader:
			h, _, err := ew.daStore.GetBlockData(ctx, height)
			if err == nil && h != nil {
				return any(h).(H), nil
			}
		case *types.Data:
			_, d, err := ew.daStore.GetBlockData(ctx, height)
			if err == nil && d != nil {
				return any(d).(H), nil
			}
		}
	}

	// Fallback to network exchange
	return ew.Exchange.GetByHeight(ctx, height)
}
