package submitting

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/config"
	datypes "github.com/evstack/ev-node/pkg/da/types"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/test/mocks"
)

// helper to build a basic submitter with provided DA mock client and config overrides
func newTestSubmitter(t *testing.T, mockClient *mocks.MockClient, override func(*config.Config)) *DASubmitter {
	cfg := config.Config{}
	// Keep retries small and backoffs minimal
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
	return NewDASubmitter(mockClient, cfg, genesis.Genesis{} /*options=*/, common.BlockOptions{}, common.NopMetrics(), zerolog.Nop())
}

func TestSubmitToDA_MempoolRetry_IncreasesGasAndSucceeds(t *testing.T) {
	t.Parallel()

	client := mocks.NewMockClient(t)

	nsBz := datypes.NamespaceFromString("ns").Bytes()
	opts := []byte("opts")
	var usedGas []float64

	client.On("Submit", mock.Anything, mock.Anything, mock.AnythingOfType("float64"), nsBz, opts).
		Run(func(args mock.Arguments) {
			usedGas = append(usedGas, args.Get(2).(float64))
		}).
		Return(datypes.ResultSubmit{BaseResult: datypes.BaseResult{Code: datypes.StatusNotIncludedInBlock, SubmittedCount: 0}}).
		Once()

	ids := [][]byte{[]byte("id1"), []byte("id2"), []byte("id3")}
	client.On("Submit", mock.Anything, mock.Anything, mock.AnythingOfType("float64"), nsBz, opts).
		Run(func(args mock.Arguments) {
			usedGas = append(usedGas, args.Get(2).(float64))
		}).
		Return(datypes.ResultSubmit{BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, IDs: ids, SubmittedCount: uint64(len(ids))}}).
		Once()

	s := newTestSubmitter(t, client, nil)

	items := []string{"a", "b", "c"}
	marshalledItems := make([][]byte, 0, len(items))
	for idx, item := range items {
		marshalledItems[idx] = []byte(item)
	}

	ctx := context.Background()
	err := submitToDA[string](
		s,
		ctx,
		items,
		marshalledItems,
		func(_ []string, _ *datypes.ResultSubmit) {},
		"item",
		nsBz,
		opts,
		nil,
	)
	assert.NoError(t, err)

	// Sentinel value is preserved on retry
	assert.Equal(t, []float64{-1, -1}, usedGas)
}

func TestSubmitToDA_UnknownError_RetriesSameGasThenSucceeds(t *testing.T) {
	t.Parallel()

	client := mocks.NewMockClient(t)

	nsBz := datypes.NamespaceFromString("ns").Bytes()

	opts := []byte("opts")
	var usedGas []float64

	// First attempt: unknown failure -> reasonFailure, gas unchanged for next attempt
	client.On("Submit", mock.Anything, mock.Anything, mock.AnythingOfType("float64"), nsBz, opts).
		Run(func(args mock.Arguments) { usedGas = append(usedGas, args.Get(2).(float64)) }).
		Return(datypes.ResultSubmit{BaseResult: datypes.BaseResult{Code: datypes.StatusError, Message: "boom"}}).
		Once()

	// Second attempt: same gas, success
	ids := [][]byte{[]byte("id1")}
	client.On("Submit", mock.Anything, mock.Anything, mock.AnythingOfType("float64"), nsBz, opts).
		Run(func(args mock.Arguments) { usedGas = append(usedGas, args.Get(2).(float64)) }).
		Return(datypes.ResultSubmit{BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, IDs: ids, SubmittedCount: uint64(len(ids))}}).
		Once()

	s := newTestSubmitter(t, client, nil)

	items := []string{"x"}
	marshalledItems := make([][]byte, 0, len(items))
	for idx, item := range items {
		marshalledItems[idx] = []byte(item)
	}

	ctx := context.Background()
	err := submitToDA[string](
		s,
		ctx,
		items,
		marshalledItems,
		func(_ []string, _ *datypes.ResultSubmit) {},
		"item",
		nsBz,
		opts,
		nil,
	)
	assert.NoError(t, err)
	assert.Equal(t, []float64{-1, -1}, usedGas)
}

