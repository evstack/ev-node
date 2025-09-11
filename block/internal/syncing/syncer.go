package syncing

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"

	goheader "github.com/celestiaorg/go-header"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	coreda "github.com/evstack/ev-node/core/da"
	coreexecutor "github.com/evstack/ev-node/core/execution"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

// Syncer handles block synchronization, DA operations, and P2P coordination
type Syncer struct {
	// Core components
	store store.Store
	exec  coreexecutor.Executor
	da    coreda.DA

	// Shared components
	cache   cache.Manager
	metrics *common.Metrics

	// Configuration
	config  config.Config
	genesis genesis.Genesis
	signer  signer.Signer
	options common.BlockOptions

	// State management
	lastState    types.State
	lastStateMtx *sync.RWMutex

	// DA state
	daHeight         uint64
	daIncludedHeight uint64
	daStateMtx       *sync.RWMutex

	// P2P stores
	headerStore goheader.Store[*types.SignedHeader]
	dataStore   goheader.Store[*types.Data]

	// Channels for coordination
	heightInCh    chan HeightEvent
	headerStoreCh chan struct{}
	dataStoreCh   chan struct{}
	daIncluderCh  chan struct{}

	// Handlers
	daHandler  *DAHandler
	p2pHandler *P2PHandler

	// Logging
	logger zerolog.Logger

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// HeightEvent represents a block height event with header and data
type HeightEvent struct {
	Header                 *types.SignedHeader
	Data                   *types.Data
	DaHeight               uint64
	HeaderDaIncludedHeight uint64
}

// NewSyncer creates a new block syncer
func NewSyncer(
	store store.Store,
	exec coreexecutor.Executor,
	da coreda.DA,
	cache cache.Manager,
	metrics *common.Metrics,
	config config.Config,
	genesis genesis.Genesis,
	signer signer.Signer,
	headerStore goheader.Store[*types.SignedHeader],
	dataStore goheader.Store[*types.Data],
	logger zerolog.Logger,
	options common.BlockOptions,
) *Syncer {
	return &Syncer{
		store:         store,
		exec:          exec,
		da:            da,
		cache:         cache,
		metrics:       metrics,
		config:        config,
		genesis:       genesis,
		signer:        signer,
		options:       options,
		headerStore:   headerStore,
		dataStore:     dataStore,
		lastStateMtx:  &sync.RWMutex{},
		daStateMtx:    &sync.RWMutex{},
		heightInCh:    make(chan HeightEvent, 10000),
		headerStoreCh: make(chan struct{}, 1),
		dataStoreCh:   make(chan struct{}, 1),
		daIncluderCh:  make(chan struct{}, 1),
		logger:        logger.With().Str("component", "syncer").Logger(),
	}
}

// Start begins the syncing component
func (s *Syncer) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)

	// Initialize state
	if err := s.initializeState(); err != nil {
		return fmt.Errorf("failed to initialize syncer state: %w", err)
	}

	// Initialize handlers
	s.daHandler = NewDAHandler(s.da, s.cache, s.config, s.genesis, s.options, s.logger)
	s.p2pHandler = NewP2PHandler(s.headerStore, s.dataStore, s.cache, s.genesis, s.signer, s.options, s.logger)

	// Start main processing loop
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.processLoop()
	}()

	// Start combined submission loop (headers + data) only for aggregators
	if s.config.Node.Aggregator {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.submissionLoop()
		}()
	} else {
		// Start sync loop (DA and P2P retrieval)
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.syncLoop()
		}()
	}

	s.logger.Info().Msg("syncer started")
	return nil
}

// Stop shuts down the syncing component
func (s *Syncer) Stop() error {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
	s.logger.Info().Msg("syncer stopped")
	return nil
}

// GetLastState returns the current state
func (s *Syncer) GetLastState() types.State {
	s.lastStateMtx.RLock()
	defer s.lastStateMtx.RUnlock()
	return s.lastState
}

