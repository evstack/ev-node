package reaping

import (
	"context"
	crand "crypto/rand"
	"testing"
	"time"

	ds "github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/block/internal/executing"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/signer/noop"
	testmocks "github.com/evstack/ev-node/test/mocks"
)

// helper to create a minimal executor to capture notifications
func newTestExecutor(t *testing.T) *executing.Executor {
	t.Helper()

	// signer is required by NewExecutor
	priv, _, err := crypto.GenerateEd25519Key(crand.Reader)
	require.NoError(t, err)
	s, err := noop.NewNoopSigner(priv)
	require.NoError(t, err)

	// Get the signer's address to use as proposer
	signerAddr, err := s.GetAddress()
	require.NoError(t, err)

	exec, err := executing.NewExecutor(
		nil, // store (unused)
		nil, // core executor (unused)
		nil, // sequencer (unused)
		s,   // signer (required)
		nil, // cache (unused)
		nil, // metrics (unused)
		config.DefaultConfig(),
		genesis.Genesis{ // minimal genesis
			ChainID:         "test-chain",
			InitialHeight:   1,
			StartTime:       time.Now(),
			ProposerAddress: signerAddr,
		},
		nil, // header broadcaster
		nil, // data broadcaster
		zerolog.Nop(),
		common.DefaultBlockOptions(),
		make(chan error, 1), // error channel
	)
	require.NoError(t, err)

	return exec
}

// reaper with mocks and in-memory seen store
func newTestReaper(t *testing.T, chainID string, execMock *testmocks.MockExecutor, seqMock *testmocks.MockSequencer, e *executing.Executor) *Reaper {
	t.Helper()

	r, err := NewReaper(execMock, seqMock, genesis.Genesis{ChainID: chainID}, zerolog.Nop(), e, 100*time.Millisecond)
	require.NoError(t, err)

	return r
}

func TestReaper_SubmitTxs_NewTxs_SubmitsAndPersistsAndNotifies(t *testing.T) {
	mockExec := testmocks.NewMockExecutor(t)
	mockSeq := testmocks.NewMockSequencer(t)

	// Two new transactions
	tx1 := []byte("tx1")
	tx2 := []byte("tx2")
	mockExec.EXPECT().GetTxs(mock.Anything).Return([][]byte{tx1, tx2}, nil).Once()

	// Expect a single SubmitBatchTxs with both txs
	mockSeq.EXPECT().SubmitBatchTxs(mock.Anything, mock.AnythingOfType("sequencer.SubmitBatchTxsRequest")).
		RunAndReturn(func(ctx context.Context, req coresequencer.SubmitBatchTxsRequest) (*coresequencer.SubmitBatchTxsResponse, error) {
			require.Equal(t, []byte("chain-A"), req.Id)
			require.NotNil(t, req.Batch)
			assert.Equal(t, [][]byte{tx1, tx2}, req.Batch.Transactions)
			return &coresequencer.SubmitBatchTxsResponse{}, nil
		}).Once()

	// Minimal executor to capture NotifyNewTransactions
	e := newTestExecutor(t)

	r := newTestReaper(t, "chain-A", mockExec, mockSeq, e)
	store := r.SeenStore()

	r.SubmitTxs()

	// Seen keys persisted
	has1, err := store.Has(context.Background(), ds.NewKey(hashTx(tx1)))
	require.NoError(t, err)
	has2, err := store.Has(context.Background(), ds.NewKey(hashTx(tx2)))
	require.NoError(t, err)
	assert.True(t, has1)
	assert.True(t, has2)

	// Executor notified - check using test helper
	if !e.HasPendingTxNotification() {
		t.Fatal("expected NotifyNewTransactions to signal txNotifyCh")
	}
}

func TestReaper_SubmitTxs_AllSeen_NoSubmit(t *testing.T) {
	mockExec := testmocks.NewMockExecutor(t)
	mockSeq := testmocks.NewMockSequencer(t)

	tx1 := []byte("tx1")
	tx2 := []byte("tx2")

	// Pre-populate seen store
	e := newTestExecutor(t)
	r := newTestReaper(t, "chain-B", mockExec, mockSeq, e)
	store := r.SeenStore()
	require.NoError(t, store.Put(context.Background(), ds.NewKey(hashTx(tx1)), []byte{1}))
	require.NoError(t, store.Put(context.Background(), ds.NewKey(hashTx(tx2)), []byte{1}))

	mockExec.EXPECT().GetTxs(mock.Anything).Return([][]byte{tx1, tx2}, nil).Once()
	// No SubmitBatchTxs expected

	r.SubmitTxs()

	// Ensure no notification occurred
	if e.HasPendingTxNotification() {
		t.Fatal("did not expect notification when all txs are seen")
	}
}

func TestReaper_SubmitTxs_PartialSeen_FiltersAndPersists(t *testing.T) {
	mockExec := testmocks.NewMockExecutor(t)
	mockSeq := testmocks.NewMockSequencer(t)

	txOld := []byte("old")
	txNew := []byte("new")

	e := newTestExecutor(t)
	r := newTestReaper(t, "chain-C", mockExec, mockSeq, e)

	// Mark txOld as seen
	store := r.SeenStore()
	require.NoError(t, store.Put(context.Background(), ds.NewKey(hashTx(txOld)), []byte{1}))

	mockExec.EXPECT().GetTxs(mock.Anything).Return([][]byte{txOld, txNew}, nil).Once()
	mockSeq.EXPECT().SubmitBatchTxs(mock.Anything, mock.AnythingOfType("sequencer.SubmitBatchTxsRequest")).
		RunAndReturn(func(ctx context.Context, req coresequencer.SubmitBatchTxsRequest) (*coresequencer.SubmitBatchTxsResponse, error) {
			// Should only include txNew
			assert.Equal(t, [][]byte{txNew}, req.Batch.Transactions)
			return &coresequencer.SubmitBatchTxsResponse{}, nil
		}).Once()

	r.SubmitTxs()

	// Both should be seen after successful submit
	hasOld, err := store.Has(context.Background(), ds.NewKey(hashTx(txOld)))
	require.NoError(t, err)
	hasNew, err := store.Has(context.Background(), ds.NewKey(hashTx(txNew)))
	require.NoError(t, err)
	assert.True(t, hasOld)
	assert.True(t, hasNew)

	// Notification should occur since a new tx was submitted
	if !e.HasPendingTxNotification() {
		t.Fatal("expected notification when new tx submitted")
	}
}

func TestReaper_SubmitTxs_SequencerError_NoPersistence_NoNotify(t *testing.T) {
	mockExec := testmocks.NewMockExecutor(t)
	mockSeq := testmocks.NewMockSequencer(t)

	tx := []byte("oops")
	mockExec.EXPECT().GetTxs(mock.Anything).Return([][]byte{tx}, nil).Once()
	mockSeq.EXPECT().SubmitBatchTxs(mock.Anything, mock.AnythingOfType("sequencer.SubmitBatchTxsRequest")).
		Return((*coresequencer.SubmitBatchTxsResponse)(nil), assert.AnError).Once()

	e := newTestExecutor(t)
	r := newTestReaper(t, "chain-D", mockExec, mockSeq, e)

	r.SubmitTxs()

	// Should not be marked seen
	store := r.SeenStore()
	has, err := store.Has(context.Background(), ds.NewKey(hashTx(tx)))
	require.NoError(t, err)
	assert.False(t, has)

	// Should not notify
	if e.HasPendingTxNotification() {
		t.Fatal("did not expect notification on sequencer error")
	}
}
