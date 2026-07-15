# Engine API Reference

Engine API methods used by ev-node to communicate with ev-reth.

## Authentication

All requests require JWT authentication via the `Authorization` header:

```text
Authorization: Bearer <jwt_token>
```

Generate JWT from shared secret:

```bash
openssl rand -hex 32 > jwt.hex
```

## Methods

ev-node selects Engine API methods by fork:

| Fork family | Forkchoice | Get payload | New payload |
|-------------|------------|-------------|-------------|
| Prague | `engine_forkchoiceUpdatedV3` | `engine_getPayloadV4` | `engine_newPayloadV4` |
| Osaka/Fusaka | `engine_forkchoiceUpdatedV3` | `engine_getPayloadV5` | `engine_newPayloadV4` |
| Amsterdam | `engine_forkchoiceUpdatedV4` | `engine_getPayloadV6` | `engine_newPayloadV5` |

### engine_forkchoiceUpdatedV3 / V4

Update fork choice and optionally build a new block.

Payload builds start with `engine_forkchoiceUpdatedV3`. If the execution layer
returns unsupported fork, ev-node retries `engine_forkchoiceUpdatedV4` with
`slotNumber` in the payload attributes and caches V4 for future calls.
`slotNumber` is derived deterministically from rollup block height.

**Request:**

```json
{
  "jsonrpc": "2.0",
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
  ],
  "id": 1
}
```

**Response:**

```json
{
  "jsonrpc": "2.0",
  "result": {
    "payloadStatus": {
      "status": "VALID",
      "latestValidHash": "0x..."
    },
    "payloadId": "0x..."
  },
  "id": 1
}
```

### engine_getPayloadV4 / V5 / V6

Get a built payload.

ev-node starts with `engine_getPayloadV4`, caches `engine_getPayloadV5` or
`engine_getPayloadV6` after successful unsupported-fork fallback, and switches
directly to V6 after an Amsterdam `forkchoiceUpdatedV4` build request.

**Request:**

```json
{
  "jsonrpc": "2.0",
  "method": "engine_getPayloadV6",
  "params": ["0x...payloadId"],
  "id": 1
}
```

**Response:**

```json
{
  "jsonrpc": "2.0",
  "result": {
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
      "withdrawals": [],
      "blobGasUsed": "0x0",
      "excessBlobGas": "0x0",
      "slotNumber": "0x...",
      "blockAccessList": []
    },
    "blockValue": "0x...",
    "blobsBundle": {
      "commitments": [],
      "proofs": [],
      "blobs": []
    },
    "shouldOverrideBuilder": false
  },
  "id": 1
}
```

### engine_newPayloadV4 / V5

Validate and execute a payload.

Amsterdam payloads use `engine_newPayloadV5`. ev-node passes through the raw
`executionPayload` object returned by `engine_getPayloadV6` so
`blockAccessList` is preserved.

**Request:**

```json
{
  "jsonrpc": "2.0",
  "method": "engine_newPayloadV5",
  "params": [
    {
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
      "withdrawals": [],
      "blobGasUsed": "0x0",
      "excessBlobGas": "0x0",
      "slotNumber": "0x...",
      "blockAccessList": []
    },
    ["0x...expectedBlobVersionedHashes"],
    "0x...parentBeaconBlockRoot",
    []
  ],
  "id": 1
}
```

**Response:**

```json
{
  "jsonrpc": "2.0",
  "result": {
    "status": "VALID",
    "latestValidHash": "0x...",
    "validationError": null
  },
  "id": 1
}
```

## Payload Status

| Status | Description |
|--------|-------------|
| `VALID` | Payload is valid |
| `INVALID` | Payload failed validation |
| `SYNCING` | Node is syncing, cannot validate |
| `ACCEPTED` | Payload accepted, validation pending |
| `INVALID_BLOCK_HASH` | Block hash mismatch |

## Ports

| Port | Purpose |
|------|---------|
| 8551 | Engine API (authenticated) |
| 8545 | JSON-RPC (public) |
