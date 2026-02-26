package syncing

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/celestiaorg/go-header"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/block/internal/da"
	coreexecutor "github.com/evstack/ev-node/core/execution"
	"github.com/evstack/ev-node/pkg/config"
	datypes "github.com/evstack/ev-node/pkg/da/types"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/raft"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

var _ BlockSyncer = (*Syncer)(nil)

const (
	// baseGracePeriodEpochs is the minimum grace window after an epoch ends.
	// A tx from epoch N must appear by the end of epoch N+1 under normal conditions.
	baseGracePeriodEpochs uint64 = 1

	// maxGracePeriodEpochs caps the grace window even under sustained congestion.
	maxGracePeriodEpochs uint64 = 4

	// fullnessThreshold is the fraction of DefaultMaxBlobSize above which a block
	// is considered full. Exceeding it extends the grace period for that epoch.
	fullnessThreshold = 0.8
)

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
	headerStore header.Store[*types.P2PSignedHeader]
	dataStore   header.Store[*types.P2PData]

	// Channels for coordination
	heightInCh chan common.DAHeightEvent
	errorCh    chan<- error // Channel to report critical execution client failures

	// Handlers
	daRetriever   DARetriever
	fiRetriever   da.ForcedInclusionRetriever
	p2pHandler    p2pHandler
	raftRetriever *raftRetriever

	// Forced inclusion tracking
	forcedInclusionMu    sync.RWMutex
	seenBlockTxs         map[string]struct{} // SHA-256 hex of every tx seen in a DA-sourced block
	seenBlockTxsByHeight map[uint64][]string // DA height → hashes at that height (for pruning)
	daBlockBytes         map[uint64]uint64   // DA height → total tx bytes (for congestion tracking)
	lastCheckedEpochEnd  uint64              // highest epochEnd fully verified so far

	// Lifecycle
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	hasCriticalError atomic.Bool

	// P2P wait coordination
	p2pWaitState atomic.Value // stores p2pWaitState

	// DA head-reached signal for recovery mode (stays true once DA head is seen)
	daHeadReached atomic.Bool

	// blockSyncer is the interface used for block sync operations.
	// defaults to self, but can be wrapped with tracing.
	blockSyncer BlockSyncer
}

// NewSyncer creates a new block syncer
func NewSyncer(
	store store.Store,
	exec coreexecutor.Executor,
	daClient da.Client,
	cache cache.Manager,
	metrics *common.Metrics,
	config config.Config,
	genesis genesis.Genesis,
	headerStore header.Store[*types.P2PSignedHeader],
	dataStore header.Store[*types.P2PData],
	logger zerolog.Logger,
	options common.BlockOptions,
	errorCh chan<- error,
	raftNode common.RaftNode,
) *Syncer {
	daRetrieverHeight := &atomic.Uint64{}
	daRetrieverHeight.Store(genesis.DAStartHeight)

	s := &Syncer{
		store:                store,
		exec:                 exec,
		cache:                cache,
		metrics:              metrics,
		config:               config,
		genesis:              genesis,
		options:              options,
		lastState:            &atomic.Pointer[types.State]{},
		daClient:             daClient,
		daRetrieverHeight:    daRetrieverHeight,
		headerStore:          headerStore,
		dataStore:            dataStore,
		heightInCh:           make(chan common.DAHeightEvent, 100),
		errorCh:              errorCh,
		logger:               logger.With().Str("component", "syncer").Logger(),
		seenBlockTxs:         make(map[string]struct{}),
		seenBlockTxsByHeight: make(map[uint64][]string),
		daBlockBytes:         make(map[uint64]uint64),
	}
	s.blockSyncer = s
	if raftNode != nil && !reflect.ValueOf(raftNode).IsNil() {
		s.raftRetriever = newRaftRetriever(raftNode, genesis, logger, eventProcessorFn(s.pipeEvent),
			func(ctx context.Context, state *raft.RaftBlockState) error {
				s.logger.Debug().Uint64("header_height", state.LastSubmittedDaHeaderHeight).Uint64("data_height", state.LastSubmittedDaDataHeight).Msg("received raft block state")
				cache.SetLastSubmittedHeaderHeight(ctx, state.LastSubmittedDaHeaderHeight)
				cache.SetLastSubmittedDataHeight(ctx, state.LastSubmittedDaDataHeight)
				return nil
			})
	}
	return s
}

