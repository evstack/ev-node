package main

import (
	"fmt"
	"os"

	cmds "github.com/evstack/ev-node/apps/testapp/cmd"
	rollcmd "github.com/evstack/ev-node/pkg/cmd"
	"github.com/evstack/ev-node/pkg/config"
)

func main() {
	// Initiate the root command
	rootCmd := cmds.RootCmd
	initCmd := cmds.InitCmd()

	// Add configuration flags to backup and restore commands
	backupCmd := rollcmd.NewBackupCmd()
	config.AddFlags(backupCmd)
	restoreCmd := rollcmd.NewRestoreCmd()
	config.AddFlags(restoreCmd)

	// Add subcommands to the root command
	rootCmd.AddCommand(
		cmds.RunCmd,
		rollcmd.VersionCmd,
		rollcmd.NetInfoCmd,
		rollcmd.StoreUnsafeCleanCmd,
		rollcmd.KeysCmd(),
		cmds.NewRollbackCmd(),
		backupCmd,
		restoreCmd,
		initCmd,
	)

	if err := rootCmd.Execute(); err != nil {
		// Print to stderr and exit with error
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
