#!/usr/bin/env bash
#
# test-catchup.sh — Automated test suite for PR #3057 (sequencer catch-up from base)
#
# This script exercises all code paths introduced by the catch-up feature:
#   1. Sequencer catch-up unit tests (detection, mempool skipping, timestamps, exit, etc.)
#   2. DA client interface changes (GetLatestDAHeight, tracing)
#   3. Syncer DA height advancement logic (epoch-based stepping)
#   4. Mock/testda updates
#   5. Force inclusion e2e tests (exercises catch-up path end-to-end)
#
# Usage:
#   ./scripts/test-catchup.sh              # Run all test stages
#   ./scripts/test-catchup.sh --unit       # Run only unit tests
#   ./scripts/test-catchup.sh --e2e        # Run only e2e tests (requires building binaries)
#   ./scripts/test-catchup.sh --verbose    # Verbose output (-v flag to go test)
#   ./scripts/test-catchup.sh --race       # Enable race detector
#
# Exit codes:
#   0  All tests passed
#   1  One or more test stages failed

set -euo pipefail

# --- Configuration -----------------------------------------------------------

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
RESET='\033[0m'

VERBOSE=""
RACE=""
RUN_UNIT=true
RUN_E2E=false
FAILURES=0
PASSED=0
SKIPPED=0
STAGE_RESULTS=()

# --- Argument parsing ---------------------------------------------------------

for arg in "$@"; do
  case "$arg" in
    --unit)
      RUN_UNIT=true
      RUN_E2E=false
      ;;
    --e2e)
      RUN_UNIT=false
      RUN_E2E=true
      ;;
    --all)
      RUN_UNIT=true
      RUN_E2E=true
      ;;
    --verbose|-v)
      VERBOSE="-v"
      ;;
    --race)
      RACE="-race"
      ;;
    --help|-h)
      echo "Usage: $0 [--unit|--e2e|--all] [--verbose|-v] [--race]"
      echo ""
      echo "Runs the test suite for the sequencer catch-up feature (PR #3057)."
      echo ""
      echo "Options:"
      echo "  --unit       Run only unit/integration tests (default)"
      echo "  --e2e        Run only e2e tests (builds binaries first)"
      echo "  --all        Run both unit and e2e tests"
      echo "  --verbose    Pass -v to go test"
      echo "  --race       Enable Go race detector"
      echo "  --help       Show this help message"
      exit 0
      ;;
    *)
      echo "Unknown argument: $arg (use --help for usage)"
      exit 1
      ;;
  esac
done

# --- Helpers ------------------------------------------------------------------

log_header() {
  echo ""
  echo -e "${CYAN}${BOLD}========================================${RESET}"
  echo -e "${CYAN}${BOLD}  $1${RESET}"
  echo -e "${CYAN}${BOLD}========================================${RESET}"
}

log_stage() {
  echo ""
  echo -e "${YELLOW}--- Stage: $1${RESET}"
}

log_pass() {
  echo -e "${GREEN}  PASS${RESET} $1"
  PASSED=$((PASSED + 1))
  STAGE_RESULTS+=("PASS: $1")
}

log_fail() {
  echo -e "${RED}  FAIL${RESET} $1"
  FAILURES=$((FAILURES + 1))
  STAGE_RESULTS+=("FAIL: $1")
}

log_skip() {
  echo -e "${YELLOW}  SKIP${RESET} $1"
  SKIPPED=$((SKIPPED + 1))
  STAGE_RESULTS+=("SKIP: $1")
}

# run_tests <stage_name> <package> [extra go test args...]
run_tests() {
  local stage_name="$1"
  shift
  local pkg="$1"
  shift

  log_stage "$stage_name"

  if go test $VERBOSE $RACE -timeout=5m -count=1 "$pkg" "$@" 2>&1; then
    log_pass "$stage_name"
  else
    log_fail "$stage_name"
  fi
}

# run_tests_with_run <stage_name> <package> <-run pattern>
run_tests_with_run() {
  local stage_name="$1"
  local pkg="$2"
  local pattern="$3"

  log_stage "$stage_name"

  if go test $VERBOSE $RACE -timeout=5m -count=1 -run "$pattern" "$pkg" 2>&1; then
    log_pass "$stage_name"
  else
    log_fail "$stage_name"
  fi
}

# --- Banner -------------------------------------------------------------------

