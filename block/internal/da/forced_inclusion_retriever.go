package da

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	datypes "github.com/evstack/ev-node/pkg/da/types"
	"github.com/evstack/ev-node/types"
)

// ErrForceInclusionNotConfigured is returned when the forced inclusion namespace is not configured.
var ErrForceInclusionNotConfigured = errors.New("forced inclusion namespace not configured")

// ForcedInclusionRetriever handles retrieval of forced inclusion transactions from DA.
type ForcedInclusionRetriever struct {
	client        Client
	logger        zerolog.Logger
	daEpochSize   uint64
	daStartHeight uint64
	asyncFetcher  *AsyncEpochFetcher // Required for async prefetching
}

// ForcedInclusionEvent contains forced inclusion transactions retrieved from DA.
type ForcedInclusionEvent struct {
	Timestamp     time.Time
	StartDaHeight uint64
	EndDaHeight   uint64
	Txs           [][]byte
}

// NewForcedInclusionRetriever creates a new forced inclusion retriever.
// The asyncFetcher parameter is required for background prefetching of DA epoch data.
func NewForcedInclusionRetriever(
	client Client,
	logger zerolog.Logger,
	daStartHeight, daEpochSize uint64,
	asyncFetcher *AsyncEpochFetcher,
) *ForcedInclusionRetriever {
	return &ForcedInclusionRetriever{
		client:        client,
		logger:        logger.With().Str("component", "forced_inclusion_retriever").Logger(),
		daStartHeight: daStartHeight,
		daEpochSize:   daEpochSize,
		asyncFetcher:  asyncFetcher,
	}
}

// RetrieveForcedIncludedTxs retrieves forced inclusion transactions at the given DA height.
// It respects epoch boundaries and only fetches at epoch start.
// If an async fetcher is configured, it will try to use cached data first for better performance.
func (r *ForcedInclusionRetriever) RetrieveForcedIncludedTxs(ctx context.Context, daHeight uint64) (*ForcedInclusionEvent, error) {
	// when daStartHeight is not set or no namespace is configured, we retrieve nothing.
	if !r.client.HasForcedInclusionNamespace() {
		return nil, ErrForceInclusionNotConfigured
	}

	if daHeight < r.daStartHeight {
		return nil, fmt.Errorf("DA height %d is before the configured start height %d", daHeight, r.daStartHeight)
	}

	epochStart, epochEnd, currentEpochNumber := types.CalculateEpochBoundaries(daHeight, r.daStartHeight, r.daEpochSize)

	if daHeight != epochEnd {
		r.logger.Debug().
			Uint64("da_height", daHeight).
			Uint64("epoch_end", epochEnd).
			Msg("not at epoch end - returning empty transactions")

		return &ForcedInclusionEvent{
			StartDaHeight: daHeight,
			EndDaHeight:   daHeight,
			Txs:           [][]byte{},
		}, nil
	}

	// Try to get from async fetcher cache first
	cachedEvent, err := r.asyncFetcher.GetCachedEpoch(ctx, daHeight)
	if err == nil && cachedEvent != nil {
		r.logger.Debug().
			Uint64("da_height", daHeight).
			Uint64("epoch_start", epochStart).
			Uint64("epoch_end", epochEnd).
			Int("tx_count", len(cachedEvent.Txs)).
			Msg("using cached epoch data from async fetcher")
		return cachedEvent, nil
	}
	// Cache miss or error, fall through to sync fetch
	if err != nil {
		r.logger.Debug().
			Err(err).
			Uint64("da_height", daHeight).
			Msg("failed to get cached epoch, falling back to sync fetch")
	}

	event := &ForcedInclusionEvent{
		StartDaHeight: epochStart,
		EndDaHeight:   epochEnd,
		Txs:           [][]byte{},
	}

	epochEndResult := r.client.Retrieve(ctx, epochEnd, r.client.GetForcedInclusionNamespace())
	if epochEndResult.Code == datypes.StatusHeightFromFuture {
		r.logger.Debug().
			Uint64("epoch_end", epochEnd).
			Msg("epoch end height not yet available on DA - backoff required")
		return nil, fmt.Errorf("%w: epoch end height %d not yet available", datypes.ErrHeightFromFuture, epochEnd)
	}

	epochStartResult := epochEndResult
	if epochStart != epochEnd {
		epochStartResult = r.client.Retrieve(ctx, epochStart, r.client.GetForcedInclusionNamespace())
		if epochStartResult.Code == datypes.StatusHeightFromFuture {
			r.logger.Debug().
				Uint64("epoch_start", epochStart).
				Msg("epoch start height not yet available on DA - backoff required")
			return nil, fmt.Errorf("%w: epoch start height %d not yet available", datypes.ErrHeightFromFuture, epochStart)
		}
	}

	r.logger.Debug().
		Uint64("da_height", daHeight).
		Uint64("epoch_start", epochStart).
		Uint64("epoch_end", epochEnd).
		Uint64("epoch_num", currentEpochNumber).
		Msg("retrieving forced included transactions from DA")

	var processErrs error
	err = r.processForcedInclusionBlobs(event, epochStartResult, epochStart)
	processErrs = errors.Join(processErrs, err)

	// Process heights between start and end (exclusive)
	for epochHeight := epochStart + 1; epochHeight < epochEnd; epochHeight++ {
		result := r.client.Retrieve(ctx, epochHeight, r.client.GetForcedInclusionNamespace())

		err = r.processForcedInclusionBlobs(event, result, epochHeight)
		processErrs = errors.Join(processErrs, err)
	}

	// Process epoch end (only if different from start)
	if epochEnd != epochStart {
		err = r.processForcedInclusionBlobs(event, epochEndResult, epochEnd)
		processErrs = errors.Join(processErrs, err)
	}

	// any error during process, need to retry at next call
	if processErrs != nil {
		r.logger.Warn().
			Uint64("da_height", daHeight).
			Uint64("epoch_start", epochStart).
			Uint64("epoch_end", epochEnd).
			Uint64("epoch_num", currentEpochNumber).
			Err(processErrs).
			Msg("Failed to retrieve DA epoch.. retrying next iteration")

		return &ForcedInclusionEvent{
			StartDaHeight: daHeight,
			EndDaHeight:   daHeight,
			Txs:           [][]byte{},
		}, nil
	}

	return event, nil
}

// processForcedInclusionBlobs processes blobs from a single DA height for forced inclusion.
func (r *ForcedInclusionRetriever) processForcedInclusionBlobs(
	event *ForcedInclusionEvent,
	result datypes.ResultRetrieve,
	height uint64,
) error {
	if result.Code == datypes.StatusNotFound {
		r.logger.Debug().Uint64("height", height).Msg("no forced inclusion blobs at height")
		return nil
	}

	if result.Code != datypes.StatusSuccess {
		return fmt.Errorf("failed to retrieve forced inclusion blobs at height %d: %s", height, result.Message)
	}

	// Process each blob as a transaction
	for _, blob := range result.Data {
		if len(blob) > 0 {
			event.Txs = append(event.Txs, blob)
		}
	}

	if result.Timestamp.After(event.Timestamp) {
		event.Timestamp = result.Timestamp
	}

	r.logger.Debug().
		Uint64("height", height).
		Int("blob_count", len(result.Data)).
		Msg("processed forced inclusion blobs")

	return nil
}
