package syncing

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	coreda "github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/types"
	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

// defaultDATimeout is the default timeout for DA retrieval operations
const defaultDATimeout = 10 * time.Second

// pendingForcedInclusionTx represents a forced inclusion transaction that couldn't fit in the current epoch
// and needs to be retried in future epochs.
type pendingForcedInclusionTx struct {
	Data           []byte // The transaction data
	OriginalHeight uint64 // Original DA height where this transaction was found
}

// DARetriever defines the interface for retrieving events from the DA layer
type DARetriever interface {
	RetrieveFromDA(ctx context.Context, daHeight uint64) ([]common.DAHeightEvent, error)
	RetrieveForcedIncludedTxsFromDA(ctx context.Context, daHeight uint64) (*common.ForcedIncludedEvent, error)
}

// daRetriever handles DA retrieval operations for syncing
type daRetriever struct {
	da      coreda.DA
	cache   cache.Manager
	genesis genesis.Genesis
	logger  zerolog.Logger

	// calculate namespaces bytes once and reuse them
	namespaceBz                []byte
	namespaceDataBz            []byte
	namespaceForcedInclusionBz []byte

	hasForcedInclusionNs bool
	daEpochSize          uint64

	// transient cache, only full event need to be passed to the syncer
	// on restart, will be refetch as da height is updated by syncer
	pendingHeaders map[uint64]*types.SignedHeader
	pendingData    map[uint64]*types.Data

	// Forced inclusion transactions that couldn't fit in the current epoch
	// and need to be retried in future epochs.
	pendingForcedInclusionTxs []pendingForcedInclusionTx
}

// NewDARetriever creates a new DA retriever
func NewDARetriever(
	da coreda.DA,
	cache cache.Manager,
	config config.Config,
	genesis genesis.Genesis,
	logger zerolog.Logger,
) *daRetriever {
	forcedInclusionNs := config.DA.GetForcedInclusionNamespace()
	hasForcedInclusionNs := forcedInclusionNs != ""

	var namespaceForcedInclusionBz []byte
	if hasForcedInclusionNs {
		namespaceForcedInclusionBz = coreda.NamespaceFromString(forcedInclusionNs).Bytes()
	}

	return &daRetriever{
		da:                         da,
		cache:                      cache,
		genesis:                    genesis,
		logger:                     logger.With().Str("component", "da_retriever").Logger(),
		namespaceBz:                coreda.NamespaceFromString(config.DA.GetNamespace()).Bytes(),
		namespaceDataBz:            coreda.NamespaceFromString(config.DA.GetDataNamespace()).Bytes(),
		namespaceForcedInclusionBz: namespaceForcedInclusionBz,
		hasForcedInclusionNs:       hasForcedInclusionNs,
		daEpochSize:                genesis.DAEpochForcedInclusion,
		pendingHeaders:             make(map[uint64]*types.SignedHeader),
		pendingData:                make(map[uint64]*types.Data),
		pendingForcedInclusionTxs:  make([]pendingForcedInclusionTx, 0),
	}
}

// RetrieveFromDA retrieves blocks from the specified DA height and returns height events
func (r *daRetriever) RetrieveFromDA(ctx context.Context, daHeight uint64) ([]common.DAHeightEvent, error) {
	r.logger.Debug().Uint64("da_height", daHeight).Msg("retrieving from DA")
	blobsResp, err := r.fetchBlobs(ctx, daHeight)
	if err != nil {
		return nil, err
	}

	// Check for context cancellation upfront
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.logger.Debug().Int("blobs", len(blobsResp.Data)).Uint64("da_height", daHeight).Msg("retrieved blob data")
	return r.processBlobs(ctx, blobsResp.Data, daHeight), nil
}

var (
	// ErrForceInclusionNotConfigured is returned when the forced inclusion namespace is not configured.
	ErrForceInclusionNotConfigured = errors.New("forced inclusion namespace not configured")

	// ErrForcedInclusionDataTooLarge is returned when forced inclusion data exceeds the maximum blob size.
	ErrForcedInclusionDataTooLarge = errors.New("forced inclusion data exceeds maximum blob size limit")
)

