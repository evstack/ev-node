package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

const (
	// Store key prefixes for different cache types
	headerDAIncludedPrefix = "cache/header-da-included/"
	dataDAIncludedPrefix   = "cache/data-da-included/"

	// DefaultTxCacheRetention is the default time to keep transaction hashes in cache
	DefaultTxCacheRetention = 24 * time.Hour
)

// CacheManager provides thread-safe cache operations for tracking seen blocks
// and DA inclusion status during block execution and syncing.
type CacheManager interface {
	DaHeight() uint64

	// Header operations
	IsHeaderSeen(hash string) bool
	SetHeaderSeen(hash string, blockHeight uint64)
	GetHeaderDAIncluded(hash string) (uint64, bool)
	SetHeaderDAIncluded(hash string, daHeight uint64, blockHeight uint64)
	RemoveHeaderDAIncluded(hash string)

	// Data operations
	IsDataSeen(hash string) bool
	SetDataSeen(hash string, blockHeight uint64)
	GetDataDAIncluded(hash string) (uint64, bool)
	SetDataDAIncluded(hash string, daHeight uint64, blockHeight uint64)
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
	store              store.Store
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
func NewCacheManager(cfg config.Config, st store.Store, logger zerolog.Logger) (CacheManager, error) {
	// Initialize caches with store-based persistence for DA inclusion data
	headerCache := NewCache[types.SignedHeader](st, headerDAIncludedPrefix)
	dataCache := NewCache[types.Data](st, dataDAIncludedPrefix)
	// TX cache and pending events cache don't need store persistence
	txCache := NewCache[struct{}](nil, "")
	pendingEventsCache := NewCache[common.DAHeightEvent](nil, "")

	impl := &implementation{
		headerCache:        headerCache,
		dataCache:          dataCache,
		txCache:            txCache,
		txTimestamps:       new(sync.Map),
		pendingEventsCache: pendingEventsCache,
		store:              st,
		config:             cfg,
		logger:             logger,
	}

	// Restore existing cache from store
	if err := impl.RestoreFromStore(); err != nil {
		logger.Warn().Err(err).Msg("failed to restore cache from store, starting with empty cache")
	}

	return impl, nil
}

// NewManager creates a new cache manager instance
func NewManager(cfg config.Config, st store.Store, logger zerolog.Logger) (Manager, error) {
	// Initialize caches with store-based persistence for DA inclusion data
	headerCache := NewCache[types.SignedHeader](st, headerDAIncludedPrefix)
	dataCache := NewCache[types.Data](st, dataDAIncludedPrefix)
	// TX cache and pending events cache don't need store persistence
	txCache := NewCache[struct{}](nil, "")
	pendingEventsCache := NewCache[common.DAHeightEvent](nil, "")

	// Initialize pending managers
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

	// Restore existing cache from store
	if err := impl.RestoreFromStore(); err != nil {
		logger.Warn().Err(err).Msg("failed to restore cache from store, starting with empty cache")
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

func (m *implementation) RemoveDataDAIncluded(hash string) {
	m.dataCache.removeDAIncluded(hash)
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

// SaveToStore persists the DA inclusion cache to the store.
// DA inclusion data is persisted on every SetHeaderDAIncluded/SetDataDAIncluded call,
// so this method ensures any remaining data is flushed.
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

// RestoreFromStore restores the DA inclusion cache from the store.
// This iterates through blocks in the store and checks for persisted DA inclusion data.
func (m *implementation) RestoreFromStore() error {
	ctx := context.Background()

	// Get current store height to know how many blocks to check
	height, err := m.store.Height(ctx)
	if err != nil {
		return fmt.Errorf("failed to get store height: %w", err)
	}

	if height == 0 {
		return nil // No blocks to restore
	}

	// Collect hashes from stored blocks
	var headerHashes []string
	var dataHashes []string

	for h := uint64(1); h <= height; h++ {
		header, data, err := m.store.GetBlockData(ctx, h)
		if err != nil {
			m.logger.Warn().Uint64("height", h).Err(err).Msg("failed to get block data during cache restore")
			continue
		}

		if header != nil {
			headerHashes = append(headerHashes, header.Hash().String())
		}
		if data != nil {
			dataHashes = append(dataHashes, data.DACommitment().String())
		}
	}

	// Restore DA inclusion data from store
	if err := m.headerCache.RestoreFromStore(ctx, headerHashes); err != nil {
		return fmt.Errorf("failed to restore header cache from store: %w", err)
	}

	if err := m.dataCache.RestoreFromStore(ctx, dataHashes); err != nil {
		return fmt.Errorf("failed to restore data cache from store: %w", err)
	}

	m.logger.Info().
		Int("header_hashes", len(headerHashes)).
		Int("data_hashes", len(dataHashes)).
		Msg("restored DA inclusion cache from store")

	return nil
}

// ClearFromStore clears in-memory caches and deletes DA inclusion entries from the store.
func (m *implementation) ClearFromStore() error {
	ctx := context.Background()

	// Get hashes from current in-memory caches and delete from store
	headerHashes := m.headerCache.daIncluded.Keys()
	if err := m.headerCache.ClearFromStore(ctx, headerHashes); err != nil {
		return fmt.Errorf("failed to clear header cache from store: %w", err)
	}

	dataHashes := m.dataCache.daIncluded.Keys()
	if err := m.dataCache.ClearFromStore(ctx, dataHashes); err != nil {
		return fmt.Errorf("failed to clear data cache from store: %w", err)
	}

	// Clear in-memory caches by creating new ones
	m.headerCache = NewCache[types.SignedHeader](m.store, headerDAIncludedPrefix)
	m.dataCache = NewCache[types.Data](m.store, dataDAIncludedPrefix)
	m.txCache = NewCache[struct{}](nil, "")
	m.pendingEventsCache = NewCache[common.DAHeightEvent](nil, "")

	return nil
}
