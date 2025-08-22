package block

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"

	coreda "github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/types"
	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

const (
	dAefetcherTimeout = 30 * time.Second
	dAFetcherRetries  = 10
)

// DARetrieveLoop is responsible for interacting with DA layer.
func (m *Manager) DARetrieveLoop(ctx context.Context) {
	// blobsFoundCh is used to track when we successfully found a header so
	// that we can continue to try and find headers that are in the next DA height.
	// This enables syncing faster than the DA block time.
	blobsFoundCh := make(chan struct{}, 1)
	defer close(blobsFoundCh)
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.retrieveCh:
		case <-blobsFoundCh:
		}
		daHeight := m.daHeight.Load()
		err := m.processNextDAHeaderAndData(ctx)
		if err != nil && ctx.Err() == nil {
			// if the requested da height is not yet available, wait silently, otherwise log the error and wait
			if !m.areAllErrorsHeightFromFuture(err) {
				m.logger.Error().Uint64("daHeight", daHeight).Str("errors", err.Error()).Msg("failed to retrieve data from DALC")
			}
			continue
		}
		// Signal the blobsFoundCh to try and retrieve the next set of blobs
		select {
		case blobsFoundCh <- struct{}{}:
		default:
		}
		m.daHeight.Store(daHeight + 1)

		// Try to process any pending DA events that might now be ready
		m.processPendingDAEvents(ctx)
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

			m.processBlobs(ctx, blobsResp.Data, daHeight)
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

// processBlobs processes all blobs to find headers and their corresponding data, then sends complete height events
func (m *Manager) processBlobs(ctx context.Context, blobs [][]byte, daHeight uint64) {
	// collect all headers and data
	headers := make(map[uint64]*types.SignedHeader)
	dataMap := make(map[uint64]*types.Data)

	for _, bz := range blobs {
		if len(bz) == 0 {
			m.logger.Debug().Uint64("daHeight", daHeight).Msg("ignoring nil or empty blob")
			continue
		}

		if header := m.tryDecodeHeader(bz, daHeight); header != nil {
			headers[header.Height()] = header
			continue
		}

		if data := m.tryDecodeData(bz, daHeight); data != nil {
			dataMap[data.Height()] = data
		}
	}

	// match headers with data and send complete height events
	for height, header := range headers {
		data := dataMap[height]

		// If no data found, check if header expects empty data or create empty data
		if data == nil {
			if bytes.Equal(header.DataHash, dataHashForEmptyTxs) || len(header.DataHash) == 0 {
				// Header expects empty data, create it
				data = m.createEmptyDataForHeader(ctx, header)
			} else {
				// Check if header's DataHash matches the hash of empty data
				emptyData := m.createEmptyDataForHeader(ctx, header)
				emptyDataHash := emptyData.Hash()
				if bytes.Equal(header.DataHash, emptyDataHash) {
					data = emptyData
				} else {
					// Header expects data but no data found - skip for now
					m.logger.Debug().Uint64("height", height).Uint64("daHeight", daHeight).Msg("header found but no matching data yet")
					continue
				}
			}
		}

		m.sendHeightEventIfValid(ctx, header, data, daHeight)
	}
}

// tryDecodeHeader attempts to decode a blob as a header, returns nil if not a valid header
func (m *Manager) tryDecodeHeader(bz []byte, daHeight uint64) *types.SignedHeader {
	header := new(types.SignedHeader)
	var headerPb pb.SignedHeader

	if err := proto.Unmarshal(bz, &headerPb); err != nil {
		m.logger.Debug().Err(err).Msg("failed to unmarshal header")
		return nil
	}

	if err := header.FromProto(&headerPb); err != nil {
		// treat as handled, but not valid
		m.logger.Debug().Err(err).Msg("failed to decode unmarshalled header")
		return nil
	}

	// early validation to reject junk headers
	if ok, err := m.isUsingExpectedSingleSequencer(header.ProposerAddress); !ok {
		m.logger.Debug().
			Uint64("headerHeight", header.Height()).
			Str("headerHash", header.Hash().String()).
			Msg("invalid header: " + err.Error())
		return nil
	}

	// validate basic header structure only (without data)
	if err := header.Header.ValidateBasic(); err != nil {
		m.logger.Debug().Uint64("daHeight", daHeight).Err(err).Msg("blob does not look like a valid header")
		return nil
	}

	if err := header.Signature.ValidateBasic(); err != nil {
		m.logger.Debug().Uint64("daHeight", daHeight).Err(err).Msg("header signature validation failed")
		return nil
	}

	// Header is valid for basic structure, will be fully validated with data later

	return header
}

