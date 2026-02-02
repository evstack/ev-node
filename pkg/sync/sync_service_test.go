package sync

import (
	"context"
	cryptoRand "crypto/rand"
	"math/rand"
	"path/filepath"
	"testing"
	"time"

	"github.com/evstack/ev-node/pkg/config"
	genesispkg "github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/p2p"
	"github.com/evstack/ev-node/pkg/p2p/key"
	"github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/signer/noop"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	"github.com/libp2p/go-libp2p/core/crypto"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockStore struct {
	mock.Mock
}

func (m *mockStore) Height(ctx context.Context) (uint64, error) {
	args := m.Called(ctx)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *mockStore) GetBlockData(ctx context.Context, height uint64) (*types.SignedHeader, *types.Data, error) {
	args := m.Called(ctx, height)
	return args.Get(0).(*types.SignedHeader), args.Get(1).(*types.Data), args.Error(2)
}

func (m *mockStore) GetBlockByHash(ctx context.Context, hash []byte) (*types.SignedHeader, *types.Data, error) {
	args := m.Called(ctx, hash)
	return args.Get(0).(*types.SignedHeader), args.Get(1).(*types.Data), args.Error(2)
}

func (m *mockStore) GetHeader(ctx context.Context, height uint64) (*types.SignedHeader, error) {
	args := m.Called(ctx, height)
	return args.Get(0).(*types.SignedHeader), args.Error(1)
}

func (m *mockStore) GetStateAtHeight(ctx context.Context, height uint64) (types.State, error) {
	args := m.Called(ctx, height)
	return args.Get(0).(types.State), args.Error(1)
}

func (m *mockStore) GetSignature(ctx context.Context, height uint64) (*types.Signature, error) {
	args := m.Called(ctx, height)
	return args.Get(0).(*types.Signature), args.Error(1)
}

func (m *mockStore) GetSignatureByHash(ctx context.Context, hash []byte) (*types.Signature, error) {
	args := m.Called(ctx, hash)
	return args.Get(0).(*types.Signature), args.Error(1)
}

func (m *mockStore) GetState(ctx context.Context) (types.State, error) {
	args := m.Called(ctx)
	return args.Get(0).(types.State), args.Error(1)
}

func (m *mockStore) GetMetadata(ctx context.Context, key string) ([]byte, error) {
	args := m.Called(ctx, key)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *mockStore) SetMetadata(ctx context.Context, key string, value []byte) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}

func (m *mockStore) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockStore) NewBatch(ctx context.Context) (store.Batch, error) {
	args := m.Called(ctx)
	return args.Get(0).(store.Batch), args.Error(1)
}

func (m *mockStore) Rollback(ctx context.Context, height uint64, aggregator bool) error {
	args := m.Called(ctx, height, aggregator)
	return args.Error(0)
}

