package executing

import (
	"crypto/sha256"
	"testing"
	"time"

	ds "github.com/ipfs/go-datastore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/types"
)

// buildHeaderAndData constructs a SignedHeader and Data at the given height.
// An empty sig yields the legacy pre-upgrade pending marker; a non-empty sig
// yields a signed block, which migration must refuse to treat as pending.
func buildHeaderAndData(
	t *testing.T,
	fx executorTestFixture,
	height uint64,
	sig types.Signature,
) (*types.SignedHeader, *types.Data) {
	t.Helper()

	state := fx.Exec.getLastState()
	gen := fx.Exec.genesis

	pubKey, err := fx.Exec.signer.GetPublic()
	require.NoError(t, err)
	validatorHash, err := common.DefaultBlockOptions().ValidatorHasherProvider(gen.ProposerAddress, pubKey)
	require.NoError(t, err)

	header := &types.SignedHeader{
		Header: types.Header{
			Version: types.Version{
				Block: state.Version.Block,
				App:   state.Version.App,
			},
			BaseHeader: types.BaseHeader{
				ChainID: gen.ChainID,
				Height:  height,
				Time:    uint64(time.Now().UnixNano()),
			},
			AppHash:         state.AppHash,
			ProposerAddress: gen.ProposerAddress,
			ValidatorHash:   validatorHash,
		},
		Signature: sig,
		Signer: types.Signer{
			PubKey:  pubKey,
			Address: gen.ProposerAddress,
		},
	}

	data := &types.Data{
		Txs: []types.Tx{[]byte("legacy_tx")},
		Metadata: &types.Metadata{
			ChainID: header.ChainID(),
			Height:  header.Height(),
			Time:    header.BaseHeader.Time,
		},
	}
	header.DataHash = data.DACommitment()

	return header, data
}

// saveLegacyBlock builds a block and stores it at the given height in the
// pre-upgrade format (unsigned = pending, signed = reject on migration).
func saveLegacyBlock(t *testing.T, fx executorTestFixture, height uint64, sig types.Signature) (*types.SignedHeader, *types.Data) {
	t.Helper()
	h, d := buildHeaderAndData(t, fx, height, sig)
	batch, err := fx.MemStore.NewBatch(t.Context())
	require.NoError(t, err)
	require.NoError(t, batch.SaveBlockData(h, d, &sig))
	require.NoError(t, batch.Commit())
	return h, d
}

func TestMigrateLegacyPendingBlock(t *testing.T) {
	t.Run("migrates unsigned legacy pending block", func(t *testing.T) {
		fx := setupTestExecutor(t, 1000)
		defer fx.Cancel()

		// Empty signature indicates legacy pending block
		header, data := saveLegacyBlock(t, fx, 1, types.Signature{})

		_, _, err := fx.MemStore.GetBlockData(t.Context(), 1)
		require.NoError(t, err, "precondition: legacy block should exist before migration")

		require.NoError(t, fx.Exec.migrateLegacyPendingBlock(fx.Exec.ctx))

		gotHeader, gotData, err := fx.Exec.getPendingBlock(fx.Exec.ctx)
		require.NoError(t, err)
		require.NotNil(t, gotHeader)
		require.NotNil(t, gotData)
		assert.Equal(t, uint64(1), gotHeader.Height())
		assert.Equal(t, data.DACommitment(), gotHeader.DataHash)
		assert.Equal(t, len(data.Txs), len(gotData.Txs))

		_, _, err = fx.MemStore.GetBlockData(t.Context(), 1)
		assert.ErrorIs(t, err, ds.ErrNotFound, "legacy header/data keys should be deleted after migration")

		_, err = fx.MemStore.GetSignature(t.Context(), 1)
		assert.ErrorIs(t, err, ds.ErrNotFound, "legacy signature key should be deleted after migration")

		headerBlob, err := header.MarshalBinary()
		require.NoError(t, err)
		hash := sha256.Sum256(headerBlob)
		_, _, err = fx.MemStore.GetBlockByHash(t.Context(), hash[:])
		assert.ErrorIs(t, err, ds.ErrNotFound, "legacy index key should be deleted after migration")
	})

	t.Run("no-op when no candidate block exists", func(t *testing.T) {
		fx := setupTestExecutor(t, 1000)
		defer fx.Cancel()

		// Ensure migrateLegacyPendingBlock does not error when no legacy pending block exists.
		require.NoError(t, fx.Exec.migrateLegacyPendingBlock(fx.Exec.ctx))

		// Assert pending block is empty.
		header, data, err := fx.Exec.getPendingBlock(fx.Exec.ctx)
		require.NoError(t, err)
		assert.Nil(t, header)
		assert.Nil(t, data)
	})

	t.Run("errors when candidate block has signature", func(t *testing.T) {
		fx := setupTestExecutor(t, 1000)
		defer fx.Cancel()

		// Non-empty signature indicates a signed block, not a legacy pending block (indicates some kind of corruption).
		sig := types.Signature([]byte("not-empty"))
		saveLegacyBlock(t, fx, 1, sig)

		// migrateLegacyPendingBlock should error when it finds a legacy block with a signature,
		// rather than silently migrate a signed block and risk losing/duplicating it.
		err := fx.Exec.migrateLegacyPendingBlock(fx.Exec.ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "pending block with signatures found")

		// Assert block was not migrated (i.e. still exists at legacy location and not at pending keys).
		_, _, err = fx.MemStore.GetBlockData(t.Context(), 1)
		assert.NoError(t, err)

		// Assert pending keys are still empty.
		_, err = fx.MemStore.GetMetadata(t.Context(), headerKey)
		assert.ErrorIs(t, err, ds.ErrNotFound)
		_, err = fx.MemStore.GetMetadata(t.Context(), dataKey)
		assert.ErrorIs(t, err, ds.ErrNotFound)
	})
}

