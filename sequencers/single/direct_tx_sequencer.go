package single

import (
	"context"
	ds "github.com/ipfs/go-datastore"
	"sync"
	"time"

	logging "github.com/ipfs/go-log/v2"

	coresequencer "github.com/evstack/ev-node/core/sequencer"
)

// DirectTxSequencer decorates a sequencer to handle direct transactions.
// It implements the DirectTxSequencer interface which extends the core sequencer interface
// with a method for handling direct transactions.
var _ coresequencer.DirectTxSequencer = &DirectTxSequencer{}

type DirectTxSequencer struct {
	sequencer coresequencer.Sequencer
	logger    logging.EventLogger

	// Separate queue for direct transactions
	directTxQueue *BatchQueue
	directTxMu    sync.Mutex
}

// NewDirectTxSequencer creates a new DirectTxSequencer that wraps the given sequencer.
func NewDirectTxSequencer(
	sequencer coresequencer.Sequencer,
	logger logging.EventLogger,
	datastore ds.Batching,
	maxSize int,
) *DirectTxSequencer {
	return &DirectTxSequencer{
		sequencer:     sequencer,
		logger:        logger,
		directTxQueue: NewBatchQueue(datastore, "direct_txs", maxSize),
	}
}

// SubmitBatchTxs implements the coresequencer.Sequencer interface.
// It delegates to the wrapped sequencer.
func (d *DirectTxSequencer) SubmitBatchTxs(ctx context.Context, req coresequencer.SubmitBatchTxsRequest) (*coresequencer.SubmitBatchTxsResponse, error) {
	return d.sequencer.SubmitBatchTxs(ctx, req)
}

// GetNextBatch implements the coresequencer.Sequencer interface.
// It first checks for direct transactions, and if there are none, delegates to the wrapped sequencer.
func (d *DirectTxSequencer) GetNextBatch(ctx context.Context, req coresequencer.GetNextBatchRequest) (*coresequencer.GetNextBatchResponse, error) {
	// First check if there are any direct transactions
	d.directTxMu.Lock()
	directBatch, err := d.directTxQueue.Next(ctx)
	d.directTxMu.Unlock()

	if err != nil {
		return nil, err
	}

	// If there are direct transactions, return them
	if directBatch != nil && len(directBatch.Transactions) > 0 {
		d.logger.Debug("Returning direct transactions batch", "txCount", len(directBatch.Transactions))
		return &coresequencer.GetNextBatchResponse{
			Batch:     directBatch,
			Timestamp: time.Now(), // Use current time as the timestamp
		}, nil
	}

	// Otherwise, delegate to the wrapped sequencer
	return d.sequencer.GetNextBatch(ctx, req)
}

// VerifyBatch implements the coresequencer.Sequencer interface.
// It delegates to the wrapped sequencer.
func (d *DirectTxSequencer) VerifyBatch(ctx context.Context, req coresequencer.VerifyBatchRequest) (*coresequencer.VerifyBatchResponse, error) {
	return d.sequencer.VerifyBatch(ctx, req)
}

// SubmitDirectTxs adds direct transactions to the direct transaction queue.
// This method is called by the DirectTxReaper.
func (d *DirectTxSequencer) SubmitDirectTxs(ctx context.Context, txs [][]byte) error {
	if len(txs) == 0 {
		return nil
	}

	d.logger.Debug("Adding direct transactions to queue", "txCount", len(txs))

	d.directTxMu.Lock()
	defer d.directTxMu.Unlock()

	batch := coresequencer.Batch{Transactions: txs}
	return d.directTxQueue.AddBatch(ctx, batch)
}
