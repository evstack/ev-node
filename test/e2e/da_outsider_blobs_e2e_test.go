//go:build e2e && evm

package e2e

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/celestiaorg/go-square/v3/share"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/stretchr/testify/require"

	blobrpc "github.com/evstack/ev-node/pkg/da/jsonrpc"
	datypes "github.com/evstack/ev-node/pkg/da/types"
	nodeclient "github.com/evstack/ev-node/pkg/rpc/client"
	"github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/signer/noop"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

// These tests post blobs signed by a key that is not the sequencer's into the
// chain's DA namespaces, and check that full nodes are not affected by them.

const (
	outsiderChainID  = "testing"
	outsiderHeaderNS = "outsider-hdr-ns"
	outsiderDataNS   = "outsider-data-ns"
)

// A full node syncing only from DA must not stall when an outsider pre-posts
// signed data for a future height.
func TestDAOutsiderDataDoesNotStallDASync(t *testing.T) {
	nw := startOutsiderNet(t)
	target := nw.seqHeight(t) + 50

	nw.outsider.postData(t, nw.daAddr, target)

	p2pNode := nw.startFullNode(t, nw.daAddr, true)
	daNode := nw.startFullNode(t, nw.daAddr, false)

	nw.awaitHeight(t, nw.seqRPC, target+20, 60*time.Second)
	nw.awaitHeight(t, p2pNode, target+5, 30*time.Second)
	nw.awaitHeight(t, daNode, target+5, 30*time.Second)
}

// A full node syncing only from DA must not stall when an outsider pre-posts
// headers for future heights.
func TestDAOutsiderHeaderDoesNotStallDASync(t *testing.T) {
	nw := startOutsiderNet(t)
	from := nw.seqHeight(t) + 50
	const squatted = 10

	for h := from; h < from+squatted; h++ {
		nw.outsider.postPhantomHeader(t, nw.daAddr, h)
	}

	daNode := nw.startFullNode(t, nw.daAddr, false)

	nw.awaitHeight(t, nw.seqRPC, from+squatted+20, 60*time.Second)
	nw.awaitHeight(t, daNode, from+squatted+5, 30*time.Second)
}

// A full node must not report blocks as DA-included when the sequencer never
// posted them to the DA layer the node follows. The sequencer submits to its
// own DA; the full node syncs blocks over P2P and follows a second DA that
// only the outsider posts to.
func TestDAOutsiderBlobsDoNotForgeDAInclusion(t *testing.T) {
	nw := startOutsiderNet(t)
	followedDA := nw.startLocalDA(t)

	node := nw.startFullNode(t, followedDA, true)
	nw.awaitHeight(t, node, 20, 30*time.Second)
	synced := nw.height(t, node)

	// Every height is at or below the node's height, so none is its "next"
	// height: the retriever skips the proposer check, and the syncer drops the
	// pair as already processed without validating it.
	for h := uint64(1); h <= synced; h++ {
		nw.outsider.postBlock(t, followedDA, h)
	}

	time.Sleep(10 * time.Second)
	if got := nw.daIncludedHeight(t, node); got > 0 {
		t.Fatalf("full node reported DA-included height %d (outsider covered 1..%d) although the sequencer posted nothing to its DA", got, synced)
	}
}

type outsiderNet struct {
	sut     *SystemUnderTest
	workDir string
	daAddr  string
	seqHome string
	seqRPC  string
	seqP2P  string

	outsider *outsider
}

func startOutsiderNet(t *testing.T) *outsiderNet {
	t.Helper()
	flag.Parse()
	n := &outsiderNet{sut: NewSystemUnderTest(t), workDir: t.TempDir(), outsider: newOutsider(t)}
	n.daAddr = n.startLocalDA(t)

	n.seqHome = filepath.Join(n.workDir, "sequencer")
	passphrase := filepath.Join(n.workDir, "passphrase.txt")
	require.NoError(t, os.WriteFile(passphrase, []byte("12345678"), 0o600))

	out, err := n.sut.RunCmd(binaryPath, "init",
		"--home="+n.seqHome,
		"--chain_id="+outsiderChainID,
		"--evnode.node.aggregator",
		"--evnode.signer.passphrase_file="+passphrase,
	)
	require.NoError(t, err, out)

	rpcPort, p2pPort := mustGetAvailablePort(t), mustGetAvailablePort(t)
	n.seqRPC = fmt.Sprintf("http://127.0.0.1:%d", rpcPort)
	n.seqP2P = fmt.Sprintf("/ip4/127.0.0.1/tcp/%d/p2p/%s", p2pPort, NodeID(t, n.seqHome))
	n.sut.ExecCmdWithLogPrefix("sequencer", binaryPath, "start",
		"--home="+n.seqHome,
		"--evnode.node.aggregator",
		"--evnode.signer.passphrase_file="+passphrase,
		"--evnode.node.block_time=100ms",
		"--evnode.da.block_time=500ms",
		"--evnode.da.address="+n.daAddr,
		"--evnode.da.namespace="+outsiderHeaderNS,
		"--evnode.da.data_namespace="+outsiderDataNS,
		fmt.Sprintf("--evnode.rpc.address=127.0.0.1:%d", rpcPort),
		fmt.Sprintf("--evnode.p2p.listen_address=/ip4/127.0.0.1/tcp/%d", p2pPort),
		fmt.Sprintf("--kv-endpoint=127.0.0.1:%d", mustGetAvailablePort(t)),
	)
	n.sut.AwaitNodeUp(t, n.seqRPC, 10*time.Second)
	n.awaitHeight(t, n.seqRPC, 3, 10*time.Second)
	return n
}

