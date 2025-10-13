#!/usr/bin/env bash

set -euo pipefail

# Load the backend library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/backup-lib.sh"

usage() {
  cat <<'EOF'
Usage: backup.sh [OPTIONS] <destination-directory>

Create a consistent backup of the ev-reth database using mdbx_copy and record
the block height captured in the snapshot.

Options:
  --mode MODE          Execution mode: 'local' or 'docker' (default: docker)
  --container NAME     Docker container name running ev-reth (default: ev-reth)
                       Only used in docker mode.
  --datadir PATH       Path to the reth datadir in the target environment
                       (default docker: /home/reth/eth-home)
                       (default local: /var/lib/reth)
  --mdbx-copy CMD      Path to the mdbx_copy binary in the target environment
                       (default: mdbx_copy; override if you compiled it elsewhere)
  --tag LABEL          Custom label for the backup directory (default: timestamp)
  --keep-remote        Leave the temporary snapshot in the target environment
  -h, --help           Show this help message

Modes:
  local                Run backup on the local machine (reth running locally)
  docker               Run backup on a Docker container (default)

Requirements:
  - mdbx_copy available in the target environment (compile it once if necessary).
  - jq installed on the host (used to parse StageCheckpoints JSON).
  - For docker mode: Docker access to the container running ev-reth.
  - For local mode: Direct filesystem access to reth datadir.

The destination directory will receive:
  <dest>/<tag>/db/mdbx.dat        MDBX snapshot
  <dest>/<tag>/db/mdbx.lck        Empty lock file placeholder
  <dest>/<tag>/static_files/...   Static files copied from the node
  <dest>/<tag>/stage_checkpoints.json
  <dest>/<tag>/height.txt         Height extracted from StageCheckpoints

Examples:
  # Backup from local reth instance
  ./backup.sh --mode local --datadir /var/lib/reth /path/to/backups

  # Backup from Docker container
  ./backup.sh --mode docker --container ev-reth /path/to/backups
EOF
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "error: required command '$1' not found in PATH" >&2
    exit 1
  fi
}

DEST=""
MODE="docker"
CONTAINER="ev-reth"
DATADIR=""
MDBX_COPY="mdbx_copy"
BACKUP_TAG=""
KEEP_REMOTE=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --mode)
      MODE="$2"
      shift 2
      ;;
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

# Validate and set defaults based on mode
case "$MODE" in
  local)
    if [[ -z "$DATADIR" ]]; then
      DATADIR="/var/lib/reth"
    fi
    ;;
  docker)
    if [[ -z "$DATADIR" ]]; then
      DATADIR="/home/reth/eth-home"
    fi
    ;;
  *)
    echo "error: invalid mode '$MODE'. Use 'local' or 'docker'." >&2
    exit 1
    ;;
esac

# Initialize the backend
if ! init_backend "$MODE"; then
  exit 1
fi

# Set container for docker mode
if [[ "$MODE" == "docker" ]]; then
  BACKEND_CONTAINER="$CONTAINER"
fi

# Check backend availability
if ! $check_backend_available; then
  exit 1
fi

require_cmd jq

if [[ -z "$BACKUP_TAG" ]]; then
  BACKUP_TAG="$(date +'%Y%m%d-%H%M%S')"
fi

REMOTE_TMP="/tmp/reth-backup-${BACKUP_TAG}"
HOST_DEST="$(mkdir -p "$DEST" && cd "$DEST" && pwd)/${BACKUP_TAG}"

echo "Mode: $MODE"
echo "Creating backup tag '$BACKUP_TAG' into ${HOST_DEST}"

# Prepare temporary workspace in target environment
echo "Preparing temporary workspace..."
$exec_remote "rm -rf '$REMOTE_TMP' && mkdir -p '$REMOTE_TMP/db' '$REMOTE_TMP/static_files'"

# Verify mdbx_copy availability
if ! verify_remote_command "$MDBX_COPY"; then
  exit 1
fi

echo "Running mdbx_copy in target environment..."
run_mdbx_copy "$MDBX_COPY" "${DATADIR}/db" "$REMOTE_TMP/db/mdbx.dat"
$exec_remote "touch '$REMOTE_TMP/db/mdbx.lck'"

echo "Copying static_files..."
$exec_remote "if [ -d '${DATADIR}/static_files' ]; then cp -a '${DATADIR}/static_files/.' '$REMOTE_TMP/static_files/' 2>/dev/null || true; fi"

echo "Querying StageCheckpoints height..."
STAGE_JSON=$(query_stage_checkpoints "$REMOTE_TMP")
HEIGHT=$(echo "$STAGE_JSON" | jq -r '.[] | select(.[0]=="Finish") | .[1].block_number' | tr -d '\r\n')

if [[ -z "$HEIGHT" || "$HEIGHT" == "null" ]]; then
  echo "warning: could not determine height from StageCheckpoints" >&2
fi

echo "Copying snapshot to host..."
mkdir -p "$HOST_DEST/db"
$copy_from_remote "${REMOTE_TMP}/db/mdbx.dat" "$HOST_DEST/db/mdbx.dat"
$copy_from_remote "${REMOTE_TMP}/db/mdbx.lck" "$HOST_DEST/db/mdbx.lck"

if remote_path_exists "${REMOTE_TMP}/static_files"; then
  mkdir -p "$HOST_DEST/static_files"
  $copy_from_remote "${REMOTE_TMP}/static_files/." "$HOST_DEST/static_files/" || true
fi

echo "$STAGE_JSON" > "$HOST_DEST/stage_checkpoints.json"
if [[ -n "$HEIGHT" && "$HEIGHT" != "null" ]]; then
  echo "$HEIGHT" > "$HOST_DEST/height.txt"
  echo "Backup height: $HEIGHT"
else
  echo "Height not captured (see stage_checkpoints.json for details)"
fi

if [[ "$KEEP_REMOTE" -ne 1 ]]; then
  echo "Cleaning up temporary files..."
  $cleanup_remote "$REMOTE_TMP"
fi

echo "Backup completed: $HOST_DEST"
