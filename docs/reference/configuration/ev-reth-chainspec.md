# ev-reth Chainspec Reference

Complete reference for ev-reth chainspec (genesis.json) configuration.

## Structure

```json
{
  "config": { },
  "alloc": { },
  "coinbase": "0x...",
  "difficulty": "0x0",
  "gasLimit": "0x...",
  "nonce": "0x0",
  "timestamp": "0x0"
}
```

## config

Chain configuration parameters.

### Standard Ethereum Fields

| Field                 | Type   | Description                       |
|-----------------------|--------|-----------------------------------|
| `chainId`             | number | Unique chain identifier           |
| `homesteadBlock`      | number | Homestead fork block (use 0)      |
| `eip150Block`         | number | EIP-150 fork block (use 0)        |
| `eip155Block`         | number | EIP-155 fork block (use 0)        |
| `eip158Block`         | number | EIP-158 fork block (use 0)        |
| `byzantiumBlock`      | number | Byzantium fork block (use 0)      |
| `constantinopleBlock` | number | Constantinople fork block (use 0) |
| `petersburgBlock`     | number | Petersburg fork block (use 0)     |
| `istanbulBlock`       | number | Istanbul fork block (use 0)       |
| `berlinBlock`         | number | Berlin fork block (use 0)         |
| `londonBlock`         | number | London fork block (use 0)         |
| `shanghaiTime`        | number | Shanghai fork timestamp (use 0)   |
| `cancunTime`          | number | Cancun fork timestamp (use 0)     |

### config.evolve

Evolve-specific extensions.

| Field                             | Type    | Description                        |
|-----------------------------------|---------|------------------------------------|
| `baseFeeSink`                     | address | Redirect base fees to this address |
| `baseFeeRedirectActivationHeight` | number  | Block height to activate redirect  |
| `deployAllowlist`                 | object  | Contract deployment restrictions   |
| `contractSizeLimit`               | number  | Max contract bytecode size (bytes) |
| `mintPrecompile`                  | object  | Native token minting precompile    |

#### deployAllowlist

```json
{
  "admin": "0x...",
  "enabled": ["0x...", "0x..."]
}
```

| Field     | Type      | Description                 |
|-----------|-----------|-----------------------------|
| `admin`   | address   | Can modify the allowlist    |
| `enabled` | address[] | Addresses allowed to deploy |

#### mintPrecompile

```json
{
  "admin": "0x...",
  "address": "0x0000000000000000000000000000000000000100"
}
```

| Field     | Type    | Description        |
|-----------|---------|--------------------|
| `admin`   | address | Can call mint()    |
| `address` | address | Precompile address |

## alloc

Pre-funded accounts and contract deployments.

```json
{
  "alloc": {
    "0xAddress1": {
      "balance": "0x..."
    },
    "0xAddress2": {
      "balance": "0x...",
      "code": "0x...",
      "storage": {
        "0x0": "0x..."
      }
    }
  }
}
```

| Field     | Type       | Description                  |
|-----------|------------|------------------------------|
| `balance` | hex string | Wei balance                  |
| `code`    | hex string | Contract bytecode (optional) |
| `storage` | object     | Storage slots (optional)     |
| `nonce`   | hex string | Account nonce (optional)     |

## Top-Level Fields

| Field        | Type       | Description                    |
|--------------|------------|--------------------------------|
| `coinbase`   | address    | Default fee recipient          |
| `difficulty` | hex string | Initial difficulty (use "0x0") |
| `gasLimit`   | hex string | Block gas limit                |
| `nonce`      | hex string | Genesis nonce (use "0x0")      |
| `timestamp`  | hex string | Genesis timestamp              |
| `extraData`  | hex string | Extra data (optional)          |
| `mixHash`    | hex string | Mix hash (optional)            |

## Example

```json
{
  "config": {
    "chainId": 1337,
    "homesteadBlock": 0,
    "eip150Block": 0,
    "eip155Block": 0,
    "eip158Block": 0,
    "byzantiumBlock": 0,
    "constantinopleBlock": 0,
    "petersburgBlock": 0,
    "istanbulBlock": 0,
    "berlinBlock": 0,
    "londonBlock": 0,
    "shanghaiTime": 0,
    "cancunTime": 0,
    "evolve": {
      "baseFeeSink": "0x1234567890123456789012345678901234567890",
      "baseFeeRedirectActivationHeight": 0,
      "contractSizeLimit": 49152,
      "mintPrecompile": {
        "admin": "0xBridgeContract",
        "address": "0x0000000000000000000000000000000000000100"
      }
    }
  },
  "alloc": {
    "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266": {
      "balance": "0x200000000000000000000000000000000000000000000000000000000000000"
    }
  },
  "coinbase": "0x0000000000000000000000000000000000000000",
  "difficulty": "0x0",
  "gasLimit": "0x1c9c380",
  "nonce": "0x0",
  "timestamp": "0x0"
}
```