func TestDAHintFromDAStore(t *testing.T) {
	mainKV := sync.MutexWrap(datastore.NewMapDatastore())
	daStore := new(mockStore)

	pk, _, err := crypto.GenerateEd25519Key(cryptoRand.Reader)
	require.NoError(t, err)
	noopSigner, err := noop.NewNoopSigner(pk)
	require.NoError(t, err)
	rnd := rand.New(rand.NewSource(1)) // nolint:gosec // test code only
	mn := mocknet.New()

	chainId := "test-chain-id"
	genesisDoc := genesispkg.Genesis{
		ChainID:         chainId,
		StartTime:       time.Now(),
		InitialHeight:   1,
		ProposerAddress: []byte("test"),
	}
	conf := config.DefaultConfig()
	conf.RootDir = t.TempDir()
	nodeKey, err := key.LoadOrGenNodeKey(filepath.Dir(conf.ConfigPath()))
	require.NoError(t, err)
	logger := zerolog.Nop()

	p2pHost, err := mn.AddPeer(nodeKey.PrivKey, nil)
	require.NoError(t, err)
	p2pClient, err := p2p.NewClientWithHost(conf.P2P, nodeKey.PrivKey, mainKV, chainId, logger, p2p.NopMetrics(), p2pHost)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	require.NoError(t, p2pClient.Start(ctx))

	// Prepare data
	height := uint64(1)
	daHeight := uint64(100)
	headerConfig := types.HeaderConfig{
		Height:   height,
		DataHash: bytesN(rnd, 32),
		AppHash:  bytesN(rnd, 32),
		Signer:   noopSigner,
	}
	signedHeader, err := types.GetRandomSignedHeaderCustom(&headerConfig, chainId)
	require.NoError(t, err)
	data := &types.Data{
		Metadata: &types.Metadata{
			ChainID: chainId,
			Height:  height,
		},
		Txs: types.Txs{[]byte("tx1")},
	}
	headerHash := signedHeader.Hash()
	dataHash := data.Hash()

	state := types.State{
		ChainID:         chainId,
		LastBlockHeight: height,
		DAHeight:        daHeight,
	}

	daStore.On("GetHeader", mock.Anything, height).Return(signedHeader, nil)
	daStore.On("GetBlockByHash", mock.Anything, []byte(headerHash)).Return(signedHeader, data, nil)
	daStore.On("GetBlockByHash", mock.Anything, []byte(dataHash)).Return(signedHeader, data, nil)
	daStore.On("GetBlockData", mock.Anything, height).Return(signedHeader, data, nil)
	daStore.On("GetStateAtHeight", mock.Anything, height).Return(state, nil)

	t.Run("header sync service", func(t *testing.T) {
		headerSvc, err := NewHeaderSyncService(mainKV, daStore, conf, genesisDoc, p2pClient, logger)
		require.NoError(t, err)

		h, err := headerSvc.getterByHeight(ctx, height)
		require.NoError(t, err)
		assert.Equal(t, daHeight, h.DAHint())
		assert.Equal(t, headerHash, h.Hash())

		h2, err := headerSvc.getter(ctx, headerHash)
		require.NoError(t, err)
		assert.Equal(t, daHeight, h2.DAHint())

		hRange, next, err := headerSvc.rangeGetter(ctx, height, height+1)
		require.NoError(t, err)
		assert.Equal(t, height+1, next)
		require.Len(t, hRange, 1)
		assert.Equal(t, daHeight, hRange[0].DAHint())
	})

	t.Run("data sync service", func(t *testing.T) {
		dataSvc, err := NewDataSyncService(mainKV, daStore, conf, genesisDoc, p2pClient, logger)
		require.NoError(t, err)

		d, err := dataSvc.getterByHeight(ctx, height)
		require.NoError(t, err)
		assert.Equal(t, daHeight, d.DAHint())
		assert.Equal(t, dataHash, d.Hash())

		d2, err := dataSvc.getter(ctx, dataHash)
		require.NoError(t, err)
		assert.Equal(t, daHeight, d2.DAHint())

		dRange, next, err := dataSvc.rangeGetter(ctx, height, height+1)
		require.NoError(t, err)
		assert.Equal(t, height+1, next)
		require.Len(t, dRange, 1)
		assert.Equal(t, daHeight, dRange[0].DAHint())
	})
}

