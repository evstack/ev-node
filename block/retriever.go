package block

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
	"google.golang.org/protobuf/proto"

	coreda "github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/types"
	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

const (
	dAefetcherTimeout = 30 * time.Second
	dAFetcherRetries  = 10

	// Parallel retrieval configuration
	defaultConcurrencyLimit = 5
	defaultPrefetchWindow   = 50
	maxRetryBackoff         = 5 * time.Second
)

var (
	// TestPrefetchWindow can be set by tests to override the default prefetch window
	TestPrefetchWindow int = 0
)

// RetrievalResult holds the result of retrieving blobs from a specific DA height
type RetrievalResult struct {
	Height      uint64
	Data        [][]byte
	Error       error
	RetryCount  int
	LastAttempt time.Time
}

// RetryInfo tracks retry state persistently across attempts
type RetryInfo struct {
	RetryCount         int
	LastAttempt        time.Time
	NextRetryTime      time.Time
	IsHeightFromFuture bool
}

// Retriever manages parallel retrieval of blocks from DA layer with simplified design
type Retriever struct {
	manager        *Manager
	prefetchWindow int
	ctx            context.Context
	cancel         context.CancelFunc

	// Fixed concurrency control
	concurrencyLimit *semaphore.Weighted

	// Processing state
	scheduledUntil uint64     // Last height scheduled for retrieval
	nextToProcess  uint64     // Next height to process in order
	mu             sync.Mutex // Protects scheduledUntil and nextToProcess

	// In-flight tracking
	inFlight   map[uint64]*RetrievalResult
	inFlightMu sync.RWMutex

	// Retry tracking - persistent across attempts
	retryInfo   map[uint64]*RetryInfo
	retryInfoMu sync.RWMutex

	// Processing synchronization - prevents concurrent processing races
	processingMu sync.Mutex

	// Work notification - efficient signaling without CPU spinning
	workAvailable *sync.Cond
	workSignal    chan struct{} // Buffered channel for work notifications

	// Goroutine lifecycle
	dispatcher sync.WaitGroup
	processor  sync.WaitGroup
}

// NewRetriever creates a new parallel retriever instance
func NewRetriever(manager *Manager, parentCtx context.Context) *Retriever {
	ctx, cancel := context.WithCancel(parentCtx)

	// Use test override if set, otherwise use default
	prefetchWindow := defaultPrefetchWindow
	if TestPrefetchWindow > 0 {
		prefetchWindow = TestPrefetchWindow
	}

	startHeight := manager.daHeight.Load()

	mu := &sync.Mutex{}
	retriever := &Retriever{
		manager:          manager,
		prefetchWindow:   prefetchWindow,
		ctx:              ctx,
		cancel:           cancel,
		concurrencyLimit: semaphore.NewWeighted(int64(defaultConcurrencyLimit)),
		scheduledUntil:   startHeight - 1, // Will start scheduling from startHeight
		nextToProcess:    startHeight,
		inFlight:         make(map[uint64]*RetrievalResult),
		retryInfo:        make(map[uint64]*RetryInfo),
		workSignal:       make(chan struct{}, 1), // Buffered to avoid blocking
	}
	retriever.workAvailable = sync.NewCond(mu)
	return retriever
}

// RetrieveLoop is responsible for interacting with DA layer using parallel retrieval.
func (m *Manager) RetrieveLoop(ctx context.Context) {
	retriever := NewRetriever(m, ctx)
	defer retriever.Stop()

	// Start the parallel retrieval process
	retriever.Start()

	// Wait for context cancellation
	<-ctx.Done()
}

// Start begins the parallel retrieval process
func (pr *Retriever) Start() {
	// Start dispatcher goroutine to schedule heights
	pr.dispatcher.Add(1)
	go pr.dispatchHeights()

	// Start processor goroutine to handle results
	pr.processor.Add(1)
	go pr.processResults()

	// Start metrics updater goroutine
	go pr.updateMetrics()
}

// Stop gracefully shuts down the parallel retriever
func (pr *Retriever) Stop() {
	// Cancel context to signal all goroutines to stop
	pr.cancel()

	// Wait for goroutines to finish
	pr.dispatcher.Wait()
	pr.processor.Wait()
}

