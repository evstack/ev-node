package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"

	coreda "github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/da/jsonrpc"
	"github.com/evstack/ev-node/types"
	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

var (
	daURL         string
	authToken     string
	timeout       time.Duration
	verbose       bool
	maxBlobSize   uint64
	gasPrice      float64
	gasMultiplier float64
	filterHeight  uint64
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "da-debug",
		Short: "DA debugging tool for blockchain data inspection",
		Long: `DA Debug Tool
A powerful DA debugging tool for inspecting blockchain data availability layers.`,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&daURL, "da-url", "http://localhost:7980", "DA layer JSON-RPC URL")
	rootCmd.PersistentFlags().StringVar(&authToken, "auth-token", "", "Authentication token for DA layer")
	rootCmd.PersistentFlags().DurationVar(&timeout, "timeout", 30*time.Second, "Request timeout")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	rootCmd.PersistentFlags().Uint64Var(&maxBlobSize, "max-blob-size", 1970176, "Maximum blob size in bytes")
	rootCmd.PersistentFlags().Float64Var(&gasPrice, "gas-price", 0.0, "Gas price for DA operations")
	rootCmd.PersistentFlags().Float64Var(&gasMultiplier, "gas-multiplier", 1.0, "Gas multiplier for DA operations")

	// Add subcommands
	rootCmd.AddCommand(queryCmd())
	rootCmd.AddCommand(searchCmd())

	if err := rootCmd.Execute(); err != nil {
		printError("Error: %v\n", err)
		os.Exit(1)
	}
}

func queryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query <height> <namespace>",
		Short: "Query and decode blobs at a specific DA height and namespace",
		Long: `Query and decode blobs at a specific DA height and namespace.
Decodes each blob as either header or data and displays detailed information.`,
		Args: cobra.ExactArgs(2),
		RunE: runQuery,
	}

	cmd.Flags().Uint64Var(&filterHeight, "filter-height", 0, "Filter blobs by specific height (0 = no filter)")

	return cmd
}

