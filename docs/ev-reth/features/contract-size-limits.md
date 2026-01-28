# Contract Size Limits

Increase the maximum contract bytecode size beyond Ethereum's 24KB limit.

## Overview

Ethereum limits contract size to 24,576 bytes (24KB) via [EIP-170](https://eips.ethereum.org/EIPS/eip-170). ev-reth allows increasing this limit for use cases requiring larger contracts:

- Complex DeFi protocols
- On-chain game logic
- ZK verification contracts

## Configuration

In your chainspec (`genesis.json`):

```json
{
  "config": {
    "evolve": {
      "contractSizeLimit": 49152
    }
  }
}
```

| Field | Description | Default |
|-------|-------------|---------|
| `contractSizeLimit` | Max bytecode size in bytes | 24576 (24KB) |

## Common Values

| Size | Bytes | Use Case |
|------|-------|----------|
| 24KB | 24576 | Ethereum default |
| 48KB | 49152 | 2x limit |
| 64KB | 65536 | 2.67x limit |
| 128KB | 131072 | Large contracts |

## Trade-offs

**Pros:**

- Deploy larger, more complex contracts
- Avoid splitting logic across multiple contracts
- Simpler contract architecture

**Cons:**

- Higher deployment gas costs
- Longer deployment times
- May impact block gas limits

## Example

Allow contracts up to 64KB:

```json
{
  "config": {
    "chainId": 1337,
    "evolve": {
      "contractSizeLimit": 65536
    }
  }
}
```

## Considerations

- This is a chain-wide setting—affects all deployments
- Existing tooling may warn about large contracts
- Consider gas costs for deployment and interaction
