//go:build fibre

package cnfibertest_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/celestiaorg/celestia-node/api/client"

	"github.com/evstack/ev-node/block"
	cnfiber "github.com/evstack/ev-node/tools/celestia-node-fiber"
	cnfibertest "github.com/evstack/ev-node/tools/celestia-node-fiber/testing"
)

const (
	// showcaseBlobs is how many distinct-payload blobs the test pushes
	// through the adapter. Large enough to surface ordering and
	// duplicate-handling bugs, small enough to keep wall time reasonable.
	showcaseBlobs = 10

	// listenEventsTimeout bounds the collection window for N BlobEvents.
	// The async MsgPayForFibre broadcasts serialize on the TxClient
	// mutex, so the dominant cost is block_time_per_tx × N. 60s gives
	// ~6s per blob which is generous for a 50ms-precommit testnode.
	listenEventsTimeout = 60 * time.Second
)

// TestShowcase spins up a single-validator Celestia chain with an
// in-process Fibre server, a celestia-node bridge, and drives the full
// adapter surface: Listen subscribes first, Upload pushes N distinct
// blobs, the async MsgPayForFibre settlements commit on-chain, the
// subscription delivers an event per blob, and Download round-trips each
// payload byte-for-byte.
func TestShowcase(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
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

	// Namespace: 10 bytes of v0 ID. Uploads share a namespace so Listen
	// sees every settlement event in one stream.
	namespace := bytes.Repeat([]byte{0xfe}, 10)

	// Subscribe BEFORE uploading so we don't race against settlements.
	// fromHeight=0 → follow from the live tip. TestShowcaseResume below
	// exercises the non-zero path.
	events, err := adapter.Listen(ctx, namespace, 0)
	require.NoError(t, err, "starting Listen subscription")

	// Build N distinctive payloads so byte-swapping or off-by-one BlobID
	// reconstruction would be caught by the download diff below.
	payloads := make([][]byte, showcaseBlobs)
	for i := range payloads {
		payloads[i] = []byte(fmt.Sprintf(
			"showcase blob %02d — payload=%s",
			i, bytes.Repeat([]byte{'a' + byte(i)}, 8+i),
		))
	}

	// expected maps hex(BlobID) → original payload; populated by Upload
	// and consulted by Listen + Download to catch misrouted bytes.
	expected := make(map[string][]byte, showcaseBlobs)
	ids := make([]block.FiberBlobID, showcaseBlobs)
	for i, payload := range payloads {
		res, err := adapter.Upload(ctx, namespace, payload)
		require.NoError(t, err, "adapter.Upload #%d", i)
		require.NotEmpty(t, res.BlobID, "upload #%d returned empty BlobID", i)
		key := hex.EncodeToString(res.BlobID)
		_, dup := expected[key]
		require.False(t, dup, "adapter.Upload #%d returned a duplicate BlobID %s", i, key)
		expected[key] = payload
		ids[i] = res.BlobID
		t.Logf("upload[%02d] ok: blob_id=%s size=%d", i, key, len(payload))
	}

	// Drain events until every Upload has a matching BlobEvent. Order is
	// not guaranteed — multiple settlements can land in the same block.
	seen := make(map[string]block.FiberBlobEvent, showcaseBlobs)
	deadline := time.After(listenEventsTimeout)
	for len(seen) < showcaseBlobs {
		select {
		case ev, ok := <-events:
			require.True(t, ok,
				"Listen channel closed with only %d/%d events", len(seen), showcaseBlobs)
			key := hex.EncodeToString(ev.BlobID)
			if _, want := expected[key]; !want {
				t.Logf("listen: ignoring unexpected BlobID %s", key)
				continue
			}
			if prev, dup := seen[key]; dup {
				t.Fatalf("listen: duplicate event for BlobID %s (prev height=%d new height=%d)",
					key, prev.Height, ev.Height)
			}
			seen[key] = ev
			t.Logf("listen[%02d/%02d] ok: blob_id=%s height=%d data_size=%d",
				len(seen), showcaseBlobs, key, ev.Height, ev.DataSize)
		case <-deadline:
			missing := make([]string, 0, showcaseBlobs-len(seen))
			for k := range expected {
				if _, got := seen[k]; !got {
					missing = append(missing, k)
				}
			}
			t.Fatalf("timed out after %s: got %d/%d events; missing=%v",
				listenEventsTimeout, len(seen), showcaseBlobs, missing)
		}
	}

	// Every event must carry the right DataSize and a non-zero block
	// height. DataSize matches the original payload length because the
	// adapter's Listen issues a Download per event to recover it (see
	// listen.go). A silent byte truncation anywhere upstream would
	// surface here before we even get to the Download round-trip.
	for key, ev := range seen {
		require.Greater(t, ev.Height, uint64(0),
			"BlobEvent %s must carry a real block height", key)
		require.Equal(t, uint64(len(expected[key])), ev.DataSize,
			"BlobEvent %s DataSize must match original payload length", key)
	}

	// Round-trip every blob through Download and diff bytes. Walking
	// ids (upload order) rather than seen (map iteration order) keeps
	// log output deterministic.
	for i, id := range ids {
		key := hex.EncodeToString(id)
		got, err := adapter.Download(ctx, id)
		require.NoError(t, err, "adapter.Download #%d (%s)", i, key)
		require.Equal(t, expected[key], got,
			"Download #%d (%s) bytes mismatch", i, key)
		t.Logf("download[%02d] ok: blob_id=%s bytes=%d", i, key, len(got))
	}
}

