package block

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	goheader "github.com/celestiaorg/go-header"
	ds "github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	coreda "github.com/evstack/ev-node/core/da"
	coreexecutor "github.com/evstack/ev-node/core/execution"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/cache"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/rpc/server"
	"github.com/evstack/ev-node/pkg/signer"
	storepkg "github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

const (
	// defaultDABlockTime is used only if DABlockTime is not configured for manager
	defaultDABlockTime = 6 * time.Second

	// defaultBlockTime is used only if BlockTime is not configured for manager
	defaultBlockTime = 1 * time.Second

	// defaultLazyBlockTime is used only if LazyBlockTime is not configured for manager
	defaultLazyBlockTime = 60 * time.Second

	// defaultMempoolTTL is the number of blocks until transaction is dropped from mempool
	defaultMempoolTTL = 25

	// Key for storing namespace migration state in the store
	namespaceMigrationKey = "namespace_migration_completed"

	// Applies to the headerInCh and dataInCh, 20000 is a large enough number for headers per DA block.
	eventInChLength = 20000
)

var (
	// dataHashForEmptyTxs to be used while only syncing headers from DA and no p2p to get the Data for no txs scenarios, the syncing can proceed without getting stuck forever.
	dataHashForEmptyTxs = []byte{110, 52, 11, 156, 255, 179, 122, 152, 156, 165, 68, 230, 187, 120, 10, 44, 120, 144, 29, 63, 179, 55, 56, 118, 133, 17, 163, 6, 23, 175, 160, 29}
)

// publishBlockFunc defines the function signature for publishing a block.
// This allows for overriding the behavior in tests.
type publishBlockFunc func(ctx context.Context) error

// MetricsRecorder defines the interface for sequencers that support recording metrics.
// This interface is used to avoid duplication of the anonymous interface definition
// across multiple files in the block package.
type MetricsRecorder interface {
	RecordMetrics(gasPrice float64, blobSize uint64, statusCode coreda.StatusCode, numPendingBlocks uint64, includedBlockHeight uint64)
}

// NewHeightEvent is used to pass an header, data and its DA height to heightInCh
type NewHeightEvent struct {
	DAHeight uint64
	Header   *types.SignedHeader
	Data     *types.Data
}

// BatchData is used to pass batch, time and data (da.IDs) to BatchQueue
type BatchData struct {
	*coresequencer.Batch
	time.Time
	Data [][]byte
}

type broadcaster[T any] interface {
	WriteToStoreAndBroadcast(ctx context.Context, payload T) error
}

// Manager is responsible for aggregating transactions into blocks.
type Manager struct {
	lastState types.State
	// lastStateMtx is used by lastState
	lastStateMtx *sync.RWMutex
	store        storepkg.Store

	config  config.Config
	genesis genesis.Genesis

	signer signer.Signer

	daHeight *atomic.Uint64

	headerBroadcaster broadcaster[*types.SignedHeader]
	dataBroadcaster   broadcaster[*types.Data]

	heightInCh  chan NewHeightEvent
	headerStore goheader.Store[*types.SignedHeader]
	dataStore   goheader.Store[*types.Data]

	headerCache *cache.Cache[types.SignedHeader]
	dataCache   *cache.Cache[types.Data]

	// headerStoreCh is used to notify sync goroutine (HeaderStoreRetrieveLoop) that it needs to retrieve headers from headerStore
	headerStoreCh chan struct{}

	// dataStoreCh is used to notify sync goroutine (DataStoreRetrieveLoop) that it needs to retrieve data from dataStore
	dataStoreCh chan struct{}

	// retrieveCh is used to notify sync goroutine (RetrieveLoop) that it needs to retrieve data
	retrieveCh chan struct{}

	// daIncluderCh is used to notify sync goroutine (DAIncluderLoop) that it needs to set DA included height
	daIncluderCh chan struct{}

	logger zerolog.Logger

	// For usage by Lazy Aggregator mode
	txsAvailable bool

	pendingHeaders *PendingHeaders
	pendingData    *PendingData

	// for reporting metrics
	metrics *Metrics

	exec coreexecutor.Executor

	// daIncludedHeight is evolve height at which all blocks have been included
	// in the DA
	daIncludedHeight atomic.Uint64
	da               coreda.DA

	sequencer     coresequencer.Sequencer
	lastBatchData [][]byte

	// publishBlock is the function used to publish blocks. It defaults to
	// the manager's publishBlock method but can be overridden for testing.
	publishBlock publishBlockFunc

	// txNotifyCh is used to signal when new transactions are available
	txNotifyCh chan struct{}

	// aggregatorSignaturePayloadProvider is used to provide a signature payload for the header.
	// It is used to sign the header with the provided signer.
	aggregatorSignaturePayloadProvider types.AggregatorNodeSignatureBytesProvider

	// syncNodeSignaturePayloadProvider is used to provide a signature payload for the header.
	// It is used to sign the header with the provided signer.
	syncNodeSignaturePayloadProvider types.SyncNodeSignatureBytesProvider

	// validatorHasherProvider is used to provide the validator hash for the header.
	// It is used to set the validator hash in the header.
	validatorHasherProvider types.ValidatorHasherProvider

	// namespaceMigrationCompleted tracks whether we have completed the migration
	// from legacy namespace to separate header/data namespaces
	namespaceMigrationCompleted *atomic.Bool
}

