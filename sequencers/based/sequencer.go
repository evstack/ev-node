package based

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block"
	coreda "github.com/evstack/ev-node/core/da"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	seqcommon "github.com/evstack/ev-node/sequencers/common"
)

// ForcedInclusionRetriever defines the interface for retrieving forced inclusion transactions from DA
type ForcedInclusionRetriever interface {
	RetrieveForcedIncludedTxs(ctx context.Context, daHeight uint64) (*block.ForcedInclusionEvent, error)
}

var _ coresequencer.Sequencer = (*BasedSequencer)(nil)

// BasedSequencer is a sequencer that only retrieves transactions from the DA layer
// via the forced inclusion mechanism. It does not accept transactions from the reaper.
type BasedSequencer struct {
	fiRetriever ForcedInclusionRetriever
	da          coreda.DA
	config      config.Config
	genesis     genesis.Genesis
	logger      zerolog.Logger

	daHeight atomic.Uint64
	txQueue  atomic.Pointer[[][]byte]
}

// NewBasedSequencer creates a new based sequencer instance
func NewBasedSequencer(
	fiRetriever ForcedInclusionRetriever,
	da coreda.DA,
	config config.Config,
	genesis genesis.Genesis,
	logger zerolog.Logger,
) *BasedSequencer {
	bs := &BasedSequencer{
		fiRetriever: fiRetriever,
		da:          da,
		config:      config,
		genesis:     genesis,
		logger:      logger.With().Str("component", "based_sequencer").Logger(),
	}
	bs.daHeight.Store(genesis.DAStartHeight)
	initialQueue := make([][]byte, 0)
	bs.txQueue.Store(&initialQueue)
	return bs
}

// SubmitBatchTxs does nothing for a based sequencer as it only pulls from DA
// This satisfies the Sequencer interface but transactions submitted here are ignored
func (s *BasedSequencer) SubmitBatchTxs(ctx context.Context, req coresequencer.SubmitBatchTxsRequest) (*coresequencer.SubmitBatchTxsResponse, error) {
	s.logger.Debug().Msg("based sequencer ignores submitted transactions - only DA transactions are processed")
	return &coresequencer.SubmitBatchTxsResponse{}, nil
}

