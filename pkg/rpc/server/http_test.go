package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/test/mocks"
	"github.com/evstack/ev-node/types"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRegisterCustomHTTPEndpoints(t *testing.T) {
	mux := http.NewServeMux()
	logger := zerolog.Nop()

	mockStore := mocks.NewMockStore(t)
	mockStore.On("Height", mock.Anything).Return(uint64(100), nil)

	RegisterCustomHTTPEndpoints(mux, mockStore, nil, config.DefaultConfig(), nil, logger, nil)

	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	req, err := http.NewRequest(http.MethodGet, testServer.URL+"/health/live", nil)
	assert.NoError(t, err)
	resp, err := http.DefaultClient.Do(req) //nolint:gosec // test-only request to httptest server
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)

	assert.Equal(t, "OK\n", string(body))

	mockStore.AssertExpectations(t)
}

type testRaftNodeSource struct {
	isLeader bool
	leaderID string
	nodeID   string
}

func (t testRaftNodeSource) IsLeader() bool {
	return t.isLeader
}

func (t testRaftNodeSource) LeaderID() string {
	return t.leaderID
}

func (t testRaftNodeSource) NodeID() string {
	return t.nodeID
}

func TestRegisterCustomHTTPEndpoints_RaftNodeStatus(t *testing.T) {
	type bodyShape struct {
		IsLeader bool   `json:"is_leader"`
		NodeID   string `json:"node_id"`
	}

	cases := []struct {
		name           string
		node           testRaftNodeSource
		method         string
		wantStatus     int
		wantIsLeader   bool
		wantNodeID     string
		skipBodyDecode bool
	}{
		{
			// leaderID == nodeID: handler derives is_leader=true from LeaderID(),
			// regardless of the IsLeader() field on testRaftNodeSource.
			name:         "leader matches — is_leader true",
			node:         testRaftNodeSource{leaderID: "node-a", nodeID: "node-a"},
			method:       http.MethodGet,
			wantStatus:   http.StatusOK,
			wantIsLeader: true,
			wantNodeID:   "node-a",
		},
		{
			// leaderID != nodeID: handler derives is_leader=false.
			name:         "leader differs — is_leader false",
			node:         testRaftNodeSource{leaderID: "node-b", nodeID: "node-a"},
			method:       http.MethodGet,
			wantStatus:   http.StatusOK,
			wantIsLeader: false,
			wantNodeID:   "node-a",
		},
		{
			// empty leaderID: fallback — is_leader=false (no elected leader known).
			name:         "empty leaderID fallback — is_leader false",
			node:         testRaftNodeSource{leaderID: "", nodeID: "node-a"},
			method:       http.MethodGet,
			wantStatus:   http.StatusOK,
			wantIsLeader: false,
			wantNodeID:   "node-a",
		},
		{
			name:           "non-GET method — 405",
			node:           testRaftNodeSource{},
			method:         http.MethodPost,
			wantStatus:     http.StatusMethodNotAllowed,
			skipBodyDecode: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mux := http.NewServeMux()
			RegisterCustomHTTPEndpoints(mux, nil, nil, config.DefaultConfig(), nil, zerolog.Nop(), tc.node)

			ts := httptest.NewServer(mux)
			t.Cleanup(ts.Close)

			req, err := http.NewRequest(tc.method, ts.URL+"/raft/node", nil)
			require.NoError(t, err)
			resp, err := http.DefaultClient.Do(req) //nolint:gosec // test-only request to httptest server
			require.NoError(t, err)
			t.Cleanup(func() { _ = resp.Body.Close() })

			require.Equal(t, tc.wantStatus, resp.StatusCode)
			if tc.skipBodyDecode {
				return
			}

			var body bodyShape
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
			assert.Equal(t, tc.wantIsLeader, body.IsLeader)
			assert.Equal(t, tc.wantNodeID, body.NodeID)
		})
	}
}

func TestHealthReady_aggregatorBlockDelay(t *testing.T) {
	logger := zerolog.Nop()

	type spec struct {
		lazy          bool
		blockTime     time.Duration
		lazyInterval  time.Duration
		delay         time.Duration
		expStatusCode int
		expBody       string
	}

	specs := map[string]spec{
		"aggregator within non-lazy threshold": {
			lazy:          false,
			blockTime:     200 * time.Millisecond,
			lazyInterval:  0,
			delay:         800 * time.Millisecond, // 5x blockTime = 1s, so 0.8s is OK
			expStatusCode: http.StatusOK,
			expBody:       "READY\n",
		},
		"aggregator exceeds non-lazy threshold": {
			lazy:          false,
			blockTime:     200 * time.Millisecond,
			lazyInterval:  0,
			delay:         1500 * time.Millisecond, // > 1s threshold
			expStatusCode: http.StatusServiceUnavailable,
			expBody:       "UNREADY: aggregator not producing blocks at expected rate\n",
		},
		"aggregator within lazy threshold": {
			lazy:          true,
			blockTime:     0,
			lazyInterval:  300 * time.Millisecond,
			delay:         500 * time.Millisecond, // 2x lazyInterval = 600ms, so 0.5s is OK
			expStatusCode: http.StatusOK,
			expBody:       "READY\n",
		},
		"aggregator exceeds lazy threshold": {
			lazy:          true,
			blockTime:     0,
			lazyInterval:  300 * time.Millisecond,
			delay:         800 * time.Millisecond, // > 600ms threshold
			expStatusCode: http.StatusServiceUnavailable,
			expBody:       "UNREADY: aggregator not producing blocks at expected rate\n",
		},
	}

	for name, tc := range specs {
		t.Run(name, func(t *testing.T) {
			mux := http.NewServeMux()

			cfg := config.DefaultConfig()
			cfg.Node.Aggregator = true
			if tc.blockTime > 0 {
				cfg.Node.BlockTime = config.DurationWrapper{Duration: tc.blockTime}
			}
			cfg.Node.LazyMode = tc.lazy
			if tc.lazy {
				cfg.Node.LazyBlockInterval = config.DurationWrapper{Duration: tc.lazyInterval}
			}

			mockStore := mocks.NewMockStore(t)
			state := types.State{
				LastBlockHeight: 10,
				LastBlockTime:   time.Now().Add(-tc.delay),
			}
			mockStore.On("GetState", mock.Anything).Return(state, nil)

			bestKnownHeightProvider := func() uint64 { return state.LastBlockHeight }

			RegisterCustomHTTPEndpoints(mux, mockStore, nil, cfg, bestKnownHeightProvider, logger, nil)

			ts := httptest.NewServer(mux)
			t.Cleanup(ts.Close)

			req, err := http.NewRequest(http.MethodGet, ts.URL+"/health/ready", nil)
			require.NoError(t, err)
			resp, err := http.DefaultClient.Do(req) //nolint:gosec // ok to use default client in tests
			require.NoError(t, err)
			t.Cleanup(func() { _ = resp.Body.Close() })

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			assert.Equal(t, tc.expStatusCode, resp.StatusCode)
			assert.Equal(t, tc.expBody, string(body))
		})
	}
}
