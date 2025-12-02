package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ipfs/go-datastore"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/evstack/ev-node/block"
	"github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/core/execution"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/da/jsonrpc"
	"github.com/evstack/ev-node/execution/evm"
	"github.com/evstack/ev-node/node"
	rollcmd "github.com/evstack/ev-node/pkg/cmd"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	genesispkg "github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/p2p"
	"github.com/evstack/ev-node/pkg/p2p/key"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/sequencers/based"
	seqcommon "github.com/evstack/ev-node/sequencers/common"
	"github.com/evstack/ev-node/sequencers/single"
)

const evmDbName = "evm-single"

var RunCmd = &cobra.Command{
	Use:     "start",
	Aliases: []string{"node", "run"},
	Short:   "Run the evolve node with EVM execution client",
	RunE: func(cmd *cobra.Command, args []string) error {
		executor, err := createExecutionClient(cmd)
		if err != nil {
			return err
		}

		nodeConfig, err := rollcmd.ParseConfig(cmd)
		if err != nil {
			return err
		}

		logger := rollcmd.SetupLogger(nodeConfig.Log)

		// Attach logger to the EVM engine client if available
		if ec, ok := executor.(*evm.EngineClient); ok {
			ec.SetLogger(logger.With().Str("module", "engine_client").Logger())
		}

		headerNamespace := da.NamespaceFromString(nodeConfig.DA.GetNamespace())
		dataNamespace := da.NamespaceFromString(nodeConfig.DA.GetDataNamespace())

		logger.Info().Str("headerNamespace", headerNamespace.HexString()).Str("dataNamespace", dataNamespace.HexString()).Msg("namespaces")

		daJrpc, err := jsonrpc.NewClient(context.Background(), logger, nodeConfig.DA.Address, nodeConfig.DA.AuthToken, seqcommon.AbsoluteMaxBlobSize)
		if err != nil {
			return err
		}

		datastore, err := store.NewDefaultKVStore(nodeConfig.RootDir, nodeConfig.DBPath, evmDbName)
		if err != nil {
			return err
		}

		genesisPath := filepath.Join(filepath.Dir(nodeConfig.ConfigPath()), "genesis.json")
		genesis, err := genesispkg.LoadGenesis(genesisPath)
		if err != nil {
			return fmt.Errorf("failed to load genesis: %w", err)
		}

		if genesis.DAStartHeight == 0 && !nodeConfig.Node.Aggregator {
			logger.Warn().Msg("da_start_height is not set in genesis.json, ask your chain developer")
		}

		// Create sequencer based on configuration
		sequencer, err := createSequencer(context.Background(), logger, datastore, &daJrpc.DA, nodeConfig, genesis)
		if err != nil {
			return err
		}

		nodeKey, err := key.LoadNodeKey(filepath.Dir(nodeConfig.ConfigPath()))
		if err != nil {
			return err
		}

		p2pClient, err := p2p.NewClient(nodeConfig.P2P, nodeKey.PrivKey, datastore, genesis.ChainID, logger, nil)
		if err != nil {
			return err
		}

		return rollcmd.StartNode(logger, cmd, executor, sequencer, &daJrpc.DA, p2pClient, datastore, nodeConfig, genesis, node.NodeOptions{})
	},
}

func init() {
	config.AddFlags(RunCmd)
	addFlags(RunCmd)
}