// getInitialState tries to load lastState from Store, and if it's not available it reads genesis.
func getInitialState(ctx context.Context, genesis genesis.Genesis, signer signer.Signer, store storepkg.Store, exec coreexecutor.Executor, logger zerolog.Logger, managerOpts ManagerOptions) (types.State, error) {
	// Load the state from store.
	s, err := store.GetState(ctx)

	if errors.Is(err, ds.ErrNotFound) {
		logger.Info().Msg("No state found in store, initializing new state")

		// If the user is starting a fresh chain (or hard-forking), we assume the stored state is empty.
		// TODO(tzdybal): handle max bytes
		stateRoot, _, err := exec.InitChain(ctx, genesis.GenesisDAStartTime, genesis.InitialHeight, genesis.ChainID)
		if err != nil {
			return types.State{}, fmt.Errorf("failed to initialize chain: %w", err)
		}

		// Initialize genesis block explicitly
		header := types.Header{
			AppHash:         stateRoot,
			DataHash:        new(types.Data).DACommitment(),
			ProposerAddress: genesis.ProposerAddress,
			BaseHeader: types.BaseHeader{
				ChainID: genesis.ChainID,
				Height:  genesis.InitialHeight,
				Time:    uint64(genesis.GenesisDAStartTime.UnixNano()),
			},
		}

		var (
			data      = &types.Data{}
			signature types.Signature
			pubKey    crypto.PubKey
		)

		// The signer is only provided in aggregator nodes. This enables the creation of a signed genesis header,
		// which includes a public key and a cryptographic signature for the header.
		// In a full node (non-aggregator), the signer will be nil, and only an unsigned genesis header will be initialized locally.
		if signer != nil {
			pubKey, err = signer.GetPublic()
			if err != nil {
				return types.State{}, fmt.Errorf("failed to get public key: %w", err)
			}

			bz, err := managerOpts.AggregatorNodeSignatureBytesProvider(&header)
			if err != nil {
				return types.State{}, fmt.Errorf("failed to get signature payload: %w", err)
			}

			signature, err = signer.Sign(bz)
			if err != nil {
				return types.State{}, fmt.Errorf("failed to get header signature: %w", err)
			}
		}

		genesisHeader := &types.SignedHeader{
			Header: header,
			Signer: types.Signer{
				PubKey:  pubKey,
				Address: genesis.ProposerAddress,
			},
			Signature: signature,
		}

		err = store.SaveBlockData(ctx, genesisHeader, data, &signature)
		if err != nil {
			return types.State{}, fmt.Errorf("failed to save genesis block: %w", err)
		}

		s := types.State{
			Version:         types.Version{},
			ChainID:         genesis.ChainID,
			InitialHeight:   genesis.InitialHeight,
			LastBlockHeight: genesis.InitialHeight - 1,
			LastBlockTime:   genesis.GenesisDAStartTime,
			AppHash:         stateRoot,
			DAHeight:        0,
		}
		return s, nil
	} else if err != nil {
		logger.Error().Err(err).Msg("error while getting state")
		return types.State{}, err
	} else {
		// Perform a sanity-check to stop the user from
		// using a higher genesis than the last stored state.
		// if they meant to hard-fork, they should have cleared the stored State
		if uint64(genesis.InitialHeight) > s.LastBlockHeight { //nolint:unconvert
			return types.State{}, fmt.Errorf("genesis.InitialHeight (%d) is greater than last stored state's LastBlockHeight (%d)", genesis.InitialHeight, s.LastBlockHeight)
		}
	}

	return s, nil
}

// ManagerOptions defines the options for creating a new block Manager.
type ManagerOptions struct {
	AggregatorNodeSignatureBytesProvider types.AggregatorNodeSignatureBytesProvider
	SyncNodeSignatureBytesProvider       types.SyncNodeSignatureBytesProvider
	ValidatorHasherProvider              types.ValidatorHasherProvider
}

func (opts *ManagerOptions) Validate() error {
	if opts.AggregatorNodeSignatureBytesProvider == nil {
		return fmt.Errorf("aggregator node signature bytes provider cannot be nil")
	}

	if opts.SyncNodeSignatureBytesProvider == nil {
		return fmt.Errorf("sync node signature bytes provider cannot be nil")
	}

	if opts.ValidatorHasherProvider == nil {
		return fmt.Errorf("validator hasher provider cannot be nil")
	}

	return nil
}

// DefaultManagerOptions returns the default options for creating a new block Manager.
func DefaultManagerOptions() ManagerOptions {
	return ManagerOptions{
		AggregatorNodeSignatureBytesProvider: types.DefaultAggregatorNodeSignatureBytesProvider,
		SyncNodeSignatureBytesProvider:       types.DefaultSyncNodeSignatureBytesProvider,
		ValidatorHasherProvider:              types.DefaultValidatorHasherProvider,
	}
}

