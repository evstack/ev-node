package block

import (
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

// RetrieveLoop is responsible for interacting with DA layer.
func (m *Manager) RetrieveLoop(ctx context.Context) {
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
			for _, bz := range blobsResp.Data {
				if len(bz) == 0 {
					m.logger.Debug().Uint64("daHeight", daHeight).Msg("ignoring nil or empty blob")
					continue
				}
				if m.handlePotentialHeader(ctx, bz, daHeight) {
					continue
				}
				m.handlePotentialData(ctx, bz, daHeight)
			}
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

// handlePotentialHeader tries to decode and process a header. Returns true if successful or skipped, false if not a header.
func (m *Manager) handlePotentialHeader(ctx context.Context, bz []byte, daHeight uint64) bool {
	header := new(types.SignedHeader)
	var headerPb pb.SignedHeader

	if err := proto.Unmarshal(bz, &headerPb); err != nil {
		m.logger.Debug().Err(err).Msg("failed to unmarshal header")
		return false
	}

	if err := header.FromProto(&headerPb); err != nil {
		// treat as handled, but not valid
		m.logger.Debug().Err(err).Msg("failed to decode unmarshalled header")
		return true
	}

	// set custom verifier to do correct header verification
	header.SetCustomVerifier(m.signaturePayloadProvider)

	// Stronger validation: check for obviously invalid headers using ValidateBasic
	if err := header.ValidateBasic(); err != nil {
		m.logger.Debug().Uint64("daHeight", daHeight).Err(err).Msg("blob does not look like a valid header")
		return false
	}

	// early validation to reject junk headers
	if !m.isUsingExpectedSingleSequencer(header) {
		m.logger.Debug().
			Uint64("headerHeight", header.Height()).
			Str("headerHash", header.Hash().String()).
			Msg("skipping header from unexpected sequencer")
		return true
	}
	headerHash := header.Hash().String()
	m.headerCache.SetDAIncluded(headerHash, daHeight)
	m.sendNonBlockingSignalToDAIncluderCh()
	m.logger.Info().Uint64("headerHeight", header.Height()).Str("headerHash", headerHash).Msg("header marked as DA included")
	if !m.headerCache.IsSeen(headerHash) {
		select {
		case <-ctx.Done():
			return true
		default:
			m.logger.Warn().Uint64("daHeight", daHeight).Msg("headerInCh backlog full, dropping header")
		}
		m.headerInCh <- NewHeaderEvent{header, daHeight}
	}
	return true
}

// handlePotentialData tries to decode and process a data. No return value.
func (m *Manager) handlePotentialData(ctx context.Context, bz []byte, daHeight uint64) {
	var signedData types.SignedData
	err := signedData.UnmarshalBinary(bz)
	if err != nil {
		m.logger.Debug().Err(err).Msg("failed to unmarshal signed data")
		return
	}
	if len(signedData.Txs) == 0 {
		m.logger.Debug().Uint64("daHeight", daHeight).Msg("ignoring empty signed data")
		return
	}

	// Early validation to reject junk data
	if !m.isValidSignedData(&signedData) {
		m.logger.Debug().Uint64("daHeight", daHeight).Msg("invalid data signature")
		return
	}

	dataHashStr := signedData.Data.DACommitment().String()
	m.dataCache.SetDAIncluded(dataHashStr, daHeight)
	m.sendNonBlockingSignalToDAIncluderCh()
	m.logger.Info().Str("dataHash", dataHashStr).Uint64("daHeight", daHeight).Uint64("height", signedData.Height()).Msg("signed data marked as DA included")
	if !m.dataCache.IsSeen(dataHashStr) {
		select {
		case <-ctx.Done():
			return
		default:
			m.logger.Warn().Uint64("daHeight", daHeight).Msg("dataInCh backlog full, dropping signed data")
		}
		m.dataInCh <- NewDataEvent{&signedData.Data, daHeight}
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

	// TODO: we should maintain the original error instead of creating a new one as we lose context by creating a new error.
	blobsRes := types.RetrieveWithHelpers(ctx, m.da, m.logger, daHeight, []byte(m.genesis.ChainID))
	switch blobsRes.Code {
	case coreda.StatusError:
		// Record failed DA retrieval
		m.recordDAMetrics("retrieval", DAModeFail)
		err = fmt.Errorf("failed to retrieve block: %s", blobsRes.Message)
	case coreda.StatusHeightFromFuture:
		// Keep the root cause intact for callers that may rely on errors.Is/As.
		err = fmt.Errorf("%w: %s", coreda.ErrHeightFromFuture, blobsRes.Message)
	}
	return blobsRes, err
}
