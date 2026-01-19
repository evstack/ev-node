package cache

import (
	"context"
	"errors"
	"fmt"

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
// Worst case scenario is when headers were successfully submitted to DA, but confirmation was not received (e.g. node was
// restarted, networking issue occurred). In this case headers are re-submitted to DA (it's extra cost).
// evolve is able to skip duplicate headers so this shouldn't affect full nodes.
// TODO(tzdybal): we shouldn't try to push all pending headers at once; this should depend on max blob size
type PendingHeaders struct {
	base *pendingBase[*types.SignedHeader]
}

var errInFlightHeader = errors.New("inflight header")

func fetchSignedHeader(ctx context.Context, store storepkg.Store, height uint64) (*types.SignedHeader, error) {
	header, err := store.GetHeader(ctx, height)
	if err != nil {
		return nil, err
	}
	// in the executor, WIP headers are temporary stored. skip them until the process is completed
	if header.Height() == 0 {
		return nil, errInFlightHeader
	}
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

// GetPendingHeaders returns a sorted slice of pending headers along with their marshalled bytes.
// It uses an internal cache to avoid re-marshalling headers on subsequent calls.
func (ph *PendingHeaders) GetPendingHeaders(ctx context.Context) ([]*types.SignedHeader, [][]byte, error) {
	headers, err := ph.base.getPending(ctx)
	if err != nil {
		return nil, nil, err
	}

	if len(headers) == 0 {
		return nil, nil, nil
	}

	marshalled := make([][]byte, len(headers))
	lastSubmitted := ph.base.lastHeight.Load()

	for i, header := range headers {
		height := lastSubmitted + uint64(i) + 1

		// Try to get from cache first
		if cached := ph.base.getMarshalledForHeight(height); cached != nil {
			marshalled[i] = cached
			continue
		}

		// Marshal if not in cache
		data, err := header.MarshalBinary()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal header at height %d: %w", height, err)
		}
		marshalled[i] = data

		// Store in cache
		ph.base.setMarshalledForHeight(height, data)
	}

	return headers, marshalled, nil
}

func (ph *PendingHeaders) NumPendingHeaders() uint64 {
	return ph.base.numPending()
}

func (ph *PendingHeaders) SetLastSubmittedHeaderHeight(ctx context.Context, newLastSubmittedHeaderHeight uint64) {
	ph.base.setLastSubmittedHeight(ctx, newLastSubmittedHeaderHeight)
}

func (ph *PendingHeaders) init() error {
	return ph.base.init()
}
