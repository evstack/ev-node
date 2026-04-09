package reaping

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block/internal/cache"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/store"
	testmocks "github.com/evstack/ev-node/test/mocks"
)

func newTestCache(t *testing.T) cache.CacheManager {
	t.Helper()
	cfg := config.Config{RootDir: t.TempDir()}
	memDS := dssync.MutexWrap(ds.NewMapDatastore())
	st := store.New(memDS)
	cm, err := cache.NewManager(cfg, st, zerolog.Nop())
	require.NoError(t, err)
	return cm
}

type testEnv struct {
	execMock *testmocks.MockExecutor
	seqMock  *testmocks.MockSequencer
	cache    cache.CacheManager
	reaper   *Reaper
	notified atomic.Bool
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	mockExec := testmocks.NewMockExecutor(t)
	mockSeq := testmocks.NewMockSequencer(t)
	cm := newTestCache(t)

	env := &testEnv{
		execMock: mockExec,
		seqMock:  mockSeq,
		cache:    cm,
	}

	r, err := NewReaper(
		mockExec, mockSeq,
		genesis.Genesis{ChainID: "test-chain"},
		zerolog.Nop(), cm,
		100*time.Millisecond,
		env.notify,
	)
	require.NoError(t, err)
	env.reaper = r

	return env
}

func (e *testEnv) notify() {
	e.notified.Store(true)
}

func (e *testEnv) wasNotified() bool {
	return e.notified.Load()
}

func TestReaper_NewTxs_SubmitsAndPersistsAndNotifies(t *testing.T) {
	env := newTestEnv(t)

	tx1 := []byte("tx1")
	tx2 := []byte("tx2")

	env.execMock.EXPECT().GetTxs(mock.Anything).Return([][]byte{tx1, tx2}, nil).Once()
	env.execMock.EXPECT().GetTxs(mock.Anything).Return(nil, nil).Once()

	env.seqMock.EXPECT().SubmitBatchTxs(mock.Anything, mock.AnythingOfType("sequencer.SubmitBatchTxsRequest")).
		RunAndReturn(func(ctx context.Context, req coresequencer.SubmitBatchTxsRequest) (*coresequencer.SubmitBatchTxsResponse, error) {
			assert.Equal(t, [][]byte{tx1, tx2}, req.Batch.Transactions)
			return &coresequencer.SubmitBatchTxsResponse{}, nil
		}).Once()

	submitted, err := env.reaper.drainMempool()
	assert.NoError(t, err)
	assert.True(t, submitted)
	assert.True(t, env.cache.IsTxSeen(hashTx(tx1)))
	assert.True(t, env.cache.IsTxSeen(hashTx(tx2)))
	assert.True(t, env.wasNotified())
}

func TestReaper_AllSeen_NoSubmit(t *testing.T) {
	env := newTestEnv(t)

	tx1 := []byte("tx1")
	tx2 := []byte("tx2")

	env.cache.SetTxSeen(hashTx(tx1))
	env.cache.SetTxSeen(hashTx(tx2))

	env.execMock.EXPECT().GetTxs(mock.Anything).Return([][]byte{tx1, tx2}, nil).Once()
	env.execMock.EXPECT().GetTxs(mock.Anything).Return(nil, nil).Once()

	submitted, err := env.reaper.drainMempool()
	assert.NoError(t, err)
	assert.False(t, submitted)
	assert.False(t, env.wasNotified())
}

func TestReaper_PartialSeen_FiltersAndPersists(t *testing.T) {
	env := newTestEnv(t)

	txOld := []byte("old")
	txNew := []byte("new")

	env.cache.SetTxSeen(hashTx(txOld))

	env.execMock.EXPECT().GetTxs(mock.Anything).Return([][]byte{txOld, txNew}, nil).Once()
	env.execMock.EXPECT().GetTxs(mock.Anything).Return(nil, nil).Once()

	env.seqMock.EXPECT().SubmitBatchTxs(mock.Anything, mock.AnythingOfType("sequencer.SubmitBatchTxsRequest")).
		RunAndReturn(func(ctx context.Context, req coresequencer.SubmitBatchTxsRequest) (*coresequencer.SubmitBatchTxsResponse, error) {
			assert.Equal(t, [][]byte{txNew}, req.Batch.Transactions)
			return &coresequencer.SubmitBatchTxsResponse{}, nil
		}).Once()

	submitted, err := env.reaper.drainMempool()
	assert.NoError(t, err)
	assert.True(t, submitted)
	assert.True(t, env.cache.IsTxSeen(hashTx(txOld)))
	assert.True(t, env.cache.IsTxSeen(hashTx(txNew)))
	assert.True(t, env.wasNotified())
}