func searchCmd() *cobra.Command {
	var searchHeight uint64
	var searchRange uint64

	cmd := &cobra.Command{
		Use:   "search <start-da-height> <namespace> --target-height <height>",
		Short: "Search for blobs containing a specific blockchain height",
		Long: `Search through multiple DA heights to find blobs containing data from a specific blockchain height.
Starting from the given DA height, searches through a range of DA heights until it finds matching blobs.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSearch(cmd, args, searchHeight, searchRange)
		},
	}

	cmd.Flags().Uint64Var(&searchHeight, "target-height", 0, "Target blockchain height to search for (required)")
	cmd.Flags().Uint64Var(&searchRange, "range", 10, "Number of DA heights to search")
	cmd.MarkFlagRequired("target-height")

	return cmd
}

func runQuery(cmd *cobra.Command, args []string) error {
	height, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid height: %w", err)
	}

	namespace, err := parseNamespace(args[1])
	if err != nil {
		return fmt.Errorf("invalid namespace: %w", err)
	}

	printBanner()
	printQueryInfo(height, namespace)

	client, err := createDAClient()
	if err != nil {
		return err
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return queryHeight(ctx, client, height, namespace)
}

func runSearch(cmd *cobra.Command, args []string, searchHeight, searchRange uint64) error {
	startHeight, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid start height: %w", err)
	}

	namespace, err := parseNamespace(args[1])
	if err != nil {
		return fmt.Errorf("invalid namespace: %w", err)
	}

	printBanner()
	printSearchInfo(startHeight, namespace, searchHeight, searchRange)

	client, err := createDAClient()
	if err != nil {
		return err
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return searchForHeight(ctx, client, startHeight, namespace, searchHeight, searchRange)
}

func searchForHeight(ctx context.Context, client *jsonrpc.Client, startHeight uint64, namespace []byte, targetHeight, searchRange uint64) error {
	fmt.Printf("Searching for height %d in DA heights %d-%d...\n", targetHeight, startHeight, startHeight+searchRange-1)
	fmt.Println()

	foundBlobs := 0
	for daHeight := startHeight; daHeight < startHeight+searchRange; daHeight++ {
		result, err := client.DA.GetIDs(ctx, daHeight, namespace)
		if err != nil {
			if err.Error() == "blob: not found" || strings.Contains(err.Error(), "blob: not found") {
				continue
			}
			if strings.Contains(err.Error(), "height") && strings.Contains(err.Error(), "future") {
				fmt.Printf("Reached future height at DA height %d\n", daHeight)
				break
			}
			continue
		}

		if result == nil || len(result.IDs) == 0 {
			continue
		}

		// Get the actual blob data
		blobs, err := client.DA.Get(ctx, result.IDs, namespace)
		if err != nil {
			continue
		}

		// Check each blob for the target height
		for i, blob := range blobs {
			found := false
			var blobHeight uint64

			// Try to decode as header first
			if header := tryDecodeHeader(blob); header != nil {
				blobHeight = header.Height()
				if blobHeight == targetHeight {
					found = true
				}
			} else if data := tryDecodeData(blob); data != nil {
				if data.Metadata != nil {
					blobHeight = data.Height()
					if blobHeight == targetHeight {
						found = true
					}
				}
			}

			if found {
				foundBlobs++
				fmt.Printf("FOUND at DA Height %d - BLOB %d\n", daHeight, foundBlobs)
				fmt.Println(strings.Repeat("-", 80))
				displayBlobInfo(result.IDs[i], blob)

				// Display the decoded content
				if header := tryDecodeHeader(blob); header != nil {
					printTypeHeader("SignedHeader", "")
					displayHeader(header)
				} else if data := tryDecodeData(blob); data != nil {
					printTypeHeader("SignedData", "")
					displayData(data)
				}

				fmt.Println()
			}
		}
	}

	fmt.Println(strings.Repeat("=", 50))
	if foundBlobs == 0 {
		fmt.Printf("No blobs found containing height %d in DA range %d-%d\n", targetHeight, startHeight, startHeight+searchRange-1)
	} else {
		fmt.Printf("Found %d blob(s) containing height %d\n", foundBlobs, targetHeight)
	}

	return nil
}

func queryHeight(ctx context.Context, client *jsonrpc.Client, height uint64, namespace []byte) error {
	result, err := client.DA.GetIDs(ctx, height, namespace)
	if err != nil {
		// Handle "blob not found" as a normal case
		if err.Error() == "blob: not found" || strings.Contains(err.Error(), "blob: not found") {
			fmt.Printf("No blobs found at height %d\n", height)
			return nil
		}
		// Handle future height errors gracefully
		if strings.Contains(err.Error(), "height") && strings.Contains(err.Error(), "future") {
			fmt.Printf("Height %d is in the future (not yet available)\n", height)
			return nil
		}
		return fmt.Errorf("failed to get IDs: %w", err)
	}

	if result == nil || len(result.IDs) == 0 {
		fmt.Printf("No blobs found at height %d\n", height)
		return nil
	}

	fmt.Printf("Found %d blob(s) at height %d\n", len(result.IDs), height)
	fmt.Printf("Timestamp: %s\n", result.Timestamp.Format(time.RFC3339))
	fmt.Println()

	// Get the actual blob data
	blobs, err := client.DA.Get(ctx, result.IDs, namespace)
	if err != nil {
		return fmt.Errorf("failed to get blob data: %w", err)
	}

	// Process each blob with optional height filtering
	displayedBlobs := 0
	for i, blob := range blobs {
		shouldDisplay := true
		var blobHeight uint64

		// Check if we need to filter by height
		if filterHeight > 0 {
			shouldDisplay = false

			// Try to decode as header first to check height
			if header := tryDecodeHeader(blob); header != nil {
				blobHeight = header.Height()
				if blobHeight == filterHeight {
					shouldDisplay = true
				}
			} else if data := tryDecodeData(blob); data != nil {
				if data.Metadata != nil {
					blobHeight = data.Height()
					if blobHeight == filterHeight {
						shouldDisplay = true
					}
				}
			}
		}

		if !shouldDisplay {
			continue
		}

		displayedBlobs++
		printBlobHeader(displayedBlobs, -1) // -1 indicates filtered mode
		displayBlobInfo(result.IDs[i], blob)

		// Try to decode as header first
		if header := tryDecodeHeader(blob); header != nil {
			printTypeHeader("SignedHeader", "")
			displayHeader(header)
		} else if data := tryDecodeData(blob); data != nil {
			printTypeHeader("SignedData", "")
			displayData(data)
		} else {
			printTypeHeader("Raw Data", "")
			displayRawData(blob)
		}

		if displayedBlobs > 1 {
			printSeparator()
		}
	}

	// Show filter results
	if filterHeight > 0 {
		if displayedBlobs == 0 {
			fmt.Printf("No blobs found matching height filter: %d\n", filterHeight)
		} else {
			fmt.Printf("Showing %d blob(s) matching height filter: %d\n", displayedBlobs, filterHeight)
		}
	}

	printFooter()
	return nil
}

func printBanner() {
	fmt.Println("DA Debug Tool - Blockchain Data Inspector")
	fmt.Println(strings.Repeat("=", 50))
}

func printQueryInfo(height uint64, namespace []byte) {
	fmt.Printf("DA Height: %d | Namespace: %s | URL: %s", height, formatHash(hex.EncodeToString(namespace)), daURL)
	if filterHeight > 0 {
		fmt.Printf(" | Filter Height: %d", filterHeight)
	}
	fmt.Println()
	fmt.Println()
}

func printSearchInfo(startHeight uint64, namespace []byte, targetHeight, searchRange uint64) {
	fmt.Printf("Start DA Height: %d | Namespace: %s | URL: %s", startHeight, formatHash(hex.EncodeToString(namespace)), daURL)
	fmt.Printf(" | Target Height: %d | Range: %d", targetHeight, searchRange)
	fmt.Println()
	fmt.Println()
}

func printBlobHeader(current, total int) {
	if total == -1 {
		fmt.Printf("BLOB %d\n", current)
	} else {
		fmt.Printf("BLOB %d/%d\n", current, total)
	}
	fmt.Println(strings.Repeat("-", 80))
}

func displayBlobInfo(id coreda.ID, blob []byte) {
	fmt.Printf("ID:           %s\n", formatHash(hex.EncodeToString(id)))
	fmt.Printf("Size:         %s\n", formatSize(len(blob)))

	// Try to parse the ID to show height and commitment
	if idHeight, commitment, err := coreda.SplitID(id); err == nil {
		fmt.Printf("ID Height:    %d\n", idHeight)
		fmt.Printf("Commitment:   %s\n", formatHash(hex.EncodeToString(commitment)))
	}
}

func printTypeHeader(title, color string) {
	fmt.Printf("Type:         %s\n", title)
}

func displayHeader(header *types.SignedHeader) {
	fmt.Printf("Height:       %d\n", header.Height())
	fmt.Printf("Time:         %s\n", header.Time().Format(time.RFC3339))
	fmt.Printf("Chain ID:     %s\n", header.ChainID())
	fmt.Printf("Version:      Block=%d, App=%d\n", header.Version.Block, header.Version.App)
	fmt.Printf("Last Header:  %s\n", formatHashField(hex.EncodeToString(header.LastHeaderHash[:])))
	fmt.Printf("Last Commit:  %s\n", formatHashField(hex.EncodeToString(header.LastCommitHash[:])))
	fmt.Printf("Data Hash:    %s\n", formatHashField(hex.EncodeToString(header.DataHash[:])))
	fmt.Printf("Consensus:    %s\n", formatHashField(hex.EncodeToString(header.ConsensusHash[:])))
	fmt.Printf("App Hash:     %s\n", formatHashField(hex.EncodeToString(header.AppHash[:])))
	fmt.Printf("Last Results: %s\n", formatHashField(hex.EncodeToString(header.LastResultsHash[:])))
	fmt.Printf("Validator:    %s\n", formatHashField(hex.EncodeToString(header.ValidatorHash[:])))
	fmt.Printf("Proposer:     %s\n", formatHashField(hex.EncodeToString(header.ProposerAddress)))
	fmt.Printf("Signature:    %s\n", formatHashField(hex.EncodeToString(header.Signature)))
	if len(header.Signer.Address) > 0 {
		fmt.Printf("Signer:       %s\n", formatHashField(hex.EncodeToString(header.Signer.Address)))
	}
}

func displayData(data *types.SignedData) {
	if data.Metadata != nil {
		fmt.Printf("Chain ID:     %s\n", data.ChainID())
		fmt.Printf("Height:       %d\n", data.Height())
		fmt.Printf("Time:         %s\n", data.Time().Format(time.RFC3339))
		fmt.Printf("Last Data:    %s\n", formatHashField(hex.EncodeToString(data.LastDataHash[:])))
	}

	dataHash := data.DACommitment()
	fmt.Printf("DA Commit:    %s\n", formatHashField(hex.EncodeToString(dataHash[:])))
	fmt.Printf("TX Count:     %d\n", len(data.Txs))
	fmt.Printf("Signature:    %s\n", formatHashField(hex.EncodeToString(data.Signature)))

	if len(data.Signer.Address) > 0 {
		fmt.Printf("Signer:       %s\n", formatHashField(hex.EncodeToString(data.Signer.Address)))
	}

	// Display transactions
	if len(data.Txs) > 0 {
		fmt.Printf("\nTransactions:\n")
		for i, tx := range data.Txs {
			fmt.Printf("  [%d] Size: %s, Hash: %s\n",
				i+1,
				formatSize(len(tx)),
				formatShortHash(hex.EncodeToString(tx)))

			if isPrintable(tx) && len(tx) < 200 {
				preview := string(tx)
				if len(preview) > 60 {
					preview = preview[:60] + "..."
				}
				fmt.Printf("       Data: %s\n", preview)
			}
		}
	}
}

func displayRawData(blob []byte) {
	hexStr := hex.EncodeToString(blob)
	if len(hexStr) > 120 {
		fmt.Printf("Hex:          %s...\n", hexStr[:120])
		fmt.Printf("Full Length:  %s\n", formatSize(len(blob)))
	} else {
		fmt.Printf("Hex:          %s\n", hexStr)
	}

	if isPrintable(blob) {
		strData := string(blob)
		if len(strData) > 200 {
			fmt.Printf("String:       %s...\n", strData[:200])
		} else {
			fmt.Printf("String:       %s\n", strData)
		}
	} else {
		fmt.Printf("String:       (Binary data - not printable)\n")
	}
}

// Helper functions for formatting

func formatHash(hash string) string {
	return hash
}

func formatHashField(hash string) string {
	return hash
}

func formatShortHash(hash string) string {
	return hash
}

func formatSize(bytes int) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	} else {
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	}
}

func printSeparator() {
	fmt.Println()
}

func printFooter() {
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("Analysis complete!\n")
}

func printError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format, args...)
}

func tryDecodeHeader(bz []byte) *types.SignedHeader {
	header := new(types.SignedHeader)
	var headerPb pb.SignedHeader

	if err := proto.Unmarshal(bz, &headerPb); err != nil {
		return nil
	}

	if err := header.FromProto(&headerPb); err != nil {
		return nil
	}

	// Basic validation
	if err := header.Header.ValidateBasic(); err != nil {
		return nil
	}

	return header
}

func tryDecodeData(bz []byte) *types.SignedData {
	var signedData types.SignedData
	if err := signedData.UnmarshalBinary(bz); err != nil {
		return nil
	}

	// Skip completely empty data
	if len(signedData.Txs) == 0 && len(signedData.Signature) == 0 {
		return nil
	}

	return &signedData
}

func createDAClient() (*jsonrpc.Client, error) {
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).Level(zerolog.InfoLevel)
	if verbose {
		logger = logger.Level(zerolog.DebugLevel)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client, err := jsonrpc.NewClient(ctx, logger, daURL, authToken, gasPrice, gasMultiplier, maxBlobSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create DA client: %w", err)
	}

	return client, nil
}

func parseNamespace(ns string) ([]byte, error) {
	// Try to parse as hex first
	if hex, err := parseHex(ns); err == nil && len(hex) == 29 {
		return hex, nil
	}

	// If not valid hex or not 29 bytes, treat as string identifier
	namespace := coreda.NamespaceFromString(ns)
	return namespace.Bytes(), nil
}

func parseHex(s string) ([]byte, error) {
	// Remove 0x prefix if present
	if len(s) >= 2 && s[:2] == "0x" {
		s = s[2:]
	}

	return hex.DecodeString(s)
}

func isPrintable(data []byte) bool {
	if len(data) > 1000 { // Only check first 1000 bytes for performance
		data = data[:1000]
	}

	for _, b := range data {
		if b < 32 || b > 126 {
			if b != '\n' && b != '\r' && b != '\t' {
				return false
			}
		}
	}
	return true
}