func TestGetPendingBlock_CorruptState(t *testing.T) {
	fx := setupTestExecutor(t, 1000)
	defer fx.Cancel()

	// Corrupt the pending block state by writing a header key without corresponding data.
	require.NoError(t, fx.MemStore.SetMetadata(t.Context(), headerKey, []byte("whatever")))

	// Should error as only a valid header/data pair or empty state are expected.
	_, _, err := fx.Exec.getPendingBlock(fx.Exec.ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "corrupt state")
}

func TestGetPendingBlock_Empty(t *testing.T) {
	fx := setupTestExecutor(t, 1000)
	defer fx.Cancel()

	// Empty state should return nil header/data without error.
	h, d, err := fx.Exec.getPendingBlock(fx.Exec.ctx)
	require.NoError(t, err)
	assert.Nil(t, h)
	assert.Nil(t, d)
}

func TestDeletePendingBlock(t *testing.T) {
	fx := setupTestExecutor(t, 1000)
	defer fx.Cancel()

	// Save a pending block directly to the store.
	header, data := buildHeaderAndData(t, fx, 1, types.Signature{})
	require.NoError(t, fx.Exec.savePendingBlock(fx.Exec.ctx, header, data))

	// Assert block is retrievable before deletion.
	gotHeader, gotData, err := fx.Exec.getPendingBlock(fx.Exec.ctx)
	require.NoError(t, err, "precondition: pending block should be readable")
	require.NotNil(t, gotHeader, "precondition: stored header should not be nil")
	require.NotNil(t, gotData, "precondition: stored data should not be nil")
	assert.Equal(t, header.Height(), gotHeader.Height(), "stored header height mismatch")
	assert.Equal(t, header.ChainID(), gotHeader.ChainID(), "stored header chain id mismatch")
	assert.Equal(t, header.DataHash, gotHeader.DataHash, "stored header data hash mismatch")
	assert.Equal(t, len(data.Txs), len(gotData.Txs), "stored data tx count mismatch")
	assert.Equal(t, data.DACommitment(), gotData.DACommitment(), "stored data DA commitment mismatch")

	// Delete the pending block.
	require.NoError(t, fx.Exec.deletePendingBlock(fx.Exec.ctx))

	// Assert pending block is no longer retrievable.
	_, err = fx.MemStore.GetMetadata(t.Context(), headerKey)
	assert.ErrorIs(t, err, ds.ErrNotFound)
	_, err = fx.MemStore.GetMetadata(t.Context(), dataKey)
	assert.ErrorIs(t, err, ds.ErrNotFound)
}
