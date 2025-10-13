#!/usr/bin/env bash

# backup-lib.sh - Abstract execution layer for reth backup operations
# Provides a common interface for local and Docker-based executions.

# Backend interface that must be implemented:
# - exec_remote <command>        Execute a command in the target environment
# - copy_from_remote <src> <dst> Copy a file/directory from target to local
# - copy_to_remote <src> <dst>   Copy a file/directory from local to target
# - cleanup_remote <path>        Remove a path in the target environment

# ============================================================================
# LOCAL BACKEND
# ============================================================================

local_exec_remote() {
  bash -c "$1"
}

local_copy_from_remote() {
  local src="$1"
  local dst="$2"
  cp -a "$src" "$dst"
}

local_copy_to_remote() {
  local src="$1"
  local dst="$2"
  cp -a "$src" "$dst"
}

local_cleanup_remote() {
  local path="$1"
  rm -rf "$path"
}

local_check_available() {
  # Always available
  return 0
}

# ============================================================================
# DOCKER BACKEND
# ============================================================================

docker_exec_remote() {
  local container="$BACKEND_CONTAINER"
  docker exec "$container" bash -lc "$1"
}

docker_copy_from_remote() {
  local container="$BACKEND_CONTAINER"
  local src="$1"
  local dst="$2"
  docker cp "${container}:${src}" "$dst"
}

docker_copy_to_remote() {
  local container="$BACKEND_CONTAINER"
  local src="$1"
  local dst="$2"
  docker cp "$src" "${container}:${dst}"
}

docker_cleanup_remote() {
  local container="$BACKEND_CONTAINER"
  local path="$1"
  docker exec "$container" rm -rf "$path"
}

docker_check_available() {
  if ! command -v docker >/dev/null 2>&1; then
    echo "error: docker command not found" >&2
    return 1
  fi
  
  local container="$BACKEND_CONTAINER"
  if [[ -z "$container" ]]; then
    echo "error: container name is required for docker mode" >&2
    return 1
  fi
  
  if ! docker ps --format '{{.Names}}' | grep -q "^${container}$"; then
    echo "error: container '$container' is not running" >&2
    return 1
  fi
  
  return 0
}

# ============================================================================
# BACKEND INITIALIZATION
# ============================================================================

# Set the backend mode and initialize function pointers
init_backend() {
  local mode="$1"
  
  case "$mode" in
    local)
      exec_remote=local_exec_remote
      copy_from_remote=local_copy_from_remote
      copy_to_remote=local_copy_to_remote
      cleanup_remote=local_cleanup_remote
      check_backend_available=local_check_available
      ;;
    docker)
      exec_remote=docker_exec_remote
      copy_from_remote=docker_copy_from_remote
      copy_to_remote=docker_copy_to_remote
      cleanup_remote=docker_cleanup_remote
      check_backend_available=docker_check_available
      ;;
    *)
      echo "error: unknown backend mode '$mode'" >&2
      echo "supported modes: local, docker" >&2
      return 1
      ;;
  esac
  
  BACKEND_MODE="$mode"
  return 0
}

# ============================================================================
# HIGH-LEVEL BACKUP OPERATIONS
# ============================================================================

# Verify that a command is available in the target environment
verify_remote_command() {
  local cmd="$1"
  if ! $exec_remote "command -v '$cmd' >/dev/null 2>&1 || [ -x '$cmd' ]"; then
    echo "error: command '$cmd' not found in target environment" >&2
    return 1
  fi
  return 0
}

# Create a directory in the target environment
create_remote_dir() {
  local path="$1"
  $exec_remote "mkdir -p '$path'"
}

# Check if a path exists in the target environment
remote_path_exists() {
  local path="$1"
  $exec_remote "test -e '$path'"
}

# Run mdbx_copy in the target environment
run_mdbx_copy() {
  local mdbx_copy="$1"
  local source_db="$2"
  local dest_file="$3"
  
  echo "Running mdbx_copy..."
  $exec_remote "'$mdbx_copy' -c '$source_db' '$dest_file'"
}

# Query ev-reth for stage checkpoints
query_stage_checkpoints() {
  local datadir="$1"
  $exec_remote "ev-reth db --datadir '$datadir' list StageCheckpoints --len 20 --json" | sed -n '/^\[/,$p'
}
