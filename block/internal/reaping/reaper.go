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
	coreexecutor "github.com/evstack/ev-node/core/execution"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/genesis"
)

const (
	// MaxBackoffInterval is the maximum backoff interval for retries
	MaxBackoffInterval = 30 * time.Second
	CleanupInterval    = 1 * time.Hour
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
	r.ctx, r.cancel = context.WithCancel(ctx)

	// Start reaper loop
	r.wg.Go(r.reaperLoop)

	r.logger.Info().Dur("idle_interval", r.interval).Msg("reaper started")
	return nil
}

func (r *Reaper) reaperLoop() {
	cleanupTicker := time.NewTicker(CleanupInterval)
	defer cleanupTicker.Stop()

	consecutiveFailures := 0

	for {
		submitted, err := r.drainMempool()

		if err != nil {
			consecutiveFailures++
			backoff := r.interval * time.Duration(1<<min(consecutiveFailures, 5))
			backoff = min(backoff, MaxBackoffInterval)
			r.logger.Warn().
				Err(err).
				Int("consecutive_failures", consecutiveFailures).
				Dur("backoff", backoff).
				Msg("reaper error, backing off")
			r.wait(backoff, cleanupTicker.C)
			continue
		}

		if consecutiveFailures > 0 {
			r.logger.Info().Msg("reaper recovered from errors")
			consecutiveFailures = 0
		}

		if submitted {
			continue
		}

		r.wait(r.interval, cleanupTicker.C)
	}
}

func (r *Reaper) wait(d time.Duration, cleanupCh <-chan time.Time) {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-r.ctx.Done():
	case <-cleanupCh:
		removed := r.cache.CleanupOldTxs(cache.DefaultTxCacheRetention)
		if removed > 0 {
			r.logger.Info().Int("removed", removed).Msg("cleaned up old transaction hashes")
		}
	case <-timer.C:
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

type pendingTx struct {
	tx   []byte
	hash string
}

func (r *Reaper) drainMempool() (bool, error) {
	var totalSubmitted int

	for {
		txs, err := r.exec.GetTxs(r.ctx)
		if err != nil {
			return totalSubmitted > 0, fmt.Errorf("failed to get txs from executor: %w", err)
		}
		if len(txs) == 0 {
			break
		}

		filtered := r.filterNewTxs(txs)
		if len(filtered) == 0 {
			continue
		}

		n, err := r.submitFiltered(filtered)
		if err != nil {
			return totalSubmitted > 0, err
		}
		totalSubmitted += n
	}

	if totalSubmitted > 0 {
		r.logger.Debug().Int("total_txs", totalSubmitted).Msg("drained mempool")
		if r.onTxsSubmitted != nil {
			r.onTxsSubmitted()
		}
	}

	return totalSubmitted > 0, nil
}

func (r *Reaper) filterNewTxs(txs [][]byte) []pendingTx {
	pending := make([]pendingTx, 0, len(txs))
	for _, tx := range txs {
		h := hashTx(tx)
		if !r.cache.IsTxSeen(h) {
			pending = append(pending, pendingTx{tx: tx, hash: h})
		}
	}
	return pending
}

func (r *Reaper) submitFiltered(batch []pendingTx) (int, error) {
	txs := make([][]byte, len(batch))
	hashes := make([]string, len(batch))
	for i, p := range batch {
		txs[i] = p.tx
		hashes[i] = p.hash
	}

	_, err := r.sequencer.SubmitBatchTxs(r.ctx, coresequencer.SubmitBatchTxsRequest{
		Id:    []byte(r.chainID),
		Batch: &coresequencer.Batch{Transactions: txs},
	})
	if err != nil {
		return 0, fmt.Errorf("failed to submit txs to sequencer: %w", err)
	}

	r.cache.SetTxsSeen(hashes)
	return len(txs), nil
}

func hashTx(tx []byte) string {
	hash := sha256.Sum256(tx)
	return hex.EncodeToString(hash[:])
}
