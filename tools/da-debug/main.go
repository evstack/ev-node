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

// ANSI color codes
const (
	ColorReset   = "\033[0m"
	ColorRed     = "\033[31m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorBlue    = "\033[34m"
	ColorMagenta = "\033[35m"
	ColorCyan    = "\033[36m"
	ColorWhite   = "\033[37m"
	ColorBold    = "\033[1m"
	ColorDim     = "\033[2m"
	ColorUnder   = "\033[4m"

	// Background colors
	ColorBgGreen  = "\033[42m"
	ColorBgBlue   = "\033[44m"
	ColorBgYellow = "\033[43m"
)

var (
	daURL         string
	authToken     string
	timeout       time.Duration
	verbose       bool
	maxBlobSize   uint64
	gasPrice      float64
	gasMultiplier float64
	noColor       bool
	filterHeight  uint64
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "da-debug <height> <namespace>",
		Short: "DA debugging tool - decode blobs at given height and namespace",
		Long: fmt.Sprintf(`%s%sDA Debug Tool%s
%sA powerful DA debugging tool that queries a specific height and namespace,
then decodes each blob as either header or data and displays detailed information.%s`,
			ColorBold, ColorCyan, ColorReset,
			ColorDim, ColorReset),
		Args: cobra.ExactArgs(2),
		RunE: runDebug,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&daURL, "da-url", "http://localhost:7980", "DA layer JSON-RPC URL")
	rootCmd.PersistentFlags().StringVar(&authToken, "auth-token", "", "Authentication token for DA layer")
	rootCmd.PersistentFlags().DurationVar(&timeout, "timeout", 30*time.Second, "Request timeout")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	rootCmd.PersistentFlags().Uint64Var(&maxBlobSize, "max-blob-size", 1970176, "Maximum blob size in bytes")
	rootCmd.PersistentFlags().Float64Var(&gasPrice, "gas-price", 0.0, "Gas price for DA operations")
	rootCmd.PersistentFlags().Float64Var(&gasMultiplier, "gas-multiplier", 1.0, "Gas multiplier for DA operations")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().Uint64Var(&filterHeight, "filter-height", 0, "Filter blobs by specific height (0 = no filter)")

	if err := rootCmd.Execute(); err != nil {
		printError("Error: %v\n", err)
		os.Exit(1)
	}
}

func runDebug(cmd *cobra.Command, args []string) error {
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

	result, err := client.DA.GetIDs(ctx, height, namespace)
	if err != nil {
		// Handle "blob not found" as a normal case
		if err.Error() == "blob: not found" || strings.Contains(err.Error(), "blob: not found") {
			printWarning("No blobs found at height %d", height)
			return nil
		}
		// Handle future height errors gracefully
		if strings.Contains(err.Error(), "height") && strings.Contains(err.Error(), "future") {
			printWarning("Height %d is in the future (not yet available)", height)
			return nil
		}
		return fmt.Errorf("failed to get IDs: %w", err)
	}

	if result == nil || len(result.IDs) == 0 {
		printWarning("No blobs found at height %d", height)
		return nil
	}

	printSuccess("Found %d blob(s) at height %d", len(result.IDs), height)
	printInfo("Timestamp: %s", result.Timestamp.Format(time.RFC3339))
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
			printTypeHeader("SignedHeader", ColorBlue)
			displayHeader(header)
		} else if data := tryDecodeData(blob); data != nil {
			printTypeHeader("SignedData", ColorGreen)
			displayData(data)
		} else {
			printTypeHeader("Raw Data", ColorYellow)
			displayRawData(blob)
		}

		if displayedBlobs > 1 {
			printSeparator()
		}
	}

	// Show filter results
	if filterHeight > 0 {
		if displayedBlobs == 0 {
			printWarning("No blobs found matching height filter: %d", filterHeight)
		} else {
			printInfo("Showing %d blob(s) matching height filter: %d", displayedBlobs, filterHeight)
		}
	}

	printFooter()
	return nil
}

