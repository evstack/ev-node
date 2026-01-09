package da

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/pkg/config"
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
	asyncFetcher  AsyncBlockRetriever
}

// ForcedInclusionEvent contains forced inclusion transactions retrieved from DA.
type ForcedInclusionEvent struct {
	Timestamp     time.Time
	StartDaHeight uint64
	EndDaHeight   uint64
	Txs           [][]byte
}

// NewForcedInclusionRetriever creates a new forced inclusion retriever.
// It internally creates and manages an AsyncBlockRetriever for background prefetching.
func NewForcedInclusionRetriever(
	client Client,
	logger zerolog.Logger,
	cfg config.Config,
	daStartHeight, daEpochSize uint64,
) *ForcedInclusionRetriever {
	retrieverLogger := logger.With().Str("component", "forced_inclusion_retriever").Logger()

	// Create async block retriever for background prefetching
	asyncFetcher := NewAsyncBlockRetriever(
		client,
		logger,
		client.GetForcedInclusionNamespace(),
		cfg,
		daStartHeight,
		daEpochSize*2, // prefetch window: 2x epoch size
	)
	asyncFetcher.Start()

	return &ForcedInclusionRetriever{
		client:        client,
		logger:        retrieverLogger,
		daStartHeight: daStartHeight,
		daEpochSize:   daEpochSize,
		asyncFetcher:  asyncFetcher,
	}
}

// Stop stops the background prefetcher.
func (r *ForcedInclusionRetriever) Stop() {
	r.asyncFetcher.Stop()
}

// RetrieveForcedIncludedTxs retrieves forced inclusion transactions at the given DA height.
// It respects epoch boundaries and only fetches at epoch end.
// It tries to get blocks from the async fetcher cache first, then falls back to sync fetching.
func (r *ForcedInclusionRetriever) RetrieveForcedIncludedTxs(ctx context.Context, daHeight uint64) (*ForcedInclusionEvent, error) {
	// when daStartHeight is not set or no namespace is configured, we retrieve nothing.
	if !r.client.HasForcedInclusionNamespace() {
		return nil, ErrForceInclusionNotConfigured
	}

	if daHeight < r.daStartHeight {
		return nil, fmt.Errorf("DA height %d is before the configured start height %d", daHeight, r.daStartHeight)
	}

	epochStart, epochEnd, currentEpochNumber := types.CalculateEpochBoundaries(daHeight, r.daStartHeight, r.daEpochSize)

	// Update the async fetcher's current height so it knows what to prefetch
	r.asyncFetcher.UpdateCurrentHeight(daHeight)

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

	r.logger.Debug().
		Uint64("da_height", daHeight).
		Uint64("epoch_start", epochStart).
		Uint64("epoch_end", epochEnd).
		Uint64("epoch_num", currentEpochNumber).
		Msg("retrieving forced included transactions from DA epoch")

	event := &ForcedInclusionEvent{
		StartDaHeight: epochStart,
		EndDaHeight:   epochEnd,
		Txs:           [][]byte{},
	}

	// Collect all heights in this epoch
	var heights []uint64
	for h := epochStart; h <= epochEnd; h++ {
		heights = append(heights, h)
	}

	// Try to get blocks from cache first
	cachedBlocks := make(map[uint64]*BlockData)
	var missingHeights []uint64

	for _, h := range heights {
		block, err := r.asyncFetcher.GetCachedBlock(ctx, h)
		if err != nil {
			r.logger.Debug().
				Err(err).
				Uint64("height", h).
				Msg("error getting cached block, will fetch synchronously")
			missingHeights = append(missingHeights, h)
			continue
		}
		if block == nil {
			// Cache miss
			missingHeights = append(missingHeights, h)
		} else {
			// Cache hit
			cachedBlocks[h] = block
			r.logger.Debug().
				Uint64("height", h).
				Int("blob_count", len(block.Blobs)).
				Msg("using cached block from async fetcher")
		}
	}

	// Fetch missing heights synchronously
	var processErrs error
	for _, h := range missingHeights {
		result := r.client.Retrieve(ctx, h, r.client.GetForcedInclusionNamespace())

		if result.Code == datypes.StatusHeightFromFuture {
			r.logger.Debug().
				Uint64("height", h).
				Msg("height not yet available on DA - backoff required")
			return nil, fmt.Errorf("%w: height %d not yet available", datypes.ErrHeightFromFuture, h)
		}

		err := r.processRetrieveResult(event, result, h)
		processErrs = errors.Join(processErrs, err)
	}

	// Process cached blocks in order
	for _, h := range heights {
		if block, ok := cachedBlocks[h]; ok {
			// Add blobs from cached block
			for _, blob := range block.Blobs {
				if len(blob) > 0 {
					event.Txs = append(event.Txs, blob)
				}
			}

			// Update timestamp if newer
			if block.Timestamp.After(event.Timestamp) {
				event.Timestamp = block.Timestamp
			}

			r.logger.Debug().
				Uint64("height", h).
				Int("blob_count", len(block.Blobs)).
				Msg("added blobs from cached block")
		}
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

	r.logger.Info().
		Uint64("da_height", daHeight).
		Uint64("epoch_start", epochStart).
		Uint64("epoch_end", epochEnd).
		Int("tx_count", len(event.Txs)).
		Int("cached_blocks", len(cachedBlocks)).
		Int("sync_fetched_blocks", len(missingHeights)).
		Msg("successfully retrieved forced inclusion epoch")

	return event, nil
}

// processRetrieveResult processes the result from a DA retrieve operation.
func (r *ForcedInclusionRetriever) processRetrieveResult(
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
