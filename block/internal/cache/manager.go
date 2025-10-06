package cache

import (
	"context"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"sync"

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

// Manager provides centralized cache management for both executing and syncing components
type Manager interface {
	// Header operations
	IsHeaderSeen(hash string) bool
	SetHeaderSeen(hash string)
	GetHeaderDAIncluded(hash string) (uint64, bool)
	SetHeaderDAIncluded(hash string, daHeight uint64)
	RemoveHeaderDAIncluded(hash string)

	// Data operations
	IsDataSeen(hash string) bool
	SetDataSeen(hash string)
	GetDataDAIncluded(hash string) (uint64, bool)
	SetDataDAIncluded(hash string, daHeight uint64)

	// Pending operations
	GetPendingHeaders(ctx context.Context) ([]*types.SignedHeader, error)
	GetPendingData(ctx context.Context) ([]*types.SignedData, error)
	SetLastSubmittedHeaderHeight(ctx context.Context, height uint64)
	SetLastSubmittedDataHeight(ctx context.Context, height uint64)
	NumPendingHeaders() uint64
	NumPendingData() uint64

	// Pending events syncing coordination
	GetNextPendingEvent(height uint64) *common.DAHeightEvent
	SetPendingEvent(height uint64, event *common.DAHeightEvent)

	// Cleanup operations
	SaveToDisk() error
	LoadFromDisk() error
	ClearFromDisk() error
}

var _ Manager = (*implementation)(nil)

// implementation provides the concrete implementation of cache Manager
type implementation struct {
	headerCache        *Cache[types.SignedHeader]
	dataCache          *Cache[types.Data]
	pendingEventsCache *Cache[common.DAHeightEvent]
	pendingHeaders     *PendingHeaders
	pendingData        *PendingData
	config             config.Config
	logger             zerolog.Logger
}

// NewManager creates a new cache manager instance
func NewManager(cfg config.Config, store store.Store, logger zerolog.Logger) (Manager, error) {
	// Initialize caches
	headerCache := NewCache[types.SignedHeader]()
	dataCache := NewCache[types.Data]()
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

	impl := &implementation{
		headerCache:        headerCache,
		dataCache:          dataCache,
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

func (m *implementation) SetHeaderSeen(hash string) {
	m.headerCache.setSeen(hash)
}

func (m *implementation) GetHeaderDAIncluded(hash string) (uint64, bool) {
	return m.headerCache.getDAIncluded(hash)
}

func (m *implementation) SetHeaderDAIncluded(hash string, daHeight uint64) {
	m.headerCache.setDAIncluded(hash, daHeight)
}

func (m *implementation) RemoveHeaderDAIncluded(hash string) {
	m.headerCache.removeDAIncluded(hash)
}

// Data operations
func (m *implementation) IsDataSeen(hash string) bool {
	return m.dataCache.isSeen(hash)
}

func (m *implementation) SetDataSeen(hash string) {
	m.dataCache.setSeen(hash)
}

func (m *implementation) GetDataDAIncluded(hash string) (uint64, bool) {
	return m.dataCache.getDAIncluded(hash)
}

func (m *implementation) SetDataDAIncluded(hash string, daHeight uint64) {
	m.dataCache.setDAIncluded(hash, daHeight)
}

// Pending operations
func (m *implementation) GetPendingHeaders(ctx context.Context) ([]*types.SignedHeader, error) {
	return m.pendingHeaders.GetPendingHeaders(ctx)
}

func (m *implementation) GetPendingData(ctx context.Context) ([]*types.SignedData, error) {
	// Get pending raw data
	dataList, err := m.pendingData.GetPendingData(ctx)
	if err != nil {
		return nil, err
	}

	// Convert to SignedData (this logic was in manager.go)
	signedDataList := make([]*types.SignedData, 0, len(dataList))
	for _, data := range dataList {
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
	}

	return signedDataList, nil
}

func (m *implementation) SetLastSubmittedHeaderHeight(ctx context.Context, height uint64) {
	m.pendingHeaders.SetLastSubmittedHeaderHeight(ctx, height)
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

	if err := m.pendingEventsCache.SaveToDisk(filepath.Join(cfgDir, pendingEventsCacheDir)); err != nil {
		return fmt.Errorf("failed to save pending events cache to disk: %w", err)
	}

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

	if err := m.pendingEventsCache.LoadFromDisk(filepath.Join(cfgDir, pendingEventsCacheDir)); err != nil {
		return fmt.Errorf("failed to load pending events cache from disk: %w", err)
	}

	return nil
}

func (m *implementation) ClearFromDisk() error {
	cachePath := filepath.Join(m.config.RootDir, "data", cacheDir)
	if err := os.RemoveAll(cachePath); err != nil {
		return fmt.Errorf("failed to clear cache from disk: %w", err)
	}
	return nil
}
