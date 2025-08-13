package block

import (
	"context"
	"fmt"
	"time"

	coreda "github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/pkg/rpc/server"
	"github.com/evstack/ev-node/types"
	"google.golang.org/protobuf/proto"
)

const (
	submissionTimeout    = 60 * time.Second
	noGasPrice           = -1
	initialBackoff       = 100 * time.Millisecond
	defaultGasPrice      = 0.0
	defaultGasMultiplier = 1.0
)

// getGasMultiplier fetches the gas multiplier from DA layer with fallback to default value
func (m *Manager) getGasMultiplier(ctx context.Context) float64 {
	gasMultiplier, err := m.da.GasMultiplier(ctx)
	if err != nil {
		m.logger.Warn().Err(err).Msg("failed to get gas multiplier from DA layer, using default")
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
func (r *retryStrategy) NextAttempt() {
	r.attempt++
}

// ResetOnSuccess resets backoff and adjusts gas price downward after a successful submission
func (r *retryStrategy) ResetOnSuccess(gasMultiplier float64) {
	r.backoff = 0
	if gasMultiplier > 0 && r.gasPrice != noGasPrice {
		r.gasPrice = r.gasPrice / gasMultiplier
		r.gasPrice = max(r.gasPrice, r.initialGasPrice)
	}
}

// BackoffOnFailure applies exponential backoff after a submission failure
func (r *retryStrategy) BackoffOnFailure() {
	r.backoff *= 2
	if r.backoff == 0 {
		r.backoff = initialBackoff // initialBackoff value
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

// HeaderSubmissionLoop is responsible for submitting headers to the DA layer.
func (m *Manager) HeaderSubmissionLoop(ctx context.Context) {
	timer := time.NewTicker(m.config.DA.BlockTime.Duration)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			m.logger.Info().Msg("header submission loop stopped")
			return
		case <-timer.C:
		}
		if m.pendingHeaders.isEmpty() {
			continue
		}
		headersToSubmit, err := m.pendingHeaders.getPendingHeaders(ctx)
		if err != nil {
			m.logger.Error().Err(err).Msg("error while fetching headers pending DA")
			continue
		}
		if len(headersToSubmit) == 0 {
			continue
		}
		err = m.submitHeadersToDA(ctx, headersToSubmit)
		if err != nil {
			m.logger.Error().Err(err).Msg("error while submitting header to DA")
		}
	}
}

// submitHeadersToDA submits a list of headers to the DA layer using the generic submitToDA helper.
func (m *Manager) submitHeadersToDA(ctx context.Context, headersToSubmit []*types.SignedHeader) error {
	return submitToDA(m, ctx, headersToSubmit,
		func(header *types.SignedHeader) ([]byte, error) {
			headerPb, err := header.ToProto()
			if err != nil {
				return nil, fmt.Errorf("failed to transform header to proto: %w", err)
			}
			return proto.Marshal(headerPb)
		},
		func(submitted []*types.SignedHeader, res *coreda.ResultSubmit, gasPrice float64) {
			for _, header := range submitted {
				m.headerCache.SetDAIncluded(header.Hash().String(), res.Height)
			}
			lastSubmittedHeaderHeight := uint64(0)
			if l := len(submitted); l > 0 {
				lastSubmittedHeaderHeight = submitted[l-1].Height()
			}
			m.pendingHeaders.setLastSubmittedHeaderHeight(ctx, lastSubmittedHeaderHeight)
			// Update sequencer metrics if the sequencer supports it
			if seq, ok := m.sequencer.(MetricsRecorder); ok {
				seq.RecordMetrics(gasPrice, res.BlobSize, res.Code, m.pendingHeaders.numPendingHeaders(), lastSubmittedHeaderHeight)
			}
			m.sendNonBlockingSignalToDAIncluderCh()
		},
		"header",
		[]byte(m.config.DA.GetHeaderNamespace()),
	)
}

// DataSubmissionLoop is responsible for submitting data to the DA layer.
func (m *Manager) DataSubmissionLoop(ctx context.Context) {
	timer := time.NewTicker(m.config.DA.BlockTime.Duration)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			m.logger.Info().Msg("data submission loop stopped")
			return
		case <-timer.C:
		}
		if m.pendingData.isEmpty() {
			continue
		}

		signedDataToSubmit, err := m.createSignedDataToSubmit(ctx)
		if err != nil {
			m.logger.Error().Err(err).Msg("failed to create signed data to submit")
			continue
		}
		if len(signedDataToSubmit) == 0 {
			continue
		}

		err = m.submitDataToDA(ctx, signedDataToSubmit)
		if err != nil {
			m.logger.Error().Err(err).Msg("failed to submit data to DA")
		}
	}
}

