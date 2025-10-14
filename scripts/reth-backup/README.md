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

### Prerequisites

1. Build the reth image with MDBX tooling:

   ```bash
   docker build -t ghcr.io/evstack/ev-reth:latest scripts/reth-backup
   ```

2. Build the ev-node image with backup/restore commands:

   ```bash
   docker build -t ghcr.io/evstack/ev-node-evm-single:main -f apps/evm/single/Dockerfile .
   ```

3. Start the stack:

   ```bash
   cd apps/evm/single && docker compose up -d
   ```

### Backup

1. Backup reth (captures MDBX snapshot at current height):

   ```bash
   ./scripts/reth-backup/backup.sh --mode docker backups/full-run/reth
   ```

   Note the printed TAG (e.g., `20251013-104816`) and height.

2. Backup ev-node (captures complete Badger datastore):

   ```bash
   TAG=<TAG>  # from previous step
   HEIGHT=$(cat backups/full-run/reth/${TAG}/height.txt)
   
   mkdir -p backups/full-run/ev-node
   
   docker exec evolveevm-ev-node-evm-single-1 \
     evm-single backup \
       --output /tmp/backup-${TAG}.badger \
       --force
   
   docker cp evolveevm-ev-node-evm-single-1:/tmp/backup-${TAG}.badger \
     backups/full-run/ev-node/
   
   echo ${HEIGHT} > backups/full-run/ev-node/target-height.txt
   ```

### Restore

1. Stop services and recreate containers:

   ```bash
   cd apps/evm/single
   docker compose down
   docker compose up --no-start
   ```

2. Restore reth volume:

   ```bash
   TAG=<TAG>
   
   # From apps/evm/single directory, use relative path to backups
   docker run --rm \
     --volumes-from ev-reth \
     -v "$PWD/../../backups/full-run/reth/${TAG}:/backup:ro" \
     alpine:3.18 \
     sh -c 'rm -rf /home/reth/eth-home/db /home/reth/eth-home/static_files && \
            mkdir -p /home/reth/eth-home/db /home/reth/eth-home/static_files && \
            cp /backup/db/mdbx.dat /home/reth/eth-home/db/ && \
            cp /backup/db/mdbx.lck /home/reth/eth-home/db/ && \
            cp -a /backup/static_files/. /home/reth/eth-home/static_files/ || true'
   ```

3. Restore ev-node volume:

   ```bash
   TAG=<TAG>
   
   # From apps/evm/single directory, use relative path to backups
   docker run --rm \
     --volumes-from evolveevm-ev-node-evm-single-1 \
     -v "$PWD/../../backups/full-run/ev-node:/backup:ro" \
     ghcr.io/evstack/ev-node-evm-single:main \
     restore \
       --input /backup/backup-${TAG}.badger \
       --home /root/.evm-single \
       --app-name evm-single \
       --force
   ```

4. Align ev-node to reth height using rollback (before starting):

   ```bash
   HEIGHT=$(cat backups/full-run/ev-node/target-height.txt)
   
   docker run --rm \
     --volumes-from evolveevm-ev-node-evm-single-1 \
     ghcr.io/evstack/ev-node-evm-single:main \
     rollback \
       --home /root/.evm-single \
       --height ${HEIGHT} \
       --sync-node
   ```

   > **Note:** The rollback may report errors for p2p header/data stores with invalid
   > ranges. This is expected and can be ignored. The main state will be correctly
   > rolled back to the target height. The `--sync-node` flag is required for
   > non-aggregator mode rollback.

5. Start reth and local-da services:

   ```bash
   docker compose start ev-reth local-da
   ```

6. Start ev-node with cache cleared (first time only):

   ```bash
   # Remove the stopped container and start with --evnode.clear_cache
   docker rm evolveevm-ev-node-evm-single-1
   
   docker run -d \
     --name evolveevm-ev-node-evm-single-1 \
     --network evolveevm_evolve-network \
     -p 7676:7676 -p 7331:7331 \
     -v evolveevm_evm-single-data:/root/.evm-single/ \
     -e EVM_ENGINE_URL=http://ev-reth:8551 \
     -e EVM_ETH_URL=http://ev-reth:8545 \
     -e EVM_JWT_SECRET=f747494bb0fb338a0d71f5f9fe5b5034c17cc988c229b59fd71e005ee692e9bf \
     -e EVM_GENESIS_HASH=0x2b8bbb1ea1e04f9c9809b4b278a8687806edc061a356c7dbc491930d8e922503 \
     -e EVM_BLOCK_TIME=1s \
     -e EVM_SIGNER_PASSPHRASE=secret \
     -e DA_ADDRESS=http://local-da:7980 \
     ghcr.io/evstack/ev-node-evm-single:main \
     start --evnode.clear_cache
   ```

   > **Important:** Use `--evnode.clear_cache` on first start after restore to clear
   > any cached p2p data that may be inconsistent after rollback. On subsequent restarts,
   > you can use `docker compose up -d` normally.

7. Verify both nodes are at the same height:

   ```bash
   HEIGHT=$(cat backups/full-run/ev-node/target-height.txt)
   echo "Expected restored height: ${HEIGHT}"
   
   # Check ev-node is producing blocks from the restored height
   docker logs evolveevm-ev-node-evm-single-1 2>&1 | grep "produced block" | head -10
   
   # Check reth current height
   docker exec ev-reth curl -s -X POST -H "Content-Type: application/json" \
     --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
     http://localhost:8545 | jq -r '.result' | xargs printf "%d\n"
   ```

## Known Limitations

### Rollback P2P Store Errors

When rolling back to a height significantly lower than the current state, the p2p
header and data sync stores may report "invalid range" errors. This occurs because
these stores track sync progress independently. The errors can be safely ignored as:

1. The main blockchain state is correctly rolled back
2. Using `--evnode.clear_cache` on restart clears the inconsistent cache
3. The node will resync p2p data from the restored height

### Timestamp Consistency

After a restore, if significant real-world time has passed since the backup was created,
you may encounter timestamp validation errors when the node attempts to continue block
production. This occurs because:

- Reth stores block timestamps based on when blocks were originally created
- After restore, the restored timestamps may be in the past relative to system time
- Block validators may reject new blocks with timestamps earlier than parent blocks

**Workaround:** In production environments, coordinate restore operations to minimize
time between backup and restore, or ensure the entire network is restored simultaneously.

## Summary

This backup/restore workflow enables point-in-time recovery for both reth (MDBX) and
ev-node (Badger) datastores. Key points:

- **Backup**: Hot backup while nodes are running (no downtime)
- **Restore**: Requires stopping services, restoring volumes, and aligning heights
- **Rollback**: May show p2p store errors that can be safely ignored
- **Production**: Test the full workflow in staging before deploying to production

The process has been validated to correctly restore state and resume block production
from the backup point, with known limitations around p2p store consistency and timestamp
validation that can be mitigated with proper operational procedures.
