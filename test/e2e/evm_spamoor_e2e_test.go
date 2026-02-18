//go:build evm

package e2e

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	spamoor "github.com/celestiaorg/tastora/framework/docker/evstack/spamoor"
)

// TestSpamoorBasicScenario starts a sequencer and runs a basic spamoor scenario
// against the sequencer's ETH RPC. It also attempts to scrape metrics from a
// configured metrics endpoint if supported by the spamoor binary.
// Optional: metrics via spamoor-daemon. Best-effort and skipped if daemon not available.
func TestSpamoorMetricsViaDaemon(t *testing.T) {
	t.Parallel()

	sut := NewSystemUnderTest(t)

	// Common EVM env (Reth + local DA ports/JWT/genesis)
	seqJWT, _, genesisHash, endpoints, rethNode := setupCommonEVMTest(t, sut, false)

	// In-process OTLP collector for ev-node traces
	collector := newOTLPCollector(t)
	t.Cleanup(func() { collector.close() })

	// Start sequencer using shared helper and only tracing extra args
	sequencerHome := filepath.Join(t.TempDir(), "sequencer")
	setupSequencerNode(t, sut, sequencerHome, seqJWT, genesisHash, endpoints,
		"--evnode.instrumentation.tracing=true",
		"--evnode.instrumentation.tracing_endpoint", collector.endpoint(),
		"--evnode.instrumentation.tracing_sample_rate", "1.0",
		"--evnode.instrumentation.tracing_service_name", "ev-node-e2e",
	)
	t.Log("Sequencer node is up")

	// Run spamoor-daemon via tastora container node using the maintained Docker image
	// so that CI only needs Docker, not local binaries.
	// Build spamoor container node in the SAME docker network
	// It can reach Reth via internal IP:port
	// Use reth INTERNAL ETH RPC so the Spamoor container talks directly over Docker network
	ni, err := rethNode.GetNetworkInfo(context.Background())
	if err != nil {
		t.Fatalf("failed to get reth network info for Spamoor RPC: %v", err)
	}
	
	internalRPC := "http://" + ni.Internal.RPCAddress()
	spBuilder := spamoor.NewNodeBuilder(t.Name()).
		WithDockerClient(rethNode.DockerClient).
		WithDockerNetworkID(rethNode.NetworkID).
		WithLogger(rethNode.Logger).
		WithRPCHosts(internalRPC).
		WithPrivateKey(TestPrivateKey).
		WithHostPort(0)
	ctx := context.Background()
	spNode, err := spBuilder.Build(ctx)
	if err != nil {
		t.Skipf("cannot build spamoor container: %v", err)
		return
	}
	t.Cleanup(func() { _ = spNode.Remove(context.Background()) })

	if err := spNode.Start(ctx); err != nil {
		t.Skipf("cannot start spamoor container: %v", err)
		return
	}

	// Discover host-mapped ports from Docker
	spInfo, err := spNode.GetNetworkInfo(ctx)
	if err != nil {
		t.Fatalf("failed to get spamoor network info: %v", err)
	}
	apiAddr := "http://127.0.0.1:" + spInfo.External.Ports.HTTP
	metricsAddr := "http://127.0.0.1:" + spInfo.External.Ports.Metrics
	// Wait for the daemon HTTP server to accept connections (use HTTP port)
	requireHTTP(t, apiAddr+"/api/spammers", 30*time.Second)
	api := NewSpamoorAPI(apiAddr)
	// Config YAML for eoatx (no 'count' — run for a short window)
	// Increase throughput and wallet pool to ensure visible on-chain activity for tx metrics
	cfg := strings.Join([]string{
		"throughput: 60",
		"max_pending: 1000",
		"max_wallets: 200",
		"amount: 100",
		"random_amount: true",
		"random_target: true",
		"refill_amount: 1000000000000000000", // 1 ETH
		"refill_balance: 500000000000000000", // 0.5 ETH
		"refill_interval: 600",
	}, "\n")
	spammerID, err := api.CreateSpammer("e2e-eoatx", "eoatx", cfg, true)
	if err != nil {
		t.Fatalf("failed to create/start spammer: %v", err)
	}
	t.Cleanup(func() { _ = api.DeleteSpammer(spammerID) })

	// Wait a bit for the spammer to be running
	runUntil := time.Now().Add(10 * time.Second)
	for time.Now().Before(runUntil) {
		s, _ := api.GetSpammer(spammerID)
		if s != nil && s.Status == 1 { // running
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	// Allow additional time to generate activity for metrics
	time.Sleep(20 * time.Second)

	// Dump metrics while/after running from the dedicated metrics port
	metricsAPI := NewSpamoorAPI(metricsAddr)
	// Poll metrics until spamoor-specific metrics appear or timeout
	var m string
	metricsDeadline := time.Now().Add(30 * time.Second)
	for {
		m, err = metricsAPI.GetMetrics()
		if err == nil && strings.Contains(m, "spamoor") {
			break
		}
		if time.Now().After(metricsDeadline) {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if err != nil {
		t.Logf("metrics not available: %v", err)
		return
	}
	if len(strings.TrimSpace(m)) == 0 {
		t.Log("empty metrics from daemon")
		return
	}
	// Print a short sample of overall metrics
	t.Logf("daemon metrics sample:\n%s", firstLines(m, 40))
	// Additionally, extract spamoor-specific metrics if present
	var spamoorLines []string
	for _, line := range strings.Split(m, "\n") {
		if strings.Contains(line, "spamoor") {
			spamoorLines = append(spamoorLines, line)
			if len(spamoorLines) >= 100 {
				break
			}
		}
	}
	if len(spamoorLines) > 0 {
		t.Logf("spamoor metrics (subset):\n%s", strings.Join(spamoorLines, "\n"))
	} else {
		t.Fatalf("no spamoor-prefixed metrics found; increase runtime/throughput or verify tx metrics are enabled")
	}
	// Done
}

// firstLines returns up to n lines of s.
func firstLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}

// requireHTTP polls a URL until it returns a 200-range response or the timeout expires.
func requireHTTP(t *testing.T, url string, timeout time.Duration) {
	t.Helper()
	client := &http.Client{Timeout: 200 * time.Millisecond}
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return
			}
			lastErr = fmt.Errorf("status %d", resp.StatusCode)
		} else {
			lastErr = err
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("daemon not ready at %s: %v", url, lastErr)
}

// requireJSONRPC checks that the ETH RPC responds to a net_version request.
func requireJSONRPC(t *testing.T, url string, timeout time.Duration) {
	t.Helper()
	client := &http.Client{Timeout: 400 * time.Millisecond}
	deadline := time.Now().Add(timeout)
	payload := []byte(`{"jsonrpc":"2.0","method":"net_version","params":[],"id":1}`)
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := client.Post(url, "application/json", bytes.NewReader(payload))
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return
			}
			lastErr = fmt.Errorf("status %d", resp.StatusCode)
		} else {
			lastErr = err
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("reth rpc not ready at %s: %v", url, lastErr)
}
