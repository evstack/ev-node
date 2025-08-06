package block

import (
	"context"
	"encoding/binary"
	"fmt"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coreda "github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/pkg/cache"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/signer/noop"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/test/mocks"
	"github.com/evstack/ev-node/types"
	ds "github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p/core/crypto"
)

const numItemsToSubmit = 3

// newTestManagerWithDA creates a Manager instance with a mocked DA layer for testing.
func newTestManagerWithDA(t *testing.T, da *mocks.MockDA) (m *Manager) {
	logger := zerolog.Nop()
	nodeConf := config.DefaultConfig

	privKey, _, err := crypto.GenerateKeyPair(crypto.Ed25519, 256)
	require.NoError(t, err)
	testSigner, err := noop.NewNoopSigner(privKey)
	require.NoError(t, err)

	proposerAddr, err := testSigner.GetAddress()
	require.NoError(t, err)
	gen := genesis.NewGenesis(
		"testchain",
		1,
		time.Now(),
		proposerAddr,
	)

	return &Manager{
		da:             da,
		logger:         logger,
		config:         nodeConf,
		gasPrice:       1.0,
		gasMultiplier:  2.0,
		headerCache:    cache.NewCache[types.SignedHeader](),
		dataCache:      cache.NewCache[types.Data](),
		signer:         testSigner,
		genesis:        gen,
		pendingData:    newPendingData(t),
		pendingHeaders: newPendingHeaders(t),
		metrics:        NopMetrics(),
	}
}

// --- Generic success test for data and headers submission ---
type submitToDASuccessCase[T any] struct {
	name        string
	fillPending func(ctx context.Context, t *testing.T, m *Manager)
	getToSubmit func(m *Manager, ctx context.Context) ([]T, error)
	submitToDA  func(m *Manager, ctx context.Context, items []T) error
	mockDASetup func(da *mocks.MockDA)
}

func runSubmitToDASuccessCase[T any](t *testing.T, tc submitToDASuccessCase[T]) {
	da := &mocks.MockDA{}
	m := newTestManagerWithDA(t, da)

	ctx := t.Context()
	tc.fillPending(ctx, t, m)
	tc.mockDASetup(da)

	items, err := tc.getToSubmit(m, ctx)
	require.NoError(t, err)
	require.NotEmpty(t, items)

	err = tc.submitToDA(m, ctx, items)
	assert.NoError(t, err)
}

func TestSubmitDataToDA_Success(t *testing.T) {
	runSubmitToDASuccessCase(t, submitToDASuccessCase[*types.SignedData]{
		name: "Data",
		fillPending: func(ctx context.Context, t *testing.T, m *Manager) {
			fillPendingData(ctx, t, m.pendingData, "Test Submitting Data", numItemsToSubmit)
		},
		getToSubmit: func(m *Manager, ctx context.Context) ([]*types.SignedData, error) {
			return m.createSignedDataToSubmit(ctx)
		},
		submitToDA: func(m *Manager, ctx context.Context, items []*types.SignedData) error {
			return m.submitDataToDA(ctx, items)
		},
		mockDASetup: func(da *mocks.MockDA) {
			da.On("SubmitWithOptions", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return([]coreda.ID{getDummyID(1, []byte("commitment"))}, nil)
		},
	})
}

func TestSubmitHeadersToDA_Success(t *testing.T) {
	runSubmitToDASuccessCase(t, submitToDASuccessCase[*types.SignedHeader]{
		name: "Headers",
		fillPending: func(ctx context.Context, t *testing.T, m *Manager) {
			fillPendingHeaders(ctx, t, m.pendingHeaders, "Test Submitting Headers", numItemsToSubmit)
		},
		getToSubmit: func(m *Manager, ctx context.Context) ([]*types.SignedHeader, error) {
			return m.pendingHeaders.getPendingHeaders(ctx)
		},
		submitToDA: func(m *Manager, ctx context.Context, items []*types.SignedHeader) error {
			return m.submitHeadersToDA(ctx, items)
		},
		mockDASetup: func(da *mocks.MockDA) {
			da.On("SubmitWithOptions", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return([]coreda.ID{getDummyID(1, []byte("commitment"))}, nil)
		},
	})
}

// --- Generic failure test for data and headers submission ---
type submitToDAFailureCase[T any] struct {
	name        string
	fillPending func(ctx context.Context, t *testing.T, m *Manager)
	getToSubmit func(m *Manager, ctx context.Context) ([]T, error)
	submitToDA  func(m *Manager, ctx context.Context, items []T) error
	errorMsg    string
	daError     error
	mockDASetup func(da *mocks.MockDA, gasPriceHistory *[]float64, daError error)
}

func runSubmitToDAFailureCase[T any](t *testing.T, tc submitToDAFailureCase[T]) {
	da := &mocks.MockDA{}
	m := newTestManagerWithDA(t, da)

	ctx := t.Context()
	tc.fillPending(ctx, t, m)

	var gasPriceHistory []float64
	tc.mockDASetup(da, &gasPriceHistory, tc.daError)

	items, err := tc.getToSubmit(m, ctx)
	require.NoError(t, err)
	require.NotEmpty(t, items)

	err = tc.submitToDA(m, ctx, items)
	assert.Error(t, err, "expected error")
	assert.Contains(t, err.Error(), tc.errorMsg)

	// Validate that gas price increased according to gas multiplier
	previousGasPrice := m.gasPrice
	assert.Equal(t, gasPriceHistory[0], m.gasPrice) // verify that the first call is done with the right price
	for _, gasPrice := range gasPriceHistory[1:] {
		assert.Equal(t, gasPrice, previousGasPrice*m.gasMultiplier)
		previousGasPrice = gasPrice
	}
}

func TestSubmitDataToDA_Failure(t *testing.T) {
	testCases := []struct {
		name    string
		daError error
	}{
		{"AlreadyInMempool", coreda.ErrTxAlreadyInMempool},
		{"TimedOut", coreda.ErrTxTimedOut},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runSubmitToDAFailureCase(t, submitToDAFailureCase[*types.SignedData]{
				name: "Data",
				fillPending: func(ctx context.Context, t *testing.T, m *Manager) {
					fillPendingData(ctx, t, m.pendingData, "Test Submitting Data", numItemsToSubmit)
				},
				getToSubmit: func(m *Manager, ctx context.Context) ([]*types.SignedData, error) {
					return m.createSignedDataToSubmit(ctx)
				},
				submitToDA: func(m *Manager, ctx context.Context, items []*types.SignedData) error {
					return m.submitDataToDA(ctx, items)
				},
				errorMsg: "failed to submit all data(s) to DA layer",
				daError:  tc.daError,
				mockDASetup: func(da *mocks.MockDA, gasPriceHistory *[]float64, daError error) {
					da.ExpectedCalls = nil
					da.On("SubmitWithOptions", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
						Run(func(args mock.Arguments) { *gasPriceHistory = append(*gasPriceHistory, args.Get(2).(float64)) }).
						Return(nil, daError)
				},
			})
		})
	}
}

func TestSubmitHeadersToDA_Failure(t *testing.T) {
	testCases := []struct {
		name    string
		daError error
	}{
		{"AlreadyInMempool", coreda.ErrTxAlreadyInMempool},
		{"TimedOut", coreda.ErrTxTimedOut},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runSubmitToDAFailureCase(t, submitToDAFailureCase[*types.SignedHeader]{
				name: "Headers",
				fillPending: func(ctx context.Context, t *testing.T, m *Manager) {
					fillPendingHeaders(ctx, t, m.pendingHeaders, "Test Submitting Headers", numItemsToSubmit)
				},
				getToSubmit: func(m *Manager, ctx context.Context) ([]*types.SignedHeader, error) {
					return m.pendingHeaders.getPendingHeaders(ctx)
				},
				submitToDA: func(m *Manager, ctx context.Context, items []*types.SignedHeader) error {
					return m.submitHeadersToDA(ctx, items)
				},
				errorMsg: "failed to submit all header(s) to DA layer",
				daError:  tc.daError,
				mockDASetup: func(da *mocks.MockDA, gasPriceHistory *[]float64, daError error) {
					da.ExpectedCalls = nil
					da.On("SubmitWithOptions", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
						Run(func(args mock.Arguments) { *gasPriceHistory = append(*gasPriceHistory, args.Get(2).(float64)) }).
						Return(nil, daError)
				},
			})
		})
	}
}

