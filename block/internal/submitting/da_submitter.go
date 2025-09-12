package submitting

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	coreda "github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/rpc/server"
	"github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/types"
)

const (
	submissionTimeout            = 60 * time.Second
	noGasPrice                   = -1
	initialBackoff               = 100 * time.Millisecond
	defaultGasPrice              = 0.0
	defaultGasMultiplier         = 1.0
	defaultMaxBlobSize           = 2 * 1024 * 1024 // 2MB fallback blob size limit
	defaultMaxGasPriceClamp      = 1000.0
	defaultMaxGasMultiplierClamp = 3.0
)

// RetryPolicy defines clamped bounds for retries, backoff, and gas pricing.
type RetryPolicy struct {
	MaxAttempts      int
	MinBackoff       time.Duration
	MaxBackoff       time.Duration
	MinGasPrice      float64
	MaxGasPrice      float64
	MaxBlobBytes     int
	MaxGasMultiplier float64
}

// RetryState holds the current retry attempt, backoff, and gas price.
type RetryState struct {
	Attempt  int
	Backoff  time.Duration
	GasPrice float64
}

type retryReason int

const (
	reasonInitial retryReason = iota
	reasonFailure
	reasonMempool
	reasonSuccess
	reasonTooBig
)

func (rs *RetryState) Next(reason retryReason, pol RetryPolicy, gasMultiplier float64, sentinelNoGas bool) {
	switch reason {
	case reasonSuccess:
		// Reduce gas price towards initial on success, if multiplier is available
		if !sentinelNoGas && gasMultiplier > 0 {
			m := clamp(gasMultiplier, 1/pol.MaxGasMultiplier, pol.MaxGasMultiplier)
			rs.GasPrice = clamp(rs.GasPrice/m, pol.MinGasPrice, pol.MaxGasPrice)
		}
		rs.Backoff = pol.MinBackoff
	case reasonMempool:
		if !sentinelNoGas && gasMultiplier > 0 {
			m := clamp(gasMultiplier, 1/pol.MaxGasMultiplier, pol.MaxGasMultiplier)
			rs.GasPrice = clamp(rs.GasPrice*m, pol.MinGasPrice, pol.MaxGasPrice)
		}
		// Honor mempool stalling by using max backoff window
		rs.Backoff = clamp(pol.MaxBackoff, pol.MinBackoff, pol.MaxBackoff)
	case reasonFailure, reasonTooBig:
		if rs.Backoff == 0 {
			rs.Backoff = pol.MinBackoff
		} else {
			rs.Backoff *= 2
		}
		rs.Backoff = clamp(rs.Backoff, pol.MinBackoff, pol.MaxBackoff)
	case reasonInitial:
		rs.Backoff = 0
	}
	rs.Attempt++
}