func TestDAHintFromDAStoreResilience(t *testing.T) {
	mainKV := sync.MutexWrap(datastore.NewMapDatastore())
	daStore := new(mockStore)

	pk, _, err := crypto.GenerateEd25519Key(cryptoRand.Reader)
	require.NoError(t, err)
	noopSigner, err := noop.NewNoopSigner(pk)
	require.NoError(t, err)
	rnd := rand.New(rand.NewSource(1)) // nolint:gosec // test code only
	mn := mocknet.New()

	chainId := "test-chain-id"
	genesisDoc := genesispkg.Genesis{
		ChainID:         chainId,
		StartTime:       time.Now(),
		InitialHeight:   1,
		ProposerAddress: []byte("test"),
	}
	conf := config.DefaultConfig()
	conf.RootDir = t.TempDir()
	nodeKey, err := key.LoadOrGenNodeKey(filepath.Dir(conf.ConfigPath()))
	require.NoError(t, err)
	logger := zerolog.Nop()

	p2pHost, err := mn.AddPeer(nodeKey.PrivKey, nil)
	require.NoError(t, err)
	p2pClient, err := p2p.NewClientWithHost(conf.P2P, nodeKey.PrivKey, mainKV, chainId, logger, p2p.NopMetrics(), p2pHost)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	require.NoError(t, p2pClient.Start(ctx))

	// Prepare data
	height := uint64(1)
	headerConfig := types.HeaderConfig{
		Height:   height,
		DataHash: bytesN(rnd, 32),
		AppHash:  bytesN(rnd, 32),
		Signer:   noopSigner,
	}
	signedHeader, err := types.GetRandomSignedHeaderCustom(&headerConfig, chainId)
	require.NoError(t, err)
	data := &types.Data{
		Metadata: &types.Metadata{
			ChainID: chainId,
			Height:  height,
		},
		Txs: types.Txs{[]byte("tx1")},
	}
	headerHash := signedHeader.Hash()
	dataHash := data.Hash()

	daStore.On("GetHeader", mock.Anything, height).Return(signedHeader, nil)
	daStore.On("GetBlockByHash", mock.Anything, []byte(headerHash)).Return(signedHeader, data, nil)
	daStore.On("GetBlockByHash", mock.Anything, []byte(dataHash)).Return(signedHeader, data, nil)
	daStore.On("GetBlockData", mock.Anything, height).Return(signedHeader, data, nil)
	// Return "not found" error for state
	daStore.On("GetStateAtHeight", mock.Anything, height).Return(types.State{}, store.ErrNotFound)

	t.Run("header sync service resilience", func(t *testing.T) {
		headerSvc, err := NewHeaderSyncService(mainKV, daStore, conf, genesisDoc, p2pClient, logger)
		require.NoError(t, err)

		h, err := headerSvc.getterByHeight(ctx, height)
		require.NoError(t, err)
		assert.Equal(t, uint64(0), h.DAHint())
		assert.Equal(t, headerHash, h.Hash())

		h2, err := headerSvc.getter(ctx, headerHash)
		require.NoError(t, err)
		assert.Equal(t, uint64(0), h2.DAHint())
	})

	t.Run("data sync service resilience", func(t *testing.T) {
		dataSvc, err := NewDataSyncService(mainKV, daStore, conf, genesisDoc, p2pClient, logger)
		require.NoError(t, err)

		d, err := dataSvc.getterByHeight(ctx, height)
		require.NoError(t, err)
		assert.Equal(t, uint64(0), d.DAHint())
		assert.Equal(t, dataHash, d.Hash())

		d2, err := dataSvc.getter(ctx, dataHash)
		require.NoError(t, err)
		assert.Equal(t, uint64(0), d2.DAHint())
	})
}

