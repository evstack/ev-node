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
	dAFetcherTimeout = 30 * time.Second
	dAFetcherRetries = 10
)

// DARetriever handles DA retrieval operations for syncing
type DARetriever struct {
	da      coreda.DA
	cache   cache.Manager
	config  config.Config
	genesis genesis.Genesis
	options common.BlockOptions
	logger  zerolog.Logger
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
		da:      da,
		cache:   cache,
		config:  config,
		genesis: genesis,
		options: options,
		logger:  logger.With().Str("component", "da_retriever").Logger(),
	}
}

// RetrieveFromDA retrieves blocks from the specified DA height and returns height events
func (r *DARetriever) RetrieveFromDA(ctx context.Context, daHeight uint64) ([]common.DAHeightEvent, error) {
	r.logger.Debug().Uint64("da_height", daHeight).Msg("retrieving from DA")

	var err error
	for retry := 0; retry < dAFetcherRetries; retry++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		blobsResp, fetchErr := r.fetchBlobs(ctx, daHeight)
		if fetchErr == nil {
			if blobsResp.Code == coreda.StatusNotFound {
				r.logger.Debug().Uint64("da_height", daHeight).Msg("no blob data found")
				return nil, coreda.ErrBlobNotFound
			}

			r.logger.Debug().Int("blobs", len(blobsResp.Data)).Uint64("da_height", daHeight).Msg("retrieved blob data")
			events := r.processBlobs(ctx, blobsResp.Data, daHeight)
			return events, nil
		}

		if strings.Contains(fetchErr.Error(), coreda.ErrHeightFromFuture.Error()) {
			return nil, fmt.Errorf("%w: height from future", coreda.ErrHeightFromFuture)
		}

		err = errors.Join(err, fetchErr)

		// Delay before retrying
		select {
		case <-ctx.Done():
			return nil, err
		case <-time.After(100 * time.Millisecond):
		}
	}

	return nil, err
}

// fetchBlobs retrieves blobs from the DA layer
func (r *DARetriever) fetchBlobs(ctx context.Context, daHeight uint64) (coreda.ResultRetrieve, error) {
	ctx, cancel := context.WithTimeout(ctx, dAFetcherTimeout)
	defer cancel()

	// Get namespaces
	headerNamespace := []byte(r.config.DA.GetNamespace())
	dataNamespace := []byte(r.config.DA.GetDataNamespace())

	// Retrieve from both namespaces
	headerRes := types.RetrieveWithHelpers(ctx, r.da, r.logger, daHeight, headerNamespace)

	// If namespaces are the same, return header result
	if string(headerNamespace) == string(dataNamespace) {
		return headerRes, r.validateBlobResponse(headerRes, daHeight)
	}

	dataRes := types.RetrieveWithHelpers(ctx, r.da, r.logger, daHeight, dataNamespace)

	// Validate responses
	headerErr := r.validateBlobResponse(headerRes, daHeight)
	dataErr := r.validateBlobResponse(dataRes, daHeight)

	// Handle errors
	if errors.Is(headerErr, coreda.ErrHeightFromFuture) || errors.Is(dataErr, coreda.ErrHeightFromFuture) {
		return coreda.ResultRetrieve{
			BaseResult: coreda.BaseResult{Code: coreda.StatusHeightFromFuture},
		}, fmt.Errorf("%w: height from future", coreda.ErrHeightFromFuture)
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
		if len(headerRes.IDs) > 0 {
			combinedResult.IDs = append(combinedResult.IDs, headerRes.IDs...)
		}
	}

	if dataRes.Code == coreda.StatusSuccess {
		combinedResult.Data = append(combinedResult.Data, dataRes.Data...)
		if len(dataRes.IDs) > 0 {
			combinedResult.IDs = append(combinedResult.IDs, dataRes.IDs...)
		}
	}

	return combinedResult, nil
}

// validateBlobResponse validates a blob response from DA layer
func (r *DARetriever) validateBlobResponse(res coreda.ResultRetrieve, daHeight uint64) error {
	switch res.Code {
	case coreda.StatusError:
		return fmt.Errorf("DA retrieval failed: %s", res.Message)
	case coreda.StatusHeightFromFuture:
		return fmt.Errorf("%w: height from future", coreda.ErrHeightFromFuture)
	case coreda.StatusSuccess:
		r.logger.Debug().Uint64("da_height", daHeight).Msg("successfully retrieved from DA")
		return nil
	default:
		return nil
	}
}

// processBlobs processes retrieved blobs to extract headers and data and returns height events
func (r *DARetriever) processBlobs(ctx context.Context, blobs [][]byte, daHeight uint64) []common.DAHeightEvent {
	headers := make(map[uint64]*types.SignedHeader)
	dataMap := make(map[uint64]*types.Data)
	headerDAHeights := make(map[uint64]uint64) // Track DA height for each header

	// Decode all blobs
	for _, bz := range blobs {
		if len(bz) == 0 {
			continue
		}

		if header := r.tryDecodeHeader(bz, daHeight); header != nil {
			headers[header.Height()] = header
			headerDAHeights[header.Height()] = daHeight
			continue
		}

		if data := r.tryDecodeData(bz, daHeight); data != nil {
			dataMap[data.Height()] = data
		}
	}

	var events []common.DAHeightEvent

	// Match headers with data and create events
	for height, header := range headers {
		data := dataMap[height]

		// Handle empty data case
		if data == nil {
			if r.isEmptyDataExpected(header) {
				data = r.createEmptyDataForHeader(ctx, header)
			} else {
				r.logger.Debug().Uint64("height", height).Msg("header found but no matching data")
				continue
			}
		}

		// Create height event
		event := common.DAHeightEvent{
			Header:                 header,
			Data:                   data,
			DaHeight:               daHeight,
			HeaderDaIncludedHeight: headerDAHeights[height],
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

	// note, we cannot mark the header as DA included
	// we haven't done any signature verification check here
	// signature verification happens with data.

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
