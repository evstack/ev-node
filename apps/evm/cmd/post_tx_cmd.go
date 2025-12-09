package cmd

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	evblock "github.com/evstack/ev-node/block"
	"github.com/evstack/ev-node/da/jsonrpc"
	rollcmd "github.com/evstack/ev-node/pkg/cmd"
	rollconf "github.com/evstack/ev-node/pkg/config"
	da "github.com/evstack/ev-node/pkg/da/types"
	genesispkg "github.com/evstack/ev-node/pkg/genesis"
	seqcommon "github.com/evstack/ev-node/sequencers/common"
	"github.com/evstack/ev-node/types"
)

const (
	flagNamespace = "namespace"
	flagGasPrice  = "gas-price"
)

// PostTxCmd returns a command to post a signed Ethereum transaction to the DA layer
func PostTxCmd() *cobra.Command {
	cobraCmd := &cobra.Command{
		Use:   "post-tx",
		Short: "Post a signed Ethereum transaction to the DA layer",
		Long: `Post a signed Ethereum transaction to the DA layer using the Evolve configuration.

This command submits a signed Ethereum transaction tzo the configured DA layer for forced inclusion.
The transaction is provided as an argument, which accepts either:
  1. A hex-encoded signed transaction (with or without 0x prefix)
  2. A path to a file containing the hex-encoded transaction
  3. A JSON object with a "raw" field containing the hex-encoded transaction

The command automatically detects the input format.

Examples:
  # From hex string
  evm post-tx 0x02f873...

  # From file
  evm post-tx tx.txt

  # From JSON
  evm post-tx '{"raw":"0x02f873..."}'
`,
		Args: cobra.ExactArgs(1),
		RunE: postTxRunE,
	}

	// Add evolve config flags
	rollconf.AddFlags(cobraCmd)

	// Add command-specific flags
	cobraCmd.Flags().String(flagNamespace, "", "DA namespace ID (if not provided, uses config namespace)")
	cobraCmd.Flags().Float64(flagGasPrice, -1, "Gas price for DA submission (if not provided, uses config gas price)")

	return cobraCmd
}

