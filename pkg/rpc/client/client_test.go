package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	goheader "github.com/celestiaorg/go-header"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/p2p"
	"github.com/evstack/ev-node/pkg/rpc/server"
	"github.com/evstack/ev-node/test/mocks"
	headerstoremocks "github.com/evstack/ev-node/test/mocks/external"
	"github.com/evstack/ev-node/types"
	rpc "github.com/evstack/ev-node/types/pb/evnode/v1/v1connect"
)

func setupTestServer(
	t *testing.T,
	mockStore *mocks.MockStore,
	headerStore goheader.Store[*types.P2PSignedHeader],
	dataStore goheader.Store[*types.P2PData],
	mockP2P *mocks.MockP2PRPC,
) (*httptest.Server, *Client) {
	t.Helper()
	mux := http.NewServeMux()

	logger := zerolog.Nop()
	storeServer := server.NewStoreServer(mockStore, headerStore, dataStore, logger)
	p2pServer := server.NewP2PServer(mockP2P)

	testConfig := config.DefaultConfig()
	testConfig.DA.Namespace = "test-headers"
	configServer := server.NewConfigServer(testConfig, nil, logger)

	storePath, storeHandler := rpc.NewStoreServiceHandler(storeServer)
	mux.Handle(storePath, storeHandler)

	p2pPath, p2pHandler := rpc.NewP2PServiceHandler(p2pServer)
	mux.Handle(p2pPath, p2pHandler)

	configPath, configHandler := rpc.NewConfigServiceHandler(configServer)
	mux.Handle(configPath, configHandler)

	testServer := httptest.NewServer(h2c.NewHandler(mux, &http2.Server{}))
	client := NewClient(testServer.URL)

	return testServer, client
}

func TestClientGetState(t *testing.T) {
	mockStore := mocks.NewMockStore(t)
	mockP2P := mocks.NewMockP2PRPC(t)

	state := types.State{
		AppHash:         []byte("app_hash"),
		InitialHeight:   10,
		LastBlockHeight: 10,
		LastBlockTime:   time.Now(),
	}

	mockStore.On("GetState", mock.Anything).Return(state, nil)

	testServer, client := setupTestServer(t, mockStore, nil, nil, mockP2P)
	defer testServer.Close()

	resultState, err := client.GetState(context.Background())

	require.NoError(t, err)
	require.Equal(t, state.AppHash, resultState.AppHash)
	require.Equal(t, state.InitialHeight, resultState.InitialHeight)
	require.Equal(t, state.LastBlockHeight, resultState.LastBlockHeight)
	require.Equal(t, state.LastBlockTime.UTC(), resultState.LastBlockTime.AsTime())
	mockStore.AssertExpectations(t)
}

func TestClientGetMetadata(t *testing.T) {
	mockStore := mocks.NewMockStore(t)
	mockP2P := mocks.NewMockP2PRPC(t)

	key := "test_key"
	value := []byte("test_value")

	mockStore.On("GetMetadata", mock.Anything, key).Return(value, nil)

	testServer, client := setupTestServer(t, mockStore, nil, nil, mockP2P)
	defer testServer.Close()

	resultValue, err := client.GetMetadata(context.Background(), key)

	require.NoError(t, err)
	require.Equal(t, value, resultValue)
	mockStore.AssertExpectations(t)
}

func TestClientGetP2PStoreInfo(t *testing.T) {
	mockStore := mocks.NewMockStore(t)
	mockP2P := mocks.NewMockP2PRPC(t)
	headerStore := headerstoremocks.NewMockStore[*types.P2PSignedHeader](t)
	dataStore := headerstoremocks.NewMockStore[*types.P2PData](t)

	now := time.Now().UTC()

	headerHead := testSignedHeader(10, now)
	headerTail := testSignedHeader(5, now.Add(-time.Minute))
	headerStore.On("Height").Return(uint64(10))
	headerStore.On("Head", mock.Anything).Return(headerHead, nil)
	headerStore.On("Tail", mock.Anything).Return(headerTail, nil)

	dataHead := testData(8, now.Add(-30*time.Second))
	dataTail := testData(4, now.Add(-2*time.Minute))
	dataStore.On("Height").Return(uint64(8))
	dataStore.On("Head", mock.Anything).Return(dataHead, nil)
	dataStore.On("Tail", mock.Anything).Return(dataTail, nil)

	testServer, client := setupTestServer(t, mockStore, headerStore, dataStore, mockP2P)
	defer testServer.Close()

	stores, err := client.GetP2PStoreInfo(context.Background())
	require.NoError(t, err)
	require.Len(t, stores, 2)

	require.Equal(t, "Header Store", stores[0].Label)
	require.Equal(t, uint64(10), stores[0].Head.Height)
	require.Equal(t, uint64(5), stores[0].Tail.Height)

	require.Equal(t, "Data Store", stores[1].Label)
	require.Equal(t, uint64(8), stores[1].Height)
	require.Equal(t, uint64(4), stores[1].Tail.Height)
}

