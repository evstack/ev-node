//go:build fibre

package cnfibertest_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/celestiaorg/celestia-node/api/client"

	"github.com/evstack/ev-node/block"
	cnfiber "github.com/evstack/ev-node/tools/celestia-node-fiber"
	cnfibertest "github.com/evstack/ev-node/tools/celestia-node-fiber/testing"
)

// listenEventTimeout is how long we wait for a Fibre BlobEvent to arrive
// from Blob.Subscribe after Upload. Block time is ~200ms on this testnode
// + async MsgPayForFibre broadcast + subscription propagation; 30s is a
// conservative upper bound that catches real breakage without making
// flakes likely.
const listenEventTimeout = 30 * time.Second

// TestShowcase spins up a single-validator Celestia chain with an
// in-process Fibre server, a celestia-node bridge, and drives the full
// adapter surface: Listen subscribes first, Upload pushes a blob, the
// async MsgPayForFibre commits on-chain, the subscription delivers the
// event, and Download reconstructs the original bytes.
func TestShowcase(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)

	network := cnfibertest.StartNetwork(t, ctx)
	bridge := cnfibertest.StartBridge(t, ctx, network)

	adapter, err := cnfiber.New(ctx, cnfiber.Config{
		Client: client.Config{
			ReadConfig: client.ReadConfig{
				BridgeDAAddr: bridge.RPCAddr(),
				DAAuthToken:  bridge.AdminToken,
				EnableDATLS:  false,
			},
			SubmitConfig: client.SubmitConfig{
				DefaultKeyName: network.ClientAccount,
				Network:        "private",
				CoreGRPCConfig: client.CoreGRPCConfig{
					Addr: network.ConsensusGRPCAddr(),
				},
			},
		},
	}, network.Consensus.Keyring)
	require.NoError(t, err, "constructing adapter")
	t.Cleanup(func() { _ = adapter.Close() })

	// Namespace: 10 bytes of v0 ID, same size the ev-node DA contract
	// passes as []byte. Using a distinctive pattern so we can spot it in
	// logs if the test gets chatty.
	namespace := bytes.Repeat([]byte{0xfe}, 10)
	payload := []byte("celestia-node-fiber adapter showcase")

	// Subscribe BEFORE uploading so we don't miss the event emitted when
	// the MsgPayForFibre commits on-chain.
	events, err := adapter.Listen(ctx, namespace)
	require.NoError(t, err, "starting Listen subscription")

	uploadResult, err := adapter.Upload(ctx, namespace, payload)
	require.NoError(t, err, "adapter.Upload")
	require.NotEmpty(t, uploadResult.BlobID, "upload returned empty BlobID")
	t.Logf("upload ok: blob_id=%x expires_at=%s", uploadResult.BlobID, uploadResult.ExpiresAt)

	// Wait for the Fibre settlement to hit the chain and propagate through
	// Blob.Subscribe. The adapter filters to share-version-2 so this
	// should be the only event on the channel.
	var event block.FiberBlobEvent
	select {
	case ev, ok := <-events:
		require.True(t, ok, "Listen channel closed without event")
		event = ev
	case <-time.After(listenEventTimeout):
		t.Fatalf("timed out waiting for BlobEvent after %s", listenEventTimeout)
	}

	require.Equal(t, uploadResult.BlobID, event.BlobID,
		"Listen emitted a different BlobID than Upload returned")
	require.Greater(t, event.Height, uint64(0), "event Height must be a real block height")
	require.Equal(t, uint64(len(payload)), event.DataSize,
		"event DataSize should match payload length")
	t.Logf("listen ok: blob_id=%x height=%d data_size=%d",
		event.BlobID, event.Height, event.DataSize)

	got, err := adapter.Download(ctx, uploadResult.BlobID)
	require.NoError(t, err, "adapter.Download")
	require.Equal(t, payload, got, "Download bytes mismatch")
	t.Logf("download ok: %d bytes recovered", len(got))
}
