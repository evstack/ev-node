package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStartFlags(t *testing.T) {
	startCmd := newStartCmd()

	err := startCmd.ParseFlags([]string{"--regular-matrix", "custom.json"})
	require.NoError(t, err)

	// Since we can't easily access the cfg inside newStartCmd's closure from here
	// without refactoring, we'll check if the flag is registered correctly.
	flag := startCmd.Flags().Lookup("regular-matrix")
	require.NotNil(t, flag)
	require.Equal(t, "custom.json", flag.Value.String())
}

func TestRunArgs(t *testing.T) {
	runCmd := newRunCmd()
	err := runCmd.Args(runCmd, []string{"matrix.json"})
	require.NoError(t, err)

	err = runCmd.Args(runCmd, []string{})
	require.Error(t, err)
}