// NewManager creates new block Manager.
func NewManager(
	ctx context.Context,
	signer signer.Signer,
	config config.Config,
	genesis genesis.Genesis,
	store storepkg.Store,
	exec coreexecutor.Executor,
	sequencer coresequencer.Sequencer,
	da coreda.DA,
	logger zerolog.Logger,
	headerStore goheader.Store[*types.SignedHeader],
	dataStore goheader.Store[*types.Data],
	headerBroadcaster broadcaster[*types.SignedHeader],
	dataBroadcaster broadcaster[*types.Data],
	seqMetrics *Metrics,
	managerOpts ManagerOptions,
) (*Manager, error) {
	s, err := getInitialState(ctx, genesis, signer, store, exec, logger, managerOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to get initial state: %w", err)
	}

	// set block height in store
	if err = store.SetHeight(ctx, s.LastBlockHeight); err != nil {
		return nil, err
	}

	if s.DAHeight < config.DA.StartHeight {
		s.DAHeight = config.DA.StartHeight
	}

	if config.DA.BlockTime.Duration == 0 {
		logger.Info().Dur("DABlockTime", defaultDABlockTime).Msg("using default DA block time")
		config.DA.BlockTime.Duration = defaultDABlockTime
	}

	if config.Node.BlockTime.Duration == 0 {
		logger.Info().Dur("BlockTime", defaultBlockTime).Msg("using default block time")
		config.Node.BlockTime.Duration = defaultBlockTime
	}

	if config.Node.LazyBlockInterval.Duration == 0 {
		logger.Info().Dur("LazyBlockTime", defaultLazyBlockTime).Msg("using default lazy block time")
		config.Node.LazyBlockInterval.Duration = defaultLazyBlockTime
	}

	if config.DA.MempoolTTL == 0 {
		logger.Info().Int("MempoolTTL", defaultMempoolTTL).Msg("using default mempool ttl")
		config.DA.MempoolTTL = defaultMempoolTTL
	}

	pendingHeaders, err := NewPendingHeaders(store, logger)
	if err != nil {
		return nil, err
	}

	pendingData, err := NewPendingData(store, logger)
	if err != nil {
		return nil, err
	}

	// If lastBatchHash is not set, retrieve the last batch hash from store
	lastBatchDataBytes, err := store.GetMetadata(ctx, storepkg.LastBatchDataKey)
	if err != nil && s.LastBlockHeight > 0 {
		logger.Error().Err(err).Msg("error while retrieving last batch hash")
	}

	lastBatchData, err := bytesToBatchData(lastBatchDataBytes)
	if err != nil {
		logger.Error().Err(err).Msg("error while converting last batch hash")
	}

	daH := atomic.Uint64{}
	daH.Store(s.DAHeight)

	m := &Manager{
		signer:            signer,
		config:            config,
		genesis:           genesis,
		lastState:         s,
		store:             store,
		daHeight:          &daH,
		headerBroadcaster: headerBroadcaster,
		dataBroadcaster:   dataBroadcaster,
		// channels are buffered to avoid blocking on input/output operations, buffer sizes are arbitrary
		heightInCh:                         make(chan NewHeightEvent, eventInChLength),
		headerStoreCh:                      make(chan struct{}, 1),
		dataStoreCh:                        make(chan struct{}, 1),
		headerStore:                        headerStore,
		dataStore:                          dataStore,
		lastStateMtx:                       new(sync.RWMutex),
		lastBatchData:                      lastBatchData,
		headerCache:                        cache.NewCache[types.SignedHeader](),
		dataCache:                          cache.NewCache[types.Data](),
		retrieveCh:                         make(chan struct{}, 1),
		daIncluderCh:                       make(chan struct{}, 1),
		logger:                             logger,
		txsAvailable:                       false,
		pendingHeaders:                     pendingHeaders,
		pendingData:                        pendingData,
		metrics:                            seqMetrics,
		sequencer:                          sequencer,
		exec:                               exec,
		da:                                 da,
		txNotifyCh:                         make(chan struct{}, 1), // Non-blocking channel
		aggregatorSignaturePayloadProvider: managerOpts.AggregatorNodeSignatureBytesProvider,
		syncNodeSignaturePayloadProvider:   managerOpts.SyncNodeSignatureBytesProvider,
		validatorHasherProvider:            managerOpts.ValidatorHasherProvider,
		namespaceMigrationCompleted:        &atomic.Bool{},
	}

	// initialize da included height
	if height, err := m.store.GetMetadata(ctx, storepkg.DAIncludedHeightKey); err == nil && len(height) == 8 {
		m.daIncludedHeight.Store(binary.LittleEndian.Uint64(height))
	}

	// initialize namespace migration state
	if migrationData, err := m.store.GetMetadata(ctx, namespaceMigrationKey); err == nil && len(migrationData) > 0 {
		m.namespaceMigrationCompleted.Store(migrationData[0] == 1)
	}

	// Set the default publishBlock implementation
	m.publishBlock = m.publishBlockInternal

	// fetch caches from disks
	if err := m.LoadCache(); err != nil {
		return nil, fmt.Errorf("failed to load cache: %w", err)
	}

	// Initialize DA visualization server if enabled
	if config.RPC.EnableDAVisualization {
		daVisualizationServer := server.NewDAVisualizationServer(da, logger.With().Str("module", "da_visualization").Logger(), config.Node.Aggregator)
		server.SetDAVisualizationServer(daVisualizationServer)
		logger.Info().Msg("DA visualization server enabled")
	} else {
		// Ensure the global server is nil when disabled
		server.SetDAVisualizationServer(nil)
	}

	return m, nil
}