// dispatchHeights manages height scheduling based on processing state
func (pr *Retriever) dispatchHeights() {
	defer pr.dispatcher.Done()
	
	// Use a timer for retry checks instead of constant ticker
	retryTimer := time.NewTimer(500 * time.Millisecond)
	defer retryTimer.Stop()

	for {
		select {
		case <-pr.ctx.Done():
			return
		case <-pr.manager.retrieveCh:
			// Explicit trigger from manager - process immediately
		case <-pr.workSignal:
			// Work available notification - drain the channel
			select {
			case <-pr.workSignal:
			default:
			}
		case <-retryTimer.C:
			// Periodic check for retries
			retryTimer.Reset(500 * time.Millisecond)
		}

		// Schedule heights up to prefetch window from nextToProcess
		pr.mu.Lock()
		nextToProcess := pr.nextToProcess
		scheduledUntil := pr.scheduledUntil
		targetHeight := nextToProcess + uint64(pr.prefetchWindow) - 1
		pr.mu.Unlock()

		// First, check for heights that need to be retried
		now := time.Now()
		pr.retryInfoMu.RLock()
		var heightsToRetry []uint64
		for height, info := range pr.retryInfo {
			if height >= nextToProcess && height <= targetHeight && now.After(info.NextRetryTime) {
				// Check if not already in flight
				pr.inFlightMu.RLock()
				_, inFlight := pr.inFlight[height]
				pr.inFlightMu.RUnlock()

				if !inFlight {
					heightsToRetry = append(heightsToRetry, height)
				}
			}
		}
		pr.retryInfoMu.RUnlock()

		// Schedule retries first (they're more important)
		for _, height := range heightsToRetry {
			// Try to acquire concurrency permit
			if !pr.concurrencyLimit.TryAcquire(1) {
				// Concurrency limit reached, try again later
				break
			}

			// Launch retrieval goroutine for this height
			go pr.retrieveHeight(height)

			// Update pending jobs metric
			if pr.manager.metrics != nil {
				pr.manager.metrics.ParallelRetrievalPendingJobs.Add(1)
			}
		}

		// Schedule any unscheduled heights within the window
		hasMoreWork := false
		for height := scheduledUntil + 1; height <= targetHeight; height++ {
			// Check if already in flight
			pr.inFlightMu.RLock()
			_, exists := pr.inFlight[height]
			pr.inFlightMu.RUnlock()

			if exists {
				continue
			}

			// Check if it's in retry state (already handled above)
			pr.retryInfoMu.RLock()
			_, hasRetryInfo := pr.retryInfo[height]
			pr.retryInfoMu.RUnlock()

			if hasRetryInfo {
				continue
			}

			// Try to acquire concurrency permit with backpressure handling
			if !pr.concurrencyLimit.TryAcquire(1) {
				// Concurrency limit reached, signal that more work is available
				hasMoreWork = true
				break
			}

			// Update scheduledUntil
			pr.mu.Lock()
			pr.scheduledUntil = height
			pr.mu.Unlock()

			// Launch retrieval goroutine for this height
			go pr.retrieveHeight(height)

			// Update pending jobs metric
			if pr.manager.metrics != nil {
				pr.manager.metrics.ParallelRetrievalPendingJobs.Add(1)
			}
		}
		
		// If we have more work but hit concurrency limit, ensure we check again soon
		if hasMoreWork {
			select {
			case pr.workSignal <- struct{}{}:
			default:
			}
		}
	}
}

