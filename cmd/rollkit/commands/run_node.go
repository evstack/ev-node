package commands

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	"cosmossdk.io/log"
	cmtcmd "github.com/cometbft/cometbft/cmd/cometbft/commands"
	cometconf "github.com/cometbft/cometbft/config"
	cometcli "github.com/cometbft/cometbft/libs/cli"
	cometnode "github.com/cometbft/cometbft/node"
	cometp2p "github.com/cometbft/cometbft/p2p"
	cometprivval "github.com/cometbft/cometbft/privval"
	comettypes "github.com/cometbft/cometbft/types"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	rollconf "github.com/rollkit/rollkit/config"
	"github.com/rollkit/rollkit/node"
	rolltypes "github.com/rollkit/rollkit/types"
)

var (
	// initialize the config with the cometBFT defaults
	config = cometconf.DefaultConfig()

	// initialize the rollkit node configuration
	nodeConfig = rollconf.DefaultNodeConfig
)

// NewRunNodeCmd returns the command that allows the CLI to start a node.
func NewRunNodeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "start",
		Aliases: []string{"node", "run"},
		Short:   "Run the rollkit node",
		// PersistentPreRunE is used to parse the config and initial the config files
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			err := parseConfig(cmd)
			if err != nil {
				return err
			}

			// use aggregator by default if the flag is not specified explicitly
			if !cmd.Flags().Lookup("rollkit.aggregator").Changed {
				nodeConfig.Aggregator = true
			}

			// Initialize the config files
			return initFiles(log.NewNopLogger()) //TODO: place the correct logger
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			genDocProvider := cometnode.DefaultGenesisDocProviderFunc(config)
			genDoc, err := genDocProvider()
			if err != nil {
				return err
			}
			nodeKey, err := cometp2p.LoadOrGenNodeKey(config.NodeKeyFile())
			if err != nil {
				return err
			}
			pval := cometprivval.LoadOrGenFilePV(config.PrivValidatorKeyFile(), config.PrivValidatorStateFile())
			p2pKey, err := rolltypes.GetNodeKey(nodeKey)
			if err != nil {
				return err
			}
			signingKey, err := rolltypes.GetNodeKey(&cometp2p.NodeKey{PrivKey: pval.Key.PrivKey})
			if err != nil {
				return err
			}

			// default to socket connections for remote clients
			if len(config.ABCI) == 0 {
				config.ABCI = "socket"
			}

			// get the node configuration
			rollconf.GetNodeConfig(&nodeConfig, config)
			if err := rollconf.TranslateAddresses(&nodeConfig); err != nil {
				return err
			}

			// initialize the metrics
			metrics := node.DefaultMetricsProvider(cometconf.DefaultInstrumentationConfig())

			// Determine which rollupID to use. If the flag has been set we want to use that value and ensure that the chainID in the genesis doc matches.
			if cmd.Flags().Lookup(rollconf.FlagSequencerRollupID).Changed {
				genDoc.ChainID = nodeConfig.Sequencer.RollupID
			}

			// initialize cleaner
			logger := log.NewLogger(os.Stdout)

			// Update log format if the flag is set
			// if config.LogFormat == cometconf.LogFormatJSON {
			// 	logger = cometlog.NewTMJSONLogger(cometlog.NewSyncWriter(os.Stdout))
			// }

			// // Parse the log level
			// logger, err = cometflags.ParseLogLevel(config.LogLevel, logger, cometconf.DefaultLogLevel)
			// if err != nil {
			// 	return err
			// }

			// // Add tracing to the logger if the flag is set
			// if viper.GetBool(cometcli.TraceFlag) {
			// 	logger = cometlog.NewTracingLogger(logger)
			// }

			logger = logger.With("module", "main")

			logger.Info("Executor address", "address", nodeConfig.ExecutorAddress)

			// use noop proxy app by default
			if !cmd.Flags().Lookup("proxy_app").Changed {
				config.ProxyApp = "noop"
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			dummyExecutor := node.NewDummyExecutor()
			dummySequencer := node.NewDummySequencer()

			// create the rollkit node
			rollnode, err := node.NewNode(
				ctx,
				nodeConfig,
				// THIS IS FOR TESTING ONLY
				dummyExecutor,
				dummySequencer,
				p2pKey,
				signingKey,
				genDoc,
				metrics,
				logger,
			)
			if err != nil {
				return fmt.Errorf("failed to create new rollkit node: %w", err)
			}

			// Start the node
			if err := rollnode.Start(ctx); err != nil {
				return fmt.Errorf("failed to start node: %w", err)
			}

			// TODO: Do rollkit nodes not have information about them? CometBFT has node.switch.NodeInfo()
			logger.Info("Started node")

			// Stop upon receiving SIGTERM or CTRL-C.
			trapSignal(logger, func() {
				if rollnode.IsRunning() {
					if err := rollnode.Stop(ctx); err != nil {
						logger.Error("unable to stop the node", "error", err)
					}
				}
			})

			// Check if we are running in CI mode
			inCI, err := cmd.Flags().GetBool("ci")
			if err != nil {
				return err
			}
			if !inCI {
				// Block forever to force user to stop node
				select {}
			}

			// CI mode. Wait for 5s and then verify the node is running before calling stop node.
			time.Sleep(5 * time.Second)
			if !rollnode.IsRunning() {
				return fmt.Errorf("node is not running")
			}

			return rollnode.Stop(ctx)
		},
	}

	addNodeFlags(cmd)

	return cmd
}

