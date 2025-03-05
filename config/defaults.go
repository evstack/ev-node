package config

import (
	"os"
	"path/filepath"
	"time"
)

const (
	// DefaultListenAddress is a default listen address for P2P client.
	DefaultListenAddress = "/ip4/0.0.0.0/tcp/7676"
	// Version is the current rollkit version
	// Please keep updated with each new release
	Version = "0.38.5"
	// DefaultDAAddress is the default address for the DA middleware
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
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".rollkit")
}

// DefaultNodeConfig keeps default values of NodeConfig
var DefaultNodeConfig = NodeConfig{
	RootDir: DefaultRootDir(),
	DBPath:  "data",
	P2P: P2PConfig{
		ListenAddress: DefaultListenAddress,
		Seeds:         "",
	},
	Aggregator: false,
	BlockManagerConfig: BlockManagerConfig{
		BlockTime:      1 * time.Second,
		DABlockTime:    15 * time.Second,
		LazyAggregator: false,
		LazyBlockTime:  60 * time.Second,
	},
	DAAddress:       DefaultDAAddress,
	DAGasPrice:      -1,
	DAGasMultiplier: 0,
	Light:           false,
	HeaderConfig: HeaderConfig{
		TrustedHash: "",
	},
	Instrumentation:   DefaultInstrumentationConfig(),
	SequencerAddress:  DefaultSequencerAddress,
	SequencerRollupID: DefaultSequencerRollupID,
	ExecutorAddress:   DefaultExecutorAddress,
}
