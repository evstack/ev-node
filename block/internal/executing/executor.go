package executing

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/evstack/ev-node/pkg/raft"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	coreexecutor "github.com/evstack/ev-node/core/execution"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

var _ BlockProducer = (*Executor)(nil)

// Executor handles block production, transaction processing, and state management
type Executor struct {
	// Core components
	store     store.Store
	exec      coreexecutor.Executor
	sequencer coresequencer.Sequencer
	signer    signer.Signer

	// Shared components
	cache   cache.Manager
	metrics *common.Metrics

	// Broadcasting
	headerBroadcaster common.Broadcaster[*types.SignedHeader]
	dataBroadcaster   common.Broadcaster[*types.Data]

	// Configuration
	config  config.Config
	genesis genesis.Genesis
	options common.BlockOptions

	// Raft consensus
	raftNode common.RaftNode

	// State management
	lastState *atomic.Pointer[types.State]

	// Channels for coordination
	txNotifyCh chan struct{}
	errorCh    chan<- error // Channel to report critical execution client failures

	// Logging
	logger zerolog.Logger

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// blockProducer is the interface used for block production operations.
	// defaults to self, but can be wrapped with tracing.
	blockProducer BlockProducer
}

// NewExecutor creates a new block executor.
// The executor is responsible for:
// - Block production from sequencer batches
// - State transitions and validation
// - P2P broadcasting of produced blocks
// - DA submission of headers and data
//
// When BasedSequencer is enabled, signer can be nil as blocks are not signed.
func NewExecutor(
	store store.Store,
	exec coreexecutor.Executor,
	sequencer coresequencer.Sequencer,
	signer signer.Signer,
	cache cache.Manager,
	metrics *common.Metrics,
	config config.Config,
	genesis genesis.Genesis,
	headerBroadcaster common.Broadcaster[*types.SignedHeader],
	dataBroadcaster common.Broadcaster[*types.Data],
	logger zerolog.Logger,
	options common.BlockOptions,
	errorCh chan<- error,
	raftNode common.RaftNode,
) (*Executor, error) {
	// For based sequencer, signer is optional as blocks are not signed
	if !config.Node.BasedSequencer {
		if signer == nil {
			return nil, errors.New("signer cannot be nil")
		}

		addr, err := signer.GetAddress()
		if err != nil {
			return nil, fmt.Errorf("failed to get address: %w", err)
		}

		if !bytes.Equal(addr, genesis.ProposerAddress) {
			return nil, common.ErrNotProposer
		}
	}
	if raftNode != nil && reflect.ValueOf(raftNode).IsNil() {
		raftNode = nil
	}

	e := &Executor{
		store:             store,
		exec:              exec,
		sequencer:         sequencer,
		signer:            signer,
		cache:             cache,
		metrics:           metrics,
		config:            config,
		genesis:           genesis,
		headerBroadcaster: headerBroadcaster,
		dataBroadcaster:   dataBroadcaster,
		options:           options,
		lastState:         &atomic.Pointer[types.State]{},
		raftNode:          raftNode,
		txNotifyCh:        make(chan struct{}, 1),
		errorCh:           errorCh,
		logger:            logger.With().Str("component", "executor").Logger(),
	}
	e.blockProducer = e
	return e, nil
}

// SetBlockProducer sets the block producer interface, allowing injection of
// a tracing wrapper or other decorator.
func (e *Executor) SetBlockProducer(bp BlockProducer) {
	e.blockProducer = bp
}

// Start begins the execution component
func (e *Executor) Start(ctx context.Context) error {
	e.ctx, e.cancel = context.WithCancel(ctx)

	// Initialize state
	if err := e.initializeState(); err != nil {
		return fmt.Errorf("failed to initialize state: %w", err)
	}

	// Start execution loop
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		e.executionLoop()
	}()

	e.logger.Info().Msg("executor started")
	return nil
}

