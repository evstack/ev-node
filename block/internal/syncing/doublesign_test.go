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
	"google.golang.org/protobuf/proto"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	signerpkg "github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/store"
	testmocks "github.com/evstack/ev-node/test/mocks"
	extmocks "github.com/evstack/ev-node/test/mocks/external"
	"github.com/evstack/ev-node/types"
	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

// dsTestEnv bundles the store, cache, genesis and signer used by the
// double-sign tests.
type dsTestEnv struct {
	t                *testing.T
	store            store.Store
	cache            cache.CacheManager
	gen              genesis.Genesis
	addr             []byte
	pub              crypto.PubKey
	signer           signerpkg.Signer
	chainID          string
	capLock          atomic.Pointer[[]*types.DoubleSignEvidence]
	detectDoubleSign doubleSignDetector
}

func newDSTestEnv(t *testing.T) *dsTestEnv {
	t.Helper()
	memDS := dssync.MutexWrap(ds.NewMapDatastore())
	st := store.New(memDS)
	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)

	addr, pub, signer := buildSyncTestSigner(t)
	gen := genesis.Genesis{
		ChainID:         "ds-test",
		InitialHeight:   1,
		StartTime:       time.Now().Add(-time.Second),
		ProposerAddress: addr,
	}

	env := &dsTestEnv{
		t:       t,
		store:   st,
		cache:   cm,
		gen:     gen,
		addr:    addr,
		pub:     pub,
		signer:  signer,
		chainID: gen.ChainID,
	}
	empty := []*types.DoubleSignEvidence{}
	env.capLock.Store(&empty)
	env.detectDoubleSign = env.makeDetectDoubleSign(st)
	return env
}

// makeDetectDoubleSign mimics Syncer.detectDoubleSign for tests, capturing
// evidence into env.capLock instead of halting.
func (e *dsTestEnv) makeDetectDoubleSign(st store.Store) doubleSignDetector {
	return func(ctx context.Context, header *types.SignedHeader, source string) bool {
		prior, priorSource, err := firstObservation(ctx, st, e.cache, header.Height())
		if err != nil {
			e.cache.SetPendingSignedHeader(header, source)
			return false
		}
		if ev := buildEvidenceFromPair(prior, header, priorSource, source); ev != nil {
			require.NoError(e.t, ev.ValidateBasic())
			for {
				cur := e.capLock.Load()
				next := append([]*types.DoubleSignEvidence(nil), *cur...)
				next = append(next, ev)
				if e.capLock.CompareAndSwap(cur, &next) {
					break
				}
			}
			require.NoError(e.t, persistEvidence(ctx, st, ev))
			return true
		}
		e.cache.SetPendingSignedHeader(header, source)
		return false
	}
}

func (e *dsTestEnv) captured() []*types.DoubleSignEvidence {
	return *e.capLock.Load()
}

// signs a header at height by the genesis proposer; variant differentiates hashes.
func (e *dsTestEnv) signHeaderAtHeight(height uint64, variant byte) *types.SignedHeader {
	e.t.Helper()
	_, hdr := makeSignedHeaderBytes(
		e.t, e.chainID, height, e.addr, e.pub, e.signer,
		[]byte{variant, variant, variant},
		nil,
		nil,
	)
	return hdr
}

// signs a header by a fresh (non-genesis) signer.
func (e *dsTestEnv) signHeaderWithOtherProposer(height uint64, variant byte) *types.SignedHeader {
	e.t.Helper()
	otherAddr, otherPub, otherSigner := buildSyncTestSigner(e.t)
	_, hdr := makeSignedHeaderBytes(
		e.t, e.chainID, height, otherAddr, otherPub, otherSigner,
		[]byte{variant, variant, variant},
		nil,
		nil,
	)
	return hdr
}

func (e *dsTestEnv) saveHeader(hdr *types.SignedHeader) {
	e.t.Helper()
	batch, err := e.store.NewBatch(context.Background())
	require.NoError(e.t, err)
	require.NoError(e.t, batch.SaveBlockData(hdr, &types.Data{
		Metadata: &types.Metadata{ChainID: e.chainID, Height: hdr.Height(), Time: hdr.BaseHeader.Time},
	}, &hdr.Signature))
	require.NoError(e.t, batch.SetHeight(hdr.Height()))
	require.NoError(e.t, batch.Commit())
}

