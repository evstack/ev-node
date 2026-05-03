package main

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

const SetupFibreSessionName = "setup-fibre"

func setupFibreCmd() *cobra.Command {
	var (
		rootDir              string
		SSHKeyPath           string
		escrowAmount         string
		fibrePort            int
		fees                 string
		workers              int
		fibreAccounts        int
		encoderFibreAccounts int
	)

	cmd := &cobra.Command{
		Use:   "setup-fibre",
		Short: "Register fibre host addresses and fund escrow accounts on remote validators",
		Long:  "SSHes into each validator and runs transactions: register the fibre host address and fund escrow accounts for the validator and all fibre worker accounts.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := LoadConfig(rootDir)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if len(cfg.Validators) == 0 {
				return fmt.Errorf("no validators found in config")
			}

			resolvedSSHKeyPath := resolveValue(SSHKeyPath, EnvVarSSHKeyPath, strings.ReplaceAll(cfg.SSHPubKeyPath, ".pub", ""))

			sem := make(chan struct{}, workers)
			var (
				wg   sync.WaitGroup
				mu   sync.Mutex
				errs []error
			)

			for _, val := range cfg.Validators {
				// Build script: register host + deposit escrow for validator + all fibre accounts
				var sb strings.Builder

				// 0. Block until the chain has produced at least one block.
				// Without this, the very next tx returns
				// `celestia-app is not ready; please wait for first block`
				// from the local node — the call appears to succeed at
				// the CLI level (`--yes` returns the txhash before block
				// inclusion), but the tx never lands. Polling explicitly
				// avoids the `sleep 10` heuristic that used to be here.
				sb.WriteString(
					"echo 'waiting for chain to produce first block...'\n" +
						"DEADLINE=$(( $(date +%s) + 300 ))\n" +
						"while true; do\n" +
						"  H=$(celestia-appd status 2>/dev/null | " +
						"      grep -oE '\"latest_block_height\":\"[0-9]+\"' | " +
						"      grep -oE '[0-9]+' | head -1)\n" +
						"  if [ -n \"$H\" ] && [ \"$H\" -gt 0 ]; then\n" +
						"    echo \"chain is at height $H\"\n" +
						"    break\n" +
						"  fi\n" +
						"  if [ $(date +%s) -gt $DEADLINE ]; then\n" +
						"    echo 'FATAL: chain never produced a block within 5m' >&2\n" +
						"    exit 1\n" +
						"  fi\n" +
						"  sleep 3\n" +
						"done\n",
				)

				// 1. Register fibre host address. Plain `host:port` form —
				// x/valaddr requires it; the gRPC client dials it via the
				// passthrough resolver. Don't prefix `dns:///` here.
				//
				// Retry until `query valaddr providers` shows OUR host
				// — `--yes` returns the txhash before inclusion, so a
				// single one-shot call can succeed at the RPC layer
				// while the chain rejects the tx (mempool full, signer
				// not yet in validator set, …) and we'd never know.
				// 5-minute deadline so a stuck chain doesn't loop
				// forever.
				sb.WriteString(fmt.Sprintf(
					"HOST=%s:%d\n"+
						"DEADLINE=$(( $(date +%%s) + 300 ))\n"+
						"while true; do\n"+
						"  celestia-appd tx valaddr set-host \"$HOST\" "+
						"--from validator --keyring-backend=test --home .celestia-app "+
						"--chain-id %s --fees %s --yes >/dev/null 2>&1 || true\n"+
						"  sleep 6\n"+
						"  if celestia-appd query valaddr providers --chain-id %s -o json 2>/dev/null \\\n"+
						"     | grep -q \"\\\"host\\\": *\\\"$HOST\\\"\"; then\n"+
						"    echo \"set-host confirmed: $HOST\"\n"+
						"    break\n"+
						"  fi\n"+
						"  if [ $(date +%%s) -gt $DEADLINE ]; then\n"+
						"    echo \"FATAL: set-host did not register $HOST after 5m\" >&2\n"+
						"    exit 1\n"+
						"  fi\n"+
						"  echo 'set-host pending, retrying...'\n"+
						"done\n",
					val.PublicIP, fibrePort,
					cfg.ChainID, fees,
					cfg.ChainID,
				))

				// 2. Deposit escrow for fibre-0 inside a retry loop.
				// Same silent-failure mode as set-host: `--yes` returns
				// the txhash before inclusion, so a single bounced tx
				// (mempool full, signer not yet propagated, …) leaves
				// the runner failing every upload with
				// `escrow account not found for signer …`. fibre-0 is
				// the one the runner actually signs with by default,
				// so it's the only one we hard-block on.
				sb.WriteString(fmt.Sprintf(
					"FIBRE0_ADDR=$(celestia-appd keys show fibre-0 --keyring-backend test --home .celestia-app -a)\n"+
						"DEADLINE=$(( $(date +%%s) + 300 ))\n"+
						"while true; do\n"+
						"  celestia-appd tx fibre deposit-to-escrow %s "+
						"--from fibre-0 --keyring-backend=test --home .celestia-app "+
						"--chain-id %s --fees %s --yes >/dev/null 2>&1 || true\n"+
						"  sleep 6\n"+
						"  if celestia-appd query fibre escrow-account \"$FIBRE0_ADDR\" --chain-id %s -o json 2>/dev/null \\\n"+
						"     | grep -q '\"found\":true'; then\n"+
						"    echo \"escrow confirmed for fibre-0 ($FIBRE0_ADDR)\"\n"+
						"    break\n"+
						"  fi\n"+
						"  if [ $(date +%%s) -gt $DEADLINE ]; then\n"+
						"    echo \"FATAL: fibre-0 escrow did not land after 5m\" >&2\n"+
						"    exit 1\n"+
						"  fi\n"+
						"  echo 'fibre-0 escrow pending, retrying...'\n"+
						"done\n",
					escrowAmount,
					cfg.ChainID, fees,
					cfg.ChainID,
				))

				// 3. Best-effort fund fibre-1..N. The runner only signs
				// with fibre-0 by default, so a missing one of these
				// doesn't block uploads — they exist as headroom for
				// future signer rotation.
				for i := 1; i < fibreAccounts; i++ {
					keyName := fmt.Sprintf("fibre-%d", i)
					sb.WriteString(fmt.Sprintf(
						"celestia-appd tx fibre deposit-to-escrow %s "+
							"--from %s --keyring-backend=test --home .celestia-app "+
							"--chain-id %s --fees %s --yes\n",
						escrowAmount,
						keyName,
						cfg.ChainID, fees,
					))
				}

				script := sb.String()

				sem <- struct{}{}
				wg.Add(1)
				go func(inst Instance, s string) {
					defer wg.Done()
					defer func() { <-sem }()

					fmt.Printf("Running setup-fibre on %s (%s) — registering host + %d escrow deposits\n", inst.Name, inst.PublicIP, fibreAccounts)
					if err := runScriptInTMux([]Instance{inst}, resolvedSSHKeyPath, s, SetupFibreSessionName, time.Minute*30); err != nil {
						mu.Lock()
						errs = append(errs, fmt.Errorf("%s: %w", inst.Name, err))
						mu.Unlock()
					}
				}(val, script)
			}

			wg.Wait()

			if len(errs) > 0 {
				return errors.Join(errs...)
			}

			fmt.Printf("Waiting for fibre setup to complete (%d accounts per validator)...\n", fibreAccounts)
			if err := waitForTmuxSessions(cfg.Validators, resolvedSSHKeyPath, SetupFibreSessionName, 10*time.Minute); err != nil {
				return fmt.Errorf("waiting for setup-fibre sessions: %w", err)
			}

			// CLI-side verification that every validator's host is on
			// the chain's provider list before we hand off to start-
			// fibre / fibre-bootstrap-evnode. The per-validator script
			// above already self-verifies its own host, but we
			// re-check here from a single vantage point so a
			// concurrent set-host race across validators surfaces
			// before downstream steps cache an empty registry.
			if len(cfg.Validators) > 0 {
				expected := len(cfg.Validators)
				queryHost := cfg.Validators[0].PublicIP
				queryCmd := fmt.Sprintf(
					"celestia-appd query valaddr providers --chain-id %s -o json 2>/dev/null | grep -o '\"host\"' | wc -l",
					cfg.ChainID,
				)
				deadline := time.Now().Add(5 * time.Minute)
				for {
					out, err := sshExec("root", queryHost, resolvedSSHKeyPath, queryCmd)
					if err == nil {
						count := 0
						_, _ = fmt.Sscanf(strings.TrimSpace(string(out)), "%d", &count)
						if count >= expected {
							break
						}
						fmt.Printf("  valaddr providers: %d/%d registered, retrying...\n", count, expected)
					}
					if time.Now().After(deadline) {
						return fmt.Errorf("only some validators registered as fibre providers within 5m — re-run setup-fibre")
					}
					time.Sleep(5 * time.Second)
				}
			}
			fmt.Println("Validator setup done!")

			// Deposit escrow for encoder accounts.
			// Each encoder runs deposit-to-escrow from its own machine using its
			// own keyring, broadcasting via the first validator's RPC endpoint.
			if len(cfg.Encoders) > 0 && len(cfg.Validators) > 0 {
				rpcNode := fmt.Sprintf("tcp://%s:26657", cfg.Validators[0].PublicIP)
				fmt.Printf("Setting up escrow for %d encoder(s) via %s...\n", len(cfg.Encoders), rpcNode)

				for _, enc := range cfg.Encoders {
					encIndex := extractIndexFromName(enc.Name)
					keyPrefix := fmt.Sprintf("enc%d", encIndex)
					nAccounts := encoderFibreAccounts

					var sb strings.Builder
					for i := range nAccounts {
						keyName := fmt.Sprintf("%s-%d", keyPrefix, i)
						sb.WriteString(fmt.Sprintf(
							"celestia-appd tx fibre deposit-to-escrow %s "+
								"--from %s --keyring-backend=test --home .celestia-app "+
								"--chain-id %s --fees %s --node %s --yes\n",
							escrowAmount,
							keyName,
							cfg.ChainID, fees, rpcNode,
						))
					}

					script := sb.String()
					fmt.Printf("Running escrow deposits on encoder %s (%s) — %d accounts\n", enc.Name, enc.PublicIP, nAccounts)
					if err := runScriptInTMux([]Instance{enc}, resolvedSSHKeyPath, script, SetupFibreSessionName, 30*time.Minute); err != nil {
						return fmt.Errorf("encoder %s escrow setup: %w", enc.Name, err)
					}
				}

				fmt.Printf("Waiting for encoder escrow deposits to complete...\n")
				if err := waitForTmuxSessions(cfg.Encoders, resolvedSSHKeyPath, SetupFibreSessionName, 15*time.Minute); err != nil {
					return fmt.Errorf("waiting for encoder setup-fibre sessions: %w", err)
				}
				fmt.Println("Encoder escrow setup done!")
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&rootDir, "directory", "d", ".", "root directory in which to initialize")
	cmd.Flags().StringVarP(&SSHKeyPath, "ssh-key-path", "k", "", "path to the user's SSH key")
	cmd.Flags().StringVar(&escrowAmount, "escrow-amount", "200000000000000utia", "amount to deposit into escrow")
	cmd.Flags().IntVar(&fibrePort, "fibre-port", 7980, "fibre gRPC port on validators")
	cmd.Flags().StringVar(&fees, "fees", "5000utia", "transaction fees")
	cmd.Flags().IntVarP(&workers, "workers", "w", 10, "number of validators to set up in parallel")
	cmd.Flags().IntVar(&fibreAccounts, "fibre-accounts", 100, "number of fibre worker accounts to deposit escrow for")
	cmd.Flags().IntVar(&encoderFibreAccounts, "encoder-fibre-accounts", 100, "number of fibre worker accounts per encoder instance")

	return cmd
}
