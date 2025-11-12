package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/test/mocks"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
