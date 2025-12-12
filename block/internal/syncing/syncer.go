package syncing

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	coreda "github.com/evstack/ev-node/core/da"
	coreexecutor "github.com/evstack/ev-node/core/execution"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

type daRetriever interface {
	RetrieveFromDA(ctx context.Context, daHeight uint64) ([]common.DAHeightEvent, error)
}

type p2pHandler interface {
	ProcessHeight(ctx context.Context, height uint64, heightInCh chan<- common.DAHeightEvent) error
	SetProcessedHeight(height uint64)
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
	lastState *atomic.Pointer[types.State]

	// DA retriever height
	daRetrieverHeight *atomic.Uint64

	// P2P stores
	headerStore common.Broadcaster[*types.SignedHeader]
	dataStore   common.Broadcaster[*types.Data]

	// Channels for coordination
	heightInCh chan common.DAHeightEvent
	errorCh    chan<- error // Channel to report critical execution client failures

	// Handlers
	daRetriever daRetriever
	p2pHandler  p2pHandler

	// Logging
	logger zerolog.Logger

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// P2P wait coordination
	p2pWaitState atomic.Value // stores p2pWaitState
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
	headerStore common.Broadcaster[*types.SignedHeader],
	dataStore common.Broadcaster[*types.Data],
	logger zerolog.Logger,
	options common.BlockOptions,
	errorCh chan<- error,
) *Syncer {
	return &Syncer{
		store:             store,
		exec:              exec,
		da:                da,
		cache:             cache,
		metrics:           metrics,
		config:            config,
		genesis:           genesis,
		options:           options,
		headerStore:       headerStore,
		dataStore:         dataStore,
		lastState:         &atomic.Pointer[types.State]{},
		daRetrieverHeight: &atomic.Uint64{},
		heightInCh:        make(chan common.DAHeightEvent, 1_000),
		errorCh:           errorCh,
		logger:            logger.With().Str("component", "syncer").Logger(),
	}
}

// Start begins the syncing component
func (s *Syncer) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)

	if err := s.initializeState(); err != nil {
		return fmt.Errorf("failed to initialize syncer state: %w", err)
	}

	// Initialize handlers
	s.daRetriever = NewDARetriever(s.da, s.cache, s.config, s.genesis, s.logger)
	s.p2pHandler = NewP2PHandler(s.headerStore.Store(), s.dataStore.Store(), s.cache, s.genesis, s.logger)
	if currentHeight, err := s.store.Height(s.ctx); err != nil {
		s.logger.Error().Err(err).Msg("failed to set initial processed height for p2p handler")
	} else {
		s.p2pHandler.SetProcessedHeight(currentHeight)
	}

	if !s.waitForGenesis() {
		return nil
	}

	// Start main processing loop
	go s.processLoop()

	// Start dedicated workers for DA, and pending processing
	s.startSyncWorkers()

	s.logger.Info().Msg("syncer started")
	return nil
}

// Stop shuts down the syncing component
func (s *Syncer) Stop() error {
	if s.cancel != nil {
		s.cancel()
	}
	s.cancelP2PWait(0)
	s.wg.Wait()
	s.logger.Info().Msg("syncer stopped")
	return nil
}

// GetLastState returns the current state
func (s *Syncer) GetLastState() types.State {
	state := s.lastState.Load()
	if state == nil {
		return types.State{}
	}

	stateCopy := *state
	stateCopy.AppHash = bytes.Clone(state.AppHash)
	stateCopy.LastHeaderHash = bytes.Clone(state.LastHeaderHash)

	return stateCopy
}

// SetLastState updates the current state
func (s *Syncer) SetLastState(state types.State) {
	s.lastState.Store(&state)
}

