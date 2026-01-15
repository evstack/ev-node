package submitting

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	coreexecutor "github.com/evstack/ev-node/core/execution"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

// daSubmitterAPI defines minimal methods needed by Submitter for DA submissions.
type daSubmitterAPI interface {
	SubmitHeaders(ctx context.Context, headers []*types.SignedHeader, marshalledHeaders [][]byte, cache cache.Manager, signer signer.Signer) error
	SubmitData(ctx context.Context, signedDataList []*types.SignedData, marshalledData [][]byte, cache cache.Manager, signer signer.Signer, genesis genesis.Genesis) error
}

// Submitter handles DA submission and inclusion processing for both sync and aggregator nodes
type Submitter struct {
	// Core components
	store     store.Store
	exec      coreexecutor.Executor
	sequencer coresequencer.Sequencer
	config    config.Config
	genesis   genesis.Genesis

	// Shared components
	cache   cache.Manager
	metrics *common.Metrics

	// DA submitter
	daSubmitter daSubmitterAPI

	// Optional signer (only for aggregator nodes)
	signer signer.Signer

	// DA state
	daIncludedHeight *atomic.Uint64

	// Submission state to prevent concurrent submissions
	headerSubmissionMtx sync.Mutex
	dataSubmissionMtx   sync.Mutex

	// Batching strategy state
	lastHeaderSubmit atomic.Int64 // stores Unix nanoseconds
	lastDataSubmit   atomic.Int64 // stores Unix nanoseconds
	batchingStrategy BatchingStrategy

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
	sequencer coresequencer.Sequencer, // Can be nil for sync nodes
	signer signer.Signer, // Can be nil for sync nodes
	logger zerolog.Logger,
	errorCh chan<- error,
) *Submitter {
	submitterLogger := logger.With().Str("component", "submitter").Logger()

	strategy, err := NewBatchingStrategy(config.DA)
	if err != nil {
		submitterLogger.Warn().Err(err).Msg("failed to create batching strategy, using time-based default")
		strategy = NewTimeBasedStrategy(config.DA.BlockTime.Duration, 0, 1)
	}

	submitter := &Submitter{
		store:            store,
		exec:             exec,
		cache:            cache,
		metrics:          metrics,
		config:           config,
		genesis:          genesis,
		daSubmitter:      daSubmitter,
		sequencer:        sequencer,
		signer:           signer,
		daIncludedHeight: &atomic.Uint64{},
		batchingStrategy: strategy,
		errorCh:          errorCh,
		logger:           submitterLogger,
	}

	now := time.Now().UnixNano()
	submitter.lastHeaderSubmit.Store(now)
	submitter.lastDataSubmit.Store(now)

	return submitter
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

// daSubmissionLoop handles submission of headers and data to DA layer (aggregator nodes only)
func (s *Submitter) daSubmissionLoop() {
	s.logger.Info().Msg("starting DA submission loop")
	defer s.logger.Info().Msg("DA submission loop stopped")

	// Use a shorter ticker interval to check batching strategy more frequently
	checkInterval := min(s.config.DA.BlockTime.Duration/4, 100*time.Millisecond)

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			// Check if we should submit headers based on batching strategy
			headersNb := s.cache.NumPendingHeaders()
			if headersNb > 0 {
				lastSubmitNanos := s.lastHeaderSubmit.Load()
				timeSinceLastSubmit := time.Since(time.Unix(0, lastSubmitNanos))

				// For strategy decision, we need to estimate the size
				// We'll fetch headers to check, but only submit if strategy approves
				if s.headerSubmissionMtx.TryLock() {
					go func() {
						defer s.headerSubmissionMtx.Unlock()

						// Get headers with marshalled bytes from cache
						headers, marshalledHeaders, err := s.cache.GetPendingHeaders(s.ctx)
						if err != nil {
							s.logger.Error().Err(err).Msg("failed to get pending headers for batching decision")
							return
						}

						// Calculate total size (excluding signature)
						totalSize := 0
						for _, marshalled := range marshalledHeaders {
							totalSize += len(marshalled)
						}

						shouldSubmit := s.batchingStrategy.ShouldSubmit(
							uint64(len(headers)),
							totalSize,
							common.DefaultMaxBlobSize,
							timeSinceLastSubmit,
						)

						if shouldSubmit {
							s.logger.Debug().
								Time("t", time.Now()).
								Uint64("headers", headersNb).
								Int("total_size_kb", totalSize/1024).
								Dur("time_since_last", timeSinceLastSubmit).
								Msg("batching strategy triggered header submission")

							if err := s.daSubmitter.SubmitHeaders(s.ctx, headers, marshalledHeaders, s.cache, s.signer); err != nil {
								// Check for unrecoverable errors that indicate a critical issue
								if errors.Is(err, common.ErrOversizedItem) {
									s.logger.Error().Err(err).
										Msg("CRITICAL: Header exceeds DA blob size limit - halting to prevent live lock")
									s.sendCriticalError(fmt.Errorf("unrecoverable DA submission error: %w", err))
									return
								}
								s.logger.Error().Err(err).Msg("failed to submit headers")
							} else {
								s.lastHeaderSubmit.Store(time.Now().UnixNano())
							}
						}
					}()
				}
			}

			// Check if we should submit data based on batching strategy
			dataNb := s.cache.NumPendingData()
			if dataNb > 0 {
				lastSubmitNanos := s.lastDataSubmit.Load()
				timeSinceLastSubmit := time.Since(time.Unix(0, lastSubmitNanos))

				if s.dataSubmissionMtx.TryLock() {
					go func() {
						defer s.dataSubmissionMtx.Unlock()

						// Get data with marshalled bytes from cache
						signedDataList, marshalledData, err := s.cache.GetPendingData(s.ctx)
						if err != nil {
							s.logger.Error().Err(err).Msg("failed to get pending data for batching decision")
							return
						}

						// Calculate total size (excluding signature)
						totalSize := 0
						for _, marshalled := range marshalledData {
							totalSize += len(marshalled)
						}

						shouldSubmit := s.batchingStrategy.ShouldSubmit(
							uint64(len(signedDataList)),
							totalSize,
							common.DefaultMaxBlobSize,
							timeSinceLastSubmit,
						)

						if shouldSubmit {
							s.logger.Debug().
								Time("t", time.Now()).
								Uint64("data", dataNb).
								Int("total_size_kb", totalSize/1024).
								Dur("time_since_last", timeSinceLastSubmit).
								Msg("batching strategy triggered data submission")

							if err := s.daSubmitter.SubmitData(s.ctx, signedDataList, marshalledData, s.cache, s.signer, s.genesis); err != nil {
								// Check for unrecoverable errors that indicate a critical issue
								if errors.Is(err, common.ErrOversizedItem) {
									s.logger.Error().Err(err).
										Msg("CRITICAL: Data exceeds DA blob size limit - halting to prevent live lock")
									s.sendCriticalError(fmt.Errorf("unrecoverable DA submission error: %w", err))
									return
								}
								s.logger.Error().Err(err).Msg("failed to submit data")
							} else {
								s.lastDataSubmit.Store(time.Now().UnixNano())
							}
						}
					}()
				}
			}
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
			s.metrics.DAInclusionHeight.Set(float64(s.GetDAIncludedHeight()))

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
					s.logger.Error().Err(err).Uint64("height", nextHeight).Msg("CRITICAL: Failed to set final height after retries - halting DA inclusion processing")
					return
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

				// Delete height cache for that height
				// This can only be performed after the height has been persisted to store
				s.cache.DeleteHeight(nextHeight)
			}
		}
	}
}

