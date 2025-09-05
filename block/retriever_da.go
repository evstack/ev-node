package block

import (
	"bytes"
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
)

// daRetriever encapsulates DA retrieval with pending events management.
// Pending events are persisted via Manager.pendingEventsCache to avoid data loss on retries or restarts.
type daRetriever struct {
	manager *Manager
	mutex   sync.RWMutex // mutex for pendingEvents
}

// daHeightEvent represents a DA event
type daHeightEvent struct {
	Header *types.SignedHeader
	Data   *types.Data
	// DaHeight corresponds to the highest DA included height between the Header and Data.
	// It is used when setting the evolve last DA height.
	DaHeight uint64

	// HeaderDaIncludedHeight corresponds to the DA height at which the Header was included.
	// Saving such is not necessary for the data, as the da included height can be set immediately after fetching because there is low verification required.
	HeaderDaIncludedHeight uint64
}

// newDARetriever creates a new DA retriever
func newDARetriever(manager *Manager) *daRetriever {
	return &daRetriever{
		manager: manager,
	}
}

// DARetrieveLoop is responsible for interacting with DA layer.
func (m *Manager) DARetrieveLoop(ctx context.Context) {
	retriever := newDARetriever(m)
	retriever.run(ctx)
}

// run executes the main DA retrieval loop
func (dr *daRetriever) run(ctx context.Context) {
	// attempt to process any pending events loaded from disk before starting retrieval loop.
	dr.processPendingEvents(ctx)

	// blobsFoundCh is used to track when we successfully found a header so
	// that we can continue to try and find headers that are in the next DA height.
	// This enables syncing faster than the DA block time.
	blobsFoundCh := make(chan struct{}, 1)
	defer close(blobsFoundCh)
	for {
		select {
		case <-ctx.Done():
			return
		case <-dr.manager.retrieveCh:
		case <-blobsFoundCh:
		}
		daHeight := dr.manager.daHeight.Load()
		err := dr.processNextDAHeaderAndData(ctx)
		if err != nil && ctx.Err() == nil {
			// if the requested da height is not yet available, wait silently, otherwise log the error and wait
			if !dr.manager.areAllErrorsHeightFromFuture(err) {
				dr.manager.logger.Error().Uint64("daHeight", daHeight).Str("errors", err.Error()).Msg("failed to retrieve data from DALC")
			}
			continue
		}
		// Signal the blobsFoundCh to try and retrieve the next set of blobs
		select {
		case blobsFoundCh <- struct{}{}:
		default:
		}
		dr.manager.daHeight.Store(daHeight + 1)

		// Try to process any pending DA events that might now be ready
		dr.processPendingEvents(ctx)
	}
}

