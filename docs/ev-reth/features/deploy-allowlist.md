# Deploy Allowlist

Restrict contract deployment to a set of approved addresses.

## Overview

By default, any address can deploy contracts. The deploy allowlist restricts deployment to explicitly approved addresses, useful for:

- Permissioned chains
- Controlled rollouts
- Compliance requirements

## Configuration

In your chainspec (`genesis.json`):

```json
{
  "config": {
    "evolve": {
      "deployAllowlist": {
        "admin": "0xADMIN_ADDRESS",
        "enabled": [
          "0xDEPLOYER_1",
          "0xDEPLOYER_2"
        ]
      }
    }
  }
}
```

| Field | Description |
|-------|-------------|
| `admin` | Address that can modify the allowlist |
| `enabled` | Addresses allowed to deploy contracts |

## How It Works

1. User attempts `CREATE` or `CREATE2` opcode
2. ev-reth checks if sender is in `enabled` list
3. If not allowed, transaction reverts

## Admin Operations

The admin can modify the allowlist via precompile calls:

```solidity
interface IDeployAllowlist {
    function addDeployer(address deployer) external;
    function removeDeployer(address deployer) external;
    function isAllowed(address deployer) external view returns (bool);
}
```

Precompile address: `0x0000000000000000000000000000000000000101`

## Disabling

To allow unrestricted deployment, omit the `deployAllowlist` config entirely or set an empty `enabled` list with no admin.

## Example: Single Deployer

```json
{
  "config": {
    "evolve": {
      "deployAllowlist": {
        "admin": "0xAdminAddress",
        "enabled": ["0xAdminAddress"]
      }
    }
  }
}
```

Only the admin can deploy contracts initially. They can add more deployers later.
