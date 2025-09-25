package syncing

import (
	"bytes"
	"context"
	"errors"
	"fmt"
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
	errorCh chan<- error // Channel to report critical execution client failures

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
		store:        store,
		exec:         exec,
		da:           da,
		cache:        cache,
		metrics:      metrics,
		config:       config,
		genesis:      genesis,
		options:      options,
		headerStore:  headerStore,
		dataStore:    dataStore,
		lastStateMtx: &sync.RWMutex{},
		daStateMtx:   &sync.RWMutex{},
		errorCh:      errorCh,
		logger:       logger.With().Str("component", "syncer").Logger(),
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
	daHeight := max(state.DAHeight, s.config.DA.StartHeight)
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

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			currentHeight, err := s.store.Height(s.ctx)
			if err != nil {
				s.logger.Error().Err(err).Msg("failed to get current height for pending events")
				continue
			}

			// Try to get the next processable event (currentHeight + 1)
			nextHeight := currentHeight + 1
			for {
				event := s.cache.GetNextPendingEvent(nextHeight)
				if event == nil {
					break
				}
				s.processHeightEvent(event)
				nextHeight++
			}
		}
	}
}

// syncLoop handles synchronization from DA and P2P sources.
// all event are sent to the cache for processing.
func (s *Syncer) syncLoop() {
	s.logger.Info().Msg("starting sync loop")
	defer s.logger.Info().Msg("sync loop stopped")

	if delay := time.Until(s.genesis.StartTime); delay > 0 {
		s.logger.Info().Dur("delay", delay).Msg("waiting until genesis to start syncing")
		select {
		case <-s.ctx.Done():
			return
		case <-time.After(delay):
		}
	}

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

	blockTicker := time.NewTicker(s.config.Node.BlockTime.Duration)
	defer blockTicker.Stop()

	// TODO: we should request to see what the head of the chain is at
	// then we know if we are falling behind or in sync mode
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

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
					s.cache.SetPendingEvent(event.Header.Height(), &event)
				}

				// increment DA height on successful retrieval and continue immediately
				s.SetDAHeight(s.GetDAHeight() + 1)
				continue
			}
		}

		// Opportunistically process any P2P signals
		select {
		case <-blockTicker.C:
			newHeaderHeight := s.headerStore.Height()
			if newHeaderHeight > lastHeaderHeight {
				events := s.p2pHandler.ProcessHeaderRange(s.ctx, lastHeaderHeight+1, newHeaderHeight)
				for _, event := range events {
					s.cache.SetPendingEvent(event.Header.Height(), &event)
				}
				lastHeaderHeight = newHeaderHeight
			}

			newDataHeight := s.dataStore.Height()
			if newDataHeight == newHeaderHeight {
				lastDataHeight = newDataHeight
				continue
			}
			if newDataHeight > lastDataHeight {
				events := s.p2pHandler.ProcessDataRange(s.ctx, lastDataHeight+1, newDataHeight)
				for _, event := range events {
					s.cache.SetPendingEvent(event.Header.Height(), &event)
				}
				lastDataHeight = newDataHeight
			}
		default:
			// Prevent busy-waiting when no events are available.
			waitTime := min(10*time.Millisecond, s.config.Node.BlockTime.Duration)
			time.Sleep(waitTime)
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

	// If this is not the next block in sequence, store as pending event
	// This check is crucial as trySyncNextBlock simply attempts to sync the next block
	if height != currentHeight+1 {
		s.cache.SetPendingEvent(height, event)
		s.logger.Debug().Uint64("height", height).Uint64("current_height", currentHeight).Msg("stored as pending event")
		return
	}

	// Try to sync the next block
	if err := s.trySyncNextBlock(event); err != nil {
		s.logger.Error().Err(err).Msg("failed to sync next block")
		// If the error is not due to an validation error, re-store the event as pending
		if !errors.Is(err, errInvalidBlock) {
			s.cache.SetPendingEvent(height, event)
		}
		return
	}
}

// errInvalidBlock is returned when a block is failing validation
var errInvalidBlock = errors.New("invalid block")

// trySyncNextBlock attempts to sync the next available block
// the event is always the next block in sequence as processHeightEvent ensures it.
func (s *Syncer) trySyncNextBlock(event *common.DAHeightEvent) error {
	select {
	case <-s.ctx.Done():
		return s.ctx.Err()
	default:
	}

	header := event.Header
	data := event.Data
	nextHeight := event.Header.Height()
	currentState := s.GetLastState()

	s.logger.Info().Uint64("height", nextHeight).Msg("syncing block")

	// Compared to the executor logic where the current block needs to be applied first,
	// here only the previous block needs to be applied to proceed to the verification.
	// The header validation must be done before applying the block to avoid executing gibberish
	if err := s.validateBlock(currentState, header, data); err != nil {
		return errors.Join(errInvalidBlock, fmt.Errorf("failed to validate block: %w", err))
	}

	// Mark as DA included
	headerHash := header.Hash().String()
	s.cache.SetHeaderDAIncluded(headerHash, event.HeaderDaIncludedHeight)

	s.logger.Info().
		Str("header_hash", headerHash).
		Uint64("da_height", event.HeaderDaIncludedHeight).
		Uint64("height", header.Height()).
		Msg("header marked as DA included")

	// Apply block
	newState, err := s.applyBlock(header.Header, data, currentState)
	if err != nil {
		return fmt.Errorf("failed to apply block: %w", err)
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
	if event.DaHeight > newState.DAHeight {
		newState.DAHeight = event.DaHeight
	}
	if err := s.updateState(newState); err != nil {
		return fmt.Errorf("failed to update state: %w", err)
	}

	// Mark as seen
	s.cache.SetHeaderSeen(header.Hash().String())
	if !bytes.Equal(header.DataHash, common.DataHashForEmptyTxs) {
		s.cache.SetDataSeen(data.DACommitment().String())
	}

	return nil
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
func (s *Syncer) validateBlock(
	lastState types.State,
	header *types.SignedHeader,
	data *types.Data,
) error {
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
