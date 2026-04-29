package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/celestiaorg/celestia-app/v9/pkg/appconsts"
	"github.com/celestiaorg/celestia-app/v9/test/util/genesis"
	"github.com/spf13/cobra"
)

const (
	chainIDFlag = "chainID"
	rootDirFlag = "directory"
)

// generateCmd is the Cobra command for creating the payload for the experiment.
func generateCmd() *cobra.Command {
	var (
		rootDir                       string
		chainID                       string // will overwrite that in the config
		squareSize                    int
		buildDirPath                  string
		appBinaryPath                 string
		nodeBinaryPath                string
		txsimBinaryPath               string
		latencyMonitorBinaryPath      string
		fibreBinaryPath               string
		fibreTxsimBinaryPath          string
		observabilityDirPath          string
		useMainnetStakingDistribution bool
		fibreAccounts                 int
		encoderFibreAccounts          int
	)
	cmd := &cobra.Command{
		Use:   "genesis",
		Short: "Create a genesis for the network.",
		Long:  "Create a genesis for the network along with everything else needed to start the network. Call this only after init and add.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := LoadConfig(rootDir)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if chainID != "" {
				cfg = cfg.WithChainID(chainID)
			}

			payloadDir := filepath.Join(rootDir, "payload")

			if err := os.RemoveAll(payloadDir); err != nil {
				return fmt.Errorf("failed to remove old payload directory: %w", err)
			}
			if err := os.RemoveAll(filepath.Join(rootDir, "encoder-payload")); err != nil {
				return fmt.Errorf("failed to remove old encoder-payload directory: %w", err)
			}

			err = createPayload(cfg.Validators, cfg.Encoders, cfg.ChainID, payloadDir, squareSize, useMainnetStakingDistribution, fibreAccounts, encoderFibreAccounts)
			if err != nil {
				log.Fatalf("Failed to create payload: %v", err)
			}

			srcAppConfig := filepath.Join(rootDir, "app.toml")

			for _, v := range cfg.Validators {
				valDir := filepath.Join(payloadDir, v.Name)
				// Note: per-validator config.toml is written by Network.InitNodes
				// with the correct persistent_peers list. Don't overwrite it
				// here — that would clobber the peer list and the chain comes
				// up with zero peers.

				if err := copyFile(srcAppConfig, filepath.Join(valDir, "app.toml"), 0o755); err != nil {
					return fmt.Errorf("failed to copy app.toml: %w", err)
				}
			}

			if err := copyDir(filepath.Join(rootDir, "scripts"), filepath.Join(rootDir, "payload")); err != nil {
				return fmt.Errorf("failed to copy scripts: %w", err)
			}

			buildDest := filepath.Join(payloadDir, "build")
			if buildDirPath != "" {
				info, err := os.Stat(buildDirPath)
				if err != nil {
					return fmt.Errorf("failed to stat build directory %q: %w", buildDirPath, err)
				}
				if !info.IsDir() {
					return fmt.Errorf("build path %q is not a directory", buildDirPath)
				}
				if err := copyDir(buildDirPath, buildDest); err != nil {
					return fmt.Errorf("failed to copy build directory: %w", err)
				}
			} else {
				if err := copyFile(appBinaryPath, filepath.Join(buildDest, "celestia-appd"), 0o755); err != nil {
					return fmt.Errorf("failed to copy app binary: %w", err)
				}

				if err := copyFile(nodeBinaryPath, filepath.Join(buildDest, "celestia"), 0o755); err != nil {
					log.Println("failed to copy celestia binary, bridge and light nodes will not be able to start")
				}

				if err := copyFile(txsimBinaryPath, filepath.Join(buildDest, "txsim"), 0o755); err != nil {
					return fmt.Errorf("failed to copy txsim binary: %w", err)
				}

				// Copy latency monitor binary
				if err := copyFile(latencyMonitorBinaryPath, filepath.Join(buildDest, "latency-monitor"), 0o755); err != nil {
					log.Printf("failed to copy latency monitor binary: %v", err)
				}

				// Copy fibre server binary
				if err := copyFile(fibreBinaryPath, filepath.Join(buildDest, "fibre"), 0o755); err != nil {
					log.Printf("failed to copy fibre binary: %v", err)
				}

				// Copy fibre-txsim binary
				if err := copyFile(fibreTxsimBinaryPath, filepath.Join(buildDest, "fibre-txsim"), 0o755); err != nil {
					log.Printf("failed to copy fibre-txsim binary: %v", err)
				}
			}

			if err := writeAWSEnv(filepath.Join(payloadDir, "vars.sh"), cfg); err != nil {
				return fmt.Errorf("failed to write aws env: %w", err)
			}

			if err := stageObservabilityPayload(cfg, observabilityDirPath, payloadDir); err != nil {
				return fmt.Errorf("failed to stage observability payload: %w", err)
			}

			// Stage encoder payload: copy binaries, genesis, and vars to the
			// encoder-payload directory so deploy can create a lightweight tar.
			if len(cfg.Encoders) > 0 {
				if err := stageEncoderPayload(rootDir, payloadDir, appBinaryPath, fibreTxsimBinaryPath, buildDirPath); err != nil {
					return fmt.Errorf("failed to stage encoder payload: %w", err)
				}
			}

			// Stage bridge payload: celestia-node binary + genesis + init
			// script. Each bridge points at validator-0's RPC for header
			// sync; talis up has already populated cfg.Validators[0].PublicIP.
			if len(cfg.Bridges) > 0 {
				if len(cfg.Validators) == 0 {
					return fmt.Errorf("bridges configured but no validators — bring up validators first")
				}
				if err := stageBridgePayload(rootDir, payloadDir, nodeBinaryPath, buildDirPath, cfg); err != nil {
					return fmt.Errorf("failed to stage bridge payload: %w", err)
				}
			}

			// Stage ev-node payload: evnode binary + templated init script.
			// ev-node needs the bridge JWT + a funded fibre keyring, both
			// of which are scp'd in a separate `talis fibre-bootstrap-evnode`
			// step (or by hand) — the init script polls for them and only
			// starts the daemon once they exist.
			if len(cfg.Evnodes) > 0 {
				if len(cfg.Validators) == 0 {
					return fmt.Errorf("evnodes configured but no validators — bring up validators first")
				}
				if len(cfg.Bridges) == 0 {
					return fmt.Errorf("evnodes configured but no bridges — at least one bridge is required")
				}
				if err := stageEvnodePayload(rootDir, payloadDir, buildDirPath, cfg); err != nil {
					return fmt.Errorf("failed to stage evnode payload: %w", err)
				}
			}

			// Stage loadgen payload: evnode-txsim binary + init script
			// templated with evnode-0's HTTP endpoint as the target.
			if len(cfg.Loadgens) > 0 {
				if len(cfg.Evnodes) == 0 {
					return fmt.Errorf("loadgens configured but no evnodes — at least one ev-node is required")
				}
				if err := stageLoadgenPayload(rootDir, payloadDir, buildDirPath, cfg); err != nil {
					return fmt.Errorf("failed to stage loadgen payload: %w", err)
				}
			}

			return cfg.Save(rootDir)
		},
	}

	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			panic("failed to determine home dir: " + err.Error())
		}
		gopath = filepath.Join(home, "go")
	}
	gopath = filepath.Join(gopath, "bin")

	cmd.Flags().StringVarP(&chainID, chainIDFlag, "c", "", "Override the chainID in the config")
	cmd.Flags().StringVarP(&rootDir, rootDirFlag, "d", ".", "root directory in which to initialize (default is the current directory)")
	cmd.Flags().IntVarP(&squareSize, "ods-size", "s", appconsts.SquareSizeUpperBound, "The size of the ODS for the network (make sure to also build a celestia-app binary with a greater SquareSizeUpperBound)")
	cmd.Flags().StringVarP(&buildDirPath, "build-dir", "b", "", "directory containing binaries to include in the payload")
	cmd.Flags().StringVarP(&appBinaryPath, "app-binary", "a", filepath.Join(gopath, "celestia-appd"), "app binary to include in the payload (assumes the binary is installed")
	cmd.Flags().StringVarP(&nodeBinaryPath, "node-binary", "n", filepath.Join(gopath, "celestia"), "node binary to include in the payload (assumes the binary is installed")
	cmd.Flags().StringVarP(&txsimBinaryPath, "txsim-binary", "t", filepath.Join(gopath, "txsim"), "txsim binary to include in the payload (assumes the binary is installed)")
	cmd.Flags().StringVar(&latencyMonitorBinaryPath, "latency-monitor-binary", filepath.Join(gopath, "latency-monitor"), "latency monitor binary to include in the payload")
	cmd.Flags().StringVar(&fibreBinaryPath, "fibre-binary", filepath.Join(gopath, "fibre"), "fibre server binary to include in the payload")
	cmd.Flags().StringVar(&fibreTxsimBinaryPath, "fibre-txsim-binary", filepath.Join(gopath, "fibre-txsim"), "fibre-txsim binary to include in the payload")
	cmd.Flags().StringVar(&observabilityDirPath, "observability-dir", "", "path to observability directory containing docker-compose, Prometheus config, and scripts (required if observability nodes are configured)")
	cmd.Flags().BoolVarP(&useMainnetStakingDistribution, "mainnet-staking-distribution", "m", false, "replace the default uniform staking distribution with the actual mainnet distribution")
	cmd.Flags().IntVar(&fibreAccounts, "fibre-accounts", 100, "number of pre-funded fibre accounts to create per validator")
	cmd.Flags().IntVar(&encoderFibreAccounts, "encoder-fibre-accounts", 100, "number of pre-funded fibre accounts to create per encoder instance")

	return cmd
}

