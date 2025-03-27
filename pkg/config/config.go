package config

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	// Base configuration flags

	// FlagRootDir is a flag for specifying the root directory
	FlagRootDir = "home"
	// FlagDBPath is a flag for specifying the database path
	FlagDBPath = "db_path"
	// FlagChainConfigDir is a flag for specifying the chain config directory
	FlagChainConfigDir = "config_dir"

	// Node configuration flags

	// FlagAggregator is a flag for running node in aggregator mode
	FlagAggregator = "node.aggregator"
	// FlagLight is a flag for running the node in light mode
	FlagLight = "node.light"
	// FlagBlockTime is a flag for specifying the block time
	FlagBlockTime = "node.block_time"
	// FlagTrustedHash is a flag for specifying the trusted hash
	FlagTrustedHash = "node.trusted_hash"
	// FlagLazyAggregator is a flag for enabling lazy aggregation mode that only produces blocks when transactions are available
	FlagLazyAggregator = "node.lazy_aggregator"
	// FlagMaxPendingBlocks is a flag to limit and pause block production when too many blocks are waiting for DA confirmation
	FlagMaxPendingBlocks = "node.max_pending_blocks"
	// FlagLazyBlockTime is a flag for specifying the maximum interval between blocks in lazy aggregation mode
	FlagLazyBlockTime = "node.lazy_block_time"
	// FlagSequencerAddress is a flag for specifying the sequencer middleware address
	FlagSequencerAddress = "node.sequencer_address"
	// FlagSequencerRollupID is a flag for specifying the sequencer middleware rollup ID
	FlagSequencerRollupID = "node.sequencer_rollup_id"
	// FlagExecutorAddress is a flag for specifying the sequencer middleware address
	FlagExecutorAddress = "node.executor_address"

	// Data Availability configuration flags

	// FlagDAAddress is a flag for specifying the data availability layer address
	FlagDAAddress = "da.address"
	// FlagDAAuthToken is a flag for specifying the data availability layer auth token
	FlagDAAuthToken = "da.auth_token" // #nosec G101
	// FlagDABlockTime is a flag for specifying the data availability layer block time
	FlagDABlockTime = "da.block_time"
	// FlagDAGasPrice is a flag for specifying the data availability layer gas price
	FlagDAGasPrice = "da.gas_price"
	// FlagDAGasMultiplier is a flag for specifying the data availability layer gas price retry multiplier
	FlagDAGasMultiplier = "da.gas_multiplier"
	// FlagDAStartHeight is a flag for specifying the data availability layer start height
	FlagDAStartHeight = "da.start_height"
	// FlagDANamespace is a flag for specifying the DA namespace ID
	FlagDANamespace = "da.namespace"
	// FlagDASubmitOptions is a flag for data availability submit options
	FlagDASubmitOptions = "da.submit_options"
	// FlagDAMempoolTTL is a flag for specifying the DA mempool TTL
	FlagDAMempoolTTL = "da.mempool_ttl"

	// P2P configuration flags

	// FlagP2PListenAddress is a flag for specifying the P2P listen address
	FlagP2PListenAddress = "p2p.listen_address"
	// FlagP2PSeeds is a flag for specifying the P2P seeds
	FlagP2PSeeds = "p2p.seeds"
	// FlagP2PBlockedPeers is a flag for specifying the P2P blocked peers
	FlagP2PBlockedPeers = "p2p.blocked_peers"
	// FlagP2PAllowedPeers is a flag for specifying the P2P allowed peers
	FlagP2PAllowedPeers = "p2p.allowed_peers"

	// Instrumentation configuration flags

	// FlagPrometheus is a flag for enabling Prometheus metrics
	FlagPrometheus = "instrumentation.prometheus"
	// FlagPrometheusListenAddr is a flag for specifying the Prometheus listen address
	FlagPrometheusListenAddr = "instrumentation.prometheus_listen_addr"
	// FlagMaxOpenConnections is a flag for specifying the maximum number of open connections
	FlagMaxOpenConnections = "instrumentation.max_open_connections"
	// FlagPprof is a flag for enabling pprof profiling endpoints for runtime debugging
	FlagPprof = "instrumentation.pprof"
	// FlagPprofListenAddr is a flag for specifying the pprof listen address
	FlagPprofListenAddr = "instrumentation.pprof_listen_addr"

	// Logging configuration flags

	// FlagLogLevel is a flag for specifying the log level
	FlagLogLevel = "log.level"
	// FlagLogFormat is a flag for specifying the log format
	FlagLogFormat = "log.format"
	// FlagLogTrace is a flag for enabling stack traces in error logs
	FlagLogTrace = "log.trace"

	// RPC configuration flags

	// FlagRPCAddress is a flag for specifying the RPC server address
	FlagRPCAddress = "rpc.address"
	// FlagRPCPort is a flag for specifying the RPC server port
	FlagRPCPort = "rpc.port"
)

