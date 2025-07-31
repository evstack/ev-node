package block

import (
	"context"
	"fmt"
	"time"

	coreda "github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/types"
	"google.golang.org/protobuf/proto"
)

// chunk represents a batch of items with their marshaled data for DA submission
type chunk[T any] struct {
	items     []T
	marshaled [][]byte
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

// submitToDA is a generic helper for submitting items to the DA layer with retry, backoff, and gas price logic.
// marshalFn marshals an item to []byte.
// postSubmit is called after a successful submission to update caches, pending lists, etc.
func submitToDA[T any](
	m *Manager,
	ctx context.Context,
	items []T,
	marshalFn func(T) ([]byte, error),
	postSubmit func([]T, *coreda.ResultSubmit, float64),
	itemType string,
) error {
	submittedAll := false
	var backoff time.Duration
	attempt := 0
	initialGasPrice := m.gasPrice
	gasPrice := initialGasPrice
	remaining := items
	numSubmitted := 0

	// Marshal all items once before the loop
	marshaled := make([][]byte, len(items))
	for i, item := range items {
		bz, err := marshalFn(item)
		if err != nil {
			return fmt.Errorf("failed to marshal item: %w", err)
		}
		marshaled[i] = bz
	}
	remLen := len(items)

	for !submittedAll && attempt < maxSubmitAttempts {
		select {
		case <-ctx.Done():
			m.logger.Info("context done, stopping submission loop")
			return nil
		case <-time.After(backoff):
		}

		// Use the current remaining items and marshaled bytes
		currMarshaled := marshaled
		remLen = len(remaining)

		submitctx, submitCtxCancel := context.WithTimeout(ctx, 60*time.Second)

		// Record DA submission retry attempt
		m.recordDAMetrics("submission", DAModeRetry)

		res := types.SubmitWithHelpers(submitctx, m.da, m.logger, currMarshaled, gasPrice, nil)
		submitCtxCancel()

		switch res.Code {
		case coreda.StatusSuccess:
			// Record successful DA submission
			m.recordDAMetrics("submission", DAModeSuccess)

			m.logger.Info(fmt.Sprintf("successfully submitted %s to DA layer with gasPrice %v and count %d", itemType, gasPrice, res.SubmittedCount))
			if res.SubmittedCount == uint64(remLen) {
				submittedAll = true
			}
			submitted := remaining[:res.SubmittedCount]
			notSubmitted := remaining[res.SubmittedCount:]
			notSubmittedMarshaled := currMarshaled[res.SubmittedCount:]
			numSubmitted += int(res.SubmittedCount)
			postSubmit(submitted, &res, gasPrice)
			remaining = notSubmitted
			marshaled = notSubmittedMarshaled
			backoff = 0
			if m.gasMultiplier > 0 && gasPrice != -1 {
				gasPrice = gasPrice / m.gasMultiplier
				gasPrice = max(gasPrice, initialGasPrice)
			}
			m.logger.Debug("resetting DA layer submission options", "backoff", backoff, "gasPrice", gasPrice)
		case coreda.StatusNotIncludedInBlock, coreda.StatusAlreadyInMempool:
			m.logger.Error("DA layer submission failed", "error", res.Message, "attempt", attempt)
			// Record failed DA submission (will retry)
			m.recordDAMetrics("submission", DAModeFail)
			backoff = m.config.DA.BlockTime.Duration * time.Duration(m.config.DA.MempoolTTL)
			if m.gasMultiplier > 0 && gasPrice != -1 {
				gasPrice = gasPrice * m.gasMultiplier
			}
			m.logger.Info("retrying DA layer submission with", "backoff", backoff, "gasPrice", gasPrice)
		case coreda.StatusContextCanceled:
			m.logger.Info("DA layer submission canceled due to context cancellation", "attempt", attempt)
			return nil
		case coreda.StatusTooBig:
			m.logger.Warn("DA layer submission failed due to blob size limit", "error", res.Message, "attempt", attempt, "batchSize", len(remaining))
			// Record failed DA submission (will retry)
			m.recordDAMetrics("submission", DAModeFail)

			// Handle batch splitting when blob is too big
			if len(remaining) > 1 {
				totalSubmitted := submitWithRecursiveSplitting(m, ctx, remaining, marshaled, gasPrice, postSubmit, itemType)

				// Update remaining items and continue main loop
				if totalSubmitted > 0 {
					// Remove submitted items from remaining
					remaining = remaining[totalSubmitted:]
					marshaled = marshaled[totalSubmitted:]
					numSubmitted += totalSubmitted

					if len(remaining) == 0 {
						submittedAll = true
					}
					backoff = 0 // Reset backoff on partial success
				} else {
					// No items were submitted - apply backoff
					backoff = m.exponentialBackoff(backoff)
				}
			} else {
				// If we have only 1 item and it's still too big, we can't split further
				m.logger.Error("single item exceeds DA blob size limit", "itemType", itemType, "attempt", attempt)
				backoff = m.exponentialBackoff(backoff)
			}
		default:
			m.logger.Error("DA layer submission failed", "error", res.Message, "attempt", attempt)
			// Record failed DA submission (will retry)
			m.recordDAMetrics("submission", DAModeFail)
			backoff = m.exponentialBackoff(backoff)
		}
		attempt++
	}
	if !submittedAll {
		// Record final failure after all retries are exhausted
		m.recordDAMetrics("submission", DAModeFail)
		// If not all items are submitted, the remaining items will be retried in the next submission loop.
		return fmt.Errorf("failed to submit all %s(s) to DA layer, submitted %d items (%d left) after %d attempts", itemType, numSubmitted, remLen, attempt)
	}
	return nil
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
	m.logger.Info("starting recursive batch splitting", "originalSize", len(items))

	// Split the original batch immediately and start with the two halves
	splitPoint := len(items) / 2
	firstHalf := items[:splitPoint]
	secondHalf := items[splitPoint:]
	firstHalfMarshaled := marshaled[:splitPoint]
	secondHalfMarshaled := marshaled[splitPoint:]

	m.logger.Info("initial split for recursive processing", "originalSize", len(items), "firstHalf", len(firstHalf), "secondHalf", len(secondHalf))

	// Start processing with both halves
	chunks := []chunk[T]{
		{firstHalf, firstHalfMarshaled},
		{secondHalf, secondHalfMarshaled},
	}

	var totalSubmitted int

	// Process chunks until all are either submitted or fail
	for len(chunks) > 0 {
		currentChunk := chunks[0]
		chunks = chunks[1:]

		result := processChunk(m, ctx, currentChunk, gasPrice, postSubmit, itemType)

		switch result.action {
		case chunkActionSubmitted:
			totalSubmitted += result.submittedCount

			// Handle partial submission within chunk
			if result.submittedCount < len(currentChunk.items) {
				remainingItems := currentChunk.items[result.submittedCount:]
				remainingMarshaled := currentChunk.marshaled[result.submittedCount:]
				chunks = append(chunks, chunk[T]{remainingItems, remainingMarshaled})
			}
		case chunkActionSplit:
			// Add split chunks to the front of the queue to preserve order
			chunks = append(result.splitChunks, chunks...)
		case chunkActionSkip:
			// Single item too big - skip it
			continue
		case chunkActionFail:
			// Unrecoverable error - stop processing
			break
		}
	}

	return totalSubmitted
}

// chunkAction represents the action to take after processing a chunk
type chunkAction int

const (
	chunkActionSubmitted chunkAction = iota // Chunk was successfully submitted
	chunkActionSplit                        // Chunk needs to be split
	chunkActionSkip                         // Chunk should be skipped (single item too big)
	chunkActionFail                         // Unrecoverable error
)

// chunkResult contains the result of processing a chunk
type chunkResult[T any] struct {
	action         chunkAction
	submittedCount int
	splitChunks    []chunk[T]
}

// processChunk processes a single chunk and returns the result.
//
// Returns chunkResult with one of the following actions:
// - chunkActionSubmitted: Chunk was successfully submitted (partial or complete)
// - chunkActionSplit: Chunk is too big and needs to be split into smaller chunks
// - chunkActionSkip: Single item is too big and cannot be split further
// - chunkActionFail: Unrecoverable error occurred (context timeout, network failure, etc.)
func processChunk[T any](
	m *Manager,
	ctx context.Context,
	ch chunk[T],
	gasPrice float64,
	postSubmit func([]T, *coreda.ResultSubmit, float64),
	itemType string,
) chunkResult[T] {
	chunkCtx, chunkCtxCancel := context.WithTimeout(ctx, 60*time.Second)
	defer chunkCtxCancel()

	chunkRes := types.SubmitWithHelpers(chunkCtx, m.da, m.logger, ch.marshaled, gasPrice, nil)

	if chunkRes.Code == coreda.StatusSuccess {
		// Successfully submitted this chunk
		submitted := ch.items[:chunkRes.SubmittedCount]
		postSubmit(submitted, &chunkRes, gasPrice)
		m.logger.Info("successfully submitted chunk to DA layer", "chunkSize", len(ch.items), "submittedCount", chunkRes.SubmittedCount)

		return chunkResult[T]{
			action:         chunkActionSubmitted,
			submittedCount: int(chunkRes.SubmittedCount),
		}
	}

	if chunkRes.Code == coreda.StatusTooBig && len(ch.items) > 1 {
		// Split this chunk in half and add both halves to the queue
		splitPoint := len(ch.items) / 2
		firstHalf := ch.items[:splitPoint]
		secondHalf := ch.items[splitPoint:]
		firstHalfMarshaled := ch.marshaled[:splitPoint]
		secondHalfMarshaled := ch.marshaled[splitPoint:]

		m.logger.Info("splitting chunk", "originalSize", len(ch.items), "firstHalf", len(firstHalf), "secondHalf", len(secondHalf))

		return chunkResult[T]{
			action: chunkActionSplit,
			splitChunks: []chunk[T]{
				{firstHalf, firstHalfMarshaled},
				{secondHalf, secondHalfMarshaled},
			},
		}
	}

	if len(ch.items) == 1 && chunkRes.Code == coreda.StatusTooBig {
		m.logger.Error("single item exceeds DA blob size limit", "itemType", itemType)
		return chunkResult[T]{action: chunkActionSkip}
	}

	// Other error - cannot continue with this chunk
	return chunkResult[T]{action: chunkActionFail}
}
