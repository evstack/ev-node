package cmd

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/da/jsonrpc"
	"github.com/evstack/ev-node/node"
	"github.com/evstack/ev-node/sequencers/single"

	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"

	"github.com/evstack/ev-node/execution/evm"

	"github.com/evstack/ev-node/core/execution"
	rollcmd "github.com/evstack/ev-node/pkg/cmd"
	"github.com/evstack/ev-node/pkg/config"
	genesispkg "github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/p2p"
	"github.com/evstack/ev-node/pkg/p2p/key"
	"github.com/evstack/ev-node/pkg/store"
)

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

		daJrpc, err := jsonrpc.NewClient(context.Background(), logger, nodeConfig.DA.Address, nodeConfig.DA.AuthToken, nodeConfig.DA.GasPrice, nodeConfig.DA.GasMultiplier, rollcmd.DefaultMaxBlobSize)
		if err != nil {
			return err
		}

		datastore, err := store.NewDefaultKVStore(nodeConfig.RootDir, nodeConfig.DBPath, "evm-single")
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

		singleMetrics, err := single.DefaultMetricsProvider(nodeConfig.Instrumentation.IsPrometheusEnabled())(genesis.ChainID)
		if err != nil {
			return err
		}

		sequencer, err := single.NewSequencer(
			context.Background(),
			logger,
			datastore,
			&daJrpc.DA,
			[]byte(genesis.ChainID),
			nodeConfig.Node.BlockTime.Duration,
			singleMetrics,
			nodeConfig.Node.Aggregator,
		)
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
	jwtSecret, err := cmd.Flags().GetString(evm.FlagEvmJWTSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to get '%s' flag: %w", evm.FlagEvmJWTSecret, err)
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
	cmd.Flags().String(evm.FlagEvmJWTSecret, "", "The JWT secret for authentication with the execution client")
	cmd.Flags().String(evm.FlagEvmGenesisHash, "", "Hash of the genesis block")
	cmd.Flags().String(evm.FlagEvmFeeRecipient, "", "Address that will receive transaction fees")
}
