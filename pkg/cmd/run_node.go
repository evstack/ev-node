package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	coreda "github.com/evstack/ev-node/core/da"
	coreexecutor "github.com/evstack/ev-node/core/execution"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/node"
	rollconf "github.com/evstack/ev-node/pkg/config"
	genesispkg "github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/p2p"
	"github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/signer/file"
)

// ParseConfig is an helpers that loads the node configuration and validates it.
func ParseConfig(cmd *cobra.Command) (rollconf.Config, error) {
	nodeConfig, err := rollconf.Load(cmd)
	if err != nil {
		return rollconf.Config{}, fmt.Errorf("failed to load node config: %w", err)
	}

	if err := nodeConfig.Validate(); err != nil {
		return rollconf.Config{}, fmt.Errorf("failed to validate node config: %w", err)
	}

	return nodeConfig, nil
}

// SetupLogger creates and configures a zap logger based on the provided configuration.
//
// Configuration options:
//   - Output format (console or JSON)
//   - Log level (debug, info, warn, error)
//   - Stack traces for error logs
//
// The returned logger is production-ready and optimized for performance.
func SetupLogger(config rollconf.LogConfig) (*zap.Logger, error) {
	// Configure encoder
	var encoderConfig zapcore.EncoderConfig
	if config.Format == "json" {
		encoderConfig = zap.NewProductionEncoderConfig()
	} else {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// Configure log level
	var level zapcore.Level
	switch config.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel // Default to info
	}

	// Build logger configuration
	zapConfig := zap.Config{
		Level:            zap.NewAtomicLevelAt(level),
		Development:      config.Format != "json",
		Encoding:         "console",
		EncoderConfig:    encoderConfig,
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
	}

	if config.Format == "json" {
		zapConfig.Encoding = "json"
	}

	// Build the logger
	logger, err := zapConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}

	return logger, nil
}

// StartNode handles the node startup logic
func StartNode(
	logger *zap.Logger,
	cmd *cobra.Command,
	executor coreexecutor.Executor,
	sequencer coresequencer.Sequencer,
	da coreda.DA,
	p2pClient *p2p.Client,
	datastore datastore.Batching,
	nodeConfig rollconf.Config,
	nodeOptions node.NodeOptions,
) error {
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	// create a new remote signer
	var signer signer.Signer
	if nodeConfig.Signer.SignerType == "file" && nodeConfig.Node.Aggregator {
		passphrase, err := cmd.Flags().GetString(rollconf.FlagSignerPassphrase)
		if err != nil {
			return err
		}

		signer, err = file.LoadFileSystemSigner(nodeConfig.Signer.SignerPath, []byte(passphrase))
		if err != nil {
			return err
		}
	} else if nodeConfig.Signer.SignerType == "grpc" {
		panic("grpc remote signer not implemented")
	} else if nodeConfig.Node.Aggregator {
		return fmt.Errorf("unknown remote signer type: %s", nodeConfig.Signer.SignerType)
	}

	metrics := node.DefaultMetricsProvider(nodeConfig.Instrumentation)

	genesisPath := filepath.Join(filepath.Dir(nodeConfig.ConfigPath()), "genesis.json")
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
		da,
		signer,
		p2pClient,
		genesis,
		datastore,
		metrics,
		logger,
		nodeOptions,
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
				logger.Error("Recovered from panic in node", zap.Any("panic", r))
				select {
				case errCh <- err:
				default:
					logger.Error("Error channel full", zap.Error(err))
				}
			}
		}()

		err := rollnode.Run(ctx)
		select {
		case errCh <- err:
		default:
			logger.Error("Error channel full", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case <-quit:
		logger.Info("shutting down node...")
		cancel()
	case err := <-errCh:
		logger.Error("node error", zap.Error(err))
		cancel()
		return err
	}

	// Wait for node to finish shutting down
	select {
	case <-time.After(5 * time.Second):
		logger.Info("Node shutdown timed out")
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("Error during shutdown", zap.Error(err))
			return err
		}
	}

	return nil
}