// tryDecodeData attempts to decode a blob as data, returns nil if not valid data
func (m *Manager) tryDecodeData(bz []byte, daHeight uint64) *types.Data {
	var signedData types.SignedData
	err := signedData.UnmarshalBinary(bz)
	if err != nil {
		m.logger.Debug().Err(err).Msg("failed to unmarshal signed data")
		return nil
	}

	// Allow empty signed data with valid signatures, but ignore completely empty blobs
	if len(signedData.Txs) == 0 && len(signedData.Signature) == 0 {
		m.logger.Debug().Uint64("daHeight", daHeight).Msg("ignoring empty signed data with no signature")
		return nil
	}

	// Early validation to reject junk data
	if !m.isValidSignedData(&signedData) {
		m.logger.Debug().Uint64("daHeight", daHeight).Msg("invalid data signature")
		return nil
	}

	dataHashStr := signedData.Data.DACommitment().String()
	m.dataCache.SetDAIncluded(dataHashStr, daHeight)
	m.sendNonBlockingSignalToDAIncluderCh()
	m.logger.Info().Str("dataHash", dataHashStr).Uint64("daHeight", daHeight).Uint64("height", signedData.Height()).Msg("signed data marked as DA included")

	return &signedData.Data
}

// isAtHeight checks if a height is available without blocking
func (m *Manager) isAtHeight(ctx context.Context, height uint64) error {
	currentHeight, err := m.GetStoreHeight(ctx)
	if err != nil {
		return err
	}
	if currentHeight >= height {
		return nil
	}
	return fmt.Errorf("height %d not yet available (current: %d)", height, currentHeight)
}

// sendHeightEventIfValid sends a height event if both header and data are valid and not seen before
func (m *Manager) sendHeightEventIfValid(ctx context.Context, header *types.SignedHeader, data *types.Data, daHeight uint64) {
	headerHash := header.Hash().String()
	dataHashStr := data.DACommitment().String()

	// Check if already seen before doing expensive validation
	if m.headerCache.IsSeen(headerHash) {
		m.logger.Debug().Str("headerHash", headerHash).Msg("header already seen, skipping")
		return
	}

	if !bytes.Equal(header.DataHash, dataHashForEmptyTxs) && m.dataCache.IsSeen(dataHashStr) {
		m.logger.Debug().Str("dataHash", dataHashStr).Msg("data already seen, skipping")
		return
	}

	// Check if we can validate this height immediately (non-blocking check)
	if err := m.isAtHeight(ctx, header.Height()-1); err != nil {
		// Queue this event for later processing when the prerequisite height is available
		m.queuePendingDAEvent(ctx, header, data, daHeight)
		return
	}

	// Process immediately since prerequisite height is available
	m.processDAEvent(ctx, header, data, daHeight)
}

// queuePendingDAEvent queues a DA event that cannot be processed immediately
func (m *Manager) queuePendingDAEvent(ctx context.Context, header *types.SignedHeader, data *types.Data, daHeight uint64) {
	heightEvent := NewHeightEvent{
		Header:   header,
		Data:     data,
		DAHeight: daHeight,
	}

	m.pendingMutex.Lock()
	defer m.pendingMutex.Unlock()

	height := header.Height()
	m.pendingSyncEvents[height] = append(m.pendingSyncEvents[height], heightEvent)

	m.logger.Debug().
		Uint64("height", height).
		Uint64("daHeight", daHeight).
		Msg("queued DA event for later processing")
}

// processDAEvent processes a DA event that is ready for validation
func (m *Manager) processDAEvent(ctx context.Context, header *types.SignedHeader, data *types.Data, daHeight uint64) {
	headerHash := header.Hash().String()

	// Validate header with its data - this requires previous height to be stored
	if err := header.ValidateBasicWithData(data); err != nil {
		m.logger.Debug().Uint64("height", header.Height()).Err(err).Msg("header validation with data failed")
		return
	}

	// Mark as DA included since validation passed
	m.headerCache.SetDAIncluded(headerHash, daHeight)
	m.sendNonBlockingSignalToDAIncluderCh()
	m.logger.Info().Uint64("headerHeight", header.Height()).Str("headerHash", headerHash).Msg("header marked as DA included")

	// Send complete height event with both header and data
	heightEvent := NewHeightEvent{
		Header:   header,
		Data:     data,
		DAHeight: daHeight,
	}

	select {
	case <-ctx.Done():
		return
	case m.heightInCh <- heightEvent:
		m.logger.Debug().
			Uint64("height", header.Height()).
			Uint64("daHeight", daHeight).
			Msg("sent complete height event with header and data")
	default:
		m.logger.Warn().
			Uint64("height", header.Height()).
			Uint64("daHeight", daHeight).
			Msg("heightInCh backlog full, dropping complete event")
	}

	// Try to process any pending events that might now be ready
	m.processPendingDAEvents(ctx)
}

// processPendingDAEvents tries to process queued DA events that might now be ready
func (m *Manager) processPendingDAEvents(ctx context.Context) {
	m.pendingMutex.Lock()
	defer m.pendingMutex.Unlock()

	if m.pendingSyncEvents == nil {
		return
	}

	currentHeight, err := m.GetStoreHeight(ctx)
	if err != nil {
		m.logger.Debug().Err(err).Msg("failed to get store height for pending DA events")
		return
	}

	// Process events that are now ready (height - 1 <= currentHeight)
	for height, events := range m.pendingSyncEvents {
		if height-1 <= currentHeight {
			for _, event := range events {
				m.logger.Debug().
					Uint64("height", height).
					Uint64("daHeight", event.DAHeight).
					Msg("processing previously queued DA event")
				go m.processDAEvent(ctx, event.Header, event.Data, event.DAHeight)
			}
			delete(m.pendingSyncEvents, height)
		}
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
