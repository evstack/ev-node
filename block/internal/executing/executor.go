package executing

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

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

const (
	// DefaultInterval is the default reaper interval
	DefaultInterval = 1 * time.Second
)

// broadcaster interface for P2P broadcasting
type broadcaster[T any] interface {
	WriteToStoreAndBroadcast(ctx context.Context, payload T) error
}

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
	headerBroadcaster broadcaster[*types.SignedHeader]
	dataBroadcaster   broadcaster[*types.Data]

	// Configuration
	config  config.Config
	genesis genesis.Genesis
	options common.BlockOptions

	// State management
	lastState    types.State
	lastStateMtx *sync.RWMutex

	// Channels for coordination
	txNotifyCh chan struct{}

	// Reaper for transaction processing
	reaper *Reaper

	// Logging
	logger zerolog.Logger

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewExecutor creates a new block executor
func NewExecutor(
	store store.Store,
	exec coreexecutor.Executor,
	sequencer coresequencer.Sequencer,
	signer signer.Signer,
	cache cache.Manager,
	metrics *common.Metrics,
	config config.Config,
	genesis genesis.Genesis,
	headerBroadcaster broadcaster[*types.SignedHeader],
	dataBroadcaster broadcaster[*types.Data],
	logger zerolog.Logger,
	options common.BlockOptions,
) *Executor {
	if signer == nil {
		panic("signer cannot be nil")
	}

	return &Executor{
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
		lastStateMtx:      &sync.RWMutex{},
		txNotifyCh:        make(chan struct{}, 1),
		logger:            logger.With().Str("component", "executor").Logger(),
	}
}

// Start begins the execution component
func (e *Executor) Start(ctx context.Context) error {
	e.ctx, e.cancel = context.WithCancel(ctx)

	// Initialize state
	if err := e.initializeState(); err != nil {
		return fmt.Errorf("failed to initialize state: %w", err)
	}

	// Initialize reaper
	reaperStore, err := store.NewDefaultInMemoryKVStore()
	if err != nil {
		return fmt.Errorf("failed to create reaper store: %w", err)
	}
	e.reaper = NewReaper(e.ctx, e.exec, e.sequencer, e.genesis.ChainID, DefaultInterval, e.logger, reaperStore, e)

	// Start execution loop
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		e.executionLoop()
	}()

	// Start reaper
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		e.reaper.Start(e.ctx)
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

// GetLastState returns the current state
func (e *Executor) GetLastState() types.State {
	e.lastStateMtx.RLock()
	defer e.lastStateMtx.RUnlock()
	return e.lastState
}

// SetLastState updates the current state
func (e *Executor) SetLastState(state types.State) {
	e.lastStateMtx.Lock()
	defer e.lastStateMtx.Unlock()
	e.lastState = state
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
	ctx := context.Background()

	// Try to load existing state
	state, err := e.store.GetState(ctx)
	if err != nil {
		// Initialize new chain
		e.logger.Info().Msg("initializing new blockchain state")

		stateRoot, _, err := e.exec.InitChain(ctx, e.genesis.StartTime,
			e.genesis.InitialHeight, e.genesis.ChainID)
		if err != nil {
			return fmt.Errorf("failed to initialize chain: %w", err)
		}

		// Create genesis block
		if err := e.createGenesisBlock(ctx, stateRoot); err != nil {
			return fmt.Errorf("failed to create genesis block: %w", err)
		}

		state = types.State{
			ChainID:         e.genesis.ChainID,
			InitialHeight:   e.genesis.InitialHeight,
			LastBlockHeight: e.genesis.InitialHeight - 1,
			LastBlockTime:   e.genesis.StartTime,
			AppHash:         stateRoot,
			DAHeight:        0,
		}
	}

	e.SetLastState(state)

	// Set store height
	if err := e.store.SetHeight(ctx, state.LastBlockHeight); err != nil {
		return fmt.Errorf("failed to set store height: %w", err)
	}

	e.logger.Info().Uint64("height", state.LastBlockHeight).
		Str("chain_id", state.ChainID).Msg("initialized state")

	return nil
}

// createGenesisBlock creates and stores the genesis block
func (e *Executor) createGenesisBlock(ctx context.Context, stateRoot []byte) error {
	header := types.Header{
		AppHash:         stateRoot,
		DataHash:        common.DataHashForEmptyTxs,
		ProposerAddress: e.genesis.ProposerAddress,
		BaseHeader: types.BaseHeader{
			ChainID: e.genesis.ChainID,
			Height:  e.genesis.InitialHeight,
			Time:    uint64(e.genesis.StartTime.UnixNano()),
		},
	}

	if e.signer == nil {
		return errors.New("signer cannot be nil")
	}

	pubKey, err := e.signer.GetPublic()
	if err != nil {
		return fmt.Errorf("failed to get public key: %w", err)
	}

	bz, err := e.options.AggregatorNodeSignatureBytesProvider(&header)
	if err != nil {
		return fmt.Errorf("failed to get signature payload: %w", err)
	}

	sig, err := e.signer.Sign(bz)
	if err != nil {
		return fmt.Errorf("failed to sign header: %w", err)
	}

	var signature types.Signature
	data := &types.Data{}

	signature = sig

	genesisHeader := &types.SignedHeader{
		Header: header,
		Signer: types.Signer{
			PubKey:  pubKey,
			Address: e.genesis.ProposerAddress,
		},
		Signature: signature,
	}

	return e.store.SaveBlockData(ctx, genesisHeader, data, &signature)
}

