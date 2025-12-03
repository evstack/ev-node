package single

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	ds "github.com/ipfs/go-datastore"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block"
	coreda "github.com/evstack/ev-node/core/da"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/genesis"
	seqcommon "github.com/evstack/ev-node/sequencers/common"
)

var (
	// ErrInvalidId is returned when the chain id is invalid
	ErrInvalidId = errors.New("invalid chain id")
)

// ForcedInclusionRetriever defines the interface for retrieving forced inclusion transactions from DA
type ForcedInclusionRetriever interface {
	RetrieveForcedIncludedTxs(ctx context.Context, daHeight uint64) (*block.ForcedInclusionEvent, error)
}

// pendingForcedInclusionTx represents a forced inclusion transaction that couldn't fit in the current epoch
type pendingForcedInclusionTx struct {
	Data           []byte
	OriginalHeight uint64
}

var _ coresequencer.Sequencer = (*Sequencer)(nil)

// Sequencer implements core sequencing interface
type Sequencer struct {
	logger zerolog.Logger

	proposer bool

	Id []byte
	da coreda.DA

	batchTime time.Duration

	queue *BatchQueue // single queue for immediate availability

	metrics *Metrics

	// Forced inclusion support
	fiRetriever               ForcedInclusionRetriever
	genesis                   genesis.Genesis
	daHeight                  atomic.Uint64
	pendingForcedInclusionTxs []pendingForcedInclusionTx
}

// NewSequencer creates a new Single Sequencer
func NewSequencer(
	ctx context.Context,
	logger zerolog.Logger,
	db ds.Batching,
	da coreda.DA,
	id []byte,
	batchTime time.Duration,
	metrics *Metrics,
	proposer bool,
	maxQueueSize int,
	fiRetriever ForcedInclusionRetriever,
	gen genesis.Genesis,
) (*Sequencer, error) {
	s := &Sequencer{
		logger:                    logger,
		da:                        da,
		batchTime:                 batchTime,
		Id:                        id,
		queue:                     NewBatchQueue(db, "batches", maxQueueSize),
		metrics:                   metrics,
		proposer:                  proposer,
		fiRetriever:               fiRetriever,
		genesis:                   gen,
		pendingForcedInclusionTxs: make([]pendingForcedInclusionTx, 0),
	}
	s.SetDAHeight(gen.DAStartHeight) // will be overridden by the executor

	loadCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.queue.Load(loadCtx); err != nil {
		return nil, fmt.Errorf("failed to load batch queue from DB: %w", err)
	}

	// No DA submission loop here; handled by central manager
	return s, nil
}

// SubmitBatchTxs implements sequencing.Sequencer.
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
func (c *Sequencer) GetNextBatch(ctx context.Context, req coresequencer.GetNextBatchRequest) (*coresequencer.GetNextBatchResponse, error) {
	if !c.isValid(req.Id) {
		return nil, ErrInvalidId
	}

	currentDAHeight := c.GetDAHeight()

	forcedTxsEvent, err := c.fiRetriever.RetrieveForcedIncludedTxs(ctx, currentDAHeight)
	if err != nil {
		if errors.Is(err, coreda.ErrHeightFromFuture) {
			c.logger.Debug().
				Uint64("da_height", currentDAHeight).
				Msg("DA height from future, waiting for DA to produce block")
		} else if !errors.Is(err, block.ErrForceInclusionNotConfigured) {
			c.logger.Error().Err(err).Uint64("da_height", currentDAHeight).Msg("failed to retrieve forced inclusion transactions")
		}

		// Still create an empty forced inclusion event
		forcedTxsEvent = &block.ForcedInclusionEvent{
			Txs:           [][]byte{},
			StartDaHeight: currentDAHeight,
			EndDaHeight:   currentDAHeight,
		}
	} else {
		// Update DA height.
		// If we are in between epochs, we still need to bump the da height.
		// At the end of an epoch, we need to bump to go to the next epoch.
		if forcedTxsEvent.EndDaHeight >= currentDAHeight {
			c.SetDAHeight(forcedTxsEvent.EndDaHeight + 1)
		}
	}

	// Always try to process forced inclusion transactions (including pending from previous epochs)
	forcedTxs := c.processForcedInclusionTxs(forcedTxsEvent, req.MaxBytes)

	c.logger.Debug().
		Int("tx_count", len(forcedTxs)).
		Uint64("da_height_start", forcedTxsEvent.StartDaHeight).
		Uint64("da_height_end", forcedTxsEvent.EndDaHeight).
		Msg("retrieved forced inclusion transactions from DA")

	// Calculate size used by forced inclusion transactions
	forcedTxsSize := 0
	for _, tx := range forcedTxs {
		forcedTxsSize += len(tx)
	}

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
					c.logger.Error().Err(err).
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

		c.logger.Debug().
			Int("forced_tx_count", len(forcedTxs)).
			Int("forced_txs_size", forcedTxsSize).
			Int("batch_tx_count", len(trimmedBatchTxs)).
			Int("batch_size", currentBatchSize).
			Int("total_tx_count", len(batch.Transactions)).
			Int("total_size", forcedTxsSize+currentBatchSize).
			Msg("combined forced inclusion and batch transactions")
	}

	return &coresequencer.GetNextBatchResponse{
		Batch:     batch,
		Timestamp: time.Now(),
		BatchData: req.LastBatchData,
	}, nil
}

