package server

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/evstack/ev-node/pkg/sync"
	ds "github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/p2p"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/test/mocks"
	headerstoremocks "github.com/evstack/ev-node/test/mocks/external"
	"github.com/evstack/ev-node/types"
	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

func TestGetBlock(t *testing.T) {
	// Create a mock store
	mockStore := mocks.NewMockStore(t)

	// Create test data
	height := uint64(10)
	// Ensure the header has the correct height for key generation
	header := &types.SignedHeader{Header: types.Header{BaseHeader: types.BaseHeader{Height: height}}}
	data := &types.Data{}
	expectedHeaderDAHeight := uint64(100)
	expectedDataDAHeight := uint64(101)

	headerDAHeightBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(headerDAHeightBytes, expectedHeaderDAHeight)
	dataDAHeightBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(dataDAHeightBytes, expectedDataDAHeight)

	// Create server with mock store
	logger := zerolog.Nop()
	server := NewStoreServer(mockStore, nil, nil, logger)

	// Test GetBlock with height - success case
	t.Run("by height with DA heights", func(t *testing.T) {
		// Setup mock expectations
		mockStore.On("GetBlockData", mock.Anything, height).Return(header, data, nil).Once()
		mockStore.On("GetMetadata", mock.Anything, store.GetHeightToDAHeightHeaderKey(height)).Return(headerDAHeightBytes, nil).Once()
		mockStore.On("GetMetadata", mock.Anything, store.GetHeightToDAHeightDataKey(height)).Return(dataDAHeightBytes, nil).Once()

		req := connect.NewRequest(&pb.GetBlockRequest{
			Identifier: &pb.GetBlockRequest_Height{
				Height: height,
			},
		})
		resp, err := server.GetBlock(context.Background(), req)

		// Assert expectations
		require.NoError(t, err)
		require.NotNil(t, resp.Msg.Block)
		require.Equal(t, expectedHeaderDAHeight, resp.Msg.HeaderDaHeight)
		require.Equal(t, expectedDataDAHeight, resp.Msg.DataDaHeight)
		mockStore.AssertExpectations(t)
	})

	// Test GetBlock with height - metadata not found
	t.Run("by height DA heights not found", func(t *testing.T) {
		mockStore.On("GetBlockData", mock.Anything, height).Return(header, data, nil).Once()
		mockStore.On("GetMetadata", mock.Anything, store.GetHeightToDAHeightHeaderKey(height)).Return(nil, ds.ErrNotFound).Once()
		mockStore.On("GetMetadata", mock.Anything, store.GetHeightToDAHeightDataKey(height)).Return(nil, ds.ErrNotFound).Once()

		req := connect.NewRequest(&pb.GetBlockRequest{
			Identifier: &pb.GetBlockRequest_Height{
				Height: height,
			},
		})
		resp, err := server.GetBlock(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, resp.Msg.Block)
		require.Equal(t, uint64(0), resp.Msg.HeaderDaHeight) // Should default to 0
		require.Equal(t, uint64(0), resp.Msg.DataDaHeight)   // Should default to 0
		mockStore.AssertExpectations(t)
	})

	// Test GetBlock with hash - success case
	t.Run("by hash with DA heights", func(t *testing.T) {
		hashBytes := []byte("test_hash")
		// Important: The header returned by GetBlockByHash must also have its height set for DA height lookup
		headerForHash := &types.SignedHeader{Header: types.Header{BaseHeader: types.BaseHeader{Height: height}}}
		mockStore.On("GetBlockByHash", mock.Anything, hashBytes).Return(headerForHash, data, nil).Once()
		mockStore.On("GetMetadata", mock.Anything, store.GetHeightToDAHeightHeaderKey(height)).Return(headerDAHeightBytes, nil).Once()
		mockStore.On("GetMetadata", mock.Anything, store.GetHeightToDAHeightDataKey(height)).Return(dataDAHeightBytes, nil).Once()

		req := connect.NewRequest(&pb.GetBlockRequest{
			Identifier: &pb.GetBlockRequest_Hash{
				Hash: hashBytes,
			},
		})
		resp, err := server.GetBlock(context.Background(), req)

		require.NoError(t, err)
		require.NotNil(t, resp.Msg.Block)
		require.Equal(t, expectedHeaderDAHeight, resp.Msg.HeaderDaHeight)
		require.Equal(t, expectedDataDAHeight, resp.Msg.DataDaHeight)
		mockStore.AssertExpectations(t)
	})

	// Test GetBlock with genesis height (0), DA heights should be 0 as per current server logic
	t.Run("by height genesis (height 0)", func(t *testing.T) {
		genesisHeight := uint64(0)                                              // Requesting latest, and store.Height will return 0
		mockStore.On("Height", mock.Anything).Return(genesisHeight, nil).Once() // Simulate store being at genesis (current height is 0)

		req := connect.NewRequest(&pb.GetBlockRequest{
			Identifier: &pb.GetBlockRequest_Height{
				Height: genesisHeight, // Requesting block 0 (interpreted as "latest")
			},
		})
		resp, err := server.GetBlock(context.Background(), req)

		require.Error(t, err)
		require.Nil(t, resp)
		var connectErr *connect.Error
		require.ErrorAs(t, err, &connectErr)
		require.Equal(t, connect.CodeNotFound, connectErr.Code())
		require.Contains(t, connectErr.Message(), "store is empty, no latest block available")
		mockStore.AssertExpectations(t)
	})
}

