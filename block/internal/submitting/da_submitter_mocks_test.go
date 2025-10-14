package submitting

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/evstack/ev-node/block/internal/common"
	coreda "github.com/evstack/ev-node/core/da"
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

func TestSubmitToDA_MempoolRetry_IncreasesGasAndSucceeds(t *testing.T) {
	t.Parallel()

	mockDA := mocks.NewMockDA(t)
	// Initial gas price fetched from DA since cfg.DA.GasPrice==0
	mockDA.On("GasPrice", mock.Anything).Return(1.0, nil).Once()
	// Gas multiplier available and used on mempool retry
	mockDA.On("GasMultiplier", mock.Anything).Return(2.0, nil).Once()

	// First attempt returns a mempool-related error (mapped to StatusNotIncludedInBlock)
	// Expect gasPrice=1.0

	nsBz := coreda.NamespaceFromString("ns").Bytes()

	opts := []byte("opts")
	// capture gas prices used
	var usedGas []float64
	mockDA.
		On("SubmitWithOptions", mock.Anything, mock.Anything, mock.AnythingOfType("float64"), nsBz, opts).
		Run(func(args mock.Arguments) {
			usedGas = append(usedGas, args.Get(2).(float64))
		}).
		Return(nil, coreda.ErrTxTimedOut).
		Once()

	// Second attempt should use doubled gas price = 2.0 and succeed for all items
	ids := [][]byte{[]byte("id1"), []byte("id2"), []byte("id3")}
	mockDA.
		On("SubmitWithOptions", mock.Anything, mock.Anything, mock.AnythingOfType("float64"), nsBz, opts).
		Run(func(args mock.Arguments) {
			usedGas = append(usedGas, args.Get(2).(float64))
		}).
		Return(ids, nil).
		Once()

	s := newTestSubmitter(mockDA, func(c *config.Config) {
		// trigger initial gas price path from DA
		c.DA.GasPrice = 0
	})

	items := []string{"a", "b", "c"}
	ctx := context.Background()
	err := submitToDA[string](
		s,
		ctx,
		items,
		marshalString,
		func(_ []string, _ *coreda.ResultSubmit, _ float64) {},
		"item",
		nsBz,
		opts,
		nil,
		nil,
	)
	assert.NoError(t, err)

	// Expect two attempts with gas 1.0 then 2.0
	assert.Equal(t, []float64{1.0, 2.0}, usedGas)
	mockDA.AssertExpectations(t)
}

func TestSubmitToDA_UnknownError_RetriesSameGasThenSucceeds(t *testing.T) {
	t.Parallel()

	mockDA := mocks.NewMockDA(t)
	// Initial gas price comes from config (set below), so DA.GasPrice is not called
	mockDA.On("GasMultiplier", mock.Anything).Return(3.0, nil).Once()

	nsBz := coreda.NamespaceFromString("ns").Bytes()

	opts := []byte("opts")
	var usedGas []float64

	// First attempt: unknown failure -> reasonFailure, gas unchanged for next attempt
	mockDA.
		On("SubmitWithOptions", mock.Anything, mock.Anything, mock.AnythingOfType("float64"), nsBz, opts).
		Run(func(args mock.Arguments) { usedGas = append(usedGas, args.Get(2).(float64)) }).
		Return(nil, errors.New("boom")).
		Once()

	// Second attempt: same gas, success
	ids := [][]byte{[]byte("id1")}
	mockDA.
		On("SubmitWithOptions", mock.Anything, mock.Anything, mock.AnythingOfType("float64"), nsBz, opts).
		Run(func(args mock.Arguments) { usedGas = append(usedGas, args.Get(2).(float64)) }).
		Return(ids, nil).
		Once()

	s := newTestSubmitter(mockDA, func(c *config.Config) {
		c.DA.GasPrice = 5.5 // fixed gas from config
	})

	items := []string{"x"}
	ctx := context.Background()
	err := submitToDA[string](
		s,
		ctx,
		items,
		marshalString,
		func(_ []string, _ *coreda.ResultSubmit, _ float64) {},
		"item",
		nsBz,
		opts,
		nil,
		nil,
	)
	assert.NoError(t, err)
	assert.Equal(t, []float64{5.5, 5.5}, usedGas)
	mockDA.AssertExpectations(t)
}

