package celestianodefiber_test

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	appfibre "github.com/celestiaorg/celestia-app/v8/fibre"
	libshare "github.com/celestiaorg/go-square/v4/share"

	nodeblob "github.com/celestiaorg/celestia-node/blob"
	blobapi "github.com/celestiaorg/celestia-node/nodebuilder/blob"
	celfibre "github.com/celestiaorg/celestia-node/fibre"
	fibreapi "github.com/celestiaorg/celestia-node/nodebuilder/fibre"
	"github.com/celestiaorg/celestia-node/state/txclient"

	"github.com/evstack/ev-node/block"
	cnfiber "github.com/evstack/ev-node/tools/celestia-node-fiber"
)

// namespaceBytes returns a 10-byte v0 namespace ID for test fixtures.
func namespaceBytes() []byte { return bytes.Repeat([]byte{0xab}, libshare.NamespaceVersionZeroIDSize) }

func namespace(t *testing.T) libshare.Namespace {
	t.Helper()
	ns, err := libshare.NewV0Namespace(namespaceBytes())
	require.NoError(t, err)
	return ns
}

// fakeFibre is a minimal stand-in for fibreapi.Module. We hand-roll it
// because celestia-node's generated mocks use two different gomock forks
// across the fibre and blob packages, which can't share a Controller.
type fakeFibre struct {
	uploadFn   func(context.Context, libshare.Namespace, []byte, *txclient.TxConfig) (*fibreapi.UploadResult, error)
	downloadFn func(context.Context, appfibre.BlobID) (*fibreapi.GetBlobResult, error)
}

var _ fibreapi.Module = (*fakeFibre)(nil)

func (f *fakeFibre) Submit(context.Context, libshare.Namespace, []byte, *txclient.TxConfig) (*fibreapi.SubmitResult, error) {
	return nil, errors.New("fakeFibre.Submit not implemented")
}

func (f *fakeFibre) Upload(ctx context.Context, ns libshare.Namespace, data []byte, cfg *txclient.TxConfig) (*fibreapi.UploadResult, error) {
	return f.uploadFn(ctx, ns, data, cfg)
}

func (f *fakeFibre) Download(ctx context.Context, id appfibre.BlobID) (*fibreapi.GetBlobResult, error) {
	return f.downloadFn(ctx, id)
}

func (f *fakeFibre) QueryEscrowAccount(context.Context, string) (*celfibre.EscrowAccount, error) {
	return nil, errors.New("fakeFibre.QueryEscrowAccount not implemented")
}

func (f *fakeFibre) Deposit(context.Context, types.Coin, *txclient.TxConfig) error {
	return errors.New("fakeFibre.Deposit not implemented")
}

func (f *fakeFibre) Withdraw(context.Context, types.Coin, *txclient.TxConfig) error {
	return errors.New("fakeFibre.Withdraw not implemented")
}

func (f *fakeFibre) PendingWithdrawals(context.Context, string) ([]celfibre.PendingWithdrawal, error) {
	return nil, errors.New("fakeFibre.PendingWithdrawals not implemented")
}

// fakeBlob is a minimal stand-in for blobapi.Module. Only Subscribe is
// exercised by the adapter, so the rest return errors if called.
type fakeBlob struct {
	subscribeFn func(context.Context, libshare.Namespace) (<-chan *nodeblob.SubscriptionResponse, error)
}

var _ blobapi.Module = (*fakeBlob)(nil)

func (b *fakeBlob) Submit(context.Context, []*nodeblob.Blob, *nodeblob.SubmitOptions) (uint64, error) {
	return 0, errors.New("fakeBlob.Submit not implemented")
}

func (b *fakeBlob) Get(context.Context, uint64, libshare.Namespace, nodeblob.Commitment) (*nodeblob.Blob, error) {
	return nil, errors.New("fakeBlob.Get not implemented")
}

func (b *fakeBlob) GetAll(context.Context, uint64, []libshare.Namespace) ([]*nodeblob.Blob, error) {
	return nil, errors.New("fakeBlob.GetAll not implemented")
}

func (b *fakeBlob) GetProof(context.Context, uint64, libshare.Namespace, nodeblob.Commitment) (*nodeblob.Proof, error) {
	return nil, errors.New("fakeBlob.GetProof not implemented")
}

func (b *fakeBlob) Included(context.Context, uint64, libshare.Namespace, *nodeblob.Proof, nodeblob.Commitment) (bool, error) {
	return false, errors.New("fakeBlob.Included not implemented")
}

func (b *fakeBlob) GetCommitmentProof(context.Context, uint64, libshare.Namespace, []byte) (*nodeblob.CommitmentProof, error) {
	return nil, errors.New("fakeBlob.GetCommitmentProof not implemented")
}

func (b *fakeBlob) Subscribe(ctx context.Context, ns libshare.Namespace) (<-chan *nodeblob.SubscriptionResponse, error) {
	return b.subscribeFn(ctx, ns)
}

// TestAdapterSatisfiesDA is a compile-time assertion that the adapter
// implements the ev-node Fiber DA contract.
func TestAdapterSatisfiesDA(t *testing.T) {
	var _ block.FiberClient = (*cnfiber.Adapter)(nil)
}

