package server

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	coreda "github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/test/mocks"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDAVisualizationServer(t *testing.T) {
	da := &mocks.MockDA{}
	logger := zerolog.New(nil)

	server := NewDAVisualizationServer(da, logger, true)

	assert.NotNil(t, server)
	assert.Equal(t, da, server.da)
	assert.Equal(t, 0, len(server.submissions))
}

func TestRecordSubmission(t *testing.T) {
	da := &mocks.MockDA{}
	logger := zerolog.New(nil)
	server := NewDAVisualizationServer(da, logger, true)

	// Test recording a successful submission
	result := &coreda.ResultSubmit{
		BaseResult: coreda.BaseResult{
			Code:           coreda.StatusSuccess,
			Height:         100,
			BlobSize:       1024,
			Timestamp:      time.Now(),
			Message:        "Success",
			IDs:            [][]byte{[]byte("test-id-1"), []byte("test-id-2")},
			SubmittedCount: 2,
		},
	}

	server.RecordSubmission(result, 0.5, 2)

	assert.Equal(t, 1, len(server.submissions))
	submission := server.submissions[0]
	assert.Equal(t, uint64(100), submission.Height)
	assert.Equal(t, uint64(1024), submission.BlobSize)
	assert.Equal(t, 0.5, submission.GasPrice)
	assert.Equal(t, "Success", submission.StatusCode)
	assert.Equal(t, uint64(2), submission.NumBlobs)
	assert.Equal(t, 2, len(submission.BlobIDs))
	assert.Equal(t, hex.EncodeToString([]byte("test-id-1")), submission.BlobIDs[0])
	assert.Equal(t, hex.EncodeToString([]byte("test-id-2")), submission.BlobIDs[1])
}

func TestRecordSubmissionMemoryLimit(t *testing.T) {
	da := &mocks.MockDA{}
	logger := zerolog.New(nil)
	server := NewDAVisualizationServer(da, logger, true)

	// Add 101 submissions (more than the limit of 100)
	for i := 0; i < 101; i++ {
		result := &coreda.ResultSubmit{
			BaseResult: coreda.BaseResult{
				Code:      coreda.StatusSuccess,
				Height:    uint64(i),
				BlobSize:  uint64(i * 10),
				Timestamp: time.Now(),
			},
		}
		server.RecordSubmission(result, float64(i)*0.1, 1)
	}

	// Should only keep the last 100 submissions
	assert.Equal(t, 100, len(server.submissions))
	// First submission should be height 1 (height 0 was dropped)
	assert.Equal(t, uint64(1), server.submissions[0].Height)
	// Last submission should be height 100
	assert.Equal(t, uint64(100), server.submissions[99].Height)
}

func TestGetStatusCodeString(t *testing.T) {
	da := &mocks.MockDA{}
	logger := zerolog.New(nil)
	server := NewDAVisualizationServer(da, logger, true)

	tests := []struct {
		code     coreda.StatusCode
		expected string
	}{
		{coreda.StatusSuccess, "Success"},
		{coreda.StatusNotFound, "Not Found"},
		{coreda.StatusError, "Error"},
		{coreda.StatusTooBig, "Too Big"},
		{coreda.StatusContextDeadline, "Context Deadline"},
		{coreda.StatusUnknown, "Unknown"},
	}

	for _, tt := range tests {
		result := server.getStatusCodeString(tt.code)
		assert.Equal(t, tt.expected, result)
	}
}

