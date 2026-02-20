package executing

import (
	"context"
	crand "crypto/rand"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	coreseq "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	pkgsigner "github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/signer/noop"
	"github.com/evstack/ev-node/pkg/store"
	testmocks "github.com/evstack/ev-node/test/mocks"
	"github.com/evstack/ev-node/types"
	"github.com/stretchr/testify/mock"
)

// buildTestSigner returns a signer and its address for use in tests
func buildTestSigner(t *testing.T) (signerAddr []byte, tSigner types.Signer, s pkgsigner.Signer) {
	t.Helper()
	priv, _, err := crypto.GenerateEd25519Key(crand.Reader)
	require.NoError(t, err)
	n, err := noop.NewNoopSigner(priv)
	require.NoError(t, err)
	addr, err := n.GetAddress()
	require.NoError(t, err)
	pub, err := n.GetPublic()
	require.NoError(t, err)
	return addr, types.Signer{PubKey: pub, Address: addr}, n
}

func TestProduceBlock_EmptyBatch_SetsEmptyDataHash(t *testing.T) {
	fx := setupTestExecutor(t, 1000)
	defer fx.Cancel()

	fx.MockSeq.EXPECT().GetNextBatch(mock.Anything, mock.AnythingOfType("sequencer.GetNextBatchRequest")).
		RunAndReturn(func(ctx context.Context, req coreseq.GetNextBatchRequest) (*coreseq.GetNextBatchResponse, error) {
			return &coreseq.GetNextBatchResponse{Batch: &coreseq.Batch{Transactions: nil}, Timestamp: time.Now()}, nil
		}).Once()

	fx.MockExec.EXPECT().ExecuteTxs(mock.Anything, mock.Anything, uint64(1), mock.AnythingOfType("time.Time"), fx.InitStateRoot).
		Return([]byte("new_root"), nil).Once()

	fx.MockSeq.EXPECT().GetDAHeight().Return(uint64(0)).Once()

	err := fx.Exec.ProduceBlock(fx.Exec.ctx)
	require.NoError(t, err)

	h, err := fx.MemStore.Height(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(1), h)

	sh, data, err := fx.MemStore.GetBlockData(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, 0, len(data.Txs))
	assert.EqualValues(t, common.DataHashForEmptyTxs, sh.DataHash)
}

func TestProduceBlock_OutputPassesValidation(t *testing.T) {
	specs := map[string]struct {
		txs [][]byte
	}{
		"empty batch": {txs: nil},
		"single tx":   {txs: [][]byte{[]byte("tx1")}},
		"multi txs":   {txs: [][]byte{[]byte("tx1"), []byte("tx2"), []byte("tx3")}},
		"large tx":    {txs: [][]byte{make([]byte, 10000)}},
	}

	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			assertProduceBlockInvariantWithTxs(t, spec.txs)
		})
	}
}

func FuzzProduceBlock_OutputPassesValidation(f *testing.F) {
	f.Add([]byte("tx1"), []byte("tx2"))
	f.Add([]byte{}, []byte{})
	f.Add(make([]byte, 1000), make([]byte, 2000))

	f.Fuzz(func(t *testing.T, tx1, tx2 []byte) {
		txs := [][]byte{tx1, tx2}
		assertProduceBlockInvariantWithTxs(t, txs)
	})
}

func TestPendingLimit_SkipsProduction(t *testing.T) {
	fx := setupTestExecutor(t, 1)
	defer fx.Cancel()

	fx.MockSeq.EXPECT().GetNextBatch(mock.Anything, mock.AnythingOfType("sequencer.GetNextBatchRequest")).
		RunAndReturn(func(ctx context.Context, req coreseq.GetNextBatchRequest) (*coreseq.GetNextBatchResponse, error) {
			return &coreseq.GetNextBatchResponse{Batch: &coreseq.Batch{Transactions: nil}, Timestamp: time.Now()}, nil
		}).Once()
	fx.MockExec.EXPECT().ExecuteTxs(mock.Anything, mock.Anything, uint64(1), mock.AnythingOfType("time.Time"), fx.InitStateRoot).
		Return([]byte("i1"), nil).Once()

	fx.MockSeq.EXPECT().GetDAHeight().Return(uint64(0)).Once()

	require.NoError(t, fx.Exec.ProduceBlock(fx.Exec.ctx))
	h1, err := fx.MemStore.Height(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(1), h1)

	require.NoError(t, fx.Exec.ProduceBlock(fx.Exec.ctx))
	h2, err := fx.MemStore.Height(context.Background())
	require.NoError(t, err)
	assert.Equal(t, h1, h2, "height should not change when production is skipped")
}