func (n *outsiderNet) startLocalDA(t *testing.T) string {
	t.Helper()
	port := mustGetAvailablePort(t)
	n.sut.ExecCmdWithLogPrefix(fmt.Sprintf("local-da-%d", port), filepath.Join(filepath.Dir(binaryPath), "local-da"), "-port", fmt.Sprint(port))
	require.Eventually(t, func() bool {
		conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			return false
		}
		conn.Close()
		return true
	}, 5*time.Second, 100*time.Millisecond)
	return fmt.Sprintf("http://127.0.0.1:%d", port)
}

// startFullNode starts a full node following daAddr, peered with the sequencer
// over P2P only when withP2P is set. It returns the node's RPC URL.
func (n *outsiderNet) startFullNode(t *testing.T, daAddr string, withP2P bool) string {
	t.Helper()
	rpcPort, p2pPort := mustGetAvailablePort(t), mustGetAvailablePort(t)
	home := filepath.Join(n.workDir, fmt.Sprintf("fullnode-%d", rpcPort))
	out, err := n.sut.RunCmd(binaryPath, "init", "--home="+home, "--chain_id="+outsiderChainID)
	require.NoError(t, err, out)
	MustCopyFile(t, filepath.Join(n.seqHome, "config", "genesis.json"), filepath.Join(home, "config", "genesis.json"))

	args := []string{"start",
		"--home=" + home,
		"--evnode.da.address=" + daAddr,
		"--evnode.da.namespace=" + outsiderHeaderNS,
		"--evnode.da.data_namespace=" + outsiderDataNS,
		"--evnode.da.block_time=500ms",
		"--evnode.log.level=debug",
		fmt.Sprintf("--evnode.rpc.address=127.0.0.1:%d", rpcPort),
		fmt.Sprintf("--evnode.p2p.listen_address=/ip4/127.0.0.1/tcp/%d", p2pPort),
		fmt.Sprintf("--kv-endpoint=127.0.0.1:%d", mustGetAvailablePort(t)),
	}
	if withP2P {
		args = append(args, "--evnode.p2p.peers="+n.seqP2P)
	}
	rpc := fmt.Sprintf("http://127.0.0.1:%d", rpcPort)
	n.sut.ExecCmdWithLogPrefix(fmt.Sprintf("fullnode-%d", rpcPort), binaryPath, args...)
	n.sut.AwaitNodeLive(t, rpc, 10*time.Second)
	return rpc
}

func (n *outsiderNet) height(t *testing.T, rpc string) uint64 {
	t.Helper()
	state, err := nodeclient.NewClient(rpc).GetState(t.Context())
	if err != nil {
		return 0
	}
	return state.LastBlockHeight
}

func (n *outsiderNet) seqHeight(t *testing.T) uint64 { return n.height(t, n.seqRPC) }

func (n *outsiderNet) awaitHeight(t *testing.T, rpc string, want uint64, timeout time.Duration) {
	t.Helper()
	for deadline := time.Now().Add(timeout); time.Now().Before(deadline); time.Sleep(200 * time.Millisecond) {
		if n.height(t, rpc) >= want {
			return
		}
	}
	t.Fatalf("%s stuck at height %d, want >= %d (sequencer at %d)", rpc, n.height(t, rpc), want, n.seqHeight(t))
}

