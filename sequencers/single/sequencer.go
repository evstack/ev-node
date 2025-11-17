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

	currentDAHeight := c.daHeight.Load()

	forcedEvent, err := c.fiRetriever.RetrieveForcedIncludedTxs(ctx, currentDAHeight)
	if err != nil {
		// If we get a height from future error, keep the current DA height and return batch
		// We'll retry the same height on the next call until DA produces that block
		if errors.Is(err, coreda.ErrHeightFromFuture) {
			c.logger.Debug().
				Uint64("da_height", currentDAHeight).
				Msg("DA height from future, waiting for DA to produce block")

			batch, err := c.queue.Next(ctx)
			if err != nil {
				return nil, err
			}

			return &coresequencer.GetNextBatchResponse{
				Batch:     batch,
				Timestamp: time.Now(),
				BatchData: req.LastBatchData,
			}, nil
		}

		// If forced inclusion is not configured, continue without forced txs
		if !errors.Is(err, block.ErrForceInclusionNotConfigured) {
			c.logger.Error().Err(err).Uint64("da_height", currentDAHeight).Msg("failed to retrieve forced inclusion transactions")
			// Continue without forced txs on other errors
		}
	}

	// Always try to process forced inclusion transactions (can be in queue)
	forcedTxs := c.processForcedInclusionTxs(forcedEvent, req.MaxBytes)
	if forcedEvent.EndDaHeight > currentDAHeight {
		c.SetDAHeight(forcedEvent.EndDaHeight)
	} else if forcedEvent.StartDaHeight > currentDAHeight {
		c.SetDAHeight(forcedEvent.StartDaHeight)
	}

	c.logger.Debug().
		Int("tx_count", len(forcedTxs)).
		Uint64("da_height_start", forcedEvent.StartDaHeight).
		Uint64("da_height_end", forcedEvent.EndDaHeight).
		Msg("retrieved forced inclusion transactions from DA")

	batch, err := c.queue.Next(ctx)
	if err != nil {
		return nil, err
	}

	// Prepend forced inclusion transactions to the batch
	if len(forcedTxs) > 0 {
		batch.Transactions = append(forcedTxs, batch.Transactions...)
		c.logger.Debug().
			Int("forced_tx_count", len(forcedTxs)).
			Int("total_tx_count", len(batch.Transactions)).
			Msg("prepended forced inclusion transactions to batch")
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

		if !seqcommon.ValidateBlobSize(pendingTx.Data, maxBytes) {
			c.logger.Warn().
				Uint64("original_height", pendingTx.OriginalHeight).
				Int("blob_size", txSize).
				Msg("pending forced inclusion blob exceeds maximum size - skipping")
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

		if !seqcommon.ValidateBlobSize(tx, maxBytes) {
			c.logger.Warn().
				Uint64("da_height", event.StartDaHeight).
				Int("blob_size", txSize).
				Msg("forced inclusion blob exceeds maximum size - skipping")
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

	c.logger.Info().
		Int("processed_tx_count", len(validatedTxs)).
		Int("pending_tx_count", len(newPendingTxs)).
		Int("current_size", currentSize).
		Msg("completed processing forced inclusion transactions")

	return validatedTxs
}
