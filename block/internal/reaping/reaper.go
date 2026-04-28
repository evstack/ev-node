package reaping

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block/internal/cache"
	coreexecutor "github.com/evstack/ev-node/core/execution"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/genesis"
)

const (
	// MaxBackoffInterval is the maximum backoff interval for retries
	MaxBackoffInterval = 30 * time.Second

	// CleanupInterval is how often the reaper sweeps expired hashes
	// out of the seen-tx cache.
	CleanupInterval = max(cache.DefaultTxCacheRetention/10, 15*time.Second)
)

// Reaper is responsible for periodically retrieving transactions from the executor,
// filtering out already seen transactions, and submitting new transactions to the sequencer.
type Reaper struct {
	exec           coreexecutor.Executor
	sequencer      coresequencer.Sequencer
	chainID        string
	interval       time.Duration
	cache          cache.CacheManager
	onTxsSubmitted func()

	logger zerolog.Logger

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewReaper(
	exec coreexecutor.Executor,
	sequencer coresequencer.Sequencer,
	genesis genesis.Genesis,
	logger zerolog.Logger,
	cache cache.CacheManager,
	scrapeInterval time.Duration,
	onTxsSubmitted func(),
) (*Reaper, error) {
	if cache == nil {
		return nil, errors.New("cache cannot be nil")
	}
	if scrapeInterval == 0 {
		return nil, errors.New("scrape interval cannot be empty")
	}

	return &Reaper{
		exec:           exec,
		sequencer:      sequencer,
		chainID:        genesis.ChainID,
		interval:       scrapeInterval,
		logger:         logger.With().Str("component", "reaper").Logger(),
		cache:          cache,
		onTxsSubmitted: onTxsSubmitted,
	}, nil
}

// Start begins the execution component
func (r *Reaper) Start(ctx context.Context) error {
	if r.cancel != nil {
		return errors.New("reaper already started")
	}
	r.ctx, r.cancel = context.WithCancel(ctx)

	r.wg.Go(r.reaperLoop)

	r.logger.Info().Dur("idle_interval", r.interval).Msg("reaper started")
	return nil
}

func (r *Reaper) reaperLoop() {
	cleanupTicker := time.NewTicker(CleanupInterval)
	defer cleanupTicker.Stop()

	consecutiveFailures := 0

	for {
		submitted, err := r.drainMempool(cleanupTicker.C)

		if err != nil && r.ctx.Err() == nil {
			consecutiveFailures++
			backoff := r.interval * time.Duration(1<<min(consecutiveFailures, 5))
			backoff = min(backoff, MaxBackoffInterval)
			r.logger.Warn().
				Err(err).
				Int("consecutive_failures", consecutiveFailures).
				Dur("next_retry_in", backoff).
				Msg("reaper error, backing off")
			if r.wait(backoff) {
				return
			}
			continue
		}

		if consecutiveFailures > 0 {
			r.logger.Info().Msg("reaper recovered from errors")
			consecutiveFailures = 0
		}

		if submitted {
			runtime.Gosched()
			continue
		}

		if r.wait(r.interval) {
			return
		}
	}
}

// wait blocks for the given duration. Returns true if the context was cancelled.
func (r *Reaper) wait(d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-r.ctx.Done():
		return true
	case <-timer.C:
		return false
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

func (r *Reaper) drainMempool(cleanupCh <-chan time.Time) (bool, error) {
	var totalSubmitted int

	defer func() {
		if totalSubmitted > 0 && r.onTxsSubmitted != nil {
			r.onTxsSubmitted()
		}
	}()

	for {
		select {
		case <-cleanupCh:
			removed := r.cache.CleanupOldTxs(cache.DefaultTxCacheRetention)
			if removed > 0 {
				r.logger.Info().Int("removed", removed).Msg("cleaned up old transaction hashes")
			}
		default:
		}

		txs, err := r.exec.GetTxs(r.ctx)
		if err != nil {
			return totalSubmitted > 0, fmt.Errorf("failed to get txs from executor: %w", err)
		}
		if len(txs) == 0 {
			break
		}

		hashes := hashTxs(txs)
		seen := r.cache.AreTxsSeen(hashes)

		newTxs := make([][]byte, 0, len(txs))
		newHashes := make([]string, 0, len(txs))
		for i, tx := range txs {
			if !seen[i] {
				newTxs = append(newTxs, tx)
				newHashes = append(newHashes, hashes[i])
			}
		}

		if len(newTxs) == 0 {
			break
		}

		_, err = r.sequencer.SubmitBatchTxs(r.ctx, coresequencer.SubmitBatchTxsRequest{
			Id:    []byte(r.chainID),
			Batch: &coresequencer.Batch{Transactions: newTxs},
		})
		if err != nil {
			return totalSubmitted > 0, fmt.Errorf("failed to submit txs to sequencer: %w", err)
		}

		r.cache.SetTxsSeen(newHashes)
		totalSubmitted += len(newTxs)
	}

	if totalSubmitted > 0 {
		r.logger.Debug().Int("total_txs", totalSubmitted).Msg("drained mempool")
	}

	return totalSubmitted > 0, nil
}

func hashTxs(txs [][]byte) []string {
	hashes := make([]string, len(txs))
	for i, tx := range txs {
		h := sha256.Sum256(tx)
		hashes[i] = hex.EncodeToString(h[:])
	}
	return hashes
}

func hashTx(tx []byte) string {
	h := sha256.Sum256(tx)
	return hex.EncodeToString(h[:])
}
