package common

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
	coreda "github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/types"
	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

const (
	dAFetcherTimeout  = 30 * time.Second
	dAFetcherRetries  = 10
	submissionTimeout = 60 * time.Second
	maxSubmitAttempts = 30
)

// HeightEvent represents a block height event with header and data
type HeightEvent struct {
	Header                 *types.SignedHeader
	Data                   *types.Data
	DaHeight               uint64
	HeaderDaIncludedHeight uint64
}

// DAHandler handles all DA layer operations for the syncer
type DAHandler struct {
	da      coreda.DA
	cache   cache.Manager
	config  config.Config
	genesis genesis.Genesis
	options BlockOptions
	logger  zerolog.Logger
}

// NewDAHandler creates a new DA handler
func NewDAHandler(
	da coreda.DA,
	cache cache.Manager,
	config config.Config,
	genesis genesis.Genesis,
	options BlockOptions,
	logger zerolog.Logger,
) *DAHandler {
	return &DAHandler{
		da:      da,
		cache:   cache,
		config:  config,
		genesis: genesis,
		options: options,
		logger:  logger.With().Str("component", "da_handler").Logger(),
	}
}

// RetrieveFromDA retrieves blocks from the specified DA height and returns height events
func (h *DAHandler) RetrieveFromDA(ctx context.Context, daHeight uint64) ([]HeightEvent, error) {
	h.logger.Debug().Uint64("da_height", daHeight).Msg("retrieving from DA")

	var err error
	for r := 0; r < dAFetcherRetries; r++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		blobsResp, fetchErr := h.fetchBlobs(ctx, daHeight)
		if fetchErr == nil {
			if blobsResp.Code == coreda.StatusNotFound {
				h.logger.Debug().Uint64("da_height", daHeight).Msg("no blob data found")
				return nil, nil
			}

			h.logger.Debug().Int("blobs", len(blobsResp.Data)).Uint64("da_height", daHeight).Msg("retrieved blob data")
			events := h.processBlobs(ctx, blobsResp.Data, daHeight)
			return events, nil

		} else if strings.Contains(fetchErr.Error(), coreda.ErrHeightFromFuture.Error()) {
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
func (h *DAHandler) fetchBlobs(ctx context.Context, daHeight uint64) (coreda.ResultRetrieve, error) {
	ctx, cancel := context.WithTimeout(ctx, dAFetcherTimeout)
	defer cancel()

	// Get namespaces
	headerNamespace := []byte(h.config.DA.GetNamespace())
	dataNamespace := []byte(h.config.DA.GetDataNamespace())

	// Retrieve from both namespaces
	headerRes := types.RetrieveWithHelpers(ctx, h.da, h.logger, daHeight, headerNamespace)

	// If namespaces are the same, return header result
	if string(headerNamespace) == string(dataNamespace) {
		return headerRes, h.validateBlobResponse(headerRes, daHeight)
	}

	dataRes := types.RetrieveWithHelpers(ctx, h.da, h.logger, daHeight, dataNamespace)

	// Validate responses
	headerErr := h.validateBlobResponse(headerRes, daHeight)
	dataErr := h.validateBlobResponse(dataRes, daHeight)

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
func (h *DAHandler) validateBlobResponse(res coreda.ResultRetrieve, daHeight uint64) error {
	switch res.Code {
	case coreda.StatusError:
		return fmt.Errorf("DA retrieval failed: %s", res.Message)
	case coreda.StatusHeightFromFuture:
		return fmt.Errorf("%w: height from future", coreda.ErrHeightFromFuture)
	case coreda.StatusSuccess:
		h.logger.Debug().Uint64("da_height", daHeight).Msg("successfully retrieved from DA")
		return nil
	default:
		return nil
	}
}

// processBlobs processes retrieved blobs to extract headers and data and returns height events
func (h *DAHandler) processBlobs(ctx context.Context, blobs [][]byte, daHeight uint64) []HeightEvent {
	headers := make(map[uint64]*types.SignedHeader)
	dataMap := make(map[uint64]*types.Data)
	headerDAHeights := make(map[uint64]uint64) // Track DA height for each header

	// Decode all blobs
	for _, bz := range blobs {
		if len(bz) == 0 {
			continue
		}

		if header := h.tryDecodeHeader(bz, daHeight); header != nil {
			headers[header.Height()] = header
			headerDAHeights[header.Height()] = daHeight
			continue
		}

		if data := h.tryDecodeData(bz, daHeight); data != nil {
			dataMap[data.Height()] = data
		}
	}

	var events []HeightEvent

	// Match headers with data and create events
	for height, header := range headers {
		data := dataMap[height]

		// Handle empty data case
		if data == nil {
			if h.isEmptyDataExpected(header) {
				data = h.createEmptyDataForHeader(ctx, header)
			} else {
				h.logger.Debug().Uint64("height", height).Msg("header found but no matching data")
				continue
			}
		}

		// Create height event
		event := HeightEvent{
			Header:                 header,
			Data:                   data,
			DaHeight:               daHeight,
			HeaderDaIncludedHeight: headerDAHeights[height],
		}

		events = append(events, event)

		h.logger.Info().Uint64("height", height).Uint64("da_height", daHeight).Msg("processed block from DA")
	}

	return events
}

// tryDecodeHeader attempts to decode a blob as a header
func (h *DAHandler) tryDecodeHeader(bz []byte, daHeight uint64) *types.SignedHeader {
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
		h.logger.Debug().Err(err).Msg("invalid header structure")
		return nil
	}

	// Check proposer
	if err := h.assertExpectedProposer(header.ProposerAddress); err != nil {
		h.logger.Debug().Err(err).Msg("unexpected proposer")
		return nil
	}

	// Mark as DA included
	headerHash := header.Hash().String()
	h.cache.SetHeaderDAIncluded(headerHash, daHeight)

	h.logger.Info().
		Str("header_hash", headerHash).
		Uint64("da_height", daHeight).
		Uint64("height", header.Height()).
		Msg("header marked as DA included")

	return header
}

// tryDecodeData attempts to decode a blob as signed data
func (h *DAHandler) tryDecodeData(bz []byte, daHeight uint64) *types.Data {
	var signedData types.SignedData
	if err := signedData.UnmarshalBinary(bz); err != nil {
		return nil
	}

	// Skip completely empty data
	if len(signedData.Txs) == 0 && len(signedData.Signature) == 0 {
		return nil
	}

	// Validate signature using the configured provider
	if err := h.assertValidSignedData(&signedData); err != nil {
		h.logger.Debug().Err(err).Msg("invalid signed data")
		return nil
	}

	// Mark as DA included
	dataHash := signedData.Data.DACommitment().String()
	h.cache.SetDataDAIncluded(dataHash, daHeight)

	h.logger.Info().
		Str("data_hash", dataHash).
		Uint64("da_height", daHeight).
		Uint64("height", signedData.Height()).
		Msg("data marked as DA included")

	return &signedData.Data
}

// isEmptyDataExpected checks if empty data is expected for a header
func (h *DAHandler) isEmptyDataExpected(header *types.SignedHeader) bool {
	return len(header.DataHash) == 0 || !bytes.Equal(header.DataHash, DataHashForEmptyTxs)
}

// createEmptyDataForHeader creates empty data for a header
func (h *DAHandler) createEmptyDataForHeader(ctx context.Context, header *types.SignedHeader) *types.Data {
	return &types.Data{
		Metadata: &types.Metadata{
			ChainID: header.ChainID(),
			Height:  header.Height(),
			Time:    header.BaseHeader.Time,
		},
	}
}

// assertExpectedProposer validates the proposer address
func (h *DAHandler) assertExpectedProposer(proposerAddr []byte) error {
	if string(proposerAddr) != string(h.genesis.ProposerAddress) {
		return fmt.Errorf("unexpected proposer: got %x, expected %x",
			proposerAddr, h.genesis.ProposerAddress)
	}
	return nil
}

// assertValidSignedData validates signed data using the configured signature provider
func (h *DAHandler) assertValidSignedData(signedData *types.SignedData) error {
	if signedData == nil || signedData.Txs == nil {
		return errors.New("empty signed data")
	}

	if err := h.assertExpectedProposer(signedData.Signer.Address); err != nil {
		return err
	}

	// Create a header from the signed data metadata for signature verification
	header := types.Header{
		BaseHeader: types.BaseHeader{
			ChainID: signedData.ChainID(),
			Height:  signedData.Height(),
			Time:    uint64(signedData.Time().UnixNano()),
		},
	}

	// Use the configured sync node signature bytes provider
	dataBytes, err := h.options.SyncNodeSignatureBytesProvider(context.Background(), &header, &signedData.Data)
	if err != nil {
		return fmt.Errorf("failed to get signature payload: %w", err)
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

// SubmitHeaders submits pending headers to DA layer
func (h *DAHandler) SubmitHeaders(ctx context.Context, cache cache.Manager) error {
	headers, err := cache.GetPendingHeaders(ctx)
	if err != nil {
		return fmt.Errorf("failed to get pending headers: %w", err)
	}

	if len(headers) == 0 {
		return nil
	}

	h.logger.Info().Int("count", len(headers)).Msg("submitting headers to DA")

	// Convert headers to blobs
	blobs := make([][]byte, len(headers))
	for i, header := range headers {
		headerPb, err := header.ToProto()
		if err != nil {
			return fmt.Errorf("failed to convert header to proto: %w", err)
		}

		blob, err := proto.Marshal(headerPb)
		if err != nil {
			return fmt.Errorf("failed to marshal header: %w", err)
		}

		blobs[i] = blob
	}

	// Submit to DA
	namespace := []byte(h.config.DA.GetNamespace())
	result := types.SubmitWithHelpers(ctx, h.da, h.logger, blobs, 0.0, namespace, nil)

	if result.Code != coreda.StatusSuccess {
		return fmt.Errorf("failed to submit headers: %s", result.Message)
	}

	// Update cache with DA inclusion
	for _, header := range headers {
		cache.SetHeaderDAIncluded(header.Hash().String(), result.Height)
	}

	// Update last submitted height
	if len(headers) > 0 {
		lastHeight := headers[len(headers)-1].Height()
		cache.SetLastSubmittedHeaderHeight(ctx, lastHeight)
	}

	h.logger.Info().Int("count", len(headers)).Uint64("da_height", result.Height).Msg("submitted headers to DA")
	return nil
}

// SubmitData submits pending data to DA layer
func (h *DAHandler) SubmitData(ctx context.Context, cache cache.Manager, signer signer.Signer, genesis genesis.Genesis) error {
	dataList, err := cache.GetPendingData(ctx)
	if err != nil {
		return fmt.Errorf("failed to get pending data: %w", err)
	}

	if len(dataList) == 0 {
		return nil
	}

	// Sign the data
	signedDataList, err := h.createSignedData(dataList, signer, genesis)
	if err != nil {
		return fmt.Errorf("failed to create signed data: %w", err)
	}

	if len(signedDataList) == 0 {
		return nil // No non-empty data to submit
	}

	h.logger.Info().Int("count", len(signedDataList)).Msg("submitting data to DA")

	// Convert to blobs
	blobs := make([][]byte, len(signedDataList))
	for i, signedData := range signedDataList {
		blob, err := signedData.MarshalBinary()
		if err != nil {
			return fmt.Errorf("failed to marshal signed data: %w", err)
		}
		blobs[i] = blob
	}

	// Submit to DA
	namespace := []byte(h.config.DA.GetDataNamespace())
	result := types.SubmitWithHelpers(ctx, h.da, h.logger, blobs, 0.0, namespace, nil)

	if result.Code != coreda.StatusSuccess {
		return fmt.Errorf("failed to submit data: %s", result.Message)
	}

	// Update cache with DA inclusion
	for _, signedData := range signedDataList {
		cache.SetDataDAIncluded(signedData.Data.DACommitment().String(), result.Height)
	}

	// Update last submitted height
	if len(signedDataList) > 0 {
		lastHeight := signedDataList[len(signedDataList)-1].Height()
		cache.SetLastSubmittedDataHeight(ctx, lastHeight)
	}

	h.logger.Info().Int("count", len(signedDataList)).Uint64("da_height", result.Height).Msg("submitted data to DA")
	return nil
}

// createSignedData creates signed data from raw data
func (h *DAHandler) createSignedData(dataList []*types.SignedData, signer signer.Signer, genesis genesis.Genesis) ([]*types.SignedData, error) {
	if signer == nil {
		return nil, fmt.Errorf("signer is nil")
	}

	pubKey, err := signer.GetPublic()
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	signerInfo := types.Signer{
		PubKey:  pubKey,
		Address: genesis.ProposerAddress,
	}

	signedDataList := make([]*types.SignedData, 0, len(dataList))

	for _, data := range dataList {
		// Skip empty data
		if len(data.Txs) == 0 {
			continue
		}

		// Sign the data
		dataBytes, err := data.Data.MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal data: %w", err)
		}

		signature, err := signer.Sign(dataBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to sign data: %w", err)
		}

		signedData := &types.SignedData{
			Data:      data.Data,
			Signature: signature,
			Signer:    signerInfo,
		}

		signedDataList = append(signedDataList, signedData)
	}

	return signedDataList, nil
}