func TestGetBlock_Latest(t *testing.T) {

	mockStore := mocks.NewMockStore(t)
	logger := zerolog.Nop()
	server := NewStoreServer(mockStore, nil, nil, logger)

	latestHeight := uint64(20)
	header := &types.SignedHeader{Header: types.Header{BaseHeader: types.BaseHeader{Height: latestHeight}}}
	data := &types.Data{}
	expectedHeaderDAHeight := uint64(200)
	expectedDataDAHeight := uint64(201)

	headerDAHeightBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(headerDAHeightBytes, expectedHeaderDAHeight)
	dataDAHeightBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(dataDAHeightBytes, expectedDataDAHeight)

	// Expectation for Height (which should be called by GetLatestBlockHeight)
	mockStore.On("Height", context.Background()).Return(latestHeight, nil).Once()
	// Expectation for GetBlockData with the latest height
	mockStore.On("GetBlockData", context.Background(), latestHeight).Return(header, data, nil).Once()
	// Expectation for DA height metadata
	mockStore.On("GetMetadata", mock.Anything, store.GetHeightToDAHeightHeaderKey(latestHeight)).Return(headerDAHeightBytes, nil).Once()
	mockStore.On("GetMetadata", mock.Anything, store.GetHeightToDAHeightDataKey(latestHeight)).Return(dataDAHeightBytes, nil).Once()

	req := connect.NewRequest(&pb.GetBlockRequest{
		Identifier: &pb.GetBlockRequest_Height{
			Height: 0, // Indicates latest block
		},
	})
	resp, err := server.GetBlock(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp.Msg.Block)
	require.Equal(t, expectedHeaderDAHeight, resp.Msg.HeaderDaHeight)
	require.Equal(t, expectedDataDAHeight, resp.Msg.DataDaHeight)
	mockStore.AssertExpectations(t)
}

func TestGetState(t *testing.T) {
	// Create a mock store
	mockStore := mocks.NewMockStore(t)

	// Create test data
	state := types.State{
		AppHash:         []byte("app_hash"),
		InitialHeight:   10,
		LastBlockHeight: 10,
		LastBlockTime:   time.Now(),
		ChainID:         "test-chain",
		Version: types.Version{
			Block: 1,
			App:   1,
		},
	}

	// Setup mock expectations
	mockStore.On("GetState", mock.Anything).Return(state, nil)

	// Create server with mock store
	logger := zerolog.Nop()
	server := NewStoreServer(mockStore, nil, nil, logger)

	// Call GetState
	req := connect.NewRequest(&emptypb.Empty{})
	resp, err := server.GetState(context.Background(), req)

	// Assert expectations
	require.NoError(t, err)
	require.NotNil(t, resp.Msg.State)
	require.Equal(t, state.AppHash, resp.Msg.State.AppHash)
	require.Equal(t, state.InitialHeight, resp.Msg.State.InitialHeight)
	require.Equal(t, state.LastBlockHeight, resp.Msg.State.LastBlockHeight)
	require.Equal(t, state.LastBlockTime.UTC(), resp.Msg.State.LastBlockTime.AsTime())
	require.Equal(t, state.ChainID, resp.Msg.State.ChainId)
	require.Equal(t, state.Version.Block, resp.Msg.State.Version.Block)
	require.Equal(t, state.Version.App, resp.Msg.State.Version.App)
	mockStore.AssertExpectations(t)
}

