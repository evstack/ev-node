# ev-node justfile

# Extract version information from git
version := `git describe --tags --abbrev=0 2>/dev/null || echo "dev"`
gitsha := `git rev-parse --short HEAD 2>/dev/null || echo "unknown"`
ldflags := "-X github.com/evstack/ev-node/pkg/cmd.Version=" + version + " -X github.com/evstack/ev-node/pkg/cmd.GitSHA=" + gitsha

# Tool build ldflags (uses main package)
tool_ldflags := "-X main.Version=" + version + " -X main.GitSHA=" + gitsha

# Build directory
build_dir := justfile_directory() / "build"

# List available recipes when running `just` with no args
[group('help')]
default:
    @just --list --unsorted

# ─── Build ────────────────────────────────────────────────────────────────────

# Build Testapp CLI
[group('build')]
build:
    @echo "--> Building Testapp CLI"
    @mkdir -p {{ build_dir }}
    @cd apps/testapp && go build -ldflags "{{ ldflags }}" -o {{ build_dir }}/testapp .
    @echo "--> Testapp CLI Built!"
    @echo "    Check the version with: build/testapp version"
    @echo "    Check the binary with: {{ build_dir }}/testapp"

# Install Testapp CLI to Go bin directory
[group('build')]
install:
    @echo "--> Installing Testapp CLI"
    @cd apps/testapp && go install -ldflags "{{ ldflags }}" .
    @echo "--> Testapp CLI Installed!"
    @echo "    Check the version with: testapp version"
    @echo "    Check the binary with: which testapp"

# Build all ev-node binaries
[group('build')]
build-all:
    @echo "--> Building all ev-node binaries"
    @mkdir -p {{ build_dir }}
    @echo "--> Building testapp"
    @cd apps/testapp && go build -ldflags "{{ ldflags }}" -o {{ build_dir }}/testapp .
    @echo "--> Building evm"
    @cd apps/evm && go build -ldflags "{{ ldflags }}" -o {{ build_dir }}/evm .
    @echo "--> Building local-da"
    @cd tools/local-da && go build -ldflags "{{ ldflags }}" -o {{ build_dir }}/local-da .
    @echo "--> All ev-node binaries built!"

# Build Testapp Bench
[group('build')]
build-testapp-bench:
    @echo "--> Building Testapp Bench"
    @mkdir -p {{ build_dir }}
    @cd apps/testapp && go build -ldflags "{{ ldflags }}" -o {{ build_dir }}/testapp-bench ./kv/bench
    @echo "    Check the binary with: {{ build_dir }}/testapp-bench"

# Build EVM binary
[group('build')]
build-evm:
    @echo "--> Building EVM"
    @mkdir -p {{ build_dir }}
    @cd apps/evm && go build -ldflags "{{ ldflags }}" -o {{ build_dir }}/evm .
    @echo "    Check the binary with: {{ build_dir }}/evm"

# Build local-da binary
[group('build')]
build-da:
    @echo "--> Building local-da"
    @mkdir -p {{ build_dir }}
    @cd tools/local-da && go build -ldflags "{{ ldflags }}" -o {{ build_dir }}/local-da .
    @echo "    Check the binary with: {{ build_dir }}/local-da"

# Build Docker image for local testing
[group('build')]
docker-build:
    @echo "--> Building Docker image for local testing"
    @docker build -t evstack:local-dev -f apps/testapp/Dockerfile .
    @echo "--> Docker image built: evstack:local-dev"
    @echo "--> Checking if image exists locally..."
    @docker images evstack:local-dev

# Clean build artifacts
[group('build')]
clean:
    @echo "--> Cleaning build directory"
    @rm -rf {{ build_dir }}
    @echo "--> Build directory cleaned!"

# ─── Testing ──────────────────────────────────────────────────────────────────

# Clear test cache
[group('test')]
clean-testcache:
    @echo "--> Clearing testcache"
    @go clean --testcache

# Run unit tests for all go.mods
[group('test')]
test:
    @echo "--> Running unit tests"
    @go run -tags='run integration' scripts/test.go

# Run all tests including Docker E2E
[group('test')]
test-all: test test-docker-e2e
    @echo "--> All tests completed"

# Run integration tests
[group('test')]
test-integration:
    @echo "--> Running integration tests"
    @cd node && go test -mod=readonly -failfast -timeout=15m -tags='integration' ./...

# Run e2e tests
[group('test')]
test-e2e: build build-da build-evm docker-build-if-local
    @echo "--> Running e2e tests"
    @cd test/e2e && go test -mod=readonly -failfast -timeout=15m -tags='e2e evm' ./... --binary=../../build/testapp --evm-binary=../../build/evm

# Run integration tests with coverage
[group('test')]
test-integration-cover:
    @echo "--> Running integration tests with coverage"
    @cd node && go test -mod=readonly -failfast -timeout=15m -tags='integration' -coverprofile=coverage.txt -covermode=atomic ./...

