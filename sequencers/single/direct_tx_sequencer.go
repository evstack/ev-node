package single

import (
	"context"
	"fmt"
	"sync"
	"time"

	ds "github.com/ipfs/go-datastore"

	logging "github.com/ipfs/go-log/v2"

	"github.com/evstack/ev-node/core/sequencer"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	rollconf "github.com/evstack/ev-node/pkg/config"
)

var _ coresequencer.DirectTxSequencer = &DirectTxSequencer{}

// DirectTxSequencer decorates a sequencer to handle direct transactions.
// It implements the DirectTxSequencer interface which extends the core sequencer interface
// with a method for handling direct transactions.
type DirectTxSequencer struct {
	sequencer coresequencer.Sequencer
	logger    logging.EventLogger

	directTxMu    sync.Mutex
	directTxQueue *DirectTXBatchQueue
	config        rollconf.ForcedInclusionConfig
}

// NewDirectTxSequencer creates a new DirectTxSequencer that wraps the given sequencer.
func NewDirectTxSequencer(
	sequencer coresequencer.Sequencer,
	logger logging.EventLogger,
	datastore ds.Batching,
	maxSize int,
	cfg rollconf.ForcedInclusionConfig,
) *DirectTxSequencer {
	return &DirectTxSequencer{
		sequencer:     sequencer,
		logger:        logger,
		directTxQueue: NewDirectTXBatchQueue(datastore, "direct_txs", maxSize),
		config:        cfg,
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
	d.logger.Info("+++ GetNextBatch", "current", req.DAIncludedHeight, "maxBytes", req.MaxBytes)

	var transactions [][]byte
	remainingBytes := int(req.MaxBytes)
	d.directTxMu.Lock()

	// First get direct transactions up to max size
	for remainingBytes > 0 {
		peekedTx, err := d.directTxQueue.Peek(ctx)
		if err != nil {
			d.directTxMu.Unlock()
			return nil, err
		}
		if peekedTx == nil || len(peekedTx.TX) > remainingBytes {
			d.logger.Info("+++ 1 GetNextBatch", "current", req.DAIncludedHeight, "peekedTx", peekedTx, "remainingBytes", remainingBytes)
			break
		}
		if false { // todo Alex: need some blocks in test to get past this da height
			// Require a minimum number of DA blocks to pass before including a direct transaction
			if req.DAIncludedHeight < peekedTx.FirstSeenHeight+d.config.MinDADelay {
				d.logger.Info("+++ skipping p3nding tx", "current", req.DAIncludedHeight, "expected", peekedTx.FirstSeenHeight+d.config.MinDADelay)
				break
			}
		}
		d.logger.Info("+++ adding p3nding tx", "current", req.DAIncludedHeight, "expected", peekedTx.FirstSeenHeight+d.config.MinDADelay)
		// Let full nodes enforce inclusion within a fixed period of time window
		// todo (Alex): what to do in this case? the sequencer may be down for unknown reasons.

		// pop from queue
		directTX, err := d.directTxQueue.Next(ctx)
		if err != nil {
			d.directTxMu.Unlock()
			return nil, err
		}
		transactions = append(transactions, directTX.TX)
		remainingBytes -= len(directTX.TX)
	}
	d.directTxMu.Unlock()
	d.logger.Info("+++ 2 GetNextBatch", "current", req.DAIncludedHeight)

	// use remaining space for regular transactions
	if remainingBytes > 0 {
		nextReq := coresequencer.GetNextBatchRequest{
			Id:            req.Id,
			LastBatchData: req.LastBatchData,
			MaxBytes:      uint64(remainingBytes),
		}
		resp, err := d.sequencer.GetNextBatch(ctx, nextReq)
		if err != nil {
			return nil, err
		}
		if resp != nil && resp.Batch != nil {
			transactions = append(transactions, resp.Batch.Transactions...)
		}
	}

	if len(transactions) == 0 {
		return nil, nil
	}

	return &coresequencer.GetNextBatchResponse{
		Batch:     &coresequencer.Batch{Transactions: transactions},
		Timestamp: time.Now().UTC(),
	}, nil
}

// VerifyBatch implements the coresequencer.Sequencer interface.
// It delegates to the wrapped sequencer.
func (d *DirectTxSequencer) VerifyBatch(ctx context.Context, req coresequencer.VerifyBatchRequest) (*coresequencer.VerifyBatchResponse, error) {
	return d.sequencer.VerifyBatch(ctx, req)
}

// SubmitDirectTxs adds direct transactions to the direct transaction queue.
// This method is called by the DirectTxReaper.
func (d *DirectTxSequencer) SubmitDirectTxs(ctx context.Context, txs ...sequencer.DirectTX) error {
	if len(txs) == 0 {
		return nil
	}
	for i, tx := range txs {
		if err := tx.ValidateBasic(); err != nil {
			return fmt.Errorf("tx %d: %w", i, err)
		}
	}
	d.logger.Debug("Adding direct transactions to queue", "txCount", len(txs))

	d.directTxMu.Lock()
	defer d.directTxMu.Unlock()
	return d.directTxQueue.AddBatch(ctx, txs...)
}
