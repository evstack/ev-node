package da

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	ds "github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
	dsync "github.com/ipfs/go-datastore/sync"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"

	datypes "github.com/evstack/ev-node/pkg/da/types"
	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

// AsyncBlockRetriever provides background prefetching of DA blocks
type AsyncBlockRetriever interface {
	Start(ctx context.Context)
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
// from a specific namespace. Wraps a da.Subscriber for the subscription
// plumbing and implements SubscriberHandler for caching.
type asyncBlockRetriever struct {
	subscriber *Subscriber
	client     Client
	logger     zerolog.Logger
	namespace  []byte

	// In-memory cache for prefetched block data
	cache ds.Batching

	// Current DA height tracking (accessed atomically via subscriber).
	daStartHeight uint64

	// Tracks DA height consumed by the sequencer to trigger cache cleanups.
	consumedHeight atomic.Uint64

	// Prefetch window - how many blocks ahead to speculatively fetch.
	prefetchWindow uint64
}

// NewAsyncBlockRetriever creates a new async block retriever with in-memory cache.
func NewAsyncBlockRetriever(
	client Client,
	logger zerolog.Logger,
	namespace []byte,
	daBlockTime time.Duration,
	daStartHeight uint64,
	prefetchWindow uint64,
) AsyncBlockRetriever {
	if prefetchWindow == 0 {
		prefetchWindow = 10
	}

	f := &asyncBlockRetriever{
		client:         client,
		logger:         logger.With().Str("component", "async_block_retriever").Logger(),
		namespace:      namespace,
		daStartHeight:  daStartHeight,
		cache:          dsync.MutexWrap(ds.NewMapDatastore()),
		prefetchWindow: prefetchWindow,
	}

	var namespaces [][]byte
	if len(namespace) > 0 {
		namespaces = [][]byte{namespace}
	}

	f.subscriber = NewSubscriber(SubscriberConfig{
		Client:              client,
		Logger:              logger,
		Namespaces:          namespaces,
		DABlockTime:         daBlockTime,
		Handler:             f,
		FetchBlockTimestamp: true,
		StartHeight:         daStartHeight,
	})

	return f
}

// Start begins the subscription follow loop and catchup loop.
func (f *asyncBlockRetriever) Start(ctx context.Context) {
	if err := f.subscriber.Start(ctx); err != nil {
		f.logger.Warn().Err(err).Msg("failed to start subscriber")
	}
	f.logger.Debug().
		Uint64("da_start_height", f.daStartHeight).
		Uint64("prefetch_window", f.prefetchWindow).
		Msg("async block retriever started")
}

// Stop gracefully stops the background goroutines.
func (f *asyncBlockRetriever) Stop() {
	f.logger.Debug().Msg("stopping async block retriever")
	f.subscriber.Stop()
}

// UpdateCurrentHeight updates the consumed DA height and triggers cache cleanup.
func (f *asyncBlockRetriever) UpdateCurrentHeight(height uint64) {
	for {
		current := f.consumedHeight.Load()
		if height <= current {
			return
		}
		if f.consumedHeight.CompareAndSwap(current, height) {
			f.logger.Debug().
				Uint64("new_height", height).
				Msg("updated consumed DA height for cleanup")
			f.cleanupOldBlocks(context.Background(), height)
			return
		}
	}
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

	var pbBlock pb.BlockData
	if err := proto.Unmarshal(data, &pbBlock); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached block: %w", err)
	}

	block := &BlockData{
		Height:    pbBlock.Height,
		Timestamp: time.Unix(0, pbBlock.Timestamp).UTC(),
		Blobs:     pbBlock.Blobs,
	}

	f.logger.Debug().
		Uint64("da_height", daHeight).
		Int("blob_count", len(block.Blobs)).
		Msg("retrieved block from cache")

	return block, nil
}

