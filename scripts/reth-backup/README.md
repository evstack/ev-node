# Reth Backup Helper

Script to snapshot the `ev-reth` MDBX database while the node keeps running and
record the block height contained in the snapshot.

## Prerequisites

- Docker access to the container running `ev-reth` (defaults to the service name
  `ev-reth` from `docker-compose`).
- The `mdbx_copy` binary available inside that container. If it is not provided
  by the image, compile it once inside the container (see [libmdbx
  documentation](https://libmdbx.dqdkfa.ru/)).
- `jq` installed on the host to parse the JSON output.

## Usage

```bash
./scripts/reth-backup/backup.sh \
  --container ev-reth \
  --datadir /home/reth/eth-home \
  --mdbx-copy /tmp/libmdbx/build/mdbx_copy \
  /path/to/backups
```

This creates a timestamped folder under `/path/to/backups` with:

- `db/mdbx.dat` – consistent MDBX snapshot.
- `db/mdbx.lck` – placeholder lock file (empty).
- `static_files/` – static files copied from the node.
- `stage_checkpoints.json` – raw StageCheckpoints table.
- `height.txt` – extracted block height (from the `Finish` stage).

Additional flags:

- `--tag LABEL` to override the timestamped folder name.
- `--keep-remote` to leave the temporary snapshot inside the container (useful
  for debugging).

The script outputs the height at the end so you can coordinate other backups
with the same block number.