// createPayload takes ips created by pulumi and the path to the payload directory
// to create the payload required for the experiment.
func createPayload(ips, encoders []Instance, chainID, ppath string, squareSize int, useMainnetDistribution bool, fibreAccounts, encoderFibreAccounts int, mods ...genesis.Modifier) error {
	n, err := NewNetwork(chainID, squareSize, mods...)
	if err != nil {
		return err
	}

	stake := int64(genesis.DefaultInitialBalance) / 2
	for index, info := range ips {
		if useMainnetDistribution {
			stake = getMainnetStake(index)
		}
		err = n.AddValidator(
			info.Name,
			info.PublicIP,
			ppath,
			info.Region,
			stake,
			fibreAccounts,
		)
		if err != nil {
			return err
		}
	}

	// Create encoder-payload directory and keyrings for each encoder.
	// Encoder keyrings are stored in <ppath>/../encoder-payload/<encoder-name>/
	// so that a separate, lighter tar can be built during deploy.
	encoderPayloadDir := filepath.Join(filepath.Dir(ppath), "encoder-payload")
	if len(encoders) > 0 {
		if err := os.MkdirAll(encoderPayloadDir, 0o755); err != nil {
			return fmt.Errorf("failed to create encoder-payload dir: %w", err)
		}
	}
	for _, enc := range encoders {
		if err := n.AddEncoder(enc.Name, encoderPayloadDir, encoderFibreAccounts); err != nil {
			return fmt.Errorf("failed to add encoder %s: %w", enc.Name, err)
		}
	}

	for _, val := range n.genesis.Validators() {
		fmt.Println(val.Name, val.ConsensusKey.PubKey())
	}

	err = n.InitNodes(ppath)
	if err != nil {
		return err
	}

	err = n.SaveAddressBook(ppath, n.Peers())
	if err != nil {
		return err
	}

	return nil
}

