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

type matrixOpts struct {
	totalTxTarget    int
	allowProbability bool
	waitForSync      bool
}

// ExecuteMatrixWithOverrides runs a matrix with BENCH_COUNT_PER_SPAMMER
// overridden to totalTxTarget / NumSpammers per entry. Skips probability
// filtering and sync waiting (caller is responsible for sync).
func ExecuteMatrixWithOverrides(ctx context.Context, matrix Matrix, api SpamoorClient, totalTxTarget int) error {
	return executeMatrix(ctx, matrix, api, matrixOpts{
		totalTxTarget: totalTxTarget,
	})
}

// ExecuteMatrix runs all entries in a matrix file as-is, with probability
// filtering and an initial WaitForSync call. Used by the one-shot `run` command.
func ExecuteMatrix(ctx context.Context, matrix Matrix, api SpamoorClient) error {
	return executeMatrix(ctx, matrix, api, matrixOpts{
		allowProbability: true,
		waitForSync:      true,
	})
}

// ExecuteMatrixFromFile loads a matrix from disk and executes it.
func ExecuteMatrixFromFile(ctx context.Context, matrixPath string, api SpamoorClient) error {
	matrix, err := LoadMatrix(matrixPath)
	if err != nil {
		return fmt.Errorf("load matrix: %w", err)
	}
	return ExecuteMatrix(ctx, *matrix, api)
}

// ExecuteMatrixWithOverridesFromFile loads a matrix from disk and executes it
// with per-run counts overridden.
func ExecuteMatrixWithOverridesFromFile(ctx context.Context, matrixPath string, api SpamoorClient, totalTxTarget int) error {
	matrix, err := LoadMatrix(matrixPath)
	if err != nil {
		return fmt.Errorf("load matrix: %w", err)
	}
	return ExecuteMatrixWithOverrides(ctx, *matrix, api, totalTxTarget)
}

