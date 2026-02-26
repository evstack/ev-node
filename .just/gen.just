# Protobuf

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

# Codegen

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