// addNodeFlags exposes some common configuration options on the command-line
// These are exposed for convenience of commands embedding a rollkit node
func addNodeFlags(cmd *cobra.Command) {
	// Add cometBFT flags
	cmtcmd.AddNodeFlags(cmd)

	cmd.Flags().String("transport", config.ABCI, "specify abci transport (socket | grpc)")
	cmd.Flags().Bool("ci", false, "run node for ci testing")

	// Add Rollkit flags
	rollconf.AddFlags(cmd)
}

// TODO (Ferret-san): modify so that it initiates files with rollkit configurations by default
// note that such a change would also require changing the cosmos-sdk
func initFiles(logger log.Logger) error {
	// Generate the private validator config files
	cometprivvalKeyFile := config.PrivValidatorKeyFile()
	cometprivvalStateFile := config.PrivValidatorStateFile()
	var pv *cometprivval.FilePV
	if fileExists(cometprivvalKeyFile) {
		pv = cometprivval.LoadFilePV(cometprivvalKeyFile, cometprivvalStateFile)
		logger.Info("Found private validator", "keyFile", cometprivvalKeyFile,
			"stateFile", cometprivvalStateFile)
	} else {
		pv = cometprivval.GenFilePV(cometprivvalKeyFile, cometprivvalStateFile)
		pv.Save()
		logger.Info("Generated private validator", "keyFile", cometprivvalKeyFile,
			"stateFile", cometprivvalStateFile)
	}

	// Generate the node key config files
	nodeKeyFile := config.NodeKeyFile()
	if fileExists(nodeKeyFile) {
		logger.Info("Found node key", "path", nodeKeyFile)
	} else {
		if _, err := cometp2p.LoadOrGenNodeKey(nodeKeyFile); err != nil {
			return err
		}
		logger.Info("Generated node key", "path", nodeKeyFile)
	}

	// Generate the genesis file
	genFile := config.GenesisFile()
	if fileExists(genFile) {
		logger.Info("Found genesis file", "path", genFile)
	} else {
		genDoc := comettypes.GenesisDoc{
			ChainID:         fmt.Sprintf("test-rollup-%08x", rand.Uint32()), //nolint:gosec
			GenesisTime:     time.Now().Round(0).UTC(),
			ConsensusParams: comettypes.DefaultConsensusParams(),
		}
		pubKey, err := pv.GetPubKey()
		if err != nil {
			return fmt.Errorf("can't get pubkey: %w", err)
		}
		genDoc.Validators = []comettypes.GenesisValidator{{
			Address: pubKey.Address(),
			PubKey:  pubKey,
			Power:   1000,
			Name:    "Rollkit Sequencer",
		}}

		if err := genDoc.SaveAs(genFile); err != nil {
			return err
		}
		logger.Info("Generated genesis file", "path", genFile)
	}

	return nil
}

func parseConfig(cmd *cobra.Command) error {
	// Set the root directory for the config to the home directory
	home := os.Getenv("RKHOME")
	if home == "" {
		var err error
		home, err = cmd.Flags().GetString(cometcli.HomeFlag)
		if err != nil {
			return err
		}
	}
	config.RootDir = home

	// Validate the root directory
	cometconf.EnsureRoot(config.RootDir)

	// Validate the config
	if err := config.ValidateBasic(); err != nil {
		return fmt.Errorf("error in config file: %w", err)
	}

	// Parse the flags
	if err := parseFlags(cmd); err != nil {
		return err
	}

	return nil
}

func parseFlags(cmd *cobra.Command) error {
	v := viper.GetViper()
	if err := v.BindPFlags(cmd.Flags()); err != nil {
		return err
	}

	// unmarshal viper into config
	err := v.Unmarshal(&config, func(c *mapstructure.DecoderConfig) {
		c.TagName = "mapstructure"
		c.DecodeHook = mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		)
	})
	if err != nil {
		return fmt.Errorf("unable to decode command flags into config: %w", err)
	}

	// special handling for the p2p external address, due to inconsistencies in mapstructure and flag name
	if cmd.Flags().Lookup("p2p.external-address").Changed {
		config.P2P.ExternalAddress = viper.GetString("p2p.external-address")
	}

	// handle rollkit node configuration
	if err := nodeConfig.GetViperConfig(v); err != nil {
		return fmt.Errorf("unable to decode command flags into nodeConfig: %w", err)
	}

	return nil
}
