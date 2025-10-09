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

## End-to-end workflow with `apps/evm/single`

The `evm/single` docker-compose setup expects the backup image to reuse the
`ghcr.io/evstack/ev-reth:latest` tag. To capture both the MDBX and ev-node
Badger backups end-to-end:

1. Build the helper image so `ev-reth` has the MDBX tooling preinstalled.
   ```bash
   docker build -t ghcr.io/evstack/ev-reth:latest scripts/reth-backup
   ```
2. Build the `ev-node-evm-single` image so it includes the latest CLI backup
   command (optional if you already pushed the binary and re-tagged it).
   ```bash
   docker build -t ghcr.io/evstack/ev-node-evm-single:main -f apps/evm/single/Dockerfile .
   ```
3. Start the stack.
   ```bash
   (cd apps/evm/single && docker compose up -d)
   ```
4. Run the MDBX snapshot script (adjust the destination as needed).
   ```bash
   ./scripts/reth-backup/backup.sh backups/full-run/reth
   ```
5. Stream a Badger backup from the ev-node into the container filesystem and
   copy it to the host.
   ```bash
   docker compose exec ev-node-evm-single \
     evm-single backup --output /tmp/evnode-backup.badger --force
   docker cp ev-node-evm-single:/tmp/evnode-backup.badger backups/full-run/ev-node/
   ```
6. When finished, tear the stack down.
   ```bash
   (cd apps/evm/single && docker compose down)
   ```

Both backups can then be found under `backups/full-run/`.
