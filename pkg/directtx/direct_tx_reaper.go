package directtx

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/evstack/ev-node/core/da"
	"github.com/ipfs/go-log/v2"
)

// DirectTxReaper is responsible for periodically retrieving direct transactions from the DA layer,
// filtering out already seen transactions, and submitting new transactions to the sequencer.
type DirectTxReaper struct {
	da                da.DA
	interval          time.Duration
	logger            log.EventLogger
	ctx               context.Context
	directTXExtractor *Extractor
	daHeight          *atomic.Uint64
}

// NewDirectTxReaper creates a new DirectTxReaper instance with persistent seenTx storage.
func NewDirectTxReaper(
	ctx context.Context,
	da da.DA,
	extractor *Extractor,
	interval time.Duration,
	logger log.EventLogger,
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
		da:                da,
		interval:          interval,
		logger:            logger,
		ctx:               ctx,
		directTXExtractor: extractor,
		daHeight:          daHeight,
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
				if strings.Contains(err.Error(), da.ErrHeightFromFuture.Error()) {
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

	if len(blobs) != len(result.IDs) {
		return fmt.Errorf("number of blobs and IDs do not match")
	}
	for blobIdx, blob := range blobs {
		_, err := r.directTXExtractor.Handle(r.ctx, daHeight, result.IDs[blobIdx], blob, result.Timestamp)
		if err != nil {
			return err
		}
	}
	return nil
}
