//go:build fibre_docker

// docker_test.go — runs the same Upload → Listen → Download flow as
// the in-process showcase, but against the docker-compose stack in
// this directory.
//
// Bring the stack up first:
//
//	cd tools/celestia-node-fiber/testing/docker
//	docker compose up -d --build
//	# wait until `docker compose logs register` says "setup.done flag written"
//	# and `docker compose logs bridge` shows the bridge serving on :26658
//
// Then from the parent dir:
//
//	go test -tags 'fibre fibre_docker' -count=1 -timeout 5m ./testing/docker/...
//
// The test reads the bridge JWT from the shared docker volume by
// running `docker compose exec bridge cat /shared/bridge-admin-jwt.txt`,
// so the docker CLI must be available on the host.
package docker_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/stretchr/testify/require"

	"github.com/celestiaorg/celestia-app/v8/app"
	"github.com/celestiaorg/celestia-app/v8/app/encoding"
	"github.com/celestiaorg/celestia-node/api/client"

	cnfiber "github.com/evstack/ev-node/tools/celestia-node-fiber"
)

// keep block import compiled out of the test binary; the assertion that
// adapter satisfies block.FiberClient lives in the unit tests.
var _ = (cnfiber.Adapter{})

const (
	bridgeAddr    = "ws://127.0.0.1:26658"
	consensusAddr = "127.0.0.1:9090"
	chainID       = "fibre-docker"
	clientAccount = "default-fibre"
	docTimeout    = 60 * time.Second
)

// envOr returns the env var if set, otherwise fallback.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func readBridgeJWT(t *testing.T) string {
	t.Helper()
	cmd := exec.Command("docker", "compose", "exec", "-T", "bridge",
		"cat", "/shared/bridge-admin-jwt.txt")
	cmd.Dir = mustDockerDir(t)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "reading bridge JWT: %s", string(out))
	return strings.TrimSpace(string(out))
}

// mustDockerDir locates this file's directory at runtime so the test can
// invoke docker compose against the correct compose file regardless of
// where `go test` was launched from.
func mustDockerDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	return wd
}

// readClientKeyring loads the keyring populated by init-genesis.sh from
// the shared docker volume into a host-side keyring.Keyring suitable
// for fiber.New. We do this by `docker cp`-ing the seed validator's
// home dir to a local temp dir.
//
// TODO: this assumes the operator has the docker CLI; the test doesn't
// validate that up front. Add a `docker version` precheck if we want a
// clearer error.
func readClientKeyring(t *testing.T) keyring.Keyring {
	t.Helper()
	tmp := t.TempDir()
	cmd := exec.Command("docker", "compose", "cp",
		"val0:/shared/val0/.celestia-app/keyring-test", tmp+"/keyring-test")
	cmd.Dir = mustDockerDir(t)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "docker cp keyring: %s", string(out))

	encCfg := encoding.MakeConfig(app.ModuleEncodingRegisters...)
	kr, err := keyring.New("docker-test", keyring.BackendTest, tmp, nil, encCfg.Codec)
	require.NoError(t, err, "constructing keyring")
	return kr
}

// TestDockerShowcase drives Upload → Listen → Download against the
// docker-compose stack. The 4-validator network exercises real 2/3
// quorum aggregation that the single-validator showcase cannot.
//
// Build tag `fibre_docker` keeps the test out of default `go test`
// runs since it requires an external docker stack.
func TestDockerShowcase(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	t.Cleanup(cancel)

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
	require.NoError(t, err, "constructing adapter against docker stack")
	t.Cleanup(func() { _ = adapter.Close() })

	namespace := bytes.Repeat([]byte{0xab}, 10)
	payload := []byte(fmt.Sprintf("docker showcase %d", time.Now().UnixNano()))

	events, err := adapter.Listen(ctx, namespace, 0)
	require.NoError(t, err, "Listen subscription")

	up, err := adapter.Upload(ctx, namespace, payload)
	require.NoError(t, err, "Upload")
	require.NotEmpty(t, up.BlobID)
	t.Logf("upload ok: blob_id=%s", hex.EncodeToString(up.BlobID))

	select {
	case ev, ok := <-events:
		require.True(t, ok, "Listen channel closed without event")
		require.Equal(t, up.BlobID, ev.BlobID, "BlobID mismatch")
		require.Equal(t, uint64(len(payload)), ev.DataSize)
		require.Greater(t, ev.Height, uint64(0))
		t.Logf("listen ok: height=%d data_size=%d", ev.Height, ev.DataSize)
	case <-time.After(docTimeout):
		t.Fatalf("timed out waiting for BlobEvent after %s", docTimeout)
	}

	got, err := adapter.Download(ctx, up.BlobID)
	require.NoError(t, err, "Download")
	require.Equal(t, payload, got, "downloaded bytes don't match payload")
	t.Logf("download ok: %d bytes", len(got))
}
