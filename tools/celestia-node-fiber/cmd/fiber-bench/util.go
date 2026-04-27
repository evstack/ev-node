package main

import (
	"os"
	"path/filepath"
)

// defaultKeyringDir is where we put the bench's cosmos keyring by default.
// Lives under the user's home so multiple bench runs share the same key.
func defaultKeyringDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".fiber-bench"
	}
	return filepath.Join(home, ".fiber-bench", "keyring")
}

// defaultNodeHome is the ev-node working directory (signer, store, config).
// Cleared on each run by default — see runCmd's --keep-home flag.
func defaultNodeHome() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".fiber-bench-node"
	}
	return filepath.Join(home, ".fiber-bench", "node")
}