// SetBlockSyncer sets the block syncer interface, allowing injection of
// a tracing wrapper or other decorator.
func (s *Syncer) SetBlockSyncer(bs BlockSyncer) {
	s.blockSyncer = bs
}

// Start begins the syncing component
func (s *Syncer) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)

	if err := s.initializeState(); err != nil {
		return fmt.Errorf("failed to initialize syncer state: %w", err)
	}

	// Initialize handlers
	s.daRetriever = NewDARetriever(s.daClient, s.cache, s.genesis, s.logger)
	if s.config.Instrumentation.IsTracingEnabled() {
		s.daRetriever = WithTracingDARetriever(s.daRetriever)
	}

	s.fiRetriever = da.NewForcedInclusionRetriever(s.daClient, s.logger, s.config, s.genesis.DAStartHeight, s.genesis.DAEpochForcedInclusion)
	s.p2pHandler = NewP2PHandler(s.headerStore, s.dataStore, s.cache, s.genesis, s.logger)
	if currentHeight, err := s.store.Height(s.ctx); err != nil {
		s.logger.Error().Err(err).Msg("failed to set initial processed height for p2p handler")
	} else {
		s.p2pHandler.SetProcessedHeight(currentHeight)
	}

	if s.raftRetriever != nil {
		if err := s.raftRetriever.Start(s.ctx); err != nil {
			return fmt.Errorf("start raft retriever: %w", err)
		}
	}

	if !s.waitForGenesis() {
		return nil
	}

	// Start main processing loop
	s.wg.Go(s.processLoop)

	// Start dedicated workers for DA, and pending processing
	s.startSyncWorkers()

	s.logger.Info().Msg("syncer started")
	return nil
}

