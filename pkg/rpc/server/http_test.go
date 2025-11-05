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
	// Create a new ServeMux
	mux := http.NewServeMux()
	logger := zerolog.Nop()

	// Create mock store
	mockStore := mocks.NewMockStore(t)
	mockStore.On("Height", mock.Anything).Return(uint64(100), nil)

	// Register custom HTTP endpoints
	RegisterCustomHTTPEndpoints(mux, mockStore, nil, config.DefaultConfig(), nil, logger)

	// Create a new HTTP test server with the mux
	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	// Make an HTTP GET request to the /health/live endpoint
	resp, err := http.Get(testServer.URL + "/health/live")
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Check the status code
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)

	// Check the response body content
	assert.Equal(t, "OK\n", string(body)) // fmt.Fprintln adds a newline

	// Verify mock expectations
	mockStore.AssertExpectations(t)
}
