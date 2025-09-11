package submitting

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	coreda "github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/rpc/server"
	"github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/types"
)

const (
	submissionTimeout    = 60 * time.Second
	noGasPrice           = -1
	initialBackoff       = 100 * time.Millisecond
	defaultGasPrice      = 0.0
	defaultGasMultiplier = 1.0
	maxBlobSize          = 2 * 1024 * 1024 // 2MB blob size limit
)

// getGasMultiplier fetches the gas multiplier from DA layer with fallback to configured/default value
func (s *DASubmitter) getGasMultiplier(ctx context.Context) float64 {
	gasMultiplier, err := s.da.GasMultiplier(ctx)
	if err != nil || gasMultiplier <= 0 {
		// fall back to configured multiplier if valid
		if s.config.DA.GasMultiplier > 0 {
			return s.config.DA.GasMultiplier
		}
		s.logger.Warn().Err(err).Msg("failed to get gas multiplier from DA layer, using default")
		return defaultGasMultiplier
	}
	return gasMultiplier
}

// retryStrategy manages retry logic with backoff and gas price adjustments for DA submissions
type retryStrategy struct {
	attempt         int
	backoff         time.Duration
	gasPrice        float64
	initialGasPrice float64
	maxAttempts     int
	maxBackoff      time.Duration
}

// newRetryStrategy creates a new retryStrategy with the given initial gas price, max backoff duration and max attempts
func newRetryStrategy(initialGasPrice float64, maxBackoff time.Duration, maxAttempts int) *retryStrategy {
	return &retryStrategy{
		attempt:         0,
		backoff:         0,
		gasPrice:        initialGasPrice,
		initialGasPrice: initialGasPrice,
		maxAttempts:     maxAttempts,
		maxBackoff:      maxBackoff,
	}
}

// ShouldContinue returns true if the retry strategy should continue attempting submissions
func (r *retryStrategy) ShouldContinue() bool {
	return r.attempt < r.maxAttempts
}

// NextAttempt increments the attempt counter
func (r *retryStrategy) NextAttempt() { r.attempt++ }

// ResetOnSuccess resets backoff and adjusts gas price downward after a successful submission
func (r *retryStrategy) ResetOnSuccess(gasMultiplier float64) {
	r.backoff = 0
	if gasMultiplier > 0 && r.gasPrice != noGasPrice {
		r.gasPrice = r.gasPrice / gasMultiplier
		if r.gasPrice < r.initialGasPrice {
			r.gasPrice = r.initialGasPrice
		}
	}
}

// BackoffOnFailure applies exponential backoff after a submission failure
func (r *retryStrategy) BackoffOnFailure() {
	r.backoff *= 2
	if r.backoff == 0 {
		r.backoff = initialBackoff
	}
	if r.backoff > r.maxBackoff {
		r.backoff = r.maxBackoff
	}
}

// BackoffOnMempool applies mempool-specific backoff and increases gas price when transaction is stuck in mempool
func (r *retryStrategy) BackoffOnMempool(mempoolTTL int, blockTime time.Duration, gasMultiplier float64) {
	r.backoff = blockTime * time.Duration(mempoolTTL)
	if gasMultiplier > 0 && r.gasPrice != noGasPrice {
		r.gasPrice = r.gasPrice * gasMultiplier
	}
}

// submissionOutcome captures the effect of a submission attempt for a batch
type submissionOutcome[T any] struct {
	SubmittedItems   []T
	RemainingItems   []T
	RemainingMarshal [][]byte
	NumSubmitted     int
	AllSubmitted     bool
}

// submissionBatch represents a batch of items with their marshaled data for DA submission
type submissionBatch[Item any] struct {
	Items     []Item
	Marshaled [][]byte
}

// DASubmitter handles DA submission operations
type DASubmitter struct {
	da      coreda.DA
	config  config.Config
	genesis genesis.Genesis
	options common.BlockOptions
	logger  zerolog.Logger
}

// NewDASubmitter creates a new DA submitter
func NewDASubmitter(
	da coreda.DA,
	config config.Config,
	genesis genesis.Genesis,
	options common.BlockOptions,
	logger zerolog.Logger,
) *DASubmitter {
	return &DASubmitter{
		da:      da,
		config:  config,
		genesis: genesis,
		options: options,
		logger:  logger.With().Str("component", "da_submitter").Logger(),
	}
}

