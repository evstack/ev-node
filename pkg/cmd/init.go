package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	rollconf "github.com/evstack/ev-node/pkg/config"
	rollconf "github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/hash"
	"github.com/evstack/ev-node/pkg/p2p/key"
	"github.com/evstack/ev-node/pkg/signer/factory"
)

// CreateSigner sets up the signer configuration and creates necessary files
func CreateSigner(config *rollconf.Config, homePath string, passphrase string) ([]byte, error) {
	if !config.Node.Aggregator {
		return nil, nil
	}

	if config.Signer.SignerType == "file" {
		signerDir := filepath.Join(homePath, "config")
		config.Signer.SignerPath = signerDir
	}

	signer, err := factory.NewSigner(context.Background(), config, passphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize signer via factory: %w", err)
	}

	pubKey, err := signer.GetPublic()
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	bz, err := pubKey.Raw()
	if err != nil {
		return nil, fmt.Errorf("failed to get public key raw bytes: %w", err)
	}

	proposerAddress := hash.SumTruncated(bz)
	return proposerAddress, nil
}

// LoadOrGenNodeKey creates the node key file if it doesn't exist.
func LoadOrGenNodeKey(homePath string) error {
	nodeKeyFile := filepath.Join(homePath, "config")

	_, err := key.LoadOrGenNodeKey(nodeKeyFile)
	if err != nil {
		return fmt.Errorf("failed to create node key: %w", err)
	}

	return nil
}
