package cmd

import (
	"github.com/evstack/ev-node/apps/benchmarks/internal"
	"github.com/spf13/cobra"
)

const defaultBurstPath = "/root/burst.json"

func newBurstCmd() *cobra.Command {
	var matrixPath string

	cmd := &cobra.Command{
		Use:   "burst",
		Short: "run probabilistic burst load (500K tx, ~15% chance per invocation)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return internal.ExecuteMatrix(matrixPath, resolveSpamoorURL())
		},
	}

	cmd.Flags().StringVar(&matrixPath, "matrix", defaultBurstPath, "path to burst matrix JSON")

	return cmd
}
