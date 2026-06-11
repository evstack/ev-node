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

```text
ev-node                                    ev-reth
   │                                          │
   │  1. engine_forkchoiceUpdatedV3/V4        │
   │     (headBlockHash, payloadAttributes)   │
   │─────────────────────────────────────────►│
   │                                          │
   │  2. {payloadId}                          │
   │◄─────────────────────────────────────────│
   │                                          │
   │  3. engine_getPayloadV4/V5/V6(payloadId) │
   │─────────────────────────────────────────►│
   │                                          │
   │  4. {executionPayload, blockValue}       │
   │◄─────────────────────────────────────────│
   │                                          │
   │  [ev-node broadcasts to P2P, submits DA] │
   │                                          │
   │  5. engine_newPayloadV4/V5(payload)      │
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

| Fork family | Forkchoice | Get payload | New payload |
|-------------|------------|-------------|-------------|
| Prague | `engine_forkchoiceUpdatedV3` | `engine_getPayloadV4` | `engine_newPayloadV4` |
| Osaka/Fusaka | `engine_forkchoiceUpdatedV3` | `engine_getPayloadV5` | `engine_newPayloadV4` |
| Amsterdam | `engine_forkchoiceUpdatedV4` | `engine_getPayloadV6` | `engine_newPayloadV5` |

Amsterdam support is more than a method rename: payload attributes include
`slotNumber`, and built payloads include `executionPayload.blockAccessList`.
ev-node preserves the raw Amsterdam `executionPayload` returned by ev-reth and
submits it unchanged to `engine_newPayloadV5`.
ev-node detects Amsterdam by retrying `engine_forkchoiceUpdatedV4` after an
unsupported-fork response from `engine_forkchoiceUpdatedV3`, then caches V4.

### engine_forkchoiceUpdatedV3 / V4

Update the fork choice and optionally start building a new block.

**Request:**

```json
{
  "method": "engine_forkchoiceUpdatedV4",
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
      "parentBeaconBlockRoot": "0x...",
      "slotNumber": "0x..."
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

### engine_getPayloadV4 / V5 / V6

Retrieve a built payload.

**Request:**

```json
{
  "method": "engine_getPayloadV6",
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
    "transactions": ["0x..."],
    "slotNumber": "0x...",
    "blockAccessList": []
  },
  "blockValue": "0x..."
}
```

### engine_newPayloadV4 / V5

Validate and execute a payload.

**Request:**

```json
{
  "method": "engine_newPayloadV5",
  "params": [
    {
      "parentHash": "0x...",
      "blockHash": "0x...",
      "slotNumber": "0x...",
      "blockAccessList": []
    },
    ["0x...versionedHashes"],
    "0x...parentBeaconBlockRoot",
    []
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