// createSequencer creates a sequencer based on the configuration.
// If BasedSequencer is enabled, it creates a based sequencer that fetches transactions from DA.
// Otherwise, it creates a single (traditional) sequencer.
func createSequencer(
	ctx context.Context,
	logger zerolog.Logger,
	datastore datastore.Batching,
	da da.DA,
	nodeConfig config.Config,
	genesis genesis.Genesis,
) (coresequencer.Sequencer, error) {
	daClient := block.NewDAClient(da, nodeConfig, logger)
	fiRetriever := block.NewForcedInclusionRetriever(daClient, genesis, logger)

	if nodeConfig.Node.BasedSequencer {
		// Based sequencer mode - fetch transactions only from DA
		if !nodeConfig.Node.Aggregator {
			return nil, fmt.Errorf("based sequencer mode requires aggregator mode to be enabled")
		}

		basedSeq := based.NewBasedSequencer(fiRetriever, da, nodeConfig, genesis, logger)

		logger.Info().
			Str("forced_inclusion_namespace", nodeConfig.DA.GetForcedInclusionNamespace()).
			Uint64("da_epoch", genesis.DAEpochForcedInclusion).
			Msg("based sequencer initialized")

		return basedSeq, nil
	}

	singleMetrics, err := single.NopMetrics()
	if err != nil {
		return nil, fmt.Errorf("failed to create single sequencer metrics: %w", err)
	}

	sequencer, err := single.NewSequencer(
		ctx,
		logger,
		datastore,
		da,
		[]byte(genesis.ChainID),
		nodeConfig.Node.BlockTime.Duration,
		singleMetrics,
		nodeConfig.Node.Aggregator,
		1000,
		fiRetriever,
		genesis,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create single sequencer: %w", err)
	}

	logger.Info().
		Str("forced_inclusion_namespace", nodeConfig.DA.GetForcedInclusionNamespace()).
		Msg("single sequencer initialized")

	return sequencer, nil
}

func createExecutionClient(cmd *cobra.Command) (execution.Executor, error) {
	// Read execution client parameters from flags
	ethURL, err := cmd.Flags().GetString(evm.FlagEvmEthURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get '%s' flag: %w", evm.FlagEvmEthURL, err)
	}
	engineURL, err := cmd.Flags().GetString(evm.FlagEvmEngineURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get '%s' flag: %w", evm.FlagEvmEngineURL, err)
	}

	// Get JWT secret file path
	jwtSecretFile, err := cmd.Flags().GetString(evm.FlagEvmJWTSecretFile)
	if err != nil {
		return nil, fmt.Errorf("failed to get '%s' flag: %w", evm.FlagEvmJWTSecretFile, err)
	}

	if jwtSecretFile == "" {
		return nil, fmt.Errorf("JWT secret file must be provided via --evm.jwt-secret-file")
	}

	// Read JWT secret from file
	secretBytes, err := os.ReadFile(jwtSecretFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read JWT secret from file '%s': %w", jwtSecretFile, err)
	}
	jwtSecret := string(bytes.TrimSpace(secretBytes))

	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT secret file '%s' is empty", jwtSecretFile)
	}

	genesisHashStr, err := cmd.Flags().GetString(evm.FlagEvmGenesisHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get '%s' flag: %w", evm.FlagEvmGenesisHash, err)
	}
	feeRecipientStr, err := cmd.Flags().GetString(evm.FlagEvmFeeRecipient)
	if err != nil {
		return nil, fmt.Errorf("failed to get '%s' flag: %w", evm.FlagEvmFeeRecipient, err)
	}

	// Convert string parameters to Ethereum types
	genesisHash := common.HexToHash(genesisHashStr)
	feeRecipient := common.HexToAddress(feeRecipientStr)

	return evm.NewEngineExecutionClient(ethURL, engineURL, jwtSecret, genesisHash, feeRecipient)
}

// addFlags adds flags related to the EVM execution client
func addFlags(cmd *cobra.Command) {
	cmd.Flags().String(evm.FlagEvmEthURL, "http://localhost:8545", "URL of the Ethereum JSON-RPC endpoint")
	cmd.Flags().String(evm.FlagEvmEngineURL, "http://localhost:8551", "URL of the Engine API endpoint")
	cmd.Flags().String(evm.FlagEvmJWTSecretFile, "", "Path to file containing the JWT secret for authentication")
	cmd.Flags().String(evm.FlagEvmGenesisHash, "", "Hash of the genesis block")
	cmd.Flags().String(evm.FlagEvmFeeRecipient, "", "Address that will receive transaction fees")
}