// DurationWrapper is a wrapper for time.Duration that implements encoding.TextMarshaler and encoding.TextUnmarshaler
// needed for YAML marshalling/unmarshalling especially for time.Duration
type DurationWrapper struct {
	time.Duration
}

// MarshalText implements encoding.TextMarshaler to format the duration as text
func (d DurationWrapper) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler to parse the duration from text
func (d *DurationWrapper) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}

// Config stores Rollkit configuration.
type Config struct {
	// Base configuration
	RootDir   string `mapstructure:"home" yaml:"home" comment:"Root directory where rollkit files are located"`
	DBPath    string `mapstructure:"db_path" yaml:"db_path" comment:"Path inside the root directory where the database is located"`
	ConfigDir string `mapstructure:"config_dir" yaml:"config_dir" comment:"Directory containing the rollup chain configuration"`

	// P2P configuration
	P2P P2PConfig `mapstructure:"p2p" yaml:"p2p"`

	// Node specific configuration
	Node NodeConfig `mapstructure:"node" yaml:"node"`

	// Data availability configuration
	DA DAConfig `mapstructure:"da" yaml:"da"`

	// RPC configuration
	RPC RPCConfig `mapstructure:"rpc" yaml:"rpc"`

	// Instrumentation configuration
	Instrumentation *InstrumentationConfig `mapstructure:"instrumentation" yaml:"instrumentation"`

	// Logging configuration
	Log LogConfig `mapstructure:"log" yaml:"log"`
}

// DAConfig contains all Data Availability configuration parameters
type DAConfig struct {
	Address       string          `mapstructure:"address" yaml:"address" comment:"Address of the data availability layer service (host:port). This is the endpoint where Rollkit will connect to submit and retrieve data."`
	AuthToken     string          `mapstructure:"auth_token" yaml:"auth_token" comment:"Authentication token for the data availability layer service. Required if the DA service needs authentication."`
	GasPrice      float64         `mapstructure:"gas_price" yaml:"gas_price" comment:"Gas price for data availability transactions. Use -1 for automatic gas price determination. Higher values may result in faster inclusion."`
	GasMultiplier float64         `mapstructure:"gas_multiplier" yaml:"gas_multiplier" comment:"Multiplier applied to gas price when retrying failed DA submissions. Values > 1 increase gas price on retries to improve chances of inclusion."`
	SubmitOptions string          `mapstructure:"submit_options" yaml:"submit_options" comment:"Additional options passed to the DA layer when submitting data. Format depends on the specific DA implementation being used."`
	Namespace     string          `mapstructure:"namespace" yaml:"namespace" comment:"Namespace ID used when submitting blobs to the DA layer."`
	BlockTime     DurationWrapper `mapstructure:"block_time" yaml:"block_time" comment:"Average block time of the DA chain (duration). Determines frequency of DA layer syncing, maximum backoff time for retries, and is multiplied by MempoolTTL to calculate transaction expiration. Examples: \"15s\", \"30s\", \"1m\", \"2m30s\", \"10m\"."`
	StartHeight   uint64          `mapstructure:"start_height" yaml:"start_height" comment:"Starting block height on the DA layer from which to begin syncing. Useful when deploying a new rollup on an existing DA chain."`
	MempoolTTL    uint64          `mapstructure:"mempool_ttl" yaml:"mempool_ttl" comment:"Number of DA blocks after which a transaction is considered expired and dropped from the mempool. Controls retry backoff timing."`
}

