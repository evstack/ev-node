package da

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/pkg/config"
	datypes "github.com/evstack/ev-node/pkg/da/types"
	"github.com/evstack/ev-node/types"
)

// ErrForceInclusionNotConfigured is returned when the forced inclusion namespace is not configured.
var ErrForceInclusionNotConfigured = errors.New("forced inclusion namespace not configured")

const (
	// subscriptionCatchupThreshold matches syncing catchupThreshold for enabling subscription.
	subscriptionCatchupThreshold = 2
)

// ForcedInclusionRetriever defines the interface for retrieving forced inclusion transactions from DA.
type ForcedInclusionRetriever interface {
	RetrieveForcedIncludedTxs(ctx context.Context, daHeight uint64) (*ForcedInclusionEvent, error)
	Stop()
}

// forcedInclusionRetriever handles retrieval of forced inclusion transactions from DA.
type forcedInclusionRetriever struct {
	client        Client
	logger        zerolog.Logger
	daEpochSize   uint64
	daStartHeight uint64
	asyncFetcher  AsyncBlockRetriever

	mu                    sync.RWMutex
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

func emptyForcedInclusionEvent(daHeight uint64) *ForcedInclusionEvent {
	return &ForcedInclusionEvent{
		StartDaHeight: daHeight,
		EndDaHeight:   daHeight,
		Txs:           [][]byte{},
	}
}

// NewForcedInclusionRetriever creates a new forced inclusion retriever.
// It internally creates and manages an AsyncBlockRetriever for background prefetching.
func NewForcedInclusionRetriever(
	client Client,
	logger zerolog.Logger,
	cfg config.Config,
	daStartHeight, daEpochSize uint64,
) ForcedInclusionRetriever {
	retrieverLogger := logger.With().Str("component", "forced_inclusion_retriever").Logger()

	// Create async block retriever for background prefetching.
	asyncFetcher := NewAsyncBlockRetriever(
		client,
		logger,
		client.GetForcedInclusionNamespace(),
		cfg,
		daStartHeight,
		daEpochSize*2, // prefetch window: 2x epoch size
	)
	asyncFetcher.Start()

	base := &forcedInclusionRetriever{
		client:        client,
		logger:        retrieverLogger,
		daStartHeight: daStartHeight,
		daEpochSize:   daEpochSize,
		asyncFetcher:  asyncFetcher,
	}
	if cfg.Instrumentation.IsTracingEnabled() {
		return withTracingForcedInclusionRetriever(base)
	}
	return base
}

// Stop stops the background prefetcher.
func (r *forcedInclusionRetriever) Stop() {
	r.asyncFetcher.Stop()
}

// RetrieveForcedIncludedTxs retrieves forced inclusion transactions at the given DA height.
// It respects epoch boundaries and only fetches at epoch end.
// It tries to get blocks from the async fetcher cache first, then falls back to sync fetching.
func (r *forcedInclusionRetriever) RetrieveForcedIncludedTxs(ctx context.Context, daHeight uint64) (*ForcedInclusionEvent, error) {
	// when daStartHeight is not set or no namespace is configured, we retrieve nothing.
	if !r.client.HasForcedInclusionNamespace() {
		return nil, ErrForceInclusionNotConfigured
	}

	if daHeight < r.daStartHeight {
		return nil, fmt.Errorf("DA height %d is before the configured start height %d", daHeight, r.daStartHeight)
	}

	epochStart, epochEnd, currentEpochNumber := types.CalculateEpochBoundaries(daHeight, r.daStartHeight, r.daEpochSize)
	r.asyncFetcher.UpdateCurrentHeight(daHeight)
	r.maybeEnableSubscription(ctx, daHeight)

	r.mu.RLock()
	lastProcessed := r.lastProcessedEpochEnd
	hasProcessed := r.hasProcessedEpoch
	r.mu.RUnlock()

	if hasProcessed {
		if r.daEpochSize == 0 {
			return emptyForcedInclusionEvent(daHeight), nil
		}
		epochStart = lastProcessed + 1
		epochEnd = epochStart + r.daEpochSize - 1
		currentEpochNumber = types.CalculateEpochNumber(epochEnd, r.daStartHeight, r.daEpochSize)
	}

	if daHeight < epochEnd {
		r.logger.Debug().
			Uint64("da_height", daHeight).
			Uint64("epoch_end", epochEnd).
			Msg("not at epoch end - returning empty transactions")
		return emptyForcedInclusionEvent(daHeight), nil
	}

	r.logger.Debug().
		Uint64("da_height", daHeight).
		Uint64("epoch_start", epochStart).
		Uint64("epoch_end", epochEnd).
		Uint64("epoch_num", currentEpochNumber).
		Msg("retrieving forced included transactions from DA epoch")

	event, err := r.retrieveEpoch(ctx, epochStart, epochEnd)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	r.lastProcessedEpochEnd = epochEnd
	r.hasProcessedEpoch = true
	r.mu.Unlock()

	return event, nil
}

func (r *forcedInclusionRetriever) maybeEnableSubscription(ctx context.Context, daHeight uint64) {
	daNodeHead, err := r.client.LocalHead(ctx)
	if err != nil {
		r.logger.Debug().Err(err).Msg("failed to get DA node head for subscription enablement")
		return
	}

	if daHeight+subscriptionCatchupThreshold >= daNodeHead {
		r.asyncFetcher.StartSubscription()
	}
}

func (r *forcedInclusionRetriever) retrieveEpoch(ctx context.Context, epochStart, epochEnd uint64) (*ForcedInclusionEvent, error) {
	epochSize := epochEnd - epochStart + 1
	blocks := make(map[uint64]*BlockData, epochSize)
	var missingHeights []uint64

	// Check cache for each height in the epoch
	for h := epochStart; h <= epochEnd; h++ {
		block, err := r.asyncFetcher.GetCachedBlock(ctx, h)
		if err != nil {
			r.logger.Debug().Err(err).Uint64("height", h).Msg("cache error, will fetch synchronously")
			missingHeights = append(missingHeights, h)
			continue
		}
		if block == nil {
			missingHeights = append(missingHeights, h)
		} else {
			blocks[h] = block
		}
	}

	// Fetch missing heights synchronously
	var processErrs error
	namespace := r.client.GetForcedInclusionNamespace()
	for _, h := range missingHeights {
		result := r.client.Retrieve(ctx, h, namespace)

		switch result.Code {
		case datypes.StatusHeightFromFuture:
			r.logger.Debug().Uint64("height", h).Msg("height not yet available on DA")
			return nil, fmt.Errorf("%w: height %d not yet available", datypes.ErrHeightFromFuture, h)

		case datypes.StatusNotFound:
			r.logger.Debug().Uint64("height", h).Msg("no forced inclusion blobs at height")

		case datypes.StatusSuccess:
			blocks[h] = &BlockData{
				Blobs:     result.Data,
				Timestamp: result.Timestamp,
			}

		default:
			processErrs = errors.Join(processErrs, fmt.Errorf("failed to retrieve at height %d: %s", h, result.Message))
		}
	}

	if processErrs != nil {
		r.logger.Warn().
			Uint64("epoch_start", epochStart).
			Uint64("epoch_end", epochEnd).
			Err(processErrs).
			Msg("failed to retrieve DA epoch")
		return nil, processErrs
	}

	// Aggregate blobs in height order
	event := &ForcedInclusionEvent{
		StartDaHeight: epochStart,
		EndDaHeight:   epochEnd,
		Txs:           [][]byte{},
	}

	for h := epochStart; h <= epochEnd; h++ {
		block, ok := blocks[h]
		if !ok {
			continue
		}

		for _, blob := range block.Blobs {
			if len(blob) > 0 {
				event.Txs = append(event.Txs, blob)
			}
		}

		if block.Timestamp.After(event.Timestamp) {
			event.Timestamp = block.Timestamp
		}

		r.logger.Debug().
			Uint64("height", h).
			Int("blob_count", len(block.Blobs)).
			Msg("added blobs from block")
	}

	return event, nil
}