// setNamespaceMigrationCompleted marks the namespace migration as completed and persists it to disk
func (m *Manager) setNamespaceMigrationCompleted(ctx context.Context) error {
	m.namespaceMigrationCompleted.Store(true)
	return m.store.SetMetadata(ctx, namespaceMigrationKey, []byte{1})
}

// loadNamespaceMigrationState loads the namespace migration state from persistent storage
func (m *Manager) loadNamespaceMigrationState(ctx context.Context) (bool, error) {
	migrationData, err := m.store.GetMetadata(ctx, namespaceMigrationKey)
	if err != nil {
		if errors.Is(err, ds.ErrNotFound) {
			return false, nil // Migration not completed
		}
		return false, fmt.Errorf("failed to load migration state: %w", err)
	}
	return len(migrationData) > 0 && migrationData[0] == 1, nil
}

// PendingHeaders returns the pending headers.
func (m *Manager) PendingHeaders() *PendingHeaders {
	return m.pendingHeaders
}

// SeqClient returns the grpc sequencing client.
func (m *Manager) SeqClient() coresequencer.Sequencer {
	return m.sequencer
}

// GetLastState returns the last recorded state.
func (m *Manager) GetLastState() types.State {
	m.lastStateMtx.RLock()
	defer m.lastStateMtx.RUnlock()
	return m.lastState
}

// GetDAIncludedHeight returns the height at which all blocks have been
// included in the DA
func (m *Manager) GetDAIncludedHeight() uint64 {
	return m.daIncludedHeight.Load()
}

// isProposer returns whether or not the manager is a proposer
func isProposer(signer signer.Signer, pubkey crypto.PubKey) (bool, error) {
	if signer == nil {
		return false, nil
	}
	pubKey, err := signer.GetPublic()
	if err != nil {
		return false, err
	}
	if pubKey == nil {
		return false, errors.New("public key is nil")
	}
	if !pubKey.Equals(pubkey) {
		return false, nil
	}
	return true, nil
}

// SetLastState is used to set lastState used by Manager.
func (m *Manager) SetLastState(state types.State) {
	m.lastStateMtx.Lock()
	defer m.lastStateMtx.Unlock()
	m.lastState = state
}

// GetStoreHeight returns the manager's store height
func (m *Manager) GetStoreHeight(ctx context.Context) (uint64, error) {
	return m.store.Height(ctx)
}

// IsBlockHashSeen returns true if the block with the given hash has been seen.
func (m *Manager) IsBlockHashSeen(blockHash string) bool {
	return m.headerCache.IsSeen(blockHash)
}

// IsDAIncluded returns true if the block with the given hash has been seen on DA.
// TODO(tac0turtle): should we use this for pending header system to verify how far ahead a chain is?
func (m *Manager) IsDAIncluded(ctx context.Context, height uint64) (bool, error) {
	syncedHeight, err := m.store.Height(ctx)
	if err != nil {
		return false, err
	}
	if syncedHeight < height {
		return false, nil
	}
	header, data, err := m.store.GetBlockData(ctx, height)
	if err != nil {
		return false, err
	}
	headerHash, dataHash := header.Hash(), data.DACommitment()
	isIncluded := m.headerCache.IsDAIncluded(headerHash.String()) && (bytes.Equal(dataHash, dataHashForEmptyTxs) || m.dataCache.IsDAIncluded(dataHash.String()))
	return isIncluded, nil
}

// SetSequencerHeightToDAHeight stores the mapping from a Evolve block height to the corresponding
// DA (Data Availability) layer heights where the block's header and data were included.
// This mapping is persisted in the store metadata and is used to track which DA heights
// contain the block components for a given Evolve height.
//
// For blocks with empty transactions, both header and data use the same DA height since
// empty transaction data is not actually published to the DA layer.
func (m *Manager) SetSequencerHeightToDAHeight(ctx context.Context, height uint64) error {
	header, data, err := m.store.GetBlockData(ctx, height)
	if err != nil {
		return err
	}
	headerHash, dataHash := header.Hash(), data.DACommitment()
	headerHeightBytes := make([]byte, 8)
	daHeightForHeader, ok := m.headerCache.GetDAIncludedHeight(headerHash.String())
	if !ok {
		return fmt.Errorf("header hash %s not found in cache", headerHash)
	}
	binary.LittleEndian.PutUint64(headerHeightBytes, daHeightForHeader)
	if err := m.store.SetMetadata(ctx, fmt.Sprintf("%s/%d/h", storepkg.HeightToDAHeightKey, height), headerHeightBytes); err != nil {
		return err
	}
	dataHeightBytes := make([]byte, 8)
	// For empty transactions, use the same DA height as the header
	if bytes.Equal(dataHash, dataHashForEmptyTxs) {
		binary.LittleEndian.PutUint64(dataHeightBytes, daHeightForHeader)
	} else {
		daHeightForData, ok := m.dataCache.GetDAIncludedHeight(dataHash.String())
		if !ok {
			return fmt.Errorf("data hash %s not found in cache", dataHash.String())
		}
		binary.LittleEndian.PutUint64(dataHeightBytes, daHeightForData)
	}
	if err := m.store.SetMetadata(ctx, fmt.Sprintf("%s/%d/d", storepkg.HeightToDAHeightKey, height), dataHeightBytes); err != nil {
		return err
	}
	return nil
}

