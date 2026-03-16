package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	rollconf "github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/hash"
	"github.com/evstack/ev-node/pkg/p2p/key"
	awssigner "github.com/evstack/ev-node/pkg/signer/aws"
	"github.com/evstack/ev-node/pkg/signer/file"
)

// CreateSigner sets up the signer configuration and creates necessary files
func CreateSigner(config *rollconf.Config, homePath string, passphrase string) ([]byte, error) {
	if !config.Node.Aggregator {
		return nil, nil
	}

	switch config.Signer.SignerType {
	case "file":
		if passphrase == "" {
			return nil, fmt.Errorf("passphrase is required when using local file signer")
		}

		signerDir := filepath.Join(homePath, "config")
		if err := os.MkdirAll(signerDir, 0o750); err != nil {
			return nil, fmt.Errorf("failed to create signer directory: %w", err)
		}

		config.Signer.SignerPath = signerDir

		signer, err := file.CreateFileSystemSigner(config.Signer.SignerPath, []byte(passphrase))
		if err != nil {
			return nil, fmt.Errorf("failed to initialize signer: %w", err)
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

	case "awskms":
		// For KMS, the key is pre-provisioned in AWS. We fetch the public key
		// to derive the proposer address.
		signer, err := awssigner.NewKmsSigner(context.Background(), config.Signer.KmsRegion, config.Signer.KmsKeyID)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize AWS KMS signer: %w", err)
		}

		pubKey, err := signer.GetPublic()
		if err != nil {
			return nil, fmt.Errorf("failed to get public key from KMS: %w", err)
		}

		bz, err := pubKey.Raw()
		if err != nil {
			return nil, fmt.Errorf("failed to get public key raw bytes: %w", err)
		}

		proposerAddress := hash.SumTruncated(bz)
		return proposerAddress, nil

	default:
		return nil, fmt.Errorf("unknown signer type: %s", config.Signer.SignerType)
	}
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