// Stop shuts down the execution component
func (e *Executor) Stop() error {
	if e.cancel != nil {
		e.cancel()
	}
	e.wg.Wait()

	e.logger.Info().Msg("executor stopped")
	return nil
}

// GetLastState returns the current state.
func (e *Executor) GetLastState() types.State {
	state := e.getLastState()
	state.AppHash = bytes.Clone(state.AppHash)
	return state
}

// getLastState returns the current state.
// getLastState should never directly mutate.
func (e *Executor) getLastState() types.State {
	state := e.lastState.Load()
	if state == nil {
		return types.State{}
	}

	return *state
}

// setLastState updates the current state
func (e *Executor) setLastState(state types.State) {
	e.lastState.Store(&state)
}

// NotifyNewTransactions signals that new transactions are available
func (e *Executor) NotifyNewTransactions() {
	select {
	case e.txNotifyCh <- struct{}{}:
	default:
		// Channel full, notification already pending
	}
}

// initializeState loads or creates the initial blockchain state
func (e *Executor) initializeState() error {
	// Try to load existing state
	state, err := e.store.GetState(e.ctx)
	if err != nil {
		// Initialize new chain
		e.logger.Info().Msg("initializing new blockchain state")

		stateRoot, _, err := e.exec.InitChain(e.ctx, e.genesis.StartTime,
			e.genesis.InitialHeight, e.genesis.ChainID)
		if err != nil {
			e.sendCriticalError(fmt.Errorf("failed to initialize chain: %w", err))
			return fmt.Errorf("failed to initialize chain: %w", err)
		}

		state = types.State{
			ChainID:         e.genesis.ChainID,
			InitialHeight:   e.genesis.InitialHeight,
			LastBlockHeight: e.genesis.InitialHeight - 1,
			LastBlockTime:   e.genesis.StartTime,
			AppHash:         stateRoot,
			// DA start height is usually 0 at InitChain unless it is a re-genesis or a based sequencer.
			DAHeight: e.genesis.DAStartHeight,
		}
	}

	if e.raftNode != nil {
		// Ensure node is fully synced before producing any blocks
		raftState := e.raftNode.GetState()
		if raftState.Height != 0 {
			// Node cannot be ahead of raft - that indicates divergence
			if state.LastBlockHeight > raftState.Height {
				return fmt.Errorf("invalid state: local height (%d) ahead of raft (%d)", state.LastBlockHeight, raftState.Height)
			}
			// Node behind raft is OK - the replayer will catch it up
			if state.LastBlockHeight < raftState.Height {
				e.logger.Warn().
					Uint64("local", state.LastBlockHeight).
					Uint64("raft", raftState.Height).
					Msg("local state behind raft, will sync during startup")
			}
			// If heights match, verify hashes as well (to detect divergence)
			if state.LastBlockHeight > 0 && state.LastBlockHeight == raftState.Height {
				header, err := e.store.GetHeader(e.ctx, state.LastBlockHeight)
				if err != nil {
					return fmt.Errorf("failed to get header at %d for sync check: %w", state.LastBlockHeight, err)
				}
				if !bytes.Equal(header.Hash(), raftState.Hash) {
					return fmt.Errorf("invalid state: block hash mismatch at height %d: raft=%x local=%x", state.LastBlockHeight, raftState.Hash, header.Hash())
				}
			}
		}
	}
	e.setLastState(state)
	e.sequencer.SetDAHeight(state.DAHeight)

	// Initialize store height using batch for atomicity
	batch, err := e.store.NewBatch(e.ctx)
	if err != nil {
		return fmt.Errorf("failed to create batch: %w", err)
	}
	if err := batch.SetHeight(state.LastBlockHeight); err != nil {
		return fmt.Errorf("failed to set store height: %w", err)
	}
	if err := batch.Commit(); err != nil {
		return fmt.Errorf("failed to commit batch: %w", err)
	}

	e.logger.Info().Uint64("height", state.LastBlockHeight).
		Str("chain_id", state.ChainID).Msg("initialized state")

	// Determine sync target: use Raft height if node is behind Raft consensus
	syncTargetHeight := state.LastBlockHeight
	if e.raftNode != nil {
		raftState := e.raftNode.GetState()
		if raftState.Height > syncTargetHeight {
			syncTargetHeight = raftState.Height
			e.logger.Info().
				Uint64("local_height", state.LastBlockHeight).
				Uint64("raft_height", raftState.Height).
				Msg("using raft height as sync target")
		}
	}

	// Sync execution layer to the target height (Raft height if behind, local height otherwise)
	execReplayer := common.NewReplayer(e.store, e.exec, e.genesis, e.logger)
	if err := execReplayer.SyncToHeight(e.ctx, syncTargetHeight); err != nil {
		e.sendCriticalError(fmt.Errorf("failed to sync execution layer: %w", err))
		return fmt.Errorf("failed to sync execution layer: %w", err)
	}

	return nil
}

