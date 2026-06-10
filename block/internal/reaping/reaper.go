package reaping

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"

	coreexecutor "github.com/evstack/ev-node/core/execution"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/genesis"
)

const (
	// MaxBackoffInterval is the maximum backoff interval for retries
	MaxBackoffInterval = 30 * time.Second
)

// Reaper is responsible for periodically retrieving transactions from the executor
// and submitting them to the sequencer.
type Reaper struct {
	exec           coreexecutor.Executor
	sequencer      coresequencer.Sequencer
	chainID        string
	interval       time.Duration
	onTxsSubmitted func()

	// pendingBatchCount reports the sequencer's pending batch count, used to
	// detect whether a submission actually enqueued new entries. Optional.
	pendingBatchCount func() int

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
	scrapeInterval time.Duration,
	onTxsSubmitted func(),
	pendingBatchCount func() int,
) (*Reaper, error) {
	if scrapeInterval == 0 {
		return nil, errors.New("scrape interval cannot be empty")
	}

	return &Reaper{
		exec:              exec,
		sequencer:         sequencer,
		chainID:           genesis.ChainID,
		interval:          scrapeInterval,
		logger:            logger.With().Str("component", "reaper").Logger(),
		onTxsSubmitted:    onTxsSubmitted,
		pendingBatchCount: pendingBatchCount,
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
	consecutiveFailures := 0

	for {
		err := r.drainMempool()

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

func (r *Reaper) drainMempool() error {
	txs, err := r.exec.GetTxs(r.ctx)
	if err != nil {
		return fmt.Errorf("failed to get txs from executor: %w", err)
	}
	if len(txs) == 0 {
		return nil
	}

	before := 0
	if r.pendingBatchCount != nil {
		before = r.pendingBatchCount()
	}

	_, err = r.sequencer.SubmitBatchTxs(r.ctx, coresequencer.SubmitBatchTxsRequest{
		Id:    []byte(r.chainID),
		Batch: &coresequencer.Batch{Transactions: txs},
	})
	if err != nil {
		return fmt.Errorf("failed to submit txs to sequencer: %w", err)
	}

	// without a pending count we cannot tell duplicates apart, so assume queued
	queued := true
	if r.pendingBatchCount != nil {
		queued = r.pendingBatchCount() > before
	}

	if r.onTxsSubmitted != nil && queued {
		r.onTxsSubmitted()
	}

	r.logger.Debug().
		Int("seen_txs", len(txs)).
		Bool("queued", queued).
		Msg("drained mempool")

	return nil
}
