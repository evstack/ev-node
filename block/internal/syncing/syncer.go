package syncing

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	coreexecutor "github.com/evstack/ev-node/core/execution"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/block/internal/da"
	"github.com/evstack/ev-node/pkg/config"
	datypes "github.com/evstack/ev-node/pkg/da/types"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

// forcedInclusionGracePeriodConfig contains internal configuration for forced inclusion grace periods.
type forcedInclusionGracePeriodConfig struct {
	// basePeriod is the base number of additional epochs allowed for including forced inclusion transactions
	// before marking the sequencer as malicious. This provides tolerance for temporary chain congestion.
	// A value of 0 means strict enforcement (no grace period).
	// A value of 1 means transactions from epoch N can be included in epoch N+1 without being marked malicious.
	// Recommended: 1 epoch.
	basePeriod uint64

	// dynamicMinMultiplier is the minimum multiplier for the base grace period.
	// The actual grace period will be at least: basePeriod * dynamicMinMultiplier.
	// Example: base=2, min=0.5 → minimum grace period is 1 epoch.
	dynamicMinMultiplier float64

	// dynamicMaxMultiplier is the maximum multiplier for the base grace period.
	// The actual grace period will be at most: basePeriod * dynamicMaxMultiplier.
	// Example: base=2, max=3.0 → maximum grace period is 6 epochs.
	dynamicMaxMultiplier float64

	// dynamicFullnessThreshold defines what percentage of block capacity is considered "full".
	// When EMA of block fullness is above this threshold, grace period increases.
	// When below, grace period decreases. Value should be between 0.0 and 1.0.
	dynamicFullnessThreshold float64

	// dynamicAdjustmentRate controls how quickly the grace period multiplier adapts.
	// Higher values make it adapt faster to congestion changes. Value should be between 0.0 and 1.0.
	// Recommended: 0.05 for gradual adjustment, 0.1 for faster response.
	dynamicAdjustmentRate float64
}

// newForcedInclusionGracePeriodConfig returns the internal grace period configuration.
func newForcedInclusionGracePeriodConfig() forcedInclusionGracePeriodConfig {
	return forcedInclusionGracePeriodConfig{
		basePeriod:               1,    // 1 epoch grace period
		dynamicMinMultiplier:     0.5,  // Minimum 0.5x base grace period
		dynamicMaxMultiplier:     3.0,  // Maximum 3x base grace period
		dynamicFullnessThreshold: 0.8,  // 80% capacity considered full
		dynamicAdjustmentRate:    0.05, // 5% adjustment per block
	}
}

// Syncer handles block synchronization from DA and P2P sources.
type Syncer struct {
	// Core components
	store store.Store
	exec  coreexecutor.Executor

	// Shared components
	cache   cache.CacheManager
	metrics *common.Metrics

	// Configuration
	config  config.Config
	genesis genesis.Genesis
	options common.BlockOptions
	logger  zerolog.Logger

	// State management
	lastState *atomic.Pointer[types.State]

	// DA retriever
	daClient          da.Client
	daRetrieverHeight *atomic.Uint64

	// P2P stores
	headerStore common.Broadcaster[*types.SignedHeader]
	dataStore   common.Broadcaster[*types.Data]

	// Channels for coordination
	heightInCh chan common.DAHeightEvent
	errorCh    chan<- error // Channel to report critical execution client failures

	// Handlers
	daRetriever DARetriever
	fiRetriever *da.ForcedInclusionRetriever
	p2pHandler  p2pHandler

	// Forced inclusion tracking
	pendingForcedInclusionTxs sync.Map // map[string]pendingForcedInclusionTx
	gracePeriodMultiplier     *atomic.Pointer[float64]
	blockFullnessEMA          *atomic.Pointer[float64]
	gracePeriodConfig         forcedInclusionGracePeriodConfig

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// P2P wait coordination
	p2pWaitState atomic.Value // stores p2pWaitState
}

