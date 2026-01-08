package da

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/pkg/config"
	datypes "github.com/evstack/ev-node/pkg/da/types"

	mocks "github.com/evstack/ev-node/test/mocks"
)

func TestAsyncBlockFetcher_GetCachedBlock_NoNamespace(t *testing.T) {
	client := &mocks.MockClient{}
	client.On("HasForcedInclusionNamespace").Return(false)

	logger := zerolog.Nop()
	fetcher := NewAsyncBlockFetcher(client, logger, config.DefaultConfig(), 100, 10)

	ctx := context.Background()
	_, err := fetcher.GetCachedBlock(ctx, 100)
	assert.ErrorIs(t, err, ErrForceInclusionNotConfigured)
}

func TestAsyncBlockFetcher_GetCachedBlock_CacheMiss(t *testing.T) {
	client := &mocks.MockClient{}
	fiNs := datypes.NamespaceFromString("test-fi-ns").Bytes()
	client.On("HasForcedInclusionNamespace").Return(true)
	client.On("GetForcedInclusionNamespace").Return(fiNs)

	logger := zerolog.Nop()
	fetcher := NewAsyncBlockFetcher(client, logger, config.DefaultConfig(), 100, 10)

	ctx := context.Background()

	// Nothing cached yet
	block, err := fetcher.GetCachedBlock(ctx, 100)
	require.NoError(t, err)
	assert.Nil(t, block) // Cache miss
}

func TestAsyncBlockFetcher_FetchAndCache(t *testing.T) {
	testBlobs := [][]byte{
		[]byte("tx1"),
		[]byte("tx2"),
		[]byte("tx3"),
	}

	client := &mocks.MockClient{}
	fiNs := datypes.NamespaceFromString("test-fi-ns").Bytes()
	client.On("HasForcedInclusionNamespace").Return(true)
	client.On("GetForcedInclusionNamespace").Return(fiNs)

	// Mock Retrieve call for height 100
	client.On("Retrieve", mock.Anything, uint64(100), fiNs).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{
			Code:      datypes.StatusSuccess,
			Timestamp: time.Unix(1000, 0),
		},
		Data: testBlobs,
	}).Once()

	// Mock other heights that will be prefetched
	for height := uint64(101); height <= 109; height++ {
		client.On("Retrieve", mock.Anything, height, fiNs).Return(datypes.ResultRetrieve{
			BaseResult: datypes.BaseResult{Code: datypes.StatusNotFound},
		}).Maybe()
	}

	logger := zerolog.Nop()
	// Use a short poll interval for faster test execution
	fetcher := NewAsyncBlockFetcher(client, logger, config.DefaultConfig(), 100, 10)
	fetcher.Start()
	defer fetcher.Stop()

	// Update current height to trigger prefetch
	fetcher.UpdateCurrentHeight(100)

	// Wait for the background fetch to complete by polling the cache
	ctx := context.Background()
	var block *BlockData
	var err error

	// Poll for up to 2 seconds for the block to be cached
	for i := 0; i < 40; i++ {
		block, err = fetcher.GetCachedBlock(ctx, 100)
		require.NoError(t, err)
		if block != nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	require.NotNil(t, block, "block should be cached after background fetch")
	assert.Equal(t, uint64(100), block.Height)
	assert.Equal(t, 3, len(block.Blobs))
	assert.Equal(t, testBlobs[0], block.Blobs[0])
	assert.Equal(t, testBlobs[1], block.Blobs[1])
	assert.Equal(t, testBlobs[2], block.Blobs[2])
}

func TestAsyncBlockFetcher_BackgroundPrefetch(t *testing.T) {
	testBlobs := [][]byte{
		[]byte("tx1"),
		[]byte("tx2"),
	}

	client := &mocks.MockClient{}
	fiNs := datypes.NamespaceFromString("test-fi-ns").Bytes()
	client.On("HasForcedInclusionNamespace").Return(true)
	client.On("GetForcedInclusionNamespace").Return(fiNs)

	// Mock for heights 100-110 (current + prefetch window)
	for height := uint64(100); height <= 110; height++ {
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

	logger := zerolog.Nop()
	fetcher := NewAsyncBlockFetcher(client, logger, config.DefaultConfig(), 100, 10)

	fetcher.Start()
	defer fetcher.Stop()

	// Update current height to trigger prefetch
	fetcher.UpdateCurrentHeight(100)

	// Wait for background prefetch to happen
	time.Sleep(200 * time.Millisecond)

	// Check if block was prefetched
	ctx := context.Background()
	block, err := fetcher.GetCachedBlock(ctx, 105)
	require.NoError(t, err)
	assert.NotNil(t, block)
	assert.Equal(t, uint64(105), block.Height)
	assert.Equal(t, 2, len(block.Blobs))
}

func TestAsyncBlockFetcher_HeightFromFuture(t *testing.T) {
	client := &mocks.MockClient{}
	fiNs := datypes.NamespaceFromString("test-fi-ns").Bytes()
	client.On("HasForcedInclusionNamespace").Return(true)
	client.On("GetForcedInclusionNamespace").Return(fiNs)

	// Height not available yet
	client.On("Retrieve", mock.Anything, uint64(100), fiNs).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusHeightFromFuture},
	}).Maybe()

	logger := zerolog.Nop()
	fetcher := NewAsyncBlockFetcher(client, logger, config.DefaultConfig(), 100, 10)
	fetcher.Start()
	defer fetcher.Stop()

	fetcher.UpdateCurrentHeight(100)

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Cache should be empty
	ctx := context.Background()
	block, err := fetcher.GetCachedBlock(ctx, 100)
	require.NoError(t, err)
	assert.Nil(t, block)
}

func TestAsyncBlockFetcher_StopGracefully(t *testing.T) {
	client := &mocks.MockClient{}
	client.On("HasForcedInclusionNamespace").Return(false)

	logger := zerolog.Nop()
	fetcher := NewAsyncBlockFetcher(client, logger, config.DefaultConfig(), 100, 10)

	fetcher.Start()
	time.Sleep(100 * time.Millisecond)

	// Should stop gracefully without panic
	fetcher.Stop()
}

func TestBlockData_Serialization(t *testing.T) {
	block := &BlockData{
		Height:    100,
		Timestamp: time.Unix(12345, 0).UTC(),
		Blobs: [][]byte{
			[]byte("blob1"),
			[]byte("blob2"),
			[]byte("another_blob"),
		},
	}

	// Serialize
	data, err := serializeBlockData(block)
	require.NoError(t, err)
	assert.Greater(t, len(data), 0)

	// Deserialize
	decoded, err := deserializeBlockData(data)
	require.NoError(t, err)
	assert.Equal(t, block.Timestamp.Unix(), decoded.Timestamp.Unix())
	assert.Equal(t, block.Height, decoded.Height)
	assert.Equal(t, len(block.Blobs), len(decoded.Blobs))
	for i := range block.Blobs {
		assert.Equal(t, block.Blobs[i], decoded.Blobs[i])
	}
}

func TestBlockData_SerializationEmpty(t *testing.T) {
	block := &BlockData{
		Height:    100,
		Timestamp: time.Unix(0, 0).UTC(),
		Blobs:     [][]byte{},
	}

	data, err := serializeBlockData(block)
	require.NoError(t, err)

	decoded, err := deserializeBlockData(data)
	require.NoError(t, err)
	assert.Equal(t, uint64(100), decoded.Height)
	assert.Equal(t, 0, len(decoded.Blobs))
}