log_header "PR #3057: Sequencer Catch-Up From Base — Test Suite"
echo ""
echo "  Branch:  $(git branch --show-current 2>/dev/null || echo 'detached')"
echo "  Commit:  $(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"
echo "  Date:    $(date -Iseconds)"
echo "  Flags:   unit=$RUN_UNIT e2e=$RUN_E2E verbose=${VERBOSE:-off} race=${RACE:-off}"

# ==============================================================================
# UNIT & INTEGRATION TESTS
# ==============================================================================

if [ "$RUN_UNIT" = true ]; then

  log_header "Unit & Integration Tests"

  # --------------------------------------------------------------------------
  # Stage 1: Sequencer catch-up unit tests
  #
  # These are the core tests added by the PR. They test:
  #   - Detecting old epochs and entering catch-up mode
  #   - Skipping mempool transactions during catch-up
  #   - Using DA timestamps for catch-up block timestamps
  #   - Exiting catch-up when HeightFromFuture is returned
  #   - Multi-epoch replay
  #   - No catch-up when epoch is recent
  #   - No catch-up when forced inclusion is not configured
  #   - Checkpoint advancement during catch-up
  #   - Monotonic timestamp generation
  # --------------------------------------------------------------------------
  run_tests_with_run \
    "Sequencer catch-up tests (all TestSequencer_CatchUp_*)" \
    "./pkg/sequencers/single/..." \
    "TestSequencer_CatchUp_"

  # --------------------------------------------------------------------------
  # Stage 2: Existing sequencer tests (ensure no regressions)
  #
  # The PR modifies GetNextBatch, fetchNextDAEpoch, and adds new fields to
  # the Sequencer struct. Run ALL sequencer tests to catch regressions.
  # --------------------------------------------------------------------------
  run_tests \
    "Sequencer full test suite (regression check)" \
    "./pkg/sequencers/single/..."

  # --------------------------------------------------------------------------
  # Stage 3: DA client interface & tracing tests
  #
  # The PR adds GetLatestDAHeight to the DA Client interface and adds tracing
  # for it. This verifies the new method integrates correctly.
  # --------------------------------------------------------------------------
  run_tests \
    "DA client tracing tests" \
    "./block/internal/da/..."

  # --------------------------------------------------------------------------
  # Stage 4: Syncer tests
  #
  # The PR modifies the DA height advancement logic in the syncer to handle
  # catch-up blocks by stepping one epoch at a time instead of jumping.
  # --------------------------------------------------------------------------
  run_tests \
    "Syncer tests (DA height advancement)" \
    "./block/internal/syncing/..."

  # --------------------------------------------------------------------------
  # Stage 5: Mock & test DA
  #
  # The PR adds GetLatestDAHeight to the mock DA client and the DummyDA.
  # Verify mocks compile and are consistent.
  # --------------------------------------------------------------------------
  run_tests \
    "Mock DA (build check)" \
    "./test/mocks/..." \
    -run "^$"  # No test functions, just compile check

  run_tests \
    "Test DA (DummyDA)" \
    "./test/testda/..."

  # --------------------------------------------------------------------------
  # Stage 6: EVM force inclusion mock update
  #
  # The PR adds GetLatestDAHeight to the mockDA in apps/evm/server test.
  # This is a separate Go module, so we run from within its directory.
  # --------------------------------------------------------------------------
  log_stage "EVM force inclusion server tests"
  if (cd apps/evm && go test $VERBOSE $RACE -timeout=5m -count=1 ./server/...) 2>&1; then
    log_pass "EVM force inclusion server tests"
  else
    log_fail "EVM force inclusion server tests"
  fi

  # --------------------------------------------------------------------------
  # Stage 7: Types package — epoch calculations
  #
  # The catch-up logic relies on CalculateEpochNumber and
  # CalculateEpochBoundaries from the types package.
  # --------------------------------------------------------------------------
  run_tests_with_run \
    "Epoch calculation tests" \
    "./types/..." \
    "Epoch"

  # --------------------------------------------------------------------------
  # Stage 8: Block package (full build & test)
  #
  # The block package contains the DA client, syncer, and forced inclusion
  # retrieval. Run the full suite to catch any integration issues.
  # --------------------------------------------------------------------------
  run_tests \
    "Block package full suite" \
    "./block/..."

fi

# ==============================================================================
# E2E TESTS
# ==============================================================================

