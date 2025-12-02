package da

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	coreda "github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/types"
)

// ErrForceInclusionNotConfigured is returned when the forced inclusion namespace is not configured.
var ErrForceInclusionNotConfigured = errors.New("forced inclusion namespace not configured")

// ForcedInclusionRetriever handles retrieval of forced inclusion transactions from DA.
type ForcedInclusionRetriever struct {
	client      Client
	genesis     genesis.Genesis
	logger      zerolog.Logger
	daEpochSize uint64
}

// ForcedInclusionEvent contains forced inclusion transactions retrieved from DA.
type ForcedInclusionEvent struct {
	StartDaHeight uint64
	EndDaHeight   uint64
	Txs           [][]byte
}

// NewForcedInclusionRetriever creates a new forced inclusion retriever.
func NewForcedInclusionRetriever(
	client Client,
	genesis genesis.Genesis,
	logger zerolog.Logger,
) *ForcedInclusionRetriever {
	return &ForcedInclusionRetriever{
		client:      client,
		genesis:     genesis,
		logger:      logger.With().Str("component", "forced_inclusion_retriever").Logger(),
		daEpochSize: genesis.DAEpochForcedInclusion,
	}
}

// RetrieveForcedIncludedTxs retrieves forced inclusion transactions at the given DA height.
// It respects epoch boundaries and only fetches at epoch start.
func (r *ForcedInclusionRetriever) RetrieveForcedIncludedTxs(ctx context.Context, daHeight uint64) (*ForcedInclusionEvent, error) {
	if !r.client.HasForcedInclusionNamespace() {
		return nil, ErrForceInclusionNotConfigured
	}

	epochStart, epochEnd, currentEpochNumber := types.CalculateEpochBoundaries(daHeight, r.genesis.DAStartHeight, r.daEpochSize)

	if daHeight != epochStart {
		r.logger.Debug().
			Uint64("da_height", daHeight).
			Uint64("epoch_start", epochStart).
			Msg("not at epoch start - returning empty transactions")

		return &ForcedInclusionEvent{
			StartDaHeight: daHeight,
			EndDaHeight:   daHeight,
			Txs:           [][]byte{},
		}, nil
	}

	event := &ForcedInclusionEvent{
		StartDaHeight: epochStart,
		Txs:           [][]byte{},
	}

	r.logger.Debug().
		Uint64("da_height", daHeight).
		Uint64("epoch_start", epochStart).
		Uint64("epoch_end", epochEnd).
		Uint64("epoch_num", currentEpochNumber).
		Msg("retrieving forced included transactions from DA")

	epochEndResult := r.client.RetrieveForcedInclusion(ctx, epochEnd)
	if epochEndResult.Code == coreda.StatusHeightFromFuture {
		r.logger.Debug().
			Uint64("epoch_end", epochEnd).
			Msg("epoch end height not yet available on DA - backoff required")
		return nil, fmt.Errorf("%w: epoch end height %d not yet available", coreda.ErrHeightFromFuture, epochEnd)
	}

	epochStartResult := epochEndResult
	if epochStart != epochEnd {
		epochStartResult = r.client.RetrieveForcedInclusion(ctx, epochStart)
		if epochStartResult.Code == coreda.StatusHeightFromFuture {
			r.logger.Debug().
				Uint64("epoch_start", epochStart).
				Msg("epoch start height not yet available on DA - backoff required")
			return nil, fmt.Errorf("%w: epoch start height %d not yet available", coreda.ErrHeightFromFuture, epochStart)
		}
	}

	lastProcessedHeight := epochStart

	if err := r.processForcedInclusionBlobs(event, &lastProcessedHeight, epochStartResult, epochStart); err != nil {
		return nil, err
	}

	// Process heights between start and end (exclusive)
	for epochHeight := epochStart + 1; epochHeight < epochEnd; epochHeight++ {
		result := r.client.RetrieveForcedInclusion(ctx, epochHeight)

		// If any intermediate height is from future, break early
		if result.Code == coreda.StatusHeightFromFuture {
			r.logger.Debug().
				Uint64("epoch_height", epochHeight).
				Uint64("last_processed", lastProcessedHeight).
				Msg("reached future DA height within epoch - stopping")
			break
		}

		if err := r.processForcedInclusionBlobs(event, &lastProcessedHeight, result, epochHeight); err != nil {
			return nil, err
		}
	}

	// Process epoch end (only if different from start)
	if epochEnd != epochStart {
		if err := r.processForcedInclusionBlobs(event, &lastProcessedHeight, epochEndResult, epochEnd); err != nil {
			return nil, err
		}
	}

	event.EndDaHeight = lastProcessedHeight

	r.logger.Info().
		Uint64("epoch_start", epochStart).
		Uint64("epoch_end", lastProcessedHeight).
		Int("tx_count", len(event.Txs)).
		Msg("retrieved forced inclusion transactions")

	return event, nil
}

// processForcedInclusionBlobs processes blobs from a single DA height for forced inclusion.
func (r *ForcedInclusionRetriever) processForcedInclusionBlobs(
	event *ForcedInclusionEvent,
	lastProcessedHeight *uint64,
	result coreda.ResultRetrieve,
	height uint64,
) error {
	if result.Code == coreda.StatusNotFound {
		r.logger.Debug().Uint64("height", height).Msg("no forced inclusion blobs at height")
		*lastProcessedHeight = height
		return nil
	}

	if result.Code != coreda.StatusSuccess {
		return fmt.Errorf("failed to retrieve forced inclusion blobs at height %d: %s", height, result.Message)
	}

	// Process each blob as a transaction
	for _, blob := range result.Data {
		if len(blob) > 0 {
			event.Txs = append(event.Txs, blob)
		}
	}

	*lastProcessedHeight = height

	r.logger.Debug().
		Uint64("height", height).
		Int("blob_count", len(result.Data)).
		Msg("processed forced inclusion blobs")

	return nil
}
