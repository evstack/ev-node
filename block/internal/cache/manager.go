package cache

import (
	"context"
	"encoding/gob"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/rs/zerolog"

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
		gob.Register(&DAHeightEvent{})
	})
}

// DAHeightEvent represents a DA event for caching
type DAHeightEvent struct {
	Header *types.SignedHeader
	Data   *types.Data
	// DaHeight corresponds to the highest DA included height between the Header and Data.
	DaHeight uint64
	// HeaderDaIncludedHeight corresponds to the DA height at which the Header was included.
	HeaderDaIncludedHeight uint64
}

// Manager provides centralized cache management for both executing and syncing components
type Manager interface {
	// Header operations
	GetHeader(height uint64) *types.SignedHeader
	SetHeader(height uint64, header *types.SignedHeader)
	IsHeaderSeen(hash string) bool
	SetHeaderSeen(hash string)
	IsHeaderDAIncluded(hash string) bool
	SetHeaderDAIncluded(hash string, daHeight uint64)

	// Data operations
	GetData(height uint64) *types.Data
	SetData(height uint64, data *types.Data)
	IsDataSeen(hash string) bool
	SetDataSeen(hash string)
	IsDataDAIncluded(hash string) bool
	SetDataDAIncluded(hash string, daHeight uint64)

	// Pending operations
	GetPendingHeaders(ctx context.Context) ([]*types.SignedHeader, error)
	GetPendingData(ctx context.Context) ([]*types.SignedData, error)
	SetLastSubmittedHeaderHeight(ctx context.Context, height uint64)
	SetLastSubmittedDataHeight(ctx context.Context, height uint64)

	NumPendingHeaders() uint64
	NumPendingData() uint64

	// Pending events for DA coordination
	SetPendingEvent(height uint64, event *DAHeightEvent)
	GetPendingEvents() map[uint64]*DAHeightEvent
	DeletePendingEvent(height uint64)

	// Cleanup operations
	ClearProcessedHeader(height uint64)
	ClearProcessedData(height uint64)
	SaveToDisk() error
	LoadFromDisk() error
}

// implementation provides the concrete implementation of cache Manager
type implementation struct {
	headerCache        *Cache[types.SignedHeader]
	dataCache          *Cache[types.Data]
	pendingEventsCache *Cache[DAHeightEvent]
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
	pendingEventsCache := NewCache[DAHeightEvent]()

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

	// Load existing cache from disk
	if err := impl.LoadFromDisk(); err != nil {
		logger.Warn().Err(err).Msg("failed to load cache from disk, starting with empty cache")
	}

	return impl, nil
}

// Header operations
func (m *implementation) GetHeader(height uint64) *types.SignedHeader {
	return m.headerCache.GetItem(height)
}

func (m *implementation) SetHeader(height uint64, header *types.SignedHeader) {
	m.headerCache.SetItem(height, header)
}

func (m *implementation) IsHeaderSeen(hash string) bool {
	return m.headerCache.IsSeen(hash)
}

func (m *implementation) SetHeaderSeen(hash string) {
	m.headerCache.SetSeen(hash)
}

func (m *implementation) IsHeaderDAIncluded(hash string) bool {
	return m.headerCache.IsDAIncluded(hash)
}

func (m *implementation) SetHeaderDAIncluded(hash string, daHeight uint64) {
	m.headerCache.SetDAIncluded(hash, daHeight)
}

// Data operations
func (m *implementation) GetData(height uint64) *types.Data {
	return m.dataCache.GetItem(height)
}

func (m *implementation) SetData(height uint64, data *types.Data) {
	m.dataCache.SetItem(height, data)
}

func (m *implementation) IsDataSeen(hash string) bool {
	return m.dataCache.IsSeen(hash)
}

func (m *implementation) SetDataSeen(hash string) {
	m.dataCache.SetSeen(hash)
}

func (m *implementation) IsDataDAIncluded(hash string) bool {
	return m.dataCache.IsDAIncluded(hash)
}

func (m *implementation) SetDataDAIncluded(hash string, daHeight uint64) {
	m.dataCache.SetDAIncluded(hash, daHeight)
}

// Pending operations
func (m *implementation) GetPendingHeaders(ctx context.Context) ([]*types.SignedHeader, error) {
	return m.pendingHeaders.getPendingHeaders(ctx)
}

func (m *implementation) GetPendingData(ctx context.Context) ([]*types.SignedData, error) {
	// Get pending raw data
	dataList, err := m.pendingData.getPendingData(ctx)
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
	m.pendingHeaders.setLastSubmittedHeaderHeight(ctx, height)
}

func (m *implementation) SetLastSubmittedDataHeight(ctx context.Context, height uint64) {
	m.pendingData.setLastSubmittedDataHeight(ctx, height)
}

func (m *implementation) NumPendingHeaders() uint64 {
	return m.pendingHeaders.numPendingHeaders()
}

func (m *implementation) NumPendingData() uint64 {
	return m.pendingData.numPendingData()
}

// Pending events operations
func (m *implementation) SetPendingEvent(height uint64, event *DAHeightEvent) {
	m.pendingEventsCache.SetItem(height, event)
}

func (m *implementation) GetPendingEvents() map[uint64]*DAHeightEvent {

	events := make(map[uint64]*DAHeightEvent)
	m.pendingEventsCache.RangeByHeight(func(height uint64, event *DAHeightEvent) bool {
		events[height] = event
		return true
	})
	return events
}

func (m *implementation) DeletePendingEvent(height uint64) {
	m.pendingEventsCache.DeleteItem(height)
}

// Cleanup operations
func (m *implementation) ClearProcessedHeader(height uint64) {
	m.headerCache.DeleteItem(height)
}

func (m *implementation) ClearProcessedData(height uint64) {
	m.dataCache.DeleteItem(height)
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
