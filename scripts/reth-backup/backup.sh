#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage: backup.sh [OPTIONS] <destination-directory>

Create a consistent backup of the ev-reth database using mdbx_copy and record
the block height captured in the snapshot.

Options:
  --container NAME     Docker container name running ev-reth (default: ev-reth)
  --datadir PATH       Path to the reth datadir inside the container
                       (default: /home/reth/eth-home)
  --mdbx-copy CMD      Path to the mdbx_copy binary inside the container
                       (default: mdbx_copy; override if you compiled it elsewhere)
  --tag LABEL          Custom label for the backup directory (default: timestamp)
  --keep-remote        Leave the temporary snapshot inside the container
  -h, --help           Show this help message

Requirements:
  - Docker access to the container running ev-reth.
  - mdbx_copy available inside that container (compile it once if necessary).
  - jq installed on the host (used to parse StageCheckpoints JSON).

The destination directory will receive:
  <dest>/<tag>/db/mdbx.dat        MDBX snapshot
  <dest>/<tag>/db/mdbx.lck        Empty lock file placeholder
  <dest>/<tag>/static_files/...   Static files copied from the node
  <dest>/<tag>/stage_checkpoints.json
  <dest>/<tag>/height.txt         Height extracted from StageCheckpoints
EOF
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "error: required command '$1' not found in PATH" >&2
    exit 1
  fi
}

DEST=""
CONTAINER="ev-reth"
DATADIR="/home/reth/eth-home"
MDBX_COPY="mdbx_copy"
BACKUP_TAG=""
KEEP_REMOTE=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --container)
      CONTAINER="$2"
      shift 2
      ;;
    --datadir)
      DATADIR="$2"
      shift 2
      ;;
    --mdbx-copy)
      MDBX_COPY="$2"
      shift 2
      ;;
    --tag)
      BACKUP_TAG="$2"
      shift 2
      ;;
    --keep-remote)
      KEEP_REMOTE=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    --)
      shift
      break
      ;;
    -*)
      echo "unknown option: $1" >&2
      usage >&2
      exit 1
      ;;
    *)
      if [[ -z "$DEST" ]]; then
        DEST="$1"
        shift
      else
        echo "unexpected argument: $1" >&2
        usage >&2
        exit 1
      fi
      ;;
  esac
done

if [[ -z "$DEST" ]]; then
  echo "error: destination directory is required" >&2
  usage >&2
  exit 1
fi

require_cmd docker
require_cmd jq

if [[ -z "$BACKUP_TAG" ]]; then
  BACKUP_TAG="$(date +'%Y%m%d-%H%M%S')"
fi

REMOTE_TMP="/tmp/reth-backup-${BACKUP_TAG}"
HOST_DEST="$(mkdir -p "$DEST" && cd "$DEST" && pwd)/${BACKUP_TAG}"

echo "Creating backup tag '$BACKUP_TAG' into ${HOST_DEST}"

# Prepare temporary workspace inside the container.
docker exec "$CONTAINER" bash -c "rm -rf '$REMOTE_TMP' && mkdir -p '$REMOTE_TMP/db' '$REMOTE_TMP/static_files'"

# Verify mdbx_copy availability.
if ! docker exec "$CONTAINER" bash -lc "command -v '$MDBX_COPY' >/dev/null 2>&1 || [ -x '$MDBX_COPY' ]"; then
  echo "error: unable to find executable '$MDBX_COPY' inside container '$CONTAINER'" >&2
  exit 1
fi

echo "Running mdbx_copy inside container..."
docker exec "$CONTAINER" bash -lc "'$MDBX_COPY' -c '${DATADIR}/db' '$REMOTE_TMP/db/mdbx.dat'"
docker exec "$CONTAINER" bash -lc "touch '$REMOTE_TMP/db/mdbx.lck'"

echo "Copying static_files..."
docker exec "$CONTAINER" bash -lc "if [ -d '${DATADIR}/static_files' ]; then cp -a '${DATADIR}/static_files/.' '$REMOTE_TMP/static_files/' 2>/dev/null || true; fi"

echo "Querying StageCheckpoints height..."
STAGE_JSON=$(docker exec "$CONTAINER" ev-reth db --datadir "$REMOTE_TMP" list StageCheckpoints --len 20 --json | sed -n '/^\[/,$p')
HEIGHT=$(echo "$STAGE_JSON" | jq -r '.[] | select(.[0]=="Finish") | .[1].block_number' | tr -d '\r\n')

if [[ -z "$HEIGHT" || "$HEIGHT" == "null" ]]; then
  echo "warning: could not determine height from StageCheckpoints" >&2
fi

echo "Copying snapshot to host..."
mkdir -p "$HOST_DEST/db"
docker cp "${CONTAINER}:${REMOTE_TMP}/db/mdbx.dat" "$HOST_DEST/db/mdbx.dat"
docker cp "${CONTAINER}:${REMOTE_TMP}/db/mdbx.lck" "$HOST_DEST/db/mdbx.lck"

if docker exec "$CONTAINER" test -d "${REMOTE_TMP}/static_files"; then
  mkdir -p "$HOST_DEST/static_files"
  docker cp "${CONTAINER}:${REMOTE_TMP}/static_files/." "$HOST_DEST/static_files/"
fi

echo "$STAGE_JSON" > "$HOST_DEST/stage_checkpoints.json"
if [[ -n "$HEIGHT" && "$HEIGHT" != "null" ]]; then
  echo "$HEIGHT" > "$HOST_DEST/height.txt"
  echo "Backup height: $HEIGHT"
else
  echo "Height not captured (see stage_checkpoints.json for details)"
fi

if [[ "$KEEP_REMOTE" -ne 1 ]]; then
  docker exec "$CONTAINER" rm -rf "$REMOTE_TMP"
fi

echo "Backup completed: $HOST_DEST"
