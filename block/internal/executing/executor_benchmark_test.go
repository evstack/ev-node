package executing

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/celestiaorg/go-header"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	coreexec "github.com/evstack/ev-node/core/execution"
	coreseq "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/signer/noop"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

func BenchmarkProduceBlock(b *testing.B) {
	specs := map[string]struct {
		txs [][]byte
	}{
		"empty batch": {
			txs: nil,
		},
		"single tx": {
			txs: [][]byte{[]byte("tx1")},
		},
		"100 txs": {
			txs: createTxs(100),
		},
	}
	for name, spec := range specs {
		b.Run(name, func(b *testing.B) {
			exec := newBenchExecutorWithStubs(b, spec.txs)
			ctx := b.Context()

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := exec.ProduceBlock(ctx); err != nil {
					b.Fatalf("ProduceBlock: %v", err)
				}
			}
		})
	}
}

// newBenchExecutorWithStubs creates an Executor using zero-overhead stubs.
func newBenchExecutorWithStubs(b *testing.B, txs [][]byte) *Executor {
	b.Helper()

	ds := sync.MutexWrap(datastore.NewMapDatastore())
	memStore := store.New(ds)

	cacheManager, err := cache.NewManager(config.DefaultConfig(), memStore, zerolog.Nop())
	require.NoError(b, err)

	priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
	require.NoError(b, err)
	signerWrapper, err := noop.NewNoopSigner(priv)
	require.NoError(b, err)
	addr, err := signerWrapper.GetAddress()
	require.NoError(b, err)

	cfg := config.DefaultConfig()
	cfg.Node.BlockTime = config.DurationWrapper{Duration: 10 * time.Millisecond}
	cfg.Node.MaxPendingHeadersAndData = 0 // disabled — avoids advancePastEmptyData store scans

	gen := genesis.Genesis{
		ChainID:         "bench-chain",
		InitialHeight:   1,
		StartTime:       time.Now().Add(-time.Hour),
		ProposerAddress: addr,
	}

	stubExec := &stubExecClient{stateRoot: []byte("init_root")}
	stubSeq := &stubSequencer{txs: txs}
	hb := &stubBroadcaster[*types.P2PSignedHeader]{}
	db := &stubBroadcaster[*types.P2PData]{}

	exec, err := NewExecutor(
		memStore, stubExec, stubSeq, signerWrapper,
		cacheManager, common.NopMetrics(), cfg, gen,
		hb, db, zerolog.Nop(), common.DefaultBlockOptions(),
		make(chan error, 1), nil,
	)
	require.NoError(b, err)

	require.NoError(b, exec.initializeState())

	exec.ctx, exec.cancel = context.WithCancel(b.Context())
	b.Cleanup(func() { exec.cancel() })

	return exec
}

// stubSequencer implements coreseq.Sequencer.
// GetNextBatch returns a monotonically-increasing timestamp on every call so
// that successive ProduceBlock iterations pass AssertValidSequence.
type stubSequencer struct {
	txs     [][]byte
	counter atomic.Int64 // incremented each call; used to advance the timestamp
}

func (s *stubSequencer) SubmitBatchTxs(context.Context, coreseq.SubmitBatchTxsRequest) (*coreseq.SubmitBatchTxsResponse, error) {
	return nil, nil
}
func (s *stubSequencer) GetNextBatch(context.Context, coreseq.GetNextBatchRequest) (*coreseq.GetNextBatchResponse, error) {
	n := s.counter.Add(1)
	ts := time.Now().Add(time.Duration(n) * time.Millisecond)
	return &coreseq.GetNextBatchResponse{
		Batch:     &coreseq.Batch{Transactions: s.txs},
		Timestamp: ts,
		BatchData: s.txs,
	}, nil
}
func (s *stubSequencer) VerifyBatch(context.Context, coreseq.VerifyBatchRequest) (*coreseq.VerifyBatchResponse, error) {
	return nil, nil
}
func (s *stubSequencer) SetDAHeight(uint64)  {}
func (s *stubSequencer) GetDAHeight() uint64 { return 0 }

func createTxs(n int) [][]byte {
	txs := make([][]byte, n)
	for i := 0; i < n; i++ {
		txs[i] = []byte(fmt.Sprintf("tx%d", i))
	}
	return txs
}

// stubExecClient implements coreexec.Executor with fixed return values.
type stubExecClient struct {
	stateRoot []byte
}

func (s *stubExecClient) InitChain(context.Context, time.Time, uint64, string) ([]byte, error) {
	return s.stateRoot, nil
}
func (s *stubExecClient) GetTxs(context.Context) ([][]byte, error) { return nil, nil }
func (s *stubExecClient) ExecuteTxs(_ context.Context, _ [][]byte, _ uint64, _ time.Time, _ []byte) ([]byte, error) {
	return s.stateRoot, nil
}
func (s *stubExecClient) SetFinal(context.Context, uint64) error { return nil }
func (s *stubExecClient) GetExecutionInfo(context.Context) (coreexec.ExecutionInfo, error) {
	return coreexec.ExecutionInfo{}, nil
}
func (s *stubExecClient) FilterTxs(context.Context, [][]byte, uint64, uint64, bool) ([]coreexec.FilterStatus, error) {
	return nil, nil
}

// stubBroadcaster implements common.Broadcaster[H] with no-ops.
type stubBroadcaster[H header.Header[H]] struct{}

func (s *stubBroadcaster[H]) WriteToStoreAndBroadcast(context.Context, H, ...pubsub.PubOpt) error {
	return nil
}
func (s *stubBroadcaster[H]) Store() header.Store[H] { return nil }
func (s *stubBroadcaster[H]) Height() uint64         { return 0 }
