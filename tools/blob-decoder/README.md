# Blob Decoder

A web-based tool for decoding and inspecting blob data from the ev-node Data Availability layer.

## Overview

The `blob-decoder` tool provides a web interface for decoding protobuf-encoded blobs used in ev-node. It can decode blocks and other data structures from both hex and base64 encoded inputs, making it useful for debugging and inspecting DA layer data.

## Installation

Install using `go install`:

```bash
go install github.com/evstack/ev-node/tools/blob-decoder@main
```

After installation, the `blob-decoder` binary will be available in your `$GOPATH/bin` directory.

## Usage

### Running the Server

Start the blob decoder server:

```bash
# Run on default port 8090
blob-decoder

# Run on custom port
blob-decoder 8080
```

The server will start and display:

```
╔═══════════════════════════════════╗
║        Evolve Blob Decoder        ║
║           by ev.xyz               ║
╚═══════════════════════════════════╝

🚀 Server running at: http://localhost:8090
⚡ Using native Evolve protobuf decoding

Press Ctrl+C to stop the server
```

### Web Interface

Open your browser and navigate to `http://localhost:8090` to access the web interface.

The interface allows you to:
- Paste blob data in either hex or base64 format
- Automatically detect and decode the blob type
- View decoded block headers, transactions, and other metadata
- Inspect raw hex data and blob size information

### API Endpoint

The decoder also provides a REST API endpoint:

```bash
POST /api/decode
Content-Type: application/json

{
  "data": "<hex-or-base64-encoded-blob>",
  "encoding": "hex" | "base64"
}
```

Response:
```json
{
  "type": "Block",
  "data": {
    // Decoded block or data structure
  },
  "rawHex": "<hex-representation>",
  "size": 1234,
  "timestamp": "2024-01-01T00:00:00Z"
}
```

## Features

- **Auto-detection**: Automatically detects blob type (blocks, commits, etc.)
- **Multiple Encodings**: Supports both hex and base64 encoded inputs
- **Protobuf Decoding**: Native decoding of ev-node protobuf structures
- **Web Interface**: User-friendly web interface for easy blob inspection
- **REST API**: Programmatic access for automation and integration
- **CORS Support**: Can be accessed from other web applications

## Examples

```bash
# Start server on default port
blob-decoder

# Start server on port 3000
blob-decoder 3000

# Use with curl to decode a hex-encoded blob
curl -X POST http://localhost:8090/api/decode \
  -H "Content-Type: application/json" \
  -d '{"data": "0x...", "encoding": "hex"}'
```

## Use Cases

- Debug DA layer blob submissions
- Inspect block structure and content
- Validate blob encoding and decoding
- Troubleshoot sync issues related to blob data
- Educational tool for understanding ev-node data structures
