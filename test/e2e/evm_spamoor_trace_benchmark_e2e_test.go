//go:build evm

package e2e

import (
	"context"
	"math/big"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	spamoor "github.com/celestiaorg/tastora/framework/docker/evstack/spamoor"
	"github.com/ethereum/go-ethereum/ethclient"
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
    // Target: ~1 Ggas/s with 100ms blocks => ~100M gas per block.
    // Use tight block_time without lazy mode to enforce cadence.
    setupSequencerNode(t, sut, sequencerHome, seqJWT, genesisHash, endpoints,
        "--evnode.instrumentation.tracing=true",
        "--evnode.instrumentation.tracing_endpoint", collector.endpoint(),
        "--evnode.instrumentation.tracing_sample_rate", "1.0",
        "--evnode.instrumentation.tracing_service_name", "ev-node-e2e",
        "--evnode.node.block_time", "100ms",
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

	// Gasburner-only configuration below; eoatx disabled for this ceiling test.

    // Start with gasburner-only to maximize gas per block and simplify inclusion checks.
    // Aim for ~100M gas per 100ms block: set 10M gas/tx and target ~10 tx/block total across spammers.
    spammerIDs := make([]int, 0, 4)

	// Add 1-2 heavier-gas spammers using known Spamoor scenarios. We validate each config
	// before starting to avoid failures on unsupported images.
	// Prefer gasburnertx spammers to maximize gas usage per transaction.
    gasBurnerCfg := strings.Join([]string{
        "throughput: 200",                  // aggressive per-spammer rate (per-second), drives continuous backlog
        "total_count: 20000",
        "max_pending: 50000",
        "max_wallets: 1000",
        "gas_units_to_burn: 10000000",      // 10M gas/tx
        "refill_amount: 5000000000000000000",  // 5 ETH
        "refill_balance: 2000000000000000000", // 2 ETH
        "refill_interval: 300",
        "base_fee: 20",  // gwei (reduce to avoid fee-cap rejection)
        "tip_fee: 5",    // gwei
        "rebroadcast: 2",
    }, "\n")
    for i := 0; i < 14; i++ { // start with 14 spammers to create a stronger continuous backlog
        name := "benchmark-gasburner-" + strconv.Itoa(i)
        if err := api.ValidateScenarioConfig(name+"-probe", "gasburnertx", gasBurnerCfg); err != nil {
            t.Logf("gasburnertx not supported or invalid config: %v", err)
            break
        }
		id, err := api.CreateSpammer(name, "gasburnertx", gasBurnerCfg, true)
		if err != nil {
			t.Logf("failed to start gasburnertx: %v", err)
			break
		}
		spammerIDs = append(spammerIDs, id)
		idx := len(spammerIDs) - 1
		t.Cleanup(func() { _ = api.DeleteSpammer(spammerIDs[idx]) })
	}

	// Wait for any spammer to report running, then proceed.
	runUntil := time.Now().Add(15 * time.Second)
	for time.Now().Before(runUntil) {
		ok := false
		for _, id := range spammerIDs {
			s, _ := api.GetSpammer(id)
			if s != nil && s.Status == 1 {
				ok = true
				break
			}
		}
		if ok {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Let the initial load run for ~30–45s to stabilize block production and metrics,
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

	// If we didn't capture metrics during the loop, fetch once now (best-effort).
	if metricsText == "" {
		if m, err := metricsAPI.GetMetrics(); err == nil {
			metricsText = m
		} else {
			t.Logf("metrics unavailable (continuing without them): %v", err)
		}
	}

    // Try to saturate block gas: dynamically ramp up gasburner spammers until utilization approaches ~100M/block or we hit a safe cap.
    if ec, err := ethclient.Dial(endpoints.GetSequencerEthURL()); err == nil {
        defer ec.Close()
        rampDeadline := time.Now().Add(60 * time.Second)
        for len(spammerIDs) < 20 && time.Now().Before(rampDeadline) { // ramp up further if still underutilized
            avg, peak := sampleGasUtilization(t, ec, 5*time.Second)
            t.Logf("Ramp check: gas utilization avg=%.1f%% peak=%.1f%% spammers=%d", avg*100, peak*100, len(spammerIDs))
            if peak >= 0.9 { // good enough
                break
            }
            // Add another gasburner spammer to push harder.
            id, err := api.CreateSpammer("benchmark-gasburner-ramp-"+strconv.Itoa(len(spammerIDs)), "gasburnertx", gasBurnerCfg, true)
            if err != nil {
                t.Logf("failed to add ramp spammer: %v", err)
                break
            }
            spammerIDs = append(spammerIDs, id)
            idx := len(spammerIDs) - 1
            t.Cleanup(func() { _ = api.DeleteSpammer(spammerIDs[idx]) })
            // give it a moment to start before next measurement
            time.Sleep(3 * time.Second)
        }
    }

	if metricsText != "" {
		sentTotal := scrapeCounter(metricsText, `^spamoor_transactions_sent_total\{.*\}\s+(\d+(?:\.\d+)?)`)
		failures := scrapeCounter(metricsText, `^spamoor_transactions_failed_total\{.*\}\s+(\d+(?:\.\d+)?)`)
		pending := scrapeGauge(metricsText, `^spamoor_pending_transactions\{.*\}\s+(\d+(?:\.\d+)?)`)
		blockGas := scrapeCounter(metricsText, `^spamoor_block_gas_usage\{.*\}\s+(\d+(?:\.\d+)?)`)
		t.Logf("Spamoor summary: sent=%.0f failed=%.0f pending=%.0f block_gas=%.0f", sentTotal, failures, pending, blockGas)
		if sentTotal < 5 {
			t.Logf("warning: low spamoor sent count (%.0f)", sentTotal)
		}
	} else {
		t.Log("Spamoor metrics unavailable; will validate activity via block headers and spans")
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

	// Report observed block gas utilization over a short window.
	// Query latest headers to compute peak and average gas usage fraction.
	if ec, err := ethclient.Dial(endpoints.GetSequencerEthURL()); err == nil {
		defer ec.Close()
		avg, peak := sampleGasUtilization(t, ec, 10*time.Second)
		// Also compute a normalized utilization vs a 30M gas cap for readability when chain gasLimit is huge.
		navg, npeak := sampleNormalizedUtilization(t, ec, 10*time.Second, 30_000_000)
		t.Logf("Block gas utilization: raw avg=%.3f%% peak=%.3f%% | normalized(30M) avg=%.1f%% peak=%.1f%%", avg*100, peak*100, navg*100, npeak*100)
		if avg < 0.05 && peak < 0.1 { // very low; dump recent block gas details for debugging
			debugLogRecentBlockGas(t, ec, 8)
		}
        // Additionally compute aggregate gas/sec over a recent time window (by timestamps),
        // which is robust even if the chain has a long history.
        gsec, gpb, btMs := computeGasThroughputWindow(t, ec, 60) // last ~60s
        t.Logf("Throughput window: avg_gas_block=%.0f avg_block_time=%.1fms gas_sec=%.0f", gpb, btMs, gsec)
        // Report simple target status for 1 Ggas/s @ 100ms cadence.
        const (
            targetGasSec     = 1_000_000_000.0
            minGasSec        = 0.9 * targetGasSec // 900M gas/sec
            minBtMs          = 80.0               // acceptable block time window
            maxBtMs          = 150.0
            minGasPerBlock   = 90_000_000.0       // ~100M +/- 10%
            maxGasPerBlock   = 110_000_000.0
        )
        hit := gsec >= minGasSec && btMs >= minBtMs && btMs <= maxBtMs && gpb >= minGasPerBlock && gpb <= maxGasPerBlock
        if hit {
            t.Logf("Target 1Ggas/s@100ms: PASS (gas_sec=%.0f, avg_bt=%.1fms, gas_block=%.0f)", gsec, btMs, gpb)
        } else {
            t.Logf("Target 1Ggas/s@100ms: NOT MET (gas_sec=%.0f, avg_bt=%.1fms, gas_block=%.0f). Thresholds: gas_sec>=900,000,000; 80ms<=avg_bt<=150ms; 90M<=gas_block<=110M.", gsec, btMs, gpb)
        }
		// If metrics were unavailable earlier, ensure on-chain activity by checking recent tx counts.
		// Fail if we observe zero transactions in the last ~8 blocks.
		if metricsText == "" {
			header, herr := ec.HeaderByNumber(context.Background(), nil)
			if herr == nil && header != nil {
				var anyTx bool
				latest := new(big.Int).Set(header.Number)
				min := new(big.Int).Sub(latest, big.NewInt(int64(7)))
				if min.Sign() < 0 {
					min.SetInt64(0)
				}
				for b := new(big.Int).Set(latest); b.Cmp(min) >= 0; b.Sub(b, big.NewInt(1)) {
					blk, e := ec.BlockByNumber(context.Background(), b)
					if e == nil && blk != nil && len(blk.Transactions()) > 0 {
						anyTx = true
						break
					}
				}
				if !anyTx {
					t.Fatalf("no on-chain activity observed in recent blocks and metrics unavailable")
				}
			}
		}
	}
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

// sampleGasUtilization samples latest blocks for the given duration and returns
// average and peak gas used fraction.
func sampleGasUtilization(t *testing.T, ec *ethclient.Client, dur time.Duration) (avg, peak float64) {
	t.Helper()
	deadline := time.Now().Add(dur)
	var sum float64
	var n int
	for time.Now().Before(deadline) {
		header, err := ec.HeaderByNumber(context.Background(), nil)
		if err == nil && header != nil && header.GasLimit > 0 {
			frac := float64(header.GasUsed) / float64(header.GasLimit)
			sum += frac
			n++
			if frac > peak {
				peak = frac
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	if n > 0 {
		avg = sum / float64(n)
	}
	return
}

// sampleNormalizedUtilization computes utilization relative to a targetGasLimit rather than the chain's gasLimit.
func sampleNormalizedUtilization(t *testing.T, ec *ethclient.Client, dur time.Duration, targetGasLimit uint64) (avg, peak float64) {
	t.Helper()
	if targetGasLimit == 0 {
		return 0, 0
	}
	deadline := time.Now().Add(dur)
	var sum float64
	var n int
	for time.Now().Before(deadline) {
		header, err := ec.HeaderByNumber(context.Background(), nil)
		if err == nil && header != nil {
			frac := float64(header.GasUsed) / float64(targetGasLimit)
			if frac > 1 {
				frac = 1 // cap at 100%
			}
			sum += frac
			n++
			if frac > peak {
				peak = frac
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	if n > 0 {
		avg = sum / float64(n)
	}
	return
}

// debugLogRecentBlockGas logs gasUsed/gasLimit and tx count for the most recent n blocks.
func debugLogRecentBlockGas(t *testing.T, ec *ethclient.Client, n int) {
	t.Helper()
	// Get latest number
	header, err := ec.HeaderByNumber(context.Background(), nil)
	if err != nil || header == nil {
		t.Logf("debug: failed to fetch latest header: %v", err)
		return
	}
	latest := new(big.Int).Set(header.Number)
	min := new(big.Int).Sub(latest, big.NewInt(int64(n-1)))
	if min.Sign() < 0 {
		min.SetInt64(0)
	}
	t.Logf("debug: recent block gas (number gasUsed/gasLimit txs)")
	for b := new(big.Int).Set(latest); b.Cmp(min) >= 0; b.Sub(b, big.NewInt(1)) {
		h, err := ec.HeaderByNumber(context.Background(), b)
		if err != nil || h == nil {
			t.Logf("debug: block %s header error: %v", b.String(), err)
			continue
		}
		blk, err := ec.BlockByNumber(context.Background(), b)
		txc := 0
		if err == nil && blk != nil {
			txc = len(blk.Transactions())
		}
		t.Logf("debug: %s %d/%d txs=%d", h.Number.String(), h.GasUsed, h.GasLimit, txc)
	}
}

// computeGasThroughputWindow scans backwards from the latest block until it accumulates
// approximately windowSec seconds of chain time, then returns gas/sec, avg gas/block,
// and avg block time (ms) over that window.
func computeGasThroughputWindow(t *testing.T, ec *ethclient.Client, windowSec int) (gasPerSec, avgGasPerBlock, avgBlockTimeMs float64) {
    t.Helper()
    if windowSec <= 0 { windowSec = 60 }
    latestHeader, err := ec.HeaderByNumber(context.Background(), nil)
    if err != nil || latestHeader == nil { return 0, 0, 0 }
    latestNum := new(big.Int).Set(latestHeader.Number)
    latestTS := latestHeader.Time

    var sumGas uint64
    var count int
    var oldestTS uint64 = latestTS
    // Walk backwards up to a cap of blocks to avoid long scans if blocks are sparse.
    capBlocks := 5000
    for i := 0; i < capBlocks; i++ {
        num := new(big.Int).Sub(latestNum, big.NewInt(int64(i)))
        if num.Sign() < 0 { break }
        blk, err := ec.BlockByNumber(context.Background(), num)
        if err != nil || blk == nil { break }
        sumGas += blk.GasUsed()
        count++
        oldestTS = blk.Time()
        if latestTS - oldestTS >= uint64(windowSec) { break }
    }
    if count < 2 || latestTS <= oldestTS { return 0, 0, 0 }
    elapsed := float64(latestTS - oldestTS) // seconds
    gasPerSec = float64(sumGas) / elapsed
    avgGasPerBlock = float64(sumGas) / float64(count)
    avgBlockTimeMs = (elapsed / float64(count-1)) * 1000.0
    return
}