if [ "$RUN_E2E" = true ]; then

  log_header "End-to-End Tests"

  # --------------------------------------------------------------------------
  # Build binaries required for e2e tests
  # --------------------------------------------------------------------------
  log_stage "Building binaries for e2e tests"
  if make build build-da build-evm 2>&1; then
    log_pass "Binary build"
  else
    log_fail "Binary build"
    echo -e "${RED}Cannot run e2e tests without binaries. Aborting e2e stage.${RESET}"
  fi

  if [ -f "./build/testapp" ] && [ -f "./build/evm" ]; then
    # Build Docker image for Docker e2e tests (if not already built)
    log_stage "Building Docker image for e2e tests"
    if make docker-build-if-local 2>&1; then
      log_pass "Docker image build"
    else
      log_skip "Docker image build (non-fatal)"
    fi

    # --------------------------------------------------------------------------
    # Stage E1: Force inclusion e2e tests
    #
    # These tests exercise the end-to-end forced inclusion flow, which is the
    # mechanism that catch-up relies on. The sequencer fetches forced inclusion
    # transactions from DA epochs and includes them in blocks.
    # --------------------------------------------------------------------------
    log_stage "Force inclusion e2e tests"
    if (cd test/e2e && go test -mod=readonly -failfast -timeout=15m \
        -tags='e2e evm' $VERBOSE $RACE \
        -run "TestEvmSequencerForceInclusionE2E|TestEvmFullNodeForceInclusionE2E" \
        ./... --binary=../../build/testapp --evm-binary=../../build/evm) 2>&1; then
      log_pass "Force inclusion e2e tests"
    else
      log_fail "Force inclusion e2e tests"
    fi

    # --------------------------------------------------------------------------
    # Stage E2: DA restart e2e test
    #
    # Tests DA layer failure and recovery — related to catch-up because the
    # sequencer must handle accumulated blocks during DA downtime.
    # --------------------------------------------------------------------------
    log_stage "DA restart e2e test"
    if (cd test/e2e && go test -mod=readonly -failfast -timeout=15m \
        -tags='e2e evm' $VERBOSE $RACE \
        -run "TestEvmDARestartWithPendingBlocksE2E" \
        ./... --binary=../../build/testapp --evm-binary=../../build/evm) 2>&1; then
      log_pass "DA restart e2e test"
    else
      log_fail "DA restart e2e test"
    fi

    # --------------------------------------------------------------------------
    # Stage E3: Sequencer restart e2e test
    #
    # Tests sequencer restart and state persistence — the catch-up feature
    # activates during sequencer restart when DA has advanced.
    # --------------------------------------------------------------------------
    log_stage "Sequencer restart e2e test"
    if (cd test/e2e && go test -mod=readonly -failfast -timeout=15m \
        -tags='e2e evm' $VERBOSE $RACE \
        -run "TestEvmSequencerFullNodeRestartE2E" \
        ./... --binary=../../build/testapp --evm-binary=../../build/evm) 2>&1; then
      log_pass "Sequencer restart e2e test"
    else
      log_fail "Sequencer restart e2e test"
    fi

    # --------------------------------------------------------------------------
    # Stage E4: Basic testapp e2e (aggregator + fullnode sanity)
    #
    # Sanity check that the DA interface changes don't break basic operation.
    # --------------------------------------------------------------------------
    log_stage "Basic testapp e2e test"
    if (cd test/e2e && go test -mod=readonly -failfast -timeout=15m \
        -tags='e2e' $VERBOSE $RACE \
        -run "TestBasic|TestNodeRestartPersistence" \
        ./... --binary=../../build/testapp --evm-binary=../../build/evm) 2>&1; then
      log_pass "Basic testapp e2e test"
    else
      log_fail "Basic testapp e2e test"
    fi

  else
    log_skip "E2E tests (binaries not found)"
  fi
fi

# ==============================================================================
# SUMMARY
# ==============================================================================

log_header "Test Summary"

for result in "${STAGE_RESULTS[@]}"; do
  case "$result" in
    PASS:*) echo -e "  ${GREEN}$result${RESET}" ;;
    FAIL:*) echo -e "  ${RED}$result${RESET}" ;;
    SKIP:*) echo -e "  ${YELLOW}$result${RESET}" ;;
  esac
done

echo ""
echo -e "  Total:   $((PASSED + FAILURES + SKIPPED))"
echo -e "  ${GREEN}Passed:  $PASSED${RESET}"
echo -e "  ${RED}Failed:  $FAILURES${RESET}"
echo -e "  ${YELLOW}Skipped: $SKIPPED${RESET}"
echo ""

if [ "$FAILURES" -gt 0 ]; then
  echo -e "${RED}${BOLD}RESULT: FAILED${RESET} ($FAILURES stage(s) failed)"
  exit 1
else
  echo -e "${GREEN}${BOLD}RESULT: PASSED${RESET}"
  exit 0
fi
