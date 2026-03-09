# Celestia

This guide covers connecting your Evolve chain to Celestia for production data availability.

## Prerequisites

- Completed an Evolve quickstart tutorial
- Familiarity with running a Celestia light node

## Running a Celestia Light Node

Before starting your Evolve chain, you need a Celestia light node running and synced.

Follow the [Celestia light node documentation](https://docs.celestia.org/how-to-guides/light-node) to install, initialize, and start a light node for your target network (Arabica, Mocha, or Mainnet).

## Configuring Evolve for Celestia

### Required Configuration

The following flags are required to connect to Celestia:

| Flag | Description |
|------|-------------|
| `--evnode.da.address` | Celestia node RPC endpoint |
| `--evnode.da.auth_token` | JWT authentication token |
| `--evnode.da.header_namespace` | Namespace for block headers |
| `--evnode.da.data_namespace` | Namespace for transaction data |

### Get DA Block Height

Query the current DA height to set as your starting point:

```bash
DA_BLOCK_HEIGHT=$(celestia header network-head | jq -r '.result.header.height')
echo "Your DA_BLOCK_HEIGHT is $DA_BLOCK_HEIGHT"
```

### Get Authentication Token

Generate a write token for your light node:

**Arabica:**

```bash
AUTH_TOKEN=$(celestia light auth write --p2p.network arabica)
```

**Mocha:**

```bash
AUTH_TOKEN=$(celestia light auth write --p2p.network mocha)
```

**Mainnet:**

```bash
AUTH_TOKEN=$(celestia light auth write)
```

### Set Namespaces

Choose unique namespaces for your chain's headers and data:

```bash
DA_HEADER_NAMESPACE="my_chain_headers"
DA_DATA_NAMESPACE="my_chain_data"
```

The namespace values are automatically encoded by ev-node for Celestia compatibility.

You can use the same namespace for both headers and data, or separate them for optimized syncing (light clients can sync headers only).

### Set DA Address

Default Celestia light node port is 26658. **Note:** Connection to a celestia-node DA requires a websocket connection:

```bash
DA_ADDRESS=ws://localhost:26658
```

## Running Your Chain

Start your chain with Celestia configuration:

```bash
evnode start \
    --evnode.node.aggregator \
    --evnode.da.auth_token $AUTH_TOKEN \
    --evnode.da.header_namespace $DA_HEADER_NAMESPACE \
    --evnode.da.data_namespace $DA_DATA_NAMESPACE \
    --evnode.da.address $DA_ADDRESS
```

For Cosmos SDK chains:

```bash
appd start \
    --evnode.node.aggregator \
    --evnode.da.auth_token $AUTH_TOKEN \
    --evnode.da.header_namespace $DA_HEADER_NAMESPACE \
    --evnode.da.data_namespace $DA_DATA_NAMESPACE \
    --evnode.da.address $DA_ADDRESS
```

## Viewing Your Chain Data

Once running, you can view your chain's data on Celestia block explorers:

- [Celenium (Arabica)](https://arabica.celenium.io/)
- [Celenium (Mocha)](https://mocha.celenium.io/)
- [Celenium (Mainnet)](https://celenium.io/)

Search by your namespace or account address to see submitted blobs.

## Configuration Options

### Gas Price

By default, ev-node uses automatic gas price detection. Keep the default unless you have an operational reason to override it:

```bash
--evnode.da.gas_price 0.01
```

Higher gas prices result in faster inclusion during congestion. Omit this flag to use the automatic default.

### Block Time

Set the expected DA block time (affects retry timing):

```bash
--evnode.da.block_time 6s
```

Celestia's block time is approximately 6 seconds.

### Multiple Signing Addresses

For high-throughput chains, use multiple signing addresses to avoid nonce conflicts:

```bash
--evnode.da.signing_addresses celestia1abc...,celestia1def...,celestia1ghi...
```

All addresses must be funded and loaded in the Celestia node's keyring.

## Funding Your Account

### Testnet (Mocha/Arabica)

Get testnet TIA from faucets:

- [Mocha Faucet](https://faucet.celestia-mocha.com/)
- [Arabica Faucet](https://faucet.celestia-arabica.com/)

### Mainnet

Purchase TIA and transfer to your Celestia light node address.

Check your address:

```bash
celestia state account-address
```

## Troubleshooting

### Out of Funds

If you see `Code: 19` errors, your account is out of TIA:

1. Fund your account
2. Increase gas price to unstick pending transactions
3. Restart your chain

See [Troubleshooting Guide](/guides/operations/troubleshooting) for details.

### Connection Refused

Verify your Celestia node is running:

```bash
curl http://localhost:26658/header/sync_state
```

### Token Expired

Regenerate your auth token:

```bash
celestia light auth write --p2p.network <network>
```

## See Also

- [Local DA Guide](/guides/da-layers/local-da) - Development setup
- [Troubleshooting](/guides/operations/troubleshooting) - Common issues
- [Configuration Reference](/reference/configuration/ev-node-config) - All DA options
