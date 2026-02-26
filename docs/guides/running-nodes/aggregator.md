# Aggregator Node

An aggregator (also called sequencer) is the node responsible for producing blocks in an Evolve chain. It collects transactions, orders them, creates blocks, and submits data to the DA layer.

## Prerequisites

- ev-node installed
- Access to a DA layer (Celestia or local-da)
- Signer key for block signing

## Configuration

Enable aggregator mode with the `--evnode.node.aggregator` flag:

```bash
evnode start --evnode.node.aggregator
```

### Required Flags

| Flag                         | Description             |
|------------------------------|-------------------------|
| `--evnode.node.aggregator`   | Enable block production |
| `--evnode.da.address`        | DA layer endpoint       |
| `--evnode.signer.passphrase` | Signer key passphrase   |

### Block Time Configuration

Control how often blocks are produced:

```bash
# Produce blocks every 500ms
evnode start \
  --evnode.node.aggregator \
  --evnode.node.block_time 500ms
```

Default block time is 1 second.

## Lazy Aggregation Mode

Lazy mode only produces blocks when there are transactions, reducing DA costs during low activity periods:

```bash
evnode start \
  --evnode.node.aggregator \
  --evnode.node.lazy_aggregator \
  --evnode.node.lazy_block_time 30s
```

| Flag                            | Description                          |
|---------------------------------|--------------------------------------|
| `--evnode.node.lazy_aggregator` | Enable lazy mode                     |
| `--evnode.node.lazy_block_time` | Max wait between blocks in lazy mode |

In lazy mode:

- Blocks are produced immediately when transactions arrive
- If no transactions, wait up to `lazy_block_time` before producing an empty block
- Reduces DA submission costs during idle periods

## DA Submission Settings

Configure how blocks are batched and submitted to DA:

```bash
evnode start \
  --evnode.node.aggregator \
  --evnode.da.address http://localhost:26658 \
  --evnode.da.namespace "my_namespace" \
  --evnode.da.gas_price 0.01 \
  --evnode.da.batching_strategy adaptive
```

### Batching Strategies

| Strategy    | Description                                 |
|-------------|---------------------------------------------|
| `immediate` | Submit as soon as blocks are ready          |
| `time`      | Wait for time interval before submitting    |
| `size`      | Wait until batch reaches size threshold     |
| `adaptive`  | Balance between size and time (recommended) |

### Max Pending Blocks

Limit how many blocks can be waiting for DA submission:

```bash
--evnode.node.max_pending_blocks 100
```

When this limit is reached, block production pauses until some blocks are confirmed on DA.

## Signer Configuration

The aggregator needs a signer key to sign blocks:

```bash
# Using file-based signer
evnode start \
  --evnode.node.aggregator \
  --evnode.signer.signer_type file \
  --evnode.signer.signer_path /path/to/keys \
  --evnode.signer.passphrase "your-passphrase"
```

## Complete Example

### EVM Chain (ev-reth)

```bash
evnode start \
  --evnode.node.aggregator \
  --evnode.node.block_time 1s \
  --evnode.da.address http://localhost:26658 \
  --evnode.da.namespace "my_evm_chain" \
  --evnode.da.gas_price 0.01 \
  --evnode.signer.passphrase "secret" \
  --evnode.rpc.address tcp://0.0.0.0:26657
```

### Cosmos SDK Chain (ev-abci)

```bash
appd start \
  --evnode.node.aggregator \
  --evnode.node.block_time 1s \
  --evnode.da.address http://localhost:26658 \
  --evnode.da.namespace "my_cosmos_chain" \
  --evnode.signer.passphrase "secret"
```

## Monitoring

Enable metrics to monitor aggregator performance:

```bash
evnode start \
  --evnode.node.aggregator \
  --evnode.instrumentation.prometheus \
  --evnode.instrumentation.prometheus_listen_addr :2112
```

Key metrics to watch:

- `evolve_block_height` - Current block height
- `evolve_da_submission_total` - DA submissions count
- `evolve_da_submission_failures` - Failed DA submissions

Enable the DA visualizer for detailed submission monitoring:

```bash
--evnode.rpc.enable_da_visualization
```

Then access `http://localhost:7331/da` in your browser.

## Health Checks

The aggregator exposes health endpoints:

```bash
# Liveness check
curl http://localhost:7331/health/live

# Readiness check (includes block production rate)
curl http://localhost:7331/health/ready
```

## Troubleshooting

### Blocks Not Being Produced

1. Verify aggregator mode is enabled in logs
2. Check DA layer connectivity
3. Ensure signer key is accessible

### DA Submission Failures

1. Check DA layer endpoint is reachable
2. Verify DA account has sufficient funds
3. Increase gas price if transactions are being outbid

### High Pending Block Count

1. Reduce block time or enable lazy mode
2. Increase DA gas price for faster inclusion
3. Check DA layer congestion

## See Also

- [Full Node Guide](/guides/running-nodes/full-node) - Running a non-producing node
- [DA Visualization](/guides/tools/visualizer) - Monitor DA submissions
- [Monitoring Guide](/guides/operations/monitoring) - Prometheus metrics
