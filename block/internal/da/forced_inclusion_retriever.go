package da

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	datypes "github.com/evstack/ev-node/pkg/da/types"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/types"
)

// ErrForceInclusionNotConfigured is returned when the forced inclusion namespace is not configured.
var ErrForceInclusionNotConfigured = errors.New("forced inclusion namespace not configured")

// ForcedInclusionRetriever handles retrieval of forced inclusion transactions from DA.
type ForcedInclusionRetriever struct {
	client      Interface
	genesis     genesis.Genesis
	logger      zerolog.Logger
	daEpochSize uint64
}

// ForcedInclusionEvent contains forced inclusion transactions retrieved from DA.
type ForcedInclusionEvent struct {
	Timestamp     time.Time
	StartDaHeight uint64
	EndDaHeight   uint64
	Txs           [][]byte
}

// NewForcedInclusionRetriever creates a new forced inclusion retriever.
func NewForcedInclusionRetriever(
	client Interface,
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

	event := &ForcedInclusionEvent{
		StartDaHeight: epochStart,
		EndDaHeight:   epochEnd,
		Txs:           [][]byte{},
	}

	epochEndResult := r.client.RetrieveForcedInclusion(ctx, epochEnd)
	if epochEndResult.Code == datypes.StatusHeightFromFuture {
		r.logger.Debug().
			Uint64("epoch_end", epochEnd).
			Msg("epoch end height not yet available on DA - backoff required")
		return nil, fmt.Errorf("%w: epoch end height %d not yet available", datypes.ErrHeightFromFuture, epochEnd)
	}

	epochStartResult := epochEndResult
	if epochStart != epochEnd {
		epochStartResult = r.client.RetrieveForcedInclusion(ctx, epochStart)
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
	err := r.processForcedInclusionBlobs(event, epochStartResult, epochStart)
	processErrs = errors.Join(processErrs, err)

	// Process heights between start and end (exclusive)
	for epochHeight := epochStart + 1; epochHeight < epochEnd; epochHeight++ {
		result := r.client.RetrieveForcedInclusion(ctx, epochHeight)

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
