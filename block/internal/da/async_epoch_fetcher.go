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

	datypes "github.com/evstack/ev-node/pkg/da/types"
	"github.com/evstack/ev-node/types"
)

// AsyncEpochFetcher provides background prefetching of DA epoch data
type AsyncEpochFetcher interface {
	Start()
	Stop()
	GetCachedEpoch(ctx context.Context, daHeight uint64) (*ForcedInclusionEvent, error)
}

// asyncEpochFetcher handles background prefetching of DA epoch data
// to speed up processing at epoch boundaries.
type asyncEpochFetcher struct {
	client        Client
	logger        zerolog.Logger
	daEpochSize   uint64
	daStartHeight uint64

	// In-memory cache for prefetched epoch data
	cache ds.Batching
	mu    sync.RWMutex

	// Background fetcher control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Current DA height tracking
	currentDAHeight uint64

	// Prefetch window - how many epochs ahead to prefetch
	prefetchWindow uint64

	// Polling interval for checking new DA heights
	pollInterval time.Duration
}

// NewAsyncEpochFetcher creates a new async epoch fetcher with in-memory cache.
func NewAsyncEpochFetcher(
	client Client,
	logger zerolog.Logger,
	daStartHeight, daEpochSize uint64,
	prefetchWindow uint64,
	pollInterval time.Duration,
) AsyncEpochFetcher {
	if prefetchWindow == 0 {
		prefetchWindow = 1 // Default: prefetch next epoch
	}
	if pollInterval == 0 {
		pollInterval = 2 * time.Second // Default polling interval
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &asyncEpochFetcher{
		client:          client,
		logger:          logger.With().Str("component", "async_epoch_fetcher").Logger(),
		daStartHeight:   daStartHeight,
		daEpochSize:     daEpochSize,
		cache:           dsync.MutexWrap(ds.NewMapDatastore()),
		ctx:             ctx,
		cancel:          cancel,
		currentDAHeight: daStartHeight,
		prefetchWindow:  prefetchWindow,
		pollInterval:    pollInterval,
	}
}

// Start begins the background prefetching process.
func (f *asyncEpochFetcher) Start() {
	f.wg.Add(1)
	go f.backgroundFetchLoop()
	f.logger.Info().
		Uint64("da_start_height", f.daStartHeight).
		Uint64("da_epoch_size", f.daEpochSize).
		Uint64("prefetch_window", f.prefetchWindow).
		Dur("poll_interval", f.pollInterval).
		Msg("async epoch fetcher started")
}

// Stop gracefully stops the background prefetching process.
func (f *asyncEpochFetcher) Stop() {
	f.logger.Info().Msg("stopping async epoch fetcher")
	f.cancel()
	f.wg.Wait()
	f.logger.Info().Msg("async epoch fetcher stopped")
}

// GetCachedEpoch retrieves a cached epoch from memory.
// Returns nil if the epoch is not cached.
func (f *asyncEpochFetcher) GetCachedEpoch(ctx context.Context, daHeight uint64) (*ForcedInclusionEvent, error) {
	if !f.client.HasForcedInclusionNamespace() {
		return nil, ErrForceInclusionNotConfigured
	}

	if daHeight < f.daStartHeight {
		return nil, fmt.Errorf("DA height %d is before the configured start height %d", daHeight, f.daStartHeight)
	}

	epochStart, epochEnd, _ := types.CalculateEpochBoundaries(daHeight, f.daStartHeight, f.daEpochSize)

	// Only return cached data at epoch end
	if daHeight != epochEnd {
		return &ForcedInclusionEvent{
			StartDaHeight: daHeight,
			EndDaHeight:   daHeight,
			Txs:           [][]byte{},
		}, nil
	}

	// Try to get from cache
	key := ds.NewKey(fmt.Sprintf("/epoch/%d", epochEnd))

	f.mu.RLock()
	data, err := f.cache.Get(ctx, key)
	f.mu.RUnlock()

	if err != nil {
		if errors.Is(err, ds.ErrNotFound) {
			return nil, nil // Not cached yet
		}
		return nil, fmt.Errorf("failed to get cached epoch: %w", err)
	}

	// Deserialize the cached event
	event, err := deserializeForcedInclusionEvent(data)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize cached epoch: %w", err)
	}

	f.logger.Debug().
		Uint64("da_height", daHeight).
		Uint64("epoch_start", epochStart).
		Uint64("epoch_end", epochEnd).
		Int("tx_count", len(event.Txs)).
		Msg("retrieved epoch from cache")

	return event, nil
}

