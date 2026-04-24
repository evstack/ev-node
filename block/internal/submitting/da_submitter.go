package submitting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
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

	submitQueueSize = 256
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

type DAHintAppender interface {
	AppendDAHint(ctx context.Context, daHeight uint64, heights ...uint64) error
}

type batchGroup struct {
	marshaled     [][]byte
	onSuccess     func(count int, daHeight uint64)
	makeRemaining func(submittedCount int, remainingMarshaled [][]byte) *batchGroup
}

// DASubmitter handles DA submission operations
type DASubmitter struct {
	client               da.Client
	config               config.Config
	genesis              genesis.Genesis
	options              common.BlockOptions
	logger               zerolog.Logger
	metrics              *common.Metrics
	headerDAHintAppender DAHintAppender
	dataDAHintAppender   DAHintAppender

	// address selector for multi-account support
	addressSelector pkgda.AddressSelector

	// envelopeCache caches fully signed DA envelopes by height to avoid re-signing on retries
	envelopeCache   *lru.Cache[uint64, []byte]
	envelopeCacheMu sync.RWMutex

	// lastSubmittedHeight tracks the last successfully submitted height for lazy cache invalidation.
	// This avoids O(N) iteration over the cache on every submission.
	lastSubmittedHeight atomic.Uint64

	// signingWorkers is the number of parallel workers for signing
	signingWorkers int

	headerSubmitCh chan *batchGroup
	dataSubmitCh   chan *batchGroup
	headerFlushCh  chan chan struct{}
	dataFlushCh    chan chan struct{}
	workerCancel   context.CancelFunc
	workerWg       sync.WaitGroup
}

// NewDASubmitter creates a new DA submitter
func NewDASubmitter(
	client da.Client,
	config config.Config,
	genesis genesis.Genesis,
	options common.BlockOptions,
	metrics *common.Metrics,
	logger zerolog.Logger,
	headerDAHintAppender DAHintAppender,
	dataDAHintAppender DAHintAppender,
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

	s := &DASubmitter{
		client:               client,
		config:               config,
		genesis:              genesis,
		options:              options,
		metrics:              metrics,
		logger:               daSubmitterLogger,
		addressSelector:      addressSelector,
		envelopeCache:        envelopeCache,
		signingWorkers:       workers,
		headerDAHintAppender: headerDAHintAppender,
		dataDAHintAppender:   dataDAHintAppender,
		headerSubmitCh:       make(chan *batchGroup, submitQueueSize),
		dataSubmitCh:         make(chan *batchGroup, submitQueueSize),
		headerFlushCh:        make(chan chan struct{}),
		dataFlushCh:          make(chan chan struct{}),
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.workerCancel = cancel
	s.workerWg.Add(2)
	go s.runNamespaceWorker(ctx, s.headerSubmitCh, s.headerFlushCh, "header", s.client.GetHeaderNamespace())
	go s.runNamespaceWorker(ctx, s.dataSubmitCh, s.dataFlushCh, "data", s.client.GetDataNamespace())

	return s
}

func (s *DASubmitter) Close() {
	if s.workerCancel != nil {
		s.Flush()
		s.workerCancel()
		s.workerWg.Wait()
	}
}

func (s *DASubmitter) Flush() {
	headerDone := make(chan struct{})
	s.headerFlushCh <- headerDone
	dataDone := make(chan struct{})
	s.dataFlushCh <- dataDone
	<-headerDone
	<-dataDone
}

func (s *DASubmitter) runNamespaceWorker(ctx context.Context, ch chan *batchGroup, flushCh chan chan struct{}, itemType string, namespace []byte) {
	defer s.workerWg.Done()
	for {
		select {
		case <-ctx.Done():
			drainChannel(ch, func(g *batchGroup) {
				s.processBatch(ctx, []*batchGroup{g}, itemType, namespace)
			})
			return
		case done := <-flushCh:
			drainChannel(ch, func(g *batchGroup) {
				s.processBatch(ctx, []*batchGroup{g}, itemType, namespace)
			})
			close(done)
		case first := <-ch:
			groups := []*batchGroup{first}
		drainLoop:
			for {
				select {
				case g := <-ch:
					if g != nil {
						groups = append(groups, g)
					}
				default:
					break drainLoop
				}
			}
			s.logger.Debug().Str("itemType", itemType).Int("groups", len(groups)).Msg("worker processing batch")
			s.processBatchSafe(ctx, groups, itemType, namespace)
			s.logger.Debug().Str("itemType", itemType).Msg("worker done processing batch")
		}
	}
}

func drainChannel[T any](ch chan T, fn func(T)) {
	for {
		select {
		case item := <-ch:
			fn(item)
		default:
			return
		}
	}
}

func (s *DASubmitter) processBatchSafe(ctx context.Context, groups []*batchGroup, itemType string, namespace []byte) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error().Any("panic", r).Str("itemType", itemType).Msg("worker panic in processBatch")
			for _, g := range groups {
				if g.onSuccess != nil {
					g.onSuccess(0, 0)
				}
			}
		}
	}()
	s.processBatch(ctx, groups, itemType, namespace)
}

