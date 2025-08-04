package block

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/evstack/ev-node/core/da"
	coreda "github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/types"
	ds "github.com/ipfs/go-datastore"
	kt "github.com/ipfs/go-datastore/keytransform"
	"github.com/ipfs/go-log/v2"
)

const (
	keyPrefixDirTXSeen = "dTX"
)

// DirectTxReaper is responsible for periodically retrieving direct transactions from the DA layer,
// filtering out already seen transactions, and submitting new transactions to the sequencer.
type DirectTxReaper struct {
	da        da.DA
	sequencer sequencer.DirectTxSequencer
	chainID   string
	interval  time.Duration
	logger    log.EventLogger
	ctx       context.Context
	seenStore ds.Batching
	manager   *Manager
	daHeight  *atomic.Uint64
}

// NewDirectTxReaper creates a new DirectTxReaper instance with persistent seenTx storage.
func NewDirectTxReaper(
	ctx context.Context,
	da coreda.DA,
	sequencer sequencer.DirectTxSequencer,
	manager *Manager,
	chainID string,
	interval time.Duration,
	logger log.EventLogger,
	store ds.Batching,
	daStartHeight uint64,
) *DirectTxReaper {
	if daStartHeight == 0 {
		daStartHeight = 1
	}
	if interval <= 0 {
		interval = 100 * time.Millisecond
	}
	daHeight := new(atomic.Uint64)
	daHeight.Store(daStartHeight)
	return &DirectTxReaper{
		da:        da,
		sequencer: sequencer,
		chainID:   chainID,
		interval:  interval,
		logger:    logger,
		ctx:       ctx,
		seenStore: kt.Wrap(store, &kt.PrefixTransform{
			Prefix: ds.NewKey(keyPrefixDirTXSeen),
		}),
		manager:  manager,
		daHeight: daHeight,
	}
}

// Start begins the reaping process at the specified interval.
func (r *DirectTxReaper) Start(ctx context.Context) {
	r.ctx = ctx
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	r.logger.Info("DirectTxReaper started", "interval", r.interval)

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("DirectTxReaper stopped")
			return
		case <-ticker.C:
			daHeight := r.daHeight.Load()
			if err := r.retrieveDirectTXs(daHeight); err != nil {
				if strings.Contains(err.Error(), coreda.ErrHeightFromFuture.Error()) {
					r.logger.Debug("IDs not found at height", "height", daHeight)
				} else {
					r.logger.Error("Submit direct txs to sequencer", "error", err)
				}
				continue
			}
			r.daHeight.Store(daHeight + 1)

		}
	}
}

// retrieveDirectTXs retrieves direct transactions from the DA layer and submits them to the sequencer.
func (r *DirectTxReaper) retrieveDirectTXs(daHeight uint64) error {
	// Get the latest DA height
	// Get all blob IDs at the current DA height
	result, err := r.da.GetIDs(r.ctx, daHeight, nil)
	if err != nil {
		return fmt.Errorf("get IDs from DA: %w", err)
	}
	if result == nil || len(result.IDs) == 0 {
		r.logger.Debug("No blobs at current DA height", "height", daHeight)
		return nil
	}
	r.logger.Debug("IDs at current DA height", "height", daHeight, "count", len(result.IDs))

	// Get the blobs for all IDs
	blobs, err := r.da.Get(r.ctx, result.IDs, nil)
	if err != nil {
		return fmt.Errorf("get blobs from DA: %w", err)
	}
	r.logger.Debug("Blobs found at height", "height", daHeight, "count", len(blobs))

	var newTxs []sequencer.DirectTX
	for _, blob := range blobs {
		r.logger.Debug("Processing blob data")

		// Process each blob to extract direct transactions
		var data types.Data
		err := data.UnmarshalBinary(blob)
		if err != nil {
			r.logger.Debug("Unexpected payload skipping ", "error", err)
			continue
		}

		// Skip blobs from different chains
		if data.Metadata.ChainID != r.chainID {
			r.logger.Debug("Ignoring data from different chain", "chainID", data.Metadata.ChainID, "expectedChainID", r.chainID)
			continue
		}

		// Process each transaction in the blob
		for i, tx := range data.Txs {
			txHash := hashTx(tx)
			has, err := r.seenStore.Has(r.ctx, ds.NewKey(txHash))
			if err != nil {
				return fmt.Errorf("check seenStore: %w", err)
			}
			if !has {
				newTxs = append(newTxs, sequencer.DirectTX{
					TX:              tx,
					ID:              result.IDs[i],
					FirstSeenHeight: daHeight,
					FirstSeenTime:   result.Timestamp.Unix(),
				})
			}
		}
	}

	if len(newTxs) == 0 {
		r.logger.Debug("No new direct txs to submit")
		return nil
	}

	r.logger.Debug("Submitting direct txs to sequencer", "txCount", len(newTxs))
	err = r.sequencer.SubmitDirectTxs(r.ctx, newTxs...)
	if err != nil {
		return fmt.Errorf("submit direct txs to sequencer: %w", err)
	}
	// Mark the transactions as seen
	for _, v := range newTxs {
		txHash := hashTx(v.TX)
		if err := r.seenStore.Put(r.ctx, ds.NewKey(txHash), []byte{1}); err != nil {
			return fmt.Errorf("persist seen tx: %w", err)
		}
	}
	r.logger.Debug("Successfully submitted direct txs")
	return nil
}