// SetLastState updates the current state
func (s *Syncer) SetLastState(state types.State) {
	s.lastStateMtx.Lock()
	defer s.lastStateMtx.Unlock()
	s.lastState = state
}

// GetDAHeight returns the current DA height
func (s *Syncer) GetDAHeight() uint64 {
	s.daStateMtx.RLock()
	defer s.daStateMtx.RUnlock()
	return s.daHeight
}

// SetDAHeight updates the DA height
func (s *Syncer) SetDAHeight(height uint64) {
	s.daStateMtx.Lock()
	defer s.daStateMtx.Unlock()
	s.daHeight = height
}

// GetDAIncludedHeight returns the DA included height
func (s *Syncer) GetDAIncludedHeight() uint64 {
	s.daStateMtx.RLock()
	defer s.daStateMtx.RUnlock()
	return s.daIncludedHeight
}

// SetDAIncludedHeight updates the DA included height
func (s *Syncer) SetDAIncludedHeight(height uint64) {
	s.daStateMtx.Lock()
	defer s.daStateMtx.Unlock()
	s.daIncludedHeight = height
}

// initializeState loads the current sync state
func (s *Syncer) initializeState() error {
	ctx := context.Background()

	// Load state from store
	state, err := s.store.GetState(ctx)
	if err != nil {
		// Use genesis state if no state exists
		state = types.State{
			ChainID:         s.genesis.ChainID,
			InitialHeight:   s.genesis.InitialHeight,
			LastBlockHeight: s.genesis.InitialHeight - 1,
			LastBlockTime:   s.genesis.StartTime,
			DAHeight:        0,
		}
	}

	s.SetLastState(state)

	// Set DA height
	daHeight := state.DAHeight
	if daHeight < s.config.DA.StartHeight {
		daHeight = s.config.DA.StartHeight
	}
	s.SetDAHeight(daHeight)

	// Load DA included height
	if height, err := s.store.GetMetadata(ctx, store.DAIncludedHeightKey); err == nil && len(height) == 8 {
		s.SetDAIncludedHeight(binary.LittleEndian.Uint64(height))
	}

	s.logger.Info().
		Uint64("height", state.LastBlockHeight).
		Uint64("da_height", s.GetDAHeight()).
		Uint64("da_included_height", s.GetDAIncludedHeight()).
		Str("chain_id", state.ChainID).
		Msg("initialized syncer state")

	return nil
}

// processLoop is the main coordination loop for processing events
func (s *Syncer) processLoop() {
	s.logger.Info().Msg("starting process loop")
	defer s.logger.Info().Msg("process loop stopped")

	blockTicker := time.NewTicker(s.config.Node.BlockTime.Duration)
	defer blockTicker.Stop()
	metricsTicker := time.NewTicker(30 * time.Second)
	defer metricsTicker.Stop()

	for {
		// Process pending events from cache on every iteration
		s.processPendingEvents()

		select {
		case <-s.ctx.Done():
			return
		case <-blockTicker.C:
			// Signal P2P stores to check for new data
			s.sendNonBlockingSignal(s.headerStoreCh, "header_store")
			s.sendNonBlockingSignal(s.dataStoreCh, "data_store")
		case heightEvent := <-s.heightInCh:
			s.processHeightEvent(&heightEvent)
		case <-metricsTicker.C:
			s.updateMetrics()
		case <-s.daIncluderCh:
			s.processDAInclusion()
		}
	}
}

// submissionLoop handles submission of headers and data to DA layer
func (s *Syncer) submissionLoop() {
	s.logger.Info().Msg("starting submission loop")
	defer s.logger.Info().Msg("submission loop stopped")

	ticker := time.NewTicker(s.config.DA.BlockTime.Duration)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			// Submit headers
			if s.cache.NumPendingHeaders() != 0 {
				if err := s.daHandler.SubmitHeaders(s.ctx, s.cache); err != nil {
					s.logger.Error().Err(err).Msg("failed to submit headers")
				}
			}

			// Submit data
			if s.cache.NumPendingData() != 0 {
				if err := s.daHandler.SubmitData(s.ctx, s.cache, s.signer, s.genesis); err != nil {
					s.logger.Error().Err(err).Msg("failed to submit data")
				}
			}
		}
	}
}

