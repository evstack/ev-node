package cmd

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/evstack/ev-node/apps/loadgen/internal"
	"github.com/spf13/cobra"
)

const (
	defaultBaselinePath = "/home/ev/baseline.json"
	defaultBurstPath    = "/home/ev/burst.json"
	burstWindow         = 24 * time.Hour
)

type startConfig struct {
	txPerDay      int
	interval      time.Duration
	burstTxCount  int
	burstPerDay   int
	regularMatrix string
	burstMatrix   string
}

func newStartCmd() *cobra.Command {
	cfg := startConfig{}

	cmd := &cobra.Command{
		Use:   "start",
		Short: "run continuous benchmark scheduler (regular + burst workloads)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScheduler(cmd.Context(), cfg)
		},
	}

	cmd.Flags().IntVar(&cfg.txPerDay, "tx-per-day", envIntOr("BENCH_TX_PER_DAY", 1000000), "sustained txs/day")
	cmd.Flags().DurationVar(&cfg.interval, "interval", envDurationOr("BENCH_INTERVAL", time.Hour), "regular workload frequency")
	cmd.Flags().IntVar(&cfg.burstTxCount, "burst-tx-count", envIntOr("BENCH_BURST_TX_COUNT", 500000), "txs per burst")
	cmd.Flags().IntVar(&cfg.burstPerDay, "burst-per-day", envIntOr("BENCH_BURST_PER_DAY", 0), "bursts per day, randomly spaced (0 = no bursts)")
	cmd.Flags().StringVar(&cfg.regularMatrix, "regular-matrix", envStringOr("BENCH_REGULAR_MATRIX", defaultBaselinePath), "path to regular matrix JSON")
	cmd.Flags().StringVar(&cfg.burstMatrix, "burst-matrix", envStringOr("BENCH_BURST_MATRIX", defaultBurstPath), "path to burst matrix JSON")

	return cmd
}

func runScheduler(parent context.Context, cfg startConfig) error {
	if err := validateStartConfig(cfg); err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(parent, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	spamoorAddr := resolveSpamoorURL()
	api := internal.NewSpamoorClient(spamoorAddr)

	runsPerDay := float64(24*time.Hour) / float64(cfg.interval)
	regularTxPerRun := int(float64(cfg.txPerDay) / runsPerDay)

	log.Printf("scheduler config: tx-per-day=%d interval=%s regular-tx-per-run=%d burst-tx-count=%d burst-per-day=%d",
		cfg.txPerDay, cfg.interval, regularTxPerRun, cfg.burstTxCount, cfg.burstPerDay)
	log.Printf("regular-matrix=%s burst-matrix=%s spamoor=%s", cfg.regularMatrix, cfg.burstMatrix, spamoorAddr)

	if err := internal.WaitForSync(ctx, api); err != nil {
		return err
	}

	var wg sync.WaitGroup
	runWorkload := func(label, matrixPath string, txCount int) {
		defer wg.Done()
		log.Printf("==> %s workload starting (%d tx)", label, txCount)
		if err := internal.ExecuteMatrixWithOverridesFromFile(ctx, matrixPath, api, txCount); err != nil {
			log.Printf("%s workload error: %v", label, err)
		}
	}

	// fire regular immediately
	wg.Add(1)
	go runWorkload("regular", cfg.regularMatrix, regularTxPerRun)

	ticker := time.NewTicker(cfg.interval)
	defer ticker.Stop()

	// burst: single timer, reschedule after each fire, reset count every 24h.
	burstsRemaining := cfg.burstPerDay
	nextReset := time.Now().Add(burstWindow)
	burstTimer := nextBurstTimer(burstsRemaining, time.Until(nextReset))
	resetTimer := time.NewTimer(burstWindow)
	defer resetTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("shutting down, waiting for in-flight workloads...")
			burstTimer.Stop()
			wg.Wait()
			log.Printf("cleaning up spammers")
			if err := internal.DeleteAllSpammers(api); err != nil {
				log.Printf("warning: shutdown cleanup failed: %v", err)
			}
			return nil

		case <-ticker.C:
			wg.Add(1)
			go runWorkload("regular", cfg.regularMatrix, regularTxPerRun)

		case <-burstTimer.C:
			wg.Add(1)
			go runWorkload("burst", cfg.burstMatrix, cfg.burstTxCount)
			burstsRemaining--
			burstTimer = nextBurstTimer(burstsRemaining, time.Until(nextReset))

		case <-resetTimer.C:
			log.Printf("24h elapsed - resetting burst count")
			burstTimer.Stop()
			burstsRemaining = cfg.burstPerDay
			nextReset = time.Now().Add(burstWindow)
			burstTimer = nextBurstTimer(burstsRemaining, time.Until(nextReset))
			resetTimer.Reset(burstWindow)
		}
	}
}

func validateStartConfig(cfg startConfig) error {
	if cfg.interval <= 0 {
		return fmt.Errorf("interval must be > 0")
	}
	if cfg.txPerDay < 0 {
		return fmt.Errorf("tx-per-day must be >= 0")
	}
	if cfg.burstTxCount < 0 {
		return fmt.Errorf("burst-tx-count must be >= 0")
	}
	if cfg.burstPerDay < 0 {
		return fmt.Errorf("burst-per-day must be >= 0")
	}
	return nil
}

// nextBurstTimer returns a timer for the next burst. If no bursts remain,
// returns a stopped timer (channel never fires).
func nextBurstTimer(remaining int, window time.Duration) *time.Timer {
	if remaining <= 0 {
		t := time.NewTimer(0)
		t.Stop()
		// drain channel in case it fired before Stop
		select {
		case <-t.C:
		default:
		}
		return t
	}
	delay := time.Duration(rand.Int64N(int64(window) / int64(remaining)))
	log.Printf("next burst in %s (%d remaining)", delay.Round(time.Second), remaining)
	return time.NewTimer(delay)
}

func envIntOr(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func envDurationOr(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func envStringOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
