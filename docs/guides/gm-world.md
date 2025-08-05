---
title: GM World tutorial
description: Learn how to build and deploy a CosmWasm-based "gm" (good morning) application using Evolve.
---

# GM world chain

## 🌞 Introduction {#introduction}

This tutorial will guide you through building a evolve `gm-world` chain (`gm` stands for "good morning") using Evolve. Unlike the [quick start guide](/guides/quick-start.md), this tutorial provides a more practical approach to understanding evolve chain development.

We will cover:

- Building and configuring a Cosmos-SDK application-specific chain.
- Posting chain data to a Data Availability (DA) network.
- Executing transactions (the end goal).

No prior understanding of the build process is required, just that it utilizes the [Cosmos SDK](https://github.com/cosmos/cosmos-sdk) for blockchain applications.

<!-- markdownlint-disable MD033 -->
<script setup>
import Callout from '../.vitepress/components/callout.vue'
import constants from '../.vitepress/constants/constants.js'
</script>

:::tip
<Callout />
:::
<!-- markdownlint-enable MD033 -->

## 🛠️ Dependencies {#dependencies}

As we move into more advanced use cases, we use [ignite](https://docs.ignite.com/welcome) to help with managing all the services we need to run. You can [install ignite here](https://docs.ignite.com/welcome/install).

Once installed, you can verify the installation by running:

```bash
ignite version
```

```bash
Ignite CLI version:             v29.2.0-dev
Ignite CLI build date:          undefined
Ignite CLI source hash:         undefined
Ignite CLI config version:      v1
Cosmos SDK version:             v0.53.0
Buf.build version:              undefined
Your OS:                        darwin
Your arch:                      arm64
Your go version:                go version go1.24.3 darwin/arm64
```

## Generate your App {#generate-your-app}

```bash
ignite scaffold chain gm --address-prefix gm
cd gm
```

Install a specific version of ignite to use evolve

```bash
ignite app install -g github.com/ignite/apps/evolve
```

Install your app locally:

```bash
make install
```

## Add Evolve Features {#add-evolve-features}

Enhance your blockchain by adding Evolve features. Use the following command:

```bash
ignite evolve add
```

## Build your chain {#build-your-chain}

Build your chain using the following command:

```bash
ignite chain build
```

This will create a `~/.gm` folder with all the necessary files to run a chain.

## Initialize Your Blockchain {#initialize-your-blockchain}

Before starting your blockchain, you need to initialize it with Evolve support. Initialize the blockchain as follows:

```bash
ignite evolve init
```

It will also initialize 2 accounts `alice` and `bob`:

## 🚀 Starting your chain {#start-your-chain}

Now that we have our gm app generated and installed, we can launch our GM chain along with the local DA by running the following command:

First lets start the local DA network:

```bash
curl -sSL https://evolve.dev/install-local-da.sh | bash -s {{constants.evolveLatestTag}}
```

you should see logs like:

```bash
4:58PM INF NewLocalDA: initialized LocalDA module=local-da
4:58PM INF Listening on host=localhost maxBlobSize=1974272 module=da port=7980
4:58PM INF server started listening on=localhost:7980 module=da
```

After which we can start the app:

```bash
gmd start --evolve.node.aggregator
```

You should see an output like this:

```bash
2:50PM INF creating new client module=evolve namespace=
2:50PM INF No state found in store, initializing new state module=BlockManager
2:50PM INF Initializing chain chainID=gm genesisTime=2025-06-26T12:50:11Z initialHeight=1 module=evolve
2:50PM INF InitChain chainID=gm initialHeight=1 module=baseapp
2:50PM INF initializing blockchain state from genesis.json module=baseapp
2:50PM INF chain initialized successfully appHash=E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855 module=evolve
2:50PM INF using default mempool ttl MempoolTTL=25 module=BlockManager
2:50PM INF service start impl=EventBus module=events msg="Starting EventBus service"
2:50PM INF service start impl=PubSub module=pubsub msg="Starting PubSub service"
2:50PM INF service start impl=IndexerService module=txindex msg="Starting IndexerService service"
2:50PM INF evolve node run loop launched in background goroutine module=server
2:50PM INF serving HTTP listen address=[::]:26657 module=evolve
2:50PM INF starting P2P client module=evolve
2:50PM INF started RPC server addr=127.0.0.1:7331 module=evolve
2:50PM INF listening on address=/ip4/127.0.0.1/tcp/7676/p2p/12D3KooWPN1jqkgZcuF8UMZEa7nSjoF7zPmGHRrCDVrXrpfTLpfJ module=p2p
2:50PM INF listening on address=/ip4/192.168.0.54/tcp/7676/p2p/12D3KooWPN1jqkgZcuF8UMZEa7nSjoF7zPmGHRrCDVrXrpfTLpfJ module=p2p
2:50PM INF no peers - only listening for connections module=p2p
2:50PM INF working in aggregator mode block time=1s module=evolve
2:50PM INF Reaper started interval=1000 module=Reaper
2:50PM INF using pending block height=1 module=BlockManager
2:50PM INF Executing block height=1 module=evolve num_txs=0 timestamp=2025-06-26T14:50:11+02:00
2:50PM INF block executed successfully appHash=678DE6BBA6E23B000DC5AC86B60245E6EAC503C5C7085495F3B71B22A762EB19 height=1 module=evolve
2:50PM INF indexed block events height=1 module=txindex
2:50PM INF attempting to start executor (Adapter.Start) module=server
2:50PM INF executor started successfully module=server
2:50PM INF creating empty block height=2 module=BlockManager
2:50PM INF Executing block height=2 module=evolve num_txs=0 timestamp=2025-06-26T14:50:30+02:00
2:50PM INF starting API server... address=tcp://0.0.0.0:1317 module=api-server
2:50PM INF serve module=api-server msg="Starting RPC HTTP server on [::]:1317"
2:50PM INF starting gRPC server... address=localhost:9090 module=grpc-server
2:50PM INF block executed successfully appHash=0B3973A50C42D0184FB86409FC427BD528A790FA45BA2C9E20FDF14A3628CEC8 height=2 module=evolve
```

Ignite has successfully launched the GM chain and the local DA network. The GM chain is running on port `7331` and the local DA network is running on port `7980`.

Good work so far, we have a Chain node, DA network node, now we can start submitting transactions.

## 💸 Transactions {#transactions}

First, list your keys:

```bash
gmd keys list --keyring-backend test
```

You should see an output like the following

```bash
- address: gm17rpwv7lnk96ka00v93rphhvcqqztpn896q0dxx
  name: alice
  pubkey: '{"@type":"/cosmos.crypto.secp256k1.PubKey","key":"A5WPM5WzfNIPrGyha/TlHt0okdlzS1O4Gb1d1kU+xuG+"}'
  type: local
- address: gm1r2udsh4za7r7sxvzy496qfazvjp04j4zgytve3
  name: bob
  pubkey: '{"@type":"/cosmos.crypto.secp256k1.PubKey","key":"A+jOX/CWInFer2IkqgXGo0da9j7Ubq+e1LJWzTMDjwdt"}'
  type: local
```

For convenience we export two of our keys like this:

```bash
export ALICE=gm17rpwv7lnk96ka00v93rphhvcqqztpn896q0dxx
export BOB=gm1r2udsh4za7r7sxvzy496qfazvjp04j4zgytve3
```

Now let's submit a transaction that sends coins from one account to another (don't worry about all the flags, for now, we just want to submit transaction from a high-level perspective):

```bash
gmd tx bank send $BOB $ALICE 42069stake --keyring-backend test --chain-id gm --fees 5000stake
```

You'll be prompted to accept the transaction:

```bash
auth_info:
  fee:
    amount: []
    gas_limit: "200000"
    granter: ""
    payer: ""
  signer_infos: []
  tip: null
body:
  extension_options: []
  memo: ""
  messages:
  - '@type': /cosmos.bank.v1beta1.MsgSend
    amount:
    - amount: "42069"
      denom: stake
    from_address: gm1r2udsh4za7r7sxvzy496qfazvjp04j4zgytve3
    to_address: gm17rpwv7lnk96ka00v93rphhvcqqztpn896q0dxx
  non_critical_extension_options: []
  timeout_height: "0"
signatures: []
confirm transaction before signing and broadcasting [y/N]: // [!code focus]
```

Confirm and sign the transaction as prompted. now you see the transaction hash at the output:

```bash
//...

txhash: 677CAF6C80B85ACEF6F9EC7906FB3CB021322AAC78B015FA07D5112F2F824BFF
```

## ⚖️ Checking Balances {#balances}

Query balances after the transaction:

```bash
gmd query bank balances $ALICE
```

The receiver’s balance should show an increase.

```bash
balances: // [!code focus]
- amount: "42069" // [!code focus]
  denom: stake
pagination:
  next_key: null
  total: "0"
```

For the sender’s balance:

```bash
gmd query bank balances $BOB
```

Output:

```bash
balances: // [!code focus]
- amount: "99957931" // [!code focus]
  denom: stake
pagination:
  next_key: null
  total: "0"
```

<!-- ## 📦 GM world UI app

Now that you have an idea of how to interact with the chain with the evolve CLI, let's look at the user interface (UI) application aspect of connecting a wallet to a chain.

Connecting your wallet to your chain is as straightforward as connecting to any other blockchain. It assumes you have the [Keplr](https://www.keplr.app/) wallet extension installed in your browser.

## 🔗 Connecting your wallet

Kurtosis spun up a UI app alongside your chain already, so to connect your Keplr wallet to the application, simply open your browser and go to [http://localhost:3000](https://localhost:3000).

Click the "Connect Wallet" button on the page, and approve the connection request in the Keplr prompt.

Once authorized, your wallet address will be displayed, confirming that your wallet is successfully connected.

![gm-world-frontend-connected](/img/gm-world-frontend-wallet-connected.png)

:::tip
If you run into any issues, make sure your Keplr wallet is updated and set to connect to your local environment.
::: -->

## 🎉 Next steps

Congratulations! You've experienced connecting to a chain from the user side — simple and straightforward. Now, you might consider exploring how to add more application logic to your chain using the Cosmos SDK, as demonstrated in our Wordle App tutorial.
