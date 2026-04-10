package reaping

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block/internal/cache"
	coreexecutor "github.com/evstack/ev-node/core/execution"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/store"
)

func newBenchCache(b *testing.B) cache.CacheManager {
	b.Helper()
	cfg := config.Config{RootDir: b.TempDir()}
	memDS := dssync.MutexWrap(ds.NewMapDatastore())
	st := store.New(memDS)
	cm, err := cache.NewManager(cfg, st, zerolog.Nop())
	if err != nil {
		b.Fatal(err)
	}
	return cm
}

type infiniteExecutor struct {
	mu      sync.Mutex
	batch   [][]byte
	pending atomic.Int64
}

func (e *infiniteExecutor) Feed(n int, txSize int) {
	txs := make([][]byte, n)
	for i := range txs {
		tx := make([]byte, txSize)
		_, _ = rand.Read(tx)
		txs[i] = tx
	}
	e.mu.Lock()
	e.batch = txs
	e.mu.Unlock()
	e.pending.Add(int64(n))
}

func (e *infiniteExecutor) GetTxs(_ context.Context) ([][]byte, error) {
	e.mu.Lock()
	txs := e.batch
	e.batch = nil
	e.mu.Unlock()
	return txs, nil
}

func (e *infiniteExecutor) ExecuteTxs(_ context.Context, _ [][]byte, _ uint64, _ time.Time, _ []byte) ([]byte, error) {
	return nil, nil
}

func (e *infiniteExecutor) FilterTxs(_ context.Context, txs [][]byte, _ uint64, _ uint64, _ bool) ([]coreexecutor.FilterStatus, error) {
	return make([]coreexecutor.FilterStatus, len(txs)), nil
}

func (e *infiniteExecutor) GetExecutionInfo(_ context.Context) (coreexecutor.ExecutionInfo, error) {
	return coreexecutor.ExecutionInfo{}, nil
}

func (e *infiniteExecutor) InitChain(_ context.Context, _ time.Time, _ uint64, _ string) ([]byte, error) {
	return nil, nil
}

func (e *infiniteExecutor) SetFinal(_ context.Context, _ uint64) error { return nil }

type countingSequencer struct {
	submitted atomic.Int64
}

func (s *countingSequencer) SubmitBatchTxs(_ context.Context, req coresequencer.SubmitBatchTxsRequest) (*coresequencer.SubmitBatchTxsResponse, error) {
	s.submitted.Add(int64(len(req.Batch.Transactions)))
	return &coresequencer.SubmitBatchTxsResponse{}, nil
}

func (s *countingSequencer) GetNextBatch(_ context.Context, _ coresequencer.GetNextBatchRequest) (*coresequencer.GetNextBatchResponse, error) {
	return &coresequencer.GetNextBatchResponse{}, nil
}

func (s *countingSequencer) VerifyBatch(_ context.Context, _ coresequencer.VerifyBatchRequest) (*coresequencer.VerifyBatchResponse, error) {
	return &coresequencer.VerifyBatchResponse{}, nil
}

func (s *countingSequencer) GetDAHeight() uint64  { return 0 }
func (s *countingSequencer) SetDAHeight(_ uint64) {}