func TestSubmitToDA_TooBig_HalvesBatch(t *testing.T) {
	t.Parallel()

	client := mocks.NewMockClient(t)

	nsBz := datypes.NamespaceFromString("ns").Bytes()

	opts := []byte("opts")
	var batchSizes []int

	client.On("Submit", mock.Anything, mock.Anything, mock.Anything, nsBz, opts).
		Run(func(args mock.Arguments) {
			blobs := args.Get(1).([][]byte)
			batchSizes = append(batchSizes, len(blobs))
		}).
		Return(datypes.ResultSubmit{BaseResult: datypes.BaseResult{Code: datypes.StatusTooBig}}).
		Once()

	ids := [][]byte{[]byte("id1"), []byte("id2")}
	client.On("Submit", mock.Anything, mock.Anything, mock.Anything, nsBz, opts).
		Run(func(args mock.Arguments) {
			blobs := args.Get(1).([][]byte)
			batchSizes = append(batchSizes, len(blobs))
		}).
		Return(datypes.ResultSubmit{BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, IDs: ids, SubmittedCount: uint64(len(ids))}}).
		Once()

	s := newTestSubmitter(t, client, nil)

	items := []string{"a", "b", "c", "d"}
	marshalledItems := make([][]byte, 0, len(items))
	for idx, item := range items {
		marshalledItems[idx] = []byte(item)
	}

	ctx := context.Background()
	err := submitToDA[string](
		s,
		ctx,
		items,
		marshalledItems,
		func(_ []string, _ *datypes.ResultSubmit) {},
		"item",
		nsBz,
		opts,
		nil,
	)
	assert.NoError(t, err)
	assert.Equal(t, []int{4, 2}, batchSizes)
}

func TestSubmitToDA_SentinelNoGas_PreservesGasAcrossRetries(t *testing.T) {
	t.Parallel()

	client := mocks.NewMockClient(t)

	nsBz := datypes.NamespaceFromString("ns").Bytes()

	opts := []byte("opts")
	var usedGas []float64

	client.On("Submit", mock.Anything, mock.Anything, mock.AnythingOfType("float64"), nsBz, opts).
		Run(func(args mock.Arguments) { usedGas = append(usedGas, args.Get(2).(float64)) }).
		Return(datypes.ResultSubmit{BaseResult: datypes.BaseResult{Code: datypes.StatusAlreadyInMempool}}).
		Once()

	ids := [][]byte{[]byte("id1")}
	client.On("Submit", mock.Anything, mock.Anything, mock.AnythingOfType("float64"), nsBz, opts).
		Run(func(args mock.Arguments) { usedGas = append(usedGas, args.Get(2).(float64)) }).
		Return(datypes.ResultSubmit{BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, IDs: ids, SubmittedCount: uint64(len(ids))}}).
		Once()

	s := newTestSubmitter(t, client, nil)

	items := []string{"only"}
	marshalledItems := make([][]byte, 0, len(items))
	for idx, item := range items {
		marshalledItems[idx] = []byte(item)
	}

	ctx := context.Background()
	err := submitToDA[string](
		s,
		ctx,
		items,
		marshalledItems,
		func(_ []string, _ *datypes.ResultSubmit) {},
		"item",
		nsBz,
		opts,
		nil,
	)
	assert.NoError(t, err)
	assert.Equal(t, []float64{-1, -1}, usedGas)
}

func TestSubmitToDA_PartialSuccess_AdvancesWindow(t *testing.T) {
	t.Parallel()

	client := mocks.NewMockClient(t)

	nsBz := datypes.NamespaceFromString("ns").Bytes()

	opts := []byte("opts")
	var totalSubmitted int

	firstIDs := [][]byte{[]byte("id1"), []byte("id2")}
	client.On("Submit", mock.Anything, mock.Anything, mock.Anything, nsBz, opts).
		Return(datypes.ResultSubmit{BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, IDs: firstIDs, SubmittedCount: uint64(len(firstIDs))}}).
		Once()

	secondIDs := [][]byte{[]byte("id3")}
	client.On("Submit", mock.Anything, mock.Anything, mock.Anything, nsBz, opts).
		Return(datypes.ResultSubmit{BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, IDs: secondIDs, SubmittedCount: uint64(len(secondIDs))}}).
		Once()

	s := newTestSubmitter(t, client, nil)

	items := []string{"a", "b", "c"}
	marshalledItems := make([][]byte, 0, len(items))
	for idx, item := range items {
		marshalledItems[idx] = []byte(item)
	}

	ctx := context.Background()
	err := submitToDA[string](
		s,
		ctx,
		items,
		marshalledItems,
		func(submitted []string, _ *datypes.ResultSubmit) { totalSubmitted += len(submitted) },
		"item",
		nsBz,
		opts,
		nil,
	)
	assert.NoError(t, err)
	assert.Equal(t, 3, totalSubmitted)
}
