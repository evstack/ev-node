package syncing

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
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
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

type daRetriever interface {
	RetrieveFromDA(ctx context.Context, daHeight uint64) ([]common.DAHeightEvent, error)
}
type p2pHandler interface {
	ProcessHeaderRange(ctx context.Context, fromHeight, toHeight uint64) []common.DAHeightEvent
	ProcessDataRange(ctx context.Context, fromHeight, toHeight uint64) []common.DAHeightEvent
}

// Syncer handles block synchronization from DA and P2P sources.
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
	options common.BlockOptions

	// State management
	lastState    types.State
	lastStateMtx *sync.RWMutex

	// DA state
	daHeight   uint64
	daStateMtx *sync.RWMutex

	// P2P stores
	headerStore goheader.Store[*types.SignedHeader]
	dataStore   goheader.Store[*types.Data]

	// Channels for coordination
	heightInCh    chan common.DAHeightEvent
	headerStoreCh chan struct{}
	dataStoreCh   chan struct{}
	errorCh       chan<- error // Channel to report critical execution client failures

	// Handlers
	daRetriever daRetriever
	p2pHandler  p2pHandler

	// Logging
	logger zerolog.Logger

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
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
	headerStore goheader.Store[*types.SignedHeader],
	dataStore goheader.Store[*types.Data],
	logger zerolog.Logger,
	options common.BlockOptions,
	errorCh chan<- error,
) *Syncer {
	return &Syncer{
		store:         store,
		exec:          exec,
		da:            da,
		cache:         cache,
		metrics:       metrics,
		config:        config,
		genesis:       genesis,
		options:       options,
		headerStore:   headerStore,
		dataStore:     dataStore,
		lastStateMtx:  &sync.RWMutex{},
		daStateMtx:    &sync.RWMutex{},
		heightInCh:    make(chan common.DAHeightEvent, 0),
		headerStoreCh: make(chan struct{}, 1),
		dataStoreCh:   make(chan struct{}, 1),
		errorCh:       errorCh,
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
	s.daRetriever = NewDARetriever(s.da, s.cache, s.config, s.genesis, s.options, s.logger)
	s.p2pHandler = NewP2PHandler(s.headerStore, s.dataStore, s.genesis, s.options, s.logger)

	// Start main processing loop
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.processLoop()
	}()

	// Start sync loop (DA and P2P retrieval)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.syncLoop()
	}()

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

// initializeState loads the current sync state
func (s *Syncer) initializeState() error {
	// Load state from store
	state, err := s.store.GetState(s.ctx)
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

	s.logger.Info().
		Uint64("height", state.LastBlockHeight).
		Uint64("da_height", s.GetDAHeight()).
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

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-blockTicker.C:
			// Signal P2P stores to check for new data
			s.sendNonBlockingSignal(s.headerStoreCh, "header_store")
			s.sendNonBlockingSignal(s.dataStoreCh, "data_store")
		case heightEvent := <-s.heightInCh:
			s.processHeightEvent(&heightEvent)
		}
	}
}