// mainnetVotingPowers contains the current Celestia mainnet staking distribution for more realistic tests.
var mainnetVotingPowers []int

func getMainnetStake(index int) int64 {
	if index < 0 {
		return 0
	}
	if len(mainnetVotingPowers) == 0 {
		// these figures reflect the exact staking values on 09/07/25.
		mainnetVotingPowers = []int{
			44706511, 44437002, 37932228, 37544929, 29421912, 27045838, 25722376, 25574864, 19573478, 17083572,
			14156979, 10990505, 10228508, 8017107, 7985256, 7465738, 7156557, 7000454, 6957695, 6816721,
			6497714, 6133878, 6061770, 6023778, 5837045, 5817421, 5788259, 5571126, 5504182, 5500773,
			5070168, 4672609, 4360060, 4326293, 3978439, 3894538, 3746172, 3608145, 3606324, 3606128,
			3600486, 3560552, 3538637, 3456887, 3449504, 3365860, 3330140, 3329077, 3242441, 3231836,
			3163103, 3162476, 3139329, 3132732, 3117200, 3071253, 3059325, 3043103, 3039694, 3038574,
			3038322, 3025332, 3025137, 3013047, 3011854, 3010337, 3004185, 3001607, 3000732, 3000592,
			3000433, 3000236, 3000215, 3000207, 3000142, 3000128, 3000126, 2689474, 2500012, 2329666,
			2242943, 2083890, 2038490, 1957574, 1619120, 1615290, 1482045, 1291544, 1286175, 1204480,
			1202416, 1156152, 1137365, 1101315, 1045017, 1000381, 977562, 948538, 820448, 445353,
		}
	}
	if index >= len(mainnetVotingPowers) {
		return int64(mainnetVotingPowers[len(mainnetVotingPowers)-1])
	}
	return int64(mainnetVotingPowers[index])
}

