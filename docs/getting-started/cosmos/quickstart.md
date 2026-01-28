# Cosmos SDK Quickstart

Get a Cosmos SDK chain running on Evolve using Ignite CLI.

## Prerequisites

- Go 1.22+
- [Ignite CLI](https://docs.ignite.com/welcome/install)

## 1. Start Local DA

```bash
go install github.com/evstack/ev-node/tools/local-da@latest
local-da
```

Keep this running in a separate terminal.

## 2. Create a New Chain

```bash
ignite scaffold chain mychain --address-prefix mychain
cd mychain
```

## 3. Add Evolve

Install the Evolve plugin for Ignite:

```bash
ignite app install -g github.com/ignite/apps/evolve
```

Add Evolve to your chain:

```bash
ignite evolve add
```

This modifies your chain to use ev-abci instead of CometBFT.

## 4. Build and Initialize

```bash
make install

mychaind init mynode --chain-id mychain-1
mychaind keys add mykey --keyring-backend test
mychaind genesis add-genesis-account mykey 1000000000stake --keyring-backend test
mychaind genesis gentx mykey 1000000stake --chain-id mychain-1 --keyring-backend test
mychaind genesis collect-gentxs
```

## 5. Start the Chain

```bash
mychaind start \
  --evnode.node.aggregator \
  --evnode.da.address http://localhost:7980 \
  --evnode.signer.passphrase secret
```

You should see blocks being produced:
```
INF block marked as DA included blockHeight=1
INF block marked as DA included blockHeight=2
```

## 6. Interact

In another terminal:

```bash
# Check balance
mychaind query bank balances $(mychaind keys show mykey -a --keyring-backend test)

# Send tokens
mychaind tx bank send mykey mychain1... 1000stake --keyring-backend test --chain-id mychain-1 -y
```

## Next Steps

- [Integrate ev-abci](/getting-started/cosmos/integrate-ev-abci) — Manual integration without Ignite
- [Migration Guide](/getting-started/cosmos/migration-guide) — Migrate existing CometBFT chain
- [Connect to Celestia](/guides/da-layers/celestia) — Production DA layer
