package submitting

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	coreda "github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/types"
)

// DASubmitter handles DA submission operations
type DASubmitter struct {
	da      coreda.DA
	config  config.Config
	genesis genesis.Genesis
	options common.BlockOptions
	logger  zerolog.Logger
}

// NewDASubmitter creates a new DA submitter
func NewDASubmitter(
	da coreda.DA,
	config config.Config,
	genesis genesis.Genesis,
	options common.BlockOptions,
	logger zerolog.Logger,
) *DASubmitter {
	return &DASubmitter{
		da:      da,
		config:  config,
		genesis: genesis,
		options: options,
		logger:  logger.With().Str("component", "da_submitter").Logger(),
	}
}

// SubmitHeaders submits pending headers to DA layer
func (s *DASubmitter) SubmitHeaders(ctx context.Context, cache cache.Manager) error {
	headers, err := cache.GetPendingHeaders(ctx)
	if err != nil {
		return fmt.Errorf("failed to get pending headers: %w", err)
	}

	if len(headers) == 0 {
		return nil
	}

	s.logger.Info().Int("count", len(headers)).Msg("submitting headers to DA")

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
	namespace := []byte(s.config.DA.GetNamespace())
	result := types.SubmitWithHelpers(ctx, s.da, s.logger, blobs, 0.0, namespace, nil)

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

	s.logger.Info().Int("count", len(headers)).Uint64("da_height", result.Height).Msg("submitted headers to DA")
	return nil
}

// SubmitData submits pending data to DA layer
func (s *DASubmitter) SubmitData(ctx context.Context, cache cache.Manager, signer signer.Signer, genesis genesis.Genesis) error {
	dataList, err := cache.GetPendingData(ctx)
	if err != nil {
		return fmt.Errorf("failed to get pending data: %w", err)
	}

	if len(dataList) == 0 {
		return nil
	}

	// Sign the data
	signedDataList, err := s.createSignedData(dataList, signer, genesis)
	if err != nil {
		return fmt.Errorf("failed to create signed data: %w", err)
	}

	if len(signedDataList) == 0 {
		return nil // No non-empty data to submit
	}

	s.logger.Info().Int("count", len(signedDataList)).Msg("submitting data to DA")

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
	namespace := []byte(s.config.DA.GetDataNamespace())
	result := types.SubmitWithHelpers(ctx, s.da, s.logger, blobs, 0.0, namespace, nil)

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

	s.logger.Info().Int("count", len(signedDataList)).Uint64("da_height", result.Height).Msg("submitted data to DA")
	return nil
}

// createSignedData creates signed data from raw data
func (s *DASubmitter) createSignedData(dataList []*types.SignedData, signer signer.Signer, genesis genesis.Genesis) ([]*types.SignedData, error) {
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