func TestExecutor_executeTxsWithRetry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		setupMock     func(*testmocks.MockExecutor)
		expectSuccess bool
		expectHash    []byte
		expectError   string
	}{
		{
			name: "success on first attempt",
			setupMock: func(exec *testmocks.MockExecutor) {
				exec.On("ExecuteTxs", mock.Anything, mock.Anything, uint64(100), mock.Anything, mock.Anything).
					Return([]byte("new-hash"), nil).Once()
			},
			expectSuccess: true,
			expectHash:    []byte("new-hash"),
		},
		{
			name: "success on second attempt",
			setupMock: func(exec *testmocks.MockExecutor) {
				exec.On("ExecuteTxs", mock.Anything, mock.Anything, uint64(100), mock.Anything, mock.Anything).
					Return([]byte(nil), errors.New("temporary failure")).Once()
				exec.On("ExecuteTxs", mock.Anything, mock.Anything, uint64(100), mock.Anything, mock.Anything).
					Return([]byte("new-hash"), nil).Once()
			},
			expectSuccess: true,
			expectHash:    []byte("new-hash"),
		},
		{
			name: "success on third attempt",
			setupMock: func(exec *testmocks.MockExecutor) {
				exec.On("ExecuteTxs", mock.Anything, mock.Anything, uint64(100), mock.Anything, mock.Anything).
					Return([]byte(nil), errors.New("temporary failure")).Times(2)
				exec.On("ExecuteTxs", mock.Anything, mock.Anything, uint64(100), mock.Anything, mock.Anything).
					Return([]byte("new-hash"), nil).Once()
			},
			expectSuccess: true,
			expectHash:    []byte("new-hash"),
		},
		{
			name: "failure after max retries",
			setupMock: func(exec *testmocks.MockExecutor) {
				exec.On("ExecuteTxs", mock.Anything, mock.Anything, uint64(100), mock.Anything, mock.Anything).
					Return([]byte(nil), errors.New("persistent failure")).Times(common.MaxRetriesBeforeHalt)
			},
			expectSuccess: false,
			expectError:   "failed to execute transactions",
		},
		{
			name: "context cancelled during retry",
			setupMock: func(exec *testmocks.MockExecutor) {
				exec.On("ExecuteTxs", mock.Anything, mock.Anything, uint64(100), mock.Anything, mock.Anything).
					Return([]byte(nil), errors.New("temporary failure")).Once()
			},
			expectSuccess: false,
			expectError:   "context cancelled during retry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				ctx := context.Background()
				execCtx := ctx

				// For context cancellation test, create a cancellable context
				if tt.name == "context cancelled during retry" {
					var cancel context.CancelFunc
					execCtx, cancel = context.WithCancel(ctx)
					// Cancel context after first failure to simulate cancellation during retry
					go func() {
						time.Sleep(100 * time.Millisecond)
						cancel()
					}()
				}

				mockExec := testmocks.NewMockExecutor(t)
				tt.setupMock(mockExec)

				e := &Executor{
					exec:   mockExec,
					ctx:    execCtx,
					logger: zerolog.Nop(),
				}

				rawTxs := [][]byte{[]byte("tx1"), []byte("tx2")}
				header := types.Header{
					BaseHeader: types.BaseHeader{Height: 100, Time: uint64(time.Now().UnixNano())},
				}
				currentState := types.State{AppHash: []byte("current-hash")}

				result, err := e.executeTxsWithRetry(ctx, rawTxs, header, currentState)

				if tt.expectSuccess {
					require.NoError(t, err)
					assert.Equal(t, tt.expectHash, result)
				} else {
					require.Error(t, err)
					if tt.expectError != "" {
						assert.Contains(t, err.Error(), tt.expectError)
					}
				}

				mockExec.AssertExpectations(t)
			})
		})
	}
}

type executorTestFixture struct {
	MemStore      store.Store
	MockExec      *testmocks.MockExecutor
	MockSeq       *testmocks.MockSequencer
	Exec          *Executor
	Cancel        context.CancelFunc
	InitStateRoot []byte
}

