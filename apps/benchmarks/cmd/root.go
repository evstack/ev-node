package cmd

import (
	"github.com/evstack/ev-node/apps/benchmarks/internal"
	"github.com/spf13/cobra"
)

var spamoorFlag string

// NewRootCmd returns the top-level cobra command for ev-benchmarks.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "ev-benchmarks",
		Short: "benchmark runner for ev-node stress testing via spamoor",
	}

	rootCmd.PersistentFlags().StringVar(&spamoorFlag, "spamoor-url", "", "spamoor-daemon API URL (env: BENCH_SPAMOOR_URL)")

	rootCmd.AddCommand(newRunCmd(), newStartCmd(), newCheckCmd())

	return rootCmd
}

func resolveSpamoorURL() string {
	if spamoorFlag != "" {
		return spamoorFlag
	}
	return internal.SpamoorURL()
}
