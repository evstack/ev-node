package internal

import (
	"os"
	"strconv"
)

const DefaultSpamoorURL = "http://spamoor-daemon:8080"

func SpamoorURL() string {
	if v := os.Getenv("BENCH_SPAMOOR_URL"); v != "" {
		return v
	}
	return DefaultSpamoorURL
}

var envMapping = map[string]string{
	"BENCH_COUNT_PER_SPAMMER": "total_count",
	"BENCH_THROUGHPUT":        "throughput",
	"BENCH_MAX_PENDING":       "max_pending",
	"BENCH_MAX_WALLETS":       "max_wallets",
	"BENCH_GAS_UNITS_TO_BURN": "gas_units_to_burn",
	"BENCH_BASE_FEE":          "base_fee",
	"BENCH_TIP_FEE":           "tip_fee",
	"BENCH_REBROADCAST":       "rebroadcast",
}

func BuildScenarioConfig(env map[string]string) map[string]any {
	cfg := map[string]any{
		"refill_amount":   "500000000000000000000",
		"refill_balance":  "200000000000000000000",
		"refill_interval": 300,
	}

	for envKey, cfgKey := range envMapping {
		val, ok := env[envKey]
		if !ok {
			continue
		}
		if n, err := strconv.Atoi(val); err == nil {
			cfg[cfgKey] = n
		} else {
			cfg[cfgKey] = val
		}
	}

	return cfg
}
