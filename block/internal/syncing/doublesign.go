package syncing

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

// ErrDoubleSign is returned when two validly-signed SignedHeaders are observed at the same height.
var ErrDoubleSign = errors.New("double-sign detected")

// doubleSignDetector reports an observed header for equivocation detection.
// Returns true on a confirmed double-sign so the caller can abort.
type doubleSignDetector func(ctx context.Context, header *types.SignedHeader, source string) bool

// firstObservation returns the first-seen SignedHeader at this height,
// preferring the cache over the store. Returns (nil, "", nil) when none.
func firstObservation(
	ctx context.Context,
	st store.Store,
	cm cache.CacheManager,
	height uint64,
) (*types.SignedHeader, string, error) {
	if cached, source, ok := cm.GetPendingSignedHeader(height); ok {
		return cached, source, nil
	}

	storedHeader, err := st.GetHeader(ctx, height)
	if err != nil {
		if store.IsNotFound(err) {
			return nil, "", nil
		}
		return nil, "", fmt.Errorf("lookup stored header at %d: %w", height, err)
	}
	if storedHeader == nil {
		return nil, "", nil
	}
	return storedHeader, types.EvidenceSourceStored, nil
}

// buildEvidenceFromPair returns evidence for two SignedHeaders at the same
// height with different hashes and matching proposer. Returns nil otherwise.
func buildEvidenceFromPair(first, alternate *types.SignedHeader, firstSource, altSource string) *types.DoubleSignEvidence {
	if first == nil || alternate == nil {
		return nil
	}
	if first.Height() != alternate.Height() {
		return nil
	}
	if bytes.Equal(first.Hash(), alternate.Hash()) {
		return nil
	}
	if !bytes.Equal(first.ProposerAddress, alternate.ProposerAddress) {
		return nil
	}
	return &types.DoubleSignEvidence{
		Height:          first.Height(),
		FirstHeader:     first,
		AlternateHeader: alternate,
		DetectedAt:      time.Now().UTC(),
		FirstSource:     firstSource,
		AlternateSource: altSource,
	}
}

// persistEvidence writes evidence to its canonical metadata key. Idempotent.
func persistEvidence(ctx context.Context, st store.Store, ev *types.DoubleSignEvidence) error {
	if err := ev.ValidateBasic(); err != nil {
		return fmt.Errorf("invalid evidence: %w", err)
	}
	blob, err := ev.MarshalBinary()
	if err != nil {
		return fmt.Errorf("marshal evidence: %w", err)
	}
	altHash := ev.AlternateHeader.Hash()
	key := store.GetDoubleSignEvidenceKey(ev.Height, altHash)
	if err := st.SetMetadata(ctx, key, blob); err != nil {
		return fmt.Errorf("persist evidence at %s: %w", key, err)
	}
	return nil
}

// reportDoubleSign persists evidence and bumps the metric, deduping by
// (height, altHash). Returns a wrapped ErrDoubleSign on first sighting,
// nil when already seen.
func reportDoubleSign(
	ctx context.Context,
	st store.Store,
	metrics *common.Metrics,
	logger zerolog.Logger,
	seen *doubleSignDedup,
	ev *types.DoubleSignEvidence,
) error {
	altHashStr := ev.AlternateHeader.Hash().String()
	firstHashStr := ev.FirstHeader.Hash().String()
	key := store.GetDoubleSignEvidenceKey(ev.Height, ev.AlternateHeader.Hash())

	if seen != nil && !seen.markSeen(ev.Height, altHashStr) {
		return nil
	}

	if err := persistEvidence(ctx, st, ev); err != nil {
		logger.Error().Err(err).
			Uint64("height", ev.Height).
			Str("first_hash", firstHashStr).
			Str("alternate_hash", altHashStr).
			Msg("failed to persist double-sign evidence")
	}

	if metrics != nil && metrics.DoubleSignsDetected != nil {
		metrics.DoubleSignsDetected.Add(1)
	}

	logger.Error().
		Uint64("height", ev.Height).
		Str("first_hash", firstHashStr).
		Str("first_source", ev.FirstSource).
		Str("alternate_hash", altHashStr).
		Str("alternate_source", ev.AlternateSource).
		Str("evidence_key", key).
		Msg("DOUBLE-SIGN DETECTED — sequencer equivocation; halting syncer")

	return fmt.Errorf(
		"double-sign detected at height %d: sequencer signed conflicting headers %s and %s. "+
			"Evidence persisted at metadata key %s. Manual intervention required: %w",
		ev.Height, firstHashStr, altHashStr, key, ErrDoubleSign,
	)
}

// doubleSignDedup collapses (height, altHash) duplicates so the same
// equivocation arriving from both P2P and DA is only reported once.
type doubleSignDedup struct {
	mu   sync.Mutex
	seen map[string]struct{}
}

func newDoubleSignDedup() *doubleSignDedup {
	return &doubleSignDedup{seen: make(map[string]struct{})}
}

// markSeen records (height, altHash) and returns true on first sight.
func (d *doubleSignDedup) markSeen(height uint64, altHash string) bool {
	key := fmt.Sprintf("%d/%s", height, altHash)
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.seen[key]; ok {
		return false
	}
	d.seen[key] = struct{}{}
	return true
}
