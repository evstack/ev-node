// Package single implements a single sequencer.
package single

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
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

	// Apply filtering to validate transactions and optionally limit by gas
	var filteredForcedTxs [][]byte
	var remainingGasFilteredTxs [][]byte
	if len(forcedTxs) > 0 {
		// Get current gas limit from execution layer
		info, err := c.executor.GetExecutionInfo(ctx, 0) // 0 = latest/next block
		if err != nil {
			c.logger.Warn().Err(err).Msg("failed to get execution info for gas filtering, proceeding without gas filter")
			filteredForcedTxs = forcedTxs
		} else {
			filteredForcedTxs, remainingGasFilteredTxs, err = c.executor.FilterDATransactions(ctx, forcedTxs, info.MaxGas)
			if err != nil {
				c.logger.Warn().Err(err).Msg("failed to filter DA transactions, proceeding without filter")
				filteredForcedTxs = forcedTxs
			} else {
				c.logger.Debug().
					Int("input_forced_txs", len(forcedTxs)).
					Int("filtered_forced_txs", len(filteredForcedTxs)).
					Int("remaining_for_next_block", len(remainingGasFilteredTxs)).
					Uint64("max_gas", info.MaxGas).
					Msg("filtered forced inclusion transactions")
			}
		}
	} else {
		filteredForcedTxs = forcedTxs
	}

	// Calculate size used by forced inclusion transactions
	forcedTxsSize := 0
	for _, tx := range filteredForcedTxs {
		forcedTxsSize += len(tx)
	}

	// Update checkpoint after consuming forced inclusion transactions
	// Only advance checkpoint by the number of txs actually included (after gas filtering)
	// Note: gibberish txs filtered by FilterDATransactions are permanently skipped,
	// but gas-filtered valid txs (remainingGasFilteredTxs) stay in cache for next block
	if daHeight > 0 || len(forcedTxs) > 0 {
		// Count how many txs we're actually consuming from the checkpoint
		// This is: filteredForcedTxs + (original forcedTxs - filteredForcedTxs - remainingGasFilteredTxs)
		// The difference (original - filtered - remaining) represents gibberish that was filtered out
		txsConsumed := uint64(len(forcedTxs) - len(remainingGasFilteredTxs))
		c.checkpoint.TxIndex += txsConsumed

		// If we have remaining gas-filtered txs, don't advance to next epoch yet
		// These will be picked up in the next GetNextBatch call
		if len(remainingGasFilteredTxs) > 0 {
			// Update cached txs to only contain the remaining ones
			c.cachedForcedInclusionTxs = remainingGasFilteredTxs
			c.checkpoint.TxIndex = 0 // Reset index since we're replacing the cache

			c.logger.Debug().
				Int("remaining_txs", len(remainingGasFilteredTxs)).
				Msg("keeping gas-filtered transactions for next block")
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
			Int("forced_tx_count", len(filteredForcedTxs)).
			Uint64("checkpoint_da_height", c.checkpoint.DAHeight).
			Uint64("checkpoint_tx_index", c.checkpoint.TxIndex).
			Msg("processed forced inclusion transactions and updated checkpoint")
	}

	// Use filtered forced txs for the rest of the function
	forcedTxs = filteredForcedTxs

	batch, err := c.queue.Next(ctx)
	if err != nil {
		return nil, err
	}

	// Prepend forced inclusion transactions to the batch
	// and ensure total size doesn't exceed maxBytes
	if len(forcedTxs) > 0 {
		// Trim batch transactions to fit within maxBytes
		remainingBytes := int(req.MaxBytes) - forcedTxsSize
		trimmedBatchTxs := make([][]byte, 0, len(batch.Transactions))
		currentBatchSize := 0

		for i, tx := range batch.Transactions {
			txSize := len(tx)
			if currentBatchSize+txSize > remainingBytes {
				// Would exceed limit, return remaining txs to the front of the queue
				excludedBatch := coresequencer.Batch{Transactions: batch.Transactions[i:]}
				if err := c.queue.Prepend(ctx, excludedBatch); err != nil {
					// tx will be lost forever, but we shouldn't halt.
					// halting doesn't not add any value.
					c.logger.Error().Err(err).
						Str("tx", hex.EncodeToString(tx)).
						Int("excluded_count", len(batch.Transactions)-i).
						Msg("failed to prepend excluded transactions back to queue")
				} else {
					c.logger.Debug().
						Int("excluded_count", len(batch.Transactions)-i).
						Msg("returned excluded batch transactions to front of queue")
				}
				break
			}
			trimmedBatchTxs = append(trimmedBatchTxs, tx)
			currentBatchSize += txSize
		}

		batch.Transactions = append(forcedTxs, trimmedBatchTxs...)

		// Create ForceIncludedMask: true for forced txs, false for mempool txs.
		// Forced included txs are always first in the batch.
		batch.ForceIncludedMask = make([]bool, len(batch.Transactions))
		for i := range len(forcedTxs) {
			batch.ForceIncludedMask[i] = true
		}

		c.logger.Debug().
			Int("forced_tx_count", len(forcedTxs)).
			Int("forced_txs_size", forcedTxsSize).
			Int("batch_tx_count", len(trimmedBatchTxs)).
			Int("batch_size", currentBatchSize).
			Int("total_tx_count", len(batch.Transactions)).
			Int("total_size", forcedTxsSize+currentBatchSize).
			Msg("combined forced inclusion and batch transactions")
	} else if len(batch.Transactions) > 0 {
		// No forced txs, but we have mempool txs - mark all as non-force-included
		batch.ForceIncludedMask = make([]bool, len(batch.Transactions))
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

		if totalBytes+txSize > maxBytes {
			break
		}

		result = append(result, tx)
		totalBytes += txSize
	}

	return result
}