// NodeConfig contains all Rollkit specific configuration parameters
type NodeConfig struct {
	// Node mode configuration
	Aggregator bool `yaml:"aggregator" comment:"Run node in aggregator mode"`
	Light      bool `yaml:"light" comment:"Run node in light mode"`

	// Block management configuration
	BlockTime        DurationWrapper `mapstructure:"block_time" yaml:"block_time" comment:"Block time (duration). Examples: \"500ms\", \"1s\", \"5s\", \"1m\", \"2m30s\", \"10m\"."`
	MaxPendingBlocks uint64          `mapstructure:"max_pending_blocks" yaml:"max_pending_blocks" comment:"Maximum number of blocks pending DA submission. When this limit is reached, the aggregator pauses block production until some blocks are confirmed. Use 0 for no limit."`
	LazyAggregator   bool            `mapstructure:"lazy_aggregator" yaml:"lazy_aggregator" comment:"Enables lazy aggregation mode, where blocks are only produced when transactions are available or after LazyBlockTime. Optimizes resources by avoiding empty block creation during periods of inactivity."`
	LazyBlockTime    DurationWrapper `mapstructure:"lazy_block_time" yaml:"lazy_block_time" comment:"Maximum interval between blocks in lazy aggregation mode (LazyAggregator). Ensures blocks are produced periodically even without transactions to keep the chain active. Generally larger than BlockTime."`

	// Header configuration
	TrustedHash string `mapstructure:"trusted_hash" yaml:"trusted_hash" comment:"Initial trusted hash used to bootstrap the header exchange service. Allows nodes to start synchronizing from a specific trusted point in the chain instead of genesis. When provided, the node will fetch the corresponding header/block from peers using this hash and use it as a starting point for synchronization. If not provided, the node will attempt to fetch the genesis block instead."`

	// Sequencer configuration
	SequencerAddress  string `mapstructure:"sequencer_address" yaml:"sequencer_address" comment:"Address of the sequencer middleware (host:port). The sequencer is responsible for ordering transactions in the rollup. If not specified, a mock sequencer will be started at this address. Default: localhost:50051."`
	SequencerRollupID string `mapstructure:"sequencer_rollup_id" yaml:"sequencer_rollup_id" comment:"Unique identifier for the rollup chain used by the sequencer. This ID is used to identify the specific rollup when submitting transactions to and retrieving batches from the sequencer. If not specified, the chain ID from genesis will be used. Default: mock-rollup."`
	ExecutorAddress   string `mapstructure:"executor_address" yaml:"executor_address" comment:"Address of the executor middleware (host:port). The executor is responsible for processing transactions and maintaining the state of the rollup. Used for connecting to an external execution environment. Default: localhost:40041."`
}

// LogConfig contains all logging configuration parameters
type LogConfig struct {
	Level  string `mapstructure:"level" yaml:"level" comment:"Log level (debug, info, warn, error)"`
	Format string `mapstructure:"format" yaml:"format" comment:"Log format (text, json)"`
	Trace  bool   `mapstructure:"trace" yaml:"trace" comment:"Enable stack traces in error logs"`
}