// SubmitHeaders submits pending headers to DA layer
func (s *DASubmitter) SubmitHeaders(ctx context.Context, cache cache.Manager) error {
	headers, err := cache.GetPendingHeaders(ctx)
	if err != nil {
		return fmt.Errorf("failed to get pending headers: %w", err)
	}

	if len(headers) == 0 {
		return nil
	}

	s.logger.Info().Int("count", len(headers)).Msg("submitting headers to DA")

	return submitToDA(s, ctx, headers,
		func(header *types.SignedHeader) ([]byte, error) {
			headerPb, err := header.ToProto()
			if err != nil {
				return nil, fmt.Errorf("failed to convert header to proto: %w", err)
			}
			return proto.Marshal(headerPb)
		},
		func(submitted []*types.SignedHeader, res *coreda.ResultSubmit, gasPrice float64) {
			for _, header := range submitted {
				cache.SetHeaderDAIncluded(header.Hash().String(), res.Height)
			}
			// Update last submitted height
			if l := len(submitted); l > 0 {
				lastHeight := submitted[l-1].Height()
				cache.SetLastSubmittedHeaderHeight(ctx, lastHeight)
			}
		},
		"header",
		[]byte(s.config.DA.GetNamespace()),
		[]byte(s.config.DA.SubmitOptions),
		cache,
	)
}

// SubmitData submits pending data to DA layer
func (s *DASubmitter) SubmitData(ctx context.Context, cache cache.Manager, signer signer.Signer, genesis genesis.Genesis) error {
	dataList, err := cache.GetPendingData(ctx)
	if err != nil {
		return fmt.Errorf("failed to get pending data: %w", err)
	}

	if len(dataList) == 0 {
		return nil
	}

	// Sign the data
	signedDataList, err := s.createSignedData(dataList, signer, genesis)
	if err != nil {
		return fmt.Errorf("failed to create signed data: %w", err)
	}

	if len(signedDataList) == 0 {
		return nil // No non-empty data to submit
	}

	s.logger.Info().Int("count", len(signedDataList)).Msg("submitting data to DA")

	return submitToDA(s, ctx, signedDataList,
		func(signedData *types.SignedData) ([]byte, error) {
			return signedData.MarshalBinary()
		},
		func(submitted []*types.SignedData, res *coreda.ResultSubmit, gasPrice float64) {
			for _, sd := range submitted {
				cache.SetDataDAIncluded(sd.Data.DACommitment().String(), res.Height)
			}
			if l := len(submitted); l > 0 {
				lastHeight := submitted[l-1].Height()
				cache.SetLastSubmittedDataHeight(ctx, lastHeight)
			}
		},
		"data",
		[]byte(s.config.DA.GetDataNamespace()),
		[]byte(s.config.DA.SubmitOptions),
		cache,
	)
}

// createSignedData creates signed data from raw data
func (s *DASubmitter) createSignedData(dataList []*types.SignedData, signer signer.Signer, genesis genesis.Genesis) ([]*types.SignedData, error) {
	if signer == nil {
		return nil, fmt.Errorf("signer is nil")
	}

	pubKey, err := signer.GetPublic()
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	signerInfo := types.Signer{
		PubKey:  pubKey,
		Address: genesis.ProposerAddress,
	}

	signedDataList := make([]*types.SignedData, 0, len(dataList))

	for _, data := range dataList {
		// Skip empty data
		if len(data.Txs) == 0 {
			continue
		}

		// Sign the data
		dataBytes, err := data.Data.MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal data: %w", err)
		}

		signature, err := signer.Sign(dataBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to sign data: %w", err)
		}

		signedData := &types.SignedData{
			Data:      data.Data,
			Signature: signature,
			Signer:    signerInfo,
		}

		signedDataList = append(signedDataList, signedData)
	}

	return signedDataList, nil
}

