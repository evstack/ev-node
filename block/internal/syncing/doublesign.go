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

// doubleSignHandler is fired when an equivocation is confirmed. It persists
// evidence, bumps metrics, and halts the syncer.
type doubleSignHandler func(ctx context.Context, evidence *types.DoubleSignEvidence)

// detectDoubleSign compares incoming against the first-seen SignedHeader at
// the same height (cache then store) and returns non-nil evidence when their
// hashes differ. Caller must verify proposer + signature first.
func detectDoubleSign(
	ctx context.Context,
	st store.Store,
	cm cache.CacheManager,
	incoming *types.SignedHeader,
	incomingSource string,
) (*types.DoubleSignEvidence, error) {
	if incoming == nil {
		return nil, errors.New("incoming header is nil")
	}
	height := incoming.Height()

	// Cache wins over store: the cached entry is the literal first observation
	// and carries the original FirstSource.
	if cached, source, ok := cm.GetPendingSignedHeader(height); ok {
		return buildEvidenceFromPair(cached, incoming, source, incomingSource), nil
	}

	storedHeader, storeErr := st.GetHeader(ctx, height)
	if storeErr != nil {
		if store.IsNotFound(storeErr) {
			return nil, nil
		}
		return nil, fmt.Errorf("lookup stored header at %d: %w", height, storeErr)
	}
	if storedHeader == nil {
		return nil, nil
	}
	return buildEvidenceFromPair(storedHeader, incoming, "stored", incomingSource), nil
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

// reportDoubleSign persists evidence, logs, bumps the metric (once per
// distinct alternate hash via seen), fires criticalErr, and returns the
// wrapped ErrDoubleSign for the caller to propagate as the halt cause.
func reportDoubleSign(
	ctx context.Context,
	st store.Store,
	metrics *common.Metrics,
	logger zerolog.Logger,
	seen *doubleSignDedup,
	criticalErr func(error),
	ev *types.DoubleSignEvidence,
) error {
	altHashStr := ev.AlternateHeader.Hash().String()
	firstHashStr := ev.FirstHeader.Hash().String()
	key := store.GetDoubleSignEvidenceKey(ev.Height, ev.AlternateHeader.Hash())

	// Persist on every call: idempotent, and a retry covers a transient
	// failure on the first attempt.
	persistErr := persistEvidence(ctx, st, ev)

	if seen != nil && !seen.markSeen(ev.Height, altHashStr) {
		return nil
	}

	if persistErr != nil {
		logger.Error().Err(persistErr).
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

	halt := fmt.Errorf(
		"double-sign detected at height %d: sequencer signed conflicting headers %s and %s. "+
			"Evidence persisted at metadata key %s. Manual intervention required: %w",
		ev.Height, firstHashStr, altHashStr, key, ErrDoubleSign,
	)
	if criticalErr != nil {
		criticalErr(halt)
	}
	return halt
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