func (s *DASubmitter) processBatch(ctx context.Context, groups []*batchGroup, itemType string, namespace []byte) {
	pol := defaultRetryPolicy(s.config.DA.MaxSubmitAttempts, s.config.DA.BlockTime.Duration)
	options := []byte(s.config.DA.SubmitOptions)

	var allMarshaled [][]byte
	for _, g := range groups {
		allMarshaled = append(allMarshaled, g.marshaled...)
	}

	if len(allMarshaled) == 0 {
		for _, g := range groups {
			if g.onSuccess != nil {
				g.onSuccess(0, 0)
			}
		}
		return
	}

	limitedMarshaled, oversized := limitBatchBySizeBytes(allMarshaled, pol.MaxBlobBytes)
	if oversized {
		s.logger.Error().
			Str("itemType", itemType).
			Uint64("maxBlobBytes", pol.MaxBlobBytes).
			Msg("CRITICAL: item exceeds maximum blob size")
		for _, g := range groups {
			if g.onSuccess != nil {
				g.onSuccess(0, 0)
			}
		}
		return
	}
	allMarshaled = limitedMarshaled

	rs := retryState{}

	for rs.Attempt < pol.MaxAttempts {
		if rs.Attempt > 0 {
			s.metrics.DASubmitterResends.Add(1)
		}

		if err := waitForBackoffOrContext(ctx, rs.Backoff); err != nil {
			for _, g := range groups {
				if g.onSuccess != nil {
					g.onSuccess(0, 0)
				}
			}
			return
		}

		signingAddress := s.addressSelector.Next()
		mergedOptions, err := mergeSubmitOptions(options, signingAddress)
		if err != nil {
			s.logger.Error().Err(err).Msg("failed to merge submit options with signing address")
			for _, g := range groups {
				if g.onSuccess != nil {
					g.onSuccess(0, 0)
				}
			}
			return
		}

		start := time.Now()
		res := s.client.Submit(ctx, allMarshaled, -1, namespace, mergedOptions)
		s.logger.Debug().Int("attempts", rs.Attempt).Dur("elapsed", time.Since(start)).Uint64("code", uint64(res.Code)).Msg("got Submit response")

		if vis := server.GetDAVisualizationServer(); vis != nil {
			vis.RecordSubmission(&res, 0, uint64(len(allMarshaled)), namespace)
		}

		switch res.Code {
		case datypes.StatusSuccess:
			submitted := int(res.SubmittedCount)
			s.distributeSuccess(groups, submitted, res.Height)
			s.logger.Info().Str("itemType", itemType).Int("count", submitted).Msg("successfully submitted items to DA layer")
			if submitted == len(allMarshaled) {
				rs.Next(reasonSuccess, pol)
				return
			}
			allMarshaled = allMarshaled[submitted:]
			groups = s.advanceGroups(groups, submitted)
			rs.Next(reasonSuccess, pol)

		case datypes.StatusTooBig:
			s.recordFailure(common.DASubmitterFailureReasonTooBig)
			if len(allMarshaled) == 1 {
				s.logger.Error().
					Str("itemType", itemType).
					Msg("CRITICAL: single item exceeds DA blob size limit")
				for _, g := range groups {
					if g.onSuccess != nil {
						g.onSuccess(0, 0)
					}
				}
				return
			}
			half := len(allMarshaled) / 2
			if half == 0 {
				half = 1
			}
			allMarshaled = allMarshaled[:half]
			groups = s.truncateGroups(groups, half)
			s.logger.Debug().Int("newBatchSize", half).Msg("batch too big; halving and retrying")
			rs.Next(reasonTooBig, pol)

		case datypes.StatusNotIncludedInBlock:
			s.recordFailure(common.DASubmitterFailureReasonNotIncludedInBlock)
			rs.Next(reasonMempool, pol)

		case datypes.StatusAlreadyInMempool:
			s.recordFailure(common.DASubmitterFailureReasonAlreadyInMempool)
			rs.Next(reasonMempool, pol)

		case datypes.StatusContextCanceled:
			s.recordFailure(common.DASubmitterFailureReasonContextCanceled)
			for _, g := range groups {
				if g.onSuccess != nil {
					g.onSuccess(0, 0)
				}
			}
			return

		default:
			s.recordFailure(common.DASubmitterFailureReasonUnknown)
			s.logger.Error().Str("error", res.Message).Int("attempt", rs.Attempt+1).Msg("DA layer submission failed")
			rs.Next(reasonFailure, pol)
		}
	}

	s.recordFailure(common.DASubmitterFailureReasonTimeout)
	s.logger.Error().Str("itemType", itemType).Int("attempts", rs.Attempt).Msg("failed to submit all items to DA layer after max attempts")
	for _, g := range groups {
		if g.onSuccess != nil {
			g.onSuccess(0, 0)
		}
	}
}

