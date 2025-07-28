package block

import (
	"context"
	"github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/types"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-log/v2"
	"time"
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
	seenStore datastore.Batching
	manager   *Manager
	namespace []byte
}

// NewDirectTxReaper creates a new DirectTxReaper instance with persistent seenTx storage.
func NewDirectTxReaper(ctx context.Context, da da.DA, sequencer sequencer.DirectTxSequencer, manager *Manager, chainID string, interval time.Duration, logger log.EventLogger, store datastore.Batching, namespace []byte, ) *DirectTxReaper {
	if interval <= 0 {
		interval = DefaultInterval
	}
	return &DirectTxReaper{
		da:        da,
		sequencer: sequencer,
		chainID:   chainID,
		interval:  interval,
		logger:    logger,
		ctx:       ctx,
		seenStore: store,
		namespace: namespace,
		manager:   manager,
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
			r.SubmitTxs()
		}
	}
}

// SubmitTxs retrieves direct transactions from the DA layer and submits them to the sequencer.
func (r *DirectTxReaper) SubmitTxs() {
	// Get the latest DA height
	daHeight := uint64(0)
	if r.manager != nil {
		daHeight = r.manager.GetDAIncludedHeight()
	}

	if daHeight == 0 {
		r.logger.Debug("DirectTxReaper: No DA height available yet")
		return
	}

	// Get all blob IDs at the current DA height
	result, err := r.da.GetIDs(r.ctx, daHeight, r.namespace)
	if err != nil {
		r.logger.Error("DirectTxReaper failed to get IDs from DA", "error", err)
		return
	}

	if len(result.IDs) == 0 {
		r.logger.Debug("DirectTxReaper found no blobs at current DA height", "height", daHeight)
		return
	}

	// Get the blobs for all IDs
	blobs, err := r.da.Get(r.ctx, result.IDs, r.namespace)
	if err != nil {
		r.logger.Error("DirectTxReaper failed to get blobs from DA", "error", err)
		return
	}

	var newTxs [][]byte
	for _, blob := range blobs {
		// Process each blob to extract direct transactions
		var data types.Data
		err := data.UnmarshalBinary(blob)
		if err != nil {
			r.logger.Debug("DirectTxReaper failed to unmarshal blob data", "error", err)
			continue
		}

		// Skip blobs from different chains
		if data.Metadata.ChainID != r.chainID {
			r.logger.Debug("DirectTxReaper ignoring data from different chain", "chainID", data.Metadata.ChainID)
			continue
		}

		// Process each transaction in the blob
		for _, tx := range data.Txs {
			txHash := hashTx(tx)
			key := datastore.NewKey(txHash)
			has, err := r.seenStore.Has(r.ctx, key)
			if err != nil {
				r.logger.Error("DirectTxReaper failed to check seenStore", "error", err)
				continue
			}
			if !has {
				newTxs = append(newTxs, tx)
			}
		}
	}

	if len(newTxs) == 0 {
		r.logger.Debug("DirectTxReaper found no new direct txs to submit")
		return
	}

	r.logger.Debug("DirectTxReaper submitting direct txs to sequencer", "txCount", len(newTxs))
	err = r.sequencer.SubmitDirectTxs(r.ctx, newTxs)
	if err != nil {
		r.logger.Error("DirectTxReaper failed to submit direct txs to sequencer", "error", err)
		return
	}

	// Mark the transactions as seen
	for _, tx := range newTxs {
		txHash := hashTx(tx)
		key := datastore.NewKey(txHash)
		if err := r.seenStore.Put(r.ctx, key, []byte{1}); err != nil {
			r.logger.Error("DirectTxReaper failed to persist seen tx", "txHash", txHash, "error", err)
		}
	}

	// Notify the manager that new transactions are available
	if r.manager != nil && len(newTxs) > 0 {
		r.logger.Debug("DirectTxReaper notifying manager of new transactions")
		r.manager.NotifyNewTransactions()
	}

	r.logger.Debug("DirectTxReaper successfully submitted direct txs")
}
