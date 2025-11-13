package based

import (
	"context"
	"errors"
	"sync"
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

	mu       sync.RWMutex
	daHeight uint64
	txQueue  [][]byte
}

// NewBasedSequencer creates a new based sequencer instance
func NewBasedSequencer(
	daRetriever DARetriever,
	da coreda.DA,
	config config.Config,
	genesis genesis.Genesis,
	logger zerolog.Logger,
) *BasedSequencer {
	return &BasedSequencer{
		daRetriever: daRetriever,
		da:          da,
		config:      config,
		genesis:     genesis,
		logger:      logger.With().Str("component", "based_sequencer").Logger(),
		daHeight:    genesis.DAStartHeight,
		txQueue:     make([][]byte, 0),
	}
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
	s.mu.Lock()
	defer s.mu.Unlock()

	// If we have transactions in the queue, return them first
	if len(s.txQueue) > 0 {
		batch := s.createBatchFromQueue(req.MaxBytes)
		if len(batch.Transactions) > 0 {
			s.logger.Debug().
				Int("tx_count", len(batch.Transactions)).
				Int("remaining", len(s.txQueue)).
				Msg("returning batch from queue")
			return &coresequencer.GetNextBatchResponse{
				Batch:     batch,
				Timestamp: time.Now(),
				BatchData: req.LastBatchData,
			}, nil
		}
	}

	// Fetch forced inclusion transactions from DA
	s.logger.Debug().Uint64("da_height", s.daHeight).Msg("fetching forced inclusion transactions from DA")

	forcedTxsEvent, err := s.daRetriever.RetrieveForcedIncludedTxsFromDA(ctx, s.daHeight)
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
				Uint64("da_height", s.daHeight).
				Msg("DA height from future, waiting for DA to produce block")
			return &coresequencer.GetNextBatchResponse{
				Batch:     &coresequencer.Batch{Transactions: nil},
				Timestamp: time.Now(),
				BatchData: req.LastBatchData,
			}, nil
		}

		s.logger.Error().Err(err).Uint64("da_height", s.daHeight).Msg("failed to retrieve forced inclusion transactions")
		return nil, err
	}

	// Update DA height based on the retrieved event
	if forcedTxsEvent.EndDaHeight > s.daHeight {
		s.daHeight = forcedTxsEvent.EndDaHeight
	} else if forcedTxsEvent.StartDaHeight > s.daHeight {
		s.daHeight = forcedTxsEvent.StartDaHeight
	}

	// Add transactions to queue
	s.txQueue = append(s.txQueue, forcedTxsEvent.Txs...)

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
	if len(s.txQueue) == 0 {
		return &coresequencer.Batch{Transactions: nil}
	}

	var batch [][]byte
	var totalBytes uint64

	for i, tx := range s.txQueue {
		txSize := uint64(len(tx))
		if totalBytes+txSize > maxBytes && len(batch) > 0 {
			// Would exceed max bytes, stop here
			s.txQueue = s.txQueue[i:]
			break
		}

		batch = append(batch, tx)
		totalBytes += txSize

		// If this is the last transaction, clear the queue
		if i == len(s.txQueue)-1 {
			s.txQueue = s.txQueue[:0]
		}
	}

	return &coresequencer.Batch{Transactions: batch}
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
	s.mu.Lock()
	defer s.mu.Unlock()
	s.daHeight = height
	s.logger.Debug().Uint64("da_height", height).Msg("DA height updated")
}

// GetDAHeight returns the current DA height
func (s *BasedSequencer) GetDAHeight() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.daHeight
}