// pendingForcedInclusionTx represents a forced inclusion transaction that hasn't been included yet
type pendingForcedInclusionTx struct {
	Data       []byte
	EpochStart uint64
	EpochEnd   uint64
	TxHash     string
}

// NewSyncer creates a new block syncer
func NewSyncer(
	store store.Store,
	exec coreexecutor.Executor,
	daClient da.Client,
	cache cache.CacheManager,
	metrics *common.Metrics,
	config config.Config,
	genesis genesis.Genesis,
	headerStore common.Broadcaster[*types.SignedHeader],
	dataStore common.Broadcaster[*types.Data],
	logger zerolog.Logger,
	options common.BlockOptions,
	errorCh chan<- error,
) *Syncer {
	daRetrieverHeight := &atomic.Uint64{}
	daRetrieverHeight.Store(genesis.DAStartHeight)

	// Initialize dynamic grace period state
	initialMultiplier := 1.0
	gracePeriodMultiplier := &atomic.Pointer[float64]{}
	gracePeriodMultiplier.Store(&initialMultiplier)

	initialFullness := 0.0
	blockFullnessEMA := &atomic.Pointer[float64]{}
	blockFullnessEMA.Store(&initialFullness)

	return &Syncer{
		store:                 store,
		exec:                  exec,
		cache:                 cache,
		metrics:               metrics,
		config:                config,
		genesis:               genesis,
		options:               options,
		lastState:             &atomic.Pointer[types.State]{},
		daClient:              daClient,
		daRetrieverHeight:     daRetrieverHeight,
		headerStore:           headerStore,
		dataStore:             dataStore,
		heightInCh:            make(chan common.DAHeightEvent, 100),
		errorCh:               errorCh,
		logger:                logger.With().Str("component", "syncer").Logger(),
		gracePeriodMultiplier: gracePeriodMultiplier,
		blockFullnessEMA:      blockFullnessEMA,
		gracePeriodConfig:     newForcedInclusionGracePeriodConfig(),
	}
}

// Start begins the syncing component
func (s *Syncer) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)

	if err := s.initializeState(); err != nil {
		return fmt.Errorf("failed to initialize syncer state: %w", err)
	}

	// Initialize handlers
	if s.daRetriever == nil {
		s.daRetriever = NewDARetriever(s.daClient, s.cache, s.genesis, s.logger)
	}
	s.fiRetriever = da.NewForcedInclusionRetriever(s.daClient, s.logger, s.genesis.DAStartHeight, s.genesis.DAEpochForcedInclusion)
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
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.processLoop()
	}()

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

// getLastState returns the current state
func (s *Syncer) getLastState() types.State {
	state := s.lastState.Load()
	if state == nil {
		return types.State{}
	}

	return *state
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
	if state.DAHeight != 0 && state.DAHeight < s.genesis.DAStartHeight {
		return fmt.Errorf("DA height (%d) is lower than DA start height (%d)", state.DAHeight, s.genesis.DAStartHeight)
	}
	s.SetLastState(state)

	// Set DA height to the maximum of the genesis start height, the state's DA height, the cached DA height, and the highest stored included DA height.
	// This ensures we resume from the highest known DA height, even if the cache is cleared on restart. If the DA height is too high because of a user error, reset it with --evnode.clear_cache. The DA height will be back to the last highest known executed DA height for a height.
	s.daRetrieverHeight.Store(max(s.genesis.DAStartHeight, s.cache.DaHeight(), state.DAHeight, s.getHighestStoredDAHeight()))

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
		// 1. Catch up mode: fetch sequentially until we are up to date
		if err := s.fetchDAUntilCaughtUp(); err != nil {
			s.logger.Error().Err(err).Msg("DA catchup failed, retrying after backoff")

			backoff := s.config.DA.BlockTime.Duration
			if backoff <= 0 {
				backoff = 2 * time.Second
			}

			select {
			case <-s.ctx.Done():
				return
			case <-time.After(backoff):
			}
			continue
		}

		// 2. Follow mode: use subscription to receive new blocks
		// If subscription fails or gap is detected, we fall back to catchup mode
		if err := s.followDA(); err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			s.logger.Warn().Err(err).Msg("DA follow disrupted, switching to catchup")
			// We don't need explicit backoff here as we'll switch to catchup immediately,
			// checking if there are new blocks to fetch.
		}
	}
}