// retrieveHeight retrieves data for a specific height with retry logic
func (pr *Retriever) retrieveHeight(height uint64) {
	defer pr.concurrencyLimit.Release(1)
	defer func() {
		// Update pending jobs metric
		if pr.manager.metrics != nil {
			pr.manager.metrics.ParallelRetrievalPendingJobs.Add(-1)
		}
		// Signal that a worker slot is available for more work
		select {
		case pr.workSignal <- struct{}{}:
		default:
		}
	}()

	// Get or create retry info for this height
	pr.retryInfoMu.Lock()
	info, exists := pr.retryInfo[height]
	if !exists {
		info = &RetryInfo{
			RetryCount:  0,
			LastAttempt: time.Now(),
		}
		pr.retryInfo[height] = info
	}
	pr.retryInfoMu.Unlock()

	// Fetch the height with concurrent namespace calls
	result := pr.fetchHeightConcurrently(pr.ctx, height)

	// Update result with persistent retry count
	result.RetryCount = info.RetryCount
	result.LastAttempt = time.Now()

	// Store result in in-flight map
	pr.inFlightMu.Lock()
	pr.inFlight[height] = result
	pr.inFlightMu.Unlock()

	// Trigger processing check
	pr.checkAndProcessResults()

	// Handle errors and schedule retries
	if result.Error != nil && pr.ctx.Err() == nil {
		isHeightFromFuture := pr.manager.areAllErrorsHeightFromFuture(result.Error)

		// Update retry info
		pr.retryInfoMu.Lock()
		info.RetryCount++
		info.LastAttempt = time.Now()
		info.IsHeightFromFuture = isHeightFromFuture

		if isHeightFromFuture {
			// For height-from-future, use a longer backoff
			backoff := 2 * time.Second
			info.NextRetryTime = time.Now().Add(backoff)
		} else if info.RetryCount < dAFetcherRetries {
			// For other errors, use exponential backoff
			backoff := time.Duration(info.RetryCount) * time.Second
			if backoff > maxRetryBackoff {
				backoff = maxRetryBackoff
			}
			info.NextRetryTime = time.Now().Add(backoff)
		}
		pr.retryInfoMu.Unlock()

		// Schedule retry removal from in-flight
		if isHeightFromFuture || info.RetryCount < dAFetcherRetries {
			// Remove from in-flight after a short delay to allow processing to complete
			time.AfterFunc(100*time.Millisecond, func() {
				pr.inFlightMu.Lock()
				delete(pr.inFlight, height)
				pr.inFlightMu.Unlock()
			})
		}
	} else if result.Error == nil {
		// Success - clean up retry info
		pr.retryInfoMu.Lock()
		delete(pr.retryInfo, height)
		pr.retryInfoMu.Unlock()
	}
}

// processResults monitors for results to process in order
func (pr *Retriever) processResults() {
	defer pr.processor.Done()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-pr.ctx.Done():
			return
		case <-ticker.C:
			pr.checkAndProcessResults()
		}
	}
}

// checkAndProcessResults processes available results in order
func (pr *Retriever) checkAndProcessResults() {
	// Use processingMu to prevent concurrent execution races
	pr.processingMu.Lock()
	defer pr.processingMu.Unlock()

	for {
		pr.mu.Lock()
		nextToProcess := pr.nextToProcess
		pr.mu.Unlock()

		// Check if we have the next height ready
		pr.inFlightMu.RLock()
		result, exists := pr.inFlight[nextToProcess]
		pr.inFlightMu.RUnlock()

		if !exists || result == nil {
			// Next height not ready yet
			break
		}

		// Get retry info for this height
		pr.retryInfoMu.RLock()
		retryInfo, hasRetryInfo := pr.retryInfo[nextToProcess]
		pr.retryInfoMu.RUnlock()

		// Process based on result status
		shouldAdvance := false
		shouldRetry := false

		if result.Error != nil && pr.ctx.Err() == nil {
			// Use persistent retry count if available
			retryCount := result.RetryCount
			if hasRetryInfo {
				retryCount = retryInfo.RetryCount
			}

			// Check if it's a height-from-future error
			if pr.manager.areAllErrorsHeightFromFuture(result.Error) {
				// Don't advance for height-from-future errors, but do retry
				pr.manager.logger.Debug().
					Uint64("daHeight", result.Height).
					Int("retries", retryCount).
					Msg("height from future, will retry")
				shouldRetry = true
			} else if retryCount >= dAFetcherRetries {
				// Max retries reached, log error but advance
				pr.manager.logger.Error().
					Uint64("daHeight", result.Height).
					Str("errors", result.Error.Error()).
					Int("retries", retryCount).
					Msg("failed to retrieve data from DALC after max retries")
				shouldAdvance = true
			} else {
				// Still have retries left, will retry
				pr.manager.logger.Debug().
					Uint64("daHeight", result.Height).
					Int("retries", retryCount).
					Int("maxRetries", dAFetcherRetries).
					Msg("retrieval failed, will retry")
				shouldRetry = true
			}
		} else if result.Error == nil {
			// Success - process and advance
			if len(result.Data) > 0 {
				pr.manager.processRetrievedData(pr.ctx, result.Data, result.Height)
			} else {
				pr.manager.logger.Debug().Uint64("daHeight", result.Height).Msg("no blob data found (NotFound)")
			}
			shouldAdvance = true
		}

		if shouldAdvance {
			// Remove from in-flight map
			pr.inFlightMu.Lock()
			delete(pr.inFlight, nextToProcess)
			pr.inFlightMu.Unlock()

			// Clean up retry info
			pr.retryInfoMu.Lock()
			delete(pr.retryInfo, nextToProcess)
			pr.retryInfoMu.Unlock()

			// Advance both pointers
			pr.mu.Lock()
			pr.nextToProcess++
			pr.mu.Unlock()

			// Update manager's DA height
			pr.manager.daHeight.Store(nextToProcess + 1)
			
			// Signal that progress was made and new heights can be scheduled
			select {
			case pr.workSignal <- struct{}{}:
			default:
			}
		} else if shouldRetry {
			// Can't advance yet, but ensure retry is scheduled
			// The retry scheduling is already handled in retrieveHeight
			break
		} else {
			// Can't advance yet, stop processing
			break
		}
	}
}

