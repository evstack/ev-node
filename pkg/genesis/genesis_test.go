package genesis

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGenesis(t *testing.T) {
	// Test valid genesis creation
	validTime := time.Now()
	proposerAddress := []byte("proposer")

	genesis := NewGenesis(
		"test-chain",
		1,
		validTime,
		proposerAddress,
	)

	assert.Equal(t, "test-chain", genesis.ChainID)
	assert.Equal(t, uint64(1), genesis.InitialHeight)
	assert.Equal(t, validTime, genesis.StartTime)
	assert.Equal(t, proposerAddress, genesis.ProposerAddress)

	// Test that NewGenesis validates and panics on invalid input
	genesis = NewGenesis(
		"", // Empty chain ID should cause panic
		1,
		validTime,
		proposerAddress,
	)
	err := genesis.Validate()
	assert.Error(t, err)

	genesis = NewGenesis(
		"test-chain",
		0, // Zero initial height should cause panic
		validTime,
		proposerAddress,
	)
	err = genesis.Validate()
	assert.Error(t, err)

	genesis = NewGenesis(
		"test-chain",
		1,
		time.Time{}, // Zero time should cause panic
		proposerAddress,
	)
	err = genesis.Validate()
	assert.Error(t, err)

	genesis = NewGenesis(
		"test-chain",
		1,
		validTime,
		nil, // Nil proposer address should cause panic
	)
	err = genesis.Validate()
	assert.Error(t, err)
}