# Generate code coverage report
[group('test')]
test-cover:
    @echo "--> Running unit tests"
    @go run -tags=cover -race scripts/test_cover.go

# Run micro-benchmarks for internal cache
[group('test')]
bench:
    @echo "--> Running internal cache benchmarks"
    @go test -bench=. -benchmem -run='^$' ./block/internal/cache

# Run EVM tests
[group('test')]
test-evm:
    @echo "--> Running EVM tests"
    @cd execution/evm/test && go test -mod=readonly -failfast -timeout=15m ./... -tags=evm

# Run Docker E2E tests
[group('test')]
test-docker-e2e: docker-build-if-local
    @echo "--> Running Docker E2E tests"
    @echo "--> Verifying Docker image exists locally..."
    @if [ -z "${EV_NODE_IMAGE_REPO:-}" ] || [ "${EV_NODE_IMAGE_REPO:-}" = "ev-node" ]; then \
        echo "--> Verifying Docker image exists locally..."; \
        docker image inspect evstack:local-dev >/dev/null 2>&1 || (echo "ERROR: evstack:local-dev image not found. Run 'just docker-build' first." && exit 1); \
    fi
    @cd test/docker-e2e && go test -mod=readonly -failfast -v -tags='docker_e2e' -timeout=30m ./...
    @just docker-cleanup-if-local

# Run Docker E2E Upgrade tests
[group('test')]
test-docker-upgrade-e2e:
    @echo "--> Running Docker Upgrade E2E tests"
    @cd test/docker-e2e && go test -mod=readonly -failfast -v -tags='docker_e2e evm' -timeout=30m -run '^TestEVMSingleUpgradeSuite$$/^TestEVMSingleUpgrade$$' ./...

# Run Docker E2E cross-version compatibility tests
[group('test')]
test-docker-compat:
    @echo "--> Running Docker Sync Compatibility E2E tests"
    @cd test/docker-e2e && go test -mod=readonly -failfast -v -tags='docker_e2e evm' -timeout=30m -run '^TestEVMCompatSuite$$/^TestCrossVersionSync$$' ./...

# Build Docker image if using local repository
[group('test')]
docker-build-if-local:
    @if [ -z "${EV_NODE_IMAGE_REPO:-}" ] || [ "${EV_NODE_IMAGE_REPO:-}" = "evstack" ]; then \
        if docker image inspect evstack:local-dev >/dev/null 2>&1; then \
            echo "--> Found local Docker image: evstack:local-dev (skipping build)"; \
        else \
            echo "--> Local repository detected, building Docker image..."; \
            just docker-build; \
        fi; \
    else \
        echo "--> Using remote repository: ${EV_NODE_IMAGE_REPO}"; \
    fi

# Clean up local Docker image if using local repository
[group('test')]
docker-cleanup-if-local:
    @if [ -z "${EV_NODE_IMAGE_REPO:-}" ] || [ "${EV_NODE_IMAGE_REPO:-}" = "evstack" ]; then \
        echo "--> Untagging local Docker image (preserving layers for faster rebuilds)..."; \
        docker rmi evstack:local-dev --no-prune 2>/dev/null || echo "Image evstack:local-dev not found or already removed"; \
    else \
        echo "--> Using remote repository, no cleanup needed"; \
    fi

# ─── Protobuf ─────────────────────────────────────────────────────────────────

# Generate protobuf files
[group('proto')]
proto-gen:
    @echo "--> Generating Protobuf files"
    buf generate --path="./proto/evnode" --template="buf.gen.yaml" --config="buf.yaml"
    buf generate --path="./proto/execution/evm" --template="buf.gen.evm.yaml" --config="buf.yaml"

# Lint protobuf files (requires Docker)
[group('proto')]
proto-lint:
    @echo "--> Linting Protobuf files"
    @docker run --rm -v {{ justfile_directory() }}:/workspace --workdir /workspace bufbuild/buf:latest lint --error-format=json

# Generate Rust protobuf files
[group('proto')]
rust-proto-gen:
    @echo "--> Generating Rust protobuf files"
    @cd client/crates/types && EV_TYPES_FORCE_PROTO_GEN=1 cargo build

# Check if Rust protobuf files are up to date
[group('proto')]
rust-proto-check:
    @echo "--> Checking Rust protobuf files"
    @rm -rf client/crates/types/src/proto/*.rs
    @cd client/crates/types && cargo build
    @if ! git diff --exit-code client/crates/types/src/proto/; then \
        echo "Error: Generated Rust proto files are not up to date."; \
        echo "Run 'just rust-proto-gen' and commit the changes."; \
        exit 1; \
    fi

# ─── Lint ──────────────────────────────────────────────────────────────────────

# Run all linters
[group('lint')]
lint: vet
    @echo "--> Running golangci-lint"
    @golangci-lint run
    @echo "--> Running markdownlint"
    @markdownlint --config .markdownlint.yaml '**/*.md'
    @echo "--> Running hadolint"
    @hadolint test/docker/mockserv.Dockerfile
    @echo "--> Running yamllint"
    @yamllint --no-warnings . -c .yamllint.yml
    @echo "--> Running goreleaser check"
    @goreleaser check
    @echo "--> Running actionlint"
    @actionlint

