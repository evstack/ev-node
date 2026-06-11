package reaping

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/genesis"
	testmocks "github.com/evstack/ev-node/test/mocks"
)

type testEnv struct {
	execMock *testmocks.MockExecutor
	seqMock  *testmocks.MockSequencer
	reaper   *Reaper
	notified atomic.Bool
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	mockExec := testmocks.NewMockExecutor(t)
	mockSeq := testmocks.NewMockSequencer(t)

	env := &testEnv{
		execMock: mockExec,
		seqMock:  mockSeq,
	}

	r, err := NewReaper(
		mockExec, mockSeq,
		genesis.Genesis{ChainID: "test-chain"},
		zerolog.Nop(),
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

func TestReaper_NewTxs_SubmitsAndNotifies(t *testing.T) {
	env := newTestEnv(t)

	tx1 := []byte("tx1")
	tx2 := []byte("tx2")

	env.execMock.EXPECT().GetTxs(mock.Anything).Return([][]byte{tx1, tx2}, nil).Once()

	env.seqMock.EXPECT().SubmitBatchTxs(mock.Anything, mock.AnythingOfType("sequencer.SubmitBatchTxsRequest")).
		RunAndReturn(func(ctx context.Context, req coresequencer.SubmitBatchTxsRequest) (*coresequencer.SubmitBatchTxsResponse, error) {
			assert.Equal(t, [][]byte{tx1, tx2}, req.Batch.Transactions)
			return &coresequencer.SubmitBatchTxsResponse{}, nil
		}).Once()

	err := env.reaper.drainMempool()
	assert.NoError(t, err)
	assert.True(t, env.wasNotified())
}

func TestReaper_SequencerError_NoNotify(t *testing.T) {
	env := newTestEnv(t)

	tx := []byte("oops")

	env.execMock.EXPECT().GetTxs(mock.Anything).Return([][]byte{tx}, nil).Once()

	env.seqMock.EXPECT().SubmitBatchTxs(mock.Anything, mock.AnythingOfType("sequencer.SubmitBatchTxsRequest")).
		Return((*coresequencer.SubmitBatchTxsResponse)(nil), assert.AnError).Once()

	err := env.reaper.drainMempool()
	assert.Error(t, err)
	assert.False(t, env.wasNotified())
}

func TestReaper_EmptyMempool_NoAction(t *testing.T) {
	env := newTestEnv(t)

	env.execMock.EXPECT().GetTxs(mock.Anything).Return(nil, nil).Once()

	err := env.reaper.drainMempool()
	assert.NoError(t, err)
	assert.False(t, env.wasNotified())
}

func TestReaper_SinglePass_SubmitsAll(t *testing.T) {
	env := newTestEnv(t)

	tx1 := []byte("tx1")
	tx2 := []byte("tx2")
	tx3 := []byte("tx3")

	// single GetTxs call returns all txs
	env.execMock.EXPECT().GetTxs(mock.Anything).Return([][]byte{tx1, tx2, tx3}, nil).Once()

	env.seqMock.EXPECT().SubmitBatchTxs(mock.Anything, mock.AnythingOfType("sequencer.SubmitBatchTxsRequest")).
		RunAndReturn(func(ctx context.Context, req coresequencer.SubmitBatchTxsRequest) (*coresequencer.SubmitBatchTxsResponse, error) {
			assert.Equal(t, [][]byte{tx1, tx2, tx3}, req.Batch.Transactions)
			return &coresequencer.SubmitBatchTxsResponse{}, nil
		}).Once()

	err := env.reaper.drainMempool()
	assert.NoError(t, err)
	assert.True(t, env.wasNotified())
}

func TestReaper_NilCallback_NoPanic(t *testing.T) {
	mockExec := testmocks.NewMockExecutor(t)
	mockSeq := testmocks.NewMockSequencer(t)

	r, err := NewReaper(
		mockExec, mockSeq,
		genesis.Genesis{ChainID: "test-chain"},
		zerolog.Nop(),
		100*time.Millisecond,
		nil,
	)
	require.NoError(t, err)

	tx := []byte("tx")
	mockExec.EXPECT().GetTxs(mock.Anything).Return([][]byte{tx}, nil).Once()
	mockSeq.EXPECT().SubmitBatchTxs(mock.Anything, mock.AnythingOfType("sequencer.SubmitBatchTxsRequest")).
		Return(&coresequencer.SubmitBatchTxsResponse{}, nil).Once()

	err = r.drainMempool()
	assert.NoError(t, err)
}

func TestReaper_StopTerminates(t *testing.T) {
	env := newTestEnv(t)
	env.execMock.EXPECT().GetTxs(mock.Anything).Return(nil, nil).Maybe()

	require.NoError(t, env.reaper.Start(context.Background()))

	done := make(chan struct{})
	go func() {
		env.reaper.Stop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop() did not return in time")
	}
}