// RetrieveForcedIncludedTxsFromDA retrieves forced inclusion transactions from the DA layer.
//
// Behavior:
//   - At epoch boundaries (when daHeight == epochStart): fetches new forced-inclusion transactions
//     from the DA layer for the entire epoch range, processes them, and returns all that fit within
//     the max blob size limit. Transactions that don't fit are stored in the pending queue for retry.
//   - Outside epoch boundaries (when daHeight != epochStart): returns any pending transactions from
//     the queue that were deferred from previous epochs.
//   - Pending transactions are kept in-memory only and will be lost on node restart.
//
// Returns:
//   - ForcedIncludedEvent with transactions that should be included in the next block (may be empty)
//   - Error if forced inclusion is not configured or DA layer is unavailable
func (r *daRetriever) RetrieveForcedIncludedTxsFromDA(ctx context.Context, daHeight uint64) (*common.ForcedIncludedEvent, error) {
	if !r.hasForcedInclusionNs {
		return nil, ErrForceInclusionNotConfigured
	}

	// Calculate deterministic epoch boundaries
	epochStart, epochEnd := types.CalculateEpochBoundaries(daHeight, r.genesis.DAStartHeight, r.daEpochSize)

	// If we're not at epoch start, return pending transactions only (if any)
	if daHeight != epochStart {
		r.logger.Debug().
			Uint64("da_height", daHeight).
			Uint64("epoch_start", epochStart).
			Int("pending_count", len(r.pendingForcedInclusionTxs)).
			Msg("not at epoch start - returning pending transactions only")

		event := &common.ForcedIncludedEvent{
			StartDaHeight: daHeight,
			EndDaHeight:   daHeight,
			Txs:           [][]byte{},
		}

		// Return pending txs if any exist
		if len(r.pendingForcedInclusionTxs) > 0 {
			pendingTxs, indicesToRemove, _ := r.processPendingForcedInclusionTxs()
			event.Txs = pendingTxs

			// Remove successfully included pending transactions
			if len(indicesToRemove) > 0 {
				r.removePendingForcedInclusionTxs(indicesToRemove)
				r.logger.Debug().
					Int("included_count", len(indicesToRemove)).
					Int("remaining_count", len(r.pendingForcedInclusionTxs)).
					Msg("included pending forced inclusion transactions")
			}
		}

		return event, nil
	}

	// We're at epoch start - fetch new transactions from DA

	currentEpochNumber := types.CalculateEpochNumber(daHeight, r.genesis.DAStartHeight, r.daEpochSize)

	event := &common.ForcedIncludedEvent{
		StartDaHeight: epochStart,
	}

	r.logger.Debug().
		Uint64("da_height", daHeight).
		Uint64("epoch_start", epochStart).
		Uint64("epoch_end", epochEnd).
		Uint64("epoch_num", currentEpochNumber).
		Msg("retrieving forced included transactions from DA")

	// Check if both epoch start and end are available before fetching
	// This ensures we can retrieve the complete epoch in one go
	epochStartResult := types.RetrieveWithHelpers(ctx, r.da, r.logger, epochStart, r.namespaceForcedInclusionBz, defaultDATimeout)
	if epochStartResult.Code == coreda.StatusHeightFromFuture {
		r.logger.Debug().
			Uint64("epoch_start", epochStart).
			Msg("epoch start height not yet available on DA - backoff required")
		return nil, fmt.Errorf("%w: epoch start height %d not yet available", coreda.ErrHeightFromFuture, epochStart)
	}

	epochEndResult := epochStartResult
	if epochStart != epochEnd {
		epochEndResult = types.RetrieveWithHelpers(ctx, r.da, r.logger, epochEnd, r.namespaceForcedInclusionBz, defaultDATimeout)
		if epochEndResult.Code == coreda.StatusHeightFromFuture {
			r.logger.Debug().
				Uint64("epoch_end", epochEnd).
				Msg("epoch end height not yet available on DA - backoff required")
			return nil, fmt.Errorf("%w: epoch end height %d not yet available", coreda.ErrHeightFromFuture, epochEnd)
		}
	}

	lastProcessedHeight := epochStart
	newPendingTxs := []pendingForcedInclusionTx{}

	// Prepend pending transactions from previous epochs at the start of this epoch
	pendingTxs, indicesToRemove, currentSize := r.processPendingForcedInclusionTxs()
	event.Txs = pendingTxs

	// Remove successfully included pending transactions
	if len(indicesToRemove) > 0 {
		r.removePendingForcedInclusionTxs(indicesToRemove)
		r.logger.Debug().
			Int("included_count", len(indicesToRemove)).
			Int("remaining_count", len(r.pendingForcedInclusionTxs)).
			Msg("included pending forced inclusion transactions")
	}

	// Process epoch start
	if err := r.processForcedInclusionBlobs(event, &currentSize, &lastProcessedHeight, &newPendingTxs, epochStartResult, epochStart); err != nil {
		return nil, err
	}

	// Process heights between start and end (exclusive)
	for epochHeight := epochStart + 1; epochHeight < epochEnd; epochHeight++ {
		result := types.RetrieveWithHelpers(ctx, r.da, r.logger, epochHeight, r.namespaceForcedInclusionBz, defaultDATimeout)

		// If any intermediate height is from future, break early
		if result.Code == coreda.StatusHeightFromFuture {
			r.logger.Debug().
				Uint64("epoch_height", epochHeight).
				Uint64("last_processed", lastProcessedHeight).
				Msg("reached future DA height within epoch - stopping")
			break
		}

		if err := r.processForcedInclusionBlobs(event, &currentSize, &lastProcessedHeight, &newPendingTxs, result, epochHeight); err != nil {
			return nil, err
		}
	}

	// Process epoch end (only if different from start)
	if epochEnd != epochStart {
		if err := r.processForcedInclusionBlobs(event, &currentSize, &lastProcessedHeight, &newPendingTxs, epochEndResult, epochEnd); err != nil {
			return nil, err
		}
	}

	// Store any new pending transactions that couldn't fit in this epoch
	if len(newPendingTxs) > 0 {
		r.pendingForcedInclusionTxs = append(r.pendingForcedInclusionTxs, newPendingTxs...)
		r.logger.Info().
			Int("new_pending_count", len(newPendingTxs)).
			Int("total_pending_count", len(r.pendingForcedInclusionTxs)).
			Msg("stored pending forced inclusion transactions for next epoch")
	}

	// Set the DA height range based on what we actually processed
	event.StartDaHeight = epochStart
	event.EndDaHeight = lastProcessedHeight

	return event, nil
}

