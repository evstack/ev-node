package block

import (
	"context"
	"github.com/evstack/ev-node/test/mocks"
	"testing"
	"time"

	"github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/types"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	"github.com/ipfs/go-log/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockDA = mocks.MockDA

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
				mockSequencer.On("SubmitDirectTxs", mock.Anything, [][]byte{tx1, tx2}).Return(nil)
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
				key := datastore.NewKey(hashTx(tx1))
				reaper.seenStore.Put(context.Background(), key, []byte{1})
				mockSequencer.On("SubmitDirectTxs", mock.Anything, [][]byte{tx2}).Return(nil)
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
			expectedError: false,
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

			err := reaper.SubmitTxs(1)

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

func (m *MockDirectTxSequencer) SubmitDirectTxs(ctx context.Context, txs [][]byte) error {
	args := m.Called(ctx, txs)
	return args.Error(0)
}

// We need to implement the Sequencer interface methods as well since DirectTxSequencer extends Sequencer
func (m *MockDirectTxSequencer) SubmitBatchTxs(ctx context.Context, req sequencer.SubmitBatchTxsRequest) (*sequencer.SubmitBatchTxsResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*sequencer.SubmitBatchTxsResponse), args.Error(1)
}

func (m *MockDirectTxSequencer) GetNextBatch(ctx context.Context, req sequencer.GetNextBatchRequest) (*sequencer.GetNextBatchResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*sequencer.GetNextBatchResponse), args.Error(1)
}

func (m *MockDirectTxSequencer) VerifyBatch(ctx context.Context, req sequencer.VerifyBatchRequest) (*sequencer.VerifyBatchResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*sequencer.VerifyBatchResponse), args.Error(1)
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
	logger := log.Logger("test-direct-tx-reaper")
	ds := sync.MutexWrap(datastore.NewMapDatastore()) // In-memory thread-safe datastore

	return NewDirectTxReaper(
		ctx,
		mockDA,
		mockSequencer,
		nil, // Manager is not used in SubmitTxs
		chainID,
		time.Second,
		logger,
		ds,
	)
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
