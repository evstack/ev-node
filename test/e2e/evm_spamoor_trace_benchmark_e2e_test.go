//go:build evm

package e2e

import (
    "context"
    "path/filepath"
    "regexp"
    "strconv"
    "strings"
    "testing"
    "time"

    spamoor "github.com/celestiaorg/tastora/framework/docker/evstack/spamoor"
)

// TestSpamoorTraceBenchmark spins up reth (Docker via Tastora), local-da, ev-node (host binary)
// with OTEL tracing exported to an in-process OTLP/HTTP collector, and a Spamoor container
// driving EOA transfers directly to reth's internal ETH RPC. It runs a short load, then
// emits a concise metrics/trace summary and fails if activity or spans are missing.
func TestSpamoorTraceBenchmark(t *testing.T) {
    t.Parallel()

    sut := NewSystemUnderTest(t)

    // Common EVM env: reth + local DA. Return JWT, genesis, endpoints, and reth node handle.
    seqJWT, _, genesisHash, endpoints, rethNode := setupCommonEVMTest(t, sut, false)

    // In-process OTLP collector to receive ev-node spans.
    collector := newOTLPCollector(t)
    t.Cleanup(func() { collector.close() })

    // Start sequencer (ev-node) with tracing enabled to our collector.
    sequencerHome := filepath.Join(t.TempDir(), "sequencer")
    setupSequencerNode(t, sut, sequencerHome, seqJWT, genesisHash, endpoints,
        "--evnode.instrumentation.tracing=true",
        "--evnode.instrumentation.tracing_endpoint", collector.endpoint(),
        "--evnode.instrumentation.tracing_sample_rate", "1.0",
        "--evnode.instrumentation.tracing_service_name", "ev-node-e2e",
    )
    t.Log("Sequencer node is up")

    // Launch Spamoor container on the same Docker network; point it at reth INTERNAL ETH RPC.
    ni, err := rethNode.GetNetworkInfo(context.Background())
    if err != nil {
        t.Fatalf("failed to get reth network info: %v", err)
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

    // Discover host-mapped ports and wait for daemon readiness.
    spInfo, err := spNode.GetNetworkInfo(ctx)
    if err != nil {
        t.Fatalf("failed to get spamoor network info: %v", err)
    }
    apiAddr := "http://127.0.0.1:" + spInfo.External.Ports.HTTP
    metricsAddr := "http://127.0.0.1:" + spInfo.External.Ports.Metrics
    requireHTTP(t, apiAddr+"/api/spammers", 30*time.Second)
    api := NewSpamoorAPI(apiAddr)

    // Configure spam: decent throughput to populate blocks and metrics.
    // Run as a daemon-managed spammer that starts immediately.
    cfg := strings.Join([]string{
        "throughput: 80",
        "total_count: 3000",
        "max_pending: 4000",
        "max_wallets: 300",
        "amount: 100",
        "random_amount: true",
        "random_target: true",
        "refill_amount: 1000000000000000000", // 1 ETH
        "refill_balance: 500000000000000000", // 0.5 ETH
        "refill_interval: 600",
    }, "\n")
    spammerID, err := api.CreateSpammer("benchmark-eoatx", "eoatx", cfg, true)
    if err != nil {
        t.Fatalf("failed to create/start spammer: %v", err)
    }
    t.Cleanup(func() { _ = api.DeleteSpammer(spammerID) })

    // Wait for running status briefly, then let it run to generate load.
    runUntil := time.Now().Add(10 * time.Second)
    for time.Now().Before(runUntil) {
        s, _ := api.GetSpammer(spammerID)
        if s != nil && s.Status == 1 {
            break
        }
        time.Sleep(200 * time.Millisecond)
    }

    // Let the load run for ~30–45s to stabilize block production and metrics,
    // but also poll metrics so we can break early once activity is observed.
    metricsAPI := NewSpamoorAPI(metricsAddr)
    var metricsText string
    deadline := time.Now().Add(45 * time.Second)
    for {
        if time.Now().After(deadline) {
            break
        }
        m, err := metricsAPI.GetMetrics()
        if err == nil && strings.Contains(m, "spamoor_transactions_sent_total") {
            metricsText = m
            if scrapeCounter(m, `^spamoor_transactions_sent_total\{.*\}\s+(\\d+(?:\\.\\d+)?)`) > 0 {
                break
            }
        }
        time.Sleep(500 * time.Millisecond)
    }

    // If we didn't capture metrics during the loop, fetch once now.
    if metricsText == "" {
        var err error
        metricsText, err = metricsAPI.GetMetrics()
        if err != nil {
            t.Fatalf("failed to fetch spamoor metrics: %v", err)
        }
    }

    sentTotal := scrapeCounter(metricsText, `^spamoor_transactions_sent_total\{.*\}\s+(\d+(?:\.\d+)?)`)
    failures := scrapeCounter(metricsText, `^spamoor_transactions_failed_total\{.*\}\s+(\d+(?:\.\d+)?)`)
    pending := scrapeGauge(metricsText, `^spamoor_pending_transactions\{.*\}\s+(\d+(?:\.\d+)?)`)
    blockGas := scrapeCounter(metricsText, `^spamoor_block_gas_usage\{.*\}\s+(\d+(?:\.\d+)?)`)

    t.Logf("Spamoor summary: sent=%.0f failed=%.0f pending=%.0f block_gas=%.0f", sentTotal, failures, pending, blockGas)

    if sentTotal < 5 {
        t.Fatalf("insufficient on-chain activity: spamoor_transactions_sent_total=%.0f", sentTotal)
    }

    // Give ev-node a moment to flush pending span batches, then aggregate spans.
    time.Sleep(2 * time.Second)
    spans := collector.getSpans()
    if len(spans) == 0 {
        t.Fatalf("no ev-node spans recorded by OTLP collector")
    }
    printCollectedTraceReport(t, collector)

    // Optional: sanity-check reth RPC responsiveness (debug tracing optional and best-effort).
    // If debug API is enabled in the reth image, a recent tx could be traced here.
    // We only check that RPC endpoint responds to a simple request to catch regressions.
    requireJSONRPC(t, endpoints.GetSequencerEthURL(), 10*time.Second)
}

// --- small helpers to scrape Prometheus text ---

func scrapeCounter(metrics, pattern string) float64 {
    return scrapeFloat(metrics, pattern)
}

func scrapeGauge(metrics, pattern string) float64 {
    return scrapeFloat(metrics, pattern)
}

func scrapeFloat(metrics, pattern string) float64 {
    re := regexp.MustCompile(pattern)
    lines := strings.Split(metrics, "\n")
    for _, line := range lines {
        if m := re.FindStringSubmatch(line); len(m) == 2 {
            if v, err := strconv.ParseFloat(m[1], 64); err == nil {
                return v
            }
        }
    }
    return 0
}

    