// submitDataToDA submits a list of signed data to the DA layer using the generic submitToDA helper.
func (m *Manager) submitDataToDA(ctx context.Context, signedDataToSubmit []*types.SignedData) error {
	return submitToDA(m, ctx, signedDataToSubmit,
		func(signedData *types.SignedData) ([]byte, error) {
			return signedData.MarshalBinary()
		},
		func(submitted []*types.SignedData, res *coreda.ResultSubmit, gasPrice float64) {
			for _, signedData := range submitted {
				m.dataCache.SetDAIncluded(signedData.Data.DACommitment().String(), res.Height)
			}
			lastSubmittedDataHeight := uint64(0)
			if l := len(submitted); l > 0 {
				lastSubmittedDataHeight = submitted[l-1].Height()
			}
			m.pendingData.setLastSubmittedDataHeight(ctx, lastSubmittedDataHeight)
			// Update sequencer metrics if the sequencer supports it
			if seq, ok := m.sequencer.(MetricsRecorder); ok {
				seq.RecordMetrics(gasPrice, res.BlobSize, res.Code, m.pendingData.numPendingData(), lastSubmittedDataHeight)
			}
			m.sendNonBlockingSignalToDAIncluderCh()
		},
		"data",
		[]byte(m.config.DA.GetDataNamespace()),
	)
}

// submitToDA is a generic helper for submitting items to the DA layer with retry, backoff, and gas price logic.
func submitToDA[T any](
	m *Manager,
	ctx context.Context,
	items []T,
	marshalFn func(T) ([]byte, error),
	postSubmit func([]T, *coreda.ResultSubmit, float64),
	itemType string,
	namespace []byte,
) error {
	marshaled, err := marshalItems(items, marshalFn, itemType)
	if err != nil {
		return err
	}

	gasPrice, err := m.da.GasPrice(ctx)
	if err != nil {
		m.logger.Warn().Err(err).Msg("failed to get gas price from DA layer, using default")
		gasPrice = defaultGasPrice
	}

	retryStrategy := newRetryStrategy(gasPrice, m.config.DA.BlockTime.Duration, m.config.DA.MaxSubmitAttempts)
	remaining := items
	numSubmitted := 0

	// Start the retry loop
	for retryStrategy.ShouldContinue() {
		if err := waitForBackoffOrContext(ctx, retryStrategy.backoff); err != nil {
			return err
		}

		retryStrategy.NextAttempt()

		submitCtx, cancel := context.WithTimeout(ctx, submissionTimeout)
		m.recordDAMetrics("submission", DAModeRetry)

		res := types.SubmitWithHelpers(submitCtx, m.da, m.logger, marshaled, retryStrategy.gasPrice, namespace, nil)
		cancel()

		outcome := handleSubmissionResult(ctx, m, res, remaining, marshaled, retryStrategy, postSubmit, itemType, namespace)

		remaining = outcome.RemainingItems
		marshaled = outcome.RemainingMarshal
		numSubmitted += outcome.NumSubmitted

		if outcome.AllSubmitted {
			return nil
		}
	}

	return fmt.Errorf("failed to submit all %s(s) to DA layer, submitted %d items (%d left) after %d attempts",
		itemType, numSubmitted, len(remaining), retryStrategy.attempt)
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
	m *Manager,
	res coreda.ResultSubmit,
	remaining []T,
	marshaled [][]byte,
	retryStrategy *retryStrategy,
	postSubmit func([]T, *coreda.ResultSubmit, float64),
	itemType string,
	namespace []byte,
) submissionOutcome[T] {
	switch res.Code {
	case coreda.StatusSuccess:
		return handleSuccessfulSubmission(ctx, m, remaining, marshaled, &res, postSubmit, retryStrategy, itemType)

	case coreda.StatusNotIncludedInBlock, coreda.StatusAlreadyInMempool:
		return handleMempoolFailure(ctx, m, &res, retryStrategy, retryStrategy.attempt, remaining, marshaled)

	case coreda.StatusContextCanceled:
		m.logger.Info().Int("attempt", retryStrategy.attempt).Msg("DA layer submission canceled due to context cancellation")

		// Record canceled submission in DA visualization server
		if daVisualizationServer := server.GetDAVisualizationServer(); daVisualizationServer != nil {
			daVisualizationServer.RecordSubmission(&res, retryStrategy.gasPrice, uint64(len(remaining)))
		}

		return submissionOutcome[T]{
			RemainingItems:   remaining,
			RemainingMarshal: marshaled,
			AllSubmitted:     false,
		}

	case coreda.StatusTooBig:
		return handleTooBigError(m, ctx, remaining, marshaled, retryStrategy, postSubmit, itemType, retryStrategy.attempt, namespace)

	default:
		return handleGenericFailure(m, &res, retryStrategy, retryStrategy.attempt, remaining, marshaled)
	}
}

