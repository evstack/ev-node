package syncing

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	gkmetrics "github.com/go-kit/kit/metrics"
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

// dsHarness wires a Syncer the way Start() does so tests can drive headers
// through processHeightEvent and assert on the halt pipeline. Equivocation is
// detected centrally in processHeightEvent by comparing an alternate header for
// an already-applied height against the canonical header in the store.
type dsHarness struct {
	t       *testing.T
	syncer  *Syncer
	store   store.Store
	cache   cache.CacheManager
	gen     genesis.Genesis
	addr    []byte
	pub     crypto.PubKey
	signer  signerpkg.Signer
	exec    *testmocks.MockExecutor
	errCh   chan error
	dsCount *atomic.Int64
}

func newDSHarness(t *testing.T, halt bool) *dsHarness {
	t.Helper()
	memDS := dssync.MutexWrap(ds.NewMapDatastore())
	st := store.New(memDS)

	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)

	addr, pub, signer := buildSyncTestSigner(t)
	cfg := config.DefaultConfig()
	cfg.Node.HaltOnDoubleSign = halt
	gen := genesis.Genesis{
		ChainID: "ds-test", InitialHeight: 1,
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
	s.ctx = context.Background()            // normally set up by Start()

	return &dsHarness{
		t: t, syncer: s, store: st, cache: cm, gen: gen,
		addr: addr, pub: pub, signer: signer, exec: mockExec, errCh: errCh, dsCount: &dsCount,
	}
}

// header builds a validly-signed header + matching data at height; variant
// differentiates the header hash via AppHash/LastHeaderHash.
func (h *dsHarness) header(height uint64, variant byte) (*types.SignedHeader, *types.Data) {
	h.t.Helper()
	data := makeData(h.gen.ChainID, height, 2)
	_, hdr := makeSignedHeaderBytes(
		h.t, h.gen.ChainID, height, h.addr, h.pub, h.signer,
		[]byte{variant, variant, variant}, data, []byte{variant},
	)
	return hdr, data
}

// headerOtherProposer builds a validly-signed header for a non-genesis proposer.
func (h *dsHarness) headerOtherProposer(height uint64, variant byte) (*types.SignedHeader, *types.Data) {
	h.t.Helper()
	otherAddr, otherPub, otherSigner := buildSyncTestSigner(h.t)
	data := makeData(h.gen.ChainID, height, 2)
	_, hdr := makeSignedHeaderBytes(
		h.t, h.gen.ChainID, height, otherAddr, otherPub, otherSigner,
		[]byte{variant, variant, variant}, data, []byte{variant},
	)
	return hdr, data
}

// persist writes the header to the store as the canonical block at its height.
func (h *dsHarness) persist(hdr *types.SignedHeader, data *types.Data) {
	h.t.Helper()
	batch, err := h.store.NewBatch(context.Background())
	require.NoError(h.t, err)
	require.NoError(h.t, batch.SaveBlockData(hdr, data, &hdr.Signature))
	require.NoError(h.t, batch.SetHeight(hdr.Height()))
	require.NoError(h.t, batch.Commit())
}

// feed drives a header/data pair through the real processHeightEvent entry point.
func (h *dsHarness) feed(hdr *types.SignedHeader, data *types.Data, source common.EventSource) {
	ev := common.DAHeightEvent{Header: hdr, Data: data, Source: source}
	h.syncer.processHeightEvent(context.Background(), &ev)
}

// newDARetriever builds a daRetriever wired to the harness syncer's in-batch reporter.
func (h *dsHarness) newDARetriever() *daRetriever {
	h.t.Helper()
	mockClient := testmocks.NewMockClient(h.t)
	mockClient.On("GetHeaderNamespace").Return([]byte("ns")).Maybe()
	mockClient.On("GetDataNamespace").Return([]byte("ns")).Maybe()
	return NewDARetriever(mockClient, h.cache, h.gen, zerolog.Nop(), h.syncer.reportInBatchDoubleSign)
}