// executionLoop handles block production and aggregation
func (e *Executor) executionLoop() {
	e.logger.Info().Msg("starting execution loop")
	defer e.logger.Info().Msg("execution loop stopped")

	var delay time.Duration
	initialHeight := e.genesis.InitialHeight
	currentState := e.getLastState()

	if currentState.LastBlockHeight < initialHeight {
		delay = time.Until(e.genesis.StartTime.Add(e.config.Node.BlockTime.Duration))
	} else {
		delay = time.Until(currentState.LastBlockTime.Add(e.config.Node.BlockTime.Duration))
	}

	if delay > 0 {
		e.logger.Info().Dur("delay", delay).Msg("waiting to start block production")
		select {
		case <-e.ctx.Done():
			return
		case <-time.After(delay):
		}
	}

	blockTimer := time.NewTimer(e.config.Node.BlockTime.Duration)
	defer blockTimer.Stop()

	var lazyTimer *time.Timer
	var lazyTimerCh <-chan time.Time
	if e.config.Node.LazyMode {
		// lazyTimer triggers block publication even during inactivity
		lazyTimer = time.NewTimer(e.config.Node.LazyBlockInterval.Duration)
		defer lazyTimer.Stop()
		lazyTimerCh = lazyTimer.C
	}
	txsAvailable := false

	for {
		select {
		case <-e.ctx.Done():
			return

		case <-blockTimer.C:
			if e.config.Node.LazyMode && !txsAvailable {
				// In lazy mode without transactions, just continue ticking
				blockTimer.Reset(e.config.Node.BlockTime.Duration)
				continue
			}

			if err := e.blockProducer.ProduceBlock(e.ctx); err != nil {
				e.logger.Error().Err(err).Msg("failed to produce block")
			}
			txsAvailable = false
			// Always reset block timer to keep ticking
			blockTimer.Reset(e.config.Node.BlockTime.Duration)

		case <-lazyTimerCh:
			e.logger.Debug().Msg("Lazy timer triggered block production")
			if err := e.blockProducer.ProduceBlock(e.ctx); err != nil {
				e.logger.Error().Err(err).Msg("failed to produce block from lazy timer")
			}
			// Reset lazy timer
			lazyTimer.Reset(e.config.Node.LazyBlockInterval.Duration)

		case <-e.txNotifyCh:
			txsAvailable = true
		}
	}
}

