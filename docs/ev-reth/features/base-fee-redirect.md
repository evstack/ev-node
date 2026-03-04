# Base Fee Redirect

Redirect EIP-1559 base fees to a treasury address instead of burning them.

## Overview

In standard Ethereum, base fees are burned. ev-reth allows redirecting these fees to a specified address, enabling:

- Protocol revenue collection
- Treasury funding
- DAO-controlled fee distribution

## Configuration

In your chainspec (`genesis.json`):

```json
{
  "config": {
    "evolve": {
      "baseFeeSink": "0xYOUR_TREASURY_ADDRESS",
      "baseFeeRedirectActivationHeight": 0
    }
  }
}
```

| Field | Description |
|-------|-------------|
| `baseFeeSink` | Address to receive base fees |
| `baseFeeRedirectActivationHeight` | Block height to activate (0 = genesis) |

## How It Works

```text
Transaction Fee = Base Fee + Priority Fee

Standard Ethereum:
├── Base Fee    → Burned
└── Priority Fee → Block producer

With Base Fee Redirect:
├── Base Fee    → baseFeeSink address
└── Priority Fee → Block producer (fee recipient)
```

## Example

Treasury at `0x1234...`:

```json
{
  "config": {
    "chainId": 1337,
    "evolve": {
      "baseFeeSink": "0x1234567890123456789012345678901234567890",
      "baseFeeRedirectActivationHeight": 0
    }
  }
}
```

All base fees from block 0 onward go to the treasury.

## Activation at Later Height

To activate after chain launch:

```json
{
  "config": {
    "evolve": {
      "baseFeeSink": "0x...",
      "baseFeeRedirectActivationHeight": 1000000
    }
  }
}
```

Fees are burned until block 1,000,000, then redirected.

## Use Cases

- **Protocol treasury** — Fund development, grants, or operations
- **Staking rewards** — Distribute to token holders
- **Burn address** — Set to `0x0` to explicitly burn (default behavior)
