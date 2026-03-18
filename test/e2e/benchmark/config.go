//go:build evm

package benchmark

import (
	"os"
	"strconv"
	"testing"
	"time"
)

// benchConfig holds all tunable parameters for a benchmark run.
// fields are populated from BENCH_* env vars with sensible defaults.
type benchConfig struct {
	ServiceName string

	// infrastructure (used by setupLocalEnv)
	BlockTime      string
	SlotDuration   string
	GasLimit       string
	ScrapeInterval string

	// load generation (used by test functions)
	NumSpammers     int
	CountPerSpammer int
	Throughput      int
	WarmupTxs       int
	GasUnitsToBurn  int
	MaxWallets      int
	WaitTimeout     time.Duration
}

func newBenchConfig(serviceName string) benchConfig {
	return benchConfig{
		ServiceName:     serviceName,
		BlockTime:       envOrDefault("BENCH_BLOCK_TIME", "100ms"),
		SlotDuration:    envOrDefault("BENCH_SLOT_DURATION", "250ms"),
		GasLimit:        envOrDefault("BENCH_GAS_LIMIT", ""),
		ScrapeInterval:  envOrDefault("BENCH_SCRAPE_INTERVAL", "1s"),
		NumSpammers:     envInt("BENCH_NUM_SPAMMERS", 2),
		CountPerSpammer: envInt("BENCH_COUNT_PER_SPAMMER", 2000),
		Throughput:      envInt("BENCH_THROUGHPUT", 200),
		WarmupTxs:       envInt("BENCH_WARMUP_TXS", 200),
		GasUnitsToBurn:  envInt("BENCH_GAS_UNITS_TO_BURN", 1_000_000),
		MaxWallets:      envInt("BENCH_MAX_WALLETS", 500),
		WaitTimeout:     envDuration("BENCH_WAIT_TIMEOUT", 10*time.Minute),
	}
}

func (c benchConfig) totalCount() int {
	return c.NumSpammers * c.CountPerSpammer
}

func (c benchConfig) log(t testing.TB) {
	t.Logf("load: spammers=%d, count_per=%d, throughput=%d, warmup=%d, gas_units=%d, max_wallets=%d",
		c.NumSpammers, c.CountPerSpammer, c.Throughput, c.WarmupTxs, c.GasUnitsToBurn, c.MaxWallets)
	t.Logf("infra: block_time=%s, slot_duration=%s, gas_limit=%s, scrape_interval=%s",
		c.BlockTime, c.SlotDuration, c.GasLimit, c.ScrapeInterval)
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// envInt returns the integer value of the given env var, or fallback if unset
// or unparseable. Invalid values silently fall back to the default.
func envInt(key string, fallback int) int {
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

// envDuration returns the duration value of the given env var (e.g. "5m", "30s"),
// or fallback if unset or unparseable.
func envDuration(key string, fallback time.Duration) time.Duration {
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