// RecordMetrics updates the metrics with the given values.
// This method is intended to be called by the block manager after submitting data to the DA layer.
func (c *Sequencer) RecordMetrics(gasPrice float64, blobSize uint64, statusCode coreda.StatusCode, numPendingBlocks uint64, includedBlockHeight uint64) {
	if c.metrics != nil {
		c.metrics.GasPrice.Set(gasPrice)
		c.metrics.LastBlobSize.Set(float64(blobSize))
		c.metrics.TransactionStatus.With("status", fmt.Sprintf("%d", statusCode)).Add(1)
		c.metrics.NumPendingBlocks.Set(float64(numPendingBlocks))
		c.metrics.IncludedBlockHeight.Set(float64(includedBlockHeight))
	}
}

// VerifyBatch implements sequencing.Sequencer.
func (c *Sequencer) VerifyBatch(ctx context.Context, req coresequencer.VerifyBatchRequest) (*coresequencer.VerifyBatchResponse, error) {
	if !c.isValid(req.Id) {
		return nil, ErrInvalidId
	}

	if !c.proposer {

		proofs, err := c.da.GetProofs(ctx, req.BatchData, c.Id)
		if err != nil {
			return nil, fmt.Errorf("failed to get proofs: %w", err)
		}

		valid, err := c.da.Validate(ctx, req.BatchData, proofs, c.Id)
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
	return &coresequencer.VerifyBatchResponse{Status: true}, nil
}

func (c *Sequencer) isValid(Id []byte) bool {
	return bytes.Equal(c.Id, Id)
}

// SetDAHeight sets the current DA height for the sequencer
// This should be called when the sequencer needs to sync to a specific DA height
func (c *Sequencer) SetDAHeight(height uint64) {
	c.daHeight.Store(height)
	c.logger.Debug().Uint64("da_height", height).Msg("DA height updated")
}

// GetDAHeight returns the current DA height
func (c *Sequencer) GetDAHeight() uint64 {
	return c.daHeight.Load()
}

// processForcedInclusionTxs processes forced inclusion transactions with size validation and pending queue management
func (c *Sequencer) processForcedInclusionTxs(event *block.ForcedInclusionEvent, maxBytes uint64) [][]byte {
	currentSize := 0
	var newPendingTxs []pendingForcedInclusionTx
	var validatedTxs [][]byte

	// First, process any pending transactions from previous epochs
	for _, pendingTx := range c.pendingForcedInclusionTxs {
		txSize := seqcommon.GetBlobSize(pendingTx.Data)

		if !seqcommon.ValidateBlobSize(pendingTx.Data) {
			c.logger.Warn().
				Uint64("original_height", pendingTx.OriginalHeight).
				Int("blob_size", txSize).
				Msg("pending forced inclusion blob exceeds absolute maximum size - skipping")
			continue
		}

		if seqcommon.WouldExceedCumulativeSize(currentSize, txSize, maxBytes) {
			c.logger.Debug().
				Uint64("original_height", pendingTx.OriginalHeight).
				Int("current_size", currentSize).
				Int("blob_size", txSize).
				Msg("pending blob would exceed max size for this epoch - deferring again")
			newPendingTxs = append(newPendingTxs, pendingTx)
			continue
		}

		validatedTxs = append(validatedTxs, pendingTx.Data)
		currentSize += txSize

		c.logger.Debug().
			Uint64("original_height", pendingTx.OriginalHeight).
			Int("blob_size", txSize).
			Int("current_size", currentSize).
			Msg("processed pending forced inclusion transaction")
	}

	// Now process new transactions from this epoch
	for _, tx := range event.Txs {
		txSize := seqcommon.GetBlobSize(tx)

		if !seqcommon.ValidateBlobSize(tx) {
			c.logger.Warn().
				Uint64("da_height", event.StartDaHeight).
				Int("blob_size", txSize).
				Msg("forced inclusion blob exceeds absolute maximum size - skipping")
			continue
		}

		if seqcommon.WouldExceedCumulativeSize(currentSize, txSize, maxBytes) {
			c.logger.Debug().
				Uint64("da_height", event.StartDaHeight).
				Int("current_size", currentSize).
				Int("blob_size", txSize).
				Msg("blob would exceed max size for this epoch - deferring to pending queue")

			// Store for next call
			newPendingTxs = append(newPendingTxs, pendingForcedInclusionTx{
				Data:           tx,
				OriginalHeight: event.StartDaHeight,
			})
			continue
		}

		validatedTxs = append(validatedTxs, tx)
		currentSize += txSize

		c.logger.Debug().
			Int("blob_size", txSize).
			Int("current_size", currentSize).
			Msg("processed forced inclusion transaction")
	}

	// Update pending queue
	c.pendingForcedInclusionTxs = newPendingTxs
	if len(newPendingTxs) > 0 {
		c.logger.Info().
			Int("new_pending_count", len(newPendingTxs)).
			Msg("stored pending forced inclusion transactions for next epoch")
	}

	if len(validatedTxs) > 0 {
		c.logger.Info().
			Int("processed_tx_count", len(validatedTxs)).
			Int("pending_tx_count", len(newPendingTxs)).
			Int("current_size", currentSize).
			Msg("completed processing forced inclusion transactions")
	}

	return validatedTxs
}
