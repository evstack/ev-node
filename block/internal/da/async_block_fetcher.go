package da

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	ds "github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
	dsync "github.com/ipfs/go-datastore/sync"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/pkg/config"
	datypes "github.com/evstack/ev-node/pkg/da/types"
)

// AsyncBlockFetcher provides background prefetching of DA blocks
type AsyncBlockFetcher interface {
	Start()
	Stop()
	GetCachedBlock(ctx context.Context, daHeight uint64) (*BlockData, error)
	UpdateCurrentHeight(height uint64)
}

// BlockData contains data retrieved from a single DA height
type BlockData struct {
	Height    uint64
	Timestamp time.Time
	Blobs     [][]byte
}

// asyncBlockFetcher handles background prefetching of individual DA blocks
// to speed up forced inclusion processing.
type asyncBlockFetcher struct {
	client        Client
	logger        zerolog.Logger
	daStartHeight uint64

	// In-memory cache for prefetched block data
	cache ds.Batching
	mu    sync.RWMutex

	// Background fetcher control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Current DA height tracking
	currentDAHeight uint64
	heightMu        sync.RWMutex

	// Prefetch window - how many blocks ahead to prefetch
	prefetchWindow uint64

	// Polling interval for checking new DA heights
	pollInterval time.Duration
}

// NewAsyncBlockFetcher creates a new async block fetcher with in-memory cache.
func NewAsyncBlockFetcher(
	client Client,
	logger zerolog.Logger,
	config config.Config,
	daStartHeight uint64,
	prefetchWindow uint64,
) AsyncBlockFetcher {
	if prefetchWindow == 0 {
		prefetchWindow = 10 // Default: prefetch next 10 blocks
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &asyncBlockFetcher{
		client:          client,
		logger:          logger.With().Str("component", "async_block_fetcher").Logger(),
		daStartHeight:   daStartHeight,
		cache:           dsync.MutexWrap(ds.NewMapDatastore()),
		ctx:             ctx,
		cancel:          cancel,
		currentDAHeight: daStartHeight,
		prefetchWindow:  prefetchWindow,
		pollInterval:    config.DA.BlockTime.Duration,
	}
}

// Start begins the background prefetching process.
func (f *asyncBlockFetcher) Start() {
	f.wg.Add(1)
	go f.backgroundFetchLoop()
	f.logger.Info().
		Uint64("da_start_height", f.daStartHeight).
		Uint64("prefetch_window", f.prefetchWindow).
		Dur("poll_interval", f.pollInterval).
		Msg("async block fetcher started")
}

// Stop gracefully stops the background prefetching process.
func (f *asyncBlockFetcher) Stop() {
	f.logger.Info().Msg("stopping async block fetcher")
	f.cancel()
	f.wg.Wait()
	f.logger.Info().Msg("async block fetcher stopped")
}

// UpdateCurrentHeight updates the current DA height for prefetching.
func (f *asyncBlockFetcher) UpdateCurrentHeight(height uint64) {
	f.heightMu.Lock()
	defer f.heightMu.Unlock()

	if height > f.currentDAHeight {
		f.currentDAHeight = height
		f.logger.Debug().
			Uint64("new_height", height).
			Msg("updated current DA height")
	}
}

// GetCachedBlock retrieves a cached block from memory.
// Returns nil if the block is not cached.
func (f *asyncBlockFetcher) GetCachedBlock(ctx context.Context, daHeight uint64) (*BlockData, error) {
	if !f.client.HasForcedInclusionNamespace() {
		return nil, ErrForceInclusionNotConfigured
	}

	if daHeight < f.daStartHeight {
		return nil, fmt.Errorf("DA height %d is before the configured start height %d", daHeight, f.daStartHeight)
	}

	// Try to get from cache
	key := ds.NewKey(fmt.Sprintf("/block/%d", daHeight))

	f.mu.RLock()
	data, err := f.cache.Get(ctx, key)
	f.mu.RUnlock()

	if err != nil {
		if errors.Is(err, ds.ErrNotFound) {
			return nil, nil // Not cached yet
		}
		return nil, fmt.Errorf("failed to get cached block: %w", err)
	}

	// Deserialize the cached block
	block, err := deserializeBlockData(data)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize cached block: %w", err)
	}

	f.logger.Debug().
		Uint64("da_height", daHeight).
		Int("blob_count", len(block.Blobs)).
		Msg("retrieved block from cache")

	return block, nil
}