func handleSuccessfulSubmission[T any](
	ctx context.Context,
	m *Manager,
	remaining []T,
	marshaled [][]byte,
	res *coreda.ResultSubmit,
	postSubmit func([]T, *coreda.ResultSubmit, float64),
	retryStrategy *retryStrategy,
	itemType string,
) submissionOutcome[T] {
	m.recordDAMetrics("submission", DAModeSuccess)

	remLen := len(remaining)
	allSubmitted := res.SubmittedCount == uint64(remLen)

	// Record submission in DA visualization server
	if daVisualizationServer := server.GetDAVisualizationServer(); daVisualizationServer != nil {
		daVisualizationServer.RecordSubmission(res, retryStrategy.gasPrice, res.SubmittedCount)
	}

	m.logger.Info().Str("itemType", itemType).Float64("gasPrice", retryStrategy.gasPrice).Uint64("count", res.SubmittedCount).Msg("successfully submitted items to DA layer")

	submitted := remaining[:res.SubmittedCount]
	notSubmitted := remaining[res.SubmittedCount:]
	notSubmittedMarshaled := marshaled[res.SubmittedCount:]

	postSubmit(submitted, res, retryStrategy.gasPrice)

	gasMultiplier := m.getGasMultiplier(ctx)

	retryStrategy.ResetOnSuccess(gasMultiplier)

	m.logger.Debug().Dur("backoff", retryStrategy.backoff).Float64("gasPrice", retryStrategy.gasPrice).Msg("resetting DA layer submission options")

	return submissionOutcome[T]{
		SubmittedItems:   submitted,
		RemainingItems:   notSubmitted,
		RemainingMarshal: notSubmittedMarshaled,
		NumSubmitted:     int(res.SubmittedCount),
		AllSubmitted:     allSubmitted,
	}
}