// P2PConfig contains all peer-to-peer networking configuration parameters
type P2PConfig struct {
	ListenAddress string `mapstructure:"listen_address" yaml:"listen_address" comment:"Address to listen for incoming connections (host:port)"`
	Seeds         string `mapstructure:"seeds" yaml:"seeds" comment:"Comma separated list of seed nodes to connect to"`
	BlockedPeers  string `mapstructure:"blocked_peers" yaml:"blocked_peers" comment:"Comma separated list of peer IDs to block from connecting"`
	AllowedPeers  string `mapstructure:"allowed_peers" yaml:"allowed_peers" comment:"Comma separated list of peer IDs to allow connections from"`
}

// AddBasicFlags registers the basic configuration flags that are common across applications
// This includes logging configuration and root directory settings
func AddBasicFlags(cmd *cobra.Command, appName string) {
	cmd.PersistentFlags().String(FlagLogLevel, DefaultLogLevel, "Set the log level (debug, info, warn, error)")
	cmd.PersistentFlags().String(FlagLogFormat, "plain", "Set the log format (text, json)")
	cmd.PersistentFlags().Bool(FlagLogTrace, false, "Enable stack traces in error logs")
	cmd.PersistentFlags().String(FlagRootDir, DefaultRootDirWithName(appName), "Root directory for application data")
}

// RPCConfig contains all RPC server configuration parameters
type RPCConfig struct {
	Address string `mapstructure:"address" yaml:"address" comment:"Address to bind the RPC server to (host). Default: tcp://0.0.0.0"`
	Port    uint16 `mapstructure:"port" yaml:"port" comment:"Port to bind the RPC server to. Default: 26657"`
}

// AddFlags adds Rollkit specific configuration options to cobra Command.
func AddFlags(cmd *cobra.Command) {
	def := DefaultNodeConfig

	// Add CI flag for testing
	cmd.Flags().Bool("ci", false, "run node for ci testing")

	// Add base flags
	cmd.Flags().String(FlagRootDir, DefaultNodeConfig.RootDir, "root directory for config and data")
	cmd.Flags().String(FlagDBPath, DefaultNodeConfig.DBPath, "path for the node database")
	cmd.Flags().String(FlagChainConfigDir, DefaultNodeConfig.ConfigDir, "directory containing chain configuration files")

	// Node configuration flags
	cmd.Flags().BoolVar(&def.Node.Aggregator, FlagAggregator, def.Node.Aggregator, "run node in aggregator mode")
	cmd.Flags().Bool(FlagLight, def.Node.Light, "run light client")
	cmd.Flags().Duration(FlagBlockTime, def.Node.BlockTime.Duration, "block time (for aggregator mode)")
	cmd.Flags().String(FlagTrustedHash, def.Node.TrustedHash, "initial trusted hash to start the header exchange service")
	cmd.Flags().Bool(FlagLazyAggregator, def.Node.LazyAggregator, "produce blocks only when transactions are available or after lazy block time")
	cmd.Flags().Uint64(FlagMaxPendingBlocks, def.Node.MaxPendingBlocks, "maximum blocks pending DA confirmation before pausing block production (0 for no limit)")
	cmd.Flags().Duration(FlagLazyBlockTime, def.Node.LazyBlockTime.Duration, "maximum interval between blocks in lazy aggregation mode")
	cmd.Flags().String(FlagSequencerAddress, def.Node.SequencerAddress, "sequencer middleware address (host:port)")
	cmd.Flags().String(FlagSequencerRollupID, def.Node.SequencerRollupID, "sequencer middleware rollup ID (default: mock-rollup)")
	cmd.Flags().String(FlagExecutorAddress, def.Node.ExecutorAddress, "executor middleware address (host:port)")

	// Data Availability configuration flags
	cmd.Flags().String(FlagDAAddress, def.DA.Address, "DA address (host:port)")
	cmd.Flags().String(FlagDAAuthToken, def.DA.AuthToken, "DA auth token")
	cmd.Flags().Duration(FlagDABlockTime, def.DA.BlockTime.Duration, "DA chain block time (for syncing)")
	cmd.Flags().Float64(FlagDAGasPrice, def.DA.GasPrice, "DA gas price for blob transactions")
	cmd.Flags().Float64(FlagDAGasMultiplier, def.DA.GasMultiplier, "DA gas price multiplier for retrying blob transactions")
	cmd.Flags().Uint64(FlagDAStartHeight, def.DA.StartHeight, "starting DA block height (for syncing)")
	cmd.Flags().String(FlagDANamespace, def.DA.Namespace, "DA namespace to submit blob transactions")
	cmd.Flags().String(FlagDASubmitOptions, def.DA.SubmitOptions, "DA submit options")
	cmd.Flags().Uint64(FlagDAMempoolTTL, def.DA.MempoolTTL, "number of DA blocks until transaction is dropped from the mempool")

	// P2P configuration flags
	cmd.Flags().String(FlagP2PListenAddress, def.P2P.ListenAddress, "P2P listen address (host:port)")
	cmd.Flags().String(FlagP2PSeeds, def.P2P.Seeds, "Comma separated list of seed nodes to connect to")
	cmd.Flags().String(FlagP2PBlockedPeers, def.P2P.BlockedPeers, "Comma separated list of nodes to ignore")
	cmd.Flags().String(FlagP2PAllowedPeers, def.P2P.AllowedPeers, "Comma separated list of nodes to whitelist")

	// RPC configuration flags
	cmd.Flags().String(FlagRPCAddress, def.RPC.Address, "RPC server address (host)")
	cmd.Flags().Uint16(FlagRPCPort, def.RPC.Port, "RPC server port")

	// Instrumentation configuration flags
	instrDef := DefaultInstrumentationConfig()
	cmd.Flags().Bool(FlagPrometheus, instrDef.Prometheus, "enable Prometheus metrics")
	cmd.Flags().String(FlagPrometheusListenAddr, instrDef.PrometheusListenAddr, "Prometheus metrics listen address")
	cmd.Flags().Int(FlagMaxOpenConnections, instrDef.MaxOpenConnections, "maximum number of simultaneous connections for metrics")
	cmd.Flags().Bool(FlagPprof, instrDef.Pprof, "enable pprof HTTP endpoint")
	cmd.Flags().String(FlagPprofListenAddr, instrDef.PprofListenAddr, "pprof HTTP server listening address")
}