// processForcedInclusionBlobs processes forced inclusion blobs from a single DA height.
// It accumulates transactions that fit within maxBlobSize and stores excess in newPendingTxs.
func (r *daRetriever) processForcedInclusionBlobs(
	event *common.ForcedIncludedEvent,
	currentSize *int,
	lastProcessedHeight *uint64,
	newPendingTxs *[]pendingForcedInclusionTx,
	result coreda.ResultRetrieve,
	daHeight uint64,
) error {
	if result.Code != coreda.StatusSuccess {
		return nil
	}

	if err := r.validateBlobResponse(result, daHeight); !errors.Is(err, coreda.ErrBlobNotFound) && err != nil {
		return err
	}

	for i, data := range result.Data {
		if len(data) > common.DefaultMaxBlobSize {
			r.logger.Debug().
				Uint64("da_height", daHeight).
				Int("index", i).
				Uint64("blob_size", uint64(len(data))).
				Msg("Following data exceeds maximum blob size. Skipping...")
			continue
		}

		// Calculate size of this specific data item
		dataSize := len(data)

		// Check if individual blob exceeds max size
		if dataSize > int(common.DefaultMaxBlobSize) {
			r.logger.Warn().
				Uint64("da_height", daHeight).
				Int("blob_size", dataSize).
				Float64("max_size", common.DefaultMaxBlobSize).
				Msg("forced inclusion blob exceeds maximum size - skipping")
			return fmt.Errorf("%w: blob size %d exceeds maximum %f", ErrForcedInclusionDataTooLarge, dataSize, common.DefaultMaxBlobSize)
		}

		// Check if adding this blob would exceed the current epoch's max size
		if *currentSize+dataSize > int(common.DefaultMaxBlobSize) {
			r.logger.Debug().
				Uint64("da_height", daHeight).
				Int("current_size", *currentSize).
				Int("blob_size", dataSize).
				Msg("blob would exceed max size for this epoch - deferring to pending queue")

			// Store for next epoch
			*newPendingTxs = append(*newPendingTxs, pendingForcedInclusionTx{
				Data:           data,
				OriginalHeight: daHeight,
			})
			continue
		}

		// Include this transaction
		event.Txs = append(event.Txs, data)
		*currentSize += dataSize
		*lastProcessedHeight = daHeight
	}

	return nil
}

