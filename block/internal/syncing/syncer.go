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

	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/block/internal/da"
	coreexecutor "github.com/evstack/ev-node/core/execution"
	"github.com/evstack/ev-node/pkg/config"
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

// Syncer handles block synchronization from the DA layer and raft sources.
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

	// Channels for coordination
	heightInCh chan common.DAHeightEvent
	errorCh    chan<- error // Channel to report critical execution client failures
	inFlight   atomic.Int64

	// Handlers
	daRetriever DARetriever
	fiRetriever da.ForcedInclusionRetriever

	raftRetriever *raftRetriever

	daFollower DAFollower

	// Forced inclusion tracking
	forcedInclusionMu    sync.RWMutex
	seenBlockTxs         map[string]struct{} // SHA-256 hex of every tx seen in a DA-sourced block
	seenBlockTxsByHeight map[uint64][]string // DA height → hashes at that height (for pruning)
	daBlockBytes         map[uint64]uint64   // DA height → total tx bytes (for congestion tracking)
	lastCheckedEpochEnd  uint64              // highest epochEnd fully verified so far

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	hasCriticalError atomic.Bool

	// blockSyncer is the interface used for block sync operations.
	// defaults to self, but can be wrapped with tracing.
	blockSyncer BlockSyncer
}

// NewSyncer creates a new block syncer
func NewSyncer(
	store store.Store,
	exec coreexecutor.Executor,
	daClient da.Client,
	cacheMgr cache.Manager,
	metrics *common.Metrics,
	config config.Config,
	genesis genesis.Genesis,
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
		cache:                cacheMgr,
		metrics:              metrics,
		config:               config,
		genesis:              genesis,
		options:              options,
		lastState:            &atomic.Pointer[types.State]{},
		daClient:             daClient,
		daRetrieverHeight:    daRetrieverHeight,
		heightInCh:           make(chan common.DAHeightEvent, 100),
		errorCh:              errorCh,
		logger:               logger.With().Str("component", "syncer").Logger(),
		seenBlockTxs:         make(map[string]struct{}),
		seenBlockTxsByHeight: make(map[uint64][]string),
		daBlockBytes:         make(map[uint64]uint64),
	}
	s.blockSyncer = s
	if raftNode != nil && !reflect.ValueOf(raftNode).IsNil() {
		s.raftRetriever = newRaftRetriever(raftNode, genesis, logger, s,
			func(ctx context.Context, state *raft.RaftBlockState) error {
				s.logger.Debug().Uint64("header_height", state.LastSubmittedDaHeaderHeight).Uint64("data_height", state.LastSubmittedDaDataHeight).Msg("received raft block state")
				cacheMgr.SetLastSubmittedHeaderHeight(ctx, state.LastSubmittedDaHeaderHeight)
				cacheMgr.SetLastSubmittedDataHeight(ctx, state.LastSubmittedDaDataHeight)
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
// The component should not be started after being stopped.
func (s *Syncer) Start(ctx context.Context) (err error) {
	if s.cancel != nil {
		return errors.New("syncer already started")
	}
	ctx, cancel := context.WithCancel(ctx)
	s.ctx, s.cancel = ctx, cancel

	defer func() {
		if err != nil {
			_ = s.Stop(context.Background())
		}
	}()

	if err = s.initializeState(); err != nil {
		return fmt.Errorf("failed to initialize syncer state: %w", err)
	}

	// Initialize handlers
	s.daRetriever = NewDARetriever(s.daClient, s.cache, s.genesis, s.logger)
	if s.config.Instrumentation.IsTracingEnabled() {
		s.daRetriever = WithTracingDARetriever(s.daRetriever)
	}

	s.fiRetriever = da.NewForcedInclusionRetriever(s.daClient, s.logger, s.config.DA.BlockTime.Duration, s.config.Instrumentation.IsTracingEnabled(), s.genesis.DAStartHeight, s.genesis.DAEpochForcedInclusion)
	s.fiRetriever.Start(ctx)

	if s.raftRetriever != nil {
		if err = s.raftRetriever.Start(ctx); err != nil {
			return fmt.Errorf("start raft retriever: %w", err)
		}
	}

	if !s.waitForGenesis() {
		return nil
	}

	// Start main processing loop
	s.wg.Go(func() { s.processLoop(ctx) })

	// Start the DA follower (subscribe + catchup) and other workers
	s.daFollower = NewDAFollower(DAFollowerConfig{
		Client:        s.daClient,
		Retriever:     s.daRetriever,
		Logger:        s.logger,
		EventSink:     s,
		Namespace:     s.daClient.GetHeaderNamespace(),
		DataNamespace: s.daClient.GetDataNamespace(),
		StartDAHeight: s.daRetrieverHeight.Load(),
		DABlockTime:   s.config.DA.BlockTime.Duration,
	})
	if err = s.daFollower.Start(ctx); err != nil {
		return fmt.Errorf("failed to start DA follower: %w", err)
	}

	s.wg.Go(func() { s.pendingWorkerLoop(ctx) })

	s.logger.Info().Msg("syncer started")
	return nil
}

// Stop shuts down the syncing component
func (s *Syncer) Stop(ctx context.Context) error {
	if s.cancel == nil {
		return nil
	}

	s.cancel()

	if s.fiRetriever != nil {
		s.fiRetriever.Stop()
	}

	if s.daFollower != nil {
		s.daFollower.Stop()
	}

	if s.raftRetriever != nil {
		s.raftRetriever.Stop()
	}

	s.wg.Wait()

	// Skip draining if we're shutting down due to a critical error (e.g. execution
	// client unavailable).
	if !s.hasCriticalError.Load() {
		drainCtx, drainCancel := context.WithTimeout(ctx, 5*time.Second)
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

	// Initialize lastCheckedEpochEnd based on the restored state's DA height so that
	// VerifyForcedInclusionTxs resumes from where we left off instead of re-scanning
	// all epochs from genesis on every startup.
	if epochSize := s.genesis.DAEpochForcedInclusion; epochSize > 0 && state.DAHeight >= s.genesis.DAStartHeight {
		firstEpochEnd := s.genesis.DAStartHeight + epochSize - 1
		if state.DAHeight >= firstEpochEnd {
			elapsed := state.DAHeight - firstEpochEnd
			completedEpochs := elapsed / epochSize
			s.lastCheckedEpochEnd = firstEpochEnd + completedEpochs*epochSize
		}
	}

	// Set DA height to the maximum of the genesis start height, the state's DA height, and the cached DA height.
	// The cache's DaHeight() is initialized from store metadata, so it's always correct even after cache clear.
	daHeight := s.genesis.DAStartHeight
	if state.DAHeight > s.genesis.DAStartHeight {
		daHeight = max(daHeight, state.DAHeight-1)
	}

	daHeight = max(daHeight, s.cache.DaHeight())

	// dev mode for da start height
	if startHeight := s.config.DA.StartHeight; startHeight > 0 {
		s.logger.Info().
			Uint64("previous_da_start_height", daHeight).
			Uint64("override_da_start_height", s.config.DA.StartHeight).
			Msg("DA start height overridden by flag")
		daHeight = startHeight
	}

	s.daRetrieverHeight.Store(daHeight)

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
func (s *Syncer) processLoop(ctx context.Context) {
	s.logger.Info().Msg("starting process loop")
	defer s.logger.Info().Msg("process loop stopped")

	for {
		select {
		case <-ctx.Done():
			return
		case heightEvent, ok := <-s.heightInCh:
			if ok {
				s.inFlight.Add(1)
				s.processHeightEvent(ctx, &heightEvent)
				s.inFlight.Add(-1)
			}
		}
	}
}

// HasReachedDAHead returns true once the DA follower has caught up to the DA head.
// Once set, it stays true.
func (s *Syncer) HasReachedDAHead() bool {
	if s.daFollower != nil {
		return s.daFollower.HasReachedHead()
	}
	return false
}

// PendingCount returns the number of unprocessed height events in the pipeline.
func (s *Syncer) PendingCount() int {
	return len(s.heightInCh) + int(s.inFlight.Load()) + s.cache.PendingEventsCount()
}

func (s *Syncer) pendingWorkerLoop(ctx context.Context) {
	s.logger.Info().Msg("starting pending worker")
	defer s.logger.Info().Msg("pending worker stopped")

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.processPendingEvents(ctx)
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

func (s *Syncer) PipeEvent(ctx context.Context, event common.DAHeightEvent) error {
	// Avoid sending already seen events to channel (would have been skipped in processHeightEvent anyway)
	if s.cache.IsHeaderSeen(event.Header.Hash().String()) {
		return nil
	}

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
		Str("source", string(event.Source)).
		Msg("processing height event")

	currentHeight, err := s.store.Height(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get current height")
		return
	}

	// Skip if already processed
	if height <= currentHeight || s.cache.IsHeaderSeen(headerHash) {
		s.logger.Debug().
			Uint64("height", height).
			Str("source", string(event.Source)).
			Msg("height already processed")
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
	if event.Source == common.SourceDA && bytes.Equal(event.Header.DataHash, common.DataHashForEmptyTxs) && currentHeight > 0 {
		_, lastData, err := s.store.GetBlockData(ctx, currentHeight)
		if err != nil {
			s.logger.Error().Err(err).Msg("failed to get last data")
			return
		}
		event.Data.LastDataHash = lastData.Hash()
	}

	// Try to sync the next block
	if err := s.blockSyncer.TrySyncNextBlock(ctx, event); err != nil {
		s.logger.Error().Err(err).
			Uint64("event-height", event.Header.Height()).
			Uint64("state-height", s.getLastState().LastBlockHeight).
			Str("source", string(event.Source)).
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
	return s.trySyncNextBlockWithState(ctx, event, s.getLastState())
}

// trySyncNextBlockWithState attempts to sync the next available block using
// the provided current state as the validation/apply baseline.
func (s *Syncer) trySyncNextBlockWithState(ctx context.Context, event *common.DAHeightEvent, currentState types.State) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	header := event.Header
	data := event.Data
	nextHeight := event.Header.Height()
	headerHash := header.Hash().String()

	s.logger.Info().Uint64("height", nextHeight).Str("source", string(event.Source)).Msg("syncing block")

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
	if counter, ok := s.metrics.BlocksSynchronized[event.Source]; ok {
		counter.Add(1)
	}

	// Mark as seen
	s.cache.SetHeaderSeen(headerHash, header.Height())
	if !bytes.Equal(header.DataHash, common.DataHashForEmptyTxs) {
		s.cache.SetDataSeen(data.DACommitment().String(), newState.LastBlockHeight)
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

	return min(baseGracePeriodEpochs+extra, maxGracePeriodEpochs)
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

	if daHeight < daStart || daHeight < s.getLastState().DAHeight {
		return nil
	}

	executionInfo, err := s.exec.GetExecutionInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get execution info: %w", err)
	}

	var maliciousCount int

	// Resume from the last checked epoch rather than re-scanning from genesis.
	// If no epoch has been checked yet, start from the first epoch end.
	firstEpochEnd := daStart + epochSize - 1
	var startEpochEnd uint64
	if s.lastCheckedEpochEnd == 0 || s.lastCheckedEpochEnd < firstEpochEnd {
		startEpochEnd = firstEpochEnd
	} else {
		startEpochEnd = s.lastCheckedEpochEnd + epochSize
	}

	for epochEnd := startEpochEnd; ; epochEnd += epochSize {
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
func (s *Syncer) processPendingEvents(ctx context.Context) {
	currentHeight, err := s.store.Height(ctx)
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
		case <-ctx.Done():
			s.cache.SetPendingEvent(nextHeight, event)
			return
		default:
			s.cache.SetPendingEvent(nextHeight, event)
			return
		}

		nextHeight++
	}
}

func (s *Syncer) sleepOrDone(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
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
	headerHash := header.Hash()
	if !bytes.Equal(headerHash, raftState.Hash) {
		return 0, fmt.Errorf("header hash mismatch: %x vs %x", headerHash, raftState.Hash)
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
		event := &common.DAHeightEvent{
			Header: &header,
			Data:   &data,
			Source: common.SourceRaft,
		}
		err := s.trySyncNextBlockWithState(ctx, event, currentState)
		if err != nil {
			return err
		}
		s.logger.Info().Uint64("height", raftState.Height).Msg("recovered from raft state")
		return nil
	}

	if currentState.LastBlockHeight > raftState.Height {
		// Local EVM is ahead of the raft snapshot. This is expected on restart when
		// the raft FSM hasn't finished replaying log entries yet (stale snapshot height),
		// or when log entries were compacted and the FSM is awaiting a new snapshot from
		// the leader. Verify that our local block at raftState.Height has the same hash
		// to confirm shared history before skipping recovery.
		localHeader, err := s.store.GetHeader(ctx, raftState.Height)
		if err != nil {
			return fmt.Errorf("local state ahead of raft snapshot (local=%d raft=%d), cannot verify hash: %w",
				currentState.LastBlockHeight, raftState.Height, err)
		}
		localHash := localHeader.Hash()
		if !bytes.Equal(localHash, raftState.Hash) {
			return fmt.Errorf("local state diverged from raft at height %d: local hash %x != raft hash %x",
				raftState.Height, localHash, raftState.Hash)
		}
		s.logger.Info().
			Uint64("local_height", currentState.LastBlockHeight).
			Uint64("raft_height", raftState.Height).
			Msg("local state ahead of stale raft snapshot with matching hash; skipping recovery, raft will catch up")
		return nil
	}

	return nil
}