// LoadNodeConfig loads the node configuration in the following order of precedence:
// 1. DefaultNodeConfig (lowest priority)
// 2. YAML configuration file
// 3. Command line flags (highest priority)
func LoadNodeConfig(cmd *cobra.Command) (Config, error) {
	// Create a new Viper instance to avoid conflicts with any global Viper
	v := viper.New()

	// 1. Start with default configuration and set defaults in Viper
	config := DefaultNodeConfig
	setDefaultsInViper(v, config)

	// 2. Try to load YAML configuration from various locations
	// First try using the current directory
	v.SetConfigName(ConfigBaseName)
	v.SetConfigType(ConfigExtension)

	// Add search paths in order of precedence
	// Current directory
	v.AddConfigPath(".")

	// Check if RootDir is set in the default config
	if config.RootDir != "" {
		v.AddConfigPath(filepath.Join(config.RootDir, DefaultConfigDir))
	}

	// Try to read the config file
	if err := v.ReadInConfig(); err != nil {
		// If it's not a "file not found" error, return the error
		var configFileNotFound viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFound) {
			return config, fmt.Errorf("error reading YAML configuration: %w", err)
		}
		// Otherwise, just continue with defaults
	} else {
		// Config file found, log it
		fmt.Printf("Using config file: %s\n", v.ConfigFileUsed())
	}

	// 3. Bind command line flags
	if err := v.BindPFlags(cmd.Flags()); err != nil {
		return config, fmt.Errorf("unable to bind flags: %w", err)
	}

	// 4. Unmarshal everything from Viper into the config struct
	if err := v.Unmarshal(&config, func(c *mapstructure.DecoderConfig) {
		c.TagName = "mapstructure"
		c.DecodeHook = mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
			func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
				if t == reflect.TypeOf(DurationWrapper{}) && f.Kind() == reflect.String {
					if str, ok := data.(string); ok {
						duration, err := time.ParseDuration(str)
						if err != nil {
							return nil, err
						}
						return DurationWrapper{Duration: duration}, nil
					}
				}
				return data, nil
			},
		)
	}); err != nil {
		return config, fmt.Errorf("unable to decode configuration: %w", err)
	}

	return config, nil
}