func TestHandleDASubmissions(t *testing.T) {
	da := &mocks.MockDA{}
	logger := zerolog.New(nil)
	server := NewDAVisualizationServer(da, logger, true)

	// Add a test submission
	result := &coreda.ResultSubmit{
		BaseResult: coreda.BaseResult{
			Code:      coreda.StatusSuccess,
			Height:    100,
			BlobSize:  1024,
			Timestamp: time.Now(),
			IDs:       [][]byte{[]byte("test-id")},
		},
	}
	server.RecordSubmission(result, 0.5, 1)

	// Create test request
	req, err := http.NewRequest("GET", "/da/submissions", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	server.handleDASubmissions(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(1), response["total"])
	submissions, ok := response["submissions"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, 1, len(submissions))

	submission := submissions[0].(map[string]interface{})
	assert.Equal(t, float64(100), submission["height"])
	assert.Equal(t, float64(1024), submission["blob_size"])
	assert.Equal(t, 0.5, submission["gas_price"])
	assert.Equal(t, "Success", submission["status_code"])
}

func TestHandleDABlobDetailsMissingID(t *testing.T) {
	da := &mocks.MockDA{}
	logger := zerolog.New(nil)
	server := NewDAVisualizationServer(da, logger, true)

	req, err := http.NewRequest("GET", "/da/blob", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	server.handleDABlobDetails(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "Missing blob ID parameter")
}

func TestHandleDABlobDetailsInvalidID(t *testing.T) {
	da := &mocks.MockDA{}
	logger := zerolog.New(nil)
	server := NewDAVisualizationServer(da, logger, true)

	req, err := http.NewRequest("GET", "/da/blob?id=invalid-hex", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	server.handleDABlobDetails(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "Invalid blob ID format")
}

func TestHandleDAVisualizationHTML(t *testing.T) {
	da := &mocks.MockDA{}
	logger := zerolog.New(nil)
	server := NewDAVisualizationServer(da, logger, true)

	// Add a test submission
	result := &coreda.ResultSubmit{
		BaseResult: coreda.BaseResult{
			Code:      coreda.StatusSuccess,
			Height:    100,
			BlobSize:  1024,
			Timestamp: time.Now(),
			Message:   "Test submission",
		},
	}
	server.RecordSubmission(result, 0.5, 1)

	req, err := http.NewRequest("GET", "/da", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	server.handleDAVisualizationHTML(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "text/html", rr.Header().Get("Content-Type"))

	body := rr.Body.String()
	assert.Contains(t, body, "DA Layer Visualization")
	assert.Contains(t, body, "100")             // Height
	assert.Contains(t, body, "Success")         // Status
	assert.Contains(t, body, "1024")            // Size
	assert.Contains(t, body, "Test submission") // Message
}

func TestGlobalDAVisualizationServer(t *testing.T) {
	da := &mocks.MockDA{}
	logger := zerolog.New(nil)
	server := NewDAVisualizationServer(da, logger, true)

	// Initially should be nil
	assert.Nil(t, GetDAVisualizationServer())

	// Set the server
	SetDAVisualizationServer(server)
	assert.Equal(t, server, GetDAVisualizationServer())

	// Set to nil
	SetDAVisualizationServer(nil)
	assert.Nil(t, GetDAVisualizationServer())
}

func TestRegisterCustomHTTPEndpointsDAVisualization(t *testing.T) {
	da := &mocks.MockDA{}
	logger := zerolog.New(nil)
	server := NewDAVisualizationServer(da, logger, true)

	// Add test submission
	result := &coreda.ResultSubmit{
		BaseResult: coreda.BaseResult{
			Code:      coreda.StatusSuccess,
			Height:    100,
			BlobSize:  1024,
			Timestamp: time.Now(),
		},
	}
	server.RecordSubmission(result, 0.5, 1)

	// Set global server
	SetDAVisualizationServer(server)
	defer SetDAVisualizationServer(nil)

	// Create mux and register endpoints
	mux := http.NewServeMux()
	RegisterCustomHTTPEndpoints(mux, nil, nil, config.DefaultConfig(), nil)

	// Test /da endpoint
	req, err := http.NewRequest("GET", "/da", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "text/html", rr.Header().Get("Content-Type"))

	// Test /da/submissions endpoint
	req, err = http.NewRequest("GET", "/da/submissions", nil)
	require.NoError(t, err)

	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	// Test /da/blob endpoint (missing ID should return 400)
	req, err = http.NewRequest("GET", "/da/blob", nil)
	require.NoError(t, err)

	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestRegisterCustomHTTPEndpointsWithoutServer(t *testing.T) {
	// Ensure no server is set
	SetDAVisualizationServer(nil)

	mux := http.NewServeMux()
	RegisterCustomHTTPEndpoints(mux, nil, nil, config.DefaultConfig(), nil)

	// Test that endpoints return service unavailable when server is not set
	endpoints := []string{"/da", "/da/submissions", "/da/blob"}

	for _, endpoint := range endpoints {
		req, err := http.NewRequest("GET", endpoint, nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
		assert.Contains(t, strings.ToLower(rr.Body.String()), "not available")
	}
}
