# How to Turn Your CometBFT App into an Evolve App

This guide will walk you through the process of turning your existing CometBFT app into an Evolve app. By integrating Evolve into your CometBFT-based blockchain, you can leverage enhanced modularity and data availability features.

<!-- markdownlint-disable MD033 -->
<script setup>
import Callout from '../.vitepress/components/callout.vue'
import constants from '../.vitepress/constants/constants.js'
</script>

This guide assumes you have a CometBFT app set up and [Ignite CLI](https://docs.ignite.com) installed.

:::warning
This tutorial is currently being updated to reflect the latest changes using the evolve ignite app.
Please check back later for the updated version.
:::

## Install Evolve {#install-evolve}

You need to install Evolve in your CometBFT app. Open a terminal in the directory where your app is located and run the following command:

```bash-vue
ignite app install github.com/ignite/apps/evolve@{{constants.evolveIgniteAppVersion}}
```

## Add Evolve Features to Your CometBFT App {#add-evolve-features}

Now that Evolve is installed, you can add Evolve features to your existing blockchain app. Run the following command to integrate Evolve:

```bash
ignite evolve add
```

## Initialize Evolve {#initialize-evolve}

To prepare your app for Evolve, you'll need to initialize it.

Run the following command to initialize Evolve:

```bash
ignite evolve init
```


<!-- TODO: update

## Initialize Evolve CLI Configuration {#initialize-evolve-cli-configuration}

Next, you'll need to initialize the Evolve CLI configuration by generating the `evolve.toml` file. This file is crucial for Evolve to understand the structure of your chain.

To create the `evolve.toml` configuration, use this command:

```bash
evolve toml init
```

This command sets up the `evolve.toml` file, where you can further customize configuration parameters as needed.

## Start Your Evolve App {#start-evolve-app}

Once everything is configured, you can start your Evolve-enabled CometBFT app or (simply evolve app). Use the following command to start your blockchain:

```bash
evolve start --evolve.aggregator <insert your flags>
```

## Summary

By following this guide, you've successfully converted your CometBFT app into an Evolve app.

To learn more about how to config your DA, Sequencing, and Execution, please check out those tutorial sections.

-->
