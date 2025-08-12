# CLAUDE.md - Types Package

This file provides guidance to Claude Code when working with the types package of ev-node.

## Package Overview

The types package defines the core data structures and types used throughout ev-node. It provides the fundamental building blocks for blocks, headers, transactions, state management, and serialization. This package is designed to be dependency-free and serves as the common type system for all other packages.

## Core Data Structures

### Header (`header.go`)

- **Purpose**: Defines the block header structure and validation
- **Key Types**:
  - `Header`: Complete block header with all metadata
  - `BaseHeader`: Minimal header data (height, time, chain ID)
  - `Hash`: 32-byte hash representation
- **Key Features**:
  - Header validation and verification
  - Chain ID and version tracking
  - Proposer address management
  - Hash computation for all header components
- **Context Usage**: Headers can be stored in context for execution access

### Data (`data.go`)

- **Purpose**: Defines block data structures and validation
- **Key Types**:
  - `Data`: Block data containing transactions
  - `Metadata`: Block metadata for P2P gossiping
  - `SignedData`: Data with signature and signer information
  - `Signature`: Block creator signature
  - `Version`: Consensus version tracking
- **Key Features**:
  - Binary marshaling/unmarshaling
  - Data validation against headers
  - Signature validation
  - Transaction organization

### State (`state.go`)

- **Purpose**: Manages blockchain state and transitions
- **Key Types**:
  - `State`: Current blockchain state
  - `InitStateVersion`: Initial version configuration
- **Key Features**:
  - Genesis state initialization
  - State transitions
  - DA height tracking
  - App hash management
  - Results hash tracking
- **Important**: Block version is set to 11 for CometBFT/IBC compatibility

### Transactions (`tx.go`)

- **Purpose**: Transaction representation
- **Key Types**:
  - `Tx`: Individual transaction as byte slice
  - `Txs`: Slice of transactions
- **Design**: Transactions are opaque byte arrays, allowing flexibility for different transaction formats

### SignedHeader (`signed_header.go`)

- **Purpose**: Headers with signatures for authentication
- **Key Features**:
  - Header signing and verification
  - Signer identification
  - Chain of trust validation
  - P2P header exchange

### Signer (`signer.go`, `signer_test.go`)

- **Purpose**: Identity and signature management
- **Key Types**:
  - `Signer`: Represents a block signer/validator
- **Key Features**:
  - Public key management
  - Signature verification
  - Identity validation

### DA Integration (`da.go`, `da_test.go`)

- **Purpose**: Data Availability layer helpers
- **Key Functions**:
  - `SubmitWithHelpers`: DA submission with error handling
- **Key Features**:
  - Error mapping to status codes
  - Namespace support
  - Gas price configuration
  - Submission options handling
- **Status Codes**:
  - `StatusContextCanceled`: Submission canceled
  - `StatusNotIncludedInBlock`: Transaction timeout
  - `StatusAlreadyInMempool`: Duplicate transaction
  - `StatusIncorrectAccountSequence`: Nonce mismatch
  - `StatusTooBig`: Blob size exceeded
  - `StatusContextDeadline`: Deadline exceeded

### Serialization (`serialization.go`, `serialization_test.go`)

- **Purpose**: Binary serialization for network and storage
- **Key Features**:
  - Protobuf integration
  - Binary marshaling/unmarshaling
  - Version compatibility
  - Efficient encoding

### Hashing (`hashing.go`, `hasher.go`, `hashing_test.go`)

- **Purpose**: Cryptographic hashing utilities
- **Key Features**:
  - Merkle tree operations
  - Hash computation for all types
  - Consistent hashing across the system
  - Hash verification

### Utilities (`utils.go`, `utils_test.go`)

- **Purpose**: Common utility functions
- **Key Features**:
  - Type conversions
  - Validation helpers
  - Common operations

## Protobuf Definitions (`pb/`)

### Directory Structure

```txt
pb/evnode/v1/
├── batch.pb.go         # Batch processing messages
├── evnode.pb.go        # Core ev-node messages
├── execution.pb.go     # Execution-related messages
├── health.pb.go        # Health check messages
├── p2p_rpc.pb.go      # P2P RPC messages
├── signer.pb.go        # Signer messages
├── state.pb.go         # State messages
├── state_rpc.pb.go     # State RPC messages
└── v1connect/          # Connect RPC implementations
```

### Connect RPC Services

- **Execution Service**: Transaction execution RPC
- **Health Service**: Node health monitoring
- **P2P RPC Service**: Peer-to-peer communication
- **Signer Service**: Signing operations
- **State RPC Service**: State queries and updates

## Type System Design Principles

### Immutability

- Core types should be immutable where possible
- Use value semantics for small types
- Deep copy when modification is needed

### Validation

- All types should have `ValidateBasic()` methods
- Validation should be fail-fast
- Return descriptive errors

### Serialization

- All network types must be serializable
- Use protobuf for cross-language compatibility
- Maintain backward compatibility

### Zero Dependencies

- Types package should not import other ev-node packages
- Exception: core package for interfaces
- Keep external dependencies minimal

## Common Development Tasks

### Adding a New Type

1. Define the type structure
2. Implement validation methods
3. Add serialization support
4. Create protobuf definition if needed
5. Write comprehensive tests
6. Update related types if necessary

### Modifying Existing Types

1. Consider backward compatibility
2. Update protobuf definitions
3. Regenerate protobuf code
4. Update validation logic
5. Fix all compilation errors
6. Update tests

### Adding Protobuf Messages

1. Create/modify `.proto` files
2. Run protobuf generation
3. Implement type conversions
4. Add validation
5. Test serialization round-trip

## Testing Guidelines

### Unit Tests

- Test all validation paths
- Test serialization round-trips
- Test edge cases and error conditions
- Use table-driven tests

### Property-Based Testing

- Test invariants hold
- Test serialization properties
- Test hash consistency

## Performance Considerations

- Keep types lightweight
- Avoid unnecessary allocations
- Cache computed values (like hashes)
- Use efficient serialization
- Consider memory alignment

## Security Considerations

- Validate all external inputs
- Prevent integer overflows
- Secure hash computations
- Validate signatures properly
- Check bounds on all arrays/slices

## Debugging Tips

- Use structured logging with type fields
- Implement String() methods for debugging
- Add JSON tags for inspection
- Validate types at boundaries
- Check for nil pointers

## Common Patterns

### Validation Pattern

```go
func (t *Type) ValidateBasic() error {
    // Check required fields
    // Validate ranges
    // Check consistency
    return nil
}
```

### Serialization Pattern

```go
func (t *Type) MarshalBinary() ([]byte, error)
func (t *Type) UnmarshalBinary(data []byte) error
```

### Hash Pattern

```go
func (t *Type) Hash() Hash {
    // Compute and cache hash
    return hash
}
```

## Version Compatibility

- Block version 11 for IBC compatibility
- App version negotiated during handshake
- Maintain version checks in validation
- Document version changes

## Integration Points

- Used by all other packages
- Critical for P2P communication
- Storage layer dependency
- Execution client interface
- DA layer integration
