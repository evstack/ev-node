package spamoor

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestURL(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Setenv("BENCH_SPAMOOR_URL", "")
		require.Equal(t, DefaultURL, URL())
	})

	t.Run("env override", func(t *testing.T) {
		t.Setenv("BENCH_SPAMOOR_URL", "http://localhost:9999")
		require.Equal(t, "http://localhost:9999", URL())
	})
}

func TestBuildScenarioConfig(t *testing.T) {
	cfg := BuildScenarioConfig(map[string]string{
		"BENCH_COUNT_PER_SPAMMER": "42",
		"BENCH_THROUGHPUT":        "100",
		"BENCH_REBROADCAST":       "true",
		"BENCH_BASE_FEE":          "500",
	})

	require.Equal(t, "500000000000000000000", cfg["refill_amount"])
	require.Equal(t, "200000000000000000000", cfg["refill_balance"])
	require.Equal(t, 300, cfg["refill_interval"])
	require.Equal(t, uint64(42), cfg["total_count"])
	require.Equal(t, uint64(100), cfg["throughput"])
	require.Equal(t, uint64(500), cfg["base_fee"])
	require.Equal(t, "true", cfg["rebroadcast"])
}