// followDA subscribes to DA events and processes them until a gap is detected or error occurs
func (s *Syncer) followDA() error {
	s.logger.Info().Msg("entering DA follow mode")
	subCh, err := s.daRetriever.Subscribe(s.ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe to DA: %w", err)
	}

	for {
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		case event, ok := <-subCh:
			if !ok {
				return errors.New("DA subscription channel closed")
			}

			// Calculate expected height
			nextExpectedHeight := max(s.daRetrieverHeight.Load(), s.cache.DaHeight())

			// If we receive an event for a future height (gap), break to trigger catchup
			if event.DaHeight > nextExpectedHeight {
				s.logger.Info().
					Uint64("event_da_height", event.DaHeight).
					Uint64("expected_da_height", nextExpectedHeight).
					Msg("gap detected in DA stream, switching to catchup")
				return nil
			}

			// If event is old/duplicate, log and ignore (or just update height if needed)
			if event.DaHeight < nextExpectedHeight {
				s.logger.Debug().
					Uint64("event_da_height", event.DaHeight).
					Uint64("expected_da_height", nextExpectedHeight).
					Msg("received old DA event, ignoring")
				continue
			}

			// Process event (same conceptual logic as catchup)
			select {
			case s.heightInCh <- event:
			default:
				// If channel is full, use cache as backup/buffer
				s.cache.SetPendingEvent(event.Header.Height(), &event)
			}

			// Update expected height
			s.daRetrieverHeight.Store(event.DaHeight + 1)
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
			case errors.Is(err, datypes.ErrBlobNotFound):
				s.daRetrieverHeight.Store(daHeight + 1)
				continue // Fetch next height immediately
			case errors.Is(err, datypes.ErrHeightFromFuture):
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
		case errors.Is(err, errMaliciousProposer):
			s.sendCriticalError(fmt.Errorf("sequencer malicious. Restart the node with --node.aggregator --node.based_sequencer or keep the chain halted: %w", err))
		case errors.Is(err, errInvalidState):
			s.sendCriticalError(fmt.Errorf("invalid state detected (block-height %d, state-height %d) "+
				"- block references do not match local state. Manual intervention required: %w", event.Header.Height(),
				s.getLastState().LastBlockHeight, err))
		default:
			s.cache.SetPendingEvent(height, event)
		}
		return
	}

	// only save to p2p stores if the event came from DA
	if event.Source == common.SourceDA { // TODO(@julienrbrt): To be reverted once DA Hints are merged (https://github.com/evstack/ev-node/pull/2891)
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
	currentState := s.getLastState()
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

	// Verify forced inclusion transactions if configured
	if event.Source == common.SourceDA {
		if err := s.verifyForcedInclusionTxs(currentState, data); err != nil {
			s.logger.Error().Err(err).Uint64("height", nextHeight).Msg("forced inclusion verification failed")
			if errors.Is(err, errMaliciousProposer) {
				s.cache.RemoveHeaderDAIncluded(headerHash)
				return err
			}
		}
	}

	// Apply block
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

// executeTxsWithRetry executes transactions with retry logic.
// NOTE: the function retries the execution client call regardless of the error. Some execution clients errors are irrecoverable, and will eventually halt the node, as expected.
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

var errMaliciousProposer = errors.New("malicious proposer detected")

// hashTx returns a hex-encoded SHA256 hash of the transaction.
func hashTx(tx []byte) string {
	hash := sha256.Sum256(tx)
	return hex.EncodeToString(hash[:])
}

// calculateBlockFullness returns a value between 0.0 and 1.0 indicating how full the block is.
// It estimates fullness based on total data size.
// This is a heuristic - actual limits may vary by execution layer.
func (s *Syncer) calculateBlockFullness(data *types.Data) float64 {
	const maxDataSize = common.DefaultMaxBlobSize

	var fullness float64
	count := 0

	// Check data size fullness
	dataSize := uint64(0)
	for _, tx := range data.Txs {
		dataSize += uint64(len(tx))
	}
	sizeFullness := float64(dataSize) / float64(maxDataSize)
	fullness += min(sizeFullness, 1.0)
	count++

	// Return average fullness
	return fullness / float64(count)
}

// updateDynamicGracePeriod updates the grace period multiplier based on block fullness.
// When blocks are consistently full, the multiplier increases (more lenient).
// When blocks have capacity, the multiplier decreases (stricter).
func (s *Syncer) updateDynamicGracePeriod(blockFullness float64) {
	// Update exponential moving average of block fullness
	currentEMA := *s.blockFullnessEMA.Load()
	alpha := s.gracePeriodConfig.dynamicAdjustmentRate
	newEMA := alpha*blockFullness + (1-alpha)*currentEMA
	s.blockFullnessEMA.Store(&newEMA)

	// Adjust grace period multiplier based on EMA
	currentMultiplier := *s.gracePeriodMultiplier.Load()
	threshold := s.gracePeriodConfig.dynamicFullnessThreshold

	var newMultiplier float64
	if newEMA > threshold {
		// Blocks are full - increase grace period (more lenient)
		adjustment := alpha * (newEMA - threshold) / (1.0 - threshold)
		newMultiplier = currentMultiplier + adjustment
	} else {
		// Blocks have capacity - decrease grace period (stricter)
		adjustment := alpha * (threshold - newEMA) / threshold
		newMultiplier = currentMultiplier - adjustment
	}

	// Clamp to min/max bounds
	newMultiplier = max(newMultiplier, s.gracePeriodConfig.dynamicMinMultiplier)
	newMultiplier = min(newMultiplier, s.gracePeriodConfig.dynamicMaxMultiplier)

	s.gracePeriodMultiplier.Store(&newMultiplier)

	// Log significant changes (more than 10% change)
	if math.Abs(newMultiplier-currentMultiplier) > 0.1 {
		s.logger.Debug().
			Float64("block_fullness", blockFullness).
			Float64("fullness_ema", newEMA).
			Float64("old_multiplier", currentMultiplier).
			Float64("new_multiplier", newMultiplier).
			Msg("dynamic grace period multiplier adjusted")
	}
}

// getEffectiveGracePeriod returns the current effective grace period considering dynamic adjustment.
func (s *Syncer) getEffectiveGracePeriod() uint64 {
	multiplier := *s.gracePeriodMultiplier.Load()
	effectivePeriod := math.Round(float64(s.gracePeriodConfig.basePeriod) * multiplier)
	minPeriod := float64(s.gracePeriodConfig.basePeriod) * s.gracePeriodConfig.dynamicMinMultiplier

	return uint64(max(effectivePeriod, minPeriod))
}

// verifyForcedInclusionTxs verifies that forced inclusion transactions from DA are properly handled.
// Note: Due to block size constraints (MaxBytes), sequencers may defer forced inclusion transactions
// to future blocks (smoothing). This is legitimate behavior within an epoch.
// However, ALL forced inclusion txs from an epoch MUST be included before the next epoch begins or grace boundary (whichever comes later).
func (s *Syncer) verifyForcedInclusionTxs(currentState types.State, data *types.Data) error {
	if s.fiRetriever == nil {
		return nil
	}

	// Update dynamic grace period based on block fullness
	blockFullness := s.calculateBlockFullness(data)
	s.updateDynamicGracePeriod(blockFullness)

	// Retrieve forced inclusion transactions from DA for current epoch
	forcedIncludedTxsEvent, err := s.fiRetriever.RetrieveForcedIncludedTxs(s.ctx, currentState.DAHeight)
	if err != nil {
		if errors.Is(err, da.ErrForceInclusionNotConfigured) {
			s.logger.Debug().Msg("forced inclusion namespace not configured, skipping verification")
			return nil
		}

		return fmt.Errorf("failed to retrieve forced included txs from DA: %w", err)
	}

	// Build map of transactions in current block
	blockTxMap := make(map[string]struct{})
	for _, tx := range data.Txs {
		blockTxMap[hashTx(tx)] = struct{}{}
	}

	// Check if any pending forced inclusion txs from previous epochs are included
	var stillPending []pendingForcedInclusionTx
	s.pendingForcedInclusionTxs.Range(func(key, value any) bool {
		pending := value.(pendingForcedInclusionTx)
		if _, ok := blockTxMap[pending.TxHash]; ok {
			s.logger.Debug().
				Uint64("height", data.Height()).
				Uint64("epoch_start", pending.EpochStart).
				Uint64("epoch_end", pending.EpochEnd).
				Str("tx_hash", pending.TxHash[:16]).
				Msg("pending forced inclusion transaction included in block")
			s.pendingForcedInclusionTxs.Delete(key)
		} else {
			stillPending = append(stillPending, pending)
		}
		return true
	})

	// Add new forced inclusion transactions from current epoch
	var newPendingCount, includedCount int
	for _, forcedTx := range forcedIncludedTxsEvent.Txs {
		txHash := hashTx(forcedTx)
		if _, ok := blockTxMap[txHash]; ok {
			// Transaction is included in this block
			includedCount++
		} else {
			// Transaction not included, add to pending
			stillPending = append(stillPending, pendingForcedInclusionTx{
				Data:       forcedTx,
				EpochStart: forcedIncludedTxsEvent.StartDaHeight,
				EpochEnd:   forcedIncludedTxsEvent.EndDaHeight,
				TxHash:     txHash,
			})
			newPendingCount++
		}
	}

	// Check if we've moved past any epoch boundaries with pending txs
	// Grace period: Allow forced inclusion txs from epoch N to be included in epoch N+1, N+2, etc.
	// Only flag as malicious if past grace boundary to prevent false positives during chain congestion.
	var maliciousTxs, remainingPending []pendingForcedInclusionTx
	var txsInGracePeriod int
	for _, pending := range stillPending {
		// Calculate grace boundary: epoch end + (effective grace periods × epoch size)
		effectiveGracePeriod := s.getEffectiveGracePeriod()
		graceBoundary := pending.EpochEnd + (effectiveGracePeriod * s.genesis.DAEpochForcedInclusion)

		if currentState.DAHeight > graceBoundary {
			maliciousTxs = append(maliciousTxs, pending)
			s.logger.Warn().
				Uint64("current_da_height", currentState.DAHeight).
				Uint64("epoch_end", pending.EpochEnd).
				Uint64("grace_boundary", graceBoundary).
				Uint64("base_grace_periods", s.gracePeriodConfig.basePeriod).
				Uint64("effective_grace_periods", effectiveGracePeriod).
				Float64("grace_multiplier", *s.gracePeriodMultiplier.Load()).
				Str("tx_hash", pending.TxHash[:16]).
				Msg("forced inclusion transaction past grace boundary - marking as malicious")
		} else {
			remainingPending = append(remainingPending, pending)
			if currentState.DAHeight > pending.EpochEnd {
				txsInGracePeriod++
			}
		}
	}

	s.metrics.ForcedInclusionTxsInGracePeriod.Set(float64(txsInGracePeriod))

	// Update pending map - clear old entries and store only remaining pending
	s.pendingForcedInclusionTxs.Range(func(key, value any) bool {
		s.pendingForcedInclusionTxs.Delete(key)
		return true
	})
	for _, pending := range remainingPending {
		s.pendingForcedInclusionTxs.Store(pending.TxHash, pending)
	}

	// If there are transactions past grace boundary that weren't included, sequencer is malicious
	if len(maliciousTxs) > 0 {
		s.metrics.ForcedInclusionTxsMalicious.Add(float64(len(maliciousTxs)))

		effectiveGracePeriod := s.getEffectiveGracePeriod()
		s.logger.Error().
			Uint64("height", data.Height()).
			Uint64("current_da_height", currentState.DAHeight).
			Int("malicious_count", len(maliciousTxs)).
			Uint64("base_grace_periods", s.gracePeriodConfig.basePeriod).
			Uint64("effective_grace_periods", effectiveGracePeriod).
			Float64("grace_multiplier", *s.gracePeriodMultiplier.Load()).
			Msg("SEQUENCER IS MALICIOUS: forced inclusion transactions past grace boundary not included")
		return errors.Join(errMaliciousProposer, fmt.Errorf("sequencer is malicious: %d forced inclusion transactions past grace boundary (base_grace_periods=%d, effective_grace_periods=%d) not included", len(maliciousTxs), s.gracePeriodConfig.basePeriod, effectiveGracePeriod))
	}

	// Log current state
	if len(forcedIncludedTxsEvent.Txs) > 0 {
		if newPendingCount > 0 {
			totalPending := 0
			s.pendingForcedInclusionTxs.Range(func(key, value any) bool {
				totalPending++
				return true
			})

			s.logger.Info().
				Uint64("height", data.Height()).
				Uint64("da_height", currentState.DAHeight).
				Uint64("epoch_start", forcedIncludedTxsEvent.StartDaHeight).
				Uint64("epoch_end", forcedIncludedTxsEvent.EndDaHeight).
				Int("included_count", includedCount).
				Int("deferred_count", newPendingCount).
				Int("total_pending", totalPending).
				Msg("forced inclusion transactions processed - some deferred due to block size constraints")
		} else {
			s.logger.Debug().
				Uint64("height", data.Height()).
				Int("forced_txs", len(forcedIncludedTxsEvent.Txs)).
				Msg("all forced inclusion transactions included in block")
		}
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

// getHighestStoredDAHeight retrieves the highest DA height from the store by checking
// the DA heights stored for the last DA included height
// this relies on the node syncing with DA and setting included heights.
func (s *Syncer) getHighestStoredDAHeight() uint64 {
	// Get the DA included height from store
	daIncludedHeightBytes, err := s.store.GetMetadata(s.ctx, store.DAIncludedHeightKey)
	if err != nil || len(daIncludedHeightBytes) != 8 {
		return 0
	}
	daIncludedHeight := binary.LittleEndian.Uint64(daIncludedHeightBytes)
	if daIncludedHeight == 0 {
		return 0
	}

	var highestDAHeight uint64

	// Get header DA height for the last included height
	headerKey := store.GetHeightToDAHeightHeaderKey(daIncludedHeight)
	if headerBytes, err := s.store.GetMetadata(s.ctx, headerKey); err == nil && len(headerBytes) == 8 {
		headerDAHeight := binary.LittleEndian.Uint64(headerBytes)
		highestDAHeight = max(highestDAHeight, headerDAHeight)
	}

	// Get data DA height for the last included height
	dataKey := store.GetHeightToDAHeightDataKey(daIncludedHeight)
	if dataBytes, err := s.store.GetMetadata(s.ctx, dataKey); err == nil && len(dataBytes) == 8 {
		dataDAHeight := binary.LittleEndian.Uint64(dataBytes)
		highestDAHeight = max(highestDAHeight, dataDAHeight)
	}

	return highestDAHeight
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