// setFinalWithRetry sets the final height in executor with retry logic.
// NOTE: the function retries the execution client call regardless of the error. Some execution client errors are irrecoverable, and will eventually halt the node, as expected.
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
	return s.daIncludedHeight.Load()
}

// SetDAIncludedHeight updates the DA included height
func (s *Submitter) SetDAIncludedHeight(height uint64) {
	s.daIncludedHeight.Store(height)
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

	headerDaHeightBytes := make([]byte, 8)
	daHeightForHeader, ok := s.cache.GetHeaderDAIncluded(headerHash.String())
	if !ok {
		return fmt.Errorf("header hash %s not found in cache", headerHash)
	}
	binary.LittleEndian.PutUint64(headerDaHeightBytes, daHeightForHeader)

	if err := s.store.SetMetadata(ctx, store.GetHeightToDAHeightHeaderKey(height), headerDaHeightBytes); err != nil {
		return err
	}

	genesisDAIncludedHeight := daHeightForHeader
	dataDaHeightBytes := make([]byte, 8)
	// For empty transactions, use the same DA height as the header
	if bytes.Equal(dataHash, common.DataHashForEmptyTxs) {
		binary.LittleEndian.PutUint64(dataDaHeightBytes, daHeightForHeader)
	} else {
		daHeightForData, ok := s.cache.GetDataDAIncluded(dataHash.String())
		if !ok {
			return fmt.Errorf("data hash %s not found in cache", dataHash.String())
		}
		binary.LittleEndian.PutUint64(dataDaHeightBytes, daHeightForData)

		// if data posted before header, use data da included height for genesis da height
		genesisDAIncludedHeight = min(daHeightForData, genesisDAIncludedHeight)
	}
	if err := s.store.SetMetadata(ctx, store.GetHeightToDAHeightDataKey(height), dataDaHeightBytes); err != nil {
		return err
	}

	if genesisInclusion {
		genesisDAIncludedHeightBytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(genesisDAIncludedHeightBytes, genesisDAIncludedHeight)

		if err := s.store.SetMetadata(ctx, store.GenesisDAHeightKey, genesisDAIncludedHeightBytes); err != nil {
			return err
		}

		// the sequencer will process DA epochs from this height.
		if s.sequencer != nil {
			s.sequencer.SetDAHeight(genesisDAIncludedHeight)
			s.logger.Debug().Uint64("genesis_da_height", genesisDAIncludedHeight).Msg("initialized sequencer DA height from persisted genesis DA height")
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
	dataCommitment := data.DACommitment()
	dataHash := dataCommitment.String()

	_, headerIncluded := s.cache.GetHeaderDAIncluded(headerHash)
	_, dataIncluded := s.cache.GetDataDAIncluded(dataHash)

	dataIncluded = bytes.Equal(dataCommitment, common.DataHashForEmptyTxs) || dataIncluded

	return headerIncluded && dataIncluded, nil
}
