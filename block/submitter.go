package block

import (
	"context"
	"fmt"
	"time"

	coreda "github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/types"
	"google.golang.org/protobuf/proto"
)

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
	namespace []byte,
) error {
	submittedAll := false
	var backoff time.Duration
	attempt := 0

	// Get gas price from DA layer
	initialGasPrice, err := m.da.GasPrice(ctx)
	if err != nil {
		m.logger.Warn().Err(err).Msg("failed to get gas price from DA layer, using default")
		initialGasPrice = 0.0 // fallback default
	}
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
			m.logger.Info().Msg("context done, stopping submission loop")
			return nil
		case <-time.After(backoff):
		}

		// Use the current remaining items and marshaled bytes
		currMarshaled := marshaled
		remLen = len(remaining)

		submitctx, submitCtxCancel := context.WithTimeout(ctx, 60*time.Second)

		// Record DA submission retry attempt
		m.recordDAMetrics("submission", DAModeRetry)

		res := types.SubmitWithHelpers(submitctx, m.da, m.logger, currMarshaled, gasPrice, namespace, nil)
		submitCtxCancel()

		switch res.Code {
		case coreda.StatusSuccess:
			// Record successful DA submission
			m.recordDAMetrics("submission", DAModeSuccess)

			m.logger.Info().Str("itemType", itemType).Float64("gasPrice", gasPrice).Uint64("count", res.SubmittedCount).Msg("successfully submitted to DA layer")
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

			// Get gas multiplier from DA layer and adjust price
			gasMultiplier, multErr := m.da.GasMultiplier(ctx)
			if multErr != nil {
				m.logger.Warn().Err(multErr).Msg("failed to get gas multiplier from DA layer, using default")
				gasMultiplier = 1.0 // fallback default
			}
			if gasMultiplier > 0 && gasPrice != -1 {
				gasPrice = gasPrice / gasMultiplier
				gasPrice = max(gasPrice, initialGasPrice)
			}
			m.logger.Debug().Dur("backoff", backoff).Float64("gasPrice", gasPrice).Msg("resetting DA layer submission options")
		case coreda.StatusNotIncludedInBlock, coreda.StatusAlreadyInMempool:
			m.logger.Error().Str("error", res.Message).Int("attempt", attempt).Msg("DA layer submission failed")
			// Record failed DA submission (will retry)
			m.recordDAMetrics("submission", DAModeFail)
			backoff = m.config.DA.BlockTime.Duration * time.Duration(m.config.DA.MempoolTTL)

			// Get gas multiplier from DA layer and increase price for retry
			gasMultiplier, multErr := m.da.GasMultiplier(ctx)
			if multErr != nil {
				m.logger.Warn().Err(multErr).Msg("failed to get gas multiplier from DA layer, using default")
				gasMultiplier = 1.0 // fallback default
			}
			if gasMultiplier > 0 && gasPrice != -1 {
				gasPrice = gasPrice * gasMultiplier
			}
			m.logger.Info().Dur("backoff", backoff).Float64("gasPrice", gasPrice).Msg("retrying DA layer submission")
		case coreda.StatusContextCanceled:
			m.logger.Info().Int("attempt", attempt).Msg("DA layer submission canceled due to context cancellation")
			return nil
		case coreda.StatusTooBig:
			fallthrough
		default:
			m.logger.Error().Str("error", res.Message).Int("attempt", attempt).Msg("DA layer submission failed")
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
		[]byte(m.config.DA.GetHeaderNamespace()),
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
		[]byte(m.config.DA.GetDataNamespace()),
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
