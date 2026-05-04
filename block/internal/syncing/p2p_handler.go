package syncing

import (
	"bytes"
	"context"
	"fmt"
	"sync/atomic"

	"github.com/celestiaorg/go-header"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

type p2pHandler interface {
	ProcessHeight(ctx context.Context, height uint64, heightInCh chan<- common.DAHeightEvent) error
	SetProcessedHeight(height uint64)
}

// P2PHandler coordinates block retrieval from P2P stores for the syncer.
// It waits for both header and data to be available at a given height,
// validates their consistency, and emits events to the syncer for processing.
//
// The handler maintains a processedHeight to track the highest block that has been
// successfully validated and sent to the syncer, preventing duplicate processing.
type P2PHandler struct {
	headerStore header.Store[*types.P2PSignedHeader]
	dataStore   header.Store[*types.P2PData]
	cache       cache.CacheManager
	genesis     genesis.Genesis
	logger      zerolog.Logger

	store        store.Store
	onDoubleSign doubleSignHandler // nil disables detection

	processedHeight atomic.Uint64
}

// NewP2PHandler creates a new P2P handler. Double-sign detection is disabled
// when st or onDoubleSign is nil.
func NewP2PHandler(
	headerStore header.Store[*types.P2PSignedHeader],
	dataStore header.Store[*types.P2PData],
	cache cache.CacheManager,
	genesis genesis.Genesis,
	logger zerolog.Logger,
	st store.Store,
	onDoubleSign doubleSignHandler,
) *P2PHandler {
	return &P2PHandler{
		headerStore:  headerStore,
		dataStore:    dataStore,
		cache:        cache,
		genesis:      genesis,
		logger:       logger.With().Str("component", "p2p_handler").Logger(),
		store:        st,
		onDoubleSign: onDoubleSign,
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
//
// When double-sign detection is enabled, the processedHeight short-circuit is
// deferred so alternates at already-processed heights still trigger detection.
func (h *P2PHandler) ProcessHeight(ctx context.Context, height uint64, heightInCh chan<- common.DAHeightEvent) error {
	if h.store == nil || h.onDoubleSign == nil {
		if height <= h.processedHeight.Load() {
			return nil
		}
	}

	p2pHeader, err := h.headerStore.GetByHeight(ctx, height)
	if err != nil {
		if ctx.Err() == nil {
			h.logger.Debug().Uint64("height", height).Err(err).Msg("header unavailable in store")
		}
		return err
	}
	if err := h.assertExpectedProposer(p2pHeader.ProposerAddress); err != nil {
		h.logger.Debug().Uint64("height", height).Err(err).Msg("invalid header from P2P")
		return err
	}

	// ValidateBasic is the precondition for treating an alternate as evidence.
	if h.store != nil && h.onDoubleSign != nil {
		if err := p2pHeader.SignedHeader.ValidateBasic(); err != nil {
			h.logger.Debug().Uint64("height", height).Err(err).Msg("invalid signed header from P2P")
			return err
		}
		if ev, derr := detectDoubleSign(ctx, h.store, h.cache, p2pHeader.SignedHeader, types.EvidenceSourceP2P); derr == nil && ev != nil {
			h.onDoubleSign(ctx, ev)
			return nil
		} else if derr != nil {
			h.logger.Warn().Err(derr).Uint64("height", height).Msg("double-sign detection error")
		}
		h.cache.SetPendingSignedHeader(p2pHeader.SignedHeader, types.EvidenceSourceP2P)
		if height <= h.processedHeight.Load() {
			return nil
		}
	}

	p2pData, err := h.dataStore.GetByHeight(ctx, height)
	if err != nil {
		if ctx.Err() == nil {
			h.logger.Debug().Uint64("height", height).Err(err).Msg("data unavailable in store")
		}
		return err
	}
	dataCommitment := p2pData.DACommitment()
	if !bytes.Equal(p2pHeader.DataHash[:], dataCommitment[:]) {
		err := fmt.Errorf("data hash mismatch: header %x, data %x", p2pHeader.DataHash, dataCommitment)
		h.logger.Warn().Uint64("height", height).Err(err).Msg("discarding inconsistent block from P2P")
		return err
	}

	// Memoize hash before the header enters the event pipeline so that downstream
	// callers (processHeightEvent, TrySyncNextBlock) get cache hits.
	p2pHeader.MemoizeHash()

	// further header validation (signature) is done in validateBlock.
	// we need to be sure that the previous block n-1 was executed before validating block n
	event := common.DAHeightEvent{
		Header:        p2pHeader.SignedHeader,
		Data:          p2pData.Data,
		Source:        common.SourceP2P,
		DaHeightHints: [2]uint64{p2pHeader.DAHint(), p2pData.DAHint()},
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
