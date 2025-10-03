package submitting

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	coreexecutor "github.com/evstack/ev-node/core/execution"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

// daSubmitterAPI defines minimal methods needed by Submitter for DA submissions.
type daSubmitterAPI interface {
	SubmitHeaders(ctx context.Context, cache cache.Manager) error
	SubmitData(ctx context.Context, cache cache.Manager, signer signer.Signer, genesis genesis.Genesis) error
}

// Submitter handles DA submission and inclusion processing for both sync and aggregator nodes
type Submitter struct {
	// Core components
	store   store.Store
	exec    coreexecutor.Executor
	config  config.Config
	genesis genesis.Genesis

	// Shared components
	cache   cache.Manager
	metrics *common.Metrics

	// DA submitter
	daSubmitter daSubmitterAPI

	// Optional signer (only for aggregator nodes)
	signer signer.Signer

	// DA state
	daIncludedHeight uint64
	daStateMtx       *sync.RWMutex

	// Submission state to prevent concurrent submissions
	headerSubmissionMtx sync.Mutex
	dataSubmissionMtx   sync.Mutex

	// Channels for coordination
	errorCh chan<- error // Channel to report critical execution client failures

	// Logging
	logger zerolog.Logger

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewSubmitter creates a new DA submitter component
func NewSubmitter(
	store store.Store,
	exec coreexecutor.Executor,
	cache cache.Manager,
	metrics *common.Metrics,
	config config.Config,
	genesis genesis.Genesis,
	daSubmitter daSubmitterAPI,
	signer signer.Signer, // Can be nil for sync nodes
	logger zerolog.Logger,
	errorCh chan<- error,
) *Submitter {
	return &Submitter{
		store:       store,
		exec:        exec,
		cache:       cache,
		metrics:     metrics,
		config:      config,
		genesis:     genesis,
		daSubmitter: daSubmitter,
		signer:      signer,
		daStateMtx:  &sync.RWMutex{},
		errorCh:     errorCh,
		logger:      logger.With().Str("component", "submitter").Logger(),
	}
}

// Start begins the submitting component
func (s *Submitter) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)

	// Initialize DA included height
	if err := s.initializeDAIncludedHeight(ctx); err != nil {
		return err
	}

	// Start DA submission loop if signer is available (aggregator nodes only)
	if s.signer != nil {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.daSubmissionLoop()
		}()
	}

	// Start DA inclusion processing loop (both sync and aggregator nodes)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.processDAInclusionLoop()
	}()

	return nil
}

// Stop shuts down the submitting component
func (s *Submitter) Stop() error {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
	s.logger.Info().Msg("submitter stopped")
	return nil
}

// updateMetrics updates sync-related metrics
func (s *Submitter) updateMetrics() {
	s.metrics.DAInclusionHeight.Set(float64(s.GetDAIncludedHeight()))
}

// daSubmissionLoop handles submission of headers and data to DA layer (aggregator nodes only)
func (s *Submitter) daSubmissionLoop() {
	s.logger.Info().Msg("starting DA submission loop")
	defer s.logger.Info().Msg("DA submission loop stopped")

	ticker := time.NewTicker(s.config.DA.BlockTime.Duration)
	defer ticker.Stop()

	metricsTicker := time.NewTicker(30 * time.Second)
	defer metricsTicker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			// Submit headers
			if s.cache.NumPendingHeaders() != 0 {
				if s.headerSubmissionMtx.TryLock() {
					go func() {
						defer s.headerSubmissionMtx.Unlock()
						if err := s.daSubmitter.SubmitHeaders(s.ctx, s.cache); err != nil {
							s.logger.Error().Err(err).Msg("failed to submit headers")
						}
					}()
				}
			}

			// Submit data
			if s.cache.NumPendingData() != 0 {
				if s.dataSubmissionMtx.TryLock() {
					go func() {
						defer s.dataSubmissionMtx.Unlock()
						if err := s.daSubmitter.SubmitData(s.ctx, s.cache, s.signer, s.genesis); err != nil {
							s.logger.Error().Err(err).Msg("failed to submit data")
						}
					}()
				}
			}
		case <-metricsTicker.C:
			s.updateMetrics()
		}
	}
}

// processDAInclusionLoop handles DA inclusion processing (both sync and aggregator nodes)
func (s *Submitter) processDAInclusionLoop() {
	s.logger.Info().Msg("starting DA inclusion processing loop")
	defer s.logger.Info().Msg("DA inclusion processing loop stopped")

	ticker := time.NewTicker(s.config.DA.BlockTime.Duration)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			currentDAIncluded := s.GetDAIncludedHeight()

			for {
				nextHeight := currentDAIncluded + 1

				// Get block data first
				header, data, err := s.store.GetBlockData(s.ctx, nextHeight)
				if err != nil {
					break
				}

				// Check if this height is DA included
				if included, err := s.IsHeightDAIncluded(nextHeight, header, data); err != nil || !included {
					break
				}

				s.logger.Debug().Uint64("height", nextHeight).Msg("advancing DA included height")

				// Set sequencer height to DA height mapping using already retrieved data
				if err := s.setSequencerHeightToDAHeight(s.ctx, nextHeight, header, data, currentDAIncluded == 0); err != nil {
					s.logger.Error().Err(err).Uint64("height", nextHeight).Msg("failed to set sequencer height to DA height mapping")
					break
				}

				// Set final height in executor
				if err := s.setFinalWithRetry(nextHeight); err != nil {
					s.sendCriticalError(fmt.Errorf("failed to set final height: %w", err))
					s.logger.Error().Err(err).Uint64("height", nextHeight).Msg("failed to set final height")
					break
				}

				// Update DA included height
				s.SetDAIncludedHeight(nextHeight)
				currentDAIncluded = nextHeight

				// Persist DA included height
				bz := make([]byte, 8)
				binary.LittleEndian.PutUint64(bz, nextHeight)
				if err := s.store.SetMetadata(s.ctx, store.DAIncludedHeightKey, bz); err != nil {
					s.logger.Error().Err(err).Uint64("height", nextHeight).Msg("failed to persist DA included height")
				}
			}
		}
	}
}

