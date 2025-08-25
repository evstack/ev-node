# DA Height Polling for Genesis Initialization

This feature allows non-aggregator nodes to automatically poll the DA (Data Availability) height from an aggregator node during genesis initialization, eliminating the need to manually configure the DA start height.

## How It Works

When a non-aggregator node starts with a genesis configuration where `genesis_da_start_height` is zero (or equivalently, `GenesisDAStartTime` is zero time), it will automatically poll the configured aggregator endpoint to get the current DA height before proceeding with chain initialization.

## Configuration

### For Non-Aggregator Nodes

Add the aggregator endpoint to your configuration:

```yaml
da:
  aggregator_endpoint: http://aggregator-node:7331
```

Or via command line:

```bash
evnode start --evnode.da.aggregator_endpoint=http://aggregator-node:7331
```

### For Aggregator Nodes

No additional configuration needed. Aggregator nodes automatically expose the `/da/height` endpoint when DA visualization is enabled.

## Genesis File Setup

For non-aggregator nodes to trigger automatic DA height polling, the genesis file should have a zero DA start time:

```json
{
  "chain_id": "my-chain",
  "genesis_da_start_height": "1970-01-01T00:00:00Z",
  "initial_height": 1,
  "proposer_address": "..."
}
```

## Workflow Example

1. **Aggregator Node Setup:**
   ```bash
   # Start an aggregator node
   evnode start --evnode.node.aggregator
   ```

2. **Non-Aggregator Node Setup:**
   ```bash
   # Initialize non-aggregator node with aggregator endpoint
   evnode init --evnode.da.aggregator_endpoint=http://aggregator:7331
   
   # Start non-aggregator node (will automatically poll DA height)
   evnode start
   ```

## API Endpoint

Aggregator nodes expose the following endpoint:

**GET `/da/height`**

Returns the current DA layer height:

```json
{
  "height": 1234,
  "timestamp": "2023-11-15T10:30:15Z"
}
```

## Polling Behavior

- **Timeout:** 5 minutes maximum
- **Interval:** 5 seconds between polls
- **Condition:** Only triggers when `genesis_da_start_height` is zero and node is not an aggregator
- **Fallback:** If no aggregator endpoint is configured, uses current time

## Error Handling

If the polling fails:
- The node will log the error and fail to start
- Check that the aggregator endpoint is reachable
- Verify the aggregator node has the DA height endpoint available
- Ensure the aggregator node is running and has advanced beyond height 0

## Use Cases

- **Simplified Deployment:** No need to manually coordinate DA start heights
- **Dynamic Chain Launching:** Non-aggregator nodes can join automatically
- **Development Testing:** Easy setup for multi-node test networks