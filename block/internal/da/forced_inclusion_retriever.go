package da

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"

	coreda "github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/types"
)

const (
	// defaultEpochLag is the default number of blocks to lag behind DA height when fetching forced inclusion txs
	defaultEpochLag = 10

	// defaultMinEpochWindow is the minimum window size for epoch lag calculation
	defaultMinEpochWindow = 5

	// defaultMaxEpochWindow is the maximum window size for epoch lag calculation
	defaultMaxEpochWindow = 100

	// defaultFetchInterval is the interval between async fetch attempts
	defaultFetchInterval = 2 * time.Second
)

// ErrForceInclusionNotConfigured is returned when the forced inclusion namespace is not configured.
var ErrForceInclusionNotConfigured = errors.New("forced inclusion namespace not configured")

// epochCache stores fetched forced inclusion events by epoch start height
type epochCache struct {
	events     atomic.Pointer[map[uint64]*ForcedInclusionEvent]
	fetchTimes atomic.Pointer[[]time.Duration]
	maxSamples int
}

func newEpochCache(maxSamples int) *epochCache {
	c := &epochCache{
		maxSamples: maxSamples,
	}
	initialEvents := make(map[uint64]*ForcedInclusionEvent)
	c.events.Store(&initialEvents)
	initialTimes := make([]time.Duration, 0, maxSamples)
	c.fetchTimes.Store(&initialTimes)
	return c
}

func (c *epochCache) get(epochStart uint64) (*ForcedInclusionEvent, bool) {
	events := c.events.Load()
	event, ok := (*events)[epochStart]
	return event, ok
}

func (c *epochCache) set(epochStart uint64, event *ForcedInclusionEvent) {
	for {
		oldEventsPtr := c.events.Load()
		oldEvents := *oldEventsPtr
		newEvents := make(map[uint64]*ForcedInclusionEvent, len(oldEvents)+1)
		for k, v := range oldEvents {
			newEvents[k] = v
		}
		newEvents[epochStart] = event
		if c.events.CompareAndSwap(oldEventsPtr, &newEvents) {
			return
		}
	}
}

func (c *epochCache) recordFetchTime(duration time.Duration) {
	for {
		oldTimesPtr := c.fetchTimes.Load()
		oldTimes := *oldTimesPtr
		newTimes := make([]time.Duration, 0, c.maxSamples)
		newTimes = append(newTimes, oldTimes...)
		newTimes = append(newTimes, duration)
		if len(newTimes) > c.maxSamples {
			newTimes = newTimes[1:]
		}
		if c.fetchTimes.CompareAndSwap(oldTimesPtr, &newTimes) {
			return
		}
	}
}

func (c *epochCache) averageFetchTime() time.Duration {
	timesPtr := c.fetchTimes.Load()
	times := *timesPtr
	if len(times) == 0 {
		return 0
	}
	var sum time.Duration
	for _, d := range times {
		sum += d
	}
	return sum / time.Duration(len(times))
}

func (c *epochCache) cleanup(beforeEpoch uint64) {
	for {
		oldEventsPtr := c.events.Load()
		oldEvents := *oldEventsPtr
		newEvents := make(map[uint64]*ForcedInclusionEvent)
		for epoch, event := range oldEvents {
			if epoch >= beforeEpoch {
				newEvents[epoch] = event
			}
		}
		if c.events.CompareAndSwap(oldEventsPtr, &newEvents) {
			return
		}
	}
}

// ForcedInclusionRetriever handles retrieval of forced inclusion transactions from DA.
type ForcedInclusionRetriever struct {
	client      Client
	genesis     genesis.Genesis
	logger      zerolog.Logger
	daEpochSize uint64

	// Async forced inclusion fetching
	epochCache      *epochCache
	fetcherCtx      context.Context
	fetcherCancel   context.CancelFunc
	fetcherWg       sync.WaitGroup
	currentDAHeight atomic.Uint64
}

// ForcedInclusionEvent contains forced inclusion transactions retrieved from DA.
type ForcedInclusionEvent struct {
	StartDaHeight uint64
	EndDaHeight   uint64
	Txs           [][]byte
}

// NewForcedInclusionRetriever creates a new forced inclusion retriever.
func NewForcedInclusionRetriever(
	client Client,
	genesis genesis.Genesis,
	logger zerolog.Logger,
) *ForcedInclusionRetriever {
	ctx, cancel := context.WithCancel(context.Background())

	r := &ForcedInclusionRetriever{
		client:        client,
		genesis:       genesis,
		logger:        logger.With().Str("component", "forced_inclusion_retriever").Logger(),
		daEpochSize:   genesis.DAEpochForcedInclusion,
		epochCache:    newEpochCache(10), // Keep last 10 fetch times for averaging
		fetcherCtx:    ctx,
		fetcherCancel: cancel,
	}
	r.currentDAHeight.Store(genesis.DAStartHeight)

	// Start background fetcher if forced inclusion is configured
	if client.HasForcedInclusionNamespace() {
		r.fetcherWg.Add(1)
		go r.backgroundFetcher()
	}

	return r
}