func TestHeaderSyncServiceRestart(t *testing.T) {
	mainKV := sync.MutexWrap(datastore.NewMapDatastore())
	pk, _, err := crypto.GenerateEd25519Key(cryptoRand.Reader)
	require.NoError(t, err)
	noopSigner, err := noop.NewNoopSigner(pk)
	require.NoError(t, err)
	rnd := rand.New(rand.NewSource(1)) // nolint:gosec // test code only
	mn := mocknet.New()

	chainId := "test-chain-id"

	proposerAddr := []byte("test")
	genesisDoc := genesispkg.Genesis{
		ChainID:         chainId,
		StartTime:       time.Now(),
		InitialHeight:   1,
		ProposerAddress: proposerAddr,
	}
	conf := config.DefaultConfig()
	conf.RootDir = t.TempDir()
	nodeKey, err := key.LoadOrGenNodeKey(filepath.Dir(conf.ConfigPath()))
	require.NoError(t, err)
	logger := zerolog.Nop()
	priv := nodeKey.PrivKey
	h, err := mn.AddPeer(priv, nil)
	require.NoError(t, err)

	p2pClient, err := p2p.NewClientWithHost(conf.P2P, nodeKey.PrivKey, mainKV, chainId, logger, p2p.NopMetrics(), h)
	require.NoError(t, err)

	// Start p2p client before creating sync service
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	require.NoError(t, p2pClient.Start(ctx))

	rktStore := store.New(mainKV)
	svc, err := NewHeaderSyncService(rktStore, conf, genesisDoc, p2pClient, logger)
	require.NoError(t, err)
	err = svc.Start(ctx)
	require.NoError(t, err)

	// broadcast genesis block
	headerConfig := types.HeaderConfig{
		Height:   genesisDoc.InitialHeight,
		DataHash: bytesN(rnd, 32),
		AppHash:  bytesN(rnd, 32),
		Signer:   noopSigner,
	}
	signedHeader, err := types.GetRandomSignedHeaderCustom(&headerConfig, genesisDoc.ChainID)
	require.NoError(t, err)
	require.NoError(t, signedHeader.Validate())
	require.NoError(t, svc.WriteToStoreAndBroadcast(ctx, &types.P2PSignedHeader{Message: signedHeader}))

	for i := genesisDoc.InitialHeight + 1; i < 2; i++ {
		signedHeader = nextHeader(t, signedHeader, genesisDoc.ChainID, noopSigner)
		t.Logf("signed header: %d", i)
		require.NoError(t, svc.WriteToStoreAndBroadcast(ctx, &types.P2PSignedHeader{Message: signedHeader}))
	}

	// then stop and restart service
	_ = p2pClient.Close()
	_ = svc.Stop(ctx)
	cancel()

	h2, err := mn.AddPeer(priv, nil)
	require.NoError(t, err)
	p2pClient, err = p2p.NewClientWithHost(conf.P2P, nodeKey.PrivKey, mainKV, chainId, logger, p2p.NopMetrics(), h2)
	require.NoError(t, err)

	// Start p2p client again
	ctx, cancel = context.WithCancel(t.Context())
	defer cancel()
	err = p2pClient.Start(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = p2pClient.Close() })

	svc, err = NewHeaderSyncService(rktStore, conf, genesisDoc, p2pClient, logger)
	require.NoError(t, err)
	err = svc.Start(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = svc.Stop(context.Background()) })
	// done with stop and restart service

	// broadcast another 2 example blocks
	for i := signedHeader.Height() + 1; i < 2; i++ {
		signedHeader = nextHeader(t, signedHeader, genesisDoc.ChainID, noopSigner)
		t.Logf("signed header: %d", i)
		require.NoError(t, svc.WriteToStoreAndBroadcast(ctx, &types.P2PSignedHeader{Message: signedHeader}))
	}
	cancel()
}

func TestHeaderSyncServiceInitFromHigherHeight(t *testing.T) {
	mainKV := sync.MutexWrap(datastore.NewMapDatastore())
	pk, _, err := crypto.GenerateEd25519Key(cryptoRand.Reader)
	require.NoError(t, err)
	noopSigner, err := noop.NewNoopSigner(pk)
	require.NoError(t, err)
	rnd := rand.New(rand.NewSource(1)) // nolint:gosec // test code only
	mn := mocknet.New()

	chainId := "test-chain-id"

	proposerAddr := []byte("test")
	genesisDoc := genesispkg.Genesis{
		ChainID:         chainId,
		StartTime:       time.Now(),
		InitialHeight:   1,
		ProposerAddress: proposerAddr,
	}
	conf := config.DefaultConfig()
	conf.RootDir = t.TempDir()
	nodeKey, err := key.LoadOrGenNodeKey(filepath.Dir(conf.ConfigPath()))
	require.NoError(t, err)
	logger := zerolog.Nop()
	priv := nodeKey.PrivKey
	h, err := mn.AddPeer(priv, nil)
	require.NoError(t, err)

	p2pClient, err := p2p.NewClientWithHost(conf.P2P, nodeKey.PrivKey, mainKV, chainId, logger, p2p.NopMetrics(), h)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	require.NoError(t, p2pClient.Start(ctx))
	t.Cleanup(func() { _ = p2pClient.Close() })

	rktStore := store.New(mainKV)
	svc, err := NewHeaderSyncService(rktStore, conf, genesisDoc, p2pClient, logger)
	require.NoError(t, err)
	require.NoError(t, svc.Start(ctx))
	t.Cleanup(func() { _ = svc.Stop(context.Background()) })

	headerConfig := types.HeaderConfig{
		Height:   genesisDoc.InitialHeight + 5,
		DataHash: bytesN(rnd, 32),
		AppHash:  bytesN(rnd, 32),
		Signer:   noopSigner,
	}
	signedHeader, err := types.GetRandomSignedHeaderCustom(&headerConfig, genesisDoc.ChainID)
	require.NoError(t, err)
	require.NoError(t, signedHeader.Validate())

	require.NoError(t, svc.WriteToStoreAndBroadcast(ctx, &types.P2PSignedHeader{Message: signedHeader}))
}

