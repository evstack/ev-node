package syncing

import (
	"bytes"
	"context"
	"sync/atomic"
	"testing"
	"time"

	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	signerpkg "github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/store"
	testmocks "github.com/evstack/ev-node/test/mocks"
	extmocks "github.com/evstack/ev-node/test/mocks/external"
	"github.com/evstack/ev-node/types"
)

// integrationHarness wires the Syncer the way Start() does for tests
// to drive headers through the real entry points and assert on the halt pipeline.
type integrationHarness struct {
	t       *testing.T
	syncer  *Syncer
	store   store.Store
	cache   cache.CacheManager
	gen     genesis.Genesis
	addr    []byte
	pub     crypto.PubKey
	signer  signerpkg.Signer
	errCh   chan error
	dsCount *atomic.Int64
}

func newIntegrationHarness(t *testing.T) *integrationHarness {
	t.Helper()
	memDS := dssync.MutexWrap(ds.NewMapDatastore())
	st := store.New(memDS)

	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)

	addr, pub, signer := buildSyncTestSigner(t)
	cfg := config.DefaultConfig()
	gen := genesis.Genesis{
		ChainID: "syncer-ds-integration", InitialHeight: 1,
		StartTime: time.Now().Add(-time.Second), ProposerAddress: addr,
	}

	mockExec := testmocks.NewMockExecutor(t)
	mockExec.EXPECT().
		InitChain(mock.Anything, mock.Anything, uint64(1), gen.ChainID).
		Return([]byte("app0"), nil).Once()

	mockHeaderStore := extmocks.NewMockStore[*types.P2PSignedHeader](t)
	mockHeaderStore.EXPECT().Height().Return(uint64(0)).Maybe()
	mockDataStore := extmocks.NewMockStore[*types.P2PData](t)
	mockDataStore.EXPECT().Height().Return(uint64(0)).Maybe()

	metrics := common.NopMetrics()
	var dsCount atomic.Int64
	metrics.DoubleSignsDetected = &counterCtr{n: &dsCount}

	errCh := make(chan error, 4)
	s := NewSyncer(
		st, mockExec, nil, cm, metrics, cfg, gen,
		mockHeaderStore, mockDataStore, zerolog.Nop(),
		common.DefaultBlockOptions(), errCh, nil,
	)
	require.NoError(t, s.initializeState())
	s.doubleSignSeen = newDoubleSignDedup() // normally set up by Start()

	return &integrationHarness{
		t:       t,
		syncer:  s,
		store:   st,
		cache:   cm,
		gen:     gen,
		addr:    addr,
		pub:     pub,
		signer:  signer,
		errCh:   errCh,
		dsCount: &dsCount,
	}
}

func (h *integrationHarness) sign(height uint64, variant byte) *types.SignedHeader {
	h.t.Helper()
	// Vary AppHash AND LastHeaderHash by variant so distinct variants always
	// produce distinct hashes regardless of timing or pool state.
	_, hdr := makeSignedHeaderBytes(h.t, h.gen.ChainID, height, h.addr, h.pub, h.signer,
		[]byte{variant, variant, variant}, nil, []byte{variant})
	return hdr
}

// persistHeader writes the header to the store as if it had been synced.
func (h *integrationHarness) persistHeader(hdr *types.SignedHeader) {
	h.t.Helper()
	batch, err := h.store.NewBatch(context.Background())
	require.NoError(h.t, err)
	require.NoError(h.t, batch.SaveBlockData(hdr, &types.Data{
		Metadata: &types.Metadata{ChainID: h.gen.ChainID, Height: hdr.Height(), Time: hdr.BaseHeader.Time},
	}, &hdr.Signature))
	require.NoError(h.t, batch.SetHeight(hdr.Height()))
	require.NoError(h.t, batch.Commit())
}