// backgroundFetchLoop runs in the background and prefetches blocks ahead of time.
func (f *asyncBlockFetcher) backgroundFetchLoop() {
	defer f.wg.Done()

	ticker := time.NewTicker(f.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-f.ctx.Done():
			return
		case <-ticker.C:
			f.prefetchBlocks()
		}
	}
}

// prefetchBlocks prefetches blocks within the prefetch window.
func (f *asyncBlockFetcher) prefetchBlocks() {
	if !f.client.HasForcedInclusionNamespace() {
		return
	}

	f.heightMu.RLock()
	currentHeight := f.currentDAHeight
	f.heightMu.RUnlock()

	// Prefetch upcoming blocks
	for i := uint64(0); i < f.prefetchWindow; i++ {
		targetHeight := currentHeight + i

		// Check if already cached
		key := ds.NewKey(fmt.Sprintf("/block/%d", targetHeight))
		f.mu.RLock()
		_, err := f.cache.Get(f.ctx, key)
		f.mu.RUnlock()

		if err == nil {
			// Already cached
			continue
		}

		// Fetch and cache the block
		f.fetchAndCacheBlock(targetHeight)
	}

	// Clean up old blocks from cache to prevent memory growth
	f.cleanupOldBlocks(currentHeight)
}

// fetchAndCacheBlock fetches a block and stores it in the cache.
func (f *asyncBlockFetcher) fetchAndCacheBlock(height uint64) {
	f.logger.Debug().
		Uint64("height", height).
		Msg("prefetching block")

	result := f.client.Retrieve(f.ctx, height, f.client.GetForcedInclusionNamespace())

	if result.Code == datypes.StatusHeightFromFuture {
		f.logger.Debug().
			Uint64("height", height).
			Msg("block height not yet available - will retry")
		return
	}

	block := &BlockData{
		Height:    height,
		Timestamp: result.Timestamp,
		Blobs:     [][]byte{},
	}

	if result.Code == datypes.StatusNotFound {
		f.logger.Debug().
			Uint64("height", height).
			Msg("no forced inclusion blobs at height")
		// Cache empty result to avoid re-fetching
	} else if result.Code == datypes.StatusSuccess {
		// Process each blob
		for _, blob := range result.Data {
			if len(blob) > 0 {
				block.Blobs = append(block.Blobs, blob)
			}
		}
		f.logger.Debug().
			Uint64("height", height).
			Int("blob_count", len(result.Data)).
			Msg("processed forced inclusion blobs for prefetch")
	} else {
		f.logger.Warn().
			Uint64("height", height).
			Str("status", result.Message).
			Msg("failed to retrieve block - will retry")
		return
	}

	// Serialize and cache the block
	data, err := serializeBlockData(block)
	if err != nil {
		f.logger.Error().
			Err(err).
			Uint64("height", height).
			Msg("failed to serialize block for caching")
		return
	}

	key := ds.NewKey(fmt.Sprintf("/block/%d", height))
	f.mu.Lock()
	err = f.cache.Put(f.ctx, key, data)
	f.mu.Unlock()

	if err != nil {
		f.logger.Error().
			Err(err).
			Uint64("height", height).
			Msg("failed to cache block")
		return
	}

	f.logger.Info().
		Uint64("height", height).
		Int("blob_count", len(block.Blobs)).
		Msg("successfully prefetched and cached block")
}

