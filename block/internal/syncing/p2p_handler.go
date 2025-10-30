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

// P2PHandler handles all P2P operations for the syncer
type P2PHandler struct {
	headerStore goheader.Store[*types.SignedHeader]
	dataStore   goheader.Store[*types.Data]
	cache       cache.Manager
	genesis     genesis.Genesis
	logger      zerolog.Logger

	mu              sync.Mutex
	processedHeight uint64
	inflight        map[uint64]context.CancelFunc
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
		inflight:    make(map[uint64]context.CancelFunc),
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

// OnHeightProcessed cancels any pending watcher for a processed height.
func (h *P2PHandler) OnHeightProcessed(height uint64) {
	h.mu.Lock()
	if height > h.processedHeight {
		h.processedHeight = height
	}
	cancel, ok := h.inflight[height]
	h.mu.Unlock()
	if ok {
		cancel()
	}
}

// ProcessHeaderRange ensures watchers are running for the provided heights.
func (h *P2PHandler) ProcessHeaderRange(ctx context.Context, startHeight, endHeight uint64, heightInCh chan<- common.DAHeightEvent) {
	if startHeight > endHeight {
		return
	}

	for height := startHeight; height <= endHeight; height++ {
		h.ensureWatcher(ctx, height, heightInCh)
	}
}

// ProcessDataRange ensures watchers are running for the provided heights.
func (h *P2PHandler) ProcessDataRange(ctx context.Context, startHeight, endHeight uint64, heightInCh chan<- common.DAHeightEvent) {
	if startHeight > endHeight {
		return
	}

	for height := startHeight; height <= endHeight; height++ {
		h.ensureWatcher(ctx, height, heightInCh)
	}
}

func (h *P2PHandler) ensureWatcher(ctx context.Context, height uint64, heightInCh chan<- common.DAHeightEvent) {
	h.mu.Lock()
	if height <= h.processedHeight {
		h.mu.Unlock()
		return
	}
	if _, exists := h.inflight[height]; exists {
		h.mu.Unlock()
		return
	}

	childCtx, cancel := context.WithCancel(ctx)
	h.inflight[height] = cancel
	h.mu.Unlock()

	go h.awaitHeight(childCtx, height, heightInCh, cancel)
}

func (h *P2PHandler) awaitHeight(ctx context.Context, height uint64, heightInCh chan<- common.DAHeightEvent, cancel context.CancelFunc) {
	defer cancel()
	defer h.removeInflight(height)

	header, err := h.headerStore.GetByHeight(ctx, height)
	if err != nil {
		if ctx.Err() == nil {
			h.logger.Debug().Uint64("height", height).Err(err).Msg("failed to get header from store")
		}
		return
	}

	if err := h.assertExpectedProposer(header.ProposerAddress); err != nil {
		h.logger.Debug().Uint64("height", height).Err(err).Msg("invalid header from P2P")
		return
	}

	data, err := h.dataStore.GetByHeight(ctx, height)
	if err != nil {
		if ctx.Err() == nil {
			h.logger.Debug().Uint64("height", height).Err(err).Msg("failed to get data from store")
		}
		return
	}

	dataCommitment := data.DACommitment()
	if !bytes.Equal(header.DataHash[:], dataCommitment[:]) {
		h.logger.Warn().
			Uint64("height", height).
			Str("header_data_hash", fmt.Sprintf("%x", header.DataHash)).
			Str("actual_data_hash", fmt.Sprintf("%x", dataCommitment)).
			Msg("DataHash mismatch: header and data do not match from P2P, discarding")
		return
	}

	h.emitEvent(height, header, data, heightInCh, "p2p_watch")
}

// assertExpectedProposer validates the proposer address
func (h *P2PHandler) assertExpectedProposer(proposerAddr []byte) error {
	if !bytes.Equal(h.genesis.ProposerAddress, proposerAddr) {
		return fmt.Errorf("proposer address mismatch: got %x, expected %x",
			proposerAddr, h.genesis.ProposerAddress)
	}
	return nil
}

func (h *P2PHandler) emitEvent(height uint64, header *types.SignedHeader, data *types.Data, heightInCh chan<- common.DAHeightEvent, source string) {
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

func (h *P2PHandler) removeInflight(height uint64) {
	h.mu.Lock()
	delete(h.inflight, height)
	h.mu.Unlock()
}
