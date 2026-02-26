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