func TestFirstObservation_StoredHeaderProducesEvidence(t *testing.T) {
	env := newDSTestEnv(t)
	first := env.signHeaderAtHeight(5, 0x01)
	env.saveHeader(first)

	alt := env.signHeaderAtHeight(5, 0x02)
	require.NotEqual(t, first.Hash().String(), alt.Hash().String())

	prior, priorSource, err := firstObservation(context.Background(), env.store, env.cache, alt.Height())
	require.NoError(t, err)
	require.Equal(t, first.Hash().String(), prior.Hash().String())
	require.Equal(t, types.EvidenceSourceStored, priorSource)

	ev := buildEvidenceFromPair(prior, alt, priorSource, types.EvidenceSourceP2P)
	require.NotNil(t, ev)
	require.Equal(t, uint64(5), ev.Height)
	require.Equal(t, first.Hash().String(), ev.FirstHeader.Hash().String())
	require.Equal(t, alt.Hash().String(), ev.AlternateHeader.Hash().String())
	require.Equal(t, types.EvidenceSourceP2P, ev.AlternateSource)

	// Round-trip through marshal/unmarshal.
	blob, err := ev.MarshalBinary()
	require.NoError(t, err)
	decoded := new(types.DoubleSignEvidence)
	require.NoError(t, decoded.UnmarshalBinary(blob))
	require.Equal(t, ev.Height, decoded.Height)
	require.Equal(t, ev.FirstHeader.Hash().String(), decoded.FirstHeader.Hash().String())
	require.Equal(t, ev.AlternateHeader.Hash().String(), decoded.AlternateHeader.Hash().String())
	require.Equal(t, ev.FirstSource, decoded.FirstSource)
	require.Equal(t, ev.AlternateSource, decoded.AlternateSource)
}

func TestFirstObservation_IdenticalHashNoEvidence(t *testing.T) {
	env := newDSTestEnv(t)
	first := env.signHeaderAtHeight(5, 0x01)
	env.saveHeader(first)

	prior, priorSource, err := firstObservation(context.Background(), env.store, env.cache, first.Height())
	require.NoError(t, err)
	require.NotNil(t, prior)
	require.Nil(t, buildEvidenceFromPair(prior, first, priorSource, types.EvidenceSourceP2P))
}

func TestFirstObservation_NoPriorRecordReturnsNil(t *testing.T) {
	env := newDSTestEnv(t)
	prior, priorSource, err := firstObservation(context.Background(), env.store, env.cache, 5)
	require.NoError(t, err)
	require.Nil(t, prior)
	require.Empty(t, priorSource)
}

func TestBuildEvidenceFromPair_ProposerMismatch(t *testing.T) {
	env := newDSTestEnv(t)
	first := env.signHeaderAtHeight(5, 0x01)
	alt := env.signHeaderWithOtherProposer(5, 0x02)
	require.Nil(t, buildEvidenceFromPair(first, alt, types.EvidenceSourceDA, types.EvidenceSourceDA))
}

func TestBuildEvidenceFromPair_HappyPath(t *testing.T) {
	env := newDSTestEnv(t)
	first := env.signHeaderAtHeight(5, 0x01)
	alt := env.signHeaderAtHeight(5, 0x02)
	ev := buildEvidenceFromPair(first, alt, types.EvidenceSourceDA, types.EvidenceSourceDA)
	require.NotNil(t, ev)
	require.NoError(t, ev.ValidateBasic())
}

func TestBuildEvidenceFromPair_EdgeCases(t *testing.T) {
	env := newDSTestEnv(t)
	a := env.signHeaderAtHeight(5, 0x01)
	b := env.signHeaderAtHeight(6, 0x02)

	require.Nil(t, buildEvidenceFromPair(nil, a, types.EvidenceSourceDA, types.EvidenceSourceDA))
	require.Nil(t, buildEvidenceFromPair(a, nil, types.EvidenceSourceDA, types.EvidenceSourceDA))
	require.Nil(t, buildEvidenceFromPair(a, b, types.EvidenceSourceDA, types.EvidenceSourceDA))
	require.Nil(t, buildEvidenceFromPair(a, a, types.EvidenceSourceDA, types.EvidenceSourceDA))
}