func printBanner() {
	banner := fmt.Sprintf(`%s%s
╔══════════════════════════════════════╗
║          DA Debug Tool               ║
║       Blockchain Data Inspector      ║
╚══════════════════════════════════════╝%s
`, ColorBold, ColorCyan, ColorReset)
	fmt.Print(banner)
}

func printQueryInfo(height uint64, namespace []byte) {
	fmt.Printf("%s%sQuery Details:%s\n", ColorBold, ColorWhite, ColorReset)
	fmt.Printf("   DA Height: %s%d%s\n", colorize(ColorYellow), height, ColorReset)
	fmt.Printf("   Namespace: %s%s%s\n", colorize(ColorMagenta), formatHash(hex.EncodeToString(namespace)), ColorReset)
	fmt.Printf("   DA URL: %s%s%s\n", colorize(ColorCyan), daURL, ColorReset)
	if filterHeight > 0 {
		fmt.Printf("   Height Filter: %s%d%s\n", colorize(ColorGreen), filterHeight, ColorReset)
	}
	fmt.Println()
}

func printBlobHeader(current, total int) {
	if total == -1 {
		// Filtered mode - don't show total
		fmt.Printf("%s%s┌─ Blob %d %s\n", ColorBold, ColorBlue, current, strings.Repeat("─", 55-len(fmt.Sprintf("┌─ Blob %d ", current)))+ColorReset)
	} else {
		fmt.Printf("%s%s┌─ Blob %d/%d %s\n", ColorBold, ColorBlue, current, total, strings.Repeat("─", 50-len(fmt.Sprintf("┌─ Blob %d/%d ", current, total)))+ColorReset)
	}
}

func displayBlobInfo(id coreda.ID, blob []byte) {
	fmt.Printf("│ %sID:%s %s\n", colorize(ColorBold), ColorReset, formatHash(hex.EncodeToString(id)))
	fmt.Printf("│ %sSize:%s %s bytes\n", colorize(ColorBold), ColorReset, formatSize(len(blob)))

	// Try to parse the ID to show height and commitment
	if idHeight, commitment, err := coreda.SplitID(id); err == nil {
		fmt.Printf("│ %sID Height:%s %d\n", colorize(ColorBold), ColorReset, idHeight)
		fmt.Printf("│ %sCommitment:%s %s\n", colorize(ColorBold), ColorReset, formatHash(hex.EncodeToString(commitment)))
	}
	fmt.Printf("│\n")
}

func printTypeHeader(title, color string) {
	fmt.Printf("│ %s%s%s %s%s\n", ColorBold, color, title, ColorReset, strings.Repeat("─", 45))
	fmt.Printf("│\n")
}