// stageEncoderPayload copies the binaries (celestia-appd, fibre-txsim), genesis,
// vars.sh, and an encoder_init.sh script into the encoder-payload directory so
// that the deploy step can create a lightweight tar for encoder instances.
func stageEncoderPayload(rootDir, payloadDir, appBinaryPath, fibreTxsimBinaryPath, buildDirPath string) error {
	encPayload := filepath.Join(rootDir, "encoder-payload")

	// Build directory with only the two binaries an encoder needs
	encBuild := filepath.Join(encPayload, "build")
	if err := os.MkdirAll(encBuild, 0o755); err != nil {
		return err
	}

	if buildDirPath != "" {
		for _, name := range []string{"celestia-appd", "fibre-txsim"} {
			src := filepath.Join(buildDirPath, name)
			if err := copyFile(src, filepath.Join(encBuild, name), 0o755); err != nil {
				return fmt.Errorf("copy %s from build dir: %w", name, err)
			}
		}
	} else {
		if err := copyFile(appBinaryPath, filepath.Join(encBuild, "celestia-appd"), 0o755); err != nil {
			return fmt.Errorf("copy celestia-appd: %w", err)
		}
		if err := copyFile(fibreTxsimBinaryPath, filepath.Join(encBuild, "fibre-txsim"), 0o755); err != nil {
			return fmt.Errorf("copy fibre-txsim: %w", err)
		}
	}

	// Copy genesis and vars.sh
	if err := copyFile(filepath.Join(payloadDir, "genesis.json"), filepath.Join(encPayload, "genesis.json"), 0o644); err != nil {
		return fmt.Errorf("copy genesis.json: %w", err)
	}
	if err := copyFile(filepath.Join(payloadDir, "vars.sh"), filepath.Join(encPayload, "vars.sh"), 0o755); err != nil {
		return fmt.Errorf("copy vars.sh: %w", err)
	}

	// Write the encoder init script
	return writeEncoderInitScript(filepath.Join(encPayload, "encoder_init.sh"))
}

// stageBridgePayload copies the celestia-node binary, the consensus
// chain's genesis.json, and a templated bridge_init.sh into a
// bridge-payload directory. Deploy uses this to ship a lightweight tar
// to each bridge instance. The first validator's public IP is baked
// into the init script as core.ip — bridges follow validator-0 for
// header / block sync. With a multi-validator chain, validator-0 is a
// fine choice since headers come from consensus regardless.
func stageBridgePayload(rootDir, payloadDir, nodeBinaryPath, buildDirPath string, cfg Config) error {
	bridgePayload := filepath.Join(rootDir, "bridge-payload")

	if err := os.RemoveAll(bridgePayload); err != nil {
		return fmt.Errorf("clean old bridge-payload: %w", err)
	}

	bridgeBuild := filepath.Join(bridgePayload, "build")
	if err := os.MkdirAll(bridgeBuild, 0o755); err != nil {
		return err
	}

	// celestia-node's binary is named "celestia". --build-dir wins over
	// the per-binary path so a single packed dir can drive validator +
	// bridge + ev-node deploys.
	if buildDirPath != "" {
		src := filepath.Join(buildDirPath, "celestia")
		if err := copyFile(src, filepath.Join(bridgeBuild, "celestia"), 0o755); err != nil {
			return fmt.Errorf("copy celestia from build dir: %w", err)
		}
	} else {
		if err := copyFile(nodeBinaryPath, filepath.Join(bridgeBuild, "celestia"), 0o755); err != nil {
			return fmt.Errorf("copy celestia binary: %w", err)
		}
	}

	if err := copyFile(filepath.Join(payloadDir, "genesis.json"), filepath.Join(bridgePayload, "genesis.json"), 0o644); err != nil {
		return fmt.Errorf("copy genesis.json: %w", err)
	}
	if err := copyFile(filepath.Join(payloadDir, "vars.sh"), filepath.Join(bridgePayload, "vars.sh"), 0o755); err != nil {
		return fmt.Errorf("copy vars.sh: %w", err)
	}

	coreIP := cfg.Validators[0].PublicIP
	if coreIP == "" || coreIP == "TBD" {
		return fmt.Errorf("validator-0 has no public IP yet — run `talis up` before genesis")
	}

	return writeBridgeInitScript(filepath.Join(bridgePayload, "bridge_init.sh"), coreIP)
}

