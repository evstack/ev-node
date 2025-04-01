package config

import (
	"os"
	"path/filepath"
	"time"
)

const (
	// DefaultDirPerm is the default permissions used when creating directories.
	DefaultDirPerm = 0750

	// DefaultConfigDir is the default directory for configuration files (e.g. rollkit.toml).
	DefaultConfigDir = "config"

	// DefaultDataDir is the default directory for data files (e.g. database).
	DefaultDataDir = "data"

	// DefaultListenAddress is a default listen address for P2P client.
	DefaultListenAddress = "/ip4/0.0.0.0/tcp/7676"
	// Version is the current rollkit version
	// Please keep updated with each new release
	Version = "0.38.5"
	// DefaultDAAddress is the default address for the data availability layer
	DefaultDAAddress = "http://localhost:26658"
	// DefaultSequencerAddress is the default address for the sequencer middleware
	DefaultSequencerAddress = "localhost:50051"
	// DefaultSequencerRollupID is the default rollup ID for the sequencer middleware
	DefaultSequencerRollupID = "mock-rollup"
	// DefaultExecutorAddress is the default address for the executor middleware
	DefaultExecutorAddress = "localhost:40041"
	// DefaultLogLevel is the default log level for the application
	DefaultLogLevel = "info"
)

// DefaultRootDir returns the default root directory for rollkit
func DefaultRootDir() string {
	return DefaultRootDirWithName("rollkit")
}

// DefaultRootDirWithName returns the default root directory for an application,
// based on the app name and the user's home directory
func DefaultRootDirWithName(appName string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "."+appName)
}

// DefaultNodeConfig keeps default values of NodeConfig
var DefaultNodeConfig = Config{
	RootDir:   DefaultRootDir(),
	DBPath:    DefaultDataDir,
	ConfigDir: DefaultConfigDir,
	ChainID:   "rollkit-test",
	P2P: P2PConfig{
		ListenAddress: DefaultListenAddress,
		Seeds:         "",
	},
	Node: NodeConfig{
		Aggregator:        false,
		BlockTime:         DurationWrapper{1 * time.Second},
		LazyAggregator:    false,
		LazyBlockTime:     DurationWrapper{60 * time.Second},
		Light:             false,
		TrustedHash:       "",
		SequencerAddress:  DefaultSequencerAddress,
		SequencerRollupID: DefaultSequencerRollupID,
		ExecutorAddress:   DefaultExecutorAddress,
	},
	DA: DAConfig{
		Address:       DefaultDAAddress,
		BlockTime:     DurationWrapper{15 * time.Second},
		GasPrice:      -1,
		GasMultiplier: 0,
	},
	Instrumentation: DefaultInstrumentationConfig(),
	Log: LogConfig{
		Level:  DefaultLogLevel,
		Format: "",
		Trace:  false,
	},
	Signer: SignerConfig{
		SignerType: "file",
		SignerPath: DefaultConfigDir,
	},
	RPC: RPCConfig{
		Address: "127.0.0.1",
		Port:    7331,
	},
}
