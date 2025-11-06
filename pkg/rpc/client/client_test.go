package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/p2p"
	"github.com/evstack/ev-node/pkg/rpc/server"
	"github.com/evstack/ev-node/test/mocks"
	"github.com/evstack/ev-node/types"
	rpc "github.com/evstack/ev-node/types/pb/evnode/v1/v1connect"
)

// setupTestServerOptions holds optional parameters for setupTestServer
type setupTestServerOptions struct {
	config                  *config.Config
	bestKnownHeightProvider server.BestKnownHeightProvider
}

// setupTestServer creates a test server with mock store and mock p2p manager.
// An optional custom config can be provided; if not provided, uses DefaultConfig with test-headers namespace.
func setupTestServer(t *testing.T, mockStore *mocks.MockStore, mockP2P *mocks.MockP2PRPC, customConfig ...config.Config) (*httptest.Server, *Client) {
	return setupTestServerWithOptions(t, mockStore, mockP2P, setupTestServerOptions{
		config: func() *config.Config {
			if len(customConfig) > 0 {
				return &customConfig[0]
			}
			return nil
		}(),
		bestKnownHeightProvider: func() uint64 { return 100 }, // Default provider
	})
}

// setupTestServerWithOptions creates a test server with full control over options
func setupTestServerWithOptions(t *testing.T, mockStore *mocks.MockStore, mockP2P *mocks.MockP2PRPC, opts setupTestServerOptions) (*httptest.Server, *Client) {
	// Create a new HTTP test server
	mux := http.NewServeMux()

	// Create the servers
	logger := zerolog.Nop()

	// Use custom config if provided, otherwise use default
	var testConfig config.Config
	if opts.config != nil {
		testConfig = *opts.config
	} else {
		testConfig = config.DefaultConfig()
		testConfig.DA.Namespace = "test-headers"
	}

	storeServer := server.NewStoreServer(mockStore, logger)
	p2pServer := server.NewP2PServer(mockP2P)
	configServer := server.NewConfigServer(testConfig, nil, logger)

	// Register the store service
	storePath, storeHandler := rpc.NewStoreServiceHandler(storeServer)
	mux.Handle(storePath, storeHandler)

	// Register the p2p service
	p2pPath, p2pHandler := rpc.NewP2PServiceHandler(p2pServer)
	mux.Handle(p2pPath, p2pHandler)

	// Register the config service
	configPath, configHandler := rpc.NewConfigServiceHandler(configServer)
	mux.Handle(configPath, configHandler)

	// Register custom HTTP endpoints (including health)
	bestKnownHeight := opts.bestKnownHeightProvider
	if bestKnownHeight == nil {
		bestKnownHeight = func() uint64 { return 100 }
	}
	server.RegisterCustomHTTPEndpoints(mux, mockStore, mockP2P, testConfig, bestKnownHeight, logger)

	// Create an HTTP server with h2c for HTTP/2 support
	testServer := httptest.NewServer(h2c.NewHandler(mux, &http2.Server{}))

	// Create a client that connects to the test server
	client := NewClient(testServer.URL)

	return testServer, client
}

func TestClientGetState(t *testing.T) {
	// Create mocks
	mockStore := mocks.NewMockStore(t)
	mockP2P := mocks.NewMockP2PRPC(t)

	// Create test data
	state := types.State{
		AppHash:         []byte("app_hash"),
		InitialHeight:   10,
		LastBlockHeight: 10,
		LastBlockTime:   time.Now(),
	}

	// Setup mock expectations
	mockStore.On("GetState", mock.Anything).Return(state, nil)

	// Setup test server and client
	testServer, client := setupTestServer(t, mockStore, mockP2P)
	defer testServer.Close()

	// Call GetState
	resultState, err := client.GetState(context.Background())

	// Assert expectations
	require.NoError(t, err)
	require.Equal(t, state.AppHash, resultState.AppHash)
	require.Equal(t, state.InitialHeight, resultState.InitialHeight)
	require.Equal(t, state.LastBlockHeight, resultState.LastBlockHeight)
	require.Equal(t, state.LastBlockTime.UTC(), resultState.LastBlockTime.AsTime())
	mockStore.AssertExpectations(t)
}

func TestClientGetMetadata(t *testing.T) {
	// Create mocks
	mockStore := mocks.NewMockStore(t)
	mockP2P := mocks.NewMockP2PRPC(t)

	// Create test data
	key := "test_key"
	value := []byte("test_value")

	// Setup mock expectations
	mockStore.On("GetMetadata", mock.Anything, key).Return(value, nil)

	// Setup test server and client
	testServer, client := setupTestServer(t, mockStore, mockP2P)
	defer testServer.Close()

	// Call GetMetadata
	resultValue, err := client.GetMetadata(context.Background(), key)

	// Assert expectations
	require.NoError(t, err)
	require.Equal(t, value, resultValue)
	mockStore.AssertExpectations(t)
}

