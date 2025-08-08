package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/evstack/ev-node/test/mocks"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNonAggregatorDAVisualizationServer(t *testing.T) {
	da := &mocks.MockDA{}
	logger := zerolog.New(nil)

	// Create a non-aggregator server
	server := NewDAVisualizationServer(da, logger, false)

	assert.NotNil(t, server)
	assert.Equal(t, da, server.da)
	assert.Equal(t, 0, len(server.submissions))
	assert.False(t, server.isAggregator)
}

func TestNonAggregatorHandleDASubmissions(t *testing.T) {
	da := &mocks.MockDA{}
	logger := zerolog.New(nil)

	// Create a non-aggregator server
	server := NewDAVisualizationServer(da, logger, false)

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

	assert.Equal(t, false, response["is_aggregator"])
	assert.Equal(t, float64(0), response["total"])
	assert.Contains(t, response["message"], "not an aggregator")

	submissions, ok := response["submissions"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, 0, len(submissions))
}

func TestNonAggregatorHandleDAStats(t *testing.T) {
	da := &mocks.MockDA{}
	logger := zerolog.New(nil)

	// Create a non-aggregator server
	server := NewDAVisualizationServer(da, logger, false)

	// Create test request
	req, err := http.NewRequest("GET", "/da/stats", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	server.handleDAStats(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, false, response["is_aggregator"])
	assert.Equal(t, float64(0), response["total_submissions"])
	assert.Contains(t, response["message"], "not an aggregator")
}

func TestNonAggregatorHandleDAHealth(t *testing.T) {
	da := &mocks.MockDA{}
	logger := zerolog.New(nil)

	// Create a non-aggregator server
	server := NewDAVisualizationServer(da, logger, false)

	// Create test request
	req, err := http.NewRequest("GET", "/da/health", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	server.handleDAHealth(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, false, response["is_aggregator"])
	assert.Equal(t, "n/a", response["status"])
	assert.Equal(t, "n/a", response["connection_status"])
	assert.Contains(t, response["message"], "not an aggregator")
}

func TestNonAggregatorHandleDAVisualizationHTML(t *testing.T) {
	da := &mocks.MockDA{}
	logger := zerolog.New(nil)

	// Create a non-aggregator server
	server := NewDAVisualizationServer(da, logger, false)

	req, err := http.NewRequest("GET", "/da", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	server.handleDAVisualizationHTML(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "text/html", rr.Header().Get("Content-Type"))

	body := rr.Body.String()
	assert.Contains(t, body, "DA Layer Visualization")
	assert.Contains(t, body, "Non-aggregator")
	assert.Contains(t, body, "does not submit data to the DA layer")

	// Should not show API endpoints for non-aggregator
	assert.NotContains(t, body, "Available API Endpoints")

	// Should not show recent submissions table
	assert.NotContains(t, body, "<table>")
	assert.NotContains(t, body, "<th>Timestamp</th>")
}

func TestAggregatorWithNoSubmissionsHTML(t *testing.T) {
	da := &mocks.MockDA{}
	logger := zerolog.New(nil)

	// Create an aggregator server but don't add any submissions
	server := NewDAVisualizationServer(da, logger, true)

	req, err := http.NewRequest("GET", "/da", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	server.handleDAVisualizationHTML(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "text/html", rr.Header().Get("Content-Type"))

	body := rr.Body.String()
	assert.Contains(t, body, "DA Layer Visualization")

	// Should show API endpoints for aggregator
	assert.Contains(t, body, "Available API Endpoints")

	// Should show message about no submissions yet
	assert.Contains(t, body, "No submissions recorded yet")
	assert.Contains(t, body, "This aggregator node has not submitted any data")

	// Should not show non-aggregator message
	assert.NotContains(t, body, "Non-aggregator")
	assert.NotContains(t, strings.ToLower(body), "non-aggregator nodes do not submit")
}
