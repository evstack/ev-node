package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	rollconf "github.com/evstack/ev-node/pkg/config"
)

func TestInitCommand(t *testing.T) {
	// Create a temporary directory for testing
	dir, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)

	// Save current directory to restore it later
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Chdir(originalDir))
	}()

	// Change to the temporary directory
	require.NoError(t, os.Chdir(dir))

	// Remove any existing rollkit.yaml files in the test directory
	configPath := filepath.Join(dir, "config", rollconf.ConfigName)
	_ = os.Remove(configPath) // Ignore error if file doesn't exist

	// Create a new test-specific command
	cmd := &cobra.Command{
		Use: "test",
	}

	// Add init command as subcommand
	initCmd := InitCmd() // Create a copy to avoid affecting other tests
	cmd.AddCommand(initCmd)

	// Register all persistent flags from root command
	rollconf.AddGlobalFlags(cmd, "testapp")

	// Set home flag to the test directory
	cmd.SetArgs([]string{"init", "--home", dir, "--rollkit.node.aggregator", "--rollkit.signer.passphrase", "test"})

	// Execute the command
	err = cmd.Execute()
	require.NoError(t, err)

	// Verify the config can be read
	cfg, err := rollconf.Load(cmd)
	require.NoError(t, err)
	require.NoError(t, cfg.Validate())

	// Read the file content directly to verify the YAML structure
	//nolint:gosec // This is a test file and we control the input
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)
	yamlContent := string(content)

	// Verify that the YAML file content contains the expected values
	// Group verifications by category

	// Verify time values
	require.Contains(t, yamlContent, "block_time: ")
	require.Contains(t, yamlContent, "1s")

	// Verify that the YAML contains the da section
	require.Contains(t, yamlContent, "da:")
	require.Contains(t, yamlContent, "block_time: ")
	require.Contains(t, yamlContent, "15s")
	require.Contains(t, yamlContent, "lazy_block_interval: ")
	require.Contains(t, yamlContent, "1m0s")

	// Verify addresses
	require.Contains(t, yamlContent, "address: ")
	require.Contains(t, yamlContent, "http://localhost:7980")
}
