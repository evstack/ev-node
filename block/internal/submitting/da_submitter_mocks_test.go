package submitting

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/evstack/ev-node/block/internal/common"
	da "github.com/evstack/ev-node/da"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/test/mocks"
)

// helper to build a basic submitter with provided DA mock and config overrides
func newTestSubmitter(mockDA *mocks.MockDA, override func(*config.Config)) *DASubmitter {
	cfg := config.Config{}
	// Keep retries small and backoffs minimal
	cfg.DA.BlockTime.Duration = 1 * time.Millisecond
	cfg.DA.MaxSubmitAttempts = 3
	cfg.DA.SubmitOptions = "opts"
	cfg.DA.Namespace = "ns"
	if override != nil {
		override(&cfg)
	}
	return NewDASubmitter(mockDA, cfg, genesis.Genesis{} /*options=*/, common.BlockOptions{}, common.NopMetrics(), zerolog.Nop())
}

// marshal helper for simple items
func marshalString(s string) ([]byte, error) { return []byte(s), nil }

// helper to create a ResultSubmit for errors
func errorResult(code da.StatusCode, msg string) da.ResultSubmit {
	return da.ResultSubmit{
		BaseResult: da.BaseResult{
			Code:    code,
			Message: msg,
		},
	}
}

// helper to create a ResultSubmit for success
func successResult(ids []da.ID) da.ResultSubmit {
	return da.ResultSubmit{
		BaseResult: da.BaseResult{
			Code:           da.StatusSuccess,
			IDs:            ids,
			SubmittedCount: uint64(len(ids)),
		},
	}
}

func TestSubmitToDA_MempoolRetry_IncreasesGasAndSucceeds(t *testing.T) {
	t.Parallel()

	mockDA := mocks.NewMockDA(t)

	nsBz := da.NamespaceFromString("ns").Bytes()
	opts := []byte("opts")
	var usedGas []float64

	// First attempt: timeout error (mapped to StatusNotIncludedInBlock)
	mockDA.
		On("SubmitWithOptions", mock.Anything, mock.Anything, mock.AnythingOfType("float64"), nsBz, opts).
		Run(func(args mock.Arguments) {
			usedGas = append(usedGas, args.Get(2).(float64))
		}).
		Return(errorResult(da.StatusNotIncludedInBlock, "timeout")).
		Once()

	// Second attempt: success
	ids := []da.ID{[]byte("id1"), []byte("id2"), []byte("id3")}
	mockDA.
		On("SubmitWithOptions", mock.Anything, mock.Anything, mock.AnythingOfType("float64"), nsBz, opts).
		Run(func(args mock.Arguments) {
			usedGas = append(usedGas, args.Get(2).(float64))
		}).
		Return(successResult(ids)).
		Once()

	s := newTestSubmitter(mockDA, nil)

	items := []string{"a", "b", "c"}
	ctx := context.Background()
	err := submitToDA[string](
		s,
		ctx,
		items,
		marshalString,
		func(_ []string, _ *da.ResultSubmit) {},
		"item",
		nsBz,
		opts,
		nil,
	)
	assert.NoError(t, err)

	// Sentinel value is preserved on retry
	assert.Equal(t, []float64{-1, -1}, usedGas)
	mockDA.AssertExpectations(t)
}

func TestSubmitToDA_UnknownError_RetriesSameGasThenSucceeds(t *testing.T) {
	t.Parallel()

	mockDA := mocks.NewMockDA(t)

	nsBz := da.NamespaceFromString("ns").Bytes()

	opts := []byte("opts")
	var usedGas []float64

	// First attempt: unknown failure -> reasonFailure, gas unchanged for next attempt
	mockDA.
		On("SubmitWithOptions", mock.Anything, mock.Anything, mock.AnythingOfType("float64"), nsBz, opts).
		Run(func(args mock.Arguments) { usedGas = append(usedGas, args.Get(2).(float64)) }).
		Return(errorResult(da.StatusError, "boom")).
		Once()

	// Second attempt: same gas, success
	ids := []da.ID{[]byte("id1")}
	mockDA.
		On("SubmitWithOptions", mock.Anything, mock.Anything, mock.AnythingOfType("float64"), nsBz, opts).
		Run(func(args mock.Arguments) { usedGas = append(usedGas, args.Get(2).(float64)) }).
		Return(successResult(ids)).
		Once()

	s := newTestSubmitter(mockDA, nil)

	items := []string{"x"}
	ctx := context.Background()
	err := submitToDA[string](
		s,
		ctx,
		items,
		marshalString,
		func(_ []string, _ *da.ResultSubmit) {},
		"item",
		nsBz,
		opts,
		nil,
	)
	assert.NoError(t, err)
	assert.Equal(t, []float64{-1, -1}, usedGas)
	mockDA.AssertExpectations(t)
}

