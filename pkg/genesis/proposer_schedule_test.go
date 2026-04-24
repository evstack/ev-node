package genesis

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/pkg/signer/noop"
)

// testGenesisStartTime is a fixed timestamp for genesis fixtures so tests do
// not depend on wall-clock time.
var testGenesisStartTime = time.Unix(1_700_000_000, 0).UTC()

func makeProposerScheduleEntry(t *testing.T, startHeight uint64) (ProposerScheduleEntry, crypto.PubKey) {
	t.Helper()

	_, pubKey, err := crypto.GenerateEd25519Key(rand.Reader)
	require.NoError(t, err)

	entry, err := NewProposerScheduleEntry(startHeight, pubKey)
	require.NoError(t, err)

	return entry, pubKey
}

func TestGenesisProposerAtHeight(t *testing.T) {
	entry1, _ := makeProposerScheduleEntry(t, 3)
	entry2, _ := makeProposerScheduleEntry(t, 10)

	genesis := Genesis{
		ChainID:                "test-chain",
		StartTime:              testGenesisStartTime,
		InitialHeight:          3,
		ProposerSchedule:       []ProposerScheduleEntry{entry1, entry2},
		DAEpochForcedInclusion: 1,
	}

	require.NoError(t, genesis.Validate())

	proposer, err := genesis.ProposerAtHeight(3)
	require.NoError(t, err)
	require.Equal(t, entry1.Address, proposer.Address)

	proposer, err = genesis.ProposerAtHeight(9)
	require.NoError(t, err)
	require.Equal(t, entry1.Address, proposer.Address)

	proposer, err = genesis.ProposerAtHeight(10)
	require.NoError(t, err)
	require.Equal(t, entry2.Address, proposer.Address)
}

func TestGenesisValidateProposerScheduleWithPinnedPubKey(t *testing.T) {
	entry1, pubKey1 := makeProposerScheduleEntry(t, 1)
	entry2, pubKey2 := makeProposerScheduleEntry(t, 20)

	genesis := Genesis{
		ChainID:                "test-chain",
		StartTime:              testGenesisStartTime,
		InitialHeight:          1,
		ProposerSchedule:       []ProposerScheduleEntry{entry1, entry2},
		DAEpochForcedInclusion: 1,
	}

	require.NoError(t, genesis.Validate())
	require.NoError(t, genesis.ValidateProposer(1, entry1.Address, pubKey1))
	require.NoError(t, genesis.ValidateProposer(21, entry2.Address, pubKey2))
	require.Error(t, genesis.ValidateProposer(21, entry2.Address, pubKey1))
}

func TestGenesisValidateAddressOnlyProposerSchedule(t *testing.T) {
	entry1, pubKey1 := makeProposerScheduleEntry(t, 1)
	entry2, pubKey2 := makeProposerScheduleEntry(t, 20)
	entry1.PubKey = nil
	entry2.PubKey = nil

	genesis := Genesis{
		ChainID:                "test-chain",
		StartTime:              testGenesisStartTime,
		InitialHeight:          1,
		ProposerSchedule:       []ProposerScheduleEntry{entry1, entry2},
		DAEpochForcedInclusion: 1,
	}

	require.NoError(t, genesis.Validate())
	require.NoError(t, genesis.ValidateProposer(1, entry1.Address, pubKey1))
	require.NoError(t, genesis.ValidateProposer(21, entry2.Address, pubKey2))
}

func TestNewProposerScheduleEntry_NilPubKey(t *testing.T) {
	_, err := NewProposerScheduleEntry(1, nil)
	require.Error(t, err)
}

func TestProposerAtHeight_BeforeFirstStartHeight(t *testing.T) {
	entry, _ := makeProposerScheduleEntry(t, 5)
	genesis := Genesis{
		ChainID:                "c",
		StartTime:              testGenesisStartTime,
		InitialHeight:          5,
		ProposerSchedule:       []ProposerScheduleEntry{entry},
		DAEpochForcedInclusion: 1,
	}

	_, err := genesis.ProposerAtHeight(4)
	require.Error(t, err)
	require.Contains(t, err.Error(), "before start_height")
}

