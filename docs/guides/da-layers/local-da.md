# Local DA

Local DA is a development-only data availability layer for testing Evolve chains without connecting to a real DA network.

## Overview

Local DA provides:

- Fast, local blob storage
- No authentication required
- No gas fees
- Instant "finality"

**Warning:** Local DA is for development only. It provides no actual data availability guarantees.

## Installation

Install the local-da binary:

```bash
go install github.com/evstack/ev-node/tools/local-da@latest
```

Or build from source:

```bash
cd ev-node/tools/local-da
go build -o local-da .
```

## Running Local DA

Start the local DA server:

```bash
local-da
```

Default output:

```text
INF NewLocalDA: initialized LocalDA module=local-da
INF Listening on host=localhost maxBlobSize=1974272 module=da port=7980
INF server started listening on=localhost:7980 module=da
```

### Configuration

| Flag | Default | Description |
|------|---------|-------------|
| `--host` | `localhost` | Listen address |
| `--port` | `7980` | Listen port |

Example with custom port:

```bash
local-da --port 8080
```

## Connecting Your Chain

Start your Evolve chain with the local DA address:

```bash
evnode start \
    --evnode.node.aggregator \
    --evnode.da.address http://localhost:7980
```

For Cosmos SDK chains:

```bash
appd start \
    --evnode.node.aggregator \
    --evnode.da.address http://localhost:7980
```

## Features

### No Authentication

Unlike Celestia, local DA requires no auth token:

```bash
# Celestia requires
--evnode.da.auth_token <token>

# Local DA does not
--evnode.da.address http://localhost:7980
```

### No Namespace Required

Namespace is optional with local DA:

```bash
# Optional
--evnode.da.namespace my_namespace
```

### Instant Submission

Blobs are stored immediately with no block time delay.

## Use Cases

### Local Development

Test your chain logic without DA layer complexity:

```bash
# Terminal 1: Start local DA
local-da

# Terminal 2: Start your chain
evnode start --evnode.da.address http://localhost:7980
```

### CI/CD Testing

Use local DA in automated tests:

```bash
# Start local DA in background
local-da &
LOCAL_DA_PID=$!

# Run tests
go test ./...

# Cleanup
kill $LOCAL_DA_PID
```

### Integration Testing

Test multi-node setups locally:

```bash
# Start local DA
local-da --port 7980

# Start sequencer
evnode start \
    --evnode.node.aggregator \
    --evnode.da.address http://localhost:7980 \
    --evnode.p2p.listen /ip4/0.0.0.0/tcp/7676

# Start full node (separate terminal)
evnode start \
    --evnode.da.address http://localhost:7980 \
    --evnode.p2p.peers /ip4/127.0.0.1/tcp/7676/p2p/<sequencer-peer-id>
```

## Limitations

Local DA is **not suitable for**:

- Production deployments
- Security testing
- Performance benchmarking (no real network latency)
- Testing DA-specific features (proofs, commitments)

## Transitioning to Celestia

When ready for production, switch to Celestia:

1. Set up a Celestia light node
2. Update your start command:

```bash
# From local DA
--evnode.da.address http://localhost:7980

# To Celestia
--evnode.da.address http://localhost:26658
--evnode.da.auth_token $AUTH_TOKEN
--evnode.da.header_namespace $HEADER_NAMESPACE
--evnode.da.data_namespace $DATA_NAMESPACE
```

See [Celestia Guide](/guides/da-layers/celestia) for full instructions.

## See Also

- [Celestia Guide](/guides/da-layers/celestia) - Production DA setup
- [EVM Quickstart](/getting-started/evm/quickstart) - Getting started with EVM
- [Cosmos Quickstart](/getting-started/cosmos/quickstart) - Getting started with Cosmos SDK
