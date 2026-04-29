package submitting

import (
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/config"
	datypes "github.com/evstack/ev-node/pkg/da/types"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/test/mocks"
)

func newTestBatchSubmitter(t *testing.T, mockClient *mocks.MockClient, override func(*config.Config)) *DASubmitter {
	t.Helper()
	cfg := config.Config{}
	cfg.DA.BlockTime.Duration = 1 * time.Millisecond
	cfg.DA.MaxSubmitAttempts = 3
	cfg.DA.SubmitOptions = "opts"
	cfg.DA.Namespace = "ns"
	cfg.DA.DataNamespace = "ns-data"
	if override != nil {
		override(&cfg)
	}
	if mockClient == nil {
		mockClient = mocks.NewMockClient(t)
	}
	mockClient.On("GetHeaderNamespace").Return([]byte(cfg.DA.Namespace)).Maybe()
	mockClient.On("GetDataNamespace").Return([]byte(cfg.DA.DataNamespace)).Maybe()
	mockClient.On("GetForcedInclusionNamespace").Return([]byte(nil)).Maybe()
	mockClient.On("HasForcedInclusionNamespace").Return(false).Maybe()
	return NewDASubmitter(mockClient, cfg, genesis.Genesis{}, common.BlockOptions{}, common.NopMetrics(), zerolog.Nop(), nil, nil)
}

func TestSubmitWithRetry_MempoolRetry_Succeeds(t *testing.T) {
	t.Parallel()

	client := mocks.NewMockClient(t)
	nsBz := datypes.NamespaceFromString("ns").Bytes()

	client.On("Submit", mock.Anything, mock.Anything, mock.AnythingOfType("float64"), nsBz, mock.Anything).
		Return(datypes.ResultSubmit{BaseResult: datypes.BaseResult{Code: datypes.StatusNotIncludedInBlock}}).
		Once()

	client.On("Submit", mock.Anything, mock.Anything, mock.AnythingOfType("float64"), nsBz, mock.Anything).
		Return(datypes.ResultSubmit{BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, SubmittedCount: 2}}).
		Once()

	s := newTestBatchSubmitter(t, client, nil)
	defer s.Close()

	var submittedCount int
	s.submitWithRetry(t.Context(), [][]byte{[]byte("a"), []byte("b")}, nsBz, func(count int, _ uint64) {
		submittedCount = count
	}, nil, "item")

	require.Equal(t, 2, submittedCount)
	client.AssertExpectations(t)
}

func TestSubmitWithRetry_UnknownError_RetriesThenSucceeds(t *testing.T) {
	t.Parallel()

	client := mocks.NewMockClient(t)
	nsBz := datypes.NamespaceFromString("ns").Bytes()

	client.On("Submit", mock.Anything, mock.Anything, mock.AnythingOfType("float64"), nsBz, mock.Anything).
		Return(datypes.ResultSubmit{BaseResult: datypes.BaseResult{Code: datypes.StatusError, Message: "boom"}}).
		Once()

	client.On("Submit", mock.Anything, mock.Anything, mock.AnythingOfType("float64"), nsBz, mock.Anything).
		Return(datypes.ResultSubmit{BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, SubmittedCount: 1}}).
		Once()

	s := newTestBatchSubmitter(t, client, nil)
	defer s.Close()

	var submittedCount int
	s.submitWithRetry(t.Context(), [][]byte{[]byte("x")}, nsBz, func(count int, _ uint64) {
		submittedCount = count
	}, nil, "item")

	require.Equal(t, 1, submittedCount)
	client.AssertExpectations(t)
}

func TestSubmitWithRetry_TooBig_HalvesBatch(t *testing.T) {
	t.Parallel()

	client := mocks.NewMockClient(t)
	nsBz := datypes.NamespaceFromString("ns").Bytes()
	var batchSizes []int

	client.On("Submit", mock.Anything, mock.Anything, mock.Anything, nsBz, mock.Anything).
		Run(func(args mock.Arguments) {
			blobs := args.Get(1).([][]byte)
			batchSizes = append(batchSizes, len(blobs))
		}).
		Return(datypes.ResultSubmit{BaseResult: datypes.BaseResult{Code: datypes.StatusTooBig}}).
		Once()

	client.On("Submit", mock.Anything, mock.Anything, mock.Anything, nsBz, mock.Anything).
		Run(func(args mock.Arguments) {
			blobs := args.Get(1).([][]byte)
			batchSizes = append(batchSizes, len(blobs))
		}).
		Return(datypes.ResultSubmit{BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, SubmittedCount: 2}}).
		Once()

	s := newTestBatchSubmitter(t, client, nil)
	defer s.Close()

	s.submitWithRetry(t.Context(), [][]byte{[]byte("a"), []byte("b"), []byte("c"), []byte("d")}, nsBz, nil, nil, "item")

	require.Equal(t, []int{4, 2}, batchSizes)
	client.AssertExpectations(t)
}

