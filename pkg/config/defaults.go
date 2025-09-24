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

// DefaultConfig keeps default values of NodeConfig
func DefaultConfig() Config {
	return Config{
		RootDir: DefaultRootDir,
		DBPath:  "data",
		P2P: P2PConfig{
			ListenAddress: "/ip4/0.0.0.0/tcp/7676",
			Peers:         "",
		},
		Node: NodeConfig{
			Aggregator:               false,
			BlockTime:                DurationWrapper{1 * time.Second},
			LazyMode:                 false,
			LazyBlockInterval:        DurationWrapper{60 * time.Second},
			Light:                    false,
			TrustedHash:              "",
			ReadinessMaxBlocksBehind: 3,
		},
		DA: DAConfig{
			Address:           "http://localhost:7980",
			BlockTime:         DurationWrapper{6 * time.Second},
			GasPrice:          -1,
			GasMultiplier:     0,
			MaxSubmitAttempts: 30,
			Namespace:         randString(10),
			DataNamespace:     "",
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
