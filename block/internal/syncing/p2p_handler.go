package syncing

import (
	"bytes"
	"context"
	"fmt"

	goheader "github.com/celestiaorg/go-header"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/types"

	"github.com/evstack/ev-node/block/internal/common"
)

// P2PHandler handles all P2P operations for the syncer
type P2PHandler struct {
	headerStore goheader.Store[*types.SignedHeader]
	dataStore   goheader.Store[*types.Data]
	cache       cache.Manager
	genesis     genesis.Genesis
	signer      signer.Signer
	options     common.BlockOptions
	logger      zerolog.Logger
}

// NewP2PHandler creates a new P2P handler
func NewP2PHandler(
	headerStore goheader.Store[*types.SignedHeader],
	dataStore goheader.Store[*types.Data],
	cache cache.Manager,
	genesis genesis.Genesis,
	signer signer.Signer,
	options common.BlockOptions,
	logger zerolog.Logger,
) *P2PHandler {
	return &P2PHandler{
		headerStore: headerStore,
		dataStore:   dataStore,
		cache:       cache,
		genesis:     genesis,
		signer:      signer,
		options:     options,
		logger:      logger.With().Str("component", "p2p_handler").Logger(),
	}
}

// ProcessHeaderRange processes headers from the header store within the given range
func (h *P2PHandler) ProcessHeaderRange(ctx context.Context, startHeight, endHeight uint64) []HeightEvent {
	if startHeight > endHeight {
		return nil
	}

	var events []HeightEvent

	for height := startHeight; height <= endHeight; height++ {
		select {
		case <-ctx.Done():
			return events
		default:
		}

		header, err := h.headerStore.GetByHeight(ctx, height)
		if err != nil {
			h.logger.Debug().Uint64("height", height).Err(err).Msg("failed to get header from store")
			continue
		}

		// Validate header
		if err := h.validateHeader(header); err != nil {
			h.logger.Debug().Uint64("height", height).Err(err).Msg("invalid header from P2P")
			continue
		}

		// Get corresponding data
		var data *types.Data
		if bytes.Equal(header.DataHash, common.DataHashForEmptyTxs) {
			// Create empty data for headers with empty data hash
			data = h.createEmptyDataForHeader(ctx, header)
		} else {
			// Try to get data from data store
			retrievedData, err := h.dataStore.GetByHeight(ctx, height)
			if err != nil {
				h.logger.Debug().Uint64("height", height).Err(err).Msg("could not retrieve data for header from data store")
				continue
			}
			data = retrievedData
		}

		// Validate header with data
		if err := header.ValidateBasicWithData(data); err != nil {
			h.logger.Debug().Uint64("height", height).Err(err).Msg("header validation with data failed")
			continue
		}

		// Create height event
		event := HeightEvent{
			Header:   header,
			Data:     data,
			DaHeight: 0, // P2P events don't have DA height context
		}

		events = append(events, event)

		h.logger.Debug().Uint64("height", height).Str("source", "p2p_headers").Msg("processed header from P2P")
	}

	return events
}

// ProcessDataRange processes data from the data store within the given range
func (h *P2PHandler) ProcessDataRange(ctx context.Context, startHeight, endHeight uint64) []HeightEvent {
	if startHeight > endHeight {
		return nil
	}

	var events []HeightEvent

	for height := startHeight; height <= endHeight; height++ {
		select {
		case <-ctx.Done():
			return events
		default:
		}

		data, err := h.dataStore.GetByHeight(ctx, height)
		if err != nil {
			h.logger.Debug().Uint64("height", height).Err(err).Msg("failed to get data from store")
			continue
		}

		// Get corresponding header
		header, err := h.headerStore.GetByHeight(ctx, height)
		if err != nil {
			h.logger.Debug().Uint64("height", height).Err(err).Msg("could not retrieve header for data from header store")
			continue
		}

		// Validate header
		if err := h.validateHeader(header); err != nil {
			h.logger.Debug().Uint64("height", height).Err(err).Msg("invalid header from P2P")
			continue
		}

		// Validate header with data
		if err := header.ValidateBasicWithData(data); err != nil {
			h.logger.Debug().Uint64("height", height).Err(err).Msg("header validation with data failed")
			continue
		}

		// Create height event
		event := HeightEvent{
			Header:   header,
			Data:     data,
			DaHeight: 0, // P2P events don't have DA height context
		}

		events = append(events, event)

		h.logger.Debug().Uint64("height", height).Str("source", "p2p_data").Msg("processed data from P2P")
	}

	return events
}

// validateHeader performs basic validation on a header from P2P
func (h *P2PHandler) validateHeader(header *types.SignedHeader) error {
	// Check proposer address
	if err := h.assertExpectedProposer(header.ProposerAddress); err != nil {
		return fmt.Errorf("unexpected proposer: %w", err)
	}

	return nil
}

// assertExpectedProposer validates the proposer address
func (h *P2PHandler) assertExpectedProposer(proposerAddr []byte) error {
	if !bytes.Equal(h.genesis.ProposerAddress, proposerAddr) {
		return fmt.Errorf("proposer address mismatch: got %x, expected %x",
			proposerAddr, h.genesis.ProposerAddress)
	}
	return nil
}

// createEmptyDataForHeader creates empty data for headers with empty data hash
func (h *P2PHandler) createEmptyDataForHeader(ctx context.Context, header *types.SignedHeader) *types.Data {
	headerHeight := header.Height()
	var lastDataHash types.Hash

	if headerHeight > 1 {
		// Try to get previous data hash, but don't fail if not available
		if prevData, err := h.dataStore.GetByHeight(ctx, headerHeight-1); err == nil && prevData != nil {
			lastDataHash = prevData.Hash()
		} else {
			h.logger.Debug().Uint64("current_height", headerHeight).Uint64("previous_height", headerHeight-1).
				Msg("previous block not available, using empty last data hash")
		}
	}

	metadata := &types.Metadata{
		ChainID:      header.ChainID(),
		Height:       headerHeight,
		Time:         header.BaseHeader.Time,
		LastDataHash: lastDataHash,
	}

	return &types.Data{
		Metadata: metadata,
	}
}