// fetchBlobs retrieves blobs from the DA layer
func (r *daRetriever) fetchBlobs(ctx context.Context, daHeight uint64) (coreda.ResultRetrieve, error) {
	// Retrieve from both namespaces
	headerRes := types.RetrieveWithHelpers(ctx, r.da, r.logger, daHeight, r.namespaceBz, defaultDATimeout)

	// If namespaces are the same, return header result
	if bytes.Equal(r.namespaceBz, r.namespaceDataBz) {
		return headerRes, r.validateBlobResponse(headerRes, daHeight)
	}

	dataRes := types.RetrieveWithHelpers(ctx, r.da, r.logger, daHeight, r.namespaceDataBz, defaultDATimeout)

	// Validate responses
	headerErr := r.validateBlobResponse(headerRes, daHeight)
	// ignoring error not found, as data can have data
	if headerErr != nil && !errors.Is(headerErr, coreda.ErrBlobNotFound) {
		return headerRes, headerErr
	}

	dataErr := r.validateBlobResponse(dataRes, daHeight)
	// ignoring error not found, as header can have data
	if dataErr != nil && !errors.Is(dataErr, coreda.ErrBlobNotFound) {
		return dataRes, dataErr
	}

	// Combine successful results
	combinedResult := coreda.ResultRetrieve{
		BaseResult: coreda.BaseResult{
			Code:   coreda.StatusSuccess,
			Height: daHeight,
		},
		Data: make([][]byte, 0),
	}

	if headerRes.Code == coreda.StatusSuccess {
		combinedResult.Data = append(combinedResult.Data, headerRes.Data...)
		combinedResult.IDs = append(combinedResult.IDs, headerRes.IDs...)
	}

	if dataRes.Code == coreda.StatusSuccess {
		combinedResult.Data = append(combinedResult.Data, dataRes.Data...)
		combinedResult.IDs = append(combinedResult.IDs, dataRes.IDs...)
	}

	// Re-throw error not found if both were not found.
	if len(combinedResult.Data) == 0 && len(combinedResult.IDs) == 0 {
		r.logger.Debug().Uint64("da_height", daHeight).Msg("no blob data found")
		combinedResult.Code = coreda.StatusNotFound
		combinedResult.Message = coreda.ErrBlobNotFound.Error()
		return combinedResult, coreda.ErrBlobNotFound
	}

	return combinedResult, nil
}

// validateBlobResponse validates a blob response from DA layer
// those are the only error code returned by da.RetrieveWithHelpers
func (r *daRetriever) validateBlobResponse(res coreda.ResultRetrieve, daHeight uint64) error {
	switch res.Code {
	case coreda.StatusError:
		return fmt.Errorf("DA retrieval failed: %s", res.Message)
	case coreda.StatusHeightFromFuture:
		return fmt.Errorf("%w: height from future", coreda.ErrHeightFromFuture)
	case coreda.StatusNotFound:
		return fmt.Errorf("%w: blob not found", coreda.ErrBlobNotFound)
	case coreda.StatusSuccess:
		r.logger.Debug().Uint64("da_height", daHeight).Msg("successfully retrieved from DA")
		return nil
	default:
		return nil
	}
}

