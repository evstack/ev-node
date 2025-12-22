package syncing

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/types"
)

func TestDARetriever_StrictEnvelopeMode_Switch(t *testing.T) {
	// Setup keys
	addr, pub, signer := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}

	r := newTestDARetriever(t, nil, config.DefaultConfig(), gen)

	// 1. Create a Legacy Header (SignedHeader marshaled directly)
	// This simulates old blobs on the network before the upgrade.
	legacyHeader := &types.SignedHeader{
		Header: types.Header{
			BaseHeader:      types.BaseHeader{ChainID: gen.ChainID, Height: 1, Time: uint64(time.Now().UnixNano())},
			ProposerAddress: addr,
		},
		Signer: types.Signer{PubKey: pub, Address: addr},
	}
	// Sign it
	bz, err := types.DefaultAggregatorNodeSignatureBytesProvider(&legacyHeader.Header)
	require.NoError(t, err)
	sig, err := signer.Sign(bz)
	require.NoError(t, err)
	legacyHeader.Signature = sig

	legacyBlob, err := legacyHeader.MarshalBinary()
	require.NoError(t, err)

	// 2. Create an Envelope Header (DAHeaderEnvelope)
	// This simulates a new blob after upgrade.
	envelopeHeader := &types.SignedHeader{
		Header: types.Header{
			BaseHeader:      types.BaseHeader{ChainID: gen.ChainID, Height: 2, Time: uint64(time.Now().UnixNano())},
			ProposerAddress: addr,
		},
		Signer: types.Signer{PubKey: pub, Address: addr},
	}
	// Sign content
	bz2, err := types.DefaultAggregatorNodeSignatureBytesProvider(&envelopeHeader.Header)
	require.NoError(t, err)
	sig2, err := signer.Sign(bz2)
	require.NoError(t, err)
	envelopeHeader.Signature = sig2

	// Create Envelope
	// We need to sign the envelope itself.
	// The `SubmitHeaders` logic wraps it. We emulate it here using `MarshalDAEnvelope`.
	// First get canonical content bytes (fields 1-3)
	contentBytes, err := envelopeHeader.MarshalBinary()
	require.NoError(t, err)
	// Sign envelope
	envSig, err := signer.Sign(contentBytes)
	require.NoError(t, err)
	// Marshal to envelope
	envelopeBlob, err := envelopeHeader.MarshalDAEnvelope(envSig)
	require.NoError(t, err)

	// --- Test Scenario ---

	// A. Initial State: StrictMode is false. Legacy blob should be accepted.
	assert.False(t, r.strictMode)

	decodedLegacy := r.tryDecodeHeader(legacyBlob, 100)
	require.NotNil(t, decodedLegacy)
	assert.Equal(t, uint64(1), decodedLegacy.Height())

	// StrictMode should still be false because it was a legacy blob
	assert.False(t, r.strictMode)

	// B. Receiving Envelope: Should be accepted and Switch StrictMode to true.
	decodedEnvelope := r.tryDecodeHeader(envelopeBlob, 101)
	require.NotNil(t, decodedEnvelope)
	assert.Equal(t, uint64(2), decodedEnvelope.Height())

	assert.True(t, r.strictMode, "retriever should have switched to strict mode")

	// C. Receiving Legacy again: Should be REJECTED now.
	// We reuse the same legacyBlob (or a new one, doesn't matter, structure is legacy).
	decodedLegacyAgain := r.tryDecodeHeader(legacyBlob, 102)
	assert.Nil(t, decodedLegacyAgain, "legacy blob should be rejected in strict mode")

	// D. Receiving Envelope again: Should still be accepted.
	decodedEnvelopeAgain := r.tryDecodeHeader(envelopeBlob, 103)
	require.NotNil(t, decodedEnvelopeAgain)
}
