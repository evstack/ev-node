package genesis

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	"github.com/libp2p/go-libp2p/core/crypto"
)

// ProposerScheduleEntry declares the proposer key that becomes active at start_height.
type ProposerScheduleEntry struct {
	StartHeight uint64 `json:"start_height"`
	Address     []byte `json:"address"`
	PubKey      []byte `json:"pub_key,omitempty"`
}

// NewProposerScheduleEntry creates a proposer schedule entry from a libp2p public key.
func NewProposerScheduleEntry(startHeight uint64, pubKey crypto.PubKey) (ProposerScheduleEntry, error) {
	if pubKey == nil {
		return ProposerScheduleEntry{}, fmt.Errorf("proposer pub_key cannot be nil")
	}

	marshalledPubKey, err := crypto.MarshalPublicKey(pubKey)
	if err != nil {
		return ProposerScheduleEntry{}, fmt.Errorf("marshal proposer pub_key: %w", err)
	}

	return ProposerScheduleEntry{
		StartHeight: startHeight,
		Address:     proposerKeyAddress(pubKey),
		PubKey:      marshalledPubKey,
	}, nil
}

// PublicKey unmarshals the configured proposer public key. Legacy single-proposer
// configs may omit the pubkey and will return nil, nil here.
func (e ProposerScheduleEntry) PublicKey() (crypto.PubKey, error) {
	if len(e.PubKey) == 0 {
		return nil, nil
	}

	pubKey, err := crypto.UnmarshalPublicKey(e.PubKey)
	if err != nil {
		return nil, fmt.Errorf("unmarshal proposer pub_key: %w", err)
	}

	return pubKey, nil
}

func (e ProposerScheduleEntry) validate(initialHeight uint64, requirePubKey bool) error {
	if e.StartHeight < initialHeight {
		return fmt.Errorf("proposer schedule start_height must be >= initial_height (%d), got %d", initialHeight, e.StartHeight)
	}

	if len(e.Address) == 0 {
		return fmt.Errorf("proposer schedule address cannot be empty")
	}

	if len(e.PubKey) == 0 {
		if requirePubKey {
			return fmt.Errorf("proposer schedule pub_key cannot be empty")
		}
		return nil
	}

	pubKey, err := e.PublicKey()
	if err != nil {
		return err
	}

	expectedAddress := proposerKeyAddress(pubKey)
	if !bytes.Equal(expectedAddress, e.Address) {
		return fmt.Errorf("proposer schedule address does not match pub_key: got %x, expected %x", e.Address, expectedAddress)
	}

	return nil
}

// EffectiveProposerSchedule returns the explicit proposer schedule when present,
// or derives a legacy single-entry schedule from proposer_address.
func (g Genesis) EffectiveProposerSchedule() []ProposerScheduleEntry {
	if len(g.ProposerSchedule) > 0 {
		out := make([]ProposerScheduleEntry, len(g.ProposerSchedule))
		copy(out, g.ProposerSchedule)
		return out
	}

	if len(g.ProposerAddress) == 0 {
		return nil
	}

	return []ProposerScheduleEntry{{
		StartHeight: g.InitialHeight,
		Address:     cloneBytes(g.ProposerAddress),
	}}
}

// InitialProposerAddress returns the first proposer address for compatibility
// with code paths that still surface a single address externally.
func (g Genesis) InitialProposerAddress() []byte {
	entry, err := g.ProposerAtHeight(g.InitialHeight)
	if err != nil {
		return nil
	}

	return cloneBytes(entry.Address)
}

func (g Genesis) normalized() Genesis {
	normalized := g
	if len(normalized.ProposerAddress) == 0 {
		normalized.ProposerAddress = normalized.InitialProposerAddress()
	}
	return normalized
}

// HasScheduledProposer reports whether the address appears in the effective proposer schedule.
func (g Genesis) HasScheduledProposer(address []byte) bool {
	for _, entry := range g.EffectiveProposerSchedule() {
		if bytes.Equal(entry.Address, address) {
			return true
		}
	}
	return false
}

// ProposerAtHeight resolves the proposer that is active for the given block height.
func (g Genesis) ProposerAtHeight(height uint64) (ProposerScheduleEntry, error) {
	schedule := g.EffectiveProposerSchedule()
	if len(schedule) == 0 {
		return ProposerScheduleEntry{}, fmt.Errorf("no proposer configured")
	}

	if height < schedule[0].StartHeight {
		return ProposerScheduleEntry{}, fmt.Errorf("no proposer configured for height %d before start_height %d", height, schedule[0].StartHeight)
	}

	entry := schedule[0]
	for i := 1; i < len(schedule); i++ {
		if height < schedule[i].StartHeight {
			break
		}
		entry = schedule[i]
	}

	return ProposerScheduleEntry{
		StartHeight: entry.StartHeight,
		Address:     cloneBytes(entry.Address),
		PubKey:      cloneBytes(entry.PubKey),
	}, nil
}

// ValidateProposer checks that the provided proposer address and public key match
// the proposer schedule entry active at the given height.
func (g Genesis) ValidateProposer(height uint64, address []byte, pubKey crypto.PubKey) error {
	entry, err := g.ProposerAtHeight(height)
	if err != nil {
		return err
	}

	if !bytes.Equal(entry.Address, address) {
		return fmt.Errorf("unexpected proposer at height %d: got %x, expected %x", height, address, entry.Address)
	}

	if len(entry.PubKey) == 0 {
		return nil
	}

	if pubKey == nil {
		return fmt.Errorf("missing proposer pub_key at height %d", height)
	}

	marshalledPubKey, err := crypto.MarshalPublicKey(pubKey)
	if err != nil {
		return fmt.Errorf("marshal proposer pub_key: %w", err)
	}

	if !bytes.Equal(entry.PubKey, marshalledPubKey) {
		return fmt.Errorf("unexpected proposer pub_key at height %d", height)
	}

	return nil
}

func cloneBytes(src []byte) []byte {
	if src == nil {
		return nil
	}

	out := make([]byte, len(src))
	copy(out, src)
	return out
}

func proposerKeyAddress(pubKey crypto.PubKey) []byte {
	if pubKey == nil {
		return nil
	}

	raw, err := pubKey.Raw()
	if err != nil {
		return nil
	}

	sum := sha256.Sum256(raw)
	return sum[:]
}
