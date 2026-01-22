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
	"github.com/evstack/ev-node/core/execution"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/config"
	datypes "github.com/evstack/ev-node/pkg/da/types"
	"github.com/evstack/ev-node/pkg/genesis"
	seqcommon "github.com/evstack/ev-node/pkg/sequencers/common"
)

var _ coresequencer.Sequencer = (*BasedSequencer)(nil)

// BasedSequencer is a sequencer that only retrieves transactions from the DA layer
// via the forced inclusion mechanism. It does not accept transactions from the reaper.
// It uses DA as a queue and only persists a checkpoint of where it is in processing.
type BasedSequencer struct {
	logger zerolog.Logger

	fiRetriever     block.ForcedInclusionRetriever
	daHeight        atomic.Uint64
	checkpointStore *seqcommon.CheckpointStore
	checkpoint      *seqcommon.Checkpoint
	executor        execution.Executor

	// Cached transactions from the current DA block being processed
	currentBatchTxs [][]byte
	// DA epoch end time for timestamp calculation
	currentDAEndTime time.Time
}

// NewBasedSequencer creates a new based sequencer instance
func NewBasedSequencer(
	daClient block.FullDAClient,
	cfg config.Config,
	db ds.Batching,
	genesis genesis.Genesis,
	logger zerolog.Logger,
	executor execution.Executor,
) (*BasedSequencer, error) {
	bs := &BasedSequencer{
		logger:          logger.With().Str("component", "based_sequencer").Logger(),
		checkpointStore: seqcommon.NewCheckpointStore(db, ds.NewKey("/based/checkpoint")),
		executor:        executor,
	}
	// based sequencers need community consensus about the da start height given no submission are done
	bs.SetDAHeight(genesis.DAStartHeight)

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

	bs.fiRetriever = block.NewForcedInclusionRetriever(daClient, cfg, logger, genesis.DAStartHeight, genesis.DAEpochForcedInclusion)

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

	// If we have no cached transactions or we've consumed all from the current DA epoch,
	// fetch the next DA epoch
	if daHeight > 0 && (len(s.currentBatchTxs) == 0 || s.checkpoint.TxIndex >= uint64(len(s.currentBatchTxs))) {
		daEndTime, daEndHeight, err := s.fetchNextDAEpoch(ctx, req.MaxBytes)
		if err != nil {
			return nil, err
		}
		daHeight = daEndHeight
		s.currentDAEndTime = daEndTime
	}

	// Get remaining transactions from checkpoint position
	// TxIndex tracks how many txs from the current epoch have been consumed (OK or Remove)
	var batchTxs [][]byte
	if len(s.currentBatchTxs) > 0 && s.checkpoint.TxIndex < uint64(len(s.currentBatchTxs)) {
		batchTxs = s.currentBatchTxs[s.checkpoint.TxIndex:]
	}

	// Get gas limit and filter transactions
	// All txs in based sequencer are force-included
	maxGas := s.getMaxGas(ctx)

	filterStatuses, err := s.executor.FilterTxs(ctx, batchTxs, req.MaxBytes, maxGas, true)
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to filter transactions, proceeding with unfiltered")
		// Default: all OK
		filterStatuses = make([]execution.FilterStatus, len(batchTxs))
		for i := range filterStatuses {
			filterStatuses[i] = execution.FilterOK
		}
	}

	// Process filter results sequentially to maintain ordering for forced inclusion.
	// TxIndex tracks consumed txs from the start of the epoch, so we must process in order.
	var validTxs [][]byte
	var consumedCount uint64
	for i, status := range filterStatuses {
		switch status {
		case execution.FilterOK:
			validTxs = append(validTxs, batchTxs[i])
			consumedCount++
		case execution.FilterRemove:
			// Skip removed transactions but count as consumed
			consumedCount++
		case execution.FilterPostpone:
			// Stop processing at first postpone to maintain order
			// Remaining txs (including this one) will be processed in next batch
			goto doneProcessing
		}
	}
doneProcessing:

	// Update checkpoint based on consumed transactions
	if daHeight > 0 || len(batchTxs) > 0 {
		s.checkpoint.TxIndex += consumedCount

		// If we've consumed all transactions from this DA epoch, move to next DA epoch
		if s.checkpoint.TxIndex >= uint64(len(s.currentBatchTxs)) {
			s.checkpoint.DAHeight = daHeight + 1
			s.checkpoint.TxIndex = 0
			s.currentBatchTxs = nil
			s.SetDAHeight(s.checkpoint.DAHeight)
		}

		if err := s.checkpointStore.Save(ctx, s.checkpoint); err != nil {
			return nil, fmt.Errorf("failed to save checkpoint: %w", err)
		}

		s.logger.Debug().
			Uint64("consumed_count", consumedCount).
			Uint64("checkpoint_tx_index", s.checkpoint.TxIndex).
			Uint64("checkpoint_da_height", s.checkpoint.DAHeight).
			Msg("updated checkpoint after processing batch")
	}

	// Calculate timestamp based on remaining transactions after this batch
	// timestamp corresponds to the last block time of a DA epoch, based on the remaining transactions to be executed
	// this is done in order to handle the case where a DA epoch must fit in multiple blocks
	var remainingTxs uint64
	if len(s.currentBatchTxs) > 0 {
		remainingTxs = uint64(len(s.currentBatchTxs)) - s.checkpoint.TxIndex
	}
	timestamp := s.currentDAEndTime.Add(-time.Duration(remainingTxs) * time.Millisecond)

	return &coresequencer.GetNextBatchResponse{
		Batch:     &coresequencer.Batch{Transactions: validTxs},
		Timestamp: timestamp,
		BatchData: req.LastBatchData,
	}, nil
}

// getMaxGas retrieves the gas limit from the execution layer.
func (s *BasedSequencer) getMaxGas(ctx context.Context) uint64 {
	info, err := s.executor.GetExecutionInfo(ctx)
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to get execution info, proceeding without gas limit")
		return 0
	}
	return info.MaxGas
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

	// Validate and filter transactions by size
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
