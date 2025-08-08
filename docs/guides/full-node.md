# Chain Full Node Setup Guide

## Introduction

This guide covers how to set up a full node to run alongside a sequencer node in a Evolve-based blockchain network. A full node maintains a complete copy of the blockchain and helps validate transactions, improving the network's decentralization and security.

## Prerequisites

Before proceeding, ensure that you have completed the [build a chain](/guides/gm-world.md) tutorial, which covers setting-up, building and running your chain.

Ensure that you have:

- A local Data Availability (DA) network node running on port `7980`.
- A Evolve sequencer node running and posting blocks to the DA network.

## Setting Up Your Full Node

### Initialize Chain Config and Copy Genesis File

Let's set a terminal variable for the chain ID.

```bash
CHAIN_ID=gm
```

Initialize the chain config for the full node, lets call it `FullNode` and set the chain ID to your chain ID:

```bash
gmd init FullNode --chain-id $CHAIN_ID --home $HOME/.${CHAIN_ID}_fn
```

Copy the genesis file from the sequencer node:

```bash
cp $HOME/.$CHAIN_ID/config/genesis.json $HOME/.${CHAIN_ID}_fn/config/genesis.json
```

### Set Up P2P Connection to Sequencer Node

Identify the sequencer node's P2P address from its logs. It will look similar to:

```
1:55PM INF listening on address=/ip4/127.0.0.1/tcp/7676/p2p/12D3KooWJbD9TQoMSSSUyfhHMmgVY3LqCjxYFz8wQ92Qa6DAqtmh module=p2p
```

Create an environment variable with the P2P address:

```bash
export P2P_ID="12D3KooWJbD9TQoMSSSUyfhHMmgVY3LqCjxYFz8wQ92Qa6DAqtmh"
```

### Start the Full Node

We are now ready to run our full node. If we are running the full node on the same machine as the sequencer, we need to make sure we update the ports to avoid conflicts.

Make sure to include these flags with your start command:

```sh
  --rpc.laddr tcp://127.0.0.1:46657 \
  --grpc.address 127.0.0.1:9390 \
  --p2p.laddr "0.0.0.0:46656" \
  --api.address tcp://localhost:1318
```

Run your full node with the following command:

```bash
gmd start \
  --evolve.da.address http://127.0.0.1:7980 \
  --p2p.seeds $P2P_ID@127.0.0.1:7676 \
  --minimum-gas-prices 0stake \
  --rpc.laddr tcp://127.0.0.1:46657 \
  --grpc.address 127.0.0.1:9390 \
  --p2p.laddr "0.0.0.0:46656" \
  --api.address tcp://localhost:1318 \
  --chain_id $CHAIN_ID \
  --home $HOME/.${CHAIN_ID}_fn
```

Key points about this command:

- `chain_id` is generally the `$CHAIN_ID`, which is `gm` in this case.
- The ports and addresses are different from the sequencer node to avoid conflicts. Not everything may be necessary for your setup.
- We use the `P2P_ID` environment variable to set the seed node.

## Verifying Full Node Operation

After starting your full node, you should see output similar to:

``` bash
2:33PM DBG indexed transactions height=1 module=txindex num_txs=0
2:33PM INF block marked as DA included blockHash=7897885B959F52BF0D772E35F8DA638CF8BBC361C819C3FD3E61DCEF5034D1CC blockHeight=5532 module=BlockManager
```

This output indicates that your full node is successfully connecting to the network and processing blocks.

:::tip
If your chain uses EVM as an execution layer and you see an error like `datadir already used by another process`, it means you have to remove all the state from chain data directory (`/root/.yourchain_fn/data/`) and specify a different data directory for the EVM client.
:::

## Conclusion

You've now set up a full node running alongside your Evolve sequencer.
