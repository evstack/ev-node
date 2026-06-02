package cmd

import (
	"github.com/evstack/ev-node/apps/loadgen/internal"
	"github.com/spf13/cobra"
)

var spamoorFlag string

// NewRootCmd returns the top-level cobra command for ev-loadgen.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "ev-loadgen",
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