// GetExecutor returns the executor used by the manager.
//
// Note: this is a temporary method to allow testing the manager.
// It will be removed once the manager is fully integrated with the execution client.
// TODO(tac0turtle): remove
func (m *Manager) GetExecutor() coreexecutor.Executor {
	return m.exec
}

func (m *Manager) retrieveBatch(ctx context.Context) (*BatchData, error) {
	m.logger.Debug().Str("chainID", m.genesis.ChainID).Interface("lastBatchData", m.lastBatchData).Msg("Attempting to retrieve next batch")

	req := coresequencer.GetNextBatchRequest{
		Id:            []byte(m.genesis.ChainID),
		LastBatchData: m.lastBatchData,
	}

	res, err := m.sequencer.GetNextBatch(ctx, req)
	if err != nil {
		return nil, err
	}

	if res != nil && res.Batch != nil {
		m.logger.Debug().Int("txCount", len(res.Batch.Transactions)).Time("timestamp", res.Timestamp).Msg("Retrieved batch")

		var errRetrieveBatch error
		// Even if there are no transactions, return the batch with timestamp
		// This allows empty blocks to maintain proper timing
		if len(res.Batch.Transactions) == 0 {
			errRetrieveBatch = ErrNoBatch
		}
		// Even if there are no transactions, update lastBatchData so we don't
		// repeatedly emit the same empty batch, and persist it to metadata.
		if err := m.store.SetMetadata(ctx, storepkg.LastBatchDataKey, convertBatchDataToBytes(res.BatchData)); err != nil {
			m.logger.Error().Err(err).Msg("error while setting last batch hash")
		}
		m.lastBatchData = res.BatchData
		return &BatchData{Batch: res.Batch, Time: res.Timestamp, Data: res.BatchData}, errRetrieveBatch
	}
	return nil, ErrNoBatch
}

// isUsingExpectedSingleSequencer checks if the header is using the expected single sequencer.
func (m *Manager) isUsingExpectedSingleSequencer(proposerAddress []byte) (bool, error) {
	if !bytes.Equal(m.genesis.ProposerAddress, proposerAddress) {
		return false, fmt.Errorf("proposer address is not the same as the genesis proposer address %x != %x", proposerAddress, m.genesis.ProposerAddress)
	}

	return true, nil
}

