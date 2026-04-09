package solo

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/core/execution"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/config"
)

var ErrInvalidID = errors.New("invalid chain id")

var _ coresequencer.Sequencer = (*SoloSequencer)(nil)

// SoloSequencer is a single-leader sequencer without forced inclusion
// support. It maintains a simple in-memory queue of mempool transactions and
// produces batches on demand.
type SoloSequencer struct {
	logger   zerolog.Logger
	id       []byte
	executor execution.Executor

	daHeight atomic.Uint64

	mu    sync.Mutex
	queue [][]byte
}

func NewSoloSequencer(
	logger zerolog.Logger,
	cfg config.Config,
	id []byte,
	executor execution.Executor,
) *SoloSequencer {
	return &SoloSequencer{
		logger:   logger,
		id:       id,
		executor: executor,
		queue:    make([][]byte, 0),
	}
}

func (s *SoloSequencer) isValid(id []byte) bool {
	return bytes.Equal(s.id, id)
}

func (s *SoloSequencer) SubmitBatchTxs(ctx context.Context, req coresequencer.SubmitBatchTxsRequest) (*coresequencer.SubmitBatchTxsResponse, error) {
	if !s.isValid(req.Id) {
		return nil, ErrInvalidID
	}

	if req.Batch == nil || len(req.Batch.Transactions) == 0 {
		return &coresequencer.SubmitBatchTxsResponse{}, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.queue = append(s.queue, req.Batch.Transactions...)
	return &coresequencer.SubmitBatchTxsResponse{}, nil
}

func (s *SoloSequencer) GetNextBatch(ctx context.Context, req coresequencer.GetNextBatchRequest) (*coresequencer.GetNextBatchResponse, error) {
	if !s.isValid(req.Id) {
		return nil, ErrInvalidID
	}

	s.mu.Lock()
	txs := s.queue
	s.queue = nil
	s.mu.Unlock()

	if len(txs) == 0 {
		return &coresequencer.GetNextBatchResponse{
			Batch:     &coresequencer.Batch{},
			Timestamp: time.Now().UTC(),
			BatchData: req.LastBatchData,
		}, nil
	}

	var maxGas uint64
	info, err := s.executor.GetExecutionInfo(ctx)
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to get execution info")
	} else {
		maxGas = info.MaxGas
	}

	filterStatuses, err := s.executor.FilterTxs(ctx, txs, req.MaxBytes, maxGas, false)
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to filter transactions, proceeding with unfiltered")
		filterStatuses = make([]execution.FilterStatus, len(txs))
		for i := range filterStatuses {
			filterStatuses[i] = execution.FilterOK
		}
	}

	var validTxs [][]byte
	var postponedTxs [][]byte
	for i, status := range filterStatuses {
		switch status {
		case execution.FilterOK:
			validTxs = append(validTxs, txs[i])
		case execution.FilterPostpone:
			postponedTxs = append(postponedTxs, txs[i])
		case execution.FilterRemove:
		}
	}

	if len(postponedTxs) > 0 {
		s.mu.Lock()
		s.queue = append(postponedTxs, s.queue...)
		s.mu.Unlock()
	}

	return &coresequencer.GetNextBatchResponse{
		Batch:     &coresequencer.Batch{Transactions: validTxs},
		Timestamp: time.Now().UTC(),
		BatchData: req.LastBatchData,
	}, nil
}

func (s *SoloSequencer) VerifyBatch(ctx context.Context, req coresequencer.VerifyBatchRequest) (*coresequencer.VerifyBatchResponse, error) {
	return &coresequencer.VerifyBatchResponse{Status: true}, nil
}

func (s *SoloSequencer) SetDAHeight(height uint64) {
	s.daHeight.Store(height)
}

func (s *SoloSequencer) GetDAHeight() uint64 {
	return s.daHeight.Load()
}
