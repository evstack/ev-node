package sync

import (
	"context"

	"github.com/celestiaorg/go-header"
)

// GetterFunc retrieves a header by hash from a backing store.
type GetterFunc[H header.Header[H]] func(context.Context, header.Hash) (H, error)

// GetterByHeightFunc retrieves a header by height from a backing store.
type GetterByHeightFunc[H header.Header[H]] func(context.Context, uint64) (H, error)

// RangeGetterFunc retrieves headers in range [from, to) from a backing store.
// Returns the contiguous headers found starting from 'from', and the next height needed.
type RangeGetterFunc[H header.Header[H]] func(ctx context.Context, from, to uint64) ([]H, uint64, error)

// P2PExchange defines the interface for the underlying P2P exchange.
type P2PExchange[H header.Header[H]] interface {
	header.Exchange[H]
	Start(context.Context) error
	Stop(context.Context) error
}

type exchangeWrapper[H header.Header[H]] struct {
	p2pExchange    P2PExchange[H]
	getter         GetterFunc[H]
	getterByHeight GetterByHeightFunc[H]
	rangeGetter    RangeGetterFunc[H]
}

func (ew *exchangeWrapper[H]) Get(ctx context.Context, hash header.Hash) (H, error) {
	// Check DA store first
	if ew.getter != nil {
		if h, err := ew.getter(ctx, hash); err == nil && !h.IsZero() {
			return h, nil
		}
	}

	// Fallback to network exchange
	return ew.p2pExchange.Get(ctx, hash)
}

func (ew *exchangeWrapper[H]) GetByHeight(ctx context.Context, height uint64) (H, error) {
	// Check DA store first
	if ew.getterByHeight != nil {
		if h, err := ew.getterByHeight(ctx, height); err == nil && !h.IsZero() {
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
	fromHeight := from.Height() + 1

	// If no range getter, fallback entirely to P2P
	if ew.rangeGetter == nil {
		return ew.p2pExchange.GetRangeByHeight(ctx, from, to)
	}

	// Try DA store first for contiguous range
	daHeaders, nextHeight, err := ew.rangeGetter(ctx, fromHeight, to)
	if err != nil {
		// DA store failed, fallback to P2P for entire range
		return ew.p2pExchange.GetRangeByHeight(ctx, from, to)
	}

	// Got everything from DA
	if nextHeight >= to {
		return daHeaders, nil
	}

	// Need remainder from P2P
	if len(daHeaders) == 0 {
		// Nothing from DA, get entire range from P2P
		return ew.p2pExchange.GetRangeByHeight(ctx, from, to)
	}

	// Get remainder from P2P starting after last DA header
	lastDAHeader := daHeaders[len(daHeaders)-1]
	p2pHeaders, err := ew.p2pExchange.GetRangeByHeight(ctx, lastDAHeader, to)
	if err != nil {
		return nil, err
	}

	return append(daHeaders, p2pHeaders...), nil
}

func (ew *exchangeWrapper[H]) Start(ctx context.Context) error {
	return ew.p2pExchange.Start(ctx)
}

func (ew *exchangeWrapper[H]) Stop(ctx context.Context) error {
	return ew.p2pExchange.Stop(ctx)
}
