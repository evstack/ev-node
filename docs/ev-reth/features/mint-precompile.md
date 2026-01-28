# Mint Precompile

A custom precompile for minting native tokens.

## Overview

The mint precompile allows authorized addresses to mint native tokens (ETH equivalent) directly. This enables:

- Bridge minting (mint when assets are bridged in)
- Inflation schedules
- Programmatic rewards
- Airdrops

## Configuration

In your chainspec (`genesis.json`):

```json
{
  "config": {
    "evolve": {
      "mintPrecompile": {
        "admin": "0xMINT_ADMIN_ADDRESS",
        "address": "0x0000000000000000000000000000000000000100"
      }
    }
  }
}
```

| Field | Description |
|-------|-------------|
| `admin` | Address authorized to call mint |
| `address` | Precompile address (conventionally `0x100`) |

## Interface

```solidity
interface IMintPrecompile {
    // Mint native tokens to recipient
    function mint(address recipient, uint256 amount) external;
}
```

## Usage

From an authorized contract:

```solidity
contract Bridge {
    IMintPrecompile constant MINT = IMintPrecompile(0x0000000000000000000000000000000000000100);

    function bridgeIn(address recipient, uint256 amount) external {
        // Verify bridge proof...

        // Mint native tokens
        MINT.mint(recipient, amount);
    }
}
```

## Security

- Only the `admin` address can call `mint()`
- Calls from other addresses revert
- The admin is typically a bridge contract or multisig

## Changing Admin

The admin cannot be changed after genesis. To update, you would need a chain upgrade with a new chainspec.

## Example: Bridge Setup

```json
{
  "config": {
    "evolve": {
      "mintPrecompile": {
        "admin": "0xBridgeContractAddress",
        "address": "0x0000000000000000000000000000000000000100"
      }
    }
  }
}
```

The bridge contract can mint tokens when users bridge assets from another chain.
