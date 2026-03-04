package types

import (
	"crypto/sha256"

	"github.com/libp2p/go-libp2p/core/crypto"
)

// Signer is a type that can verify messages.
type Signer struct {
	PubKey  crypto.PubKey
	Address []byte
	// marshalledPubKey caches the result of crypto.MarshalPublicKey to avoid
	// repeated serialization in SignedHeader.ToProto().
	marshalledPubKey []byte
}

// MarshalledPubKey returns the marshalled public key bytes, caching the result.
func (s *Signer) MarshalledPubKey() ([]byte, error) {
	if len(s.marshalledPubKey) > 0 {
		return s.marshalledPubKey, nil
	}
	if s.PubKey == nil {
		return nil, nil
	}
	bz, err := crypto.MarshalPublicKey(s.PubKey)
	if err != nil {
		return nil, err
	}
	s.marshalledPubKey = bz
	return bz, nil
}

// NewSigner creates a new signer from a public key.
func NewSigner(pubKey crypto.PubKey) (Signer, error) {
	bz, err := pubKey.Raw()
	if err != nil {
		return Signer{}, err
	}

	// Pre-cache marshalled pub key for later use in ToProto.
	marshalledPubKey, err := crypto.MarshalPublicKey(pubKey)
	if err != nil {
		return Signer{}, err
	}

	address := sha256.Sum256(bz)
	return Signer{
		PubKey:           pubKey,
		Address:          address[:],
		marshalledPubKey: marshalledPubKey,
	}, nil
}

// Verify verifies a vote with a signature.
func (s *Signer) Verify(vote []byte, signature []byte) (bool, error) {
	return s.PubKey.Verify(vote, signature)
}

func KeyAddress(pubKey crypto.PubKey) []byte {
	bz, err := pubKey.Raw()
	if err != nil {
		return nil
	}
	hash := sha256.Sum256(bz)
	return hash[:]
}
