package cmd

import (
	"log"

	"github.com/evstack/ev-node/apps/loadgen/internal"
	"github.com/spf13/cobra"
)

func newBurstCmd() *cobra.Command {
	var (
		txCount    int
		matrixPath string
	)

	cmd := &cobra.Command{
		Use:   "burst",
		Short: "trigger a single burst workload immediately",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			spamoorAddr := resolveSpamoorURL()
			api := internal.NewSpamoorClient(spamoorAddr)

			if err := internal.WaitForSync(cmd.Context(), api); err != nil {
				return err
			}

			log.Printf("==> burst workload starting (%d tx)", txCount)
			return internal.ExecuteMatrixWithOverridesFromFile(cmd.Context(), matrixPath, api, txCount)
		},
	}

	cmd.Flags().IntVar(&txCount, "tx-count", envIntOr("BENCH_BURST_TX_COUNT", 500000), "total transactions for the burst")
	cmd.Flags().StringVar(&matrixPath, "burst-matrix", envStringOr("BENCH_BURST_MATRIX", defaultBurstPath), "path to burst matrix JSON")

	return cmd
}