// StopBackgroundFetcher stops the background fetcher goroutine
func (r *ForcedInclusionRetriever) StopBackgroundFetcher() {
	if r.fetcherCancel != nil {
		r.fetcherCancel()
	}
	r.fetcherWg.Wait()
}

// SetDAHeight updates the current DA height for async fetching
func (r *ForcedInclusionRetriever) SetDAHeight(height uint64) {
	for {
		current := r.currentDAHeight.Load()
		if height <= current {
			return
		}
		if r.currentDAHeight.CompareAndSwap(current, height) {
			return
		}
	}
}

// GetDAHeight returns the current DA height
func (r *ForcedInclusionRetriever) GetDAHeight() uint64 {
	return r.currentDAHeight.Load()
}

// calculateAdaptiveEpochWindow calculates the epoch lag window based on average fetch time
func (r *ForcedInclusionRetriever) calculateAdaptiveEpochWindow() uint64 {
	avgFetchTime := r.epochCache.averageFetchTime()
	if avgFetchTime == 0 {
		return defaultEpochLag
	}

	// Scale window based on fetch time: faster fetches = smaller window
	// If fetch takes 1 second, window = 5
	// If fetch takes 5 seconds, window = 25
	// If fetch takes 10 seconds, window = 50
	window := uint64(avgFetchTime.Seconds() * 5)

	if window < defaultMinEpochWindow {
		window = defaultMinEpochWindow
	}
	if window > defaultMaxEpochWindow {
		window = defaultMaxEpochWindow
	}

	return window
}

// backgroundFetcher continuously fetches forced inclusion transactions ahead of time
func (r *ForcedInclusionRetriever) backgroundFetcher() {
	defer r.fetcherWg.Done()

	ticker := time.NewTicker(defaultFetchInterval)
	defer ticker.Stop()

	r.logger.Info().Msg("started background forced inclusion fetcher")

	for {
		select {
		case <-r.fetcherCtx.Done():
			r.logger.Info().Msg("stopped background forced inclusion fetcher")
			return
		case <-ticker.C:
			r.fetchNextEpoch()
		}
	}
}

// fetchNextEpoch fetches the next epoch that should be available based on current DA height and lag
func (r *ForcedInclusionRetriever) fetchNextEpoch() {
	currentHeight := r.GetDAHeight()
	if currentHeight == 0 {
		return
	}

	window := r.calculateAdaptiveEpochWindow()

	// Calculate which epoch the sequencer will need soon (lagging behind current height)
	// We want to prefetch this epoch before it's actually requested
	laggedHeight := currentHeight
	if currentHeight > window {
		laggedHeight = currentHeight - window
	}

	epochStart, _ := types.CalculateEpochBoundaries(laggedHeight, r.genesis.DAStartHeight, r.daEpochSize)

	// Check if we already have this epoch cached
	if _, ok := r.epochCache.get(epochStart); ok {
		return
	}

	// Fetch this epoch in the background
	r.logger.Debug().
		Uint64("current_height", currentHeight).
		Uint64("lagged_height", laggedHeight).
		Uint64("epoch_start", epochStart).
		Uint64("window", window).
		Msg("fetching epoch in background")

	startTime := time.Now()
	ctx, cancel := context.WithTimeout(r.fetcherCtx, 30*time.Second)
	defer cancel()

	event, err := r.fetchEpochSync(ctx, epochStart)
	if err != nil {
		r.logger.Debug().Err(err).Uint64("epoch_start", epochStart).Msg("failed to fetch epoch in background")
		return
	}

	// Record fetch time for adaptive window
	fetchDuration := time.Since(startTime)
	r.epochCache.recordFetchTime(fetchDuration)

	// Cache the event
	r.epochCache.set(epochStart, event)

	r.logger.Debug().
		Uint64("epoch_start", epochStart).
		Int("tx_count", len(event.Txs)).
		Dur("fetch_duration", fetchDuration).
		Msg("cached epoch in background")

	// Cleanup old epochs (keep last 5 epochs)
	if epochStart >= r.genesis.DAStartHeight+r.daEpochSize*5 {
		cleanupBefore := epochStart - r.daEpochSize*5
		if cleanupBefore < r.genesis.DAStartHeight {
			cleanupBefore = r.genesis.DAStartHeight
		}
		r.epochCache.cleanup(cleanupBefore)
	}
}

