package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"cosmossdk.io/log"
	cometprivval "github.com/cometbft/cometbft/privval"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	coreda "github.com/rollkit/rollkit/core/da"
	coreexecutor "github.com/rollkit/rollkit/core/execution"
	coresequencer "github.com/rollkit/rollkit/core/sequencer"
	"github.com/rollkit/rollkit/node"
	rollconf "github.com/rollkit/rollkit/pkg/config"
	genesispkg "github.com/rollkit/rollkit/pkg/genesis"
	rollos "github.com/rollkit/rollkit/pkg/os"
	"github.com/rollkit/rollkit/pkg/signer"
)

var (
	// initialize the rollkit node configuration
	nodeConfig = rollconf.DefaultNodeConfig

	// initialize the logger with the cometBFT defaults
	logger = log.NewLogger(os.Stdout)
)

func parseConfig(cmd *cobra.Command) error {
	// Load configuration with the correct order of precedence:
	// DefaultNodeConfig -> Yaml -> Flags
	var err error
	nodeConfig, err = rollconf.LoadNodeConfig(cmd)
	if err != nil {
		return fmt.Errorf("failed to load node config: %w", err)
	}

	// Validate the root directory
	if err := rollconf.EnsureRoot(nodeConfig.RootDir); err != nil {
		return fmt.Errorf("failed to ensure root directory: %w", err)
	}

	return nil
}

// setupLogger configures and returns a logger based on the provided configuration.
// It applies the following settings from the config:
//   - Log format (text or JSON)
//   - Log level (debug, info, warn, error)
//   - Stack traces for error logs
//
// The returned logger is already configured with the "module" field set to "main".
func setupLogger(config rollconf.LogConfig) log.Logger {
	var logOptions []log.Option

	// Configure logger format
	if config.Format == "json" {
		logOptions = append(logOptions, log.OutputJSONOption())
	}

	// Configure logger level
	switch strings.ToLower(config.Level) {
	case "debug":
		logOptions = append(logOptions, log.LevelOption(zerolog.DebugLevel))
	case "info":
		logOptions = append(logOptions, log.LevelOption(zerolog.InfoLevel))
	case "warn":
		logOptions = append(logOptions, log.LevelOption(zerolog.WarnLevel))
	case "error":
		logOptions = append(logOptions, log.LevelOption(zerolog.ErrorLevel))
	default:
		logOptions = append(logOptions, log.LevelOption(zerolog.InfoLevel))
	}

	// Configure stack traces
	if config.Trace {
		logOptions = append(logOptions, log.TraceOption(true))
	}

	// Initialize logger with configured options
	configuredLogger := log.NewLogger(os.Stdout)
	if len(logOptions) > 0 {
		configuredLogger = log.NewLogger(os.Stdout, logOptions...)
	}

	// Add module to logger
	return configuredLogger.With("module", "main")
}

// NewRunNodeCmd returns the command that allows the CLI to start a node.
func NewRunNodeCmd(
	executor coreexecutor.Executor,
	sequencer coresequencer.Sequencer,
	dac coreda.Client,
	keyProvider signer.KeyProvider,
) *cobra.Command {
	if executor == nil {
		panic("executor cannot be nil")
	}
	if sequencer == nil {
		panic("sequencer cannot be nil")
	}
	if dac == nil {
		panic("da client cannot be nil")
	}
	if keyProvider == nil {
		panic("key provider cannot be nil")
	}

	cmd := &cobra.Command{
		Use:     "start",
		Aliases: []string{"node", "run"},
		Short:   "Run the rollkit node",
		// PersistentPreRunE is used to parse the config and initial the config files
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := parseConfig(cmd); err != nil {
				return err
			}

			logger = setupLogger(nodeConfig.Log)

			return initConfigFiles()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return startNode(cmd, executor, sequencer, dac, keyProvider)
		},
	}

	// Add Rollkit flags
	rollconf.AddFlags(cmd)

	return cmd
}

