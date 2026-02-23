//go:build evm

package e2e

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tastoradocker "github.com/celestiaorg/tastora/framework/docker"
	reth "github.com/celestiaorg/tastora/framework/docker/evstack/reth"
	spamoor "github.com/celestiaorg/tastora/framework/docker/evstack/spamoor"
	jaeger "github.com/celestiaorg/tastora/framework/docker/jaeger"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestSpamoorSmoke spins up reth + sequencer and a Spamoor node, starts a few
// basic spammers, waits briefly, then prints a concise metrics summary.
func TestSpamoorSmoke(t *testing.T) {
	t.Parallel()

	sut := NewSystemUnderTest(t)
	// Prepare a shared docker client and network for Jaeger and reth.
	ctx := t.Context()
	dcli, netID := tastoradocker.Setup(t)
	jcfg := jaeger.Config{Logger: zaptest.NewLogger(t), DockerClient: dcli, DockerNetworkID: netID}
	jg, err := jaeger.New(ctx, jcfg, t.Name(), 0)
	require.NoError(t, err, "failed to create jaeger node")
	t.Cleanup(func() { _ = jg.Remove(t.Context()) })
	require.NoError(t, jg.Start(ctx), "failed to start jaeger node")

	// Bring up reth + local DA and start sequencer with default settings.
	seqJWT, _, genesisHash, endpoints, rethNode := setupCommonEVMTest(t, sut, false,
		func(b *reth.NodeBuilder) {
			b.WithDockerClient(dcli).
				WithDockerNetworkID(netID).
				WithEnv(
					"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT="+jg.IngestHTTPEndpoint()+"/v1/traces",
					"OTEL_EXPORTER_OTLP_TRACES_PROTOCOL=http",
					"RUST_LOG=info",
					"OTEL_SDK_DISABLED=false",
				)
		},
	)
	sequencerHome := filepath.Join(t.TempDir(), "sequencer")

	// ev-node runs on the host, so use Jaeger's host-mapped OTLP/HTTP port (external address).
	jinfo, err := jg.GetNetworkInfo(ctx)
	require.NoError(t, err, "failed to get jaeger network info")
	otlpHTTP := fmt.Sprintf("http://127.0.0.1:%s", jinfo.External.Ports.HTTP)

	// Configure ev-reth to export traces to Jaeger (Rust OTLP exporter expects explicit /v1/traces path).
	//evmtest.SetExtraRethEnvForTest(t.Name(),
	//	"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT="+jg.IngestHTTPEndpoint()+"/v1/traces",
	//	"OTEL_EXPORTER_OTLP_TRACES_PROTOCOL=http",
	//	"RUST_LOG=info",
	//	"OTEL_SDK_DISABLED=false",
	//)

	// Start sequencer with tracing to Jaeger collector.
	setupSequencerNode(t, sut, sequencerHome, seqJWT, genesisHash, endpoints,
		"--evnode.instrumentation.tracing=true",
		"--evnode.instrumentation.tracing_endpoint", otlpHTTP,
		"--evnode.instrumentation.tracing_sample_rate", "1.0",
		"--evnode.instrumentation.tracing_service_name", "ev-node-smoke",
	)
	t.Log("Sequencer node is up")

	// Start Spamoor within the same Docker network, targeting reth internal RPC.
	ni, err := rethNode.GetNetworkInfo(context.Background())
	require.NoError(t, err, "failed to get network info")

	internalRPC := "http://" + ni.Internal.RPCAddress()
	// Preferred typed clients from tastora's reth node helpers
	ethCli, err := rethNode.GetEthClient(ctx)
	require.NoError(t, err, "failed to get ethclient")
	rpcCli, err := rethNode.GetRPCClient(ctx)
	require.NoError(t, err, "failed to get rpc client")

	spBuilder := spamoor.NewNodeBuilder(t.Name()).
		WithDockerClient(rethNode.DockerClient).
		WithDockerNetworkID(rethNode.NetworkID).
		WithLogger(rethNode.Logger).
		WithRPCHosts(internalRPC).
		WithPrivateKey(TestPrivateKey)

	ctx = t.Context()
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

	// Probe ev-reth via JSON-RPC as proxy metrics: head height should advance; peer count should be >= 0.
	h1, err := ethCli.BlockNumber(ctx)
	require.NoError(t, err, "failed to query initial block number")
	time.Sleep(5 * time.Second)
	h2, err := ethCli.BlockNumber(ctx)
	require.NoError(t, err, "failed to query subsequent block number")
	var peerCountHex string
	require.NoError(t, rpcCli.CallContext(ctx, &peerCountHex, "net_peerCount"))
	t.Logf("reth head: %d -> %d, net_peerCount=%s", h1, h2, strings.TrimSpace(peerCountHex))

	// Verify Jaeger received traces from ev-node.
	// Service name is set above via --evnode.instrumentation.tracing_service_name "ev-node-smoke".
	traceCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	ok, err := jg.External.WaitForTraces(traceCtx, "ev-node-smoke", 1, 2*time.Second)
	require.NoError(t, err, "error while waiting for Jaeger traces; UI: %s", jg.QueryHostURL())
	require.True(t, ok, "expected at least one trace in Jaeger; UI: %s", jg.QueryHostURL())

	// Also wait for traces from ev-reth and print a small sample.
	ok, err = jg.External.WaitForTraces(traceCtx, "ev-reth", 1, 2*time.Second)
	require.NoError(t, err, "error while waiting for ev-reth traces; UI: %s", jg.External.URL())
	require.True(t, ok, "expected at least one trace from ev-reth; UI: %s", jg.External.URL())
	if traces, err := jg.External.Traces(traceCtx, "ev-reth", 3); err == nil && len(traces) > 0 {
		t.Logf("sample ev-reth traces: %v", traces[0])
	}

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
