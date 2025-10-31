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

	// Try direct badger4 cast first
	if badgerDatastore, ok := s.db.(*badger4.Datastore); ok {
		return backupBadger(badgerDatastore, writer, since)
	}

	// Try to unwrap one level (e.g., PrefixTransform wrapper)
	if shim, ok := s.db.(ds.Shim); ok {
		children := shim.Children()
		if len(children) > 0 {
			if badgerDatastore, ok := children[0].(*badger4.Datastore); ok {
				return backupBadger(badgerDatastore, writer, since)
			}
		}
	}

	return 0, fmt.Errorf("backup is only supported for badger4 datastore")
}

func backupBadger(badgerDatastore *badger4.Datastore, writer io.Writer, since uint64) (uint64, error) {
	// `badger.DB.Backup` internally orchestrates a consistent snapshot without pausing writes.
	version, err := badgerDatastore.DB.Backup(writer, since)
	if err != nil {
		return 0, fmt.Errorf("badger backup failed: %w", err)
	}
	return version, nil
}