// ProduceBlock creates, validates, and stores a new block.
func (e *Executor) ProduceBlock(ctx context.Context) error {
	start := time.Now()
	defer func() {
		if e.metrics.OperationDuration["block_production"] != nil {
			duration := time.Since(start).Seconds()
			e.metrics.OperationDuration["block_production"].Observe(duration)
		}
	}()

	// Check raft cluster health before producing block - ensures quorum is available
	if e.raftNode != nil && !e.raftNode.HasQuorum() {
		return errors.New("raft cluster does not have quorum")
	}

	currentState := e.getLastState()
	newHeight := currentState.LastBlockHeight + 1

	e.logger.Debug().Uint64("height", newHeight).Msg("producing block")

	// check pending limits
	if e.config.Node.MaxPendingHeadersAndData > 0 {
		pendingHeaders := e.cache.NumPendingHeaders()
		pendingData := e.cache.NumPendingData()
		if pendingHeaders >= e.config.Node.MaxPendingHeadersAndData ||
			pendingData >= e.config.Node.MaxPendingHeadersAndData {
			e.logger.Warn().
				Uint64("pending_headers", pendingHeaders).
				Uint64("pending_data", pendingData).
				Uint64("limit", e.config.Node.MaxPendingHeadersAndData).
				Msg("pending limit reached, skipping block production")
			return nil
		}
	}

	var (
		header    *types.SignedHeader
		data      *types.Data
		batchData *BatchData
	)

	// Check if there's an already stored block at the newHeight
	// If there is use that instead of creating a new block
	pendingHeader, pendingData, err := e.store.GetBlockData(ctx, newHeight)
	if err == nil {
		e.logger.Info().Uint64("height", newHeight).Msg("using pending block")
		header = pendingHeader
		data = pendingData
	} else if !errors.Is(err, datastore.ErrNotFound) {
		return fmt.Errorf("failed to get block data: %w", err)
	} else {
		// get batch from sequencer
		batchData, err = e.blockProducer.RetrieveBatch(ctx)
		if errors.Is(err, common.ErrNoBatch) {
			e.logger.Debug().Msg("no batch available")
			return nil
		} else if errors.Is(err, common.ErrNoTransactionsInBatch) {
			e.logger.Debug().Msg("no transactions in batch")
		} else if err != nil {
			return fmt.Errorf("failed to retrieve batch: %w", err)
		}

		header, data, err = e.blockProducer.CreateBlock(ctx, newHeight, batchData)
		if err != nil {
			return fmt.Errorf("failed to create block: %w", err)
		}

		// saved early for crash recovery, will be overwritten later with the final signature
		batch, err := e.store.NewBatch(ctx)
		if err != nil {
			return fmt.Errorf("failed to create batch for early save: %w", err)
		}
		if err = batch.SaveBlockData(header, data, &types.Signature{}); err != nil {
			return fmt.Errorf("failed to save block data: %w", err)
		}
		if err = batch.Commit(); err != nil {
			return fmt.Errorf("failed to commit early save batch: %w", err)
		}
	}

	// Pass force-included mask through context for execution optimization
	// Force-included txs (from DA) MUST be validated as they're from untrusted sources
	// Mempool txs can skip validation as they were validated when added to mempool
	applyCtx := ctx
	if batchData != nil && batchData.Batch != nil && batchData.ForceIncludedMask != nil {
		applyCtx = coreexecutor.WithForceIncludedMask(applyCtx, batchData.ForceIncludedMask)
	}

	newState, err := e.blockProducer.ApplyBlock(applyCtx, header.Header, data)
	if err != nil {
		return fmt.Errorf("failed to apply block: %w", err)
	}

	// set the DA height in the sequencer
	newState.DAHeight = e.sequencer.GetDAHeight()

	// signing the header is done after applying the block
	// as for signing, the state of the block may be required by the signature payload provider.
	// For based sequencer, this will return an empty signature
	signature, err := e.signHeader(header.Header)
	if err != nil {
		return fmt.Errorf("failed to sign header: %w", err)
	}
	header.Signature = signature

	if err := e.blockProducer.ValidateBlock(ctx, currentState, header, data); err != nil {
		e.sendCriticalError(fmt.Errorf("failed to validate block: %w", err))
		e.logger.Error().Err(err).Msg("CRITICAL: Permanent block validation error - halting block production")
		return fmt.Errorf("failed to validate block: %w", err)
	}

	batch, err := e.store.NewBatch(ctx)
	if err != nil {
		return fmt.Errorf("failed to create batch: %w", err)
	}

	if err := batch.SaveBlockData(header, data, &signature); err != nil {
		return fmt.Errorf("failed to save block: %w", err)
	}

	if err := batch.SetHeight(newHeight); err != nil {
		return fmt.Errorf("failed to update store height: %w", err)
	}

	if err := batch.UpdateState(newState); err != nil {
		return fmt.Errorf("failed to update state: %w", err)
	}

	// Propose block to raft to share state in the cluster
	if e.raftNode != nil {
		headerBytes, err := header.MarshalBinary()
		if err != nil {
			return fmt.Errorf("failed to marshal header: %w", err)
		}
		dataBytes, err := data.MarshalBinary()
		if err != nil {
			return fmt.Errorf("failed to marshal data: %w", err)
		}

		raftState := &raft.RaftBlockState{
			Height:                      newHeight,
			Hash:                        header.Hash(),
			Timestamp:                   header.BaseHeader.Time,
			Header:                      headerBytes,
			Data:                        dataBytes,
			LastSubmittedDaHeaderHeight: e.cache.GetLastSubmittedHeaderHeight(),
			LastSubmittedDaDataHeight:   e.cache.GetLastSubmittedDataHeight(),
		}
		if err := e.raftNode.Broadcast(e.ctx, raftState); err != nil {
			return fmt.Errorf("failed to propose block to raft: %w", err)
		}
		e.logger.Debug().Uint64("height", newHeight).Msg("proposed block to raft")
	}
	if err := batch.Commit(); err != nil {
		return fmt.Errorf("failed to commit batch: %w", err)
	}

	// Sync db to disk after each block
	if err := e.store.Sync(e.ctx); err != nil {
		return fmt.Errorf("failed to sync store: %w", err)
	}

	// Update in-memory state after successful commit
	e.setLastState(newState)

	// broadcast header and data to P2P network
	g, broadcastCtx := errgroup.WithContext(ctx)
	g.Go(func() error { return e.headerBroadcaster.WriteToStoreAndBroadcast(broadcastCtx, header) })
	g.Go(func() error { return e.dataBroadcaster.WriteToStoreAndBroadcast(broadcastCtx, data) })
	if err := g.Wait(); err != nil {
		e.logger.Error().Err(err).Msg("failed to broadcast header and/data")
		// don't fail block production on broadcast error
	}

	e.recordBlockMetrics(newState, data)

	e.logger.Info().
		Uint64("height", newHeight).
		Int("txs", len(data.Txs)).
		Msg("produced block")

	return nil
}

