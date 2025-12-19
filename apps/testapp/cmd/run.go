package cmd

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/ipfs/go-datastore"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	kvexecutor "github.com/evstack/ev-node/apps/testapp/kv"
	"github.com/evstack/ev-node/block"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/node"
	"github.com/evstack/ev-node/pkg/cmd"
	"github.com/evstack/ev-node/pkg/config"
	blobrpc "github.com/evstack/ev-node/pkg/da/jsonrpc"
	da "github.com/evstack/ev-node/pkg/da/types"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/p2p"
	"github.com/evstack/ev-node/pkg/p2p/key"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/sequencers/based"
	"github.com/evstack/ev-node/sequencers/single"
)

const testDbName = "testapp"

var RunCmd = &cobra.Command{
	Use:     "start",
	Aliases: []string{"node", "run"},
	Short:   "Run the testapp node",
	RunE: func(command *cobra.Command, args []string) error {
		nodeConfig, err := cmd.ParseConfig(command)
		if err != nil {
			return err
		}

		logger := cmd.SetupLogger(nodeConfig.Log)

		// Get KV endpoint flag
		kvEndpoint, _ := command.Flags().GetString(flagKVEndpoint)
		if kvEndpoint == "" {
			logger.Info().Msg("KV endpoint flag not set, using default from http_server")
		}

		// Create test implementations
		executor, err := kvexecutor.NewKVExecutor(nodeConfig.RootDir, nodeConfig.DBPath)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		headerNamespace := da.NamespaceFromString(nodeConfig.DA.GetNamespace())
		dataNamespace := da.NamespaceFromString(nodeConfig.DA.GetDataNamespace())

		logger.Info().Str("headerNamespace", headerNamespace.HexString()).Str("dataNamespace", dataNamespace.HexString()).Msg("namespaces")

		nodeKey, err := key.LoadNodeKey(filepath.Dir(nodeConfig.ConfigPath()))
		if err != nil {
			return err
		}

		datastore, err := store.NewDefaultKVStore(nodeConfig.RootDir, nodeConfig.DBPath, testDbName)
		if err != nil {
			return err
		}

		// Start the KV executor HTTP server
		if kvEndpoint != "" { // Only start if endpoint is provided
			httpServer := kvexecutor.NewHTTPServer(executor, kvEndpoint)
			err = httpServer.Start(ctx) // Use the main context for lifecycle management
			if err != nil {
				return fmt.Errorf("failed to start KV executor HTTP server: %w", err)
			} else {
				logger.Info().Str("endpoint", kvEndpoint).Msg("KV executor HTTP server started")
			}
		}

		genesisPath := filepath.Join(filepath.Dir(nodeConfig.ConfigPath()), "genesis.json")
		genesis, err := genesis.LoadGenesis(genesisPath)
		if err != nil {
			return fmt.Errorf("failed to load genesis: %w", err)
		}

		if genesis.DAStartHeight == 0 && !nodeConfig.Node.Aggregator {
			logger.Warn().Msg("da_start_height is not set in genesis.json, ask your chain developer")
		}

		// Create sequencer based on configuration
		sequencer, err := createSequencer(ctx, logger, datastore, nodeConfig, genesis)
		if err != nil {
			return err
		}

		p2pClient, err := p2p.NewClient(nodeConfig.P2P, nodeKey.PrivKey, datastore, genesis.ChainID, logger, p2p.NopMetrics())
		if err != nil {
			return err
		}

		return cmd.StartNode(logger, command, executor, sequencer, p2pClient, datastore, nodeConfig, genesis, node.NodeOptions{})
	},
}

// createSequencer creates a sequencer based on the configuration.
// If BasedSequencer is enabled, it creates a based sequencer that fetches transactions from DA.
// Otherwise, it creates a single (traditional) sequencer.
func createSequencer(
	ctx context.Context,
	logger zerolog.Logger,
	datastore datastore.Batching,
	nodeConfig config.Config,
	genesis genesis.Genesis,
) (coresequencer.Sequencer, error) {
	blobClient, err := blobrpc.NewClient(ctx, nodeConfig.DA.Address, nodeConfig.DA.AuthToken, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create blob client: %w", err)
	}

	daClient := block.NewDAClient(blobClient, nodeConfig, logger)

	if nodeConfig.Node.BasedSequencer {
		// Based sequencer mode - fetch transactions only from DA
		if !nodeConfig.Node.Aggregator {
			return nil, fmt.Errorf("based sequencer mode requires aggregator mode to be enabled")
		}

		fiRetriever := block.NewForcedInclusionRetriever(daClient, logger, genesis.DAStartHeight, genesis.DAEpochForcedInclusion)
		basedSeq, err := based.NewBasedSequencer(fiRetriever, datastore, genesis, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create based sequencer: %w", err)
		}

		logger.Info().
			Str("forced_inclusion_namespace", nodeConfig.DA.GetForcedInclusionNamespace()).
			Uint64("da_epoch", genesis.DAEpochForcedInclusion).
			Msg("based sequencer initialized")

		return basedSeq, nil
	}

	sequencer, err := single.NewSequencer(
		logger,
		datastore,
		daClient,
		[]byte(genesis.ChainID),
		nodeConfig.Node.BlockTime.Duration,
		1000,
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
