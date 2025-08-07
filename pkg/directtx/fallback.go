package directtx

import (
	"context"

	"github.com/rs/zerolog"
)

// FallbackBlockProducer creates deterministic blocks from direct transactions when sequencer is down
type FallbackBlockProducer struct {
	directTxProvider Provider
	daHeightSource   DAHeightSource
	logger           zerolog.Logger
	maxBlockBytes    uint64
}

// NewFallbackBlockProducer creates a new fallback block producer
func NewFallbackBlockProducer(
	directTxProvider Provider,
	daHeightSource DAHeightSource,
	logger zerolog.Logger,
	maxBlockBytes uint64,
) *FallbackBlockProducer {
	return &FallbackBlockProducer{
		directTxProvider: directTxProvider,
		daHeightSource:   daHeightSource,
		logger:           logger,
		maxBlockBytes:    maxBlockBytes,
	}
}

// CreateFallbackBlock creates a deterministic block containing only direct transactions
func (f *FallbackBlockProducer) CreateFallbackBlock(ctx context.Context) ([][]byte, error) {
	//todo: we need deterministic header and data

	daIncludedHeight := f.daHeightSource.GetDAIncludedHeight()
	txs, err := f.directTxProvider.GetPendingDirectTXs(ctx, daIncludedHeight, f.maxBlockBytes)
	if err != nil {
		return nil, err
	}

	if len(txs) == 0 {
		f.logger.Debug().Msg("No direct transactions available for fallback block")
		return nil, nil
	}

	f.logger.Info().Int("tx_count", len(txs)).Msg("Creating fallback block with direct transactions")
	return txs, nil
}
