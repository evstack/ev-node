package types

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/celestiaorg/go-header"
)

var (
	// ErrLastHeaderHashMismatch is returned when the last header hash doesn't match.
	ErrLastHeaderHashMismatch = errors.New("last header hash mismatch")
)

var _ header.Header[*SignedHeader] = &SignedHeader{}

// SignedHeader combines Header and its signature.
//
// Used mostly for gossiping.
type SignedHeader struct {
	Header
	// Note: This is backwards compatible as ABCI exported types are not affected.
	Signature Signature
	Signer    Signer

	aggregatorSignatureProvider    AggregatorNodeSignatureBytesProvider
	syncNodeSignatureBytesProvider SyncNodeSignatureBytesProvider
}

// New creates a new SignedHeader.
func (sh *SignedHeader) New() *SignedHeader {
	return new(SignedHeader)
}

// IsZero returns true if the SignedHeader is nil
func (sh *SignedHeader) IsZero() bool {
	return sh == nil
}

// SetCustomVerifierForAggregator sets a custom signature provider for the SignedHeader.
// If set, ValidateBasic will use this function to verify the signature.
func (sh *SignedHeader) SetCustomVerifierForAggregator(provider AggregatorNodeSignatureBytesProvider) {
	sh.aggregatorSignatureProvider = provider
}

// SetCustomVerifierForSyncNode sets a custom signature provider for the SignedHeader.
// If set, ValidateBasic will use this function to verify the signature.
func (sh *SignedHeader) SetCustomVerifierForSyncNode(provider SyncNodeSignatureBytesProvider) {
	sh.syncNodeSignatureBytesProvider = provider
}

// Verify verifies the signed header.
func (sh *SignedHeader) Verify(untrstH *SignedHeader) error {
	// go-header ensures untrustH already passed ValidateBasic.
	if err := sh.Header.Verify(&untrstH.Header); err != nil {
		return &header.VerifyError{
			Reason: err,
		}
	}

	if sh.isAdjacent(untrstH) {
		if err := sh.verifyHeaderHash(untrstH); err != nil {
			return err
		}
	}

	return nil
}

// verifyHeaderHash verifies the header hash.
func (sh *SignedHeader) verifyHeaderHash(untrstH *SignedHeader) error {
	hash := sh.Hash()
	if !bytes.Equal(hash, untrstH.LastHeader()) {
		return sh.newVerifyError(ErrLastHeaderHashMismatch, hash, untrstH.LastHeader())
	}
	return nil
}

// isAdjacent checks if the height of headers are adjacent.
func (sh *SignedHeader) isAdjacent(untrstH *SignedHeader) bool {
	return sh.Height()+1 == untrstH.Height()
}

// newVerifyError creates and returns a new error verification.
func (sh *SignedHeader) newVerifyError(err error, expected, got []byte) *header.VerifyError {
	return &header.VerifyError{
		Reason: fmt.Errorf("verification error at height %d: %w: expected %X, but got %X", sh.Height(), err, expected, got),
	}
}

var (
	// ErrAggregatorSetHashMismatch is returned when the aggregator set hash
	// in the signed header doesn't match the hash of the validator set.
	ErrAggregatorSetHashMismatch = errors.New("aggregator set hash in signed header and hash of validator set do not match")

	// ErrSignatureVerificationFailed is returned when the signature
	// verification fails
	ErrSignatureVerificationFailed = errors.New("signature verification failed")

	// ErrProposerAddressMismatch is returned when the proposer address in the signed header does not match the proposer address in the validator set
	ErrProposerAddressMismatch = errors.New("proposer address in SignedHeader does not match the proposer address in the validator set")

	// ErrSignatureEmpty is returned when signature is empty
	ErrSignatureEmpty = errors.New("signature is empty")
)

// Validate performs basic validation of a signed header for the aggregator node.
func (sh *SignedHeader) ValidateBasic() error {
	if err := sh.Header.ValidateBasic(); err != nil {
		return err
	}

	if err := sh.Signature.ValidateBasic(); err != nil {
		return err
	}

	// Check that the proposer address in the signed header matches the proposer address in the validator set
	if !bytes.Equal(sh.ProposerAddress, sh.Signer.Address) {
		return ErrProposerAddressMismatch
	}

	// Track tried payloads using a slice since we have at most 3 attempts.
	// This avoids allocating strings for map keys.
	tried := make([][]byte, 0, 3)
	tryPayload := func(payload []byte) (bool, error) {
		if len(payload) == 0 {
			return false, nil
		}

		// Check if we've already tried this payload
		for _, p := range tried {
			if bytes.Equal(p, payload) {
				return false, nil
			}
		}
		tried = append(tried, payload)

		verified, err := sh.Signer.PubKey.Verify(payload, sh.Signature)
		if err != nil {
			return false, err
		}
		return verified, nil
	}

	if sh.aggregatorSignatureProvider != nil {
		bz, err := sh.aggregatorSignatureProvider(&sh.Header)
		if err != nil {
			return fmt.Errorf("custom signature payload provider failed: %w", err)
		}
		ok, err := tryPayload(bz)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
	}

	slim, err := sh.Header.MarshalBinary()
	if err != nil {
		return err
	}
	ok, err := tryPayload(slim)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}

	legacy, err := sh.MarshalBinaryLegacy()
	if err != nil {
		return err
	}
	ok, err = tryPayload(legacy)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}

	return ErrSignatureVerificationFailed
}

// ValidateBasicWithData performs basic validation of a signed header, given data for syncing node.
func (sh *SignedHeader) ValidateBasicWithData(data *Data) error {
	if err := sh.Header.ValidateBasic(); err != nil {
		return err
	}

	if err := sh.Signature.ValidateBasic(); err != nil {
		return err
	}

	// Check that the proposer address in the signed header matches the proposer address in the validator set
	if !bytes.Equal(sh.ProposerAddress, sh.Signer.Address) {
		return ErrProposerAddressMismatch
	}

	// Track tried payloads using a slice since we have at most 3 attempts.
	// This avoids allocating strings for map keys.
	tried := make([][]byte, 0, 3)
	tryPayload := func(payload []byte) (bool, error) {
		if len(payload) == 0 {
			return false, nil
		}

		// Check if we've already tried this payload
		for _, p := range tried {
			if bytes.Equal(p, payload) {
				return false, nil
			}
		}
		tried = append(tried, payload)

		verified, err := sh.Signer.PubKey.Verify(payload, sh.Signature)
		if err != nil {
			return false, err
		}
		return verified, nil
	}

	if sh.syncNodeSignatureBytesProvider != nil {
		bz, err := sh.syncNodeSignatureBytesProvider(context.Background(), &sh.Header, data)
		if err != nil {
			return fmt.Errorf("custom signature payload provider failed: %w", err)
		}
		ok, err := tryPayload(bz)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
	}

	slim, err := sh.Header.MarshalBinary()
	if err != nil {
		return err
	}
	ok, err := tryPayload(slim)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}

	legacy, err := sh.MarshalBinaryLegacy()
	if err != nil {
		return err
	}
	ok, err = tryPayload(legacy)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}

	return ErrSignatureVerificationFailed
}