// publishBlockInternal is the internal implementation for publishing a block.
// It's assigned to the publishBlock field by default.
// Any error will be returned, unless the error is due to a publishing error.
func (m *Manager) publishBlockInternal(ctx context.Context) error {
	// Start timing block production
	timer := NewMetricsTimer("block_production", m.metrics)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if m.config.Node.MaxPendingHeadersAndData != 0 && (m.pendingHeaders.numPendingHeaders() >= m.config.Node.MaxPendingHeadersAndData || m.pendingData.numPendingData() >= m.config.Node.MaxPendingHeadersAndData) {
		m.logger.Warn().Uint64("pending_headers", m.pendingHeaders.numPendingHeaders()).Uint64("pending_data", m.pendingData.numPendingData()).Uint64("limit", m.config.Node.MaxPendingHeadersAndData).Msg("refusing to create block: pending headers or data reached limit")
		return nil
	}

	var (
		lastSignature  *types.Signature
		lastHeaderHash types.Hash
		lastDataHash   types.Hash
		lastHeaderTime time.Time
		err            error
	)

	height, err := m.store.Height(ctx)
	if err != nil {
		return fmt.Errorf("error while getting store height: %w", err)
	}

	newHeight := height + 1
	// this is a special case, when first block is produced - there is no previous commit
	if newHeight <= m.genesis.InitialHeight {
		// Special handling for genesis block
		lastSignature = &types.Signature{}
	} else {
		lastSignature, err = m.store.GetSignature(ctx, height)
		if err != nil {
			return fmt.Errorf("error while loading last commit: %w, height: %d", err, height)
		}
		lastHeader, lastData, err := m.store.GetBlockData(ctx, height)
		if err != nil {
			return fmt.Errorf("error while loading last block: %w, height: %d", err, height)
		}
		lastHeaderHash = lastHeader.Hash()
		lastDataHash = lastData.Hash()
		lastHeaderTime = lastHeader.Time()
	}

	var (
		header    *types.SignedHeader
		data      *types.Data
		signature types.Signature
	)

	// Check if there's an already stored block at a newer height
	// If there is use that instead of creating a new block
	pendingHeader, pendingData, err := m.store.GetBlockData(ctx, newHeight)
	if err == nil {
		m.logger.Info().Uint64("height", newHeight).Msg("using pending block")
		header = pendingHeader
		data = pendingData
	} else {
		batchData, err := m.retrieveBatch(ctx)
		if err != nil {
			if errors.Is(err, ErrNoBatch) {
				if batchData == nil {
					m.logger.Info().Msg("no batch retrieved from sequencer, skipping block production")
					return nil
				}
				m.logger.Info().Uint64("height", newHeight).Msg("creating empty block")
			} else {
				m.logger.Warn().Err(err).Msg("failed to get transactions from batch")
				return nil
			}
		} else {
			if batchData.Before(lastHeaderTime) {
				return fmt.Errorf("timestamp is not monotonically increasing: %s < %s", batchData.Time, m.getLastBlockTime())
			}
			m.logger.Info().Uint64("height", newHeight).Int("num_tx", len(batchData.Transactions)).Msg("creating and publishing block")
		}

		header, data, err = m.createBlock(ctx, newHeight, lastSignature, lastHeaderHash, batchData)
		if err != nil {
			return err
		}

		if err = m.store.SaveBlockData(ctx, header, data, &signature); err != nil { // saved early for crash recovery, will be overwritten later with the final signature
			return fmt.Errorf("failed to save block: %w", err)
		}
	}

	newState, err := m.applyBlock(ctx, header.Header, data)
	if err != nil {
		return fmt.Errorf("error applying block: %w", err)
	}

	// append metadata to Data before validating and saving
	data.Metadata = &types.Metadata{
		ChainID:      header.ChainID(),
		Height:       header.Height(),
		Time:         header.BaseHeader.Time,
		LastDataHash: lastDataHash,
	}

	// we sign the header after executing the block, as a signature payload provider could depend on the block's data
	signature, err = m.signHeader(header.Header)
	if err != nil {
		return err
	}

	// set the signature to current block's signed header
	header.Signature = signature

	// set the custom verifier to ensure proper signature validation
	header.SetCustomVerifierForAggregator(m.aggregatorSignaturePayloadProvider)

	// Validate the created block before storing
	if err := m.Validate(ctx, header, data); err != nil {
		return fmt.Errorf("failed to validate block: %w", err)
	}

	headerHash := header.Hash().String()
	m.headerCache.SetSeen(headerHash)

	// SaveBlock commits the DB tx
	err = m.store.SaveBlockData(ctx, header, data, &signature)
	if err != nil {
		return fmt.Errorf("failed to save block: %w", err)
	}

	// Update the store height before submitting to the DA layer but after committing to the DB
	headerHeight := header.Height()
	if err = m.store.SetHeight(ctx, headerHeight); err != nil {
		return err
	}

	newState.DAHeight = m.daHeight.Load()
	// After this call m.lastState is the NEW state returned from ApplyBlock
	// updateState also commits the DB tx
	if err = m.updateState(ctx, newState); err != nil {
		return fmt.Errorf("failed to update state: %w", err)
	}

	m.recordMetrics(data)

	// Record extended block production metrics
	productionTime := time.Since(timer.start)
	isLazy := m.config.Node.LazyMode
	m.recordBlockProductionMetrics(len(data.Txs), isLazy, productionTime)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error { return m.headerBroadcaster.WriteToStoreAndBroadcast(ctx, header) })
	g.Go(func() error { return m.dataBroadcaster.WriteToStoreAndBroadcast(ctx, data) })
	if err := g.Wait(); err != nil {
		return err
	}

	m.logger.Debug().Str("proposer", hex.EncodeToString(header.ProposerAddress)).Uint64("height", headerHeight).Msg("successfully proposed header")
	return nil
}

func (m *Manager) recordMetrics(data *types.Data) {
	m.metrics.NumTxs.Set(float64(len(data.Txs)))
	m.metrics.TotalTxs.Add(float64(len(data.Txs)))
	m.metrics.BlockSizeBytes.Set(float64(data.Size()))
	m.metrics.CommittedHeight.Set(float64(data.Metadata.Height))
}

func (m *Manager) getLastBlockTime() time.Time {
	m.lastStateMtx.RLock()
	defer m.lastStateMtx.RUnlock()
	return m.lastState.LastBlockTime
}

func (m *Manager) createBlock(ctx context.Context, height uint64, lastSignature *types.Signature, lastHeaderHash types.Hash, batchData *BatchData) (*types.SignedHeader, *types.Data, error) {
	m.lastStateMtx.RLock()
	defer m.lastStateMtx.RUnlock()
	return m.execCreateBlock(ctx, height, lastSignature, lastHeaderHash, m.lastState, batchData)
}

func (m *Manager) applyBlock(ctx context.Context, header types.Header, data *types.Data) (types.State, error) {
	m.lastStateMtx.RLock()
	defer m.lastStateMtx.RUnlock()
	return m.execApplyBlock(ctx, m.lastState, header, data)
}

