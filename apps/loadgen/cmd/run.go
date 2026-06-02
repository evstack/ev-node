package cmd

import (
	"github.com/evstack/ev-node/apps/loadgen/internal"
	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run <matrix.json>",
		Short: "run benchmarks from a matrix JSON file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return internal.ExecuteMatrixFromFile(cmd.Context(), args[0], internal.NewSpamoorClient(resolveSpamoorURL()))
		},
	}
}
