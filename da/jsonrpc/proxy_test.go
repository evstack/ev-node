package jsonrpc_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	proxy "github.com/evstack/ev-node/da/jsonrpc"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coreda "github.com/evstack/ev-node/core/da"
	datypes "github.com/evstack/ev-node/pkg/da/types"
)

const (
	// ServerHost is the listen host for the test JSONRPC server
	ServerHost = "localhost"
	// ServerPort is the listen port for the test JSONRPC server
	ServerPort = "3450"
	// ClientURL is the url to dial for the test JSONRPC client
	ClientURL = "http://localhost:3450"

	testMaxBlobSize = 100

	DefaultMaxBlobSize = 2 * 1024 * 1024 // 2MB
)

// testNamespace is a 15-byte namespace that will be hex encoded to 30 chars and truncated to 29
var testNamespace = []byte("test-namespace1")

// TestProxy runs the go-da DA test suite against the JSONRPC service
// NOTE: This test requires a test JSONRPC service to run on the port
// 3450 which is chosen to be sufficiently distinct from the default port

func getTestDABlockTime() time.Duration {
	return 100 * time.Millisecond
}

// coreDAAdapter adapts the legacy core/da.DA to the new datypes-based interface.
type coreDAAdapter struct{ core coreda.DA }

func (a coreDAAdapter) Get(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Blob, error) {
	return a.core.Get(ctx, ids, namespace)
}
func (a coreDAAdapter) GetIDs(ctx context.Context, height uint64, namespace []byte) (*datypes.GetIDsResult, error) {
	res, err := a.core.GetIDs(ctx, height, namespace)
	if res == nil {
		return nil, err
	}
	return &datypes.GetIDsResult{IDs: res.IDs, Timestamp: res.Timestamp}, err
}
func (a coreDAAdapter) GetProofs(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Proof, error) {
	return a.core.GetProofs(ctx, ids, namespace)
}
func (a coreDAAdapter) Commit(ctx context.Context, blobs []datypes.Blob, namespace []byte) ([]datypes.Commitment, error) {
	return a.core.Commit(ctx, blobs, namespace)
}
func (a coreDAAdapter) Validate(ctx context.Context, ids []datypes.ID, proofs []datypes.Proof, namespace []byte) ([]bool, error) {
	return a.core.Validate(ctx, ids, proofs, namespace)
}
func (a coreDAAdapter) Submit(ctx context.Context, blobs []datypes.Blob, gasPrice float64, namespace []byte) ([]datypes.ID, error) {
	return a.core.Submit(ctx, blobs, gasPrice, namespace)
}
func (a coreDAAdapter) SubmitWithOptions(ctx context.Context, blobs []datypes.Blob, gasPrice float64, namespace []byte, options []byte) ([]datypes.ID, error) {
	return a.core.SubmitWithOptions(ctx, blobs, gasPrice, namespace, options)
}

func TestProxy(t *testing.T) {
	dummy := coreda.NewDummyDA(100_000, getTestDABlockTime())
	dummy.StartHeightTicker()
	logger := zerolog.Nop()
	server := proxy.NewServer(logger, ServerHost, ServerPort, coreDAAdapter{core: dummy})
	err := server.Start(context.Background())
	require.NoError(t, err)
	defer func() {
		if err := server.Stop(context.Background()); err != nil {
			require.NoError(t, err)
		}
	}()

	client, err := proxy.NewClient(context.Background(), logger, ClientURL, "74657374", DefaultMaxBlobSize)
	require.NoError(t, err)

	t.Run("Basic DA test", func(t *testing.T) {
		BasicDATest(t, &client.DA)
	})
	t.Run("Get IDs and all data", func(t *testing.T) {
		GetIDsTest(t, &client.DA)
	})
	t.Run("Check Errors", func(t *testing.T) {
		CheckErrors(t, &client.DA)
	})
	t.Run("Concurrent read/write test", func(t *testing.T) {
		ConcurrentReadWriteTest(t, &client.DA)
	})
	t.Run("Given height is from the future", func(t *testing.T) {
		HeightFromFutureTest(t, &client.DA)
	})
	dummy.StopHeightTicker()
}