func TestPersistEvidence_RejectsInvalid(t *testing.T) {
	env := newDSTestEnv(t)
	first := env.signHeaderAtHeight(5, 0x01)
	bad := &types.DoubleSignEvidence{
		Height:          5,
		FirstHeader:     first,
		AlternateHeader: first,
	}
	require.Error(t, persistEvidence(context.Background(), env.store, bad))
}

func TestDoubleSignDedup(t *testing.T) {
	d := newDoubleSignDedup()
	require.True(t, d.markSeen(7, "abc"))
	require.False(t, d.markSeen(7, "abc"))
	require.True(t, d.markSeen(7, "def"))
	require.True(t, d.markSeen(8, "abc"))
}

func TestReportDoubleSign_PersistsAndHalts(t *testing.T) {
	env := newDSTestEnv(t)
	first := env.signHeaderAtHeight(5, 0x01)
	alt := env.signHeaderAtHeight(5, 0x02)
	ev := buildEvidenceFromPair(first, alt, types.EvidenceSourceP2P, types.EvidenceSourceDA)
	require.NotNil(t, ev)

	metrics := common.NopMetrics()
	seen := newDoubleSignDedup()

	halt1 := reportDoubleSign(context.Background(), env.store, metrics, zerolog.Nop(), seen, ev)
	require.Error(t, halt1)
	require.ErrorIs(t, halt1, ErrDoubleSign)

	// Second call must be a no-op via dedup.
	halt2 := reportDoubleSign(context.Background(), env.store, metrics, zerolog.Nop(), seen, ev)
	require.NoError(t, halt2)

	key := store.GetDoubleSignEvidenceKey(ev.Height, ev.AlternateHeader.Hash())
	blob, err := env.store.GetMetadata(context.Background(), key)
	require.NoError(t, err)
	decoded := new(types.DoubleSignEvidence)
	require.NoError(t, decoded.UnmarshalBinary(blob))
	require.Equal(t, ev.Height, decoded.Height)
	require.Equal(t, ev.AlternateHeader.Hash().String(), decoded.AlternateHeader.Hash().String())
}

func TestP2PHandler_DoubleSignTriggersCriticalError(t *testing.T) {
	env := newDSTestEnv(t)

	// Persist canonical header, then arrange a conflicting one to come in via P2P.
	first := env.signHeaderAtHeight(5, 0x01)
	env.saveHeader(first)
	alt := env.signHeaderAtHeight(5, 0x02)
	require.NotEqual(t, first.Hash().String(), alt.Hash().String())

	headerStoreMock := extmocks.NewMockStore[*types.P2PSignedHeader](t)
	dataStoreMock := extmocks.NewMockStore[*types.P2PData](t)
	headerStoreMock.EXPECT().
		GetByHeight(mock.Anything, uint64(5)).
		Return(&types.P2PSignedHeader{SignedHeader: alt}, nil).
		Once()

	h := NewP2PHandler(headerStoreMock, dataStoreMock, env.cache, env.gen, zerolog.Nop(), env.detectDoubleSign)

	ch := make(chan common.DAHeightEvent, 1)
	require.NoError(t, h.ProcessHeight(context.Background(), 5, ch))

	captured := env.captured()
	require.Len(t, captured, 1)
	require.Equal(t, alt.Hash().String(), captured[0].AlternateHeader.Hash().String())
	require.Equal(t, types.EvidenceSourceP2P, captured[0].AlternateSource)

	key := store.GetDoubleSignEvidenceKey(5, alt.Hash())
	blob, err := env.store.GetMetadata(context.Background(), key)
	require.NoError(t, err)
	require.NotEmpty(t, blob)

	select {
	case evt := <-ch:
		t.Fatalf("expected no event on double-sign; got %+v", evt)
	default:
	}
}

