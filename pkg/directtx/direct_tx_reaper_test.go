package directtx

import (
	"context"
	"testing"
	"time"

	"github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/types"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockDA is a simple mock implementation for testing
type MockDA struct {
	mock.Mock
}

func (m *MockDA) GetIDs(ctx context.Context, height uint64, namespace []byte) (*da.GetIDsResult, error) {
	args := m.Called(ctx, height, namespace)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*da.GetIDsResult), args.Error(1)
}

func (m *MockDA) Get(ctx context.Context, ids []da.ID, namespace []byte) ([]da.Blob, error) {
	args := m.Called(ctx, ids, namespace)
	return args.Get(0).([]da.Blob), args.Error(1)
}

func (m *MockDA) GetProofs(ctx context.Context, ids []da.ID, namespace []byte) ([]da.Proof, error) {
	args := m.Called(ctx, ids, namespace)
	return args.Get(0).([]da.Proof), args.Error(1)
}

func (m *MockDA) Commit(ctx context.Context, blobs []da.Blob, namespace []byte) ([]da.Commitment, error) {
	args := m.Called(ctx, blobs, namespace)
	return args.Get(0).([]da.Commitment), args.Error(1)
}

func (m *MockDA) Submit(ctx context.Context, blobs []da.Blob, gasPrice float64, namespace []byte) ([]da.ID, error) {
	args := m.Called(ctx, blobs, gasPrice, namespace)
	return args.Get(0).([]da.ID), args.Error(1)
}

func (m *MockDA) SubmitWithOptions(ctx context.Context, blobs []da.Blob, gasPrice float64, namespace []byte, options []byte) ([]da.ID, error) {
	args := m.Called(ctx, blobs, gasPrice, namespace, options)
	return args.Get(0).([]da.ID), args.Error(1)
}

func (m *MockDA) Validate(ctx context.Context, ids []da.ID, proofs []da.Proof, namespace []byte) ([]bool, error) {
	args := m.Called(ctx, ids, proofs, namespace)
	return args.Get(0).([]bool), args.Error(1)
}

func (m *MockDA) GasPrice(ctx context.Context) (float64, error) {
	args := m.Called(ctx)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockDA) GasMultiplier(ctx context.Context) (float64, error) {
	args := m.Called(ctx)
	return args.Get(0).(float64), args.Error(1)
}