// TestShowcaseResume verifies that a subscriber can rejoin the stream
// from a historical block height and receive every matching blob that
// was settled since. This is the fromHeight > 0 path added by
// celestia-node#4962: Listen opens a WaitForHeight loop starting at the
// requested height, so callers can resume after a restart without
// missing blobs.
func TestShowcaseResume(t *testing.T) {
	const resumeBlobs = 3

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
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

	namespace := bytes.Repeat([]byte{0xfd}, 10)

	// Phase 1: open a live subscription, upload N blobs, and harvest
	// each blob's settlement height as reported by Listen. These
	// heights are the ground truth for the resume test below.
	liveCtx, liveCancel := context.WithCancel(ctx)
	liveEvents, err := adapter.Listen(liveCtx, namespace, 0)
	require.NoError(t, err, "starting live Listen for height discovery")

	payloads := make([][]byte, resumeBlobs)
	ids := make([]block.FiberBlobID, resumeBlobs)
	expected := make(map[string][]byte, resumeBlobs)
	for i := range payloads {
		payloads[i] = []byte(fmt.Sprintf("resume blob %d", i))
		res, err := adapter.Upload(ctx, namespace, payloads[i])
		require.NoError(t, err, "upload #%d", i)
		ids[i] = res.BlobID
		expected[hex.EncodeToString(res.BlobID)] = payloads[i]
		t.Logf("phase1 upload[%d] blob_id=%s", i, hex.EncodeToString(res.BlobID))
	}

	heights := make(map[string]uint64, resumeBlobs)
	for len(heights) < resumeBlobs {
		select {
		case ev, ok := <-liveEvents:
			require.True(t, ok, "live Listen channel closed early")
			key := hex.EncodeToString(ev.BlobID)
			if _, want := expected[key]; !want {
				continue
			}
			heights[key] = ev.Height
			t.Logf("phase1 listen blob_id=%s height=%d", key, ev.Height)
		case <-time.After(listenEventsTimeout):
			t.Fatalf("timed out collecting heights: got %d/%d", len(heights), resumeBlobs)
		}
	}
	liveCancel()

	// Pick the smallest height across all uploads. Resuming from this
	// height must replay every blob we uploaded, regardless of whether
	// multiple settlements landed in the same block.
	var fromHeight uint64
	for _, h := range heights {
		if fromHeight == 0 || h < fromHeight {
			fromHeight = h
		}
	}
	t.Logf("resume fromHeight=%d", fromHeight)

	// Phase 2: fresh Listen starting at fromHeight. Expect every blob
	// we uploaded in phase 1 to be replayed.
	resumeEvents, err := adapter.Listen(ctx, namespace, fromHeight)
	require.NoError(t, err, "starting resume Listen")

	seen := make(map[string]block.FiberBlobEvent, resumeBlobs)
	for len(seen) < resumeBlobs {
		select {
		case ev, ok := <-resumeEvents:
			require.True(t, ok, "resume Listen channel closed early")
			key := hex.EncodeToString(ev.BlobID)
			if _, want := expected[key]; !want {
				continue
			}
			if _, dup := seen[key]; dup {
				t.Fatalf("resume Listen emitted duplicate for BlobID %s", key)
			}
			seen[key] = ev
			t.Logf("phase2 listen[%d/%d] blob_id=%s height=%d data_size=%d",
				len(seen), resumeBlobs, key, ev.Height, ev.DataSize)
		case <-time.After(listenEventsTimeout):
			missing := make([]string, 0, resumeBlobs-len(seen))
			for k := range expected {
				if _, got := seen[k]; !got {
					missing = append(missing, k)
				}
			}
			t.Fatalf("timed out on resume Listen: got %d/%d; missing=%v",
				len(seen), resumeBlobs, missing)
		}
	}

	// Every resume event must carry the correct DataSize (Download-
	// resolved, same as the live Listen path) and the right height.
	for key, ev := range seen {
		require.Equal(t, uint64(len(expected[key])), ev.DataSize,
			"resume BlobEvent %s DataSize must match original payload length", key)
		require.Equal(t, heights[key], ev.Height,
			"resume BlobEvent %s Height must match the block it settled in", key)
	}
}
