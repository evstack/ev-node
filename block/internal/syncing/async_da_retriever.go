package syncing

import (
	"context"
	"sync"

	"github.com/evstack/ev-node/block/internal/common"
	"github.com/rs/zerolog"
)

// AsyncDARetriever handles concurrent DA retrieval operations.
type AsyncDARetriever struct {
	retriever DARetriever
	resultCh  chan<- common.DAHeightEvent
	workCh    chan uint64
	inFlight  map[uint64]struct{}
	mu        sync.Mutex
	logger    zerolog.Logger
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewAsyncDARetriever creates a new AsyncDARetriever.
func NewAsyncDARetriever(
	retriever DARetriever,
	resultCh chan<- common.DAHeightEvent,
	logger zerolog.Logger,
) *AsyncDARetriever {
	return &AsyncDARetriever{
		retriever: retriever,
		resultCh:  resultCh,
		workCh:    make(chan uint64, 100), // Buffer size 100
		inFlight:  make(map[uint64]struct{}),
		logger:    logger.With().Str("component", "async_da_retriever").Logger(),
	}
}

// Start starts the worker pool.
func (r *AsyncDARetriever) Start(ctx context.Context) {
	r.ctx, r.cancel = context.WithCancel(ctx)
	// Start 5 workers
	for i := 0; i < 5; i++ {
		r.wg.Add(1)
		go r.worker()
	}
	r.logger.Info().Msg("AsyncDARetriever started")
}

// Stop stops the worker pool.
func (r *AsyncDARetriever) Stop() {
	if r.cancel != nil {
		r.cancel()
	}
	r.wg.Wait()
	r.logger.Info().Msg("AsyncDARetriever stopped")
}

// RequestRetrieval requests a DA retrieval for the given height.
// It is non-blocking and idempotent.
func (r *AsyncDARetriever) RequestRetrieval(height uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.inFlight[height]; exists {
		return
	}

	select {
	case r.workCh <- height:
		r.inFlight[height] = struct{}{}
		r.logger.Debug().Uint64("height", height).Msg("queued DA retrieval request")
	default:
		r.logger.Debug().Uint64("height", height).Msg("DA retrieval worker pool full, dropping request")
	}
}

func (r *AsyncDARetriever) worker() {
	defer r.wg.Done()

	for {
		select {
		case <-r.ctx.Done():
			return
		case height := <-r.workCh:
			r.processRetrieval(height)
		}
	}
}

func (r *AsyncDARetriever) processRetrieval(height uint64) {
	defer func() {
		r.mu.Lock()
		delete(r.inFlight, height)
		r.mu.Unlock()
	}()

	events, err := r.retriever.RetrieveFromDA(r.ctx, height)
	if err != nil {
		r.logger.Debug().Err(err).Uint64("height", height).Msg("async DA retrieval failed")
		return
	}

	for _, event := range events {
		select {
		case r.resultCh <- event:
		case <-r.ctx.Done():
			return
		}
	}
}
