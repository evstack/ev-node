# ev-reth Configuration

Configure ev-reth through chainspec (genesis.json) and command-line flags.

## Chainspec

The chainspec defines chain parameters. ev-reth uses standard Ethereum genesis format with Evolve extensions.

### Basic Structure

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
  "alloc": {},
  "coinbase": "0x0000000000000000000000000000000000000000",
  "difficulty": "0x0",
  "gasLimit": "0x1c9c380",
  "nonce": "0x0",
  "timestamp": "0x0"
}
```

### Evolve Extensions

Add under `config.evolve`:

```json
{
  "config": {
    "chainId": 1337,
    "evolve": {
      "baseFeeSink": "0x...",
      "baseFeeRedirectActivationHeight": 0,
      "deployAllowlist": {
        "admin": "0x...",
        "enabled": ["0x..."]
      },
      "contractSizeLimit": 49152,
      "mintPrecompile": {
        "admin": "0x...",
        "address": "0x0000000000000000000000000000000000000100"
      }
    }
  }
}
```

See [Features](/ev-reth/features/base-fee-redirect) for detailed configuration of each extension.

## Command-Line Flags

### RPC

```bash
--http                      # Enable HTTP JSON-RPC
--http.addr 0.0.0.0         # Listen address
--http.port 8545            # Listen port
--http.api eth,net,web3     # Enabled APIs
```

### Engine API

```bash
--authrpc.addr 0.0.0.0      # Engine API address
--authrpc.port 8551         # Engine API port
--authrpc.jwtsecret jwt.hex # JWT secret file
```

### Data

```bash
--datadir /data             # Data directory
--chain genesis.json        # Chainspec file
```

## Docker

Default `docker-compose.yml`:

```yaml
services:
  reth:
    image: ghcr.io/evstack/ev-reth:latest
    ports:
      - "8545:8545"
      - "8551:8551"
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

## JWT Secret

Generate for Engine API authentication:

```bash
openssl rand -hex 32 > jwt.hex
```

Both ev-reth and ev-node must use the same secret.

## Next Steps

- [Engine API](/ev-reth/engine-api) — Communication protocol
- [Chainspec Reference](/reference/configuration/ev-reth-chainspec) — Full field reference
