package cmd

import (
	"github.com/evstack/ev-node/apps/benchmarks/internal"
	"github.com/spf13/cobra"
)

const defaultBaselinePath = "/root/baseline.json"

func newRegularCmd() *cobra.Command {
	var matrixPath string

	cmd := &cobra.Command{
		Use:   "regular",
		Short: "run sustained load from baseline matrix (~1M tx/day at @hourly)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return internal.ExecuteMatrix(matrixPath, resolveSpamoorURL())
		},
	}

	cmd.Flags().StringVar(&matrixPath, "matrix", defaultBaselinePath, "path to baseline matrix JSON")

	return cmd
}