// syncLoop handles sync from DA and P2P sources
func (s *Syncer) syncLoop() {
	s.logger.Info().Msg("starting sync loop")
	defer s.logger.Info().Msg("sync loop stopped")

	daTicker := time.NewTicker(s.config.DA.BlockTime.Duration)
	defer daTicker.Stop()

	initialHeight, err := s.store.Height(s.ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get initial height")
		return
	}

	lastHeaderHeight := initialHeight
	lastDataHeight := initialHeight

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-daTicker.C:
			// Retrieve from DA
			events, err := s.daHandler.RetrieveFromDA(s.ctx, s.GetDAHeight())
			if err != nil {
				if !s.isHeightFromFutureError(err) {
					s.logger.Error().Err(err).Msg("failed to retrieve from DA")
				}
			} else {
				// Process DA events
				for _, event := range events {
					select {
					case s.heightInCh <- event:
					default:
						s.logger.Warn().Msg("height channel full, dropping DA event")
					}
				}
				// Increment DA height on successful retrieval
				s.SetDAHeight(s.GetDAHeight() + 1)
			}
		case <-s.headerStoreCh:
			// Check for new P2P headers
			newHeaderHeight := s.headerStore.Height()
			if newHeaderHeight > lastHeaderHeight {
				events := s.p2pHandler.ProcessHeaderRange(s.ctx, lastHeaderHeight+1, newHeaderHeight)
				for _, event := range events {
					select {
					case s.heightInCh <- event:
					default:
						s.logger.Warn().Msg("height channel full, dropping P2P header event")
					}
				}
				lastHeaderHeight = newHeaderHeight
			}
		case <-s.dataStoreCh:
			// Check for new P2P data
			newDataHeight := s.dataStore.Height()
			if newDataHeight > lastDataHeight {
				events := s.p2pHandler.ProcessDataRange(s.ctx, lastDataHeight+1, newDataHeight)
				for _, event := range events {
					select {
					case s.heightInCh <- event:
					default:
						s.logger.Warn().Msg("height channel full, dropping P2P data event")
					}
				}
				lastDataHeight = newDataHeight
			}
		}
	}
}

// processHeightEvent processes a height event for synchronization
func (s *Syncer) processHeightEvent(event *HeightEvent) {
	height := event.Header.Height()
	headerHash := event.Header.Hash().String()

	s.logger.Debug().
		Uint64("height", height).
		Uint64("da_height", event.DaHeight).
		Str("hash", headerHash).
		Msg("processing height event")

	currentHeight, err := s.store.Height(s.ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get current height")
		return
	}

	// Skip if already processed
	if height <= currentHeight || s.cache.IsHeaderSeen(headerHash) {
		s.logger.Debug().Uint64("height", height).Msg("height already processed")
		return
	}

	// Cache the header and data
	s.cache.SetHeader(height, event.Header)
	s.cache.SetData(height, event.Data)

	// If this is not the next block in sequence, store as pending event
	if height != currentHeight+1 {
		// Create a DAHeightEvent that matches the cache interface
		pendingEvent := &cache.DAHeightEvent{
			Header:                 event.Header,
			Data:                   event.Data,
			DaHeight:               event.DaHeight,
			HeaderDaIncludedHeight: event.HeaderDaIncludedHeight,
		}
		s.cache.SetPendingEvent(height, pendingEvent)
		s.logger.Debug().Uint64("height", height).Uint64("current_height", currentHeight).Msg("stored as pending event")
		return
	}

	// Try to sync the next block
	if err := s.trySyncNextBlock(event.DaHeight); err != nil {
		s.logger.Error().Err(err).Msg("failed to sync next block")
		return
	}

	// Mark as seen
	s.cache.SetHeaderSeen(headerHash)
	if !bytes.Equal(event.Header.DataHash, common.DataHashForEmptyTxs) {
		s.cache.SetDataSeen(event.Data.DACommitment().String())
	}
}

