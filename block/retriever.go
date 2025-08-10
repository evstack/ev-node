package block

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

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
	defaultResultBufferSize = 100
)

var (
	// TestPrefetchWindow can be set by tests to override the default prefetch window
	TestPrefetchWindow int = 0
)

// RetrievalResult holds the result of retrieving blobs from a specific DA height
type RetrievalResult struct {
	Height uint64
	Data   [][]byte
	Error  error
}

// ParallelRetriever manages parallel retrieval of blocks from DA layer
type ParallelRetriever struct {
	manager          *Manager
	concurrencyLimit int
	prefetchWindow   int
	workChan         chan uint64
	resultChan       chan *RetrievalResult
	workers          sync.WaitGroup
	processors       sync.WaitGroup // Track result processor goroutine
	ctx              context.Context
	cancel           context.CancelFunc

	// Result ordering and buffering
	resultBuffer      map[uint64]*RetrievalResult
	resultMutex       sync.RWMutex
	nextHeight        uint64
	maxBufferSize     int             // Prevent unbounded growth
	pendingHeights    map[uint64]bool // Track dispatched heights
	pendingHeightsMux sync.RWMutex
}

// NewParallelRetriever creates a new parallel retriever instance
func NewParallelRetriever(manager *Manager, parentCtx context.Context) *ParallelRetriever {
	ctx, cancel := context.WithCancel(parentCtx)
	
	// Use test override if set, otherwise use default
	prefetchWindow := defaultPrefetchWindow
	if TestPrefetchWindow > 0 {
		prefetchWindow = TestPrefetchWindow
	}
	
	return &ParallelRetriever{
		manager:          manager,
		concurrencyLimit: defaultConcurrencyLimit,
		prefetchWindow:   prefetchWindow,
		workChan:         make(chan uint64, defaultResultBufferSize),
		resultChan:       make(chan *RetrievalResult, defaultResultBufferSize),
		ctx:              ctx,
		cancel:           cancel,
		resultBuffer:     make(map[uint64]*RetrievalResult),
		nextHeight:       manager.daHeight.Load(),
		maxBufferSize:    defaultResultBufferSize * 2, // Allow some buffering but prevent unbounded growth
		pendingHeights:   make(map[uint64]bool),
	}
}

// RetrieveLoop is responsible for interacting with DA layer using parallel retrieval.
func (m *Manager) RetrieveLoop(ctx context.Context) {
	retriever := NewParallelRetriever(m, ctx)
	defer retriever.Stop()

	// Start the parallel retrieval process
	retriever.Start()

	// Wait for context cancellation
	<-ctx.Done()
}

// Start begins the parallel retrieval process
func (pr *ParallelRetriever) Start() {
	// Start worker pool
	for i := 0; i < pr.concurrencyLimit; i++ {
		pr.workers.Add(1)
		go pr.worker()
	}

	// Update worker count metric
	if pr.manager.metrics != nil {
		pr.manager.metrics.ParallelRetrievalWorkers.Set(float64(pr.concurrencyLimit))
	}

	// Start height dispatcher goroutine
	go pr.dispatchHeights()

	// Start result processor goroutine (non-blocking)
	pr.processors.Add(1)
	go pr.processResults()

	// Start metrics updater goroutine
	go pr.updateMetrics()
}

// Stop gracefully shuts down the parallel retriever
func (pr *ParallelRetriever) Stop() {
	// Cancel context to signal all goroutines to stop
	pr.cancel()

	// Close work channel to signal workers to exit
	close(pr.workChan)

	// Wait for all workers to finish
	pr.workers.Wait()

	// Close result channel after workers are done
	close(pr.resultChan)

	// Wait for result processor to finish
	pr.processors.Wait()
}

