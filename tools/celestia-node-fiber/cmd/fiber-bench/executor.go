package main

import (
	"context"
	"sync/atomic"
	"time"

	coreexecution "github.com/evstack/ev-node/core/execution"
)

// inMemExecutor is a minimal core.Executor that:
//   - accepts injected txs via a buffered channel (the "mempool")
//   - drains them in GetTxs (non-blocking)
//   - "executes" by counting (no state machine)
//   - returns a constant state root, so we don't pay O(N) state-root cost on
//     every block (which would dominate the measurement and tell us nothing
//     about ev-node's batching/submitting performance).
//
// Use FilterTxs's size cap to enforce the configured per-block byte budget.
type inMemExecutor struct {
	txCh chan []byte

	injected         atomic.Uint64
	dropped          atomic.Uint64
	blocksProduced   atomic.Uint64
	totalExecutedTxs atomic.Uint64

	// mempoolHigh tracks the maximum mempool depth observed (snapshot).
	mempoolHigh atomic.Int64

	// constStateRoot is what every block reports as its post-state. The
	// measurement target is ev-node, not state computation.
	constStateRoot []byte
}

func newInMemExecutor(mempoolSize int) *inMemExecutor {
	return &inMemExecutor{
		txCh:           make(chan []byte, mempoolSize),
		constStateRoot: []byte("fiber-bench-const-state-root"),
	}
}

// InjectTx is the bench's "mempool entry". Backpressures via channel
// capacity: full → drop and increment counter so the operator sees it.
func (e *inMemExecutor) InjectTx(tx []byte) bool {
	select {
	case e.txCh <- tx:
		e.injected.Add(1)
		// Loose mempool-depth high-water; not a hot-path concern.
		if d := int64(len(e.txCh)); d > e.mempoolHigh.Load() {
			e.mempoolHigh.Store(d)
		}
		return true
	default:
		e.dropped.Add(1)
		return false
	}
}

func (e *inMemExecutor) MempoolDepth() int { return len(e.txCh) }

func (e *inMemExecutor) Stats() (injected, dropped, blocks, txs uint64, mempoolHigh int64) {
	return e.injected.Load(),
		e.dropped.Load(),
		e.blocksProduced.Load(),
		e.totalExecutedTxs.Load(),
		e.mempoolHigh.Load()
}

// InitChain is called once at genesis.
func (e *inMemExecutor) InitChain(_ context.Context, _ time.Time, _ uint64, _ string) ([]byte, error) {
	return e.constStateRoot, nil
}

// GetTxs drains the mempool channel. Non-blocking — returns whatever is
// currently buffered. ev-node's reaper polls this on its own cadence.
func (e *inMemExecutor) GetTxs(ctx context.Context) ([][]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	n := len(e.txCh)
	if n == 0 {
		return nil, nil
	}
	txs := make([][]byte, 0, n)
	for i := 0; i < n; i++ {
		select {
		case tx := <-e.txCh:
			txs = append(txs, tx)
		default:
			return txs, nil
		}
	}
	return txs, nil
}

// ExecuteTxs is intentionally a no-op state transition: count txs, return
// a constant root. The whole point of this executor is to take state
// computation out of the measurement.
func (e *inMemExecutor) ExecuteTxs(_ context.Context, txs [][]byte, _ uint64, _ time.Time, _ []byte) ([]byte, error) {
	e.blocksProduced.Add(1)
	e.totalExecutedTxs.Add(uint64(len(txs)))
	return e.constStateRoot, nil
}

func (e *inMemExecutor) SetFinal(_ context.Context, _ uint64) error { return nil }
func (e *inMemExecutor) Rollback(_ context.Context, _ uint64) error { return nil }

func (e *inMemExecutor) GetExecutionInfo(_ context.Context) (coreexecution.ExecutionInfo, error) {
	// MaxGas=0 means "no gas-based filter"; the size cap (FilterTxs) is what
	// bounds per-block bytes.
	return coreexecution.ExecutionInfo{MaxGas: 0}, nil
}

// FilterTxs enforces the configured per-block byte budget. Mirrors the
// existing testapp KV executor's behavior: oversized txs are dropped, the
// rest fill until the budget is hit and overflow is postponed for the
// next block. We don't validate tx content — txs from the load generator
// are well-formed by construction.
//
// We honor maxBytes as-is. Per-block proto/Metadata overhead is the
// responsibility of the block-size cap (now anchored to Fibre's actual
// MaxPayload in block/internal/common/consts.go), not the executor.
func (e *inMemExecutor) FilterTxs(_ context.Context, txs [][]byte, maxBytes, _ uint64, _ bool) ([]coreexecution.FilterStatus, error) {
	out := make([]coreexecution.FilterStatus, len(txs))
	var used uint64
	limitReached := false
	for i, tx := range txs {
		size := uint64(len(tx))
		if size == 0 {
			out[i] = coreexecution.FilterRemove
			continue
		}
		if maxBytes > 0 && size > maxBytes {
			out[i] = coreexecution.FilterRemove
			continue
		}
		if limitReached {
			out[i] = coreexecution.FilterPostpone
			continue
		}
		if maxBytes > 0 && used+size > maxBytes {
			limitReached = true
			out[i] = coreexecution.FilterPostpone
			continue
		}
		used += size
		out[i] = coreexecution.FilterOK
	}
	return out, nil
}

var _ coreexecution.Executor = (*inMemExecutor)(nil)