// Stop shuts down the syncing component
func (s *Syncer) Stop() error {
	if s.cancel == nil {
		return nil
	}

	s.cancel()
	s.cancelP2PWait(0)
	s.wg.Wait()

	// Skip draining if we're shutting down due to a critical error (e.g. execution
	// client unavailable).
	if !s.hasCriticalError.Load() {
		drainCtx, drainCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer drainCancel()

		drained := 0
	drainLoop:
		for {
			select {
			case event, ok := <-s.heightInCh:
				if !ok {
					break drainLoop
				}
				s.processHeightEvent(drainCtx, &event)
				drained++
			case <-drainCtx.Done():
				s.logger.Warn().Int("remaining", len(s.heightInCh)).Msg("timeout draining height events during shutdown")
				break drainLoop
			default:
				break drainLoop
			}
		}
		if drained > 0 {
			s.logger.Info().Int("count", drained).Msg("drained pending height events during shutdown")
		}
	}

	s.logger.Info().Msg("syncer stopped")
	close(s.heightInCh)
	s.cancel = nil
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
		stateRoot, initErr := s.exec.InitChain(
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

	// Persist the initialized state to the store
	batch, err := s.store.NewBatch(s.ctx)
	if err != nil {
		return fmt.Errorf("failed to create batch: %w", err)
	}
	if err := batch.SetHeight(state.LastBlockHeight); err != nil {
		return fmt.Errorf("failed to set store height: %w", err)
	}
	if err := batch.UpdateState(state); err != nil {
		return fmt.Errorf("failed to update state: %w", err)
	}
	if err := batch.Commit(); err != nil {
		return fmt.Errorf("failed to commit batch: %w", err)
	}
	s.SetLastState(state)

	// Set DA height to the maximum of the genesis start height, the state's DA height, and the cached DA height.
	// The cache's DaHeight() is initialized from store metadata, so it's always correct even after cache clear.
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
	s.logger.Info().Msg("starting process loop")
	defer s.logger.Info().Msg("process loop stopped")

	for {
		select {
		case <-s.ctx.Done():
			return
		case heightEvent, ok := <-s.heightInCh:
			if ok {
				s.processHeightEvent(s.ctx, &heightEvent)
			}
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
			s.daHeadReached.Store(true)
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

// HasReachedDAHead returns true once the DA worker has reached the DA head.
// Once set, it stays true.
func (s *Syncer) HasReachedDAHead() bool {
	return s.daHeadReached.Load()
}

func (s *Syncer) fetchDAUntilCaughtUp() error {
	for {
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		default:
		}

		// Check for priority heights from P2P hints first
		var daHeight uint64
		if priorityHeight := s.daRetriever.PopPriorityHeight(); priorityHeight > 0 {
			// Skip if we've already fetched past this height
			currentHeight := s.daRetrieverHeight.Load()
			if priorityHeight < currentHeight {
				continue
			}
			daHeight = priorityHeight
			s.logger.Debug().Uint64("da_height", daHeight).Msg("fetching priority DA height from P2P hint")
		} else {
			daHeight = max(s.daRetrieverHeight.Load(), s.cache.DaHeight())
		}

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
			if err := s.pipeEvent(s.ctx, event); err != nil {
				return err
			}
		}

		// Update DA retrieval height on successful retrieval
		// For priority fetches, only update if the priority height is ahead of current
		// For sequential fetches, always increment
		newHeight := daHeight + 1
		for {
			current := s.daRetrieverHeight.Load()
			if newHeight <= current {
				break // Already at or past this height
			}
			if s.daRetrieverHeight.CompareAndSwap(current, newHeight) {
				break
			}
		}
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

func (s *Syncer) pipeEvent(ctx context.Context, event common.DAHeightEvent) error {
	select {
	case s.heightInCh <- event:
		return nil
	case <-ctx.Done():
		s.cache.SetPendingEvent(event.Header.Height(), &event)
		return ctx.Err()
	default:
		s.cache.SetPendingEvent(event.Header.Height(), &event)
	}
	return nil
}

func (s *Syncer) processHeightEvent(ctx context.Context, event *common.DAHeightEvent) {
	height := event.Header.Height()
	headerHash := event.Header.Hash().String()

	s.logger.Debug().
		Uint64("height", height).
		Uint64("da_height", event.DaHeight).
		Str("hash", headerHash).
		Msg("processing height event")

	currentHeight, err := s.store.Height(ctx)
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

	// If this is a P2P event with a DA height hint, trigger targeted DA retrieval
	// This allows us to fetch the block directly from the specified DA height instead of sequential scanning
	if event.Source == common.SourceP2P {
		var daHeightHints []uint64
		switch {
		case event.DaHeightHints == [2]uint64{0, 0}:
		// empty, nothing to do
		case event.DaHeightHints[0] == 0:
			// check only data
			if _, exists := s.cache.GetDataDAIncluded(event.Data.Hash().String()); !exists {
				daHeightHints = []uint64{event.DaHeightHints[1]}
			}
		case event.DaHeightHints[1] == 0:
			// check only header
			if _, exists := s.cache.GetHeaderDAIncluded(event.Header.Hash().String()); !exists {
				daHeightHints = []uint64{event.DaHeightHints[0]}
			}
		default:
			// check both
			if _, exists := s.cache.GetHeaderDAIncluded(event.Header.Hash().String()); !exists {
				daHeightHints = []uint64{event.DaHeightHints[0]}
			}
			if _, exists := s.cache.GetDataDAIncluded(event.Data.Hash().String()); !exists {
				daHeightHints = append(daHeightHints, event.DaHeightHints[1])
			}
			if len(daHeightHints) == 2 && daHeightHints[0] == daHeightHints[1] {
				daHeightHints = daHeightHints[0:1]
			}
		}
		if len(daHeightHints) > 0 {
			for _, daHeightHint := range daHeightHints {
				// Skip if we've already fetched past this height
				if daHeightHint < s.daRetrieverHeight.Load() {
					continue
				}

				s.logger.Debug().
					Uint64("height", height).
					Uint64("da_height_hint", daHeightHint).
					Msg("P2P event with DA height hint, queuing priority DA retrieval")

				// Queue priority DA retrieval - will be processed in fetchDAUntilCaughtUp
				s.daRetriever.QueuePriorityHeight(daHeightHint)
			}
		}
	}

	// Last data must be got from store if the event comes from DA and the data hash is empty.
	// When if the event comes from P2P, the sequencer and then all the full nodes contains the data.
	if event.Source == common.SourceDA && bytes.Equal(event.Header.DataHash, common.DataHashForEmptyTxs) && currentHeight > 0 {
		_, lastData, err := s.store.GetBlockData(ctx, currentHeight)
		if err != nil {
			s.logger.Error().Err(err).Msg("failed to get last data")
			return
		}
		event.Data.LastDataHash = lastData.Hash()
	}

	// Cancel any P2P wait that might still be blocked on this height, as we have a block for it.
	s.cancelP2PWait(height)

	// Try to sync the next block
	if err := s.blockSyncer.TrySyncNextBlock(ctx, event); err != nil {
		s.logger.Error().Err(err).
			Uint64("event-height", event.Header.Height()).
			Uint64("state-height", s.getLastState().LastBlockHeight).
			Msg("failed to sync next block")
		// If the error is not due to a validation error, re-store the event as pending
		switch {
		case errors.Is(err, errInvalidBlock) || s.hasCriticalError.Load():
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
}

var (
	// errInvalidBlock is returned when a block is failing validation
	errInvalidBlock = errors.New("invalid block")
	// errInvalidState is returned when the state has diverged from the DA blocks
	errInvalidState = errors.New("invalid state")
)

// TrySyncNextBlock attempts to sync the next available block
// the event is always the next block in sequence as processHeightEvent ensures it.
func (s *Syncer) TrySyncNextBlock(ctx context.Context, event *common.DAHeightEvent) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
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
	if err := s.ValidateBlock(ctx, currentState, data, header); err != nil {
		// remove header as da included from cache
		s.cache.RemoveHeaderDAIncluded(headerHash)
		s.cache.RemoveDataDAIncluded(data.DACommitment().String())

		if !errors.Is(err, errInvalidState) && !errors.Is(err, errInvalidBlock) {
			return errors.Join(errInvalidBlock, err)
		}
		return err
	}

	// Verify forced inclusion transactions if configured.
	// The check is actually only performed on DA-sourced blocks.
	// P2P nodes aren't actually able to verify forced inclusion txs as DA inclusion happens later
	// (so DA hints are not available) and DA hints cannot be trusted. This is a known limitation
	// described in the ADR.
	if event.Source == common.SourceDA {
		if err := s.VerifyForcedInclusionTxs(ctx, event.DaHeight, data); err != nil {
			s.logger.Error().Err(err).Uint64("height", nextHeight).Msg("forced inclusion verification failed")
			if errors.Is(err, errMaliciousProposer) {
				// remove header as da included from cache
				s.cache.RemoveHeaderDAIncluded(headerHash)
				s.cache.RemoveDataDAIncluded(data.DACommitment().String())

				return err
			}
		}
	}

	// Apply block
	newState, err := s.ApplyBlock(ctx, header.Header, data, currentState)
	if err != nil {
		return fmt.Errorf("failed to apply block: %w", err)
	}

	// Update DA height if needed.
	// state.DAHeight is used for state persistence and restart recovery.
	if event.DaHeight > newState.DAHeight {
		newState.DAHeight = event.DaHeight
	}

	batch, err := s.store.NewBatch(ctx)
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

// ApplyBlock applies a block to get the new state
func (s *Syncer) ApplyBlock(ctx context.Context, header types.Header, data *types.Data, currentState types.State) (types.State, error) {
	// Prepare transactions
	rawTxs := make([][]byte, len(data.Txs))
	for i, tx := range data.Txs {
		rawTxs[i] = []byte(tx)
	}

	// Execute transactions
	ctx = context.WithValue(ctx, types.HeaderContextKey, header)
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
		newAppHash, err := s.exec.ExecuteTxs(ctx, rawTxs, header.Height(), header.Time(), currentState.AppHash)
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
			case <-ctx.Done():
				return nil, fmt.Errorf("context cancelled during retry: %w", ctx.Err())
			}
		}

		return newAppHash, nil
	}

	return nil, nil
}

// ValidateBlock validates a synced block
// NOTE: if the header was gibberish and somehow passed all validation prior but the data was correct
// or if the data was gibberish and somehow passed all validation prior but the header was correct
// we are still losing both in the pending event. This should never happen.
func (s *Syncer) ValidateBlock(_ context.Context, currState types.State, data *types.Data, header *types.SignedHeader) error {
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

// gracePeriodForEpoch returns the grace window for an epoch based on average
// block fullness. For each fullnessThreshold-sized band above the threshold one
// extra epoch is granted, up to maxGracePeriodEpochs.
func (s *Syncer) gracePeriodForEpoch(epochStart, epochEnd uint64) uint64 {
	if epochEnd < epochStart {
		return baseGracePeriodEpochs
	}

	// Empty DA heights contribute 0 bytes and still count toward the average,
	// so spare capacity reduces the grace extension.
	heightCount := epochEnd - epochStart + 1

	s.forcedInclusionMu.RLock()
	var totalBytes uint64
	for h := epochStart; h <= epochEnd; h++ {
		totalBytes += s.daBlockBytes[h]
	}
	s.forcedInclusionMu.RUnlock()

	avgBytes := totalBytes / heightCount
	threshold := uint64(math.Round(fullnessThreshold * float64(common.DefaultMaxBlobSize)))

	var extra uint64
	if avgBytes > threshold {
		extra = (avgBytes - threshold) / threshold
	}

	grace := baseGracePeriodEpochs + extra
	if grace > maxGracePeriodEpochs {
		grace = maxGracePeriodEpochs
	}
	return grace
}

// VerifyForcedInclusionTxs checks that every forced-inclusion tx submitted
// during epochs whose grace window has elapsed appears in seenBlockTxs.
// Txs may be spread across multiple blocks; what matters is that each one
// landed somewhere before its epoch's grace deadline.
func (s *Syncer) VerifyForcedInclusionTxs(ctx context.Context, daHeight uint64, data *types.Data) error {
	if s.fiRetriever == nil || s.genesis.DAEpochForcedInclusion == 0 {
		return nil
	}

	epochSize := s.genesis.DAEpochForcedInclusion
	daStart := s.genesis.DAStartHeight

	// Record txs and byte count for this DA height.
	var blockBytes uint64
	for _, tx := range data.Txs {
		blockBytes += uint64(len(tx))
	}
	s.forcedInclusionMu.Lock()
	hashes := make([]string, 0, len(data.Txs))
	for _, tx := range data.Txs {
		h := hashTx(tx)
		s.seenBlockTxs[h] = struct{}{}
		hashes = append(hashes, h)
	}
	s.seenBlockTxsByHeight[daHeight] = hashes
	s.daBlockBytes[daHeight] = blockBytes
	s.forcedInclusionMu.Unlock()

	if daHeight < daStart {
		return nil
	}

	executionInfo, err := s.exec.GetExecutionInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get execution info: %w", err)
	}

	var maliciousCount int

	for epochEnd := daStart + epochSize - 1; ; epochEnd += epochSize {
		epochStart := epochEnd - (epochSize - 1)
		gracePeriod := s.gracePeriodForEpoch(epochStart, epochEnd)
		graceBoundary := epochEnd + gracePeriod*epochSize

		if graceBoundary >= daHeight {
			break
		}

		event, retrieveErr := s.fiRetriever.RetrieveForcedIncludedTxs(ctx, epochEnd)
		if retrieveErr != nil {
			if errors.Is(retrieveErr, da.ErrForceInclusionNotConfigured) {
				return nil
			}
			return fmt.Errorf("failed to retrieve forced inclusion txs for epoch ending at %d: %w", epochEnd, retrieveErr)
		}

		if len(event.Txs) == 0 {
			if epochEnd > s.lastCheckedEpochEnd {
				s.pruneUpTo(epochEnd)
			}
			continue
		}

		// Skip intrinsically invalid txs so the sequencer isn't blamed for dropping them.
		filterStatuses, filterErr := s.exec.FilterTxs(ctx, event.Txs, common.DefaultMaxBlobSize, executionInfo.MaxGas, true)
		if filterErr != nil {
			return fmt.Errorf("failed to filter forced inclusion txs: %w", filterErr)
		}

		for i, tx := range event.Txs {
			if filterStatuses[i] != coreexecutor.FilterOK {
				continue
			}
			txHash := hashTx(tx)
			s.forcedInclusionMu.RLock()
			_, seen := s.seenBlockTxs[txHash]
			s.forcedInclusionMu.RUnlock()
			if !seen {
				maliciousCount++
				s.logger.Warn().
					Uint64("current_da_height", daHeight).
					Uint64("epoch_end", epochEnd).
					Uint64("grace_boundary", graceBoundary).
					Str("tx_hash", txHash[:16]).
					Msg("forced inclusion transaction past grace boundary not included - marking as malicious")
			}
		}

		if epochEnd > s.lastCheckedEpochEnd {
			s.pruneUpTo(epochEnd)
		}
	}

	if maliciousCount > 0 {
		s.metrics.ForcedInclusionTxsMalicious.Add(float64(maliciousCount))
		s.logger.Error().
			Uint64("current_da_height", daHeight).
			Int("malicious_count", maliciousCount).
			Uint64("base_grace_period_epochs", baseGracePeriodEpochs).
			Uint64("max_grace_period_epochs", maxGracePeriodEpochs).
			Msg("SEQUENCER IS MALICIOUS: forced inclusion transactions past grace boundary not included")
		return errors.Join(errMaliciousProposer,
			fmt.Errorf("sequencer is malicious: %d forced inclusion transaction(s) past grace boundary not included",
				maliciousCount))
	}

	return nil
}

// pruneUpTo deletes seenBlockTxs, seenBlockTxsByHeight, and daBlockBytes entries
// for all DA heights ≤ upTo and advances lastCheckedEpochEnd. Safe to call once
// an epoch is fully checked: no future epoch check can reference those heights.
func (s *Syncer) pruneUpTo(upTo uint64) {
	s.forcedInclusionMu.Lock()
	defer s.forcedInclusionMu.Unlock()

	for h := s.lastCheckedEpochEnd; h <= upTo; h++ {
		for _, txHash := range s.seenBlockTxsByHeight[h] {
			delete(s.seenBlockTxs, txHash)
		}
		delete(s.seenBlockTxsByHeight, h)
		delete(s.daBlockBytes, h)
	}
	s.lastCheckedEpochEnd = upTo
}

// sendCriticalError sends a critical error to the error channel without blocking
func (s *Syncer) sendCriticalError(err error) {
	s.hasCriticalError.Store(true)
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

// IsSyncedWithRaft checks if the local state is synced with the given raft state, including hash check.
func (s *Syncer) IsSyncedWithRaft(raftState *raft.RaftBlockState) (int, error) {
	state, err := s.store.GetState(s.ctx)
	if err != nil {
		return 0, err
	}

	diff := int64(state.LastBlockHeight) - int64(raftState.Height)
	if diff != 0 {
		return int(diff), nil
	}

	if raftState.Height == 0 { // initial
		return 0, nil
	}

	header, err := s.store.GetHeader(s.ctx, raftState.Height)
	if err != nil {
		s.logger.Error().Err(err).Uint64("height", raftState.Height).Msg("failed to get header for sync check")
		return 0, fmt.Errorf("get header for sync check at height %d: %w", raftState.Height, err)
	}
	if !bytes.Equal(header.Hash(), raftState.Hash) {
		return 0, fmt.Errorf("header hash mismatch: %x vs %x", header.Hash(), raftState.Hash)
	}

	return 0, nil
}

// RecoverFromRaft attempts to recover the state from a raft block state
func (s *Syncer) RecoverFromRaft(ctx context.Context, raftState *raft.RaftBlockState) error {
	s.logger.Info().Uint64("height", raftState.Height).Msg("recovering state from raft")

	var header types.SignedHeader
	if err := header.UnmarshalBinary(raftState.Header); err != nil {
		return fmt.Errorf("unmarshal header: %w", err)
	}

	var data types.Data
	if err := data.UnmarshalBinary(raftState.Data); err != nil {
		return fmt.Errorf("unmarshal data: %w", err)
	}

	currentState := s.getLastState()

	// Defensive: if lastState is not yet initialized (e.g., RecoverFromRaft called before Start),
	// load it from the store to ensure we have valid state for validation.
	if currentState.ChainID == "" {
		s.logger.Debug().Msg("lastState not initialized, loading from store for recovery")
		var err error
		currentState, err = s.store.GetState(ctx)
		if err != nil {
			// If store has no state either, initialize from genesis
			s.logger.Debug().Err(err).Msg("no state in store, using genesis defaults for recovery")
			currentState = types.State{
				ChainID:         s.genesis.ChainID,
				InitialHeight:   s.genesis.InitialHeight,
				LastBlockHeight: s.genesis.InitialHeight - 1,
				LastBlockTime:   s.genesis.StartTime,
				DAHeight:        s.genesis.DAStartHeight,
			}
		}
	}

	if currentState.LastBlockHeight == raftState.Height {
		if !bytes.Equal(currentState.LastHeaderHash, raftState.Hash) {
			return fmt.Errorf("header hash mismatch: %x vs %x", currentState.LastHeaderHash, raftState.Hash)
		}
		s.logger.Debug().Msg("header hash matches")
		return nil
	} else if currentState.LastBlockHeight+1 == raftState.Height { // raft is 1 block ahead
		// apply block
		err := s.TrySyncNextBlock(ctx, &common.DAHeightEvent{
			Header: &header,
			Data:   &data,
			Source: "",
		})
		if err != nil {
			return err
		}
		s.logger.Info().Uint64("height", raftState.Height).Msg("recovered from raft state")
		return nil
	}

	if currentState.LastBlockHeight > raftState.Height {
		return fmt.Errorf("invalid block height: %d (expected %d)", raftState.Height, currentState.LastBlockHeight+1)
	}

	return nil
}
