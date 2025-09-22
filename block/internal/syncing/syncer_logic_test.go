package syncing

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
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	signerpkg "github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/signer/noop"
	"github.com/evstack/ev-node/pkg/store"
	testmocks "github.com/evstack/ev-node/test/mocks"
	"github.com/evstack/ev-node/types"
)

// helper to create a signer, pubkey and address for tests
func buildSyncTestSigner(t *testing.T) (addr []byte, pub crypto.PubKey, signer signerpkg.Signer) {
	t.Helper()
	priv, _, err := crypto.GenerateEd25519Key(crand.Reader)
	require.NoError(t, err)
	n, err := noop.NewNoopSigner(priv)
	require.NoError(t, err)
	a, err := n.GetAddress()
	require.NoError(t, err)
	p, err := n.GetPublic()
	require.NoError(t, err)
	return a, p, n
}

func makeSignedHeader(t *testing.T, chainID string, height uint64, proposer []byte, pub crypto.PubKey, signer signerpkg.Signer, appHash []byte) *types.SignedHeader {
	hdr := &types.SignedHeader{
		Header: types.Header{
			BaseHeader:      types.BaseHeader{ChainID: chainID, Height: height, Time: uint64(time.Now().Add(time.Duration(height) * time.Second).UnixNano())},
			AppHash:         appHash,
			ProposerAddress: proposer,
		},
		Signer: types.Signer{PubKey: pub, Address: proposer},
	}
	// sign using aggregator provider (sync node will re-verify using sync provider, which defaults to same header bytes)
	bz, err := types.DefaultAggregatorNodeSignatureBytesProvider(&hdr.Header)
	require.NoError(t, err)
	sig, err := signer.Sign(bz)
	require.NoError(t, err)
	hdr.Signature = sig
	return hdr
}

func makeData(chainID string, height uint64, txs int) *types.Data {
	d := &types.Data{Metadata: &types.Metadata{ChainID: chainID, Height: height, Time: uint64(time.Now().UnixNano())}}
	if txs > 0 {
		d.Txs = make(types.Txs, txs)
		for i := 0; i < txs; i++ {
			d.Txs[i] = types.Tx([]byte{byte(height), byte(i)})
		}
	}
	return d
}

func TestProcessHeightEvent_SyncsAndUpdatesState(t *testing.T) {
	ds := sync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)

	addr, pub, signer := buildSyncTestSigner(t)

	cfg := config.DefaultConfig()
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}

	mockExec := testmocks.NewMockExecutor(t)

	s := NewSyncer(
		st,
		mockExec,
		nil,
		cm,
		common.NopMetrics(),
		cfg,
		gen,
		nil,
		nil,
		zerolog.Nop(),
		common.DefaultBlockOptions(),
		make(chan error, 1),
	)

	require.NoError(t, s.initializeState())
	// set a context for internal loops that expect it
	s.ctx = context.Background()
	// Create signed header & data for height 1
	lastState := s.GetLastState()
	hdr := makeSignedHeader(t, gen.ChainID, 1, addr, pub, signer, lastState.AppHash)
	data := makeData(gen.ChainID, 1, 0)
	// For empty data, header.DataHash should be set by producer; here we don't rely on it for syncing

	// Expect ExecuteTxs call for height 1
	mockExec.EXPECT().ExecuteTxs(mock.Anything, mock.Anything, uint64(1), mock.Anything, lastState.AppHash).
		Return([]byte("app1"), uint64(1024), nil).Once()

	evt := common.DAHeightEvent{Header: hdr, Data: data, DaHeight: 1}
	s.processHeightEvent(&evt)

	h, err := st.Height(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(1), h)
	st1, err := st.GetState(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(1), st1.LastBlockHeight)
}

func TestSequentialBlockSync(t *testing.T) {
	ds := sync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)

	addr, pub, signer := buildSyncTestSigner(t)
	cfg := config.DefaultConfig()
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}

	mockExec := testmocks.NewMockExecutor(t)

	s := NewSyncer(
		st,
		mockExec,
		nil,
		cm,
		common.NopMetrics(),
		cfg,
		gen,
		nil,
		nil,
		zerolog.Nop(),
		common.DefaultBlockOptions(),
		make(chan error, 1),
	)
	require.NoError(t, s.initializeState())
	s.ctx = context.Background()

	// Sync two consecutive blocks via processHeightEvent so ExecuteTxs is called and state stored
	st0 := s.GetLastState()
	hdr1 := makeSignedHeader(t, gen.ChainID, 1, addr, pub, signer, st0.AppHash)
	data1 := makeData(gen.ChainID, 1, 1) // non-empty
	// Expect ExecuteTxs call for height 1
	mockExec.EXPECT().ExecuteTxs(mock.Anything, mock.Anything, uint64(1), mock.Anything, st0.AppHash).
		Return([]byte("app1"), uint64(1024), nil).Once()
	evt1 := common.DAHeightEvent{Header: hdr1, Data: data1, DaHeight: 10}
	s.processHeightEvent(&evt1)

	st1, _ := st.GetState(context.Background())
	hdr2 := makeSignedHeader(t, gen.ChainID, 2, addr, pub, signer, st1.AppHash)
	data2 := makeData(gen.ChainID, 2, 0) // empty data
	// Expect ExecuteTxs call for height 2
	mockExec.EXPECT().ExecuteTxs(mock.Anything, mock.Anything, uint64(2), mock.Anything, st1.AppHash).
		Return([]byte("app2"), uint64(1024), nil).Once()
	evt2 := common.DAHeightEvent{Header: hdr2, Data: data2, DaHeight: 11}
	s.processHeightEvent(&evt2)

	// Mark DA inclusion in cache (as DA retrieval would)
	cm.SetDataDAIncluded(data1.DACommitment().String(), 10)
	cm.SetDataDAIncluded(data2.DACommitment().String(), 11) // empty data still needs cache entry
	// Header DA inclusion is marked by syncing loop, so it should be true without manual setting.

	// Verify both blocks were synced correctly
	finalState, _ := st.GetState(context.Background())
	assert.Equal(t, uint64(2), finalState.LastBlockHeight)

	// Verify DA inclusion markers are set
	_, ok := cm.GetHeaderDAIncluded(hdr1.Hash().String())
	assert.True(t, ok)
	_, ok = cm.GetHeaderDAIncluded(hdr2.Hash().String())
	assert.True(t, ok)
	_, ok = cm.GetDataDAIncluded(data1.DACommitment().String())
	assert.True(t, ok)
	_, ok = cm.GetDataDAIncluded(data2.DACommitment().String())
	assert.True(t, ok)
}
