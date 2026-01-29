package submitting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/block/internal/da"
	"github.com/evstack/ev-node/pkg/config"
	pkgda "github.com/evstack/ev-node/pkg/da"
	datypes "github.com/evstack/ev-node/pkg/da/types"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/rpc/server"
	"github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/types"
)

const (
	// DefaultEnvelopeCacheSize is the default size for caching signed DA envelopes.
	// This avoids re-signing headers on retry scenarios.
	DefaultEnvelopeCacheSize = 10_000

	// signingWorkerPoolSize determines how many parallel signing goroutines to use.
	// Ed25519 signing is CPU-bound, so we use GOMAXPROCS workers.
	signingWorkerPoolSize = 0 // 0 means use runtime.GOMAXPROCS(0)
)

const initialBackoff = 100 * time.Millisecond

// retryPolicy defines clamped bounds for retries and backoff.
type retryPolicy struct {
	MaxAttempts  int
	MinBackoff   time.Duration
	MaxBackoff   time.Duration
	MaxBlobBytes int
}

func defaultRetryPolicy(maxAttempts int, maxDuration time.Duration) retryPolicy {
	return retryPolicy{
		MaxAttempts:  maxAttempts,
		MinBackoff:   initialBackoff,
		MaxBackoff:   maxDuration,
		MaxBlobBytes: common.DefaultMaxBlobSize,
	}
}

// retryState holds the current retry attempt and backoff.
type retryState struct {
	Attempt int
	Backoff time.Duration
}

type retryReason int

const (
	reasonUndefined retryReason = iota
	reasonFailure
	reasonMempool
	reasonSuccess
	reasonTooBig
)

func (rs *retryState) Next(reason retryReason, pol retryPolicy) {
	switch reason {
	case reasonSuccess:
		rs.Backoff = pol.MinBackoff
	case reasonMempool:
		rs.Backoff = pol.MaxBackoff
	case reasonFailure, reasonTooBig:
		if rs.Backoff == 0 {
			rs.Backoff = pol.MinBackoff
		} else {
			rs.Backoff *= 2
		}
		rs.Backoff = clamp(rs.Backoff, pol.MinBackoff, pol.MaxBackoff)
	default:
		rs.Backoff = 0
	}
	rs.Attempt++
}

