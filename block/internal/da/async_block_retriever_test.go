package da

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	datypes "github.com/evstack/ev-node/pkg/da/types"
	mocks "github.com/evstack/ev-node/test/mocks"
	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

func TestAsyncBlockRetriever_GetCachedBlock_NoNamespace(t *testing.T) {
	client := &mocks.MockClient{}

	logger := zerolog.Nop()
	fetcher := NewAsyncBlockRetriever(client, logger, nil, 100*time.Millisecond, 100, 10)

	ctx := context.Background()
	block, err := fetcher.GetCachedBlock(ctx, 100)
	assert.NoError(t, err)
	assert.Nil(t, block)
}

func TestAsyncBlockRetriever_GetCachedBlock_CacheMiss(t *testing.T) {
	client := &mocks.MockClient{}
	fiNs := datypes.NamespaceFromString("test-fi-ns").Bytes()

	logger := zerolog.Nop()
	fetcher := NewAsyncBlockRetriever(client, logger, fiNs, 100*time.Millisecond, 100, 10)

	ctx := context.Background()

	// Nothing cached yet
	block, err := fetcher.GetCachedBlock(ctx, 100)
	require.NoError(t, err)
	assert.Nil(t, block) // Cache miss
}

func TestAsyncBlockRetriever_SubscriptionDrivenCaching(t *testing.T) {
	// Test that blobs arriving via subscription are cached inline.
	testBlobs := [][]byte{
		[]byte("tx1"),
		[]byte("tx2"),
	}

	client := &mocks.MockClient{}
	fiNs := datypes.NamespaceFromString("test-fi-ns").Bytes()

	// Create a subscription channel that delivers one event then blocks.
	subCh := make(chan datypes.SubscriptionEvent, 1)
	subCh <- datypes.SubscriptionEvent{
		Height: 100,
		Blobs:  testBlobs,
	}

	client.On("Subscribe", mock.Anything, fiNs, mock.Anything).Return((<-chan datypes.SubscriptionEvent)(subCh), nil).Once()
	// Catchup loop may call Retrieve for heights beyond 100 — stub those.
	client.On("Retrieve", mock.Anything, mock.Anything, fiNs).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusHeightFromFuture},
	}).Maybe()

	// On second subscribe (after watchdog timeout) just block forever.
	blockCh := make(chan datypes.SubscriptionEvent)
	client.On("Subscribe", mock.Anything, fiNs, mock.Anything).Return((<-chan datypes.SubscriptionEvent)(blockCh), nil).Maybe()

	logger := zerolog.Nop()
	fetcher := NewAsyncBlockRetriever(client, logger, fiNs, 200*time.Millisecond, 100, 5)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fetcher.Start(ctx)
	defer fetcher.Stop()

	// Wait for the subscription event to be processed.
	var block *BlockData
	var err error
	for range 40 {
		block, err = fetcher.GetCachedBlock(ctx, 100)
		require.NoError(t, err)
		if block != nil {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}

	require.NotNil(t, block, "block should be cached after subscription event")
	assert.Equal(t, uint64(100), block.Height)
	assert.Equal(t, 2, len(block.Blobs))
	assert.Equal(t, []byte("tx1"), block.Blobs[0])
	assert.Equal(t, []byte("tx2"), block.Blobs[1])
}

