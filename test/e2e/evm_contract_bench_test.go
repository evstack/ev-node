//go:build evm

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"path/filepath"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	collpb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
)

// BenchmarkEvmContractRoundtrip measures the store → retrieve roundtrip latency
// against a real reth node with a pre-deployed contract.
//
// The node is started with OpenTelemetry tracing enabled, exporting to an
// in-process OTLP/HTTP receiver. After the timed loop, the collected spans are
// aggregated into a hierarchical timing report showing where time is spent
// inside ev-node (Engine API calls, executor, sequencer, etc).
//
// Run with (after building local-da and evm binaries):
//
//	PATH="/path/to/binaries:$PATH" go test -tags evm \
//	  -bench BenchmarkEvmContractRoundtrip -benchmem -benchtime=5x \
//	  -run='^$' -timeout=10m -v --evm-binary=/path/to/evm .
func BenchmarkEvmContractRoundtrip(b *testing.B) {
	workDir := b.TempDir()
	sequencerHome := filepath.Join(workDir, "evm-bench-sequencer")
	const blockTime = 5 * time.Millisecond

	// Start an in-process OTLP/HTTP receiver to collect traces from ev-node.
	collector := newOTLPCollector(b)
	defer collector.close() // nolint: errcheck // test only

	// Start sequencer with tracing enabled, exporting to our in-process collector.
	client, _, cleanup := setupTestSequencer(b, sequencerHome,
		"--evnode.instrumentation.tracing=true",
		"--evnode.instrumentation.tracing_endpoint", collector.endpoint(),
		"--evnode.instrumentation.tracing_sample_rate", "1.0",
		"--evnode.instrumentation.tracing_service_name", "ev-node-bench",
		"--evnode.node.block_time="+blockTime.String(),
	)
	defer cleanup()

	ctx := b.Context()
	privateKey, err := crypto.HexToECDSA(TestPrivateKey)
	require.NoError(b, err)
	chainID, ok := new(big.Int).SetString(DefaultChainID, 10)
	require.True(b, ok)
	signer := types.NewEIP155Signer(chainID)

	// Deploy contract once during setup.
	contractAddr, nonce := deployContract(b, ctx, client, StorageContractBytecode, 0, privateKey, chainID)

	// Pre-build signed store(42) transactions for all iterations.
	storeData, err := hexutil.Decode("0x000000000000000000000000000000000000000000000000000000000000002a")
	require.NoError(b, err)

	const maxIter = 1024
	signedTxs := make([]*types.Transaction, maxIter)
	for i := range maxIter {
		tx := types.NewTx(&types.LegacyTx{
			Nonce:    nonce + uint64(i),
			To:       &contractAddr,
			Value:    big.NewInt(0),
			Gas:      500000,
			GasPrice: big.NewInt(30000000000),
			Data:     storeData,
		})
		signedTxs[i], err = types.SignTx(tx, signer, privateKey)
		require.NoError(b, err)
	}

	expected := common.HexToHash("0x000000000000000000000000000000000000000000000000000000000000002a").Bytes()
	callMsg := ethereum.CallMsg{To: &contractAddr, Data: []byte{}}

	b.ResetTimer()
	b.ReportAllocs()

	var i int
	for b.Loop() {
		require.Less(b, i, maxIter, "increase maxIter for longer benchmark runs")

		// 1. Submit pre-signed store(42) transaction.
		err = client.SendTransaction(ctx, signedTxs[i])
		require.NoError(b, err)

		// 2. Wait for inclusion with fast polling to reduce variance while avoiding RPC overload.
		require.Eventually(b, func() bool {
			receipt, err := client.TransactionReceipt(ctx, signedTxs[i].Hash())
			return err == nil && receipt != nil
		}, 2*time.Second, blockTime/2, "transaction %s not included", signedTxs[i].Hash().Hex())

		// 3. Retrieve and verify.
		result, err := client.CallContract(ctx, callMsg, nil)
		require.NoError(b, err)
		require.Equal(b, expected, result, "retrieve() should return 42")

		i++
	}

	b.StopTimer()

	// Give the node a moment to flush pending span batches.
	time.Sleep(2 * time.Second)

	// Print the trace breakdown from the collected spans.
	printCollectedTraceReport(b, collector)
}

