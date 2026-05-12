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

type DASubmitterAPI interface {
	SubmitBlocks(ctx context.Context, headers []*types.SignedHeader, data []*types.Data, cache cache.Manager, signer signer.Signer, onSubmitError func(error)) error
	Close()
}

type Submitter struct {
	store     store.Store
	exec      coreexecutor.Executor
	sequencer coresequencer.Sequencer
	config    config.Config
	genesis   genesis.Genesis

	cache   cache.Manager
	metrics *common.Metrics

	daSubmitter DASubmitterAPI

	signer signer.Signer

	daIncludedHeight *atomic.Uint64

	submissionMtx sync.Mutex

	lastSubmit        atomic.Int64
	batchingStrategy BatchingStrategy

	errorCh chan<- error

	logger zerolog.Logger

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewSubmitter(
	store store.Store,
	exec coreexecutor.Executor,
	cache cache.Manager,
	metrics *common.Metrics,
	config config.Config,
	genesis genesis.Genesis,
	daSubmitter DASubmitterAPI,
	sequencer coresequencer.Sequencer,
	signer signer.Signer,
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
	submitter.lastSubmit.Store(now)

	return submitter
}

func (s *Submitter) Start(ctx context.Context) (err error) {
	if s.cancel != nil {
		return errors.New("submitter already started")
	}

	s.ctx, s.cancel = context.WithCancel(ctx)
	defer func() {
		if err != nil {
			s.cancel()
			s.ctx, s.cancel = nil, nil
		}
	}()

	if err = s.initializeDAIncludedHeight(ctx); err != nil {
		return err
	}

	if s.signer != nil {
		s.logger.Info().Msg("starting DA submission loop")
		s.wg.Go(s.daSubmissionLoop)
	}

	s.wg.Go(s.processDAInclusionLoop)

	return nil
}

func (s *Submitter) Stop() error {
	if s.cancel != nil {
		s.cancel()
	}
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		s.daSubmitter.Close()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		s.logger.Warn().Msg("submitter shutdown timed out waiting for goroutines, proceeding anyway")
		s.daSubmitter.Close()
	}
	s.logger.Info().Msg("submitter stopped")
	return nil
}

func (s *Submitter) daSubmissionLoop() {
	s.logger.Info().Msg("starting DA submission loop")
	defer s.logger.Info().Msg("DA submission loop stopped")

	checkInterval := max(s.config.DA.BlockTime.Duration/4, 100*time.Millisecond)

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			pendingHeaders := s.cache.NumPendingHeaders()
			if pendingHeaders > 0 {
				lastSubmitNanos := s.lastSubmit.Load()
				timeSinceLastSubmit := time.Since(time.Unix(0, lastSubmitNanos))

				if s.submissionMtx.TryLock() {
					s.wg.Add(1)
					go func() {
						defer func() {
							s.submissionMtx.Unlock()
							s.wg.Done()
						}()

						headers, marshalledHeaders, err := s.cache.GetPendingHeaders(s.ctx)
						if err != nil {
							if len(headers) > 0 {
								s.cache.ResetInFlightHeaderRange(headers[0].Height(), headers[len(headers)-1].Height())
							}
							s.logger.Error().Err(err).Msg("failed to get pending headers")
							return
						}
						if len(headers) == 0 {
							return
						}

						totalSize := uint64(0)
						for _, marshalled := range marshalledHeaders {
							totalSize += uint64(len(marshalled))
						}

						shouldSubmit := s.batchingStrategy.ShouldSubmit(
							uint64(len(headers)),
							totalSize,
							common.DefaultMaxBlobSize,
							timeSinceLastSubmit,
						)

						if !shouldSubmit {
							if len(headers) > 0 {
								s.cache.ResetInFlightHeaderRange(headers[0].Height(), headers[len(headers)-1].Height())
							}
							return
						}

						dataList := make([]*types.Data, len(headers))
						for i, hdr := range headers {
							_, data, fetchErr := s.store.GetBlockData(s.ctx, hdr.Height())
							if fetchErr != nil {
								s.cache.ResetInFlightHeaderRange(headers[0].Height(), headers[len(headers)-1].Height())
								s.logger.Error().Err(fetchErr).Msg("failed to get block data for pending header")
								return
							}
							dataList[i] = data
						}

						s.lastSubmit.Store(time.Now().UnixNano())
						onError := func(err error) {
							if errors.Is(err, common.ErrOversizedItem) {
								s.logger.Error().Err(err).
									Msg("CRITICAL: Block exceeds DA blob size limit - halting to prevent live lock")
								s.sendCriticalError(fmt.Errorf("unrecoverable DA submission error: %w", err))
								return
							}
							if len(headers) > 0 {
								s.cache.ResetInFlightHeaderRange(headers[0].Height(), headers[len(headers)-1].Height())
							}
							if err != nil {
								s.logger.Error().Err(err).Msg("failed to submit blocks")
							}
						}
						if err := s.daSubmitter.SubmitBlocks(s.ctx, headers, dataList, s.cache, s.signer, onError); err != nil {
							if len(headers) > 0 {
								s.cache.ResetInFlightHeaderRange(headers[0].Height(), headers[len(headers)-1].Height())
							}
							s.logger.Error().Err(err).Msg("failed to enqueue block submission")
						}
					}()
				}
			}

			s.metrics.DASubmitterPendingBlobs.Set(float64(pendingHeaders))
		}
	}
}

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
			s.metrics.DAInclusionHeight.Set(float64(currentDAIncluded))

			for {
				nextHeight := currentDAIncluded + 1

				_, data, err := s.store.GetBlockData(s.ctx, nextHeight)
				if err != nil {
					break
				}

				if included, err := s.IsHeightDAIncluded(nextHeight, data); err != nil || !included {
					break
				}

				s.logger.Debug().Uint64("height", nextHeight).Msg("advancing DA included height")

				if err := s.setNodeHeightToDAHeight(s.ctx, nextHeight, data, currentDAIncluded == 0); err != nil {
					s.logger.Error().Err(err).Uint64("height", nextHeight).Msg("failed to set node height to DA height mapping")
					break
				}

				if err := s.setFinalWithRetry(nextHeight); err != nil {
					s.sendCriticalError(fmt.Errorf("failed to set final height: %w", err))
					s.logger.Error().Err(err).Uint64("height", nextHeight).Msg("CRITICAL: Failed to set final height after retries - halting DA inclusion processing")
					return
				}

				if err := putUint64Metadata(s.ctx, s.store, store.DAIncludedHeightKey, nextHeight); err != nil {
					s.logger.Error().Err(err).Uint64("height", nextHeight).Msg("failed to persist DA included height")
					break
				}

				s.SetDAIncludedHeight(nextHeight)
				currentDAIncluded = nextHeight

				s.cache.DeleteHeight(nextHeight)
			}
		}
	}
}

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

