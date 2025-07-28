package cmd

import (
	"context"
	"fmt"
	"path/filepath"

	kvexecutor "github.com/evstack/ev-node/apps/testapp/kv"
	"github.com/evstack/ev-node/block"
	"github.com/evstack/ev-node/da/jsonrpc"
	rollcmd "github.com/evstack/ev-node/pkg/cmd"
	genesispkg "github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/p2p"
	"github.com/evstack/ev-node/pkg/p2p/key"
	"github.com/evstack/ev-node/pkg/signer/file"
	"github.com/evstack/ev-node/pkg/store"
	rollkitsync "github.com/evstack/ev-node/pkg/sync"
	"github.com/evstack/ev-node/sequencers/single"
	"github.com/spf13/cobra"
)

var RollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback the testapp node",
	RunE: func(cmd *cobra.Command, args []string) error {
		nodeConfig, err := rollcmd.ParseConfig(cmd)
		if err != nil {
			return err
		}

		logger := rollcmd.SetupLogger(nodeConfig.Log)

		// Get KV endpoint flag
		kvEndpoint, _ := cmd.Flags().GetString(flagKVEndpoint)
		if kvEndpoint == "" {
			logger.Info("KV endpoint flag not set, using default from http_server")
		}

		// Create test implementations
		executor, err := kvexecutor.NewKVExecutor(nodeConfig.RootDir, nodeConfig.DBPath)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		daJrpc, err := jsonrpc.NewClient(ctx, logger, nodeConfig.DA.Address, nodeConfig.DA.AuthToken, nodeConfig.DA.Namespace)
		if err != nil {
			return err
		}

		nodeKey, err := key.LoadNodeKey(filepath.Dir(nodeConfig.ConfigPath()))
		if err != nil {
			return err
		}

		datastore, err := store.NewDefaultKVStore(nodeConfig.RootDir, nodeConfig.DBPath, "testapp")
		if err != nil {
			return err
		}

		singleMetrics, err := single.NopMetrics()
		if err != nil {
			return err
		}

		sequencer, err := single.NewSequencer(
			ctx,
			logger,
			datastore,
			&daJrpc.DA,
			[]byte(nodeConfig.ChainID),
			nodeConfig.Node.BlockTime.Duration,
			singleMetrics,
			nodeConfig.Node.Aggregator,
		)
		if err != nil {
			return err
		}

		p2pClient, err := p2p.NewClient(nodeConfig, nodeKey, datastore, logger, p2p.NopMetrics())
		if err != nil {
			return err
		}

		// Load genesis
		genesisPath := filepath.Join(filepath.Dir(nodeConfig.ConfigPath()), "genesis.json")
		gen, err := genesispkg.LoadGenesis(genesisPath)
		if err != nil {
			return fmt.Errorf("failed to load genesis: %w", err)
		}

		// Create signer - use empty passphrase for simplicity in rollback
		signerInstance, err := file.LoadFileSystemSigner(nodeConfig.Signer.SignerPath, []byte(""))
		if err != nil {
			return fmt.Errorf("failed to create signer: %w", err)
		}

		// Create sync services
		headerSyncService, err := rollkitsync.NewHeaderSyncService(datastore, nodeConfig, gen, p2pClient, logger)
		if err != nil {
			return fmt.Errorf("failed to create header sync service: %w", err)
		}

		dataSyncService, err := rollkitsync.NewDataSyncService(datastore, nodeConfig, gen, p2pClient, logger)
		if err != nil {
			return fmt.Errorf("failed to create data sync service: %w", err)
		}

		// Create block manager metrics
		seqMetrics := block.NopMetrics()

		// Create store wrapper
		storeWrapper := store.New(datastore)

		// Create block manager
		blockManager, err := block.NewManager(
			ctx,
			signerInstance,
			nodeConfig,
			gen,
			storeWrapper,
			executor,
			sequencer,
			&daJrpc.DA,
			logger,
			headerSyncService.Store(),
			dataSyncService.Store(),
			headerSyncService,
			dataSyncService,
			seqMetrics,
			1.0, // gasPrice
			1.0, // gasMultiplier
			block.DefaultManagerOptions(),
		)
		if err != nil {
			return fmt.Errorf("failed to create block manager: %w", err)
		}

		// Perform rollback
		logger.Info("Starting rollback operation")
		if err := blockManager.RollbackOneBlock(ctx); err != nil {
			return fmt.Errorf("rollback failed: %w", err)
		}

		logger.Info("Rollback completed successfully")
		return nil
	},
}
