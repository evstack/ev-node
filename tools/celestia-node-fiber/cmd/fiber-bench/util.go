package main

import (
	"os"
	"path/filepath"
)

// defaultKeyringDir is where we put the bench's cosmos keyring by default.
// Sibling of the ev-node home (~/.fiber-bench/node) so --keep-home=false
// runs cannot wipe it.
func defaultKeyringDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".fiber-bench-keyring"
	}
	return filepath.Join(home, ".fiber-bench", "keyring")
}