func (s *Submitter) GetDAIncludedHeight() uint64 {
	return s.daIncludedHeight.Load()
}

func (s *Submitter) SetDAIncludedHeight(height uint64) {
	s.daIncludedHeight.Store(height)
}

func (s *Submitter) initializeDAIncludedHeight(ctx context.Context) error {
	if height, err := s.store.GetMetadata(ctx, store.DAIncludedHeightKey); err == nil && len(height) == 8 {
		s.SetDAIncludedHeight(binary.LittleEndian.Uint64(height))
	}
	return nil
}

func putUint64Metadata(ctx context.Context, st store.Store, key string, val uint64) error {
	bz := make([]byte, 8)
	binary.LittleEndian.PutUint64(bz, val)
	return st.SetMetadata(ctx, key, bz)
}

func (s *Submitter) sendCriticalError(err error) {
	if s.cancel != nil {
		s.cancel()
	}
	if s.errorCh != nil {
		select {
		case s.errorCh <- err:
		default:
		}
	}
}

func (s *Submitter) setNodeHeightToDAHeight(ctx context.Context, height uint64, data *types.Data, genesisInclusion bool) error {
	daHeightForHeader, ok := s.cache.GetHeaderDAIncludedByHeight(height)
	if !ok {
		return fmt.Errorf("header for height %d not found in cache", height)
	}

	if err := putUint64Metadata(ctx, s.store, store.GetHeightToDAHeightHeaderKey(height), daHeightForHeader); err != nil {
		return err
	}

	genesisDAIncludedHeight := daHeightForHeader
	dataDAHeight := daHeightForHeader
	dataHash := data.DACommitment()
	if !bytes.Equal(dataHash, common.DataHashForEmptyTxs) {
		daHeightForData, ok := s.cache.GetDataDAIncludedByHeight(height)
		if !ok {
			return fmt.Errorf("data for height %d not found in cache", height)
		}
		dataDAHeight = daHeightForData
		genesisDAIncludedHeight = min(daHeightForData, genesisDAIncludedHeight)
	}
	if err := putUint64Metadata(ctx, s.store, store.GetHeightToDAHeightDataKey(height), dataDAHeight); err != nil {
		return err
	}

	if genesisInclusion {
		if err := putUint64Metadata(ctx, s.store, store.GenesisDAHeightKey, genesisDAIncludedHeight); err != nil {
			return err
		}

		if s.sequencer != nil {
			s.sequencer.SetDAHeight(genesisDAIncludedHeight)
			s.logger.Debug().Uint64("genesis_da_height", genesisDAIncludedHeight).Msg("initialized sequencer DA height from persisted genesis DA height")
		}
	}

	return nil
}

func (s *Submitter) IsHeightDAIncluded(height uint64, data *types.Data) (bool, error) {
	if height <= s.GetDAIncludedHeight() {
		return true, nil
	}

	currentHeight, err := s.store.Height(s.ctx)
	if err != nil {
		return false, err
	}

	if currentHeight < height {
		return false, nil
	}

	_, headerIncluded := s.cache.GetHeaderDAIncludedByHeight(height)
	_, dataIncluded := s.cache.GetDataDAIncludedByHeight(height)

	dataCommitment := data.DACommitment()
	dataIncluded = bytes.Equal(dataCommitment, common.DataHashForEmptyTxs) || dataIncluded

	return headerIncluded && dataIncluded, nil
}