func TestP2PHandler_ProposerMismatchIsNotEvidence(t *testing.T) {
	env := newDSTestEnv(t)
	first := env.signHeaderAtHeight(5, 0x01)
	env.saveHeader(first)

	// A header from a different signer must be rejected before the detector runs.
	badHdr := env.signHeaderWithOtherProposer(5, 0x02)

	headerStoreMock := extmocks.NewMockStore[*types.P2PSignedHeader](t)
	dataStoreMock := extmocks.NewMockStore[*types.P2PData](t)
	headerStoreMock.EXPECT().
		GetByHeight(mock.Anything, uint64(5)).
		Return(&types.P2PSignedHeader{SignedHeader: badHdr}, nil).
		Once()

	h := NewP2PHandler(headerStoreMock, dataStoreMock, env.cache, env.gen, zerolog.Nop(), env.detectDoubleSign)

	ch := make(chan common.DAHeightEvent, 1)
	err := h.ProcessHeight(context.Background(), 5, ch)
	require.Error(t, err)
	require.Empty(t, env.captured())
}

func TestDARetriever_DoubleSignSamePendingBatch(t *testing.T) {
	env := newDSTestEnv(t)

	first := env.signHeaderAtHeight(5, 0x01)
	alt := env.signHeaderAtHeight(5, 0x02)
	require.NotEqual(t, first.Hash().String(), alt.Hash().String())

	firstBin, err := first.MarshalBinary()
	require.NoError(t, err)
	altBin, err := alt.MarshalBinary()
	require.NoError(t, err)

	mockClient := testmocks.NewMockClient(t)
	mockClient.On("GetHeaderNamespace").Return([]byte("ns")).Maybe()
	mockClient.On("GetDataNamespace").Return([]byte("ns")).Maybe()

	r := NewDARetriever(mockClient, env.cache, env.gen, zerolog.Nop(), env.detectDoubleSign)
	events := r.ProcessBlobs(context.Background(), [][]byte{firstBin, altBin}, 100)
	require.Empty(t, events)

	captured := env.captured()
	require.Len(t, captured, 1)
	require.Equal(t, uint64(5), captured[0].Height)
	require.Equal(t, types.EvidenceSourceDA, captured[0].FirstSource)
	require.Equal(t, types.EvidenceSourceDA, captured[0].AlternateSource)
}

func TestDARetriever_DoubleSignAcrossBatches(t *testing.T) {
	env := newDSTestEnv(t)

	first := env.signHeaderAtHeight(5, 0x01)
	env.saveHeader(first)

	alt := env.signHeaderAtHeight(5, 0x02)
	altBin, err := alt.MarshalBinary()
	require.NoError(t, err)

	mockClient := testmocks.NewMockClient(t)
	mockClient.On("GetHeaderNamespace").Return([]byte("ns")).Maybe()
	mockClient.On("GetDataNamespace").Return([]byte("ns")).Maybe()

	r := NewDARetriever(mockClient, env.cache, env.gen, zerolog.Nop(), env.detectDoubleSign)
	events := r.ProcessBlobs(context.Background(), [][]byte{altBin}, 101)
	require.Empty(t, events)

	captured := env.captured()
	require.Len(t, captured, 1)
	require.Equal(t, alt.Hash().String(), captured[0].AlternateHeader.Hash().String())
}

func TestDARetriever_BenignDuplicateAcrossBatchesDoesNotFire(t *testing.T) {
	env := newDSTestEnv(t)

	first := env.signHeaderAtHeight(5, 0x01)
	env.saveHeader(first)

	// Same header re-observed from DA (e.g. re-posted at a different DA height).
	sameBin, err := first.MarshalBinary()
	require.NoError(t, err)

	mockClient := testmocks.NewMockClient(t)
	mockClient.On("GetHeaderNamespace").Return([]byte("ns")).Maybe()
	mockClient.On("GetDataNamespace").Return([]byte("ns")).Maybe()

	r := NewDARetriever(mockClient, env.cache, env.gen, zerolog.Nop(), env.detectDoubleSign)
	_ = r.ProcessBlobs(context.Background(), [][]byte{sameBin}, 101)
	require.Empty(t, env.captured())
}