// postTxRunE executes the post-tx command
func postTxRunE(cmd *cobra.Command, args []string) error {
	nodeConfig, err := rollcmd.ParseConfig(cmd)
	if err != nil {
		return err
	}

	logger := rollcmd.SetupLogger(nodeConfig.Log)

	txInput := args[0]
	if txInput == "" {
		return fmt.Errorf("transaction cannot be empty")
	}

	var txData []byte
	if _, err := os.Stat(txInput); err == nil {
		// Input is a file path
		txData, err = decodeTxFromFile(txInput)
		if err != nil {
			return fmt.Errorf("failed to decode transaction from file: %w", err)
		}
	} else {
		// Input is a JSON string
		txData, err = decodeTxFromJSON(txInput)
		if err != nil {
			return fmt.Errorf("failed to decode transaction from JSON: %w", err)
		}
	}

	if len(txData) == 0 {
		return fmt.Errorf("transaction data cannot be empty")
	}

	// Get namespace (use flag if provided, otherwise use config)
	namespace, _ := cmd.Flags().GetString(flagNamespace)
	if namespace == "" {
		namespace = nodeConfig.DA.GetForcedInclusionNamespace()
	}

	if namespace == "" {
		return fmt.Errorf("forced inclusionnamespace cannot be empty")
	}

	namespaceBz := da.NamespaceFromString(namespace).Bytes()

	// Get gas price (use flag if provided, otherwise use config)
	gasPrice, err := cmd.Flags().GetFloat64(flagGasPrice)
	if err != nil {
		return fmt.Errorf("failed to get gas-price flag: %w", err)
	}

	logger.Info().Str("namespace", namespace).Float64("gas_price", gasPrice).Int("tx_size", len(txData)).Msg("posting transaction to DA layer")

	daClient, err := jsonrpc.NewClient(
		cmd.Context(),
		logger,
		nodeConfig.DA.Address,
		nodeConfig.DA.AuthToken,
		seqcommon.AbsoluteMaxBlobSize,
	)
	if err != nil {
		return fmt.Errorf("failed to create DA client: %w", err)
	}

	// Submit transaction to DA layer
	logger.Info().Msg("submitting transaction to DA layer...")

	blobs := [][]byte{txData}
	options := []byte(nodeConfig.DA.SubmitOptions)

	dac := evblock.NewDAClient(&daClient.DA, nodeConfig, logger)
	result := dac.Submit(cmd.Context(), blobs, gasPrice, namespaceBz, options)

	// Check result
	switch result.Code {
	case da.StatusSuccess:
		logger.Info().Msg("transaction successfully submitted to DA layer")
		cmd.Printf("\n✓ Transaction posted successfully\n\n")
		cmd.Printf("Namespace:  %s\n", namespace)
		cmd.Printf("DA Height:  %d\n", result.Height)
		cmd.Printf("Data Size:  %d bytes\n", len(txData))

		genesisPath := filepath.Join(filepath.Dir(nodeConfig.ConfigPath()), "genesis.json")
		genesis, err := genesispkg.LoadGenesis(genesisPath)
		if err != nil {
			return fmt.Errorf("failed to load genesis for calculating inclusion time estimate: %w", err)
		}

		_, epochEnd, _ := types.CalculateEpochBoundaries(result.Height, genesis.DAStartHeight, genesis.DAEpochForcedInclusion)
		cmd.Printf(
			"DA Blocks until inclusion: %d (at DA height %d)\n",
			epochEnd-(result.Height+1),
			epochEnd+1,
		)

		cmd.Printf("\n")
		return nil

	case da.StatusTooBig:
		return fmt.Errorf("transaction too large for DA layer: %s", result.Message)

	case da.StatusNotIncludedInBlock:
		return fmt.Errorf("transaction not included in DA block: %s", result.Message)

	case da.StatusAlreadyInMempool:
		cmd.Printf("⚠ Transaction already in mempool\n")
		if result.Height > 0 {
			cmd.Printf("  DA Height: %d\n", result.Height)
		}
		return nil

	case da.StatusContextCanceled:
		return fmt.Errorf("submission canceled: %s", result.Message)

	default:
		return fmt.Errorf("DA submission failed (code: %d): %s", result.Code, result.Message)
	}
}

// decodeTxFromFile reads an Ethereum transaction from a file and decodes it to bytes
func decodeTxFromFile(filePath string) ([]byte, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	return decodeTxFromJSON(string(data))
}

// decodeTxFromJSON decodes an Ethereum transaction from various formats to bytes
func decodeTxFromJSON(input string) ([]byte, error) {
	input = strings.TrimSpace(input)

	// Try to decode as JSON with "raw" field
	var txJSON map[string]any
	if err := json.Unmarshal([]byte(input), &txJSON); err == nil {
		if rawTx, ok := txJSON["raw"].(string); ok {
			return decodeHexTx(rawTx)
		}
		return nil, fmt.Errorf("JSON must contain 'raw' field with hex-encoded transaction")
	}

	// Try to decode as hex string directly
	return decodeHexTx(input)
}

// decodeHexTx decodes a hex-encoded Ethereum transaction
func decodeHexTx(hexStr string) ([]byte, error) {
	hexStr = strings.TrimSpace(hexStr)

	// Remove 0x prefix if present
	if strings.HasPrefix(hexStr, "0x") || strings.HasPrefix(hexStr, "0X") {
		hexStr = hexStr[2:]
	}

	// Decode hex string to bytes
	txBytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, fmt.Errorf("decoding hex transaction: %w", err)
	}

	if len(txBytes) == 0 {
		return nil, fmt.Errorf("decoded transaction is empty")
	}

	return txBytes, nil
}
