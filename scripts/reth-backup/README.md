# Reth Backup Helper

Script to snapshot the `ev-reth` MDBX database while the node keeps running and
record the block height contained in the snapshot.

The script supports two execution modes:

- **local**: Backup a reth instance running directly on the host machine
- **docker**: Backup a reth instance running in a Docker container

## Prerequisites

### Common requirements

- The `mdbx_copy` binary available in the target environment (see [libmdbx
  documentation](https://libmdbx.dqdkfa.ru/)).
- `jq` installed on the host to parse the JSON output.

### Docker mode

- Docker access to the container running `ev-reth` (defaults to the service name
  `ev-reth` from `docker-compose`).

### Local mode

- Direct filesystem access to the reth datadir.
- Sufficient permissions to read the database files.

## Usage

### Local mode

When reth is running directly on your machine:

```bash
./scripts/reth-backup/backup.sh \
  --mode local \
  --datadir /var/lib/reth \
  --mdbx-copy /usr/local/bin/mdbx_copy \
  /path/to/backups
```

### Docker mode

When reth is running in a Docker container:

```bash
./scripts/reth-backup/backup.sh \
  --mode docker \
  --container ev-reth \
  --datadir /home/reth/eth-home \
  --mdbx-copy /tmp/libmdbx/build/mdbx_copy \
  /path/to/backups
```

### Output structure

Both modes create a timestamped folder under `/path/to/backups` with:

- `db/mdbx.dat` – consistent MDBX snapshot.
- `db/mdbx.lck` – placeholder lock file (empty).
- `static_files/` – static files copied from the node.
- `stage_checkpoints.json` – raw StageCheckpoints table.
- `height.txt` – extracted block height (from the `Finish` stage).

Additional flags:

- `--tag LABEL` to override the timestamped folder name.
- `--keep-remote` to leave the temporary snapshot in the target environment
  (useful for debugging).

The script outputs the height at the end so you can coordinate other backups
with the same block number.

## Architecture

The backup script is split into two components:

- **`backup-lib.sh`**: Abstract execution layer providing a common interface for
  different execution modes (local, docker). This library defines functions like
  `exec_remote`, `copy_from_remote`, `copy_to_remote`, and `cleanup_remote`
  that are implemented differently for each backend.
- **`backup.sh`**: Main script that uses the library and orchestrates the backup
  workflow. It's mode-agnostic and works with any backend that implements the
  required interface.

This separation allows easy extension to support additional execution
environments (SSH, Kubernetes, etc.) without modifying the core backup logic.

## End-to-end workflow with `apps/evm/single` (Docker mode)

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
   ./scripts/reth-backup/backup.sh --mode docker backups/full-run/reth
   ```

   The script prints the generated tag (for example `20251013-104816`) and the
   captured height (stored under
   `backups/full-run/reth/<TAG>/height.txt`).

5. Align the ev-node datastore to that height and take the Badger backup:

   ```bash
   HEIGHT=$(cat backups/full-run/reth/<TAG>/height.txt)
   BACKUP_ROOT="$(pwd)/backups/full-run"

   # Stop the managed container so it cannot advance.
   (cd apps/evm/single && docker compose stop ev-node-evm-single)

   # Roll the datastore back to the captured height. The --sync-node and
   # --skip-p2p-stores flags avoid DA-finality and header-store checks.
   (cd apps/evm/single && docker compose run --rm \
     ev-node-evm-single rollback \
       --height "${HEIGHT}" \
       --home /root/.evm-single \
       --sync-node \
       --skip-p2p-stores)

   # Stream the Badger backup without producing new blocks.
   cat <<'EOF' > /tmp/evnode_backup.sh
set -euo pipefail
OUT="$1"
TARGET="$2"
START_CMD="evm-single start \
  --home=/root/.evm-single \
  --evm.jwt-secret $EVM_JWT_SECRET \
  --evm.genesis-hash $EVM_GENESIS_HASH \
  --evm.engine-url $EVM_ENGINE_URL \
  --evm.eth-url $EVM_ETH_URL \
  --rollkit.node.block_time 1h \
  --rollkit.node.aggregator=true \
  --rollkit.signer.passphrase $EVM_SIGNER_PASSPHRASE \
  --rollkit.da.address $DA_ADDRESS \
  --evnode.clear_cache"
rm -f "$OUT"
sh -c "$START_CMD" &
PID=$!
trap "kill $PID 2>/dev/null || true; wait $PID 2>/dev/null || true" EXIT
for i in $(seq 1 30); do
  sleep 2
  if evm-single backup --output "$OUT" --force --target-height "$TARGET" >/tmp/backup.log 2>&1; then
    cat /tmp/backup.log
    exit 0
  fi
  cat /tmp/backup.log
done
echo "backup did not succeed within timeout" >&2
exit 1
EOF
   chmod +x /tmp/evnode_backup.sh

   (cd apps/evm/single && docker compose run --rm \
     --entrypoint sh \
     -v /tmp/evnode_backup.sh:/tmp/evnode_backup.sh \
     -v "${BACKUP_ROOT}/ev-node:/host-backup" \
     ev-node-evm-single \
     -c "/tmp/evnode_backup.sh /host-backup/evnode-backup-aligned.badger ${HEIGHT}")

   rm /tmp/evnode_backup.sh

   # Bring the managed container back with its usual supervisor.
   (cd apps/evm/single && docker compose start ev-node-evm-single)
   ```

   The CLI will report the streamed metadata, and the backup lands at
   `backups/full-run/ev-node/evnode-backup-aligned.badger`.

6. When finished, tear the stack down.

   ```bash
   (cd apps/evm/single && docker compose down)
   ```

Both backups can then be found under `backups/full-run/`.

## Restoring and validating the backups

To verify that both snapshots can be replayed, you can shut everything down, mutate the live data, and then restore from the artifacts collected above.

1. **Let the stack advance after the backup (optional but recommended).** Keep `docker compose` running for a few more blocks or submit a dev transaction so the live height diverges from the one recorded in `backups/full-run/reth/height.txt`.
2. **Stop the services.**
   ```bash
   (cd apps/evm/single && docker compose down)
   ```
3. **Recreate the containers without starting them.** This gives you stopped containers that already own the right named volumes.
   ```bash
   (cd apps/evm/single && docker compose up --no-start)
   ```
4. **Restore the `ev-reth` MDBX volume from the snapshot.** Run the following from the repository root (adjust `BACKUP_ROOT` if you saved the files elsewhere):
   ```bash
   BACKUP_ROOT="$(pwd)/backups/full-run"
   docker run --rm \
     --volumes-from ev-reth \
     -v "${BACKUP_ROOT}/reth:/backup:ro" \
     alpine:3.18 \
     sh -ec 'rm -rf /home/reth/eth-home/db /home/reth/eth-home/static_files && \
             mkdir -p /home/reth/eth-home/db /home/reth/eth-home/static_files && \
             cp /backup/db/mdbx.dat /home/reth/eth-home/db/mdbx.dat && \
             cp /backup/db/mdbx.lck /home/reth/eth-home/db/mdbx.lck && \
             cp -a /backup/static_files/. /home/reth/eth-home/static_files/ || true'
   ```
   > `docker run --volumes-from ev-reth` reuses the stopped container's volumes; adjust `alpine:3.18` if you prefer another image that provides `cp`.
5. **Restore the `ev-node` Badger datastore.**
   ```bash
   TEMP_RESTORE="$(mktemp -d backups/full-run/ev-node/restore-XXXXXX)"
    badger restore --dir "${TEMP_RESTORE}" --files "${BACKUP_ROOT}/ev-node/evnode-backup-aligned.badger"
   docker run --rm \
     --volumes-from ev-node-evm-single \
     -v "${TEMP_RESTORE}:/restore:ro" \
     alpine:3.18 \
     sh -ec 'rm -rf /root/.evm-single/data && mkdir -p /root/.evm-single/data/evm-single && \
             cp -a /restore/. /root/.evm-single/data/evm-single/'
   rm -rf "${TEMP_RESTORE}"
   ```
   > Install the Badger CLI once via `go install github.com/dgraph-io/badger/v4/cmd/badger@latest` if it is not already on your `$PATH`.
6. **Start the services back up.**
   ```bash
   (cd apps/evm/single && docker compose up -d ev-reth local-da)
   (cd apps/evm/single && docker compose up -d ev-node-evm-single)
   ```
   If you prefer to launch the sequencer manually first, you can run:
   ```bash
   (cd apps/evm/single && docker compose run --rm \
     ev-node-evm-single start \
       --home /root/.evm-single \
       --evm.jwt-secret "$EVM_JWT_SECRET" \
       --evm.genesis-hash "$EVM_GENESIS_HASH" \
       --evm.engine-url "$EVM_ENGINE_URL" \
       --evm.eth-url "$EVM_ETH_URL" \
       --rollkit.node.block_time 1s \
       --rollkit.node.aggregator=true \
       --rollkit.signer.passphrase "$EVM_SIGNER_PASSPHRASE" \
       --rollkit.da.address "$DA_ADDRESS" \
       --evnode.clear_cache)
   ```
   and once it exits cleanly, start the managed container with `docker compose up -d ev-node-evm-single`.
7. **Confirm the node resumes from the backed-up height.** Compare the logged chain height with `backups/full-run/reth/height.txt` and run:
   ```bash
   (cd apps/evm/single && docker compose exec ev-node-evm-single evm-single net-info --home /root/.evm-single)
   ```
   The reported height should match the snapshot before new blocks are produced.

With this round-trip check you can be confident that the snapshot pair (MDBX + Badger) fully recreates the state captured during the backup.
