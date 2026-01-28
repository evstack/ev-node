# ev-abci Flags Reference

Command-line flags for Cosmos SDK applications using ev-abci.

## ev-node Flags

These flags configure the underlying ev-node instance.

### Node Configuration

| Flag                            | Type     | Default | Description                  |
|---------------------------------|----------|---------|------------------------------|
| `--evnode.node.aggregator`      | bool     | `false` | Run as block producer        |
| `--evnode.node.block_time`      | duration | `1s`    | Block production interval    |
| `--evnode.node.lazy_aggregator` | bool     | `false` | Only produce blocks with txs |
| `--evnode.node.lazy_block_time` | duration | `1s`    | Max wait in lazy mode        |

### DA Configuration

| Flag                     | Type   | Default  | Description             |
|--------------------------|--------|----------|-------------------------|
| `--evnode.da.address`    | string | required | DA layer URL            |
| `--evnode.da.auth_token` | string | `""`     | DA authentication token |
| `--evnode.da.namespace`  | string | `""`     | DA namespace (hex)      |
| `--evnode.da.gas_price`  | float  | `0.01`   | DA gas price            |

### P2P Configuration

| Flag                   | Type   | Default                  | Description                    |
|------------------------|--------|--------------------------|--------------------------------|
| `--evnode.p2p.listen`  | string | `/ip4/0.0.0.0/tcp/26656` | P2P listen address             |
| `--evnode.p2p.peers`   | string | `""`                     | Comma-separated peer addresses |
| `--evnode.p2p.blocked` | string | `""`                     | Blocked peer IDs               |

### Signer Configuration

| Flag                         | Type   | Default  | Description           |
|------------------------------|--------|----------|-----------------------|
| `--evnode.signer.passphrase` | string | required | Signer key passphrase |

### RPC Configuration

| Flag                   | Type   | Default               | Description        |
|------------------------|--------|-----------------------|--------------------|
| `--evnode.rpc.address` | string | `tcp://0.0.0.0:26657` | RPC listen address |

## Cosmos SDK Flags

Standard Cosmos SDK flags remain available:

| Flag           | Description                          |
|----------------|--------------------------------------|
| `--home`       | Application home directory           |
| `--log_level`  | Log level (debug, info, warn, error) |
| `--log_format` | Log format (plain, json)             |
| `--trace`      | Enable full stack traces             |

## Environment Variables

Flags can be set via environment variables:

```bash
EVNODE_NODE_AGGREGATOR=true
EVNODE_DA_ADDRESS=http://localhost:7980
EVNODE_SIGNER_PASSPHRASE=secret
```

Pattern: `EVNODE_<SECTION>_<FLAG>` (uppercase, underscores)

## Examples

### Sequencer Node

```bash
appd start \
  --evnode.node.aggregator \
  --evnode.node.block_time 500ms \
  --evnode.da.address http://localhost:7980 \
  --evnode.signer.passphrase secret
```

### Full Node

```bash
appd start \
  --evnode.da.address http://localhost:7980 \
  --evnode.p2p.peers 12D3KooW...@sequencer.example.com:26656
```

### Lazy Aggregator

```bash
appd start \
  --evnode.node.aggregator \
  --evnode.node.lazy_aggregator \
  --evnode.node.lazy_block_time 5s \
  --evnode.da.address http://localhost:7980 \
  --evnode.signer.passphrase secret
```