// RetrieveBatch gets the next batch of transactions from the sequencer.
func (e *Executor) RetrieveBatch(ctx context.Context) (*BatchData, error) {
	req := coresequencer.GetNextBatchRequest{
		Id:            []byte(e.genesis.ChainID),
		MaxBytes:      common.DefaultMaxBlobSize,
		LastBatchData: [][]byte{}, // Can be populated if needed for sequencer context
	}

	res, err := e.sequencer.GetNextBatch(ctx, req)
	if err != nil {
		return nil, err
	}

	if res == nil || res.Batch == nil {
		return nil, common.ErrNoBatch
	}

	if len(res.Batch.Transactions) == 0 {
		return &BatchData{
			Batch: res.Batch,
			Time:  res.Timestamp,
			Data:  res.BatchData,
		}, common.ErrNoTransactionsInBatch
	}

	return &BatchData{
		Batch: res.Batch,
		Time:  res.Timestamp,
		Data:  res.BatchData,
	}, nil
}

// CreateBlock creates a new block from the given batch.
func (e *Executor) CreateBlock(ctx context.Context, height uint64, batchData *BatchData) (*types.SignedHeader, *types.Data, error) {
	currentState := e.getLastState()
	headerTime := uint64(e.genesis.StartTime.UnixNano())

	// Get last block info
	var lastHeaderHash types.Hash
	var lastDataHash types.Hash
	var lastSignature types.Signature

	if height > e.genesis.InitialHeight {
		headerTime = uint64(batchData.UnixNano())

		lastHeader, lastData, err := e.store.GetBlockData(ctx, height-1)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get last block: %w", err)
		}
		lastHeaderHash = lastHeader.Hash()
		lastDataHash = lastData.Hash()

		lastSignaturePtr, err := e.store.GetSignature(ctx, height-1)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get last signature: %w", err)
		}
		lastSignature = *lastSignaturePtr
	}

	// Get signer info and validator hash
	var pubKey crypto.PubKey
	var validatorHash types.Hash

	if e.signer != nil {
		var err error
		pubKey, err = e.signer.GetPublic()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get public key: %w", err)
		}

		validatorHash, err = e.options.ValidatorHasherProvider(e.genesis.ProposerAddress, pubKey)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get validator hash: %w", err)
		}
	} else {
		// For based sequencer without signer, use nil pubkey and compute validator hash
		var err error
		validatorHash, err = e.options.ValidatorHasherProvider(e.genesis.ProposerAddress, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get validator hash: %w", err)
		}
	}

	// Create header
	header := &types.SignedHeader{
		Header: types.Header{
			Version: types.Version{
				Block: currentState.Version.Block,
				App:   currentState.Version.App,
			},
			BaseHeader: types.BaseHeader{
				ChainID: e.genesis.ChainID,
				Height:  height,
				Time:    headerTime,
			},
			LastHeaderHash:  lastHeaderHash,
			AppHash:         currentState.AppHash,
			ProposerAddress: e.genesis.ProposerAddress,
			ValidatorHash:   validatorHash,
		},
		Signature: lastSignature,
		Signer: types.Signer{
			PubKey:  pubKey,
			Address: e.genesis.ProposerAddress,
		},
	}

	// Create data
	data := &types.Data{
		Txs: make(types.Txs, len(batchData.Transactions)),
		Metadata: &types.Metadata{
			ChainID:      header.ChainID(),
			Height:       header.Height(),
			Time:         header.BaseHeader.Time,
			LastDataHash: lastDataHash,
		},
	}

	for i, tx := range batchData.Transactions {
		data.Txs[i] = tx
	}

	// Set data hash
	if len(data.Txs) == 0 {
		header.DataHash = common.DataHashForEmptyTxs
	} else {
		header.DataHash = data.DACommitment()
	}

	return header, data, nil
}