// clamp constrains a duration between min and max bounds
func clamp(v, min, max time.Duration) time.Duration {
	if min > max {
		min, max = max, min
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// DASubmitter handles DA submission operations
type DASubmitter struct {
	client  da.Client
	config  config.Config
	genesis genesis.Genesis
	options common.BlockOptions
	logger  zerolog.Logger
	metrics *common.Metrics

	// address selector for multi-account support
	addressSelector pkgda.AddressSelector

	// envelopeCache caches fully signed DA envelopes by height to avoid re-signing on retries
	envelopeCache   *lru.Cache[uint64, []byte]
	envelopeCacheMu sync.RWMutex

	// signingWorkers is the number of parallel workers for signing
	signingWorkers int
}

// NewDASubmitter creates a new DA submitter
func NewDASubmitter(
	client da.Client,
	config config.Config,
	genesis genesis.Genesis,
	options common.BlockOptions,
	metrics *common.Metrics,
	logger zerolog.Logger,
) *DASubmitter {
	daSubmitterLogger := logger.With().Str("component", "da_submitter").Logger()

	if config.RPC.EnableDAVisualization {
		visualizerLogger := logger.With().Str("component", "da_visualization").Logger()
		server.SetDAVisualizationServer(server.NewDAVisualizationServer(client, visualizerLogger, config.Node.Aggregator))
	}

	// Use NoOp metrics if nil to avoid nil checks throughout the code
	if metrics == nil {
		metrics = common.NopMetrics()
	}

	// Create address selector based on configuration
	var addressSelector pkgda.AddressSelector
	if len(config.DA.SigningAddresses) > 0 {
		addressSelector = pkgda.NewRoundRobinSelector(config.DA.SigningAddresses)
		daSubmitterLogger.Info().
			Int("num_addresses", len(config.DA.SigningAddresses)).
			Msg("initialized round-robin address selector for multi-account DA submissions")
	} else {
		addressSelector = pkgda.NewNoOpSelector()
	}

	// Create envelope cache for avoiding re-signing on retries
	envelopeCache, err := lru.New[uint64, []byte](DefaultEnvelopeCacheSize)
	if err != nil {
		daSubmitterLogger.Warn().Err(err).Msg("failed to create envelope cache, continuing without caching")
	}

	// Determine number of signing workers
	workers := signingWorkerPoolSize
	if workers <= 0 || workers > runtime.GOMAXPROCS(0) {
		workers = runtime.GOMAXPROCS(0)
	}

	return &DASubmitter{
		client:          client,
		config:          config,
		genesis:         genesis,
		options:         options,
		metrics:         metrics,
		logger:          daSubmitterLogger,
		addressSelector: addressSelector,
		envelopeCache:   envelopeCache,
		signingWorkers:  workers,
	}
}

// recordFailure records a DA submission failure in metrics
func (s *DASubmitter) recordFailure(reason common.DASubmitterFailureReason) {
	counter, ok := s.metrics.DASubmitterFailures[reason]
	if !ok {
		s.logger.Warn().Str("reason", string(reason)).Msg("unregistered failure reason, metric not recorded")
		return
	}
	counter.Add(1)

	if gauge, ok := s.metrics.DASubmitterLastFailure[reason]; ok {
		gauge.Set(float64(time.Now().Unix()))
	}
}

// SubmitHeaders submits pending headers to DA layer
func (s *DASubmitter) SubmitHeaders(ctx context.Context, headers []*types.SignedHeader, marshalledHeaders [][]byte, cache cache.Manager, signer signer.Signer) error {
	if len(headers) == 0 {
		return nil
	}

	if signer == nil {
		return fmt.Errorf("signer is nil")
	}

	if len(marshalledHeaders) != len(headers) {
		return fmt.Errorf("marshalledHeaders length (%d) does not match headers length (%d)", len(marshalledHeaders), len(headers))
	}

	s.logger.Info().Int("count", len(headers)).Msg("submitting headers to DA")

	// Create DA envelopes with parallel signing and caching
	envelopes, err := s.createDAEnvelopes(headers, marshalledHeaders, signer)
	if err != nil {
		return err
	}

	return submitToDA(s, ctx, headers, envelopes,
		func(submitted []*types.SignedHeader, res *datypes.ResultSubmit) {
			for _, header := range submitted {
				cache.SetHeaderDAIncluded(header.Hash().String(), res.Height, header.Height())
			}
			if l := len(submitted); l > 0 {
				lastHeight := submitted[l-1].Height()
				cache.SetLastSubmittedHeaderHeight(ctx, lastHeight)
				// Clear envelope cache for successfully submitted heights
				s.clearEnvelopeCacheUpTo(lastHeight)
			}
		},
		"header",
		s.client.GetHeaderNamespace(),
		[]byte(s.config.DA.SubmitOptions),
		func() uint64 { return cache.NumPendingHeaders() },
	)
}

// createDAEnvelopes creates signed DA envelopes for the given headers.
// It uses caching to avoid re-signing on retries and parallel signing for new envelopes.
func (s *DASubmitter) createDAEnvelopes(headers []*types.SignedHeader, marshalledHeaders [][]byte, signer signer.Signer) ([][]byte, error) {
	envelopes := make([][]byte, len(headers))

	// First pass: check cache for already-signed envelopes
	var needSigning []int // indices that need signing
	for i, header := range headers {
		height := header.Height()
		if cached := s.getCachedEnvelope(height); cached != nil {
			envelopes[i] = cached
		} else {
			needSigning = append(needSigning, i)
		}
	}

	// If all envelopes were cached, we're done
	if len(needSigning) == 0 {
		s.logger.Debug().Int("cached", len(headers)).Msg("all envelopes retrieved from cache")
		return envelopes, nil
	}

	s.logger.Debug().
		Int("cached", len(headers)-len(needSigning)).
		Int("to_sign", len(needSigning)).
		Msg("signing DA envelopes")

	// For small batches, sign sequentially to avoid goroutine overhead
	if len(needSigning) <= 2 || s.signingWorkers <= 1 {
		for _, i := range needSigning {
			envelope, err := s.signAndCacheEnvelope(headers[i], marshalledHeaders[i], signer)
			if err != nil {
				return nil, fmt.Errorf("failed to create envelope for header %d: %w", i, err)
			}
			envelopes[i] = envelope
		}
		return envelopes, nil
	}

	// Parallel signing for larger batches
	return s.signEnvelopesParallel(headers, marshalledHeaders, envelopes, needSigning, signer)
}

// signEnvelopesParallel signs envelopes in parallel using a worker pool.
func (s *DASubmitter) signEnvelopesParallel(
	headers []*types.SignedHeader,
	marshalledHeaders [][]byte,
	envelopes [][]byte,
	needSigning []int,
	signer signer.Signer,
) ([][]byte, error) {
	type signJob struct {
		index int
	}
	type signResult struct {
		index    int
		envelope []byte
		err      error
	}

	jobs := make(chan signJob, len(needSigning))
	results := make(chan signResult, len(needSigning))

	// Start workers
	numWorkers := min(s.signingWorkers, len(needSigning))
	var wg sync.WaitGroup
	for range numWorkers {
		wg.Go(func() {
			for job := range jobs {
				envelope, err := s.signAndCacheEnvelope(headers[job.index], marshalledHeaders[job.index], signer)
				results <- signResult{index: job.index, envelope: envelope, err: err}
			}
		})
	}

	// Send jobs
	for _, i := range needSigning {
		jobs <- signJob{index: i}
	}
	close(jobs)

	// Wait for workers to finish and close results
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var firstErr error
	for result := range results {
		if result.err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to create envelope for header %d: %w", result.index, result.err)
			continue
		}
		if result.err == nil {
			envelopes[result.index] = result.envelope
		}
	}

	if firstErr != nil {
		return nil, firstErr
	}

	return envelopes, nil
}

// signAndCacheEnvelope signs a single header and caches the result.
func (s *DASubmitter) signAndCacheEnvelope(header *types.SignedHeader, marshalledHeader []byte, signer signer.Signer) ([]byte, error) {
	// Sign the pre-marshalled header content
	envelopeSignature, err := signer.Sign(marshalledHeader)
	if err != nil {
		return nil, fmt.Errorf("failed to sign envelope: %w", err)
	}

	// Create the envelope and marshal it
	envelope, err := header.MarshalDAEnvelope(envelopeSignature)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal DA envelope: %w", err)
	}

	// Cache for potential retries
	s.setCachedEnvelope(header.Height(), envelope)

	return envelope, nil
}