// submitToDA is a generic helper for submitting items to the DA layer with retry, backoff, and gas price logic.
func submitToDA[T any](
    s *DASubmitter,
    ctx context.Context,
    items []T,
    marshalFn func(T) ([]byte, error),
    postSubmit func([]T, *coreda.ResultSubmit, float64),
    itemType string,
    namespace []byte,
    options []byte,
    cache cache.Manager,
) error {
    marshaled, err := marshalItems(items, marshalFn, itemType)
    if err != nil {
        return err
    }

    // Limit this submission to a single size-capped batch to respect DA load and avoid >2MB submissions
    if len(marshaled) > 0 {
        batchItems, batchMarshaled, err := limitBatchBySize(items, marshaled, maxBlobSize)
        if err != nil {
            return fmt.Errorf("no %s items fit within max blob size: %w", itemType, err)
        }
        items = batchItems
        marshaled = batchMarshaled
    }

	// Choose initial gas price: prefer configured value when >= 0, else DA-provided or default
	gasPrice := defaultGasPrice
	if s.config.DA.GasPrice == noGasPrice {
		gasPrice = noGasPrice
	} else if s.config.DA.GasPrice > 0 {
		gasPrice = s.config.DA.GasPrice
	} else if gp, err := s.da.GasPrice(ctx); err == nil {
		gasPrice = gp
	} else {
		s.logger.Warn().Err(err).Msg("failed to get gas price from DA layer, using default")
		gasPrice = defaultGasPrice
	}

	retry := newRetryStrategy(gasPrice, s.config.DA.BlockTime.Duration, s.config.DA.MaxSubmitAttempts)
	remaining := items
	numSubmitted := 0

	// Start the retry loop
	for retry.ShouldContinue() {
		if err := waitForBackoffOrContext(ctx, retry.backoff); err != nil {
			return err
		}

		retry.NextAttempt()

		submitCtx, cancel := context.WithTimeout(ctx, submissionTimeout)
		// Perform submission
		res := types.SubmitWithHelpers(submitCtx, s.da, s.logger, marshaled, retry.gasPrice, namespace, options)
		cancel()

		outcome := handleSubmissionResult(ctx, s, res, remaining, marshaled, retry, postSubmit, itemType, namespace, options)

		remaining = outcome.RemainingItems
		marshaled = outcome.RemainingMarshal
		numSubmitted += outcome.NumSubmitted

		if outcome.AllSubmitted {
			return nil
		}
	}

	return fmt.Errorf("failed to submit all %s(s) to DA layer, submitted %d items (%d left) after %d attempts",
		itemType, numSubmitted, len(remaining), retry.attempt)
}

