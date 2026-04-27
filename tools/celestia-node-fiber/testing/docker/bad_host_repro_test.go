//go:build fibre_docker

// bad_host_repro_test.go — empirically validates which fibre provider
// host formats survive end-to-end through the chain + fibre client +
// gRPC dialer against a docker stack built from celestia-app
// `feat/fibre-payments` (i.e. a chain with the strict host:port
// validation in x/valaddr).
//
// Three formats are exercised:
//
//   - `host:port`             : the canonical accepted form. Upload
//                               succeeds end-to-end.
//   - `http://host:port`      : rejected by `MsgSetFibreProviderInfo`
//                               `ValidateBasic` — set-host tx fails.
//   - `dns:///host:port`      : also rejected by `ValidateBasic` for
//                               the same reason. Used to be the only
//                               working form pre-fix, see
//                               celestia-app PR #7183.
//
// Pre-fix (celestia-app on `main` without #7183): the chain accepted
// every string, the failures surfaced at upload time as either
// "too many colons in address" (http:// case) or
// "first path segment in URL cannot contain colon" (host:port case).
// Those production-symptom assertions live in this test's git history
// before the fix landed; once the chain enforces format, the failure
// surfaces earlier (set-host tx rejection) which is what we assert
// here.
//
// Run with:
//
//	go test -tags 'fibre fibre_docker' -count=1 -timeout 5m \
//	    -run TestFibreClient_HostRegistrationFormats ./testing/docker/...

package docker_test

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/celestiaorg/celestia-node/api/client"

	cnfiber "github.com/evstack/ev-node/tools/celestia-node-fiber"
)

// canonicalHosts records the per-validator hosts that register-fsps.sh
// installs at boot. The repro test re-registers each validator with a
// chosen format, asserts the resulting Upload behaviour, then restores
// these canonical values so other tests remain runnable.
var canonicalHosts = map[int]string{
	0: "127.0.0.1:7980",
	1: "127.0.0.1:7981",
	2: "127.0.0.1:7982",
	3: "127.0.0.1:7983",
}

// TestFibreClient_HostRegistrationFormats exercises three host formats
// against the chain's MsgSetFibreProviderInfo + the fibre client's
// HostRegistry + gRPC dialer:
//
//   - host_port      → set-host succeeds, Upload succeeds (positive)
//   - http_prefix    → set-host fails at chain ValidateBasic (negative)
//   - dns_prefix     → set-host fails at chain ValidateBasic (negative)
//
// After each subtest the canonical registrations are restored so
// sibling tests on the shared docker stack continue to pass.
func TestFibreClient_HostRegistrationFormats(t *testing.T) {
	cases := []struct {
		name string
		// hostFor returns the host string to register for the given
		// validator index (0..3).
		hostFor func(i int) string
		// wantSetHostErr, when non-empty, marks this case as expected
		// to fail at `tx valaddr set-host` time. The substring must
		// appear in the CLI's stderr/stdout output (the chain's
		// ValidateBasic / MsgSetFibreProviderInfo response includes
		// "host must be in host:port form" or similar).
		wantSetHostErr string
	}{
		{
			name: "host_port",
			hostFor: func(i int) string {
				return fmt.Sprintf("127.0.0.1:%d", 7980+i)
			},
		},
		{
			name: "http_prefix",
			hostFor: func(i int) string {
				return fmt.Sprintf("http://127.0.0.1:%d", 7980+i)
			},
			// celestia-app's x/valaddr ValidateBasic returns this
			// error chain via the SDK CLI broadcast path.
			wantSetHostErr: "host must be in host:port form",
		},
		{
			name: "dns_prefix",
			hostFor: func(i int) string {
				return fmt.Sprintf("dns:///127.0.0.1:%d", 7980+i)
			},
			wantSetHostErr: "host must be in host:port form",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			t.Cleanup(cancel)

			t.Cleanup(func() {
				restoreCtx, restoreCancel := context.WithTimeout(context.Background(), 90*time.Second)
				defer restoreCancel()
				for i := 0; i < 4; i++ {
					if err := setValHost(restoreCtx, t, i, canonicalHosts[i]); err != nil {
						t.Logf("WARNING: failed to restore val%d host: %v", i, err)
					}
				}
				if err := waitForHost(restoreCtx, t, canonicalHosts[3]); err != nil {
					t.Logf("WARNING: canonical hosts did not propagate within timeout: %v", err)
				}
			})

			// For the negative cases we register only val0 — that's
			// enough to demonstrate the chain rejects the format. We
			// don't bother trying all four since the rejection comes
			// from ValidateBasic, which doesn't depend on which
			// validator submits the tx.
			if tc.wantSetHostErr != "" {
				err := setValHost(ctx, t, 0, tc.hostFor(0))
				require.Error(t, err, "chain must reject %q at set-host tx", tc.hostFor(0))
				require.Contains(t, err.Error(), tc.wantSetHostErr,
					"set-host error should match expected ValidateBasic message")
				t.Logf("set-host rejected as expected (%s): %v", tc.name, err)
				return
			}

			// Positive case: register all four validators with the
			// canonical format, then run a real Upload to confirm the
			// gRPC dial path works end-to-end.
			for i := 0; i < 4; i++ {
				h := tc.hostFor(i)
				require.NoError(t, setValHost(ctx, t, i, h),
					"chain should accept set-host for val%d host=%q", i, h)
			}
			require.NoError(t, waitForHost(ctx, t, tc.hostFor(3)),
				"%s registrations should be visible on chain", tc.name)

			jwt := readBridgeJWT(t)
			kr := readClientKeyring(t)

			adapter, err := cnfiber.New(ctx, cnfiber.Config{
				Client: client.Config{
					ReadConfig: client.ReadConfig{
						BridgeDAAddr: envOr("FIBRE_BRIDGE_ADDR", bridgeAddr),
						DAAuthToken:  jwt,
						EnableDATLS:  false,
					},
					SubmitConfig: client.SubmitConfig{
						DefaultKeyName: clientAccount,
						Network:        chainID,
						CoreGRPCConfig: client.CoreGRPCConfig{
							Addr: envOr("FIBRE_CONSENSUS_ADDR", consensusAddr),
						},
					},
				},
			}, kr)
			require.NoError(t, err, "constructing adapter")
			t.Cleanup(func() { _ = adapter.Close() })

			namespace := bytes.Repeat([]byte{0xcd}, 10)
			payload := []byte(fmt.Sprintf("host-format-repro-%s-%d", tc.name, time.Now().UnixNano()))

			uploadCtx, uploadCancel := context.WithTimeout(ctx, 60*time.Second)
			defer uploadCancel()

			res, uploadErr := adapter.Upload(uploadCtx, namespace, payload)
			require.NoError(t, uploadErr, "Upload should succeed for %s host format", tc.name)
			require.NotEmpty(t, res.BlobID)
			t.Logf("upload ok (%s): blob_id=%x", tc.name, res.BlobID)
		})
	}
}

