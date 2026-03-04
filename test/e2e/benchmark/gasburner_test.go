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
	const (
		numSpammers     = 8
		countPerSpammer = 5000
		totalCount      = numSpammers * countPerSpammer
		warmupTxs       = 500
		serviceName     = "ev-node" // TODO: temp change
		waitTimeout     = 10 * time.Minute
	)

	t := s.T()
	ctx := t.Context()
	w := newResultWriter(t, "GasBurner")
	defer w.flush()

	e := s.setupEnv(config{
		serviceName: serviceName,
	})
	api := e.spamoorAPI

	s.Require().NoError(deleteAllSpammers(api), "failed to delete stale spammers")

	gasburnerCfg := map[string]any{
		"gas_units_to_burn": 1_000_000,
		"total_count":       countPerSpammer,
		"throughput":        200,
		"max_pending":       50000,
		"max_wallets":       500,
		"rebroadcast":       0,
		"base_fee":          100,
		"tip_fee":           50,
		"refill_amount":     "5000000000000000000",
		"refill_balance":    "2000000000000000000",
		"refill_interval":   300,
	}

	for i := range numSpammers {
		name := fmt.Sprintf("bench-gasburner-%d", i)
		id, err := api.CreateSpammer(name, spamoor.ScenarioGasBurnerTX, gasburnerCfg, true)
		s.Require().NoError(err, "failed to create spammer %s", name)
		t.Cleanup(func() { _ = api.DeleteSpammer(id) })
	}

	// wait for wallet prep and contract deployment to finish before
	// recording start block so warmup is excluded from the measurement.
	pollSentTotal := func() (float64, error) {
		metrics, mErr := api.GetMetrics()
		if mErr != nil {
			return 0, mErr
		}
		return sumCounter(metrics["spamoor_transactions_sent_total"]), nil
	}
	waitForMetricTarget(t, "spamoor_transactions_sent_total (warmup)", pollSentTotal, warmupTxs, waitTimeout)

	startHeader, err := e.ethClient.HeaderByNumber(ctx, nil)
	s.Require().NoError(err, "failed to get start block header")
	startBlock := startHeader.Number.Uint64()
	loadStart := time.Now()
	t.Logf("start block: %d (after warmup)", startBlock)

	// wait for all transactions to be sent
	waitForMetricTarget(t, "spamoor_transactions_sent_total", pollSentTotal, float64(totalCount), waitTimeout)

	// wait for pending txs to drain
	drainCtx, drainCancel := context.WithTimeout(ctx, 30*time.Second)
	defer drainCancel()
	waitForDrain(drainCtx, t.Logf, e.ethClient, 10)
	wallClock := time.Since(loadStart)

	endHeader, err := e.ethClient.HeaderByNumber(ctx, nil)
	s.Require().NoError(err, "failed to get end block header")
	endBlock := endHeader.Number.Uint64()
	t.Logf("end block: %d (range %d blocks)", endBlock, endBlock-startBlock)

	// collect block-level gas/tx metrics
	bm, err := collectBlockMetrics(ctx, e.ethClient, startBlock, endBlock)
	s.Require().NoError(err, "failed to collect block metrics")

	summary := bm.summarize()
	s.Require().Greater(summary.SteadyState, time.Duration(0), "expected non-zero steady-state duration")
	summary.log(t, startBlock, endBlock, bm.TotalBlockCount, bm.BlockCount, wallClock)

	// derive seconds_per_gigagas from the summary's MGas/s
	var secsPerGigagas float64
	if summary.AchievedMGas > 0 {
		// MGas/s -> Ggas/s = MGas/s / 1000, then invert
		secsPerGigagas = 1000.0 / summary.AchievedMGas
	}
	t.Logf("seconds_per_gigagas: %.4f", secsPerGigagas)

	// collect and report traces
	traces := s.collectTraces(e, serviceName)

	if overhead, ok := evNodeOverhead(traces.evNode); ok {
		t.Logf("ev-node overhead: %.1f%%", overhead)
		w.addEntry(entry{Name: "GasBurner - ev-node overhead", Unit: "%", Value: overhead})
	}

	w.addEntries(summary.entries("GasBurner"))
	w.addSpans(traces.allSpans())
	w.addEntry(entry{
		Name:  fmt.Sprintf("%s - seconds_per_gigagas", w.label),
		Unit:  "s/Ggas",
		Value: secsPerGigagas,
	})
}
