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

	// Bring up reth + local DA on the same docker network as Jaeger so reth can export traces.
	env := setupCommonEVMEnv(t, sut, dcli, netID,
		WithRethOpts(func(b *reth.NodeBuilder) {
			b.WithEnv(
				"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT="+jg.Internal.IngestHTTPEndpoint()+"/v1/traces",
				"OTEL_EXPORTER_OTLP_TRACES_PROTOCOL=http",
				"RUST_LOG=info",
				"OTEL_SDK_DISABLED=false",
			)
		}),
	)
	sequencerHome := filepath.Join(t.TempDir(), "sequencer")

	// ev-node runs on the host, so use Jaeger's external OTLP/HTTP endpoint.
	otlpHTTP := jg.External.IngestHTTPEndpoint()

	// Start sequencer with tracing to Jaeger collector.
	setupSequencerNode(t, sut, sequencerHome, env.SequencerJWT, env.GenesisHash, env.Endpoints,
		"--evnode.instrumentation.tracing=true",
		"--evnode.instrumentation.tracing_endpoint", otlpHTTP,
		"--evnode.instrumentation.tracing_sample_rate", "1.0",
		"--evnode.instrumentation.tracing_service_name", "ev-node-smoke",
	)
	t.Log("Sequencer node is up")

	// Start Spamoor within the same Docker network, targeting reth internal RPC.
	ni, err := env.RethNode.GetNetworkInfo(ctx)
	require.NoError(t, err, "failed to get network info")

	internalRPC := "http://" + ni.Internal.RPCAddress()

	spBuilder := spamoor.NewNodeBuilder(t.Name()).
		WithDockerClient(env.RethNode.DockerClient).
		WithDockerNetworkID(env.RethNode.NetworkID).
		WithLogger(env.RethNode.Logger).
		WithRPCHosts(internalRPC).
		WithPrivateKey(TestPrivateKey)

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

	// allow spamoor enough time to generate transaction throughput
	// so that the expected tracing spans appear in Jaeger.
	time.Sleep(60 * time.Second)

	// Fetch parsed metrics and print a concise summary.
	metrics, err := api.GetMetrics()
	require.NoError(t, err, "failed to get metrics")
	sent := sumCounter(metrics["spamoor_transactions_sent_total"])
	fail := sumCounter(metrics["spamoor_transactions_failed_total"])

	// Verify Jaeger received traces from ev-node.
	// Service name is set above via --evnode.instrumentation.tracing_service_name "ev-node-smoke".
	traceCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	ok, err := jg.External.WaitForTraces(traceCtx, "ev-node-smoke", 1, 2*time.Second)
	require.NoError(t, err, "error while waiting for Jaeger traces; UI: %s", jg.External.QueryURL())
	require.True(t, ok, "expected at least one trace in Jaeger; UI: %s", jg.External.QueryURL())

	// Also wait for traces from ev-reth and print a small sample.
	ok, err = jg.External.WaitForTraces(traceCtx, "ev-reth", 1, 2*time.Second)
	require.NoError(t, err, "error while waiting for ev-reth traces; UI: %s", jg.External.QueryURL())
	require.True(t, ok, "expected at least one trace from ev-reth; UI: %s", jg.External.QueryURL())

	// fetch traces and print reports for both services.
	// use a large limit to fetch all traces from the test run.
	evNodeTraces, err := jg.External.Traces(traceCtx, "ev-node-smoke", 10000)
	require.NoError(t, err, "failed to fetch ev-node-smoke traces from Jaeger")
	evNodeSpans := extractSpansFromTraces(evNodeTraces)
	printTraceReport(t, "ev-node-smoke", toTraceSpans(evNodeSpans))

	evRethTraces, err := jg.External.Traces(traceCtx, "ev-reth", 10000)
	require.NoError(t, err, "failed to fetch ev-reth traces from Jaeger")
	evRethSpans := extractSpansFromTraces(evRethTraces)
	printTraceReport(t, "ev-reth", toTraceSpans(evRethSpans))

	// assert expected ev-node span names are present.
	// these spans reliably appear during block production with transactions flowing.
	expectedSpans := []string{
		"BlockExecutor.ProduceBlock",
		"BlockExecutor.ApplyBlock",
		"BlockExecutor.CreateBlock",
		"BlockExecutor.RetrieveBatch",
		"Executor.ExecuteTxs",
		"Executor.SetFinal",
		"Engine.ForkchoiceUpdated",
		"Engine.NewPayload",
		"Engine.GetPayload",
		"Eth.GetBlockByNumber",
		"Sequencer.GetNextBatch",
		"DASubmitter.SubmitHeaders",
		"DASubmitter.SubmitData",
		"DA.Submit",
	}
	opNames := make(map[string]struct{}, len(evNodeSpans))
	for _, s := range evNodeSpans {
		opNames[s.operationName] = struct{}{}
	}
	for _, name := range expectedSpans {
		require.Contains(t, opNames, name, "expected span %q not found in ev-node-smoke traces", name)
	}

	// ev-reth span names are internal to the Rust OTLP exporter and may change
	// across versions, so we only assert that spans were collected at all.
	// TODO: check for more specific spans once implemented.
	require.NotEmpty(t, evRethSpans, "expected at least one span from ev-reth")

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

// jaegerSpan holds the fields we extract from Jaeger's untyped JSON response.
type jaegerSpan struct {
	operationName string
	duration      float64 // microseconds
}

func (j jaegerSpan) SpanName() string            { return j.operationName }
func (j jaegerSpan) SpanDuration() time.Duration { return time.Duration(j.duration) * time.Microsecond }

// extractSpansFromTraces walks Jaeger's []any response and pulls out span operation names and durations.
func extractSpansFromTraces(traces []any) []jaegerSpan {
	var out []jaegerSpan
	for _, t := range traces {
		traceMap, ok := t.(map[string]any)
		if !ok {
			continue
		}
		spans, ok := traceMap["spans"].([]any)
		if !ok {
			continue
		}
		for _, s := range spans {
			spanMap, ok := s.(map[string]any)
			if !ok {
				continue
			}
			name, _ := spanMap["operationName"].(string)
			dur, _ := spanMap["duration"].(float64)
			if name != "" {
				out = append(out, jaegerSpan{operationName: name, duration: dur})
			}
		}
	}
	return out
}

func toTraceSpans(spans []jaegerSpan) []traceSpan {
	out := make([]traceSpan, len(spans))
	for i, s := range spans {
		out[i] = s
	}
	return out
}