// getCachedEnvelope retrieves a cached envelope for the given height.
func (s *DASubmitter) getCachedEnvelope(height uint64) []byte {
	if s.envelopeCache == nil {
		return nil
	}
	s.envelopeCacheMu.RLock()
	defer s.envelopeCacheMu.RUnlock()

	if envelope, ok := s.envelopeCache.Get(height); ok {
		return envelope
	}
	return nil
}

// setCachedEnvelope stores an envelope in the cache.
func (s *DASubmitter) setCachedEnvelope(height uint64, envelope []byte) {
	if s.envelopeCache == nil {
		return
	}
	s.envelopeCacheMu.Lock()
	defer s.envelopeCacheMu.Unlock()

	s.envelopeCache.Add(height, envelope)
}

// clearEnvelopeCacheUpTo removes cached envelopes up to and including the given height.
func (s *DASubmitter) clearEnvelopeCacheUpTo(height uint64) {
	if s.envelopeCache == nil {
		return
	}
	s.envelopeCacheMu.Lock()
	defer s.envelopeCacheMu.Unlock()

	keys := s.envelopeCache.Keys()
	for _, h := range keys {
		if h <= height {
			s.envelopeCache.Remove(h)
		}
	}
}

// SubmitData submits pending data to DA layer
func (s *DASubmitter) SubmitData(ctx context.Context, unsignedDataList []*types.SignedData, marshalledData [][]byte, cache cache.Manager, signer signer.Signer, genesis genesis.Genesis) error {
	if len(unsignedDataList) == 0 {
		return nil
	}

	if len(marshalledData) != len(unsignedDataList) {
		return fmt.Errorf("marshalledData length (%d) does not match unsignedDataList length (%d)", len(marshalledData), len(unsignedDataList))
	}

	// Sign the data (cache returns unsigned SignedData structs)
	signedDataList, signedDataListBz, err := s.signData(unsignedDataList, marshalledData, signer, genesis)
	if err != nil {
		return fmt.Errorf("failed to sign data: %w", err)
	}

	if len(signedDataList) == 0 {
		return nil // No non-empty data to submit
	}

	s.logger.Info().Int("count", len(signedDataList)).Msg("submitting data to DA")

	return submitToDA(s, ctx, signedDataList, signedDataListBz,
		func(submitted []*types.SignedData, res *datypes.ResultSubmit) {
			for _, sd := range submitted {
				cache.SetDataDAIncluded(sd.Data.DACommitment().String(), res.Height, sd.Height())
			}
			if l := len(submitted); l > 0 {
				lastHeight := submitted[l-1].Height()
				cache.SetLastSubmittedDataHeight(ctx, lastHeight)
			}
		},
		"data",
		s.client.GetDataNamespace(),
		[]byte(s.config.DA.SubmitOptions),
		func() uint64 { return cache.NumPendingData() },
	)
}

