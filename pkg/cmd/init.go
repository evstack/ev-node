package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	rollconf "github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/hash"
	"github.com/evstack/ev-node/pkg/p2p/key"
	"github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/signer/file"
)

// CreateSigner sets up the signer configuration and creates necessary files
func CreateSigner(config *rollconf.Config, homePath string, passphrase string) ([]byte, error) {
	if config.Signer.SignerType == "file" && config.Node.Aggregator {
		if passphrase == "" {
			return nil, fmt.Errorf("passphrase is required when using local file signer")
		}

		signerPath := config.Signer.SignerPath
		if signerPath == "" {
			signerPath = filepath.Join("config")
		}
		if !filepath.IsAbs(signerPath) {
			signerPath = filepath.Join(homePath, signerPath)
		}
		if strings.HasSuffix(strings.ToLower(signerPath), "signer.json") {
			signerPath = filepath.Dir(signerPath)
		}

		if info, err := os.Stat(signerPath); err == nil {
			if !info.IsDir() {
				return nil, fmt.Errorf("signer path %s must be a directory", signerPath)
			}
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("failed to inspect signer path: %w", err)
		}

		if err := os.MkdirAll(signerPath, 0o750); err != nil {
			return nil, fmt.Errorf("failed to create signer directory: %w", err)
		}

		keyFile := filepath.Join(signerPath, "signer.json")

		var (
			signer signer.Signer
			err    error
		)

		if _, err = os.Stat(keyFile); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("failed to check signer file: %w", err)
			}

			signer, err = file.CreateFileSystemSigner(signerPath, []byte(passphrase))
			if err != nil {
				return nil, fmt.Errorf("failed to initialize signer: %w", err)
			}
		} else {
			signer, err = file.LoadFileSystemSigner(signerPath, []byte(passphrase))
			if err != nil {
				return nil, fmt.Errorf("failed to load signer: %w", err)
			}
		}

		config.Signer.SignerPath = signerPath

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
	} else if config.Signer.SignerType != "file" && config.Node.Aggregator {
		return nil, fmt.Errorf("remote signer not implemented for aggregator nodes, use local signer instead")
	}

	return nil, nil
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
