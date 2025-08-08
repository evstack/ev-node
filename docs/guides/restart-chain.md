# 🔄 How to restart your chain

This guide will teach you how to restart your Evolve chain.

## Restart chain

This section covers the case where you need to restart your chain.

In order to restart your chain, you simply need to run the `<your-binary>d start [...args]`
command for your chain.

For example, if you ran the [quick start](/guides/quick-start.md) tutorial, you started your chain with:

```bash
testapp start
```

You would have see output similar to:

```bash
I[2024-10-17|14:52:12.845] Creating and publishing block                module=BlockManager height=7
I[2024-10-17|14:52:12.845] finalized block                              module=BlockManager height=7 num_txs_res=0 num_val_updates=0 block_app_hash=
I[2024-10-17|14:52:12.845] executed block                               module=BlockManager height=7 app_hash=
I[2024-10-17|14:52:12.846] indexed block events                         module=txindex height=7
```

If you need to restart your chain, you can run the same command again:

```bash
testapp start
```

You will see that the block height will continue from where it left off:

```bash
I[2024-10-17|14:52:13.845] Creating and publishing block                module=BlockManager height=8
I[2024-10-17|14:52:13.845] finalized block                              module=BlockManager height=8 num_txs_res=0 num_val_updates=0 block_app_hash=
I[2024-10-17|14:52:13.845] executed block                               module=BlockManager height=8 app_hash=
I[2024-10-17|14:52:13.845] indexed block events                         module=txindex height=8
```

It is important to include any additional flags that you used when you first started your chain. For example, if you used the `--evolve.da.namespace` flag, you will need to include that flag when restarting your chain to ensure your chain continues to publish blobs to the same namespace.

## Restart chain after running out of funds

This section covers the case that the node that
you are using to post blocks to your DA and consensus layer runs out of funds (tokens),
and you need to restart your chain.

In this example, we're using Celestia's [Mocha testnet](https://docs.celestia.org/how-to-guides/mocha-testnet/)
and running the [quick start](/guides/quick-start.md). In this example, our Celestia DA light node
ran out of Mocha testnet TIA and we are unable to post new blocks to Celestia due to a
[`Code: 19`](https://github.com/cosmos/cosmos-sdk/blob/main/types/errors/errors.go#L95)
error. This error is defined by Cosmos SDK as:

```go
// ErrTxInMempoolCache defines an ABCI typed error where a tx already exists in the mempool.
ErrTxInMempoolCache = Register(RootCodespace, 19, "tx already in mempool")
```

In order to get around this error, and the same error on other Evolve chains, you will need to re-fund your Celestia account and increase the gas fee. This will override the transaction that is stuck in the mempool.

If you top up the balance of your node and don't increase the gas fee, you will still encounter the `Code: 19` error because there is a transaction (posting block to DA) that is duplicate to one that already exists. In order to get around this, you'll need to increase the gas fee and restart the chain.

### 🟠 Errors in this example {#errors}

This is what the errors will look like if your DA node runs out of funding or you restart the chain without changing the gas fee:

```bash
4:51PM INF submitting block to DA layer height=28126 module=BlockManager
4:51PM ERR DA layer submission failed error="Codespace: 'sdk', Code: 19, Message: " attempt=1 module=BlockManager
4:51PM ERR DA layer submission failed Error="Codespace: 'sdk', Code: 19, Message: " attempt=2 module=BlockManager
4:51PM ERR DA layer submission failed error="Codespace: 'sdk', Code: 19, Message: " attempt=3 module=BlockManager
```

### 💰 Re-fund your account {#refund-your-account}

First, you'll need to send more tokens to the account running your Celestia node. If you didn't keep track of your key, you can run the following to get your address:

```bash
cd $HOME && cd celestia-node
./cel-key list --keyring-backend test --node.type light --p2p.network <network>
```

### 🛑 Stopping your chain {#stopping-your-chain}

You can stop your chain by using `Control + C` in your terminal where the node is running.

### ⛽ Increase the gas fee {#increase-gas-fee}

To reiterate, before restarting the chain, you will need to increase the gas fee in order to avoid a `Code: 19` error. See the [How to configure gas price](/guides/config.md#da-gas-price) guide for more information.

### 🔁 Restarting your chain {#restarting-your-chain}

Follow the [restart chain](#restart-chain) section above.

### 🛢️ Reduce gas fee & restart again {#reduce-gas-fee-restart-again}

In order to save your TIA, we also recommend stopping the chain with `Control + C`, changing the gas fee back to the default (in our case, 8000 utia) and restarting the chain:

🎊 Congrats! You've successfully restarted your Evolve chain after running out of TIA.