func displayHeader(header *types.SignedHeader) {
	fmt.Printf("│ %sCore Information:%s\n", colorizeSection("Core Information"), ColorReset)
	heightColor := ColorYellow
	heightPrefix := ""
	if filterHeight > 0 && header.Height() == filterHeight {
		heightColor = ColorGreen
		heightPrefix = "[MATCH] "
	}
	fmt.Printf("│   Height: %s%s%d%s\n", heightPrefix, colorize(heightColor), header.Height(), ColorReset)
	fmt.Printf("│   Time: %s%s%s\n", colorize(ColorCyan), header.Time().Format(time.RFC3339), ColorReset)
	fmt.Printf("│   Chain ID: %s%s%s\n", colorize(ColorMagenta), header.ChainID(), ColorReset)
	fmt.Printf("│   Version: Block=%s%d%s, App=%s%d%s\n",
		colorize(ColorYellow), header.Version.Block, ColorReset,
		colorize(ColorYellow), header.Version.App, ColorReset)
	fmt.Printf("│\n")

	fmt.Printf("│ %sHashes:%s\n", colorizeSection("Hashes"), ColorReset)
	fmt.Printf("│   Last Header: %s\n", formatHashField(hex.EncodeToString(header.LastHeaderHash[:])))
	fmt.Printf("│   Last Commit: %s\n", formatHashField(hex.EncodeToString(header.LastCommitHash[:])))
	fmt.Printf("│   Data Hash: %s\n", formatHashField(hex.EncodeToString(header.DataHash[:])))
	fmt.Printf("│   Consensus: %s\n", formatHashField(hex.EncodeToString(header.ConsensusHash[:])))
	fmt.Printf("│   App Hash: %s\n", formatHashField(hex.EncodeToString(header.AppHash[:])))
	fmt.Printf("│   Last Results: %s\n", formatHashField(hex.EncodeToString(header.LastResultsHash[:])))
	fmt.Printf("│   Validator: %s\n", formatHashField(hex.EncodeToString(header.ValidatorHash[:])))
	fmt.Printf("│\n")

	fmt.Printf("│ %sSignature Information:%s\n", colorizeSection("Signature Information"), ColorReset)
	fmt.Printf("│   Proposer: %s\n", formatHashField(hex.EncodeToString(header.ProposerAddress)))
	fmt.Printf("│   Signature: %s\n", formatHashField(hex.EncodeToString(header.Signature)))
	if len(header.Signer.Address) > 0 {
		fmt.Printf("│   Signer: %s\n", formatHashField(hex.EncodeToString(header.Signer.Address)))
	}
}

func displayData(data *types.SignedData) {
	fmt.Printf("│ %sMetadata:%s\n", colorizeSection("Metadata"), ColorReset)
	if data.Metadata != nil {
		fmt.Printf("│   Chain ID: %s%s%s\n", colorize(ColorMagenta), data.ChainID(), ColorReset)
		heightColor := ColorYellow
		heightPrefix := ""
		if filterHeight > 0 && data.Height() == filterHeight {
			heightColor = ColorGreen
			heightPrefix = "[MATCH] "
		}
		fmt.Printf("│   Height: %s%s%d%s\n", heightPrefix, colorize(heightColor), data.Height(), ColorReset)
		fmt.Printf("│   Time: %s%s%s\n", colorize(ColorCyan), data.Time().Format(time.RFC3339), ColorReset)
		fmt.Printf("│   Last Data Hash: %s\n", formatHashField(hex.EncodeToString(data.LastDataHash[:])))
	}

	dataHash := data.DACommitment()
	fmt.Printf("│   DA Commitment: %s\n", formatHashField(hex.EncodeToString(dataHash[:])))
	fmt.Printf("│\n")

	fmt.Printf("│ %sTransaction Summary:%s\n", colorizeSection("Transaction Summary"), ColorReset)
	fmt.Printf("│   Count: %s%d%s transactions\n", colorize(ColorYellow), len(data.Txs), ColorReset)
	fmt.Printf("│   Signature: %s\n", formatHashField(hex.EncodeToString(data.Signature)))

	if len(data.Signer.Address) > 0 {
		fmt.Printf("│   Signer: %s\n", formatHashField(hex.EncodeToString(data.Signer.Address)))
	}

	// Display transactions
	if len(data.Txs) > 0 {
		fmt.Printf("│\n")
		fmt.Printf("│ %sTransactions:%s\n", colorizeSection("Transactions"), ColorReset)
		for i, tx := range data.Txs {
			fmt.Printf("│   %s%s[%d]%s Size: %s, Hash: %s\n",
				ColorBold, ColorYellow, i+1, ColorReset,
				formatSize(len(tx)),
				formatShortHash(hex.EncodeToString(tx)))

			if isPrintable(tx) && len(tx) < 200 {
				preview := string(tx)
				if len(preview) > 60 {
					preview = preview[:60] + "..."
				}
				fmt.Printf("│       %sData:%s %s%s%s\n", colorize(ColorDim), ColorReset, colorize(ColorGreen), preview, ColorReset)
			}
		}
	}
}

