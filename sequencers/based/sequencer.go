package based

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	ds "github.com/ipfs/go-datastore"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	datypes "github.com/evstack/ev-node/pkg/da/types"
	"github.com/evstack/ev-node/pkg/genesis"
	seqcommon "github.com/evstack/ev-node/sequencers/common"
)

// ForcedInclusionRetriever defines the interface for retrieving forced inclusion transactions from DA
type ForcedInclusionRetriever interface {
	RetrieveForcedIncludedTxs(ctx context.Context, daHeight uint64) (*block.ForcedInclusionEvent, error)
}

var _ coresequencer.Sequencer = (*BasedSequencer)(nil)

// BasedSequencer is a sequencer that only retrieves transactions from the DA layer
// via the forced inclusion mechanism. It does not accept transactions from the reaper.
// It uses DA as a queue and only persists a checkpoint of where it is in processing.
type BasedSequencer struct {
	fiRetriever ForcedInclusionRetriever
	logger      zerolog.Logger

	daHeight        atomic.Uint64
	checkpointStore *seqcommon.CheckpointStore
	checkpoint      *seqcommon.Checkpoint

	// Cached transactions from the current DA block being processed
	currentBatchTxs [][]byte
	// DA epoch end time for timestamp calculation
	currentDAEndTime time.Time
}

// NewBasedSequencer creates a new based sequencer instance
func NewBasedSequencer(
	ctx context.Context,
	fiRetriever ForcedInclusionRetriever,
	db ds.Batching,
	genesis genesis.Genesis,
	logger zerolog.Logger,
) (*BasedSequencer, error) {
	bs := &BasedSequencer{
		fiRetriever:     fiRetriever,
		logger:          logger.With().Str("component", "based_sequencer").Logger(),
		checkpointStore: seqcommon.NewCheckpointStore(db, ds.NewKey("/based/checkpoint")),
	}
	bs.SetDAHeight(genesis.DAStartHeight) // based sequencers need community consensus about the da start height given no submission are done

	// Load checkpoint from DB, or initialize if none exists
	loadCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	checkpoint, err := bs.checkpointStore.Load(loadCtx)
	if err != nil {
		if errors.Is(err, seqcommon.ErrCheckpointNotFound) {
			// No checkpoint exists, initialize with current DA height
			bs.checkpoint = &seqcommon.Checkpoint{
				DAHeight: bs.GetDAHeight(),
				TxIndex:  0,
			}
		} else {
			return nil, fmt.Errorf("failed to load checkpoint from DB: %w", err)
		}
	} else {
		bs.checkpoint = checkpoint
		// If we had a non-zero tx index, we're resuming from a crash mid-block
		// The transactions starting from that index are what we need
		if checkpoint.TxIndex > 0 {
			bs.logger.Debug().
				Uint64("tx_index", checkpoint.TxIndex).
				Uint64("da_height", checkpoint.DAHeight).
				Msg("resuming from checkpoint within DA epoch")
		}
	}

	return bs, nil
}

// SubmitBatchTxs does nothing for a based sequencer as it only pulls from DA
// This satisfies the Sequencer interface but transactions submitted here are ignored
func (s *BasedSequencer) SubmitBatchTxs(ctx context.Context, req coresequencer.SubmitBatchTxsRequest) (*coresequencer.SubmitBatchTxsResponse, error) {
	s.logger.Debug().Msg("based sequencer ignores submitted transactions - only DA transactions are processed")
	return &coresequencer.SubmitBatchTxsResponse{}, nil
}

// GetNextBatch retrieves the next batch of transactions from the DA layer using the checkpoint
// It treats DA as a queue and only persists where it is in processing
func (s *BasedSequencer) GetNextBatch(ctx context.Context, req coresequencer.GetNextBatchRequest) (*coresequencer.GetNextBatchResponse, error) {
	daHeight := s.GetDAHeight()

	// If we have no cached transactions or we've consumed all from the current DA block,
	// fetch the next DA epoch
	if daHeight > 0 && (len(s.currentBatchTxs) == 0 || s.checkpoint.TxIndex >= uint64(len(s.currentBatchTxs))) {
		daEndTime, daEndHeight, err := s.fetchNextDAEpoch(ctx, req.MaxBytes)
		if err != nil {
			return nil, err
		}

		daHeight = daEndHeight
		s.currentDAEndTime = daEndTime
	}

	// Create batch from current position up to MaxBytes
	batch := s.createBatchFromCheckpoint(req.MaxBytes)

	// Update checkpoint with how many transactions we consumed
	if daHeight > 0 || len(batch.Transactions) > 0 {
		s.checkpoint.TxIndex += uint64(len(batch.Transactions))

		// If we've consumed all transactions from this DA epoch, move to next
		if s.checkpoint.TxIndex >= uint64(len(s.currentBatchTxs)) {
			s.checkpoint.DAHeight = daHeight + 1
			s.checkpoint.TxIndex = 0
			s.currentBatchTxs = nil

			// Update the global DA height
			s.SetDAHeight(s.checkpoint.DAHeight)
		}

		// Persist checkpoint
		if err := s.checkpointStore.Save(ctx, s.checkpoint); err != nil {
			return nil, fmt.Errorf("failed to save checkpoint: %w", err)
		}
	}

	// Calculate timestamp based on remaining transactions after this batch
	// timestamp corresponds to the last block time of a DA epoch, based on the remaining transactions to be executed
	// this is done in order to handle the case where a DA epoch must fit in multiple blocks
	remainingTxs := uint64(len(s.currentBatchTxs)) - s.checkpoint.TxIndex
	timestamp := s.currentDAEndTime.Add(-time.Duration(remainingTxs) * time.Millisecond)

	return &coresequencer.GetNextBatchResponse{
		Batch:     batch,
		Timestamp: timestamp,
		BatchData: req.LastBatchData,
	}, nil
}

