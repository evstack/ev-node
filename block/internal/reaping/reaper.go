package reaping

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	ds "github.com/ipfs/go-datastore"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block/internal/executing"
	coreexecutor "github.com/evstack/ev-node/core/execution"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/store"
)

// DefaultInterval is the default reaper interval
const DefaultInterval = 1 * time.Second

// Reaper is responsible for periodically retrieving transactions from the executor,
// filtering out already seen transactions, and submitting new transactions to the sequencer.
type Reaper struct {
	exec      coreexecutor.Executor
	sequencer coresequencer.Sequencer
	chainID   string
	interval  time.Duration
	seenStore ds.Batching
	executor  *executing.Executor

	// shared components
	logger zerolog.Logger

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewReaper creates a new Reaper instance with persistent seenTx storage.
func NewReaper(
	exec coreexecutor.Executor,
	sequencer coresequencer.Sequencer,
	genesis genesis.Genesis,
	logger zerolog.Logger,
	executor *executing.Executor,
	scrapeInterval time.Duration,
) (*Reaper, error) {
	if executor == nil {
		return nil, errors.New("executor cannot be nil")
	}
	if scrapeInterval == 0 {
		return nil, errors.New("scrape interval cannot be empty")
	}
	store, err := store.NewDefaultInMemoryKVStore()
	if err != nil {
		return nil, fmt.Errorf("failed to create reaper store: %w", err)
	}

	return &Reaper{
		exec:      exec,
		sequencer: sequencer,
		chainID:   genesis.ChainID,
		interval:  scrapeInterval,
		logger:    logger.With().Str("component", "reaper").Logger(),
		seenStore: store,
		executor:  executor,
	}, nil
}

// Start begins the execution component
func (r *Reaper) Start(ctx context.Context) error {
	r.ctx, r.cancel = context.WithCancel(ctx)

	// Start repear loop
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

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			r.SubmitTxs()
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
func (r *Reaper) SubmitTxs() {
	txs, err := r.exec.GetTxs(r.ctx)
	if err != nil {
		r.logger.Error().Err(err).Msg("failed to get txs from executor")
		return
	}
	if len(txs) == 0 {
		r.logger.Debug().Msg("no new txs")
		return
	}

	var newTxs [][]byte
	for _, tx := range txs {
		txHash := hashTx(tx)
		key := ds.NewKey(txHash)
		has, err := r.seenStore.Has(r.ctx, key)
		if err != nil {
			r.logger.Error().Err(err).Msg("failed to check seenStore")
			continue
		}
		if !has {
			newTxs = append(newTxs, tx)
		}
	}

	if len(newTxs) == 0 {
		r.logger.Debug().Msg("no new txs to submit")
		return
	}

	r.logger.Debug().Int("txCount", len(newTxs)).Msg("submitting txs to sequencer")

	_, err = r.sequencer.SubmitBatchTxs(r.ctx, coresequencer.SubmitBatchTxsRequest{
		Id:    []byte(r.chainID),
		Batch: &coresequencer.Batch{Transactions: newTxs},
	})
	if err != nil {
		r.logger.Error().Err(err).Msg("failed to submit txs to sequencer")
		return
	}

	for _, tx := range newTxs {
		txHash := hashTx(tx)
		key := ds.NewKey(txHash)
		if err := r.seenStore.Put(r.ctx, key, []byte{1}); err != nil {
			r.logger.Error().Err(err).Str("txHash", txHash).Msg("failed to persist seen tx")
		}
	}

	// Notify the executor that new transactions are available
	if len(newTxs) > 0 {
		r.logger.Debug().Msg("notifying executor of new transactions")
		r.executor.NotifyNewTransactions()
	}

	r.logger.Debug().Msg("successfully submitted txs")
}

// SeenStore returns the datastore used to track seen transactions.
func (r *Reaper) SeenStore() ds.Datastore {
	return r.seenStore
}

func hashTx(tx []byte) string {
	hash := sha256.Sum256(tx)
	return hex.EncodeToString(hash[:])
}