// envelopeBlob builds a DA-envelope blob (sequencer-signed) for an empty-data header.
func (h *dsHarness) envelopeBlob(height uint64, variant byte) []byte {
	h.t.Helper()
	_, hdr := makeSignedHeaderBytes(
		h.t, h.gen.ChainID, height, h.addr, h.pub, h.signer,
		[]byte{variant, variant, variant}, nil, []byte{variant},
	)
	content, err := hdr.MarshalBinary()
	require.NoError(h.t, err)
	envSig, err := h.signer.Sign(h.t.Context(), content)
	require.NoError(h.t, err)
	blob, err := hdr.MarshalDAEnvelope(envSig)
	require.NoError(h.t, err)
	return blob
}

// applyNext builds a chain-valid block at currentHeight+1 and drives it through
// the real validate + apply pipeline (execution mocked), returning the applied
// header. This is how a canonical block becomes "finalized" the way production does.
func (h *dsHarness) applyNext(variant byte) *types.SignedHeader {
	h.t.Helper()
	st := h.syncer.getLastState()
	height := st.LastBlockHeight + 1
	data := makeData(h.gen.ChainID, height, 1)
	_, hdr := makeSignedHeaderBytes(
		h.t, h.gen.ChainID, height, h.addr, h.pub, h.signer,
		st.AppHash, data, st.LastHeaderHash,
	)
	h.exec.EXPECT().
		ExecuteTxs(mock.Anything, mock.Anything, height, mock.Anything, st.AppHash).
		Return([]byte{0xab, variant}, nil).Once()
	ev := common.DAHeightEvent{Header: hdr, Data: data, DaHeight: 10 + height, Source: common.SourceDA}
	h.syncer.processHeightEvent(context.Background(), &ev)
	return hdr
}

// envelopeBlobNonEmptyData builds a DA-envelope blob whose header expects data.
// With no matching data blob supplied, it stays in the retriever's in-flight set,
// which lets a later batch's conflicting header be compared against it.
func (h *dsHarness) envelopeBlobNonEmptyData(height uint64, variant byte) []byte {
	h.t.Helper()
	data := makeData(h.gen.ChainID, height, 2)
	_, hdr := makeSignedHeaderBytes(
		h.t, h.gen.ChainID, height, h.addr, h.pub, h.signer,
		[]byte{variant, variant, variant}, data, []byte{variant},
	)
	content, err := hdr.MarshalBinary()
	require.NoError(h.t, err)
	envSig, err := h.signer.Sign(h.t.Context(), content)
	require.NoError(h.t, err)
	blob, err := hdr.MarshalDAEnvelope(envSig)
	require.NoError(h.t, err)
	return blob
}

// legacyBlob builds a raw (non-envelope) header blob — the pre-upgrade DA format,
// which carries no envelope signature.
func (h *dsHarness) legacyBlob(height uint64, variant byte) []byte {
	h.t.Helper()
	bin, _ := makeSignedHeaderBytes(
		h.t, h.gen.ChainID, height, h.addr, h.pub, h.signer,
		[]byte{variant, variant, variant}, nil, []byte{variant},
	)
	return bin
}

func (h *dsHarness) requireHalted() {
	h.t.Helper()
	select {
	case got := <-h.errCh:
		require.ErrorIs(h.t, got, errMaliciousProposer)
	case <-time.After(time.Second):
		h.t.Fatal("timed out waiting for critical error on errCh")
	}
	require.True(h.t, h.syncer.hasCriticalError.Load())
	require.Equal(h.t, int64(1), h.dsCount.Load())
}

func (h *dsHarness) requireNoHalt() {
	h.t.Helper()
	require.False(h.t, h.syncer.hasCriticalError.Load())
	require.Equal(h.t, int64(0), h.dsCount.Load())
	require.Empty(h.t, h.errCh)
}

// A validly-signed alternate at an already-applied height halts the syncer.
func TestDoubleSign_AlternateAtAppliedHeight_Halts(t *testing.T) {
	h := newDSHarness(t, true)

	canonical, cdata := h.header(5, 0x01)
	h.persist(canonical, cdata)

	alt, adata := h.header(5, 0x02)
	require.NotEqual(t, canonical.Hash().String(), alt.Hash().String())

	h.feed(alt, adata, common.SourceDA)
	h.requireHalted()
}