func (m *Manager) Validate(ctx context.Context, header *types.SignedHeader, data *types.Data) error {
	m.lastStateMtx.RLock()
	defer m.lastStateMtx.RUnlock()
	return m.execValidate(m.lastState, header, data)
}

// execValidate validates a pair of header and data against the last state
func (m *Manager) execValidate(lastState types.State, header *types.SignedHeader, data *types.Data) error {
	// Validate the basic structure of the header
	if err := header.ValidateBasic(); err != nil {
		return fmt.Errorf("invalid header: %w", err)
	}

	// Validate the header against the data
	if err := types.Validate(header, data); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Ensure the header's Chain ID matches the expected state
	if header.ChainID() != lastState.ChainID {
		return fmt.Errorf("chain ID mismatch: expected %s, got %s", lastState.ChainID, header.ChainID())
	}

	// Check that the header's height is the expected next height
	expectedHeight := lastState.LastBlockHeight + 1
	if header.Height() != expectedHeight {
		return fmt.Errorf("invalid height: expected %d, got %d", expectedHeight, header.Height())
	}

	// Verify that the header's timestamp is strictly greater than the last block's time
	headerTime := header.Time()
	if header.Height() > 1 && lastState.LastBlockTime.After(headerTime) {
		return fmt.Errorf("block time must be strictly increasing: got %v, last block time was %v",
			headerTime, lastState.LastBlockTime)
	}

	// AppHash should match the last state's AppHash
	if !bytes.Equal(header.AppHash, lastState.AppHash) {
		return fmt.Errorf("appHash mismatch in delayed execution mode: expected %x, got %x at height %d", lastState.AppHash, header.AppHash, header.Height())
	}

	return nil
}

func (m *Manager) execCreateBlock(_ context.Context, height uint64, lastSignature *types.Signature, lastHeaderHash types.Hash, _ types.State, batchData *BatchData) (*types.SignedHeader, *types.Data, error) {
	// Use when batchData is set to data IDs from the DA layer
	// batchDataIDs := convertBatchDataToBytes(batchData.Data)

	if m.signer == nil {
		return nil, nil, fmt.Errorf("signer is nil; cannot create block")
	}

	key, err := m.signer.GetPublic()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get proposer public key: %w", err)
	}

	// check that the proposer address is the same as the genesis proposer address
	address, err := m.signer.GetAddress()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get proposer address: %w", err)
	}

	if _, err := m.isUsingExpectedSingleSequencer(address); err != nil {
		return nil, nil, err
	}

	// determine if this is an empty block
	isEmpty := batchData.Batch == nil || len(batchData.Transactions) == 0

	// build validator hash
	validatorHash, err := m.validatorHasherProvider(m.genesis.ProposerAddress, key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get validator hash: %w", err)
	}

	header := &types.SignedHeader{
		Header: types.Header{
			Version: types.Version{
				Block: m.lastState.Version.Block,
				App:   m.lastState.Version.App,
			},
			BaseHeader: types.BaseHeader{
				ChainID: m.lastState.ChainID,
				Height:  height,
				Time:    uint64(batchData.UnixNano()),
			},
			LastHeaderHash: lastHeaderHash,
			// DataHash is set at the end of the function
			ConsensusHash:   make(types.Hash, 32),
			AppHash:         m.lastState.AppHash,
			ProposerAddress: m.genesis.ProposerAddress,
			ValidatorHash:   validatorHash,
		},
		Signature: *lastSignature,
		Signer: types.Signer{
			PubKey:  key,
			Address: m.genesis.ProposerAddress,
		},
	}

	// Create block data with appropriate transactions
	blockData := &types.Data{
		Txs: make(types.Txs, 0), // Start with empty transaction list
	}

	// Only add transactions if this is not an empty block
	if !isEmpty {
		blockData.Txs = make(types.Txs, len(batchData.Transactions))
		for i := range batchData.Transactions {
			blockData.Txs[i] = types.Tx(batchData.Transactions[i])
		}
		header.DataHash = blockData.DACommitment()
	} else {
		header.DataHash = dataHashForEmptyTxs
	}

	return header, blockData, nil
}

func (m *Manager) execApplyBlock(ctx context.Context, lastState types.State, header types.Header, data *types.Data) (types.State, error) {
	rawTxs := make([][]byte, len(data.Txs))
	for i := range data.Txs {
		rawTxs[i] = data.Txs[i]
	}

	ctx = context.WithValue(ctx, types.HeaderContextKey, header)
	newStateRoot, _, err := m.exec.ExecuteTxs(ctx, rawTxs, header.Height(), header.Time(), lastState.AppHash)
	if err != nil {
		return types.State{}, fmt.Errorf("failed to execute transactions: %w", err)
	}

	s, err := lastState.NextState(header, newStateRoot)
	if err != nil {
		return types.State{}, err
	}

	return s, nil
}