// --- Generic retry partial failures test for data and headers ---
type retryPartialFailuresCase[T any] struct {
	name               string
	metaKey            string
	fillPending        func(ctx context.Context, t *testing.T, m *Manager)
	submitToDA         func(m *Manager, ctx context.Context, items []T) error
	getLastSubmitted   func(m *Manager) uint64
	getPendingToSubmit func(m *Manager, ctx context.Context) ([]T, error)
	setupStoreAndDA    func(m *Manager, mockStore *mocks.MockStore, da *mocks.MockDA)
}

func runRetryPartialFailuresCase[T any](t *testing.T, tc retryPartialFailuresCase[T]) {
	m := newTestManagerWithDA(t, nil)
	mockStore := mocks.NewMockStore(t)
	m.store = mockStore
	m.logger = zerolog.Nop()
	da := &mocks.MockDA{}
	m.da = da
	m.gasPrice = 1.0
	m.gasMultiplier = 2.0
	tc.setupStoreAndDA(m, mockStore, da)
	ctx := t.Context()
	tc.fillPending(ctx, t, m)

	// Prepare items to submit
	items, err := tc.getPendingToSubmit(m, ctx)
	require.NoError(t, err)
	require.Len(t, items, 3)

	// Set up DA mock: three calls, each time only one item is accepted
	da.On("GasMultiplier", mock.Anything).Return(2.0, nil).Times(3)
	da.On("SubmitWithOptions", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			t.Logf("DA Submit call 1: args=%#v", args)
		}).Once().Return([]coreda.ID{getDummyID(1, []byte("commitment2"))}, nil)
	da.On("SubmitWithOptions", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			t.Logf("DA Submit call 2: args=%#v", args)
		}).Once().Return([]coreda.ID{getDummyID(1, []byte("commitment3"))}, nil)
	da.On("SubmitWithOptions", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			t.Logf("DA Submit call 3: args=%#v", args)
		}).Once().Return([]coreda.ID{getDummyID(1, []byte("commitment4"))}, nil)

	err = tc.submitToDA(m, ctx, items)
	assert.NoError(t, err)

	// After all succeed, lastSubmitted should be 3
	assert.Equal(t, uint64(3), tc.getLastSubmitted(m))
}

