package syncing

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	goheader "github.com/celestiaorg/go-header"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/types"
)

// P2PHandler fan-outs store updates from the go-header stores into the syncer.
// It ensures both the header and data for a given height are present and
// consistent before emitting an event to the syncer.
type P2PHandler struct {
	headerStore goheader.Store[*types.SignedHeader]
	dataStore   goheader.Store[*types.Data]
	cache       cache.Manager
	genesis     genesis.Genesis
	logger      zerolog.Logger

	mu              sync.Mutex
	processedHeight uint64 // highest block fully applied by the syncer
}

// NewP2PHandler creates a new P2P handler.
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

// SetProcessedHeight updates the highest processed block height.
func (h *P2PHandler) SetProcessedHeight(height uint64) {
	h.mu.Lock()
	if height > h.processedHeight {
		h.processedHeight = height
	}
	h.mu.Unlock()
}

// ProcessHeaderRange scans the provided heights and emits events when both the
// header and data are available.
func (h *P2PHandler) ProcessHeaderRange(ctx context.Context, startHeight, endHeight uint64, heightInCh chan<- common.DAHeightEvent) {
	if startHeight > endHeight {
		return
	}

	for height := startHeight; height <= endHeight; height++ {
		h.mu.Lock()
		shouldProcess := height > h.processedHeight
		h.mu.Unlock()

		if !shouldProcess {
			continue
		}
		h.processHeight(ctx, height, heightInCh, "header_range")
	}
}

// ProcessDataRange scans the provided heights and emits events when both the
// header and data are available.
func (h *P2PHandler) ProcessDataRange(ctx context.Context, startHeight, endHeight uint64, heightInCh chan<- common.DAHeightEvent) {
	if startHeight > endHeight {
		return
	}

	for height := startHeight; height <= endHeight; height++ {
		h.mu.Lock()
		shouldProcess := height > h.processedHeight
		h.mu.Unlock()

		if !shouldProcess {
			continue
		}
		h.processHeight(ctx, height, heightInCh, "data_range")
	}
}

func (h *P2PHandler) processHeight(ctx context.Context, height uint64, heightInCh chan<- common.DAHeightEvent, source string) {
	header, err := h.headerStore.GetByHeight(ctx, height)
	if err != nil {
		if ctx.Err() == nil {
			h.logger.Debug().Uint64("height", height).Err(err).Str("source", source).Msg("header unavailable in store")
		}
		return
	}
	if err := h.assertExpectedProposer(header.ProposerAddress); err != nil {
		h.logger.Debug().Uint64("height", height).Err(err).Str("source", source).Msg("invalid header from P2P")
		return
	}

	data, err := h.dataStore.GetByHeight(ctx, height)
	if err != nil {
		if ctx.Err() == nil {
			h.logger.Debug().Uint64("height", height).Err(err).Str("source", source).Msg("data unavailable in store")
		}
		return
	}

	dataCommitment := data.DACommitment()
	if !bytes.Equal(header.DataHash[:], dataCommitment[:]) {
		h.logger.Warn().
			Uint64("height", height).
			Str("header_data_hash", fmt.Sprintf("%x", header.DataHash)).
			Str("actual_data_hash", fmt.Sprintf("%x", dataCommitment)).
			Str("source", source).
			Msg("DataHash mismatch: header and data do not match from P2P, discarding")
		return
	}

	event := common.DAHeightEvent{
		Header:   header,
		Data:     data,
		DaHeight: 0,
		Source:   common.SourceP2P,
	}

	select {
	case heightInCh <- event:
	default:
		h.cache.SetPendingEvent(event.Header.Height(), &event)
	}

	h.logger.Debug().Uint64("height", height).Str("source", source).Msg("processed event from P2P")
}

// assertExpectedProposer validates the proposer address.
func (h *P2PHandler) assertExpectedProposer(proposerAddr []byte) error {
	if !bytes.Equal(h.genesis.ProposerAddress, proposerAddr) {
		return fmt.Errorf("proposer address mismatch: got %x, expected %x",
			proposerAddr, h.genesis.ProposerAddress)
	}
	return nil
}
