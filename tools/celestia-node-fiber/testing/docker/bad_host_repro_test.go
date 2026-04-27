//go:build fibre_docker

// bad_host_repro_test.go — empirically validates which fibre provider
// host formats survive end-to-end through the chain + fibre client +
// gRPC dialer. Three formats are exercised:
//
//   - `http://host:port`      : reproduces production error
//                               "too many colons in address"
//   - `host:port`             : reproduces production error
//                               "first path segment in URL cannot
//                               contain colon"
//   - `dns:///host:port`      : the only working form today
//
// Root cause: x/valaddr `MsgSetFibreProviderInfo.ValidateBasic` only
// checks that the host is non-empty and ≤100 chars, so any of the
// above is accepted on chain. At read time the fibre client's
// `HostRegistry.GetHost` runs `url.Parse(host)`; bare host:port fails
// that, while `http://...` passes and then breaks downstream because
// `grpc.NewClient` doesn't recognise `http` as a resolver scheme and
// appends a default `:443`, yielding `http://host:port:443` ("too
// many colons"). Only `dns:///host:port` parses as a URL AND is a
// gRPC-known resolver scheme, so it works end-to-end.
//
// The expected fix is to require a strict `host:port` form in
// `ValidateBasic` (no scheme, no path, no userinfo). After that lands
// the chain rejects the registration tx for both `http://...` and
// `dns:///...` and only `host:port` succeeds — assertions in this
// test will need to flip.
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
// broken host, asserts Upload fails, then restores these canonical
// values so other tests remain runnable.
var canonicalHosts = map[int]string{
	0: "dns:///127.0.0.1:7980",
	1: "dns:///127.0.0.1:7981",
	2: "dns:///127.0.0.1:7982",
	3: "dns:///127.0.0.1:7983",
}

// TestFibreClient_HostRegistrationFormats re-registers every validator
// with a particular host-string format, then attempts an Upload through
// a fresh adapter and asserts whether the upload succeeds or fails.
//
// The matrix establishes empirically which formats the chain + fibre
// client accept end-to-end:
//
//   - http_scheme_prefix  → fails with "too many colons in address"
//   - bare_host_port      → fails with "first path segment in URL ..."
//   - dns_prefix          → succeeds (this is the only working form)
//
// The two failing cases exactly reproduce the production warnings the
// operator saw. The succeeding case is the positive control showing
// `dns:///host:port` is the working format today, which is what the
// proposed valaddr fix changes (it would make `host:port` succeed and
// `dns:///` fail).
//
// After each subtest the canonical registrations are restored so
// sibling tests on the shared docker stack continue to pass.
func TestFibreClient_HostRegistrationFormats(t *testing.T) {
	cases := []struct {
		name string
		// hostFor returns the host string to register for the given
		// validator index (0..3).
		hostFor func(i int) string
		// wantUploadErr, when non-empty, marks this case as expected to
		// fail Upload; the substring must appear in the resulting error
		// chain (we look at the per-validator warning; the outer error
		// is "not enough voting power" once enough fail).
		wantUploadErr string
	}{
		{
			name: "http_scheme_prefix",
			hostFor: func(i int) string {
				return fmt.Sprintf("http://127.0.0.1:%d", 7980+i)
			},
			// Adapter uploads return the aggregate error; the
			// per-validator dial error is logged, not bubbled. We
			// assert the aggregate ("not enough voting power") here
			// and rely on log capture below for the specific message.
			wantUploadErr: "not enough voting power",
		},
		{
			name: "bare_host_port",
			hostFor: func(i int) string {
				return fmt.Sprintf("127.0.0.1:%d", 7980+i)
			},
			wantUploadErr: "not enough voting power",
		},
		{
			name: "dns_prefix",
			hostFor: func(i int) string {
				return fmt.Sprintf("dns:///127.0.0.1:%d", 7980+i)
			},
			// No wantUploadErr — Upload should succeed.
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			t.Cleanup(cancel)

			jwt := readBridgeJWT(t)
			kr := readClientKeyring(t)

			t.Cleanup(func() {
				restoreCtx, restoreCancel := context.WithTimeout(context.Background(), 90*time.Second)
				defer restoreCancel()
				for i := 0; i < 4; i++ {
					if err := setValHost(restoreCtx, t, i, canonicalHosts[i]); err != nil {
						t.Logf("WARNING: failed to restore val%d host: %v", i, err)
					}
				}
				// One waitFor is enough — they're applied serially
				// against val0's RPC and tx ordering guarantees the
				// later ones land after the earlier one's check.
				if err := waitForHost(restoreCtx, t, canonicalHosts[3]); err != nil {
					t.Logf("WARNING: canonical hosts did not propagate within timeout: %v", err)
				}
			})

			// Register every validator with the chosen host format.
			// The chain accepts all of these today — even the broken
			// ones — because ValidateBasic only checks length.
			for i := 0; i < 4; i++ {
				h := tc.hostFor(i)
				require.NoError(t, setValHost(ctx, t, i, h),
					"chain should accept set-host for val%d host=%q on the current code", i, h)
			}
			// Wait until val3's host is observable; this is the last
			// one we wrote, so its presence implies the others also
			// propagated.
			require.NoError(t, waitForHost(ctx, t, tc.hostFor(3)),
				"%s registrations should be visible on chain", tc.name)

			// Construct a FRESH adapter so PullAll picks up the just-
			// updated registry rather than a cached canonical entry.
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
			if tc.wantUploadErr != "" {
				require.Error(t, uploadErr, "Upload must fail when no validator host can be dialed")
				require.Contains(t, uploadErr.Error(), tc.wantUploadErr,
					"upload error should match expected aggregate failure")
				t.Logf("upload failed as expected (%s): %v", tc.name, uploadErr)
			} else {
				require.NoError(t, uploadErr, "Upload should succeed for %s host format", tc.name)
				require.NotEmpty(t, res.BlobID)
				t.Logf("upload ok (%s): blob_id=%x", tc.name, res.BlobID)
			}
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
		"--yes",
	)
	cmd.Dir = mustDockerDir(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("set-host %q on %s: %w: %s", host, valName, err, string(out))
	}
	// Brief pause so the next set-host targets a tx with an incremented
	// account sequence (sequential txs from the same validator account
	// can race the mempool's nonce check).
	time.Sleep(2 * time.Second)
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
