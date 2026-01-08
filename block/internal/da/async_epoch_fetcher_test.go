package da

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	datypes "github.com/evstack/ev-node/pkg/da/types"

	mocks "github.com/evstack/ev-node/test/mocks"
)

func TestAsyncEpochFetcher_GetCachedEpoch_NotAtEpochEnd(t *testing.T) {
	client := &mocks.MockClient{}
	fiNs := datypes.NamespaceFromString("test-fi-ns").Bytes()
	client.On("HasForcedInclusionNamespace").Return(true)
	client.On("GetForcedInclusionNamespace").Return(fiNs)

	logger := zerolog.Nop()
	fetcher := NewAsyncEpochFetcher(client, logger, 100, 10, 1, time.Second)

	ctx := context.Background()

	// Height 105 is not an epoch end (100, 109, 118, etc. are epoch ends for size 10)
	event, err := fetcher.GetCachedEpoch(ctx, 105)
	require.NoError(t, err)
	assert.NotNil(t, event)
	assert.Equal(t, 0, len(event.Txs))
	assert.Equal(t, uint64(105), event.StartDaHeight)
	assert.Equal(t, uint64(105), event.EndDaHeight)
}

func TestAsyncEpochFetcher_GetCachedEpoch_NoNamespace(t *testing.T) {
	client := &mocks.MockClient{}
	client.On("HasForcedInclusionNamespace").Return(false)

	logger := zerolog.Nop()
	fetcher := NewAsyncEpochFetcher(client, logger, 100, 10, 1, time.Second)

	ctx := context.Background()
	_, err := fetcher.GetCachedEpoch(ctx, 109)
	assert.ErrorIs(t, err, ErrForceInclusionNotConfigured)
}

func TestAsyncEpochFetcher_GetCachedEpoch_CacheMiss(t *testing.T) {
	client := &mocks.MockClient{}
	fiNs := datypes.NamespaceFromString("test-fi-ns").Bytes()
	client.On("HasForcedInclusionNamespace").Return(true)
	client.On("GetForcedInclusionNamespace").Return(fiNs)

	logger := zerolog.Nop()
	fetcher := NewAsyncEpochFetcher(client, logger, 100, 10, 1, time.Second)

	ctx := context.Background()

	// Epoch end at 109, but nothing cached
	event, err := fetcher.GetCachedEpoch(ctx, 109)
	require.NoError(t, err)
	assert.Nil(t, event) // Cache miss
}