// backgroundFetchLoop runs in the background and prefetches epochs ahead of time.
func (f *asyncEpochFetcher) backgroundFetchLoop() {
	defer f.wg.Done()

	ticker := time.NewTicker(f.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-f.ctx.Done():
			return
		case <-ticker.C:
			f.prefetchEpochs()
		}
	}
}

// prefetchEpochs prefetches epochs within the prefetch window.
func (f *asyncEpochFetcher) prefetchEpochs() {
	if !f.client.HasForcedInclusionNamespace() {
		return
	}

	// Calculate the current epoch and epochs to prefetch
	_, currentEpochEnd, _ := types.CalculateEpochBoundaries(f.currentDAHeight, f.daStartHeight, f.daEpochSize)

	// Prefetch upcoming epochs
	for i := uint64(0); i < f.prefetchWindow; i++ {
		targetEpochEnd := currentEpochEnd + (i * f.daEpochSize)

		// Check if already cached
		key := ds.NewKey(fmt.Sprintf("/epoch/%d", targetEpochEnd))
		f.mu.RLock()
		_, err := f.cache.Get(f.ctx, key)
		f.mu.RUnlock()

		if err == nil {
			// Already cached
			continue
		}

		// Fetch and cache the epoch
		f.fetchAndCacheEpoch(targetEpochEnd)
	}

	// Clean up old epochs from cache to prevent memory growth
	f.cleanupOldEpochs(currentEpochEnd)
}

// fetchAndCacheEpoch fetches an epoch and stores it in the cache.
func (f *asyncEpochFetcher) fetchAndCacheEpoch(epochEnd uint64) {
	epochStart := max(epochEnd-(f.daEpochSize-1), f.daStartHeight)
	f.logger.Debug().
		Uint64("epoch_start", epochStart).
		Uint64("epoch_end", epochEnd).
		Msg("prefetching epoch")

	event := &ForcedInclusionEvent{
		StartDaHeight: epochStart,
		EndDaHeight:   epochEnd,
		Txs:           [][]byte{},
	}

	// Fetch epoch end first to check availability
	epochEndResult := f.client.Retrieve(f.ctx, epochEnd, f.client.GetForcedInclusionNamespace())
	if epochEndResult.Code == datypes.StatusHeightFromFuture {
		f.logger.Debug().
			Uint64("epoch_end", epochEnd).
			Msg("epoch end height not yet available - will retry")
		return
	}

	epochStartResult := epochEndResult
	if epochStart != epochEnd {
		epochStartResult = f.client.Retrieve(f.ctx, epochStart, f.client.GetForcedInclusionNamespace())
		if epochStartResult.Code == datypes.StatusHeightFromFuture {
			f.logger.Debug().
				Uint64("epoch_start", epochStart).
				Msg("epoch start height not yet available - will retry")
			return
		}
	}

	// Process all heights in the epoch
	var processErrs error
	err := f.processForcedInclusionBlobs(event, epochStartResult, epochStart)
	processErrs = errors.Join(processErrs, err)

	// Process heights between start and end (exclusive)
	for epochHeight := epochStart + 1; epochHeight < epochEnd; epochHeight++ {
		result := f.client.Retrieve(f.ctx, epochHeight, f.client.GetForcedInclusionNamespace())
		err = f.processForcedInclusionBlobs(event, result, epochHeight)
		processErrs = errors.Join(processErrs, err)
	}

	// Process epoch end (only if different from start)
	if epochEnd != epochStart {
		err = f.processForcedInclusionBlobs(event, epochEndResult, epochEnd)
		processErrs = errors.Join(processErrs, err)
	}

	if processErrs != nil {
		f.logger.Warn().
			Uint64("epoch_start", epochStart).
			Uint64("epoch_end", epochEnd).
			Err(processErrs).
			Msg("failed to prefetch epoch - will retry")
		return
	}

	// Serialize and cache the event
	data, err := serializeForcedInclusionEvent(event)
	if err != nil {
		f.logger.Error().
			Err(err).
			Uint64("epoch_end", epochEnd).
			Msg("failed to serialize epoch for caching")
		return
	}

	key := ds.NewKey(fmt.Sprintf("/epoch/%d", epochEnd))
	f.mu.Lock()
	err = f.cache.Put(f.ctx, key, data)
	f.mu.Unlock()

	if err != nil {
		f.logger.Error().
			Err(err).
			Uint64("epoch_end", epochEnd).
			Msg("failed to cache epoch")
		return
	}

	f.logger.Info().
		Uint64("epoch_start", epochStart).
		Uint64("epoch_end", epochEnd).
		Int("tx_count", len(event.Txs)).
		Msg("successfully prefetched and cached epoch")
}

