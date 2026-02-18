package executing

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	coreseq "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/signer/noop"
	"github.com/evstack/ev-node/pkg/store"
	testmocks "github.com/evstack/ev-node/test/mocks"
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
	}
	for name, spec := range specs {
		b.Run(name, func(b *testing.B) {
			exec := newBenchExecutor(b, spec.txs)
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

func newBenchExecutor(b *testing.B, txs [][]byte) *Executor {
	b.Helper()

	ds := sync.MutexWrap(datastore.NewMapDatastore())
	memStore := store.New(ds)

	cacheManager, err := cache.NewManager(config.DefaultConfig(), memStore, zerolog.Nop())
	require.NoError(b, err)

	// Generate signer without depending on *testing.T
	priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
	require.NoError(b, err)
	signerWrapper, err := noop.NewNoopSigner(priv)
	require.NoError(b, err)
	addr, err := signerWrapper.GetAddress()
	require.NoError(b, err)

	cfg := config.DefaultConfig()
	cfg.Node.BlockTime = config.DurationWrapper{Duration: 10 * time.Millisecond}
	cfg.Node.MaxPendingHeadersAndData = 100000

	gen := genesis.Genesis{
		ChainID:         "bench-chain",
		InitialHeight:   1,
		StartTime:       time.Now().Add(-time.Hour),
		ProposerAddress: addr,
	}

	mockExec := testmocks.NewMockExecutor(b)
	mockSeq := testmocks.NewMockSequencer(b)
	hb := common.NewMockBroadcaster[*types.P2PSignedHeader](b)
	db := common.NewMockBroadcaster[*types.P2PData](b)

	hb.EXPECT().WriteToStoreAndBroadcast(mock.Anything, mock.Anything).Return(nil)
	db.EXPECT().WriteToStoreAndBroadcast(mock.Anything, mock.Anything).Return(nil)

	exec, err := NewExecutor(
		memStore, mockExec, mockSeq, signerWrapper,
		cacheManager, common.NopMetrics(), cfg, gen,
		hb, db, zerolog.Nop(), common.DefaultBlockOptions(),
		make(chan error, 1), nil,
	)
	require.NoError(b, err)

	// One-time init expectations
	mockExec.EXPECT().InitChain(mock.Anything, mock.AnythingOfType("time.Time"), gen.InitialHeight, gen.ChainID).
		Return([]byte("init_root"), nil).Once()
	mockSeq.EXPECT().SetDAHeight(uint64(0)).Return().Once()

	require.NoError(b, exec.initializeState())

	exec.ctx, exec.cancel = context.WithCancel(b.Context())
	b.Cleanup(func() { exec.cancel() })

	// Loop expectations (unlimited calls)
	lastBatchTime := gen.StartTime
	mockSeq.EXPECT().GetNextBatch(mock.Anything, mock.AnythingOfType("sequencer.GetNextBatchRequest")).
		RunAndReturn(func(ctx context.Context, req coreseq.GetNextBatchRequest) (*coreseq.GetNextBatchResponse, error) {
			lastBatchTime = lastBatchTime.Add(cfg.Node.BlockTime.Duration)
			return &coreseq.GetNextBatchResponse{
				Batch:     &coreseq.Batch{Transactions: txs},
				Timestamp: lastBatchTime,
				BatchData: txs,
			}, nil
		})
	mockExec.EXPECT().ExecuteTxs(mock.Anything, mock.Anything, mock.Anything, mock.AnythingOfType("time.Time"), mock.Anything).
		Return([]byte("new_root"), nil)
	mockSeq.EXPECT().GetDAHeight().Return(uint64(0))

	return exec
}