func TestSubmitToDA_RetryPartialFailures_Generic(t *testing.T) {
	casesData := retryPartialFailuresCase[*types.SignedData]{
		name:    "Data",
		metaKey: "last-submitted-data-height",
		fillPending: func(ctx context.Context, t *testing.T, m *Manager) {
			fillPendingData(ctx, t, m.pendingData, "Test", numItemsToSubmit)
		},
		submitToDA: func(m *Manager, ctx context.Context, items []*types.SignedData) error {
			return m.submitDataToDA(ctx, items)
		},
		getLastSubmitted: func(m *Manager) uint64 {
			return m.pendingData.getLastSubmittedDataHeight()
		},
		getPendingToSubmit: func(m *Manager, ctx context.Context) ([]*types.SignedData, error) {
			return m.createSignedDataToSubmit(ctx)
		},
		setupStoreAndDA: func(m *Manager, mockStore *mocks.MockStore, da *mocks.MockDA) {
			lastSubmittedBytes := make([]byte, 8)
			lastHeight := uint64(0)
			binary.LittleEndian.PutUint64(lastSubmittedBytes, lastHeight)
			mockStore.On("GetMetadata", mock.Anything, "last-submitted-data-height").Return(lastSubmittedBytes, nil).Maybe()
			mockStore.On("SetMetadata", mock.Anything, "last-submitted-data-height", mock.Anything).Return(nil).Maybe()
			mockStore.On("Height", mock.Anything).Return(uint64(4), nil).Maybe()
			for h := uint64(2); h <= 4; h++ {
				mockStore.On("GetBlockData", mock.Anything, h).Return(nil, &types.Data{
					Txs:      types.Txs{types.Tx(fmt.Sprintf("tx%d", h))},
					Metadata: &types.Metadata{Height: h},
				}, nil).Maybe()
			}
		},
	}

	casesHeader := retryPartialFailuresCase[*types.SignedHeader]{
		name:    "Header",
		metaKey: "last-submitted-header-height",
		fillPending: func(ctx context.Context, t *testing.T, m *Manager) {
			fillPendingHeaders(ctx, t, m.pendingHeaders, "Test", numItemsToSubmit)
		},
		submitToDA: func(m *Manager, ctx context.Context, items []*types.SignedHeader) error {
			return m.submitHeadersToDA(ctx, items)
		},
		getLastSubmitted: func(m *Manager) uint64 {
			return m.pendingHeaders.getLastSubmittedHeaderHeight()
		},
		getPendingToSubmit: func(m *Manager, ctx context.Context) ([]*types.SignedHeader, error) {
			return m.pendingHeaders.getPendingHeaders(ctx)
		},
		setupStoreAndDA: func(m *Manager, mockStore *mocks.MockStore, da *mocks.MockDA) {
			lastSubmittedBytes := make([]byte, 8)
			lastHeight := uint64(0)
			binary.LittleEndian.PutUint64(lastSubmittedBytes, lastHeight)
			mockStore.On("GetMetadata", mock.Anything, "last-submitted-header-height").Return(lastSubmittedBytes, nil).Maybe()
			mockStore.On("SetMetadata", mock.Anything, "last-submitted-header-height", mock.Anything).Return(nil).Maybe()
			mockStore.On("Height", mock.Anything).Return(uint64(4), nil).Maybe()
			for h := uint64(2); h <= 4; h++ {
				header := &types.SignedHeader{Header: types.Header{BaseHeader: types.BaseHeader{Height: h}}}
				mockStore.On("GetBlockData", mock.Anything, h).Return(header, nil, nil).Maybe()
			}
		},
	}

	t.Run(casesData.name, func(t *testing.T) {
		runRetryPartialFailuresCase(t, casesData)
	})

	t.Run(casesHeader.name, func(t *testing.T) {
		runRetryPartialFailuresCase(t, casesHeader)
	})
}

// TestCreateSignedDataToSubmit tests createSignedDataToSubmit for normal, empty, and error cases.
func TestCreateSignedDataToSubmit(t *testing.T) {
	// Normal case: pending data exists and is signed correctly
	t.Run("normal case", func(t *testing.T) {
		m := newTestManagerWithDA(t, nil)
		fillPendingData(t.Context(), t, m.pendingData, "Test Creating Signed Data", numItemsToSubmit)
		pubKey, err := m.signer.GetPublic()
		require.NoError(t, err)
		proposerAddr, err := m.signer.GetAddress()
		require.NoError(t, err)
		signedDataList, err := m.createSignedDataToSubmit(t.Context())
		require.NoError(t, err)
		require.Len(t, signedDataList, numItemsToSubmit)
		assert.Equal(t, types.Tx("tx1"), signedDataList[0].Txs[0])
		assert.Equal(t, types.Tx("tx2"), signedDataList[0].Txs[1])
		assert.Equal(t, pubKey, signedDataList[0].Signer.PubKey)
		assert.Equal(t, proposerAddr, signedDataList[0].Signer.Address)
		assert.NotEmpty(t, signedDataList[0].Signature)
		assert.Equal(t, types.Tx("tx3"), signedDataList[1].Txs[0])
		assert.Equal(t, types.Tx("tx4"), signedDataList[1].Txs[1])
		assert.Equal(t, pubKey, signedDataList[1].Signer.PubKey)
		assert.Equal(t, proposerAddr, signedDataList[1].Signer.Address)
		assert.NotEmpty(t, signedDataList[1].Signature)
	})

	// Empty pending data: should return no error and an empty list
	t.Run("empty pending data", func(t *testing.T) {
		m := newTestManagerWithDA(t, nil)
		signedDataList, err := m.createSignedDataToSubmit(t.Context())
		assert.NoError(t, err, "expected no error when pending data is empty")
		assert.Empty(t, signedDataList, "expected signedDataList to be empty when no pending data")
	})

	// getPendingData returns error: should return error and error message should match
	t.Run("getPendingData returns error", func(t *testing.T) {
		m := newTestManagerWithDA(t, nil)
		mockStore := mocks.NewMockStore(t)
		logger := zerolog.Nop()
		mockStore.On("GetMetadata", mock.Anything, "last-submitted-data-height").Return(nil, ds.ErrNotFound).Once()
		mockStore.On("Height", mock.Anything).Return(uint64(1), nil).Once()
		mockStore.On("GetBlockData", mock.Anything, uint64(1)).Return(nil, nil, fmt.Errorf("mock error")).Once()
		pendingData, err := NewPendingData(mockStore, logger)
		require.NoError(t, err)
		m.pendingData = pendingData
		signedDataList, err := m.createSignedDataToSubmit(t.Context())
		assert.Error(t, err, "expected error when getPendingData fails")
		assert.Contains(t, err.Error(), "mock error", "error message should contain 'mock error'")
		assert.Nil(t, signedDataList, "signedDataList should be nil on error")
	})

	// signer returns error: should return error and error message should match
	t.Run("signer returns error", func(t *testing.T) {
		m := newTestManagerWithDA(t, nil)
		m.signer = nil
		signedDataList, err := m.createSignedDataToSubmit(t.Context())
		assert.Error(t, err, "expected error when signer is nil")
		assert.Contains(t, err.Error(), "signer is nil; cannot sign data", "error message should mention nil signer")
		assert.Nil(t, signedDataList, "signedDataList should be nil on error")
	})
}