// BasicDATest tests round trip of messages to DA and back.
func BasicDATest(t *testing.T, d proxy.DA) {
	msg1 := []byte("message 1")
	msg2 := []byte("message 2")

	ctx := t.Context()
	id1, err := d.Submit(ctx, []datypes.Blob{msg1}, 0, testNamespace)
	assert.NoError(t, err)
	assert.NotEmpty(t, id1)

	id2, err := d.Submit(ctx, []datypes.Blob{msg2}, 0, testNamespace)
	assert.NoError(t, err)
	assert.NotEmpty(t, id2)

	time.Sleep(getTestDABlockTime())

	id3, err := d.SubmitWithOptions(ctx, []datypes.Blob{msg1}, 0, testNamespace, []byte("random options"))
	assert.NoError(t, err)
	assert.NotEmpty(t, id3)

	assert.NotEqual(t, id1, id2)
	assert.NotEqual(t, id1, id3)

	ret, err := d.Get(ctx, id1, testNamespace)
	assert.NoError(t, err)
	assert.Equal(t, []datypes.Blob{msg1}, ret)

	commitment1, err := d.Commit(ctx, []datypes.Blob{msg1}, []byte{})
	assert.NoError(t, err)
	assert.NotEmpty(t, commitment1)

	commitment2, err := d.Commit(ctx, []datypes.Blob{msg2}, []byte{})
	assert.NoError(t, err)
	assert.NotEmpty(t, commitment2)

	ids := []datypes.ID{id1[0], id2[0], id3[0]}
	proofs, err := d.GetProofs(ctx, ids, testNamespace)
	assert.NoError(t, err)
	assert.NotEmpty(t, proofs)
	oks, err := d.Validate(ctx, ids, proofs, testNamespace)
	assert.NoError(t, err)
	assert.NotEmpty(t, oks)
	for _, ok := range oks {
		assert.True(t, ok)
	}
}

// CheckErrors ensures that errors are handled properly by DA.
func CheckErrors(t *testing.T, d proxy.DA) {
	ctx := t.Context()
	blob, err := d.Get(ctx, []datypes.ID{[]byte("invalid blob id")}, testNamespace)
	assert.Error(t, err)
	assert.ErrorContains(t, err, datypes.ErrBlobNotFound.Error())
	assert.Empty(t, blob)
}

// GetIDsTest tests iteration over DA
func GetIDsTest(t *testing.T, d proxy.DA) {
	msgs := []datypes.Blob{[]byte("msg1"), []byte("msg2"), []byte("msg3")}

	ctx := t.Context()
	ids, err := d.Submit(ctx, msgs, 0, testNamespace)
	time.Sleep(getTestDABlockTime())
	assert.NoError(t, err)
	assert.Len(t, ids, len(msgs))
	found := false
	end := time.Now().Add(1 * time.Second)

	// To Keep It Simple: we assume working with DA used exclusively for this test (mock, devnet, etc)
	// As we're the only user, we don't need to handle external data (that could be submitted in real world).
	// There is no notion of height, so we need to scan the DA to get test data back.
	for i := uint64(1); !found && !time.Now().After(end); i++ {
		ret, err := d.GetIDs(ctx, i, testNamespace)
		if err != nil {
			if strings.Contains(err.Error(), datypes.ErrHeightFromFuture.Error()) {
				break
			}
			t.Error("failed to get IDs:", err)
		}
		assert.NotNil(t, ret)
		assert.NotZero(t, ret.Timestamp)
		if len(ret.IDs) > 0 {
			blobs, err := d.Get(ctx, ret.IDs, testNamespace)
			assert.NoError(t, err)

			// Submit ensures atomicity of batch, so it makes sense to compare actual blobs (bodies) only when lengths
			// of slices is the same.
			if len(blobs) >= len(msgs) {
				found = true
				for _, msg := range msgs {
					msgFound := false
					for _, blob := range blobs {
						if bytes.Equal(blob, msg) {
							msgFound = true
							break
						}
					}
					if !msgFound {
						found = false
						break
					}
				}
			}
		}
	}

	assert.True(t, found)
}

// ConcurrentReadWriteTest tests the use of mutex lock in DummyDA by calling separate methods that use `d.data` and making sure there's no race conditions
func ConcurrentReadWriteTest(t *testing.T, d proxy.DA) {
	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	writeDone := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := uint64(1); i <= 50; i++ {
			_, err := d.Submit(ctx, []datypes.Blob{[]byte(fmt.Sprintf("test-%d", i))}, 0, []byte("test"))
			assert.NoError(t, err)
		}
		close(writeDone)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-writeDone:
				return
			default:
				_, _ = d.GetIDs(ctx, 0, []byte("test"))
			}
		}
	}()

	wg.Wait()
}

// HeightFromFutureTest tests the case when the given height is from the future
func HeightFromFutureTest(t *testing.T, d proxy.DA) {
	ctx := t.Context()
	_, err := d.GetIDs(ctx, 999999999, []byte("test"))
	assert.Error(t, err)
	// Specifically check if the error contains the error message ErrHeightFromFuture
	assert.ErrorContains(t, err, datypes.ErrHeightFromFuture.Error())
}