func TestClientGetBlockByHeight(t *testing.T) {
	// Create mocks
	mockStore := mocks.NewMockStore(t)
	mockP2P := mocks.NewMockP2PRPC(t)

	// Create test data
	height := uint64(10)
	header := &types.SignedHeader{}
	data := &types.Data{}

	// Setup mock expectations
	mockStore.On("GetBlockData", mock.Anything, height).Return(header, data, nil)

	// Setup test server and client
	testServer, client := setupTestServer(t, mockStore, mockP2P)
	defer testServer.Close()

	// Call GetBlockByHeight
	block, err := client.GetBlockByHeight(context.Background(), height)

	// Assert expectations
	require.NoError(t, err)
	require.NotNil(t, block)
	mockStore.AssertExpectations(t)
}

func TestClientGetBlockByHash(t *testing.T) {
	// Create mocks
	mockStore := mocks.NewMockStore(t)
	mockP2P := mocks.NewMockP2PRPC(t)

	// Create test data
	hash := []byte("block_hash")
	header := &types.SignedHeader{}
	data := &types.Data{}

	// Setup mock expectations
	mockStore.On("GetBlockByHash", mock.Anything, hash).Return(header, data, nil)

	// Setup test server and client
	testServer, client := setupTestServer(t, mockStore, mockP2P)
	defer testServer.Close()

	// Call GetBlockByHash
	block, err := client.GetBlockByHash(context.Background(), hash)

	// Assert expectations
	require.NoError(t, err)
	require.NotNil(t, block)
	mockStore.AssertExpectations(t)
}

func TestClientGetPeerInfo(t *testing.T) {
	// Create mocks
	mockStore := mocks.NewMockStore(t)
	mockP2P := mocks.NewMockP2PRPC(t)

	addr, err := multiaddr.NewMultiaddr("/ip4/0.0.0.0/tcp/8000")
	require.NoError(t, err)

	// Create test data
	peers := []peer.AddrInfo{
		{
			ID:    "3bM8hezDN5",
			Addrs: []multiaddr.Multiaddr{addr},
		},
		{
			ID:    "3tSMH9AUGpeoe4",
			Addrs: []multiaddr.Multiaddr{addr},
		},
	}

	// Setup mock expectations
	mockP2P.On("GetPeers").Return(peers, nil)

	// Setup test server and client
	testServer, client := setupTestServer(t, mockStore, mockP2P)
	defer testServer.Close()

	// Call GetPeerInfo
	resultPeers, err := client.GetPeerInfo(context.Background())

	// Assert expectations
	require.NoError(t, err)
	require.Len(t, resultPeers, 2)
	require.Equal(t, "3tSMH9AUGpeoe4", resultPeers[0].Id)
	require.Equal(t, "{3tSMH9AUGpeoe4: [/ip4/0.0.0.0/tcp/8000]}", resultPeers[0].Address)
	require.Equal(t, "Kv9im1EaxaZ2KEviHvT", resultPeers[1].Id)
	require.Equal(t, "{Kv9im1EaxaZ2KEviHvT: [/ip4/0.0.0.0/tcp/8000]}", resultPeers[1].Address)
	mockP2P.AssertExpectations(t)
}

func TestClientGetNetInfo(t *testing.T) {
	// Create mocks
	mockStore := mocks.NewMockStore(t)
	mockP2P := mocks.NewMockP2PRPC(t)

	// Create test data
	netInfo := p2p.NetworkInfo{
		ID:            "node1",
		ListenAddress: []string{"0.0.0.0:26656"},
	}

	// Setup mock expectations
	mockP2P.On("GetNetworkInfo").Return(netInfo, nil)

	// Setup test server and client
	testServer, client := setupTestServer(t, mockStore, mockP2P)
	defer testServer.Close()

	// Call GetNetInfo
	resultNetInfo, err := client.GetNetInfo(context.Background())

	// Assert expectations
	require.NoError(t, err)
	require.Equal(t, "node1", resultNetInfo.Id)
	require.Equal(t, "0.0.0.0:26656", resultNetInfo.ListenAddresses[0])
	mockP2P.AssertExpectations(t)
}