func TestGetState_Error(t *testing.T) {
	mockStore := mocks.NewMockStore(t)
	mockStore.On("GetState", mock.Anything).Return(types.State{}, fmt.Errorf("state error"))
	logger := zerolog.Nop()
	server := NewStoreServer(mockStore, nil, nil, logger)
	resp, err := server.GetState(context.Background(), connect.NewRequest(&emptypb.Empty{}))
	require.Error(t, err)
	require.Nil(t, resp)
}

func TestGetMetadata(t *testing.T) {
	// Create a mock store
	mockStore := mocks.NewMockStore(t)

	// Create test data
	key := "test_key"
	value := []byte("test_value")

	// Setup mock expectations
	mockStore.On("GetMetadata", mock.Anything, key).Return(value, nil)

	// Create server with mock store
	logger := zerolog.Nop()
	server := NewStoreServer(mockStore, nil, nil, logger)

	// Call GetMetadata
	req := connect.NewRequest(&pb.GetMetadataRequest{
		Key: key,
	})
	resp, err := server.GetMetadata(context.Background(), req)

	// Assert expectations
	require.NoError(t, err)
	require.Equal(t, value, resp.Msg.Value)
	mockStore.AssertExpectations(t)
}

func TestGetMetadata_Error(t *testing.T) {
	mockStore := mocks.NewMockStore(t)
	mockStore.On("GetMetadata", mock.Anything, "bad").Return(nil, fmt.Errorf("meta error"))
	logger := zerolog.Nop()
	server := NewStoreServer(mockStore, nil, nil, logger)
	resp, err := server.GetMetadata(context.Background(), connect.NewRequest(&pb.GetMetadataRequest{Key: "bad"}))
	require.Error(t, err)
	require.Nil(t, resp)
}

func TestGetGenesisDaHeight(t *testing.T) {
	expectedHeight := uint64(123)
	heightBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(heightBytes, expectedHeight)

	mockStore := mocks.NewMockStore(t)
	mockStore.On("GetMetadata", mock.Anything, store.GenesisDAHeightKey).Return(heightBytes, nil).Once()

	logger := zerolog.Nop()
	server := NewStoreServer(mockStore, nil, nil, logger)

	t.Run("success", func(t *testing.T) {
		req := connect.NewRequest(&emptypb.Empty{})
		resp, err := server.GetGenesisDaHeight(context.Background(), req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, expectedHeight, resp.Msg.Height)
		mockStore.AssertExpectations(t)
	})
}

func TestGetGenesisDaHeight_NotFound(t *testing.T) {
	mockStore := mocks.NewMockStore(t)
	mockStore.On("GetMetadata", mock.Anything, store.GenesisDAHeightKey).Return(nil, fmt.Errorf("genesis DA height not found")).Once()

	logger := zerolog.Nop()
	server := NewStoreServer(mockStore, nil, nil, logger)

	req := connect.NewRequest(&emptypb.Empty{})
	resp, err := server.GetGenesisDaHeight(context.Background(), req)

	require.Error(t, err)
	require.Nil(t, resp)
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeNotFound, connectErr.Code())
	mockStore.AssertExpectations(t)
}

func TestGetGenesisDaHeight_InvalidLength(t *testing.T) {
	mockStore := mocks.NewMockStore(t)
	// Return invalid length bytes (not 8 bytes)
	invalidBytes := []byte{1, 2, 3, 4} // Only 4 bytes
	mockStore.On("GetMetadata", mock.Anything, store.GenesisDAHeightKey).Return(invalidBytes, nil).Once()

	logger := zerolog.Nop()
	server := NewStoreServer(mockStore, nil, nil, logger)

	req := connect.NewRequest(&emptypb.Empty{})
	resp, err := server.GetGenesisDaHeight(context.Background(), req)

	require.Error(t, err)
	require.Nil(t, resp)
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeNotFound, connectErr.Code())
	require.Contains(t, connectErr.Message(), "invalid metadata value")
	mockStore.AssertExpectations(t)
}