func TestClientGetBlockByHeight(t *testing.T) {
	mockStore := mocks.NewMockStore(t)
	mockP2P := mocks.NewMockP2PRPC(t)

	height := uint64(10)
	header := &types.SignedHeader{}
	data := &types.Data{}

	mockStore.On("GetBlockData", mock.Anything, height).Return(header, data, nil)

	testServer, client := setupTestServer(t, mockStore, nil, nil, mockP2P)
	defer testServer.Close()

	block, err := client.GetBlockByHeight(context.Background(), height)

	require.NoError(t, err)
	require.NotNil(t, block)
	mockStore.AssertExpectations(t)
}

func TestClientGetBlockByHash(t *testing.T) {
	mockStore := mocks.NewMockStore(t)
	mockP2P := mocks.NewMockP2PRPC(t)

	hash := []byte("block_hash")
	header := &types.SignedHeader{}
	data := &types.Data{}

	mockStore.On("GetBlockByHash", mock.Anything, hash).Return(header, data, nil)

	testServer, client := setupTestServer(t, mockStore, nil, nil, mockP2P)
	defer testServer.Close()

	block, err := client.GetBlockByHash(context.Background(), hash)

	require.NoError(t, err)
	require.NotNil(t, block)
	mockStore.AssertExpectations(t)
}

func TestClientGetPeerInfo(t *testing.T) {
	mockStore := mocks.NewMockStore(t)
	mockP2P := mocks.NewMockP2PRPC(t)

	addr, err := multiaddr.NewMultiaddr("/ip4/0.0.0.0/tcp/8000")
	require.NoError(t, err)

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

	mockP2P.On("GetPeers").Return(peers, nil)

	testServer, client := setupTestServer(t, mockStore, nil, nil, mockP2P)
	defer testServer.Close()

	resultPeers, err := client.GetPeerInfo(context.Background())

	require.NoError(t, err)
	require.Len(t, resultPeers, 2)
	require.Equal(t, "3tSMH9AUGpeoe4", resultPeers[0].Id)
	require.Equal(t, "{3tSMH9AUGpeoe4: [/ip4/0.0.0.0/tcp/8000]}", resultPeers[0].Address)
	require.Equal(t, "Kv9im1EaxaZ2KEviHvT", resultPeers[1].Id)
	require.Equal(t, "{Kv9im1EaxaZ2KEviHvT: [/ip4/0.0.0.0/tcp/8000]}", resultPeers[1].Address)
	mockP2P.AssertExpectations(t)
}

func TestClientGetNetInfo(t *testing.T) {
	mockStore := mocks.NewMockStore(t)
	mockP2P := mocks.NewMockP2PRPC(t)

	netInfo := p2p.NetworkInfo{
		ID:            "node1",
		ListenAddress: []string{"0.0.0.0:26656"},
	}

	mockP2P.On("GetNetworkInfo").Return(netInfo, nil)

	testServer, client := setupTestServer(t, mockStore, nil, nil, mockP2P)
	defer testServer.Close()

	resultNetInfo, err := client.GetNetInfo(context.Background())

	require.NoError(t, err)
	require.Equal(t, "node1", resultNetInfo.Id)
	require.Equal(t, "0.0.0.0:26656", resultNetInfo.ListenAddresses[0])
	mockP2P.AssertExpectations(t)
}

func TestClientGetNamespace(t *testing.T) {
	mockStore := mocks.NewMockStore(t)
	mockP2P := mocks.NewMockP2PRPC(t)

	testServer, client := setupTestServer(t, mockStore, nil, nil, mockP2P)
	defer testServer.Close()

	namespaceResp, err := client.GetNamespace(context.Background())

	require.NoError(t, err)
	require.NotNil(t, namespaceResp)
	// The namespace should be derived from the config we set in setupTestServer
	require.NotEmpty(t, namespaceResp.HeaderNamespace)
	require.NotEmpty(t, namespaceResp.DataNamespace)
}

func testSignedHeader(height uint64, ts time.Time) *types.P2PSignedHeader {
	return &types.P2PSignedHeader{
		SignedHeader: types.SignedHeader{
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

func testData(height uint64, ts time.Time) *types.P2PData {
	return &types.P2PData{
		Data: types.Data{
			Metadata: &types.Metadata{
				ChainID: "test-chain",
				Height:  height,
				Time:    uint64(ts.UnixNano()),
			},
		},
	}
}
