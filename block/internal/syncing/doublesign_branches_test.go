package syncing

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/block/internal/da"
	"github.com/evstack/ev-node/pkg/store"
	testmocks "github.com/evstack/ev-node/test/mocks"
	"github.com/evstack/ev-node/types"
	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

// DA client stub with shared namespace mocks.
func newMockDAClient(t *testing.T) da.Client {
	t.Helper()
	c := testmocks.NewMockClient(t)
	c.On("GetHeaderNamespace").Return([]byte("ns")).Maybe()
	c.On("GetDataNamespace").Return([]byte("ns")).Maybe()
	return c
}

// errStore wraps a store and injects errors on selected reads/writes to
// exercise error-handling branches an in-memory store can't hit.
type errStore struct {
	store.Store
	getHeaderErr   error
	setMetadataErr error
}

func (e *errStore) GetHeader(ctx context.Context, height uint64) (*types.SignedHeader, error) {
	if e.getHeaderErr != nil {
		return nil, e.getHeaderErr
	}
	return e.Store.GetHeader(ctx, height)
}

func (e *errStore) SetMetadata(ctx context.Context, key string, value []byte) error {
	if e.setMetadataErr != nil {
		return e.setMetadataErr
	}
	return e.Store.SetMetadata(ctx, key, value)
}

// nilHeaderStore returns (nil, nil) from GetHeader; the detector must treat
// that as "no record" rather than crashing.
type nilHeaderStore struct{ store.Store }

func (nilHeaderStore) GetHeader(context.Context, uint64) (*types.SignedHeader, error) {
	return nil, nil
}

func TestDetectDoubleSign_NilIncomingReturnsError(t *testing.T) {
	env := newDSTestEnv(t)
	ev, err := detectDoubleSign(context.Background(), env.store, env.cache, nil, types.EvidenceSourceP2P)
	require.Error(t, err)
	require.Nil(t, ev)
}

// A non-NotFound store failure must be surfaced, not swallowed.
func TestDetectDoubleSign_StoreErrorWrapped(t *testing.T) {
	env := newDSTestEnv(t)
	wrapped := &errStore{Store: env.store, getHeaderErr: errors.New("backend down")}

	alt := env.signHeaderAtHeight(5, 0x01)
	ev, err := detectDoubleSign(context.Background(), wrapped, env.cache, alt, types.EvidenceSourceP2P)
	require.Error(t, err)
	require.Nil(t, ev)
	require.Contains(t, err.Error(), "lookup stored header")
	require.ErrorContains(t, err, "backend down")
}

func TestDetectDoubleSign_StoredHeaderNilDefensive(t *testing.T) {
	env := newDSTestEnv(t)
	wrapped := nilHeaderStore{Store: env.store}

	alt := env.signHeaderAtHeight(5, 0x01)
	ev, err := detectDoubleSign(context.Background(), wrapped, env.cache, alt, types.EvidenceSourceP2P)
	require.NoError(t, err)
	require.Nil(t, ev)
}

// Store-path detections must use the "stored" sentinel as FirstSource so
// downstream consumers can disambiguate it from in-flight observations.
func TestDetectDoubleSign_FirstSourceStoredSentinel(t *testing.T) {
	env := newDSTestEnv(t)
	first := env.signHeaderAtHeight(5, 0x01)
	env.saveHeader(first)

	alt := env.signHeaderAtHeight(5, 0x02)
	ev, err := detectDoubleSign(context.Background(), env.store, env.cache, alt, types.EvidenceSourceP2P)
	require.NoError(t, err)
	require.NotNil(t, ev)
	require.Equal(t, "stored", ev.FirstSource)
}

// SetMetadata failures must include the canonical key so an operator can
// recover the persistence target from logs alone.
func TestPersistEvidence_StoreError(t *testing.T) {
	env := newDSTestEnv(t)
	wrapped := &errStore{Store: env.store, setMetadataErr: errors.New("disk full")}

	first := env.signHeaderAtHeight(5, 0x01)
	alt := env.signHeaderAtHeight(5, 0x02)
	ev := buildEvidenceFromPair(first, alt, types.EvidenceSourceP2P, types.EvidenceSourceDA)
	require.NotNil(t, ev)

	err := persistEvidence(context.Background(), wrapped, ev)
	require.Error(t, err)
	require.ErrorContains(t, err, "disk full")
	require.Contains(t, err.Error(), store.GetDoubleSignEvidenceKey(ev.Height, ev.AlternateHeader.Hash()))
}