// executionLoop handles block production and aggregation
func (e *Executor) executionLoop() {
	e.logger.Info().Msg("starting execution loop")
	defer e.logger.Info().Msg("execution loop stopped")

	var delay time.Duration
	initialHeight := e.genesis.InitialHeight
	currentState := e.GetLastState()

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
	if e.config.Node.LazyMode {
		// lazyTimer triggers block publication even during inactivity
		lazyTimer = time.NewTimer(e.config.Node.BlockTime.Duration * 2) // 2x block time for lazy timer
		defer lazyTimer.Stop()
	}

	txsAvailable := false

	for {
		select {
		case <-e.ctx.Done():
			return

		case <-blockTimer.C:
			shouldProduce := !e.config.Node.LazyMode || txsAvailable
			if shouldProduce {
				if err := e.produceBlock(); err != nil {
					e.logger.Error().Err(err).Msg("failed to produce block")
				}
				txsAvailable = false
			}
			// Always reset block timer to keep ticking
			blockTimer.Reset(e.config.Node.BlockTime.Duration)

		case <-func() <-chan time.Time {
			if lazyTimer != nil {
				return lazyTimer.C
			}
			return make(chan time.Time) // Never fires for non-lazy mode
		}():
			e.logger.Debug().Msg("Lazy timer triggered block production")
			if err := e.produceBlock(); err != nil {
				e.logger.Error().Err(err).Msg("failed to produce block from lazy timer")
			}
			// Reset lazy timer
			lazyTimer.Reset(e.config.Node.LazyBlockInterval.Duration)

		case <-e.txNotifyCh:
			txsAvailable = true
		}
	}
}

// produceBlock creates, validates, and stores a new block
func (e *Executor) produceBlock() error {
	start := time.Now()
	defer func() {
		if e.metrics.OperationDuration["block_production"] != nil {
			duration := time.Since(start).Seconds()
			e.metrics.OperationDuration["block_production"].Observe(duration)
		}
	}()

	ctx := context.Background()
	currentState := e.GetLastState()
	newHeight := currentState.LastBlockHeight + 1

	e.logger.Debug().Uint64("height", newHeight).Msg("producing block")

	// Check pending limits
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

	// Get batch from sequencer
	batchData, err := e.retrieveBatch(ctx)
	if errors.Is(err, common.ErrNoBatch) {
		e.logger.Debug().Msg("no batch available")
		return nil
	} else if errors.Is(err, common.ErrNoTransactionsInBatch) {
		e.logger.Debug().Msg("no transactions in batch")
	} else if err != nil {
		return fmt.Errorf("failed to retrieve batch: %w", err)
	}

	// Create block
	header, data, err := e.createBlock(ctx, newHeight, batchData)
	if err != nil {
		return fmt.Errorf("failed to create block: %w", err)
	}

	// Apply block to get new state
	newState, err := e.applyBlock(ctx, header.Header, data)
	if err != nil {
		return fmt.Errorf("failed to apply block: %w", err)
	}

	// Sign the header
	signature, err := e.signHeader(header.Header)
	if err != nil {
		return fmt.Errorf("failed to sign header: %w", err)
	}
	header.Signature = signature

	// Validate block
	if err := e.validateBlock(currentState, header, data); err != nil {
		return fmt.Errorf("failed to validate block: %w", err)
	}

	// Save block
	if err := e.store.SaveBlockData(ctx, header, data, &signature); err != nil {
		return fmt.Errorf("failed to save block: %w", err)
	}

	// Update store height
	if err := e.store.SetHeight(ctx, newHeight); err != nil {
		return fmt.Errorf("failed to update store height: %w", err)
	}

	// Update state
	if err := e.updateState(ctx, newState); err != nil {
		return fmt.Errorf("failed to update state: %w", err)
	}

	// Broadcast header and data to P2P network
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error { return e.headerBroadcaster.WriteToStoreAndBroadcast(ctx, header) })
	g.Go(func() error { return e.dataBroadcaster.WriteToStoreAndBroadcast(ctx, data) })
	if err := g.Wait(); err != nil {
		e.logger.Error().Err(err).Msg("failed to broadcast header and/data")
		// Don't fail block production on broadcast error
	}

	// Record metrics
	e.recordBlockMetrics(data)

	e.logger.Info().
		Uint64("height", newHeight).
		Int("txs", len(data.Txs)).
		Msg("produced block")

	return nil
}