func handleMempoolFailure[T any](
	ctx context.Context,
	m *Manager,
	res *coreda.ResultSubmit,
	retryStrategy *retryStrategy,
	attempt int,
	remaining []T,
	marshaled [][]byte,
) submissionOutcome[T] {
	m.logger.Error().Str("error", res.Message).Int("attempt", attempt).Msg("DA layer submission failed")

	m.recordDAMetrics("submission", DAModeFail)

	// Record failed submission in DA visualization server
	if daVisualizationServer := server.GetDAVisualizationServer(); daVisualizationServer != nil {
		daVisualizationServer.RecordSubmission(res, retryStrategy.gasPrice, uint64(len(remaining)))
	}

	gasMultiplier := m.getGasMultiplier(ctx)
	retryStrategy.BackoffOnMempool(int(m.config.DA.MempoolTTL), m.config.DA.BlockTime.Duration, gasMultiplier)
	m.logger.Info().Dur("backoff", retryStrategy.backoff).Float64("gasPrice", retryStrategy.gasPrice).Msg("retrying DA layer submission with")

	return submissionOutcome[T]{
		RemainingItems:   remaining,
		RemainingMarshal: marshaled,
		AllSubmitted:     false,
	}
}

func handleTooBigError[T any](
	m *Manager,
	ctx context.Context,
	remaining []T,
	marshaled [][]byte,
	retryStrategy *retryStrategy,
	postSubmit func([]T, *coreda.ResultSubmit, float64),
	itemType string,
	attempt int,
	namespace []byte,
) submissionOutcome[T] {
	m.logger.Debug().Str("error", "blob too big").Int("attempt", attempt).Int("batchSize", len(remaining)).Msg("DA layer submission failed due to blob size limit")

	m.recordDAMetrics("submission", DAModeFail)

	// Record failed submission in DA visualization server (create a result for TooBig error)
	if daVisualizationServer := server.GetDAVisualizationServer(); daVisualizationServer != nil {
		tooBigResult := &coreda.ResultSubmit{
			BaseResult: coreda.BaseResult{
				Code:    coreda.StatusTooBig,
				Message: "blob too big",
			},
		}
		daVisualizationServer.RecordSubmission(tooBigResult, retryStrategy.gasPrice, uint64(len(remaining)))
	}

	if len(remaining) > 1 {
		totalSubmitted, err := submitWithRecursiveSplitting(m, ctx, remaining, marshaled, retryStrategy.gasPrice, postSubmit, itemType, namespace)
		if err != nil {
			// If splitting failed, we cannot continue with this batch
			m.logger.Error().Err(err).Str("itemType", itemType).Msg("recursive splitting failed")
			retryStrategy.BackoffOnFailure()
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
			gasMultiplier := m.getGasMultiplier(ctx)
			retryStrategy.ResetOnSuccess(gasMultiplier)

			return submissionOutcome[T]{
				RemainingItems:   newRemaining,
				RemainingMarshal: newMarshaled,
				NumSubmitted:     totalSubmitted,
				AllSubmitted:     len(newRemaining) == 0,
			}
		} else {
			retryStrategy.BackoffOnFailure()
		}
	} else {
		m.logger.Error().Str("itemType", itemType).Int("attempt", attempt).Msg("single item exceeds DA blob size limit")
		retryStrategy.BackoffOnFailure()
	}

	return submissionOutcome[T]{
		RemainingItems:   remaining,
		RemainingMarshal: marshaled,
		NumSubmitted:     0,
		AllSubmitted:     false,
	}
}

func handleGenericFailure[T any](
	m *Manager,
	res *coreda.ResultSubmit,
	retryStrategy *retryStrategy,
	attempt int,
	remaining []T,
	marshaled [][]byte,
) submissionOutcome[T] {
	m.logger.Error().Str("error", res.Message).Int("attempt", attempt).Msg("DA layer submission failed")

	m.recordDAMetrics("submission", DAModeFail)

	// Record failed submission in DA visualization server
	if daVisualizationServer := server.GetDAVisualizationServer(); daVisualizationServer != nil {
		daVisualizationServer.RecordSubmission(res, retryStrategy.gasPrice, uint64(len(remaining)))
	}

	retryStrategy.BackoffOnFailure()

	return submissionOutcome[T]{
		RemainingItems:   remaining,
		RemainingMarshal: marshaled,
		AllSubmitted:     false,
	}
}