// fillPendingHeaders populates the given PendingHeaders with a sequence of mock SignedHeader objects for testing.
// It generates headers with consecutive heights and stores them in the underlying store so that PendingHeaders logic can retrieve them.
//
// Parameters:
//
//	ctx: context for store operations
//	t: the testing.T instance
//	pendingHeaders: the PendingHeaders instance to fill
//	chainID: the chain ID to use for generated headers
//	startHeight: the starting height for headers (default 1 if 0)
//	count: the number of headers to generate (default 3 if 0)
func fillPendingHeaders(ctx context.Context, t *testing.T, pendingHeaders *PendingHeaders, chainID string, numBlocks uint64) {
	t.Helper()

	s := pendingHeaders.base.store
	for i := uint64(0); i < numBlocks; i++ {
		height := i + 1
		header, data := types.GetRandomBlock(height, 0, chainID)
		sig := &header.Signature
		err := s.SaveBlockData(ctx, header, data, sig)
		require.NoError(t, err, "failed to save block data for header at height %d", height)
		err = s.SetHeight(ctx, height)
		require.NoError(t, err, "failed to set store height for header at height %d", height)
	}
}

func fillPendingData(ctx context.Context, t *testing.T, pendingData *PendingData, chainID string, numBlocks uint64) {
	t.Helper()
	s := pendingData.base.store
	txNum := 1
	for i := uint64(0); i < numBlocks; i++ {
		height := i + 1
		header, data := types.GetRandomBlock(height, 2, chainID)
		data.Txs = make(types.Txs, len(data.Txs))
		for i := 0; i < len(data.Txs); i++ {
			data.Txs[i] = types.Tx(fmt.Sprintf("tx%d", txNum))
			txNum++
		}
		sig := &header.Signature
		err := s.SaveBlockData(ctx, header, data, sig)
		require.NoError(t, err, "failed to save block data for data at height %d", height)
		err = s.SetHeight(ctx, height)
		require.NoError(t, err, "failed to set store height for data at height %d", height)
	}
}

func newPendingHeaders(t *testing.T) *PendingHeaders {
	kv, err := store.NewDefaultInMemoryKVStore()
	require.NoError(t, err)
	logger := zerolog.Nop()
	pendingHeaders, err := NewPendingHeaders(store.New(kv), logger)
	require.NoError(t, err)
	return pendingHeaders
}

func newPendingData(t *testing.T) *PendingData {
	kv, err := store.NewDefaultInMemoryKVStore()
	require.NoError(t, err)
	logger := zerolog.Nop()
	pendingData, err := NewPendingData(store.New(kv), logger)
	require.NoError(t, err)
	return pendingData
}

