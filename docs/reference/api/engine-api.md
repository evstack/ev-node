# Engine API Reference

Engine API methods used by ev-node to communicate with ev-reth.

## Authentication

All requests require JWT authentication via the `Authorization` header:

```
Authorization: Bearer <jwt_token>
```

Generate JWT from shared secret:

```bash
openssl rand -hex 32 > jwt.hex
```

## Methods

### engine_forkchoiceUpdatedV3

Update fork choice and optionally build a new block.

**Request:**

```json
{
  "jsonrpc": "2.0",
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

### engine_getPayloadV3

Get a built payload.

**Request:**

```json
{
  "jsonrpc": "2.0",
  "method": "engine_getPayloadV3",
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
      "excessBlobGas": "0x0"
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

### engine_newPayloadV3

Validate and execute a payload.

**Request:**

```json
{
  "jsonrpc": "2.0",
  "method": "engine_newPayloadV3",
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
      "excessBlobGas": "0x0"
    },
    ["0x...expectedBlobVersionedHashes"],
    "0x...parentBeaconBlockRoot"
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