// setDefaultsInViper sets all the default values from NodeConfig into Viper
func setDefaultsInViper(v *viper.Viper, config Config) {
	// Base configuration defaults
	v.SetDefault(FlagRootDir, config.RootDir)
	v.SetDefault(FlagDBPath, config.DBPath)
	v.SetDefault(FlagChainConfigDir, config.ConfigDir)

	// Node configuration defaults
	v.SetDefault(FlagAggregator, config.Node.Aggregator)
	v.SetDefault(FlagLight, config.Node.Light)
	v.SetDefault(FlagBlockTime, config.Node.BlockTime)
	v.SetDefault(FlagTrustedHash, config.Node.TrustedHash)
	v.SetDefault(FlagLazyAggregator, config.Node.LazyAggregator)
	v.SetDefault(FlagMaxPendingBlocks, config.Node.MaxPendingBlocks)
	v.SetDefault(FlagLazyBlockTime, config.Node.LazyBlockTime)
	v.SetDefault(FlagSequencerAddress, config.Node.SequencerAddress)
	v.SetDefault(FlagSequencerRollupID, config.Node.SequencerRollupID)
	v.SetDefault(FlagExecutorAddress, config.Node.ExecutorAddress)

	// Data Availability configuration defaults
	v.SetDefault(FlagDAAddress, config.DA.Address)
	v.SetDefault(FlagDAAuthToken, config.DA.AuthToken)
	v.SetDefault(FlagDABlockTime, config.DA.BlockTime)
	v.SetDefault(FlagDAGasPrice, config.DA.GasPrice)
	v.SetDefault(FlagDAGasMultiplier, config.DA.GasMultiplier)
	v.SetDefault(FlagDAStartHeight, config.DA.StartHeight)
	v.SetDefault(FlagDANamespace, config.DA.Namespace)
	v.SetDefault(FlagDASubmitOptions, config.DA.SubmitOptions)
	v.SetDefault(FlagDAMempoolTTL, config.DA.MempoolTTL)

	// P2P configuration defaults
	v.SetDefault(FlagP2PListenAddress, config.P2P.ListenAddress)
	v.SetDefault(FlagP2PSeeds, config.P2P.Seeds)
	v.SetDefault(FlagP2PBlockedPeers, config.P2P.BlockedPeers)
	v.SetDefault(FlagP2PAllowedPeers, config.P2P.AllowedPeers)

	// RPC configuration defaults
	v.SetDefault(FlagRPCAddress, config.RPC.Address)
	v.SetDefault(FlagRPCPort, config.RPC.Port)

	// Instrumentation configuration defaults
	if config.Instrumentation != nil {
		v.SetDefault(FlagPrometheus, config.Instrumentation.Prometheus)
		v.SetDefault(FlagPrometheusListenAddr, config.Instrumentation.PrometheusListenAddr)
		v.SetDefault(FlagMaxOpenConnections, config.Instrumentation.MaxOpenConnections)
		v.SetDefault(FlagPprof, config.Instrumentation.Pprof)
		v.SetDefault(FlagPprofListenAddr, config.Instrumentation.PprofListenAddr)
	}

	// Logging configuration defaults
	v.SetDefault(FlagLogLevel, "info")
	v.SetDefault(FlagLogFormat, "")
	v.SetDefault(FlagLogTrace, false)
}