func setupTestExecutor(t *testing.T, pendingLimit uint64) executorTestFixture {
	t.Helper()

	ds := sync.MutexWrap(datastore.NewMapDatastore())
	memStore := store.New(ds)

	cacheManager, err := cache.NewManager(config.DefaultConfig(), memStore, zerolog.Nop())
	require.NoError(t, err)

	metrics := common.NopMetrics()

	addr, _, signerWrapper := buildTestSigner(t)

	cfg := config.DefaultConfig()
	cfg.Node.BlockTime = config.DurationWrapper{Duration: 10 * time.Millisecond}
	cfg.Node.MaxPendingHeadersAndData = pendingLimit

	gen := genesis.Genesis{
		ChainID:         "test-chain",
		InitialHeight:   1,
		StartTime:       time.Now().Add(-time.Second),
		ProposerAddress: addr,
	}

	mockExec := testmocks.NewMockExecutor(t)
	mockSeq := testmocks.NewMockSequencer(t)

	hb := common.NewMockBroadcaster[*types.P2PSignedHeader](t)
	hb.EXPECT().WriteToStoreAndBroadcast(mock.Anything, mock.Anything).Return(nil).Maybe()
	db := common.NewMockBroadcaster[*types.P2PData](t)
	db.EXPECT().WriteToStoreAndBroadcast(mock.Anything, mock.Anything).Return(nil).Maybe()

	exec, err := NewExecutor(
		memStore, mockExec, mockSeq, signerWrapper, cacheManager, metrics, cfg, gen, hb, db,
		zerolog.Nop(), common.DefaultBlockOptions(), make(chan error, 1), nil,
	)
	require.NoError(t, err)

	initStateRoot := []byte("init_root")
	mockExec.EXPECT().InitChain(mock.Anything, mock.AnythingOfType("time.Time"), gen.InitialHeight, gen.ChainID).
		Return(initStateRoot, nil).Once()
	mockSeq.EXPECT().SetDAHeight(uint64(0)).Return().Once()

	require.NoError(t, exec.initializeState())

	ctx, cancel := context.WithCancel(context.Background())
	exec.ctx = ctx

	return executorTestFixture{
		MemStore:      memStore,
		MockExec:      mockExec,
		MockSeq:       mockSeq,
		Exec:          exec,
		Cancel:        cancel,
		InitStateRoot: initStateRoot,
	}
}

func assertProduceBlockInvariantWithTxs(t *testing.T, txs [][]byte) {
	t.Helper()
	fx := setupTestExecutor(t, 1000)
	defer fx.Cancel()

	timestamp := time.Now().Add(time.Duration(1+len(txs)*2) * time.Millisecond)

	newRoot := make([]byte, 32)
	for i, tx := range txs {
		if len(tx) > 0 {
			newRoot[(i+int(tx[0]))%32] ^= byte(len(tx))
		}
	}
	newRoot[31] ^= byte(len(txs))

	fx.MockSeq.EXPECT().GetNextBatch(mock.Anything, mock.AnythingOfType("sequencer.GetNextBatchRequest")).
		RunAndReturn(func(ctx context.Context, req coreseq.GetNextBatchRequest) (*coreseq.GetNextBatchResponse, error) {
			return &coreseq.GetNextBatchResponse{Batch: &coreseq.Batch{Transactions: txs}, Timestamp: timestamp}, nil
		}).Once()

	fx.MockExec.EXPECT().ExecuteTxs(mock.Anything, txs, uint64(1), mock.AnythingOfType("time.Time"), fx.InitStateRoot).
		Return(newRoot, nil).Once()
	fx.MockSeq.EXPECT().GetDAHeight().Return(uint64(0)).Once()

	prevState := fx.Exec.getLastState()

	err := fx.Exec.ProduceBlock(fx.Exec.ctx)
	require.NoError(t, err)

	h, err := fx.MemStore.Height(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(1), h)

	header, data, err := fx.MemStore.GetBlockData(context.Background(), h)
	require.NoError(t, err)

	err = header.ValidateBasicWithData(data)
	require.NoError(t, err, "Produced header failed ValidateBasicWithData")

	err = prevState.AssertValidForNextState(header, data)
	require.NoError(t, err, "Produced block failed AssertValidForNextState")
}
