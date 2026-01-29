package syncing

import (
	"bytes"
	"context"
	"fmt"
	"sync/atomic"

	"github.com/celestiaorg/go-header"
	"github.com/evstack/ev-node/types"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/genesis"
)

type p2pHandler interface {
	ProcessHeight(ctx context.Context, height uint64, heightInCh chan<- common.DAHeightEvent) error
	SetProcessedHeight(height uint64)
}

// HeightStore is a subset of goheader.Store
type HeightStore[H header.Header[H]] interface {
	GetByHeight(ctx context.Context, height uint64) (H, uint64, error)
}

// P2PHandler coordinates block retrieval from P2P stores for the syncer.
// It waits for both header and data to be available at a given height,
// validates their consistency, and emits events to the syncer for processing.
//
// The handler maintains a processedHeight to track the highest block that has been
// successfully validated and sent to the syncer, preventing duplicate processing.
type P2PHandler struct {
	headerStore HeightStore[*types.P2PSignedHeader]
	dataStore   HeightStore[*types.P2PData]
	cache       cache.CacheManager
	genesis     genesis.Genesis
	logger      zerolog.Logger

	processedHeight atomic.Uint64
}

// NewP2PHandler creates a new P2P handler.
func NewP2PHandler(
	headerStore HeightStore[*types.P2PSignedHeader],
	dataStore HeightStore[*types.P2PData],
	cache cache.CacheManager,
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
	for range 1_000 {
		current := h.processedHeight.Load()
		if height <= current {
			return
		}
		if h.processedHeight.CompareAndSwap(current, height) {
			return
		}
	}
}

// ProcessHeight retrieves and validates both header and data for the given height from P2P stores.
// It blocks until both are available, validates consistency (proposer address and data hash match),
// then emits the event to heightInCh or stores it as pending. Updates processedHeight on success.
func (h *P2PHandler) ProcessHeight(ctx context.Context, height uint64, heightInCh chan<- common.DAHeightEvent) error {
	if height <= h.processedHeight.Load() {
		return nil
	}

	p2pHeader, headerDAHint, err := h.headerStore.GetByHeight(ctx, height)
	if err != nil {
		if ctx.Err() == nil {
			h.logger.Debug().Uint64("height", height).Err(err).Msg("header unavailable in store")
		}
		return err
	}
	header := p2pHeader.Message
	if err := h.assertExpectedProposer(header.ProposerAddress); err != nil {
		h.logger.Debug().Uint64("height", height).Err(err).Msg("invalid header from P2P")
		return err
	}

	p2pData, dataDAHint, err := h.dataStore.GetByHeight(ctx, height)
	if err != nil {
		if ctx.Err() == nil {
			h.logger.Debug().Uint64("height", height).Err(err).Msg("data unavailable in store")
		}
		return err
	}
	data := p2pData.Message
	dataCommitment := data.DACommitment()
	if !bytes.Equal(header.DataHash[:], dataCommitment[:]) {
		err := fmt.Errorf("data hash mismatch: header %x, data %x", header.DataHash, dataCommitment)
		h.logger.Warn().Uint64("height", height).Err(err).Msg("discarding inconsistent block from P2P")
		return err
	}

	// further header validation (signature) is done in validateBlock.
	// we need to be sure that the previous block n-1 was executed before validating block n
	event := common.DAHeightEvent{
		Header:        header,
		Data:          data,
		Source:        common.SourceP2P,
		DaHeightHints: [2]uint64{headerDAHint, dataDAHint},
	}

	select {
	case heightInCh <- event:
	default:
		h.cache.SetPendingEvent(event.Header.Height(), &event)
	}

	h.SetProcessedHeight(height)

	h.logger.Debug().Uint64("height", height).Msg("processed event from P2P")
	return nil
}

// assertExpectedProposer validates the proposer address.
func (h *P2PHandler) assertExpectedProposer(proposerAddr []byte) error {
	if !bytes.Equal(h.genesis.ProposerAddress, proposerAddr) {
		return fmt.Errorf("proposer address mismatch: got %x, expected %x",
			proposerAddr, h.genesis.ProposerAddress)
	}
	return nil
}
