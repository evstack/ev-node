package cmd

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/ipfs/go-datastore"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/evstack/ev-node/block"
	"github.com/evstack/ev-node/core/execution"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	executiongrpc "github.com/evstack/ev-node/execution/grpc"
	"github.com/evstack/ev-node/node"
	rollcmd "github.com/evstack/ev-node/pkg/cmd"
	"github.com/evstack/ev-node/pkg/config"
	blobrpc "github.com/evstack/ev-node/pkg/da/jsonrpc"
	da "github.com/evstack/ev-node/pkg/da/types"
	"github.com/evstack/ev-node/pkg/genesis"
	rollgenesis "github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/p2p"
	"github.com/evstack/ev-node/pkg/p2p/key"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/sequencers/based"
	"github.com/evstack/ev-node/sequencers/single"
)

const (
	grpcDbName = "grpc-single"
	// FlagGrpcExecutorURL is the flag for the gRPC executor endpoint
	FlagGrpcExecutorURL = "grpc-executor-url"
)

var RunCmd = &cobra.Command{
	Use:     "start",
	Aliases: []string{"node", "run"},
	Short:   "Run the evolve node with gRPC execution client",
	Long: `Start a Evolve node that connects to a remote execution client via gRPC.
The execution client must implement the Evolve execution gRPC interface.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create gRPC execution client
		executor, err := createGRPCExecutionClient(cmd)
		if err != nil {
			return err
		}

		// Parse node configuration
		nodeConfig, err := rollcmd.ParseConfig(cmd)
		if err != nil {
			return err
		}

		logger := rollcmd.SetupLogger(nodeConfig.Log)

		headerNamespace := da.NamespaceFromString(nodeConfig.DA.GetNamespace())
		dataNamespace := da.NamespaceFromString(nodeConfig.DA.GetDataNamespace())

		logger.Info().Str("headerNamespace", headerNamespace.HexString()).Str("dataNamespace", dataNamespace.HexString()).Msg("namespaces")

		// Create datastore
		datastore, err := store.NewDefaultKVStore(nodeConfig.RootDir, nodeConfig.DBPath, grpcDbName)
		if err != nil {
			return err
		}

		// Load genesis
		genesis, err := rollgenesis.LoadGenesis(rollgenesis.GenesisPath(nodeConfig.RootDir))
		if err != nil {
			return err
		}

		if genesis.DAStartHeight == 0 && !nodeConfig.Node.Aggregator {
			logger.Warn().Msg("da_start_height is not set in genesis.json, ask your chain developer")
		}

		// Create sequencer based on configuration
		sequencer, err := createSequencer(cmd.Context(), logger, datastore, nodeConfig, genesis)
		if err != nil {
			return err
		}

		// Load node key
		nodeKey, err := key.LoadNodeKey(filepath.Dir(nodeConfig.ConfigPath()))
		if err != nil {
			return err
		}

		// Create P2P client
		p2pClient, err := p2p.NewClient(nodeConfig.P2P, nodeKey.PrivKey, datastore, genesis.ChainID, logger, nil)
		if err != nil {
			return err
		}

		// Start the node
		return rollcmd.StartNode(logger, cmd, executor, sequencer, p2pClient, datastore, nodeConfig, genesis, node.NodeOptions{})
	},
}

func init() {
	// Add evolve configuration flags
	config.AddFlags(RunCmd)

	// Add gRPC-specific flags
	addGRPCFlags(RunCmd)
}

// createSequencer creates a sequencer based on the configuration.
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
	fiRetriever := block.NewForcedInclusionRetriever(daClient, genesis, logger)

	if nodeConfig.Node.BasedSequencer {
		// Based sequencer mode - fetch transactions only from DA
		if !nodeConfig.Node.Aggregator {
			return nil, fmt.Errorf("based sequencer mode requires aggregator mode to be enabled")
		}

		basedSeq, err := based.NewBasedSequencer(ctx, fiRetriever, datastore, genesis, logger)
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
		ctx,
		logger,
		datastore,
		daClient,
		[]byte(genesis.ChainID),
		nodeConfig.Node.BlockTime.Duration,
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

// createGRPCExecutionClient creates a new gRPC execution client from command flags
func createGRPCExecutionClient(cmd *cobra.Command) (execution.Executor, error) {
	// Get the gRPC executor URL from flags
	executorURL, err := cmd.Flags().GetString(FlagGrpcExecutorURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get '%s' flag: %w", FlagGrpcExecutorURL, err)
	}

	if executorURL == "" {
		return nil, fmt.Errorf("%s flag is required", FlagGrpcExecutorURL)
	}

	// Create and return the gRPC client
	return executiongrpc.NewClient(executorURL), nil
}

// addGRPCFlags adds flags specific to the gRPC execution client
func addGRPCFlags(cmd *cobra.Command) {
	cmd.Flags().String(FlagGrpcExecutorURL, "http://localhost:50051", "URL of the gRPC execution service")
}