// dispatchHeights manages the work distribution to workers
func (pr *ParallelRetriever) dispatchHeights() {
	blobsFoundCh := make(chan struct{}, 1)
	defer close(blobsFoundCh)

	for {
		select {
		case <-pr.ctx.Done():
			return
		case <-pr.manager.retrieveCh:
		case <-blobsFoundCh:
		}

		// Check buffer size before dispatching more work
		pr.resultMutex.RLock()
		bufferSize := len(pr.resultBuffer)
		pr.resultMutex.RUnlock()

		if bufferSize >= pr.maxBufferSize {
			// Buffer is full, wait before dispatching more
			select {
			case <-pr.ctx.Done():
				return
			case <-time.After(100 * time.Millisecond):
				continue
			}
		}

		// Dispatch work for current height and prefetch window
		currentHeight := pr.manager.daHeight.Load()
		for i := uint64(0); i < uint64(pr.prefetchWindow); i++ {
			height := currentHeight + i

			// Check if height is already being processed
			pr.pendingHeightsMux.RLock()
			alreadyPending := pr.pendingHeights[height]
			pr.pendingHeightsMux.RUnlock()

			if alreadyPending {
				continue
			}

			// Try to dispatch the height
			select {
			case pr.workChan <- height:
				// Mark height as pending
				pr.pendingHeightsMux.Lock()
				pr.pendingHeights[height] = true
				pr.pendingHeightsMux.Unlock()
				
				// Update pending jobs metric
				if pr.manager.metrics != nil {
					pr.manager.metrics.ParallelRetrievalPendingJobs.Add(1)
				}
			case <-pr.ctx.Done():
				return
			default:
				// Channel full, will retry in next iteration
				break
			}
		}

		// Signal to continue dispatching
		select {
		case blobsFoundCh <- struct{}{}:
		default:
		}
	}
}

// worker processes DA height retrieval tasks
func (pr *ParallelRetriever) worker() {
	defer pr.workers.Done()

	for {
		select {
		case height, ok := <-pr.workChan:
			if !ok {
				return
			}

			// Fetch the height
			result := pr.fetchHeightConcurrently(pr.ctx, height)

			// Remove from pending heights
			pr.pendingHeightsMux.Lock()
			delete(pr.pendingHeights, height)
			pr.pendingHeightsMux.Unlock()
			
			// Update pending jobs metric
			if pr.manager.metrics != nil {
				pr.manager.metrics.ParallelRetrievalPendingJobs.Add(-1)
			}

			// Send result
			select {
			case pr.resultChan <- result:
			case <-pr.ctx.Done():
				return
			}
		case <-pr.ctx.Done():
			return
		}
	}
}

// processResults handles retrieved results in order
func (pr *ParallelRetriever) processResults() {
	defer pr.processors.Done()

	for {
		select {
		case <-pr.ctx.Done():
			return
		case result, ok := <-pr.resultChan:
			if !ok {
				// Channel closed, exit
				return
			}
			if result == nil {
				continue
			}

			// Buffer the result
			pr.resultMutex.Lock()
			pr.resultBuffer[result.Height] = result
			pr.resultMutex.Unlock()

			// Process results in order
			pr.processOrderedResults()
		}
	}
}

// processOrderedResults processes buffered results in height order
func (pr *ParallelRetriever) processOrderedResults() {
	for {
		// Get and remove the next result while holding the lock
		pr.resultMutex.Lock()
		result, exists := pr.resultBuffer[pr.nextHeight]
		if !exists {
			pr.resultMutex.Unlock()
			break
		}

		// Remove from buffer and advance height while still holding lock
		delete(pr.resultBuffer, pr.nextHeight)
		pr.nextHeight++
		pr.resultMutex.Unlock()

		// Process the result without holding the lock to avoid blocking other operations
		if result.Error != nil && pr.ctx.Err() == nil {
			// if the requested da height is not yet available, wait silently, otherwise log the error
			if !pr.manager.areAllErrorsHeightFromFuture(result.Error) {
				pr.manager.logger.Error().
					Uint64("daHeight", result.Height).
					Str("errors", result.Error.Error()).
					Msg("failed to retrieve data from DALC")
			}
		} else if result.Error == nil {
			// Process successful retrieval
			pr.manager.processRetrievedData(pr.ctx, result.Data, result.Height)
			pr.manager.daHeight.Store(result.Height + 1)
		}
	}
}