func executeMatrix(ctx context.Context, matrix Matrix, api SpamoorClient, opts matrixOpts) error {
	if err := validateMatrix(&matrix); err != nil {
		return err
	}

	log.Printf("spamoor API: %s", api.URL())
	if opts.totalTxTarget > 0 {
		log.Printf("loaded %d matrix entries (totalTxTarget=%d)", len(matrix.Entries), opts.totalTxTarget)
	} else {
		log.Printf("loaded %d matrix entries", len(matrix.Entries))
	}

	if opts.waitForSync {
		if err := WaitForSync(ctx, api); err != nil {
			return fmt.Errorf("waiting for spamoor sync: %w", err)
		}
	}

	var failures []string
	for i, entry := range matrix.Entries {
		if err := ctx.Err(); err != nil {
			log.Printf("cancelled before entry %d/%d", i+1, len(matrix.Entries))
			return err
		}

		log.Printf("--- [%d/%d] %s ---", i+1, len(matrix.Entries), entry.TestName)

		if opts.allowProbability && entry.Probability != nil {
			roll := rand.Float64()
			if roll >= *entry.Probability {
				log.Printf("[%s] skipped (probability=%.2f, roll=%.4f)", entry.TestName, *entry.Probability, roll)
				continue
			}
			log.Printf("[%s] triggered (probability=%.2f, roll=%.4f)", entry.TestName, *entry.Probability, roll)
		}

		if opts.totalTxTarget > 0 {
			countPerSpammer := opts.totalTxTarget / entry.NumSpammers
			if countPerSpammer < 1 {
				countPerSpammer = 1
			}
			entry.Env["BENCH_COUNT_PER_SPAMMER"] = fmt.Sprintf("%d", countPerSpammer)
			entry.CountPerSpammer = countPerSpammer
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

		entryCtx, cancel := context.WithTimeout(ctx, timeout)
		if err := runEntry(entryCtx, api, entry); err != nil {
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

// RunCheck verifies connectivity by sending a single eoatx through spamoor.
func RunCheck(ctx context.Context, spamoorAddr string, timeout time.Duration) error {
	return runCheck(ctx, NewSpamoorClient(spamoorAddr), timeout)
}

func runCheck(ctx context.Context, api SpamoorClient, timeout time.Duration) error {
	spamoorAddr := api.URL()
	log.Printf("checking connectivity via %s", spamoorAddr)

	if _, err := api.ListSpammers(); err != nil {
		return fmt.Errorf("cannot reach spamoor at %s: %w", spamoorAddr, err)
	}
	log.Printf("spamoor reachable")

	if err := WaitForSync(ctx, api); err != nil {
		return fmt.Errorf("waiting for spamoor sync: %w", err)
	}

	if err := DeleteAllSpammers(api); err != nil {
		return fmt.Errorf("cleanup: %w", err)
	}
	defer func() { _ = DeleteAllSpammers(api) }()

	baselineSent, baselineFailed, err := getSpamoorCounters(api)
	if err != nil {
		return fmt.Errorf("get baseline metrics: %w", err)
	}

	cfg := map[string]any{
		"total_count":     1,
		"throughput":      1,
		"max_pending":     10,
		"max_wallets":     1,
		"base_fee":        500,
		"tip_fee":         50,
		"refill_amount":   "500000000000000000000",
		"refill_balance":  "200000000000000000000",
		"refill_interval": 300,
	}

	id, err := api.CreateSpammer("check-eoatx", spamoor.ScenarioEOATX, cfg, true)
	if err != nil {
		return fmt.Errorf("create spammer: %w", err)
	}
	log.Printf("created check spammer (id=%d)", id)

	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	sent, failed, err := waitForSpamoorDone(checkCtx, api, 1, baselineSent, baselineFailed)
	if err != nil {
		return fmt.Errorf("tx not confirmed: %w", err)
	}

	if failed > 0 {
		return fmt.Errorf("tx failed (sent=%.0f failed=%.0f)", sent, failed)
	}

	log.Printf("check passed: 1 tx sent successfully")
	return nil
}

type waitForDoneFunc func(context.Context, SpamoorClient, int, float64, float64) (float64, float64, error)

func runEntry(ctx context.Context, api SpamoorClient, entry Entry) error {
	return runEntryWithWait(ctx, api, entry, waitForSpamoorDone)
}

func runEntryWithWait(ctx context.Context, api SpamoorClient, entry Entry, wait waitForDoneFunc) error {
	totalCount := entry.NumSpammers * entry.CountPerSpammer

	log.Printf("[%s] scenario=%s spammers=%d count_per=%d total=%d",
		entry.TestName, entry.Scenario, entry.NumSpammers, entry.CountPerSpammer, totalCount)

	if err := DeleteAllSpammers(api); err != nil {
		return fmt.Errorf("delete stale spammers: %w", err)
	}
	defer func() {
		if err := DeleteAllSpammers(api); err != nil {
			log.Printf("[%s] warning: cleanup failed: %v", entry.TestName, err)
		}
	}()

	baselineSent, baselineFailed, err := getSpamoorCounters(api)
	if err != nil {
		return fmt.Errorf("get baseline metrics: %w", err)
	}

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
	sent, failed, err := wait(ctx, api, totalCount, baselineSent, baselineFailed)
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

// DeleteAllSpammers removes all active spammers from the spamoor daemon.
func DeleteAllSpammers(api SpamoorClient) error {
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

func waitForSpamoorDone(ctx context.Context, api SpamoorClient, targetCount int, baselineSent, baselineFailed float64) (sent, failed float64, err error) {
	return waitForSpamoorDoneWithInterval(ctx, api, targetCount, baselineSent, baselineFailed, 2*time.Second)
}

func waitForSpamoorDoneWithInterval(ctx context.Context, api SpamoorClient, targetCount int, baselineSent, baselineFailed float64, pollInterval time.Duration) (sent, failed float64, err error) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	start := time.Now()
	var prevSent float64

	for {
		select {
		case <-ctx.Done():
			return sent, failed, fmt.Errorf("timed out waiting for %d txs (sent %.0f): %w", targetCount, sent, ctx.Err())
		case <-ticker.C:
			currentSent, currentFailed, mErr := getSpamoorCounters(api)
			if mErr != nil {
				log.Printf("warning: failed to get metrics: %v", mErr)
				continue
			}

			sent = currentSent - baselineSent
			failed = currentFailed - baselineFailed
			if sent < 0 {
				sent = 0
			}
			if failed < 0 {
				failed = 0
			}

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

func getSpamoorCounters(api SpamoorClient) (sent, failed float64, err error) {
	metrics, err := api.GetMetrics()
	if err != nil {
		return 0, 0, err
	}
	return sumCounter(metrics["spamoor_transactions_sent_total"]),
		sumCounter(metrics["spamoor_transactions_failed_total"]),
		nil
}

// WaitForSync polls spamoor until its block processing rate stabilizes,
// indicating it has caught up to the chain tip. Returns immediately if
// ctx is cancelled.
func WaitForSync(ctx context.Context, api SpamoorClient) error {
	return waitForSync(ctx, api, 3*time.Second)
}

func waitForSync(ctx context.Context, api SpamoorClient, pollInterval time.Duration) error {
	log.Printf("waiting for spamoor to sync to chain tip...")

	const syncThreshold uint64 = 10

	var lastHeight uint64

	for {
		height, err := getSpamoorHeight(api)
		if err != nil {
			log.Printf("warning: %v", err)
		} else {
			if lastHeight > 0 && height > 0 {
				delta := height - lastHeight
				if delta < syncThreshold {
					log.Printf("spamoor synced at block %d (delta=%d in %s)", height, delta, pollInterval)
					return nil
				}
				log.Printf("syncing... block %d (+%d in %s)", height, delta, pollInterval)
			} else {
				log.Printf("syncing... block %d", height)
			}
			lastHeight = height
		}

		timer := time.NewTimer(pollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("cancelled waiting for sync: %w", ctx.Err())
		case <-timer.C:
		}
	}
}

func getSpamoorHeight(api SpamoorClient) (uint64, error) {
	clients, err := api.GetClients()
	if err != nil {
		return 0, fmt.Errorf("get clients: %w", err)
	}
	if len(clients) == 0 {
		return 0, fmt.Errorf("no RPC clients configured")
	}
	return clients[0].Height, nil
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