func TestDAHintStorageHeader(t *testing.T) {
	mainKV := sync.MutexWrap(datastore.NewMapDatastore())
	pk, _, err := crypto.GenerateEd25519Key(cryptoRand.Reader)
	require.NoError(t, err)
	noopSigner, err := noop.NewNoopSigner(pk)
	require.NoError(t, err)
	rnd := rand.New(rand.NewSource(1)) // nolint:gosec // test code only
	mn := mocknet.New()

	chainId := "test-chain-id"

	proposerAddr := []byte("test")
	genesisDoc := genesispkg.Genesis{
		ChainID:         chainId,
		StartTime:       time.Now(),
		InitialHeight:   1,
		ProposerAddress: proposerAddr,
	}
	conf := config.DefaultConfig()
	conf.RootDir = t.TempDir()
	nodeKey, err := key.LoadOrGenNodeKey(filepath.Dir(conf.ConfigPath()))
	require.NoError(t, err)
	logger := zerolog.Nop()
	priv := nodeKey.PrivKey
	p2pHost, err := mn.AddPeer(priv, nil)
	require.NoError(t, err)

	p2pClient, err := p2p.NewClientWithHost(conf.P2P, nodeKey.PrivKey, mainKV, chainId, logger, p2p.NopMetrics(), p2pHost)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	require.NoError(t, p2pClient.Start(ctx))

	headerSvc, err := NewHeaderSyncService(mainKV, nil, conf, genesisDoc, p2pClient, logger)
	require.NoError(t, err)
	require.NoError(t, headerSvc.Start(ctx))

	headerConfig := types.HeaderConfig{
		Height:   genesisDoc.InitialHeight,
		DataHash: bytesN(rnd, 32),
		AppHash:  bytesN(rnd, 32),
		Signer:   noopSigner,
	}
	signedHeader, err := types.GetRandomSignedHeaderCustom(&headerConfig, genesisDoc.ChainID)
	require.NoError(t, err)
	require.NoError(t, signedHeader.Validate())

	require.NoError(t, headerSvc.WriteToStoreAndBroadcast(ctx, &types.P2PSignedHeader{Message: signedHeader}))

	daHeight := uint64(100)
	require.NoError(t, headerSvc.AppendDAHint(ctx, daHeight, signedHeader.Hash()))

	h, hint, err := headerSvc.GetByHeight(ctx, signedHeader.Height())
	require.NoError(t, err)
	require.Equal(t, signedHeader.Hash(), h.Hash())
	require.Equal(t, daHeight, hint)

	_ = p2pClient.Close()
	_ = headerSvc.Stop(ctx)
	cancel()

	// Restart
	h2, err := mn.AddPeer(priv, nil)
	require.NoError(t, err)
	p2pClient, err = p2p.NewClientWithHost(conf.P2P, nodeKey.PrivKey, mainKV, chainId, logger, p2p.NopMetrics(), h2)
	require.NoError(t, err)

	ctx, cancel = context.WithCancel(t.Context())
	defer cancel()
	require.NoError(t, p2pClient.Start(ctx))
	t.Cleanup(func() { _ = p2pClient.Close() })

	headerSvc, err = NewHeaderSyncService(mainKV, nil, conf, genesisDoc, p2pClient, logger)
	require.NoError(t, err)
	require.NoError(t, headerSvc.Start(ctx))
	t.Cleanup(func() { _ = headerSvc.Stop(context.Background()) })

	h, hint, err = headerSvc.GetByHeight(ctx, signedHeader.Height())
	require.NoError(t, err)
	require.Equal(t, signedHeader.Hash(), h.Hash())
	require.Equal(t, daHeight, hint)
}