// initConfigFiles initializes the config and data directories
func initConfigFiles() error {
	// Create config and data directories using nodeConfig values
	configDir := filepath.Join(nodeConfig.RootDir, nodeConfig.ConfigDir)
	dataDir := filepath.Join(nodeConfig.RootDir, nodeConfig.DBPath)

	if err := os.MkdirAll(configDir, rollconf.DefaultDirPerm); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.MkdirAll(dataDir, rollconf.DefaultDirPerm); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Generate the private validator config files
	cometprivvalKeyFile := filepath.Join(configDir, "priv_validator_key.json")
	cometprivvalStateFile := filepath.Join(dataDir, "priv_validator_state.json")
	if rollos.FileExists(cometprivvalKeyFile) {
		logger.Info("Found private validator", "keyFile", cometprivvalKeyFile,
			"stateFile", cometprivvalStateFile)
	} else {
		pv := cometprivval.GenFilePV(cometprivvalKeyFile, cometprivvalStateFile)
		pv.Save()
		logger.Info("Generated private validator", "keyFile", cometprivvalKeyFile,
			"stateFile", cometprivvalStateFile)
	}

	// Generate the genesis file
	genFile := filepath.Join(configDir, "genesis.json")
	if rollos.FileExists(genFile) {
		logger.Info("Found genesis file", "path", genFile)
	} else {
		// Create a default genesis
		genesis := genesispkg.NewGenesis(
			"test-chain",
			uint64(1),
			time.Now(),
			genesispkg.GenesisExtraData{}, // No proposer address for now
			nil,                           // No raw bytes for now
		)

		// Marshal the genesis struct directly
		genesisBytes, err := json.MarshalIndent(genesis, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal genesis: %w", err)
		}

		// Write genesis bytes directly to file
		if err := os.WriteFile(genFile, genesisBytes, 0600); err != nil {
			return fmt.Errorf("failed to write genesis file: %w", err)
		}
		logger.Info("Generated genesis file", "path", genFile)
	}

	return nil
}

// startNode handles the node startup logic
func startNode(cmd *cobra.Command, executor coreexecutor.Executor, sequencer coresequencer.Sequencer, dac coreda.Client, keyProvider signer.KeyProvider) error {
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	if err := parseConfig(cmd); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	if err := initConfigFiles(); err != nil {
		return fmt.Errorf("failed to initialize files: %w", err)
	}

	signingKey, err := keyProvider.GetSigningKey()
	if err != nil {
		return fmt.Errorf("failed to get signing key: %w", err)
	}

	metrics := node.DefaultMetricsProvider(rollconf.DefaultInstrumentationConfig())

	genesisPath := filepath.Join(nodeConfig.RootDir, nodeConfig.ConfigDir, "genesis.json")
	genesis, err := genesispkg.LoadGenesis(genesisPath)
	if err != nil {
		return fmt.Errorf("failed to load genesis: %w", err)
	}

	// Create and start the node
	rollnode, err := node.NewNode(
		ctx,
		nodeConfig,
		executor,
		sequencer,
		dac,
		signingKey,
		genesis,
		metrics,
		logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create node: %w", err)
	}

	// Run the node with graceful shutdown
	errCh := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("node panicked: %v", r)
				logger.Error("Recovered from panic in node", "panic", r)
				select {
				case errCh <- err:
				default:
					logger.Error("Error channel full", "error", err)
				}
			}
		}()

		err := rollnode.Run(ctx)
		select {
		case errCh <- err:
		default:
			logger.Error("Error channel full", "error", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	select {
	case <-quit:
		logger.Info("shutting down node...")
		cancel()
	case err := <-errCh:
		logger.Error("node error", "error", err)
		cancel()
		return err
	}

	// Wait for node to finish shutting down
	select {
	case <-time.After(5 * time.Second):
		logger.Info("Node shutdown timed out")
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("Error during shutdown", "error", err)
			return err
		}
	}

	return nil
}
