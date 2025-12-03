package single

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	ds "github.com/ipfs/go-datastore"
	"github.com/rs/zerolog"

	coreda "github.com/evstack/ev-node/core/da"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
)

// ErrInvalidId is returned when the chain id is invalid
var (
	ErrInvalidId = errors.New("invalid chain id")
)

var _ coresequencer.Sequencer = &Sequencer{}

// Sequencer implements core sequencing interface
type Sequencer struct {
	logger zerolog.Logger

	proposer bool

	Id []byte
	da coreda.DA

	batchTime time.Duration
	queue     *BatchQueue // single queue for immediate availability
}

// NewSequencer creates a new Single Sequencer
func NewSequencer(
	ctx context.Context,
	logger zerolog.Logger,
	db ds.Batching,
	da coreda.DA,
	id []byte,
	batchTime time.Duration,
	proposer bool,
) (*Sequencer, error) {
	return NewSequencerWithQueueSize(ctx, logger, db, da, id, batchTime, proposer, 1000)
}

// NewSequencerWithQueueSize creates a new Single Sequencer with configurable queue size
func NewSequencerWithQueueSize(
	ctx context.Context,
	logger zerolog.Logger,
	db ds.Batching,
	da coreda.DA,
	id []byte,
	batchTime time.Duration,
	proposer bool,
	maxQueueSize int,
) (*Sequencer, error) {
	s := &Sequencer{
		logger:    logger,
		da:        da,
		batchTime: batchTime,
		Id:        id,
		queue:     NewBatchQueue(db, "batches", maxQueueSize),
		proposer:  proposer,
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

	batch, err := c.queue.Next(ctx)
	if err != nil {
		return nil, err
	}

	return &coresequencer.GetNextBatchResponse{
		Batch:     batch,
		Timestamp: time.Now(),
	}, nil
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