// limitBatchBySize returns a prefix of items whose total marshaled size does not exceed maxBytes.
// If the first item exceeds maxBytes, it returns an error.
func limitBatchBySize[T any](items []T, marshaled [][]byte, maxBytes int) ([]T, [][]byte, error) {
    total := 0
    count := 0
    for i := 0; i < len(items); i++ {
        sz := len(marshaled[i])
        if sz > maxBytes {
            if i == 0 {
                return nil, nil, fmt.Errorf("item size %d exceeds max %d", sz, maxBytes)
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

func marshalItems[T any](
	items []T,
	marshalFn func(T) ([]byte, error),
	itemType string,
) ([][]byte, error) {
	marshaled := make([][]byte, len(items))
	for i, item := range items {
		bz, err := marshalFn(item)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal %s item: %w", itemType, err)
		}
		marshaled[i] = bz
	}
	return marshaled, nil
}

func waitForBackoffOrContext(ctx context.Context, backoff time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(backoff):
		return nil
	}
}

func handleSubmissionResult[T any](
	ctx context.Context,
	s *DASubmitter,
	res coreda.ResultSubmit,
	remaining []T,
	marshaled [][]byte,
	retry *retryStrategy,
	postSubmit func([]T, *coreda.ResultSubmit, float64),
	itemType string,
	namespace []byte,
	options []byte,
) submissionOutcome[T] {
	switch res.Code {
	case coreda.StatusSuccess:
		return handleSuccessfulSubmission(ctx, s, remaining, marshaled, &res, postSubmit, retry, itemType)

	case coreda.StatusNotIncludedInBlock, coreda.StatusAlreadyInMempool:
		return handleMempoolFailure(ctx, s, &res, retry, retry.attempt, remaining, marshaled)

	case coreda.StatusContextCanceled:
		s.logger.Info().Int("attempt", retry.attempt).Msg("DA layer submission canceled due to context cancellation")
		if daVisualizationServer := server.GetDAVisualizationServer(); daVisualizationServer != nil {
			daVisualizationServer.RecordSubmission(&res, retry.gasPrice, uint64(len(remaining)))
		}
		return submissionOutcome[T]{
			RemainingItems:   remaining,
			RemainingMarshal: marshaled,
			AllSubmitted:     false,
		}

	case coreda.StatusTooBig:
		return handleTooBigError(s, ctx, remaining, marshaled, retry, postSubmit, itemType, retry.attempt, namespace, options)

	default:
		return handleGenericFailure(s, &res, retry, retry.attempt, remaining, marshaled)
	}
}

func handleSuccessfulSubmission[T any](
	ctx context.Context,
	s *DASubmitter,
	remaining []T,
	marshaled [][]byte,
	res *coreda.ResultSubmit,
	postSubmit func([]T, *coreda.ResultSubmit, float64),
	retry *retryStrategy,
	itemType string,
) submissionOutcome[T] {
	// Record submission in DA visualization server
	if daVisualizationServer := server.GetDAVisualizationServer(); daVisualizationServer != nil {
		daVisualizationServer.RecordSubmission(res, retry.gasPrice, res.SubmittedCount)
	}

	s.logger.Info().Str("itemType", itemType).Float64("gasPrice", retry.gasPrice).Uint64("count", res.SubmittedCount).Msg("successfully submitted items to DA layer")

	submitted := remaining[:res.SubmittedCount]
	notSubmitted := remaining[res.SubmittedCount:]
	notSubmittedMarshaled := marshaled[res.SubmittedCount:]

	postSubmit(submitted, res, retry.gasPrice)

	gasMultiplier := s.getGasMultiplier(ctx)
	retry.ResetOnSuccess(gasMultiplier)
	s.logger.Debug().Dur("backoff", retry.backoff).Float64("gasPrice", retry.gasPrice).Msg("resetting DA layer submission options")

	return submissionOutcome[T]{
		SubmittedItems:   submitted,
		RemainingItems:   notSubmitted,
		RemainingMarshal: notSubmittedMarshaled,
		NumSubmitted:     int(res.SubmittedCount),
		AllSubmitted:     len(notSubmitted) == 0,
	}
}

func handleMempoolFailure[T any](
	ctx context.Context,
	s *DASubmitter,
	res *coreda.ResultSubmit,
	retry *retryStrategy,
	attempt int,
	remaining []T,
	marshaled [][]byte,
) submissionOutcome[T] {
	s.logger.Error().Str("error", res.Message).Int("attempt", attempt).Msg("DA layer submission failed")
	// Record failed submission in DA visualization server
	if daVisualizationServer := server.GetDAVisualizationServer(); daVisualizationServer != nil {
		daVisualizationServer.RecordSubmission(res, retry.gasPrice, uint64(len(remaining)))
	}

	gasMultiplier := s.getGasMultiplier(ctx)
	retry.BackoffOnMempool(int(s.config.DA.MempoolTTL), s.config.DA.BlockTime.Duration, gasMultiplier)
	s.logger.Info().Dur("backoff", retry.backoff).Float64("gasPrice", retry.gasPrice).Msg("retrying DA layer submission")

	return submissionOutcome[T]{
		RemainingItems:   remaining,
		RemainingMarshal: marshaled,
		AllSubmitted:     false,
	}
}

func handleTooBigError[T any](
	s *DASubmitter,
	ctx context.Context,
	remaining []T,
	marshaled [][]byte,
	retry *retryStrategy,
	postSubmit func([]T, *coreda.ResultSubmit, float64),
	itemType string,
	attempt int,
	namespace []byte,
	options []byte,
) submissionOutcome[T] {
	s.logger.Debug().Str("error", "blob too big").Int("attempt", attempt).Int("batchSize", len(remaining)).Msg("DA layer submission failed due to blob size limit")

	// Record failed submission in DA visualization server (create a result for TooBig error)
	if daVisualizationServer := server.GetDAVisualizationServer(); daVisualizationServer != nil {
		tooBigResult := &coreda.ResultSubmit{BaseResult: coreda.BaseResult{Code: coreda.StatusTooBig, Message: "blob too big"}}
		daVisualizationServer.RecordSubmission(tooBigResult, retry.gasPrice, uint64(len(remaining)))
	}

	if len(remaining) > 1 {
		totalSubmitted, err := submitWithRecursiveSplitting(s, ctx, remaining, marshaled, retry.gasPrice, postSubmit, itemType, namespace, options)
		if err != nil {
			// If splitting failed, we cannot continue with this batch
			s.logger.Error().Err(err).Str("itemType", itemType).Msg("recursive splitting failed")
			retry.BackoffOnFailure()
			return submissionOutcome[T]{
				RemainingItems:   remaining,
				RemainingMarshal: marshaled,
				NumSubmitted:     0,
				AllSubmitted:     false,
			}
		}

		if totalSubmitted > 0 {
			newRemaining := remaining[totalSubmitted:]
			newMarshaled := marshaled[totalSubmitted:]
			gasMultiplier := s.getGasMultiplier(ctx)
			retry.ResetOnSuccess(gasMultiplier)

			return submissionOutcome[T]{
				RemainingItems:   newRemaining,
				RemainingMarshal: newMarshaled,
				NumSubmitted:     totalSubmitted,
				AllSubmitted:     len(newRemaining) == 0,
			}
		}
		retry.BackoffOnFailure()
	} else {
		s.logger.Error().Str("itemType", itemType).Int("attempt", attempt).Msg("single item exceeds DA blob size limit")
		retry.BackoffOnFailure()
	}

	return submissionOutcome[T]{
		RemainingItems:   remaining,
		RemainingMarshal: marshaled,
		NumSubmitted:     0,
		AllSubmitted:     false,
	}
}

func handleGenericFailure[T any](
	s *DASubmitter,
	res *coreda.ResultSubmit,
	retry *retryStrategy,
	attempt int,
	remaining []T,
	marshaled [][]byte,
) submissionOutcome[T] {
	s.logger.Error().Str("error", res.Message).Int("attempt", attempt).Msg("DA layer submission failed")

	// Record failed submission in DA visualization server
	if daVisualizationServer := server.GetDAVisualizationServer(); daVisualizationServer != nil {
		daVisualizationServer.RecordSubmission(res, retry.gasPrice, uint64(len(remaining)))
	}

	retry.BackoffOnFailure()

	return submissionOutcome[T]{
		RemainingItems:   remaining,
		RemainingMarshal: marshaled,
		AllSubmitted:     false,
	}
}

// submitWithRecursiveSplitting handles recursive batch splitting when items are too big for DA submission.
// It returns the total number of items successfully submitted.
func submitWithRecursiveSplitting[T any](
	s *DASubmitter,
	ctx context.Context,
	items []T,
	marshaled [][]byte,
	gasPrice float64,
	postSubmit func([]T, *coreda.ResultSubmit, float64),
	itemType string,
	namespace []byte,
	options []byte,
) (int, error) {
	// Base case: no items to process
	if len(items) == 0 {
		return 0, nil
	}
	// Base case: single item that's too big - return error
	if len(items) == 1 {
		s.logger.Error().Str("itemType", itemType).Msg("single item exceeds DA blob size limit")
		return 0, fmt.Errorf("single %s item exceeds DA blob size limit", itemType)
	}

	// Split and submit recursively
	s.logger.Debug().Int("batchSize", len(items)).Msg("splitting batch for recursive submission")
	splitPoint := len(items) / 2
	if splitPoint == 0 {
		splitPoint = 1
	}
	firstHalf := items[:splitPoint]
	secondHalf := items[splitPoint:]
	firstHalfMarshaled := marshaled[:splitPoint]
	secondHalfMarshaled := marshaled[splitPoint:]

	s.logger.Debug().Int("originalSize", len(items)).Int("firstHalf", len(firstHalf)).Int("secondHalf", len(secondHalf)).Msg("splitting batch for recursion")

	firstSubmitted, err := submitHalfBatch[T](s, ctx, firstHalf, firstHalfMarshaled, gasPrice, postSubmit, itemType, namespace, options)
	if err != nil {
		return firstSubmitted, fmt.Errorf("first half submission failed: %w", err)
	}
	secondSubmitted, err := submitHalfBatch[T](s, ctx, secondHalf, secondHalfMarshaled, gasPrice, postSubmit, itemType, namespace, options)
	if err != nil {
		return firstSubmitted, fmt.Errorf("second half submission failed: %w", err)
	}
	return firstSubmitted + secondSubmitted, nil
}

// submitHalfBatch handles submission of a half batch, including recursive splitting if needed
func submitHalfBatch[T any](
	s *DASubmitter,
	ctx context.Context,
	items []T,
	marshaled [][]byte,
	gasPrice float64,
	postSubmit func([]T, *coreda.ResultSubmit, float64),
	itemType string,
	namespace []byte,
	options []byte,
) (int, error) {
	if len(items) == 0 {
		return 0, nil
	}

	batch := submissionBatch[T]{Items: items, Marshaled: marshaled}
	result := processBatch(s, ctx, batch, gasPrice, postSubmit, itemType, namespace, options)

	switch result.action {
	case batchActionSubmitted:
		if result.submittedCount < len(items) {
			remainingItems := items[result.submittedCount:]
			remainingMarshaled := marshaled[result.submittedCount:]
			remainingSubmitted, err := submitHalfBatch[T](s, ctx, remainingItems, remainingMarshaled, gasPrice, postSubmit, itemType, namespace, options)
			if err != nil {
				return result.submittedCount, err
			}
			return result.submittedCount + remainingSubmitted, nil
		}
		return result.submittedCount, nil
	case batchActionTooBig:
		return submitWithRecursiveSplitting[T](s, ctx, items, marshaled, gasPrice, postSubmit, itemType, namespace, options)
	case batchActionSkip:
		s.logger.Error().Str("itemType", itemType).Msg("single item exceeds DA blob size limit")
		return 0, fmt.Errorf("single %s item exceeds DA blob size limit", itemType)
	case batchActionFail:
		s.logger.Error().Str("itemType", itemType).Msg("unrecoverable error during batch submission")
		return 0, fmt.Errorf("unrecoverable error during %s batch submission", itemType)
	}
	return 0, nil
}

// batchAction represents the action to take after processing a batch
type batchAction int

const (
	batchActionSubmitted batchAction = iota // Batch was successfully submitted
	batchActionTooBig                       // Batch is too big and needs to be handled by caller
	batchActionSkip                         // Batch should be skipped (single item too big)
	batchActionFail                         // Unrecoverable error
)

// batchResult contains the result of processing a batch
type batchResult[T any] struct {
	action         batchAction
	submittedCount int
}

// processBatch processes a single batch and returns the result.
//
// Returns batchResult with one of the following actions:
// - batchActionSubmitted: Batch was successfully submitted (partial or complete)
// - batchActionTooBig: Batch is too big and needs to be handled by caller
// - batchActionSkip: Single item is too big and cannot be split further
// - batchActionFail: Unrecoverable error occurred (context timeout, network failure, etc.)
func processBatch[T any](
	s *DASubmitter,
	ctx context.Context,
	batch submissionBatch[T],
	gasPrice float64,
	postSubmit func([]T, *coreda.ResultSubmit, float64),
	itemType string,
	namespace []byte,
	options []byte,
) batchResult[T] {
	batchCtx, batchCtxCancel := context.WithTimeout(ctx, submissionTimeout)
	defer batchCtxCancel()

	batchRes := types.SubmitWithHelpers(batchCtx, s.da, s.logger, batch.Marshaled, gasPrice, namespace, options)

	if batchRes.Code == coreda.StatusSuccess {
		submitted := batch.Items[:batchRes.SubmittedCount]
		postSubmit(submitted, &batchRes, gasPrice)
		s.logger.Info().Int("batchSize", len(batch.Items)).Uint64("submittedCount", batchRes.SubmittedCount).Msg("successfully submitted batch to DA layer")
		if daVisualizationServer := server.GetDAVisualizationServer(); daVisualizationServer != nil {
			daVisualizationServer.RecordSubmission(&batchRes, gasPrice, batchRes.SubmittedCount)
		}
		return batchResult[T]{action: batchActionSubmitted, submittedCount: int(batchRes.SubmittedCount)}
	}

	if daVisualizationServer := server.GetDAVisualizationServer(); daVisualizationServer != nil {
		daVisualizationServer.RecordSubmission(&batchRes, gasPrice, uint64(len(batch.Items)))
	}

	if batchRes.Code == coreda.StatusTooBig && len(batch.Items) > 1 {
		s.logger.Debug().Int("batchSize", len(batch.Items)).Msg("batch too big, returning to caller for splitting")
		return batchResult[T]{action: batchActionTooBig}
	}

	if len(batch.Items) == 1 && batchRes.Code == coreda.StatusTooBig {
		s.logger.Error().Str("itemType", itemType).Msg("single item exceeds DA blob size limit")
		return batchResult[T]{action: batchActionSkip}
	}

	return batchResult[T]{action: batchActionFail}
}