// clamp constrains a value between min and max bounds for any comparable type
func clamp[T ~float64 | time.Duration](v, min, max T) T {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// getGasMultiplier fetches the gas multiplier from DA layer with fallback and clamping
func (s *DASubmitter) getGasMultiplier(ctx context.Context, pol RetryPolicy) float64 {
	gasMultiplier, err := s.da.GasMultiplier(ctx)
	if err != nil || gasMultiplier <= 0 {
		if s.config.DA.GasMultiplier > 0 {
			return clamp(s.config.DA.GasMultiplier, 0.1, pol.MaxGasMultiplier)
		}
		s.logger.Warn().Err(err).Msg("failed to get gas multiplier from DA layer, using default 1.0")
		return defaultGasMultiplier
	}
	return clamp(gasMultiplier, 0.1, pol.MaxGasMultiplier)
}

// initialGasPrice determines the starting gas price with clamping and sentinel handling
func (s *DASubmitter) initialGasPrice(ctx context.Context, pol RetryPolicy) (price float64, sentinelNoGas bool) {
	if s.config.DA.GasPrice == noGasPrice {
		return noGasPrice, true
	}
	if s.config.DA.GasPrice > 0 {
		return clamp(s.config.DA.GasPrice, pol.MinGasPrice, pol.MaxGasPrice), false
	}
	if gp, err := s.da.GasPrice(ctx); err == nil {
		return clamp(gp, pol.MinGasPrice, pol.MaxGasPrice), false
	}
	s.logger.Warn().Msg("DA gas price unavailable; using default 0.0")
	return pol.MinGasPrice, false
}

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

	return submitToDA(s, ctx, headers,
		func(header *types.SignedHeader) ([]byte, error) {
			headerPb, err := header.ToProto()
			if err != nil {
				return nil, fmt.Errorf("failed to convert header to proto: %w", err)
			}
			return proto.Marshal(headerPb)
		},
		func(submitted []*types.SignedHeader, res *coreda.ResultSubmit, gasPrice float64) {
			for _, header := range submitted {
				cache.SetHeaderDAIncluded(header.Hash().String(), res.Height)
			}
			// Update last submitted height
			if l := len(submitted); l > 0 {
				lastHeight := submitted[l-1].Height()
				cache.SetLastSubmittedHeaderHeight(ctx, lastHeight)
			}
		},
		"header",
		[]byte(s.config.DA.GetNamespace()),
		[]byte(s.config.DA.SubmitOptions),
		cache,
	)
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

	return submitToDA(s, ctx, signedDataList,
		func(signedData *types.SignedData) ([]byte, error) {
			return signedData.MarshalBinary()
		},
		func(submitted []*types.SignedData, res *coreda.ResultSubmit, gasPrice float64) {
			for _, sd := range submitted {
				cache.SetDataDAIncluded(sd.Data.DACommitment().String(), res.Height)
			}
			if l := len(submitted); l > 0 {
				lastHeight := submitted[l-1].Height()
				cache.SetLastSubmittedDataHeight(ctx, lastHeight)
			}
		},
		"data",
		[]byte(s.config.DA.GetDataNamespace()),
		[]byte(s.config.DA.SubmitOptions),
		cache,
	)
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

	addr, err := signer.GetAddress()
	if err != nil {
		return nil, fmt.Errorf("failed to get address: %w", err)
	}

	if len(genesis.ProposerAddress) > 0 && !bytes.Equal(addr, genesis.ProposerAddress) {
		return nil, fmt.Errorf("signer address mismatch with genesis proposer")
	}

	signerInfo := types.Signer{
		PubKey:  pubKey,
		Address: addr,
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

// submitToDA is a generic helper for submitting items to the DA layer with retry, backoff, and gas price logic.
func submitToDA[T any](
	s *DASubmitter,
	ctx context.Context,
	items []T,
	marshalFn func(T) ([]byte, error),
	postSubmit func([]T, *coreda.ResultSubmit, float64),
	itemType string,
	namespace []byte,
	options []byte,
	cache cache.Manager,
) error {
	marshaled, err := marshalItems(items, marshalFn, itemType)
	if err != nil {
		return err
	}

	// Build retry policy from config with sane defaults
	pol := RetryPolicy{
		MaxAttempts:      s.config.DA.MaxSubmitAttempts,
		MinBackoff:       initialBackoff,
		MaxBackoff:       s.config.DA.BlockTime.Duration,
		MinGasPrice:      defaultGasPrice,
		MaxGasPrice:      defaultMaxGasPriceClamp,
		MaxBlobBytes:     defaultMaxBlobSize,
		MaxGasMultiplier: defaultMaxGasMultiplierClamp,
	}

	// Choose initial gas price with clamp
	gasPrice, sentinelNoGas := s.initialGasPrice(ctx, pol)
	rs := RetryState{Attempt: 0, Backoff: 0, GasPrice: gasPrice}
	gm := s.getGasMultiplier(ctx, pol)

	// Limit this submission to a single size-capped batch
	if len(marshaled) > 0 {
		batchItems, batchMarshaled, err := limitBatchBySize(items, marshaled, pol.MaxBlobBytes)
		if err != nil {
			return fmt.Errorf("no %s items fit within max blob size: %w", itemType, err)
		}
		items = batchItems
		marshaled = batchMarshaled
	}

	// Start the retry loop
	for rs.Attempt < pol.MaxAttempts {
		if err := waitForBackoffOrContext(ctx, rs.Backoff); err != nil {
			return err
		}

		submitCtx, cancel := context.WithTimeout(ctx, submissionTimeout)
		defer cancel()
		// Perform submission
		res := types.SubmitWithHelpers(submitCtx, s.da, s.logger, marshaled, rs.GasPrice, namespace, options)

		// Record submission result for observability
		if daVisualizationServer := server.GetDAVisualizationServer(); daVisualizationServer != nil {
			daVisualizationServer.RecordSubmission(&res, rs.GasPrice, uint64(len(items)))
		}

		switch res.Code {
		case coreda.StatusSuccess:
			submitted := items[:res.SubmittedCount]
			postSubmit(submitted, &res, rs.GasPrice)
			s.logger.Info().Str("itemType", itemType).Float64("gasPrice", rs.GasPrice).Uint64("count", res.SubmittedCount).Msg("successfully submitted items to DA layer")
			if int(res.SubmittedCount) == len(items) {
				rs.Next(reasonSuccess, pol, gm, sentinelNoGas)
				return nil
			}
			// partial success: advance window
			items = items[res.SubmittedCount:]
			marshaled = marshaled[res.SubmittedCount:]
			rs.Next(reasonSuccess, pol, gm, sentinelNoGas)

		case coreda.StatusTooBig:
			// Iteratively halve until it fits or single-item too big
			if len(items) == 1 {
				s.logger.Error().Str("itemType", itemType).Msg("single item exceeds DA blob size limit")
				rs.Next(reasonTooBig, pol, gm, sentinelNoGas)
				return fmt.Errorf("single %s item exceeds DA blob size limit", itemType)
			}
			half := len(items) / 2
			if half == 0 {
				half = 1
			}
			items = items[:half]
			marshaled = marshaled[:half]
			s.logger.Debug().Int("newBatchSize", half).Msg("batch too big; halving and retrying")
			rs.Next(reasonTooBig, pol, gm, sentinelNoGas)

		case coreda.StatusNotIncludedInBlock, coreda.StatusAlreadyInMempool:
			s.logger.Info().Dur("backoff", pol.MaxBackoff).Float64("gasPrice", rs.GasPrice).Msg("retrying due to mempool state")
			rs.Next(reasonMempool, pol, gm, sentinelNoGas)

		case coreda.StatusContextCanceled:
			s.logger.Info().Msg("DA layer submission canceled due to context cancellation")
			return context.Canceled

		default:
			s.logger.Error().Str("error", res.Message).Int("attempt", rs.Attempt+1).Msg("DA layer submission failed")
			rs.Next(reasonFailure, pol, gm, sentinelNoGas)
		}
	}

	return fmt.Errorf("failed to submit all %s(s) to DA layer after %d attempts", itemType, rs.Attempt)
}

// limitBatchBySize returns a prefix of items whose total marshaled size does not exceed maxBytes.
// If the first item exceeds maxBytes, it returns an error.
func limitBatchBySize[T any](items []T, marshaled [][]byte, maxBytes int) ([]T, [][]byte, error) {
	total := 0
	count := 0
	for i := 0; i < len(items); i++ {
		sz := len(marshaled[i])
		if sz > maxBytes {
			if i == 0 {
				return nil, nil, fmt.Errorf("item size %d exceeds max %d", sz, maxBytes)
			}
			break
		}
		if total+sz > maxBytes {
			break
		}
		total += sz
		count++
	}
	if count == 0 {
		return nil, nil, fmt.Errorf("no items fit within %d bytes", maxBytes)
	}
	return items[:count], marshaled[:count], nil
}

func marshalItems[T any](
	items []T,
	marshalFn func(T) ([]byte, error),
	itemType string,
) ([][]byte, error) {
	marshaled := make([][]byte, len(items))

	// Use a channel to collect errors from goroutines
	errCh := make(chan error, len(items))

	// Marshal items concurrently
	for i, item := range items {
		go func(idx int, itm T) {
			bz, err := marshalFn(itm)
			if err != nil {
				errCh <- fmt.Errorf("failed to marshal %s item at index %d: %w", itemType, idx, err)
				return
			}
			marshaled[idx] = bz
			errCh <- nil
		}(i, item)
	}

	// Wait for all goroutines to complete and check for errors
	for i := 0; i < len(items); i++ {
		if err := <-errCh; err != nil {
			return nil, err
		}
	}

	return marshaled, nil
}

func waitForBackoffOrContext(ctx context.Context, backoff time.Duration) error {
	if backoff <= 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	}
	timer := time.NewTimer(backoff)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