// Persistence failure must not break the halt contract: metric still
// increments, criticalErr still fires, returned error still wraps ErrDoubleSign.
func TestReportDoubleSign_PersistFailureLoggedNotBlocking(t *testing.T) {
	env := newDSTestEnv(t)
	wrapped := &errStore{Store: env.store, setMetadataErr: errors.New("disk full")}

	first := env.signHeaderAtHeight(5, 0x01)
	alt := env.signHeaderAtHeight(5, 0x02)
	ev := buildEvidenceFromPair(first, alt, types.EvidenceSourceP2P, types.EvidenceSourceDA)
	require.NotNil(t, ev)

	var dsCount atomic.Int64
	metrics := common.NopMetrics()
	metrics.DoubleSignsDetected = &counterCtr{n: &dsCount}

	var fired atomic.Pointer[error]
	crit := func(err error) { fired.Store(&err) }

	halt := reportDoubleSign(context.Background(), wrapped, metrics, zerolog.Nop(),
		newDoubleSignDedup(), crit, ev)
	require.Error(t, halt)
	require.ErrorIs(t, halt, ErrDoubleSign)

	require.Equal(t, int64(1), dsCount.Load())
	require.NotNil(t, fired.Load())
}

// Dedup is keyed on (height, altHash), so two distinct alts at the same
// height must each produce evidence.
func TestReportDoubleSign_TwoDistinctAltsAtSameHeight(t *testing.T) {
	env := newDSTestEnv(t)
	first := env.signHeaderAtHeight(5, 0x01)
	alt1 := env.signHeaderAtHeight(5, 0x02)
	alt2 := env.signHeaderAtHeight(5, 0x03)
	require.NotEqual(t, alt1.Hash().String(), alt2.Hash().String())

	ev1 := buildEvidenceFromPair(first, alt1, types.EvidenceSourceP2P, types.EvidenceSourceDA)
	ev2 := buildEvidenceFromPair(first, alt2, types.EvidenceSourceP2P, types.EvidenceSourceDA)
	require.NotNil(t, ev1)
	require.NotNil(t, ev2)

	var dsCount atomic.Int64
	metrics := common.NopMetrics()
	metrics.DoubleSignsDetected = &counterCtr{n: &dsCount}

	seen := newDoubleSignDedup()
	noopCrit := func(error) {}

	require.Error(t, reportDoubleSign(context.Background(), env.store, metrics,
		zerolog.Nop(), seen, noopCrit, ev1))
	require.Error(t, reportDoubleSign(context.Background(), env.store, metrics,
		zerolog.Nop(), seen, noopCrit, ev2))

	require.Equal(t, int64(2), dsCount.Load())

	for _, ev := range []*types.DoubleSignEvidence{ev1, ev2} {
		key := store.GetDoubleSignEvidenceKey(ev.Height, ev.AlternateHeader.Hash())
		blob, err := env.store.GetMetadata(context.Background(), key)
		require.NoError(t, err)
		require.NotEmpty(t, blob)
	}
}

func TestReportDoubleSign_NilSeenAndNilGuards(t *testing.T) {
	env := newDSTestEnv(t)
	first := env.signHeaderAtHeight(5, 0x01)
	alt := env.signHeaderAtHeight(5, 0x02)
	ev := buildEvidenceFromPair(first, alt, types.EvidenceSourceP2P, types.EvidenceSourceDA)
	require.NotNil(t, ev)

	t.Run("nil seen still halts", func(t *testing.T) {
		halt := reportDoubleSign(context.Background(), env.store, common.NopMetrics(),
			zerolog.Nop(), nil, func(error) {}, ev)
		require.Error(t, halt)
		require.ErrorIs(t, halt, ErrDoubleSign)
	})

	t.Run("nil metrics still halts", func(t *testing.T) {
		halt := reportDoubleSign(context.Background(), env.store, nil,
			zerolog.Nop(), newDoubleSignDedup(), func(error) {}, ev)
		require.Error(t, halt)
		require.ErrorIs(t, halt, ErrDoubleSign)
	})

	t.Run("nil counter inside metrics still halts", func(t *testing.T) {
		m := common.NopMetrics()
		m.DoubleSignsDetected = nil
		halt := reportDoubleSign(context.Background(), env.store, m,
			zerolog.Nop(), newDoubleSignDedup(), func(error) {}, ev)
		require.Error(t, halt)
		require.ErrorIs(t, halt, ErrDoubleSign)
	})

	t.Run("nil criticalErr still halts", func(t *testing.T) {
		halt := reportDoubleSign(context.Background(), env.store, common.NopMetrics(),
			zerolog.Nop(), newDoubleSignDedup(), nil, ev)
		require.Error(t, halt)
		require.ErrorIs(t, halt, ErrDoubleSign)
	})
}

func TestDoubleSignEvidence_FromProtoNil(t *testing.T) {
	dst := new(types.DoubleSignEvidence)
	require.Error(t, dst.FromProto(nil))
}