// retrieveBatch gets the next batch of transactions from the sequencer
func (e *Executor) retrieveBatch(ctx context.Context) (*BatchData, error) {
	req := coresequencer.GetNextBatchRequest{
		Id: []byte(e.genesis.ChainID),
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

// createBlock creates a new block from the given batch
func (e *Executor) createBlock(ctx context.Context, height uint64, batchData *BatchData) (*types.SignedHeader, *types.Data, error) {
	currentState := e.GetLastState()

	// Get last block info
	var lastHeaderHash types.Hash
	var lastDataHash types.Hash

	if height > e.genesis.InitialHeight {
		lastHeader, lastData, err := e.store.GetBlockData(ctx, height-1)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get last block: %w", err)
		}
		lastHeaderHash = lastHeader.Hash()
		lastDataHash = lastData.Hash()
	}

	// Get signer info
	pubKey, err := e.signer.GetPublic()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get public key: %w", err)
	}

	// Build validator hash
	// Get validator hash
	validatorHash, err := e.options.ValidatorHasherProvider(e.genesis.ProposerAddress, pubKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get validator hash: %w", err)
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
				Time:    uint64(batchData.UnixNano()),
			},
			LastHeaderHash:  lastHeaderHash,
			ConsensusHash:   make(types.Hash, 32),
			AppHash:         currentState.AppHash,
			ProposerAddress: e.genesis.ProposerAddress,
			ValidatorHash:   validatorHash,
		},
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
		data.Txs[i] = types.Tx(tx)
	}

	// Set data hash
	if len(data.Txs) == 0 {
		header.DataHash = common.DataHashForEmptyTxs
	} else {
		header.DataHash = data.DACommitment()
	}

	return header, data, nil
}

// applyBlock applies the block to get the new state
func (e *Executor) applyBlock(ctx context.Context, header types.Header, data *types.Data) (types.State, error) {
	currentState := e.GetLastState()

	// Prepare transactions
	rawTxs := make([][]byte, len(data.Txs))
	for i, tx := range data.Txs {
		rawTxs[i] = []byte(tx)
	}

	// Execute transactions
	ctx = context.WithValue(ctx, types.HeaderContextKey, header)
	newAppHash, _, err := e.exec.ExecuteTxs(ctx, rawTxs, header.Height(),
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

// signHeader signs the block header
func (e *Executor) signHeader(header types.Header) (types.Signature, error) {
	bz, err := e.options.AggregatorNodeSignatureBytesProvider(&header)
	if err != nil {
		return nil, fmt.Errorf("failed to get signature payload: %w", err)
	}

	return e.signer.Sign(bz)
}

// validateBlock validates the created block
func (e *Executor) validateBlock(lastState types.State, header *types.SignedHeader, data *types.Data) error {
	// Set custom verifier for aggregator node signature
	header.SetCustomVerifierForAggregator(e.options.AggregatorNodeSignatureBytesProvider)

	// Basic header validation
	if err := header.ValidateBasic(); err != nil {
		return fmt.Errorf("invalid header: %w", err)
	}

	// Validate header against data
	if err := types.Validate(header, data); err != nil {
		return fmt.Errorf("header-data validation failed: %w", err)
	}

	// Check chain ID
	if header.ChainID() != lastState.ChainID {
		return fmt.Errorf("chain ID mismatch: expected %s, got %s",
			lastState.ChainID, header.ChainID())
	}

	// Check height
	expectedHeight := lastState.LastBlockHeight + 1
	if header.Height() != expectedHeight {
		return fmt.Errorf("invalid height: expected %d, got %d",
			expectedHeight, header.Height())
	}

	// Check timestamp
	if header.Height() > 1 && lastState.LastBlockTime.After(header.Time()) {
		return fmt.Errorf("block time must be strictly increasing")
	}

	// Check app hash
	if !bytes.Equal(header.AppHash, lastState.AppHash) {
		return fmt.Errorf("app hash mismatch")
	}

	return nil
}

// updateState saves the new state
func (e *Executor) updateState(ctx context.Context, newState types.State) error {
	if err := e.store.UpdateState(ctx, newState); err != nil {
		return err
	}

	e.SetLastState(newState)
	e.metrics.Height.Set(float64(newState.LastBlockHeight))

	return nil
}

// recordBlockMetrics records metrics for the produced block
func (e *Executor) recordBlockMetrics(data *types.Data) {
	e.metrics.NumTxs.Set(float64(len(data.Txs)))
	e.metrics.TotalTxs.Add(float64(len(data.Txs)))
	e.metrics.BlockSizeBytes.Set(float64(data.Size()))
	e.metrics.CommittedHeight.Set(float64(data.Metadata.Height))
}

// BatchData represents batch data from sequencer
type BatchData struct {
	*coresequencer.Batch
	time.Time
	Data [][]byte
}