func TestDAHintStorageData(t *testing.T) {
	mainKV := sync.MutexWrap(datastore.NewMapDatastore())
	pk, _, err := crypto.GenerateEd25519Key(cryptoRand.Reader)
	require.NoError(t, err)
	noopSigner, err := noop.NewNoopSigner(pk)
	require.NoError(t, err)
	rnd := rand.New(rand.NewSource(1)) // nolint:gosec // test code only
	mn := mocknet.New()

	chainId := "test-chain-id"

	proposerAddr := []byte("test")
	genesisDoc := genesispkg.Genesis{
		ChainID:         chainId,
		StartTime:       time.Now(),
		InitialHeight:   1,
		ProposerAddress: proposerAddr,
	}
	conf := config.DefaultConfig()
	conf.RootDir = t.TempDir()
	nodeKey, err := key.LoadOrGenNodeKey(filepath.Dir(conf.ConfigPath()))
	require.NoError(t, err)
	logger := zerolog.Nop()
	priv := nodeKey.PrivKey
	p2pHost, err := mn.AddPeer(priv, nil)
	require.NoError(t, err)

	p2pClient, err := p2p.NewClientWithHost(conf.P2P, nodeKey.PrivKey, mainKV, chainId, logger, p2p.NopMetrics(), p2pHost)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	require.NoError(t, p2pClient.Start(ctx))

	dataSvc, err := NewDataSyncService(mainKV, nil, conf, genesisDoc, p2pClient, logger)
	require.NoError(t, err)
	require.NoError(t, dataSvc.Start(ctx))

	// Need a valid header height for data metadata
	headerConfig := types.HeaderConfig{
		Height:   genesisDoc.InitialHeight,
		DataHash: bytesN(rnd, 32),
		AppHash:  bytesN(rnd, 32),
		Signer:   noopSigner,
	}
	signedHeader, err := types.GetRandomSignedHeaderCustom(&headerConfig, genesisDoc.ChainID)
	require.NoError(t, err)

	data := types.Data{
		Txs: types.Txs{[]byte("tx1")},
		Metadata: &types.Metadata{
			Height: signedHeader.Height(),
		},
	}

	require.NoError(t, dataSvc.WriteToStoreAndBroadcast(ctx, &types.P2PData{Message: &data}))

	daHeight := uint64(100)
	require.NoError(t, dataSvc.AppendDAHint(ctx, daHeight, data.Hash()))

	d, hint, err := dataSvc.GetByHeight(ctx, signedHeader.Height())
	require.NoError(t, err)
	require.Equal(t, data.Hash(), d.Hash())
	require.Equal(t, daHeight, hint)

	_ = p2pClient.Close()
	_ = dataSvc.Stop(ctx)
	cancel()

	// Restart
	h2, err := mn.AddPeer(priv, nil)
	require.NoError(t, err)
	p2pClient, err = p2p.NewClientWithHost(conf.P2P, nodeKey.PrivKey, mainKV, chainId, logger, p2p.NopMetrics(), h2)
	require.NoError(t, err)

	ctx, cancel = context.WithCancel(t.Context())
	defer cancel()
	require.NoError(t, p2pClient.Start(ctx))
	t.Cleanup(func() { _ = p2pClient.Close() })

	dataSvc, err = NewDataSyncService(mainKV, nil, conf, genesisDoc, p2pClient, logger)
	require.NoError(t, err)
	require.NoError(t, dataSvc.Start(ctx))
	t.Cleanup(func() { _ = dataSvc.Stop(context.Background()) })

	d, hint, err = dataSvc.GetByHeight(ctx, signedHeader.Height())
	require.NoError(t, err)
	require.Equal(t, data.Hash(), d.Hash())
	require.Equal(t, daHeight, hint)
}

func nextHeader(t *testing.T, previousHeader *types.SignedHeader, chainID string, noopSigner signer.Signer) *types.SignedHeader {
	newSignedHeader := &types.SignedHeader{
		Header: types.GetRandomNextHeader(previousHeader.Header, chainID),
		Signer: previousHeader.Signer,
	}
	b, err := newSignedHeader.Header.MarshalBinary()
	require.NoError(t, err)
	signature, err := noopSigner.Sign(b)
	require.NoError(t, err)
	newSignedHeader.Signature = signature
	require.NoError(t, newSignedHeader.Validate())
	return newSignedHeader
}

func bytesN(r *rand.Rand, n int) []byte {
	data := make([]byte, n)
	_, _ = r.Read(data)
	return data
}
