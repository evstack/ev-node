package genesis

import (
	"crypto/rand"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/stretchr/testify/require"
)

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
		StartTime:              time.Now().UTC(),
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
		StartTime:              time.Now().UTC(),
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
		StartTime:              time.Now().UTC(),
		InitialHeight:          1,
		ProposerSchedule:       []ProposerScheduleEntry{entry1, entry2},
		DAEpochForcedInclusion: 1,
	}

	require.NoError(t, genesis.Validate())
	require.NoError(t, genesis.ValidateProposer(1, entry1.Address, pubKey1))
	require.NoError(t, genesis.ValidateProposer(21, entry2.Address, pubKey2))
}

func TestLoadGenesisNormalizesLegacyProposerAddressFromSchedule(t *testing.T) {
	entry1, _ := makeProposerScheduleEntry(t, 1)
	entry2, _ := makeProposerScheduleEntry(t, 50)

	rawGenesis := Genesis{
		ChainID:                "test-chain",
		StartTime:              time.Now().UTC(),
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