// initializeState loads the current sync state
func (s *Syncer) initializeState() error {
	// Load state from store
	state, err := s.store.GetState(s.ctx)
	if err != nil {
		// Initialize new chain state for a fresh full node (no prior state on disk)
		// Mirror executor initialization to ensure AppHash matches headers produced by the sequencer.
		stateRoot, _, initErr := s.exec.InitChain(
			s.ctx,
			s.genesis.StartTime,
			s.genesis.InitialHeight,
			s.genesis.ChainID,
		)
		if initErr != nil {
			return fmt.Errorf("failed to initialize execution client: %w", initErr)
		}

		state = types.State{
			ChainID:         s.genesis.ChainID,
			InitialHeight:   s.genesis.InitialHeight,
			LastBlockHeight: s.genesis.InitialHeight - 1,
			LastBlockTime:   s.genesis.StartTime,
			DAHeight:        s.genesis.DAStartHeight,
			AppHash:         stateRoot,
		}
	}
	if state.DAHeight != 0 && state.DAHeight < s.genesis.DAStartHeight { // state.DaHeight can be 0 if the node starts from sequencer state
		return fmt.Errorf("DA height (%d) is lower than DA start height (%d)", state.DAHeight, s.genesis.DAStartHeight)
	}
	s.SetLastState(state)

	// Set DA height to the maximum of the genesis start height, the state's DA height, and the cached DA height.
	// This ensures we resume from the highest known DA height, even if the cache is cleared on restart. If the DA height is too high because of a user error, reset it with --evnode.clear_cache. The DA height will be back to the last highest known executed DA height for a height.
	s.daRetrieverHeight.Store(max(s.genesis.DAStartHeight, s.cache.DaHeight(), state.DAHeight))

	s.logger.Info().
		Uint64("height", state.LastBlockHeight).
		Uint64("da_height", s.daRetrieverHeight.Load()).
		Str("chain_id", state.ChainID).
		Msg("initialized syncer state")

	// Sync execution layer with store on startup
	execReplayer := common.NewReplayer(s.store, s.exec, s.genesis, s.logger)
	if err := execReplayer.SyncToHeight(s.ctx, state.LastBlockHeight); err != nil {
		return fmt.Errorf("failed to sync execution layer on startup: %w", err)
	}

	return nil
}

// processLoop is the main coordination loop for processing events
func (s *Syncer) processLoop() {
	s.wg.Add(1)
	defer s.wg.Done()

	s.logger.Info().Msg("starting process loop")
	defer s.logger.Info().Msg("process loop stopped")

	for {
		select {
		case <-s.ctx.Done():
			return
		case heightEvent := <-s.heightInCh:
			s.processHeightEvent(&heightEvent)
		}
	}
}

func (s *Syncer) startSyncWorkers() {
	s.wg.Add(3)
	go s.daWorkerLoop()
	go s.pendingWorkerLoop()
	go s.p2pWorkerLoop()
}

func (s *Syncer) daWorkerLoop() {
	defer s.wg.Done()

	s.logger.Info().Msg("starting DA worker")
	defer s.logger.Info().Msg("DA worker stopped")

	for {
		err := s.fetchDAUntilCaughtUp()

		var backoff time.Duration
		if err == nil {
			// No error, means we are caught up.
			backoff = s.config.DA.BlockTime.Duration
		} else {
			// Error, back off for a shorter duration.
			backoff = s.config.DA.BlockTime.Duration
			if backoff <= 0 {
				backoff = 2 * time.Second
			}
		}

		select {
		case <-s.ctx.Done():
			return
		case <-time.After(backoff):
		}
	}
}

func (s *Syncer) fetchDAUntilCaughtUp() error {
	for {
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		default:
		}

		daHeight := max(s.daRetrieverHeight.Load(), s.cache.DaHeight())

		events, err := s.daRetriever.RetrieveFromDA(s.ctx, daHeight)
		if err != nil {
			switch {
			case errors.Is(err, coreda.ErrBlobNotFound):
				s.daRetrieverHeight.Store(daHeight + 1)
				continue // Fetch next height immediately
			case errors.Is(err, coreda.ErrHeightFromFuture):
				s.logger.Debug().Err(err).Uint64("da_height", daHeight).Msg("DA is ahead of local target; backing off future height requests")
				return nil // Caught up
			default:
				s.logger.Error().Err(err).Uint64("da_height", daHeight).Msg("failed to retrieve from DA; backing off DA requests")
				return err // Other errors
			}
		}

		if len(events) == 0 {
			// This can happen if RetrieveFromDA returns no events and no error.
			s.logger.Debug().Uint64("da_height", daHeight).Msg("no events returned from DA, but no error either.")
		}

		// Process DA events
		for _, event := range events {
			select {
			case s.heightInCh <- event:
			default:
				s.cache.SetPendingEvent(event.Header.Height(), &event)
			}
		}

		// increment DA retrieval height on successful retrieval
		s.daRetrieverHeight.Store(daHeight + 1)
	}
}

func (s *Syncer) pendingWorkerLoop() {
	defer s.wg.Done()

	s.logger.Info().Msg("starting pending worker")
	defer s.logger.Info().Msg("pending worker stopped")

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.processPendingEvents()
		}
	}
}

func (s *Syncer) p2pWorkerLoop() {
	defer s.wg.Done()

	logger := s.logger.With().Str("worker", "p2p").Logger()
	logger.Info().Msg("starting P2P worker")
	defer logger.Info().Msg("P2P worker stopped")

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		currentHeight, err := s.store.Height(s.ctx)
		if err != nil {
			logger.Error().Err(err).Msg("failed to get current height for P2P worker")
			if !s.sleepOrDone(50 * time.Millisecond) {
				return
			}
			continue
		}

		targetHeight := currentHeight + 1
		waitCtx, cancel := context.WithCancel(s.ctx)
		s.setP2PWaitState(targetHeight, cancel)

		err = s.p2pHandler.ProcessHeight(waitCtx, targetHeight, s.heightInCh)
		s.cancelP2PWait(targetHeight)

		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				continue
			}

			if waitCtx.Err() == nil {
				logger.Warn().Err(err).Uint64("height", targetHeight).Msg("P2P handler failed to process height")
			}

			if !s.sleepOrDone(50 * time.Millisecond) {
				return
			}
			continue
		}

		if err := s.waitForStoreHeight(targetHeight); err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			logger.Error().Err(err).Uint64("height", targetHeight).Msg("failed waiting for height commit")
		}
	}
}

