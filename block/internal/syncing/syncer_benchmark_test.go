package syncing

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/store"
	testmocks "github.com/evstack/ev-node/test/mocks"
	"github.com/evstack/ev-node/types"
	"github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const daHeightOffset = 100

func BenchmarkSyncerIO(b *testing.B) {
	cases := map[string]struct {
		heights    uint64
		shuffledTx bool
		daDelay    time.Duration
		execDelay  time.Duration
		p2pEnabled bool
		p2pDelay   time.Duration
	}{
		"slow producer": {heights: 100, shuffledTx: true, daDelay: 200 * time.Microsecond, execDelay: 0, p2pDelay: 0, p2pEnabled: false},
		//"slow producer big backlog":  {heights: 1_000, daDelay: 100 * time.Microsecond, execDelay: 0, p2pDelay: 0, p2pEnabled: false},
		//"slow producer + p2p":        {heights: 100, daDelay: 200 * time.Microsecond, execDelay: 0, p2pDelay: 150 * time.Microsecond, p2pEnabled: true},
		"slow consumer": {heights: 100, daDelay: 0, execDelay: 200 * time.Microsecond, p2pDelay: 0, p2pEnabled: false},
		//"slow consumer big backlog ": {heights: 1_000, daDelay: 0, execDelay: 100 * time.Microsecond, p2pDelay: 0, p2pEnabled: false},
		//"slow consumer + p2p":        {heights: 100, daDelay: 0, execDelay: 200 * time.Microsecond, p2pDelay: 0, p2pEnabled: true},
	}
	for name, spec := range cases {
		b.Run(name, func(b *testing.B) {
			addr, pub, signer := buildSyncTestSigner(b)
			gen := genesis.Genesis{ChainID: "bchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				fixt := newBenchFixture(b, spec.heights, spec.shuffledTx, spec.daDelay, spec.execDelay, true)
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
				require.Len(b, fixt.s.heightInCh, 0)
				require.Equal(b, 0, fixt.s.pendingQueue.Len())

				assert.Equal(b, spec.heights+daHeightOffset, fixt.s.daHeight)
				gotStoreHeight, err := fixt.s.store.Height(b.Context())
				require.NoError(b, err)
				assert.Equal(b, spec.heights, gotStoreHeight)
			}
		})
	}
}

// benchFixture encapsulates a ready-to-run Syncer for benchmarks
type benchFixture struct {
	s      *Syncer
	st     store.Store
	cm     cache.Manager
	cancel context.CancelFunc
}

func newBenchFixture(b *testing.B, totalHeights uint64, shuffledTx bool, daDelay, execDelay time.Duration, includeP2P bool) *benchFixture {
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
	cfg.DA.StartHeight = daHeightOffset
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

	// prepare height events to emit
	var lastHeader *types.SignedHeader
	heightEvents := make([]common.DAHeightEvent, totalHeights)
	for i := uint64(0); i < totalHeights; i++ {
		blockHeight, daHeight := i+gen.InitialHeight, i+daHeightOffset
		_, sh := makeSignedHeaderBytes(b, gen.ChainID, blockHeight, addr, pub, signer, nil)
		lastHeader = sh
		d := &types.Data{Metadata: &types.Metadata{ChainID: gen.ChainID, Height: blockHeight, Time: uint64(time.Now().UnixNano())}}
		heightEvents[i] = common.DAHeightEvent{Header: sh, Data: d, DaHeight: daHeight, HeaderDaIncludedHeight: daHeight}
	}
	if shuffledTx {
		rand.New(rand.NewSource(1)).Shuffle(len(heightEvents), func(i, j int) {
			heightEvents[i], heightEvents[j] = heightEvents[j], heightEvents[i]
		})
	}

	// Mock DA retriever to emit exactly totalHeights events, then HFF and cancel
	daR := newMockdaRetriever(b)
	for i := uint64(0); i < totalHeights; i++ {
		daHeight := i + daHeightOffset
		daR.On("RetrieveFromDA", mock.Anything, daHeight).
			Run(func(_ mock.Arguments) {
				if daDelay > 0 {
					time.Sleep(daDelay)
				}
			}).
			Return([]common.DAHeightEvent{heightEvents[i]}, nil).Once()
	}
	// after last, return height-from-future and stop when queue drains
	daR.On("RetrieveFromDA", mock.Anything, totalHeights+daHeightOffset).
		Run(func(_ mock.Arguments) {
			// wait until producer -> consumer queue is drained to end loop
			if s.cache.IsHeaderSeen(lastHeader.Hash().String()) {
				cancel()
			}
		}).
		Return(nil, common.ErrHeightFromFutureStr).Maybe()

	// Attach mocks
	s.daRetriever = daR
	s.p2pHandler = newMockp2pHandler(b) // not used directly in this benchmark path

	return &benchFixture{s: s, st: st, cm: cm, cancel: cancel}
}
