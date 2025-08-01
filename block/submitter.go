package block

import (
	"context"
	"fmt"
	"time"

	coreda "github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/types"
	"google.golang.org/protobuf/proto"
)

const (
	submissionTimeout = 60 * time.Second
	noGasPrice        = -1
	initialBackoff    = 100 * time.Millisecond
)

// retryStrategy manages retry logic with backoff and gas price adjustments for DA submissions
type retryStrategy struct {
	attempt         int
	backoff         time.Duration
	gasPrice        float64
	initialGasPrice float64
	maxAttempts     int
	maxBackoff      time.Duration
}

// newRetryStrategy creates a new retryStrategy with the given initial gas price and max backoff duration
func newRetryStrategy(initialGasPrice float64, maxBackoff time.Duration) *retryStrategy {
	return &retryStrategy{
		attempt:         0,
		backoff:         0,
		gasPrice:        initialGasPrice,
		initialGasPrice: initialGasPrice,
		maxAttempts:     maxSubmitAttempts,
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

type SubmissionOutcome[T any] struct {
	SubmittedItems   []T
	RemainingItems   []T
	RemainingMarshal [][]byte
	NumSubmitted     int
	AllSubmitted     bool
}

// SubmissionBatch represents a batch of items with their marshaled data for DA submission
type SubmissionBatch[Item any] struct {
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
			m.logger.Info("header submission loop stopped")
			return
		case <-timer.C:
		}
		if m.pendingHeaders.isEmpty() {
			continue
		}
		headersToSubmit, err := m.pendingHeaders.getPendingHeaders(ctx)
		if err != nil {
			m.logger.Error("error while fetching headers pending DA", "err", err)
			continue
		}
		if len(headersToSubmit) == 0 {
			continue
		}
		err = m.submitHeadersToDA(ctx, headersToSubmit)
		if err != nil {
			m.logger.Error("error while submitting header to DA", "error", err)
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
	)
}

// DataSubmissionLoop is responsible for submitting data to the DA layer.
func (m *Manager) DataSubmissionLoop(ctx context.Context) {
	timer := time.NewTicker(m.config.DA.BlockTime.Duration)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			m.logger.Info("data submission loop stopped")
			return
		case <-timer.C:
		}
		if m.pendingData.isEmpty() {
			continue
		}

		signedDataToSubmit, err := m.createSignedDataToSubmit(ctx)
		if err != nil {
			m.logger.Error("failed to create signed data to submit", "error", err)
			continue
		}
		if len(signedDataToSubmit) == 0 {
			continue
		}

		err = m.submitDataToDA(ctx, signedDataToSubmit)
		if err != nil {
			m.logger.Error("failed to submit data to DA", "error", err)
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
) error {
	marshaled, err := marshalItems(items, marshalFn, itemType)
	if err != nil {
		return err
	}

	retryStrategy := newRetryStrategy(m.gasPrice, m.config.DA.BlockTime.Duration)
	remaining := items
	numSubmitted := 0

	// Start the retry loop
	for retryStrategy.ShouldContinue() {
		if err := waitForBackoffOrContext(ctx, retryStrategy.backoff); err != nil {
			return err
		}

		submitCtx, cancel := context.WithTimeout(ctx, submissionTimeout)
		m.recordDAMetrics("submission", DAModeRetry)

		res := types.SubmitWithHelpers(submitCtx, m.da, m.logger, marshaled, retryStrategy.gasPrice, nil)
		cancel()

		outcome := handleSubmissionResult(m, ctx, res, remaining, marshaled, retryStrategy, postSubmit, itemType)

		remaining = outcome.RemainingItems
		marshaled = outcome.RemainingMarshal
		numSubmitted += outcome.NumSubmitted

		if outcome.AllSubmitted {
			return nil
		}

		retryStrategy.NextAttempt()
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
	m *Manager,
	ctx context.Context,
	res coreda.ResultSubmit,
	remaining []T,
	marshaled [][]byte,
	retryStrategy *retryStrategy,
	postSubmit func([]T, *coreda.ResultSubmit, float64),
	itemType string,
) SubmissionOutcome[T] {
	switch res.Code {
	case coreda.StatusSuccess:
		return handleSuccessfulSubmission(m, remaining, marshaled, &res, postSubmit, retryStrategy, itemType)

	case coreda.StatusNotIncludedInBlock, coreda.StatusAlreadyInMempool:
		return handleMempoolFailure(m, &res, retryStrategy, retryStrategy.attempt, remaining, marshaled)

	case coreda.StatusContextCanceled:
		m.logger.Info("DA layer submission canceled due to context cancellation", "attempt", retryStrategy.attempt)
		return SubmissionOutcome[T]{AllSubmitted: true}

	case coreda.StatusTooBig:
		return handleTooBigError(m, ctx, remaining, marshaled, retryStrategy, postSubmit, itemType, retryStrategy.attempt)

	default:
		return handleGenericFailure(m, &res, retryStrategy, retryStrategy.attempt, remaining, marshaled)
	}
}

func handleSuccessfulSubmission[T any](
	m *Manager,
	remaining []T,
	marshaled [][]byte,
	res *coreda.ResultSubmit,
	postSubmit func([]T, *coreda.ResultSubmit, float64),
	retryStrategy *retryStrategy,
	itemType string,
) SubmissionOutcome[T] {
	m.recordDAMetrics("submission", DAModeSuccess)

	remLen := len(remaining)
	allSubmitted := res.SubmittedCount == uint64(remLen)

	m.logger.Info("successfully submitted items to DA layer",
		"itemType", itemType,
		"gasPrice", retryStrategy.gasPrice,
		"count", res.SubmittedCount)

	submitted := remaining[:res.SubmittedCount]
	notSubmitted := remaining[res.SubmittedCount:]
	notSubmittedMarshaled := marshaled[res.SubmittedCount:]

	postSubmit(submitted, res, retryStrategy.gasPrice)
	retryStrategy.ResetOnSuccess(m.gasMultiplier)

	m.logger.Debug("resetting DA layer submission options",
		"backoff", retryStrategy.backoff,
		"gasPrice", retryStrategy.gasPrice)

	return SubmissionOutcome[T]{
		SubmittedItems:   submitted,
		RemainingItems:   notSubmitted,
		RemainingMarshal: notSubmittedMarshaled,
		NumSubmitted:     int(res.SubmittedCount),
		AllSubmitted:     allSubmitted,
	}
}

func handleMempoolFailure[T any](
	m *Manager,
	res *coreda.ResultSubmit,
	retryStrategy *retryStrategy,
	attempt int,
	remaining []T,
	marshaled [][]byte,
) SubmissionOutcome[T] {
	m.logger.Error("DA layer submission failed",
		"error", res.Message,
		"attempt", attempt)

	m.recordDAMetrics("submission", DAModeFail)
	retryStrategy.BackoffOnMempool(int(m.config.DA.MempoolTTL), m.config.DA.BlockTime.Duration, m.gasMultiplier)

	m.logger.Info("retrying DA layer submission with",
		"backoff", retryStrategy.backoff,
		"gasPrice", retryStrategy.gasPrice)

	return SubmissionOutcome[T]{
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
) SubmissionOutcome[T] {
	m.logger.Warn("DA layer submission failed due to blob size limit",
		"error", "blob too big",
		"attempt", attempt,
		"batchSize", len(remaining))

	m.recordDAMetrics("submission", DAModeFail)

	if len(remaining) > 1 {
		totalSubmitted := submitWithRecursiveSplitting(m, ctx, remaining, marshaled, retryStrategy.gasPrice, postSubmit, itemType)

		if totalSubmitted > 0 {
			newRemaining := remaining[totalSubmitted:]
			newMarshaled := marshaled[totalSubmitted:]
			retryStrategy.ResetOnSuccess(m.gasMultiplier)

			return SubmissionOutcome[T]{
				RemainingItems:   newRemaining,
				RemainingMarshal: newMarshaled,
				NumSubmitted:     totalSubmitted,
				AllSubmitted:     len(newRemaining) == 0,
			}
		} else {
			retryStrategy.BackoffOnFailure()
		}
	} else {
		m.logger.Error("single item exceeds DA blob size limit",
			"itemType", itemType,
			"attempt", attempt)
		retryStrategy.BackoffOnFailure()
	}

	return SubmissionOutcome[T]{
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
) SubmissionOutcome[T] {
	m.logger.Error("DA layer submission failed",
		"error", res.Message,
		"attempt", attempt)

	m.recordDAMetrics("submission", DAModeFail)
	retryStrategy.BackoffOnFailure()

	return SubmissionOutcome[T]{
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
) int {
	// Base case: no items to process
	if len(items) == 0 {
		return 0
	}

	// Base case: single item that's too big - skip it
	if len(items) == 1 {
		m.logger.Error("single item exceeds DA blob size limit", "itemType", itemType)
		return 0
	}

	// Split and submit recursively - we know the batch is too big
	m.logger.Info("splitting batch for recursive submission", "batchSize", len(items))

	splitPoint := len(items) / 2
	// Ensure we actually split (avoid infinite recursion)
	if splitPoint == 0 {
		splitPoint = 1
	}
	firstHalf := items[:splitPoint]
	secondHalf := items[splitPoint:]
	firstHalfMarshaled := marshaled[:splitPoint]
	secondHalfMarshaled := marshaled[splitPoint:]

	m.logger.Info("splitting batch for recursion", "originalSize", len(items), "firstHalf", len(firstHalf), "secondHalf", len(secondHalf))

	// Recursively submit both halves using processBatch directly
	firstSubmitted := submitHalfBatch[T](m, ctx, firstHalf, firstHalfMarshaled, gasPrice, postSubmit, itemType)
	secondSubmitted := submitHalfBatch[T](m, ctx, secondHalf, secondHalfMarshaled, gasPrice, postSubmit, itemType)

	return firstSubmitted + secondSubmitted
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
) int {
	// Base case: no items to process
	if len(items) == 0 {
		return 0
	}

	// Try to submit the batch as-is first
	batch := SubmissionBatch[T]{Items: items, Marshaled: marshaled}
	result := processBatch(m, ctx, batch, gasPrice, postSubmit, itemType)

	switch result.action {
	case batchActionSubmitted:
		// Success! Handle potential partial submission
		if result.submittedCount < len(items) {
			// Some items were submitted, recursively handle the rest
			remainingItems := items[result.submittedCount:]
			remainingMarshaled := marshaled[result.submittedCount:]
			remainingSubmitted := submitHalfBatch[T](m, ctx, remainingItems, remainingMarshaled, gasPrice, postSubmit, itemType)
			return result.submittedCount + remainingSubmitted
		}
		// All items submitted
		return result.submittedCount

	case batchActionTooBig:
		// Batch too big - need to split further
		return submitWithRecursiveSplitting[T](m, ctx, items, marshaled, gasPrice, postSubmit, itemType)

	case batchActionSkip:
		// Single item too big - skip it
		m.logger.Error("skipping item that exceeds DA blob size limit", "itemType", itemType)
		return 0

	case batchActionFail:
		// Unrecoverable error - stop processing
		m.logger.Error("unrecoverable error during batch submission", "itemType", itemType)
		return 0
	}

	return 0
}

// BatchAction represents the action to take after processing a batch
type BatchAction int

const (
	batchActionSubmitted BatchAction = iota // Batch was successfully submitted
	batchActionTooBig                       // Batch is too big and needs to be handled by caller
	batchActionSkip                         // Batch should be skipped (single item too big)
	batchActionFail                         // Unrecoverable error
)

// BatchResult contains the result of processing a batch
type BatchResult[T any] struct {
	action         BatchAction
	submittedCount int
	splitBatches   []SubmissionBatch[T]
}

// processBatch processes a single batch and returns the result.
//
// Returns BatchResult with one of the following actions:
// - batchActionSubmitted: Batch was successfully submitted (partial or complete)
// - batchActionTooBig: Batch is too big and needs to be handled by caller
// - batchActionSkip: Single item is too big and cannot be split further
// - batchActionFail: Unrecoverable error occurred (context timeout, network failure, etc.)
func processBatch[T any](
	m *Manager,
	ctx context.Context,
	batch SubmissionBatch[T],
	gasPrice float64,
	postSubmit func([]T, *coreda.ResultSubmit, float64),
	itemType string,
) BatchResult[T] {
	batchCtx, batchCtxCancel := context.WithTimeout(ctx, submissionTimeout)
	defer batchCtxCancel()

	batchRes := types.SubmitWithHelpers(batchCtx, m.da, m.logger, batch.Marshaled, gasPrice, nil)

	if batchRes.Code == coreda.StatusSuccess {
		// Successfully submitted this batch
		submitted := batch.Items[:batchRes.SubmittedCount]
		postSubmit(submitted, &batchRes, gasPrice)
		m.logger.Info("successfully submitted batch to DA layer", "batchSize", len(batch.Items), "submittedCount", batchRes.SubmittedCount)

		return BatchResult[T]{
			action:         batchActionSubmitted,
			submittedCount: int(batchRes.SubmittedCount),
		}
	}

	if batchRes.Code == coreda.StatusTooBig && len(batch.Items) > 1 {
		// Batch is too big - let the caller handle splitting
		m.logger.Info("batch too big, returning to caller for splitting", "batchSize", len(batch.Items))
		return BatchResult[T]{action: batchActionTooBig}
	}

	if len(batch.Items) == 1 && batchRes.Code == coreda.StatusTooBig {
		m.logger.Error("single item exceeds DA blob size limit", "itemType", itemType)
		return BatchResult[T]{action: batchActionSkip}
	}

	// Other error - cannot continue with this batch
	return BatchResult[T]{action: batchActionFail}
}
