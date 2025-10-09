package submitting

import (
	"context"
	crand "crypto/rand"
	"testing"
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	coreda "github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/signer/noop"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

func TestDASubmitter_SubmitHeadersAndData_MarksInclusionAndUpdatesLastSubmitted(t *testing.T) {
	ds := sync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)

	// signer and proposer
	priv, _, err := crypto.GenerateEd25519Key(crand.Reader)
	require.NoError(t, err)
	n, err := noop.NewNoopSigner(priv)
	require.NoError(t, err)
	addr, err := n.GetAddress()
	require.NoError(t, err)
	pub, err := n.GetPublic()
	require.NoError(t, err)

	cfg := config.DefaultConfig()
	cfg.DA.Namespace = "ns-header"
	cfg.DA.DataNamespace = "ns-data"

	gen := genesis.Genesis{ChainID: "chain1", InitialHeight: 1, StartTime: time.Now(), ProposerAddress: addr}

	// seed store with two heights
	stateRoot := []byte{1, 2, 3}
	// height 1
	hdr1 := &types.SignedHeader{Header: types.Header{BaseHeader: types.BaseHeader{ChainID: gen.ChainID, Height: 1, Time: uint64(time.Now().UnixNano())}, AppHash: stateRoot, ProposerAddress: addr}, Signer: types.Signer{PubKey: pub, Address: addr}}
	bz1, err := types.DefaultAggregatorNodeSignatureBytesProvider(&hdr1.Header)
	require.NoError(t, err)
	sig1, err := n.Sign(bz1)
	require.NoError(t, err)
	hdr1.Signature = sig1
	data1 := &types.Data{Metadata: &types.Metadata{ChainID: gen.ChainID, Height: 1, Time: uint64(time.Now().UnixNano())}, Txs: types.Txs{types.Tx("a")}}
	// height 2
	hdr2 := &types.SignedHeader{Header: types.Header{BaseHeader: types.BaseHeader{ChainID: gen.ChainID, Height: 2, Time: uint64(time.Now().Add(time.Second).UnixNano())}, AppHash: stateRoot, ProposerAddress: addr}, Signer: types.Signer{PubKey: pub, Address: addr}}
	bz2, err := types.DefaultAggregatorNodeSignatureBytesProvider(&hdr2.Header)
	require.NoError(t, err)
	sig2, err := n.Sign(bz2)
	require.NoError(t, err)
	hdr2.Signature = sig2
	data2 := &types.Data{Metadata: &types.Metadata{ChainID: gen.ChainID, Height: 2, Time: uint64(time.Now().Add(time.Second).UnixNano())}, Txs: types.Txs{types.Tx("b")}}

	// persist to store
	sig1t := types.Signature(sig1)
	sig2t := types.Signature(sig2)

	// Save block 1
	batch1, err := st.NewBatch(context.Background())
	require.NoError(t, err)
	require.NoError(t, batch1.SaveBlockData(hdr1, data1, &sig1t))
	require.NoError(t, batch1.SetHeight(1))
	require.NoError(t, batch1.Commit())

	// Save block 2
	batch2, err := st.NewBatch(context.Background())
	require.NoError(t, err)
	require.NoError(t, batch2.SaveBlockData(hdr2, data2, &sig2t))
	require.NoError(t, batch2.SetHeight(2))
	require.NoError(t, batch2.Commit())

	// Dummy DA
	dummyDA := coreda.NewDummyDA(10_000_000, 0, 0, 10*time.Millisecond)

	// Create DA submitter
	daSubmitter := NewDASubmitter(dummyDA, cfg, gen, common.DefaultBlockOptions(), zerolog.Nop())

	// Submit headers and data
	require.NoError(t, daSubmitter.SubmitHeaders(context.Background(), cm))
	require.NoError(t, daSubmitter.SubmitData(context.Background(), cm, n, gen))

	// After submission, inclusion markers should be set
	_, ok := cm.GetHeaderDAIncluded(hdr1.Hash().String())
	assert.True(t, ok)
	_, ok = cm.GetHeaderDAIncluded(hdr2.Hash().String())
	assert.True(t, ok)
	_, ok = cm.GetDataDAIncluded(data1.DACommitment().String())
	assert.True(t, ok)
	_, ok = cm.GetDataDAIncluded(data2.DACommitment().String())
	assert.True(t, ok)

}