func displayRawData(blob []byte) {
	fmt.Printf("│ %sRaw Binary Data:%s\n", colorizeSection("Raw Binary Data"), ColorReset)

	hexStr := hex.EncodeToString(blob)
	if len(hexStr) > 120 {
		fmt.Printf("│   Hex (preview): %s%s...%s\n", colorize(ColorDim), hexStr[:120], ColorReset)
		fmt.Printf("│   Full length: %s bytes\n", formatSize(len(blob)))
	} else {
		fmt.Printf("│   Hex: %s%s%s\n", colorize(ColorDim), hexStr, ColorReset)
	}

	if isPrintable(blob) {
		strData := string(blob)
		if len(strData) > 200 {
			fmt.Printf("│   String (preview): %s%s...%s\n", colorize(ColorGreen), strData[:200], ColorReset)
		} else {
			fmt.Printf("│   String: %s%s%s\n", colorize(ColorGreen), strData, ColorReset)
		}
	} else {
		fmt.Printf("│   %s(Binary data - not printable as string)%s\n", colorize(ColorDim), ColorReset)
	}
}

// Helper functions for formatting and colors
func colorize(color string) string {
	if noColor {
		return ""
	}
	return color
}

func colorizeSection(title string) string {
	return fmt.Sprintf("%s%s%s%s", ColorBold, ColorWhite, title, ColorReset)
}

func formatHash(hash string) string {
	if len(hash) > 16 {
		return fmt.Sprintf("%s%s...%s%s", colorize(ColorDim), hash[:8], hash[len(hash)-8:], ColorReset)
	}
	return fmt.Sprintf("%s%s%s", colorize(ColorDim), hash, ColorReset)
}

func formatHashField(hash string) string {
	if len(hash) > 20 {
		return fmt.Sprintf("%s%s...%s%s", colorize(ColorDim), hash[:10], hash[len(hash)-10:], ColorReset)
	}
	return fmt.Sprintf("%s%s%s", colorize(ColorDim), hash, ColorReset)
}

func formatShortHash(hash string) string {
	if len(hash) > 12 {
		return fmt.Sprintf("%s%s...%s", colorize(ColorDim), hash[:6], hash[len(hash)-6:], ColorReset)
	}
	return fmt.Sprintf("%s%s%s", colorize(ColorDim), hash, ColorReset)
}

func formatSize(bytes int) string {
	if bytes < 1024 {
		return fmt.Sprintf("%s%d B%s", colorize(ColorYellow), bytes, ColorReset)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%s%.1f KB%s", colorize(ColorYellow), float64(bytes)/1024, ColorReset)
	} else {
		return fmt.Sprintf("%s%.1f MB%s", colorize(ColorYellow), float64(bytes)/(1024*1024), ColorReset)
	}
}

func printSeparator() {
	fmt.Printf("%s%s├%s\n", ColorDim, strings.Repeat("─", 60), ColorReset)
}

func printFooter() {
	fmt.Printf("%s%s└%s\n", ColorBold, ColorBlue, strings.Repeat("─", 60)+ColorReset)
	fmt.Printf("%sAnalysis complete!%s\n", colorize(ColorGreen), ColorReset)
}

func printSuccess(format string, args ...interface{}) {
	fmt.Printf("%s%s"+format+"%s\n", ColorBold, ColorGreen, ColorReset)
	if len(args) > 0 {
		fmt.Printf(format+"\n", args...)
	}
}

func printWarning(format string, args ...interface{}) {
	fmt.Printf("%s%s"+format+"%s\n", ColorBold, ColorYellow, ColorReset)
	if len(args) > 0 {
		fmt.Printf(format+"\n", args...)
	}
}

func printInfo(format string, args ...interface{}) {
	fmt.Printf("%s"+format+"%s\n", colorize(ColorCyan), ColorReset)
	if len(args) > 0 {
		fmt.Printf(format+"\n", args...)
	}
}

func printError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "%s%s"+format+"%s", ColorBold, ColorRed, ColorReset)
	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, format, args...)
	}
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
