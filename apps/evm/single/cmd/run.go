package cmd

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

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

		daJrpc, err := jsonrpc.NewClient(context.Background(), logger, nodeConfig.DA.Address, nodeConfig.DA.AuthToken, nodeConfig.DA.GasPrice, nodeConfig.DA.GasMultiplier)
		if err != nil {
			return err
		}

		migrations, err := parseMigrations(cmd)
		if err != nil {
			return fmt.Errorf("failed to parse migrations: %w", err)
		}
		daAPI := newNamespaceMigrationDAAPI(daJrpc.DA, nodeConfig, migrations)

		datastore, err := store.NewDefaultKVStore(nodeConfig.RootDir, nodeConfig.DBPath, "evm-single")
		if err != nil {
			return err
		}

		genesisPath := filepath.Join(filepath.Dir(nodeConfig.ConfigPath()), "genesis.json")
		genesis, err := genesispkg.LoadGenesis(genesisPath)
		if err != nil {
			return fmt.Errorf("failed to load genesis: %w", err)
		}

		singleMetrics, err := single.DefaultMetricsProvider(nodeConfig.Instrumentation.IsPrometheusEnabled())(genesis.ChainID)
		if err != nil {
			return err
		}

		sequencer, err := single.NewSequencer(
			context.Background(),
			logger,
			datastore,
			daAPI,
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

		return rollcmd.StartNode(logger, cmd, executor, sequencer, daAPI, p2pClient, datastore, nodeConfig, genesis, node.NodeOptions{})
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
	cmd.Flags().String(flagNSMigrations, "", "Namespace migrations in format: da_height1:namespace1:dataNamespace1,da_height2:namespace2:dataNamespace2")
}

var flagNSMigrations = "ns-migrations"

// parseMigrations parses the migrations flag into a map[uint64]namespaces
func parseMigrations(cmd *cobra.Command) (map[uint64]namespaces, error) {
	migrationsStr, err := cmd.Flags().GetString(flagNSMigrations)
	if err != nil {
		return nil, err
	}

	migrations := make(map[uint64]namespaces)
	if migrationsStr == "" {
		return migrations, nil
	}

	// Parse format: height1:namespace1:dataNamespace1,height2:namespace2:dataNamespace2
	entries := strings.Split(migrationsStr, ",")
	for _, entry := range entries {
		parts := strings.Split(strings.TrimSpace(entry), ":")
		if len(parts) < 2 || len(parts) > 3 {
			return nil, fmt.Errorf("invalid migration entry format: %s (expected height:namespace[:dataNamespace])", entry)
		}

		height, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid height in migration entry %s: %w", entry, err)
		}

		ns := namespaces{
			namespace: parts[1],
		}
		if len(parts) == 3 {
			ns.dataNamespace = parts[2]
		}

		migrations[height] = ns
	}

	return migrations, nil
}

// namespaces defines the namespace used for namespace migration
type namespaces struct {
	namespace     string
	dataNamespace string
}

func (n namespaces) GetNamespace() string {
	return n.namespace
}

func (n namespaces) GetDataNamespace() string {
	if n.dataNamespace == "" {
		return n.namespace
	}

	return n.dataNamespace
}

// namespaceMigrationDAAPI is wrapper around the da json rpc to use when handling namespace migrations
type namespaceMigrationDAAPI struct {
	jsonrpc.API

	migrations map[uint64]namespaces

	currentNamespace     []byte
	currentDataNamespace []byte
}

func newNamespaceMigrationDAAPI(api jsonrpc.API, cfg config.Config, migrations map[uint64]namespaces) *namespaceMigrationDAAPI {
	return &namespaceMigrationDAAPI{
		API:                  api,
		migrations:           migrations,
		currentNamespace:     da.NamespaceFromString(cfg.DA.GetNamespace()).Bytes(),
		currentDataNamespace: da.NamespaceFromString(cfg.DA.GetDataNamespace()).Bytes(),
	}
}

// findNamespaceForHeight determines the correct namespace to use for a given height
func (api *namespaceMigrationDAAPI) findNamespaceForHeight(height uint64, isDataNamespace bool) []byte {
	if len(api.migrations) == 0 {
		if isDataNamespace {
			return api.currentDataNamespace
		}
		return api.currentNamespace
	}

	// Find the highest migration height that is <= requested height
	var selectedHeight uint64
	var found bool
	for migrationHeight := range api.migrations {
		if migrationHeight <= height && migrationHeight > selectedHeight {
			selectedHeight = migrationHeight
			found = true
		}
	}

	// If no migration applies to this height, use current namespace
	if !found {
		if isDataNamespace {
			return api.currentDataNamespace
		}
		return api.currentNamespace
	}

	// Use the namespace from the migration
	migration := api.migrations[selectedHeight]
	if isDataNamespace {
		return da.NamespaceFromString(migration.GetDataNamespace()).Bytes()
	}
	return da.NamespaceFromString(migration.GetNamespace()).Bytes()
}

// GetIDs returns IDs of all Blobs located in DA at given height.
// This method handles namespace migrations by determining the correct namespace based on height
func (api *namespaceMigrationDAAPI) GetIDs(ctx context.Context, height uint64, ns []byte) (*da.GetIDsResult, error) {
	ns = api.findNamespaceForHeight(height, bytes.Equal(ns, api.currentDataNamespace))
	return api.API.GetIDs(ctx, height, ns)
}
