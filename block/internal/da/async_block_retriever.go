package da

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	ds "github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
	dsync "github.com/ipfs/go-datastore/sync"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"

	"github.com/evstack/ev-node/pkg/config"
	datypes "github.com/evstack/ev-node/pkg/da/types"
	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

// AsyncBlockRetriever provides background prefetching of DA blocks
type AsyncBlockRetriever interface {
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

// asyncBlockRetriever handles background prefetching of individual DA blocks
// to speed up forced inclusion processing.
type asyncBlockRetriever struct {
	client        Client
	logger        zerolog.Logger
	daStartHeight uint64

	// In-memory cache for prefetched block data
	cache ds.Batching

	// Background fetcher control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Current DA height tracking (accessed atomically)
	currentDAHeight atomic.Uint64

	// Prefetch window - how many blocks ahead to prefetch
	prefetchWindow uint64

	// Polling interval for checking new DA heights
	pollInterval time.Duration
}

// NewAsyncBlockRetriever creates a new async block retriever with in-memory cache.
func NewAsyncBlockRetriever(
	client Client,
	logger zerolog.Logger,
	config config.Config,
	daStartHeight uint64,
	prefetchWindow uint64,
) AsyncBlockRetriever {
	if prefetchWindow == 0 {
		prefetchWindow = 10 // Default: prefetch next 10 blocks
	}

	ctx, cancel := context.WithCancel(context.Background())

	fetcher := &asyncBlockRetriever{
		client:         client,
		logger:         logger.With().Str("component", "async_block_retriever").Logger(),
		daStartHeight:  daStartHeight,
		cache:          dsync.MutexWrap(ds.NewMapDatastore()),
		ctx:            ctx,
		cancel:         cancel,
		prefetchWindow: prefetchWindow,
		pollInterval:   config.DA.BlockTime.Duration,
	}
	fetcher.currentDAHeight.Store(daStartHeight)
	return fetcher
}

// Start begins the background prefetching process.
func (f *asyncBlockRetriever) Start() {
	f.wg.Add(1)
	go f.backgroundFetchLoop()
	f.logger.Info().
		Uint64("da_start_height", f.daStartHeight).
		Uint64("prefetch_window", f.prefetchWindow).
		Dur("poll_interval", f.pollInterval).
		Msg("async block retriever started")
}

// Stop gracefully stops the background prefetching process.
func (f *asyncBlockRetriever) Stop() {
	f.logger.Info().Msg("stopping async block retriever")
	f.cancel()
	f.wg.Wait()
}

// UpdateCurrentHeight updates the current DA height for prefetching.
func (f *asyncBlockRetriever) UpdateCurrentHeight(height uint64) {
	// Use atomic compare-and-swap to update only if the new height is greater
	for {
		current := f.currentDAHeight.Load()
		if height <= current {
			return
		}
		if f.currentDAHeight.CompareAndSwap(current, height) {
			f.logger.Debug().
				Uint64("new_height", height).
				Msg("updated current DA height")
			return
		}
	}
}

// GetCachedBlock retrieves a cached block from memory.
// Returns nil if the block is not cached.
func (f *asyncBlockRetriever) GetCachedBlock(ctx context.Context, daHeight uint64) (*BlockData, error) {
	if !f.client.HasForcedInclusionNamespace() {
		return nil, ErrForceInclusionNotConfigured
	}

	if daHeight < f.daStartHeight {
		return nil, fmt.Errorf("DA height %d is before the configured start height %d", daHeight, f.daStartHeight)
	}

	// Try to get from cache
	key := ds.NewKey(fmt.Sprintf("/block/%d", daHeight))

	data, err := f.cache.Get(ctx, key)
	if err != nil {
		if errors.Is(err, ds.ErrNotFound) {
			return nil, nil // Not cached yet
		}
		return nil, fmt.Errorf("failed to get cached block: %w", err)
	}

	// Deserialize the cached block
	var pbBlock pb.BlockData
	if err := proto.Unmarshal(data, &pbBlock); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached block: %w", err)
	}

	block := &BlockData{
		Height:    pbBlock.Height,
		Timestamp: time.Unix(pbBlock.Timestamp, 0).UTC(),
		Blobs:     pbBlock.Blobs,
	}

	f.logger.Debug().
		Uint64("da_height", daHeight).
		Int("blob_count", len(block.Blobs)).
		Msg("retrieved block from cache")

	return block, nil
}

// backgroundFetchLoop runs in the background and prefetches blocks ahead of time.
func (f *asyncBlockRetriever) backgroundFetchLoop() {
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
func (f *asyncBlockRetriever) prefetchBlocks() {
	if !f.client.HasForcedInclusionNamespace() {
		return
	}

	currentHeight := f.currentDAHeight.Load()

	// Prefetch upcoming blocks
	for i := uint64(0); i < f.prefetchWindow; i++ {
		targetHeight := currentHeight + i

		// Check if already cached
		key := ds.NewKey(fmt.Sprintf("/block/%d", targetHeight))
		_, err := f.cache.Get(f.ctx, key)
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
func (f *asyncBlockRetriever) fetchAndCacheBlock(height uint64) {
	f.logger.Debug().
		Uint64("height", height).
		Msg("prefetching block")

	result := f.client.Retrieve(f.ctx, height, f.client.GetForcedInclusionNamespace())

	block := &BlockData{
		Height:    height,
		Timestamp: result.Timestamp,
		Blobs:     [][]byte{},
	}

	switch result.Code {
	case datypes.StatusHeightFromFuture:
		f.logger.Debug().
			Uint64("height", height).
			Msg("block height not yet available - will retry")
		return
	case datypes.StatusNotFound:
		f.logger.Debug().
			Uint64("height", height).
			Msg("no forced inclusion blobs at height")
		// Cache empty result to avoid re-fetching
	case datypes.StatusSuccess:
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
	default:
		f.logger.Warn().
			Uint64("height", height).
			Str("status", result.Message).
			Msg("failed to retrieve block - will retry")
		return
	}

	// Serialize and cache the block
	pbBlock := &pb.BlockData{
		Height:    block.Height,
		Timestamp: block.Timestamp.Unix(),
		Blobs:     block.Blobs,
	}
	data, err := proto.Marshal(pbBlock)
	if err != nil {
		f.logger.Error().
			Err(err).
			Uint64("height", height).
			Msg("failed to marshal block for caching")
		return
	}

	key := ds.NewKey(fmt.Sprintf("/block/%d", height))
	err = f.cache.Put(f.ctx, key, data)

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
func (f *asyncBlockRetriever) cleanupOldBlocks(currentHeight uint64) {
	// Remove blocks older than current - prefetchWindow
	if currentHeight < f.prefetchWindow {
		return
	}

	cleanupThreshold := currentHeight - f.prefetchWindow

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