// A legacy blob with the correct proposer but a tampered signature must be
// rejected before reaching the detector or pending cache.
func TestDARetriever_LegacyForgedSignatureRejected(t *testing.T) {
	env := newDSTestEnv(t)

	// Tamper the signature byte to invalidate verification while preserving
	// every other field (proposer address included).
	good := env.signHeaderAtHeight(5, 0x01)
	pbHdr, err := good.ToProto()
	require.NoError(t, err)
	pbHdr.Signature = append([]byte(nil), good.Signature...)
	pbHdr.Signature[0] ^= 0xff
	bin, err := proto.Marshal(pbHdr)
	require.NoError(t, err)

	mockClient := testmocks.NewMockClient(t)
	mockClient.On("GetHeaderNamespace").Return([]byte("ns")).Maybe()
	mockClient.On("GetDataNamespace").Return([]byte("ns")).Maybe()

	r := NewDARetriever(mockClient, env.cache, env.gen, zerolog.Nop(), env.detectDoubleSign)
	require.Nil(t, r.tryDecodeHeader(bin, 100))

	_, _, ok := env.cache.GetPendingSignedHeader(5)
	require.False(t, ok)
}

// Detection must trigger from a pending cache entry too, before persistence.
func TestFirstObservation_PendingCacheHitProducesEvidence(t *testing.T) {
	env := newDSTestEnv(t)

	first := env.signHeaderAtHeight(5, 0x01)
	env.cache.SetPendingSignedHeader(first, types.EvidenceSourceDA)
	// First header is in-flight, not yet on disk.

	alt := env.signHeaderAtHeight(5, 0x02)
	prior, priorSource, err := firstObservation(context.Background(), env.store, env.cache, alt.Height())
	require.NoError(t, err)
	require.Equal(t, first.Hash().String(), prior.Hash().String())
	require.Equal(t, types.EvidenceSourceDA, priorSource)

	ev := buildEvidenceFromPair(prior, alt, priorSource, types.EvidenceSourceP2P)
	require.NotNil(t, ev)
	require.Equal(t, first.Hash().String(), ev.FirstHeader.Hash().String())
	require.Equal(t, alt.Hash().String(), ev.AlternateHeader.Hash().String())
	require.Equal(t, types.EvidenceSourceDA, ev.FirstSource)
	require.Equal(t, types.EvidenceSourceP2P, ev.AlternateSource)
}

func TestFirstObservation_PendingCacheBenignDuplicate(t *testing.T) {
	env := newDSTestEnv(t)

	first := env.signHeaderAtHeight(5, 0x01)
	env.cache.SetPendingSignedHeader(first, types.EvidenceSourceDA)

	prior, priorSource, err := firstObservation(context.Background(), env.store, env.cache, first.Height())
	require.NoError(t, err)
	require.NotNil(t, prior)
	require.Nil(t, buildEvidenceFromPair(prior, first, priorSource, types.EvidenceSourceP2P))
}

func TestFirstObservation_PendingEvictedAfterRemoval(t *testing.T) {
	env := newDSTestEnv(t)

	first := env.signHeaderAtHeight(5, 0x01)
	env.cache.SetPendingSignedHeader(first, types.EvidenceSourceDA)
	env.cache.RemovePendingSignedHeader(5)

	prior, _, err := firstObservation(context.Background(), env.store, env.cache, first.Height())
	require.NoError(t, err)
	require.Nil(t, prior)
}

func TestDoubleSignEvidence_ValidateBasic(t *testing.T) {
	env := newDSTestEnv(t)
	first := env.signHeaderAtHeight(5, 0x01)
	alt := env.signHeaderAtHeight(5, 0x02)

	t.Run("nil receiver", func(t *testing.T) {
		var e *types.DoubleSignEvidence
		require.Error(t, e.ValidateBasic())
	})
	t.Run("missing headers", func(t *testing.T) {
		require.Error(t, (&types.DoubleSignEvidence{Height: 5}).ValidateBasic())
	})
	t.Run("height mismatch", func(t *testing.T) {
		e := &types.DoubleSignEvidence{Height: 99, FirstHeader: first, AlternateHeader: alt}
		require.Error(t, e.ValidateBasic())
	})
	t.Run("identical hashes", func(t *testing.T) {
		e := &types.DoubleSignEvidence{Height: 5, FirstHeader: first, AlternateHeader: first}
		require.Error(t, e.ValidateBasic())
	})
	t.Run("proposer mismatch", func(t *testing.T) {
		other := env.signHeaderWithOtherProposer(5, 0x02)
		e := &types.DoubleSignEvidence{Height: 5, FirstHeader: first, AlternateHeader: other}
		require.ErrorContains(t, e.ValidateBasic(), "different proposers")
	})
	t.Run("happy path", func(t *testing.T) {
		e := &types.DoubleSignEvidence{Height: 5, FirstHeader: first, AlternateHeader: alt}
		require.NoError(t, e.ValidateBasic())
	})
}