// fetchHeightConcurrently retrieves blobs from a specific DA height using concurrent namespace calls
func (pr *ParallelRetriever) fetchHeightConcurrently(ctx context.Context, height uint64) *RetrievalResult {
	var err error
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

	// Retry logic for the entire height
	for r := 0; r < dAFetcherRetries; r++ {
		select {
		case <-ctx.Done():
			return &RetrievalResult{Height: height, Error: ctx.Err()}
		case <-fetchCtx.Done():
			return &RetrievalResult{Height: height, Error: fetchCtx.Err()}
		default:
		}

		combinedResult, fetchErr := pr.fetchBlobsConcurrently(fetchCtx, height)
		if fetchErr == nil {
			// Record successful DA retrieval
			pr.manager.recordDAMetrics("retrieval", DAModeSuccess)
			return &RetrievalResult{Height: height, Data: combinedResult.Data, Error: nil}
		} else if strings.Contains(fetchErr.Error(), coreda.ErrHeightFromFuture.Error()) {
			return &RetrievalResult{Height: height, Error: fetchErr}
		}

		// Track the error and retry
		err = errors.Join(err, fetchErr)
		select {
		case <-ctx.Done():
			return &RetrievalResult{Height: height, Error: err}
		case <-time.After(100 * time.Millisecond):
		}
	}

	return &RetrievalResult{Height: height, Error: err}
}

