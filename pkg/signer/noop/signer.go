package noop

import (
	"github.com/libp2p/go-libp2p/core/crypto"

	"github.com/rollkit/rollkit/pkg/signer"
)

// NoopSigner implements the remote_signer.Signer interface.
// It generates a new Ed25519 key pair for each instance.
type NoopSigner struct {
	privKey crypto.PrivKey
	pubKey  crypto.PubKey
}

// NewNoopSigner creates a new signer with a fresh Ed25519 key pair.
func NewNoopSigner(privKey crypto.PrivKey) (signer.Signer, error) {

	return &NoopSigner{
		privKey: privKey,
		pubKey:  privKey.GetPublic(),
	}, nil
}

// Sign implements the Signer interface by signing the message with the Ed25519 private key.
func (n *NoopSigner) Sign(message []byte) ([]byte, error) {
	return n.privKey.Sign(message)
}

// GetPublic implements the Signer interface by returning the Ed25519 public key.
func (n *NoopSigner) GetPublic() (crypto.PubKey, error) {
	return n.pubKey, nil
}
