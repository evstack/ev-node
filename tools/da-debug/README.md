# DA Debug Tool

A simple debugging tool for querying and decoding Data Availability (DA) layer data in ev-node.

## Overview

The `da-debug` tool provides a straightforward command-line interface to query a specific DA height and namespace, then automatically decode each blob found as either a SignedHeader, SignedData, or raw data, displaying detailed information about the contents.

## Usage

### Basic Command

```bash
da-debug [flags] <height> <namespace>
```

### Arguments

- `height`: DA height to query (integer)
- `namespace`: Namespace to query (hex string or string identifier)

### Flags

- `--da-url`: DA layer JSON-RPC URL (default: "http://localhost:7980")
- `--auth-token`: Authentication token for DA layer
- `--timeout`: Request timeout (default: 30s)
- `--verbose`: Enable verbose logging
- `--max-blob-size`: Maximum blob size in bytes (default: 1970176)
- `--gas-price`: Gas price for DA operations (default: 0.0)
- `--gas-multiplier`: Gas multiplier for DA operations (default: 1.0)

## Examples

### Basic Usage

```bash
# Query height 100 with string namespace
da-debug 100 "my-rollup"

# Query height 50 with hex namespace
da-debug 50 0x000000000000000000000000000000000000000000000000000000746573743031

# Enable verbose logging
da-debug --verbose 25 "test-namespace"
```

### With Custom DA Server

```bash
# Connect to remote DA server
da-debug --da-url http://celestia-node:26658 1000 "my-app"

# With authentication
da-debug --da-url https://da.example.com --auth-token "your-token" 500 "rollup-data"
```

### Advanced Options

```bash
# Custom timeout and max blob size
da-debug --timeout 60s --max-blob-size 2000000 100 "large-data"
```

## Output Format

The tool displays detailed information for each blob found:

### SignedHeader Blobs

```
=== Blob 1 ===
ID: 0x0100000000000000abcdef1234567890...
Size: 1024 bytes
ID Height: 1
ID Commitment: abcdef1234567890...
Type: SignedHeader
Header Details:
  Height: 100
  Time: 2024-01-15T10:30:45Z
  Chain ID: my-chain
  Version: Block=1, App=1
  Last Header Hash: 1234567890abcdef...
  Data Hash: abcdef1234567890...
  Proposer Address: fedcba0987654321...
  Signature: 987654321fedcba0...
```

### SignedData Blobs

```
=== Blob 2 ===
ID: 0x0200000000000000fedcba0987654321...
Size: 2048 bytes
Type: SignedData
Data Details:
  Chain ID: my-chain
  Height: 100
  Time: 2024-01-15T10:30:45Z
  Last Data Hash: 1234567890abcdef...
  DA Commitment Hash: abcdef1234567890...
  Transaction Count: 3
  Signature: 987654321fedcba0...
  Transactions:
    Tx 1: Size=256 bytes, Hash=abcd1234...
    Tx 2: Size=128 bytes, Hash=5678efgh...
    Tx 3: Size=64 bytes, Hash=ijkl9012...
```

### Raw/Unknown Blobs

```
=== Blob 3 ===
ID: 0x0300000000000000...
Size: 512 bytes
Type: Unknown/Raw Data
Raw Data:
  Hex: 48656c6c6f20576f726c64...
  String: Hello World...
```

## Namespace Format

Namespaces can be provided in two formats:

1. **Hex String**: A 29-byte hex string (with or without `0x` prefix)
   - Example: `0x000000000000000000000000000000000000000000000000000000746573743031`

2. **String Identifier**: Any string that gets automatically converted to a valid namespace
   - Example: `"my-app"` or `"test-namespace"`
   - The string is hashed and converted to a valid version 0 namespace

## Integration with ev-node

### Using with Local DA Server

Start a local DA server:

```bash
# From ev-node root directory
./build/local-da --port 7980
```

Then query it:

```bash
./build/da-debug 1 "test-data"
```

### Development Workflow

```bash
# Start local DA in background
./build/local-da --port 7980 &

# Query recent data
./build/da-debug 1 "my-namespace"

# Stop DA server
pkill local-da
```

## Error Handling

The tool provides clear error messages for common scenarios:

- **No blobs found**: `No blobs found at height X with namespace Y`
- **Future height**: `Height X is in the future (not yet available)`
- **Invalid height**: `invalid height: strconv.ParseUint: parsing "abc": invalid syntax`
- **Connection failed**: Network connectivity issues are handled gracefully
- **Invalid namespace**: Malformed hex strings are rejected

## Debugging Tips

### Common Issues

1. **Connection refused**: Ensure the DA server is running on the specified URL
2. **No data found**: Check that data was submitted to the correct namespace and height
3. **Timeout errors**: Increase timeout with `--timeout 60s`
4. **Invalid namespace**: Use string identifiers for automatic conversion

### Verbose Mode

Enable verbose logging to see all RPC calls and responses:

```bash
da-debug --verbose 100 "debug-namespace"
```

### Testing Connectivity

```bash
# Test basic connectivity
da-debug 0 "test"

# Test with different timeout
da-debug --timeout 5s 0 "test"
```

## Performance Considerations

- The tool fetches all blob IDs first, then retrieves the actual blob data
- Large blobs are handled efficiently with streaming
- Timeout settings can be adjusted for slow networks
- Verbose mode adds logging overhead

## Security Notes

- Authentication tokens should be kept secure
- Use HTTPS URLs for production DA servers
- The tool only reads data, never writes or modifies DA layer state

## Contributing

When modifying the tool:

1. Follow Go best practices and the project's coding standards
2. Add tests for new functionality
3. Update this README for interface changes
4. Run integration tests before submitting changes

## Troubleshooting

### Build Issues

```bash
# Clean and rebuild
cd tools/da-debug
go clean
go mod tidy
go build .
```

### Runtime Issues

```bash
# Check DA server is accessible
curl -X POST http://localhost:7980 -H "Content-Type: application/json" -d '{"jsonrpc":"2.0","id":1,"method":"da.GasPrice","params":[]}'

# Test with minimal command
da-debug 0 "test"
```