func TestGetP2PStoreInfo(t *testing.T) {
	t.Run("returns snapshots for configured stores", func(t *testing.T) {
		mockStore := mocks.NewMockStore(t)
		headerStore := headerstoremocks.NewMockStore[*sync.SignedHeaderWithDAHint](t)
		dataStore := headerstoremocks.NewMockStore[*sync.DataWithDAHint](t)
		logger := zerolog.Nop()
		server := NewStoreServer(mockStore, headerStore, dataStore, logger)

		now := time.Now().UTC()
		headerStore.On("Height").Return(uint64(12))
		headerStore.On("Head", mock.Anything).Return(makeTestSignedHeader(12, now), nil)
		headerStore.On("Tail", mock.Anything).Return(makeTestSignedHeader(7, now.Add(-time.Minute)), nil)

		dataStore.On("Height").Return(uint64(9))
		dataStore.On("Head", mock.Anything).Return(makeTestData(9, now.Add(-30*time.Second)), nil)
		dataStore.On("Tail", mock.Anything).Return(makeTestData(4, now.Add(-2*time.Minute)), nil)

		resp, err := server.GetP2PStoreInfo(context.Background(), connect.NewRequest(&emptypb.Empty{}))
		require.NoError(t, err)
		require.Len(t, resp.Msg.Stores, 2)

		require.Equal(t, "Header Store", resp.Msg.Stores[0].Label)
		require.Equal(t, uint64(12), resp.Msg.Stores[0].Height)

		require.Equal(t, "Data Store", resp.Msg.Stores[1].Label)
		require.Equal(t, uint64(9), resp.Msg.Stores[1].Height)
		require.Equal(t, uint64(9), resp.Msg.Stores[1].Head.Height)
		require.Equal(t, uint64(4), resp.Msg.Stores[1].Tail.Height)
	})

	t.Run("returns error when a store edge fails", func(t *testing.T) {
		mockStore := mocks.NewMockStore(t)
		headerStore := headerstoremocks.NewMockStore[*sync.SignedHeaderWithDAHint](t)
		logger := zerolog.Nop()
		headerStore.On("Height").Return(uint64(0))
		headerStore.On("Head", mock.Anything).Return((*sync.SignedHeaderWithDAHint)(nil), fmt.Errorf("boom"))

		server := NewStoreServer(mockStore, headerStore, nil, logger)
		resp, err := server.GetP2PStoreInfo(context.Background(), connect.NewRequest(&emptypb.Empty{}))
		require.Error(t, err)
		require.Nil(t, resp)
	})
}

func TestP2PServer_GetPeerInfo(t *testing.T) {
	mockP2P := &mocks.MockP2PRPC{}
	addr, err := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/4001")
	require.NoError(t, err)
	mockP2P.On("GetPeers").Return([]peer.AddrInfo{{ID: "id1", Addrs: []multiaddr.Multiaddr{addr}}}, nil)
	server := NewP2PServer(mockP2P)
	resp, err := server.GetPeerInfo(context.Background(), connect.NewRequest(&emptypb.Empty{}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Peers, 1)
	mockP2P.AssertExpectations(t)

	// Error case
	mockP2P2 := &mocks.MockP2PRPC{}
	mockP2P2.On("GetPeers").Return(nil, fmt.Errorf("p2p error"))
	server2 := NewP2PServer(mockP2P2)
	resp2, err2 := server2.GetPeerInfo(context.Background(), connect.NewRequest(&emptypb.Empty{}))
	require.Error(t, err2)
	require.Nil(t, resp2)
}

func TestP2PServer_GetNetInfo(t *testing.T) {
	mockP2P := &mocks.MockP2PRPC{}
	netInfo := p2p.NetworkInfo{ID: "nid", ListenAddress: []string{"addr1"}}
	mockP2P.On("GetNetworkInfo").Return(netInfo, nil)
	server := NewP2PServer(mockP2P)
	resp, err := server.GetNetInfo(context.Background(), connect.NewRequest(&emptypb.Empty{}))
	require.NoError(t, err)
	require.Equal(t, netInfo.ID, resp.Msg.NetInfo.Id)
	mockP2P.AssertExpectations(t)

	// Error case
	mockP2P2 := &mocks.MockP2PRPC{}
	mockP2P2.On("GetNetworkInfo").Return(p2p.NetworkInfo{}, fmt.Errorf("netinfo error"))
	server2 := NewP2PServer(mockP2P2)
	resp2, err2 := server2.GetNetInfo(context.Background(), connect.NewRequest(&emptypb.Empty{}))
	require.Error(t, err2)
	require.Nil(t, resp2)
}

func TestHealthLiveEndpoint(t *testing.T) {
	logger := zerolog.Nop()

	t.Run("returns OK when store is accessible", func(t *testing.T) {
		mockStore := mocks.NewMockStore(t)
		mockP2PManager := &mocks.MockP2PRPC{}
		testConfig := config.DefaultConfig()

		// Mock successful store access
		mockStore.On("Height", mock.Anything).Return(uint64(100), nil)

		handler, err := NewServiceHandler(mockStore, nil, nil, mockP2PManager, nil, logger, testConfig, nil)
		require.NoError(t, err)

		server := httptest.NewServer(handler)
		defer server.Close()

		resp, err := http.Get(server.URL + "/health/live")
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, "OK\n", string(body))
		mockStore.AssertExpectations(t)
	})

	t.Run("returns FAIL when store is not accessible", func(t *testing.T) {
		mockStore := mocks.NewMockStore(t)
		mockP2PManager := &mocks.MockP2PRPC{}
		testConfig := config.DefaultConfig()

		// Mock store access failure
		mockStore.On("Height", mock.Anything).Return(uint64(0), fmt.Errorf("store unavailable"))

		handler, err := NewServiceHandler(mockStore, nil, nil, mockP2PManager, nil, logger, testConfig, nil)
		require.NoError(t, err)

		server := httptest.NewServer(handler)
		defer server.Close()

		resp, err := http.Get(server.URL + "/health/live")
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Contains(t, string(body), "FAIL")
		mockStore.AssertExpectations(t)
	})

	t.Run("returns OK even at height 0", func(t *testing.T) {
		mockStore := mocks.NewMockStore(t)
		mockP2PManager := &mocks.MockP2PRPC{}
		testConfig := config.DefaultConfig()

		// Mock successful store access at genesis
		mockStore.On("Height", mock.Anything).Return(uint64(0), nil)

		handler, err := NewServiceHandler(mockStore, nil, nil, mockP2PManager, nil, logger, testConfig, nil)
		require.NoError(t, err)

		server := httptest.NewServer(handler)
		defer server.Close()

		resp, err := http.Get(server.URL + "/health/live")
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, "OK\n", string(body))
		mockStore.AssertExpectations(t)
	})
}