// processBlobs processes retrieved blobs to extract headers and data and returns height events
func (r *daRetriever) processBlobs(ctx context.Context, blobs [][]byte, daHeight uint64) []common.DAHeightEvent {
	// Decode all blobs
	for _, bz := range blobs {
		if len(bz) == 0 {
			continue
		}

		if header := r.tryDecodeHeader(bz, daHeight); header != nil {
			if _, ok := r.pendingHeaders[header.Height()]; ok {
				// a (malicious) node may have re-published valid header to another da height (should never happen)
				// we can already discard it, only the first one is valid
				r.logger.Debug().Uint64("height", header.Height()).Uint64("da_height", daHeight).Msg("header blob already exists for height, discarding")
				continue
			}

			r.pendingHeaders[header.Height()] = header
			continue
		}

		if data := r.tryDecodeData(bz, daHeight); data != nil {
			if _, ok := r.pendingData[data.Height()]; ok {
				// a (malicious) node may have re-published valid data to another da height (should never happen)
				// we can already discard it, only the first one is valid
				r.logger.Debug().Uint64("height", data.Height()).Uint64("da_height", daHeight).Msg("data blob already exists for height, discarding")
				continue
			}

			r.pendingData[data.Height()] = data
		}
	}

	var events []common.DAHeightEvent

	// Match headers with data and create events
	for height, header := range r.pendingHeaders {
		data := r.pendingData[height]

		// Handle empty data case
		if data == nil {
			if isEmptyDataExpected(header) {
				data = createEmptyDataForHeader(ctx, header)
				delete(r.pendingHeaders, height)
			} else {
				// keep header in pending headers until data lands
				r.logger.Debug().Uint64("height", height).Msg("header found but no matching data")
				continue
			}
		} else {
			delete(r.pendingHeaders, height)
			delete(r.pendingData, height)
		}

		// Create height event
		event := common.DAHeightEvent{
			Header:   header,
			Data:     data,
			DaHeight: daHeight,
			Source:   common.SourceDA,
		}

		events = append(events, event)

		r.logger.Info().Uint64("height", height).Uint64("da_height", daHeight).Msg("processed block from DA")
	}

	return events
}

// tryDecodeHeader attempts to decode a blob as a header
func (r *daRetriever) tryDecodeHeader(bz []byte, daHeight uint64) *types.SignedHeader {
	header := new(types.SignedHeader)
	var headerPb pb.SignedHeader

	if err := proto.Unmarshal(bz, &headerPb); err != nil {
		return nil
	}

	if err := header.FromProto(&headerPb); err != nil {
		return nil
	}

	// Basic validation
	if err := header.Header.ValidateBasic(); err != nil {
		r.logger.Debug().Err(err).Msg("invalid header structure")
		return nil
	}

	// Check proposer
	if err := r.assertExpectedProposer(header.ProposerAddress); err != nil {
		r.logger.Debug().Err(err).Msg("unexpected proposer")
		return nil
	}

	// Optimistically mark as DA included
	// This has to be done for all fetched DA headers prior to validation because P2P does not confirm
	// da inclusion. This is not an issue, as an invalid header will be rejected. There cannot be hash collisions.
	headerHash := header.Hash().String()
	r.cache.SetHeaderDAIncluded(headerHash, daHeight, header.Height())

	r.logger.Info().
		Str("header_hash", headerHash).
		Uint64("da_height", daHeight).
		Uint64("height", header.Height()).
		Msg("optimistically marked header as DA included")

	return header
}