// processNextDAHeaderAndData is responsible for retrieving a header and data from the DA layer.
// It returns an error if the context is done or if the DA layer returns an error.
func (dr *daRetriever) processNextDAHeaderAndData(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	daHeight := dr.manager.daHeight.Load()

	var err error
	dr.manager.logger.Debug().Uint64("daHeight", daHeight).Msg("trying to retrieve data from DA")
	for r := 0; r < dAFetcherRetries; r++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		blobsResp, fetchErr := dr.manager.fetchBlobs(ctx, daHeight)
		if fetchErr == nil {
			if blobsResp.Code == coreda.StatusNotFound {
				dr.manager.logger.Debug().Uint64("daHeight", daHeight).Str("reason", blobsResp.Message).Msg("no blob data found")
				return nil
			}
			dr.manager.logger.Debug().Int("n", len(blobsResp.Data)).Uint64("daHeight", daHeight).Msg("retrieved potential blob data")

			dr.processBlobs(ctx, blobsResp.Data, daHeight)
			return nil
		} else if strings.Contains(fetchErr.Error(), coreda.ErrHeightFromFuture.Error()) {
			dr.manager.logger.Debug().Uint64("daHeight", daHeight).Str("reason", fetchErr.Error()).Msg("height from future")
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
func (dr *daRetriever) processBlobs(ctx context.Context, blobs [][]byte, daHeight uint64) {
	// collect all headers and data
	type headerWithDaHeight struct {
		*types.SignedHeader
		daHeight uint64
	}
	headers := make(map[uint64]headerWithDaHeight)
	dataMap := make(map[uint64]*types.Data)

	for _, bz := range blobs {
		if len(bz) == 0 {
			dr.manager.logger.Debug().Uint64("daHeight", daHeight).Msg("ignoring nil or empty blob")
			continue
		}

		if header := dr.manager.tryDecodeHeader(bz, daHeight); header != nil {
			headers[header.Height()] = headerWithDaHeight{header, daHeight}
			continue
		}

		if data := dr.manager.tryDecodeData(bz, daHeight); data != nil {
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
				data = dr.manager.createEmptyDataForHeader(ctx, header.SignedHeader)
			} else {
				// Check if header's DataHash matches the hash of empty data
				emptyData := dr.manager.createEmptyDataForHeader(ctx, header.SignedHeader)
				emptyDataHash := emptyData.Hash()
				if bytes.Equal(header.DataHash, emptyDataHash) {
					data = emptyData
				} else {
					// Header expects data but no data found - skip for now
					dr.manager.logger.Debug().Uint64("height", height).Uint64("daHeight", daHeight).Msg("header found but no matching data yet")
					continue
				}
			}
		}

		// both available, proceed with complete event
		dr.sendHeightEventIfValid(ctx, daHeightEvent{
			Header:                 header.SignedHeader,
			HeaderDaIncludedHeight: header.daHeight,
			Data:                   data,
			DaHeight:               daHeight,
		})
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
	if err := m.assertUsingExpectedSingleSequencer(header.ProposerAddress); err != nil {
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
	if err := m.assertValidSignedData(&signedData); err != nil {
		m.logger.Debug().Uint64("daHeight", daHeight).Err(err).Msg("invalid data signature")
		return nil
	}

	dataHashStr := signedData.Data.DACommitment().String()
	m.dataCache.SetDAIncluded(dataHashStr, daHeight)
	m.sendNonBlockingSignalToDAIncluderCh()
	m.logger.Info().Str("dataHash", dataHashStr).Uint64("daHeight", daHeight).Uint64("height", signedData.Height()).Msg("signed data marked as DA included")

	return &signedData.Data
}

// assertValidSignedData validates the data signature and returns an error if it's invalid.
func (m *Manager) assertValidSignedData(signedData *types.SignedData) error {
	if signedData == nil || signedData.Txs == nil {
		return errors.New("empty signed data")
	}

	if err := m.assertUsingExpectedSingleSequencer(signedData.Signer.Address); err != nil {
		return err
	}

	dataBytes, err := signedData.Data.MarshalBinary()
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	valid, err := signedData.Signer.PubKey.Verify(dataBytes, signedData.Signature)
	if err != nil {
		return fmt.Errorf("failed to verify signature: %w", err)
	}

	if !valid {
		return fmt.Errorf("invalid signature")
	}

	return nil
}

// isAtHeight checks if a height is available.
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
func (dr *daRetriever) sendHeightEventIfValid(ctx context.Context, heightEvent daHeightEvent) {
	headerHash := heightEvent.Header.Hash().String()
	dataHashStr := heightEvent.Data.DACommitment().String()

	// Check if already seen before doing expensive validation
	if dr.manager.headerCache.IsSeen(headerHash) {
		dr.manager.logger.Debug().Str("headerHash", headerHash).Msg("header already seen, skipping")
		return
	}

	if !bytes.Equal(heightEvent.Header.DataHash, dataHashForEmptyTxs) && dr.manager.dataCache.IsSeen(dataHashStr) {
		dr.manager.logger.Debug().Str("dataHash", dataHashStr).Msg("data already seen, skipping")
		return
	}

	// Check if we can validate this height immediately
	if err := dr.manager.isAtHeight(ctx, heightEvent.Header.Height()-1); err != nil {
		// Queue this event for later processing when the prerequisite height is available
		dr.queuePendingEvent(heightEvent)
		return
	}

	// Process immediately since prerequisite height is available
	dr.processEvent(ctx, heightEvent)
}

// queuePendingEvent queues a DA event that cannot be processed immediately.
// The event is persisted via pendingEventsCache to survive restarts.
func (dr *daRetriever) queuePendingEvent(heightEvent daHeightEvent) {
	dr.mutex.Lock()
	defer dr.mutex.Unlock()

	if dr.manager.pendingEventsCache == nil {
		return
	}

	height := heightEvent.Header.Height()
	dr.manager.pendingEventsCache.SetItem(height, &heightEvent)

	dr.manager.logger.Debug().
		Uint64("height", height).
		Uint64("daHeight", heightEvent.DaHeight).
		Msg("queued DA event for later processing")
}

// processEvent processes a DA event that is ready for validation
func (dr *daRetriever) processEvent(ctx context.Context, heightEvent daHeightEvent) {
	// Validate header with its data - some execution environment may require previous height to be stored
	if err := heightEvent.Header.ValidateBasicWithData(heightEvent.Data); err != nil {
		dr.manager.logger.Debug().Uint64("height", heightEvent.Header.Height()).Err(err).Msg("header validation with data failed")
		return
	}

	// Record successful DA retrieval
	dr.manager.recordDAMetrics("retrieval", DAModeSuccess)

	// Mark as DA included since validation passed
	headerHash := heightEvent.Header.Hash().String()
	dr.manager.headerCache.SetDAIncluded(headerHash, heightEvent.HeaderDaIncludedHeight)
	dr.manager.sendNonBlockingSignalToDAIncluderCh()
	dr.manager.logger.Info().Uint64("headerHeight", heightEvent.Header.Height()).Str("headerHash", headerHash).Msg("header marked as DA included")

	select {
	case <-ctx.Done():
		return
	case dr.manager.heightInCh <- heightEvent:
		dr.manager.logger.Debug().
			Uint64("height", heightEvent.Header.Height()).
			Uint64("daHeight", heightEvent.DaHeight).
			Str("source", "da data sync").
			Msg("sent complete height event with header and data")
	default:
		// Channel full: keep event in pending cache for retry
		dr.queuePendingEvent(heightEvent)
		dr.manager.logger.Warn().
			Uint64("height", heightEvent.Header.Height()).
			Uint64("daHeight", heightEvent.DaHeight).
			Str("source", "da data sync").
			Msg("heightInCh backlog full, re-queued event to pending cache")
	}

	// Try to process any pending events that might now be ready
	dr.processPendingEvents(ctx)
}

// processPendingEvents tries to process queued DA events that might now be ready
func (dr *daRetriever) processPendingEvents(ctx context.Context) {
	if dr.manager.pendingEventsCache == nil {
		return
	}

	currentHeight, err := dr.manager.GetStoreHeight(ctx)
	if err != nil {
		dr.manager.logger.Debug().Err(err).Msg("failed to get store height for pending DA events")
		return
	}

	dr.mutex.Lock()
	defer dr.mutex.Unlock()

	toDelete := make([]uint64, 0)
	dr.manager.pendingEventsCache.RangeByHeight(func(height uint64, event *daHeightEvent) bool {
		if height <= currentHeight+1 {
			dr.manager.logger.Debug().
				Uint64("height", height).
				Uint64("daHeight", event.DaHeight).
				Msg("processing previously queued DA event")
			go dr.processEvent(ctx, *event)
			toDelete = append(toDelete, height)
		}
		return true
	})

	for _, h := range toDelete {
		dr.manager.pendingEventsCache.DeleteItem(h)
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

	// Try to retrieve from both header and data namespaces
	headerNamespace := []byte(m.config.DA.GetHeaderNamespace())
	dataNamespace := []byte(m.config.DA.GetDataNamespace())

	// Retrieve headers
	headerRes := types.RetrieveWithHelpers(ctx, m.da, m.logger, daHeight, headerNamespace)

	// Both namespace are the same, so we are using the legacy flow.
	if bytes.Equal(headerNamespace, dataNamespace) {
		err := m.validateBlobResponse(headerRes, daHeight)
		return headerRes, err
	}

	// Retrieve data
	dataRes := types.RetrieveWithHelpers(ctx, m.da, m.logger, daHeight, dataNamespace)

	// Combine results or handle errors appropriately
	errHeader := m.validateBlobResponse(headerRes, daHeight)
	errData := m.validateBlobResponse(dataRes, daHeight)

	if errors.Is(errHeader, coreda.ErrHeightFromFuture) || errors.Is(errData, coreda.ErrHeightFromFuture) {
		return coreda.ResultRetrieve{
			BaseResult: coreda.BaseResult{Code: coreda.StatusHeightFromFuture},
		}, fmt.Errorf("%w: height from future", coreda.ErrHeightFromFuture)
	}

	if errors.Is(errHeader, ErrRetrievalFailed) && errors.Is(errData, ErrRetrievalFailed) {
		return headerRes, errHeader
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

	return combinedResult, err
}

// ErrRetrievalFailed is returned when a namespace retrieval fails.
var ErrRetrievalFailed = errors.New("failed to retrieve namespaces")

func (m *Manager) validateBlobResponse(res coreda.ResultRetrieve, daHeight uint64) error {
	if res.Code == coreda.StatusError {
		m.recordDAMetrics("retrieval", DAModeFail)
		return fmt.Errorf("%w: %s", ErrRetrievalFailed, res.Message)
	}

	if res.Code == coreda.StatusHeightFromFuture {
		return fmt.Errorf("%w: height from future", coreda.ErrHeightFromFuture)
	}

	if res.Code == coreda.StatusSuccess {
		m.logger.Debug().Uint64("daHeight", daHeight).Msg("found data in namespace")
	}

	return nil
}
