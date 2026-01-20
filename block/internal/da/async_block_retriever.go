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
	blobrpc "github.com/evstack/ev-node/pkg/da/jsonrpc"
	datypes "github.com/evstack/ev-node/pkg/da/types"
	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

// AsyncBlockRetriever provides background prefetching of DA blocks
type AsyncBlockRetriever interface {
	Start()
	Stop()
	GetCachedBlock(ctx context.Context, daHeight uint64) (*BlockData, error)
	UpdateCurrentHeight(height uint64)
	StartSubscription()
}

// BlockData contains data retrieved from a single DA height
type BlockData struct {
	Height    uint64
	Timestamp time.Time
	Blobs     [][]byte
}

// asyncBlockRetriever handles background prefetching of individual DA blocks
// from a specific namespace. It can optionally subscribe to the namespace for
// real-time updates instead of relying solely on polling.
type asyncBlockRetriever struct {
	client        Client
	logger        zerolog.Logger
	namespace     []byte
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

	// Subscription support
	subscriptionStarted atomic.Bool
}

// NewAsyncBlockRetriever creates a new async block retriever with in-memory cache.
func NewAsyncBlockRetriever(
	client Client,
	logger zerolog.Logger,
	namespace []byte,
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
		namespace:      namespace,
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

	f.logger.Debug().
		Uint64("da_start_height", f.daStartHeight).
		Uint64("prefetch_window", f.prefetchWindow).
		Dur("poll_interval", f.pollInterval).
		Msg("async block retriever started")
}

// Stop gracefully stops the background prefetching process.
func (f *asyncBlockRetriever) Stop() {
	f.logger.Debug().Msg("stopping async block retriever")
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

// StartSubscription starts the subscription loop once; it is safe to call repeatedly.
func (f *asyncBlockRetriever) StartSubscription() {
	if len(f.namespace) == 0 {
		return
	}
	if !f.subscriptionStarted.CompareAndSwap(false, true) {
		return
	}
	f.wg.Add(1)
	go f.subscriptionLoop()
}

// storeBlock caches a block's blobs, favoring existing data to avoid churn.
// This is used internally by the subscription loop.
func (f *asyncBlockRetriever) storeBlock(ctx context.Context, height uint64, blobs [][]byte, timestamp time.Time) {
	if len(f.namespace) == 0 {
		return
	}
	if height < f.daStartHeight {
		return
	}
	if len(blobs) == 0 {
		return
	}

	filtered := make([][]byte, 0, len(blobs))
	for _, blob := range blobs {
		if len(blob) > 0 {
			filtered = append(filtered, blob)
		}
	}
	if len(filtered) == 0 {
		return
	}

	key := newBlockDataKey(height)
	if existing, err := f.cache.Get(ctx, key); err == nil {
		var pbBlock pb.BlockData
		if err := proto.Unmarshal(existing, &pbBlock); err == nil && len(pbBlock.Blobs) > 0 {
			return
		}
	}

	pbBlock := &pb.BlockData{
		Height:    height,
		Timestamp: timestamp.Unix(),
		Blobs:     filtered,
	}
	data, err := proto.Marshal(pbBlock)
	if err != nil {
		f.logger.Error().
			Err(err).
			Uint64("height", height).
			Msg("failed to marshal block for caching")
		return
	}

	if err := f.cache.Put(ctx, key, data); err != nil {
		f.logger.Error().
			Err(err).
			Uint64("height", height).
			Msg("failed to cache block")
		return
	}

	f.logger.Debug().
		Uint64("height", height).
		Int("blob_count", len(filtered)).
		Msg("cached block from subscription")
}

func newBlockDataKey(height uint64) ds.Key {
	return ds.NewKey(fmt.Sprintf("/block/%d", height))
}

// GetCachedBlock retrieves a cached block from memory.
// Returns nil if the block is not cached.
func (f *asyncBlockRetriever) GetCachedBlock(ctx context.Context, daHeight uint64) (*BlockData, error) {
	if len(f.namespace) == 0 {
		return nil, nil
	}

	if daHeight < f.daStartHeight {
		return nil, fmt.Errorf("DA height %d is before the configured start height %d", daHeight, f.daStartHeight)
	}

	key := newBlockDataKey(daHeight)
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

// subscriptionLoop subscribes to the namespace and caches incoming blobs.
func (f *asyncBlockRetriever) subscriptionLoop() {
	defer f.wg.Done()

	for {
		select {
		case <-f.ctx.Done():
			return
		default:
		}

		if err := f.runSubscription(); err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			f.logger.Warn().Err(err).Msg("subscription error, will retry")
			// Backoff before retry
			select {
			case <-f.ctx.Done():
				return
			case <-time.After(f.pollInterval):
			}
		}
	}
}

// runSubscription runs a single subscription session.
func (f *asyncBlockRetriever) runSubscription() error {
	ch, err := f.client.Subscribe(f.ctx, f.namespace)
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	f.logger.Debug().Msg("subscribed to namespace for real-time updates")

	for {
		select {
		case <-f.ctx.Done():
			return f.ctx.Err()
		case resp, ok := <-ch:
			if !ok {
				return errors.New("subscription channel closed")
			}
			f.handleSubscriptionResponse(resp)
		}
	}
}

// handleSubscriptionResponse processes a subscription response and caches the blobs.
func (f *asyncBlockRetriever) handleSubscriptionResponse(resp *blobrpc.SubscriptionResponse) {
	if resp == nil {
		return
	}

	f.UpdateCurrentHeight(resp.Height)

	if len(resp.Blobs) == 0 {
		return
	}

	// Extract raw blob data
	blobs := make([][]byte, 0, len(resp.Blobs))
	for _, b := range resp.Blobs {
		if b != nil && len(b.Data()) > 0 {
			blobs = append(blobs, b.Data())
		}
	}

	if len(blobs) > 0 {
		// TODO: Use Celestia subscription timestamps once available:
		// https://github.com/celestiaorg/celestia-node/pull/4752
		// Subscription responses do not carry DA timestamps; keep zero to avoid lying.
		f.storeBlock(f.ctx, resp.Height, blobs, time.Time{})
	}
}

// prefetchBlocks prefetches blocks within the prefetch window.
func (f *asyncBlockRetriever) prefetchBlocks() {
	if len(f.namespace) == 0 {
		return
	}

	currentHeight := f.currentDAHeight.Load()

	// Prefetch upcoming blocks
	for i := uint64(0); i < f.prefetchWindow; i++ {
		targetHeight := currentHeight + i

		// Check if already cached
		key := newBlockDataKey(targetHeight)
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

	result := f.client.Retrieve(f.ctx, height, f.namespace)

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
			Msg("no blobs at height")
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
			Msg("processed blobs for prefetch")
	default:
		f.logger.Debug().
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

	key := newBlockDataKey(height)
	err = f.cache.Put(f.ctx, key, data)
	if err != nil {
		f.logger.Error().
			Err(err).
			Uint64("height", height).
			Msg("failed to cache block")
		return
	}

	f.logger.Debug().
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
		f.logger.Debug().Err(err).Msg("failed to query cache for cleanup")
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
				f.logger.Debug().
					Err(err).
					Uint64("height", height).
					Msg("failed to delete old block from cache")
			}
		}
	}
}