func TestPBDoubleSignEvidence_RoundTrip(t *testing.T) {
	env := newDSTestEnv(t)
	ev := &types.DoubleSignEvidence{
		Height:          7,
		FirstHeader:     env.signHeaderAtHeight(7, 0x01),
		AlternateHeader: env.signHeaderAtHeight(7, 0x02),
		DetectedAt:      time.Unix(1_700_000_000, 500).UTC(),
		FirstSource:     types.EvidenceSourceDA,
		AlternateSource: types.EvidenceSourceP2P,
	}
	p, err := ev.ToProto()
	require.NoError(t, err)
	blob, err := proto.Marshal(p)
	require.NoError(t, err)

	decoded := new(pb.DoubleSignEvidence)
	require.NoError(t, proto.Unmarshal(blob, decoded))
	require.Equal(t, ev.Height, decoded.Height)
	require.Equal(t, ev.DetectedAt.UnixNano(), decoded.DetectedAt)
	require.Equal(t, ev.FirstSource, decoded.FirstSource)
	require.Equal(t, ev.AlternateSource, decoded.AlternateSource)
	require.NotNil(t, decoded.FirstHeader)
	require.NotNil(t, decoded.AlternateHeader)
	require.Equal(t, ev.FirstHeader.Height(), decoded.FirstHeader.Header.Height)
}

func TestDARetriever_DoubleSignEvidenceHasMatchingProposers(t *testing.T) {
	env := newDSTestEnv(t)

	first := env.signHeaderAtHeight(5, 0x01)
	alt := env.signHeaderAtHeight(5, 0x02)
	firstBin, err := first.MarshalBinary()
	require.NoError(t, err)
	altBin, err := alt.MarshalBinary()
	require.NoError(t, err)

	mockClient := testmocks.NewMockClient(t)
	mockClient.On("GetHeaderNamespace").Return([]byte("ns")).Maybe()
	mockClient.On("GetDataNamespace").Return([]byte("ns")).Maybe()

	r := NewDARetriever(mockClient, env.cache, env.gen, zerolog.Nop(), env.detectDoubleSign)
	_ = r.ProcessBlobs(context.Background(), [][]byte{firstBin, altBin}, 100)

	captured := env.captured()
	require.Len(t, captured, 1)
	require.Equal(t, env.gen.ProposerAddress, []byte(captured[0].FirstHeader.ProposerAddress))
	require.Equal(t, env.gen.ProposerAddress, []byte(captured[0].AlternateHeader.ProposerAddress))
}

func TestSyncer_EvictsPendingHeaderOnPersist(t *testing.T) {
	memDS := dssync.MutexWrap(ds.NewMapDatastore())
	st := store.New(memDS)

	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)

	addr, pub, signer := buildSyncTestSigner(t)
	cfg := config.DefaultConfig()
	gen := genesis.Genesis{
		ChainID: "syncer-evict", InitialHeight: 1,
		StartTime: time.Now().Add(-time.Second), ProposerAddress: addr,
	}

	mockExec := testmocks.NewMockExecutor(t)
	mockExec.EXPECT().InitChain(mock.Anything, mock.Anything, uint64(1), gen.ChainID).
		Return([]byte("app0"), nil).Once()
	mockExec.EXPECT().ExecuteTxs(mock.Anything, mock.Anything, uint64(1), mock.Anything, mock.Anything).
		Return([]byte("app1"), nil).Once()

	mockHeaderStore := extmocks.NewMockStore[*types.P2PSignedHeader](t)
	mockHeaderStore.EXPECT().Height().Return(uint64(0)).Maybe()
	mockDataStore := extmocks.NewMockStore[*types.P2PData](t)
	mockDataStore.EXPECT().Height().Return(uint64(0)).Maybe()

	s := NewSyncer(
		st, mockExec, nil, cm, common.NopMetrics(), cfg, gen,
		mockHeaderStore, mockDataStore, zerolog.Nop(),
		common.DefaultBlockOptions(), make(chan error, 1), nil,
	)
	require.NoError(t, s.initializeState())
	s.ctx = t.Context()

	state := s.getLastState()
	data := makeData(gen.ChainID, 1, 0)
	_, hdr := makeSignedHeaderBytes(t, gen.ChainID, 1, addr, pub, signer, state.AppHash, data, nil)

	cm.SetPendingSignedHeader(hdr, types.EvidenceSourceP2P)
	_, _, ok := cm.GetPendingSignedHeader(1)
	require.True(t, ok)

	evt := common.DAHeightEvent{Header: hdr, Data: data, DaHeight: 1}
	s.processHeightEvent(s.ctx, &evt)

	_, _, ok = cm.GetPendingSignedHeader(1)
	require.False(t, ok)
}

