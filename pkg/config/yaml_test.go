package config

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestYamlConfigOperations(t *testing.T) {
	testCases := []struct {
		name     string
		setup    func(t *testing.T, dir string) *Config
		validate func(t *testing.T, cfg *Config)
	}{
		{
			name: "Write and read custom config values",
			setup: func(t *testing.T, dir string) *Config {
				cfg := DefaultConfig()
				cfg.RootDir = dir
				cfg.P2P.ListenAddress = DefaultConfig().P2P.ListenAddress
				cfg.P2P.Peers = "seed1.example.com:26656,seed2.example.com:26656"

				require.NoError(t, cfg.SaveAsYaml())

				return &cfg
			},
			validate: func(t *testing.T, cfg *Config) {
				require.Equal(t, DefaultConfig().P2P.ListenAddress, cfg.P2P.ListenAddress)
				require.Equal(t, "seed1.example.com:26656,seed2.example.com:26656", cfg.P2P.Peers)
			},
		},
		{
			name: "Initialize default config values",
			setup: func(t *testing.T, dir string) *Config {
				cfg := DefaultConfig()
				cfg.RootDir = dir

				require.NoError(t, cfg.SaveAsYaml())

				return &cfg
			},
			validate: func(t *testing.T, cfg *Config) {
				require.Equal(t, DefaultConfig().P2P.ListenAddress, cfg.P2P.ListenAddress)
				require.Equal(t, DefaultConfig().P2P.Peers, cfg.P2P.Peers)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()

			// Setup the test case and write the initial config
			tc.setup(t, tempDir)

			// Read the config
			cmd := &cobra.Command{Use: "test"}
			AddFlags(cmd)
			AddGlobalFlags(cmd, "")
			args := []string{"--home=" + tempDir}
			cmd.SetArgs(args)
			require.NoError(t, cmd.ParseFlags(args))

			cfg, err := Load(cmd)
			require.NoError(t, err)
			require.NoError(t, cfg.Validate())

			// Validate the config
			tc.validate(t, &cfg)
		})
	}
}

// TestSaveAsYamlNestedComments ensures that SaveAsYaml emits field comments for
// deeply nested config sections. Regression test: comments for sections nested
// three or more levels deep (e.g. signer.kms.*) and for pointer-to-struct
// sections (e.g. instrumentation) were previously dropped because the comment
// path lost its parent prefix and pointer fields were never traversed.
func TestSaveAsYamlNestedComments(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig()
	cfg.RootDir = dir

	require.NoError(t, cfg.SaveAsYaml())

	data, err := os.ReadFile(cfg.ConfigPath())
	require.NoError(t, err)
	yaml := string(data)

	// Each entry: the comment text and the yaml key it must sit directly above.
	cases := []struct {
		comment string
		key     string
	}{
		// pointer-to-struct section (instrumentation)
		{"Enable Prometheus metrics", "prometheus:"},
		// signer.kms (nested three levels deep)
		{"KMS provider for signer type 'kms' (aws, gcp)", "provider:"},
		// signer.kms.aws (nested four levels deep)
		{"AWS KMS Key ID or ARN", "key_id:"},
		// da.tx_signer (nested three levels deep)
		{"Sign blob submissions locally and broadcast them over consensus gRPC instead of calling blob.Submit on the DA node.", "enabled:"},
	}

	for _, tc := range cases {
		require.Contains(t, yaml, tc.comment, "missing comment %q in generated config", tc.comment)
		require.True(t, commentPrecedesKey(yaml, tc.comment, tc.key),
			"comment %q is not placed directly above key %q", tc.comment, tc.key)
	}
}

// commentPrecedesKey reports whether a line containing the comment text is
// immediately followed (ignoring blank lines) by a line whose first token is
// the given yaml key.
func commentPrecedesKey(yaml, comment, key string) bool {
	lines := strings.Split(yaml, "\n")
	for i, line := range lines {
		if !strings.Contains(line, comment) {
			continue
		}
		for j := i + 1; j < len(lines); j++ {
			trimmed := strings.TrimSpace(lines[j])
			if trimmed == "" {
				continue
			}
			return strings.HasPrefix(trimmed, key)
		}
	}
	return false
}
