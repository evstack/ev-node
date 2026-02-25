//go:build evm

package benchmark

import (
	"time"

	"github.com/celestiaorg/tastora/framework/docker/evstack/spamoor"
	e2e "github.com/evstack/ev-node/test/e2e"
)

// TestSpamoorSmoke spins up reth + sequencer and a Spamoor node, starts a few
// basic spammers, waits briefly, then validates trace spans and prints a concise
// metrics summary.
func (s *SpamoorSuite) TestSpamoorSmoke() {
	t := s.T()
	w := newResultWriter(t, "SpamoorSmoke")
	defer w.flush()

	e := s.setupEnv(config{
		rethTag:     "pr-140",
		serviceName: "ev-node-smoke",
	})
	api := e.spamoorAPI

	eoatx := map[string]any{
		"throughput":      100,
		"total_count":     3000,
		"max_pending":     4000,
		"max_wallets":     300,
		"amount":          100,
		"random_amount":   true,
		"random_target":   true,
		"base_fee":        20,
		"tip_fee":         2,
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
	s.Require().NoError(err, "failed to create eoatx spammer")
	ids = append(ids, id)
	id, err = api.CreateSpammer("smoke-gasburner", spamoor.ScenarioGasBurnerTX, gasburner, true)
	s.Require().NoError(err, "failed to create gasburner spammer")
	ids = append(ids, id)

	for _, id := range ids {
		idToDelete := id
		t.Cleanup(func() { _ = api.DeleteSpammer(idToDelete) })
	}

	// allow spamoor enough time to generate transaction throughput
	// so that the expected tracing spans appear in Jaeger.
	time.Sleep(60 * time.Second)

	// fetch parsed metrics and print a concise summary.
	metrics, err := api.GetMetrics()
	s.Require().NoError(err, "failed to get metrics")
	sent := sumCounter(metrics["spamoor_transactions_sent_total"])
	fail := sumCounter(metrics["spamoor_transactions_failed_total"])

	// collect traces
	evNodeSpans := s.collectServiceTraces(e, "ev-node-smoke")
	evRethSpans := s.collectServiceTraces(e, "ev-reth")
	e2e.PrintTraceReport(t, "ev-node-smoke", evNodeSpans)
	e2e.PrintTraceReport(t, "ev-reth", evRethSpans)

	w.addSpans(append(evNodeSpans, evRethSpans...))

	// assert expected ev-node span names
	assertSpanNames(t, evNodeSpans, []string{
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
	}, "ev-node-smoke")

	// assert expected ev-reth span names
	assertSpanNames(t, evRethSpans, []string{
		"build_payload",
		"execute_tx",
		"try_build",
		"validate_transaction",
	}, "ev-reth")

	s.Require().Greater(sent, float64(0), "at least one transaction should have been sent")
	s.Require().Zero(fail, "no transactions should have failed")
}
