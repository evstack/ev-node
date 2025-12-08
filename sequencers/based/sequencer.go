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
	txQueue  [][]byte
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
		txQueue:     make([][]byte, 0),
	}
	bs.SetDAHeight(genesis.DAStartHeight) // will be overridden by the executor

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
	currentDAHeight := s.GetDAHeight()

	s.logger.Debug().Uint64("da_height", currentDAHeight).Msg("fetching forced inclusion transactions from DA")

	forcedTxsEvent, err := s.fiRetriever.RetrieveForcedIncludedTxs(ctx, currentDAHeight)
	if err != nil {
		// Check if forced inclusion is not configured
		if errors.Is(err, block.ErrForceInclusionNotConfigured) {
			return nil, errors.New("forced inclusion not configured")
		} else if errors.Is(err, coreda.ErrHeightFromFuture) {
			// If we get a height from future error, keep the current DA height and return batch
			// We'll retry the same height on the next call until DA produces that block
			s.logger.Debug().
				Uint64("da_height", currentDAHeight).
				Msg("DA height from future, waiting for DA to produce block")
		} else {
			s.logger.Error().Err(err).Uint64("da_height", currentDAHeight).Msg("failed to retrieve forced inclusion transactions")
			return nil, err
		}
	} else {
		// Update DA height.
		// If we are in between epochs, we still need to bump the da height.
		// At the end of an epoch, we need to bump to go to the next epoch.
		if forcedTxsEvent.EndDaHeight >= currentDAHeight {
			s.SetDAHeight(forcedTxsEvent.EndDaHeight + 1)
		}
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
		s.txQueue = append(s.txQueue, tx)
		validTxs++
	}

	s.logger.Info().
		Int("valid_tx_count", validTxs).
		Int("skipped_tx_count", skippedTxs).
		Int("queue_size", len(s.txQueue)).
		Uint64("da_height_start", forcedTxsEvent.StartDaHeight).
		Uint64("da_height_end", forcedTxsEvent.EndDaHeight).
		Msg("processed forced inclusion transactions from DA")

	batch := s.createBatchFromQueue(req.MaxBytes)

	return &coresequencer.GetNextBatchResponse{
		Batch:     batch,
		Timestamp: time.Time{}, // TODO(@julienrbrt): we need to use DA block timestamp for determinism
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
		// Always respect maxBytes, even for the first transaction
		if totalBytes+txSize > maxBytes {
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

	// Mark all transactions as force-included since based sequencer only pulls from DA
	forceIncludedMask := make([]bool, len(batch))
	for i := range forceIncludedMask {
		forceIncludedMask[i] = true
	}

	return &coresequencer.Batch{
		Transactions:      batch,
		ForceIncludedMask: forceIncludedMask,
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
func (c *BasedSequencer) SetDAHeight(height uint64) {
	c.daHeight.Store(height)
	c.logger.Debug().Uint64("da_height", height).Msg("DA height updated")
}

// GetDAHeight returns the current DA height
func (c *BasedSequencer) GetDAHeight() uint64 {
	return c.daHeight.Load()
}