func setValHost(ctx context.Context, t *testing.T, valIdx int, host string) error {
	t.Helper()
	valName := fmt.Sprintf("val%d", valIdx)
	home := fmt.Sprintf("/shared/%s/.celestia-app", valName)
	cmd := exec.CommandContext(ctx, "docker", "compose", "exec", "-T", valName,
		"celestia-appd", "tx", "valaddr", "set-host", host,
		"--from", "validator",
		"--keyring-backend", "test",
		"--home", home,
		"--chain-id", chainID,
		"--node", fmt.Sprintf("tcp://%s:26657", valName),
		"--fees", "5000utia",
		"--output", "json",
		"--yes",
	)
	cmd.Dir = mustDockerDir(t)
	out, err := cmd.CombinedOutput()
	// Wider sleep so the next set-host on the same validator account
	// doesn't race the mempool's nonce check ("tx already exists").
	defer time.Sleep(4 * time.Second)
	if err != nil {
		// Two flavours: (a) pre-broadcast ValidateBasic rejection — CLI
		// exits non-zero with the validation error in stderr, no JSON
		// payload; (b) broadcast accepted but the chain returned a
		// non-zero code in the JSON ack. Surface either to the caller.
		return fmt.Errorf("set-host %q on %s: %w: %s", host, valName, err, string(out))
	}
	// Successful broadcast: parse the JSON to confirm the chain code is 0.
	if !strings.Contains(string(out), `"code":0`) {
		return fmt.Errorf("chain rejected set-host %q on %s (non-zero code): %s",
			host, valName, string(out))
	}
	return nil
}

func waitForHost(ctx context.Context, t *testing.T, want string) error {
	t.Helper()
	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		json, err := queryAllProviders(ctx, t)
		if err == nil && strings.Contains(json, fmt.Sprintf(`"host":"%s"`, want)) {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("timed out waiting for host=%q to appear in providers query", want)
}

func queryAllProviders(ctx context.Context, t *testing.T) (string, error) {
	t.Helper()
	cmd := exec.CommandContext(ctx, "docker", "compose", "exec", "-T", "val0",
		"celestia-appd", "query", "valaddr", "providers",
		"--home", "/shared/val0/.celestia-app",
		"--node", "tcp://val0:26657",
		"--output", "json",
	)
	cmd.Dir = mustDockerDir(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("query providers: %w: %s", err, string(out))
	}
	return string(out), nil
}
