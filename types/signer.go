package types

import (
	"crypto/sha256"

	"github.com/libp2p/go-libp2p/core/crypto"
)

// Signer is a type that can verify messages.
type Signer struct {
	PubKey  crypto.PubKey
	Address []byte
}

// NewSigner creates a new signer from a public key.
func NewSigner(pubKey crypto.PubKey) (Signer, error) {
	bz, err := pubKey.Raw()
	if err != nil {
		return Signer{}, err
	}

	address := sha256.Sum256(bz)
	return Signer{
		PubKey:  pubKey,
		Address: address[:],
	}, nil
}

// Verify verifies a vote with a signature.
func (s *Signer) Verify(vote []byte, signature []byte) (bool, error) {
	return s.PubKey.Verify(vote, signature)
}