func TestClientGetHealth(t *testing.T) {
	t.Run("returns PASS when store is accessible", func(t *testing.T) {
		mockStore := mocks.NewMockStore(t)
		mockP2P := mocks.NewMockP2PRPC(t)

		// Mock Height to return successfully
		mockStore.On("Height", mock.Anything).Return(uint64(100), nil)

		testServer, client := setupTestServer(t, mockStore, mockP2P)
		defer testServer.Close()

		healthStatus, err := client.GetHealth(context.Background())

		require.NoError(t, err)
		require.Equal(t, HealthStatus_PASS, healthStatus)
		mockStore.AssertExpectations(t)
	})

	t.Run("returns FAIL when store is not accessible", func(t *testing.T) {
		mockStore := mocks.NewMockStore(t)
		mockP2P := mocks.NewMockP2PRPC(t)

		// Mock Height to return an error
		mockStore.On("Height", mock.Anything).Return(uint64(0), assert.AnError)

		testServer, client := setupTestServer(t, mockStore, mockP2P)
		defer testServer.Close()

		healthStatus, err := client.GetHealth(context.Background())

		require.NoError(t, err)
		require.Equal(t, HealthStatus_FAIL, healthStatus)
		mockStore.AssertExpectations(t)
	})

	t.Run("returns PASS even at height 0", func(t *testing.T) {
		mockStore := mocks.NewMockStore(t)
		mockP2P := mocks.NewMockP2PRPC(t)

		// Mock Height to return 0 successfully (genesis state)
		mockStore.On("Height", mock.Anything).Return(uint64(0), nil)

		testServer, client := setupTestServer(t, mockStore, mockP2P)
		defer testServer.Close()

		healthStatus, err := client.GetHealth(context.Background())

		require.NoError(t, err)
		require.Equal(t, HealthStatus_PASS, healthStatus)
		mockStore.AssertExpectations(t)
	})
}

func TestClientGetReadiness(t *testing.T) {
	t.Run("returns READY when all checks pass", func(t *testing.T) {
		mockStore := mocks.NewMockStore(t)
		mockP2P := mocks.NewMockP2PRPC(t)

		// Setup P2P network info
		netInfo := p2p.NetworkInfo{
			ID:            "test-node",
			ListenAddress: []string{"/ip4/0.0.0.0/tcp/26656"},
		}
		mockP2P.On("GetNetworkInfo").Return(netInfo, nil)

		// Setup peers (non-aggregator needs peers)
		peers := []peer.AddrInfo{{}}
		mockP2P.On("GetPeers").Return(peers, nil)

		// Setup state
		state := types.State{
			LastBlockHeight: 100,
			LastBlockTime:   time.Now(),
		}
		mockStore.On("GetState", mock.Anything).Return(state, nil)

		testServer, client := setupTestServer(t, mockStore, mockP2P)
		defer testServer.Close()

		readiness, err := client.GetReadiness(context.Background())

		require.NoError(t, err)
		require.Equal(t, ReadinessStatus_READY, readiness)
		mockStore.AssertExpectations(t)
		mockP2P.AssertExpectations(t)
	})

	t.Run("returns UNREADY when P2P not listening", func(t *testing.T) {
		mockStore := mocks.NewMockStore(t)
		mockP2P := mocks.NewMockP2PRPC(t)

		// Setup P2P network info with no listen addresses
		netInfo := p2p.NetworkInfo{
			ID: "test-node",
		}
		mockP2P.On("GetNetworkInfo").Return(netInfo, nil)

		testServer, client := setupTestServer(t, mockStore, mockP2P)
		defer testServer.Close()

		readiness, err := client.GetReadiness(context.Background())

		require.NoError(t, err)
		require.Equal(t, ReadinessStatus_UNREADY, readiness)
		mockP2P.AssertExpectations(t)
	})

	t.Run("returns UNREADY when no peers (non-aggregator)", func(t *testing.T) {
		mockStore := mocks.NewMockStore(t)
		mockP2P := mocks.NewMockP2PRPC(t)

		// Setup P2P network info
		netInfo := p2p.NetworkInfo{
			ID:            "test-node",
			ListenAddress: []string{"/ip4/0.0.0.0/tcp/26656"},
		}
		mockP2P.On("GetNetworkInfo").Return(netInfo, nil)

		// No peers
		mockP2P.On("GetPeers").Return([]peer.AddrInfo{}, nil)

		testServer, client := setupTestServer(t, mockStore, mockP2P)
		defer testServer.Close()

		readiness, err := client.GetReadiness(context.Background())

		require.NoError(t, err)
		require.Equal(t, ReadinessStatus_UNREADY, readiness)
		mockP2P.AssertExpectations(t)
	})
}

func TestClientGetNamespace(t *testing.T) {
	// Create mocks
	mockStore := mocks.NewMockStore(t)
	mockP2P := mocks.NewMockP2PRPC(t)

	// Setup test server and client
	testServer, client := setupTestServer(t, mockStore, mockP2P)
	defer testServer.Close()

	// Call GetNamespace
	namespaceResp, err := client.GetNamespace(context.Background())

	// Assert expectations
	require.NoError(t, err)
	require.NotNil(t, namespaceResp)
	// The namespace should be derived from the config we set in setupTestServer
	require.NotEmpty(t, namespaceResp.HeaderNamespace)
	require.NotEmpty(t, namespaceResp.DataNamespace)
}