// processForcedInclusionBlobs processes blobs from a single DA height for forced inclusion.
func (f *asyncEpochFetcher) processForcedInclusionBlobs(
	event *ForcedInclusionEvent,
	result datypes.ResultRetrieve,
	height uint64,
) error {
	if result.Code == datypes.StatusNotFound {
		f.logger.Debug().Uint64("height", height).Msg("no forced inclusion blobs at height")
		return nil
	}

	if result.Code != datypes.StatusSuccess {
		return fmt.Errorf("failed to retrieve forced inclusion blobs at height %d: %s", height, result.Message)
	}

	// Process each blob as a transaction
	for _, blob := range result.Data {
		if len(blob) > 0 {
			event.Txs = append(event.Txs, blob)
		}
	}

	if result.Timestamp.After(event.Timestamp) {
		event.Timestamp = result.Timestamp
	}

	f.logger.Debug().
		Uint64("height", height).
		Int("blob_count", len(result.Data)).
		Msg("processed forced inclusion blobs for prefetch")

	return nil
}

// cleanupOldEpochs removes epochs older than the current epoch from cache.
func (f *asyncEpochFetcher) cleanupOldEpochs(currentEpochEnd uint64) {
	// Remove epochs older than current - 1
	// Keep current and previous in case of reorgs or restarts
	cleanupThreshold := currentEpochEnd - f.daEpochSize

	f.mu.Lock()
	defer f.mu.Unlock()

	// Query all keys
	query := dsq.Query{Prefix: "/epoch/"}
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
		// Extract epoch end from key
		var epochEnd uint64
		_, err := fmt.Sscanf(key.String(), "/epoch/%d", &epochEnd)
		if err != nil {
			continue
		}

		if epochEnd < cleanupThreshold {
			if err := f.cache.Delete(f.ctx, key); err != nil {
				f.logger.Warn().
					Err(err).
					Uint64("epoch_end", epochEnd).
					Msg("failed to delete old epoch from cache")
			} else {
				f.logger.Debug().
					Uint64("epoch_end", epochEnd).
					Msg("cleaned up old epoch from cache")
			}
		}
	}
}

// serializeForcedInclusionEvent serializes an event to bytes.
// Simple format: timestamp (int64) + startHeight (uint64) + endHeight (uint64) + txCount (uint32) + txs
func serializeForcedInclusionEvent(event *ForcedInclusionEvent) ([]byte, error) {
	// Calculate total size
	size := 8 + 8 + 8 + 4 // timestamp + startHeight + endHeight + txCount
	for _, tx := range event.Txs {
		size += 4 + len(tx) // txLen + tx
	}

	buf := make([]byte, size)
	offset := 0

	// Timestamp
	writeUint64(buf[offset:], uint64(event.Timestamp.Unix()))
	offset += 8

	// StartDaHeight
	writeUint64(buf[offset:], event.StartDaHeight)
	offset += 8

	// EndDaHeight
	writeUint64(buf[offset:], event.EndDaHeight)
	offset += 8

	// TxCount
	writeUint32(buf[offset:], uint32(len(event.Txs)))
	offset += 4

	// Txs
	for _, tx := range event.Txs {
		writeUint32(buf[offset:], uint32(len(tx)))
		offset += 4
		copy(buf[offset:], tx)
		offset += len(tx)
	}

	return buf, nil
}

// deserializeForcedInclusionEvent deserializes bytes to an event.
func deserializeForcedInclusionEvent(data []byte) (*ForcedInclusionEvent, error) {
	if len(data) < 28 {
		return nil, errors.New("invalid data: too short")
	}

	offset := 0
	event := &ForcedInclusionEvent{}

	// Timestamp
	timestamp := readUint64(data[offset:])
	event.Timestamp = time.Unix(int64(timestamp), 0).UTC()
	offset += 8

	// StartDaHeight
	event.StartDaHeight = readUint64(data[offset:])
	offset += 8

	// EndDaHeight
	event.EndDaHeight = readUint64(data[offset:])
	offset += 8

	// TxCount
	txCount := readUint32(data[offset:])
	offset += 4

	// Txs
	event.Txs = make([][]byte, txCount)
	for i := uint32(0); i < txCount; i++ {
		if offset+4 > len(data) {
			return nil, errors.New("invalid data: unexpected end while reading tx length")
		}
		txLen := readUint32(data[offset:])
		offset += 4

		if offset+int(txLen) > len(data) {
			return nil, errors.New("invalid data: unexpected end while reading tx")
		}
		event.Txs[i] = make([]byte, txLen)
		copy(event.Txs[i], data[offset:offset+int(txLen)])
		offset += int(txLen)
	}

	return event, nil
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