// ApplyBlock applies the block to get the new state.
func (e *Executor) ApplyBlock(ctx context.Context, header types.Header, data *types.Data) (types.State, error) {
	currentState := e.getLastState()

	// Prepare transactions
	rawTxs := make([][]byte, len(data.Txs))
	for i, tx := range data.Txs {
		rawTxs[i] = []byte(tx)
	}

	// Execute transactions
	ctx = context.WithValue(ctx, types.HeaderContextKey, header)

	newAppHash, err := e.executeTxsWithRetry(ctx, rawTxs, header, currentState)
	if err != nil {
		e.sendCriticalError(fmt.Errorf("failed to execute transactions: %w", err))
		return types.State{}, fmt.Errorf("failed to execute transactions: %w", err)
	}

	// Create new state
	newState, err := currentState.NextState(header, newAppHash)
	if err != nil {
		return types.State{}, fmt.Errorf("failed to create next state: %w", err)
	}

	return newState, nil
}

// signHeader signs the block header
func (e *Executor) signHeader(header types.Header) (types.Signature, error) {
	// For based sequencer, return empty signature as there is no signer
	if e.signer == nil {
		return types.Signature{}, nil
	}

	bz, err := e.options.AggregatorNodeSignatureBytesProvider(&header)
	if err != nil {
		return nil, fmt.Errorf("failed to get signature payload: %w", err)
	}

	return e.signer.Sign(bz)
}

