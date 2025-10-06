package cache

import (
	"context"
	"fmt"
	"iter"

	"github.com/ipfs/go-datastore/query"
	"github.com/rs/zerolog"

	storepkg "github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

// PendingHeaders maintains headers that need to be published to DA layer
//
// Important assertions:
// - headers are safely stored in database before submission to DA
// - headers are always pushed to DA in order (by height)
// - DA submission of multiple headers is atomic - it's impossible to submit only part of a batch
//
// lastSubmittedHeaderHeight is updated only after receiving confirmation from DA.
// Worst case scenario is when headers was successfully submitted to DA, but confirmation was not received (e.g. node was
// restarted, networking issue occurred). In this case headers are re-submitted to DA (it's extra cost).
// evolve is able to skip duplicate headers so this shouldn't affect full nodes.
// TODO(tzdybal): we shouldn't try to push all pending headers at once; this should depend on max blob size
type PendingHeaders struct {
	base *pendingBase[*types.SignedHeader]
}

func fetchSignedHeader(ctx context.Context, store storepkg.Store, height uint64) (*types.SignedHeader, error) {
	header, err := store.GetHeader(ctx, height)
	return header, err
}

// NewPendingHeaders returns a new PendingHeaders struct
func NewPendingHeaders(store storepkg.Store, logger zerolog.Logger) (*PendingHeaders, error) {
	base, err := newPendingBase(store, logger, storepkg.LastSubmittedHeaderHeightKey, fetchSignedHeader)
	if err != nil {
		return nil, err
	}
	return &PendingHeaders{base: base}, nil
}

// GetPendingHeaders returns a sorted slice of pending headers.
func (ph *PendingHeaders) GetPendingHeaders(ctx context.Context) ([]*types.SignedHeader, error) {
	return ph.base.getPending(ctx)
}

func (ph *PendingHeaders) NumPendingHeaders() uint64 {
	return ph.base.numPending()
}

func (ph *PendingHeaders) SetLastSubmittedHeaderHeight(ctx context.Context, newLastSubmittedHeaderHeight uint64) {
	ph.base.setLastSubmittedHeight(ctx, newLastSubmittedHeaderHeight)
}

// Iterator returns an iterator that walks pending headers in ascending height order.
func (ph *PendingHeaders) Iterator(ctx context.Context) (Iterator[*types.SignedHeader], error) {
	return ph.base.iterator(ctx)
}
func (ph *PendingHeaders) Query(ctx context.Context) iter.Seq2[*types.SignedHeader, error] {
	return func(yield func(*types.SignedHeader, error) bool) {
		lastSubmittedHeight := ph.base.lastSubmittedHeight.Load()
		processedHeight, err := ph.base.store.Height(ctx)
		if err != nil {
			yield(nil, fmt.Errorf("stored height: %w", err))
			return
		}
		// Query headers from the store
		q := query.Query{
			Prefix: storepkg.HeaderPrefix(),
			Offset: int(lastSubmittedHeight + 1),
		}

		results, err := ph.base.store.Query(ctx, q)
		if err != nil {
			yield(nil, fmt.Errorf("query headers: %w", err))
			return
		}
		defer results.Close() // nolint: errcheck // error can be ignored

		for result := range results.Next() {
			if result.Error != nil {
				// Yield error for this entry
				if !yield(nil, fmt.Errorf("query result error: %w", result.Error)) {
					return
				}
				continue
			}
			header := new(types.SignedHeader)
			if err := header.UnmarshalBinary(result.Value); err != nil {
				if !yield(nil, fmt.Errorf("unmarshal header: %w", err)) {
					return
				}
				continue
			}

			if current := header.Height(); current > lastSubmittedHeight && current <= processedHeight {
				if !yield(header, nil) {
					return
				}
			}
		}
	}
}
