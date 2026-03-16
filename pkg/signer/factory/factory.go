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
	return newSigner(ctx, config, passphrase, false)
}

// NewSignerForInit creates a new Signer for init-time flows.
// For file signer, it creates a new key if signer.json is missing.
func NewSignerForInit(ctx context.Context, config *rollconf.Config, passphrase string) (signer.Signer, error) {
	return newSigner(ctx, config, passphrase, true)
}

func newSigner(ctx context.Context, config *rollconf.Config, passphrase string, allowCreate bool) (signer.Signer, error) {
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

		signerFile := filepath.Join(signerPath, "signer.json")
		if allowCreate {
			if err := os.MkdirAll(signerPath, 0o750); err != nil {
				return nil, fmt.Errorf("failed to create signer directory: %w", err)
			}
			if _, err := os.Stat(signerFile); os.IsNotExist(err) {
				return file.CreateFileSystemSigner(signerPath, []byte(passphrase))
			}
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