// trySyncNextBlock attempts to sync the next available block
func (s *Syncer) trySyncNextBlock(daHeight uint64) error {
	for {
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		default:
		}

		currentHeight, err := s.store.Height(s.ctx)
		if err != nil {
			return fmt.Errorf("failed to get current height: %w", err)
		}

		nextHeight := currentHeight + 1
		header := s.cache.GetHeader(nextHeight)
		if header == nil {
			s.logger.Debug().Uint64("height", nextHeight).Msg("header not available")
			return nil
		}

		data := s.cache.GetData(nextHeight)
		if data == nil {
			s.logger.Debug().Uint64("height", nextHeight).Msg("data not available")
			return nil
		}

		s.logger.Info().Uint64("height", nextHeight).Msg("syncing block")

		// Set custom verifier for sync node
		header.SetCustomVerifierForSyncNode(types.DefaultSyncNodeSignatureBytesProvider)

		// Apply block
		currentState := s.GetLastState()
		newState, err := s.applyBlock(header.Header, data, currentState)
		if err != nil {
			return fmt.Errorf("failed to apply block: %w", err)
		}

		// Validate block
		if err := s.validateBlock(currentState, header, data); err != nil {
			return fmt.Errorf("failed to validate block: %w", err)
		}

		// Save block
		if err := s.store.SaveBlockData(s.ctx, header, data, &header.Signature); err != nil {
			return fmt.Errorf("failed to save block: %w", err)
		}

		// Update height
		if err := s.store.SetHeight(s.ctx, nextHeight); err != nil {
			return fmt.Errorf("failed to update height: %w", err)
		}

		// Update state
		if daHeight > newState.DAHeight {
			newState.DAHeight = daHeight
		}
		if err := s.updateState(newState); err != nil {
			return fmt.Errorf("failed to update state: %w", err)
		}

		// Clear cache
		s.cache.ClearProcessedHeader(nextHeight)
		s.cache.ClearProcessedData(nextHeight)

		// Mark as seen
		s.cache.SetHeaderSeen(header.Hash().String())
		if !bytes.Equal(header.DataHash, common.DataHashForEmptyTxs) {
			s.cache.SetDataSeen(data.DACommitment().String())
		}
	}
}

// applyBlock applies a block to get the new state
func (s *Syncer) applyBlock(header types.Header, data *types.Data, currentState types.State) (types.State, error) {
	// Prepare transactions
	rawTxs := make([][]byte, len(data.Txs))
	for i, tx := range data.Txs {
		rawTxs[i] = []byte(tx)
	}

	// Execute transactions
	ctx := context.WithValue(s.ctx, types.HeaderContextKey, header)
	newAppHash, _, err := s.exec.ExecuteTxs(ctx, rawTxs, header.Height(),
		header.Time(), currentState.AppHash)
	if err != nil {
		return types.State{}, fmt.Errorf("failed to execute transactions: %w", err)
	}

	// Create new state
	newState, err := currentState.NextState(header, newAppHash)
	if err != nil {
		return types.State{}, fmt.Errorf("failed to create next state: %w", err)
	}

	return newState, nil
}

// validateBlock validates a synced block
func (s *Syncer) validateBlock(lastState types.State, header *types.SignedHeader, data *types.Data) error {
	// Set custom verifier for aggregator node signature
	header.SetCustomVerifierForSyncNode(s.options.SyncNodeSignatureBytesProvider)

	// Validate header with data
	if err := header.ValidateBasicWithData(data); err != nil {
		return fmt.Errorf("header-data validation failed: %w", err)
	}

	return nil
}