// RetrieveForcedIncludedTxs retrieves forced inclusion transactions at the given DA height.
// It respects epoch boundaries and only fetches at epoch start.
// Uses cached results from background fetcher when available.
func (r *ForcedInclusionRetriever) RetrieveForcedIncludedTxs(ctx context.Context, daHeight uint64) (*ForcedInclusionEvent, error) {
	if !r.client.HasForcedInclusionNamespace() {
		return nil, ErrForceInclusionNotConfigured
	}

	// Update our tracking of DA height
	r.SetDAHeight(daHeight)

	epochStart, _ := types.CalculateEpochBoundaries(daHeight, r.genesis.DAStartHeight, r.daEpochSize)

	if daHeight != epochStart {
		r.logger.Debug().
			Uint64("da_height", daHeight).
			Uint64("epoch_start", epochStart).
			Msg("not at epoch start - returning empty transactions")

		return &ForcedInclusionEvent{
			StartDaHeight: daHeight,
			EndDaHeight:   daHeight,
			Txs:           [][]byte{},
		}, nil
	}

	// Check if we have this epoch cached from background fetcher
	if cachedEvent, ok := r.epochCache.get(epochStart); ok {
		r.logger.Debug().
			Uint64("epoch_start", epochStart).
			Int("tx_count", len(cachedEvent.Txs)).
			Msg("using cached forced inclusion transactions")
		return cachedEvent, nil
	}

	// Not cached, fetch synchronously
	r.logger.Debug().
		Uint64("da_height", daHeight).
		Uint64("epoch_start", epochStart).
		Msg("cache miss, fetching forced inclusion transactions synchronously")

	return r.fetchEpochSync(ctx, epochStart)
}

// fetchEpochSync synchronously fetches an entire epoch's forced inclusion transactions
func (r *ForcedInclusionRetriever) fetchEpochSync(ctx context.Context, epochStart uint64) (*ForcedInclusionEvent, error) {
	epochEnd := epochStart + r.daEpochSize - 1
	currentEpochNumber := types.CalculateEpochNumber(epochStart, r.genesis.DAStartHeight, r.daEpochSize)

	event := &ForcedInclusionEvent{
		StartDaHeight: epochStart,
		Txs:           [][]byte{},
	}

	r.logger.Debug().
		Uint64("epoch_start", epochStart).
		Uint64("epoch_end", epochEnd).
		Uint64("epoch_num", currentEpochNumber).
		Msg("fetching forced included transactions from DA")

	epochStartResult := r.client.RetrieveForcedInclusion(ctx, epochStart)
	if epochStartResult.Code == coreda.StatusHeightFromFuture {
		r.logger.Debug().
			Uint64("epoch_start", epochStart).
			Msg("epoch start height not yet available on DA - backoff required")
		return nil, fmt.Errorf("%w: epoch start height %d not yet available", coreda.ErrHeightFromFuture, epochStart)
	}

	epochEndResult := epochStartResult
	if epochStart != epochEnd {
		epochEndResult = r.client.RetrieveForcedInclusion(ctx, epochEnd)
		if epochEndResult.Code == coreda.StatusHeightFromFuture {
			r.logger.Debug().
				Uint64("epoch_end", epochEnd).
				Msg("epoch end height not yet available on DA - backoff required")
			return nil, fmt.Errorf("%w: epoch end height %d not yet available", coreda.ErrHeightFromFuture, epochEnd)
		}
	}

	lastProcessedHeight := epochStart

	if err := r.processForcedInclusionBlobs(event, &lastProcessedHeight, epochStartResult, epochStart); err != nil {
		return nil, err
	}

	// Process heights between start and end (exclusive)
	for epochHeight := epochStart + 1; epochHeight < epochEnd; epochHeight++ {
		result := r.client.RetrieveForcedInclusion(ctx, epochHeight)

		// If any intermediate height is from future, break early
		if result.Code == coreda.StatusHeightFromFuture {
			r.logger.Debug().
				Uint64("epoch_height", epochHeight).
				Uint64("last_processed", lastProcessedHeight).
				Msg("reached future DA height within epoch - stopping")
			break
		}

		if err := r.processForcedInclusionBlobs(event, &lastProcessedHeight, result, epochHeight); err != nil {
			return nil, err
		}
	}

	// Process epoch end (only if different from start)
	if epochEnd != epochStart {
		if err := r.processForcedInclusionBlobs(event, &lastProcessedHeight, epochEndResult, epochEnd); err != nil {
			return nil, err
		}
	}

	event.EndDaHeight = lastProcessedHeight

	r.logger.Info().
		Uint64("epoch_start", epochStart).
		Uint64("epoch_end", lastProcessedHeight).
		Int("tx_count", len(event.Txs)).
		Msg("retrieved forced inclusion transactions")

	return event, nil
}

// processForcedInclusionBlobs processes blobs from a single DA height for forced inclusion.
func (r *ForcedInclusionRetriever) processForcedInclusionBlobs(
	event *ForcedInclusionEvent,
	lastProcessedHeight *uint64,
	result coreda.ResultRetrieve,
	height uint64,
) error {
	if result.Code == coreda.StatusNotFound {
		r.logger.Debug().Uint64("height", height).Msg("no forced inclusion blobs at height")
		*lastProcessedHeight = height
		return nil
	}

	if result.Code != coreda.StatusSuccess {
		return fmt.Errorf("failed to retrieve forced inclusion blobs at height %d: %s", height, result.Message)
	}

	// Process each blob as a transaction
	for _, blob := range result.Data {
		if len(blob) > 0 {
			event.Txs = append(event.Txs, blob)
		}
	}

	*lastProcessedHeight = height

	r.logger.Debug().
		Uint64("height", height).
		Int("blob_count", len(result.Data)).
		Msg("processed forced inclusion blobs")

	return nil
}