func TestDoubleSignEvidence_FromProtoInnerHeaderError(t *testing.T) {
	// Both inner SignedHeader fields nil — the wrapper must surface the error.
	p := &pb.DoubleSignEvidence{Height: 5}
	dst := new(types.DoubleSignEvidence)
	require.Error(t, dst.FromProto(p))
}

func TestDoubleSignEvidence_UnmarshalBinaryGarbage(t *testing.T) {
	dst := new(types.DoubleSignEvidence)
	require.Error(t, dst.UnmarshalBinary([]byte{0xff, 0xff, 0xff, 0xff}))
}

// Once equivocation is detected, the rest of the batch must be dropped.
func TestDARetriever_AbortsBatchOnDetection(t *testing.T) {
	env := newDSTestEnv(t)

	first := env.signHeaderAtHeight(5, 0x01)
	alt := env.signHeaderAtHeight(5, 0x02)
	next := env.signHeaderAtHeight(6, 0x01)

	firstBin, err := first.MarshalBinary()
	require.NoError(t, err)
	altBin, err := alt.MarshalBinary()
	require.NoError(t, err)
	nextBin, err := next.MarshalBinary()
	require.NoError(t, err)

	mockClient := newMockDAClient(t)

	r := NewDARetriever(mockClient, env.cache, env.gen, zerolog.Nop(), env.store, env.onDouble)

	events := r.ProcessBlobs(context.Background(),
		[][]byte{firstBin, altBin, nextBin}, 100)
	require.Empty(t, events)
	require.NotContains(t, r.pendingHeaders, uint64(6))
}

// On a detector error, the retriever still caches the header so a later
// alternate can be matched once the store recovers.
func TestDARetriever_DetectorErrorWarnAndContinue(t *testing.T) {
	env := newDSTestEnv(t)
	wrapped := &errStore{Store: env.store, getHeaderErr: errors.New("flapping disk")}

	hdr := env.signHeaderAtHeight(5, 0x01)
	bin, err := hdr.MarshalBinary()
	require.NoError(t, err)

	mockClient := newMockDAClient(t)

	r := NewDARetriever(mockClient, env.cache, env.gen, zerolog.Nop(), wrapped, env.onDouble)

	_ = r.ProcessBlobs(context.Background(), [][]byte{bin}, 100)

	require.Empty(t, env.captured())

	got, src, ok := env.cache.GetPendingSignedHeader(hdr.Height())
	require.True(t, ok)
	require.Equal(t, hdr.Hash().String(), got.Hash().String())
	require.Equal(t, types.EvidenceSourceDA, src)
}

// Double-sign detection through the envelope (strict-mode) path.
func TestDARetriever_StrictModeEnvelopeDoubleSign(t *testing.T) {
	env := newDSTestEnv(t)

	first := env.signHeaderAtHeight(5, 0x01)
	alt := env.signHeaderAtHeight(5, 0x02)

	mkEnvelope := func(h *types.SignedHeader) []byte {
		content, err := h.MarshalBinary()
		require.NoError(t, err)
		envSig, err := env.signer.Sign(t.Context(), content)
		require.NoError(t, err)
		envBin, err := h.MarshalDAEnvelope(envSig)
		require.NoError(t, err)
		return envBin
	}

	firstBin := mkEnvelope(first)
	altBin := mkEnvelope(alt)

	mockClient := newMockDAClient(t)

	r := NewDARetriever(mockClient, env.cache, env.gen, zerolog.Nop(), env.store, env.onDouble)

	events := r.ProcessBlobs(context.Background(), [][]byte{firstBin, altBin}, 100)
	require.Empty(t, events)
	require.True(t, r.strictMode)

	captured := env.captured()
	require.Len(t, captured, 1)
	require.Equal(t, uint64(5), captured[0].Height)
	require.Equal(t, types.EvidenceSourceDA, captured[0].FirstSource)
	require.Equal(t, types.EvidenceSourceDA, captured[0].AlternateSource)
}

// First DA observation must populate the pending cache so a later cross-source
// alternate can be matched against it.
func TestDARetriever_SetsPendingSignedHeaderOnFirstObservation(t *testing.T) {
	env := newDSTestEnv(t)

	first := env.signHeaderAtHeight(5, 0x01)
	bin, err := first.MarshalBinary()
	require.NoError(t, err)

	mockClient := newMockDAClient(t)

	r := NewDARetriever(mockClient, env.cache, env.gen, zerolog.Nop(), env.store, env.onDouble)
	_ = r.ProcessBlobs(context.Background(), [][]byte{bin}, 100)

	got, src, ok := env.cache.GetPendingSignedHeader(5)
	require.True(t, ok)
	require.Equal(t, first.Hash().String(), got.Hash().String())
	require.Equal(t, types.EvidenceSourceDA, src)
}

