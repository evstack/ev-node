package reaping

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/executing"
	coreexecutor "github.com/evstack/ev-node/core/execution"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/genesis"
)

const (
	// MaxBackoffInterval is the maximum backoff interval for retries
	MaxBackoffInterval = 30 * time.Second
)

// Reaper is responsible for periodically retrieving transactions from the executor,
// filtering out already seen transactions, and submitting new transactions to the sequencer.
type Reaper struct {
	exec      coreexecutor.Executor
	sequencer coresequencer.Sequencer
	chainID   string
	interval  time.Duration
	cache     cache.CacheManager
	executor  *executing.Executor

	// shared components
	logger zerolog.Logger

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewReaper creates a new Reaper instance.
func NewReaper(
	exec coreexecutor.Executor,
	sequencer coresequencer.Sequencer,
	genesis genesis.Genesis,
	logger zerolog.Logger,
	executor *executing.Executor,
	cache cache.CacheManager,
	scrapeInterval time.Duration,
) (*Reaper, error) {
	if executor == nil {
		return nil, errors.New("executor cannot be nil")
	}
	if cache == nil {
		return nil, errors.New("cache cannot be nil")
	}
	if scrapeInterval == 0 {
		return nil, errors.New("scrape interval cannot be empty")
	}

	return &Reaper{
		exec:      exec,
		sequencer: sequencer,
		chainID:   genesis.ChainID,
		interval:  scrapeInterval,
		logger:    logger.With().Str("component", "reaper").Logger(),
		cache:     cache,
		executor:  executor,
	}, nil
}

// Start begins the execution component
func (r *Reaper) Start(ctx context.Context) error {
	r.ctx, r.cancel = context.WithCancel(ctx)

	// Start reaper loop
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.reaperLoop()
	}()

	r.logger.Info().Dur("interval", r.interval).Msg("reaper started")
	return nil
}

func (r *Reaper) reaperLoop() {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	cleanupTicker := time.NewTicker(1 * time.Hour)
	defer cleanupTicker.Stop()

	consecutiveFailures := 0

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			err := r.SubmitTxs()
			if err != nil {
				// Increment failure counter and apply exponential backoff
				consecutiveFailures++
				backoff := r.interval * time.Duration(1<<min(consecutiveFailures, 5)) // Cap at 2^5 = 32x
				backoff = min(backoff, MaxBackoffInterval)
				r.logger.Warn().
					Err(err).
					Int("consecutive_failures", consecutiveFailures).
					Dur("next_retry_in", backoff).
					Msg("reaper encountered error, applying backoff")

				// Reset ticker with backoff interval
				ticker.Reset(backoff)
			} else {
				// Reset failure counter and backoff on success
				if consecutiveFailures > 0 {
					r.logger.Info().Msg("reaper recovered from errors, resetting backoff")
					consecutiveFailures = 0
					ticker.Reset(r.interval)
				}
			}
		case <-cleanupTicker.C:
			// Clean up transaction hashes older than 24 hours
			// This prevents unbounded growth of the transaction seen cache
			removed := r.cache.CleanupOldTxs(cache.DefaultTxCacheRetention)
			if removed > 0 {
				r.logger.Info().Int("removed", removed).Msg("cleaned up old transaction hashes")
			}
		}
	}
}

// Stop shuts down the reaper component
func (r *Reaper) Stop() error {
	if r.cancel != nil {
		r.cancel()
	}
	r.wg.Wait()

	r.logger.Info().Msg("reaper stopped")
	return nil
}

// SubmitTxs retrieves transactions from the executor and submits them to the sequencer.
// Returns an error if any critical operation fails.
func (r *Reaper) SubmitTxs() error {
	txs, err := r.exec.GetTxs(r.ctx)
	if err != nil {
		r.logger.Error().Err(err).Msg("failed to get txs from executor")
		return fmt.Errorf("failed to get txs from executor: %w", err)
	}
	if len(txs) == 0 {
		r.logger.Debug().Msg("no new txs")
		return nil
	}

	var newTxs [][]byte
	for _, tx := range txs {
		txHash := hashTx(tx)
		if !r.cache.IsTxSeen(txHash) {
			newTxs = append(newTxs, tx)
		}
	}

	if len(newTxs) == 0 {
		r.logger.Debug().Msg("no new txs to submit")
		return nil
	}

	r.logger.Debug().Int("txCount", len(newTxs)).Msg("submitting txs to sequencer")

	_, err = r.sequencer.SubmitBatchTxs(r.ctx, coresequencer.SubmitBatchTxsRequest{
		Id:    []byte(r.chainID),
		Batch: &coresequencer.Batch{Transactions: newTxs},
	})
	if err != nil {
		return fmt.Errorf("failed to submit txs to sequencer: %w", err)
	}

	for _, tx := range newTxs {
		txHash := hashTx(tx)
		r.cache.SetTxSeen(txHash)
	}

	// Notify the executor that new transactions are available
	if len(newTxs) > 0 {
		r.logger.Debug().Msg("notifying executor of new transactions")
		r.executor.NotifyNewTransactions()
	}

	r.logger.Debug().Msg("successfully submitted txs")
	return nil
}

func hashTx(tx []byte) string {
	hash := sha256.Sum256(tx)
	return hex.EncodeToString(hash[:])
}