# Auto-fix linting issues
[group('lint')]
lint-fix:
    @echo "--> Formatting go"
    @golangci-lint run --fix
    @echo "--> Formatting markdownlint"
    @markdownlint --config .markdownlint.yaml --ignore './changelog.md' '**/*.md' -f

# Run go vet
[group('lint')]
vet:
    @echo "--> Running go vet"
    @go vet ./...

# ─── Codegen ──────────────────────────────────────────────────────────────────

# Generate mocks
[group('codegen')]
mock-gen:
    @echo "-> Generating mocks"
    go run github.com/vektra/mockery/v3@latest

# Install dependencies and tidy all modules
[group('codegen')]
deps:
    @echo "--> Installing dependencies"
    @go mod download
    @go mod tidy
    @go run scripts/tidy.go

# Tidy all go.mod files
[group('codegen')]
tidy-all:
    @go run -tags=tidy scripts/tidy.go

# ─── Run ──────────────────────────────────────────────────────────────────────

# Run 'n' nodes (default: 1). Usage: just run-n 3
[group('run')]
run-n nodes="1": build build-da
    @go run -tags=run scripts/run.go --nodes={{ nodes }}

# Run EVM nodes (one sequencer and one full node)
[group('run')]
run-evm-nodes nodes="1": build-da build-evm
    @echo "Starting EVM nodes..."
    @go run -tags=run_evm scripts/run-evm-nodes.go --nodes={{ nodes }}

# ─── Tools ─────────────────────────────────────────────────────────────────────

# Build da-debug tool
[group('tools')]
build-tool-da-debug:
    @echo "--> Building da-debug tool"
    @mkdir -p {{ build_dir }}
    @cd tools/da-debug && go build -ldflags "{{ tool_ldflags }}" -o {{ build_dir }}/da-debug .
    @echo "--> da-debug built: {{ build_dir }}/da-debug"

# Build blob-decoder tool
[group('tools')]
build-tool-blob-decoder:
    @echo "--> Building blob-decoder tool"
    @mkdir -p {{ build_dir }}
    @cd tools/blob-decoder && go build -ldflags "{{ tool_ldflags }}" -o {{ build_dir }}/blob-decoder .
    @echo "--> blob-decoder built: {{ build_dir }}/blob-decoder"

# Build cache-analyzer tool
[group('tools')]
build-tool-cache-analyzer:
    @echo "--> Building cache-analyzer tool"
    @mkdir -p {{ build_dir }}
    @cd tools/cache-analyzer && go build -ldflags "{{ tool_ldflags }}" -o {{ build_dir }}/cache-analyzer .
    @echo "--> cache-analyzer built: {{ build_dir }}/cache-analyzer"

# Build all tools
[group('tools')]
build-tools: build-tool-da-debug build-tool-blob-decoder build-tool-cache-analyzer
    @echo "--> All tools built successfully!"

# Install da-debug tool to Go bin
[group('tools')]
install-tool-da-debug:
    @echo "--> Installing da-debug tool"
    @cd tools/da-debug && go install -ldflags "{{ tool_ldflags }}" .
    @echo "--> da-debug installed to Go bin"

# Install blob-decoder tool to Go bin
[group('tools')]
install-tool-blob-decoder:
    @echo "--> Installing blob-decoder tool"
    @cd tools/blob-decoder && go install -ldflags "{{ tool_ldflags }}" .
    @echo "--> blob-decoder installed to Go bin"

# Install cache-analyzer tool to Go bin
[group('tools')]
install-tool-cache-analyzer:
    @echo "--> Installing cache-analyzer tool"
    @cd tools/cache-analyzer && go install -ldflags "{{ tool_ldflags }}" .
    @echo "--> cache-analyzer installed to Go bin"

# Install all tools to Go bin
[group('tools')]
install-tools: install-tool-da-debug install-tool-blob-decoder install-tool-cache-analyzer
    @echo "--> All tools installed successfully!"

# Remove built tool binaries
[group('tools')]
clean-tools:
    @echo "--> Cleaning built tools"
    @rm -f {{ build_dir }}/da-debug {{ build_dir }}/blob-decoder {{ build_dir }}/cache-analyzer
    @echo "--> Tools cleaned"

# List available tools
[group('tools')]
list-tools:
    @echo "Available tools:"
    @echo "  - da-debug"
    @echo "  - blob-decoder"
    @echo "  - cache-analyzer"
