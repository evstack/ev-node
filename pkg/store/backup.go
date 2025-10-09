package store

import (
	"context"
	"fmt"
	"io"

	ds "github.com/ipfs/go-datastore"
	badger4 "github.com/ipfs/go-ds-badger4"
)

// Backup streams the underlying Badger datastore snapshot into the provided writer.
// The returned uint64 corresponds to the last version contained in the backup stream,
// which can be re-used to generate incremental backups via the since parameter.
func (s *DefaultStore) Backup(ctx context.Context, writer io.Writer, since uint64) (uint64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	visited := make(map[ds.Datastore]struct{})
	current, ok := any(s.db).(ds.Datastore)
	if !ok {
		return 0, fmt.Errorf("backup is not supported by the configured datastore")
	}

	for {
		// Try to leverage a native backup implementation if the underlying datastore exposes one.
		type backupable interface {
			Backup(io.Writer, uint64) (uint64, error)
		}
		if dsBackup, ok := current.(backupable); ok {
			version, err := dsBackup.Backup(writer, since)
			if err != nil {
				return 0, fmt.Errorf("datastore backup failed: %w", err)
			}
			return version, nil
		}

		// Default Badger datastore used across ev-node.
		if badgerDatastore, ok := current.(*badger4.Datastore); ok {
			// `badger.DB.Backup` internally orchestrates a consistent snapshot without pausing writes.
			version, err := badgerDatastore.DB.Backup(writer, since)
			if err != nil {
				return 0, fmt.Errorf("badger backup failed: %w", err)
			}
			return version, nil
		}

		// Attempt to unwrap shimmed datastores (e.g., prefix or mutex wrappers) to reach the backing store.
		if _, seen := visited[current]; seen {
			break
		}
		visited[current] = struct{}{}

		shim, ok := current.(ds.Shim)
		if !ok {
			break
		}
		children := shim.Children()
		if len(children) == 0 {
			break
		}
		current = children[0]
	}

	return 0, fmt.Errorf("backup is not supported by the configured datastore")
}