// createSignedDataToSubmit converts the list of pending data to a list of SignedData.
func (m *Manager) createSignedDataToSubmit(ctx context.Context) ([]*types.SignedData, error) {
	dataList, err := m.pendingData.getPendingData(ctx)
	if err != nil {
		return nil, err
	}

	if m.signer == nil {
		return nil, fmt.Errorf("signer is nil; cannot sign data")
	}

	pubKey, err := m.signer.GetPublic()
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	signer := types.Signer{
		PubKey:  pubKey,
		Address: m.genesis.ProposerAddress,
	}

	signedDataToSubmit := make([]*types.SignedData, 0, len(dataList))

	for _, data := range dataList {
		if len(data.Txs) == 0 {
			continue
		}
		signature, err := m.getDataSignature(data)
		if err != nil {
			return nil, fmt.Errorf("failed to get data signature: %w", err)
		}
		signedDataToSubmit = append(signedDataToSubmit, &types.SignedData{
			Data:      *data,
			Signature: signature,
			Signer:    signer,
		})
	}

	return signedDataToSubmit, nil
}

// submitWithRecursiveSplitting handles recursive batch splitting when items are too big for DA submission.
// It returns the total number of items successfully submitted.
func submitWithRecursiveSplitting[T any](
	m *Manager,
	ctx context.Context,
	items []T,
	marshaled [][]byte,
	gasPrice float64,
	postSubmit func([]T, *coreda.ResultSubmit, float64),
	itemType string,
	namespace []byte,
) (int, error) {
	// Base case: no items to process
	if len(items) == 0 {
		return 0, nil
	}

	// Base case: single item that's too big - return error
	if len(items) == 1 {
		m.logger.Error().Str("itemType", itemType).Msg("single item exceeds DA blob size limit")
		return 0, fmt.Errorf("single %s item exceeds DA blob size limit", itemType)
	}

	// Split and submit recursively - we know the batch is too big
	m.logger.Debug().Int("batchSize", len(items)).Msg("splitting batch for recursive submission")

	splitPoint := len(items) / 2
	// Ensure we actually split (avoid infinite recursion)
	if splitPoint == 0 {
		splitPoint = 1
	}
	firstHalf := items[:splitPoint]
	secondHalf := items[splitPoint:]
	firstHalfMarshaled := marshaled[:splitPoint]
	secondHalfMarshaled := marshaled[splitPoint:]

	m.logger.Debug().Int("originalSize", len(items)).Int("firstHalf", len(firstHalf)).Int("secondHalf", len(secondHalf)).Msg("splitting batch for recursion")

	// Recursively submit both halves using processBatch directly
	firstSubmitted, err := submitHalfBatch[T](m, ctx, firstHalf, firstHalfMarshaled, gasPrice, postSubmit, itemType, namespace)
	if err != nil {
		return firstSubmitted, fmt.Errorf("first half submission failed: %w", err)
	}

	secondSubmitted, err := submitHalfBatch[T](m, ctx, secondHalf, secondHalfMarshaled, gasPrice, postSubmit, itemType, namespace)
	if err != nil {
		return firstSubmitted, fmt.Errorf("second half submission failed: %w", err)
	}

	return firstSubmitted + secondSubmitted, nil
}

