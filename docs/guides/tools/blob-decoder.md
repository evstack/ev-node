# Blob Decoder Tool

The blob decoder is a utility tool for decoding and inspecting blobs from Celestia (DA) layers. It provides both a web interface and API for decoding blob data into human-readable format.

## Overview

The blob decoder helps developers and operators inspect the contents of blobs submitted to DA layers. It can decode:

- Raw blob data (hex or base64 encoded)
- Block data structures
- Transaction payloads
- Protobuf-encoded messages

## Usage

### Starting the Server

```bash
# Run with default port (8080)
go run tools/blob-decoder/main.go
```

The server will start and display:

- Web interface URL: `http://localhost:8080`
- API endpoint: `http://localhost:8080/api/decode`

### Web Interface

1. Open your browser to `http://localhost:8080`
2. Paste your blob data in the input field
3. Select the encoding format (hex or base64)
4. Click "Decode" to see the parsed output

### API Usage

The decoder provides a REST API for programmatic access:

```bash
# Decode hex-encoded blob
curl -X POST http://localhost:8080/api/decode \
  -H "Content-Type: application/json" \
  -d '{
    "data": "0x1234abcd...",
    "encoding": "hex"
  }'

# Decode base64-encoded blob
curl -X POST http://localhost:8080/api/decode \
  -H "Content-Type: application/json" \
  -d '{
    "data": "SGVsbG8gV29ybGQ=",
    "encoding": "base64"
  }'
```

#### API Request Format

```json
{
  "data": "string",      // The encoded blob data
  "encoding": "string"   // Either "hex" or "base64"
}
```

#### API Response Format

```json
{
  "success": true,
  "decoded": {
    // Decoded data structure
  },
  "error": "string"  // Only present if success is false
}
```

## Supported Data Types

### Block Data

The decoder can parse ev-node block structures:

- Block height
- Timestamp
- Parent hash
- Transaction list
- Validator information
- Data commitments

### Transaction Data

Decodes individual transactions including:

- Transaction type
- Sender/receiver addresses
- Value/amount
- Gas parameters
- Payload data

### Protobuf Messages

Automatically detects and decodes protobuf-encoded messages used in ev-node:

- Block headers
- Transaction batches
- State updates
- DA commitments

## Examples

### Decoding a Block Blob

```bash
# Example block blob (hex encoded)
curl -X POST http://localhost:8080/api/decode \
  -H "Content-Type: application/json" \
  -d '{
    "data": "0a2408011220...",
    "encoding": "hex"
  }'
```

Response:

```json
{
  "success": true,
  "decoded": {
    "height": 100,
    "timestamp": "2024-01-15T10:30:00Z",
    "parentHash": "0xabc123...",
    "transactions": [
      {
        "type": "transfer",
        "from": "0x123...",
        "to": "0x456...",
        "value": "1000000000000000000"
      }
    ]
  }
}
```

### Decoding DA Commitment

```bash
curl -X POST http://localhost:8080/api/decode \
  -H "Content-Type: application/json" \
  -d '{
    "data": "eyJjb21taXRtZW50IjogIi4uLiJ9",
    "encoding": "base64"
  }'
```

### Celestia

For Celestia blobs, you can decode namespace data and payment information from [celenium](https://celenium.io/namespaces).