// writeBridgeInitScript writes the per-bridge init script. It runs
// `celestia bridge init`, points the bridge at validator-0's gRPC for
// state sync, generates an admin JWT (printed to a known file so
// downstream ev-node deploys can scp it), and starts the bridge in a
// detached tmux session.
//
// All values that change per-experiment are baked in at staging time.
// CHAIN_ID comes from sourced vars.sh; coreIP is templated literally
// since it's only known after talis up has populated config.json.
func writeBridgeInitScript(path string, coreIP string) error {
	script := `#!/bin/bash
set -euo pipefail

CELES_BRIDGE_HOME="$HOME/.celestia-bridge"
CORE_IP="` + coreIP + `"

export DEBIAN_FRONTEND=noninteractive
apt-get update -y -o Dpkg::Options::="--force-confdef" -o Dpkg::Options::="--force-confold"
apt-get install curl jq chrony tmux --yes -o Dpkg::Options::="--force-confdef" -o Dpkg::Options::="--force-confold"

systemctl enable chrony
systemctl start chrony

# TCP BBR — same tuning as validators / encoders.
modprobe tcp_bbr || true
sysctl -w net.core.default_qdisc=fq
sysctl -w net.ipv4.tcp_congestion_control=bbr

# Install celestia-node binary
cp bridge-payload/build/celestia /bin/celestia
chmod +x /bin/celestia

source bridge-payload/vars.sh
echo "Bridge bootstrap: chain_id=$CHAIN_ID core_ip=$CORE_IP"

# Initialize node store. p2p.network is the chain id; celestia-node
# accepts it because vars.sh exported CELESTIA_CUSTOM=$CHAIN_ID,
# which registers the chain id as a custom network at startup.
if [ ! -f "$CELES_BRIDGE_HOME/config.toml" ]; then
  celestia bridge init --p2p.network "$CHAIN_ID" --node.store "$CELES_BRIDGE_HOME"
fi

# Drop the consensus chain's genesis next to the bridge config so
# anything that reads it (peer discovery, header validation) sees
# the same genesis as validators.
mkdir -p "$CELES_BRIDGE_HOME/config"
cp bridge-payload/genesis.json "$CELES_BRIDGE_HOME/genesis.json"

# Generate the admin JWT and stash it where downstream consumers
# (ev-node deploy) can scp it. With CELESTIA_CUSTOM set, celestia
# prints a multi-line "WARNING: custom network..." banner to stdout
# alongside the token, so we grep for the JWT line specifically —
# otherwise downstream consumers send the warning text as the auth
# header and get a 401.
celestia bridge auth admin --node.store "$CELES_BRIDGE_HOME" 2>/dev/null \
  | grep -E '^eyJ' | tail -1 > /root/bridge-jwt.txt
echo "Wrote /root/bridge-jwt.txt ($(wc -c < /root/bridge-jwt.txt) bytes)"

ufw allow 26658/tcp || true   # RPC (admin API)
ufw allow 2121/tcp  || true   # P2P
ufw allow 2121/udp  || true

# Run in tmux so the SSH session can detach. RPC is exposed on
# 0.0.0.0:26658 (auth required via JWT). Core gRPC connection to
# validator-0 is plaintext for testnet.
tmux kill-session -t bridge 2>/dev/null || true
# tmux sessions inherit env from the tmux server, not the caller, so
# CELESTIA_CUSTOM has to be re-exported inside the inner command.
tmux new-session -d -s bridge "env CELESTIA_CUSTOM=${CHAIN_ID} celestia bridge start \
  --p2p.network ${CHAIN_ID} \
  --node.store ${CELES_BRIDGE_HOME} \
  --core.ip ${CORE_IP} \
  --core.port 9091 \
  --core.tls=false \
  --rpc.addr 0.0.0.0 \
  --rpc.port 26658 \
  --metrics 2>&1 | tee -a /root/bridge.log"

echo "Bridge started in tmux session 'bridge' — attach with: tmux attach -t bridge"
`
	return os.WriteFile(path, []byte(script), 0o755)
}

// stageEvnodePayload copies the evnode-fibre binary + a templated init
// script into evnode-payload/ so the deploy step can build a small tar
// per ev-node. The init script poll-waits for /root/bridge-jwt.txt and
// /root/keyring-fibre/ to exist before starting — both are scp'd in by
// a separate bootstrap step (or manually) so that JWT + keyring don't
// need to be embedded in the payload.
func stageEvnodePayload(rootDir, payloadDir, buildDirPath string, cfg Config) error {
	evPayload := filepath.Join(rootDir, "evnode-payload")

	if err := os.RemoveAll(evPayload); err != nil {
		return fmt.Errorf("clean old evnode-payload: %w", err)
	}

	evBuild := filepath.Join(evPayload, "build")
	if err := os.MkdirAll(evBuild, 0o755); err != nil {
		return err
	}

	if buildDirPath == "" {
		return fmt.Errorf("--build-dir is required when evnodes are configured (must contain `evnode` binary)")
	}
	src := filepath.Join(buildDirPath, "evnode")
	if err := copyFile(src, filepath.Join(evBuild, "evnode"), 0o755); err != nil {
		return fmt.Errorf("copy evnode from build dir: %w", err)
	}

	if err := copyFile(filepath.Join(payloadDir, "vars.sh"), filepath.Join(evPayload, "vars.sh"), 0o755); err != nil {
		return fmt.Errorf("copy vars.sh: %w", err)
	}

	bridgeIP := cfg.Bridges[0].PublicIP
	coreIP := cfg.Validators[0].PublicIP
	if bridgeIP == "" || bridgeIP == "TBD" {
		return fmt.Errorf("bridge-0 has no public IP yet — run `talis up` before genesis")
	}
	if coreIP == "" || coreIP == "TBD" {
		return fmt.Errorf("validator-0 has no public IP yet — run `talis up` before genesis")
	}

	return writeEvnodeInitScript(filepath.Join(evPayload, "evnode_init.sh"), bridgeIP, coreIP)
}