// End-to-end: a double-sign through a real Syncer must halt on errorCh,
// flip hasCriticalError, and bump DoubleSignsDetected only once for duplicate evidence.
func TestSyncer_DoubleSignHaltsAndEmitsCriticalError(t *testing.T) {
	memDS := dssync.MutexWrap(ds.NewMapDatastore())
	st := store.New(memDS)

	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)

	addr, pub, signer := buildSyncTestSigner(t)
	cfg := config.DefaultConfig()
	gen := genesis.Genesis{
		ChainID: "syncer-ds", InitialHeight: 1,
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

	// Wire a counting metric so we can assert exact increments.
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

	// Fire two identical alternate events to simulate P2P + DA converging.
	first := makeHeaderForSyncer(t, gen, addr, pub, signer, 1, 0x01)
	saveHeaderViaBatch(t, st, gen, first)

	alt := makeHeaderForSyncer(t, gen, addr, pub, signer, 1, 0x02)
	require.NotEqual(t, first.Hash().String(), alt.Hash().String())

	p2pEv := &types.DoubleSignEvidence{
		Height: 1, FirstHeader: first, AlternateHeader: alt,
		DetectedAt: time.Now(), FirstSource: types.EvidenceSourceStored, AlternateSource: types.EvidenceSourceP2P,
	}
	daEv := &types.DoubleSignEvidence{
		Height: 1, FirstHeader: first, AlternateHeader: alt,
		DetectedAt: time.Now(), FirstSource: types.EvidenceSourceStored, AlternateSource: types.EvidenceSourceDA,
	}

	s.handleDoubleSign(context.Background(), p2pEv)
	s.handleDoubleSign(context.Background(), daEv)

	require.Equal(t, int64(1), dsCount.Load(), "duplicate evidence must not double-count")
	require.True(t, s.hasCriticalError.Load())

	select {
	case got := <-errCh:
		require.ErrorIs(t, got, ErrDoubleSign)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for critical error on errCh")
	}

	key := store.GetDoubleSignEvidenceKey(1, alt.Hash())
	blob, err := st.GetMetadata(context.Background(), key)
	require.NoError(t, err)
	require.NotEmpty(t, blob)
}

func makeHeaderForSyncer(t *testing.T, gen genesis.Genesis, addr []byte, pub crypto.PubKey, signer signerpkg.Signer, height uint64, variant byte) *types.SignedHeader {
	t.Helper()
	_, hdr := makeSignedHeaderBytes(t, gen.ChainID, height, addr, pub, signer,
		[]byte{variant, variant, variant}, nil, nil)
	return hdr
}

// persists a signed header + empty data + signature and bumps the store height.
func saveHeaderViaBatch(t *testing.T, st store.Store, gen genesis.Genesis, hdr *types.SignedHeader) {
	t.Helper()
	batch, err := st.NewBatch(context.Background())
	require.NoError(t, err)
	require.NoError(t, batch.SaveBlockData(hdr, &types.Data{
		Metadata: &types.Metadata{ChainID: gen.ChainID, Height: hdr.Height(), Time: hdr.BaseHeader.Time},
	}, &hdr.Signature))
	require.NoError(t, batch.SetHeight(hdr.Height()))
	require.NoError(t, batch.Commit())
}

// go-kit Counter backed by an atomic int64 so tests can read exact increments.
type counterCtr struct {
	n *atomic.Int64
}

func (c *counterCtr) Add(delta float64)                            { c.n.Add(int64(delta)) }
func (c *counterCtr) With(labelValues ...string) gkmetrics.Counter { return c }