func (s *DASubmitter) distributeSuccess(groups []*batchGroup, submittedCount int, daHeight uint64) {
	remaining := submittedCount
	for _, g := range groups {
		if remaining <= 0 {
			break
		}
		n := min(remaining, len(g.marshaled))
		if g.onSuccess != nil {
			g.onSuccess(n, daHeight)
		}
		remaining -= n
	}
}

func (s *DASubmitter) advanceGroups(groups []*batchGroup, submittedCount int) []*batchGroup {
	remaining := submittedCount
	for i, g := range groups {
		if remaining <= 0 {
			return groups[i:]
		}
		if remaining >= len(g.marshaled) {
			remaining -= len(g.marshaled)
		} else {
			if g.makeRemaining != nil {
				if next := g.makeRemaining(remaining, g.marshaled[remaining:]); next != nil {
					groups[i] = next
					return groups[i:]
				}
			}
			return groups[i+1:]
		}
	}
	return nil
}

func (s *DASubmitter) truncateGroups(groups []*batchGroup, maxItems int) []*batchGroup {
	count := 0
	for i, g := range groups {
		if count+len(g.marshaled) > maxItems {
			trim := maxItems - count
			if g.makeRemaining != nil {
				if next := g.makeRemaining(trim, g.marshaled[trim:]); next != nil {
					groups[i] = next
					return groups[:i+1]
				}
			}
			return groups[:i]
		}
		count += len(g.marshaled)
		if count == maxItems {
			return groups[:i+1]
		}
	}
	return groups
}

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
	envelopes, err := s.createDAEnvelopes(ctx, headers, marshalledHeaders, signer)
	if err != nil {
		return err
	}

	postSubmit := s.makeHeaderPostSubmit(ctx, cache)

	s.headerSubmitCh <- &batchGroup{
		marshaled: envelopes,
		onSuccess: func(count int, daHeight uint64) {
			if count > 0 {
				postSubmit(headers[:count], &datypes.ResultSubmit{BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, SubmittedCount: uint64(count), Height: daHeight}})
			}
		},
		makeRemaining: func(submittedCount int, remainingMarshaled [][]byte) *batchGroup {
			remainingHeaders := headers[submittedCount:]
			return &batchGroup{
				marshaled: remainingMarshaled,
				onSuccess: func(count int, daHeight uint64) {
					if count > 0 {
						postSubmit(remainingHeaders[:count], &datypes.ResultSubmit{BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, SubmittedCount: uint64(count), Height: daHeight}})
					}
				},
				makeRemaining: nil,
			}
		},
	}
	return nil
}

