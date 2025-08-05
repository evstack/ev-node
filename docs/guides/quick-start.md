---
description: Quickly start a chain node using the Testapp CLI.
---

<script setup>
import constants from '../.vitepress/constants/constants.js'
</script>

# Quick start guide

Welcome to Evolve, a chain framework! The easiest way to launch your network node is by using the Testapp CLI.

## 📦 Install Testapp (CLI)

To install Evolve, run the following command in your terminal:

```bash-vue
curl -sSL https://evolve.dev/install.sh | sh -s {{constants.evolveLatestTag}}
```

Verify the installation by checking the Evolve version:

```bash
testapp version
```

A successful installation will display the version number and its associated git commit hash.

```bash
evolve version:  execution/evm/v1.0.0-beta.1
evolve git sha:  cd1970de
```

## 🗂️ Initialize a evolve network node

To initialize a evolve network node, execute the following command:

```bash
testapp init --evolve.node.aggregator --evolve.signer.passphrase secret
```

## 🚀 Run your evolve network node

Now that we have our testapp generated and installed, we can launch our chain along with the local DA by running the following command:

First lets start the local DA network:

```bash
curl -sSL https://evolve.dev/install-local-da.sh | bash -s {{constants.evolveLatestTag}}
```

You should see logs like:

```bash
4:58PM INF NewLocalDA: initialized LocalDA module=local-da
4:58PM INF Listening on host=localhost maxBlobSize=1974272 module=da port=7980
4:58PM INF server started listening on=localhost:7980 module=da
```

To start a basic evolve network node, execute:

```bash
testapp start --evolve.signer.passphrase secret
```

Upon execution, the CLI will output log entries that provide insights into the node's initialization and operation:

```bash
I[2024-05-01|09:58:46.001] Found private validator                      module=main keyFile=/root/.evolve/config/priv_validator_key.json stateFile=/root/.evolve/data/priv_validator_state.json
I[2024-05-01|09:58:46.002] Found node key                               module=main path=/root/.evolve/config/node_key.json
I[2024-05-01|09:58:46.002] Found genesis file                           module=main path=/root/.evolve/config/genesis.json
...
I[2024-05-01|09:58:46.080] Started node                                 module=main
I[2024-05-01|09:58:46.081] Creating and publishing block                module=BlockManager height=223
I[2024-05-01|09:58:46.082] Finalized block                              module=BlockManager height=223 num_txs_res=0 num_val_updates=0 block_app_hash=
```

## 🎉 Conclusion

That's it! Your evolve network node is now up and running. It's incredibly simple to start a blockchain (which is essentially what a chain is) these days using Evolve. Explore further and discover how you can build useful applications on Evolve. Good luck!