// newDARetriever builds a daRetriever wired to the harness's syncer.
func (h *integrationHarness) newDARetriever() *daRetriever {
	mockClient := testmocks.NewMockClient(h.t)
	mockClient.On("GetHeaderNamespace").Return([]byte("ns")).Maybe()
	mockClient.On("GetDataNamespace").Return([]byte("ns")).Maybe()
	return NewDARetriever(mockClient, h.cache, h.gen, zerolog.Nop(), h.syncer.detectDoubleSign)
}

// newP2PHandler builds a P2PHandler with the header store mocked to return hdr at the given height.
func (h *integrationHarness) newP2PHandler(height uint64, hdr *types.SignedHeader) (*P2PHandler, chan common.DAHeightEvent) {
	headerStore := extmocks.NewMockStore[*types.P2PSignedHeader](h.t)
	dataStore := extmocks.NewMockStore[*types.P2PData](h.t)
	headerStore.EXPECT().
		GetByHeight(mock.Anything, height).
		Return(&types.P2PSignedHeader{SignedHeader: hdr}, nil).
		Once()
	p2p := NewP2PHandler(headerStore, dataStore, h.cache, h.gen, zerolog.Nop(), h.syncer.detectDoubleSign)
	return p2p, make(chan common.DAHeightEvent, 1)
}

// requireHalted asserts the full halt pipeline fired exactly once.
func (h *integrationHarness) requireHalted(altHash []byte, height uint64) {
	h.t.Helper()
	select {
	case got := <-h.errCh:
		require.ErrorIs(h.t, got, ErrDoubleSign)
	case <-time.After(time.Second):
		h.t.Fatal("timed out waiting for critical error on errCh")
	}
	require.True(h.t, h.syncer.hasCriticalError.Load())
	require.Equal(h.t, int64(1), h.dsCount.Load())

	blob, err := h.store.GetMetadata(context.Background(), store.GetDoubleSignEvidenceKey(height, altHash))
	require.NoError(h.t, err)
	require.NotEmpty(h.t, blob)
}

// Two conflicting headers in the same DA blob batch must halt the syncer.
func TestSyncerIntegration_InBatchDA_HaltsViaRealRetrieverPipeline(t *testing.T) {
	h := newIntegrationHarness(t)
	r := h.newDARetriever()

	first := h.sign(5, 0x01)
	alt := h.sign(5, 0x02)
	require.NotEqual(t, first.Hash().String(), alt.Hash().String())

	firstBin, err := first.MarshalBinary()
	require.NoError(t, err)
	altBin, err := alt.MarshalBinary()
	require.NoError(t, err)

	events := r.ProcessBlobs(context.Background(), [][]byte{firstBin, altBin}, 100)
	require.Empty(t, events)

	h.requireHalted(alt.Hash(), 5)
}

// An alternate in a later DA batch is detected against the persisted canonical.
func TestSyncerIntegration_CrossBatchDA_HaltsViaStoreLookup(t *testing.T) {
	h := newIntegrationHarness(t)
	r := h.newDARetriever()

	first := h.sign(5, 0x01)
	h.persistHeader(first)

	alt := h.sign(5, 0x02)
	altBin, err := alt.MarshalBinary()
	require.NoError(t, err)

	events := r.ProcessBlobs(context.Background(), [][]byte{altBin}, 101)
	require.Empty(t, events)

	h.requireHalted(alt.Hash(), 5)
}

// An alternate via P2P must halt when the canonical was first observed via DA.
func TestSyncerIntegration_CrossSource_DAFirstThenP2P_Halts(t *testing.T) {
	h := newIntegrationHarness(t)
	r := h.newDARetriever()

	first := h.sign(7, 0x01)
	firstBin, err := first.MarshalBinary()
	require.NoError(t, err)
	_ = r.ProcessBlobs(context.Background(), [][]byte{firstBin}, 100)

	cached, source, ok := h.cache.GetPendingSignedHeader(7)
	require.True(t, ok)
	require.True(t, bytes.Equal(cached.Hash(), first.Hash()))
	require.Equal(t, types.EvidenceSourceDA, source)

	alt := h.sign(7, 0x02)
	p2p, ch := h.newP2PHandler(7, alt)
	require.NoError(t, p2p.ProcessHeight(context.Background(), 7, ch))
	require.Empty(t, ch)

	h.requireHalted(alt.Hash(), 7)
}