// GetNextBatch retrieves the next batch of transactions from the DA layer
// It fetches forced inclusion transactions and returns them as the next batch
func (s *BasedSequencer) GetNextBatch(ctx context.Context, req coresequencer.GetNextBatchRequest) (*coresequencer.GetNextBatchResponse, error) {
	// If we have transactions in the queue, return them first
	queuePtr := s.txQueue.Load()
	queue := *queuePtr
	if len(queue) > 0 {
		batch := s.createBatchFromQueue(req.MaxBytes)
		if len(batch.Transactions) > 0 {
			s.logger.Debug().
				Int("tx_count", len(batch.Transactions)).
				Int("remaining", len(*s.txQueue.Load())).
				Msg("returning batch from queue")
			return &coresequencer.GetNextBatchResponse{
				Batch:     batch,
				Timestamp: time.Now(),
				BatchData: req.LastBatchData,
			}, nil
		}
	}

	// Fetch forced inclusion transactions from DA
	currentDAHeight := s.daHeight.Load()
	s.logger.Debug().Uint64("da_height", currentDAHeight).Msg("fetching forced inclusion transactions from DA")

	forcedTxsEvent, err := s.fiRetriever.RetrieveForcedIncludedTxs(ctx, currentDAHeight)
	if err != nil {
		// Check if forced inclusion is not configured
		if errors.Is(err, block.ErrForceInclusionNotConfigured) {
			s.logger.Error().Msg("forced inclusion not configured, returning empty batch")
			return &coresequencer.GetNextBatchResponse{
				Batch:     &coresequencer.Batch{Transactions: nil},
				Timestamp: time.Now(),
				BatchData: req.LastBatchData,
			}, nil
		} else if errors.Is(err, coreda.ErrHeightFromFuture) {
			// If we get a height from future error, keep the current DA height and return batch
			// We'll retry the same height on the next call until DA produces that block
			s.logger.Debug().
				Uint64("da_height", currentDAHeight).
				Msg("DA height from future, waiting for DA to produce block")
			return &coresequencer.GetNextBatchResponse{
				Batch:     &coresequencer.Batch{Transactions: nil},
				Timestamp: time.Now(),
				BatchData: req.LastBatchData,
			}, nil
		}

		s.logger.Error().Err(err).Uint64("da_height", currentDAHeight).Msg("failed to retrieve forced inclusion transactions")
		return nil, err
	}

	// Update DA height based on the retrieved event
	if forcedTxsEvent.EndDaHeight > currentDAHeight {
		s.SetDAHeight(forcedTxsEvent.EndDaHeight)
	} else if forcedTxsEvent.StartDaHeight > currentDAHeight {
		s.SetDAHeight(forcedTxsEvent.StartDaHeight)
	}

	// Add forced inclusion transactions to the queue with validation
	validTxs := 0
	skippedTxs := 0
	for _, tx := range forcedTxsEvent.Txs {
		// Validate blob size against absolute maximum
		if !seqcommon.ValidateBlobSize(tx) {
			s.logger.Warn().
				Uint64("da_height", forcedTxsEvent.StartDaHeight).
				Int("blob_size", len(tx)).
				Msg("forced inclusion blob exceeds absolute maximum size - skipping")
			skippedTxs++
			continue
		}

		// Add to queue atomically
		for {
			oldQueuePtr := s.txQueue.Load()
			oldQueue := *oldQueuePtr
			newQueue := append(oldQueue, tx)
			if s.txQueue.CompareAndSwap(oldQueuePtr, &newQueue) {
				validTxs++
				break
			}
		}
	}

	s.logger.Info().
		Int("valid_tx_count", validTxs).
		Int("skipped_tx_count", skippedTxs).
		Int("queue_size", len(*s.txQueue.Load())).
		Uint64("da_height_start", forcedTxsEvent.StartDaHeight).
		Uint64("da_height_end", forcedTxsEvent.EndDaHeight).
		Msg("processed forced inclusion transactions from DA")

	batch := s.createBatchFromQueue(req.MaxBytes)

	return &coresequencer.GetNextBatchResponse{
		Batch:     batch,
		Timestamp: time.Now(),
		BatchData: req.LastBatchData,
	}, nil
}

// createBatchFromQueue creates a batch from the transaction queue respecting MaxBytes
func (s *BasedSequencer) createBatchFromQueue(maxBytes uint64) *coresequencer.Batch {
	for {
		queuePtr := s.txQueue.Load()
		queue := *queuePtr
		if len(queue) == 0 {
			return &coresequencer.Batch{Transactions: nil}
		}

		var batch [][]byte
		var totalBytes uint64
		var remaining [][]byte

		for i, tx := range queue {
			txSize := uint64(len(tx))
			if totalBytes+txSize > maxBytes && len(batch) > 0 {
				// Would exceed max bytes, stop here
				remaining = queue[i:]
				break
			}

			batch = append(batch, tx)
			totalBytes += txSize

			// If this is the last transaction, clear the queue
			if i == len(queue)-1 {
				remaining = nil
			}
		}

		// Try to update queue atomically
		if s.txQueue.CompareAndSwap(queuePtr, &remaining) {
			return &coresequencer.Batch{Transactions: batch}
		}
		// If CAS failed, retry with new queue state
	}
}

// VerifyBatch verifies a batch of transactions
// For a based sequencer, we always return true as all transactions come from DA
func (s *BasedSequencer) VerifyBatch(ctx context.Context, req coresequencer.VerifyBatchRequest) (*coresequencer.VerifyBatchResponse, error) {
	return &coresequencer.VerifyBatchResponse{
		Status: true,
	}, nil
}

// SetDAHeight sets the current DA height for the sequencer
// This should be called when the sequencer needs to sync to a specific DA height
func (s *BasedSequencer) SetDAHeight(height uint64) {
	s.daHeight.Store(height)
	s.logger.Debug().Uint64("da_height", height).Msg("DA height updated")
}

// GetDAHeight returns the current DA height
func (s *BasedSequencer) GetDAHeight() uint64 {
	return s.daHeight.Load()
}
