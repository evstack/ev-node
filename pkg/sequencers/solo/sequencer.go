package solo

import (
	"bytes"
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/core/execution"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/sequencers/common"
)

var (
	ErrInvalidID = common.ErrInvalidID
	ErrQueueFull = common.ErrQueueFull
)

var (
	emptyBatch        = &coresequencer.Batch{}
	submitBatchResp   = &coresequencer.SubmitBatchTxsResponse{}
	verifyBatchOKResp = &coresequencer.VerifyBatchResponse{Status: true}
)

var _ coresequencer.Sequencer = (*SoloSequencer)(nil)

// Option configures a SoloSequencer.
type Option func(*SoloSequencer)

// WithMaxQueueBytes sets a soft cap on the sequencer's in-memory tx queue.
// SubmitBatchTxs admits txs while the cap has room and returns ErrQueueFull
// when the incoming batch would exceed it. A zero value (default) disables
// the cap.
func WithMaxQueueBytes(n uint64) Option {
	return func(s *SoloSequencer) {
		s.maxQueueBytes = n
	}
}

// SoloSequencer is a single-leader sequencer without forced inclusion
// support. It maintains a simple in-memory queue of mempool transactions and
// produces batches on demand.
type SoloSequencer struct {
	logger   zerolog.Logger
	id       []byte
	executor execution.Executor

	daHeight atomic.Uint64

	mu            sync.Mutex
	queue         [][]byte
	queueBytes    uint64
	maxQueueBytes uint64
}

func NewSoloSequencer(
	logger zerolog.Logger,
	id []byte,
	executor execution.Executor,
	opts ...Option,
) *SoloSequencer {
	if executor == nil {
		panic("solo: executor must not be nil")
	}

	s := &SoloSequencer{
		logger:   logger,
		id:       id,
		executor: executor,
		queue:    make([][]byte, 0),
	}

	for _, opt := range opts {
		opt(s)
	}

	logger.Debug().
		Uint64("max_queue_bytes", s.maxQueueBytes).
		Msg("solo sequencer initialized")

	return s
}

func (s *SoloSequencer) isValid(id []byte) bool {
	return bytes.Equal(s.id, id)
}

func (s *SoloSequencer) SubmitBatchTxs(ctx context.Context, req coresequencer.SubmitBatchTxsRequest) (*coresequencer.SubmitBatchTxsResponse, error) {
	if !s.isValid(req.Id) {
		return nil, ErrInvalidID
	}

	if req.Batch == nil || len(req.Batch.Transactions) == 0 {
		return submitBatchResp, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.maxQueueBytes == 0 {
		s.queue = append(s.queue, req.Batch.Transactions...)
		return submitBatchResp, nil
	}

	// All-or-nothing: if the whole incoming batch doesn't fit, reject
	// it untouched. Partial admission would force the caller (e.g.
	// the reaper bridging executor mempool → sequencer) to reason
	// about which prefix was admitted and re-feed only the suffix on
	// retry, which it doesn't currently do — leading to duplicate-tx
	// resubmission on each retry. Rejecting the whole batch lets the
	// reaper just retry with the same batch later when the queue has
	// drained.
	var batchBytes uint64
	for _, tx := range req.Batch.Transactions {
		batchBytes += uint64(len(tx))
	}

	if s.queueBytes+batchBytes > s.maxQueueBytes {
		return submitBatchResp, ErrQueueFull
	}

	s.queue = append(s.queue, req.Batch.Transactions...)
	s.queueBytes += batchBytes

	return submitBatchResp, nil
}

func (s *SoloSequencer) GetNextBatch(ctx context.Context, req coresequencer.GetNextBatchRequest) (*coresequencer.GetNextBatchResponse, error) {
	if !s.isValid(req.Id) {
		return nil, ErrInvalidID
	}

	s.mu.Lock()
	txs := s.queue
	s.queue = nil
	s.queueBytes = 0
	s.mu.Unlock()

	if len(txs) == 0 {
		return &coresequencer.GetNextBatchResponse{
			Batch:     emptyBatch,
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
		return &coresequencer.GetNextBatchResponse{
			Batch:     &coresequencer.Batch{Transactions: txs},
			Timestamp: time.Now().UTC(),
			BatchData: req.LastBatchData,
		}, nil
	}

	write := 0
	var postponedTxs [][]byte
	for i, status := range filterStatuses {
		switch status {
		case execution.FilterOK:
			txs[write] = txs[i]
			write++
		case execution.FilterPostpone:
			postponedTxs = append(postponedTxs, txs[i])
		}
	}

	if len(postponedTxs) > 0 {
		s.mu.Lock()
		s.queue = append(postponedTxs, s.queue...)
		// Postponed txs were already in the queue's byte count when
		// SubmitBatchTxs admitted them. We zeroed queueBytes on drain
		// above, so re-queuing requires re-counting whatever survived.
		var bytes uint64
		for _, tx := range postponedTxs {
			bytes += uint64(len(tx))
		}
		s.queueBytes += bytes
		s.mu.Unlock()
	}

	return &coresequencer.GetNextBatchResponse{
		Batch:     &coresequencer.Batch{Transactions: txs[:write]},
		Timestamp: time.Now().UTC(),
		BatchData: req.LastBatchData,
	}, nil
}

func (s *SoloSequencer) VerifyBatch(ctx context.Context, req coresequencer.VerifyBatchRequest) (*coresequencer.VerifyBatchResponse, error) {
	if !s.isValid(req.Id) {
		return nil, ErrInvalidID
	}

	return verifyBatchOKResp, nil
}

func (s *SoloSequencer) SetDAHeight(height uint64) {
	s.daHeight.Store(height)
}

func (s *SoloSequencer) GetDAHeight() uint64 {
	return s.daHeight.Load()
}
