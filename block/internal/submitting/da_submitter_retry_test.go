package submitting

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/celestiaorg/go-square/v3/share"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/block/internal/da"
	"github.com/evstack/ev-node/pkg/blob"
	"github.com/evstack/ev-node/pkg/config"
	datypes "github.com/evstack/ev-node/pkg/da/types"
	"github.com/evstack/ev-node/pkg/genesis"
)

type mockBlobAPI struct{ mock.Mock }

func (m *mockBlobAPI) Submit(ctx context.Context, blobs []*blob.Blob, opts *blob.SubmitOptions) (uint64, error) {
	args := m.Called(ctx, blobs, opts)
	return args.Get(0).(uint64), args.Error(1)
}
func (m *mockBlobAPI) GetAll(ctx context.Context, height uint64, namespaces []share.Namespace) ([]*blob.Blob, error) {
	return nil, nil
}
func (m *mockBlobAPI) GetProof(ctx context.Context, height uint64, namespace share.Namespace, commitment blob.Commitment) (*blob.Proof, error) {
	return nil, nil
}
func (m *mockBlobAPI) Included(ctx context.Context, height uint64, namespace share.Namespace, proof *blob.Proof, commitment blob.Commitment) (bool, error) {
	return false, nil
}

func newRetrySubmitter(api *mockBlobAPI) *DASubmitter {
	cfg := config.Config{}
	cfg.DA.BlockTime.Duration = 1 * time.Millisecond
	cfg.DA.MaxSubmitAttempts = 3
	cfg.DA.SubmitOptions = "opts"
	cfg.DA.Namespace = "ns"
	cfg.DA.DataNamespace = "ns"

	client := da.NewClient(da.Config{
		BlobAPI:       api,
		Logger:        zerolog.Nop(),
		Namespace:     cfg.DA.Namespace,
		DataNamespace: cfg.DA.DataNamespace,
	})

	return NewDASubmitter(client, cfg, genesis.Genesis{}, common.BlockOptions{}, common.NopMetrics(), zerolog.Nop())
}

func TestSubmitToDA_MempoolRetry(t *testing.T) {
	api := &mockBlobAPI{}
	ns := share.MustNewV0Namespace([]byte("ns")).Bytes()
	opts := []byte("{}")

	api.On("Submit", mock.Anything, mock.Anything, mock.AnythingOfType("*blob.SubmitOptions")).
		Return(uint64(0), datypes.ErrTxTimedOut).Once()
	api.On("Submit", mock.Anything, mock.Anything, mock.AnythingOfType("*blob.SubmitOptions")).
		Return(uint64(1), nil).Once()

	s := newRetrySubmitter(api)
	err := submitToDA[string](s, context.Background(), []string{"a", "b"}, func(s string) ([]byte, error) { return []byte(s), nil }, func([]string, *datypes.ResultSubmit) {}, "item", ns, opts, nil)
	assert.NoError(t, err)
	api.AssertNumberOfCalls(t, "Submit", 2)
}

func TestSubmitToDA_UnknownError_RetryThenSuccess(t *testing.T) {
	api := &mockBlobAPI{}
	ns := share.MustNewV0Namespace([]byte("ns")).Bytes()
	opts := []byte("{}")

	api.On("Submit", mock.Anything, mock.Anything, mock.AnythingOfType("*blob.SubmitOptions")).
		Return(uint64(0), errors.New("boom")).Once()
	api.On("Submit", mock.Anything, mock.Anything, mock.AnythingOfType("*blob.SubmitOptions")).
		Return(uint64(1), nil).Once()

	s := newRetrySubmitter(api)
	err := submitToDA[string](s, context.Background(), []string{"x"}, func(s string) ([]byte, error) { return []byte(s), nil }, func([]string, *datypes.ResultSubmit) {}, "item", ns, opts, nil)
	assert.NoError(t, err)
	api.AssertNumberOfCalls(t, "Submit", 2)
}

func TestSubmitToDA_TooBig_SplitsAndSucceeds(t *testing.T) {
	api := &mockBlobAPI{}
	ns := share.MustNewV0Namespace([]byte("ns")).Bytes()
	opts := []byte("{}")

	api.On("Submit", mock.Anything, mock.Anything, mock.AnythingOfType("*blob.SubmitOptions")).
		Return(uint64(0), datypes.ErrBlobSizeOverLimit).Once()
	api.On("Submit", mock.Anything, mock.Anything, mock.AnythingOfType("*blob.SubmitOptions")).
		Return(uint64(1), nil).Once()

	s := newRetrySubmitter(api)
	err := submitToDA[string](s, context.Background(), []string{"a", "b", "c", "d"}, func(s string) ([]byte, error) { return []byte(s), nil }, func([]string, *datypes.ResultSubmit) {}, "item", ns, opts, nil)
	assert.NoError(t, err)
	api.AssertNumberOfCalls(t, "Submit", 2)
}
