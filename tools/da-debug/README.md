# DA Debug Tool

A simple debugging tool for querying and decoding Data Availability (DA) layer data in ev-node.

## Overview

The `da-debug` tool provides a straightforward command-line interface to query a specific DA height and namespace, then automatically decode each blob found as either a SignedHeader, SignedData, or raw data, displaying detailed information about the contents.

## Usage

### Basic Command

```bash
da-debug [flags] <height> <namespace>
```

## Examples

### Basic Usage

```bash
# Query height 100 with string namespace
da-debug 100 "my-rollup"

# Query height 50 with hex namespace
da-debug 50 0x000000000000000000000000000000000000000000000000000000746573743031

# Enable verbose logging
da-debug --verbose 25 "test-namespace"

# Filter for blobs containing data from height 12345
da-debug --filter-height 12345 100 "my-rollup"
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

# Disable colors for scripting
da-debug --no-color 100 "my-rollup"
```

#### Namespaces

Namespaces can be provided in two formats:

1. **Hex String**: A 29-byte hex string (with or without `0x` prefix)
   - Example: `0x000000000000000000000000000000000000000000000000000000746573743031`

2. **String Identifier**: Any string that gets automatically converted to a valid namespace
   - Example: `"my-app"` or `"test-namespace"`
   - The string is hashed and converted to a valid version 0 namespace
