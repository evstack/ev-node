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

// DARetriever handles DA retrieval operations for syncing
type DARetriever struct {
	da      coreda.DA
	cache   cache.Manager
	genesis genesis.Genesis
	logger  zerolog.Logger

	// calculate namespaces bytes once and reuse them
	namespaceBz     []byte
	namespaceDataBz []byte

	// transient cache, only full event need to be passed to the syncer
	// on restart, will be refetch as da height is updated by syncer
	pendingHeaders map[uint64]*types.SignedHeader
	pendingData    map[uint64]*types.Data
}

// NewDARetriever creates a new DA retriever
func NewDARetriever(
	da coreda.DA,
	cache cache.Manager,
	config config.Config,
	genesis genesis.Genesis,
	logger zerolog.Logger,
) *DARetriever {
	return &DARetriever{
		da:              da,
		cache:           cache,
		genesis:         genesis,
		logger:          logger.With().Str("component", "da_retriever").Logger(),
		namespaceBz:     coreda.NamespaceFromString(config.DA.GetNamespace()).Bytes(),
		namespaceDataBz: coreda.NamespaceFromString(config.DA.GetDataNamespace()).Bytes(),
		pendingHeaders:  make(map[uint64]*types.SignedHeader),
		pendingData:     make(map[uint64]*types.Data),
	}
}

// RetrieveFromDA retrieves blocks from the specified DA height and returns height events
func (r *DARetriever) RetrieveFromDA(ctx context.Context, daHeight uint64) ([]common.DAHeightEvent, error) {
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

// fetchBlobs retrieves blobs from the DA layer
func (r *DARetriever) fetchBlobs(ctx context.Context, daHeight uint64) (coreda.ResultRetrieve, error) {
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
func (r *DARetriever) validateBlobResponse(res coreda.ResultRetrieve, daHeight uint64) error {
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
func (r *DARetriever) processBlobs(ctx context.Context, blobs [][]byte, daHeight uint64) []common.DAHeightEvent {
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
func (r *DARetriever) tryDecodeHeader(bz []byte, daHeight uint64) *types.SignedHeader {
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
	r.cache.SetHeaderDAIncluded(headerHash, daHeight)

	r.logger.Info().
		Str("header_hash", headerHash).
		Uint64("da_height", daHeight).
		Uint64("height", header.Height()).
		Msg("optimistically marked header as DA included")

	return header
}

// tryDecodeData attempts to decode a blob as signed data
func (r *DARetriever) tryDecodeData(bz []byte, daHeight uint64) *types.Data {
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
	r.cache.SetDataDAIncluded(dataHash, daHeight)

	r.logger.Info().
		Str("data_hash", dataHash).
		Uint64("da_height", daHeight).
		Uint64("height", signedData.Height()).
		Msg("data marked as DA included")

	return &signedData.Data
}

// assertExpectedProposer validates the proposer address
func (r *DARetriever) assertExpectedProposer(proposerAddr []byte) error {
	if string(proposerAddr) != string(r.genesis.ProposerAddress) {
		return fmt.Errorf("unexpected proposer: got %x, expected %x",
			proposerAddr, r.genesis.ProposerAddress)
	}
	return nil
}

// assertValidSignedData validates signed data using the configured signature provider
func (r *DARetriever) assertValidSignedData(signedData *types.SignedData) error {
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