// TestSubmitWithOptions tests the SubmitWithOptions method with various scenarios
func TestSubmitWithOptions(t *testing.T) {
	ctx := context.Background()
	testNamespace := "options_test"
	// The client will convert the namespace string to a proper Celestia namespace
	// using SHA256 hashing and version 0 format (1 version byte + 28 ID bytes)
	namespace := datypes.NamespaceFromString(testNamespace)
	encodedNamespace := namespace.Bytes()
	testOptions := []byte("test_options")
	gasPrice := 0.0

	t.Run("Happy Path - All blobs fit", func(t *testing.T) {
		client := &proxy.Client{}
		client.DA.MaxBlobSize = testMaxBlobSize
		client.DA.Logger = zerolog.Nop()

		blobs := []datypes.Blob{[]byte("blob1"), []byte("blob2")}
		expectedIDs := []datypes.ID{[]byte("id1"), []byte("id2")}
		client.DA.Internal.SubmitWithOptions = func(ctx context.Context, blobs []datypes.Blob, gasPrice float64, namespace []byte, options []byte) ([]datypes.ID, error) {
			return expectedIDs, nil
		}

		mockAPI.On("SubmitWithOptions", ctx, blobs, gasPrice, encodedNamespace, testOptions).Return(expectedIDs, nil).Once()

		ids, err := client.DA.SubmitWithOptions(ctx, blobs, gasPrice, encodedNamespace, testOptions)

		require.NoError(t, err)
		assert.Equal(t, expectedIDs, ids)
		mockAPI.AssertExpectations(t)
	})

	t.Run("Single Blob Too Large", func(t *testing.T) {
		client := &proxy.Client{}
		client.DA.MaxBlobSize = testMaxBlobSize
		client.DA.Logger = zerolog.Nop()

		largerBlob := make([]byte, testMaxBlobSize+1)
		blobs := []datypes.Blob{largerBlob, []byte("this blob is definitely too large")}

		_, err := client.DA.SubmitWithOptions(ctx, blobs, gasPrice, encodedNamespace, testOptions)

		require.Error(t, err)
	})

	t.Run("Total Size Exceeded", func(t *testing.T) {
		client := &proxy.Client{}
		client.DA.MaxBlobSize = testMaxBlobSize
		client.DA.Logger = zerolog.Nop()

		blobsizes := make([]byte, testMaxBlobSize/3)
		blobsizesOver := make([]byte, testMaxBlobSize)

		blobs := []datypes.Blob{blobsizes, blobsizes, blobsizesOver}

		ids, err := client.DA.SubmitWithOptions(ctx, blobs, gasPrice, encodedNamespace, testOptions)

		require.Error(t, err)
		assert.ErrorIs(t, err, datypes.ErrBlobSizeOverLimit)
		assert.Nil(t, ids)
	})

	t.Run("First Blob Too Large", func(t *testing.T) {
		client := &proxy.Client{}
		client.DA.MaxBlobSize = testMaxBlobSize
		client.DA.Logger = zerolog.Nop()

		largerBlob := make([]byte, testMaxBlobSize+1)
		blobs := []datypes.Blob{largerBlob, []byte("small")}

		ids, err := client.DA.SubmitWithOptions(ctx, blobs, gasPrice, encodedNamespace, testOptions)

		require.Error(t, err)
		assert.ErrorIs(t, err, datypes.ErrBlobSizeOverLimit)
		assert.Nil(t, ids)
	})

	t.Run("Empty Input Blobs", func(t *testing.T) {
		client := &proxy.Client{}
		client.DA.MaxBlobSize = testMaxBlobSize
		client.DA.Logger = zerolog.Nop()

		var blobs []datypes.Blob

		ids, err := client.DA.SubmitWithOptions(ctx, blobs, gasPrice, encodedNamespace, testOptions)

		require.NoError(t, err)
		assert.Empty(t, ids)
	})

	t.Run("Error During SubmitWithOptions RPC", func(t *testing.T) {
		client := &proxy.Client{}
		client.DA.MaxBlobSize = testMaxBlobSize
		client.DA.Logger = zerolog.Nop()

		blobs := []datypes.Blob{[]byte("blob1")}
		expectedError := errors.New("rpc submit failed")

		client.DA.Internal.SubmitWithOptions = func(ctx context.Context, blobs []datypes.Blob, gasPrice float64, namespace []byte, options []byte) ([]datypes.ID, error) {
			return nil, expectedError
		}

		ids, err := client.DA.SubmitWithOptions(ctx, blobs, gasPrice, encodedNamespace, testOptions)

		require.Error(t, err)
		assert.ErrorIs(t, err, expectedError)
		assert.Nil(t, ids)
	})
}
