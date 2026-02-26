# Engine API

ev-node communicates with ev-reth through the Ethereum Engine API, the same protocol used by Ethereum consensus clients.

## Overview

The Engine API is a JSON-RPC interface authenticated with JWT. ev-node acts as the consensus client, driving ev-reth (execution client) to build and finalize blocks.

## Authentication

All Engine API calls require JWT authentication:

```bash
# Generate shared secret
openssl rand -hex 32 > jwt.hex
```

Configure both sides:

- ev-reth: `--authrpc.jwtsecret jwt.hex`
- ev-node: `--evm.jwt-secret jwt.hex`

## Block Production Flow

```
ev-node                                    ev-reth
   │                                          │
   │  1. engine_forkchoiceUpdatedV3           │
   │     (headBlockHash, payloadAttributes)   │
   │─────────────────────────────────────────►│
   │                                          │
   │  2. {payloadId}                          │
   │◄─────────────────────────────────────────│
   │                                          │
   │  3. engine_getPayloadV3(payloadId)       │
   │─────────────────────────────────────────►│
   │                                          │
   │  4. {executionPayload, blockValue}       │
   │◄─────────────────────────────────────────│
   │                                          │
   │  [ev-node broadcasts to P2P, submits DA] │
   │                                          │
   │  5. engine_newPayloadV3(executionPayload)│
   │─────────────────────────────────────────►│
   │                                          │
   │  6. {status: VALID}                      │
   │◄─────────────────────────────────────────│
   │                                          │
   │  7. engine_forkchoiceUpdatedV3           │
   │     (newHeadBlockHash)                   │
   │─────────────────────────────────────────►│
   │                                          │
```

## Methods

### engine_forkchoiceUpdatedV3

Update the fork choice and optionally start building a new block.

**Request:**

```json
{
  "method": "engine_forkchoiceUpdatedV3",
  "params": [
    {
      "headBlockHash": "0x...",
      "safeBlockHash": "0x...",
      "finalizedBlockHash": "0x..."
    },
    {
      "timestamp": "0x...",
      "prevRandao": "0x...",
      "suggestedFeeRecipient": "0x...",
      "withdrawals": [],
      "parentBeaconBlockRoot": "0x..."
    }
  ]
}
```

**Response:**

```json
{
  "payloadStatus": {
    "status": "VALID",
    "latestValidHash": "0x..."
  },
  "payloadId": "0x..."
}
```

### engine_getPayloadV3

Retrieve a built payload.

**Request:**

```json
{
  "method": "engine_getPayloadV3",
  "params": ["0x...payloadId"]
}
```

**Response:**

```json
{
  "executionPayload": {
    "parentHash": "0x...",
    "feeRecipient": "0x...",
    "stateRoot": "0x...",
    "receiptsRoot": "0x...",
    "logsBloom": "0x...",
    "prevRandao": "0x...",
    "blockNumber": "0x1",
    "gasLimit": "0x...",
    "gasUsed": "0x...",
    "timestamp": "0x...",
    "extraData": "0x",
    "baseFeePerGas": "0x...",
    "blockHash": "0x...",
    "transactions": ["0x..."]
  },
  "blockValue": "0x..."
}
```

### engine_newPayloadV3

Validate and execute a payload.

**Request:**

```json
{
  "method": "engine_newPayloadV3",
  "params": [
    { "executionPayload": "..." },
    ["0x...versionedHashes"],
    "0x...parentBeaconBlockRoot"
  ]
}
```

**Response:**

```json
{
  "status": "VALID",
  "latestValidHash": "0x..."
}
```

## Status Codes

| Status | Meaning |
|--------|---------|
| `VALID` | Payload is valid |
| `INVALID` | Payload is invalid |
| `SYNCING` | Node is syncing |
| `ACCEPTED` | Payload accepted but not yet validated |

## Ports

| Port | Purpose |
|------|---------|
| 8545 | JSON-RPC (public) |
| 8551 | Engine API (authenticated) |

## Next Steps

- [Engine API Reference](/reference/api/engine-api) — Full method reference
- [Configuration](/ev-reth/configuration) — ev-reth settings