// fetchHeightConcurrently retrieves blobs from a specific DA height using concurrent namespace calls
func (pr *Retriever) fetchHeightConcurrently(ctx context.Context, height uint64) *RetrievalResult {
	start := time.Now()
	fetchCtx, cancel := context.WithTimeout(ctx, dAefetcherTimeout)
	defer cancel()

	// Record latency metric at the end
	defer func() {
		if pr.manager.metrics != nil {
			pr.manager.metrics.ParallelRetrievalLatency.Observe(time.Since(start).Seconds())
		}
	}()

	// Record DA retrieval attempt
	pr.manager.recordDAMetrics("retrieval", DAModeRetry)

	// Single attempt (retry is handled at higher level now)
	select {
	case <-ctx.Done():
		return &RetrievalResult{Height: height, Error: ctx.Err()}
	case <-fetchCtx.Done():
		return &RetrievalResult{Height: height, Error: fetchCtx.Err()}
	default:
	}

	combinedResult, fetchErr := pr.manager.fetchBlobsConcurrently(fetchCtx, height)
	if fetchErr == nil {
		// Record successful DA retrieval
		pr.manager.recordDAMetrics("retrieval", DAModeSuccess)
		return &RetrievalResult{Height: height, Data: combinedResult.Data, Error: nil}
	}

	// Return error (retry logic is handled in retrieveHeight)
	return &RetrievalResult{Height: height, Error: fetchErr}
}