// signData signs unsigned SignedData structs returned from cache
func (s *DASubmitter) signData(unsignedDataList []*types.SignedData, unsignedDataListBz [][]byte, signer signer.Signer, genesis genesis.Genesis) ([]*types.SignedData, [][]byte, error) {
	if signer == nil {
		return nil, nil, fmt.Errorf("signer is nil")
	}

	pubKey, err := signer.GetPublic()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get public key: %w", err)
	}

	addr, err := signer.GetAddress()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get address: %w", err)
	}

	if len(genesis.ProposerAddress) > 0 && !bytes.Equal(addr, genesis.ProposerAddress) {
		return nil, nil, fmt.Errorf("signer address mismatch with genesis proposer")
	}

	signerInfo := types.Signer{
		PubKey:  pubKey,
		Address: addr,
	}

	signedDataList := make([]*types.SignedData, 0, len(unsignedDataList))
	signedDataListBz := make([][]byte, 0, len(unsignedDataListBz))

	for i, unsignedData := range unsignedDataList {
		// Skip empty data
		if len(unsignedData.Txs) == 0 {
			continue
		}

		signature, err := signer.Sign(unsignedDataListBz[i])
		if err != nil {
			return nil, nil, fmt.Errorf("failed to sign data: %w", err)
		}

		signedData := &types.SignedData{
			Data:      unsignedData.Data,
			Signer:    signerInfo,
			Signature: signature,
		}

		signedDataList = append(signedDataList, signedData)

		signedDataBz, err := signedData.MarshalBinary()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal signed data: %w", err)
		}

		signedDataListBz = append(signedDataListBz, signedDataBz)
	}

	return signedDataList, signedDataListBz, nil
}

