# ABCI RPC Reference

CometBFT-compatible RPC endpoints provided by ev-abci.

## Query Methods

### /abci_query

Query application state.

**Request:**

```bash
curl 'http://localhost:26657/abci_query?path="/store/bank/key"&data=0x...'
```

**Response:**

```json
{
  "jsonrpc": "2.0",
  "result": {
    "response": {
      "code": 0,
      "value": "base64encodedvalue",
      "height": "1000"
    }
  },
  "id": 1
}
```

### /block

Get block at height.

**Request:**

```bash
curl 'http://localhost:26657/block?height=100'
```

### /block_results

Get block results (tx results, events).

**Request:**

```bash
curl 'http://localhost:26657/block_results?height=100'
```

### /commit

Get commit (signatures) at height.

**Request:**

```bash
curl 'http://localhost:26657/commit?height=100'
```

### /validators

Get validator set (returns sequencer in Evolve).

**Request:**

```bash
curl 'http://localhost:26657/validators?height=100'
```

### /status

Get node status.

**Request:**

```bash
curl 'http://localhost:26657/status'
```

### /genesis

Get genesis document.

**Request:**

```bash
curl 'http://localhost:26657/genesis'
```

### /health

Health check.

**Request:**

```bash
curl 'http://localhost:26657/health'
```

## Transaction Methods

### /broadcast_tx_async

Broadcast transaction, return immediately.

**Request:**

```bash
curl 'http://localhost:26657/broadcast_tx_async?tx=0x...'
```

### /broadcast_tx_sync

Broadcast transaction, wait for CheckTx.

**Request:**

```bash
curl 'http://localhost:26657/broadcast_tx_sync?tx=0x...'
```

### /broadcast_tx_commit

Broadcast transaction, wait for inclusion.

**Request:**

```bash
curl 'http://localhost:26657/broadcast_tx_commit?tx=0x...'
```

### /tx

Get transaction by hash.

**Request:**

```bash
curl 'http://localhost:26657/tx?hash=0x...'
```

### /tx_search

Search transactions.

**Request:**

```bash
curl 'http://localhost:26657/tx_search?query="tx.height=100"'
```

## WebSocket

### /subscribe

Subscribe to events.

```json
{
  "jsonrpc": "2.0",
  "method": "subscribe",
  "params": {"query": "tm.event='NewBlock'"},
  "id": 1
}
```

Event types:

- `NewBlock` — New block committed
- `Tx` — Transaction included
- `NewBlockHeader` — New block header

## Unsupported Methods

These CometBFT methods are not supported in ev-abci:

| Method | Reason |
|--------|--------|
| `/consensus_state` | No BFT consensus |
| `/dump_consensus_state` | No BFT consensus |
| `/net_info` | Different P2P model |
| `/unconfirmed_txs` | Different mempool |
| `/num_unconfirmed_txs` | Different mempool |

## Port

Default: `26657`

Configure:

```bash
--evnode.rpc.address tcp://0.0.0.0:26657
```