func (n *outsiderNet) daIncludedHeight(t *testing.T, rpc string) uint64 {
	t.Helper()
	bz, err := nodeclient.NewClient(rpc).GetMetadata(t.Context(), store.DAIncludedHeightKey)
	if err != nil || len(bz) != 8 {
		return 0
	}
	return binary.LittleEndian.Uint64(bz)
}

type outsider struct {
	signer signer.Signer
	pub    crypto.PubKey
	addr   []byte
}

func newOutsider(t *testing.T) *outsider {
	t.Helper()
	priv, pub, err := crypto.GenerateEd25519Key(rand.Reader)
	require.NoError(t, err)
	s, err := noop.NewNoopSigner(priv)
	require.NoError(t, err)
	addr, err := s.GetAddress()
	require.NoError(t, err)
	return &outsider{signer: s, pub: pub, addr: addr}
}

// postData posts outsider-signed data with valid metadata for height.
func (o *outsider) postData(t *testing.T, daAddr string, height uint64) {
	t.Helper()
	submitBlobs(t, daAddr, o.dataBlob(t, o.data(height)))
}

// postBlock posts an outsider header and the data its DataHash commits to.
func (o *outsider) postBlock(t *testing.T, daAddr string, height uint64) {
	t.Helper()
	d := o.data(height)
	submitBlobs(t, daAddr, o.headerBlob(t, height, d.Time(), d.DACommitment()), o.dataBlob(t, d))
}

// postPhantomHeader posts an outsider header for height whose DataHash points
// at data that never lands.
func (o *outsider) postPhantomHeader(t *testing.T, daAddr string, height uint64) {
	t.Helper()
	phantom := types.Data{Txs: types.Txs{types.Tx(fmt.Sprintf("phantom-%d", height))}}
	submitBlobs(t, daAddr, o.headerBlob(t, height, time.Now(), phantom.DACommitment()))
}

func (o *outsider) data(height uint64) types.Data {
	return types.Data{
		Metadata: &types.Metadata{ChainID: outsiderChainID, Height: height, Time: uint64(time.Now().UnixNano())},
		Txs:      types.Txs{types.Tx(fmt.Sprintf("outsider-%d", height))},
	}
}

func (o *outsider) dataBlob(t *testing.T, d types.Data) *blobrpc.Blob {
	t.Helper()
	payload, err := d.MarshalBinary()
	require.NoError(t, err)
	sig, err := o.signer.Sign(t.Context(), payload)
	require.NoError(t, err)
	sd := types.SignedData{Data: d, Signature: sig, Signer: types.Signer{PubKey: o.pub, Address: o.addr}}
	bz, err := sd.MarshalBinary()
	require.NoError(t, err)
	return newBlob(t, outsiderDataNS, bz)
}

// headerBlob builds an outsider header wrapped in a DA envelope signed by the outsider.
func (o *outsider) headerBlob(t *testing.T, height uint64, ts time.Time, dataHash types.Hash) *blobrpc.Blob {
	t.Helper()
	h := types.SignedHeader{
		Header: types.Header{
			BaseHeader:      types.BaseHeader{ChainID: outsiderChainID, Height: height, Time: uint64(ts.UnixNano())},
			DataHash:        dataHash,
			ProposerAddress: o.addr,
		},
		Signer: types.Signer{PubKey: o.pub, Address: o.addr},
	}
	sigBytes, err := types.DefaultAggregatorNodeSignatureBytesProvider(&h.Header)
	require.NoError(t, err)
	h.Signature, err = o.signer.Sign(t.Context(), sigBytes)
	require.NoError(t, err)

	payload, err := h.MarshalBinary()
	require.NoError(t, err)
	envelopeSig, err := o.signer.Sign(t.Context(), payload)
	require.NoError(t, err)
	bz, err := h.MarshalDAEnvelope(envelopeSig)
	require.NoError(t, err)
	return newBlob(t, outsiderHeaderNS, bz)
}

func newBlob(t *testing.T, namespace string, bz []byte) *blobrpc.Blob {
	t.Helper()
	ns, err := share.NewNamespaceFromBytes(datypes.NamespaceFromString(namespace).Bytes())
	require.NoError(t, err)
	blob, err := blobrpc.NewBlobV0(ns, bz)
	require.NoError(t, err)
	return blob
}

// submitBlobs posts blobs in a single DA submission, so they land at one DA height.
func submitBlobs(t *testing.T, daAddr string, blobs ...*blobrpc.Blob) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	c, err := blobrpc.NewClient(ctx, daAddr, "", "")
	require.NoError(t, err)
	defer c.Close()
	_, err = c.Blob.Submit(ctx, blobs, nil)
	require.NoError(t, err)
}