func TestAsyncBlockRetriever_CatchupFillsGaps(t *testing.T) {
	// When subscription reports height 105 but current is 100,
	// catchup loop should Retrieve heights 100-114 (100 + prefetch window).
	testBlobs := [][]byte{[]byte("gap-tx")}

	client := &mocks.MockClient{}
	fiNs := datypes.NamespaceFromString("test-fi-ns").Bytes()

	// Subscription delivers height 105 (no blobs — just a signal).
	subCh := make(chan datypes.SubscriptionEvent, 1)
	subCh <- datypes.SubscriptionEvent{Height: 105}

	client.On("Subscribe", mock.Anything, fiNs, mock.Anything).Return((<-chan datypes.SubscriptionEvent)(subCh), nil).Once()
	blockCh := make(chan datypes.SubscriptionEvent)
	client.On("Subscribe", mock.Anything, fiNs, mock.Anything).Return((<-chan datypes.SubscriptionEvent)(blockCh), nil).Maybe()

	// Height 102 has blobs; rest return not found or future.
	client.On("Retrieve", mock.Anything, uint64(102), fiNs).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{
			Code:      datypes.StatusSuccess,
			Timestamp: time.Unix(2000, 0),
		},
		Data: testBlobs,
	}).Maybe()
	client.On("Retrieve", mock.Anything, mock.Anything, fiNs).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusNotFound},
	}).Maybe()

	logger := zerolog.Nop()
	fetcher := NewAsyncBlockRetriever(client, logger, fiNs, 100*time.Millisecond, 100, 10)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fetcher.Start(ctx)
	defer fetcher.Stop()

	// Wait for catchup to fill the gap.
	var block *BlockData
	var err error
	for range 40 {
		block, err = fetcher.GetCachedBlock(ctx, 102)
		require.NoError(t, err)
		if block != nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	require.NotNil(t, block, "block at 102 should be cached via catchup")
	assert.Equal(t, uint64(102), block.Height)
	assert.Equal(t, 1, len(block.Blobs))
	assert.Equal(t, []byte("gap-tx"), block.Blobs[0])
}

func TestAsyncBlockRetriever_HeightFromFuture(t *testing.T) {
	client := &mocks.MockClient{}
	fiNs := datypes.NamespaceFromString("test-fi-ns").Bytes()

	// Subscription delivers height 100 with no blobs.
	subCh := make(chan datypes.SubscriptionEvent)
	client.On("Subscribe", mock.Anything, fiNs, mock.Anything).Return((<-chan datypes.SubscriptionEvent)(subCh), nil).Once()
	blockCh := make(chan datypes.SubscriptionEvent)
	client.On("Subscribe", mock.Anything, fiNs, mock.Anything).Return((<-chan datypes.SubscriptionEvent)(blockCh), nil).Maybe()

	// All Retrieve calls return HeightFromFuture.
	client.On("Retrieve", mock.Anything, mock.Anything, fiNs).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusHeightFromFuture},
	}).Maybe()

	logger := zerolog.Nop()
	fetcher := NewAsyncBlockRetriever(client, logger, fiNs, time.Millisecond, 100, 10)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	fetcher.Start(ctx)
	defer fetcher.Stop()

	// Wait a bit for catchup to attempt fetches.
	require.Eventually(t, func() bool {
		return fetcher.(*asyncBlockRetriever).subscriber.HasReachedHead()
	}, 1250*time.Millisecond, time.Millisecond)

	// Cache should be empty since all heights are from the future.
	block, err := fetcher.GetCachedBlock(ctx, 100)
	require.NoError(t, err)
	assert.Nil(t, block)
}

func TestAsyncBlockRetriever_StopGracefully(t *testing.T) {
	client := &mocks.MockClient{}
	fiNs := datypes.NamespaceFromString("test-fi-ns").Bytes()

	blockCh := make(chan datypes.SubscriptionEvent)
	client.On("Subscribe", mock.Anything, fiNs, mock.Anything).Return((<-chan datypes.SubscriptionEvent)(blockCh), nil).Maybe()

	logger := zerolog.Nop()
	fetcher := NewAsyncBlockRetriever(client, logger, fiNs, 100*time.Millisecond, 100, 10)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fetcher.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	// Should stop gracefully without panic
	fetcher.Stop()
}

