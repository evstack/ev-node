package da

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/config"
	blobrpc "github.com/evstack/ev-node/pkg/da/jsonrpc"
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

	mu                    sync.Mutex
	pendingEpochStart     uint64
	pendingEpochEnd       uint64
	lastProcessedEpochEnd uint64
	hasProcessedEpoch     bool
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

// HandleSubscriptionResponse caches forced inclusion blobs from subscription updates.
func (r *ForcedInclusionRetriever) HandleSubscriptionResponse(resp *blobrpc.SubscriptionResponse) {
	if resp == nil {
		return
	}
	if !r.client.HasForcedInclusionNamespace() {
		return
	}

	r.asyncFetcher.UpdateCurrentHeight(resp.Height)

	blobs := common.BlobsFromSubscription(resp)
	if len(blobs) == 0 {
		return
	}

	r.asyncFetcher.StoreBlock(context.Background(), resp.Height, blobs, time.Now().UTC())
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

	r.mu.Lock()
	pendingStart := r.pendingEpochStart
	pendingEnd := r.pendingEpochEnd
	lastProcessed := r.lastProcessedEpochEnd
	hasProcessed := r.hasProcessedEpoch
	r.mu.Unlock()

	if pendingEnd != 0 {
		if daHeight < pendingEnd {
			return &ForcedInclusionEvent{
				StartDaHeight: daHeight,
				EndDaHeight:   daHeight,
				Txs:           [][]byte{},
			}, nil
		}

		event, err := r.retrieveEpoch(ctx, pendingStart, pendingEnd)
		if err != nil {
			return nil, err
		}

		r.mu.Lock()
		r.pendingEpochStart = 0
		r.pendingEpochEnd = 0
		r.lastProcessedEpochEnd = pendingEnd
		r.hasProcessedEpoch = true
		r.mu.Unlock()

		return event, nil
	}

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

	if hasProcessed && epochEnd <= lastProcessed {
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

	event, err := r.retrieveEpoch(ctx, epochStart, epochEnd)
	if err != nil {
		r.mu.Lock()
		r.pendingEpochStart = epochStart
		r.pendingEpochEnd = epochEnd
		r.mu.Unlock()
		return nil, err
	}

	r.mu.Lock()
	r.lastProcessedEpochEnd = epochEnd
	r.hasProcessedEpoch = true
	r.mu.Unlock()

	return event, nil
}

func (r *ForcedInclusionRetriever) retrieveEpoch(ctx context.Context, epochStart, epochEnd uint64) (*ForcedInclusionEvent, error) {
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
		if block == nil { // Cache miss
			missingHeights = append(missingHeights, h)
		} else { // Cache hit
			cachedBlocks[h] = block
		}
	}

	// Fetch missing heights synchronously and store in map
	syncFetchedBlocks := make(map[uint64]*BlockData)
	var processErrs error
	for _, h := range missingHeights {
		result := r.client.Retrieve(ctx, h, r.client.GetForcedInclusionNamespace())
		if result.Code == datypes.StatusHeightFromFuture {
			r.logger.Debug().
				Uint64("height", h).
				Msg("height not yet available on DA - backoff required")
			return nil, fmt.Errorf("%w: height %d not yet available", datypes.ErrHeightFromFuture, h)
		}

		if result.Code == datypes.StatusNotFound {
			r.logger.Debug().Uint64("height", h).Msg("no forced inclusion blobs at height")
			continue
		}

		if result.Code != datypes.StatusSuccess {
			err := fmt.Errorf("failed to retrieve forced inclusion blobs at height %d: %s", h, result.Message)
			processErrs = errors.Join(processErrs, err)
			continue
		}

		// Store the sync-fetched block data
		syncFetchedBlocks[h] = &BlockData{
			Blobs:     result.Data,
			Timestamp: result.Timestamp,
		}
	}

	// Process all blocks in height order
	for _, h := range heights {
		var block *BlockData
		var source string

		// Check cached blocks first, then sync-fetched
		if cachedBlock, ok := cachedBlocks[h]; ok {
			block = cachedBlock
			source = "cache"
		} else if syncBlock, ok := syncFetchedBlocks[h]; ok {
			block = syncBlock
			source = "sync"
		}

		if block != nil {
			// Add blobs from block
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
				Str("source", source).
				Msg("added blobs from block")
		}

		// Clean up maps to prevent unbounded memory growth
		delete(cachedBlocks, h)
		delete(syncFetchedBlocks, h)
	}

	// any error during process, need to retry at next call
	if processErrs != nil {
		r.logger.Warn().
			Uint64("epoch_start", epochStart).
			Uint64("epoch_end", epochEnd).
			Err(processErrs).
			Msg("failed to retrieve DA epoch")

		return nil, processErrs
	}

	return event, nil
}