// HandleEvent caches blobs from the subscription inline, even empty ones,
// to record that the DA height was seen and has 0 blobs.
func (f *asyncBlockRetriever) HandleEvent(ctx context.Context, ev datypes.SubscriptionEvent, isInline bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.cacheBlock(ctx, ev.Height, ev.Timestamp, ev.Blobs)
	if isInline {
		return errors.New("async block retriever relies on catchup state machine")
	}
	return nil
}

// HandleCatchup fetches a single height via Retrieve and caches it.
// Also applies the prefetch window for speculative forward fetching.
func (f *asyncBlockRetriever) HandleCatchup(ctx context.Context, daHeight uint64) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if _, err := f.cache.Get(ctx, newBlockDataKey(daHeight)); err != nil {
		if err := f.fetchAndCacheBlock(ctx, daHeight); err != nil {
			return err
		}
	}
	// Speculatively prefetch ahead.
	target := daHeight + f.prefetchWindow
	for h := daHeight + 1; h <= target; h++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, err := f.cache.Get(ctx, newBlockDataKey(h)); err == nil {
			continue // Already cached.
		}
		if err := f.fetchAndCacheBlock(ctx, h); err != nil {
			return err
		}
	}

	return nil
}

// fetchAndCacheBlock fetches a block via Retrieve and caches it.
func (f *asyncBlockRetriever) fetchAndCacheBlock(ctx context.Context, height uint64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.logger.Debug().Uint64("height", height).Msg("prefetching block")
	result := f.client.Retrieve(ctx, height, f.namespace)

	switch result.Code {
	case datypes.StatusHeightFromFuture:
		f.logger.Debug().Uint64("height", height).Msg("block height not yet available - will retry")
		return datypes.ErrHeightFromFuture
	case datypes.StatusNotFound:
		f.cacheBlock(ctx, height, result.Timestamp, nil)
	case datypes.StatusSuccess:
		blobs := make([][]byte, 0, len(result.Data))
		for _, blob := range result.Data {
			if len(blob) > 0 {
				blobs = append(blobs, blob)
			}
		}
		f.cacheBlock(ctx, height, result.Timestamp, blobs)
	default:
		f.logger.Debug().
			Uint64("height", height).
			Str("status", result.Message).
			Msg("failed to retrieve block - will retry")
		return fmt.Errorf("retrieve block at height %d: %s", height, result.Message)
	}
	return nil
}

// cacheBlock serializes and stores a block in the in-memory cache.
func (f *asyncBlockRetriever) cacheBlock(ctx context.Context, daHeight uint64, daTimestamp time.Time, blobs [][]byte) {
	if blobs == nil {
		blobs = [][]byte{}
	}

	pbBlock := &pb.BlockData{
		Height:    daHeight,
		Timestamp: daTimestamp.UnixNano(),
		Blobs:     blobs,
	}
	data, err := proto.Marshal(pbBlock)
	if err != nil {
		f.logger.Error().Err(err).Uint64("height", daHeight).Msg("failed to marshal block for caching")
		return
	}

	key := newBlockDataKey(daHeight)
	if err := f.cache.Put(ctx, key, data); err != nil {
		f.logger.Error().Err(err).Uint64("height", daHeight).Msg("failed to cache block")
		return
	}

	f.logger.Debug().Uint64("height", daHeight).Int("blob_count", len(blobs)).Msg("cached block")
}

// cleanupOldBlocks removes blocks older than currentHeight − prefetchWindow.
func (f *asyncBlockRetriever) cleanupOldBlocks(ctx context.Context, currentHeight uint64) {
	if currentHeight < f.prefetchWindow {
		return
	}

	cleanupThreshold := currentHeight - f.prefetchWindow

	query := dsq.Query{Prefix: "/block/"}
	results, err := f.cache.Query(ctx, query)
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
		var height uint64
		_, err := fmt.Sscanf(key.String(), "/block/%d", &height)
		if err != nil {
			continue
		}

		if height < cleanupThreshold {
			if err := f.cache.Delete(ctx, key); err != nil {
				f.logger.Debug().Err(err).Uint64("height", height).Msg("failed to delete old block from cache")
			}
		}
	}
}