func TestSubmitWithRetry_PartialSuccess_Advances(t *testing.T) {
	t.Parallel()

	client := mocks.NewMockClient(t)
	nsBz := datypes.NamespaceFromString("ns").Bytes()

	client.On("Submit", mock.Anything, mock.Anything, mock.Anything, nsBz, mock.Anything).
		Return(datypes.ResultSubmit{BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, SubmittedCount: 2}}).
		Once()

	client.On("Submit", mock.Anything, mock.Anything, mock.Anything, nsBz, mock.Anything).
		Return(datypes.ResultSubmit{BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, SubmittedCount: 1}}).
		Once()

	s := newTestBatchSubmitter(t, client, nil)
	defer s.Close()

	var totalSubmitted int
	s.submitWithRetry(t.Context(), [][]byte{[]byte("a"), []byte("b"), []byte("c")}, nsBz, func(count int, _ uint64) {
		totalSubmitted += count
	}, nil, "item")

	require.Equal(t, 3, totalSubmitted)
	client.AssertExpectations(t)
}

func TestSubmitWithRetry_MaxAttempts_Exhausted(t *testing.T) {
	t.Parallel()

	client := mocks.NewMockClient(t)
	nsBz := datypes.NamespaceFromString("ns").Bytes()

	client.On("Submit", mock.Anything, mock.Anything, mock.Anything, nsBz, mock.Anything).
		Return(datypes.ResultSubmit{BaseResult: datypes.BaseResult{Code: datypes.StatusError, Message: "fail"}}).
		Times(3)

	s := newTestBatchSubmitter(t, client, nil)
	defer s.Close()

	var errReceived error
	s.submitWithRetry(t.Context(), [][]byte{[]byte("a")}, nsBz, nil, func(err error) {
		errReceived = err
	}, "item")

	require.Error(t, errReceived)
	client.AssertExpectations(t)
}

func TestSubmitWithRetry_AlreadyInMempool_Retries(t *testing.T) {
	t.Parallel()

	client := mocks.NewMockClient(t)
	nsBz := datypes.NamespaceFromString("ns").Bytes()

	client.On("Submit", mock.Anything, mock.Anything, mock.Anything, nsBz, mock.Anything).
		Return(datypes.ResultSubmit{BaseResult: datypes.BaseResult{Code: datypes.StatusAlreadyInMempool}}).
		Once()

	client.On("Submit", mock.Anything, mock.Anything, mock.Anything, nsBz, mock.Anything).
		Return(datypes.ResultSubmit{BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, SubmittedCount: 1}}).
		Once()

	s := newTestBatchSubmitter(t, client, nil)
	defer s.Close()

	var submittedCount int
	s.submitWithRetry(t.Context(), [][]byte{[]byte("a")}, nsBz, func(count int, _ uint64) {
		submittedCount = count
	}, nil, "item")

	require.Equal(t, 1, submittedCount)
	client.AssertExpectations(t)
}

func TestSubmitWithRetry_EmptyBatch_Noop(t *testing.T) {
	t.Parallel()

	client := mocks.NewMockClient(t)
	s := newTestBatchSubmitter(t, client, nil)
	defer s.Close()

	var errReceived error
	s.submitWithRetry(t.Context(), nil, nil, nil, func(err error) {
		errReceived = err
	}, "item")

	require.NoError(t, errReceived)
}

func TestSubmitWithRetry_ContextCanceled_Stops(t *testing.T) {
	t.Parallel()

	client := mocks.NewMockClient(t)
	nsBz := datypes.NamespaceFromString("ns").Bytes()

	client.On("Submit", mock.Anything, mock.Anything, mock.Anything, nsBz, mock.Anything).
		Return(datypes.ResultSubmit{BaseResult: datypes.BaseResult{Code: datypes.StatusContextCanceled}}).
		Once()

	s := newTestBatchSubmitter(t, client, nil)
	defer s.Close()

	var errReceived error
	s.submitWithRetry(t.Context(), [][]byte{[]byte("a")}, nsBz, nil, func(err error) {
		errReceived = err
	}, "item")

	require.NoError(t, errReceived)
	client.AssertExpectations(t)
}

func TestSubmitWithRetry_SingleItemTooBig_Fails(t *testing.T) {
	t.Parallel()

	client := mocks.NewMockClient(t)
	nsBz := datypes.NamespaceFromString("ns").Bytes()

	client.On("Submit", mock.Anything, mock.Anything, mock.Anything, nsBz, mock.Anything).
		Return(datypes.ResultSubmit{BaseResult: datypes.BaseResult{Code: datypes.StatusTooBig}}).
		Once()

	s := newTestBatchSubmitter(t, client, nil)
	defer s.Close()

	var errReceived error
	s.submitWithRetry(t.Context(), [][]byte{[]byte("a")}, nsBz, nil, func(err error) {
		errReceived = err
	}, "item")

	require.ErrorIs(t, errReceived, common.ErrOversizedItem)
	client.AssertExpectations(t)
}
