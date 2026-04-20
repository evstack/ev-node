package localfiber

import (
	"fmt"
	"os"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
)

func NewKeyring(keyringPath, keyName, chainID string) (keyring.Keyring, error) {
	if keyringPath == "" {
		return nil, fmt.Errorf("keyring path is required")
	}
	if keyName == "" {
		return nil, fmt.Errorf("key name is required")
	}
	if _, err := os.Stat(keyringPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("keyring directory does not exist: %s", keyringPath)
	}

	kr, err := keyring.New("ev-node", keyring.BackendTest, keyringPath, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("creating keyring: %w", err)
	}

	info, err := kr.Key(keyName)
	if err != nil {
		return nil, fmt.Errorf("key %q not found in keyring: %w", keyName, err)
	}
	_ = info

	return kr, nil
}