// writeEvnodeInitScript writes the evnode aggregator init script.
// Templated values: BRIDGE_IP (bridge-0 RPC for blob.Subscribe / Submit)
// and CORE_GRPC_ADDR (validator-0 gRPC for state queries via
// celestia-node's submit path). CHAIN_ID flows through vars.sh.
//
// The script does NOT copy bridge-jwt.txt or the fibre keyring itself —
// those must already exist on the box (manually scp'd or pushed by a
// future `talis fibre-bootstrap-evnode` command). The poll loop makes
// the script restartable: re-running deploy after copying the missing
// pieces will cleanly start the daemon.
func writeEvnodeInitScript(path string, bridgeIP, coreIP string) error {
	script := `#!/bin/bash
set -euo pipefail

EVNODE_HOME="$HOME/.evnode-fibre"
# celestia-node API client requires a URL scheme on the bridge addr.
BRIDGE_ADDR="http://` + bridgeIP + `:26658"
# celestia-app exposes Tendermint RPC services on :9090 and the
# cosmos.* / celestia.* state services on :9091. The fiber adapter's
# submit path queries cosmos.base.tendermint.v1beta1.Service so it
# needs the :9091 endpoint.
CORE_GRPC_ADDR="` + coreIP + `:9091"
BRIDGE_JWT_FILE="/root/bridge-jwt.txt"
FIBRE_KEYRING_DIR="/root/keyring-fibre"

export DEBIAN_FRONTEND=noninteractive
apt-get update -y -o Dpkg::Options::="--force-confdef" -o Dpkg::Options::="--force-confold"
apt-get install curl jq chrony tmux --yes -o Dpkg::Options::="--force-confdef" -o Dpkg::Options::="--force-confold"

systemctl enable chrony
systemctl start chrony

modprobe tcp_bbr || true
sysctl -w net.core.default_qdisc=fq
sysctl -w net.ipv4.tcp_congestion_control=bbr

cp evnode-payload/build/evnode /bin/evnode
chmod +x /bin/evnode

source evnode-payload/vars.sh
echo "evnode bootstrap: chain_id=$CHAIN_ID bridge=$BRIDGE_ADDR core=$CORE_GRPC_ADDR"

mkdir -p "$EVNODE_HOME"

# Wait for the operator-supplied dependencies. These come from a
# separate step (manual scp or 'talis fibre-bootstrap-evnode'):
#   1. /root/bridge-jwt.txt              admin JWT from the bridge
#   2. /root/keyring-fibre/keyring-test  cosmos-sdk file keyring with
#                                        a Fibre payment account
# Without them the daemon would crash immediately on startup.
echo "Waiting for $BRIDGE_JWT_FILE and $FIBRE_KEYRING_DIR..."
WAITED=0
until [ -s "$BRIDGE_JWT_FILE" ] && [ -d "$FIBRE_KEYRING_DIR/keyring-test" ]; do
  sleep 5
  WAITED=$((WAITED + 5))
  if [ $((WAITED % 60)) -eq 0 ]; then
    echo "  still waiting after ${WAITED}s..."
  fi
done
echo "Dependencies present after ${WAITED}s"

ufw allow 7777/tcp || true   # tx-ingest HTTP
ufw allow 7331/tcp || true   # ev-node RPC
ufw allow 7676/tcp || true   # libp2p (idle when Fiber on)

# A passphrase file keeps the file-signer reproducible across restarts
# without baking creds into the script.
mkdir -p "$EVNODE_HOME/.signer"
if [ ! -f "$EVNODE_HOME/.signer/passphrase" ]; then
  echo "evnode-fibre-passphrase" > "$EVNODE_HOME/.signer/passphrase"
  chmod 600 "$EVNODE_HOME/.signer/passphrase"
fi

tmux kill-session -t evnode 2>/dev/null || true
# CELESTIA_CUSTOM has to be present in the env that evnode runs under,
# not the caller's: tmux sessions inherit from the tmux server, not the
# shell that issues new-session. Without it, celestia-node refuses to
# accept --core-network=$CHAIN_ID since it's not in the hard-coded
# networksList.
tmux new-session -d -s evnode "env CELESTIA_CUSTOM=${CHAIN_ID} evnode \
  --home ${EVNODE_HOME} \
  --chain-id ${CHAIN_ID} \
  --bridge-addr ${BRIDGE_ADDR} \
  --bridge-token-file ${BRIDGE_JWT_FILE} \
  --core-grpc-addr ${CORE_GRPC_ADDR} \
  --core-network ${CHAIN_ID} \
  --keyring-path ${FIBRE_KEYRING_DIR} \
  --key-name fibre-0 \
  --signer-passphrase-file ${EVNODE_HOME}/.signer/passphrase \
  --log-level info \
  2>&1 | tee -a /root/evnode.log"

echo "ev-node started in tmux session 'evnode' — attach with: tmux attach -t evnode"
`
	return os.WriteFile(path, []byte(script), 0o755)
}

