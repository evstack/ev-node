package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// fibreExperimentCmd is the one-command driver for the Fibre throughput
// experiment. It assumes the operator has already populated the
// experiment directory with a config.json + scripts/ + base config.toml +
// app.toml (i.e. ran `talis init` + `talis add` for validators / bridge
// / evnode / loadgen) and that build artefacts are at $rootDir/build.
//
// It then invokes — in order — the same subcommands the operator would
// run by hand:
//
//  1. up                       — provision instances
//  2. genesis -b <build>       — stage validator/bridge/evnode/loadgen payloads
//  3. deploy                   — ship payloads + start init scripts
//  4. setup-fibre              — register host + deposit escrow on each validator
//  5. start-fibre              — launch the fibre server on each validator
//  6. fibre-bootstrap-evnode   — scp bridge JWT + fibre keyring onto evnode-*
//
// Each step is invoked via os/exec on the running binary. Any failure
// surfaces immediately; nothing is retried at this layer (the
// individual subcommands handle their own waits + retries).
//
// After step 6 returns, evnode-* daemons start within ~10 s and the
// load-gen's init script auto-launches evnode-txsim. The operator
// reads the final TXSIM: line from the load-gen.
func fibreExperimentCmd() *cobra.Command {
	var (
		rootDir  string
		buildDir string
	)

	cmd := &cobra.Command{
		Use:   "fibre-experiment",
		Short: "End-to-end driver: up → genesis → deploy → setup-fibre → start-fibre → fibre-bootstrap-evnode",
		Long: `Run every step needed to bring up a Fibre throughput experiment from a
prepared root directory. Equivalent to invoking each subcommand in
sequence; included so the operator doesn't have to remember the order
or watch for inter-step races.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			self, err := os.Executable()
			if err != nil {
				return fmt.Errorf("locate own binary: %w", err)
			}

			sshKeyPath := resolveSSHPrivateKeyFromEnv()

			steps := []struct {
				name string
				args []string
			}{
				{"up", []string{"up", "-d", rootDir}},
				{"genesis", []string{"genesis", "-d", rootDir, "-b", buildDir}},
				{"deploy", []string{"deploy", "-d", rootDir, "--direct-payload-upload", "-s", sshKeyPath}},
				{"setup-fibre", []string{"setup-fibre", "-d", rootDir}},
				{"start-fibre", []string{"start-fibre", "-d", rootDir}},
				{"fibre-bootstrap-evnode", []string{"fibre-bootstrap-evnode", "-d", rootDir}},
			}

			for _, s := range steps {
				fmt.Printf("\n=== talis %s ===\n", s.name)
				c := exec.Command(self, s.args...)
				c.Stdout = os.Stdout
				c.Stderr = os.Stderr
				c.Env = os.Environ()
				if err := c.Run(); err != nil {
					return fmt.Errorf("step %q failed: %w", s.name, err)
				}
			}

			fmt.Println()
			fmt.Println("=== fibre-experiment complete ===")
			fmt.Println("evnode aggregator(s) start within ~10 s and load-gen init")
			fmt.Println("scripts auto-launch evnode-txsim once evnode's /stats responds.")
			fmt.Println("Final TXSIM: line lands at /root/txsim.log on each load-gen host.")
			return nil
		},
	}

	cmd.Flags().StringVarP(&rootDir, "directory", "d", ".", "experiment root directory")
	cmd.Flags().StringVarP(&buildDir, "build-dir", "b", "./build", "directory containing the cross-compiled linux/amd64 binaries")

	return cmd
}

func resolveSSHPrivateKeyFromEnv() string {
	if pubKeyPath := os.Getenv(EnvVarSSHKeyPath); pubKeyPath != "" {
		return strings.TrimSuffix(pubKeyPath, ".pub")
	}

	panic("SSH key path not set via " + EnvVarSSHKeyPath)
}
