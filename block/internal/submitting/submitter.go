package submitting

import (
	"bytes"
	"context"
	"encoding/binary"
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
)

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
	daSubmitter *DASubmitter

	// Optional signer (only for aggregator nodes)
	signer signer.Signer

	// DA state
	daIncludedHeight uint64
	daStateMtx       *sync.RWMutex

	// Submission state to prevent concurrent submissions
	headerSubmissionMtx sync.Mutex
	dataSubmissionMtx   sync.Mutex

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
	daSubmitter *DASubmitter,
	signer signer.Signer, // Can be nil for sync nodes
	logger zerolog.Logger,
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
			s.processDAInclusion()
		}
	}
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

// processDAInclusion processes DA inclusion tracking
func (s *Submitter) processDAInclusion() {
	currentDAIncluded := s.GetDAIncludedHeight()

	for {
		nextHeight := currentDAIncluded + 1

		// Check if this height is DA included
		if included, err := s.isHeightDAIncluded(nextHeight); err != nil || !included {
			break
		}

		s.logger.Debug().Uint64("height", nextHeight).Msg("advancing DA included height")

		// Set final height in executor
		if err := s.exec.SetFinal(s.ctx, nextHeight); err != nil {
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

// isHeightDAIncluded checks if a height is included in DA
func (s *Submitter) isHeightDAIncluded(height uint64) (bool, error) {
	currentHeight, err := s.store.Height(s.ctx)
	if err != nil {
		return false, err
	}
	if currentHeight < height {
		return false, nil
	}

	header, data, err := s.store.GetBlockData(s.ctx, height)
	if err != nil {
		return false, err
	}

	headerHash := header.Hash().String()
	dataHash := data.DACommitment().String()

	headerIncluded := s.cache.IsHeaderDAIncluded(headerHash)
	dataIncluded := bytes.Equal(data.DACommitment(), common.DataHashForEmptyTxs) || s.cache.IsDataDAIncluded(dataHash)

	return headerIncluded && dataIncluded, nil
}