func TestProposerAtHeight_NoProposerConfigured(t *testing.T) {
	genesis := Genesis{ChainID: "c", InitialHeight: 1}
	_, err := genesis.ProposerAtHeight(1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no proposer configured")
}

func TestProposerAtHeight_ReturnedEntryIsCopy(t *testing.T) {
	entry, _ := makeProposerScheduleEntry(t, 1)
	genesis := Genesis{
		ChainID:                "c",
		StartTime:              testGenesisStartTime,
		InitialHeight:          1,
		ProposerSchedule:       []ProposerScheduleEntry{entry},
		DAEpochForcedInclusion: 1,
	}

	got, err := genesis.ProposerAtHeight(1)
	require.NoError(t, err)
	got.Address[0] ^= 0xFF
	got.PubKey[0] ^= 0xFF

	same, err := genesis.ProposerAtHeight(1)
	require.NoError(t, err)
	require.Equal(t, entry.Address, same.Address)
	require.Equal(t, entry.PubKey, same.PubKey)
}

func TestValidateProposer_WrongAddress(t *testing.T) {
	entry, pubKey := makeProposerScheduleEntry(t, 1)
	other, _ := makeProposerScheduleEntry(t, 1)
	genesis := Genesis{
		ChainID:                "c",
		StartTime:              testGenesisStartTime,
		InitialHeight:          1,
		ProposerSchedule:       []ProposerScheduleEntry{entry},
		DAEpochForcedInclusion: 1,
	}

	err := genesis.ValidateProposer(1, other.Address, pubKey)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unexpected proposer at height 1")
}

func TestValidateProposer_MissingPubKey(t *testing.T) {
	entry, _ := makeProposerScheduleEntry(t, 1)
	genesis := Genesis{
		ChainID:                "c",
		StartTime:              testGenesisStartTime,
		InitialHeight:          1,
		ProposerSchedule:       []ProposerScheduleEntry{entry},
		DAEpochForcedInclusion: 1,
	}

	err := genesis.ValidateProposer(1, entry.Address, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing proposer pub_key")
}

// TestValidateProposer_AddressOnly_RejectsForgedPubKey ensures that an address-only
// schedule entry still binds the caller-provided pubkey to the scheduled address.
// Without this check, a forger could claim Signer.Address = scheduled_addr with an
// arbitrary Signer.PubKey and later pass signature validation that trusts that pubkey.
func TestValidateProposer_AddressOnly_RejectsForgedPubKey(t *testing.T) {
	scheduled, _ := makeProposerScheduleEntry(t, 1)
	_, attackerPub := makeProposerScheduleEntry(t, 1)

	scheduled.PubKey = nil // address-only entry

	genesis := Genesis{
		ChainID:                "c",
		StartTime:              testGenesisStartTime,
		InitialHeight:          1,
		ProposerSchedule:       []ProposerScheduleEntry{scheduled},
		DAEpochForcedInclusion: 1,
	}

	// Scheduled address paired with a different pubkey must be rejected.
	err := genesis.ValidateProposer(1, scheduled.Address, attackerPub)
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not match scheduled address")
}

func TestValidateProposer_UsesActiveEntryAtHeight(t *testing.T) {
	entry1, pub1 := makeProposerScheduleEntry(t, 1)
	entry2, pub2 := makeProposerScheduleEntry(t, 10)
	genesis := Genesis{
		ChainID:                "c",
		StartTime:              testGenesisStartTime,
		InitialHeight:          1,
		ProposerSchedule:       []ProposerScheduleEntry{entry1, entry2},
		DAEpochForcedInclusion: 1,
	}

	// entry2 signer trying to sign height within entry1's active range must fail.
	require.Error(t, genesis.ValidateProposer(9, entry2.Address, pub2))
	// entry1 signer trying to sign height within entry2's active range must fail.
	require.Error(t, genesis.ValidateProposer(10, entry1.Address, pub1))
}

func TestHasScheduledProposer(t *testing.T) {
	entry1, _ := makeProposerScheduleEntry(t, 1)
	entry2, _ := makeProposerScheduleEntry(t, 10)
	unknown, _ := makeProposerScheduleEntry(t, 99)

	explicit := Genesis{
		ChainID:                "c",
		StartTime:              testGenesisStartTime,
		InitialHeight:          1,
		ProposerSchedule:       []ProposerScheduleEntry{entry1, entry2},
		DAEpochForcedInclusion: 1,
	}
	require.True(t, explicit.HasScheduledProposer(entry1.Address))
	require.True(t, explicit.HasScheduledProposer(entry2.Address))
	require.False(t, explicit.HasScheduledProposer(unknown.Address))

	legacy := Genesis{
		ChainID:                "c",
		StartTime:              testGenesisStartTime,
		InitialHeight:          1,
		ProposerAddress:        entry1.Address,
		DAEpochForcedInclusion: 1,
	}
	require.True(t, legacy.HasScheduledProposer(entry1.Address))
	require.False(t, legacy.HasScheduledProposer(entry2.Address))

	empty := Genesis{ChainID: "c", InitialHeight: 1}
	require.False(t, empty.HasScheduledProposer(entry1.Address))
}

func TestEffectiveProposerSchedule_ExplicitScheduleIsDeepCopy(t *testing.T) {
	entry1, _ := makeProposerScheduleEntry(t, 1)
	entry2, _ := makeProposerScheduleEntry(t, 10)
	origAddr := bytes.Clone(entry1.Address)
	origPub := bytes.Clone(entry1.PubKey)

	genesis := Genesis{
		ChainID:                "c",
		StartTime:              testGenesisStartTime,
		InitialHeight:          1,
		ProposerSchedule:       []ProposerScheduleEntry{entry1, entry2},
		DAEpochForcedInclusion: 1,
	}

	// Mutating returned byte slices must not corrupt the genesis-backed data.
	got := genesis.EffectiveProposerSchedule()
	got[0].Address[0] ^= 0xFF
	got[0].PubKey[0] ^= 0xFF

	require.Equal(t, origAddr, genesis.ProposerSchedule[0].Address)
	require.Equal(t, origPub, genesis.ProposerSchedule[0].PubKey)
}

func TestEffectiveProposerSchedule_LegacyFallback(t *testing.T) {
	addr := []byte("some-address-bytes")
	origAddr := bytes.Clone(addr)
	legacy := Genesis{
		ChainID:         "c",
		InitialHeight:   7,
		ProposerAddress: addr,
	}
	schedule := legacy.EffectiveProposerSchedule()
	require.Len(t, schedule, 1)
	require.Equal(t, uint64(7), schedule[0].StartHeight)
	require.Equal(t, addr, schedule[0].Address)
	require.Empty(t, schedule[0].PubKey)

	// mutating the derived slice must not affect the genesis backing data.
	schedule[0].Address[0] ^= 0xFF
	require.Equal(t, origAddr, legacy.ProposerAddress)
}

func TestEffectiveProposerSchedule_Empty(t *testing.T) {
	require.Nil(t, Genesis{}.EffectiveProposerSchedule())
}

func TestInitialProposerAddress_EmptyGenesisReturnsNil(t *testing.T) {
	require.Nil(t, Genesis{InitialHeight: 1}.InitialProposerAddress())
}

// TestProposerKeyAddressMatchesSignerGetAddress pins the invariant that the
// genesis-side address derivation matches the signer implementations. If a
// signer ever changes its address formula this test will fail and flag the
// break instead of silently producing rejected blocks after a key rotation.
func TestProposerKeyAddressMatchesSignerGetAddress(t *testing.T) {
	priv, pub, err := crypto.GenerateEd25519Key(rand.Reader)
	require.NoError(t, err)

	s, err := noop.NewNoopSigner(priv)
	require.NoError(t, err)

	signerAddr, err := s.GetAddress()
	require.NoError(t, err)

	genesisAddr := proposerKeyAddress(pub)
	require.Equal(t, signerAddr, genesisAddr)

	entry, err := NewProposerScheduleEntry(1, pub)
	require.NoError(t, err)
	require.Equal(t, signerAddr, entry.Address)
}

func TestLoadGenesisNormalizesLegacyProposerAddressFromSchedule(t *testing.T) {
	entry1, _ := makeProposerScheduleEntry(t, 1)
	entry2, _ := makeProposerScheduleEntry(t, 50)

	rawGenesis := Genesis{
		ChainID:                "test-chain",
		StartTime:              testGenesisStartTime,
		InitialHeight:          1,
		ProposerSchedule:       []ProposerScheduleEntry{entry1, entry2},
		DAEpochForcedInclusion: 1,
	}

	genesisPath := filepath.Join(t.TempDir(), "genesis.json")
	genesisJSON, err := json.Marshal(rawGenesis)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(genesisPath, genesisJSON, 0o600))

	loaded, err := LoadGenesis(genesisPath)
	require.NoError(t, err)
	require.Equal(t, entry1.Address, loaded.ProposerAddress)
	require.Equal(t, rawGenesis.ProposerSchedule, loaded.ProposerSchedule)
}
