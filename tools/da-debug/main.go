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
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "da-debug <height> <namespace>",
		Short: "DA debugging tool - decode blobs at given height and namespace",
		Long: `A simple DA debugging tool that queries a specific height and namespace,
then decodes each blob as either header or data and displays the information.`,
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

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
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

	client, err := createDAClient()
	if err != nil {
		return err
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	fmt.Printf("Querying DA layer at height %d with namespace %s...\n", height, hex.EncodeToString(namespace))

	result, err := client.DA.GetIDs(ctx, height, namespace)
	if err != nil {
		// Handle "blob not found" as a normal case
		if err.Error() == "blob: not found" || strings.Contains(err.Error(), "blob: not found") {
			fmt.Printf("No blobs found at height %d with namespace %s\n", height, hex.EncodeToString(namespace))
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
		fmt.Printf("No blobs found at height %d with namespace %s\n", height, hex.EncodeToString(namespace))
		return nil
	}

	fmt.Printf("Found %d blob(s) at height %d:\n", len(result.IDs), height)
	fmt.Printf("Timestamp: %s\n\n", result.Timestamp.Format(time.RFC3339))

	// Get the actual blob data
	blobs, err := client.DA.Get(ctx, result.IDs, namespace)
	if err != nil {
		return fmt.Errorf("failed to get blob data: %w", err)
	}

	// Process each blob
	for i, blob := range blobs {
		fmt.Printf("=== Blob %d ===\n", i+1)
		fmt.Printf("ID: %s\n", hex.EncodeToString(result.IDs[i]))
		fmt.Printf("Size: %d bytes\n", len(blob))

		// Try to parse the ID to show height and commitment
		if idHeight, commitment, err := coreda.SplitID(result.IDs[i]); err == nil {
			fmt.Printf("ID Height: %d\n", idHeight)
			fmt.Printf("ID Commitment: %s\n", hex.EncodeToString(commitment))
		}

		// Try to decode as header first
		if header := tryDecodeHeader(blob); header != nil {
			fmt.Printf("Type: SignedHeader\n")
			displayHeader(header)
		} else if data := tryDecodeData(blob); data != nil {
			fmt.Printf("Type: SignedData\n")
			displayData(data)
		} else {
			fmt.Printf("Type: Unknown/Raw Data\n")
			displayRawData(blob)
		}
		fmt.Println()
	}

	return nil
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

func displayHeader(header *types.SignedHeader) {
	fmt.Printf("Header Details:\n")
	fmt.Printf("  Height: %d\n", header.Height())
	fmt.Printf("  Time: %s\n", header.Time().Format(time.RFC3339))
	fmt.Printf("  Chain ID: %s\n", header.ChainID())
	fmt.Printf("  Version: Block=%d, App=%d\n", header.Version.Block, header.Version.App)
	fmt.Printf("  Last Header Hash: %s\n", hex.EncodeToString(header.LastHeaderHash[:]))
	fmt.Printf("  Last Commit Hash: %s\n", hex.EncodeToString(header.LastCommitHash[:]))
	fmt.Printf("  Data Hash: %s\n", hex.EncodeToString(header.DataHash[:]))
	fmt.Printf("  Consensus Hash: %s\n", hex.EncodeToString(header.ConsensusHash[:]))
	fmt.Printf("  App Hash: %s\n", hex.EncodeToString(header.AppHash[:]))
	fmt.Printf("  Last Results Hash: %s\n", hex.EncodeToString(header.LastResultsHash[:]))
	fmt.Printf("  Validator Hash: %s\n", hex.EncodeToString(header.ValidatorHash[:]))
	fmt.Printf("  Proposer Address: %s\n", hex.EncodeToString(header.ProposerAddress))
	fmt.Printf("  Signature: %s\n", hex.EncodeToString(header.Signature))
	if len(header.Signer.Address) > 0 {
		fmt.Printf("  Signer Address: %s\n", hex.EncodeToString(header.Signer.Address))
	}
}

func displayData(data *types.SignedData) {
	fmt.Printf("Data Details:\n")
	if data.Metadata != nil {
		fmt.Printf("  Chain ID: %s\n", data.ChainID())
		fmt.Printf("  Height: %d\n", data.Height())
		fmt.Printf("  Time: %s\n", data.Time().Format(time.RFC3339))
		fmt.Printf("  Last Data Hash: %s\n", hex.EncodeToString(data.LastDataHash[:]))
	}

	dataHash := data.DACommitment()
	fmt.Printf("  DA Commitment Hash: %s\n", hex.EncodeToString(dataHash[:]))
	fmt.Printf("  Transaction Count: %d\n", len(data.Txs))
	fmt.Printf("  Signature: %s\n", hex.EncodeToString(data.Signature))

	if len(data.Signer.Address) > 0 {
		fmt.Printf("  Signer Address: %s\n", hex.EncodeToString(data.Signer.Address))
	}

	// Display transactions
	if len(data.Txs) > 0 {
		fmt.Printf("  Transactions:\n")
		for i, tx := range data.Txs {
			fmt.Printf("    Tx %d: Size=%d bytes, Hash=%s\n", i+1, len(tx), hex.EncodeToString(tx)[:40]+"...")
			if isPrintable(tx) && len(tx) < 200 {
				fmt.Printf("      Data: %s\n", string(tx))
			}
		}
	}
}

func displayRawData(blob []byte) {
	fmt.Printf("Raw Data:\n")
	fmt.Printf("  Hex: %s\n", hex.EncodeToString(blob))

	if len(blob) > 200 {
		fmt.Printf("  Hex Preview (first 200 bytes): %s...\n", hex.EncodeToString(blob[:200]))
	}

	if isPrintable(blob) {
		if len(blob) > 500 {
			fmt.Printf("  String Preview (first 500 chars): %s...\n", string(blob[:500]))
		} else {
			fmt.Printf("  String: %s\n", string(blob))
		}
	}
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