// fetchBlobsConcurrently retrieves blobs using concurrent calls to header and data namespaces
func (pr *ParallelRetriever) fetchBlobsConcurrently(ctx context.Context, daHeight uint64) (coreda.ResultRetrieve, error) {
	var err error

	// TODO: Remove this once XO resets their testnet
	// Check if we should still try the old namespace for backward compatibility
	if !pr.manager.namespaceMigrationCompleted.Load() {
		// First, try the legacy namespace if we haven't completed migration
		legacyNamespace := []byte(pr.manager.config.DA.Namespace)
		if len(legacyNamespace) > 0 {
			legacyRes := types.RetrieveWithHelpers(ctx, pr.manager.da, pr.manager.logger, daHeight, legacyNamespace)

			// Handle legacy namespace errors
			if legacyRes.Code == coreda.StatusError {
				pr.manager.recordDAMetrics("retrieval", DAModeFail)
				err = fmt.Errorf("failed to retrieve from legacy namespace: %s", legacyRes.Message)
				return legacyRes, err
			}

			if legacyRes.Code == coreda.StatusHeightFromFuture {
				err = fmt.Errorf("%w: height from future", coreda.ErrHeightFromFuture)
				return coreda.ResultRetrieve{BaseResult: coreda.BaseResult{Code: coreda.StatusHeightFromFuture}}, err
			}

			// If legacy namespace has data, use it and return
			if legacyRes.Code == coreda.StatusSuccess {
				pr.manager.logger.Debug().Uint64("daHeight", daHeight).Msg("found data in legacy namespace")
				return legacyRes, nil
			}

			// Legacy namespace returned not found, so try new namespaces
			pr.manager.logger.Debug().Uint64("daHeight", daHeight).Msg("no data in legacy namespace, trying new namespaces")
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
		headerNamespace := []byte(pr.manager.config.DA.GetHeaderNamespace())
		res := types.RetrieveWithHelpers(ctx, pr.manager.da, pr.manager.logger, daHeight, headerNamespace)
		var err error
		if res.Code == coreda.StatusError {
			err = fmt.Errorf("header namespace error: %s", res.Message)
		}
		headerCh <- namespaceResult{res: res, err: err}
	}()

	// Retrieve from data namespace concurrently
	go func() {
		dataNamespace := []byte(pr.manager.config.DA.GetDataNamespace())
		res := types.RetrieveWithHelpers(ctx, pr.manager.da, pr.manager.logger, daHeight, dataNamespace)
		var err error
		if res.Code == coreda.StatusError {
			err = fmt.Errorf("data namespace error: %s", res.Message)
		}
		dataCh <- namespaceResult{res: res, err: err}
	}()

	// Wait for both calls to complete
	headerResult := <-headerCh
	dataResult := <-dataCh

	headerRes := headerResult.res
	headerErr := headerResult.err
	dataRes := dataResult.res
	dataErr := dataResult.err

	// Handle errors
	if headerErr != nil && dataErr != nil {
		// Both failed
		pr.manager.recordDAMetrics("retrieval", DAModeFail)
		err = fmt.Errorf("failed to retrieve from both namespaces - %v, %v", headerErr, dataErr)
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
		if !pr.manager.namespaceMigrationCompleted.Load() {
			if err := pr.manager.setNamespaceMigrationCompleted(ctx); err != nil {
				pr.manager.logger.Error().Err(err).Msg("failed to mark namespace migration as completed")
			} else {
				pr.manager.logger.Info().Uint64("daHeight", daHeight).Msg("marked namespace migration as completed - no more legacy namespace checks")
			}
		}
	} else if (headerRes.Code == coreda.StatusSuccess || dataRes.Code == coreda.StatusSuccess) && !pr.manager.namespaceMigrationCompleted.Load() {
		// Found data in new namespaces, mark migration as complete
		if err := pr.manager.setNamespaceMigrationCompleted(ctx); err != nil {
			pr.manager.logger.Error().Err(err).Msg("failed to mark namespace migration as completed")
		} else {
			pr.manager.logger.Info().Uint64("daHeight", daHeight).Msg("found data in new namespaces - marked migration as completed")
		}
	}

	return combinedResult, err
}

// updateMetrics periodically updates metrics for parallel retrieval
func (pr *ParallelRetriever) updateMetrics() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-pr.ctx.Done():
			return
		case <-ticker.C:
			if pr.manager.metrics != nil {
				// Update buffer size metric
				pr.resultMutex.RLock()
				bufferSize := len(pr.resultBuffer)
				pr.resultMutex.RUnlock()
				pr.manager.metrics.ParallelRetrievalBufferSize.Set(float64(bufferSize))
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
		blobsResp, fetchErr := m.fetchBlobs(ctx, daHeight)
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

// fetchBlobs retrieves blobs from the DA layer
func (m *Manager) fetchBlobs(ctx context.Context, daHeight uint64) (coreda.ResultRetrieve, error) {
	var err error
	ctx, cancel := context.WithTimeout(ctx, dAefetcherTimeout)
	defer cancel()

	// Record DA retrieval retry attempt
	m.recordDAMetrics("retrieval", DAModeRetry)

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

	// Try to retrieve from both header and data namespaces
	headerNamespace := []byte(m.config.DA.GetHeaderNamespace())
	dataNamespace := []byte(m.config.DA.GetDataNamespace())

	// Retrieve headers
	headerRes := types.RetrieveWithHelpers(ctx, m.da, m.logger, daHeight, headerNamespace)

	// Retrieve data
	dataRes := types.RetrieveWithHelpers(ctx, m.da, m.logger, daHeight, dataNamespace)

	// Combine results or handle errors appropriately
	if headerRes.Code == coreda.StatusError && dataRes.Code == coreda.StatusError {
		// Both failed
		m.recordDAMetrics("retrieval", DAModeFail)
		err = fmt.Errorf("failed to retrieve from both namespaces - headers: %s, data: %s", headerRes.Message, dataRes.Message)
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