func convertBatchDataToBytes(batchData [][]byte) []byte {
	// If batchData is nil or empty, return an empty byte slice
	if len(batchData) == 0 {
		return []byte{}
	}

	// For a single item, we still need to length-prefix it for consistency
	// First, calculate the total size needed
	// Format: 4 bytes (length) + data for each entry
	totalSize := 0
	for _, data := range batchData {
		totalSize += 4 + len(data) // 4 bytes for length prefix + data length
	}

	// Allocate buffer with calculated capacity
	result := make([]byte, 0, totalSize)

	// Add length-prefixed data
	for _, data := range batchData {
		// Encode length as 4-byte big-endian integer
		lengthBytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(lengthBytes, uint32(len(data)))

		// Append length prefix
		result = append(result, lengthBytes...)

		// Append actual data
		result = append(result, data...)
	}

	return result
}

// bytesToBatchData converts a length-prefixed byte array back to a slice of byte slices
func bytesToBatchData(data []byte) ([][]byte, error) {
	if len(data) == 0 {
		return [][]byte{}, nil
	}

	var result [][]byte
	offset := 0

	for offset < len(data) {
		// Check if we have at least 4 bytes for the length prefix
		if offset+4 > len(data) {
			return nil, fmt.Errorf("corrupted data: insufficient bytes for length prefix at offset %d", offset)
		}

		// Read the length prefix
		length := binary.LittleEndian.Uint32(data[offset : offset+4])
		offset += 4

		// Check if we have enough bytes for the data
		if offset+int(length) > len(data) {
			return nil, fmt.Errorf("corrupted data: insufficient bytes for entry of length %d at offset %d", length, offset)
		}

		// Extract the data entry
		entry := make([]byte, length)
		copy(entry, data[offset:offset+int(length)])
		result = append(result, entry)

		// Move to the next entry
		offset += int(length)
	}

	return result, nil
}

func (m *Manager) signHeader(header types.Header) (types.Signature, error) {
	b, err := m.aggregatorSignaturePayloadProvider(&header)
	if err != nil {
		return nil, err
	}

	if m.signer == nil {
		return nil, fmt.Errorf("signer is nil; cannot sign header")
	}

	return m.signer.Sign(b)
}

func (m *Manager) getDataSignature(data *types.Data) (types.Signature, error) {
	dataBz, err := data.MarshalBinary()
	if err != nil {
		return nil, err
	}

	if m.signer == nil {
		return nil, fmt.Errorf("signer is nil; cannot sign data")
	}
	return m.signer.Sign(dataBz)
}

// NotifyNewTransactions signals that new transactions are available for processing
// This method will be called by the Reaper when it receives new transactions
func (m *Manager) NotifyNewTransactions() {
	// Non-blocking send to avoid slowing down the transaction submission path
	select {
	case m.txNotifyCh <- struct{}{}:
		// Successfully sent notification
	default:
		// Channel buffer is full, which means a notification is already pending
		// This is fine, as we just need to trigger one block production
	}
}

var (
	cacheDir       = "cache"
	headerCacheDir = filepath.Join(cacheDir, "header")
	dataCacheDir   = filepath.Join(cacheDir, "data")
)

// HeaderCache returns the headerCache used by the manager.
func (m *Manager) HeaderCache() *cache.Cache[types.SignedHeader] {
	return m.headerCache
}

// DataCache returns the dataCache used by the manager.
func (m *Manager) DataCache() *cache.Cache[types.Data] {
	return m.dataCache
}

// LoadCache loads the header and data caches from disk.
func (m *Manager) LoadCache() error {
	gob.Register(&types.SignedHeader{})
	gob.Register(&types.Data{})

	cfgDir := filepath.Join(m.config.RootDir, "data")

	if err := m.headerCache.LoadFromDisk(filepath.Join(cfgDir, headerCacheDir)); err != nil {
		return fmt.Errorf("failed to load header cache from disk: %w", err)
	}

	if err := m.dataCache.LoadFromDisk(filepath.Join(cfgDir, dataCacheDir)); err != nil {
		return fmt.Errorf("failed to load data cache from disk: %w", err)
	}

	return nil
}

// SaveCache saves the header and data caches to disk.
func (m *Manager) SaveCache() error {
	cfgDir := filepath.Join(m.config.RootDir, "data")

	if err := m.headerCache.SaveToDisk(filepath.Join(cfgDir, headerCacheDir)); err != nil {
		return fmt.Errorf("failed to save header cache to disk: %w", err)
	}

	if err := m.dataCache.SaveToDisk(filepath.Join(cfgDir, dataCacheDir)); err != nil {
		return fmt.Errorf("failed to save data cache to disk: %w", err)
	}
	return nil
}

// isValidSignedData returns true if the data signature is valid for the expected sequencer.
func (m *Manager) isValidSignedData(signedData *types.SignedData) bool {
	if signedData == nil || signedData.Txs == nil {
		return false
	}

	if ok, _ := m.isUsingExpectedSingleSequencer(signedData.Signer.Address); !ok {
		return false
	}

	dataBytes, err := signedData.Data.MarshalBinary()
	if err != nil {
		return false
	}

	valid, err := signedData.Signer.PubKey.Verify(dataBytes, signedData.Signature)
	return err == nil && valid
}