func TestSubmitToDA_TooBig_HalvesBatch(t *testing.T) {
	t.Parallel()

	mockDA := mocks.NewMockDA(t)
	// Use fixed gas from config to simplify
	mockDA.On("GasMultiplier", mock.Anything).Return(2.0, nil).Once()

	nsBz := coreda.NamespaceFromString("ns").Bytes()

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
		Return(nil, coreda.ErrBlobSizeOverLimit).
		Once()

	// Second attempt: expect half the size, succeed
	ids := [][]byte{[]byte("id1"), []byte("id2")}
	mockDA.
		On("SubmitWithOptions", mock.Anything, mock.Anything, mock.Anything, nsBz, opts).
		Run(func(args mock.Arguments) {
			blobs := args.Get(1).([][]byte)
			batchSizes = append(batchSizes, len(blobs))
		}).
		Return(ids, nil).
		Once()

	s := newTestSubmitter(mockDA, func(c *config.Config) {
		c.DA.GasPrice = 1.0
	})

	items := []string{"a", "b", "c", "d"}
	ctx := context.Background()
	err := submitToDA[string](
		s,
		ctx,
		items,
		marshalString,
		func(_ []string, _ *coreda.ResultSubmit, _ float64) {},
		"item",
		nsBz,
		opts,
		nil,
		nil,
	)
	assert.NoError(t, err)
	assert.Equal(t, []int{4, 2}, batchSizes)
	mockDA.AssertExpectations(t)
}

func TestSubmitToDA_SentinelNoGas_PreservesGasAcrossRetries(t *testing.T) {
	t.Parallel()

	mockDA := mocks.NewMockDA(t)
	// GasMultiplier is still called once, but should not affect gas when sentinel is used
	mockDA.On("GasMultiplier", mock.Anything).Return(10.0, nil).Once()

	nsBz := coreda.NamespaceFromString("ns").Bytes()

	opts := []byte("opts")
	var usedGas []float64

	// First attempt: mempool-ish error
	mockDA.
		On("SubmitWithOptions", mock.Anything, mock.Anything, mock.AnythingOfType("float64"), nsBz, opts).
		Run(func(args mock.Arguments) { usedGas = append(usedGas, args.Get(2).(float64)) }).
		Return(nil, coreda.ErrTxAlreadyInMempool).
		Once()

	// Second attempt: should use same sentinel gas (-1), succeed
	ids := [][]byte{[]byte("id1")}
	mockDA.
		On("SubmitWithOptions", mock.Anything, mock.Anything, mock.AnythingOfType("float64"), nsBz, opts).
		Run(func(args mock.Arguments) { usedGas = append(usedGas, args.Get(2).(float64)) }).
		Return(ids, nil).
		Once()

	s := newTestSubmitter(mockDA, func(c *config.Config) {
		c.DA.GasPrice = -1 // sentinel no-gas behavior
	})

	items := []string{"only"}
	ctx := context.Background()
	err := submitToDA[string](
		s,
		ctx,
		items,
		marshalString,
		func(_ []string, _ *coreda.ResultSubmit, _ float64) {},
		"item",
		nsBz,
		opts,
		nil,
		nil,
	)
	assert.NoError(t, err)
	assert.Equal(t, []float64{-1, -1}, usedGas)
	mockDA.AssertExpectations(t)
}

func TestSubmitToDA_PartialSuccess_AdvancesWindow(t *testing.T) {
	t.Parallel()

	mockDA := mocks.NewMockDA(t)
	mockDA.On("GasMultiplier", mock.Anything).Return(2.0, nil).Once()

	nsBz := coreda.NamespaceFromString("ns").Bytes()

	opts := []byte("opts")
	// track how many items postSubmit sees across attempts
	var totalSubmitted int

	// First attempt: success for first 2 of 3
	firstIDs := [][]byte{[]byte("id1"), []byte("id2")}
	mockDA.On("SubmitWithOptions", mock.Anything, mock.Anything, mock.Anything, nsBz, opts).Return(firstIDs, nil).Once()

	// Second attempt: success for remaining 1
	secondIDs := [][]byte{[]byte("id3")}
	mockDA.On("SubmitWithOptions", mock.Anything, mock.Anything, mock.Anything, nsBz, opts).Return(secondIDs, nil).Once()

	s := newTestSubmitter(mockDA, func(c *config.Config) { c.DA.GasPrice = 1.0 })

	items := []string{"a", "b", "c"}
	ctx := context.Background()
	err := submitToDA[string](
		s,
		ctx,
		items,
		marshalString,
		func(submitted []string, _ *coreda.ResultSubmit, _ float64) { totalSubmitted += len(submitted) },
		"item",
		nsBz,
		opts,
		nil,
		nil,
	)
	assert.NoError(t, err)
	assert.Equal(t, 3, totalSubmitted)
	mockDA.AssertExpectations(t)
}