// cleanupOldBlocks removes blocks older than a threshold from cache.
func (f *asyncBlockFetcher) cleanupOldBlocks(currentHeight uint64) {
	// Remove blocks older than current - prefetchWindow
	// Keep some history in case of reorgs or restarts
	if currentHeight < f.prefetchWindow {
		return
	}

	cleanupThreshold := currentHeight - f.prefetchWindow

	f.mu.Lock()
	defer f.mu.Unlock()

	// Query all keys
	query := dsq.Query{Prefix: "/block/"}
	results, err := f.cache.Query(f.ctx, query)
	if err != nil {
		f.logger.Warn().Err(err).Msg("failed to query cache for cleanup")
		return
	}
	defer results.Close()

	for result := range results.Next() {
		if result.Error != nil {
			continue
		}

		key := ds.NewKey(result.Key)
		// Extract height from key
		var height uint64
		_, err := fmt.Sscanf(key.String(), "/block/%d", &height)
		if err != nil {
			continue
		}

		if height < cleanupThreshold {
			if err := f.cache.Delete(f.ctx, key); err != nil {
				f.logger.Warn().
					Err(err).
					Uint64("height", height).
					Msg("failed to delete old block from cache")
			} else {
				f.logger.Debug().
					Uint64("height", height).
					Msg("cleaned up old block from cache")
			}
		}
	}
}

// serializeBlockData serializes block data to bytes.
// Format: timestamp (int64) + height (uint64) + blobCount (uint32) + blobs
func serializeBlockData(block *BlockData) ([]byte, error) {
	// Calculate total size
	size := 8 + 8 + 4 // timestamp + height + blobCount
	for _, blob := range block.Blobs {
		size += 4 + len(blob) // blobLen + blob
	}

	buf := make([]byte, size)
	offset := 0

	// Timestamp
	writeUint64(buf[offset:], uint64(block.Timestamp.Unix()))
	offset += 8

	// Height
	writeUint64(buf[offset:], block.Height)
	offset += 8

	// BlobCount
	writeUint32(buf[offset:], uint32(len(block.Blobs)))
	offset += 4

	// Blobs
	for _, blob := range block.Blobs {
		writeUint32(buf[offset:], uint32(len(blob)))
		offset += 4
		copy(buf[offset:], blob)
		offset += len(blob)
	}

	return buf, nil
}

// deserializeBlockData deserializes bytes to block data.
func deserializeBlockData(data []byte) (*BlockData, error) {
	if len(data) < 20 {
		return nil, errors.New("invalid data: too short")
	}

	offset := 0
	block := &BlockData{}

	// Timestamp
	timestamp := readUint64(data[offset:])
	block.Timestamp = time.Unix(int64(timestamp), 0).UTC()
	offset += 8

	// Height
	block.Height = readUint64(data[offset:])
	offset += 8

	// BlobCount
	blobCount := readUint32(data[offset:])
	offset += 4

	// Blobs
	block.Blobs = make([][]byte, blobCount)
	for i := uint32(0); i < blobCount; i++ {
		if offset+4 > len(data) {
			return nil, errors.New("invalid data: unexpected end while reading blob length")
		}
		blobLen := readUint32(data[offset:])
		offset += 4

		if offset+int(blobLen) > len(data) {
			return nil, errors.New("invalid data: unexpected end while reading blob")
		}
		block.Blobs[i] = make([]byte, blobLen)
		copy(block.Blobs[i], data[offset:offset+int(blobLen)])
		offset += int(blobLen)
	}

	return block, nil
}

func writeUint64(buf []byte, val uint64) {
	buf[0] = byte(val)
	buf[1] = byte(val >> 8)
	buf[2] = byte(val >> 16)
	buf[3] = byte(val >> 24)
	buf[4] = byte(val >> 32)
	buf[5] = byte(val >> 40)
	buf[6] = byte(val >> 48)
	buf[7] = byte(val >> 56)
}

func readUint64(buf []byte) uint64 {
	return uint64(buf[0]) |
		uint64(buf[1])<<8 |
		uint64(buf[2])<<16 |
		uint64(buf[3])<<24 |
		uint64(buf[4])<<32 |
		uint64(buf[5])<<40 |
		uint64(buf[6])<<48 |
		uint64(buf[7])<<56
}

func writeUint32(buf []byte, val uint32) {
	buf[0] = byte(val)
	buf[1] = byte(val >> 8)
	buf[2] = byte(val >> 16)
	buf[3] = byte(val >> 24)
}

func readUint32(buf []byte) uint32 {
	return uint32(buf[0]) |
		uint32(buf[1])<<8 |
		uint32(buf[2])<<16 |
		uint32(buf[3])<<24
}
