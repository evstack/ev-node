package sync

import (
	"context"
	cryptoRand "crypto/rand"
	"errors"
	"math/rand"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/celestiaorg/go-header"
	"github.com/evstack/ev-node/pkg/config"
	genesispkg "github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/p2p"
	"github.com/evstack/ev-node/pkg/p2p/key"
	"github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/signer/noop"
	"github.com/evstack/ev-node/types"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	"github.com/libp2p/go-libp2p/core/crypto"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/pkg/store"
)

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
	svc, err := NewHeaderSyncService(mainKV, rktStore, conf, genesisDoc, p2pClient, logger)
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
	require.NoError(t, svc.WriteToStoreAndBroadcast(ctx, signedHeader))

	for i := genesisDoc.InitialHeight + 1; i < 2; i++ {
		signedHeader = nextHeader(t, signedHeader, genesisDoc.ChainID, noopSigner)
		t.Logf("signed header: %d", i)
		require.NoError(t, svc.WriteToStoreAndBroadcast(ctx, signedHeader))
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

	svc, err = NewHeaderSyncService(mainKV, rktStore, conf, genesisDoc, p2pClient, logger)
	require.NoError(t, err)
	err = svc.Start(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = svc.Stop(context.Background()) })
	// done with stop and restart service

	// broadcast another 2 example blocks
	for i := signedHeader.Height() + 1; i < 2; i++ {
		signedHeader = nextHeader(t, signedHeader, genesisDoc.ChainID, noopSigner)
		t.Logf("signed header: %d", i)
		require.NoError(t, svc.WriteToStoreAndBroadcast(ctx, signedHeader))
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
	svc, err := NewHeaderSyncService(mainKV, rktStore, conf, genesisDoc, p2pClient, logger)
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

	require.NoError(t, svc.WriteToStoreAndBroadcast(ctx, signedHeader))
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
	previousHeader = newSignedHeader
	return previousHeader
}

func bytesN(r *rand.Rand, n int) []byte {
	data := make([]byte, n)
	_, _ = r.Read(data)
	return data
}

// TestBackgroundRetryEventuallySucceeds verifies that when the sync service cannot
// initially connect to peers, the background retry mechanism is triggered and
// eventually succeeds once headers become available from the DA store.
func TestBackgroundRetryEventuallySucceeds(t *testing.T) {
	mn := mocknet.New()
	defer mn.Close()

	pk, _, err := crypto.GenerateEd25519Key(cryptoRand.Reader)
	require.NoError(t, err)
	noopSigner, err := noop.NewNoopSigner(pk)
	require.NoError(t, err)
	rnd := rand.New(rand.NewSource(1)) // nolint:gosec // test code only

	chainId := "test-chain-id"
	proposerAddr := []byte("test")
	genesisDoc := genesispkg.Genesis{
		ChainID:         chainId,
		StartTime:       time.Now(),
		InitialHeight:   1,
		ProposerAddress: proposerAddr,
	}

	// Use a shared DA store that we can populate later
	mainKV := sync.MutexWrap(datastore.NewMapDatastore())
	rktStore := store.New(mainKV)

	conf := config.DefaultConfig()
	conf.RootDir = t.TempDir()
	nodeKey, err := key.LoadOrGenNodeKey(filepath.Dir(conf.ConfigPath()))
	require.NoError(t, err)
	logger := zerolog.Nop()

	h, err := mn.AddPeer(nodeKey.PrivKey, nil)
	require.NoError(t, err)

	p2pClient, err := p2p.NewClientWithHost(conf.P2P, nodeKey.PrivKey, mainKV, chainId, logger, p2p.NopMetrics(), h)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	require.NoError(t, p2pClient.Start(ctx))
	t.Cleanup(func() { _ = p2pClient.Close() })

	// Create the sync service - it has no peers to connect to, so it won't be able to sync via P2P
	// But it DOES have a DA store (rktStore) that we can populate
	svc, err := NewHeaderSyncService(mainKV, rktStore, conf, genesisDoc, p2pClient, logger)
	require.NoError(t, err)

	// Start the service - this will return without starting the syncer because there are no peers
	require.NoError(t, svc.Start(ctx))
	t.Cleanup(func() { _ = svc.Stop(context.Background()) })

	// Verify the syncer hasn't started yet (no peers to get headers from)
	require.False(t, svc.syncerStatus.isStarted(), "syncer should not be started without peers")

	// Verify that querying the header store before syncer starts returns an empty store error
	_, headErr := svc.Store().Head(ctx)
	require.Error(t, headErr, "querying head before syncer starts should return an error")
	require.True(t, errors.Is(headErr, header.ErrNotFound) || errors.Is(headErr, header.ErrEmptyStore),
		"error should be ErrNotFound or ErrEmptyStore, got: %v", headErr)

	// Create the genesis header that we'll add to the DA store
	headerConfig := types.HeaderConfig{
		Height:   genesisDoc.InitialHeight,
		DataHash: bytesN(rnd, 32),
		AppHash:  bytesN(rnd, 32),
		Signer:   noopSigner,
	}
	signedHeader, err := types.GetRandomSignedHeaderCustom(&headerConfig, genesisDoc.ChainID)
	require.NoError(t, err)
	require.NoError(t, signedHeader.Validate())

	// Track background retry completion
	var retryCompleted atomic.Bool

	// Manually trigger the background retry (simulating what happens after 2min timeout in initFromP2PWithRetry)
	go func() {
		svc.retryInitInBackground()
		retryCompleted.Store(true)
	}()

	// Give the retry a moment to start and fail at least once (no headers available yet)
	time.Sleep(100 * time.Millisecond)

	// Now add the header to the DA store - the background retry's next attempt should find it
	// via the exchangeWrapper's getterByHeight which checks the DA store first
	batch, err := rktStore.NewBatch(ctx)
	require.NoError(t, err)
	require.NoError(t, batch.SaveBlockData(signedHeader, &types.Data{}, &types.Signature{}))
	require.NoError(t, batch.SetHeight(signedHeader.Height()))
	require.NoError(t, batch.Commit())

	// Wait for the background retry to succeed and start the syncer
	require.Eventually(t, func() bool {
		return svc.syncerStatus.isStarted()
	}, 30*time.Second, 100*time.Millisecond, "syncer should eventually start after background retry finds header in DA store")

	// Verify the retry goroutine completed successfully
	require.Eventually(t, func() bool {
		return retryCompleted.Load()
	}, 5*time.Second, 100*time.Millisecond, "background retry goroutine should complete")

	// Verify the store was initialized with the header
	head, err := svc.Store().Head(ctx)
	require.NoError(t, err)
	require.Equal(t, genesisDoc.InitialHeight, head.Height())
}
