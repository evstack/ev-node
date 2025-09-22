package syncing

import (
	"context"
	"testing"
	"time"

	"github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/store"
	testmocks "github.com/evstack/ev-node/test/mocks"
	"github.com/evstack/ev-node/types"
)

// benchFixture encapsulates a ready-to-run Syncer for benchmarks
type benchFixture struct {
	s      *Syncer
	st     store.Store
	cm     cache.Manager
	cancel context.CancelFunc
}

func newBenchFixture(b *testing.B, totalHeights uint64, daDelay, execDelay time.Duration, includeP2P bool) *benchFixture {
	b.Helper()
	ctx, cancel := context.WithCancel(b.Context())

	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(b, err)

	addr, pub, signer := buildSyncTestSigner(b)
	cfg := config.DefaultConfig()
	// keep P2P ticker dormant unless we manually inject P2P events
	cfg.Node.BlockTime = config.DurationWrapper{Duration: 10 * time.Second}
	cfg.DA.StartHeight = 1
	gen := genesis.Genesis{ChainID: "bchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}

	mockExec := testmocks.NewMockExecutor(b)
	// if execDelay > 0, sleep on ExecuteTxs to simulate slow consumer
	mockExec.On("ExecuteTxs", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			if execDelay > 0 {
				time.Sleep(execDelay)
			}
		}).
		Return([]byte("app"), uint64(1024), nil).Maybe()

	// Build syncer with mocks
	s := NewSyncer(
		st,
		mockExec,
		nil, // DA injected via mock retriever below
		cm,
		common.NopMetrics(),
		cfg,
		gen,
		nil, // headerStore not used; we inject P2P directly to channel when needed
		nil, // dataStore not used
		zerolog.Nop(),
		common.DefaultBlockOptions(),
		make(chan error, 1),
	)
	require.NoError(b, s.initializeState())
	s.ctx = ctx

	// Mock DA retriever to emit exactly totalHeights events, then HFF and cancel
	daR := newMockdaRetriever(b)
	for i := uint64(1); i <= totalHeights; i++ {
		i := i // capture
		_, sh := makeSignedHeaderBytes(b, gen.ChainID, i, addr, pub, signer, nil)
		d := &types.Data{Metadata: &types.Metadata{ChainID: gen.ChainID, Height: i, Time: uint64(time.Now().UnixNano())}}
		ev := common.DAHeightEvent{Header: sh, Data: d, DaHeight: i, HeaderDaIncludedHeight: i}
		daR.On("RetrieveFromDA", mock.Anything, i).
			Run(func(_ mock.Arguments) {
				if daDelay > 0 {
					time.Sleep(daDelay)
				}
			}).
			Return([]common.DAHeightEvent{ev}, nil).Once()
	}
	// after last, return height-from-future and stop when queue drains
	daR.On("RetrieveFromDA", mock.Anything, totalHeights+1).
		Run(func(_ mock.Arguments) {
			// wait until producer -> consumer queue is mostly drained for determinism
			require.Eventually(b, func() bool { return len(s.heightInCh) == 0 }, 2*time.Second, 5*time.Millisecond)
			cancel()
		}).
		Return(nil, common.ErrHeightFromFutureStr).Maybe()

	// Attach mocks
	s.daRetriever = daR
	s.p2pHandler = newMockp2pHandler(b) // not used directly in this benchmark path

	return &benchFixture{s: s, st: st, cm: cm, cancel: cancel}
}

func BenchmarkSyncerIO(b *testing.B) {
	cases := map[string]struct {
		heights    uint64
		daDelay    time.Duration
		execDelay  time.Duration
		p2pEnabled bool
		p2pDelay   time.Duration
	}{
		"slow producer":          {heights: 100, daDelay: 200 * time.Microsecond, execDelay: 0, p2pDelay: 0, p2pEnabled: false},
		"slow producer big gap":  {heights: 5_000, daDelay: 200 * time.Microsecond, execDelay: 0, p2pDelay: 0, p2pEnabled: false},
		"slow producer + p2p":    {heights: 100, daDelay: 200 * time.Microsecond, execDelay: 0, p2pDelay: 150 * time.Microsecond, p2pEnabled: true},
		"slow consumer":          {heights: 100, daDelay: 0, execDelay: 200 * time.Microsecond, p2pDelay: 0, p2pEnabled: false},
		"slow consumer big gap ": {heights: 5_000, daDelay: 0, execDelay: 200 * time.Microsecond, p2pDelay: 0, p2pEnabled: false},
		"slow consumer + p2p":    {heights: 100, daDelay: 0, execDelay: 200 * time.Microsecond, p2pDelay: 0, p2pEnabled: true},
	}
	for name, spec := range cases {
		b.Run(name, func(b *testing.B) {
			addr, pub, signer := buildSyncTestSigner(b)
			gen := genesis.Genesis{ChainID: "bchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				fixt := newBenchFixture(b, spec.heights, spec.daDelay, spec.execDelay, true)
				// kick off a P2P injector to provide interleaved events deterministically
				if spec.p2pEnabled {
					go func() {
						for h := uint64(1); h <= spec.heights; h += 2 { // half of the heights via P2P
							if spec.p2pDelay > 0 {
								time.Sleep(spec.p2pDelay)
							}
							_, sh := makeSignedHeaderBytes(b, gen.ChainID, h, addr, pub, signer, nil)
							d := &types.Data{Metadata: &types.Metadata{ChainID: gen.ChainID, Height: h, Time: uint64(time.Now().UnixNano())}}
							select {
							case fixt.s.heightInCh <- common.DAHeightEvent{Header: sh, Data: d, DaHeight: 0}:
							case <-fixt.s.ctx.Done():
								return
							}
						}
					}()
				}
				// run both loops
				go fixt.s.processLoop()
				fixt.s.syncLoop()

				// Ensure clean end-state per iteration
				require.Len(b, fixt.s.cache.GetPendingEvents(), 0)
				assert.Len(b, fixt.s.heightInCh, 0)
			}
		})
	}
}