func TestReaper_SequencerError_NoPersistence_NoNotify(t *testing.T) {
	env := newTestEnv(t)

	tx := []byte("oops")

	env.execMock.EXPECT().GetTxs(mock.Anything).Return([][]byte{tx}, nil).Once()

	env.seqMock.EXPECT().SubmitBatchTxs(mock.Anything, mock.AnythingOfType("sequencer.SubmitBatchTxsRequest")).
		Return((*coresequencer.SubmitBatchTxsResponse)(nil), assert.AnError).Once()

	_, err := env.reaper.drainMempool()
	assert.Error(t, err)
	assert.False(t, env.cache.IsTxSeen(hashTx(tx)))
	assert.False(t, env.wasNotified())
}

func TestReaper_DrainsMempoolInMultipleRounds(t *testing.T) {
	env := newTestEnv(t)

	tx1 := []byte("tx1")
	tx2 := []byte("tx2")
	tx3 := []byte("tx3")

	env.execMock.EXPECT().GetTxs(mock.Anything).Return([][]byte{tx1, tx2}, nil).Once()
	env.execMock.EXPECT().GetTxs(mock.Anything).Return([][]byte{tx3}, nil).Once()
	env.execMock.EXPECT().GetTxs(mock.Anything).Return(nil, nil).Once()

	env.seqMock.EXPECT().SubmitBatchTxs(mock.Anything, mock.AnythingOfType("sequencer.SubmitBatchTxsRequest")).
		RunAndReturn(func(ctx context.Context, req coresequencer.SubmitBatchTxsRequest) (*coresequencer.SubmitBatchTxsResponse, error) {
			return &coresequencer.SubmitBatchTxsResponse{}, nil
		}).Twice()

	submitted, err := env.reaper.drainMempool()
	assert.NoError(t, err)
	assert.True(t, submitted)
	assert.True(t, env.cache.IsTxSeen(hashTx(tx1)))
	assert.True(t, env.cache.IsTxSeen(hashTx(tx2)))
	assert.True(t, env.cache.IsTxSeen(hashTx(tx3)))
	assert.True(t, env.wasNotified())
}

func TestReaper_EmptyMempool_NoAction(t *testing.T) {
	env := newTestEnv(t)

	env.execMock.EXPECT().GetTxs(mock.Anything).Return(nil, nil).Once()

	submitted, err := env.reaper.drainMempool()
	assert.NoError(t, err)
	assert.False(t, submitted)
	assert.False(t, env.wasNotified())
}

func TestReaper_HashComputedOnce(t *testing.T) {
	env := newTestEnv(t)

	tx := []byte("unique-tx")
	expectedHash := hashTx(tx)

	env.execMock.EXPECT().GetTxs(mock.Anything).Return([][]byte{tx}, nil).Once()
	env.execMock.EXPECT().GetTxs(mock.Anything).Return(nil, nil).Once()

	env.seqMock.EXPECT().SubmitBatchTxs(mock.Anything, mock.AnythingOfType("sequencer.SubmitBatchTxsRequest")).
		Return(&coresequencer.SubmitBatchTxsResponse{}, nil).Once()

	submitted, err := env.reaper.drainMempool()
	assert.NoError(t, err)
	assert.True(t, submitted)
	assert.True(t, env.cache.IsTxSeen(expectedHash))
}

func TestReaper_NilCallback_NoPanic(t *testing.T) {
	mockExec := testmocks.NewMockExecutor(t)
	mockSeq := testmocks.NewMockSequencer(t)
	cm := newTestCache(t)

	r, err := NewReaper(
		mockExec, mockSeq,
		genesis.Genesis{ChainID: "test-chain"},
		zerolog.Nop(), cm,
		100*time.Millisecond,
		nil,
	)
	require.NoError(t, err)

	tx := []byte("tx")
	mockExec.EXPECT().GetTxs(mock.Anything).Return([][]byte{tx}, nil).Once()
	mockExec.EXPECT().GetTxs(mock.Anything).Return(nil, nil).Once()
	mockSeq.EXPECT().SubmitBatchTxs(mock.Anything, mock.AnythingOfType("sequencer.SubmitBatchTxsRequest")).
		Return(&coresequencer.SubmitBatchTxsResponse{}, nil).Once()

	submitted, err := r.drainMempool()
	assert.NoError(t, err)
	assert.True(t, submitted)
}