// An alternate via DA must halt when the canonical was first observed via P2P.
func TestSyncerIntegration_CrossSource_P2PFirstThenDA_Halts(t *testing.T) {
	h := newIntegrationHarness(t)

	first := h.sign(7, 0x01)
	p2p, ch := h.newP2PHandler(7, first)
	p2p.SetProcessedHeight(7)
	require.NoError(t, p2p.ProcessHeight(context.Background(), 7, ch))
	require.Empty(t, ch)

	cached, source, ok := h.cache.GetPendingSignedHeader(7)
	require.True(t, ok)
	require.True(t, bytes.Equal(cached.Hash(), first.Hash()))
	require.Equal(t, types.EvidenceSourceP2P, source)

	alt := h.sign(7, 0x02)
	altBin, err := alt.MarshalBinary()
	require.NoError(t, err)

	r := h.newDARetriever()
	events := r.ProcessBlobs(context.Background(), [][]byte{altBin}, 100)
	require.Empty(t, events)

	h.requireHalted(alt.Hash(), 7)
}

// An alternate via P2P must halt when the canonical was already persisted.
func TestSyncerIntegration_CrossSource_StoreFirstThenP2P_Halts(t *testing.T) {
	h := newIntegrationHarness(t)

	first := h.sign(5, 0x01)
	h.persistHeader(first)

	alt := h.sign(5, 0x02)
	p2p, ch := h.newP2PHandler(5, alt)
	require.NoError(t, p2p.ProcessHeight(context.Background(), 5, ch))
	require.Empty(t, ch)

	h.requireHalted(alt.Hash(), 5)
}

// Identical bytes seen twice in the same DA batch must not halt.
func TestSyncerIntegration_BenignDuplicate_InBatch_DoesNotHalt(t *testing.T) {
	h := newIntegrationHarness(t)
	r := h.newDARetriever()

	hdr := h.sign(5, 0x01)
	bin, err := hdr.MarshalBinary()
	require.NoError(t, err)

	_ = r.ProcessBlobs(context.Background(), [][]byte{bin, bin}, 100)

	require.False(t, h.syncer.hasCriticalError.Load())
	require.Equal(t, int64(0), h.dsCount.Load())
	require.Empty(t, h.errCh)
}

// Re-publishing the canonical at a different DA height must not halt.
func TestSyncerIntegration_BenignDuplicate_AcrossBatches_DoesNotHalt(t *testing.T) {
	h := newIntegrationHarness(t)
	r := h.newDARetriever()

	first := h.sign(5, 0x01)
	h.persistHeader(first)

	bin, err := first.MarshalBinary()
	require.NoError(t, err)
	_ = r.ProcessBlobs(context.Background(), [][]byte{bin}, 101)

	require.False(t, h.syncer.hasCriticalError.Load())
	require.Equal(t, int64(0), h.dsCount.Load())
	require.Empty(t, h.errCh)
}

// The same (height, altHash) seen twice must report only once.
func TestSyncerIntegration_DuplicateAlternates_DedupedToOneHalt(t *testing.T) {
	h := newIntegrationHarness(t)
	r := h.newDARetriever()

	first := h.sign(11, 0x01)
	alt := h.sign(11, 0x02)
	firstBin, err := first.MarshalBinary()
	require.NoError(t, err)
	altBin, err := alt.MarshalBinary()
	require.NoError(t, err)

	_ = r.ProcessBlobs(context.Background(), [][]byte{firstBin, altBin}, 100)
	_ = r.ProcessBlobs(context.Background(), [][]byte{altBin}, 101)

	require.Equal(t, int64(1), h.dsCount.Load())

	timeout := time.After(50 * time.Millisecond)
	count := 0
loop:
	for {
		select {
		case <-h.errCh:
			count++
		case <-timeout:
			break loop
		}
	}
	require.Equal(t, 1, count)
}