// executeTxsWithRetry executes transactions with retry logic.
// NOTE: the function retries the execution client call regardless of the error. Some execution clients errors are irrecoverable, and will eventually halt the node, as expected.
func (e *Executor) executeTxsWithRetry(ctx context.Context, rawTxs [][]byte, header types.Header, currentState types.State) ([]byte, error) {
	for attempt := 1; attempt <= common.MaxRetriesBeforeHalt; attempt++ {
		newAppHash, _, err := e.exec.ExecuteTxs(ctx, rawTxs, header.Height(), header.Time(), currentState.AppHash)
		if err != nil {
			if attempt == common.MaxRetriesBeforeHalt {
				return nil, fmt.Errorf("failed to execute transactions: %w", err)
			}

			e.logger.Error().Err(err).
				Int("attempt", attempt).
				Int("max_attempts", common.MaxRetriesBeforeHalt).
				Uint64("height", header.Height()).
				Msg("failed to execute transactions, retrying")

			select {
			case <-time.After(common.MaxRetriesTimeout):
				continue
			case <-e.ctx.Done():
				return nil, fmt.Errorf("context cancelled during retry: %w", e.ctx.Err())
			}
		}

		return newAppHash, nil
	}

	return nil, nil
}

// ValidateBlock validates the created block.
func (e *Executor) ValidateBlock(_ context.Context, lastState types.State, header *types.SignedHeader, data *types.Data) error {
	// Set custom verifier for aggregator node signature
	header.SetCustomVerifierForAggregator(e.options.AggregatorNodeSignatureBytesProvider)

	// Basic header validation
	if err := header.ValidateBasic(); err != nil {
		return fmt.Errorf("invalid header: %w", err)
	}

	return lastState.AssertValidForNextState(header, data)
}

// sendCriticalError sends a critical error to the error channel without blocking
func (e *Executor) sendCriticalError(err error) {
	if e.errorCh != nil {
		select {
		case e.errorCh <- err:
		default:
			// Channel full, error already reported
		}
	}
}

// recordBlockMetrics records metrics for the produced block
func (e *Executor) recordBlockMetrics(newState types.State, data *types.Data) {
	e.metrics.Height.Set(float64(newState.LastBlockHeight))

	if data == nil || data.Metadata == nil {
		return
	}

	e.metrics.NumTxs.Set(float64(len(data.Txs)))
	e.metrics.TotalTxs.Add(float64(len(data.Txs)))
	e.metrics.TxsPerBlock.Observe(float64(len(data.Txs)))
	e.metrics.BlockSizeBytes.Set(float64(data.Size()))
	e.metrics.CommittedHeight.Set(float64(data.Metadata.Height))
}

// IsSynced checks if the last block height in the stored state matches the expected height and returns true if they are equal.
func (e *Executor) IsSynced(expHeight uint64) bool {
	state, err := e.store.GetState(e.ctx)
	if err != nil {
		return false
	}
	return state.LastBlockHeight == expHeight
}

// IsSyncedWithRaft checks if the local state is synced with the given raft state, including hash check.
func (e *Executor) IsSyncedWithRaft(raftState *raft.RaftBlockState) (int, error) {
	state, err := e.store.GetState(e.ctx)
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
	header, err := e.store.GetHeader(e.ctx, raftState.Height)
	if err != nil {
		e.logger.Error().Err(err).Uint64("height", raftState.Height).Msg("failed to get header for sync check")
		return 0, fmt.Errorf("get header for sync check: %w", err)
	}

	if !bytes.Equal(header.Hash(), raftState.Hash) {
		return 0, fmt.Errorf("block hash mismatch: %s != %s", header.Hash(), raftState.Hash)
	}

	return 0, nil
}

// BatchData represents batch data from sequencer
type BatchData struct {
	*coresequencer.Batch
	time.Time
	Data [][]byte
}
