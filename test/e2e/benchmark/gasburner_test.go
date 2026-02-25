//go:build evm

package benchmark

import (
	"fmt"
	"time"

	"github.com/celestiaorg/tastora/framework/docker/evstack/spamoor"
	e2e "github.com/evstack/ev-node/test/e2e"
)

// TestGasBurner measures gas throughput using a deterministic gasburner
// workload. The result is tracked via BENCH_JSON_OUTPUT as seconds_per_gigagas
// (lower is better) on the benchmark dashboard.
func (s *SpamoorSuite) TestGasBurner() {
	t := s.T()
	w := newResultWriter(t, "GasBurner")
	defer w.flush()

	e := s.setupEnv(config{
		rethTag:     "pr-142",
		serviceName: "ev-node-gasburner",
	})
	api := e.spamoorAPI

	const totalCount = 10000
	gasburnerCfg := map[string]any{
		"gas_units_to_burn": 3_000_000,
		"total_count":       totalCount,
		"throughput":         1000,
		"max_pending":        5000,
		"max_wallets":        500,
		"rebroadcast":        0,
		"base_fee":           20,
		"tip_fee":            5,
		"refill_amount":     "5000000000000000000",
		"refill_balance":    "2000000000000000000",
		"refill_interval":   300,
	}

	id, err := api.CreateSpammer("bench-gasburner", spamoor.ScenarioGasBurnerTX, gasburnerCfg, true)
	s.Require().NoError(err, "failed to create gasburner spammer")
	t.Cleanup(func() { _ = api.DeleteSpammer(id) })

	// wait for wallet prep and contract deployment to finish before
	// recording start block so warmup is excluded from the measurement.
	const warmupTxs = 50
	pollSentTotal := func() (float64, error) {
		metrics, err := api.GetMetrics()
		if err != nil {
			return 0, err
		}
		return sumCounter(metrics["spamoor_transactions_sent_total"]), nil
	}
	waitForMetricTarget(t, "spamoor_transactions_sent_total (warmup)", pollSentTotal, warmupTxs, 5*time.Minute)

	startHeader, err := e.ethClient.HeaderByNumber(t.Context(), nil)
	s.Require().NoError(err, "failed to get start block header")
	startBlock := startHeader.Number.Uint64()
	t.Logf("start block: %d (after wallet prep)", startBlock)

	waitForMetricTarget(t, "spamoor_transactions_sent_total", pollSentTotal, float64(totalCount), 5*time.Minute)

	endHeader, err := e.ethClient.HeaderByNumber(t.Context(), nil)
	s.Require().NoError(err, "failed to get end block header")
	endBlock := endHeader.Number.Uint64()
	t.Logf("end block: %d (range %d blocks)", endBlock, endBlock-startBlock)

	gas := measureGasThroughput(t, t.Context(), e.ethClient, startBlock, endBlock)

	// collect traces
	evNodeSpans := s.collectServiceTraces(e, "ev-node-gasburner")
	evRethSpans := s.collectServiceTraces(e, "ev-reth")
	e2e.PrintTraceReport(t, "ev-node-gasburner", evNodeSpans)
	e2e.PrintTraceReport(t, "ev-reth", evRethSpans)

	// assert expected ev-reth spans
	assertSpanNames(t, evRethSpans, []string{
		"build_payload",
		"try_build",
		"validate_transaction",
		"validate_evnode",
		"try_new",
		"execute_tx",
	}, "ev-reth")

	w.addSpans(append(evNodeSpans, evRethSpans...))
	w.addEntry(entry{
		Name:  fmt.Sprintf("%s - seconds_per_gigagas", w.label),
		Unit:  "s/Ggas",
		Value: 1.0 / gas.gigagasPerSec,
	})
}
