# Build

# Helpers

[private]
_build_go dir out pkg:
    @mkdir -p {{ build_dir }}
    @cd {{ dir }} && go build -ldflags "{{ ldflags }}" -o {{ build_dir }}/{{ out }} {{ pkg }}

[private]
_install_go dir:
    @cd {{ dir }} && go install -ldflags "{{ ldflags }}" .

[private]
_build_tool dir out:
    @mkdir -p {{ build_dir }}
    @cd {{ dir }} && go build -ldflags "{{ tool_ldflags }}" -o {{ build_dir }}/{{ out }} .

[private]
_install_tool dir:
    @cd {{ dir }} && go install -ldflags "{{ tool_ldflags }}" .

# Build Testapp CLI
[group('build')]
build:
    @echo "--> Building Testapp CLI"
    @just _build_go "apps/testapp" "testapp" "."
    @echo "--> Testapp CLI Built!"
    @echo "    Check the version with: build/testapp version"
    @echo "    Check the binary with: {{ build_dir }}/testapp"

# Install Testapp CLI to Go bin directory
[group('build')]
install:
    @echo "--> Installing Testapp CLI"
    @just _install_go "apps/testapp"
    @echo "--> Testapp CLI Installed!"
    @echo "    Check the version with: testapp version"
    @echo "    Check the binary with: which testapp"

# Build all ev-node binaries
[group('build')]
build-all:
    @echo "--> Building all ev-node binaries"
    @echo "--> Building testapp"
    @just _build_go "apps/testapp" "testapp" "."
    @echo "--> Building evm"
    @just _build_go "apps/evm" "evm" "."
    @echo "--> Building local-da"
    @just _build_go "tools/local-da" "local-da" "."
    @echo "--> All ev-node binaries built!"

# Build Testapp Bench
[group('build')]
build-testapp-bench:
    @echo "--> Building Testapp Bench"
    @just _build_go "apps/testapp" "testapp-bench" "./kv/bench"
    @echo "    Check the binary with: {{ build_dir }}/testapp-bench"

# Build EVM binary
[group('build')]
build-evm:
    @echo "--> Building EVM"
    @just _build_go "apps/evm" "evm" "."
    @echo "    Check the binary with: {{ build_dir }}/evm"

# Build local-da binary
[group('build')]
build-da:
    @echo "--> Building local-da"
    @just _build_go "tools/local-da" "local-da" "."
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

# Tools

# Build da-debug tool
[group('tools')]
build-tool-da-debug:
    @echo "--> Building da-debug tool"
    @just _build_tool "tools/da-debug" "da-debug"
    @echo "--> da-debug built: {{ build_dir }}/da-debug"

# Build blob-decoder tool
[group('tools')]
build-tool-blob-decoder:
    @echo "--> Building blob-decoder tool"
    @just _build_tool "tools/blob-decoder" "blob-decoder"
    @echo "--> blob-decoder built: {{ build_dir }}/blob-decoder"

# Build cache-analyzer tool
[group('tools')]
build-tool-cache-analyzer:
    @echo "--> Building cache-analyzer tool"
    @just _build_tool "tools/cache-analyzer" "cache-analyzer"
    @echo "--> cache-analyzer built: {{ build_dir }}/cache-analyzer"

# Build all tools
[group('tools')]
build-tools: build-tool-da-debug build-tool-blob-decoder build-tool-cache-analyzer
    @echo "--> All tools built successfully!"

# Install da-debug tool to Go bin
[group('tools')]
install-tool-da-debug:
    @echo "--> Installing da-debug tool"
    @just _install_tool "tools/da-debug"
    @echo "--> da-debug installed to Go bin"

# Install blob-decoder tool to Go bin
[group('tools')]
install-tool-blob-decoder:
    @echo "--> Installing blob-decoder tool"
    @just _install_tool "tools/blob-decoder"
    @echo "--> blob-decoder installed to Go bin"

# Install cache-analyzer tool to Go bin
[group('tools')]
install-tool-cache-analyzer:
    @echo "--> Installing cache-analyzer tool"
    @just _install_tool "tools/cache-analyzer"
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