func TestUpload_ForwardsNamespaceDataAndBlobID(t *testing.T) {
	var commit appfibre.Commitment
	copy(commit[:], bytes.Repeat([]byte{0x11}, appfibre.CommitmentSize))
	expectedBlobID := appfibre.NewBlobID(0, commit)

	var seenNs libshare.Namespace
	var seenData []byte
	var seenCfg *txclient.TxConfig
	fibre := &fakeFibre{
		uploadFn: func(_ context.Context, ns libshare.Namespace, data []byte, cfg *txclient.TxConfig) (*fibreapi.UploadResult, error) {
			seenNs = ns
			seenData = data
			seenCfg = cfg
			return &fibreapi.UploadResult{BlobID: expectedBlobID}, nil
		},
	}
	a := cnfiber.FromModules(fibre, &fakeBlob{}, 0)

	data := []byte("hello fibre")
	before := time.Now()
	got, err := a.Upload(context.Background(), namespaceBytes(), data)
	require.NoError(t, err)
	require.Equal(t, block.FiberBlobID(expectedBlobID), got.BlobID)
	require.Equal(t, namespace(t), seenNs)
	require.Equal(t, data, seenData)
	require.Nil(t, seenCfg, "adapter passes nil TxConfig to honour client defaults")
	require.True(t, got.ExpiresAt.After(before.Add(time.Hour)))
}

func TestUpload_RejectsWrongNamespaceLength(t *testing.T) {
	a := cnfiber.FromModules(&fakeFibre{}, &fakeBlob{}, 0)
	_, err := a.Upload(context.Background(), []byte{0x01, 0x02}, []byte("x"))
	require.Error(t, err)
}

func TestUpload_PropagatesFibreError(t *testing.T) {
	fibreErr := errors.New("boom")
	fibre := &fakeFibre{
		uploadFn: func(context.Context, libshare.Namespace, []byte, *txclient.TxConfig) (*fibreapi.UploadResult, error) {
			return nil, fibreErr
		},
	}
	a := cnfiber.FromModules(fibre, &fakeBlob{}, 0)
	_, err := a.Upload(context.Background(), namespaceBytes(), []byte("x"))
	require.ErrorIs(t, err, fibreErr)
}

func TestDownload_ReturnsResultData(t *testing.T) {
	var commit appfibre.Commitment
	copy(commit[:], bytes.Repeat([]byte{0x22}, appfibre.CommitmentSize))
	id := appfibre.NewBlobID(0, commit)
	payload := []byte("payload")

	var seenID appfibre.BlobID
	fibre := &fakeFibre{
		downloadFn: func(_ context.Context, arg appfibre.BlobID) (*fibreapi.GetBlobResult, error) {
			seenID = arg
			return &fibreapi.GetBlobResult{Data: payload}, nil
		},
	}

	a := cnfiber.FromModules(fibre, &fakeBlob{}, 0)
	got, err := a.Download(context.Background(), block.FiberBlobID(id))
	require.NoError(t, err)
	require.Equal(t, payload, got)
	require.Equal(t, id, seenID)
}

func TestListen_FiltersFibreOnlyAndEmitsEvent(t *testing.T) {
	ns := namespace(t)

	// v0 (non-Fibre) — must be filtered out.
	v0Lib, err := libshare.NewV0Blob(ns, []byte("pfb blob"))
	require.NoError(t, err)
	v0 := &nodeblob.Blob{Blob: v0Lib}

	// v2 (Fibre) — must be forwarded. libshare requires a 20-byte signer
	// for v2 blobs; the content of the signer is irrelevant to the filter.
	fibreCommit := bytes.Repeat([]byte{0x33}, appfibre.CommitmentSize)
	signer := bytes.Repeat([]byte{0x01}, libshare.SignerSize)
	v2Lib, err := libshare.NewV2Blob(ns, 0, fibreCommit, signer)
	require.NoError(t, err)
	v2 := &nodeblob.Blob{Blob: v2Lib}

	ch := make(chan *nodeblob.SubscriptionResponse, 1)
	ch <- &nodeblob.SubscriptionResponse{
		Blobs:  []*nodeblob.Blob{v0, v2},
		Height: 42,
	}
	close(ch)

	var seenNs libshare.Namespace
	blob := &fakeBlob{
		subscribeFn: func(_ context.Context, sub libshare.Namespace) (<-chan *nodeblob.SubscriptionResponse, error) {
			seenNs = sub
			return ch, nil
		},
	}
	a := cnfiber.FromModules(&fakeFibre{}, blob, 0)
	events, err := a.Listen(context.Background(), namespaceBytes())
	require.NoError(t, err)
	require.Equal(t, ns, seenNs)

	select {
	case ev, ok := <-events:
		require.True(t, ok, "expected one event before channel closes")
		var expectedCommit appfibre.Commitment
		copy(expectedCommit[:], fibreCommit)
		expectedID := appfibre.NewBlobID(0, expectedCommit)
		require.Equal(t, block.FiberBlobID(expectedID), ev.BlobID)
		require.Equal(t, uint64(42), ev.Height)
		require.Equal(t, uint64(v2Lib.DataLen()), ev.DataSize)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for blob event")
	}

	select {
	case _, ok := <-events:
		require.False(t, ok, "expected adapter channel to close after upstream close")
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for output channel to close")
	}
}

func TestListen_CancelledContextClosesOutput(t *testing.T) {
	ns := namespace(t)
	upstream := make(chan *nodeblob.SubscriptionResponse)
	blob := &fakeBlob{
		subscribeFn: func(_ context.Context, arg libshare.Namespace) (<-chan *nodeblob.SubscriptionResponse, error) {
			require.Equal(t, ns, arg)
			return upstream, nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	a := cnfiber.FromModules(&fakeFibre{}, blob, 0)
	events, err := a.Listen(ctx, namespaceBytes())
	require.NoError(t, err)

	cancel()

	select {
	case _, ok := <-events:
		require.False(t, ok, "expected adapter channel to close on ctx cancel")
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ctx-triggered close")
	}
}