// fetchNextDAEpoch fetches transactions from the next DA epoch
func (s *BasedSequencer) fetchNextDAEpoch(ctx context.Context, maxBytes uint64) (time.Time, uint64, error) {
	currentDAHeight := s.checkpoint.DAHeight

	s.logger.Debug().
		Uint64("da_height", currentDAHeight).
		Uint64("tx_index", s.checkpoint.TxIndex).
		Msg("fetching forced inclusion transactions from DA")

	forcedTxsEvent, err := s.fiRetriever.RetrieveForcedIncludedTxs(ctx, currentDAHeight)
	if err != nil {
		// Check if forced inclusion is not configured
		if errors.Is(err, block.ErrForceInclusionNotConfigured) {
			return time.Time{}, 0, block.ErrForceInclusionNotConfigured
		} else if errors.Is(err, datypes.ErrHeightFromFuture) {
			// If we get a height from future error, stay at current position
			// We'll retry the same height on the next call until DA produces that block
			s.logger.Debug().
				Uint64("da_height", currentDAHeight).
				Msg("DA height from future, waiting for DA to produce block")
			return time.Time{}, 0, nil
		}
		return time.Time{}, 0, fmt.Errorf("failed to retrieve forced inclusion transactions: %w", err)
	}

	// Validate and filter transactions
	validTxs := make([][]byte, 0, len(forcedTxsEvent.Txs))
	skippedTxs := 0
	for _, tx := range forcedTxsEvent.Txs {
		// Validate blob size against absolute maximum
		if uint64(len(tx)) > maxBytes {
			s.logger.Warn().
				Uint64("da_height", forcedTxsEvent.StartDaHeight).
				Int("blob_size", len(tx)).
				Uint64("max_bytes", maxBytes).
				Msg("forced inclusion blob exceeds maximum size - skipping")
			skippedTxs++
			continue
		}
		validTxs = append(validTxs, tx)
	}

	s.logger.Info().
		Int("valid_tx_count", len(validTxs)).
		Int("skipped_tx_count", skippedTxs).
		Uint64("da_height_start", forcedTxsEvent.StartDaHeight).
		Uint64("da_height_end", forcedTxsEvent.EndDaHeight).
		Msg("fetched forced inclusion transactions from DA")

	// Cache the transactions for this DA epoch
	s.currentBatchTxs = validTxs

	return forcedTxsEvent.Timestamp.UTC(), forcedTxsEvent.EndDaHeight, nil
}

// createBatchFromCheckpoint creates a batch from the current checkpoint position respecting MaxBytes
func (s *BasedSequencer) createBatchFromCheckpoint(maxBytes uint64) *coresequencer.Batch {
	if len(s.currentBatchTxs) == 0 || s.checkpoint.TxIndex >= uint64(len(s.currentBatchTxs)) {
		return &coresequencer.Batch{Transactions: nil}
	}

	var result [][]byte
	var totalBytes uint64

	// Start from the checkpoint index
	for i := s.checkpoint.TxIndex; i < uint64(len(s.currentBatchTxs)); i++ {
		tx := s.currentBatchTxs[i]
		txSize := uint64(len(tx))

		if totalBytes+txSize > maxBytes {
			break
		}

		result = append(result, tx)
		totalBytes += txSize
	}

	// Mark all transactions as force-included since based sequencer only pulls from DA
	forceIncludedMask := make([]bool, len(result))
	for i := range forceIncludedMask {
		forceIncludedMask[i] = true
	}

	return &coresequencer.Batch{
		Transactions:      result,
		ForceIncludedMask: forceIncludedMask,
	}
}

// VerifyBatch verifies a batch of transactions
// For a based sequencer, we always return true as all transactions come from DA
func (s *BasedSequencer) VerifyBatch(ctx context.Context, req coresequencer.VerifyBatchRequest) (*coresequencer.VerifyBatchResponse, error) {
	return &coresequencer.VerifyBatchResponse{
		Status: true,
	}, nil
}

// SetDAHeight sets the current DA height for the sequencer
// This should be called when the sequencer needs to sync to a specific DA height
func (s *BasedSequencer) SetDAHeight(height uint64) {
	s.daHeight.Store(height)
}

// GetDAHeight returns the current DA height
func (s *BasedSequencer) GetDAHeight() uint64 {
	return s.daHeight.Load()
}
