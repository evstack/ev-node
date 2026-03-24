//go:build evm

package benchmark

import (
	"context"
	"fmt"
	"time"

	"github.com/celestiaorg/tastora/framework/docker/evstack/spamoor"
)

// TestDeFiSimulation measures throughput under a realistic DeFi workload using
// Uniswap V2 swaps. Each tx involves deep call chains, event emission, and
// multi-contract storage operations — representative of production L2 traffic.
//
// Shares system configuration with TestERC20Throughput (100ms blocks, 100M gas,
// 25ms scrape) so the only variable is workload type. The gap between ERC20
// and Uniswap MGas/s shows the cost of workload complexity.
//
// Primary metrics: MGas/s, TPS.
// Diagnostic metrics: per-span latency breakdown, ev-node overhead %.
func (s *SpamoorSuite) TestDeFiSimulation() {
	cfg := newBenchConfig("ev-node-defi")

	t := s.T()
	ctx := t.Context()
	cfg.log(t)
	w := newResultWriter(t, "DeFiSimulation")
	defer w.flush()

	var result *benchmarkResult
	var wallClock time.Duration
	var spamoorStats *runSpamoorStats
	defer func() {
		if result != nil {
			emitRunResult(t, cfg, result, wallClock, spamoorStats)
		}
	}()

	e := s.setupEnv(cfg)

	uniswapConfig := map[string]any{
		"throughput":      cfg.Throughput,
		"total_count":     cfg.CountPerSpammer,
		"max_pending":     cfg.MaxPending,
		"max_wallets":     cfg.MaxWallets,
		"pair_count":      envInt("BENCH_PAIR_COUNT", 1),
		"rebroadcast":     cfg.Rebroadcast,
		"base_fee":        cfg.BaseFee,
		"tip_fee":         cfg.TipFee,
		"refill_amount":   "10000000000000000000", // 10 ETH (swaps need ETH for WETH wrapping and router approvals)
		"refill_balance":  "5000000000000000000",  // 5 ETH
		"refill_interval": 600,
	}

	s.Require().NoError(deleteAllSpammers(e.spamoorAPI), "failed to delete stale spammers")

	// launch all spammers before recording startBlock so warm-up
	// (Uniswap contract deploys + liquidity provision + wallet funding)
	// is excluded from the measurement window.
	var spammerIDs []int
	for i := range cfg.NumSpammers {
		name := fmt.Sprintf("bench-defi-%d", i)
		id, err := e.spamoorAPI.CreateSpammer(name, spamoor.ScenarioUniswapSwaps, uniswapConfig, true)
		s.Require().NoError(err, "failed to create spammer %s", name)
		spammerIDs = append(spammerIDs, id)
		t.Cleanup(func() { _ = e.spamoorAPI.DeleteSpammer(id) })
	}

	// give spammers time to deploy contracts and provision liquidity,
	// then verify none failed during warmup.
	time.Sleep(5 * time.Second)
	requireSpammersRunning(t, e.spamoorAPI, spammerIDs)

	// wait for warmup transactions (contract deploys, liquidity adds) to land
	// before recording start block.
	pollSentTotal := func() (float64, error) {
		metrics, mErr := e.spamoorAPI.GetMetrics()
		if mErr != nil {
			return 0, mErr
		}
		return sumCounter(metrics["spamoor_transactions_sent_total"]), nil
	}
	waitForMetricTarget(t, "spamoor_transactions_sent_total (warmup)", pollSentTotal, float64(cfg.WarmupTxs), cfg.WaitTimeout)

	// reset trace window to exclude warmup spans
	e.traces.resetStartTime()

	startHeader, err := e.ethClient.HeaderByNumber(ctx, nil)
	s.Require().NoError(err, "failed to get start block header")
	startBlock := startHeader.Number.Uint64()
	loadStart := time.Now()
	t.Logf("start block: %d (after warmup)", startBlock)

	// wait for all transactions to be sent
	waitForMetricTarget(t, "spamoor_transactions_sent_total", pollSentTotal, float64(cfg.totalCount()), cfg.WaitTimeout)

	// wait for pending txs to drain
	drainCtx, drainCancel := context.WithTimeout(ctx, 30*time.Second)
	defer drainCancel()
	if err := waitForDrain(drainCtx, t.Logf, e.ethClient, 10); err != nil {
		t.Logf("warning: %v", err)
	}
	wallClock = time.Since(loadStart)

	endHeader, err := e.ethClient.HeaderByNumber(ctx, nil)
	s.Require().NoError(err, "failed to get end block header")
	endBlock := endHeader.Number.Uint64()
	t.Logf("end block: %d (range %d blocks)", endBlock, endBlock-startBlock)

	// collect block-level gas/tx metrics
	bm, err := collectBlockMetrics(ctx, e.ethClient, startBlock, endBlock)
	s.Require().NoError(err, "failed to collect block metrics")

	traces := s.collectTraces(e, cfg.ServiceName)

	result = newBenchmarkResult("DeFiSimulation", bm, traces)
	s.Require().Greater(result.summary.SteadyState, time.Duration(0), "expected non-zero steady-state duration")
	result.log(t, wallClock)
	w.addEntries(result.entries())

	metrics, mErr := e.spamoorAPI.GetMetrics()
	s.Require().NoError(mErr, "failed to get final metrics")
	sent := sumCounter(metrics["spamoor_transactions_sent_total"])
	failed := sumCounter(metrics["spamoor_transactions_failed_total"])
	spamoorStats = &runSpamoorStats{Sent: sent, Failed: failed}
}
