# How to create a genesis for your chain

This guide will walk you through the process of setting up a genesis for your chain. Follow the steps below to initialize your chain, add a genesis account, and start the chain.

## Pre-requisities

For this guide you need to have a chain directory where you have created and built your chain.

If you don't have a chain directory yet, you can initialize a simple ignite chain by following [this tutorial](/guides/gm-world.md)

:::tip
This guide will use the simple ignite chain created in linked guide. Make sure to update any relevant variables to match your chain.
:::

## 1. Setting variables

First, set the necessary variables for your chain in the terminal, here is an example:

```sh
VALIDATOR_NAME=validator1
CHAIN_ID=gm
KEY_NAME=chain-key
CHAINFLAG="--chain-id ${CHAIN_ID}"
TOKEN_AMOUNT="10000000000000000000000000stake"
STAKING_AMOUNT="1000000000stake"
```

## Rebuild your chain

Ensure that `.gm` folder is present at `/Users/you/.gm` (if not, follow a [Guide](/guides/gm-world.md) to set it up) and run the following command to (re)generate an entrypoint binary out of the code:

```sh
make install
```

Once completed, run the following command to ensure that the `/Users/you/.gm` directory is present:

```sh
ignite evolve init
```

This (re)creates an `gmd` binary that will be used for the rest of the tutorials to run all the operations on the chain.

## Resetting existing genesis/chain data

Reset any existing chain data:

```sh
gmd comet unsafe-reset-all
```

Reset any existing genesis data:

```sh
rm -rf $HOME/.$CHAIN_ID/config/gentx
rm $HOME/.$CHAIN_ID/config/genesis.json
```

## Initializing the validator

Initialize the validator with the chain ID you set:

```sh
gmd init $VALIDATOR_NAME --chain-id $CHAIN_ID
```

## Adding a key to keyring backend

Add a key to the keyring-backend:

```sh
gmd keys add $KEY_NAME --keyring-backend test
```

## Adding a genesis account

Add a genesis account with the specified token amount:

```sh
gmd genesis add-genesis-account $KEY_NAME $TOKEN_AMOUNT --keyring-backend test
```

## Setting the staking amount in the genesis transaction

Set the staking amount in the genesis transaction:

```sh
gmd genesis gentx $KEY_NAME $STAKING_AMOUNT --chain-id $CHAIN_ID --keyring-backend test
```

## Collecting genesis transactions

Collect the genesis transactions:

```sh
gmd genesis collect-gentxs
```

## Configuring the genesis file

Copy the centralized sequencer address into `genesis.json`:

```sh
ADDRESS=$(jq -r '.address' ~/.$CHAIN_ID/config/priv_validator_key.json)
PUB_KEY=$(jq -r '.pub_key' ~/.$CHAIN_ID/config/priv_validator_key.json)
jq --argjson pubKey "$PUB_KEY" '.consensus["validators"]=[{"address": "'$ADDRESS'", "pub_key": $pubKey, "power": "1000", "name": "Evolve Sequencer"}]' ~/.$CHAIN_ID/config/genesis.json > temp.json && mv temp.json ~/.$CHAIN_ID/config/genesis.json
```

## Starting the chain

Finally, start the chain with your start command.

For example, start the simple ignite chain with the following command:

```sh
gmd start --evolve.node.aggregator --chain_id $CHAIN_ID
```

## Summary

By following these steps, you will set up the genesis for your chain, initialize the validator, add a genesis account, and started the chain. This guide provides a basic framework for configuring and starting your chain using the gm-world binary. Make sure you initialized your chain correctly, and use the `gmd` command for all operations.
