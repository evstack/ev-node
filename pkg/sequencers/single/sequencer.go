// Package single implements a single sequencer.
package single

import (
	"bytes"
	"context"
	"encoding/binary"
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
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

// ErrInvalidId is returned when the chain id is invalid
var ErrInvalidId = errors.New("invalid chain id")

// Catch-up state machine states
const (
	catchUpUnchecked  int32 = iota // haven't checked DA height
	catchUpInProgress              // replaying missed DA epochs
	catchUpDone                    // caught up or never behind
)

var _ coresequencer.Sequencer = (*Sequencer)(nil)

// Sequencer implements core sequencing interface
type Sequencer struct {
	logger  zerolog.Logger
	genesis genesis.Genesis
	db      ds.Batching
	cfg     config.Config

	Id       []byte
	daClient block.FullDAClient

	batchTime time.Duration
	queue     *BatchQueue // single queue for immediate availability

	// Forced inclusion support
	fiRetriever     block.ForcedInclusionRetriever
	daHeight        atomic.Uint64
	daStartHeight   atomic.Uint64
	checkpointStore *seqcommon.CheckpointStore
	checkpoint      *seqcommon.Checkpoint
	executor        execution.Executor

	// Cached forced inclusion transactions from the current epoch
	cachedForcedInclusionTxs [][]byte

	// catchUpState tracks catch-up lifecycle (see constants above)
	catchUpState atomic.Int32
	// currentDAEndTime is the DA epoch end timestamp, used during catch-up
	currentDAEndTime time.Time
	// currentEpochTxCount is the total number of txs in the current DA epoch (used for timestamp jitter)
	currentEpochTxCount uint64
	// lastCatchUpTimestamp is the floor for catch-up timestamps to guarantee
	// monotonicity after a restart.  Initialised from the last block time in
	// the store when catch-up mode is entered.
	lastCatchUpTimestamp time.Time
}

// NewSequencer creates a new Single Sequencer
func NewSequencer(
	logger zerolog.Logger,
	db ds.Batching,
	daClient block.FullDAClient,
	cfg config.Config,
	id []byte,
	maxQueueSize int,
	genesis genesis.Genesis,
	executor execution.Executor,
) (*Sequencer, error) {
	s := &Sequencer{
		db:               db,
		logger:           logger,
		daClient:         daClient,
		cfg:              cfg,
		batchTime:        cfg.Node.BlockTime.Duration,
		Id:               id,
		queue:            NewBatchQueue(db, "batches", maxQueueSize),
		checkpointStore:  seqcommon.NewCheckpointStore(db, ds.NewKey("/single/checkpoint")),
		genesis:          genesis,
		currentDAEndTime: genesis.StartTime,
		executor:         executor,
	}
	s.SetDAHeight(genesis.DAStartHeight) // default value, will be overridden by executor or submitter
	s.daStartHeight.Store(genesis.DAStartHeight)

	loadCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Load batch queue from DB
	if err := s.queue.Load(loadCtx); err != nil {
		return nil, fmt.Errorf("failed to load batch queue from DB: %w", err)
	}

	// Load checkpoint from DB or initialize
	checkpoint, err := s.checkpointStore.Load(loadCtx)
	if err != nil {
		if errors.Is(err, seqcommon.ErrCheckpointNotFound) {
			// No checkpoint exists, initialize with current DA height
			s.checkpoint = &seqcommon.Checkpoint{
				DAHeight: s.GetDAHeight(),
				TxIndex:  0,
			}
		} else {
			return nil, fmt.Errorf("failed to load checkpoint from DB: %w", err)
		}
	} else {
		s.checkpoint = checkpoint
		// If we had a non-zero tx index, we're resuming from a crash mid-block
		// The transactions starting from that index are what we need
		if checkpoint.TxIndex > 0 {
			s.logger.Debug().
				Uint64("tx_index", checkpoint.TxIndex).
				Uint64("da_height", checkpoint.DAHeight).
				Msg("resuming from checkpoint within DA epoch")
		}
	}

	// Determine initial DA height for forced inclusion
	initialDAHeight := s.getInitialDAStartHeight(context.Background())

	s.fiRetriever = block.NewForcedInclusionRetriever(daClient, cfg, logger, initialDAHeight, genesis.DAEpochForcedInclusion)

	return s, nil
}

// getInitialDAStartHeight retrieves the DA height of the first included chain height from store.
func (c *Sequencer) getInitialDAStartHeight(ctx context.Context) uint64 {
	if daStartHeight := c.daStartHeight.Load(); daStartHeight != 0 {
		return daStartHeight
	}

	s := store.New(store.NewEvNodeKVStore(c.db))
	daIncludedHeightBytes, err := s.GetMetadata(ctx, store.GenesisDAHeightKey)
	if err != nil || len(daIncludedHeightBytes) != 8 {
		return 0
	}

	daStartHeight := binary.LittleEndian.Uint64(daIncludedHeightBytes)
	c.daStartHeight.Store(daStartHeight)

	return daStartHeight
}

// SubmitBatchTxs implements sequencing.Sequencer.
// It adds mempool transactions to a batch.
func (c *Sequencer) SubmitBatchTxs(ctx context.Context, req coresequencer.SubmitBatchTxsRequest) (*coresequencer.SubmitBatchTxsResponse, error) {
	if !c.isValid(req.Id) {
		return nil, ErrInvalidId
	}

	if req.Batch == nil || len(req.Batch.Transactions) == 0 {
		c.logger.Info().Str("Id", string(req.Id)).Msg("Skipping submission of empty batch")
		return &coresequencer.SubmitBatchTxsResponse{}, nil
	}

	batch := coresequencer.Batch{Transactions: req.Batch.Transactions}

	err := c.queue.AddBatch(ctx, batch)
	if err != nil {
		if errors.Is(err, ErrQueueFull) {
			c.logger.Warn().
				Int("txCount", len(batch.Transactions)).
				Str("chainId", string(req.Id)).
				Msg("Batch queue is full, rejecting batch submission")
			return nil, fmt.Errorf("batch queue is full, cannot accept more batches: %w", err)
		}
		return nil, fmt.Errorf("failed to add batch: %w", err)
	}

	return &coresequencer.SubmitBatchTxsResponse{}, nil
}

// GetNextBatch gets the next batch. During catch-up, only forced inclusion txs
// are returned to match based sequencing behavior.
func (c *Sequencer) GetNextBatch(ctx context.Context, req coresequencer.GetNextBatchRequest) (*coresequencer.GetNextBatchResponse, error) {
	if !c.isValid(req.Id) {
		return nil, ErrInvalidId
	}

	daHeight := c.GetDAHeight()

	// checkpoint init path (sequencer bootstrapping)
	if daHeight > 0 && c.checkpoint.DAHeight == 0 {
		c.checkpoint = &seqcommon.Checkpoint{
			DAHeight: daHeight,
			TxIndex:  0,
		}

		// Reinitialize forced inclusion retriever with updated DA start height
		if c.fiRetriever != nil {
			c.fiRetriever.Stop()
		}
		c.fiRetriever = block.NewForcedInclusionRetriever(c.daClient, c.cfg, c.logger, c.getInitialDAStartHeight(ctx), c.genesis.DAEpochForcedInclusion)
	}

	// If we have no cached transactions or we've consumed all from the current epoch,
	// fetch the next DA epoch
	if daHeight > 0 && (len(c.cachedForcedInclusionTxs) == 0 || c.checkpoint.TxIndex >= uint64(len(c.cachedForcedInclusionTxs))) {
		daEndHeight, err := c.fetchNextDAEpoch(ctx, req.MaxBytes)
		if err != nil {
			return nil, err
		}
		daHeight = daEndHeight
	}

	// Get remaining forced inclusion transactions from checkpoint position
	// TxIndex tracks how many txs from the current epoch have been consumed (OK or Remove)
	var forcedTxs [][]byte
	if len(c.cachedForcedInclusionTxs) > 0 && c.checkpoint.TxIndex < uint64(len(c.cachedForcedInclusionTxs)) {
		forcedTxs = c.cachedForcedInclusionTxs[c.checkpoint.TxIndex:]
	}

	// Skip mempool during catch-up to match based sequencing
	var mempoolBatch *coresequencer.Batch
	if c.catchUpState.Load() != catchUpInProgress {
		var err error
		mempoolBatch, err = c.queue.Next(ctx)
		if err != nil {
			return nil, err
		}
	} else {
		mempoolBatch = &coresequencer.Batch{}
		c.logger.Debug().
			Uint64("checkpoint_da_height", c.checkpoint.DAHeight).
			Int("forced_txs", len(forcedTxs)).
			Msg("catch-up mode: skipping mempool transactions")
	}

	// Build combined tx list
	allTxs := make([][]byte, 0, len(forcedTxs)+len(mempoolBatch.Transactions))
	allTxs = append(allTxs, forcedTxs...)
	allTxs = append(allTxs, mempoolBatch.Transactions...)
	forcedTxCount := len(forcedTxs)

	// Get gas limit from execution layer
	var maxGas uint64
	info, err := c.executor.GetExecutionInfo(ctx)
	if err != nil {
		c.logger.Warn().Err(err).Msg("failed to get execution info, proceeding without gas limit")
	} else {
		maxGas = info.MaxGas
	}

	// Filter transactions - validates txs and applies size and gas filtering
	filterStatuses, err := c.executor.FilterTxs(ctx, allTxs, req.MaxBytes, maxGas, forcedTxCount > 0)
	if err != nil {
		c.logger.Warn().Err(err).Msg("failed to filter transactions, proceeding with unfiltered")
		filterStatuses = make([]execution.FilterStatus, len(allTxs))
		for i := range filterStatuses {
			filterStatuses[i] = execution.FilterOK
		}
	}

	// Process filter results sequentially for forced txs to maintain ordering.
	// We stop at the first Postpone to ensure crash recovery works correctly:
	// TxIndex tracks consumed txs from the start of the epoch, so we must process in order.
	var validForcedTxs [][]byte
	var validMempoolTxs [][]byte
	var forcedTxConsumedCount uint64
	var forcedTxPostponed bool

	for i, status := range filterStatuses {
		isForcedTx := i < forcedTxCount
		if isForcedTx {
			if forcedTxPostponed {
				// Already hit a postpone, skip remaining forced txs for this batch
				continue
			}
			switch status {
			case execution.FilterOK:
				validForcedTxs = append(validForcedTxs, allTxs[i])
				forcedTxConsumedCount++
			case execution.FilterRemove:
				// Skip removed transactions but count as consumed
				forcedTxConsumedCount++
			case execution.FilterPostpone:
				// Stop processing forced txs at first postpone to maintain order
				forcedTxPostponed = true
			}
		} else {
			// Mempool txs can be processed in any order
			switch status {
			case execution.FilterOK:
				validMempoolTxs = append(validMempoolTxs, allTxs[i])
			case execution.FilterPostpone, execution.FilterRemove:
				// Mempool txs that are postponed/removed are handled separately
			}
		}
	}

	// Return any postponed mempool txs to the queue for the next batch
	// (they were valid but didn't fit due to size/gas limits)
	var postponedMempoolTxs [][]byte
	for i, status := range filterStatuses {
		if i >= forcedTxCount && status == execution.FilterPostpone {
			postponedMempoolTxs = append(postponedMempoolTxs, allTxs[i])
		}
	}
	if len(postponedMempoolTxs) > 0 {
		if err := c.queue.Prepend(ctx, coresequencer.Batch{Transactions: postponedMempoolTxs}); err != nil {
			c.logger.Error().Err(err).Int("count", len(postponedMempoolTxs)).Msg("failed to prepend postponed mempool txs")
		}
	}

	// Update checkpoint after consuming forced inclusion transactions.
	// txIndexForTimestamp is captured before the epoch-boundary reset so the
	// final block of an epoch lands exactly on daEndTime.
	var txIndexForTimestamp uint64
	if daHeight > 0 || len(forcedTxs) > 0 {
		// Advance TxIndex by the number of consumed forced transactions
		c.checkpoint.TxIndex += forcedTxConsumedCount
		txIndexForTimestamp = c.checkpoint.TxIndex

		if c.checkpoint.TxIndex >= uint64(len(c.cachedForcedInclusionTxs)) {
			// All forced txs were consumed (OK or Remove), move to next DA epoch
			c.checkpoint.DAHeight = daHeight + 1
			c.checkpoint.TxIndex = 0
			c.cachedForcedInclusionTxs = nil
			c.SetDAHeight(c.checkpoint.DAHeight)
		}

		if err := c.checkpointStore.Save(ctx, c.checkpoint); err != nil {
			return nil, fmt.Errorf("failed to save checkpoint: %w", err)
		}

		c.logger.Debug().
			Uint64("consumed_count", forcedTxConsumedCount).
			Uint64("checkpoint_tx_index", c.checkpoint.TxIndex).
			Uint64("checkpoint_da_height", c.checkpoint.DAHeight).
			Bool("catching_up", c.catchUpState.Load() == catchUpInProgress).
			Msg("updated checkpoint after processing forced inclusion transactions")
	}

	// Build final batch: forced txs first, then mempool txs
	batchTxs := make([][]byte, 0, len(validForcedTxs)+len(validMempoolTxs))
	batchTxs = append(batchTxs, validForcedTxs...)
	batchTxs = append(batchTxs, validMempoolTxs...)

	// Spread catch-up blocks across the DA epoch window for monotonically increasing timestamps:
	//   epochStart     = daEndTime - totalEpochTxs * 1ms
	//   blockTimestamp = epochStart + txIndexForTimestamp * 1ms
	// The last block of an epoch lands exactly on daEndTime; the first block of
	// the next epoch starts at nextDaEndTime - N*1ms >= prevDaEndTime.
	// During normal operation, use wall-clock time instead.
	timestamp := time.Now()
	if c.catchUpState.Load() == catchUpInProgress {
		epochStart := c.currentDAEndTime.Add(-time.Duration(c.currentEpochTxCount) * time.Millisecond)
		timestamp = epochStart.Add(time.Duration(txIndexForTimestamp) * time.Millisecond)

		// Clamp: the DA-derived timestamp may predate blocks that were
		// produced with time.Now() before the sequencer was restarted.
		// Ensure strict monotonicity relative to the last produced block.
		if !c.lastCatchUpTimestamp.IsZero() && !timestamp.After(c.lastCatchUpTimestamp) {
			timestamp = c.lastCatchUpTimestamp.Add(time.Millisecond)
		}
		c.lastCatchUpTimestamp = timestamp
	}

	if c.isCatchingUp() && len(batchTxs) == 0 {
		return nil, block.ErrNoBatch
	}

	return &coresequencer.GetNextBatchResponse{
		Batch: &coresequencer.Batch{
			Transactions: batchTxs,
		},
		Timestamp: timestamp,
		BatchData: req.LastBatchData,
	}, nil
}

// VerifyBatch implements sequencing.Sequencer.
func (c *Sequencer) VerifyBatch(ctx context.Context, req coresequencer.VerifyBatchRequest) (*coresequencer.VerifyBatchResponse, error) {
	if !c.isValid(req.Id) {
		return nil, ErrInvalidId
	}

	proofs, err := c.daClient.GetProofs(ctx, req.BatchData, c.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to get proofs: %w", err)
	}

	valid, err := c.daClient.Validate(ctx, req.BatchData, proofs, c.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to validate proof: %w", err)
	}

	for _, v := range valid {
		if !v {
			return &coresequencer.VerifyBatchResponse{Status: false}, nil
		}
	}
	return &coresequencer.VerifyBatchResponse{Status: true}, nil
}

func (c *Sequencer) isValid(Id []byte) bool {
	return bytes.Equal(c.Id, Id)
}

// SetDAHeight sets the current DA height for the sequencer
// This should be called when the sequencer needs to sync to a specific DA height
func (c *Sequencer) SetDAHeight(height uint64) {
	c.daHeight.Store(height)
}

// GetDAHeight returns the current DA height
func (c *Sequencer) GetDAHeight() uint64 {
	return c.daHeight.Load()
}

// isCatchingUp returns whether the sequencer is in catch-up mode.
func (c *Sequencer) isCatchingUp() bool {
	return c.catchUpState.Load() == catchUpInProgress
}

// fetchNextDAEpoch fetches transactions from the next DA epoch. It also
// updates catch-up state: entering catch-up if behind, exiting when reaching DA head.
func (c *Sequencer) fetchNextDAEpoch(ctx context.Context, maxBytes uint64) (uint64, error) {
	currentDAHeight := c.checkpoint.DAHeight

	// Determine catch-up state before the (potentially expensive) epoch fetch.
	// This is done once per sequencer lifecycle — subsequent catch-up exits are
	// handled by ErrHeightFromFuture below.
	c.updateCatchUpState(ctx)

	c.logger.Debug().
		Uint64("da_height", currentDAHeight).
		Uint64("tx_index", c.checkpoint.TxIndex).
		Bool("catching_up", c.catchUpState.Load() == catchUpInProgress).
		Msg("fetching forced inclusion transactions from DA")

	forcedTxsEvent, err := c.fiRetriever.RetrieveForcedIncludedTxs(ctx, currentDAHeight)
	if err != nil {
		if errors.Is(err, datypes.ErrHeightFromFuture) {
			c.logger.Debug().
				Uint64("da_height", currentDAHeight).
				Msg("DA height from future, waiting for DA to produce block")

			if c.catchUpState.Load() == catchUpInProgress {
				c.logger.Info().Uint64("da_height", currentDAHeight).
					Msg("catch-up complete: reached DA head, resuming normal sequencing")
				c.catchUpState.Store(catchUpDone)
			}

			return 0, nil
		} else if errors.Is(err, block.ErrForceInclusionNotConfigured) {
			c.cachedForcedInclusionTxs = [][]byte{}
			c.catchUpState.Store(catchUpDone)
			return 0, nil
		}

		return 0, fmt.Errorf("failed to retrieve forced inclusion transactions: %w", err)
	}

	// Record total tx count for the epoch so the timestamp jitter can be computed
	// after oversized txs are filtered out below.

	// Filter out oversized transactions
	validTxs := make([][]byte, 0, len(forcedTxsEvent.Txs))
	skippedTxs := 0
	for _, tx := range forcedTxsEvent.Txs {
		if uint64(len(tx)) > maxBytes {
			c.logger.Warn().
				Uint64("da_height", forcedTxsEvent.StartDaHeight).
				Int("blob_size", len(tx)).
				Uint64("max_bytes", maxBytes).
				Msg("forced inclusion blob exceeds maximum size - skipping")
			skippedTxs++
			continue
		}
		validTxs = append(validTxs, tx)
	}

	c.logger.Info().
		Int("valid_tx_count", len(validTxs)).
		Int("skipped_tx_count", skippedTxs).
		Uint64("da_height_start", forcedTxsEvent.StartDaHeight).
		Uint64("da_height_end", forcedTxsEvent.EndDaHeight).
		Bool("catching_up", c.catchUpState.Load() == catchUpInProgress).
		Msg("fetched forced inclusion transactions from DA")

	// Cache the transactions for this DA epoch
	c.cachedForcedInclusionTxs = validTxs
	c.currentEpochTxCount = uint64(len(validTxs))

	// Store DA epoch end time for timestamp usage during catch-up
	if daEndTime := forcedTxsEvent.Timestamp.UTC(); forcedTxsEvent.Timestamp.UTC().After(c.currentDAEndTime) {
		c.currentDAEndTime = daEndTime
	}

	return forcedTxsEvent.EndDaHeight, nil
}

// updateCatchUpState checks if catch-up is needed by comparing checkpoint
// DA height with latest DA height. Runs once per sequencer lifecycle.
// If more than one epoch behind, enters catch-up mode (forced txs only, no mempool).
func (c *Sequencer) updateCatchUpState(ctx context.Context) {
	if c.catchUpState.Load() != catchUpUnchecked {
		return
	}
	// Optimistically mark as done; overridden to catchUpInProgress below if
	// catch-up is actually needed.
	c.catchUpState.Store(catchUpDone)

	epochSize := c.genesis.DAEpochForcedInclusion
	if epochSize == 0 {
		return
	}

	currentDAHeight := c.checkpoint.DAHeight
	daStartHeight := c.genesis.DAStartHeight

	latestDAHeight, err := c.daClient.GetLatestDAHeight(ctx)
	if err != nil {
		c.logger.Warn().Err(err).
			Msg("failed to get latest DA height for catch-up detection, skipping check")
		return
	}

	// At head, no catch-up needed
	if latestDAHeight <= currentDAHeight {
		return
	}

	// Calculate missed epochs
	currentEpoch := types.CalculateEpochNumber(currentDAHeight, daStartHeight, epochSize)
	latestEpoch := types.CalculateEpochNumber(latestDAHeight, daStartHeight, epochSize)
	missedEpochs := latestEpoch - currentEpoch

	if missedEpochs == 0 {
		c.logger.Debug().
			Uint64("checkpoint_da_height", currentDAHeight).
			Uint64("latest_da_height", latestDAHeight).
			Uint64("current_epoch", currentEpoch).
			Uint64("latest_epoch", latestEpoch).
			Msg("sequencer at DA head, no catch-up needed")
		return
	}

	// At least one epoch behind - enter catch-up mode.
	// Read the last block time from the store so that catch-up timestamps
	// are guaranteed to be strictly after any previously produced block.
	s := store.New(store.NewEvNodeKVStore(c.db))
	state, err := s.GetState(ctx)
	if err == nil && !state.LastBlockTime.IsZero() {
		c.lastCatchUpTimestamp = state.LastBlockTime
		c.logger.Debug().
			Time("last_block_time", state.LastBlockTime).
			Msg("initialized catch-up timestamp floor from last block time")
	}

	if c.lastCatchUpTimestamp.After(c.currentDAEndTime) {
		c.currentDAEndTime = c.lastCatchUpTimestamp
	}

	c.catchUpState.Store(catchUpInProgress)
	c.logger.Warn().
		Uint64("checkpoint_da_height", currentDAHeight).
		Uint64("latest_da_height", latestDAHeight).
		Uint64("current_epoch", currentEpoch).
		Uint64("latest_epoch", latestEpoch).
		Uint64("missed_epochs", missedEpochs).
		Msg("entering catch-up mode: replaying missed epochs with forced inclusion txs only")
}