// fetchBlobsConcurrently retrieves blobs using concurrent calls to header and data namespaces
func (m *Manager) fetchBlobsConcurrently(ctx context.Context, daHeight uint64) (coreda.ResultRetrieve, error) {
	var err error

	// TODO: Remove this once XO resets their testnet
	// Check if we should still try the old namespace for backward compatibility
	if !m.namespaceMigrationCompleted.Load() {
		// First, try the legacy namespace if we haven't completed migration
		legacyNamespace := []byte(m.config.DA.Namespace)
		if len(legacyNamespace) > 0 {
			legacyRes := types.RetrieveWithHelpers(ctx, m.da, m.logger, daHeight, legacyNamespace)

			// Handle legacy namespace errors
			if legacyRes.Code == coreda.StatusError {
				m.recordDAMetrics("retrieval", DAModeFail)
				err = fmt.Errorf("failed to retrieve from legacy namespace: %s", legacyRes.Message)
				return legacyRes, err
			}

			if legacyRes.Code == coreda.StatusHeightFromFuture {
				err = fmt.Errorf("%w: height from future", coreda.ErrHeightFromFuture)
				return coreda.ResultRetrieve{BaseResult: coreda.BaseResult{Code: coreda.StatusHeightFromFuture}}, err
			}

			// If legacy namespace has data, use it and return
			if legacyRes.Code == coreda.StatusSuccess {
				m.logger.Debug().Uint64("daHeight", daHeight).Msg("found data in legacy namespace")
				return legacyRes, nil
			}

			// Legacy namespace returned not found, so try new namespaces
			m.logger.Debug().Uint64("daHeight", daHeight).Msg("no data in legacy namespace, trying new namespaces")
		}
	}

	// Channels for concurrent namespace fetching
	type namespaceResult struct {
		res coreda.ResultRetrieve
		err error
	}

	headerCh := make(chan namespaceResult, 1)
	dataCh := make(chan namespaceResult, 1)

	// Retrieve from header namespace concurrently
	go func() {
		headerNamespace := []byte(m.config.DA.GetHeaderNamespace())
		res := types.RetrieveWithHelpers(ctx, m.da, m.logger, daHeight, headerNamespace)
		var err error
		if res.Code == coreda.StatusError {
			err = fmt.Errorf("header namespace error: %s", res.Message)
		}
		select {
		case headerCh <- namespaceResult{res: res, err: err}:
		case <-ctx.Done():
			return
		}
	}()

	// Retrieve from data namespace concurrently
	go func() {
		dataNamespace := []byte(m.config.DA.GetDataNamespace())
		res := types.RetrieveWithHelpers(ctx, m.da, m.logger, daHeight, dataNamespace)
		var err error
		if res.Code == coreda.StatusError {
			err = fmt.Errorf("data namespace error: %s", res.Message)
		}
		select {
		case dataCh <- namespaceResult{res: res, err: err}:
		case <-ctx.Done():
			return
		}
	}()

	// Wait for both calls to complete with context cancellation support
	var headerResult, dataResult namespaceResult
	for i := 0; i < 2; i++ {
		select {
		case result := <-headerCh:
			headerResult = result
		case result := <-dataCh:
			dataResult = result
		case <-ctx.Done():
			return coreda.ResultRetrieve{}, ctx.Err()
		}
	}

	headerRes := headerResult.res
	headerErr := headerResult.err
	dataRes := dataResult.res
	dataErr := dataResult.err

	// Handle errors
	if headerErr != nil && dataErr != nil {
		// Both failed
		m.recordDAMetrics("retrieval", DAModeFail)
		err = fmt.Errorf("failed to retrieve from both namespaces - %w, %w", headerErr, dataErr)
		return headerRes, err
	}

	if headerRes.Code == coreda.StatusHeightFromFuture || dataRes.Code == coreda.StatusHeightFromFuture {
		// At least one is from future
		err = fmt.Errorf("%w: height from future", coreda.ErrHeightFromFuture)
		return coreda.ResultRetrieve{BaseResult: coreda.BaseResult{Code: coreda.StatusHeightFromFuture}}, err
	}

	// Combine successful results
	combinedResult := coreda.ResultRetrieve{
		BaseResult: coreda.BaseResult{
			Code:   coreda.StatusSuccess,
			Height: daHeight,
		},
		Data: make([][]byte, 0),
	}

	// Add header data if successful
	if headerRes.Code == coreda.StatusSuccess {
		combinedResult.Data = append(combinedResult.Data, headerRes.Data...)
		if len(headerRes.IDs) > 0 {
			combinedResult.IDs = append(combinedResult.IDs, headerRes.IDs...)
		}
	}

	// Add data blobs if successful
	if dataRes.Code == coreda.StatusSuccess {
		combinedResult.Data = append(combinedResult.Data, dataRes.Data...)
		if len(dataRes.IDs) > 0 {
			combinedResult.IDs = append(combinedResult.IDs, dataRes.IDs...)
		}
	}

	// Handle not found cases and migration completion
	if headerRes.Code == coreda.StatusNotFound && dataRes.Code == coreda.StatusNotFound {
		combinedResult.Code = coreda.StatusNotFound
		combinedResult.Message = "no blobs found in either namespace"

		// If we haven't completed migration and found no data in new namespaces,
		// mark migration as complete to avoid future legacy namespace checks
		if !m.namespaceMigrationCompleted.Load() {
			if err := m.setNamespaceMigrationCompleted(ctx); err != nil {
				m.logger.Error().Err(err).Msg("failed to mark namespace migration as completed")
			} else {
				m.logger.Info().Uint64("daHeight", daHeight).Msg("marked namespace migration as completed - no more legacy namespace checks")
			}
		}
	} else if (headerRes.Code == coreda.StatusSuccess || dataRes.Code == coreda.StatusSuccess) && !m.namespaceMigrationCompleted.Load() {
		// Found data in new namespaces, mark migration as complete
		if err := m.setNamespaceMigrationCompleted(ctx); err != nil {
			m.logger.Error().Err(err).Msg("failed to mark namespace migration as completed")
		} else {
			m.logger.Info().Uint64("daHeight", daHeight).Msg("found data in new namespaces - marked migration as completed")
		}
	}

	return combinedResult, err
}