func (s *DASubmitter) makeHeaderPostSubmit(ctx context.Context, cache cache.Manager) func([]*types.SignedHeader, *datypes.ResultSubmit) {
	return func(submitted []*types.SignedHeader, res *datypes.ResultSubmit) {
		heights := make([]uint64, len(submitted))
		for i, header := range submitted {
			cache.SetHeaderDAIncluded(header.Hash().String(), res.Height, header.Height())
			heights[i] = header.Height()
		}
		if err := s.headerDAHintAppender.AppendDAHint(ctx, res.Height, heights...); err != nil {
			s.logger.Error().Err(err).Msg("failed to append da height hint in header p2p store")
		}
		if l := len(submitted); l > 0 {
			lastHeight := submitted[l-1].Height()
			cache.SetLastSubmittedHeaderHeight(ctx, lastHeight)
			s.lastSubmittedHeight.Store(lastHeight)
		}
	}
}

func (s *DASubmitter) createDAEnvelopes(ctx context.Context, headers []*types.SignedHeader, marshalledHeaders [][]byte, signer signer.Signer) ([][]byte, error) {
	envelopes := make([][]byte, len(headers))

	var needSigning []int
	for i, header := range headers {
		height := header.Height()
		if cached := s.getCachedEnvelope(height); cached != nil {
			envelopes[i] = cached
		} else {
			needSigning = append(needSigning, i)
		}
	}

	if len(needSigning) == 0 {
		s.logger.Debug().Int("cached", len(headers)).Msg("all envelopes retrieved from cache")
		return envelopes, nil
	}

	s.logger.Debug().
		Int("cached", len(headers)-len(needSigning)).
		Int("to_sign", len(needSigning)).
		Msg("signing DA envelopes")

	if len(needSigning) <= 2 || s.signingWorkers <= 1 {
		for _, i := range needSigning {
			envelope, err := s.signAndCacheEnvelope(ctx, headers[i], marshalledHeaders[i], signer)
			if err != nil {
				return nil, fmt.Errorf("failed to create envelope for header %d: %w", i, err)
			}
			envelopes[i] = envelope
		}
		return envelopes, nil
	}

	return s.signEnvelopesParallel(ctx, headers, marshalledHeaders, envelopes, needSigning, signer)
}

