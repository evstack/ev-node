package cache

import (
	"context"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

// isNotFound is a convenience alias for store.IsNotFound.
var isNotFound = store.IsNotFound

const (
	// HeaderDAIncludedPrefix is the store key prefix for header DA inclusion tracking.
	HeaderDAIncludedPrefix = "cache/header-da-included/"

	// DataDAIncludedPrefix is the store key prefix for data DA inclusion tracking.
	DataDAIncludedPrefix = "cache/data-da-included/"

	// DefaultTxCacheRetention is the default time to keep transaction hashes in cache.
	DefaultTxCacheRetention = 24 * time.Hour
)

// CacheManager provides thread-safe cache operations for tracking seen blocks
// and DA inclusion status.
type CacheManager interface {
	DaHeight() uint64

	// Header operations
	IsHeaderSeen(hash string) bool
	SetHeaderSeen(hash string, blockHeight uint64)
	GetHeaderDAIncludedByHash(hash string) (uint64, bool)
	GetHeaderDAIncludedByHeight(blockHeight uint64) (uint64, bool)
	SetHeaderDAIncluded(hash string, daHeight uint64, blockHeight uint64)
	RemoveHeaderDAIncluded(hash string)

	// Data operations
	IsDataSeen(hash string) bool
	SetDataSeen(hash string, blockHeight uint64)
	GetDataDAIncludedByHash(daCommitmentHash string) (uint64, bool)
	GetDataDAIncludedByHeight(blockHeight uint64) (uint64, bool)
	SetDataDAIncluded(daCommitmentHash string, daHeight uint64, blockHeight uint64)
	RemoveDataDAIncluded(hash string)

	// Transaction operations
	IsTxSeen(hash string) bool
	SetTxSeen(hash string)
	CleanupOldTxs(olderThan time.Duration) int

	// Pending events syncing coordination
	GetNextPendingEvent(blockHeight uint64) *common.DAHeightEvent
	SetPendingEvent(blockHeight uint64, event *common.DAHeightEvent)

	// Store operations
	SaveToStore() error
	RestoreFromStore() error

	// Cleanup operations
	DeleteHeight(blockHeight uint64)
}

// PendingManager provides operations for managing pending headers and data.
type PendingManager interface {
	GetPendingHeaders(ctx context.Context) ([]*types.SignedHeader, [][]byte, error)
	GetPendingData(ctx context.Context) ([]*types.SignedData, [][]byte, error)
	SetLastSubmittedHeaderHeight(ctx context.Context, height uint64)
	GetLastSubmittedHeaderHeight() uint64
	SetLastSubmittedDataHeight(ctx context.Context, height uint64)
	GetLastSubmittedDataHeight() uint64
	NumPendingHeaders() uint64
	NumPendingData() uint64
}

// Manager combines CacheManager and PendingManager.
type Manager interface {
	CacheManager
	PendingManager
}

var _ Manager = (*implementation)(nil)

type implementation struct {
	headerCache        *Cache[types.SignedHeader]
	dataCache          *Cache[types.Data]
	txCache            *Cache[struct{}]
	txTimestamps       *sync.Map // map[string]time.Time
	pendingEventsCache *Cache[common.DAHeightEvent]
	pendingHeaders     *PendingHeaders
	pendingData        *PendingData
	store              store.Store
	config             config.Config
	logger             zerolog.Logger
}

// NewManager creates a new Manager, restoring or clearing persisted state as configured.
func NewManager(cfg config.Config, st store.Store, logger zerolog.Logger) (Manager, error) {
	headerCache := NewCache[types.SignedHeader](st, HeaderDAIncludedPrefix)
	dataCache := NewCache[types.Data](st, DataDAIncludedPrefix)
	txCache := NewCache[struct{}](nil, "")
	pendingEventsCache := NewCache[common.DAHeightEvent](nil, "")

	pendingHeaders, err := NewPendingHeaders(st, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create pending headers: %w", err)
	}

	pendingData, err := NewPendingData(st, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create pending data: %w", err)
	}

	impl := &implementation{
		headerCache:        headerCache,
		dataCache:          dataCache,
		txCache:            txCache,
		txTimestamps:       new(sync.Map),
		pendingEventsCache: pendingEventsCache,
		pendingHeaders:     pendingHeaders,
		pendingData:        pendingData,
		store:              st,
		config:             cfg,
		logger:             logger,
	}

	if cfg.ClearCache {
		if err := impl.ClearFromStore(); err != nil {
			logger.Warn().Err(err).Msg("failed to clear cache from disk, starting with empty cache")
		}
	} else {
		// Restore existing cache from store
		if err := impl.RestoreFromStore(); err != nil {
			logger.Warn().Err(err).Msg("failed to load cache from disk, starting with empty cache")
		}
	}

	return impl, nil
}

// Header operations
func (m *implementation) IsHeaderSeen(hash string) bool {
	return m.headerCache.isSeen(hash)
}

func (m *implementation) SetHeaderSeen(hash string, blockHeight uint64) {
	m.headerCache.setSeen(hash, blockHeight)
}

func (m *implementation) GetHeaderDAIncludedByHash(hash string) (uint64, bool) {
	return m.headerCache.getDAIncluded(hash)
}

func (m *implementation) GetHeaderDAIncludedByHeight(blockHeight uint64) (uint64, bool) {
	return m.headerCache.getDAIncludedByHeight(blockHeight)
}

func (m *implementation) SetHeaderDAIncluded(hash string, daHeight uint64, blockHeight uint64) {
	m.headerCache.setDAIncluded(hash, daHeight, blockHeight)
}

func (m *implementation) RemoveHeaderDAIncluded(hash string) {
	m.headerCache.removeDAIncluded(hash)
}

// DaHeight returns the highest DA height seen across header and data caches.
func (m *implementation) DaHeight() uint64 {
	return max(m.headerCache.daHeight(), m.dataCache.daHeight())
}

func (m *implementation) IsDataSeen(hash string) bool {
	return m.dataCache.isSeen(hash)
}

func (m *implementation) SetDataSeen(hash string, blockHeight uint64) {
	m.dataCache.setSeen(hash, blockHeight)
}

func (m *implementation) GetDataDAIncludedByHash(daCommitmentHash string) (uint64, bool) {
	return m.dataCache.getDAIncluded(daCommitmentHash)
}

func (m *implementation) GetDataDAIncludedByHeight(blockHeight uint64) (uint64, bool) {
	return m.dataCache.getDAIncludedByHeight(blockHeight)
}

func (m *implementation) SetDataDAIncluded(daCommitmentHash string, daHeight uint64, blockHeight uint64) {
	m.dataCache.setDAIncluded(daCommitmentHash, daHeight, blockHeight)
}

func (m *implementation) RemoveDataDAIncluded(hash string) {
	m.dataCache.removeDAIncluded(hash)
}

func (m *implementation) IsTxSeen(hash string) bool {
	return m.txCache.isSeen(hash)
}

func (m *implementation) SetTxSeen(hash string) {
	// Use 0 as height since transactions don't have a block height yet
	m.txCache.setSeen(hash, 0)
	// Track timestamp for cleanup purposes
	m.txTimestamps.Store(hash, time.Now())
}

// CleanupOldTxs removes transaction hashes older than olderThan and returns
// the count removed. Defaults to DefaultTxCacheRetention if olderThan <= 0.
func (m *implementation) CleanupOldTxs(olderThan time.Duration) int {
	if olderThan <= 0 {
		olderThan = DefaultTxCacheRetention
	}

	cutoff := time.Now().Add(-olderThan)
	removed := 0

	m.txTimestamps.Range(func(key, value any) bool {
		hash, ok := key.(string)
		if !ok {
			return true
		}
		timestamp, ok := value.(time.Time)
		if !ok {
			return true
		}
		if timestamp.Before(cutoff) {
			m.txCache.removeSeen(hash)
			m.txTimestamps.Delete(hash)
			removed++
		}
		return true
	})

	if removed > 0 {
		m.logger.Debug().
			Int("removed", removed).
			Dur("older_than", olderThan).
			Msg("cleaned up old transaction hashes from cache")
	}

	return removed
}

// DeleteHeight removes from all caches the given height.
// This can be done when a height has been da included.
func (m *implementation) DeleteHeight(blockHeight uint64) {
	m.headerCache.deleteAllForHeight(blockHeight)
	m.dataCache.deleteAllForHeight(blockHeight)
	m.pendingEventsCache.deleteAllForHeight(blockHeight)

	// Note: txCache is intentionally NOT deleted here because:
	// 1. Transactions are tracked by hash, not by block height (they use height 0)
	// 2. A transaction seen at one height may be resubmitted at a different height
	// 3. The cache prevents duplicate submissions across block heights
	// 4. Cleanup is handled separately via CleanupOldTxs() based on time, not height
}

// Pending operations
func (m *implementation) GetPendingHeaders(ctx context.Context) ([]*types.SignedHeader, [][]byte, error) {
	return m.pendingHeaders.GetPendingHeaders(ctx)
}

func (m *implementation) GetPendingData(ctx context.Context) ([]*types.SignedData, [][]byte, error) {
	// Get pending raw data with marshalled bytes
	dataList, marshalledData, err := m.pendingData.GetPendingData(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Convert to SignedData (this logic was in manager.go)
	signedDataList := make([]*types.SignedData, 0, len(dataList))
	marshalledSignedData := make([][]byte, 0, len(dataList))
	for i, data := range dataList {
		if len(data.Txs) == 0 {
			continue
		}
		// Note: Actual signing needs to be done by the executing component
		// as it has access to the signer. This method returns unsigned data
		// that will be signed by the executing component when needed.
		signedDataList = append(signedDataList, &types.SignedData{
			Data: *data,
			// Signature and Signer will be set by executing component
		})
		marshalledSignedData = append(marshalledSignedData, marshalledData[i])
	}

	return signedDataList, marshalledSignedData, nil
}

func (m *implementation) GetLastSubmittedHeaderHeight() uint64 {
	return m.pendingHeaders.GetLastSubmittedHeaderHeight()
}

func (m *implementation) SetLastSubmittedHeaderHeight(ctx context.Context, height uint64) {
	m.pendingHeaders.SetLastSubmittedHeaderHeight(ctx, height)
}

func (m *implementation) GetLastSubmittedDataHeight() uint64 {
	return m.pendingData.GetLastSubmittedDataHeight()
}

func (m *implementation) SetLastSubmittedDataHeight(ctx context.Context, height uint64) {
	m.pendingData.SetLastSubmittedDataHeight(ctx, height)
}

func (m *implementation) NumPendingHeaders() uint64 {
	return m.pendingHeaders.NumPendingHeaders()
}

func (m *implementation) NumPendingData() uint64 {
	return m.pendingData.NumPendingData()
}

// SetPendingEvent sets the event at the specified height.
func (m *implementation) SetPendingEvent(height uint64, event *common.DAHeightEvent) {
	m.pendingEventsCache.setItem(height, event)
}

// GetNextPendingEvent efficiently retrieves and removes the event at the specified height.
// Returns nil if no event exists at that height.
func (m *implementation) GetNextPendingEvent(height uint64) *common.DAHeightEvent {
	return m.pendingEventsCache.getNextItem(height)
}

// SaveToStore flushes the DA inclusion snapshot to the store.
// Mutations already persist on every call, so this is mainly a final flush.
func (m *implementation) SaveToStore() error {
	ctx := context.Background()
	if err := m.headerCache.SaveToStore(ctx); err != nil {
		return fmt.Errorf("failed to save header cache to store: %w", err)
	}
	if err := m.dataCache.SaveToStore(ctx); err != nil {
		return fmt.Errorf("failed to save data cache to store: %w", err)
	}

	// TX cache and pending events are ephemeral - not persisted
	return nil
}

// RestoreFromStore loads the in-flight snapshot (O(1)) then seeds maxDAHeight
// from the finalized-tip HeightToDAHeight metadata so DaHeight() is correct
// even when the snapshot is empty (all blocks finalized).
func (m *implementation) RestoreFromStore() error {
	ctx := context.Background()

	if err := m.headerCache.RestoreFromStore(ctx); err != nil {
		return fmt.Errorf("failed to restore header cache from store: %w", err)
	}
	if err := m.dataCache.RestoreFromStore(ctx); err != nil {
		return fmt.Errorf("failed to restore data cache from store: %w", err)
	}

	// Initialize DA height from store metadata to ensure DaHeight() is never 0.
	m.initDAHeightFromStore(ctx)

	m.logger.Info().
		Int("header_entries", m.headerCache.daIncluded.Len()).
		Int("data_entries", m.dataCache.daIncluded.Len()).
		Uint64("da_height", m.DaHeight()).
		Msg("restored DA inclusion cache from store")

	return nil
}

// ClearFromStore wipes the snapshot keys and rebuilds empty in-memory caches,
// then seeds maxDAHeight from persisted metadata so DaHeight() stays correct.
func (m *implementation) ClearFromStore() error {
	ctx := context.Background()

	if err := m.headerCache.ClearFromStore(ctx); err != nil {
		return fmt.Errorf("failed to clear header cache from store: %w", err)
	}
	if err := m.dataCache.ClearFromStore(ctx); err != nil {
		return fmt.Errorf("failed to clear data cache from store: %w", err)
	}

	m.headerCache = NewCache[types.SignedHeader](m.store, HeaderDAIncludedPrefix)
	m.dataCache = NewCache[types.Data](m.store, DataDAIncludedPrefix)
	m.txCache = NewCache[struct{}](nil, "")
	m.pendingEventsCache = NewCache[common.DAHeightEvent](nil, "")

	// Initialize DA height from store metadata to ensure DaHeight() is never 0.
	m.initDAHeightFromStore(ctx)

	return nil
}

// getMetadataUint64 reads an 8-byte little-endian uint64 from store metadata.
// Returns (0, false, nil) when the key is absent and a non-nil error for
// genuine backend failures or malformed values.
func getMetadataUint64(ctx context.Context, st store.Store, key string) (uint64, bool, error) {
	b, err := st.GetMetadata(ctx, key)
	if err != nil {
		// Key absent — not an error, just missing.
		if isNotFound(err) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("read metadata %q: %w", key, err)
	}
	if len(b) != 8 {
		return 0, false, fmt.Errorf("invalid metadata length for %q: %d", key, len(b))
	}
	return binary.LittleEndian.Uint64(b), true, nil
}

// initDAHeightFromStore seeds maxDAHeight from the HeightToDAHeight metadata
// written by the submitter for the last finalized block. This ensures
// DaHeight() is non-zero on restart even when the in-flight snapshot is empty.
func (m *implementation) initDAHeightFromStore(ctx context.Context) {
	daIncludedHeight, ok, err := getMetadataUint64(ctx, m.store, store.DAIncludedHeightKey)
	if err != nil {
		m.logger.Error().Err(err).Msg("failed to read DA included height from store")
		return
	}
	if !ok || daIncludedHeight == 0 {
		return
	}

	if h, ok, err := getMetadataUint64(ctx, m.store, store.GetHeightToDAHeightHeaderKey(daIncludedHeight)); err != nil {
		m.logger.Error().Err(err).Msg("failed to read header DA height from store")
	} else if ok {
		m.headerCache.setMaxDAHeight(h)
	}
	if h, ok, err := getMetadataUint64(ctx, m.store, store.GetHeightToDAHeightDataKey(daIncludedHeight)); err != nil {
		m.logger.Error().Err(err).Msg("failed to read data DA height from store")
	} else if ok {
		m.dataCache.setMaxDAHeight(h)
	}
}
