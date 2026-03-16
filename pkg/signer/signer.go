package signer

import (
	"context"

	"github.com/libp2p/go-libp2p/core/crypto"
)

// Signer is an interface for signing and verifying messages.
type Signer interface {
	// Sign takes a message as bytes and returns its signature.
	Sign(ctx context.Context, message []byte) ([]byte, error)

	// GetPublic returns the public key paired with this private key.
	GetPublic() (crypto.PubKey, error)

	// GetAddress returns the address of the signer.
	GetAddress() ([]byte, error)
}

