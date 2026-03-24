//go:build evm

package benchmark

import (
	"context"
	"fmt"
	"time"

	"github.com/celestiaorg/tastora/framework/docker/evstack/spamoor"
)

// TestGasBurner measures gas throughput using a deterministic gasburner
// workload. The result is tracked via BENCH_JSON_OUTPUT as seconds_per_gigagas
// (lower is better) on the benchmark dashboard.
func (s *SpamoorSuite) TestGasBurner() {
	cfg := newBenchConfig("ev-node-gasburner")

	t := s.T()
	ctx := t.Context()
	w := newResultWriter(t, "GasBurner")
	defer w.flush()

	cfg.log(t)

	e := s.setupEnv(cfg)
	api := e.spamoorAPI

	s.Require().NoError(deleteAllSpammers(api), "failed to delete stale spammers")

	gasburnerCfg := map[string]any{
		"gas_units_to_burn": cfg.GasUnitsToBurn,
		"total_count":       cfg.CountPerSpammer,
		"throughput":        cfg.Throughput,
		"max_pending":       50000,
		"max_wallets":       cfg.MaxWallets,
		"rebroadcast":       5,
		"base_fee":          100,
		"tip_fee":           50,
		"refill_amount":     "500000000000000000000",
		"refill_balance":    "200000000000000000000",
		"refill_interval":   300,
	}

	var spammerIDs []int
	for i := range cfg.NumSpammers {
		name := fmt.Sprintf("bench-gasburner-%d", i)
		id, err := api.CreateSpammer(name, spamoor.ScenarioGasBurnerTX, gasburnerCfg, true)
		s.Require().NoError(err, "failed to create spammer %s", name)
		spammerIDs = append(spammerIDs, id)
		t.Cleanup(func() { _ = api.DeleteSpammer(id) })
	}

	// verify spammers started successfully
	requireSpammersRunning(t, api, spammerIDs)

	// wait for wallet prep and contract deployment to finish before
	// recording start block so warmup is excluded from the measurement.
	pollSentTotal := func() (float64, error) {
		metrics, mErr := api.GetMetrics()
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
	wallClock := time.Since(loadStart)

	endHeader, err := e.ethClient.HeaderByNumber(ctx, nil)
	s.Require().NoError(err, "failed to get end block header")
	endBlock := endHeader.Number.Uint64()
	t.Logf("end block: %d (range %d blocks)", endBlock, endBlock-startBlock)

	// collect block-level gas/tx metrics
	bm, err := collectBlockMetrics(ctx, e.ethClient, startBlock, endBlock)
	s.Require().NoError(err, "failed to collect block metrics")

	traces := s.collectTraces(e, cfg.ServiceName)

	result := newBenchmarkResult("GasBurner", bm, traces)
	s.Require().Greater(result.summary.SteadyState, time.Duration(0), "expected non-zero steady-state duration")
	result.log(t, wallClock)
	w.addEntries(result.entries())

	metrics, mErr := api.GetMetrics()
	s.Require().NoError(mErr, "failed to get final metrics")
	sent := sumCounter(metrics["spamoor_transactions_sent_total"])
	failed := sumCounter(metrics["spamoor_transactions_failed_total"])

	emitRunResult(t, cfg, result, wallClock, &runSpamoorStats{Sent: sent, Failed: failed})
}