func (s *DASubmitter) signEnvelopesParallel(
	ctx context.Context,
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

	numWorkers := min(s.signingWorkers, len(needSigning))
	var wg sync.WaitGroup
	for range numWorkers {
		wg.Go(func() {
			for job := range jobs {
				envelope, err := s.signAndCacheEnvelope(ctx, headers[job.index], marshalledHeaders[job.index], signer)
				results <- signResult{index: job.index, envelope: envelope, err: err}
			}
		})
	}

	for _, i := range needSigning {
		jobs <- signJob{index: i}
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

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

func (s *DASubmitter) signAndCacheEnvelope(ctx context.Context, header *types.SignedHeader, marshalledHeader []byte, signer signer.Signer) ([]byte, error) {
	envelopeSignature, err := signer.Sign(ctx, marshalledHeader)
	if err != nil {
		return nil, fmt.Errorf("failed to sign envelope: %w", err)
	}

	envelope, err := header.MarshalDAEnvelope(envelopeSignature)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal DA envelope: %w", err)
	}

	s.setCachedEnvelope(header.Height(), envelope)

	return envelope, nil
}

func (s *DASubmitter) getCachedEnvelope(height uint64) []byte {
	if s.envelopeCache == nil {
		return nil
	}
	if height <= s.lastSubmittedHeight.Load() {
		return nil
	}
	s.envelopeCacheMu.RLock()
	defer s.envelopeCacheMu.RUnlock()

	if envelope, ok := s.envelopeCache.Get(height); ok {
		return envelope
	}
	return nil
}

func (s *DASubmitter) setCachedEnvelope(height uint64, envelope []byte) {
	if s.envelopeCache == nil {
		return
	}
	if height <= s.lastSubmittedHeight.Load() {
		return
	}
	s.envelopeCacheMu.Lock()
	defer s.envelopeCacheMu.Unlock()

	s.envelopeCache.Add(height, envelope)
}

func (s *DASubmitter) SubmitData(ctx context.Context, unsignedDataList []*types.SignedData, marshalledData [][]byte, cache cache.Manager, signer signer.Signer, genesis genesis.Genesis) error {
	if len(unsignedDataList) == 0 {
		return nil
	}

	if len(marshalledData) != len(unsignedDataList) {
		return fmt.Errorf("marshalledData length (%d) does not match unsignedDataList length (%d)", len(marshalledData), len(unsignedDataList))
	}

	signedDataList, signedDataListBz, err := s.signData(ctx, unsignedDataList, marshalledData, signer, genesis)
	if err != nil {
		return fmt.Errorf("failed to sign data: %w", err)
	}

	if len(signedDataList) == 0 {
		return nil
	}

	s.logger.Info().Int("count", len(signedDataList)).Msg("submitting data to DA")

	postSubmit := s.makeDataPostSubmit(ctx, cache)

	s.dataSubmitCh <- &batchGroup{
		marshaled: signedDataListBz,
		onSuccess: func(count int, daHeight uint64) {
			if count > 0 {
				postSubmit(signedDataList[:count], &datypes.ResultSubmit{BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, SubmittedCount: uint64(count), Height: daHeight}})
			}
		},
		makeRemaining: func(submittedCount int, remainingMarshaled [][]byte) *batchGroup {
			remainingData := signedDataList[submittedCount:]
			return &batchGroup{
				marshaled: remainingMarshaled,
				onSuccess: func(count int, daHeight uint64) {
					if count > 0 {
						postSubmit(remainingData[:count], &datypes.ResultSubmit{BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, SubmittedCount: uint64(count), Height: daHeight}})
					}
				},
				makeRemaining: nil,
			}
		},
	}
	return nil
}

func (s *DASubmitter) makeDataPostSubmit(ctx context.Context, cache cache.Manager) func([]*types.SignedData, *datypes.ResultSubmit) {
	return func(submitted []*types.SignedData, res *datypes.ResultSubmit) {
		heights := make([]uint64, len(submitted))
		for i, sd := range submitted {
			cache.SetDataDAIncluded(sd.Data.DACommitment().String(), res.Height, sd.Height())
			heights[i] = sd.Height()
		}
		if err := s.dataDAHintAppender.AppendDAHint(ctx, res.Height, heights...); err != nil {
			s.logger.Error().Err(err).Msg("failed to append da height hint in data p2p store")
		}
		if l := len(submitted); l > 0 {
			lastHeight := submitted[l-1].Height()
			cache.SetLastSubmittedDataHeight(ctx, lastHeight)
		}
	}
}

func (s *DASubmitter) signData(ctx context.Context, unsignedDataList []*types.SignedData, unsignedDataListBz [][]byte, signer signer.Signer, genesis genesis.Genesis) ([]*types.SignedData, [][]byte, error) {
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
		if len(unsignedData.Txs) == 0 {
			continue
		}

		signature, err := signer.Sign(ctx, unsignedDataListBz[i])
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

func mergeSubmitOptions(baseOptions []byte, signingAddress string) ([]byte, error) {
	if signingAddress == "" {
		return baseOptions, nil
	}

	var optionsMap map[string]any

	if len(baseOptions) > 0 {
		if err := json.Unmarshal(baseOptions, &optionsMap); err != nil {
			optionsMap = make(map[string]any)
		}
	}

	if optionsMap == nil {
		optionsMap = make(map[string]any)
	}

	optionsMap["signer_address"] = signingAddress

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
