package syncing

import (
	"context"
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

func TestProcessHeightEvent_UpdatesDAHeight_WhenAlreadyProcessed(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)

	addr, pub, signer := buildSyncTestSigner(t)

	cfg := config.DefaultConfig()
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}

	mockExec := testmocks.NewMockExecutor(t)
	mockExec.EXPECT().InitChain(mock.Anything, mock.Anything, uint64(1), "tchain").Return([]byte("app0"), uint64(1024), nil).Once()

	errChan := make(chan error, 1)
	s := NewSyncer(
		st,
		mockExec,
		nil,
		cm,
		common.NopMetrics(),
		cfg,
		gen,
		common.NewMockBroadcaster[*types.SignedHeader](t),
		common.NewMockBroadcaster[*types.Data](t),
		zerolog.Nop(),
		common.DefaultBlockOptions(),
		errChan,
	)

	require.NoError(t, s.initializeState())
	s.ctx = context.Background()

	// 1. Sync a block normally (simulating P2P sync or previous DA sync)
	lastState := s.GetLastState()
	data := makeData(gen.ChainID, 1, 0)
	_, hdr := makeSignedHeaderBytes(t, gen.ChainID, 1, addr, pub, signer, lastState.AppHash, data, nil)

	// Expect ExecuteTxs call for height 1
	mockExec.EXPECT().ExecuteTxs(mock.Anything, mock.Anything, uint64(1), mock.Anything, lastState.AppHash).
		Return([]byte("app1"), uint64(1024), nil).Once()

	// Process the event (simulate P2P source to NOT update DA height initially if we wanted,
	// but here we just want to get the block synced.
	// Let's say it came from P2P, so DA height remains at genesis (0 or whatever start is).
	evtP2P := common.DAHeightEvent{Header: hdr, Data: data, DaHeight: 0, Source: common.SourceP2P}
	s.processHeightEvent(&evtP2P)

	requireEmptyChan(t, errChan)

	// Verify block is synced
	h, err := st.Height(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(1), h)

	// Verify current DA height is still initial (0)
	currentState := s.GetLastState()
	assert.Equal(t, uint64(0), currentState.DAHeight)

	// 2. Now receive the SAME block from DA, but with a higher DA height
	// This simulates the "processed block is higher (or equal) than what is being synced from da" scenario
	// where we want to ensure DA height is updated.
	daHeight := uint64(100)
	evtDA := common.DAHeightEvent{Header: hdr, Data: data, DaHeight: daHeight, Source: common.SourceDA}

	s.processHeightEvent(&evtDA)

	// 3. Verify DA height is updated in state
	updatedState := s.GetLastState()
	assert.Equal(t, daHeight, updatedState.DAHeight)

	// Verify it's persisted
	storedState, err := st.GetState(context.Background())
	require.NoError(t, err)
	assert.Equal(t, daHeight, storedState.DAHeight)
}