// Detection is source-independent: a P2P-sourced alternate halts the same way.
func TestDoubleSign_CrossSource_P2PAlternate_Halts(t *testing.T) {
	h := newDSHarness(t, true)

	canonical, cdata := h.header(7, 0x01)
	h.persist(canonical, cdata)

	alt, adata := h.header(7, 0x02)
	h.feed(alt, adata, common.SourceP2P)
	h.requireHalted()
}

// Re-observing the canonical header (identical hash) must not halt.
func TestDoubleSign_BenignDuplicate_DoesNotHalt(t *testing.T) {
	h := newDSHarness(t, true)

	canonical, cdata := h.header(5, 0x01)
	h.persist(canonical, cdata)

	h.feed(canonical, cdata, common.SourceDA)
	h.requireNoHalt()
}

// With halting disabled, equivocation is counted but the node continues.
func TestDoubleSign_HaltFlagOff_WarnsButContinues(t *testing.T) {
	h := newDSHarness(t, false)

	canonical, cdata := h.header(5, 0x01)
	h.persist(canonical, cdata)

	alt, adata := h.header(5, 0x02)
	h.feed(alt, adata, common.SourceDA)

	require.Equal(t, int64(1), h.dsCount.Load())
	require.False(t, h.syncer.hasCriticalError.Load())
	require.Empty(t, h.errCh)
}

// The same (height, alternate-hash) seen twice is reported only once.
func TestDoubleSign_DuplicateAlternate_ReportedOnce(t *testing.T) {
	h := newDSHarness(t, true)

	canonical, cdata := h.header(11, 0x01)
	h.persist(canonical, cdata)

	alt, adata := h.header(11, 0x02)
	h.feed(alt, adata, common.SourceDA)
	h.feed(alt, adata, common.SourceP2P)

	require.Equal(t, int64(1), h.dsCount.Load())

	count := 0
	timeout := time.After(100 * time.Millisecond)
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

// A conflicting header from a different proposer is not sequencer equivocation.
func TestDoubleSign_ProposerMismatch_NotEvidence(t *testing.T) {
	h := newDSHarness(t, true)

	canonical, cdata := h.header(5, 0x01)
	h.persist(canonical, cdata)

	alt, adata := h.headerOtherProposer(5, 0x02)
	require.NotEqual(t, canonical.Hash().String(), alt.Hash().String())

	h.feed(alt, adata, common.SourceDA)
	h.requireNoHalt()
}

// A forged alternate (genesis proposer address but invalid signature) must not
// halt the node — guards against a forged-header denial of service.
func TestDoubleSign_ForgedAlternate_NotEvidence(t *testing.T) {
	h := newDSHarness(t, true)

	canonical, cdata := h.header(5, 0x01)
	h.persist(canonical, cdata)

	alt, adata := h.header(5, 0x02)
	require.NotEmpty(t, alt.Signature)
	alt.Signature[0] ^= 0xFF // corrupt the signature, leaving the header hash intact

	h.feed(alt, adata, common.SourceDA)
	h.requireNoHalt()
}

// Two distinct, envelope-signed headers for the same height in one DA batch are
// detected by the retriever before either is applied, and halt the syncer.
func TestDoubleSign_InBatchDA_EnvelopeAuthored_Halts(t *testing.T) {
	h := newDSHarness(t, true)
	r := h.newDARetriever()

	first := h.envelopeBlob(5, 0x01)
	alt := h.envelopeBlob(5, 0x02)

	r.ProcessBlobs(context.Background(), [][]byte{first, alt}, 100)
	h.requireHalted()
}

// Identical bytes seen twice in one DA batch are a benign duplicate, not equivocation.
func TestDoubleSign_InBatchDA_BenignDuplicate_DoesNotHalt(t *testing.T) {
	h := newDSHarness(t, true)
	r := h.newDARetriever()

	blob := h.envelopeBlob(5, 0x01)
	r.ProcessBlobs(context.Background(), [][]byte{blob, blob}, 100)

	require.False(t, h.syncer.hasCriticalError.Load())
	require.Equal(t, int64(0), h.dsCount.Load())
	require.Empty(t, h.errCh)
}

// Tip race: when two conflicting headers target the next height, the first is
// validated and applied through the real pipeline; the second then arrives at the
// now-finalized height and is caught. (processHeightEvent is single-threaded, so
// this models how the race resolves.)
func TestDoubleSign_TipRace_FirstAppliesThenSecondCaught(t *testing.T) {
	h := newDSHarness(t, true)

	canonical := h.applyNext(0x01) // validated + applied at height 1 via the real pipeline
	require.Equal(t, uint64(1), h.syncer.getLastState().LastBlockHeight)

	alt, adata := h.header(1, 0x02)
	require.NotEqual(t, canonical.Hash().String(), alt.Hash().String())

	h.feed(alt, adata, common.SourceP2P)
	h.requireHalted()
}

// Two conflicting headers arriving in SEPARATE DA batches, both before either is
// applied (kept in flight via non-empty data), are still caught in-batch.
func TestDoubleSign_InBatchDA_CrossBatchInFlight_Halts(t *testing.T) {
	h := newDSHarness(t, true)
	r := h.newDARetriever()

	r.ProcessBlobs(context.Background(), [][]byte{h.envelopeBlobNonEmptyData(5, 0x01)}, 100)
	r.ProcessBlobs(context.Background(), [][]byte{h.envelopeBlobNonEmptyData(5, 0x02)}, 101)

	h.requireHalted()
}

// Characterization of the Cut-2 gap: a conflicting header gossiped over P2P for a
// height we have already finalized is short-circuited by the P2P handler and never
// reaches detection — the header store is not even queried. If this changes, this
// test should fail and the gap (and its docs) revisited.
func TestDoubleSign_P2PReplayAtFinalizedHeight_NotDetected(t *testing.T) {
	addr, _, _ := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "ds-test", InitialHeight: 1, ProposerAddress: addr}
	memDS := dssync.MutexWrap(ds.NewMapDatastore())
	cm, err := cache.NewManager(config.DefaultConfig(), store.New(memDS), zerolog.Nop())
	require.NoError(t, err)

	// strict mocks: any GetByHeight call would fail the test
	headerStore := extmocks.NewMockStore[*types.P2PSignedHeader](t)
	dataStore := extmocks.NewMockStore[*types.P2PData](t)
	handler := NewP2PHandler(headerStore, dataStore, cm, gen, zerolog.Nop())
	handler.SetProcessedHeight(5) // height 5 already finalized

	ch := make(chan common.DAHeightEvent, 1)
	require.NoError(t, handler.ProcessHeight(context.Background(), 5, ch))
	require.Empty(t, ch) // no event emitted; header never fetched → conflict not surfaced
}

