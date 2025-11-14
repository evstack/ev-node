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
)

// ForcedInclusionEvent represents forced inclusion transactions retrieved from DA
type ForcedInclusionEvent = struct {
	Txs           [][]byte
	StartDaHeight uint64
	EndDaHeight   uint64
}

// DARetriever defines the interface for retrieving forced inclusion transactions from DA
// This interface is intentionally generic to allow different implementations
type DARetriever interface {
	RetrieveForcedIncludedTxsFromDA(ctx context.Context, daHeight uint64) (*ForcedInclusionEvent, error)
	SetDAHeight(height uint64)
}

var _ coresequencer.Sequencer = (*BasedSequencer)(nil)

// BasedSequencer is a sequencer that only retrieves transactions from the DA layer
// via the forced inclusion mechanism. It does not accept transactions from the reaper.
type BasedSequencer struct {
	daRetriever DARetriever
	da          coreda.DA
	config      config.Config
	genesis     genesis.Genesis
	logger      zerolog.Logger

	daHeight atomic.Uint64
	txQueue  atomic.Pointer[[][]byte]
}

// NewBasedSequencer creates a new based sequencer instance
func NewBasedSequencer(
	daRetriever DARetriever,
	da coreda.DA,
	config config.Config,
	genesis genesis.Genesis,
	logger zerolog.Logger,
) *BasedSequencer {
	s := &BasedSequencer{
		daRetriever: daRetriever,
		da:          da,
		config:      config,
		genesis:     genesis,
		logger:      logger.With().Str("component", "based_sequencer").Logger(),
	}
	s.daHeight.Store(genesis.DAStartHeight)
	initialQueue := make([][]byte, 0)
	s.txQueue.Store(&initialQueue)
	return s
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
			queuePtr := s.txQueue.Load()
			queue := *queuePtr
			s.logger.Debug().
				Int("tx_count", len(batch.Transactions)).
				Int("remaining", len(queue)).
				Msg("returning batch from queue")
			return &coresequencer.GetNextBatchResponse{
				Batch:     batch,
				Timestamp: time.Now(),
				BatchData: req.LastBatchData,
			}, nil
		}
	}

	// Fetch forced inclusion transactions from DA
	currentHeight := s.daHeight.Load()
	s.logger.Debug().Uint64("da_height", currentHeight).Msg("fetching forced inclusion transactions from DA")

	forcedTxsEvent, err := s.daRetriever.RetrieveForcedIncludedTxsFromDA(ctx, currentHeight)
	if err != nil {
		if errors.Is(err, block.ErrForceInclusionNotConfigured) {
			s.logger.Error().Msg("forced inclusion not configured, returning empty batch")
			return &coresequencer.GetNextBatchResponse{
				Batch:     &coresequencer.Batch{Transactions: nil},
				Timestamp: time.Now(),
				BatchData: req.LastBatchData,
			}, nil
		}

		// If we get a height from future error, keep the current DA height and return batch
		// We'll retry the same height on the next call until DA produces that block
		if errors.Is(err, coreda.ErrHeightFromFuture) {
			s.logger.Debug().
				Uint64("da_height", currentHeight).
				Msg("DA height from future, waiting for DA to produce block")
			return &coresequencer.GetNextBatchResponse{
				Batch:     &coresequencer.Batch{Transactions: nil},
				Timestamp: time.Now(),
				BatchData: req.LastBatchData,
			}, nil
		}

		s.logger.Error().Err(err).Uint64("da_height", currentHeight).Msg("failed to retrieve forced inclusion transactions")
		return nil, err
	}

	// Update DA height based on the retrieved event
	for {
		current := s.daHeight.Load()
		newHeight := current
		if forcedTxsEvent.EndDaHeight > current {
			newHeight = forcedTxsEvent.EndDaHeight
		} else if forcedTxsEvent.StartDaHeight > current {
			newHeight = forcedTxsEvent.StartDaHeight
		}
		if newHeight == current || s.daHeight.CompareAndSwap(current, newHeight) {
			break
		}
	}

	// Add transactions to queue
	for {
		oldQueuePtr := s.txQueue.Load()
		oldQueue := *oldQueuePtr
		newQueue := append(oldQueue, forcedTxsEvent.Txs...)
		if s.txQueue.CompareAndSwap(oldQueuePtr, &newQueue) {
			break
		}
	}

	s.logger.Info().
		Int("tx_count", len(forcedTxsEvent.Txs)).
		Uint64("da_height_start", forcedTxsEvent.StartDaHeight).
		Uint64("da_height_end", forcedTxsEvent.EndDaHeight).
		Msg("retrieved forced inclusion transactions from DA")

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
	if s.daRetriever != nil {
		s.daRetriever.SetDAHeight(height)
	}
	s.logger.Debug().Uint64("da_height", height).Msg("DA height updated")
}

// GetDAHeight returns the current DA height
func (s *BasedSequencer) GetDAHeight() uint64 {
	return s.daHeight.Load()
}
