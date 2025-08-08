# ev-types

Proto-generated types for Ev-node.

## Features

- `grpc` (enabled by default) - Includes gRPC client and server code
- `transport` - Enables tonic transport features

## Usage

### Default usage (with gRPC)

```toml
[dependencies]
ev-types = "0.0.1"
```

### Types only (without gRPC)

If you only need the message types without gRPC client/server code:

```toml
[dependencies]
ev-types = { version = "0.0.1", default-features = false }
```

This is useful when:

- You only need to serialize/deserialize Ev-node messages
- You're using a different RPC framework
- You want to minimize dependencies
- You're building for environments where gRPC is not needed

## How It Works

This crate generates two versions of the protobuf code:

1. **`evnode.v1.messages.rs`** - Contains only the message types (structs/enums) with no gRPC dependencies
2. **`evnode.v1.services.rs`** - Contains everything including gRPC client/server code

Both files are pre-generated and checked into the repository, so users don't need `protoc` installed or need to regenerate based on their feature selection.

## Building

The crate uses pre-generated proto files that are checked into version control. This ensures that the crate can be built from crates.io without requiring access to the original `.proto` files.

```bash
cargo build
```

The build script will:

1. Check if pre-generated files exist (`src/proto/evnode.v1.*.rs`)
2. If they exist, use them (this is the default behavior)
3. If they don't exist, attempt to generate them from source proto files

## Proto Generation

The generated code is committed to the repository. If you modify the proto files, you need to regenerate:

```bash
# Force regeneration by setting the environment variable
EV_TYPES_FORCE_PROTO_GEN=1 cargo build

# Or from the repository root (if a make target exists)
make rust-proto-gen
```

**Important**:

- The build process generates both `evnode.v1.messages.rs` and `evnode.v1.services.rs`
- Both files should be committed to ensure users can use the crate without needing to regenerate
- When publishing to crates.io, the pre-generated files are included in the package
- Users installing from crates.io will use the pre-generated files automatically

## Version Consistency

**Important**: The CI uses protoc version 25.1. If your local protoc version differs, you may see formatting differences in the generated files.

To check your protoc version:

```bash
protoc --version
```

To ensure consistency with CI:

1. Install protoc version 25.1
2. Use the provided script: `./client/scripts/generate-protos.sh`
3. Or use the Makefile: `make rust-proto-gen`

### Common Issues

If you see differences in generated files between local and CI:

- It's usually due to protoc version differences
- Different versions may format the generated code slightly differently
- The functionality remains the same, only formatting changes

To avoid this:

- Use the same protoc version as CI (25.1)
- Or accept the formatting from your version and update CI if needed

## Dependencies

This crate requires:

- `protoc` (Protocol Buffers compiler)
- `tonic-build` for code generation
- `prost` for runtime support

The build dependencies are specified in `Cargo.toml` and use workspace versions for consistency.