// tryDecodeData attempts to decode a blob as signed data
func (r *daRetriever) tryDecodeData(bz []byte, daHeight uint64) *types.Data {
	var signedData types.SignedData
	if err := signedData.UnmarshalBinary(bz); err != nil {
		return nil
	}

	// Skip completely empty data
	if len(signedData.Txs) == 0 && len(signedData.Signature) == 0 {
		return nil
	}

	// Validate signature using the configured provider
	if err := r.assertValidSignedData(&signedData); err != nil {
		r.logger.Debug().Err(err).Msg("invalid signed data")
		return nil
	}

	// Mark as DA included
	dataHash := signedData.Data.DACommitment().String()
	r.cache.SetDataDAIncluded(dataHash, daHeight, signedData.Height())

	r.logger.Info().
		Str("data_hash", dataHash).
		Uint64("da_height", daHeight).
		Uint64("height", signedData.Height()).
		Msg("data marked as DA included")

	return &signedData.Data
}

// assertExpectedProposer validates the proposer address
func (r *daRetriever) assertExpectedProposer(proposerAddr []byte) error {
	if string(proposerAddr) != string(r.genesis.ProposerAddress) {
		return fmt.Errorf("unexpected proposer: got %x, expected %x",
			proposerAddr, r.genesis.ProposerAddress)
	}
	return nil
}

// assertValidSignedData validates signed data using the configured signature provider
func (r *daRetriever) assertValidSignedData(signedData *types.SignedData) error {
	if signedData == nil || signedData.Txs == nil {
		return errors.New("empty signed data")
	}

	if err := r.assertExpectedProposer(signedData.Signer.Address); err != nil {
		return err
	}

	dataBytes, err := signedData.Data.MarshalBinary()
	if err != nil {
		return fmt.Errorf("failed to get signed data payload: %w", err)
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

// isEmptyDataExpected checks if empty data is expected for a header
func isEmptyDataExpected(header *types.SignedHeader) bool {
	return len(header.DataHash) == 0 || bytes.Equal(header.DataHash, common.DataHashForEmptyTxs)
}

// createEmptyDataForHeader creates empty data for a header
func createEmptyDataForHeader(ctx context.Context, header *types.SignedHeader) *types.Data {
	return &types.Data{
		Txs: make(types.Txs, 0),
		Metadata: &types.Metadata{
			ChainID:      header.ChainID(),
			Height:       header.Height(),
			Time:         header.BaseHeader.Time,
			LastDataHash: nil, // LastDataHash must be filled in the syncer, as it is not available here, block n-1 has not been processed yet.
		},
	}
}

// processPendingForcedInclusionTxs processes pending transactions and returns those that fit within the max blob size.
// Returns the transactions to include, the indices of transactions to remove, and the total size used.
func (r *daRetriever) processPendingForcedInclusionTxs() ([][]byte, []int, int) {
	var (
		currentSize     int
		txs             [][]byte
		indicesToRemove []int
	)

	for i, pendingTx := range r.pendingForcedInclusionTxs {
		dataSize := len(pendingTx.Data)
		if currentSize+dataSize > int(common.DefaultMaxBlobSize) {
			r.logger.Debug().
				Int("current_size", currentSize).
				Int("data_size", dataSize).
				Msg("pending transaction would exceed max blob size, will retry later")
			break
		}

		txs = append(txs, pendingTx.Data)
		currentSize += dataSize
		indicesToRemove = append(indicesToRemove, i)
	}

	return txs, indicesToRemove, currentSize
}

// removePendingForcedInclusionTxs removes pending transactions at the specified indices.
// Indices must be sorted in ascending order.
func (r *daRetriever) removePendingForcedInclusionTxs(indices []int) {
	if len(indices) == 0 {
		return
	}

	// Create a new slice without the removed elements
	newPending := make([]pendingForcedInclusionTx, 0, len(r.pendingForcedInclusionTxs)-len(indices))
	removeMap := make(map[int]bool, len(indices))
	for _, idx := range indices {
		removeMap[idx] = true
	}

	for i, tx := range r.pendingForcedInclusionTxs {
		if !removeMap[i] {
			newPending = append(newPending, tx)
		}
	}

	r.pendingForcedInclusionTxs = newPending
}
