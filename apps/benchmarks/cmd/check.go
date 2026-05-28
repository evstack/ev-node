package cmd

import (
	"time"

	"github.com/evstack/ev-node/apps/benchmarks/internal"
	"github.com/spf13/cobra"
)

func newCheckCmd() *cobra.Command {
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:   "check",
		Short: "verify connectivity by sending a single eoatx through spamoor to ev-reth",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return internal.RunCheck(resolveSpamoorURL(), timeout)
		},
	}

	cmd.Flags().DurationVar(&timeout, "timeout", 60*time.Second, "max time to wait for tx")

	return cmd
}
