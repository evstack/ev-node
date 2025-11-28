package single

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/celestiaorg/go-square/v3/share"
	ds "github.com/ipfs/go-datastore"
	"github.com/rs/zerolog"

	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/blob"
	datypes "github.com/evstack/ev-node/pkg/da/types"
)

// ErrInvalidId is returned when the chain id is invalid
var (
	ErrInvalidId = errors.New("invalid chain id")
)

var _ coresequencer.Sequencer = &Sequencer{}

// blobProofVerifier captures the blob proof APIs needed to validate inclusion.
type blobProofVerifier interface {
	GetProof(ctx context.Context, height uint64, namespace share.Namespace, commitment blob.Commitment) (*blob.Proof, error)
	Included(ctx context.Context, height uint64, namespace share.Namespace, proof *blob.Proof, commitment blob.Commitment) (bool, error)
}

// Sequencer implements core sequencing interface
type Sequencer struct {
	logger zerolog.Logger

	proposer bool

	Id        []byte
	verifier  blobProofVerifier
	namespace share.Namespace

	batchTime time.Duration

	queue *BatchQueue // single queue for immediate availability

	metrics *Metrics
}

// NewSequencer creates a new Single Sequencer
func NewSequencer(
	ctx context.Context,
	logger zerolog.Logger,
	db ds.Batching,
	verifier blobProofVerifier,
	id []byte,
	namespace []byte,
	batchTime time.Duration,
	metrics *Metrics,
	proposer bool,
) (*Sequencer, error) {
	return NewSequencerWithQueueSize(ctx, logger, db, verifier, id, namespace, batchTime, metrics, proposer, 1000)
}

// NewSequencerWithQueueSize creates a new Single Sequencer with configurable queue size
func NewSequencerWithQueueSize(
	ctx context.Context,
	logger zerolog.Logger,
	db ds.Batching,
	verifier blobProofVerifier,
	id []byte,
	namespace []byte,
	batchTime time.Duration,
	metrics *Metrics,
	proposer bool,
	maxQueueSize int,
) (*Sequencer, error) {
	if verifier == nil {
		return nil, fmt.Errorf("blob proof verifier is nil")
	}

	ns, err := share.NewNamespaceFromBytes(namespace)
	if err != nil {
		return nil, fmt.Errorf("invalid namespace: %w", err)
	}

	s := &Sequencer{
		logger:    logger,
		verifier:  verifier,
		namespace: ns,
		batchTime: batchTime,
		Id:        id,
		queue:     NewBatchQueue(db, "batches", maxQueueSize),
		metrics:   metrics,
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

// RecordMetrics updates the metrics with the given values.
// This method is intended to be called by the block manager after submitting data to the DA layer.
func (c *Sequencer) RecordMetrics(gasPrice float64, blobSize uint64, statusCode datypes.StatusCode, numPendingBlocks uint64, includedBlockHeight uint64) {
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
		for _, id := range req.BatchData {
			height, commitment := blob.SplitID(id)
			if commitment == nil {
				return nil, fmt.Errorf("invalid blob ID: %x", id)
			}

			proof, err := c.verifier.GetProof(ctx, height, c.namespace, commitment)
			if err != nil {
				return nil, fmt.Errorf("failed to get proof: %w", err)
			}

			included, err := c.verifier.Included(ctx, height, c.namespace, proof, commitment)
			if err != nil {
				return nil, fmt.Errorf("failed to verify inclusion: %w", err)
			}

			if !included {
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