func TestGenesis_Validate(t *testing.T) {
	validTime := time.Now()
	tests := []struct {
		name    string
		genesis Genesis
		wantErr bool
	}{
		{
			name: "valid genesis - chain ID can contain any character",
			genesis: Genesis{
				ChainID:                "test@chain#123!",
				StartTime:              validTime,
				InitialHeight:          1,
				ProposerAddress:        []byte("proposer"),
				DAEpochForcedInclusion: 1,
			},
			wantErr: false,
		},
		{
			name: "invalid - empty chain_id",
			genesis: Genesis{
				ChainID:                "",
				StartTime:              validTime,
				InitialHeight:          1,
				ProposerAddress:        []byte("proposer"),
				DAEpochForcedInclusion: 1,
			},
			wantErr: true,
		},
		{
			name: "invalid - zero initial height",
			genesis: Genesis{
				ChainID:                "test-chain",
				StartTime:              validTime,
				InitialHeight:          0,
				ProposerAddress:        []byte("proposer"),
				DAEpochForcedInclusion: 1,
			},
			wantErr: true,
		},
		{
			name: "invalid - zero time DA start height",
			genesis: Genesis{
				ChainID:                "test-chain",
				StartTime:              time.Time{},
				InitialHeight:          1,
				ProposerAddress:        []byte("proposer"),
				DAEpochForcedInclusion: 1,
			},
			wantErr: true,
		},
		{
			name: "invalid - nil proposer address",
			genesis: Genesis{
				ChainID:                "test-chain",
				StartTime:              validTime,
				InitialHeight:          1,
				ProposerAddress:        nil,
				DAEpochForcedInclusion: 1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.genesis.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Genesis.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGenesis_ValidateProposerSchedule(t *testing.T) {
	validTime := time.Unix(1_700_000_000, 0).UTC()

	newEntry := func(startHeight uint64) (ProposerScheduleEntry, crypto.PubKey) {
		_, pub, err := crypto.GenerateEd25519Key(rand.Reader)
		require.NoError(t, err)
		entry, err := NewProposerScheduleEntry(startHeight, pub)
		require.NoError(t, err)
		return entry, pub
	}

	entry1, _ := newEntry(1)
	entry10, _ := newEntry(10)
	entry20, _ := newEntry(20)

	tests := []struct {
		name    string
		mutate  func() Genesis
		wantErr string
	}{
		{
			name: "valid - schedule without proposer_address",
			mutate: func() Genesis {
				return Genesis{
					ChainID:                "c",
					StartTime:              validTime,
					InitialHeight:          1,
					ProposerSchedule:       []ProposerScheduleEntry{entry1, entry10},
					DAEpochForcedInclusion: 1,
				}
			},
		},
		{
			name: "valid - schedule with matching proposer_address",
			mutate: func() Genesis {
				return Genesis{
					ChainID:                "c",
					StartTime:              validTime,
					InitialHeight:          1,
					ProposerAddress:        entry1.Address,
					ProposerSchedule:       []ProposerScheduleEntry{entry1, entry10},
					DAEpochForcedInclusion: 1,
				}
			},
		},
		{
			name: "invalid - first entry start_height != initial_height",
			mutate: func() Genesis {
				return Genesis{
					ChainID:                "c",
					StartTime:              validTime,
					InitialHeight:          5,
					ProposerSchedule:       []ProposerScheduleEntry{entry10, entry20},
					DAEpochForcedInclusion: 1,
				}
			},
			wantErr: "start_height must equal initial_height",
		},
		{
			name: "invalid - first entry start_height below initial_height",
			mutate: func() Genesis {
				return Genesis{
					ChainID:                "c",
					StartTime:              validTime,
					InitialHeight:          5,
					ProposerSchedule:       []ProposerScheduleEntry{entry1, entry10},
					DAEpochForcedInclusion: 1,
				}
			},
			wantErr: "start_height must be >= initial_height",
		},
		{
			name: "invalid - non-increasing (equal start_heights)",
			mutate: func() Genesis {
				return Genesis{
					ChainID:                "c",
					StartTime:              validTime,
					InitialHeight:          1,
					ProposerSchedule:       []ProposerScheduleEntry{entry1, entry1},
					DAEpochForcedInclusion: 1,
				}
			},
			wantErr: "strictly increasing",
		},
		{
			name: "invalid - non-increasing (decreasing start_heights)",
			mutate: func() Genesis {
				return Genesis{
					ChainID:                "c",
					StartTime:              validTime,
					InitialHeight:          1,
					ProposerSchedule:       []ProposerScheduleEntry{entry1, entry20, entry10},
					DAEpochForcedInclusion: 1,
				}
			},
			wantErr: "strictly increasing",
		},
		{
			name: "invalid - entry address does not match pub_key",
			mutate: func() Genesis {
				tampered := entry10
				tampered.Address = append([]byte(nil), entry10.Address...)
				tampered.Address[0] ^= 0xFF
				return Genesis{
					ChainID:                "c",
					StartTime:              validTime,
					InitialHeight:          1,
					ProposerSchedule:       []ProposerScheduleEntry{entry1, tampered},
					DAEpochForcedInclusion: 1,
				}
			},
			wantErr: "address does not match pub_key",
		},
		{
			name: "invalid - proposer_address mismatches schedule[0].address",
			mutate: func() Genesis {
				return Genesis{
					ChainID:                "c",
					StartTime:              validTime,
					InitialHeight:          1,
					ProposerAddress:        entry10.Address,
					ProposerSchedule:       []ProposerScheduleEntry{entry1, entry10},
					DAEpochForcedInclusion: 1,
				}
			},
			wantErr: "proposer_address must match proposer_schedule[0].address",
		},
		{
			name: "invalid - empty address in entry",
			mutate: func() Genesis {
				empty := entry10
				empty.Address = nil
				return Genesis{
					ChainID:                "c",
					StartTime:              validTime,
					InitialHeight:          1,
					ProposerSchedule:       []ProposerScheduleEntry{entry1, empty},
					DAEpochForcedInclusion: 1,
				}
			},
			wantErr: "address cannot be empty",
		},
		{
			name: "invalid - malformed pub_key bytes",
			mutate: func() Genesis {
				bad := entry10
				bad.PubKey = []byte{0x00, 0x01, 0x02}
				return Genesis{
					ChainID:                "c",
					StartTime:              validTime,
					InitialHeight:          1,
					ProposerSchedule:       []ProposerScheduleEntry{entry1, bad},
					DAEpochForcedInclusion: 1,
				}
			},
			wantErr: "unmarshal proposer pub_key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.mutate().Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