// setFinalWithRetry sets the final height in executor with retry logic
func (s *Submitter) setFinalWithRetry(nextHeight uint64) error {
	for attempt := 1; attempt <= common.MaxRetriesBeforeHalt; attempt++ {
		if err := s.exec.SetFinal(s.ctx, nextHeight); err != nil {
			if attempt == common.MaxRetriesBeforeHalt {
				return fmt.Errorf("failed to set final height after %d attempts: %w", attempt, err)
			}

			s.logger.Error().Err(err).
				Int("attempt", attempt).
				Int("max_attempts", common.MaxRetriesBeforeHalt).
				Uint64("da_height", nextHeight).
				Msg("failed to set final height, retrying")

			select {
			case <-time.After(common.MaxRetriesTimeout):
				continue
			case <-s.ctx.Done():
				return fmt.Errorf("context cancelled during retry: %w", s.ctx.Err())
			}
		}

		return nil
	}

	return nil
}

// GetDAIncludedHeight returns the DA included height
func (s *Submitter) GetDAIncludedHeight() uint64 {
	s.daStateMtx.RLock()
	defer s.daStateMtx.RUnlock()
	return s.daIncludedHeight
}

// SetDAIncludedHeight updates the DA included height
func (s *Submitter) SetDAIncludedHeight(height uint64) {
	s.daStateMtx.Lock()
	defer s.daStateMtx.Unlock()
	s.daIncludedHeight = height
}

// initializeDAIncludedHeight loads the DA included height from store
func (s *Submitter) initializeDAIncludedHeight(ctx context.Context) error {
	if height, err := s.store.GetMetadata(ctx, store.DAIncludedHeightKey); err == nil && len(height) == 8 {
		s.SetDAIncludedHeight(binary.LittleEndian.Uint64(height))
	}
	return nil
}

// sendCriticalError sends a critical error to the error channel without blocking
func (s *Submitter) sendCriticalError(err error) {
	if s.errorCh != nil {
		select {
		case s.errorCh <- err:
		default:
			// Channel full, error already reported
		}
	}
}

// setSequencerHeightToDAHeight stores the mapping from a ev-node block height to the corresponding
// DA (Data Availability) layer heights where the block's header and data were included.
// This mapping is persisted in the store metadata and is used to track which DA heights
// contain the block components for a given ev-node height.
//
// For blocks with empty transactions, both header and data use the same DA height since
// empty transaction data is not actually published to the DA layer.
func (s *Submitter) setSequencerHeightToDAHeight(ctx context.Context, height uint64, header *types.SignedHeader, data *types.Data, genesisInclusion bool) error {

	headerHash, dataHash := header.Hash(), data.DACommitment()

	headerHeightBytes := make([]byte, 8)
	daHeightForHeader, ok := s.cache.GetHeaderDAIncluded(headerHash.String())
	if !ok {
		return fmt.Errorf("header hash %s not found in cache", headerHash)
	}
	binary.LittleEndian.PutUint64(headerHeightBytes, daHeightForHeader)
	genesisDAIncludedHeight := daHeightForHeader

	if err := s.store.SetMetadata(ctx, fmt.Sprintf("%s/%d/h", store.HeightToDAHeightKey, height), headerHeightBytes); err != nil {
		return err
	}

	dataHeightBytes := make([]byte, 8)
	// For empty transactions, use the same DA height as the header
	if bytes.Equal(dataHash, common.DataHashForEmptyTxs) {
		binary.LittleEndian.PutUint64(dataHeightBytes, daHeightForHeader)
	} else {
		daHeightForData, ok := s.cache.GetDataDAIncluded(dataHash.String())
		if !ok {
			return fmt.Errorf("data hash %s not found in cache", dataHash.String())
		}
		binary.LittleEndian.PutUint64(dataHeightBytes, daHeightForData)

		// if data posted before header, use data da included height
		if daHeightForData < genesisDAIncludedHeight {
			genesisDAIncludedHeight = daHeightForData
		}
	}
	if err := s.store.SetMetadata(ctx, fmt.Sprintf("%s/%d/d", store.HeightToDAHeightKey, height), dataHeightBytes); err != nil {
		return err
	}

	if genesisInclusion {
		genesisDAIncludedHeightBytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(genesisDAIncludedHeightBytes, genesisDAIncludedHeight)

		if err := s.store.SetMetadata(ctx, store.GenesisDAHeightKey, genesisDAIncludedHeightBytes); err != nil {
			return err
		}
	}

	return nil
}

// IsHeightDAIncluded checks if a height is included in DA
func (s *Submitter) IsHeightDAIncluded(height uint64, header *types.SignedHeader, data *types.Data) (bool, error) {
	currentHeight, err := s.store.Height(s.ctx)
	if err != nil {
		return false, err
	}
	if currentHeight < height {
		return false, nil
	}

	headerHash := header.Hash().String()
	dataHash := data.DACommitment().String()

	_, headerIncluded := s.cache.GetHeaderDAIncluded(headerHash)
	_, dataIncluded := s.cache.GetDataDAIncluded(dataHash)

	dataIncluded = bytes.Equal(data.DACommitment(), common.DataHashForEmptyTxs) || dataIncluded

	return headerIncluded && dataIncluded, nil
}