// updateMetrics periodically updates metrics for parallel retrieval
func (pr *Retriever) updateMetrics() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-pr.ctx.Done():
			return
		case <-ticker.C:
			if pr.manager.metrics != nil {
				// Update in-flight size metric
				pr.inFlightMu.RLock()
				inFlightSize := len(pr.inFlight)
				pr.inFlightMu.RUnlock()
				pr.manager.metrics.ParallelRetrievalBufferSize.Set(float64(inFlightSize))

				// Update concurrency metric
				pr.manager.metrics.ParallelRetrievalWorkers.Set(float64(defaultConcurrencyLimit))
			}
		}
	}
}

// processRetrievedData processes successfully retrieved data blobs
func (m *Manager) processRetrievedData(ctx context.Context, data [][]byte, daHeight uint64) {
	if len(data) == 0 {
		m.logger.Debug().Uint64("daHeight", daHeight).Str("reason", "no data returned").Msg("no blob data found")
		return
	}

	m.logger.Debug().Int("n", len(data)).Uint64("daHeight", daHeight).Msg("retrieved potential blob data")

	for _, bz := range data {
		if len(bz) == 0 {
			m.logger.Debug().Uint64("daHeight", daHeight).Msg("ignoring nil or empty blob")
			continue
		}
		if m.handlePotentialHeader(ctx, bz, daHeight) {
			continue
		}
		m.handlePotentialData(ctx, bz, daHeight)
	}
}

// processNextDAHeaderAndData is responsible for retrieving a header and data from the DA layer.
// It returns an error if the context is done or if the DA layer returns an error.
// This method is used for sequential retrieval mode (e.g., in tests).
func (m *Manager) processNextDAHeaderAndData(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	daHeight := m.daHeight.Load()

	var err error
	m.logger.Debug().Uint64("daHeight", daHeight).Msg("trying to retrieve data from DA")
	for r := 0; r < dAFetcherRetries; r++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Use the unified concurrent fetch method
		blobsResp, fetchErr := m.fetchBlobsConcurrently(ctx, daHeight)
		if fetchErr == nil {
			// Record successful DA retrieval
			m.recordDAMetrics("retrieval", DAModeSuccess)

			if blobsResp.Code == coreda.StatusNotFound {
				m.logger.Debug().Uint64("daHeight", daHeight).Str("reason", blobsResp.Message).Msg("no blob data found")
				return nil
			}
			m.logger.Debug().Int("n", len(blobsResp.Data)).Uint64("daHeight", daHeight).Msg("retrieved potential blob data")
			for _, bz := range blobsResp.Data {
				if len(bz) == 0 {
					m.logger.Debug().Uint64("daHeight", daHeight).Msg("ignoring nil or empty blob")
					continue
				}
				if m.handlePotentialHeader(ctx, bz, daHeight) {
					continue
				}
				m.handlePotentialData(ctx, bz, daHeight)
			}
			return nil
		} else if strings.Contains(fetchErr.Error(), coreda.ErrHeightFromFuture.Error()) {
			m.logger.Debug().Uint64("daHeight", daHeight).Str("reason", fetchErr.Error()).Msg("height from future")
			return fetchErr
		}

		// Track the error
		err = errors.Join(err, fetchErr)
		// Delay before retrying
		select {
		case <-ctx.Done():
			return err
		case <-time.After(100 * time.Millisecond):
		}
	}
	return err
}