func TestAsyncEpochFetcher_FetchAndCache(t *testing.T) {
	testBlobs := [][]byte{
		[]byte("tx1"),
		[]byte("tx2"),
		[]byte("tx3"),
	}

	client := &mocks.MockClient{}
	fiNs := datypes.NamespaceFromString("test-fi-ns").Bytes()
	client.On("HasForcedInclusionNamespace").Return(true)
	client.On("GetForcedInclusionNamespace").Return(fiNs)

	// Mock Retrieve calls for epoch [100, 109]
	for height := uint64(100); height <= 109; height++ {
		if height == 100 {
			client.On("Retrieve", mock.Anything, height, fiNs).Return(datypes.ResultRetrieve{
				BaseResult: datypes.BaseResult{
					Code:      datypes.StatusSuccess,
					Timestamp: time.Unix(1000, 0),
				},
				Data: testBlobs,
			}).Once()
		} else {
			client.On("Retrieve", mock.Anything, height, fiNs).Return(datypes.ResultRetrieve{
				BaseResult: datypes.BaseResult{Code: datypes.StatusNotFound},
			}).Once()
		}
	}

	logger := zerolog.Nop()
	// Use a short poll interval for faster test execution
	fetcher := NewAsyncEpochFetcher(client, logger, 100, 10, 1, 50*time.Millisecond)
	fetcher.Start()
	defer fetcher.Stop()

	// Wait for the background fetch to complete by polling the cache
	ctx := context.Background()
	var event *ForcedInclusionEvent
	var err error

	// Poll for up to 2 seconds for the epoch to be cached
	for i := 0; i < 40; i++ {
		event, err = fetcher.GetCachedEpoch(ctx, 109)
		require.NoError(t, err)
		if event != nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	require.NotNil(t, event, "epoch should be cached after background fetch")
	assert.Equal(t, 3, len(event.Txs))
	assert.Equal(t, testBlobs[0], event.Txs[0])
	assert.Equal(t, uint64(100), event.StartDaHeight)
	assert.Equal(t, uint64(109), event.EndDaHeight)
}

func TestAsyncEpochFetcher_BackgroundPrefetch(t *testing.T) {
	testBlobs := [][]byte{
		[]byte("tx1"),
		[]byte("tx2"),
	}

	client := &mocks.MockClient{}
	fiNs := datypes.NamespaceFromString("test-fi-ns").Bytes()
	client.On("HasForcedInclusionNamespace").Return(true)
	client.On("GetForcedInclusionNamespace").Return(fiNs)

	// Mock for current epoch [100, 109]
	for height := uint64(100); height <= 109; height++ {
		if height == 105 {
			client.On("Retrieve", mock.Anything, height, fiNs).Return(datypes.ResultRetrieve{
				BaseResult: datypes.BaseResult{
					Code:      datypes.StatusSuccess,
					Timestamp: time.Unix(2000, 0),
				},
				Data: testBlobs,
			}).Maybe()
		} else {
			client.On("Retrieve", mock.Anything, height, fiNs).Return(datypes.ResultRetrieve{
				BaseResult: datypes.BaseResult{Code: datypes.StatusNotFound},
			}).Maybe()
		}
	}

	// Mock for next epoch [110, 119]
	for height := uint64(110); height <= 119; height++ {
		client.On("Retrieve", mock.Anything, height, fiNs).Return(datypes.ResultRetrieve{
			BaseResult: datypes.BaseResult{Code: datypes.StatusNotFound},
		}).Maybe()
	}

	logger := zerolog.Nop()
	fetcher := NewAsyncEpochFetcher(client, logger, 100, 10, 1, 50*time.Millisecond)

	fetcher.Start()
	defer fetcher.Stop()

	// Wait for background prefetch to happen
	time.Sleep(200 * time.Millisecond)

	// Check if epoch was prefetched
	ctx := context.Background()
	event, err := fetcher.GetCachedEpoch(ctx, 109)
	require.NoError(t, err)
	assert.NotNil(t, event)
	assert.Equal(t, 2, len(event.Txs))
}

func TestAsyncEpochFetcher_Serialization(t *testing.T) {
	event := &ForcedInclusionEvent{
		Timestamp:     time.Unix(12345, 0).UTC(),
		StartDaHeight: 100,
		EndDaHeight:   109,
		Txs: [][]byte{
			[]byte("transaction1"),
			[]byte("tx2"),
			[]byte("another_transaction"),
		},
	}

	// Serialize
	data, err := serializeForcedInclusionEvent(event)
	require.NoError(t, err)
	assert.Greater(t, len(data), 0)

	// Deserialize
	decoded, err := deserializeForcedInclusionEvent(data)
	require.NoError(t, err)
	assert.Equal(t, event.Timestamp.Unix(), decoded.Timestamp.Unix())
	assert.Equal(t, event.StartDaHeight, decoded.StartDaHeight)
	assert.Equal(t, event.EndDaHeight, decoded.EndDaHeight)
	assert.Equal(t, len(event.Txs), len(decoded.Txs))
	for i := range event.Txs {
		assert.Equal(t, event.Txs[i], decoded.Txs[i])
	}
}

func TestAsyncEpochFetcher_SerializationEmpty(t *testing.T) {
	event := &ForcedInclusionEvent{
		Timestamp:     time.Unix(0, 0).UTC(),
		StartDaHeight: 100,
		EndDaHeight:   100,
		Txs:           [][]byte{},
	}

	data, err := serializeForcedInclusionEvent(event)
	require.NoError(t, err)

	decoded, err := deserializeForcedInclusionEvent(data)
	require.NoError(t, err)
	assert.Equal(t, uint64(100), decoded.StartDaHeight)
	assert.Equal(t, 0, len(decoded.Txs))
}

func TestAsyncEpochFetcher_HeightFromFuture(t *testing.T) {
	client := &mocks.MockClient{}
	fiNs := datypes.NamespaceFromString("test-fi-ns").Bytes()
	client.On("HasForcedInclusionNamespace").Return(true)
	client.On("GetForcedInclusionNamespace").Return(fiNs)

	// Epoch end not available yet
	client.On("Retrieve", mock.Anything, uint64(109), fiNs).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusHeightFromFuture},
	}).Once()

	logger := zerolog.Nop()
	fetcher := NewAsyncEpochFetcher(client, logger, 100, 10, 1, time.Second)
	fetcher.Start()
	defer fetcher.Stop()

	// Cache should be empty
	ctx := context.Background()
	event, err := fetcher.GetCachedEpoch(ctx, 109)
	require.NoError(t, err)
	assert.Nil(t, event)
}

func TestAsyncEpochFetcher_StopGracefully(t *testing.T) {
	client := &mocks.MockClient{}
	client.On("HasForcedInclusionNamespace").Return(false)

	logger := zerolog.Nop()
	fetcher := NewAsyncEpochFetcher(client, logger, 100, 10, 1, 50*time.Millisecond)

	fetcher.Start()
	time.Sleep(100 * time.Millisecond)

	// Should stop gracefully without panic
	fetcher.Stop()
}