// stageLoadgenPayload stages the evnode-txsim binary + a templated
// init script for each load-gen instance. The script bursts traffic at
// evnode-0's HTTP /tx endpoint for a fixed duration (override via the
// TXSIM_DURATION / TXSIM_CONCURRENCY / TXSIM_TX_SIZE env vars on the
// box). Final TXSIM: line lands in /root/txsim.log.
func stageLoadgenPayload(rootDir, payloadDir, buildDirPath string, cfg Config) error {
	lgPayload := filepath.Join(rootDir, "loadgen-payload")

	if err := os.RemoveAll(lgPayload); err != nil {
		return fmt.Errorf("clean old loadgen-payload: %w", err)
	}
	lgBuild := filepath.Join(lgPayload, "build")
	if err := os.MkdirAll(lgBuild, 0o755); err != nil {
		return err
	}

	if buildDirPath == "" {
		return fmt.Errorf("--build-dir is required when loadgens are configured (must contain `evnode-txsim` binary)")
	}
	src := filepath.Join(buildDirPath, "evnode-txsim")
	if err := copyFile(src, filepath.Join(lgBuild, "evnode-txsim"), 0o755); err != nil {
		return fmt.Errorf("copy evnode-txsim from build dir: %w", err)
	}
	if err := copyFile(filepath.Join(payloadDir, "vars.sh"), filepath.Join(lgPayload, "vars.sh"), 0o755); err != nil {
		return fmt.Errorf("copy vars.sh: %w", err)
	}

	evnodeIP := cfg.Evnodes[0].PublicIP
	if evnodeIP == "" || evnodeIP == "TBD" {
		return fmt.Errorf("evnode-0 has no public IP yet — run `talis up` before genesis")
	}

	return writeLoadgenInitScript(filepath.Join(lgPayload, "loadgen_init.sh"), evnodeIP)
}

// writeLoadgenInitScript writes the per-loadgen init script. evnode-0's
// HTTP endpoint is templated literally because it's only known after
// `talis up`. Tunables (duration, concurrency, tx size) come through
// env vars at start time so a single deploy can drive multiple
// experiments via SSH-set environment.
func writeLoadgenInitScript(path string, evnodeIP string) error {
	script := `#!/bin/bash
set -euo pipefail

EVNODE_IP="` + evnodeIP + `"
TARGET="${TXSIM_TARGET:-http://${EVNODE_IP}:7777/tx}"
DURATION="${TXSIM_DURATION:-30s}"
CONCURRENCY="${TXSIM_CONCURRENCY:-8}"
TX_SIZE="${TXSIM_TX_SIZE:-10240}"

export DEBIAN_FRONTEND=noninteractive
apt-get update -y -o Dpkg::Options::="--force-confdef" -o Dpkg::Options::="--force-confold"
apt-get install curl chrony tmux --yes -o Dpkg::Options::="--force-confdef" -o Dpkg::Options::="--force-confold"

systemctl enable chrony
systemctl start chrony

modprobe tcp_bbr || true
sysctl -w net.core.default_qdisc=fq
sysctl -w net.ipv4.tcp_congestion_control=bbr

cp loadgen-payload/build/evnode-txsim /bin/evnode-txsim
chmod +x /bin/evnode-txsim

source loadgen-payload/vars.sh
echo "loadgen bootstrap: target=$TARGET duration=$DURATION concurrency=$CONCURRENCY tx_size=$TX_SIZE chain_id=$CHAIN_ID"

# Wait for ev-node's tx endpoint to come up (it will only start once
# bridge JWT + fibre keyring are scp'd in by the operator).
echo "Waiting for $TARGET to accept tx (testing /stats)..."
STATS_URL="${TARGET%/tx}/stats"
WAITED=0
until curl --silent --max-time 2 --output /dev/null "$STATS_URL" 2>/dev/null; do
  sleep 5
  WAITED=$((WAITED + 5))
  if [ $((WAITED % 60)) -eq 0 ]; then
    echo "  still waiting for ev-node after ${WAITED}s..."
  fi
done
echo "ev-node reachable after ${WAITED}s; starting txsim run"

tmux kill-session -t txsim 2>/dev/null || true
tmux new-session -d -s txsim "evnode-txsim \
  --target $TARGET \
  --duration $DURATION \
  --concurrency $CONCURRENCY \
  --tx-size $TX_SIZE \
  2>&1 | tee -a /root/txsim.log"

echo "txsim started in tmux session 'txsim' — attach with: tmux attach -t txsim"
echo "Final summary lands at /root/txsim.log; grep TXSIM: for the machine-parseable line"
`
	return os.WriteFile(path, []byte(script), 0o755)
}

