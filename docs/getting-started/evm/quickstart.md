# EVM Quickstart

Get an EVM rollup running locally in under 5 minutes.

## Prerequisites

- Go 1.22+
- Docker
- Git

## 1. Start Local DA

```bash
go install github.com/evstack/ev-node/tools/local-da@latest
local-da
```

You should see:
```
INF Listening on host=localhost port=7980
```

Keep this running in a separate terminal.

## 2. Start ev-reth

```bash
git clone https://github.com/evstack/ev-reth.git
cd ev-reth
docker compose up -d
```

This starts reth with Evolve's Engine API configuration. The default ports:
- `8545` — JSON-RPC
- `8551` — Engine API

## 3. Start ev-node

In a new terminal:

```bash
git clone https://github.com/evstack/ev-node.git
cd ev-node
make build-evm
```

Initialize and start:

```bash
./build/evm init --evnode.node.aggregator --evnode.signer.passphrase secret

./build/evm start \
  --evnode.node.aggregator \
  --evnode.signer.passphrase secret \
  --evnode.node.block_time 1s
```

You should see blocks being produced:
```
INF block marked as DA included blockHeight=1
INF block marked as DA included blockHeight=2
```

## 4. Connect a Wallet

Add the network to MetaMask:

| Setting | Value |
|---------|-------|
| Network Name | Evolve Local |
| RPC URL | http://localhost:8545 |
| Chain ID | 1337 |
| Currency | ETH |

## 5. Deploy a Contract

With Foundry:

```bash
forge create src/Counter.sol:Counter --rpc-url http://localhost:8545 --private-key <YOUR_KEY>
```

## Next Steps

- [Configure ev-reth](/getting-started/evm/setup-ev-reth) — Customize chainspec, features
- [Deploy Contracts](/getting-started/evm/deploy-contracts) — Foundry and Hardhat setup
- [Connect to Celestia](/guides/da-layers/celestia) — Production DA layer
- [Run a Full Node](/guides/running-nodes/full-node) — Non-sequencer node setup