// TestSubmitHeadersToDA_WithMetricsRecorder verifies that submitHeadersToDA calls RecordMetrics
// when the sequencer implements the MetricsRecorder interface.
func TestSubmitHeadersToDA_WithMetricsRecorder(t *testing.T) {
	da := &mocks.MockDA{}
	m := newTestManagerWithDA(t, da)

	// Set up mock sequencer with metrics
	mockSequencer := new(MockSequencerWithMetrics)
	m.sequencer = mockSequencer

	// Fill the pending headers with test data
	ctx := context.Background()
	fillPendingHeaders(ctx, t, m.pendingHeaders, "Test Submitting Headers", numItemsToSubmit)

	// Get the headers to submit
	headers, err := m.pendingHeaders.getPendingHeaders(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, headers)

	// Simulate DA layer successfully accepting the header submission
	da.On("SubmitWithOptions", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return([]coreda.ID{[]byte("id")}, nil)

	// Expect RecordMetrics to be called with the correct parameters
	mockSequencer.On("RecordMetrics",
		float64(1.0),                  // gasPrice (from newTestManagerWithDA)
		uint64(0),                     // blobSize (mocked as 0)
		coreda.StatusSuccess,          // statusCode
		mock.AnythingOfType("uint64"), // numPendingBlocks (varies based on test data)
		mock.AnythingOfType("uint64"), // lastSubmittedHeight
	).Maybe()

	// Call submitHeadersToDA and expect no error
	err = m.submitHeadersToDA(ctx, headers)
	assert.NoError(t, err)

	// Verify that RecordMetrics was called at least once
	mockSequencer.AssertExpectations(t)
}

// TestSubmitDataToDA_WithMetricsRecorder verifies that submitDataToDA calls RecordMetrics
// when the sequencer implements the MetricsRecorder interface.
func TestSubmitDataToDA_WithMetricsRecorder(t *testing.T) {
	da := &mocks.MockDA{}
	m := newTestManagerWithDA(t, da)

	// Set up mock sequencer with metrics
	mockSequencer := new(MockSequencerWithMetrics)
	m.sequencer = mockSequencer

	// Fill pending data for testing
	ctx := context.Background()
	fillPendingData(ctx, t, m.pendingData, "Test Submitting Data", numItemsToSubmit)

	// Get the data to submit
	signedDataList, err := m.createSignedDataToSubmit(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, signedDataList)

	// Simulate DA success
	da.On("GasMultiplier", mock.Anything).Return(2.0, nil)
	da.On("SubmitWithOptions", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return([]coreda.ID{[]byte("id")}, nil)

	// Expect RecordMetrics to be called with the correct parameters
	mockSequencer.On("RecordMetrics",
		float64(1.0),                  // gasPrice (from newTestManagerWithDA)
		uint64(0),                     // blobSize (mocked as 0)
		coreda.StatusSuccess,          // statusCode
		mock.AnythingOfType("uint64"), // numPendingBlocks (varies based on test data)
		mock.AnythingOfType("uint64"), // daIncludedHeight
	).Maybe()

	err = m.submitDataToDA(ctx, signedDataList)
	assert.NoError(t, err)

	// Verify that RecordMetrics was called
	mockSequencer.AssertExpectations(t)
}

// TestSubmitToDA_ItChunksBatchWhenSizeExceedsLimit verifies that when DA submission
// fails with StatusTooBig, the submitter automatically splits the batch in half and
// retries until successful. This prevents infinite retry loops when batches exceed
// DA layer size limits.
func TestSubmitToDA_ItChunksBatchWhenSizeExceedsLimit(t *testing.T) {

	da := &mocks.MockDA{}
	m := newTestManagerWithDA(t, da)
	ctx := context.Background()

	// Fill with items that would be too big as a single batch
	largeItemCount := uint64(10)
	fillPendingHeaders(ctx, t, m.pendingHeaders, "TestFix", largeItemCount)

	headers, err := m.pendingHeaders.getPendingHeaders(ctx)
	require.NoError(t, err)
	require.Len(t, headers, int(largeItemCount))

	var submitAttempts int
	var batchSizes []int

	// Mock DA behavior for recursive splitting:
	// - First call: full batch (10 items) -> StatusTooBig
	// - Second call: first half (5 items) -> Success
	// - Third call: second half (5 items) -> Success
	da.On("SubmitWithOptions", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			submitAttempts++
			blobs := args.Get(1).([]coreda.Blob)
			batchSizes = append(batchSizes, len(blobs))
			t.Logf("DA Submit attempt %d: batch size %d", submitAttempts, len(blobs))
		}).
		Return(nil, coreda.ErrBlobSizeOverLimit).Once() // First attempt fails (full batch)

	da.On("SubmitWithOptions", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			submitAttempts++
			blobs := args.Get(1).([]coreda.Blob)
			batchSizes = append(batchSizes, len(blobs))
			t.Logf("DA Submit attempt %d: batch size %d", submitAttempts, len(blobs))
		}).
		Return([]coreda.ID{getDummyID(1, []byte("id1")), getDummyID(1, []byte("id2")), getDummyID(1, []byte("id3")), getDummyID(1, []byte("id4")), getDummyID(1, []byte("id5"))}, nil).Once() // Second attempt succeeds (first half)

	da.On("SubmitWithOptions", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			submitAttempts++
			blobs := args.Get(1).([]coreda.Blob)
			batchSizes = append(batchSizes, len(blobs))
			t.Logf("DA Submit attempt %d: batch size %d", submitAttempts, len(blobs))
		}).
		Return([]coreda.ID{getDummyID(1, []byte("id6")), getDummyID(1, []byte("id7")), getDummyID(1, []byte("id8")), getDummyID(1, []byte("id9")), getDummyID(1, []byte("id10"))}, nil).Once() // Third attempt succeeds (second half)

	err = m.submitHeadersToDA(ctx, headers)

	assert.NoError(t, err, "Should succeed by recursively splitting and submitting all chunks")
	assert.Equal(t, 3, submitAttempts, "Should make 3 attempts: 1 large batch + 2 split chunks")
	assert.Equal(t, []int{10, 5, 5}, batchSizes, "Should try full batch, then both halves")

	// All 10 items should be successfully submitted in a single submitHeadersToDA call
}

// TestSubmitToDA_SingleItemTooLarge verifies behavior when even a single item
// exceeds DA size limits and cannot be split further. This should result in
// exponential backoff and eventual failure after maxSubmitAttempts.
func TestSubmitToDA_SingleItemTooLarge(t *testing.T) {
	da := &mocks.MockDA{}
	m := newTestManagerWithDA(t, da)

	// Set a small MaxSubmitAttempts for fast testing
	m.config.DA.MaxSubmitAttempts = 3

	ctx := context.Background()

	// Create a single header that will always be "too big"
	fillPendingHeaders(ctx, t, m.pendingHeaders, "TestSingleLarge", 1)

	headers, err := m.pendingHeaders.getPendingHeaders(ctx)
	require.NoError(t, err)
	require.Len(t, headers, 1)

	var submitAttempts int

	// Mock DA to always return "too big" for this single item
	da.On("SubmitWithOptions", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			submitAttempts++
			blobs := args.Get(1).([]coreda.Blob)
			t.Logf("DA Submit attempt %d: batch size %d (single item too large)", submitAttempts, len(blobs))
		}).
		Return(nil, coreda.ErrBlobSizeOverLimit) // Always fails

	// This should fail after MaxSubmitAttempts (3) attempts
	err = m.submitHeadersToDA(ctx, headers)

	// Expected behavior: Should fail after exhausting all attempts
	assert.Error(t, err, "Should fail when single item is too large")
	assert.Contains(t, err.Error(), "failed to submit all header(s) to DA layer")
	assert.Contains(t, err.Error(), "after 3 attempts") // MaxSubmitAttempts

	// Should have made exactly MaxSubmitAttempts (3) attempts
	assert.Equal(t, 3, submitAttempts, "Should make exactly MaxSubmitAttempts before giving up")

	da.AssertExpectations(t)
}

