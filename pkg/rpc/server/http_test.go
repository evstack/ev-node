package server

import (
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

	resp, err := http.Get(testServer.URL + "/health/live")
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)

	assert.Equal(t, "OK\n", string(body))

	mockStore.AssertExpectations(t)
}

func TestHealthReady_aggregatorBlockDelay(t *testing.T) {
	ctx := t.Context()
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

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/health/ready", nil)
			require.NoError(t, err)
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			t.Cleanup(func() { _ = resp.Body.Close() })

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			assert.Equal(t, tc.expStatusCode, resp.StatusCode)
			assert.Equal(t, tc.expBody, string(body))
		})
	}
}
