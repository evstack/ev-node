package syncing

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
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

const (
	dAFetcherTimeout = 10 * time.Second
	batchSize        = 100
)

// DARetriever handles DA retrieval operations for syncing
type DARetriever struct {
	da      coreda.DA
	cache   cache.Manager
	genesis genesis.Genesis
	options common.BlockOptions
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
	options common.BlockOptions,
	logger zerolog.Logger,
) *DARetriever {
	return &DARetriever{
		da:              da,
		cache:           cache,
		genesis:         genesis,
		options:         options,
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
	ctx, cancel := context.WithTimeout(ctx, dAFetcherTimeout)
	defer cancel()

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
	headerRes := r.retrieveNamespace(ctx, daHeight, r.namespaceBz)
	headerErr := retrievalError(headerRes)

	if bytes.Equal(r.namespaceBz, r.namespaceDataBz) {
		if errors.Is(headerErr, coreda.ErrBlobNotFound) {
			r.logger.Debug().Uint64("da_height", daHeight).Msg("no blob data found")
		}
		return headerRes, headerErr
	}

	dataRes := r.retrieveNamespace(ctx, daHeight, r.namespaceDataBz)
	dataErr := retrievalError(dataRes)

	// Propagate unexpected errors but allow NotFound so callers can combine results
	if headerErr != nil && !errors.Is(headerErr, coreda.ErrBlobNotFound) {
		return headerRes, headerErr
	}
	if dataErr != nil && !errors.Is(dataErr, coreda.ErrBlobNotFound) {
		return dataRes, dataErr
	}

	combined := coreda.ResultRetrieve{
		BaseResult: coreda.BaseResult{
			Height: daHeight,
		},
	}

	if headerErr == nil && headerRes.Code == coreda.StatusSuccess {
		combined.Data = append(combined.Data, headerRes.Data...)
		combined.BaseResult.IDs = append(combined.BaseResult.IDs, headerRes.IDs...)
		combined.BaseResult.Timestamp = headerRes.Timestamp
	}

	if dataErr == nil && dataRes.Code == coreda.StatusSuccess {
		combined.Data = append(combined.Data, dataRes.Data...)
		combined.BaseResult.IDs = append(combined.BaseResult.IDs, dataRes.IDs...)
		if combined.BaseResult.Timestamp.IsZero() {
			combined.BaseResult.Timestamp = dataRes.Timestamp
		}
	}

	if len(combined.Data) == 0 {
		r.logger.Debug().Uint64("da_height", daHeight).Msg("no blob data found")
		combined.BaseResult.Code = coreda.StatusNotFound
		combined.BaseResult.Message = coreda.ErrBlobNotFound.Error()
		combined.BaseResult.Timestamp = time.Now()
		return combined, coreda.ErrBlobNotFound
	}

	combined.BaseResult.Code = coreda.StatusSuccess
	if combined.BaseResult.Timestamp.IsZero() {
		combined.BaseResult.Timestamp = time.Now()
	}

	return combined, nil
}

// retrieveNamespace performs the DA calls for a single namespace and maps errors to ResultRetrieve.
func (r *DARetriever) retrieveNamespace(ctx context.Context, daHeight uint64, namespace []byte) coreda.ResultRetrieve {
	idsResult, err := r.da.GetIDs(ctx, daHeight, namespace)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), coreda.ErrBlobNotFound.Error()):
			return coreda.ResultRetrieve{
				BaseResult: coreda.BaseResult{
					Code:      coreda.StatusNotFound,
					Message:   coreda.ErrBlobNotFound.Error(),
					Height:    daHeight,
					Timestamp: time.Now(),
				},
			}
		case strings.Contains(err.Error(), coreda.ErrHeightFromFuture.Error()):
			return coreda.ResultRetrieve{
				BaseResult: coreda.BaseResult{
					Code:      coreda.StatusHeightFromFuture,
					Message:   coreda.ErrHeightFromFuture.Error(),
					Height:    daHeight,
					Timestamp: time.Now(),
				},
			}
		default:
			r.logger.Error().Uint64("height", daHeight).Err(err).Msg("failed to get IDs from DA")
			return coreda.ResultRetrieve{
				BaseResult: coreda.BaseResult{
					Code:      coreda.StatusError,
					Message:   fmt.Sprintf("failed to get IDs: %s", err.Error()),
					Height:    daHeight,
					Timestamp: time.Now(),
				},
			}
		}
	}

	if idsResult == nil || len(idsResult.IDs) == 0 {
		return coreda.ResultRetrieve{
			BaseResult: coreda.BaseResult{
				Code:      coreda.StatusNotFound,
				Message:   coreda.ErrBlobNotFound.Error(),
				Height:    daHeight,
				Timestamp: time.Now(),
			},
		}
	}

	blobs := make([][]byte, 0, len(idsResult.IDs))
	for i := 0; i < len(idsResult.IDs); i += batchSize {
		end := min(i+batchSize, len(idsResult.IDs))

		batchBlobs, err := r.da.Get(ctx, idsResult.IDs[i:end], namespace)
		if err != nil {
			r.logger.Error().Uint64("height", daHeight).Int("batch_start", i).Int("batch_end", end-1).Err(err).Msg("failed to get blobs from DA")
			return coreda.ResultRetrieve{
				BaseResult: coreda.BaseResult{
					Code:      coreda.StatusError,
					Message:   fmt.Sprintf("failed to get blobs for batch %d-%d: %s", i, end-1, err.Error()),
					Height:    daHeight,
					Timestamp: time.Now(),
				},
			}
		}
		blobs = append(blobs, batchBlobs...)
	}

	r.logger.Debug().Uint64("height", daHeight).Int("num_blobs", len(blobs)).Msg("retrieved blobs from namespace")
	return coreda.ResultRetrieve{
		BaseResult: coreda.BaseResult{
			Code:      coreda.StatusSuccess,
			Height:    daHeight,
			IDs:       idsResult.IDs,
			Timestamp: idsResult.Timestamp,
		},
		Data: blobs,
	}
}

