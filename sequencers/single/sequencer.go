package single

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	ds "github.com/ipfs/go-datastore"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block"
	coreda "github.com/evstack/ev-node/core/da"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/genesis"
)

var (
	// ErrInvalidId is returned when the chain id is invalid
	ErrInvalidId = errors.New("invalid chain id")
)

// ForcedInclusionEvent represents forced inclusion transactions retrieved from DA
type ForcedInclusionEvent = struct {
	Txs           [][]byte
	StartDaHeight uint64
	EndDaHeight   uint64
}

// DARetriever defines the interface for retrieving forced inclusion transactions from DA
type DARetriever interface {
	RetrieveForcedIncludedTxsFromDA(ctx context.Context, daHeight uint64) (*ForcedInclusionEvent, error)
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
	daRetriever DARetriever
	genesis     genesis.Genesis
	mu          sync.RWMutex
	daHeight    uint64
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
	daRetriever DARetriever,
	gen genesis.Genesis,
) (*Sequencer, error) {
	s := &Sequencer{
		logger:      logger,
		da:          da,
		batchTime:   batchTime,
		Id:          id,
		queue:       NewBatchQueue(db, "batches", maxQueueSize),
		metrics:     metrics,
		proposer:    proposer,
		daRetriever: daRetriever,
		genesis:     gen,
		daHeight:    gen.DAStartHeight,
	}

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

	// Retrieve forced inclusion transactions if DARetriever is configured
	var forcedTxs [][]byte
	if c.daRetriever != nil {
		c.mu.Lock()
		currentDAHeight := c.daHeight
		c.mu.Unlock()

		forcedEvent, err := c.daRetriever.RetrieveForcedIncludedTxsFromDA(ctx, currentDAHeight)
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
		} else {
			forcedTxs = forcedEvent.Txs

			// Update DA height based on the retrieved event
			c.mu.Lock()
			if forcedEvent.EndDaHeight > c.daHeight {
				c.daHeight = forcedEvent.EndDaHeight
			} else if forcedEvent.StartDaHeight > c.daHeight {
				c.daHeight = forcedEvent.StartDaHeight
			}
			c.mu.Unlock()

			c.logger.Info().
				Int("tx_count", len(forcedEvent.Txs)).
				Uint64("da_height_start", forcedEvent.StartDaHeight).
				Uint64("da_height_end", forcedEvent.EndDaHeight).
				Msg("retrieved forced inclusion transactions from DA")
		}
	}

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
	c.mu.Lock()
	defer c.mu.Unlock()
	c.daHeight = height
	c.logger.Debug().Uint64("da_height", height).Msg("DA height updated")
}

// GetDAHeight returns the current DA height
func (c *Sequencer) GetDAHeight() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.daHeight
}
