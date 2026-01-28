# RPC Endpoints Reference

ev-node JSON-RPC endpoints.

## Health

### GET /health

Check node health.

**Response:**

```json
{
  "status": "ok"
}
```

## Block Queries

### POST /block

Get block by height.

**Request:**

```json
{
  "jsonrpc": "2.0",
  "method": "block",
  "params": { "height": "100" },
  "id": 1
}
```

**Response:**

```json
{
  "jsonrpc": "2.0",
  "result": {
    "block": {
      "header": {
        "height": "100",
        "time": "2024-01-01T00:00:00Z",
        "last_header_hash": "0x...",
        "data_hash": "0x...",
        "app_hash": "0x...",
        "proposer_address": "0x..."
      },
      "data": {
        "txs": ["0x..."]
      }
    }
  },
  "id": 1
}
```

### POST /header

Get header by height.

**Request:**

```json
{
  "jsonrpc": "2.0",
  "method": "header",
  "params": { "height": "100" },
  "id": 1
}
```

### POST /block_by_hash

Get block by hash.

**Request:**

```json
{
  "jsonrpc": "2.0",
  "method": "block_by_hash",
  "params": { "hash": "0x..." },
  "id": 1
}
```

## Transaction Queries

### POST /tx

Get transaction by hash.

**Request:**

```json
{
  "jsonrpc": "2.0",
  "method": "tx",
  "params": { "hash": "0x..." },
  "id": 1
}
```

## Status

### POST /status

Get node status.

**Response:**

```json
{
  "jsonrpc": "2.0",
  "result": {
    "node_info": {
      "network": "chain-id",
      "version": "1.0.0"
    },
    "sync_info": {
      "latest_block_height": "1000",
      "latest_block_time": "2024-01-01T00:00:00Z",
      "catching_up": false
    }
  },
  "id": 1
}
```

## DA Status

### POST /da_status

Get DA layer status.

**Response:**

```json
{
  "jsonrpc": "2.0",
  "result": {
    "da_height": "5000",
    "last_submitted_height": "999",
    "pending_blocks": 1
  },
  "id": 1
}
```

## Configuration

Default port: `26657`

Configure via flag:

```bash
--evnode.rpc.address tcp://0.0.0.0:26657
```

## WebSocket

Subscribe to events via WebSocket at `ws://localhost:26657/websocket`.

### Subscribe to new blocks

```json
{
  "jsonrpc": "2.0",
  "method": "subscribe",
  "params": { "query": "tm.event='NewBlock'" },
  "id": 1
}
```
