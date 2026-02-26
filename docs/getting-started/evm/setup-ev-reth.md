# Configure ev-reth

ev-reth is a modified [reth](https://github.com/paradigmxyz/reth) client with Evolve-specific features. This guide covers configuration options.

## Chainspec

The chainspec (`genesis.json`) defines your chain's parameters. ev-reth extends the standard Ethereum genesis format with Evolve-specific fields.

### Minimal Chainspec

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
    "cancunTime": 0
  },
  "alloc": {
    "0xYOUR_ADDRESS": {
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

### Evolve Extensions

Add these under `config.evolve`:

```json
{
  "config": {
    "chainId": 1337,
    "evolve": {
      "baseFeeSink": "0xTREASURY_ADDRESS",
      "baseFeeRedirectActivationHeight": 0,
      "deployAllowlist": {
        "admin": "0xADMIN_ADDRESS",
        "enabled": ["0xDEPLOYER1", "0xDEPLOYER2"]
      },
      "contractSizeLimit": 49152,
      "mintPrecompile": {
        "admin": "0xMINT_ADMIN",
        "address": "0x0000000000000000000000000000000000000100"
      }
    }
  }
}
```

| Field | Description |
|-------|-------------|
| `baseFeeSink` | Address to receive base fees instead of burning |
| `deployAllowlist` | Restrict contract deployment to allowlisted addresses |
| `contractSizeLimit` | Override default 24KB contract size limit |
| `mintPrecompile` | Enable native token minting precompile |

## Docker Configuration

The default `docker-compose.yml` in ev-reth:

```yaml
services:
  reth:
    image: ghcr.io/evstack/ev-reth:latest
    ports:
      - "8545:8545"  # JSON-RPC
      - "8551:8551"  # Engine API
    volumes:
      - ./data:/data
      - ./genesis.json:/genesis.json
      - ./jwt.hex:/jwt.hex
    command:
      - node
      - --chain=/genesis.json
      - --http
      - --http.addr=0.0.0.0
      - --http.api=eth,net,web3,txpool
      - --authrpc.addr=0.0.0.0
      - --authrpc.jwtsecret=/jwt.hex
```

### JWT Secret

Generate a JWT secret for Engine API authentication:

```bash
openssl rand -hex 32 > jwt.hex
```

Both ev-reth and ev-node must use the same JWT secret.

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `RUST_LOG` | Log level | `info` |
| `RETH_DATA_DIR` | Data directory | `/data` |

## Command Line Flags

Common flags when running ev-reth directly:

```bash
ev-reth node \
  --chain genesis.json \
  --http \
  --http.addr 0.0.0.0 \
  --http.port 8545 \
  --http.api eth,net,web3,txpool,debug,trace \
  --authrpc.addr 0.0.0.0 \
  --authrpc.port 8551 \
  --authrpc.jwtsecret jwt.hex
```

## Next Steps

- [ev-reth Features](/ev-reth/features/base-fee-redirect) — Detailed feature documentation
- [ev-reth Chainspec Reference](/reference/configuration/ev-reth-chainspec) — Full configuration reference