func (s *Syncer) waitForGenesis() bool {
	if delay := time.Until(s.genesis.StartTime); delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-s.ctx.Done():
			return false
		case <-timer.C:
		}
	}
	return true
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

	// Last data must be got from store if the event comes from DA and the data hash is empty.
	// When if the event comes from P2P, the sequencer and then all the full nodes contains the data.
	if event.Source == common.SourceDA && bytes.Equal(event.Header.DataHash, common.DataHashForEmptyTxs) && currentHeight > 0 {
		_, lastData, err := s.store.GetBlockData(s.ctx, currentHeight)
		if err != nil {
			s.logger.Error().Err(err).Msg("failed to get last data")
			return
		}
		event.Data.LastDataHash = lastData.Hash()
	}

	// Cancel any P2P wait that might still be blocked on this height, as we have a block for it.
	s.cancelP2PWait(height)

	// Try to sync the next block
	if err := s.trySyncNextBlock(event); err != nil {
		s.logger.Error().Err(err).Msg("failed to sync next block")
		// If the error is not due to an validation error, re-store the event as pending
		switch {
		case errors.Is(err, errInvalidBlock):
			// do not reschedule
		case errors.Is(err, errInvalidState):
			s.sendCriticalError(fmt.Errorf("invalid state detected (block-height %d, state-height %d) "+
				"- block references do not match local state. Manual intervention required: %w", event.Header.Height(),
				s.GetLastState().LastBlockHeight, err))
		default:
			s.cache.SetPendingEvent(height, event)
		}
		return
	}

	// only save to p2p stores if the event came from DA
	if event.Source == common.SourceDA {
		g, ctx := errgroup.WithContext(s.ctx)
		g.Go(func() error {
			// broadcast header locally only — prevents spamming the p2p network with old height notifications,
			// allowing the syncer to update its target and fill missing blocks
			return s.headerStore.WriteToStoreAndBroadcast(ctx, event.Header, pubsub.WithLocalPublication(true))
		})
		g.Go(func() error {
			// broadcast data locally only — prevents spamming the p2p network with old height notifications,
			// allowing the syncer to update its target and fill missing blocks
			return s.dataStore.WriteToStoreAndBroadcast(ctx, event.Data, pubsub.WithLocalPublication(true))
		})
		if err := g.Wait(); err != nil {
			s.logger.Error().Err(err).Msg("failed to append event header and/or data to p2p store")
		}
	}
}

