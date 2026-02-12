package config

import (
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

const (
	// ConfigFileName is the base name of the evolve configuration file without extension.
	ConfigFileName = "evnode"
	// ConfigExtension is the file extension for the configuration file without the leading dot.
	ConfigExtension = "yaml"
	// ConfigPath is the filename for the evolve configuration file.
	ConfigName = ConfigFileName + "." + ConfigExtension
	// AppConfigDir is the directory name for the app configuration.
	AppConfigDir = "config"

	defaultRecoveryHistoryDepth = uint64(0)
)

// DefaultRootDir returns the default root directory for evolve
var DefaultRootDir = DefaultRootDirWithName(ConfigFileName)

// DefaultRootDirWithName returns the default root directory for an application,
// based on the app name and the user's home directory
func DefaultRootDirWithName(appName string) string {
	if appName == "" {
		appName = ConfigFileName
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, "."+appName)
}

// calculateReadinessMaxBlocksBehind calculates how many blocks represent the readiness window
// based on the given block time and window duration in seconds. This allows for normal
// batch-sync latency while detecting stuck nodes.
func calculateReadinessMaxBlocksBehind(blockTime time.Duration, windowSeconds uint64) uint64 {
	if blockTime == 0 {
		return 30 // fallback to safe default if blockTime is not set
	}
	if windowSeconds == 0 {
		windowSeconds = 15 // fallback to default 15s window
	}
	return uint64(time.Duration(windowSeconds) * time.Second / blockTime)
}

// DefaultConfig keeps default values of NodeConfig
func DefaultConfig() Config {
	defaultBlockTime := DurationWrapper{1 * time.Second}
	defaultReadinessWindowSeconds := uint64(15)
	return Config{
		RootDir: DefaultRootDir,
		DBPath:  "data",
		P2P: P2PConfig{
			ListenAddress: "/ip4/0.0.0.0/tcp/7676",
			Peers:         "",
		},
		Node: NodeConfig{
			Aggregator:               false,
			BlockTime:                defaultBlockTime,
			LazyMode:                 false,
			LazyBlockInterval:        DurationWrapper{60 * time.Second},
			Light:                    false,
			RecoveryHistoryDepth:     defaultRecoveryHistoryDepth,
			ReadinessWindowSeconds:   defaultReadinessWindowSeconds,
			ReadinessMaxBlocksBehind: calculateReadinessMaxBlocksBehind(defaultBlockTime.Duration, defaultReadinessWindowSeconds),
			ScrapeInterval:           DurationWrapper{1 * time.Second},
		},
		DA: DAConfig{
			Address:                  "http://localhost:7980",
			BlockTime:                DurationWrapper{6 * time.Second},
			MaxSubmitAttempts:        30,
			RequestTimeout:           DurationWrapper{60 * time.Second},
			Namespace:                randString(10),
			DataNamespace:            "",
			ForcedInclusionNamespace: "",
			BatchingStrategy:         "time",
			BatchSizeThreshold:       0.8,
			BatchMaxDelay:            DurationWrapper{0}, // 0 means use DA BlockTime
			BatchMinItems:            1,
		},
		Instrumentation: DefaultInstrumentationConfig(),
		Log: LogConfig{
			Level:  "info",
			Format: "text",
			Trace:  false,
		},
		Signer: SignerConfig{
			SignerType: "file",
			SignerPath: "config",
		},
		RPC: RPCConfig{
			Address:               "127.0.0.1:7331",
			EnableDAVisualization: false,
		},
		Raft: RaftConfig{
			SendTimeout:        200 * time.Millisecond,
			HeartbeatTimeout:   350 * time.Millisecond,
			LeaderLeaseTimeout: 175 * time.Millisecond,
			RaftDir:            filepath.Join(DefaultRootDir, "raft"),
		},
	}
}

func randString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	rng := rand.New(rand.NewSource(time.Now().Unix())) //nolint:gosec // even half random is good enough here.
	for i := range result {
		result[i] = charset[rng.Intn(len(charset))]
	}

	return string(result)
}
