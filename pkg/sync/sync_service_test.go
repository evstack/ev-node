package sync

import (
	"context"
	cryptoRand "crypto/rand"
	"math/rand"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	goheader "github.com/celestiaorg/go-header"
	"github.com/celestiaorg/go-header/headertest"
	goheaderlocal "github.com/celestiaorg/go-header/local"
	goheadersync "github.com/celestiaorg/go-header/sync"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/net/conngater"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"github.com/multiformats/go-multiaddr"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/pkg/config"
	genesispkg "github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/p2p"
	"github.com/evstack/ev-node/pkg/p2p/key"
	"github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/signer/noop"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

type countingP2PDataGetter struct {
	goheader.Getter[*types.P2PData]
	getByHeightCalls atomic.Uint64
	rangeCalls       atomic.Uint64
}

func (g *countingP2PDataGetter) GetByHeight(ctx context.Context, height uint64) (*types.P2PData, error) {
	g.getByHeightCalls.Add(1)
	return g.Getter.GetByHeight(ctx, height)
}

func (g *countingP2PDataGetter) GetRangeByHeight(
	ctx context.Context,
	from *types.P2PData,
	to uint64,
) ([]*types.P2PData, error) {
	g.rangeCalls.Add(1)
	return g.Getter.GetRangeByHeight(ctx, from, to)
}

type signalingP2PDataStore struct {
	goheader.Store[*types.P2PData]
	targetHeight uint64
	syncComplete chan struct{}
	signaled     atomic.Bool
}

func (s *signalingP2PDataStore) Append(ctx context.Context, data ...*types.P2PData) error {
	if err := s.Store.Append(ctx, data...); err != nil {
		return err
	}

	for _, item := range data {
		if item.Height() == s.targetHeight && s.signaled.CompareAndSwap(false, true) {
			close(s.syncComplete)
			break
		}
	}

	return nil
}

type verifierCapturingP2PDataSubscriber struct {
	*headertest.Subscriber[*types.P2PData]
	verifier func(context.Context, *types.P2PData) error
}

type peerListP2PClient struct {
	peerIDs []peer.ID
}

func (c peerListP2PClient) PubSub() *pubsub.PubSub                           { return nil }
func (c peerListP2PClient) Info() (string, string, string, error)            { return "", "", "", nil }
func (c peerListP2PClient) Host() host.Host                                  { return nil }
func (c peerListP2PClient) ConnectionGater() *conngater.BasicConnectionGater { return nil }
func (c peerListP2PClient) PeerIDs() []peer.ID                               { return c.peerIDs }

func TestCatchupAggregatorIncludesConfiguredPeerIDs(t *testing.T) {
	configuredKey, _, err := crypto.GenerateEd25519Key(cryptoRand.Reader)
	require.NoError(t, err)
	configuredID, err := peer.IDFromPrivateKey(configuredKey)
	require.NoError(t, err)

	conf := config.DefaultConfig()
	conf.Node.Aggregator = true
	conf.Node.CatchupTimeout = config.DurationWrapper{Duration: time.Second}
	conf.P2P.Peers = "/ip4/127.0.0.1/tcp/7676/p2p/" + configuredID.String()

	svc := &SyncService[*types.P2PData]{
		conf:   conf,
		p2p:    peerListP2PClient{},
		logger: zerolog.Nop(),
	}
	require.Equal(t, []peer.ID{configuredID}, svc.getPeerIDs())
}

func TestKeepRetryingP2PInit(t *testing.T) {
	configuredKey, _, err := crypto.GenerateEd25519Key(cryptoRand.Reader)
	require.NoError(t, err)
	configuredID, err := peer.IDFromPrivateKey(configuredKey)
	require.NoError(t, err)
	peerAddr := "/ip4/127.0.0.1/tcp/7676/p2p/" + configuredID.String()

	tests := []struct {
		name   string
		mutate func(*config.Config)
		want   bool
	}{
		{
			name: "catchup aggregator with peers",
			mutate: func(conf *config.Config) {
				conf.Node.Aggregator = true
				conf.Node.CatchupTimeout = config.DurationWrapper{Duration: time.Minute}
				conf.P2P.Peers = peerAddr
			},
			want: true,
		},
		{
			name: "catchup disabled",
			mutate: func(conf *config.Config) {
				conf.Node.Aggregator = true
				conf.P2P.Peers = peerAddr
			},
		},
		{
			name: "no configured peers",
			mutate: func(conf *config.Config) {
				conf.Node.Aggregator = true
				conf.Node.CatchupTimeout = config.DurationWrapper{Duration: time.Minute}
			},
		},
		{
			name: "full node with catchup config",
			mutate: func(conf *config.Config) {
				conf.Node.CatchupTimeout = config.DurationWrapper{Duration: time.Minute}
				conf.P2P.Peers = peerAddr
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := config.DefaultConfig()
			tt.mutate(&conf)
			svc := &SyncService[*types.P2PData]{conf: conf}
			require.Equal(t, tt.want, svc.keepRetryingP2PInit())
		})
	}
}

func TestContinueP2PInitIfNeededNoopsWithoutCatchupRequirement(t *testing.T) {
	svc := &SyncService[*types.P2PData]{
		conf:   config.DefaultConfig(),
		logger: zerolog.Nop(),
	}
	// Would panic in retryInitFromP2P if a goroutine were started with a nil store.
	svc.continueP2PInitIfNeeded(t.Context(), []peer.ID{"12D3KooWCatchupPeer"})
}

func (s *verifierCapturingP2PDataSubscriber) SetVerifier(
	verifier func(context.Context, *types.P2PData) error,
) error {
	s.verifier = verifier
	return nil
}

func TestDataSyncerDistantHeadUsesRangeSync(t *testing.T) {
	const targetHeight uint64 = 64

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	chain := make([]*types.P2PData, 0, targetHeight)
	blockTime := time.Now().Add(-time.Duration(targetHeight) * time.Millisecond)
	var previousHash types.Hash
	for height := uint64(1); height <= targetHeight; height++ {
		_, data := types.GetRandomBlock(height, 1, "data-sync-catchup")
		data.Metadata.Time = uint64(blockTime.Add(time.Duration(height) * time.Millisecond).UnixNano())
		data.LastDataHash = previousHash
		previousHash = data.Hash()
		chain = append(chain, &types.P2PData{Data: data})
	}

	remoteStore := &headertest.Store[*types.P2PData]{Headers: make(map[uint64]*types.P2PData)}
	require.NoError(t, remoteStore.Append(ctx, chain...))
	localKV := sync.MutexWrap(datastore.NewMapDatastore())
	localStore := &signalingP2PDataStore{
		Store: store.NewDataStoreAdapter(store.New(localKV), genesispkg.Genesis{
			ChainID:       "data-sync-catchup",
			InitialHeight: 1,
		}),
		targetHeight: targetHeight,
		syncComplete: make(chan struct{}),
	}
	require.NoError(t, localStore.Append(ctx, chain[0]))

	getter := &countingP2PDataGetter{Getter: goheaderlocal.NewExchange(remoteStore)}
	subscriber := &verifierCapturingP2PDataSubscriber{
		Subscriber: &headertest.Subscriber[*types.P2PData]{},
	}
	syncer, err := goheadersync.NewSyncer(
		getter,
		localStore,
		subscriber,
		goheadersync.WithBlockTime(time.Second),
	)
	require.NoError(t, err)
	require.NoError(t, syncer.Start(ctx))
	t.Cleanup(func() { _ = syncer.Stop(context.Background()) })

	getter.getByHeightCalls.Store(0)
	getter.rangeCalls.Store(0)
	require.NoError(t, subscriber.verifier(ctx, chain[targetHeight-1]))

	select {
	case <-localStore.syncComplete:
	case <-ctx.Done():
		require.NoError(t, ctx.Err(), "waiting for data synchronization to complete")
	}
	require.LessOrEqual(t, getter.getByHeightCalls.Load(), uint64(1),
		"validating a distant head must not fetch intermediate blocks one by one")
	require.Positive(t, getter.rangeCalls.Load())
}

func TestHeaderSyncServiceStartForPublishingWithPeers(t *testing.T) {
	mainKV := sync.MutexWrap(datastore.NewMapDatastore())
	pk, _, err := crypto.GenerateEd25519Key(cryptoRand.Reader)
	require.NoError(t, err)
	noopSigner, err := noop.NewNoopSigner(pk)
	require.NoError(t, err)
	rnd := rand.New(rand.NewSource(1)) // nolint:gosec // test code only
	mn := mocknet.New()

	chainID := "test-chain-id"
	genesisDoc := genesispkg.Genesis{
		ChainID:         chainID,
		StartTime:       time.Now(),
		InitialHeight:   1,
		ProposerAddress: []byte("test"),
	}

	conf := config.DefaultConfig()
	conf.RootDir = t.TempDir()
	logger := zerolog.Nop()

	nodeKey1, err := key.LoadOrGenNodeKey(filepath.Join(conf.RootDir, "node1_key.json"))
	require.NoError(t, err)
	host1, err := mn.AddPeer(nodeKey1.PrivKey, multiaddr.StringCast("/ip4/127.0.0.1/tcp/0"))
	require.NoError(t, err)

	nodeKey2, err := key.LoadOrGenNodeKey(filepath.Join(conf.RootDir, "node2_key.json"))
	require.NoError(t, err)
	host2, err := mn.AddPeer(nodeKey2.PrivKey, multiaddr.StringCast("/ip4/127.0.0.1/tcp/0"))
	require.NoError(t, err)

	require.NoError(t, mn.LinkAll())
	require.NoError(t, mn.ConnectAllButSelf())

	client1, err := p2p.NewClientWithHost(conf.P2P, nodeKey1.PrivKey, mainKV, chainID, logger, p2p.NopMetrics(), host1)
	require.NoError(t, err)
	client2, err := p2p.NewClientWithHost(conf.P2P, nodeKey2.PrivKey, mainKV, chainID, logger, p2p.NopMetrics(), host2)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	require.NoError(t, client1.Start(ctx))
	require.NoError(t, client2.Start(ctx))
	t.Cleanup(func() { _ = client1.Close() })
	t.Cleanup(func() { _ = client2.Close() })

	require.Eventually(t, func() bool {
		return len(client1.PeerIDs()) > 0
	}, time.Second, 10*time.Millisecond)

	evStore := store.New(mainKV)
	svc, err := NewHeaderSyncService(evStore, conf, genesisDoc, client1, logger)
	require.NoError(t, err)
	require.NoError(t, svc.StartForPublishing(ctx))
	t.Cleanup(func() { _ = svc.Stop(context.Background()) })

	headerConfig := types.HeaderConfig{
		Height:   genesisDoc.InitialHeight,
		DataHash: bytesN(rnd, 32),
		AppHash:  bytesN(rnd, 32),
		Signer:   noopSigner,
	}
	signedHeader, err := types.GetRandomSignedHeaderCustom(t.Context(), &headerConfig, genesisDoc.ChainID)
	require.NoError(t, err)
	require.NoError(t, signedHeader.Validate())

	require.NoError(t, svc.WriteToStoreAndBroadcast(ctx, &types.P2PSignedHeader{SignedHeader: signedHeader}))
	require.True(t, svc.storeInitialized.Load())
	require.False(t, svc.P2PInitialized(), "publishing must not count as P2P initialization")
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

	evStore := store.New(mainKV)
	svc, err := NewHeaderSyncService(evStore, conf, genesisDoc, p2pClient, logger)
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
	signedHeader, err := types.GetRandomSignedHeaderCustom(ctx, &headerConfig, genesisDoc.ChainID)
	require.NoError(t, err)
	require.NoError(t, signedHeader.Validate())
	require.NoError(t, svc.WriteToStoreAndBroadcast(ctx, &types.P2PSignedHeader{SignedHeader: signedHeader}))

	for i := genesisDoc.InitialHeight + 1; i < 10; i++ {
		signedHeader = nextHeader(t, signedHeader, genesisDoc.ChainID, noopSigner)
		t.Logf("signed header: %d", i)
		require.NoError(t, svc.WriteToStoreAndBroadcast(ctx, &types.P2PSignedHeader{SignedHeader: signedHeader}))
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

	svc, err = NewHeaderSyncService(evStore, conf, genesisDoc, p2pClient, logger)
	require.NoError(t, err)
	err = svc.Start(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = svc.Stop(context.Background()) })
	// done with stop and restart service

	// broadcast another 2 example blocks
	for i := signedHeader.Height() + 1; i < 2; i++ {
		signedHeader = nextHeader(t, signedHeader, genesisDoc.ChainID, noopSigner)
		t.Logf("signed header: %d", i)
		require.NoError(t, svc.WriteToStoreAndBroadcast(ctx, &types.P2PSignedHeader{SignedHeader: signedHeader}))
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

	evStore := store.New(mainKV)
	svc, err := NewHeaderSyncService(evStore, conf, genesisDoc, p2pClient, logger)
	require.NoError(t, err)
	require.NoError(t, svc.Start(ctx))
	t.Cleanup(func() { _ = svc.Stop(context.Background()) })

	headerConfig := types.HeaderConfig{
		Height:   genesisDoc.InitialHeight + 5,
		DataHash: bytesN(rnd, 32),
		AppHash:  bytesN(rnd, 32),
		Signer:   noopSigner,
	}
	signedHeader, err := types.GetRandomSignedHeaderCustom(ctx, &headerConfig, genesisDoc.ChainID)
	require.NoError(t, err)
	require.NoError(t, signedHeader.Validate())

	require.NoError(t, svc.WriteToStoreAndBroadcast(ctx, &types.P2PSignedHeader{SignedHeader: signedHeader}))
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

	evStore := store.New(mainKV)
	headerSvc, err := NewHeaderSyncService(evStore, conf, genesisDoc, p2pClient, logger)
	require.NoError(t, err)
	require.NoError(t, headerSvc.Start(ctx))

	headerConfig := types.HeaderConfig{
		Height:   genesisDoc.InitialHeight,
		DataHash: bytesN(rnd, 32),
		AppHash:  bytesN(rnd, 32),
		Signer:   noopSigner,
	}
	signedHeader, err := types.GetRandomSignedHeaderCustom(ctx, &headerConfig, genesisDoc.ChainID)
	require.NoError(t, err)
	require.NoError(t, signedHeader.Validate())

	require.NoError(t, headerSvc.WriteToStoreAndBroadcast(ctx, &types.P2PSignedHeader{SignedHeader: signedHeader}))

	daHeight := uint64(100)
	require.NoError(t, headerSvc.AppendDAHint(ctx, daHeight, signedHeader.Height()))

	h, err := headerSvc.Store().GetByHeight(ctx, signedHeader.Height())
	require.NoError(t, err)
	require.Equal(t, signedHeader.Hash(), h.Hash())
	require.Equal(t, daHeight, h.DAHint())

	// Persist header to underlying store so it survives restart
	// (WriteToStoreAndBroadcast only writes to P2P pending cache)
	data := &types.Data{Metadata: &types.Metadata{Height: signedHeader.Height()}}
	batch, err := evStore.NewBatch(ctx)
	require.NoError(t, err)
	require.NoError(t, batch.SaveBlockData(signedHeader, data, &signedHeader.Signature))
	require.NoError(t, batch.SetHeight(signedHeader.Height()))
	require.NoError(t, batch.Commit())

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

	headerSvc, err = NewHeaderSyncService(evStore, conf, genesisDoc, p2pClient, logger)
	require.NoError(t, err)
	require.NoError(t, headerSvc.Start(ctx))
	t.Cleanup(func() { _ = headerSvc.Stop(context.Background()) })

	h, err = headerSvc.Store().GetByHeight(ctx, signedHeader.Height())
	require.NoError(t, err)
	require.Equal(t, signedHeader.Hash(), h.Hash())
	require.Equal(t, daHeight, h.DAHint())
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

	evStore := store.New(mainKV)
	dataSvc, err := NewDataSyncService(evStore, conf, genesisDoc, p2pClient, logger)
	require.NoError(t, err)
	require.NoError(t, dataSvc.Start(ctx))

	// Need a valid header height for data metadata
	headerConfig := types.HeaderConfig{
		Height:   genesisDoc.InitialHeight,
		DataHash: bytesN(rnd, 32),
		AppHash:  bytesN(rnd, 32),
		Signer:   noopSigner,
	}
	signedHeader, err := types.GetRandomSignedHeaderCustom(ctx, &headerConfig, genesisDoc.ChainID)
	require.NoError(t, err)

	data := types.Data{
		Txs: types.Txs{[]byte("tx1")},
		Metadata: &types.Metadata{
			Height: signedHeader.Height(),
		},
	}

	require.NoError(t, dataSvc.WriteToStoreAndBroadcast(ctx, &types.P2PData{Data: &data}))

	daHeight := uint64(100)
	require.NoError(t, dataSvc.AppendDAHint(ctx, daHeight, data.Height()))

	d, err := dataSvc.Store().GetByHeight(ctx, signedHeader.Height())
	require.NoError(t, err)
	require.Equal(t, data.Hash(), d.Hash())
	require.Equal(t, daHeight, d.DAHint())

	// Persist data to underlying store so it survives restart
	// (WriteToStoreAndBroadcast only writes to P2P pending cache)
	batch, err := evStore.NewBatch(ctx)
	require.NoError(t, err)
	require.NoError(t, batch.SaveBlockData(signedHeader, &data, &signedHeader.Signature))
	require.NoError(t, batch.SetHeight(signedHeader.Height()))
	require.NoError(t, batch.Commit())

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

	dataSvc, err = NewDataSyncService(evStore, conf, genesisDoc, p2pClient, logger)
	require.NoError(t, err)
	require.NoError(t, dataSvc.Start(ctx))
	t.Cleanup(func() { _ = dataSvc.Stop(context.Background()) })

	d, err = dataSvc.Store().GetByHeight(ctx, signedHeader.Height())
	require.NoError(t, err)
	require.Equal(t, data.Hash(), d.Hash())
	require.Equal(t, daHeight, d.DAHint())
}

func nextHeader(t *testing.T, previousHeader *types.SignedHeader, chainID string, noopSigner signer.Signer) *types.SignedHeader {
	newSignedHeader := &types.SignedHeader{
		Header: types.GetRandomNextHeader(previousHeader.Header, chainID),
		Signer: previousHeader.Signer,
	}
	b, err := newSignedHeader.Header.MarshalBinary()
	require.NoError(t, err)
	signature, err := noopSigner.Sign(t.Context(), b)
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
