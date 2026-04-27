//go:build fibre_docker

// bad_host_repro_test.go — reproduces the production "too many colons in
// address" / "first path segment in URL cannot contain colon" failures
// observed when an operator registers a Fibre provider with a host
// string that isn't in the canonical `dns:///host:port` form.
//
// Root cause: x/valaddr `MsgSetFibreProviderInfo.ValidateBasic` only
// checks that the host is non-empty and ≤100 chars. Anything else
// passes — including `http://10.0.37.242:7980`, bare `host:port`, or
// arbitrary garbage. At read time the fibre client's
// `HostRegistry.GetHost` runs `url.Parse(host)`; bare host:port fails
// that, while `http://...` passes and then breaks downstream because
// `grpc.NewClient` doesn't recognise `http` as a resolver scheme and
// appends a default `:443`, yielding `http://host:port:443` ("too
// many colons").
//
// The expected fix is to require a strict `host:port` form in
// `ValidateBasic` (no scheme, no path, no userinfo). After that lands
// the chain rejects the registration tx itself and the assertions
// here flip — see assertChainAcceptsBadHost.
//
// Run with:
//
//	go test -tags 'fibre fibre_docker' -count=1 -timeout 5m \
//	    -run TestFibreClient_BadHostRegistration ./testing/docker/...

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

// TestFibreClient_BadHostRegistration re-registers every validator with
// a malformed host string, confirms the chain accepts the registration
// (the bug), then confirms Upload fails because none of the validators
// can be dialed (the symptom). After each subtest the canonical
// registrations are restored so sibling tests on the shared docker
// stack continue to pass.
func TestFibreClient_BadHostRegistration(t *testing.T) {
	cases := []struct {
		name string
		// hostFor returns the bad host string to register for the given
		// validator index (0..3).
		hostFor func(i int) string
	}{
		{
			name: "http_scheme_prefix",
			hostFor: func(i int) string {
				return fmt.Sprintf("http://127.0.0.1:%d", 7980+i)
			},
		},
		{
			name: "bare_host_port",
			hostFor: func(i int) string {
				return fmt.Sprintf("127.0.0.1:%d", 7980+i)
			},
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

			// Register every validator with the broken host. The chain
			// should accept all of them — that's the bug.
			for i := 0; i < 4; i++ {
				bad := tc.hostFor(i)
				require.NoError(t, setValHost(ctx, t, i, bad),
					"chain accepted MsgSetFibreProviderInfo for val%d host=%q (no format validation)", i, bad)
			}
			// Wait until val3's bad host is observable; this is the
			// last one we wrote, so its presence implies the others
			// also propagated.
			require.NoError(t, waitForHost(ctx, t, tc.hostFor(3)),
				"bad registrations should be visible on chain")

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
			payload := []byte(fmt.Sprintf("bad-host-repro-%s-%d", tc.name, time.Now().UnixNano()))

			uploadCtx, uploadCancel := context.WithTimeout(ctx, 60*time.Second)
			defer uploadCancel()

			_, uploadErr := adapter.Upload(uploadCtx, namespace, payload)
			require.Error(t, uploadErr, "Upload must fail when no validator host can be dialed")
			t.Logf("upload failed as expected (%s): %v", tc.name, uploadErr)
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
