# Evolve EVM Single Sequencer Setup Guide

## Introduction

This guide covers how to set up and run the Single Sequencer implementation of Evolve EVM chains. This implementation provides a centralized approach to transaction sequencing while using EVM as the execution layer.

## Prerequisites

Before starting, ensure you have:

- Go 1.20 or later
- Docker and Docker Compose
- Access to the ev-node and ev-reth repositories
- Git

## Setting Up the Environment

### 1. Clone the Evolve Repository

```bash
git clone --depth 1 --branch {{constants.evolveLatestTag}} https://github.com/evstack/ev-node.git
cd evolve
```

### 2. Build the Evolve EVM Single Sequencer Implementation

```bash
make build-evm-single
make build-da
```

This will create the following binaries in the `build` directory:

- `evm-single` - Single sequencer implementation
- `local-da` - Local data availability node for testing

## Setting Up the Data Availability (DA) Layer

### Start the Local DA Node

```bash
cd build
./local-da start
```

This will start a local DA node on the default port (26658).

## Setting Up the EVM Layer

### 1. Clone the ev-node Repository

```bash
git clone https://github.com/evstack/ev-node.git
cd ev-node
```

### 2. Start the EVM Layer Using Docker Compose

```bash
docker compose up -d
```

This will start Reth (Rust Ethereum client) with the appropriate configuration for Evolve.

### 3. Note the JWT Secret Path

The JWT secret is typically located at `ev-node/execution/evm/docker/jwttoken/jwt.hex`. You'll need this path for the sequencer configuration.

## Running the Single Sequencer Implementation

### 1. Initialize the Sequencer

```bash
cd build
./evm-single init --evolve.node.aggregator=true --evolve.signer.passphrase secret
```

### 2. Start the Sequencer

```bash
./evm-single start \
  --evm.jwt-secret $(cat /path/to/ev-node/execution/evm/docker/jwttoken/jwt.hex) \
  --evm.genesis-hash 0x0a962a0d163416829894c89cb604ae422323bcdf02d7ea08b94d68d3e026a380 \
  --evolve.node.block_time 1s \
  --evolve.node.aggregator=true \
  --evolve.signer.passphrase secret
```

Replace `/path/to/` with the actual path to your ev-node repository.

## Setting Up a Full Node

To run a full node alongside your sequencer, follow these steps:

### 1. Initialize a New Node Directory

```bash
./evm-single init --home ~/.evolve/evm-single-fullnode
```

### 2. Copy the Genesis File

Copy the genesis file from the sequencer node to the full node:

```bash
cp ~/.evolve/evm-single/config/genesis.json ~/.evolve/evm-single-fullnode/config/
```

### 3. Get the Sequencer's P2P Address

Find the sequencer's P2P address in its logs. It will look similar to:

```bash
INF listening on address=/ip4/127.0.0.1/tcp/26659/p2p/12D3KooWXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
```

### 4. Start the Full Node

```bash
./evm-single start \
  --home ~/.evolve/evm-single-fullnode \
  --evm.jwt-secret $(cat /path/to/ev-node/execution/evm/docker/jwttoken/jwt.hex) \
  --evm.genesis-hash 0x0a962a0d163416829894c89cb604ae422323bcdf02d7ea08b94d68d3e026a380 \
  --evolve.node.block_time 1s \
  --evolve.node.aggregator=false \
  --evolve.p2p.peers <SEQUENCER_P2P_ID>@127.0.0.1:26659
```

Replace `<SEQUENCER_P2P_ID>` with the actual P2P ID from your sequencer's logs.

## Verifying Node Operation

After starting your nodes, you should see logs indicating successful block processing:

```bash
INF block marked as DA included blockHash=XXXX blockHeight=XX module=BlockManager
```

## Configuration Reference

### Common Flags

| Flag | Description |
|------|-------------|
| `--evolve.node.aggregator` | Set to true for sequencer mode, false for full node |
| `--evolve.signer.passphrase` | Passphrase for the signer |
| `--evolve.node.block_time` | Block time for the Evolve node |

### EVM Flags

| Flag | Description |
|------|-------------|
| `--evm.eth-url` | Ethereum JSON-RPC URL (default `http://localhost:8545`) |
| `--evm.engine-url` | Engine API URL (default `http://localhost:8551`) |
| `--evm.jwt-secret` | JWT secret file path for the Engine API |
| `--evm.genesis-hash` | Genesis block hash of the chain |
| `--evm.fee-recipient` | Address to receive priority fees |

## Conclusion

You've now set up and configured the Single Sequencer implementation of Evolve EVM chains. This implementation provides a centralized approach to transaction sequencing while using EVM as the execution layer.