func TestSubmitTxs(t *testing.T) {
	tests := map[string]struct {
		setupMocks    func(*MockDA, *MockDirectTxSequencer, *DirectTxReaper)
		expectedError bool
		verifyMocks   func(*testing.T, *MockDA, *MockDirectTxSequencer)
	}{
		"Empty blobs": {
			setupMocks: func(mockDA *MockDA, mockSequencer *MockDirectTxSequencer, reaper *DirectTxReaper) {
				mockDA.On("GetIDs", mock.Anything, uint64(1), mock.Anything).Return(
					&da.GetIDsResult{
						IDs:       []da.ID{},
						Timestamp: time.Now(),
					},
					nil,
				)
			},
			expectedError: false,
			verifyMocks: func(t *testing.T, mockDA *MockDA, mockSequencer *MockDirectTxSequencer) {
				mockDA.AssertExpectations(t)
				mockSequencer.AssertNotCalled(t, "SubmitDirectTxs")
			},
		},
		"With new transactions": {
			setupMocks: func(mockDA *MockDA, mockSequencer *MockDirectTxSequencer, reaper *DirectTxReaper) {
				tx1 := []byte("tx1")
				tx2 := []byte("tx2")
				blob := createTestData("test-chain", [][]byte{tx1, tx2})

				mockDA.On("GetIDs", mock.Anything, uint64(1), mock.Anything).Return(
					&da.GetIDsResult{
						IDs:       []da.ID{[]byte("id1")},
						Timestamp: time.Now(),
					},
					nil,
				)
				mockDA.On("Get", mock.Anything, []da.ID{[]byte("id1")}, mock.Anything).Return(
					[]da.Blob{blob},
					nil,
				)
				mockSequencer.On("SubmitDirectTxs", mock.Anything, mock.MatchedBy(func(txs []DirectTX) bool {
					return len(txs) == 2 && string(txs[0].TX) == string(tx1) && string(txs[1].TX) == string(tx2)
				})).Return(nil)
			},
			expectedError: false,
			verifyMocks: func(t *testing.T, mockDA *MockDA, mockSequencer *MockDirectTxSequencer) {
				mockDA.AssertExpectations(t)
				mockSequencer.AssertExpectations(t)
			},
		},
		"With already seen transactions": {
			setupMocks: func(mockDA *MockDA, mockSequencer *MockDirectTxSequencer, reaper *DirectTxReaper) {
				tx1 := []byte("tx1")
				tx2 := []byte("tx2")
				blob := createTestData("test-chain", [][]byte{tx1, tx2})

				mockDA.On("GetIDs", mock.Anything, uint64(1), mock.Anything).Return(
					&da.GetIDsResult{
						IDs:       []da.ID{[]byte("id1")},
						Timestamp: time.Now(),
					},
					nil,
				)
				mockDA.On("Get", mock.Anything, []da.ID{[]byte("id1")}, mock.Anything).Return(
					[]da.Blob{blob},
					nil,
				)
				// Pre-populate the extractor's seenStore with tx1
				dTx1 := DirectTX{
					TX:              tx1,
					ID:              []byte("id1"),
					FirstSeenHeight: 1,
					FirstSeenTime:   time.Now().Unix(),
				}
				key := buildStoreKey(dTx1)
				err := reaper.directTXExtractor.seenStore.Put(context.Background(), key, []byte{1})
				require.NoError(t, err)
				mockSequencer.On("SubmitDirectTxs", mock.Anything, mock.MatchedBy(func(txs []DirectTX) bool {
					return len(txs) == 1 && string(txs[0].TX) == string(tx2)
				})).Return(nil)
			},
			expectedError: false,
			verifyMocks: func(t *testing.T, mockDA *MockDA, mockSequencer *MockDirectTxSequencer) {
				mockDA.AssertExpectations(t)
				mockSequencer.AssertExpectations(t)
			},
		},
		"With different chain ID": {
			setupMocks: func(mockDA *MockDA, mockSequencer *MockDirectTxSequencer, reaper *DirectTxReaper) {
				tx1 := []byte("tx1")
				tx2 := []byte("tx2")
				blob := createTestData("different-chain", [][]byte{tx1, tx2})

				mockDA.On("GetIDs", mock.Anything, uint64(1), mock.Anything).Return(
					&da.GetIDsResult{
						IDs:       []da.ID{[]byte("id1")},
						Timestamp: time.Now(),
					},
					nil,
				)
				mockDA.On("Get", mock.Anything, []da.ID{[]byte("id1")}, mock.Anything).Return(
					[]da.Blob{blob},
					nil,
				)
			},
			expectedError: false,
			verifyMocks: func(t *testing.T, mockDA *MockDA, mockSequencer *MockDirectTxSequencer) {
				mockDA.AssertExpectations(t)
				mockSequencer.AssertNotCalled(t, "SubmitDirectTxs")
			},
		},
		"Height from future error": {
			setupMocks: func(mockDA *MockDA, mockSequencer *MockDirectTxSequencer, reaper *DirectTxReaper) {
				mockDA.On("GetIDs", mock.Anything, uint64(1), mock.Anything).Return(
					(*da.GetIDsResult)(nil),
					da.ErrHeightFromFuture,
				)
			},
			expectedError: true,
			verifyMocks: func(t *testing.T, mockDA *MockDA, mockSequencer *MockDirectTxSequencer) {
				mockDA.AssertExpectations(t)
				mockSequencer.AssertNotCalled(t, "SubmitDirectTxs")
			},
		},
		"Generic error": {
			setupMocks: func(mockDA *MockDA, mockSequencer *MockDirectTxSequencer, reaper *DirectTxReaper) {
				mockDA.On("GetIDs", mock.Anything, uint64(1), mock.Anything).Return(
					(*da.GetIDsResult)(nil),
					assert.AnError,
				)
			},
			expectedError: true,
			verifyMocks: func(t *testing.T, mockDA *MockDA, mockSequencer *MockDirectTxSequencer) {
				mockDA.AssertExpectations(t)
				mockSequencer.AssertNotCalled(t, "SubmitDirectTxs")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockDA := new(MockDA)
			mockSequencer := new(MockDirectTxSequencer)
			chainID := "test-chain"

			reaper := createTestDirectTxReaper(t, mockDA, mockSequencer, chainID)
			tt.setupMocks(mockDA, mockSequencer, reaper)

			err := reaper.retrieveDirectTXs(1)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			tt.verifyMocks(t, mockDA, mockSequencer)
		})
	}
}

// MockDirectTxSequencer is a mock implementation of the DirectTxSequencer interface
type MockDirectTxSequencer struct {
	mock.Mock
}

func (m *MockDirectTxSequencer) SubmitDirectTxs(ctx context.Context, txs ...DirectTX) error {
	args := m.Called(ctx, txs)
	return args.Error(0)
}

// Helper function to create a test DirectTxReaper
func createTestDirectTxReaper(
	t *testing.T,
	mockDA *MockDA,
	mockSequencer *MockDirectTxSequencer,
	chainID string,
) *DirectTxReaper {
	t.Helper()
	ctx := context.Background()
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	ds := sync.MutexWrap(datastore.NewMapDatastore()) // In-memory thread-safe datastore

	extractor := NewExtractor(mockSequencer, chainID, logger, ds)
	return NewDirectTxReaper(ctx, mockDA, extractor, time.Second, logger, 1)
}

// Helper function to create a test Data object
func createTestData(chainID string, txs [][]byte) []byte {
	// Convert [][]byte to types.Txs
	var typeTxs types.Txs
	for _, tx := range txs {
		typeTxs = append(typeTxs, tx)
	}

	data := types.Data{
		Metadata: &types.Metadata{
			ChainID: chainID,
		},
		Txs: typeTxs,
	}

	bz, _ := data.MarshalBinary()
	return bz
}
