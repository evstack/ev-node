//go:build fibre

package cnfibertest

import (
	"context"
	"crypto/rand"
	"net"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cristalhq/jwt/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	appfibre "github.com/celestiaorg/celestia-app/v8/fibre"

	"github.com/celestiaorg/celestia-node/api/client"
	"github.com/celestiaorg/celestia-node/api/rpc/perms"
	"github.com/celestiaorg/celestia-node/nodebuilder"
	"github.com/celestiaorg/celestia-node/nodebuilder/node"
	"github.com/celestiaorg/celestia-node/nodebuilder/p2p"
	stateapi "github.com/celestiaorg/celestia-node/nodebuilder/state"
)

// Bridge bundles an in-process celestia-node bridge node and the admin
// JWT that grants it authenticated RPC access. The adapter's ReadConfig
// needs both the address and the token for Blob.Subscribe to work.
type Bridge struct {
	Node       *nodebuilder.Node
	AdminToken string
}

// RPCAddr returns a WebSocket URL the adapter uses in
// Config.ReadConfig.BridgeDAAddr. WebSocket (not HTTP) is required
// because Blob.Subscribe returns a channel; go-jsonrpc only supports
// channel-returning methods over a streaming transport.
func (b *Bridge) RPCAddr() string {
	return "ws://" + b.Node.RPCServer.ListenAddr()
}

// StartBridge brings up an in-process celestia-node bridge connected to
// the Network's consensus gRPC endpoint. Mirrors celestia-node's
// api/client test helpers so TestShowcase has a real JSON-RPC server for
// Blob.Subscribe.
func StartBridge(t *testing.T, ctx context.Context, network *Network) *Bridge {
	t.Helper()

	cfg := nodebuilder.DefaultConfig(node.Bridge)

	ip, port, err := net.SplitHostPort(network.ConsensusGRPCAddr())
	require.NoError(t, err, "splitting consensus gRPC addr")
	cfg.Core.IP = ip
	cfg.Core.Port = port
	// Pin the bridge RPC to an ephemeral port; the test discovers it via
	// Node.RPCServer.ListenAddr() after Start.
	cfg.RPC.Port = "0"

	tempDir := t.TempDir()
	store := nodebuilder.MockStore(t, cfg)

	auth, adminToken := bridgeAuth(t)
	kr := bridgeKeyring(t, tempDir)

	bn, err := nodebuilder.New(node.Bridge, p2p.Private, store,
		auth,
		stateapi.WithKeyring(kr),
		stateapi.WithKeyName(stateapi.AccountName(bridgeSigningKey)),
		fx.Replace(node.StorePath(tempDir)),
	)
	require.NoError(t, err, "constructing bridge node")

	require.NoError(t, bn.Start(ctx), "starting bridge node")
	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = bn.Stop(stopCtx)
	})

	return &Bridge{
		Node:       bn,
		AdminToken: adminToken,
	}
}

// bridgeSigningKey is the keyring account the bridge uses for its own
// tx submissions. Distinct from the client's account so the two keyrings
// don't collide.
const bridgeSigningKey = "bridge-signer"

func bridgeKeyring(t *testing.T, tempDir string) keyring.Keyring {
	t.Helper()

	kr, err := client.KeyringWithNewKey(client.KeyringConfig{
		KeyName:     bridgeSigningKey,
		BackendName: keyring.BackendTest,
	}, tempDir)
	require.NoError(t, err, "creating bridge keyring")

	// The Fibre module on the bridge expects a key under
	// appfibre.DefaultKeyName to exist, even though our client never uses
	// the bridge's Fibre module for Upload/Download. Without it,
	// fx.Start fails during fibre module wiring.
	_, _, err = kr.NewMnemonic(
		appfibre.DefaultKeyName,
		keyring.English, "", "", hd.Secp256k1,
	)
	require.NoError(t, err, "provisioning bridge fibre key")
	return kr
}

// bridgeAuth creates an HS256 JWT signer pair and an admin token. The
// returned fx option injects the signer/verifier into the node; the
// token is what the adapter puts in ReadConfig.DAAuthToken.
func bridgeAuth(t *testing.T) (fx.Option, string) {
	t.Helper()

	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err, "rand.Read jwt key")

	signer, err := jwt.NewSignerHS(jwt.HS256, key)
	require.NoError(t, err)
	verifier, err := jwt.NewVerifierHS(jwt.HS256, key)
	require.NoError(t, err)

	token, err := perms.NewTokenWithPerms(signer, perms.AllPerms)
	require.NoError(t, err)

	return fx.Decorate(func() (jwt.Signer, jwt.Verifier, error) {
		return signer, verifier, nil
	}), string(token)
}
