package directtx

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"
	"time"

	rollconf "github.com/evstack/ev-node/pkg/config"
	ds "github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log/v2"
)

var ErrDirectTXWindowMissed = errors.New("direct tx window missed")

const (
	maxQueueSize = 1000
	dbPrefix     = "dTXS"
)

type Sink struct {
	config rollconf.ForcedInclusionConfig
	logger logging.EventLogger

	directTxQueue  *TXQueue
	directTxMu     sync.Mutex
	directTXByHash map[string]struct{}
}

func NewSink(
	ctx context.Context,
	config rollconf.ForcedInclusionConfig,
	db ds.Batching,
	logger logging.EventLogger,
) (*Sink, error) {
	r := Sink{config: config,
		logger:         logger,
		directTxQueue:  NewTXQueue(db, dbPrefix, maxQueueSize),
		directTXByHash: make(map[string]struct{}, 0),
	}
	return &r, r.init(ctx)
}

func (d *Sink) init(ctx context.Context) error {
	d.directTxMu.Lock()
	defer d.directTxMu.Unlock()
	err := d.directTxQueue.Load(ctx, func(tx DirectTX) {
		d.directTXByHash[cacheKey(tx)] = struct{}{}
	})
	return err
}

func (d *Sink) SubmitDirectTxs(ctx context.Context, txs ...DirectTX) error {
	if len(txs) == 0 {
		return nil
	}
	for i, tx := range txs {
		if err := tx.ValidateBasic(); err != nil {
			return fmt.Errorf("tx %d: %w", i, err)
		}
	}
	d.logger.Debug("Adding direct transactions to queue", "txCount", len(txs))

	d.directTxMu.Lock()
	defer d.directTxMu.Unlock()
	if err := d.directTxQueue.AddBatch(ctx, txs...); err != nil {
		return err
	}
	for _, tx := range txs {
		d.directTXByHash[cacheKey(tx)] = struct{}{}
	}
	return nil
}

func (d *Sink) GetPendingDirectTXs(ctx context.Context, daIncludedHeight uint64, maxBytes uint64) ([][]byte, error) {
	d.logger.Debug("GetNextBatch", "currentHeight", daIncludedHeight, "maxBytes", maxBytes)

	var transactions [][]byte
	remainingBytes := int(maxBytes)
	d.directTxMu.Lock()
	defer d.directTxMu.Unlock()

	// First get direct transactions up to max size
	for remainingBytes > 0 {
		peekedTx, err := d.directTxQueue.Peek(ctx)
		if err != nil {
			return nil, err
		}
		if peekedTx == nil || len(peekedTx.TX) > remainingBytes {
			break
		}
		if false { // todo Alex: need some blocks in test to get past this da height
			// Require a minimum number of DA blocks to pass before including a direct transaction
			if daIncludedHeight < peekedTx.FirstSeenHeight+d.config.MinDADelay {
				break
			}
			// Let full nodes enforce inclusion within a fixed period of time window
			if time.Since(time.Unix(peekedTx.FirstSeenTime, 0)) > d.config.MaxInclusionDelay {
				// todo (Alex): what to do in this case? the sequencer may be down for unknown reasons.
				return nil, ErrDirectTXWindowMissed
			}
		}

		// pop from the queue
		directTX, err := d.directTxQueue.Next(ctx)
		if err != nil {
			return nil, err
		}
		if directTX == nil {
			continue
		}
		transactions = append(transactions, directTX.TX)
		remainingBytes -= len(directTX.TX)
	}
	return transactions, nil
}

func (d *Sink) MarkTXIncluded(ctx context.Context, txs [][]byte, blockHeight uint64, timestamp time.Time) error {
	d.directTxMu.Lock()
	defer d.directTxMu.Unlock()
	for _, tx := range txs {
		ck := cacheKey(DirectTX{TX: tx})
		if _, ok := d.directTXByHash[ck]; ok {
			if err := d.directTxQueue.delete(ctx, tx); err != nil {
				return err
			}
			delete(d.directTXByHash, ck)
			d.logger.Debug("Removed direct tx from queue", "txHash", ck)
		}
	}
	return nil
}

func (d *Sink) HasMissedDirectTX(ctx context.Context, blockHeight uint64, timestamp time.Time) error {
	d.directTxMu.Lock()
	defer d.directTxMu.Unlock()

	peekedTx, err := d.directTxQueue.Peek(ctx)
	if err != nil {
		return err
	}
	if peekedTx == nil {
		return nil
	}
	// Let full nodes enforce inclusion within a fixed period of time window
	if time.Since(time.Unix(peekedTx.FirstSeenTime, 0)) > d.config.MaxInclusionDelay {
		return ErrDirectTXWindowMissed
	}

	return nil
}

func cacheKey(tx DirectTX) string {
	sum256 := sha256.Sum256(tx.TX)
	cacheKey := string(sum256[:])
	return cacheKey
}
