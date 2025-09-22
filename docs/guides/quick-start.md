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

```bash
# Clone the repository
git clone --depth 1 https://github.com/evstack/ev-node.git
cd ev-node

# Build the testapp binary
make build

# Optional: Install to your Go bin directory for system-wide access
make install
```

Verify the installation by checking the Evolve version:

```bash
# If you ran 'make install'
testapp version

# Or if you only ran 'make build'
./build/testapp version
```

A successful installation will display the version number and its associated git commit hash.

```bash
evolve version:  v1.0.0-beta.4
```

## 🗂️ Initialize a evolve network node

To initialize a evolve network node, execute the following command:

```bash
testapp init --evnode.node.aggregator --evnode.signer.passphrase secret
```

## 🚀 Run your evolve network node

Now that we have our testapp generated and installed, we can launch our chain along with the local DA by running the following command:

First lets start the local DA network:

```bash
go install github.com/evstack/ev-node/da/cmd/local-da@latest
local-da
```

You should see logs like:

```bash
9:22AM INF NewLocalDA: initialized LocalDA component=da
9:22AM INF Listening on component=da host=localhost maxBlobSize=1974272 port=7980
9:22AM INF server started component=da listening_on=localhost:7980
```

To start a basic evolve network node, execute:

```bash
testapp start --evnode.signer.passphrase secret
```

Upon execution, the CLI will output log entries that provide insights into the node's initialization and operation:

```bash
9:23AM INF creating new client component=main namespace=
KV Executor HTTP server starting on 127.0.0.1:9090
9:23AM INF KV executor HTTP server started component=main endpoint=127.0.0.1:9090
9:23AM INF No state found in store, initializing new state component=BlockManager
9:23AM INF using default mempool ttl MempoolTTL=25 component=BlockManager
9:23AM INF starting P2P client component=main
9:23AM INF started RPC server addr=127.0.0.1:7331 component=main
9:23AM INF listening on address address=/ip4/127.0.0.1/tcp/7676/p2p/12D3KooWRzvJuFoQKhQNfaCZWvJFDY4vrCTocdL6H1GCMzywugnV component=main
9:23AM INF listening on address address=/ip4/172.20.10.14/tcp/7676/p2p/12D3KooWRzvJuFoQKhQNfaCZWvJFDY4vrCTocdL6H1GCMzywugnV component=main
9:23AM INF no peers - only listening for connections component=main
9:23AM INF working in aggregator mode block_time=1000 component=main
9:23AM INF using pending block component=BlockManager height=1
9:23AM INF Reaper started component=Reaper interval=1000
```

## 🎉 Conclusion

That's it! Your evolve network node is now up and running. It's incredibly simple to start a blockchain (which is essentially what a chain is) these days using Evolve. Explore further and discover how you can build useful applications on Evolve. Good luck!
