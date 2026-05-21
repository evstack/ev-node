package main

import (
	"context"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/evstack/ev-node/block"
)

// instrumentedAdapter wraps a block.FiberClient and records latency
// per Upload (and per Download) call. The bench's stats printer
// reads percentiles from here so we can answer "is the bottleneck
// ev-node's submitter serialization, or actual Fibre Upload time?".
//
// We keep the last N samples in a ring buffer rather than an
// unbounded slice so a long run does not grow memory; N is sized for
// a 30-minute run at peak block rate.
type instrumentedAdapter struct {
	inner block.FiberClient

	uploadCount     atomic.Uint64
	uploadFailures  atomic.Uint64
	uploadBytesSent atomic.Uint64

	mu      sync.Mutex
	samples []time.Duration // ring buffer of recent durations
	idx     int             // next slot to write
	full    bool            // ring buffer has wrapped at least once
}

const uploadSampleCapacity = 4096

func newInstrumentedAdapter(inner block.FiberClient) *instrumentedAdapter {
	return &instrumentedAdapter{
		inner:   inner,
		samples: make([]time.Duration, uploadSampleCapacity),
	}
}

func (m *instrumentedAdapter) Head(ctx context.Context) (uint64, error) {
	return 0, nil
}

func (a *instrumentedAdapter) Upload(ctx context.Context, namespace []byte, data []byte) (block.FiberUploadResult, error) {
	start := time.Now()
	res, err := a.inner.Upload(ctx, namespace, data)
	elapsed := time.Since(start)

	a.uploadCount.Add(1)
	if err != nil {
		a.uploadFailures.Add(1)
	} else {
		a.uploadBytesSent.Add(uint64(len(data)))
	}

	a.mu.Lock()
	a.samples[a.idx] = elapsed
	a.idx = (a.idx + 1) % len(a.samples)
	if a.idx == 0 {
		a.full = true
	}
	a.mu.Unlock()

	return res, err
}

func (a *instrumentedAdapter) Download(ctx context.Context, blobID block.FiberBlobID) ([]byte, error) {
	return a.inner.Download(ctx, blobID)
}

func (a *instrumentedAdapter) Listen(ctx context.Context, namespace []byte, fromHeight uint64) (<-chan block.FiberBlobEvent, error) {
	return a.inner.Listen(ctx, namespace, fromHeight)
}

// uploadStats returns snapshot p50, p99, mean of recent Upload
// durations plus cumulative counters. Returns zero durations when
// no samples have been recorded yet.
type uploadStats struct {
	Count    uint64
	Failures uint64
	BytesOK  uint64
	P50      time.Duration
	P99      time.Duration
	Mean     time.Duration
	Max      time.Duration
}

func (a *instrumentedAdapter) uploadStats() uploadStats {
	a.mu.Lock()
	var n int
	if a.full {
		n = len(a.samples)
	} else {
		n = a.idx
	}
	if n == 0 {
		a.mu.Unlock()
		return uploadStats{
			Count:    a.uploadCount.Load(),
			Failures: a.uploadFailures.Load(),
			BytesOK:  a.uploadBytesSent.Load(),
		}
	}
	// Copy under lock so we can sort outside it.
	cp := make([]time.Duration, n)
	copy(cp, a.samples[:n])
	a.mu.Unlock()

	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })

	var sum time.Duration
	for _, d := range cp {
		sum += d
	}

	pct := func(p float64) time.Duration {
		idx := int(float64(n-1) * p)
		if idx < 0 {
			idx = 0
		}
		if idx >= n {
			idx = n - 1
		}
		return cp[idx]
	}

	return uploadStats{
		Count:    a.uploadCount.Load(),
		Failures: a.uploadFailures.Load(),
		BytesOK:  a.uploadBytesSent.Load(),
		P50:      pct(0.50),
		P99:      pct(0.99),
		Mean:     sum / time.Duration(n),
		Max:      cp[n-1],
	}
}

// Compile-time guard: must satisfy the same interface ev-node consumes.
var _ block.FiberClient = (*instrumentedAdapter)(nil)