// writeEncoderInitScript creates a minimal init script for encoder instances.
// Encoders only need the fibre-txsim binary, celestia-appd (for escrow deposits),
// a keyring, and genesis.
func writeEncoderInitScript(path string) error {
	script := `#!/bin/bash
set -euo pipefail

CELES_HOME="$HOME/.celestia-app"

export DEBIAN_FRONTEND=noninteractive
apt-get update -y -o Dpkg::Options::="--force-confdef" -o Dpkg::Options::="--force-confold"
apt-get install curl jq chrony --yes -o Dpkg::Options::="--force-confdef" -o Dpkg::Options::="--force-confold"

systemctl enable chrony
systemctl start chrony

# TCP BBR
modprobe tcp_bbr || true
sysctl -w net.core.default_qdisc=fq
sysctl -w net.ipv4.tcp_congestion_control=bbr

# Install binaries
cp encoder-payload/build/celestia-appd /bin/celestia-appd
cp encoder-payload/build/fibre-txsim /bin/fibre-txsim

source encoder-payload/vars.sh

# Determine this encoder's directory from hostname (e.g. "encoder-0")
hostname=$(hostname)
parsed_hostname=$(echo "$hostname" | awk -F'-' '{print $1 "-" $2}')

# Set up celestia-app home with keyring + genesis
rm -rf "$CELES_HOME"
mkdir -p "$CELES_HOME/config"
cp encoder-payload/genesis.json "$CELES_HOME/config/genesis.json"
cp -r "encoder-payload/$parsed_hostname/keyring-test" "$CELES_HOME/"

echo "Encoder $parsed_hostname initialized"
`
	return os.WriteFile(path, []byte(script), 0o755)
}

func writeAWSEnv(varsPath string, cfg Config) error {
	f, err := os.OpenFile(varsPath,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0o755,
	)
	if err != nil {
		return fmt.Errorf("failed to open vars.sh for append: %w", err)
	}
	defer f.Close()

	exports := []string{
		fmt.Sprintf("export AWS_DEFAULT_REGION=%q\n", cfg.S3Config.Region),
		fmt.Sprintf("export AWS_ACCESS_KEY_ID=%q\n", cfg.S3Config.AccessKeyID),
		fmt.Sprintf("export AWS_SECRET_ACCESS_KEY=%q\n", cfg.S3Config.SecretAccessKey),
		fmt.Sprintf("export AWS_S3_BUCKET=%q\n", cfg.S3Config.BucketName),
		fmt.Sprintf("export AWS_S3_ENDPOINT=%q\n", cfg.S3Config.Endpoint),
		fmt.Sprintf("export CHAIN_ID=%q\n", cfg.ChainID),
		// celestia-node refuses any --p2p.network value that's not in
		// its known networksList unless CELESTIA_CUSTOM registers a
		// custom one. Format: <netID>:<genesisHash>:<bootstrapPeers>
		// — only netID is required. We use the chain id so bridge +
		// evnode-fibre's Network identifier matches the consensus
		// chain id and celestia-node's "wrong network in core.ip"
		// validation passes.
		fmt.Sprintf("export CELESTIA_CUSTOM=%q\n", cfg.ChainID),
	}

	for _, line := range exports {
		if _, err := f.WriteString(line); err != nil {
			return fmt.Errorf("failed to append to vars.sh: %w", err)
		}
	}

	return nil
}
