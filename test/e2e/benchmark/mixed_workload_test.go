//go:build evm

package benchmark

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/celestiaorg/tastora/framework/docker/evstack/spamoor"
)

// TestMixedWorkload measures chain performance under a realistic mix of
// transaction types running concurrently. The default mix is 40% ERC20
// transfers, 30% Uniswap swaps, 20% gasburner, 10% storage writes.
//
// Mix proportions are configurable via BENCH_MIX_ERC20_PCT, BENCH_MIX_DEFI_PCT,
// BENCH_MIX_GASBURN_PCT, BENCH_MIX_STATE_PCT. These must sum to 100.
//
// BENCH_NUM_SPAMMERS is distributed across workload types proportionally,
// with a minimum of 1 spammer per type (so the minimum is 4 spammers).
//
// Primary metrics: MGas/s, TPS.
// Diagnostic metrics: per-span latency breakdown, ev-node overhead %.
func (s *SpamoorSuite) TestMixedWorkload() {
	cfg := newBenchConfig("ev-node-mixed")

	t := s.T()
	ctx := t.Context()
	cfg.log(t)
	w := newResultWriter(t, "MixedWorkload")
	defer w.flush()

	var result *benchmarkResult
	var wallClock time.Duration
	var spamoorStats *runSpamoorStats
	defer func() {
		if result != nil {
			emitRunResult(t, cfg, result, wallClock, spamoorStats)
		}
	}()

	// read mix proportions
	pcts := [4]int{
		envInt("BENCH_MIX_ERC20_PCT", 40),
		envInt("BENCH_MIX_DEFI_PCT", 30),
		envInt("BENCH_MIX_GASBURN_PCT", 20),
		envInt("BENCH_MIX_STATE_PCT", 10),
	}
	pctSum := pcts[0] + pcts[1] + pcts[2] + pcts[3]
	s.Require().Equal(100, pctSum, "BENCH_MIX_*_PCT values must sum to 100, got %d", pctSum)
	s.Require().GreaterOrEqual(cfg.NumSpammers, 4, "mixed workload requires at least 4 spammers (1 per type)")

	counts := distributeSpammers(cfg.NumSpammers, pcts)
	t.Logf("mix distribution: erc20=%d, defi=%d, gasburner=%d, state=%d (total=%d)",
		counts[0], counts[1], counts[2], counts[3], cfg.NumSpammers)

	e := s.setupEnv(cfg)
	s.Require().NoError(deleteAllSpammers(e.spamoorAPI), "failed to delete stale spammers")

	type workloadType struct {
		scenario string
		prefix   string
		config   map[string]any
		count    int
	}

	workloads := []workloadType{
		{
			scenario: spamoor.ScenarioERC20TX,
			prefix:   "bench-mixed-erc20",
			count:    counts[0],
			config: map[string]any{
				"throughput":      cfg.Throughput,
				"total_count":     cfg.CountPerSpammer,
				"max_pending":     cfg.MaxPending,
				"max_wallets":     cfg.MaxWallets,
				"base_fee":        cfg.BaseFee,
				"tip_fee":         cfg.TipFee,
				"refill_amount":   "5000000000000000000", // 5 ETH
				"refill_balance":  "2000000000000000000", // 2 ETH
				"refill_interval": 600,
			},
		},
		{
			scenario: spamoor.ScenarioUniswapSwaps,
			prefix:   "bench-mixed-defi",
			count:    counts[1],
			config: map[string]any{
				"throughput":      cfg.Throughput,
				"total_count":     cfg.CountPerSpammer,
				"max_pending":     cfg.MaxPending,
				"max_wallets":     cfg.MaxWallets,
				"pair_count":      envInt("BENCH_PAIR_COUNT", 1),
				"rebroadcast":     cfg.Rebroadcast,
				"base_fee":        cfg.BaseFee,
				"tip_fee":         cfg.TipFee,
				"refill_amount":   "10000000000000000000", // 10 ETH (swaps need ETH for WETH wrapping)
				"refill_balance":  "5000000000000000000",  // 5 ETH
				"refill_interval": 600,
			},
		},
		{
			scenario: spamoor.ScenarioGasBurnerTX,
			prefix:   "bench-mixed-gasburner",
			count:    counts[2],
			config: map[string]any{
				"gas_units_to_burn": cfg.GasUnitsToBurn,
				"total_count":       cfg.CountPerSpammer,
				"throughput":        cfg.Throughput,
				"max_pending":       cfg.MaxPending,
				"max_wallets":       cfg.MaxWallets,
				"rebroadcast":       cfg.Rebroadcast,
				"base_fee":          cfg.BaseFee,
				"tip_fee":           cfg.TipFee,
				"refill_amount":     "500000000000000000000", // 500 ETH
				"refill_balance":    "200000000000000000000", // 200 ETH
				"refill_interval":   300,
			},
		},
		{
			scenario: spamoor.ScenarioStorageSpam,
			prefix:   "bench-mixed-state",
			count:    counts[3],
			config: map[string]any{
				"throughput":        cfg.Throughput,
				"total_count":       cfg.CountPerSpammer,
				"gas_units_to_burn": cfg.GasUnitsToBurn,
				"max_pending":       cfg.MaxPending,
				"max_wallets":       cfg.MaxWallets,
				"rebroadcast":       cfg.Rebroadcast,
				"base_fee":          cfg.BaseFee,
				"tip_fee":           cfg.TipFee,
				"refill_amount":     "5000000000000000000", // 5 ETH
				"refill_balance":    "2000000000000000000", // 2 ETH
				"refill_interval":   600,
			},
		},
	}

	spammerIDs := make([]int, 0, cfg.NumSpammers)
	for _, wl := range workloads {
		for i := range wl.count {
			name := fmt.Sprintf("%s-%d", wl.prefix, i)
			id, err := e.spamoorAPI.CreateSpammer(name, wl.scenario, wl.config, true)
			s.Require().NoError(err, "failed to create spammer %s", name)
			spammerIDs = append(spammerIDs, id)
			t.Cleanup(func() { _ = e.spamoorAPI.DeleteSpammer(id) })
		}
	}

	requireSpammersRunning(t, e.spamoorAPI, spammerIDs)

	pollSentTotal := func() (float64, error) {
		metrics, mErr := e.spamoorAPI.GetMetrics()
		if mErr != nil {
			return 0, mErr
		}
		return sumCounter(metrics["spamoor_transactions_sent_total"]), nil
	}

	// wait for at least one tx per spammer to confirm contract deployments are done
	waitForMetricTarget(t, "spamoor_transactions_sent_total (deploy)", pollSentTotal, float64(len(spammerIDs)), cfg.WaitTimeout)
	waitForMetricTarget(t, "spamoor_transactions_sent_total (warmup)", pollSentTotal, float64(cfg.WarmupTxs), cfg.WaitTimeout)

	e.traces.resetStartTime()

	startHeader, err := e.ethClient.HeaderByNumber(ctx, nil)
	s.Require().NoError(err, "failed to get start block header")
	startBlock := startHeader.Number.Uint64()
	loadStart := time.Now()
	t.Logf("start block: %d (after warmup)", startBlock)

	waitForMetricTarget(t, "spamoor_transactions_sent_total", pollSentTotal, float64(cfg.totalCount()), cfg.WaitTimeout)

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

	bm, err := collectBlockMetrics(ctx, e.ethClient, startBlock, endBlock)
	s.Require().NoError(err, "failed to collect block metrics")

	traces := s.collectTraces(e)

	result = newBenchmarkResult("MixedWorkload", bm, traces)
	s.Require().Greater(result.summary.SteadyState, time.Duration(0), "expected non-zero steady-state duration")
	result.log(t, wallClock)
	w.addEntries(result.entries())

	metrics, mErr := e.spamoorAPI.GetMetrics()
	s.Require().NoError(mErr, "failed to get final metrics")
	sent := sumCounter(metrics["spamoor_transactions_sent_total"])
	failed := sumCounter(metrics["spamoor_transactions_failed_total"])
	spamoorStats = &runSpamoorStats{Sent: sent, Failed: failed}

	s.Require().Greater(sent, float64(0), "at least one transaction should have been sent")
	s.Require().Zero(failed, "no transactions should have failed")
}

// distributeSpammers divides total spammers across 4 workload types
// proportionally to pcts using the largest-remainder method. Each type
// gets at least 1 spammer. total must be >= 4.
func distributeSpammers(total int, pcts [4]int) [4]int {
	if total < 4 {
		panic(fmt.Sprintf("distributeSpammers: total must be >= 4, got %d", total))
	}
	var counts [4]int
	for i := range counts {
		counts[i] = 1
	}
	remaining := total - 4
	if remaining == 0 {
		return counts
	}

	type indexedFrac struct {
		index int
		frac  float64
	}

	var fracs [4]indexedFrac
	allocated := 0
	for i, pct := range pcts {
		ideal := float64(remaining) * float64(pct) / 100.0
		floor := int(ideal)
		counts[i] += floor
		allocated += floor
		fracs[i] = indexedFrac{index: i, frac: ideal - float64(floor)}
	}

	// distribute leftover by descending fractional part
	sort.Slice(fracs[:], func(a, b int) bool {
		return fracs[a].frac > fracs[b].frac
	})
	leftover := remaining - allocated
	for i := 0; i < leftover && i < 4; i++ {
		counts[fracs[i].index]++
	}

	return counts
}
