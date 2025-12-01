package cmd

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	ds "github.com/ipfs/go-datastore"
	kt "github.com/ipfs/go-datastore/keytransform"
	"github.com/spf13/cobra"

	"github.com/evstack/ev-node/node"
	rollcmd "github.com/evstack/ev-node/pkg/cmd"
	"github.com/evstack/ev-node/pkg/store"
)

// daHeightInfo holds information about DA heights for a given block height
type daHeightInfo struct {
	Height         uint64
	HeaderDAHeight uint64
	DataDAHeight   uint64
	Missing        bool
}

// NewDAHeightsCmd creates a command to list the last X DA included heights
func NewDAHeightsCmd() *cobra.Command {
	var count uint64

	cmd := &cobra.Command{
		Use:   "da-heights",
		Short: "List the last X DA included heights and detect missing entries",
		Long: `Fetches and displays the last X included heights with their corresponding
DA (Data Availability) layer heights for both headers and data.
Shows which block heights are missing DA inclusion information.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			nodeConfig, err := rollcmd.ParseConfig(cmd)
			if err != nil {
				return err
			}

			goCtx := cmd.Context()
			if goCtx == nil {
				goCtx = context.Background()
			}

			// Open database
			rawDB, err := store.NewDefaultKVStore(nodeConfig.RootDir, nodeConfig.DBPath, "evm-single")
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer func() {
				if closeErr := rawDB.Close(); closeErr != nil {
					fmt.Printf("Warning: failed to close database: %v\n", closeErr)
				}
			}()

			// Create prefixed store
			prefixedDB := kt.Wrap(rawDB, &kt.PrefixTransform{
				Prefix: ds.NewKey(node.EvPrefix),
			})

			st := store.New(prefixedDB)

			// Fetch DA heights
			heights, err := fetchDAHeights(goCtx, st, count)
			if err != nil {
				return fmt.Errorf("failed to fetch DA heights: %w", err)
			}

			// Format and display output
			formatDAHeightsOutput(cmd.OutOrStdout(), heights)

			return nil
		},
	}

	cmd.Flags().Uint64VarP(&count, "count", "n", 10, "number of heights to fetch")

	return cmd
}

// fetchDAHeights retrieves the last N DA included heights from the store
func fetchDAHeights(ctx context.Context, st store.Store, count uint64) ([]daHeightInfo, error) {
	// Get the current DA included height
	daIncludedHeightBytes, err := st.GetMetadata(ctx, store.DAIncludedHeightKey)
	if err != nil {
		if errors.Is(err, ds.ErrNotFound) {
			return []daHeightInfo{}, nil
		}
		return nil, fmt.Errorf("failed to get DA included height: %w", err)
	}

	if len(daIncludedHeightBytes) != 8 {
		return []daHeightInfo{}, nil
	}

	daIncludedHeight := binary.LittleEndian.Uint64(daIncludedHeightBytes)
	if daIncludedHeight == 0 {
		return []daHeightInfo{}, nil
	}

	// Calculate the starting height (inclusive)
	startHeight := uint64(1)
	if daIncludedHeight > count {
		startHeight = daIncludedHeight - count + 1
	}

	// Collect height information
	var results []daHeightInfo

	for height := daIncludedHeight; height >= startHeight; height-- {
		info := daHeightInfo{
			Height: height,
		}

		// Try to get header DA height
		headerKey := store.GetHeightToDAHeightHeaderKey(height)
		headerBytes, headerErr := st.GetMetadata(ctx, headerKey)

		// Try to get data DA height
		dataKey := store.GetHeightToDAHeightDataKey(height)
		dataBytes, dataErr := st.GetMetadata(ctx, dataKey)

		// Check if either is missing or has invalid length
		headerMissing := headerErr != nil || len(headerBytes) != 8
		dataMissing := dataErr != nil || len(dataBytes) != 8

		if headerMissing || dataMissing {
			info.Missing = true
		} else {
			info.HeaderDAHeight = binary.LittleEndian.Uint64(headerBytes)
			info.DataDAHeight = binary.LittleEndian.Uint64(dataBytes)
		}

		results = append(results, info)

		if height == 1 {
			break
		}
	}

	return results, nil
}

// formatDAHeightsOutput formats and writes the DA heights information to the output
func formatDAHeightsOutput(w io.Writer, heights []daHeightInfo) {
	if len(heights) == 0 {
		fmt.Fprintln(w, "No DA included heights found")
		return
	}

	// Print header
	fmt.Fprintln(w, "Block Height | Header DA Height | Data DA Height | Status")
	fmt.Fprintln(w, "------------ | ---------------- | -------------- | ------")

	// Print each height
	for _, h := range heights {
		if h.Missing {
			fmt.Fprintf(w, "%12d | %16s | %14s | MISSING\n", h.Height, "-", "-")
		} else {
			fmt.Fprintf(w, "%12d | %16d | %14d | OK\n", h.Height, h.HeaderDAHeight, h.DataDAHeight)
		}
	}
}
