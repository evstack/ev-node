package syncing

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	goheader "github.com/celestiaorg/go-header"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/types"
)

// P2PHandler handles all P2P operations for the syncer
type P2PHandler struct {
	headerStore goheader.Store[*types.SignedHeader]
	dataStore   goheader.Store[*types.Data]
	cache       cache.Manager
	genesis     genesis.Genesis
	logger      zerolog.Logger
}

// NewP2PHandler creates a new P2P handler
func NewP2PHandler(
	headerStore goheader.Store[*types.SignedHeader],
	dataStore goheader.Store[*types.Data],
	cache cache.Manager,
	genesis genesis.Genesis,
	logger zerolog.Logger,
) *P2PHandler {
	return &P2PHandler{
		headerStore: headerStore,
		dataStore:   dataStore,
		cache:       cache,
		genesis:     genesis,
		logger:      logger.With().Str("component", "p2p_handler").Logger(),
	}
}

// ProcessHeaderRange processes headers from the header store within the given range
func (h *P2PHandler) ProcessHeaderRange(ctx context.Context, startHeight, endHeight uint64, heightInCh chan<- common.DAHeightEvent) {
	if startHeight > endHeight {
		return
	}

	for height := startHeight; height <= endHeight; height++ {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Create a timeout context for each GetByHeight call to prevent blocking
		timeoutCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		header, err := h.headerStore.GetByHeight(timeoutCtx, height)
		cancel()

		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				h.logger.Debug().Uint64("height", height).Msg("timeout waiting for header from store, will retry later")
				// Don't continue processing further heights if we timeout on one
				// This prevents blocking on sequential heights
				return
			}
			h.logger.Debug().Uint64("height", height).Err(err).Msg("failed to get header from store")
			continue
		}

		// basic header validation
		if err := h.assertExpectedProposer(header.ProposerAddress); err != nil {
			h.logger.Debug().Uint64("height", height).Err(err).Msg("invalid header from P2P")
			continue
		}

		// Get corresponding data (empty data are still broadcasted by peers)
		var data *types.Data
		timeoutCtx, cancel = context.WithTimeout(ctx, 500*time.Millisecond)
		retrievedData, err := h.dataStore.GetByHeight(timeoutCtx, height)
		cancel()

		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				h.logger.Debug().Uint64("height", height).Msg("timeout waiting for data from store, will retry later")
				// Don't continue processing if data is not available
				// Store event with header only for later processing
				continue
			}
			h.logger.Debug().Uint64("height", height).Err(err).Msg("could not retrieve data for header from data store")
			continue
		}
		data = retrievedData

		// further header validation (signature) is done in validateBlock.
		// we need to be sure that the previous block n-1 was executed before validating block n

		// Create height event
		event := common.DAHeightEvent{
			Header:   header,
			Data:     data,
			DaHeight: 0, // P2P events don't have DA height context
			Source:   common.SourceP2P,
		}

		select {
		case heightInCh <- event:
		default:
			h.cache.SetPendingEvent(event.Header.Height(), &event)
		}

		h.logger.Debug().Uint64("height", height).Str("source", "p2p_headers").Msg("processed header from P2P")
	}
}

// ProcessDataRange processes data from the data store within the given range
func (h *P2PHandler) ProcessDataRange(ctx context.Context, startHeight, endHeight uint64, heightInCh chan<- common.DAHeightEvent) {
	if startHeight > endHeight {
		return
	}

	for height := startHeight; height <= endHeight; height++ {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Create a timeout context for each GetByHeight call to prevent blocking
		timeoutCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		data, err := h.dataStore.GetByHeight(timeoutCtx, height)
		cancel()

		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				h.logger.Debug().Uint64("height", height).Msg("timeout waiting for data from store, will retry later")
				// Don't continue processing further heights if we timeout on one
				// This prevents blocking on sequential heights
				return
			}
			h.logger.Debug().Uint64("height", height).Err(err).Msg("failed to get data from store")
			continue
		}

		// Get corresponding header with timeout
		timeoutCtx, cancel = context.WithTimeout(ctx, 500*time.Millisecond)
		header, err := h.headerStore.GetByHeight(timeoutCtx, height)
		cancel()

		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				h.logger.Debug().Uint64("height", height).Msg("timeout waiting for header from store, will retry later")
				// Don't continue processing if header is not available
				continue
			}
			h.logger.Debug().Uint64("height", height).Err(err).Msg("could not retrieve header for data from header store")
			continue
		}

		// basic header validation
		if err := h.assertExpectedProposer(header.ProposerAddress); err != nil {
			h.logger.Debug().Uint64("height", height).Err(err).Msg("invalid header from P2P")
			continue
		}

		// further header validation (signature) is done in validateBlock.
		// we need to be sure that the previous block n-1 was executed before validating block n

		// Create height event
		event := common.DAHeightEvent{
			Header:   header,
			Data:     data,
			DaHeight: 0, // P2P events don't have DA height context
			Source:   common.SourceP2P,
		}

		select {
		case heightInCh <- event:
		default:
			h.cache.SetPendingEvent(event.Header.Height(), &event)
		}

		h.logger.Debug().Uint64("height", height).Str("source", "p2p_data").Msg("processed data from P2P")
	}
}

// assertExpectedProposer validates the proposer address
func (h *P2PHandler) assertExpectedProposer(proposerAddr []byte) error {
	if !bytes.Equal(h.genesis.ProposerAddress, proposerAddr) {
		return fmt.Errorf("proposer address mismatch: got %x, expected %x",
			proposerAddr, h.genesis.ProposerAddress)
	}
	return nil
}