// TestProcessBatch tests the processBatch function with different scenarios
func TestProcessBatch(t *testing.T) {
	da := &mocks.MockDA{}
	m := newTestManagerWithDA(t, da)
	ctx := context.Background()

	// Test data setup
	testItems := []string{"item1", "item2", "item3"}
	testMarshaled := [][]byte{[]byte("marshaled1"), []byte("marshaled2"), []byte("marshaled3")}
	testBatch := submissionBatch[string]{
		Items:     testItems,
		Marshaled: testMarshaled,
	}

	var postSubmitCalled bool
	postSubmit := func(submitted []string, res *coreda.ResultSubmit, gasPrice float64) {
		postSubmitCalled = true
		assert.Equal(t, testItems, submitted)
		assert.Equal(t, float64(1.0), gasPrice)
	}

	t.Run("batchActionSubmitted - full chunk success", func(t *testing.T) {
		postSubmitCalled = false
		da.ExpectedCalls = nil

		// Mock successful submission of all items
		da.On("SubmitWithOptions", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return([]coreda.ID{getDummyID(1, []byte("id1")), getDummyID(1, []byte("id2")), getDummyID(1, []byte("id3"))}, nil).Once()

		result := processBatch(m, ctx, testBatch, 1.0, postSubmit, "test")

		assert.Equal(t, batchActionSubmitted, result.action)
		assert.Equal(t, 3, result.submittedCount)
		assert.True(t, postSubmitCalled)
		da.AssertExpectations(t)
	})

	t.Run("batchActionSubmitted - partial chunk success", func(t *testing.T) {
		postSubmitCalled = false
		da.ExpectedCalls = nil

		// Create a separate postSubmit function for partial success test
		var partialPostSubmitCalled bool
		partialPostSubmit := func(submitted []string, res *coreda.ResultSubmit, gasPrice float64) {
			partialPostSubmitCalled = true
			// Only the first 2 items should be submitted
			assert.Equal(t, []string{"item1", "item2"}, submitted)
			assert.Equal(t, float64(1.0), gasPrice)
		}

		// Mock partial submission (only 2 out of 3 items)
		da.On("SubmitWithOptions", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return([]coreda.ID{getDummyID(1, []byte("id1")), getDummyID(1, []byte("id2"))}, nil).Once()

		result := processBatch(m, ctx, testBatch, 1.0, partialPostSubmit, "test")

		assert.Equal(t, batchActionSubmitted, result.action)
		assert.Equal(t, 2, result.submittedCount)
		assert.True(t, partialPostSubmitCalled)
		da.AssertExpectations(t)
	})

	t.Run("batchActionTooBig - chunk too big", func(t *testing.T) {
		da.ExpectedCalls = nil

		// Mock "too big" error for multi-item chunk
		da.On("SubmitWithOptions", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil, coreda.ErrBlobSizeOverLimit).Once()

		result := processBatch(m, ctx, testBatch, 1.0, postSubmit, "test")

		assert.Equal(t, batchActionTooBig, result.action)
		assert.Equal(t, 0, result.submittedCount)
		assert.Empty(t, result.splitBatches)

		da.AssertExpectations(t)
	})

	t.Run("batchActionSkip - single item too big", func(t *testing.T) {
		da.ExpectedCalls = nil

		// Create single-item batch
		singleBatch := submissionBatch[string]{
			Items:     []string{"large_item"},
			Marshaled: [][]byte{[]byte("large_marshaled")},
		}

		// Mock "too big" error for single item
		da.On("SubmitWithOptions", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil, coreda.ErrBlobSizeOverLimit).Once()

		result := processBatch(m, ctx, singleBatch, 1.0, postSubmit, "test")

		assert.Equal(t, batchActionSkip, result.action)
		assert.Equal(t, 0, result.submittedCount)
		assert.Empty(t, result.splitBatches)
		da.AssertExpectations(t)
	})

	t.Run("batchActionFail - unrecoverable error", func(t *testing.T) {
		da.ExpectedCalls = nil

		// Mock network error or other unrecoverable failure
		da.On("SubmitWithOptions", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil, fmt.Errorf("network error")).Once()

		result := processBatch(m, ctx, testBatch, 1.0, postSubmit, "test")

		assert.Equal(t, batchActionFail, result.action)
		assert.Equal(t, 0, result.submittedCount)
		assert.Empty(t, result.splitBatches)
		da.AssertExpectations(t)
	})
}

func getDummyID(height uint64, commitment []byte) coreda.ID {
	id := make([]byte, len(commitment)+8)
	binary.LittleEndian.PutUint64(id, height)
	copy(id[8:], commitment)
	return id
}

// TestRetryStrategy tests all retry strategy functionality using table-driven tests
func TestRetryStrategy(t *testing.T) {
	t.Run("ExponentialBackoff", func(t *testing.T) {
		tests := []struct {
			name            string
			maxBackoff      time.Duration
			initialBackoff  time.Duration
			expectedBackoff time.Duration
			description     string
		}{
			{
				name:            "initial_backoff_from_zero",
				maxBackoff:      10 * time.Second,
				initialBackoff:  0,
				expectedBackoff: 100 * time.Millisecond,
				description:     "should start at 100ms when backoff is 0",
			},
			{
				name:            "doubling_from_100ms",
				maxBackoff:      10 * time.Second,
				initialBackoff:  100 * time.Millisecond,
				expectedBackoff: 200 * time.Millisecond,
				description:     "should double from 100ms to 200ms",
			},
			{
				name:            "doubling_from_500ms",
				maxBackoff:      10 * time.Second,
				initialBackoff:  500 * time.Millisecond,
				expectedBackoff: 1 * time.Second,
				description:     "should double from 500ms to 1s",
			},
			{
				name:            "capped_at_max_backoff",
				maxBackoff:      5 * time.Second,
				initialBackoff:  20 * time.Second,
				expectedBackoff: 5 * time.Second,
				description:     "should cap at max backoff when exceeding limit",
			},
			{
				name:            "zero_max_backoff",
				maxBackoff:      0,
				initialBackoff:  100 * time.Millisecond,
				expectedBackoff: 0,
				description:     "should cap at 0 when max backoff is 0",
			},
			{
				name:            "small_max_backoff",
				maxBackoff:      1 * time.Millisecond,
				initialBackoff:  100 * time.Millisecond,
				expectedBackoff: 1 * time.Millisecond,
				description:     "should cap at very small max backoff",
			},
			{
				name:            "normal_progression_1s",
				maxBackoff:      1 * time.Hour,
				initialBackoff:  1 * time.Second,
				expectedBackoff: 2 * time.Second,
				description:     "should double from 1s to 2s with large max",
			},
			{
				name:            "normal_progression_2s",
				maxBackoff:      1 * time.Hour,
				initialBackoff:  2 * time.Second,
				expectedBackoff: 4 * time.Second,
				description:     "should double from 2s to 4s with large max",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				strategy := newRetryStrategy(1.0, tt.maxBackoff, 30)
				strategy.backoff = tt.initialBackoff

				strategy.BackoffOnFailure()

				assert.Equal(t, tt.expectedBackoff, strategy.backoff, tt.description)
			})
		}
	})

	t.Run("ShouldContinue", func(t *testing.T) {
		strategy := newRetryStrategy(1.0, 1*time.Second, 30)

		// Should continue when attempts are below max
		require.True(t, strategy.ShouldContinue())

		// Simulate reaching max attempts
		strategy.attempt = 30
		require.False(t, strategy.ShouldContinue())
	})

	t.Run("NextAttempt", func(t *testing.T) {
		strategy := newRetryStrategy(1.0, 1*time.Second, 30)

		initialAttempt := strategy.attempt
		strategy.NextAttempt()
		require.Equal(t, initialAttempt+1, strategy.attempt)
	})

	t.Run("ResetOnSuccess", func(t *testing.T) {
		initialGasPrice := 2.0
		strategy := newRetryStrategy(initialGasPrice, 1*time.Second, 30)

		// Set some backoff and higher gas price
		strategy.backoff = 500 * time.Millisecond
		strategy.gasPrice = 4.0
		gasMultiplier := 2.0

		strategy.ResetOnSuccess(gasMultiplier)

		// Backoff should be reset to 0
		require.Equal(t, 0*time.Duration(0), strategy.backoff)

		// Gas price should be reduced but not below initial
		expectedGasPrice := 4.0 / gasMultiplier // 2.0
		require.Equal(t, expectedGasPrice, strategy.gasPrice)
	})

	t.Run("ResetOnSuccess_GasPriceFloor", func(t *testing.T) {
		initialGasPrice := 2.0
		strategy := newRetryStrategy(initialGasPrice, 1*time.Second, 30)

		// Set gas price below what would be the reduced amount
		strategy.gasPrice = 1.0 // Lower than initial
		gasMultiplier := 2.0

		strategy.ResetOnSuccess(gasMultiplier)

		// Gas price should be reset to initial, not go lower
		require.Equal(t, initialGasPrice, strategy.gasPrice)
	})

	t.Run("BackoffOnMempool", func(t *testing.T) {
		strategy := newRetryStrategy(1.0, 10*time.Second, 30)

		mempoolTTL := 25
		blockTime := 1 * time.Second
		gasMultiplier := 1.5

		strategy.BackoffOnMempool(mempoolTTL, blockTime, gasMultiplier)

		// Should set backoff to blockTime * mempoolTTL
		expectedBackoff := blockTime * time.Duration(mempoolTTL)
		require.Equal(t, expectedBackoff, strategy.backoff)

		// Should increase gas price
		expectedGasPrice := 1.0 * gasMultiplier
		require.Equal(t, expectedGasPrice, strategy.gasPrice)
	})

}

// TestSubmitHalfBatch tests all scenarios for submitHalfBatch function using table-driven tests
func TestSubmitHalfBatch(t *testing.T) {
	tests := []struct {
		name                string
		items               []string
		marshaled           [][]byte
		mockSetup           func(*mocks.MockDA)
		expectedSubmitted   int
		expectError         bool
		expectedErrorMsg    string
		postSubmitValidator func(*testing.T, [][]string) // validates postSubmit calls
	}{
		{
			name:              "EmptyItems",
			items:             []string{},
			marshaled:         [][]byte{},
			mockSetup:         func(da *mocks.MockDA) {}, // no DA calls expected
			expectedSubmitted: 0,
			expectError:       false,
			postSubmitValidator: func(t *testing.T, calls [][]string) {
				assert.Empty(t, calls, "postSubmit should not be called for empty items")
			},
		},
		{
			name:      "FullSuccess",
			items:     []string{"item1", "item2", "item3"},
			marshaled: [][]byte{[]byte("m1"), []byte("m2"), []byte("m3")},
			mockSetup: func(da *mocks.MockDA) {
				da.On("SubmitWithOptions", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]coreda.ID{getDummyID(1, []byte("id1")), getDummyID(1, []byte("id2")), getDummyID(1, []byte("id3"))}, nil).Once()
			},
			expectedSubmitted: 3,
			expectError:       false,
			postSubmitValidator: func(t *testing.T, calls [][]string) {
				require.Len(t, calls, 1, "postSubmit should be called once")
				assert.Equal(t, []string{"item1", "item2", "item3"}, calls[0])
			},
		},
		{
			name:      "PartialSuccess",
			items:     []string{"item1", "item2", "item3"},
			marshaled: [][]byte{[]byte("m1"), []byte("m2"), []byte("m3")},
			mockSetup: func(da *mocks.MockDA) {
				// First call: submit 2 out of 3 items
				da.On("SubmitWithOptions", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]coreda.ID{getDummyID(1, []byte("id1")), getDummyID(1, []byte("id2"))}, nil).Once()
				// Second call (recursive): submit remaining 1 item
				da.On("SubmitWithOptions", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]coreda.ID{getDummyID(1, []byte("id3"))}, nil).Once()
			},
			expectedSubmitted: 3,
			expectError:       false,
			postSubmitValidator: func(t *testing.T, calls [][]string) {
				require.Len(t, calls, 2, "postSubmit should be called twice")
				assert.Equal(t, []string{"item1", "item2"}, calls[0])
				assert.Equal(t, []string{"item3"}, calls[1])
			},
		},
		{
			name:      "PartialSuccessWithRecursionError",
			items:     []string{"item1", "item2", "item3"},
			marshaled: [][]byte{[]byte("m1"), []byte("m2"), []byte("m3")},
			mockSetup: func(da *mocks.MockDA) {
				// First call: submit 2 out of 3 items successfully
				da.On("SubmitWithOptions", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]coreda.ID{getDummyID(1, []byte("id1")), getDummyID(1, []byte("id2"))}, nil).Once()
				// Second call (recursive): remaining item is too big
				da.On("SubmitWithOptions", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil, coreda.ErrBlobSizeOverLimit).Once()
			},
			expectedSubmitted: 2,
			expectError:       true,
			expectedErrorMsg:  "single test item exceeds DA blob size limit",
			postSubmitValidator: func(t *testing.T, calls [][]string) {
				require.Len(t, calls, 1, "postSubmit should be called once for successful part")
				assert.Equal(t, []string{"item1", "item2"}, calls[0])
			},
		},
		{
			name:      "SingleItemTooLarge",
			items:     []string{"large_item"},
			marshaled: [][]byte{[]byte("large_marshaled_data")},
			mockSetup: func(da *mocks.MockDA) {
				da.On("SubmitWithOptions", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil, coreda.ErrBlobSizeOverLimit).Once()
			},
			expectedSubmitted: 0,
			expectError:       true,
			expectedErrorMsg:  "single test item exceeds DA blob size limit",
			postSubmitValidator: func(t *testing.T, calls [][]string) {
				assert.Empty(t, calls, "postSubmit should not be called on error")
			},
		},
		{
			name:      "UnrecoverableError",
			items:     []string{"item1", "item2"},
			marshaled: [][]byte{[]byte("m1"), []byte("m2")},
			mockSetup: func(da *mocks.MockDA) {
				da.On("SubmitWithOptions", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil, fmt.Errorf("network timeout")).Once()
			},
			expectedSubmitted: 0,
			expectError:       true,
			expectedErrorMsg:  "unrecoverable error during test batch submission",
			postSubmitValidator: func(t *testing.T, calls [][]string) {
				assert.Empty(t, calls, "postSubmit should not be called on failure")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			da := &mocks.MockDA{}
			m := newTestManagerWithDA(t, da)
			ctx := context.Background()

			// Track postSubmit calls
			var postSubmitCalls [][]string
			postSubmit := func(submitted []string, res *coreda.ResultSubmit, gasPrice float64) {
				postSubmitCalls = append(postSubmitCalls, submitted)
				assert.Equal(t, float64(1.0), gasPrice, "gasPrice should be 1.0")
			}

			// Setup DA mock
			tt.mockSetup(da)

			// Call submitHalfBatch
			submitted, err := submitHalfBatch(m, ctx, tt.items, tt.marshaled, 1.0, postSubmit, "test")

			// Validate results
			if tt.expectError {
				assert.Error(t, err, "Expected error for test case %s", tt.name)
				if tt.expectedErrorMsg != "" {
					assert.Contains(t, err.Error(), tt.expectedErrorMsg, "Error message should contain expected text")
				}
			} else {
				assert.NoError(t, err, "Expected no error for test case %s", tt.name)
			}

			assert.Equal(t, tt.expectedSubmitted, submitted, "Submitted count should match expected")

			// Validate postSubmit calls
			if tt.postSubmitValidator != nil {
				tt.postSubmitValidator(t, postSubmitCalls)
			}

			da.AssertExpectations(t)
		})
	}
}
