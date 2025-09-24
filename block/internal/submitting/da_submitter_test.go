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
	"github.com/evstack/ev-node/pkg/rpc/server"
	"github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/signer/noop"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

func setupDASubmitterTest(t *testing.T) (*DASubmitter, store.Store, cache.Manager, coreda.DA, genesis.Genesis) {
	t.Helper()

	// Create store and cache
	ds := sync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)

	// Create dummy DA
	dummyDA := coreda.NewDummyDA(10_000_000, 0, 0, 10*time.Millisecond)

	// Create config
	cfg := config.DefaultConfig()
	cfg.DA.Namespace = "test-headers"
	cfg.DA.DataNamespace = "test-data"

	// Create genesis
	gen := genesis.Genesis{
		ChainID:         "test-chain",
		InitialHeight:   1,
		StartTime:       time.Now(),
		ProposerAddress: []byte("test-proposer"),
	}

	// Create DA submitter
	daSubmitter := NewDASubmitter(
		dummyDA,
		cfg,
		gen,
		common.DefaultBlockOptions(),
		zerolog.Nop(),
	)

	return daSubmitter, st, cm, dummyDA, gen
}

func createTestSigner(t *testing.T) ([]byte, crypto.PubKey, signer.Signer) {
	t.Helper()
	priv, _, err := crypto.GenerateEd25519Key(crand.Reader)
	require.NoError(t, err)
	signer, err := noop.NewNoopSigner(priv)
	require.NoError(t, err)
	addr, err := signer.GetAddress()
	require.NoError(t, err)
	pub, err := signer.GetPublic()
	require.NoError(t, err)
	return addr, pub, signer
}

func TestDASubmitter_NewDASubmitter(t *testing.T) {
	submitter, _, _, _, _ := setupDASubmitterTest(t)

	assert.NotNil(t, submitter)
	assert.NotNil(t, submitter.da)
	assert.NotNil(t, submitter.config)
	assert.NotNil(t, submitter.genesis)
}

func TestNewDASubmitterSetsVisualizerWhenEnabled(t *testing.T) {
	t.Helper()
	defer server.SetDAVisualizationServer(nil)

	cfg := config.DefaultConfig()
	cfg.RPC.EnableDAVisualization = true
	cfg.Node.Aggregator = true

	dummyDA := coreda.NewDummyDA(10_000_000, 0, 0, 10*time.Millisecond)

	NewDASubmitter(
		dummyDA,
		cfg,
		genesis.Genesis{},
		common.DefaultBlockOptions(),
		zerolog.Nop(),
	)

	require.NotNil(t, server.GetDAVisualizationServer())
}

func TestDASubmitter_SubmitHeaders_Success(t *testing.T) {
	submitter, st, cm, _, gen := setupDASubmitterTest(t)
	ctx := context.Background()

	// Create test signer
	addr, pub, signer := createTestSigner(t)
	gen.ProposerAddress = addr

	// Create test headers
	header1 := &types.SignedHeader{
		Header: types.Header{
			BaseHeader: types.BaseHeader{
				ChainID: gen.ChainID,
				Height:  1,
				Time:    uint64(time.Now().UnixNano()),
			},
			ProposerAddress: addr,
		},
		Signer: types.Signer{PubKey: pub, Address: addr},
	}

	header2 := &types.SignedHeader{
		Header: types.Header{
			BaseHeader: types.BaseHeader{
				ChainID: gen.ChainID,
				Height:  2,
				Time:    uint64(time.Now().UnixNano()),
			},
			ProposerAddress: addr,
		},
		Signer: types.Signer{PubKey: pub, Address: addr},
	}

	// Sign headers
	for _, header := range []*types.SignedHeader{header1, header2} {
		bz, err := types.DefaultAggregatorNodeSignatureBytesProvider(&header.Header)
		require.NoError(t, err)
		sig, err := signer.Sign(bz)
		require.NoError(t, err)
		header.Signature = sig
	}

	// Create empty data for the headers
	data1 := &types.Data{
		Metadata: &types.Metadata{
			ChainID: gen.ChainID,
			Height:  1,
			Time:    uint64(time.Now().UnixNano()),
		},
	}

	data2 := &types.Data{
		Metadata: &types.Metadata{
			ChainID: gen.ChainID,
			Height:  2,
			Time:    uint64(time.Now().UnixNano()),
		},
	}

	// Save to store to make them pending
	sig1 := header1.Signature
	sig2 := header2.Signature
	require.NoError(t, st.SaveBlockData(ctx, header1, data1, &sig1))
	require.NoError(t, st.SaveBlockData(ctx, header2, data2, &sig2))
	require.NoError(t, st.SetHeight(ctx, 2))

	// Submit headers
	err := submitter.SubmitHeaders(ctx, cm)
	require.NoError(t, err)

	// Verify headers are marked as DA included
	hash1 := header1.Hash().String()
	hash2 := header2.Hash().String()
	_, ok1 := cm.GetHeaderDAIncluded(hash1)
	assert.True(t, ok1)
	_, ok2 := cm.GetHeaderDAIncluded(hash2)
	assert.True(t, ok2)
}