// --- In-process OTLP/HTTP Collector ---

// otlpCollector is a lightweight OTLP/HTTP receiver that collects trace spans
// in memory. It serves the /v1/traces endpoint that the node's OTLP exporter
// posts protobuf-encoded ExportTraceServiceRequest messages to.
type otlpCollector struct {
	mu     sync.Mutex
	spans  []*tracepb.Span
	server *http.Server
	addr   string
}

func newOTLPCollector(t testing.TB) *otlpCollector {
	t.Helper()

	c := &otlpCollector{}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/traces", c.handleTraces)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	c.addr = listener.Addr().String()

	c.server = &http.Server{Handler: mux}
	go func() { _ = c.server.Serve(listener) }()

	t.Logf("OTLP collector listening on %s", c.addr)
	return c
}

func (c *otlpCollector) endpoint() string {
	return "http://" + c.addr
}

func (c *otlpCollector) close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return c.server.Shutdown(ctx)
}

func (c *otlpCollector) handleTraces(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Try protobuf first (default for otlptracehttp).
	var req collpb.ExportTraceServiceRequest
	if err := proto.Unmarshal(body, &req); err != nil {
		// Fallback: try JSON (some configurations use JSON encoding).
		if jsonErr := json.Unmarshal(body, &req); jsonErr != nil {
			http.Error(w, fmt.Sprintf("proto: %v; json: %v", err, jsonErr), http.StatusBadRequest)
			return
		}
	}

	c.mu.Lock()
	for _, rs := range req.GetResourceSpans() {
		for _, ss := range rs.GetScopeSpans() {
			c.spans = append(c.spans, ss.GetSpans()...)
		}
	}
	c.mu.Unlock()

	// Respond with an empty ExportTraceServiceResponse (protobuf).
	resp := &collpb.ExportTraceServiceResponse{}
	out, _ := proto.Marshal(resp)
	w.Header().Set("Content-Type", "application/x-protobuf")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(out)
}

func (c *otlpCollector) getSpans() []*tracepb.Span {
	c.mu.Lock()
	defer c.mu.Unlock()
	cp := make([]*tracepb.Span, len(c.spans))
	copy(cp, c.spans)
	return cp
}

// printCollectedTraceReport aggregates collected spans by operation name and
// prints a timing breakdown.
func printCollectedTraceReport(b testing.TB, collector *otlpCollector) {
	b.Helper()

	spans := collector.getSpans()
	if len(spans) == 0 {
		b.Logf("WARNING: no spans collected from ev-node")
		return
	}

	type stats struct {
		count int
		total time.Duration
		min   time.Duration
		max   time.Duration
	}
	m := make(map[string]*stats)

	for _, span := range spans {
		// Duration: end - start in nanoseconds.
		d := time.Duration(span.GetEndTimeUnixNano()-span.GetStartTimeUnixNano()) * time.Nanosecond
		if d <= 0 {
			continue
		}
		name := span.GetName()
		s, ok := m[name]
		if !ok {
			s = &stats{min: d, max: d}
			m[name] = s
		}
		s.count++
		s.total += d
		if d < s.min {
			s.min = d
		}
		if d > s.max {
			s.max = d
		}
	}

	// Sort by total time descending.
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		return m[names[i]].total > m[names[j]].total
	})

	// Calculate overall total for percentages.
	var overallTotal time.Duration
	for _, s := range m {
		overallTotal += s.total
	}

	b.Logf("\n--- ev-node Trace Breakdown (%d spans collected) ---", len(spans))
	b.Logf("%-40s %6s %12s %12s %12s %7s", "OPERATION", "COUNT", "AVG", "MIN", "MAX", "% TOTAL")
	for _, name := range names {
		s := m[name]
		avg := s.total / time.Duration(s.count)
		pct := float64(s.total) / float64(overallTotal) * 100
		b.Logf("%-40s %6d %12s %12s %12s %6.1f%%", name, s.count, avg, s.min, s.max, pct)
	}

	b.Logf("\n--- Time Distribution ---")
	for _, name := range names {
		s := m[name]
		pct := float64(s.total) / float64(overallTotal) * 100
		bar := ""
		for range int(pct / 2) {
			bar += "█"
		}
		b.Logf("%-40s %5.1f%% %s", name, pct, bar)
	}
}