func TestHealthReadyEndpoint(t *testing.T) {
	t.Run("non-aggregator tests", func(t *testing.T) {
		cases := []struct {
			name          string
			local         uint64
			bestKnown     uint64
			peers         int
			p2pListening  bool
			lastBlockTime time.Time
			expectedCode  int
		}{
			{name: "at_head", local: 100, bestKnown: 100, peers: 1, p2pListening: true, lastBlockTime: time.Now(), expectedCode: http.StatusOK},
			{name: "within_1_block", local: 99, bestKnown: 100, peers: 1, p2pListening: true, lastBlockTime: time.Now(), expectedCode: http.StatusOK},
			{name: "within_15_blocks", local: 85, bestKnown: 100, peers: 1, p2pListening: true, lastBlockTime: time.Now(), expectedCode: http.StatusOK},
			{name: "just_over_15_blocks", local: 84, bestKnown: 100, peers: 1, p2pListening: true, lastBlockTime: time.Now(), expectedCode: http.StatusServiceUnavailable},
			{name: "local_ahead", local: 101, bestKnown: 100, peers: 1, p2pListening: true, lastBlockTime: time.Now(), expectedCode: http.StatusOK},
			{name: "no_blocks_yet", local: 0, bestKnown: 100, peers: 1, p2pListening: true, lastBlockTime: time.Now(), expectedCode: http.StatusServiceUnavailable},
			{name: "unknown_best_known", local: 100, bestKnown: 0, peers: 1, p2pListening: true, lastBlockTime: time.Now(), expectedCode: http.StatusServiceUnavailable},
			{name: "no_peers", local: 100, bestKnown: 100, peers: 0, p2pListening: true, lastBlockTime: time.Now(), expectedCode: http.StatusServiceUnavailable},
			{name: "p2p_not_listening", local: 100, bestKnown: 100, peers: 1, p2pListening: false, lastBlockTime: time.Now(), expectedCode: http.StatusServiceUnavailable},
		}

		logger := zerolog.Nop()
		testConfig := config.DefaultConfig()
		testConfig.Node.Aggregator = false

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				mockStore := mocks.NewMockStore(t)
				mockP2P := mocks.NewMockP2PRPC(t)

				// Setup P2P network info
				netInfo := p2p.NetworkInfo{
					ID: "test-node",
				}
				if tc.p2pListening {
					netInfo.ListenAddress = []string{"/ip4/0.0.0.0/tcp/26656"}
				}
				mockP2P.On("GetNetworkInfo").Return(netInfo, nil)

				// Only expect GetPeers() when P2P is listening (handler returns early if not listening)
				if tc.p2pListening {
					var peers []peer.AddrInfo
					for i := 0; i < tc.peers; i++ {
						peers = append(peers, peer.AddrInfo{})
					}
					mockP2P.On("GetPeers").Return(peers, nil)
				}

				// Only expect GetState() when peers are present (handler returns early on no peers)
				if tc.peers > 0 && tc.p2pListening {
					state := types.State{
						LastBlockHeight: tc.local,
						LastBlockTime:   tc.lastBlockTime,
					}
					mockStore.On("GetState", mock.Anything).Return(state, nil)
				}

				bestKnown := func() uint64 { return tc.bestKnown }
				handler, err := NewServiceHandler(mockStore, nil, nil, mockP2P, nil, logger, testConfig, bestKnown)
				require.NoError(t, err)
				server := httptest.NewServer(handler)
				defer server.Close()

				resp, err := http.Get(server.URL + "/health/ready")
				require.NoError(t, err)
				defer resp.Body.Close()
				require.Equal(t, tc.expectedCode, resp.StatusCode)
			})
		}
	})

	t.Run("aggregator tests", func(t *testing.T) {
		logger := zerolog.Nop()
		testConfig := config.DefaultConfig()
		testConfig.Node.Aggregator = true
		testConfig.Node.BlockTime.Duration = 1 * time.Second

		t.Run("producing blocks at expected rate", func(t *testing.T) {
			mockStore := mocks.NewMockStore(t)
			mockP2P := mocks.NewMockP2PRPC(t)

			// Setup P2P
			netInfo := p2p.NetworkInfo{
				ID:            "test-node",
				ListenAddress: []string{"/ip4/0.0.0.0/tcp/26656"},
			}
			mockP2P.On("GetNetworkInfo").Return(netInfo, nil)

			// Aggregators don't need peers check
			// No GetPeers() call expected

			// Recent block (within 5x block time)
			state := types.State{
				LastBlockHeight: 100,
				LastBlockTime:   time.Now().Add(-2 * time.Second), // 2 seconds ago, within 5x1s = 5s
			}
			mockStore.On("GetState", mock.Anything).Return(state, nil)

			bestKnown := func() uint64 { return 100 }
			handler, err := NewServiceHandler(mockStore, nil, nil, mockP2P, nil, logger, testConfig, bestKnown)
			require.NoError(t, err)
			server := httptest.NewServer(handler)
			defer server.Close()

			resp, err := http.Get(server.URL + "/health/ready")
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})

		t.Run("not producing blocks at expected rate", func(t *testing.T) {
			mockStore := mocks.NewMockStore(t)
			mockP2P := mocks.NewMockP2PRPC(t)

			// Setup P2P
			netInfo := p2p.NetworkInfo{
				ID:            "test-node",
				ListenAddress: []string{"/ip4/0.0.0.0/tcp/26656"},
			}
			mockP2P.On("GetNetworkInfo").Return(netInfo, nil)

			// Old block (beyond 5x block time)
			state := types.State{
				LastBlockHeight: 100,
				LastBlockTime:   time.Now().Add(-10 * time.Second), // 10 seconds ago, beyond 5x1s = 5s
			}
			mockStore.On("GetState", mock.Anything).Return(state, nil)

			bestKnown := func() uint64 { return 100 }
			handler, err := NewServiceHandler(mockStore, nil, nil, mockP2P, nil, logger, testConfig, bestKnown)
			require.NoError(t, err)
			server := httptest.NewServer(handler)
			defer server.Close()

			resp, err := http.Get(server.URL + "/health/ready")
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
		})
	})
}

func makeTestSignedHeader(height uint64, ts time.Time) *sync.SignedHeaderWithDAHint {
	return &sync.SignedHeaderWithDAHint{Entry: &types.SignedHeader{
		Header: types.Header{
			BaseHeader: types.BaseHeader{
				Height:  height,
				Time:    uint64(ts.UnixNano()),
				ChainID: "test-chain",
			},
			ProposerAddress: []byte{0x01},
			DataHash:        []byte{0x02},
			AppHash:         []byte{0x03},
		},
	},
	}
}

func makeTestData(height uint64, ts time.Time) *sync.DataWithDAHint {
	return &sync.DataWithDAHint{Entry: &types.Data{
		Metadata: &types.Metadata{
			ChainID: "test-chain",
			Height:  height,
			Time:    uint64(ts.UnixNano()),
		},
	},
	}
}
