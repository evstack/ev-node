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
)

// ErrInvalidId is returned when the chain id is invalid
var ErrInvalidId = errors.New("invalid chain id")

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
		db:              db,
		logger:          logger,
		daClient:        daClient,
		cfg:             cfg,
		batchTime:       cfg.Node.BlockTime.Duration,
		Id:              id,
		queue:           NewBatchQueue(db, "batches", maxQueueSize),
		checkpointStore: seqcommon.NewCheckpointStore(db, ds.NewKey("/single/checkpoint")),
		genesis:         genesis,
		executor:        executor,
	}
	s.SetDAHeight(genesis.DAStartHeight) // default value, will be overridden by executor or submitter
	s.daStartHeight.Store(genesis.DAStartHeight)

	loadCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Load batch queue from DB
	if err := s.queue.Load(loadCtx); err != nil {
		return nil, fmt.Errorf("failed to load batch queue from DB: %w", err)
	}

	// Load checkpoint from DB, or initialize if none exists
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

// GetNextBatch implements sequencing.Sequencer.
// It gets the next batch of transactions and fetch for forced included transactions.
func (c *Sequencer) GetNextBatch(ctx context.Context, req coresequencer.GetNextBatchRequest) (*coresequencer.GetNextBatchResponse, error) {
	if !c.isValid(req.Id) {
		return nil, ErrInvalidId
	}

	daHeight := c.GetDAHeight()

	// checkpoint init path, only hit when sequencer is bootstrapping
	if daHeight > 0 && c.checkpoint.DAHeight == 0 {
		c.checkpoint = &seqcommon.Checkpoint{
			DAHeight: daHeight,
			TxIndex:  0,
		}

		// override forced inclusion retriever, as the da start height have been updated
		// Stop the old retriever first
		if c.fiRetriever != nil {
			c.fiRetriever.Stop()
		}
		c.fiRetriever = block.NewForcedInclusionRetriever(c.daClient, c.cfg, c.logger, c.getInitialDAStartHeight(ctx), c.genesis.DAEpochForcedInclusion)
	}

	// If we have no cached transactions or we've consumed all from the current cache,
	// fetch the next DA epoch
	if daHeight > 0 && (len(c.cachedForcedInclusionTxs) == 0 || c.checkpoint.TxIndex >= uint64(len(c.cachedForcedInclusionTxs))) {
		daEndHeight, err := c.fetchNextDAEpoch(ctx, req.MaxBytes)
		if err != nil {
			return nil, err
		}

		daHeight = daEndHeight
	}

	// Process forced inclusion transactions from checkpoint position
	forcedTxs := c.processForcedInclusionTxsFromCheckpoint(req.MaxBytes)

	// Get mempool transactions from queue
	mempoolBatch, err := c.queue.Next(ctx)
	if err != nil {
		return nil, err
	}

	// If no forced txs, skip filtering entirely and just use mempool batch
	if len(forcedTxs) == 0 {
		// Update checkpoint - advance to next DA epoch since this one is empty/consumed
		if daHeight > 0 && (len(c.cachedForcedInclusionTxs) == 0 || c.checkpoint.TxIndex >= uint64(len(c.cachedForcedInclusionTxs))) {
			c.checkpoint.DAHeight = daHeight + 1
			c.checkpoint.TxIndex = 0
			c.cachedForcedInclusionTxs = nil
			c.SetDAHeight(c.checkpoint.DAHeight)

			// Persist checkpoint
			if err := c.checkpointStore.Save(ctx, c.checkpoint); err != nil {
				return nil, fmt.Errorf("failed to save checkpoint: %w", err)
			}
		}

		// Trim mempool transactions to fit within maxBytes
		var trimmedMempoolTxs [][]byte
		currentBatchSize := 0

		for i, tx := range mempoolBatch.Transactions {
			txSize := len(tx)
			if req.MaxBytes > 0 && currentBatchSize+txSize > int(req.MaxBytes) {
				// Return remaining mempool txs to the front of the queue
				excludedBatch := coresequencer.Batch{Transactions: mempoolBatch.Transactions[i:]}
				if err := c.queue.Prepend(ctx, excludedBatch); err != nil {
					c.logger.Error().Err(err).
						Int("excluded_count", len(mempoolBatch.Transactions)-i).
						Msg("failed to prepend excluded transactions back to queue")
				}
				break
			}
			trimmedMempoolTxs = append(trimmedMempoolTxs, tx)
			currentBatchSize += txSize
		}

		// No forced txs, so no force included mask needed (all false)
		batch := &coresequencer.Batch{
			Transactions:      trimmedMempoolTxs,
			ForceIncludedMask: make([]bool, len(trimmedMempoolTxs)),
		}

		return &coresequencer.GetNextBatchResponse{
			Batch:     batch,
			Timestamp: time.Now(),
			BatchData: req.LastBatchData,
		}, nil
	}

	// We have forced txs - need to filter them (validation + gas limiting)
	// Build combined tx list and mask for filtering
	allTxs := make([][]byte, 0, len(forcedTxs)+len(mempoolBatch.Transactions))
	forceIncludedMask := make([]bool, 0, len(forcedTxs)+len(mempoolBatch.Transactions))

	allTxs = append(allTxs, forcedTxs...)
	for range forcedTxs {
		forceIncludedMask = append(forceIncludedMask, true)
	}
	allTxs = append(allTxs, mempoolBatch.Transactions...)
	for range mempoolBatch.Transactions {
		forceIncludedMask = append(forceIncludedMask, false)
	}

	// Get current gas limit from execution layer
	var maxGas uint64
	info, err := c.executor.GetExecutionInfo(ctx, 0) // 0 = latest/next block
	if err != nil {
		c.logger.Warn().Err(err).Msg("failed to get execution info, proceeding without gas limit")
	} else {
		maxGas = info.MaxGas
	}

	// Filter transactions - validates force-included txs and applies gas filtering
	filterResult, err := c.executor.FilterTxs(ctx, allTxs, forceIncludedMask, maxGas)
	if err != nil {
		c.logger.Warn().Err(err).Msg("failed to filter transactions, proceeding with unfiltered")
		// Fall back to using original txs without filtering
		filterResult = &execution.FilterTxsResult{
			ValidTxs:          allTxs,
			ForceIncludedMask: forceIncludedMask,
			RemainingTxs:      nil,
		}
	}

	// RemainingTxs contains valid force-included txs that didn't fit due to gas limit
	remainingDATxs := filterResult.RemainingTxs

	// Separate forced txs and mempool txs from the result
	var validForcedTxs [][]byte
	var validMempoolTxs [][]byte

	for i, tx := range filterResult.ValidTxs {
		isForced := i < len(filterResult.ForceIncludedMask) && filterResult.ForceIncludedMask[i]
		if isForced {
			validForcedTxs = append(validForcedTxs, tx)
		} else {
			validMempoolTxs = append(validMempoolTxs, tx)
		}
	}

	// Calculate size used by forced inclusion transactions
	forcedTxsSize := 0
	for _, tx := range validForcedTxs {
		forcedTxsSize += len(tx)
	}

	// Calculate remaining bytes for mempool txs
	var remainingBytes int
	if req.MaxBytes > 0 {
		remainingBytes = int(req.MaxBytes) - forcedTxsSize
	} else {
		remainingBytes = int(^uint(0) >> 1) // Max int value for unlimited
	}

	// Trim mempool txs to fit within size limit
	var trimmedMempoolTxs [][]byte
	currentBatchSize := 0

	for i, tx := range validMempoolTxs {
		txSize := len(tx)

		// Check size limit
		if req.MaxBytes > 0 && currentBatchSize+txSize > remainingBytes {
			// Return remaining mempool txs to the front of the queue
			remainingMempoolTxs := validMempoolTxs[i:]
			if len(remainingMempoolTxs) > 0 {
				excludedBatch := coresequencer.Batch{Transactions: remainingMempoolTxs}
				if err := c.queue.Prepend(ctx, excludedBatch); err != nil {
					c.logger.Error().Err(err).
						Int("excluded_count", len(remainingMempoolTxs)).
						Msg("failed to prepend excluded transactions back to queue")
				}
			}
			break
		}

		trimmedMempoolTxs = append(trimmedMempoolTxs, tx)
		currentBatchSize += txSize
	}

	// Update checkpoint after consuming forced inclusion transactions
	// Note: gibberish txs are permanently skipped, but gas-filtered valid DA txs stay for next block
	if daHeight > 0 || len(forcedTxs) > 0 {
		// Count how many DA txs we're actually consuming from the checkpoint
		// Original forcedTxs count minus remaining DA txs = consumed (including gibberish filtered out)
		txsConsumed := uint64(len(forcedTxs) - len(remainingDATxs))
		c.checkpoint.TxIndex += txsConsumed

		// If we have remaining DA txs, don't advance to next epoch yet
		if len(remainingDATxs) > 0 {
			// Update cached txs to only contain the remaining ones
			c.cachedForcedInclusionTxs = remainingDATxs
			c.checkpoint.TxIndex = 0 // Reset index since we're replacing the cache

			c.logger.Debug().
				Int("remaining_da_txs", len(remainingDATxs)).
				Msg("keeping gas-filtered DA transactions for next block")
		} else if c.checkpoint.TxIndex >= uint64(len(c.cachedForcedInclusionTxs)) {
			// If we've consumed all transactions from this DA epoch, move to next
			c.checkpoint.DAHeight = daHeight + 1
			c.checkpoint.TxIndex = 0
			c.cachedForcedInclusionTxs = nil

			// Update the global DA height
			c.SetDAHeight(c.checkpoint.DAHeight)
		}

		// Persist checkpoint
		if err := c.checkpointStore.Save(ctx, c.checkpoint); err != nil {
			return nil, fmt.Errorf("failed to save checkpoint: %w", err)
		}

		c.logger.Debug().
			Int("included_da_tx_count", len(validForcedTxs)).
			Uint64("checkpoint_da_height", c.checkpoint.DAHeight).
			Uint64("checkpoint_tx_index", c.checkpoint.TxIndex).
			Msg("processed forced inclusion transactions and updated checkpoint")
	}

	// Build the final batch: forced txs first, then mempool txs
	var finalTxs [][]byte
	finalTxs = append(finalTxs, validForcedTxs...)
	finalTxs = append(finalTxs, trimmedMempoolTxs...)

	// Build the final mask: true for forced, false for mempool
	finalMask := make([]bool, len(finalTxs))
	for i := 0; i < len(validForcedTxs); i++ {
		finalMask[i] = true
	}

	batch := &coresequencer.Batch{
		Transactions:      finalTxs,
		ForceIncludedMask: finalMask,
	}

	if len(finalTxs) > 0 {
		c.logger.Debug().
			Int("forced_tx_count", len(validForcedTxs)).
			Int("forced_txs_size", forcedTxsSize).
			Int("mempool_tx_count", len(trimmedMempoolTxs)).
			Int("mempool_txs_size", currentBatchSize).
			Int("total_tx_count", len(finalTxs)).
			Int("total_size", forcedTxsSize+currentBatchSize).
			Int("remaining_da_txs", len(remainingDATxs)).
			Uint64("max_gas", maxGas).
			Msg("combined forced inclusion and mempool transactions for batch")
	}

	return &coresequencer.GetNextBatchResponse{
		Batch:     batch,
		Timestamp: time.Now(),
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

// fetchNextDAEpoch fetches transactions from the next DA epoch using checkpoint
func (c *Sequencer) fetchNextDAEpoch(ctx context.Context, maxBytes uint64) (uint64, error) {
	currentDAHeight := c.checkpoint.DAHeight

	c.logger.Debug().
		Uint64("da_height", currentDAHeight).
		Uint64("tx_index", c.checkpoint.TxIndex).
		Msg("fetching forced inclusion transactions from DA")

	forcedTxsEvent, err := c.fiRetriever.RetrieveForcedIncludedTxs(ctx, currentDAHeight)
	if err != nil {
		if errors.Is(err, datypes.ErrHeightFromFuture) {
			c.logger.Debug().
				Uint64("da_height", currentDAHeight).
				Msg("DA height from future, waiting for DA to produce block")
			return 0, nil
		} else if errors.Is(err, block.ErrForceInclusionNotConfigured) {
			// Forced inclusion not configured, continue without forced txs
			c.cachedForcedInclusionTxs = [][]byte{}
			return 0, nil
		}

		return 0, fmt.Errorf("failed to retrieve forced inclusion transactions: %w", err)
	}

	// Validate and filter transactions
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
		Msg("fetched forced inclusion transactions from DA")

	// Cache the transactions
	c.cachedForcedInclusionTxs = validTxs

	return forcedTxsEvent.EndDaHeight, nil
}

// processForcedInclusionTxsFromCheckpoint processes forced inclusion transactions from checkpoint position
func (c *Sequencer) processForcedInclusionTxsFromCheckpoint(maxBytes uint64) [][]byte {
	if len(c.cachedForcedInclusionTxs) == 0 || c.checkpoint.TxIndex >= uint64(len(c.cachedForcedInclusionTxs)) {
		return [][]byte{}
	}

	var result [][]byte
	var totalBytes uint64

	// Start from the checkpoint index
	for i := c.checkpoint.TxIndex; i < uint64(len(c.cachedForcedInclusionTxs)); i++ {
		tx := c.cachedForcedInclusionTxs[i]
		txSize := uint64(len(tx))

		// maxBytes=0 means unlimited (no size limit)
		if maxBytes > 0 && totalBytes+txSize > maxBytes {
			break
		}

		result = append(result, tx)
		totalBytes += txSize
	}

	return result
}