// handlePotentialHeader tries to decode and process a header. Returns true if successful or skipped, false if not a header.
func (m *Manager) handlePotentialHeader(ctx context.Context, bz []byte, daHeight uint64) bool {
	header := new(types.SignedHeader)
	var headerPb pb.SignedHeader

	if err := proto.Unmarshal(bz, &headerPb); err != nil {
		m.logger.Debug().Err(err).Msg("failed to unmarshal header")
		return false
	}

	if err := header.FromProto(&headerPb); err != nil {
		// treat as handled, but not valid
		m.logger.Debug().Err(err).Msg("failed to decode unmarshalled header")
		return true
	}

	// set custom verifier to do correct header verification
	header.SetCustomVerifier(m.signaturePayloadProvider)

	// Stronger validation: check for obviously invalid headers using ValidateBasic
	if err := header.ValidateBasic(); err != nil {
		m.logger.Debug().Uint64("daHeight", daHeight).Err(err).Msg("blob does not look like a valid header")
		return false
	}

	// early validation to reject junk headers
	if !m.isUsingExpectedSingleSequencer(header) {
		m.logger.Debug().
			Uint64("headerHeight", header.Height()).
			Str("headerHash", header.Hash().String()).
			Msg("skipping header from unexpected sequencer")
		return true
	}
	headerHash := header.Hash().String()
	m.headerCache.SetDAIncluded(headerHash, daHeight)
	m.sendNonBlockingSignalToDAIncluderCh()
	m.logger.Info().Uint64("headerHeight", header.Height()).Str("headerHash", headerHash).Msg("header marked as DA included")
	if !m.headerCache.IsSeen(headerHash) {
		select {
		case <-ctx.Done():
			return true
		default:
			m.logger.Warn().Uint64("daHeight", daHeight).Msg("headerInCh backlog full, dropping header")
		}
		m.headerInCh <- NewHeaderEvent{header, daHeight}
	}
	return true
}

// handlePotentialData tries to decode and process a data. No return value.
func (m *Manager) handlePotentialData(ctx context.Context, bz []byte, daHeight uint64) {
	var signedData types.SignedData
	err := signedData.UnmarshalBinary(bz)
	if err != nil {
		m.logger.Debug().Err(err).Msg("failed to unmarshal signed data")
		return
	}
	if len(signedData.Txs) == 0 {
		m.logger.Debug().Uint64("daHeight", daHeight).Msg("ignoring empty signed data")
		return
	}

	// Early validation to reject junk data
	if !m.isValidSignedData(&signedData) {
		m.logger.Debug().Uint64("daHeight", daHeight).Msg("invalid data signature")
		return
	}

	dataHashStr := signedData.Data.DACommitment().String()
	m.dataCache.SetDAIncluded(dataHashStr, daHeight)
	m.sendNonBlockingSignalToDAIncluderCh()
	m.logger.Info().Str("dataHash", dataHashStr).Uint64("daHeight", daHeight).Uint64("height", signedData.Height()).Msg("signed data marked as DA included")
	if !m.dataCache.IsSeen(dataHashStr) {
		select {
		case <-ctx.Done():
			return
		default:
			m.logger.Warn().Uint64("daHeight", daHeight).Msg("dataInCh backlog full, dropping signed data")
		}
		m.dataInCh <- NewDataEvent{&signedData.Data, daHeight}
	}
}

// areAllErrorsHeightFromFuture checks if all errors in a joined error are ErrHeightFromFutureStr
func (m *Manager) areAllErrorsHeightFromFuture(err error) bool {
	if err == nil {
		return false
	}

	// Check if the error itself is ErrHeightFromFutureStr
	if strings.Contains(err.Error(), ErrHeightFromFutureStr.Error()) {
		return true
	}

	// If it's a joined error, check each error recursively
	if joinedErr, ok := err.(interface{ Unwrap() []error }); ok {
		for _, e := range joinedErr.Unwrap() {
			if !m.areAllErrorsHeightFromFuture(e) {
				return false
			}
		}
		return true
	}

	return false
}