// Characterization of the legacy gap: two conflicting non-envelope (legacy) headers
// in one batch are NOT detected, because the early envelope-signature check is
// unavailable in non-strict mode. (Such a conflict is only caught later via
// checkDoubleSign if it reappears on DA after one header is applied.)
func TestDoubleSign_InBatchDA_LegacyNotDetected(t *testing.T) {
	h := newDSHarness(t, true)
	r := h.newDARetriever()

	events := r.ProcessBlobs(context.Background(), [][]byte{h.legacyBlob(5, 0x01), h.legacyBlob(5, 0x02)}, 100)

	// Non-vacuous: the legacy headers DID decode (one survives first-write-wins and
	// is emitted), proving we exercised the in-batch path — it just didn't detect.
	require.Len(t, events, 1)
	h.requireNoHalt()
}

func TestDoubleSignDedup(t *testing.T) {
	d := newDoubleSignDedup()
	require.True(t, d.markSeen(5, "a"))
	require.False(t, d.markSeen(5, "a"))
	require.True(t, d.markSeen(5, "b"))
	require.True(t, d.markSeen(6, "a"))
	require.False(t, d.markSeen(6, "a"))
}

// go-kit Counter backed by an atomic int64 so tests can read exact increments.
type counterCtr struct {
	n *atomic.Int64
}

func (c *counterCtr) Add(delta float64)                            { c.n.Add(int64(delta)) }
func (c *counterCtr) With(labelValues ...string) gkmetrics.Counter { return c }