// retrievalError converts a ResultRetrieve status into an error for flow-control purposes.
func retrievalError(res coreda.ResultRetrieve) error {
	switch res.Code {
	case coreda.StatusSuccess:
		return nil
	case coreda.StatusNotFound:
		return coreda.ErrBlobNotFound
	case coreda.StatusHeightFromFuture:
		return coreda.ErrHeightFromFuture
	case coreda.StatusContextCanceled:
		return context.Canceled
	case coreda.StatusError:
		if res.Message != "" {
			return errors.New(res.Message)
		}
		return fmt.Errorf("DA retrieval failed")
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
			r.pendingHeaders[header.Height()] = header
			continue
		}

		if data := r.tryDecodeData(bz, daHeight); data != nil {
			r.pendingData[data.Height()] = data
		}
	}

	var events []common.DAHeightEvent

	// Match headers with data and create events
	for height, header := range r.pendingHeaders {
		data := r.pendingData[height]

		// Handle empty data case
		if data == nil {
			if r.isEmptyDataExpected(header) {
				data = r.createEmptyDataForHeader(ctx, header)
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

// isEmptyDataExpected checks if empty data is expected for a header
func (r *DARetriever) isEmptyDataExpected(header *types.SignedHeader) bool {
	return len(header.DataHash) == 0 || bytes.Equal(header.DataHash, common.DataHashForEmptyTxs)
}

// createEmptyDataForHeader creates empty data for a header
func (r *DARetriever) createEmptyDataForHeader(ctx context.Context, header *types.SignedHeader) *types.Data {
	return &types.Data{
		Metadata: &types.Metadata{
			ChainID: header.ChainID(),
			Height:  header.Height(),
			Time:    header.BaseHeader.Time,
		},
	}
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
