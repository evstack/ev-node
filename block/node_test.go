package block

import (
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/types"
)

func TestNodeAPI(t *testing.T) {
	logger := zerolog.Nop()

	// Test that the node API compiles and basic structure works
	t.Run("Node interface methods compile", func(t *testing.T) {
		// Create a minimal config
		cfg := config.Config{
			Node: config.NodeConfig{
				BlockTime: config.DurationWrapper{Duration: time.Second},
			},
			DA: config.DAConfig{
				BlockTime: config.DurationWrapper{Duration: time.Second},
			},
		}

		// Create genesis
		gen := genesis.Genesis{
			ChainID:         "test-chain",
			InitialHeight:   1,
			StartTime:       time.Now(),
			ProposerAddress: []byte("test-proposer"),
		}

		// Test that Dependencies struct compiles
		deps := Dependencies{
			Store:             nil, // Will be nil for compilation test
			Executor:          nil,
			Sequencer:         nil,
			DA:                nil,
			HeaderStore:       nil,
			DataStore:         nil,
			HeaderBroadcaster: &mockBroadcaster[*types.SignedHeader]{},
			DataBroadcaster:   &mockBroadcaster[*types.Data]{},
			Signer:            nil,
		}

		// Test that NewFullNodeComponents requires signer (just check the error without panicking)
		_, err := NewFullNodeComponents(cfg, gen, deps, logger)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "signer is required for full nodes")

		// Test that the function signatures compile - don't actually call with nil deps
		// Just verify the API exists and compiles
		var components *BlockComponents
		assert.Nil(t, components) // Just a compilation check

		// Test dependencies structure
		assert.NotNil(t, deps.HeaderBroadcaster)
		assert.NotNil(t, deps.DataBroadcaster)
	})

	t.Run("BlockOptions compiles", func(t *testing.T) {
		opts := common.DefaultBlockOptions()
		assert.NotNil(t, opts.AggregatorNodeSignatureBytesProvider)
		assert.NotNil(t, opts.SyncNodeSignatureBytesProvider)
		assert.NotNil(t, opts.ValidatorHasherProvider)
	})
}

func TestBlockComponents(t *testing.T) {
	// Test that BlockComponents struct compiles and works
	var components *BlockComponents
	assert.Nil(t, components)

	// Test that we can create the struct
	components = &BlockComponents{
		Executor: nil,
		Syncer:   nil,
		Cache:    nil,
	}
	assert.NotNil(t, components)

	// Test GetLastState with nil components returns empty state
	state := components.GetLastState()
	assert.Equal(t, types.State{}, state)
}