// mergeSubmitOptions merges the base submit options with a signing address.
// If the base options are valid JSON, the signing address is added to the JSON object.
// Otherwise, a new JSON object is created with just the signing address.
// Returns the base options unchanged if no signing address is provided.
func mergeSubmitOptions(baseOptions []byte, signingAddress string) ([]byte, error) {
	if signingAddress == "" {
		return baseOptions, nil
	}

	var optionsMap map[string]any

	// If base options are provided, try to parse them as JSON
	if len(baseOptions) > 0 {
		// Try to unmarshal existing options, ignoring errors for non-JSON input
		if err := json.Unmarshal(baseOptions, &optionsMap); err != nil {
			// Not valid JSON - start with empty map
			optionsMap = make(map[string]any)
		}
	}

	// Ensure map is initialized even if unmarshal returned nil
	if optionsMap == nil {
		optionsMap = make(map[string]any)
	}

	// Add or override the signing address
	// Note: Uses "signer_address" to match Celestia's TxConfig JSON schema
	optionsMap["signer_address"] = signingAddress

	// Marshal back to JSON
	mergedOptions, err := json.Marshal(optionsMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal submit options: %w", err)
	}

	return mergedOptions, nil
}

// submitToDA is a generic helper for submitting items to the DA layer with retry, backoff, and gas price logic.
func submitToDA[T any](
	s *DASubmitter,
	ctx context.Context,
	items []T,
	marshaled [][]byte,
	postSubmit func([]T, *datypes.ResultSubmit),
	itemType string,
	namespace []byte,
	options []byte,
	getTotalPendingFn func() uint64,
) error {
	if len(items) != len(marshaled) {
		return fmt.Errorf("items length (%d) does not match marshaled length (%d)", len(items), len(marshaled))
	}

	pol := defaultRetryPolicy(s.config.DA.MaxSubmitAttempts, s.config.DA.BlockTime.Duration)

	rs := retryState{Attempt: 0, Backoff: 0}

	// Limit this submission to a single size-capped batch
	if len(marshaled) > 0 {
		batchItems, batchMarshaled, err := limitBatchBySize(items, marshaled, pol.MaxBlobBytes)
		if err != nil {
			s.logger.Error().
				Str("itemType", itemType).
				Int("maxBlobBytes", pol.MaxBlobBytes).
				Err(err).
				Msg("CRITICAL: Unrecoverable error - item exceeds maximum blob size")
			return fmt.Errorf("unrecoverable error: no %s items fit within max blob size: %w", itemType, err)
		}
		items = batchItems
		marshaled = batchMarshaled
	}

	// Update pending blobs metric to track total backlog
	if getTotalPendingFn != nil {
		s.metrics.DASubmitterPendingBlobs.Set(float64(getTotalPendingFn()))
	}

	// Start the retry loop
	for rs.Attempt < pol.MaxAttempts {
		// Record resend metric for retry attempts (not the first attempt)
		if rs.Attempt > 0 {
			s.metrics.DASubmitterResends.Add(1)
		}

		if err := waitForBackoffOrContext(ctx, rs.Backoff); err != nil {
			return err
		}

		// Select signing address and merge with options
		signingAddress := s.addressSelector.Next()
		mergedOptions, err := mergeSubmitOptions(options, signingAddress)
		if err != nil {
			s.logger.Error().Err(err).Msg("failed to merge submit options with signing address")
			return fmt.Errorf("failed to merge submit options: %w", err)
		}

		if signingAddress != "" {
			s.logger.Debug().Str("signingAddress", signingAddress).Msg("using signing address for DA submission")
		}

		// Perform submission
		start := time.Now()
		res := s.client.Submit(ctx, marshaled, -1, namespace, mergedOptions)
		s.logger.Debug().Int("attempts", rs.Attempt).Dur("elapsed", time.Since(start)).Uint64("code", uint64(res.Code)).Msg("got SubmitWithHelpers response from celestia")

		// Record submission result for observability
		if daVisualizationServer := server.GetDAVisualizationServer(); daVisualizationServer != nil {
			daVisualizationServer.RecordSubmission(&res, 0, uint64(len(items)), namespace)
		}

		switch res.Code {
		case datypes.StatusSuccess:
			submitted := items[:res.SubmittedCount]
			postSubmit(submitted, &res)
			s.logger.Info().Str("itemType", itemType).Uint64("count", res.SubmittedCount).Msg("successfully submitted items to DA layer")
			if int(res.SubmittedCount) == len(items) {
				rs.Next(reasonSuccess, pol)
				// Update pending blobs metric to reflect total backlog
				if getTotalPendingFn != nil {
					s.metrics.DASubmitterPendingBlobs.Set(float64(getTotalPendingFn()))
				}
				return nil
			}
			// partial success: advance window
			items = items[res.SubmittedCount:]
			marshaled = marshaled[res.SubmittedCount:]
			rs.Next(reasonSuccess, pol)
			// Update pending blobs count to reflect total backlog
			if getTotalPendingFn != nil {
				s.metrics.DASubmitterPendingBlobs.Set(float64(getTotalPendingFn()))
			}

		case datypes.StatusTooBig:
			// Record failure metric
			s.recordFailure(common.DASubmitterFailureReasonTooBig)
			// Iteratively halve until it fits or single-item too big
			if len(items) == 1 {
				s.logger.Error().
					Str("itemType", itemType).
					Int("maxBlobBytes", pol.MaxBlobBytes).
					Msg("CRITICAL: Unrecoverable error - single item exceeds DA blob size limit")
				return fmt.Errorf("unrecoverable error: %w: single %s item exceeds DA blob size limit", common.ErrOversizedItem, itemType)
			}
			half := len(items) / 2
			if half == 0 {
				half = 1
			}
			items = items[:half]
			marshaled = marshaled[:half]
			s.logger.Debug().Int("newBatchSize", half).Msg("batch too big; halving and retrying")
			rs.Next(reasonTooBig, pol)
			// Update pending blobs count to reflect total backlog
			if getTotalPendingFn != nil {
				s.metrics.DASubmitterPendingBlobs.Set(float64(getTotalPendingFn()))
			}

		case datypes.StatusNotIncludedInBlock:
			// Record failure metric
			s.recordFailure(common.DASubmitterFailureReasonNotIncludedInBlock)
			s.logger.Info().Dur("backoff", pol.MaxBackoff).Msg("retrying due to mempool state")
			rs.Next(reasonMempool, pol)

		case datypes.StatusAlreadyInMempool:
			// Record failure metric
			s.recordFailure(common.DASubmitterFailureReasonAlreadyInMempool)
			s.logger.Info().Dur("backoff", pol.MaxBackoff).Msg("retrying due to mempool state")
			rs.Next(reasonMempool, pol)

		case datypes.StatusContextCanceled:
			// Record failure metric
			s.recordFailure(common.DASubmitterFailureReasonContextCanceled)
			s.logger.Info().Msg("DA layer submission canceled due to context cancellation")
			return context.Canceled

		default:
			// Record failure metric
			s.recordFailure(common.DASubmitterFailureReasonUnknown)
			s.logger.Error().Str("error", res.Message).Int("attempt", rs.Attempt+1).Msg("DA layer submission failed")
			rs.Next(reasonFailure, pol)
		}
	}

	// Final failure after max attempts
	s.recordFailure(common.DASubmitterFailureReasonTimeout)
	return fmt.Errorf("failed to submit all %s(s) to DA layer after %d attempts", itemType, rs.Attempt)
}

// limitBatchBySize returns a prefix of items whose total marshaled size does not exceed maxBytes.
// If the first item exceeds maxBytes, it returns ErrOversizedItem which is unrecoverable.
func limitBatchBySize[T any](items []T, marshaled [][]byte, maxBytes int) ([]T, [][]byte, error) {
	total := 0
	count := 0
	for i := 0; i < len(items); i++ {
		sz := len(marshaled[i])
		if sz > maxBytes {
			if i == 0 {
				return nil, nil, fmt.Errorf("%w: item size %d exceeds max %d", common.ErrOversizedItem, sz, maxBytes)
			}
			break
		}
		if total+sz > maxBytes {
			break
		}
		total += sz
		count++
	}
	if count == 0 {
		return nil, nil, fmt.Errorf("no items fit within %d bytes", maxBytes)
	}
	return items[:count], marshaled[:count], nil
}

func waitForBackoffOrContext(ctx context.Context, backoff time.Duration) error {
	if backoff <= 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	}
	timer := time.NewTimer(backoff)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
