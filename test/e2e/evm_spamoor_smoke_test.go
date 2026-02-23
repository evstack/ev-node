//go:build evm

package e2e

import (
    "context"
    "fmt"
    "net/http"
    "path/filepath"
    "testing"
    "time"

    tastoradocker "github.com/celestiaorg/tastora/framework/docker"
    spamoor "github.com/celestiaorg/tastora/framework/docker/evstack/spamoor"
    dto "github.com/prometheus/client_model/go"
    "github.com/stretchr/testify/require"
)

// TestSpamoorSmoke spins up reth + sequencer and a Spamoor node, starts a few
// basic spammers, waits briefly, then prints a concise metrics summary.
func TestSpamoorSmoke(t *testing.T) {
	t.Parallel()

	sut := NewSystemUnderTest(t)
    // Bring up reth + local DA and start sequencer with default settings.
    dcli, netID := tastoradocker.Setup(t)
    env := setupCommonEVMEnv(t, sut, dcli, netID)
    sequencerHome := filepath.Join(t.TempDir(), "sequencer")

	// In-process OTLP/HTTP collector to capture ev-node spans.
	collector := newOTLPCollector(t)
	t.Cleanup(func() {
		_ = collector.close()
	})

	// Start sequencer with tracing to our collector.
    setupSequencerNode(t, sut, sequencerHome, env.SequencerJWT, env.GenesisHash, env.Endpoints,
		"--evnode.instrumentation.tracing=true",
		"--evnode.instrumentation.tracing_endpoint", collector.endpoint(),
		"--evnode.instrumentation.tracing_sample_rate", "1.0",
		"--evnode.instrumentation.tracing_service_name", "ev-node-smoke",
	)
	t.Log("Sequencer node is up")

	// Start Spamoor within the same Docker network, targeting reth internal RPC.
    ni, err := env.RethNode.GetNetworkInfo(context.Background())
	require.NoError(t, err, "failed to get network info")

	internalRPC := "http://" + ni.Internal.RPCAddress()

    spBuilder := spamoor.NewNodeBuilder(t.Name()).
        WithDockerClient(env.RethNode.DockerClient).
        WithDockerNetworkID(env.RethNode.NetworkID).
        WithLogger(env.RethNode.Logger).
		WithRPCHosts(internalRPC).
		WithPrivateKey(TestPrivateKey)

	ctx := t.Context()
	spNode, err := spBuilder.Build(ctx)
	require.NoError(t, err, "failed to build sp node")

	t.Cleanup(func() { _ = spNode.Remove(t.Context()) })
	require.NoError(t, spNode.Start(ctx), "failed to start spamoor node")

	// Wait for daemon readiness.
	spInfo, err := spNode.GetNetworkInfo(ctx)
	require.NoError(t, err, "failed to get network info")

	apiAddr := "http://127.0.0.1:" + spInfo.External.Ports.HTTP
	requireHTTP(t, apiAddr+"/api/spammers", 30*time.Second)
	api := spNode.API()

	// Basic scenarios (structs that YAML-marshal into the daemon config).
	eoatx := map[string]any{
		"throughput":      100,
		"total_count":     3000,
		"max_pending":     4000,
		"max_wallets":     300,
		"amount":          100,
		"random_amount":   true,
		"random_target":   true,
		"base_fee":        20, // gwei
		"tip_fee":         2,  // gwei
		"refill_amount":   "1000000000000000000",
		"refill_balance":  "500000000000000000",
		"refill_interval": 600,
	}

	gasburner := map[string]any{
		"throughput":        25,
		"total_count":       2000,
		"max_pending":       8000,
		"max_wallets":       500,
		"gas_units_to_burn": 3000000,
		"base_fee":          20,
		"tip_fee":           5,
		"rebroadcast":       5,
		"refill_amount":     "5000000000000000000",
		"refill_balance":    "2000000000000000000",
		"refill_interval":   300,
	}

	var ids []int
	id, err := api.CreateSpammer("smoke-eoatx", spamoor.ScenarioEOATX, eoatx, true)
	require.NoError(t, err, "failed to create eoatx spammer")
	ids = append(ids, id)
	id, err = api.CreateSpammer("smoke-gasburner", spamoor.ScenarioGasBurnerTX, gasburner, true)
	require.NoError(t, err, "failed to create gasburner spammer")
	ids = append(ids, id)

	for _, id := range ids {
		idToDelete := id
		t.Cleanup(func() { _ = api.DeleteSpammer(idToDelete) })
	}

	// Allow additional time to accumulate activity.
	time.Sleep(60 * time.Second)

	// Fetch parsed metrics and print a concise summary.
	metrics, err := api.GetMetrics()
	require.NoError(t, err, "failed to get metrics")
	sent := sumCounter(metrics["spamoor_transactions_sent_total"])
	fail := sumCounter(metrics["spamoor_transactions_failed_total"])

	time.Sleep(2 * time.Second)
	printCollectedTraceReport(t, collector)

	require.Greater(t, sent, float64(0), "at least one transaction should have been sent")
	require.Zero(t, fail, "no transactions should have failed")
}

// --- helpers ---

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

// Metric family helpers.
func sumCounter(f *dto.MetricFamily) float64 {
	if f == nil || f.GetType() != dto.MetricType_COUNTER {
		return 0
	}
	var sum float64
	for _, m := range f.GetMetric() {
		if m.GetCounter() != nil && m.GetCounter().Value != nil {
			sum += m.GetCounter().GetValue()
		}
	}
	return sum
}
func sumGauge(f *dto.MetricFamily) float64 {
	if f == nil || f.GetType() != dto.MetricType_GAUGE {
		return 0
	}
	var sum float64
	for _, m := range f.GetMetric() {
		if m.GetGauge() != nil && m.GetGauge().Value != nil {
			sum += m.GetGauge().GetValue()
		}
	}
	return sum
}
