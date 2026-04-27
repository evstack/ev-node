package main

import (
	"context"
	"encoding/binary"
	"sync"
	"sync/atomic"
	"time"
)

// loaderBackoff is what each worker waits when InjectTx returns false
// because the mempool channel is full. A real sleep (rather than
// runtime.Gosched) caps the per-worker drop rate so allocation
// pressure scales with actual drain throughput; without it, full-
// mempool workers spin a tight allocate-then-drop loop at ~200k
// iter/s/worker — millions of short-lived slices per second across the
// pool, which drove the OOM kills we hit early in the investigation.
// 100 µs caps each worker at ~10k drops/s when the mempool is
// permanently full.
const loaderBackoff = 100 * time.Microsecond

// loader pumps fixed-size payloads into the in-mem executor as fast as it
// can. Backpressure comes from the executor's bounded mempool channel:
// when full, InjectTx returns false and we count it as dropped.
//
// Each payload is `txSize` bytes: a tx-id (uint64) prefix + zero filler.
// Non-deterministic content isn't important — ev-node hashes them for
// the seen-tx cache, so any unique-per-tx prefix is enough to avoid
// dedup hits.
type loader struct {
	exec    *inMemExecutor
	workers int
	txSize  int

	// counter monotonically increments per generated tx so the
	// SHA-256-based seen cache never falsely dedups.
	counter atomic.Uint64
}

func newLoader(exec *inMemExecutor, workers, txSize int) *loader {
	if workers < 1 {
		workers = 1
	}
	if txSize < 8 {
		txSize = 8
	}
	return &loader{
		exec:    exec,
		workers: workers,
		txSize:  txSize,
	}
}

// run blocks until ctx is done. Each worker spins on InjectTx — when
// full, it briefly yields. We don't sleep-back-off because the entire
// point of the bench is to keep the executor's mempool pressed against
// its bound.
func (l *loader) run(ctx context.Context) {
	var wg sync.WaitGroup
	for i := 0; i < l.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			buf := make([]byte, l.txSize)
			for {
				if ctx.Err() != nil {
					return
				}
				id := l.counter.Add(1)
				binary.BigEndian.PutUint64(buf, id)
				// Copy on each Inject — the executor's mempool is a
				// channel of []byte, and the consumer keeps a
				// reference. Reusing the same buffer would corrupt
				// in-flight items.
				tx := make([]byte, l.txSize)
				copy(tx, buf)
				if !l.exec.InjectTx(tx) {
					// Mempool full — back off briefly and retry.
					select {
					case <-ctx.Done():
						return
					case <-time.After(loaderBackoff):
					}
				}
			}
		}()
	}
	wg.Wait()
}
