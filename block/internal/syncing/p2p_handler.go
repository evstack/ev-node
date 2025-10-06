package syncing

import (
	"bytes"
	"context"
	"fmt"

	goheader "github.com/celestiaorg/go-header"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/types"
)

// P2PHandler handles all P2P operations for the syncer
type P2PHandler struct {
	headerStore goheader.Store[*types.SignedHeader]
	dataStore   goheader.Store[*types.Data]
	genesis     genesis.Genesis
	options     common.BlockOptions
	logger      zerolog.Logger
}

// NewP2PHandler creates a new P2P handler
func NewP2PHandler(
	headerStore goheader.Store[*types.SignedHeader],
	dataStore goheader.Store[*types.Data],
	genesis genesis.Genesis,
	options common.BlockOptions,
	logger zerolog.Logger,
) *P2PHandler {
	return &P2PHandler{
		headerStore: headerStore,
		dataStore:   dataStore,
		genesis:     genesis,
		options:     options,
		logger:      logger.With().Str("component", "p2p_handler").Logger(),
	}
}

// ProcessHeaderRange processes headers from the header store within the given range
func (h *P2PHandler) ProcessHeaderRange(ctx context.Context, startHeight, endHeight uint64) []common.DAHeightEvent {
	if startHeight > endHeight {
		return nil
	}

	var events []common.DAHeightEvent

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

		// basic header validation
		if err := h.assertExpectedProposer(header.ProposerAddress); err != nil {
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

		// further header validation (signature) is done in validateBlock.
		// we need to be sure that the previous block n-1 was executed before validating block n

		// Create height event
		event := common.DAHeightEvent{
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
func (h *P2PHandler) ProcessDataRange(ctx context.Context, startHeight, endHeight uint64) []common.DAHeightEvent {
	if startHeight > endHeight {
		return nil
	}

	var events []common.DAHeightEvent

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
		}

		events = append(events, event)

		h.logger.Debug().Uint64("height", height).Str("source", "p2p_data").Msg("processed data from P2P")
	}

	return events
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
