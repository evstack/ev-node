package signer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	rollconf "github.com/evstack/ev-node/pkg/config"
)

func TestNewSigner_ErrorPaths(t *testing.T) {

	specs := map[string]struct {
		mutateCfg func(cfg *rollconf.Config)
		pass      string
		wantErr   string
	}{
		"unknown-type": {
			mutateCfg: func(cfg *rollconf.Config) {
				cfg.Signer.SignerType = "unknown"
			},
			pass:    "test-passphrase",
			wantErr: "unknown signer type: unknown",
		},
		"file-empty-passphrase": {
			mutateCfg: func(cfg *rollconf.Config) {
				cfg.Signer.SignerType = "file"
				cfg.Signer.SignerPath = t.TempDir()
			},
			pass:    "",
			wantErr: "passphrase is required when using local file signer",
		},
		"awskms-empty-key-id": {
			mutateCfg: func(cfg *rollconf.Config) {
				cfg.Signer.SignerType = "awskms"
				cfg.Signer.KmsKeyID = ""
			},
			pass:    "test-passphrase",
			wantErr: "aws kms key ID is required",
		},
	}

	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			cfg := rollconf.DefaultConfig()
			spec.mutateCfg(&cfg)

			s, err := NewSigner(t.Context(), &cfg, spec.pass)
			require.Error(t, err)
			require.Nil(t, s)
			require.ErrorContains(t, err, spec.wantErr)
		})
	}
}

func TestNewSignerForInit_FileFlow(t *testing.T) {
	specs := map[string]struct {
		setupExisting bool
	}{
		"create-missing-file": {
			setupExisting: false,
		},
		"load-existing-file": {
			setupExisting: true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			cfg := rollconf.DefaultConfig()
			cfg.Signer.SignerType = "file"
			cfg.Signer.SignerPath = t.TempDir()
			const passphrase = "test-passphrase"

			if spec.setupExisting {
				seedSigner, err := NewSignerForInit(t.Context(), &cfg, passphrase)
				require.NoError(t, err)
				require.NotNil(t, seedSigner)
			}

			initSigner, err := NewSignerForInit(t.Context(), &cfg, passphrase)
			require.NoError(t, err)
			require.NotNil(t, initSigner)

			_, err = os.Stat(filepath.Join(cfg.Signer.SignerPath, "signer.json"))
			require.NoError(t, err)

			runtimeSigner, err := NewSigner(t.Context(), &cfg, passphrase)
			require.NoError(t, err)
			require.NotNil(t, runtimeSigner)
		})
	}
}
