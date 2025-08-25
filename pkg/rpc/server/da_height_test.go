package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	coreda "github.com/evstack/ev-node/core/da"
	"github.com/rs/zerolog"
)

func TestDAHeightEndpoint(t *testing.T) {
	// Create a DummyDA with a known height
	dummyDA := coreda.NewDummyDA(1024, 0, 0, 100*time.Millisecond)
	dummyDA.StartHeightTicker()
	defer dummyDA.StopHeightTicker()

	// Wait for height to increment
	time.Sleep(200 * time.Millisecond)

	// Create DA visualization server
	logger := zerolog.Nop()
	server := NewDAVisualizationServer(dummyDA, logger, true) // isAggregator = true

	// Set up HTTP test server
	mux := http.NewServeMux()
	mux.HandleFunc("/da/height", server.handleDAHeight)

	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	// Test the endpoint
	resp, err := http.Get(testServer.URL + "/da/height")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Parse the response
	var heightResp struct {
		Height    uint64    `json:"height"`
		Timestamp time.Time `json:"timestamp"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&heightResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify we got a height greater than 0
	if heightResp.Height == 0 {
		t.Errorf("Expected height > 0, got %d", heightResp.Height)
	}
}

func TestDAHeightEndpoint_NonAggregator(t *testing.T) {
	// Create a DummyDA  
	dummyDA := coreda.NewDummyDA(1024, 0, 0, 100*time.Millisecond)

	// Create DA visualization server for non-aggregator
	logger := zerolog.Nop()
	server := NewDAVisualizationServer(dummyDA, logger, false) // isAggregator = false

	// Set up HTTP test server
	mux := http.NewServeMux()
	mux.HandleFunc("/da/height", server.handleDAHeight)

	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	// Test the endpoint - should return error for non-aggregator
	resp, err := http.Get(testServer.URL + "/da/height")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("Expected status %d for non-aggregator, got %d", http.StatusServiceUnavailable, resp.StatusCode)
	}
}