// syncLoop handles synchronization from DA and P2P sources.
func (s *Syncer) syncLoop() {
	s.logger.Info().Msg("starting sync loop")
	defer s.logger.Info().Msg("sync loop stopped")

	initialHeight, err := s.store.Height(s.ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get initial height")
		return
	}

	lastHeaderHeight := initialHeight
	lastDataHeight := initialHeight

	// Backoff control when DA replies with height-from-future
	var hffDelay time.Duration
	var nextDARequestAt time.Time

	//TODO: we should request to see what the head of the chain is at, then we know if we are falling behinf or in sync mode
syncLoop:
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}
		// Process pending events from cache on every iteration
		s.processPendingEvents()

		now := time.Now()
		// Respect backoff window if set
		if nextDARequestAt.IsZero() || now.After(nextDARequestAt) || now.Equal(nextDARequestAt) {
			// Retrieve from DA as fast as possible (unless throttled by HFF)
			events, err := s.daRetriever.RetrieveFromDA(s.ctx, s.GetDAHeight())
			if err != nil {
				if s.isHeightFromFutureError(err) {
					// Back off exactly by DA block time to avoid overloading
					hffDelay = s.config.DA.BlockTime.Duration
					if hffDelay <= 0 {
						hffDelay = 2 * time.Second
					}
					s.logger.Debug().Dur("delay", hffDelay).Uint64("da_height", s.GetDAHeight()).Msg("height from future; backing off DA requests")
					nextDARequestAt = now.Add(hffDelay)
				} else if errors.Is(err, coreda.ErrBlobNotFound) {
					// no data at this height, increase DA height
					s.SetDAHeight(s.GetDAHeight() + 1)
				} else {
					// Non-HFF errors: do not backoff artificially
					nextDARequestAt = time.Time{}
					s.logger.Error().Err(err).Msg("failed to retrieve from DA")
				}
			} else {
				// Reset backoff on success
				nextDARequestAt = time.Time{}

				// Process DA events
				for _, event := range events {
					select {
					case s.heightInCh <- event:
					default:
						s.logger.Warn().Msg("height channel full, dropping DA event")
						time.Sleep(10 * time.Millisecond)
						continue syncLoop
					}
				}

				// increment DA height on successful retrieval and continue immediately
				s.SetDAHeight(s.GetDAHeight() + 1)
				continue
			}
		}

		// Opportunistically process any P2P signals
		processedP2P := false
		select {
		case <-s.headerStoreCh:
			newHeaderHeight := s.headerStore.Height()
			if newHeaderHeight > lastHeaderHeight {
				events := s.p2pHandler.ProcessHeaderRange(s.ctx, lastHeaderHeight+1, newHeaderHeight)
				for _, event := range events {
					select {
					case s.heightInCh <- event:
					default:
						s.logger.Warn().Msg("height channel full, dropping P2P header event")
						time.Sleep(10 * time.Millisecond)
						continue syncLoop
					}
				}

				lastHeaderHeight = newHeaderHeight
			}
			processedP2P = true
		case <-s.dataStoreCh:
			newDataHeight := s.dataStore.Height()
			if newDataHeight > lastDataHeight {
				events := s.p2pHandler.ProcessDataRange(s.ctx, lastDataHeight+1, newDataHeight)
				for _, event := range events {
					select {
					case s.heightInCh <- event:
					default:
						s.logger.Warn().Msg("height channel full, dropping P2P data event")
						time.Sleep(10 * time.Millisecond)
						continue syncLoop
					}
				}
				lastDataHeight = newDataHeight
			}
			processedP2P = true
		default:
		}

		if !processedP2P {
			// Yield CPU to avoid tight spin when no events are available
			runtime.Gosched()
		}
	}
}

func (s *Syncer) processHeightEvent(event *common.DAHeightEvent) {
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
		pendingEvent := &common.DAHeightEvent{
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
		s.errorCh <- fmt.Errorf("failed to sync next block: %w", err)
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
		s.sendCriticalError(fmt.Errorf("failed to execute transactions: %w", err))
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

// sendCriticalError sends a critical error to the error channel without blocking
func (s *Syncer) sendCriticalError(err error) {
	if s.errorCh != nil {
		select {
		case s.errorCh <- err:
		default:
			// Channel full, error already reported
		}
	}
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

// sendNonBlockingSignal sends a signal without blocking
func (s *Syncer) sendNonBlockingSignal(ch chan struct{}, name string) {
	select {
	case ch <- struct{}{}:
	default:
		s.logger.Debug().Str("channel", name).Msg("channel full, signal dropped")
	}
}

// isHeightFromFutureError checks if the error is a height from future error
func (s *Syncer) isHeightFromFutureError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, coreda.ErrHeightFromFuture) || errors.Is(err, common.ErrHeightFromFutureStr) {
		return true
	}
	msg := err.Error()
	if msg == "" {
		return false
	}
	if strings.Contains(msg, coreda.ErrHeightFromFuture.Error()) || strings.Contains(msg, common.ErrHeightFromFutureStr.Error()) {
		return true
	}
	return false
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
			heightEvent := common.DAHeightEvent{
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
