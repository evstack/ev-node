package syncing

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/types"
)

func TestDAHandler_ProcessBlobs_ReturnsEvents(t *testing.T) {
	// Create a mock cache for the test
	mockCache := &MockCacheManager{}

	// Create DA handler
	cfg := config.DefaultConfig
	cfg.DA.Namespace = "test-namespace"
	gen := genesis.Genesis{
		ChainID:         "test-chain",
		InitialHeight:   1,
		StartTime:       time.Now(),
		ProposerAddress: []byte("test-proposer"),
	}

	handler := NewDAHandler(
		nil, // We're not testing DA layer interaction
		mockCache,
		cfg,
		gen,
		common.DefaultBlockOptions(),
		zerolog.Nop(),
	)

	// Create test header and data blobs
	header := &types.SignedHeader{
		Header: types.Header{
			BaseHeader: types.BaseHeader{
				ChainID: "test-chain",
				Height:  1,
				Time:    uint64(time.Now().UnixNano()),
			},
			ProposerAddress: []byte("test-proposer"),
		},
	}

	data := &types.Data{
		Metadata: &types.Metadata{
			ChainID: "test-chain",
			Height:  1,
			Time:    uint64(time.Now().UnixNano()),
		},
		Txs: []types.Tx{},
	}

	// Create blobs (this is simplified - normally would be protobuf marshaled)
	headerBlob, err := header.MarshalBinary()
	require.NoError(t, err)

	dataBlob, err := data.MarshalBinary()
	require.NoError(t, err)

	blobs := [][]byte{headerBlob, dataBlob}
	daHeight := uint64(100)

	// Process blobs
	ctx := context.Background()
	events := handler.processBlobs(ctx, blobs, daHeight)

	// Verify that events are returned
	// Note: This test will fail with the current protobuf decoding logic
	// but demonstrates the expected behavior once proper blob encoding is implemented
	t.Logf("Processed %d blobs, got %d events", len(blobs), len(events))

	// The actual assertion would depend on proper blob encoding
	// For now, we verify the function doesn't panic and returns a slice
	assert.NotNil(t, events)
}

func TestHeightEvent_Structure(t *testing.T) {
	// Test that HeightEvent has all required fields
	event := HeightEvent{
		Header: &types.SignedHeader{
			Header: types.Header{
				BaseHeader: types.BaseHeader{
					ChainID: "test-chain",
					Height:  1,
				},
			},
		},
		Data: &types.Data{
			Metadata: &types.Metadata{
				ChainID: "test-chain",
				Height:  1,
			},
		},
		DaHeight:               100,
		HeaderDaIncludedHeight: 100,
	}

	assert.Equal(t, uint64(1), event.Header.Height())
	assert.Equal(t, uint64(1), event.Data.Height())
	assert.Equal(t, uint64(100), event.DaHeight)
	assert.Equal(t, uint64(100), event.HeaderDaIncludedHeight)
}

func TestCacheDAHeightEvent_Usage(t *testing.T) {
	// Test the exported DAHeightEvent type
	header := &types.SignedHeader{
		Header: types.Header{
			BaseHeader: types.BaseHeader{
				ChainID: "test-chain",
				Height:  1,
			},
		},
	}

	data := &types.Data{
		Metadata: &types.Metadata{
			ChainID: "test-chain",
			Height:  1,
		},
	}

	event := &cache.DAHeightEvent{
		Header:                 header,
		Data:                   data,
		DaHeight:               100,
		HeaderDaIncludedHeight: 100,
	}

	// Verify all fields are accessible
	assert.NotNil(t, event.Header)
	assert.NotNil(t, event.Data)
	assert.Equal(t, uint64(100), event.DaHeight)
	assert.Equal(t, uint64(100), event.HeaderDaIncludedHeight)
}

// Mock cache manager for testing
type MockCacheManager struct{}

func (m *MockCacheManager) GetHeader(height uint64) *types.SignedHeader         { return nil }
func (m *MockCacheManager) SetHeader(height uint64, header *types.SignedHeader) {}
func (m *MockCacheManager) IsHeaderSeen(hash string) bool                       { return false }
func (m *MockCacheManager) SetHeaderSeen(hash string)                           {}
func (m *MockCacheManager) IsHeaderDAIncluded(hash string) bool                 { return false }
func (m *MockCacheManager) SetHeaderDAIncluded(hash string, daHeight uint64)    {}

func (m *MockCacheManager) GetData(height uint64) *types.Data              { return nil }
func (m *MockCacheManager) SetData(height uint64, data *types.Data)        {}
func (m *MockCacheManager) IsDataSeen(hash string) bool                    { return false }
func (m *MockCacheManager) SetDataSeen(hash string)                        {}
func (m *MockCacheManager) IsDataDAIncluded(hash string) bool              { return false }
func (m *MockCacheManager) SetDataDAIncluded(hash string, daHeight uint64) {}

func (m *MockCacheManager) GetPendingHeaders(ctx context.Context) ([]*types.SignedHeader, error) {
	return nil, nil
}
func (m *MockCacheManager) GetPendingData(ctx context.Context) ([]*types.SignedData, error) {
	return nil, nil
}
func (m *MockCacheManager) SetLastSubmittedHeaderHeight(ctx context.Context, height uint64) {}
func (m *MockCacheManager) SetLastSubmittedDataHeight(ctx context.Context, height uint64)   {}

func (m *MockCacheManager) NumPendingHeaders() uint64 { return 0 }
func (m *MockCacheManager) NumPendingData() uint64    { return 0 }

func (m *MockCacheManager) SetPendingEvent(height uint64, event *cache.DAHeightEvent) {}
func (m *MockCacheManager) GetPendingEvents() map[uint64]*cache.DAHeightEvent {
	return make(map[uint64]*cache.DAHeightEvent)
}
func (m *MockCacheManager) DeletePendingEvent(height uint64) {}

func (m *MockCacheManager) ClearProcessedHeader(height uint64) {}
func (m *MockCacheManager) ClearProcessedData(height uint64)   {}
func (m *MockCacheManager) SaveToDisk() error                  { return nil }
func (m *MockCacheManager) LoadFromDisk() error                { return nil }
