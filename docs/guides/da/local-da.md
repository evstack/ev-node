# Using Local DA

<!-- markdownlint-disable MD033 -->
<script setup>
import constants from '../../.vitepress/constants/constants.js'
</script>

## Introduction {#introduction}

This tutorial serves as a comprehensive guide for using the [local-da](https://github.com/evstack/ev-node/tree/main/da/cmd/local-da) with your chain.

Before proceeding, ensure that you have completed the [build a chain](./gm-world.md) tutorial, which covers setting-up, building and running your chain.

## Setting Up a Local DA Network

To set up a local DA network node on your machine, run the following script to install and start the local DA node:

```bash
git clone  --depth=1 --branch v1.0.0-beta.2 https://github.com/evstack/ev-node.git
cd ev-node
make build-da
./build/local-da
```

This script will build and run the node, which will then listen on port `7980`.

## Configuring your chain to connect to the local DA network

To connect your chain to the local DA network, you need to pass the `--evnode.da.address` flag with the local DA node address.

## Run your chain

Start your chain node with the following command, ensuring to include the DA address flag:

::: code-group

```sh [Quick Start]
testapp start --evnode.da.address http://localhost:7980
```

```sh [gm-world Chain]
gmd start \
    --evnode.node.aggregator \
    --evnode.da.address http://localhost:7980 \
```

:::

You should see the following log message indicating that your chain is connected to the local DA network:

```shell
11:07AM INF NewLocalDA: initialized LocalDA module=local-da
11:07AM INF Listening on host=localhost maxBlobSize=1974272 module=da port=7980
11:07AM INF server started listening on=localhost:7980 module=da
```

## Summary

By following these steps, you will set up a local DA network node and configure your chain to post data to it. This setup is useful for testing and development in a controlled environment. You can find more information on running the local-da binary [here](https://github.com/evstack/ev-node/blob/main/da/cmd/local-da/README.md)