func TestDASubmitter_SubmitHeaders_NoPendingHeaders(t *testing.T) {
	submitter, _, cm, _, _ := setupDASubmitterTest(t)
	ctx := context.Background()

	// Submit headers when none are pending
	err := submitter.SubmitHeaders(ctx, cm)
	require.NoError(t, err) // Should succeed with no action
}

func TestDASubmitter_SubmitData_Success(t *testing.T) {
	submitter, st, cm, _, gen := setupDASubmitterTest(t)
	ctx := context.Background()

	// Create test signer
	addr, pub, signer := createTestSigner(t)
	gen.ProposerAddress = addr

	// Update submitter genesis to use correct proposer
	submitter.genesis.ProposerAddress = addr

	// Create test data with transactions
	data1 := &types.Data{
		Metadata: &types.Metadata{
			ChainID: gen.ChainID,
			Height:  1,
			Time:    uint64(time.Now().UnixNano()),
		},
		Txs: types.Txs{types.Tx("tx1"), types.Tx("tx2")},
	}

	data2 := &types.Data{
		Metadata: &types.Metadata{
			ChainID: gen.ChainID,
			Height:  2,
			Time:    uint64(time.Now().UnixNano()),
		},
		Txs: types.Txs{types.Tx("tx3")},
	}

	// Create headers for the data
	header1 := &types.SignedHeader{
		Header: types.Header{
			BaseHeader: types.BaseHeader{
				ChainID: gen.ChainID,
				Height:  1,
				Time:    uint64(time.Now().UnixNano()),
			},
			ProposerAddress: addr,
			DataHash:        data1.DACommitment(),
		},
		Signer: types.Signer{PubKey: pub, Address: addr},
	}

	header2 := &types.SignedHeader{
		Header: types.Header{
			BaseHeader: types.BaseHeader{
				ChainID: gen.ChainID,
				Height:  2,
				Time:    uint64(time.Now().UnixNano()),
			},
			ProposerAddress: addr,
			DataHash:        data2.DACommitment(),
		},
		Signer: types.Signer{PubKey: pub, Address: addr},
	}

	// Save to store to make them pending
	sig1 := types.Signature([]byte("sig1"))
	sig2 := types.Signature([]byte("sig2"))
	require.NoError(t, st.SaveBlockData(ctx, header1, data1, &sig1))
	require.NoError(t, st.SaveBlockData(ctx, header2, data2, &sig2))
	require.NoError(t, st.SetHeight(ctx, 2))

	// Submit data
	err := submitter.SubmitData(ctx, cm, signer, gen)
	require.NoError(t, err)

	// Verify data is marked as DA included
	_, ok := cm.GetDataDAIncluded(data1.DACommitment().String())
	assert.True(t, ok)
	_, ok = cm.GetDataDAIncluded(data2.DACommitment().String())
	assert.True(t, ok)
}

func TestDASubmitter_SubmitData_SkipsEmptyData(t *testing.T) {
	submitter, st, cm, _, gen := setupDASubmitterTest(t)
	ctx := context.Background()

	// Create test signer
	addr, pub, signer := createTestSigner(t)
	gen.ProposerAddress = addr

	// Create empty data
	emptyData := &types.Data{
		Metadata: &types.Metadata{
			ChainID: gen.ChainID,
			Height:  1,
			Time:    uint64(time.Now().UnixNano()),
		},
		Txs: types.Txs{}, // Empty
	}

	// Create header for the empty data
	header := &types.SignedHeader{
		Header: types.Header{
			BaseHeader: types.BaseHeader{
				ChainID: gen.ChainID,
				Height:  1,
				Time:    uint64(time.Now().UnixNano()),
			},
			ProposerAddress: addr,
			DataHash:        common.DataHashForEmptyTxs,
		},
		Signer: types.Signer{PubKey: pub, Address: addr},
	}

	// Save to store
	sig := types.Signature([]byte("sig"))
	require.NoError(t, st.SaveBlockData(ctx, header, emptyData, &sig))
	require.NoError(t, st.SetHeight(ctx, 1))

	// Submit data - should succeed but skip empty data
	err := submitter.SubmitData(ctx, cm, signer, gen)
	require.NoError(t, err)

	// Empty data should not be marked as DA included (it's implicitly included)
	_, ok := cm.GetDataDAIncluded(emptyData.DACommitment().String())
	assert.False(t, ok)
}

