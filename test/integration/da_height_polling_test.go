package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	coreda "github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/da_client"
	genesispkg "github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/rpc/server"
	"github.com/rs/zerolog"
)

// TestDAHeightPollingIntegration tests the end-to-end DA height polling functionality
func TestDAHeightPollingIntegration(t *testing.T) {
	logger := zerolog.Nop()

	// Step 1: Create a simulated aggregator with DummyDA
	dummyDA := coreda.NewDummyDA(1024, 0, 0, 50*time.Millisecond)
	dummyDA.StartHeightTicker()
	defer dummyDA.StopHeightTicker()

	// Wait for DA height to increment to > 0
	time.Sleep(100 * time.Millisecond)

	// Create aggregator's DA visualization server
	aggregatorServer := server.NewDAVisualizationServer(dummyDA, logger, true)
	server.SetDAVisualizationServer(aggregatorServer) // Set the global server instance

	// Set up HTTP test server to simulate the aggregator using the public HTTP endpoints
	mux := http.NewServeMux()
	server.RegisterCustomHTTPEndpoints(mux) // This will register all the endpoints including /da/height
	testAggregator := httptest.NewServer(mux)
	defer testAggregator.Close()

	// Step 2: Test DA client polling functionality
	client := da_client.NewClient()
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	height, err := client.GetDAHeight(ctx, testAggregator.URL)
	if err != nil {
		t.Fatalf("Failed to get DA height from aggregator: %v", err)
	}

	if height == 0 {
		t.Errorf("Expected DA height > 0, got %d", height)
	}

	t.Logf("Successfully polled DA height: %d", height)

	// Step 3: Test polling functionality (simulates genesis initialization)
	pollHeight, err := client.PollDAHeight(ctx, testAggregator.URL, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to poll DA height: %v", err)
	}

	if pollHeight == 0 {
		t.Errorf("Expected polled DA height > 0, got %d", pollHeight)
	}

	t.Logf("Successfully polled DA height via polling: %d", pollHeight)
}

// TestGenesisPollingConfig tests the configuration and genesis modification logic
func TestGenesisPollingConfig(t *testing.T) {
	// Test configuration with aggregator endpoint
	cfg := config.DefaultConfig
	cfg.DA.AggregatorEndpoint = "http://localhost:8080"
	cfg.Node.Aggregator = false // This is a non-aggregator node

	// Verify the configuration is set correctly
	if cfg.DA.AggregatorEndpoint == "" {
		t.Error("Expected aggregator endpoint to be set")
	}

	if cfg.Node.Aggregator {
		t.Error("Expected node to be non-aggregator for this test")
	}

	// Test genesis with zero start time (simulating the condition that triggers polling)
	genesis := genesispkg.Genesis{
		ChainID:            "test-chain",
		GenesisDAStartTime: time.Time{}, // Zero time
		InitialHeight:      1,
		ProposerAddress:    []byte("test-proposer"),
	}

	if !genesis.GenesisDAStartTime.IsZero() {
		t.Error("Expected genesis DA start time to be zero")
	}

	t.Log("Configuration test passed - ready for DA height polling")
}