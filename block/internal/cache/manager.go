package cache

import (
	"context"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

var (
	cacheDir              = "cache"
	headerCacheDir        = filepath.Join(cacheDir, "header")
	dataCacheDir          = filepath.Join(cacheDir, "data")
	pendingEventsCacheDir = filepath.Join(cacheDir, "pending_da_events")
	txCacheDir            = filepath.Join(cacheDir, "tx")

	// DefaultTxCacheRetention is the default time to keep transaction hashes in cache
	DefaultTxCacheRetention = 24 * time.Hour
)

// gobRegisterOnce ensures gob type registration happens exactly once process-wide.
var gobRegisterOnce sync.Once

// registerGobTypes registers all concrete types that may be encoded/decoded by the cache.
// Gob registration is global and must not be performed repeatedly to avoid conflicts.
func registerGobTypes() {
	gobRegisterOnce.Do(func() {
		gob.Register(&types.SignedHeader{})
		gob.Register(&types.Data{})
		gob.Register(&common.DAHeightEvent{})
	})
}

// CacheManager provides thread-safe cache operations for tracking seen blocks
// and DA inclusion status during block execution and syncing.
type CacheManager interface {
	// Header operations
	IsHeaderSeen(hash string) bool
	SetHeaderSeen(hash string, blockHeight uint64)
	GetHeaderDAIncluded(hash string) (uint64, bool)
	SetHeaderDAIncluded(hash string, daHeight uint64, blockHeight uint64)
	RemoveHeaderDAIncluded(hash string)
	DaHeight() uint64

	// Data operations
	IsDataSeen(hash string) bool
	SetDataSeen(hash string, blockHeight uint64)
	GetDataDAIncluded(hash string) (uint64, bool)
	SetDataDAIncluded(hash string, daHeight uint64, blockHeight uint64)

	// Transaction operations
	IsTxSeen(hash string) bool
	SetTxSeen(hash string)
	CleanupOldTxs(olderThan time.Duration) int

	// Pending events syncing coordination
	GetNextPendingEvent(blockHeight uint64) *common.DAHeightEvent
	SetPendingEvent(blockHeight uint64, event *common.DAHeightEvent)

	// Disk operations
	SaveToDisk() error
	LoadFromDisk() error
	ClearFromDisk() error

	// Cleanup operations
	DeleteHeight(blockHeight uint64)
}

// PendingManager provides operations for managing pending headers and data
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

// Manager provides centralized cache management for both executing and syncing components
type Manager interface {
	CacheManager
	PendingManager
}

var _ Manager = (*implementation)(nil)

// implementation provides the concrete implementation of cache Manager
type implementation struct {
	headerCache        *Cache[types.SignedHeader]
	dataCache          *Cache[types.Data]
	txCache            *Cache[struct{}]
	txTimestamps       *sync.Map // map[string]time.Time - tracks when each tx was seen
	pendingEventsCache *Cache[common.DAHeightEvent]
	pendingHeaders     *PendingHeaders
	pendingData        *PendingData
	config             config.Config
	logger             zerolog.Logger
}

// NewPendingManager creates a new pending manager instance
func NewPendingManager(store store.Store, logger zerolog.Logger) (PendingManager, error) {
	pendingHeaders, err := NewPendingHeaders(store, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create pending headers: %w", err)
	}

	pendingData, err := NewPendingData(store, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create pending data: %w", err)
	}

	return &implementation{
		pendingHeaders: pendingHeaders,
		pendingData:    pendingData,
		logger:         logger,
	}, nil
}

// NewCacheManager creates a new cache manager instance
func NewCacheManager(cfg config.Config, logger zerolog.Logger) (CacheManager, error) {
	// Initialize caches
	headerCache := NewCache[types.SignedHeader]()
	dataCache := NewCache[types.Data]()
	txCache := NewCache[struct{}]()
	pendingEventsCache := NewCache[common.DAHeightEvent]()

	registerGobTypes()
	impl := &implementation{
		headerCache:        headerCache,
		dataCache:          dataCache,
		txCache:            txCache,
		txTimestamps:       new(sync.Map),
		pendingEventsCache: pendingEventsCache,
		config:             cfg,
		logger:             logger,
	}

	if cfg.ClearCache {
		// Clear the cache from disk
		if err := impl.ClearFromDisk(); err != nil {
			logger.Warn().Err(err).Msg("failed to clear cache from disk, starting with empty cache")
		}
	} else {
		// Load existing cache from disk
		if err := impl.LoadFromDisk(); err != nil {
			logger.Warn().Err(err).Msg("failed to load cache from disk, starting with empty cache")
		}
	}

	return impl, nil
}

// NewManager creates a new cache manager instance
func NewManager(cfg config.Config, store store.Store, logger zerolog.Logger) (Manager, error) {
	// Initialize caches
	headerCache := NewCache[types.SignedHeader]()
	dataCache := NewCache[types.Data]()
	txCache := NewCache[struct{}]()
	pendingEventsCache := NewCache[common.DAHeightEvent]()

	// Initialize pending managers
	pendingHeaders, err := NewPendingHeaders(store, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create pending headers: %w", err)
	}

	pendingData, err := NewPendingData(store, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create pending data: %w", err)
	}

	registerGobTypes()
	impl := &implementation{
		headerCache:        headerCache,
		dataCache:          dataCache,
		txCache:            txCache,
		txTimestamps:       new(sync.Map),
		pendingEventsCache: pendingEventsCache,
		pendingHeaders:     pendingHeaders,
		pendingData:        pendingData,
		config:             cfg,
		logger:             logger,
	}

	if cfg.ClearCache {
		// Clear the cache from disk
		if err := impl.ClearFromDisk(); err != nil {
			logger.Warn().Err(err).Msg("failed to clear cache from disk, starting with empty cache")
		}
	} else {
		// Load existing cache from disk
		if err := impl.LoadFromDisk(); err != nil {
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

func (m *implementation) GetHeaderDAIncluded(hash string) (uint64, bool) {
	return m.headerCache.getDAIncluded(hash)
}

func (m *implementation) SetHeaderDAIncluded(hash string, daHeight uint64, blockHeight uint64) {
	m.headerCache.setDAIncluded(hash, daHeight, blockHeight)
}

func (m *implementation) RemoveHeaderDAIncluded(hash string) {
	m.headerCache.removeDAIncluded(hash)
}

// DaHeight fetches the heights da height contained in the processed cache.
func (m *implementation) DaHeight() uint64 {
	return max(m.headerCache.daHeight(), m.dataCache.daHeight())
}

// Data operations
func (m *implementation) IsDataSeen(hash string) bool {
	return m.dataCache.isSeen(hash)
}

func (m *implementation) SetDataSeen(hash string, blockHeight uint64) {
	m.dataCache.setSeen(hash, blockHeight)
}

func (m *implementation) GetDataDAIncluded(hash string) (uint64, bool) {
	return m.dataCache.getDAIncluded(hash)
}

func (m *implementation) SetDataDAIncluded(hash string, daHeight uint64, blockHeight uint64) {
	m.dataCache.setDAIncluded(hash, daHeight, blockHeight)
}

// Transaction operations
func (m *implementation) IsTxSeen(hash string) bool {
	return m.txCache.isSeen(hash)
}

func (m *implementation) SetTxSeen(hash string) {
	// Use 0 as height since transactions don't have a block height yet
	m.txCache.setSeen(hash, 0)
	// Track timestamp for cleanup purposes
	m.txTimestamps.Store(hash, time.Now())
}

// CleanupOldTxs removes transaction hashes older than the specified duration.
// Returns the number of transactions removed.
// This prevents unbounded growth of the transaction cache.
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
			// Remove from both caches
			m.txCache.hashes.Delete(hash)
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
			continue // Skip empty data
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

func (m *implementation) SaveToDisk() error {
	cfgDir := filepath.Join(m.config.RootDir, "data")

	// Ensure gob types are registered before encoding
	registerGobTypes()

	if err := m.headerCache.SaveToDisk(filepath.Join(cfgDir, headerCacheDir)); err != nil {
		return fmt.Errorf("failed to save header cache to disk: %w", err)
	}

	if err := m.dataCache.SaveToDisk(filepath.Join(cfgDir, dataCacheDir)); err != nil {
		return fmt.Errorf("failed to save data cache to disk: %w", err)
	}

	if err := m.txCache.SaveToDisk(filepath.Join(cfgDir, txCacheDir)); err != nil {
		return fmt.Errorf("failed to save tx cache to disk: %w", err)
	}

	if err := m.pendingEventsCache.SaveToDisk(filepath.Join(cfgDir, pendingEventsCacheDir)); err != nil {
		return fmt.Errorf("failed to save pending events cache to disk: %w", err)
	}

	// Note: txTimestamps are not persisted to disk intentionally.
	// On restart, all cached transactions will be treated as "new" for cleanup purposes,
	// which is acceptable as they will be cleaned up on the next cleanup cycle if old enough.

	return nil
}

func (m *implementation) LoadFromDisk() error {
	// Ensure types are registered exactly once prior to decoding
	registerGobTypes()

	cfgDir := filepath.Join(m.config.RootDir, "data")

	if err := m.headerCache.LoadFromDisk(filepath.Join(cfgDir, headerCacheDir)); err != nil {
		return fmt.Errorf("failed to load header cache from disk: %w", err)
	}

	if err := m.dataCache.LoadFromDisk(filepath.Join(cfgDir, dataCacheDir)); err != nil {
		return fmt.Errorf("failed to load data cache from disk: %w", err)
	}

	if err := m.txCache.LoadFromDisk(filepath.Join(cfgDir, txCacheDir)); err != nil {
		return fmt.Errorf("failed to load tx cache from disk: %w", err)
	}

	if err := m.pendingEventsCache.LoadFromDisk(filepath.Join(cfgDir, pendingEventsCacheDir)); err != nil {
		return fmt.Errorf("failed to load pending events cache from disk: %w", err)
	}

	// After loading tx cache from disk, initialize timestamps for loaded transactions
	// Set them to current time so they won't be immediately cleaned up
	now := time.Now()
	m.txCache.hashes.Range(func(key, value any) bool {
		if hash, ok := key.(string); ok {
			m.txTimestamps.Store(hash, now)
		}
		return true
	})

	return nil
}

func (m *implementation) ClearFromDisk() error {
	cachePath := filepath.Join(m.config.RootDir, "data", cacheDir)
	if err := os.RemoveAll(cachePath); err != nil {
		return fmt.Errorf("failed to clear cache from disk: %w", err)
	}
	return nil
}