func TestSubmitToDA_TooBig_HalvesBatch(t *testing.T) {
	t.Parallel()

	mockDA := mocks.NewMockDA(t)

	nsBz := da.NamespaceFromString("ns").Bytes()

	opts := []byte("opts")
	// record sizes of batches sent to DA
	var batchSizes []int

	// First attempt: too big -> should halve and retry
	mockDA.
		On("SubmitWithOptions", mock.Anything, mock.Anything, mock.Anything, nsBz, opts).
		Run(func(args mock.Arguments) {
			blobs := args.Get(1).([][]byte)
			batchSizes = append(batchSizes, len(blobs))
		}).
		Return(errorResult(da.StatusTooBig, "blob too big")).
		Once()

	// Second attempt: expect half the size, succeed
	ids := []da.ID{[]byte("id1"), []byte("id2")}
	mockDA.
		On("SubmitWithOptions", mock.Anything, mock.Anything, mock.Anything, nsBz, opts).
		Run(func(args mock.Arguments) {
			blobs := args.Get(1).([][]byte)
			batchSizes = append(batchSizes, len(blobs))
		}).
		Return(successResult(ids)).
		Once()

	s := newTestSubmitter(mockDA, nil)

	items := []string{"a", "b", "c", "d"}
	ctx := context.Background()
	err := submitToDA[string](
		s,
		ctx,
		items,
		marshalString,
		func(_ []string, _ *da.ResultSubmit) {},
		"item",
		nsBz,
		opts,
		nil,
	)
	assert.NoError(t, err)
	assert.Equal(t, []int{4, 2}, batchSizes)
	mockDA.AssertExpectations(t)
}

func TestSubmitToDA_SentinelNoGas_PreservesGasAcrossRetries(t *testing.T) {
	t.Parallel()

	mockDA := mocks.NewMockDA(t)

	nsBz := da.NamespaceFromString("ns").Bytes()

	opts := []byte("opts")
	var usedGas []float64

	// First attempt: mempool-ish error
	mockDA.
		On("SubmitWithOptions", mock.Anything, mock.Anything, mock.AnythingOfType("float64"), nsBz, opts).
		Run(func(args mock.Arguments) { usedGas = append(usedGas, args.Get(2).(float64)) }).
		Return(errorResult(da.StatusAlreadyInMempool, "already in mempool")).
		Once()

	// Second attempt: should use same sentinel gas (-1), succeed
	ids := []da.ID{[]byte("id1")}
	mockDA.
		On("SubmitWithOptions", mock.Anything, mock.Anything, mock.AnythingOfType("float64"), nsBz, opts).
		Run(func(args mock.Arguments) { usedGas = append(usedGas, args.Get(2).(float64)) }).
		Return(successResult(ids)).
		Once()

	s := newTestSubmitter(mockDA, nil)

	items := []string{"only"}
	ctx := context.Background()
	err := submitToDA[string](
		s,
		ctx,
		items,
		marshalString,
		func(_ []string, _ *da.ResultSubmit) {},
		"item",
		nsBz,
		opts,
		nil,
	)
	assert.NoError(t, err)
	assert.Equal(t, []float64{-1, -1}, usedGas)
	mockDA.AssertExpectations(t)
}

func TestSubmitToDA_PartialSuccess_AdvancesWindow(t *testing.T) {
	t.Parallel()

	mockDA := mocks.NewMockDA(t)

	nsBz := da.NamespaceFromString("ns").Bytes()

	opts := []byte("opts")
	// track how many items postSubmit sees across attempts
	var totalSubmitted int

	// First attempt: success for first 2 of 3
	firstIDs := []da.ID{[]byte("id1"), []byte("id2")}
	mockDA.On("SubmitWithOptions", mock.Anything, mock.Anything, mock.Anything, nsBz, opts).Return(successResult(firstIDs)).Once()

	// Second attempt: success for remaining 1
	secondIDs := []da.ID{[]byte("id3")}
	mockDA.On("SubmitWithOptions", mock.Anything, mock.Anything, mock.Anything, nsBz, opts).Return(successResult(secondIDs)).Once()

	s := newTestSubmitter(mockDA, nil)

	items := []string{"a", "b", "c"}
	ctx := context.Background()
	err := submitToDA[string](
		s,
		ctx,
		items,
		marshalString,
		func(submitted []string, _ *da.ResultSubmit) { totalSubmitted += len(submitted) },
		"item",
		nsBz,
		opts,
		nil,
	)
	assert.NoError(t, err)
	assert.Equal(t, 3, totalSubmitted)
	mockDA.AssertExpectations(t)
}
