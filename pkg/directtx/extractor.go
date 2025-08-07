package directtx

import (
	"context"
	"fmt"
	"time"

	"github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/types"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/keytransform"
	"github.com/rs/zerolog"
)

const (
	keyPrefixDirTXSeen = "dTXB"
)

// DirectTXSubmitter defines an interface for submitting direct transactions to a sequencer or similar process.
type DirectTXSubmitter interface {
	// SubmitDirectTxs submits one or more direct transactions and returns an error if the operation fails.
	SubmitDirectTxs(ctx context.Context, txs ...DirectTX) error
}

type Extractor struct {
	dTxSink   DirectTXSubmitter
	chainID   string
	logger    zerolog.Logger
	seenStore datastore.Batching
}

func NewExtractor(dTxSink DirectTXSubmitter, chainID string, logger zerolog.Logger, store datastore.Batching) *Extractor {
	return &Extractor{
		dTxSink: dTxSink,
		chainID: chainID,
		logger:  logger,
		seenStore: keytransform.Wrap(store, &keytransform.PrefixTransform{
			Prefix: datastore.NewKey(keyPrefixDirTXSeen),
		}),
	}
}

func (d *Extractor) Handle(ctx context.Context, daHeight uint64, id da.ID, blob da.Blob, daBlockTimestamp time.Time) ([]DirectTX, error) {
	d.logger.Debug().Msg("Processing blob data")
	var newTxs []DirectTX

	// Process each blob to extract direct transactions
	var data types.Data
	err := data.UnmarshalBinary(blob)
	if err != nil {
		d.logger.Debug().Err(err).Msg("Unexpected payload skipping ")
		return nil, nil
	}

	// Skip blobs from different chains
	if data.Metadata.ChainID != d.chainID {
		d.logger.Debug().Str("chainID", data.Metadata.ChainID).Str("expectedChainID", d.chainID).
			Msg("Ignoring data from different chain")
		return nil, nil
	}

	// Process each transaction in the blob
	for _, tx := range data.Txs {
		dTx := DirectTX{
			TX:              tx,
			ID:              id,
			FirstSeenHeight: daHeight,
			FirstSeenTime:   daBlockTimestamp.Unix(),
		}
		has, err := d.seenStore.Has(ctx, buildStoreKey(dTx))
		if err != nil {
			return nil, fmt.Errorf("check seenStore: %w", err)
		}
		if !has {
			newTxs = append(newTxs, dTx)
		}
	}
	d.logger.Debug().Int("txCount", len(newTxs)).Msg("Submitting direct txs to sequencer")
	err = d.dTxSink.SubmitDirectTxs(ctx, newTxs...)
	if err != nil {
		return nil, fmt.Errorf("submit direct txs to sequencer: %w", err)
	}
	// Mark the transactions as seen
	for _, v := range newTxs {
		if err := d.seenStore.Put(ctx, buildStoreKey(v), []byte{1}); err != nil {
			return nil, fmt.Errorf("persist seen tx: %w", err)
		}
	}
	return newTxs, nil
}