var (
	// errInvalidBlock is returned when a block is failing validation
	errInvalidBlock = errors.New("invalid block")
	// errInvalidState is returned when the state has diverged from the DA blocks
	errInvalidState = errors.New("invalid state")
)

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
	headerHash := header.Hash().String()

	s.logger.Info().Uint64("height", nextHeight).Msg("syncing block")

	// Compared to the executor logic where the current block needs to be applied first,
	// here only the previous block needs to be applied to proceed to the verification.
	// The header validation must be done before applying the block to avoid executing gibberish
	if err := s.validateBlock(currentState, data, header); err != nil {
		// remove header as da included (not per se needed, but keep cache clean)
		s.cache.RemoveHeaderDAIncluded(headerHash)
		if !errors.Is(err, errInvalidState) && !errors.Is(err, errInvalidBlock) {
			return errors.Join(errInvalidBlock, err)
		}
		return err
	}

	newState, err := s.applyBlock(header.Header, data, currentState)
	if err != nil {
		return fmt.Errorf("failed to apply block: %w", err)
	}

	// Update DA height if needed
	// This height is only updated when a height is processed from DA as P2P
	// events do not contain DA height information
	if event.DaHeight > newState.DAHeight {
		newState.DAHeight = event.DaHeight
	}

	batch, err := s.store.NewBatch(s.ctx)
	if err != nil {
		return fmt.Errorf("failed to create batch: %w", err)
	}

	if err := batch.SaveBlockData(header, data, &header.Signature); err != nil {
		return fmt.Errorf("failed to save block: %w", err)
	}

	if err := batch.SetHeight(nextHeight); err != nil {
		return fmt.Errorf("failed to update height: %w", err)
	}

	if err := batch.UpdateState(newState); err != nil {
		return fmt.Errorf("failed to update state: %w", err)
	}

	if err := batch.Commit(); err != nil {
		return fmt.Errorf("failed to commit batch: %w", err)
	}

	// Update in-memory state after successful commit
	s.SetLastState(newState)
	s.metrics.Height.Set(float64(newState.LastBlockHeight))

	// Mark as seen
	s.cache.SetHeaderSeen(headerHash, header.Height())
	if !bytes.Equal(header.DataHash, common.DataHashForEmptyTxs) {
		s.cache.SetDataSeen(data.DACommitment().String(), newState.LastBlockHeight)
	}

	if s.p2pHandler != nil {
		s.p2pHandler.SetProcessedHeight(newState.LastBlockHeight)
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
	newAppHash, err := s.executeTxsWithRetry(ctx, rawTxs, header, currentState)
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

// executeTxsWithRetry executes transactions with retry logic
func (s *Syncer) executeTxsWithRetry(ctx context.Context, rawTxs [][]byte, header types.Header, currentState types.State) ([]byte, error) {
	for attempt := 1; attempt <= common.MaxRetriesBeforeHalt; attempt++ {
		newAppHash, _, err := s.exec.ExecuteTxs(ctx, rawTxs, header.Height(), header.Time(), currentState.AppHash)
		if err != nil {
			if attempt == common.MaxRetriesBeforeHalt {
				return nil, fmt.Errorf("failed to execute transactions: %w", err)
			}

			s.logger.Error().Err(err).
				Int("attempt", attempt).
				Int("max_attempts", common.MaxRetriesBeforeHalt).
				Uint64("height", header.Height()).
				Msg("failed to execute transactions, retrying")

			select {
			case <-time.After(common.MaxRetriesTimeout):
				continue
			case <-s.ctx.Done():
				return nil, fmt.Errorf("context cancelled during retry: %w", s.ctx.Err())
			}
		}

		return newAppHash, nil
	}

	return nil, nil
}

// validateBlock validates a synced block
// NOTE: if the header was gibberish and somehow passed all validation prior but the data was correct
// or if the data was gibberish and somehow passed all validation prior but the header was correct
// we are still losing both in the pending event. This should never happen.
func (s *Syncer) validateBlock(currState types.State, data *types.Data, header *types.SignedHeader) error {
	// Set custom verifier for aggregator node signature
	header.SetCustomVerifierForSyncNode(s.options.SyncNodeSignatureBytesProvider)

	if err := header.ValidateBasicWithData(data); err != nil {
		return fmt.Errorf("invalid header: %w", err)
	}

	if err := currState.AssertValidForNextState(header, data); err != nil {
		return errors.Join(errInvalidState, err)
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

// processPendingEvents fetches and processes pending events from cache
// optimistically fetches the next events from cache until no matching heights are found
func (s *Syncer) processPendingEvents() {
	currentHeight, err := s.store.Height(s.ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get current height for pending events")
		return
	}

	// Try to get the next processable event (currentHeight + 1)
	nextHeight := currentHeight + 1
	for {
		event := s.cache.GetNextPendingEvent(nextHeight)
		if event == nil {
			return
		}

		heightEvent := common.DAHeightEvent{
			Header:   event.Header,
			Data:     event.Data,
			DaHeight: event.DaHeight,
			Source:   event.Source,
		}

		select {
		case s.heightInCh <- heightEvent:
			// Event was successfully sent and already removed by GetNextPendingEvent
			s.logger.Debug().Uint64("height", nextHeight).Msg("sent pending event to processing")
		case <-s.ctx.Done():
			s.cache.SetPendingEvent(nextHeight, event)
			return
		default:
			s.cache.SetPendingEvent(nextHeight, event)
			return
		}

		nextHeight++
	}
}

func (s *Syncer) waitForStoreHeight(target uint64) error {
	for {
		currentHeight, err := s.store.Height(s.ctx)
		if err != nil {
			return err
		}

		if currentHeight >= target {
			return nil
		}

		if !s.sleepOrDone(10 * time.Millisecond) {
			if s.ctx.Err() != nil {
				return s.ctx.Err()
			}
		}
	}
}

func (s *Syncer) sleepOrDone(duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-s.ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

type p2pWaitState struct {
	height uint64
	cancel context.CancelFunc
}

func (s *Syncer) setP2PWaitState(height uint64, cancel context.CancelFunc) {
	s.p2pWaitState.Store(p2pWaitState{height: height, cancel: cancel})
}

func (s *Syncer) cancelP2PWait(height uint64) {
	val := s.p2pWaitState.Load()
	if val == nil {
		return
	}
	state, ok := val.(p2pWaitState)
	if !ok || state.cancel == nil {
		return
	}

	if height == 0 || state.height <= height {
		s.p2pWaitState.Store(p2pWaitState{})
		state.cancel()
	}
}