func TestDASubmitter_SubmitData_NoPendingData(t *testing.T) {
	submitter, _, cm, _, gen := setupDASubmitterTest(t)
	ctx := context.Background()

	// Create test signer
	_, _, signer := createTestSigner(t)

	// Submit data when none are pending
	err := submitter.SubmitData(ctx, cm, signer, gen)
	require.NoError(t, err) // Should succeed with no action
}

func TestDASubmitter_SubmitData_NilSigner(t *testing.T) {
	submitter, st, cm, _, gen := setupDASubmitterTest(t)
	ctx := context.Background()

	// Create test data with transactions
	data := &types.Data{
		Metadata: &types.Metadata{
			ChainID: gen.ChainID,
			Height:  1,
			Time:    uint64(time.Now().UnixNano()),
		},
		Txs: types.Txs{types.Tx("tx1")},
	}

	header := &types.SignedHeader{
		Header: types.Header{
			BaseHeader: types.BaseHeader{
				ChainID: gen.ChainID,
				Height:  1,
				Time:    uint64(time.Now().UnixNano()),
			},
			DataHash: data.DACommitment(),
		},
	}

	// Save to store
	sig := types.Signature([]byte("sig"))
	require.NoError(t, st.SaveBlockData(ctx, header, data, &sig))
	require.NoError(t, st.SetHeight(ctx, 1))

	// Submit data with nil signer - should fail
	err := submitter.SubmitData(ctx, cm, nil, gen)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "signer is nil")
}

func TestDASubmitter_CreateSignedData(t *testing.T) {
	submitter, _, _, _, gen := setupDASubmitterTest(t)

	// Create test signer
	addr, _, signer := createTestSigner(t)
	gen.ProposerAddress = addr

	// Create test data
	signedData1 := &types.SignedData{
		Data: types.Data{
			Metadata: &types.Metadata{
				ChainID: gen.ChainID,
				Height:  1,
				Time:    uint64(time.Now().UnixNano()),
			},
			Txs: types.Txs{types.Tx("tx1")},
		},
	}

	signedData2 := &types.SignedData{
		Data: types.Data{
			Metadata: &types.Metadata{
				ChainID: gen.ChainID,
				Height:  2,
				Time:    uint64(time.Now().UnixNano()),
			},
			Txs: types.Txs{}, // Empty - should be skipped
		},
	}

	signedData3 := &types.SignedData{
		Data: types.Data{
			Metadata: &types.Metadata{
				ChainID: gen.ChainID,
				Height:  3,
				Time:    uint64(time.Now().UnixNano()),
			},
			Txs: types.Txs{types.Tx("tx3")},
		},
	}

	dataList := []*types.SignedData{signedData1, signedData2, signedData3}

	// Create signed data
	result, err := submitter.createSignedData(dataList, signer, gen)
	require.NoError(t, err)

	// Should have 2 items (empty data skipped)
	assert.Len(t, result, 2)

	// Verify signatures are set
	for _, signedData := range result {
		assert.NotEmpty(t, signedData.Signature)
		assert.NotNil(t, signedData.Signer.PubKey)
		assert.Equal(t, gen.ProposerAddress, signedData.Signer.Address)
	}
}

func TestDASubmitter_CreateSignedData_NilSigner(t *testing.T) {
	submitter, _, _, _, gen := setupDASubmitterTest(t)

	// Create test data
	signedData := &types.SignedData{
		Data: types.Data{
			Metadata: &types.Metadata{
				ChainID: gen.ChainID,
				Height:  1,
			},
			Txs: types.Txs{types.Tx("tx1")},
		},
	}

	dataList := []*types.SignedData{signedData}

	// Create signed data with nil signer - should fail
	_, err := submitter.createSignedData(dataList, nil, gen)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "signer is nil")
}
