package store

import (
	"context"
	"fmt"
	"io"

	ds "github.com/ipfs/go-datastore"
	badger4 "github.com/ipfs/go-ds-badger4"
)

// Restore loads a Badger backup from the provided reader into the datastore.
// This operation will fail if the datastore already contains data, unless the datastore
// supports merging backups. The restore process is atomic and will either complete
// fully or leave the datastore unchanged.
func (s *DefaultStore) Restore(ctx context.Context, reader io.Reader) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	visited := make(map[ds.Datastore]struct{})
	current, ok := any(s.db).(ds.Datastore)
	if !ok {
		return fmt.Errorf("restore is not supported by the configured datastore")
	}

	for {
		// Try to leverage a native restore implementation if the underlying datastore exposes one.
		type restorable interface {
			Load(io.Reader) error
		}
		if dsRestore, ok := current.(restorable); ok {
			if err := dsRestore.Load(reader); err != nil {
				return fmt.Errorf("datastore restore failed: %w", err)
			}
			return nil
		}

		// Default Badger datastore used across ev-node.
		if badgerDatastore, ok := current.(*badger4.Datastore); ok {
			// `badger.DB.Load` internally restores the backup atomically.
			if err := badgerDatastore.DB.Load(reader, 16); err != nil {
				return fmt.Errorf("badger restore failed: %w", err)
			}
			return nil
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

	return fmt.Errorf("restore is not supported by the configured datastore")
}
