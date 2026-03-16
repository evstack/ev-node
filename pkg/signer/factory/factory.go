package factory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	rollconf "github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/signer"
	awssigner "github.com/evstack/ev-node/pkg/signer/aws"
	"github.com/evstack/ev-node/pkg/signer/file"
)

// NewSigner creates a new Signer based on the configuration.
func NewSigner(ctx context.Context, config *rollconf.Config, passphrase string) (signer.Signer, error) {
	switch config.Signer.SignerType {
	case "file":
		if passphrase == "" {
			return nil, fmt.Errorf("passphrase is required when using local file signer")
		}

		// Resolve signer path; allow absolute, relative to node root, or relative to CWD if resolution fails
		signerPath, err := filepath.Abs(strings.TrimSuffix(config.Signer.SignerPath, "signer.json"))
		if err != nil {
			return nil, err
		}

		// Ensure directory exists for init command cases
		if err := os.MkdirAll(signerPath, 0o750); err != nil {
			return nil, fmt.Errorf("failed to create signer directory: %w", err)
		}

		// This will either create (if it doesn't exist) or load (if it does).
		// In a strictly decoupled factory we'd differentiate Create vs Load, but given the 
		// underlying CreateFileSystemSigner and LoadFileSystemSigner are typically idempotent 
		// in behavior if files are provided, let's check for existing files:
		signerFile := filepath.Join(signerPath, "signer.json")
		if _, err := os.Stat(signerFile); os.IsNotExist(err) {
			return file.CreateFileSystemSigner(signerPath, []byte(passphrase))
		}
		
		return file.LoadFileSystemSigner(signerPath, []byte(passphrase))

	case "awskms":
		opts := &awssigner.Options{
			Timeout:    config.Signer.KmsTimeout.Duration,
			MaxRetries: config.Signer.KmsMaxRetries,
			CacheTTL:   config.Signer.KmsCacheTTL.Duration,
		}
		return awssigner.NewKmsSigner(ctx, config.Signer.KmsRegion, config.Signer.KmsProfile, config.Signer.KmsKeyID, opts)

	default:
		return nil, fmt.Errorf("unknown signer type: %s", config.Signer.SignerType)
	}
}