// updateState saves the new state
func (s *Syncer) updateState(newState types.State) error {
	if err := s.store.UpdateState(s.ctx, newState); err != nil {
		return err
	}

	s.SetLastState(newState)
	s.metrics.Height.Set(float64(newState.LastBlockHeight))

	return nil
}

// processDAInclusion processes DA inclusion tracking
func (s *Syncer) processDAInclusion() {
	currentDAIncluded := s.GetDAIncludedHeight()

	for {
		nextHeight := currentDAIncluded + 1

		// Check if this height is DA included
		if included, err := s.isHeightDAIncluded(nextHeight); err != nil || !included {
			break
		}

		s.logger.Debug().Uint64("height", nextHeight).Msg("advancing DA included height")

		// Set final height in executor
		if err := s.exec.SetFinal(s.ctx, nextHeight); err != nil {
			s.logger.Error().Err(err).Uint64("height", nextHeight).Msg("failed to set final height")
			break
		}

		// Update DA included height
		s.SetDAIncludedHeight(nextHeight)
		currentDAIncluded = nextHeight
	}
}

// isHeightDAIncluded checks if a height is included in DA
func (s *Syncer) isHeightDAIncluded(height uint64) (bool, error) {
	currentHeight, err := s.store.Height(s.ctx)
	if err != nil {
		return false, err
	}
	if currentHeight < height {
		return false, nil
	}

	header, data, err := s.store.GetBlockData(s.ctx, height)
	if err != nil {
		return false, err
	}

	headerHash := header.Hash().String()
	dataHash := data.DACommitment().String()

	headerIncluded := s.cache.IsHeaderDAIncluded(headerHash)
	dataIncluded := bytes.Equal(data.DACommitment(), common.DataHashForEmptyTxs) || s.cache.IsDataDAIncluded(dataHash)

	return headerIncluded && dataIncluded, nil
}

// sendNonBlockingSignal sends a signal without blocking
func (s *Syncer) sendNonBlockingSignal(ch chan struct{}, name string) {
	select {
	case ch <- struct{}{}:
	default:
		s.logger.Debug().Str("channel", name).Msg("channel full, signal dropped")
	}
}

// updateMetrics updates sync-related metrics
func (s *Syncer) updateMetrics() {
	// Update pending counts (only relevant for aggregators)
	if s.config.Node.Aggregator {
		s.metrics.PendingHeadersCount.Set(float64(s.cache.NumPendingHeaders()))
		s.metrics.PendingDataCount.Set(float64(s.cache.NumPendingData()))
	}
	s.metrics.DAInclusionHeight.Set(float64(s.GetDAIncludedHeight()))
}

// isHeightFromFutureError checks if the error is a height from future error
func (s *Syncer) isHeightFromFutureError(err error) bool {
	return err != nil && (errors.Is(err, common.ErrHeightFromFutureStr) ||
		(err.Error() != "" && bytes.Contains([]byte(err.Error()), []byte(common.ErrHeightFromFutureStr.Error()))))
}

// processPendingEvents fetches and processes pending events from cache
func (s *Syncer) processPendingEvents() {
	pendingEvents := s.cache.GetPendingEvents()

	for height, event := range pendingEvents {
		currentHeight, err := s.store.Height(s.ctx)
		if err != nil {
			s.logger.Error().Err(err).Msg("failed to get current height for pending events")
			continue
		}

		// Only process events for blocks we haven't synced yet
		if height > currentHeight {
			heightEvent := HeightEvent{
				Header:                 event.Header,
				Data:                   event.Data,
				DaHeight:               event.DaHeight,
				HeaderDaIncludedHeight: event.HeaderDaIncludedHeight,
			}

			select {
			case s.heightInCh <- heightEvent:
				// Remove from pending events once sent
				s.cache.DeletePendingEvent(height)
			case <-s.ctx.Done():
				return
			}
		} else {
			// Clean up events for blocks we've already processed
			s.cache.DeletePendingEvent(height)
		}
	}
}
