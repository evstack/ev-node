package localfiber

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/celestiaorg/celestia-app/v9/fibre"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
)

type Config struct {
	KeyringPath    string
	KeyName        string
	StateAddress   string
	ChainID        string
	UploadConc     int
	DownloadConc   int
}

func NewFiberClient(ctx context.Context, cfg Config) (*fibre.Client, error) {
	kr, err := NewKeyring(cfg.KeyringPath, cfg.KeyName, cfg.ChainID)
	if err != nil {
		return nil, fmt.Errorf("creating keyring: %w", err)
	}

	return NewFiberClientWithKeyring(ctx, kr, cfg)
}

func NewFiberClientWithKeyring(ctx context.Context, kr keyring.Keyring, cfg Config) (*fibre.Client, error) {
	fibreCfg := fibre.DefaultClientConfig()
	fibreCfg.DefaultKeyName = cfg.KeyName
	fibreCfg.StateAddress = cfg.StateAddress
	if cfg.UploadConc > 0 {
		fibreCfg.UploadConcurrency = cfg.UploadConc
	}
	if cfg.DownloadConc > 0 {
		fibreCfg.DownloadConcurrency = cfg.DownloadConc
	}
	fibreCfg.Log = slog.Default().WithGroup("fibre")

	client, err := fibre.NewClient(kr, fibreCfg)
	if err != nil {
		return nil, fmt.Errorf("creating fibre client: %w", err)
	}

	if err := client.Start(ctx); err != nil {
		return nil, fmt.Errorf("starting fibre client: %w", err)
	}

	return client, nil
}
