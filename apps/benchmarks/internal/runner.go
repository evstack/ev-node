package internal

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"time"

	"github.com/celestiaorg/tastora/framework/docker/evstack/spamoor"
	dto "github.com/prometheus/client_model/go"
)

func ExecuteMatrix(matrixPath, spamoorAddr string) error {
	matrix, err := LoadMatrix(matrixPath)
	if err != nil {
		return fmt.Errorf("load matrix: %w", err)
	}

	api := spamoor.NewAPI(spamoorAddr)
	log.Printf("spamoor API: %s", spamoorAddr)
	log.Printf("loaded %d matrix entries from %s", len(matrix.Entries), matrixPath)

	var failures []string
	for i, entry := range matrix.Entries {
		log.Printf("--- [%d/%d] %s ---", i+1, len(matrix.Entries), entry.TestName)

		if entry.Probability != nil {
			roll := rand.Float64()
			if roll >= *entry.Probability {
				log.Printf("[%s] skipped (probability=%.2f, roll=%.4f)", entry.TestName, *entry.Probability, roll)
				continue
			}
			log.Printf("[%s] triggered (probability=%.2f, roll=%.4f)", entry.TestName, *entry.Probability, roll)
		}

		timeout := 15 * time.Minute
		if entry.Timeout != "" {
			parsed, pErr := time.ParseDuration(entry.Timeout)
			if pErr != nil {
				log.Printf("warning: invalid timeout %q, using default %s", entry.Timeout, timeout)
			} else {
				timeout = parsed
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		if err := runEntry(ctx, api, entry); err != nil {
			log.Printf("FAIL [%s]: %v", entry.TestName, err)
			failures = append(failures, entry.TestName)
		}
		cancel()
	}

	log.Printf("=== completed: %d/%d succeeded ===", len(matrix.Entries)-len(failures), len(matrix.Entries))
	if len(failures) > 0 {
		return fmt.Errorf("failed entries: %v", failures)
	}
	return nil
}

func RunCheck(spamoorAddr string, timeout time.Duration) error {
	api := spamoor.NewAPI(spamoorAddr)

	log.Printf("checking connectivity via %s", spamoorAddr)

	if _, err := api.ListSpammers(); err != nil {
		return fmt.Errorf("cannot reach spamoor at %s: %w", spamoorAddr, err)
	}
	log.Printf("spamoor reachable")

	if err := deleteAllSpammers(api); err != nil {
		return fmt.Errorf("cleanup: %w", err)
	}
	defer func() { _ = deleteAllSpammers(api) }()

	cfg := map[string]any{
		"total_count":     1,
		"throughput":      1,
		"max_pending":     10,
		"max_wallets":     1,
		"base_fee":        20,
		"tip_fee":         2,
		"refill_amount":   "500000000000000000000",
		"refill_balance":  "200000000000000000000",
		"refill_interval": 300,
	}

	id, err := api.CreateSpammer("check-eoatx", spamoor.ScenarioEOATX, cfg, true)
	if err != nil {
		return fmt.Errorf("create spammer: %w", err)
	}
	log.Printf("created check spammer (id=%d)", id)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	sent, failed, err := waitForSpamoorDone(ctx, api, 1)
	if err != nil {
		return fmt.Errorf("tx not confirmed: %w", err)
	}

	if failed > 0 {
		return fmt.Errorf("tx failed (sent=%.0f failed=%.0f)", sent, failed)
	}

	log.Printf("check passed: 1 tx sent successfully")
	return nil
}

func runEntry(ctx context.Context, api *spamoor.API, entry Entry) error {
	totalCount := entry.NumSpammers * entry.CountPerSpammer

	log.Printf("[%s] scenario=%s spammers=%d count_per=%d total=%d",
		entry.TestName, entry.Scenario, entry.NumSpammers, entry.CountPerSpammer, totalCount)

	if err := deleteAllSpammers(api); err != nil {
		return fmt.Errorf("delete stale spammers: %w", err)
	}
	defer func() {
		if err := deleteAllSpammers(api); err != nil {
			log.Printf("[%s] warning: cleanup failed: %v", entry.TestName, err)
		}
	}()

	scenarioCfg := BuildScenarioConfig(entry.Env)

	var spammerIDs []int
	for i := range entry.NumSpammers {
		name := fmt.Sprintf("bench-%s-%d", entry.TestName, i)
		id, err := api.CreateSpammer(name, entry.Scenario, scenarioCfg, true)
		if err != nil {
			return fmt.Errorf("create spammer %s: %w", name, err)
		}
		spammerIDs = append(spammerIDs, id)
		log.Printf("[%s] created spammer %s (id=%d)", entry.TestName, name, id)
	}

	for _, id := range spammerIDs {
		sp, err := api.GetSpammer(id)
		if err != nil {
			return fmt.Errorf("get spammer %d: %w", id, err)
		}
		if sp.Status == 0 {
			return fmt.Errorf("spammer %d (%s) failed to start (status=0)", id, sp.Name)
		}
	}

	start := time.Now()
	sent, failed, err := waitForSpamoorDone(ctx, api, totalCount)
	elapsed := time.Since(start)

	if err != nil {
		log.Printf("[%s] ERROR: %v (sent=%.0f failed=%.0f elapsed=%s)",
			entry.TestName, err, sent, failed, elapsed.Round(time.Second))
		return err
	}

	log.Printf("[%s] DONE: sent=%.0f failed=%.0f elapsed=%s avg_rate=%.1f tx/s",
		entry.TestName, sent, failed, elapsed.Round(time.Second), sent/elapsed.Seconds())

	return nil
}

func deleteAllSpammers(api *spamoor.API) error {
	existing, err := api.ListSpammers()
	if err != nil {
		return fmt.Errorf("list spammers: %w", err)
	}
	for _, sp := range existing {
		if err := api.DeleteSpammer(sp.ID); err != nil {
			return fmt.Errorf("delete spammer %d: %w", sp.ID, err)
		}
	}
	return nil
}

func waitForSpamoorDone(ctx context.Context, api *spamoor.API, targetCount int) (sent, failed float64, err error) {
	const pollInterval = 2 * time.Second
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	start := time.Now()
	var prevSent float64

	for {
		select {
		case <-ctx.Done():
			return sent, failed, fmt.Errorf("timed out waiting for %d txs (sent %.0f): %w", targetCount, sent, ctx.Err())
		case <-ticker.C:
			metrics, mErr := api.GetMetrics()
			if mErr != nil {
				log.Printf("warning: failed to get metrics: %v", mErr)
				continue
			}
			sent = sumCounter(metrics["spamoor_transactions_sent_total"])
			failed = sumCounter(metrics["spamoor_transactions_failed_total"])

			delta := sent - prevSent
			rate := delta / pollInterval.Seconds()
			elapsed := time.Since(start).Round(time.Second)
			log.Printf("  progress: %.0f/%d sent (%.0f tx/s instant, %.1f tx/s avg, %.0f failed) [%s]",
				sent, targetCount, rate, sent/time.Since(start).Seconds(), failed, elapsed)
			prevSent = sent

			if sent >= float64(targetCount) {
				return sent, failed, nil
			}
		}
	}
}

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