func benchmarkReaperFlow(b *testing.B, batchSize int, txSize int, feedInterval time.Duration) {
	scenario := fmt.Sprintf("batch=%d/txSize=%d", batchSize, txSize)

	b.Run(scenario, func(b *testing.B) {
		exec := &infiniteExecutor{}
		seq := &countingSequencer{}

		cm := newBenchCache(b)
		var notified atomic.Int64

		r, err := NewReaper(
			exec,
			seq,
			genesis.Genesis{ChainID: "bench"},
			zerolog.Nop(),
			cm,
			50*time.Millisecond,
			func() { notified.Add(1) },
		)
		if err != nil {
			b.Fatal(err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		if err := r.Start(ctx); err != nil {
			b.Fatal(err)
		}

		stopFeeding := make(chan struct{})
		feederDone := make(chan struct{})
		go func() {
			defer close(feederDone)
			ticker := time.NewTicker(feedInterval)
			defer ticker.Stop()
			for {
				select {
				case <-stopFeeding:
					return
				case <-ticker.C:
					exec.Feed(batchSize, txSize)
				}
			}
		}()

		b.ResetTimer()

		var lastCount int64
		for i := 0; i < b.N; i++ {
			exec.Feed(batchSize, txSize)

			deadline := time.After(5 * time.Second)
			expected := seq.submitted.Load() + int64(batchSize)
			for {
				cur := seq.submitted.Load()
				if cur >= expected {
					lastCount = cur
					break
				}
				select {
				case <-deadline:
					b.Fatalf("timeout: submitted %d, expected >= %d", cur, expected)
				default:
				}
			}
		}

		b.StopTimer()
		close(stopFeeding)
		<-feederDone
		cancel()
		r.Stop()

		b.ReportMetric(float64(lastCount)/b.Elapsed().Seconds(), "txs/sec")
	})
}

func BenchmarkReaperFlow_Throughput(b *testing.B) {
	sizes := []int{256, 1024, 4096}
	batches := []int{10, 100, 500}

	for _, batchSize := range batches {
		for _, txSize := range sizes {
			benchmarkReaperFlow(b, batchSize, txSize, 10*time.Millisecond)
		}
	}
}

func BenchmarkReaperFlow_Sustained(b *testing.B) {
	b.Run("steady_100txs_256B", func(b *testing.B) {
		exec := &infiniteExecutor{}
		seq := &countingSequencer{}
		cm := newBenchCache(b)
		var notified atomic.Int64

		r, err := NewReaper(
			exec, seq,
			genesis.Genesis{ChainID: "bench"},
			zerolog.Nop(), cm,
			10*time.Millisecond,
			func() { notified.Add(1) },
		)
		if err != nil {
			b.Fatal(err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		if err := r.Start(ctx); err != nil {
			b.Fatal(err)
		}

		const batchSize = 100
		const txSize = 256
		const duration = 3 * time.Second

		feederDone := make(chan struct{})
		go func() {
			defer close(feederDone)
			for {
				exec.Feed(batchSize, txSize)
				time.Sleep(time.Millisecond)
				select {
				case <-ctx.Done():
					return
				default:
				}
			}
		}()

		b.ResetTimer()
		time.Sleep(duration)
		b.StopTimer()

		cancel()
		r.Stop()
		<-feederDone

		total := seq.submitted.Load()
		elapsed := b.Elapsed().Seconds()
		b.ReportMetric(float64(total)/elapsed, "txs/sec")
		b.ReportMetric(float64(notified.Load())/elapsed, "notifies/sec")
		b.Logf("submitted %d txs in %.1fs (%.0f txs/sec, %d notifications)", total, elapsed, float64(total)/elapsed, notified.Load())
	})
}

func BenchmarkReaperFlow_StartStop(b *testing.B) {
	b.Run("lifecycle", func(b *testing.B) {
		exec := &infiniteExecutor{}
		seq := &countingSequencer{}
		cm := newBenchCache(b)

		for i := 0; i < b.N; i++ {
			r, err := NewReaper(
				exec, seq,
				genesis.Genesis{ChainID: "bench"},
				zerolog.Nop(), cm,
				100*time.Millisecond,
				func() {},
			)
			if err != nil {
				b.Fatal(err)
			}

			ctx, cancel := context.WithCancel(context.Background())
			if err := r.Start(ctx); err != nil {
				cancel()
				b.Fatal(err)
			}
			r.Stop()
			cancel()
		}
	})

	b.Run("start_with_backlog", func(b *testing.B) {
		const batchSize = 500
		const txSize = 256

		for i := 0; i < b.N; i++ {
			b.StopTimer()

			exec := &infiniteExecutor{}
			seq := &countingSequencer{}
			cm := newBenchCache(b)

			txs := make([][]byte, batchSize)
			for j := range txs {
				txs[j] = make([]byte, txSize)
				_, _ = rand.Read(txs[j])
			}
			exec.mu.Lock()
			exec.batch = txs
			exec.mu.Unlock()

			r, err := NewReaper(
				exec, seq,
				genesis.Genesis{ChainID: "bench"},
				zerolog.Nop(), cm,
				10*time.Millisecond,
				func() {},
			)
			if err != nil {
				b.Fatal(err)
			}

			ctx, cancel := context.WithCancel(context.Background())

			b.StartTimer()
			if err := r.Start(ctx); err != nil {
				cancel()
				b.Fatal(err)
			}

			deadline := time.After(5 * time.Second)
			for seq.submitted.Load() < batchSize {
				select {
				case <-deadline:
					b.Fatal("timeout waiting for backlog drain")
				default:
				}
			}
			b.StopTimer()

			r.Stop()
			cancel()
		}
	})
}