// submitHalfBatch handles submission of a half batch, including recursive splitting if needed
func submitHalfBatch[T any](
	m *Manager,
	ctx context.Context,
	items []T,
	marshaled [][]byte,
	gasPrice float64,
	postSubmit func([]T, *coreda.ResultSubmit, float64),
	itemType string,
	namespace []byte,
) (int, error) {
	// Base case: no items to process
	if len(items) == 0 {
		return 0, nil
	}

	// Try to submit the batch as-is first
	batch := submissionBatch[T]{Items: items, Marshaled: marshaled}
	result := processBatch(m, ctx, batch, gasPrice, postSubmit, itemType, namespace)

	switch result.action {
	case batchActionSubmitted:
		// Success! Handle potential partial submission
		if result.submittedCount < len(items) {
			// Some items were submitted, recursively handle the rest
			remainingItems := items[result.submittedCount:]
			remainingMarshaled := marshaled[result.submittedCount:]
			remainingSubmitted, err := submitHalfBatch[T](m, ctx, remainingItems, remainingMarshaled, gasPrice, postSubmit, itemType, namespace)
			if err != nil {
				return result.submittedCount, err
			}
			return result.submittedCount + remainingSubmitted, nil
		}
		// All items submitted
		return result.submittedCount, nil

	case batchActionTooBig:
		// Batch too big - need to split further
		return submitWithRecursiveSplitting[T](m, ctx, items, marshaled, gasPrice, postSubmit, itemType, namespace)

	case batchActionSkip:
		// Single item too big - return error
		m.logger.Error().Str("itemType", itemType).Msg("single item exceeds DA blob size limit")
		return 0, fmt.Errorf("single %s item exceeds DA blob size limit", itemType)

	case batchActionFail:
		// Unrecoverable error - stop processing
		m.logger.Error().Str("itemType", itemType).Msg("unrecoverable error during batch submission")
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
	splitBatches   []submissionBatch[T]
}

// processBatch processes a single batch and returns the result.
//
// Returns batchResult with one of the following actions:
// - batchActionSubmitted: Batch was successfully submitted (partial or complete)
// - batchActionTooBig: Batch is too big and needs to be handled by caller
// - batchActionSkip: Single item is too big and cannot be split further
// - batchActionFail: Unrecoverable error occurred (context timeout, network failure, etc.)
func processBatch[T any](
	m *Manager,
	ctx context.Context,
	batch submissionBatch[T],
	gasPrice float64,
	postSubmit func([]T, *coreda.ResultSubmit, float64),
	itemType string,
	namespace []byte,
) batchResult[T] {
	batchCtx, batchCtxCancel := context.WithTimeout(ctx, submissionTimeout)
	defer batchCtxCancel()

	batchRes := types.SubmitWithHelpers(batchCtx, m.da, m.logger, batch.Marshaled, gasPrice, namespace, nil)

	if batchRes.Code == coreda.StatusSuccess {
		// Successfully submitted this batch
		submitted := batch.Items[:batchRes.SubmittedCount]
		postSubmit(submitted, &batchRes, gasPrice)
		m.logger.Info().Int("batchSize", len(batch.Items)).Uint64("submittedCount", batchRes.SubmittedCount).Msg("successfully submitted batch to DA layer")

		// Record successful submission in DA visualization server
		if daVisualizationServer := server.GetDAVisualizationServer(); daVisualizationServer != nil {
			daVisualizationServer.RecordSubmission(&batchRes, gasPrice, batchRes.SubmittedCount)
		}

		return batchResult[T]{
			action:         batchActionSubmitted,
			submittedCount: int(batchRes.SubmittedCount),
		}
	}

	// Record failed submission in DA visualization server for all error cases
	if daVisualizationServer := server.GetDAVisualizationServer(); daVisualizationServer != nil {
		daVisualizationServer.RecordSubmission(&batchRes, gasPrice, uint64(len(batch.Items)))
	}

	if batchRes.Code == coreda.StatusTooBig && len(batch.Items) > 1 {
		// Batch is too big - let the caller handle splitting
		m.logger.Debug().Int("batchSize", len(batch.Items)).Msg("batch too big, returning to caller for splitting")
		return batchResult[T]{action: batchActionTooBig}
	}

	if len(batch.Items) == 1 && batchRes.Code == coreda.StatusTooBig {
		m.logger.Error().Str("itemType", itemType).Msg("single item exceeds DA blob size limit")
		return batchResult[T]{action: batchActionSkip}
	}

	// Other error - cannot continue with this batch
	return batchResult[T]{action: batchActionFail}
}
