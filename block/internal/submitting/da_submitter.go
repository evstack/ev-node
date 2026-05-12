package submitting

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

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
	DefaultEnvelopeCacheSize = 10_000

	// signingWorkerPoolSize determines how many parallel signing goroutines to use.
	// 0 means use runtime.GOMAXPROCS(0)
	signingWorkerPoolSize = 0
)

const initialBackoff = 100 * time.Millisecond

// retryPolicy defines clamped bounds for retries and backoff.
type retryPolicy struct {
	MaxAttempts  int
	MinBackoff   time.Duration
	MaxBackoff   time.Duration
	MaxBlobBytes uint64
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
func clamp(v, minTime, maxTime time.Duration) time.Duration {
	if minTime > maxTime {
		minTime, maxTime = maxTime, minTime
	}
	if v < minTime {
		return minTime
	}
	if v > maxTime {
		return maxTime
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

	// lastSubmittedHeight tracks the last successfully submitted height for lazy cache invalidation
	lastSubmittedHeight atomic.Uint64

	// signingWorkers is the number of parallel workers for signing
	signingWorkers int

	wg sync.WaitGroup
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
		signingWorkers:  workers,
	}
}

func (s *DASubmitter) Close() {
	s.wg.Wait()
}

func (s *DASubmitter) SubmitBlocks(ctx context.Context, headers []*types.SignedHeader, dataList []*types.Data, cacheMgr cache.Manager, signer signer.Signer, onSubmitError func(error)) error {
	if len(headers) == 0 {
		return nil
	}

	if signer == nil {
		return fmt.Errorf("signer is nil")
	}

	if len(dataList) != len(headers) {
		return fmt.Errorf("dataList length (%d) does not match headers length (%d)", len(dataList), len(headers))
	}

	s.logger.Info().Int("count", len(headers)).Msg("submitting combined blocks to DA")

	blobs := make([][]byte, 0, len(headers))
	submittedHeaders := make([]*types.SignedHeader, 0, len(headers))
	submittedData := make([]*types.Data, 0, len(headers))

	for i, header := range headers {
		data := dataList[i]

		headerBz, err := header.MarshalBinary()
		if err != nil {
			return fmt.Errorf("failed to marshal header at height %d: %w", header.Height(), err)
		}

		envelopeSig, err := signer.Sign(ctx, headerBz)
		if err != nil {
			return fmt.Errorf("failed to sign envelope for header %d: %w", header.Height(), err)
		}

		blob, err := common.MarshalBlockBlob(header, data, envelopeSig)
		if err != nil {
			return fmt.Errorf("failed to create combined blob for height %d: %w", header.Height(), err)
		}

		blobs = append(blobs, blob)
		submittedHeaders = append(submittedHeaders, header)
		submittedData = append(submittedData, data)
	}

	namespace := s.client.GetHeaderNamespace()
	submittedOffset := 0

	s.wg.Go(func() {
		s.submitWithRetry(ctx, blobs, namespace, func(submittedCount int, daHeight uint64) {
			if submittedCount > 0 {
				end := submittedOffset + submittedCount
				for _, hdr := range submittedHeaders[submittedOffset:end] {
					cacheMgr.SetHeaderDAIncluded(hdr.Hash().String(), daHeight, hdr.Height())
					cacheMgr.SetLastSubmittedHeaderHeight(ctx, hdr.Height())
				}
				for _, d := range submittedData[submittedOffset:end] {
					if len(d.Txs) > 0 {
						cacheMgr.SetDataDAIncluded(d.DACommitment().String(), daHeight, d.Height())
					}
				}
				cacheMgr.SetLastSubmittedDataHeight(ctx, submittedHeaders[end-1].Height())
				submittedOffset = end
			}
		}, onSubmitError, "block")
	})

	return nil
}

func (s *DASubmitter) submitWithRetry(
	ctx context.Context,
	marshaled [][]byte,
	namespace []byte,
	onSuccess func(submittedCount int, daHeight uint64),
	onError func(error),
	itemType string,
) {
	pol := defaultRetryPolicy(s.config.DA.MaxSubmitAttempts, s.config.DA.BlockTime.Duration)
	options := []byte(s.config.DA.SubmitOptions)

	if len(marshaled) == 0 {
		if onError != nil {
			onError(nil)
		}
		return
	}

	limitedMarshaled, oversized := limitBatchBySizeBytes(marshaled, pol.MaxBlobBytes)
	if oversized {
		s.logger.Error().
			Str("itemType", itemType).
			Uint64("maxBlobBytes", pol.MaxBlobBytes).
			Msg("CRITICAL: item exceeds maximum blob size")
		if onError != nil {
			onError(common.ErrOversizedItem)
		}
		return
	}
	marshaled = limitedMarshaled

	rs := retryState{}

	// Start the retry loop
	for rs.Attempt < pol.MaxAttempts {
		// Record resend metric for retry attempts (not the first attempt)
		if rs.Attempt > 0 {
			s.metrics.DASubmitterResends.Add(1)
		}

		if err := waitForBackoffOrContext(ctx, rs.Backoff); err != nil {
			if onError != nil {
				onError(nil)
			}
			return
		}

		// Select signing address and merge with options
		signingAddress := s.addressSelector.Next()
		mergedOptions, err := mergeSubmitOptions(options, signingAddress)
		if err != nil {
			s.logger.Error().Err(err).Msg("failed to merge submit options with signing address")
			if onError != nil {
				onError(err)
			}
			return
		}

		// Perform submission
		start := time.Now()
		res := s.client.Submit(ctx, marshaled, -1, namespace, mergedOptions)
		s.logger.Debug().Int("attempts", rs.Attempt).Dur("elapsed", time.Since(start)).Uint64("code", uint64(res.Code)).Msg("got Submit response")

		// Record submission result for observability
		if vis := server.GetDAVisualizationServer(); vis != nil {
			vis.RecordSubmission(&res, 0, uint64(len(marshaled)), namespace)
		}

		switch res.Code {
		case datypes.StatusSuccess:
			submitted := int(res.SubmittedCount)
			if submitted <= 0 || submitted > len(marshaled) {
				err := fmt.Errorf("invalid submitted count %d for batch size %d", submitted, len(marshaled))
				s.recordFailure(common.DASubmitterFailureReasonUnknown)
				s.logger.Error().Err(err).Str("itemType", itemType).Msg("DA layer returned invalid submitted count")
				if onError != nil {
					onError(err)
				}
				return
			}
			if onSuccess != nil {
				onSuccess(submitted, res.Height)
			}
			s.logger.Info().Str("itemType", itemType).Int("count", submitted).Msg("successfully submitted items to DA layer")
			if submitted == len(marshaled) {
				return
			}
			// partial success: advance window
			marshaled = marshaled[submitted:]
			rs.Next(reasonSuccess, pol)

		case datypes.StatusTooBig:
			// Record failure metric
			s.recordFailure(common.DASubmitterFailureReasonTooBig)
			// Iteratively halve until it fits or single-item too big
			if len(marshaled) == 1 {
				s.logger.Error().
					Str("itemType", itemType).
					Msg("CRITICAL: single item exceeds DA blob size limit")
				if onError != nil {
					onError(common.ErrOversizedItem)
				}
				return
			}
			half := len(marshaled) / 2
			if half == 0 {
				half = 1
			}
			marshaled = marshaled[:half]
			s.logger.Debug().Int("newBatchSize", half).Msg("batch too big; halving and retrying")
			rs.Next(reasonTooBig, pol)

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
			if onError != nil {
				onError(nil)
			}
			return

		default:
			// Record failure metric
			s.recordFailure(common.DASubmitterFailureReasonUnknown)
			s.logger.Error().Str("error", res.Message).Int("attempt", rs.Attempt+1).Msg("DA layer submission failed")
			rs.Next(reasonFailure, pol)
		}
	}

	// Final failure after max attempts
	s.recordFailure(common.DASubmitterFailureReasonTimeout)
	s.logger.Error().Str("itemType", itemType).Int("attempts", rs.Attempt).Msg("failed to submit all items to DA layer after max attempts")
	if onError != nil {
		onError(fmt.Errorf("failed to submit after %d attempts", rs.Attempt))
	}
}

// limitBatchBySizeBytes returns a prefix of marshaled blobs whose total size does not exceed maxBytes.
// If the first blob exceeds maxBytes, it returns (nil, true) to indicate an unrecoverable oversized item.
func limitBatchBySizeBytes(marshaled [][]byte, maxBytes uint64) ([][]byte, bool) {
	total := uint64(0)
	count := 0
	for i, b := range marshaled {
		sz := uint64(len(b))
		if sz > maxBytes {
			if i == 0 {
				return nil, true
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
		return nil, true
	}
	return marshaled[:count], false
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
	optionsMap["signer_address"] = signingAddress

	// Marshal back to JSON
	mergedOptions, err := json.Marshal(optionsMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal submit options: %w", err)
	}

	return mergedOptions, nil
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
