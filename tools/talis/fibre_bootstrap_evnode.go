package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

// fibreBootstrapEvnodeCmd wires the two operator-supplied dependencies
// that ev-node needs before its init script will start the daemon:
//
//  1. Bridge admin JWT (/root/bridge-jwt.txt on the bridge box, written
//     by bridge_init.sh).
//  2. cosmos-sdk file keyring with the Fibre payment account (lives at
//     /root/.celestia-app/keyring-test on validator-0, populated during
//     validator_init.sh + setup-fibre).
//
// Both get pulled to the operator's local machine first (keeps the
// transfers serial and observable), then pushed to every evnode-* in
// the config. After this command returns, evnode_init.sh's poll loop
// observes the files and starts the daemon.
//
// Run after `talis up && talis genesis && talis deploy && talis
// setup-fibre`. Idempotent — re-running just overwrites the files.
func fibreBootstrapEvnodeCmd() *cobra.Command {
	var (
		rootDir    string
		sshKeyPath string
		sshUser    string
		jwtTimeout time.Duration
	)

	cmd := &cobra.Command{
		Use:   "fibre-bootstrap-evnode",
		Short: "Pull bridge JWT + validator-0 keyring and push them to every ev-node instance",
		Long: `After deploy + setup-fibre, this command stitches the two operator-
supplied dependencies onto each ev-node box so its init script's poll
loop unblocks and starts the daemon. SSHes to bridge-0 + validator-0
to fetch, then SCPs to each evnode-*.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := LoadConfig(rootDir)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			if len(cfg.Bridges) == 0 {
				return fmt.Errorf("no bridges in config — run `talis add --type bridge` first")
			}
			if len(cfg.Validators) == 0 {
				return fmt.Errorf("no validators in config")
			}
			if len(cfg.Evnodes) == 0 {
				return fmt.Errorf("no evnodes in config — nothing to bootstrap")
			}

			resolvedSSHKeyPath := resolveSSHPrivateKey(sshKeyPath, cfg.SSHPubKeyPath)
			bridge := cfg.Bridges[0]
			validator := cfg.Validators[0]
			if bridge.PublicIP == "" || bridge.PublicIP == "TBD" {
				return fmt.Errorf("bridge-0 has no public IP — run `talis up` first")
			}
			if validator.PublicIP == "" || validator.PublicIP == "TBD" {
				return fmt.Errorf("validator-0 has no public IP — run `talis up` first")
			}

			tmpDir, err := os.MkdirTemp("", "talis-evnode-bootstrap-")
			if err != nil {
				return fmt.Errorf("create temp dir: %w", err)
			}
			defer os.RemoveAll(tmpDir)

			localJWT := filepath.Join(tmpDir, "bridge-jwt.txt")
			localKeyringRoot := filepath.Join(tmpDir, "keyring-fibre")
			if err := os.MkdirAll(localKeyringRoot, 0o700); err != nil {
				return fmt.Errorf("create local keyring root: %w", err)
			}

			// Pull JWT (poll-retry: bridge_init.sh writes it after
			// `celestia bridge auth admin`, which can take ~30s after
			// the bridge process starts).
			log.Printf("Fetching bridge JWT from bridge-0 (%s) — up to %s", bridge.PublicIP, jwtTimeout)
			deadline := time.Now().Add(jwtTimeout)
			for {
				if err := scpFromRemote(sshUser, bridge.PublicIP, resolvedSSHKeyPath, "/root/bridge-jwt.txt", localJWT, false); err == nil {
					if info, statErr := os.Stat(localJWT); statErr == nil && info.Size() > 0 {
						break
					}
				}
				if time.Now().After(deadline) {
					return fmt.Errorf("bridge JWT not ready at /root/bridge-jwt.txt within %s — check bridge tmux session: tmux attach -t bridge", jwtTimeout)
				}
				time.Sleep(5 * time.Second)
			}
			log.Printf("✓ pulled JWT to %s", localJWT)

			// Pull validator-0's keyring directory. The cosmos-sdk
			// file backend stores per-account keys under
			// keyring-test/, so we mirror that layout locally so the
			// outbound push lands at /root/keyring-fibre/keyring-test/
			// — exactly where evnode_init.sh expects it.
			log.Printf("Fetching keyring-test from validator-0 (%s)", validator.PublicIP)
			if err := scpFromRemote(sshUser, validator.PublicIP, resolvedSSHKeyPath, "/root/.celestia-app/keyring-test", localKeyringRoot, true); err != nil {
				return fmt.Errorf("scp keyring from validator-0: %w", err)
			}
			if _, err := os.Stat(filepath.Join(localKeyringRoot, "keyring-test")); err != nil {
				return fmt.Errorf("keyring-test directory not present after pull (got %s): %w", localKeyringRoot, err)
			}
			log.Printf("✓ pulled keyring to %s/keyring-test", localKeyringRoot)

			// Push to every ev-node in parallel. The init script's
			// poll loop checks every 5s, so a successful push here
			// means daemon startup within ~10s.
			var wg sync.WaitGroup
			errCh := make(chan error, len(cfg.Evnodes))
			for _, ev := range cfg.Evnodes {
				if ev.PublicIP == "" || ev.PublicIP == "TBD" {
					errCh <- fmt.Errorf("evnode %s has no public IP", ev.Name)
					continue
				}
				wg.Add(1)
				go func(ev Instance) {
					defer wg.Done()
					log.Printf("[%s] pushing JWT + keyring", ev.Name)

					// JWT is small + atomic on the receive side because
					// it's a single file, so we push it directly.
					if err := scpToRemote(sshUser, ev.PublicIP, resolvedSSHKeyPath, localJWT, "/root/bridge-jwt.txt", false); err != nil {
						errCh <- fmt.Errorf("[%s] push JWT: %w", ev.Name, err)
						return
					}

					// Keyring push is staged through a tmp dir and
					// promoted via mv. Without staging, evnode_init.sh's
					// poll loop (which tests `[ -d keyring-test ]`)
					// passes the moment scp -r mkdir's the directory,
					// long before fibre-0.info is on disk. evnode then
					// launches mid-scp and dies with `keyring entry
					// "fibre-0" not found`. mv is atomic on the same
					// filesystem so the init script either sees nothing
					// (keep waiting) or the fully-populated dir (start
					// the daemon cleanly).
					stageDir := "/root/.keyring-fibre.staging"
					prep := fmt.Sprintf(
						"rm -rf %s && mkdir -p %s && mkdir -p /root/keyring-fibre && rm -rf /root/keyring-fibre/keyring-test",
						stageDir, stageDir,
					)
					if _, err := sshExec(sshUser, ev.PublicIP, resolvedSSHKeyPath, prep); err != nil {
						errCh <- fmt.Errorf("[%s] stage keyring: %w", ev.Name, err)
						return
					}
					stageDest := stageDir + "/keyring-test"
					if err := scpToRemote(sshUser, ev.PublicIP, resolvedSSHKeyPath, filepath.Join(localKeyringRoot, "keyring-test"), stageDest, true); err != nil {
						errCh <- fmt.Errorf("[%s] push keyring: %w", ev.Name, err)
						return
					}
					promote := fmt.Sprintf(
						"mv %s /root/keyring-fibre/keyring-test && rmdir %s",
						stageDest, stageDir,
					)
					if _, err := sshExec(sshUser, ev.PublicIP, resolvedSSHKeyPath, promote); err != nil {
						errCh <- fmt.Errorf("[%s] promote keyring: %w", ev.Name, err)
						return
					}

					log.Printf("[%s] ✓ pushed; daemon should start within ~10s", ev.Name)
				}(ev)
			}
			wg.Wait()
			close(errCh)
			var errs []error
			for e := range errCh {
				errs = append(errs, e)
			}
			if len(errs) > 0 {
				for _, e := range errs {
					log.Println(e)
				}
				return fmt.Errorf("%d evnode(s) failed to bootstrap", len(errs))
			}
			log.Printf("✓ bootstrap complete for %d evnode(s)", len(cfg.Evnodes))
			return nil
		},
	}

	homeDir, _ := os.UserHomeDir()
	defaultKeyPath := filepath.Join(homeDir, ".ssh", "id_ed25519")

	cmd.Flags().StringVarP(&rootDir, "directory", "d", ".", "experiment root directory")
	cmd.Flags().StringVarP(&sshKeyPath, "ssh-key-path", "s", defaultKeyPath, "SSH private key for talis instances")
	cmd.Flags().StringVar(&sshUser, "ssh-user", "root", "SSH user (talis instances boot as root)")
	cmd.Flags().DurationVar(&jwtTimeout, "jwt-timeout", 5*time.Minute, "max wall time to wait for the bridge JWT to appear on bridge-0")

	return cmd
}

// scpFromRemote pulls a file or directory off a remote box. recursive=true
// uses scp -r so directories transfer with their contents.
func scpFromRemote(user, host, sshKeyPath, remotePath, localPath string, recursive bool) error {
	args := []string{
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-i", sshKeyPath,
	}
	if recursive {
		args = append(args, "-r")
	}
	args = append(args, fmt.Sprintf("%s@%s:%s", user, host, remotePath), localPath)
	cmd := exec.Command("scp", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("scp pull: %w (%s)", err, string(out))
	}
	return nil
}

// scpToRemote pushes a file or directory onto a remote box.
func scpToRemote(user, host, sshKeyPath, localPath, remotePath string, recursive bool) error {
	args := []string{
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-i", sshKeyPath,
	}
	if recursive {
		args = append(args, "-r")
	}
	args = append(args, localPath, fmt.Sprintf("%s@%s:%s", user, host, remotePath))
	cmd := exec.Command("scp", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("scp push: %w (%s)", err, string(out))
	}
	return nil
}