func TestAsyncBlockRetriever_ReconnectOnSubscriptionError(t *testing.T) {
	// Verify that the follow loop reconnects after a subscription channel closes.
	testBlobs := [][]byte{[]byte("reconnect-tx")}

	client := &mocks.MockClient{}
	fiNs := datypes.NamespaceFromString("test-fi-ns").Bytes()

	// First subscription closes immediately (simulating error).
	closedCh := make(chan datypes.SubscriptionEvent)
	close(closedCh)
	client.On("Subscribe", mock.Anything, fiNs, mock.Anything).Return((<-chan datypes.SubscriptionEvent)(closedCh), nil).Once()

	// Second subscription delivers a blob.
	subCh := make(chan datypes.SubscriptionEvent, 1)
	subCh <- datypes.SubscriptionEvent{
		Height: 100,
		Blobs:  testBlobs,
	}
	client.On("Subscribe", mock.Anything, fiNs, mock.Anything).Return((<-chan datypes.SubscriptionEvent)(subCh), nil).Once()

	// Third+ subscribe returns a blocking channel so it doesn't loop forever.
	blockCh := make(chan datypes.SubscriptionEvent)
	client.On("Subscribe", mock.Anything, fiNs, mock.Anything).Return((<-chan datypes.SubscriptionEvent)(blockCh), nil).Maybe()

	// Stub Retrieve for catchup.
	client.On("Retrieve", mock.Anything, mock.Anything, fiNs).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusHeightFromFuture},
	}).Maybe()

	logger := zerolog.Nop()
	// Very short backoff so reconnect is fast in tests.
	fetcher := NewAsyncBlockRetriever(client, logger, fiNs, 50*time.Millisecond, 100, 5)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fetcher.Start(ctx)
	defer fetcher.Stop()

	// Wait for reconnect + event processing.
	var block *BlockData
	var err error
	for range 60 {
		block, err = fetcher.GetCachedBlock(ctx, 100)
		require.NoError(t, err)
		if block != nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	require.NotNil(t, block, "block should be cached after reconnect")
	assert.Equal(t, 1, len(block.Blobs))
	assert.Equal(t, []byte("reconnect-tx"), block.Blobs[0])
}

func TestBlockData_Serialization(t *testing.T) {
	block := &BlockData{
		Height:    100,
		Timestamp: time.Unix(12345, 123456789).UTC(),
		Blobs: [][]byte{
			[]byte("blob1"),
			[]byte("blob2"),
			[]byte("another_blob"),
		},
	}

	// Serialize using protobuf
	pbBlock := &pb.BlockData{
		Height:    block.Height,
		Timestamp: block.Timestamp.UnixNano(),
		Blobs:     block.Blobs,
	}
	data, err := proto.Marshal(pbBlock)
	require.NoError(t, err)
	assert.Greater(t, len(data), 0)

	// Deserialize using protobuf
	var decodedPb pb.BlockData
	err = proto.Unmarshal(data, &decodedPb)
	require.NoError(t, err)

	decoded := &BlockData{
		Height:    decodedPb.Height,
		Timestamp: time.Unix(0, decodedPb.Timestamp).UTC(),
		Blobs:     decodedPb.Blobs,
	}

	assert.Equal(t, block.Timestamp.UnixNano(), decoded.Timestamp.UnixNano())
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

	// Serialize using protobuf
	pbBlock := &pb.BlockData{
		Height:    block.Height,
		Timestamp: block.Timestamp.UnixNano(),
		Blobs:     block.Blobs,
	}
	data, err := proto.Marshal(pbBlock)
	require.NoError(t, err)

	// Deserialize using protobuf
	var decodedPb pb.BlockData
	err = proto.Unmarshal(data, &decodedPb)
	require.NoError(t, err)

	decoded := &BlockData{
		Height:    decodedPb.Height,
		Timestamp: time.Unix(0, decodedPb.Timestamp).UTC(),
		Blobs:     decodedPb.Blobs,
	}

	assert.Equal(t, uint64(100), decoded.Height)
	assert.Equal(t, time.Unix(0, 0).UTC(), decoded.Timestamp)
	assert.Equal(t, 0, len(decoded.Blobs))
}
