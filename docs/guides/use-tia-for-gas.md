# How to use IBC token (TIA) as gas token in your chain

## 🌞 Introduction {#introduction}

This tutorial will guide you through building a sovereign `gm-world` chain using Evolve, with TIA as the gas token. Unlike the [quick start guide](/guides/quick-start.md), which uses a native chain token for gas, this tutorial demonstrates how to integrate an IBC-enabled token, TIA, as the gas token within the chain, providing a deeper exploration of sovereign chain development.

No prior understanding of the build process is required, just that it utilizes the [Cosmos SDK](https://github.com/cosmos/cosmos-sdk) for blockchain applications.

## Requirements {#requirements}

Before proceeding, ensure that you have completed the [build a chain](/guides/gm-world.md) tutorial, which covers setting-up, building and running your chain.

<!-- markdownlint-disable MD033 -->
<script setup>
import Callout from '../.vitepress/components/callout.vue'
import constants from '../.vitepress/constants/constants.js'
</script>

:::tip
<Callout />
:::

<!-- markdownlint-enable MD033 -->

## Setup your local DA network {#setup-local-da}

Your local DA network is already running if you followed the [quick start guide](/guides/quick-start.md) or the [build a chain](/guides/gm-world.md). If not, you can start it with the following command:

```bash
curl -sSL https://evolve.dev/install-local-da.sh | bash -s {{constants.evolveLatestTag}}
```

## 🚀 Starting your chain {#start-your-chain}

Start the chain, posting to the local DA network:

```bash
gmd start --evolve.node.aggregator --evolve.da.address http://localhost:7980 --minimum-gas-prices="0.02ibc/C3E53D20BC7A4CC993B17C7971F8ECD06A433C10B6A96F4C4C3714F0624C56DA,0.025stake"
```

Note that we specified the gas token to be IBC TIA. We still haven't made an IBC connection to Celestia's Mocha testnet, however, if we assume our first channel will be an ICS-20 transfer channel to Celestia, we can already calculate the token denom using this formula:

```js
"ibc/" + toHex(sha256(toUtf8("transfer/channel-0/utia"))).toUpperCase();
```

Now you should see the logs of the running node:

```bash
12:21PM INF starting node with ABCI CometBFT in-process module=server
12:21PM INF starting node with Evolve in-process module=server
12:21PM INF service start impl=multiAppConn module=proxy msg="Starting multiAppConn service"
12:21PM INF service start connection=query impl=localClient module=abci-client msg="Starting localClient service"
12:21PM INF service start connection=snapshot impl=localClient module=abci-client msg="Starting localClient service"
12:21PM INF service start connection=mempool impl=localClient module=abci-client msg="Starting localClient service"
12:21PM INF service start connection=consensus impl=localClient module=abci-client msg="Starting localClient service"
12:21PM INF service start impl=EventBus module=events msg="Starting EventBus service"
12:21PM INF service start impl=PubSub module=pubsub msg="Starting PubSub service"
12:21PM INF Using default mempool ttl MempoolTTL=25 module=BlockManager
12:21PM INF service start impl=IndexerService module=txindex msg="Starting IndexerService service"
12:21PM INF service start impl=RPC module=server msg="Starting RPC service"
12:21PM INF service start impl=Node module=server msg="Starting Node service"
12:21PM INF starting P2P client module=server
12:21PM INF serving HTTP listen address=127.0.0.1:26657 module=server
12:21PM INF listening on address=/ip4/127.0.0.1/tcp/26656/p2p/12D3KooWSicdPmMTLf9fJbSSHZc9UVP1CbNqKPpbYVbgxHvbhAUY module=p2p
12:21PM INF listening on address=/ip4/163.172.162.109/tcp/26656/p2p/12D3KooWSicdPmMTLf9fJbSSHZc9UVP1CbNqKPpbYVbgxHvbhAUY module=p2p
12:21PM INF no seed nodes - only listening for connections module=p2p
12:21PM INF working in aggregator mode block time=1000 module=server
12:21PM INF Creating and publishing block height=22 module=BlockManager
12:21PM INF starting gRPC server... address=127.0.0.1:9290 module=grpc-server
12:21PM INF finalized block block_app_hash=235D3710D61F347DBBBDD6FD63AA7687842D1EF9CB475C712856D7DA32F82F09 height=22 module=BlockManager num_txs_res=0 num_val_updates=0
12:21PM INF executed block app_hash=235D3710D61F347DBBBDD6FD63AA7687842D1EF9CB475C712856D7DA32F82F09 height=22 module=BlockManager
12:21PM INF indexed block events height=22 module=txindex
...
```

## ✨ Connecting to Celestia Mocha testnet using IBC {#ibc-to-celestia}

Next, we will establish an IBC connection with the Celestia Mocha testnet to enable TIA transfers for gas usage on our chain.

Install the IBC relayer:

```bash
git clone --depth 1 --branch v2.5.2 https://github.com/cosmos/relayer.git /tmp/relayer
cd /tmp/relayer
make install
```

Configure the relayer:

```bash
rly config init

mkdir -p "$HOME/.relayer/keys/{gm,mocha-4}"

echo "global:
    api-listen-addr: :5183
    timeout: 10s
    memo: ''
    light-cache-size: 20
    log-level: info
    ics20-memo-limit: 0
    max-receiver-size: 150
chains:
    gm_chain:
        type: cosmos
        value:
            key-directory: '$HOME/.relayer/keys/gm'
            key: a
            chain-id: gm
            rpc-addr: http://localhost:26657
            account-prefix: gm
            keyring-backend: test
            gas-adjustment: 1.5
            gas-prices: 0.025stake
            min-gas-amount: 0
            max-gas-amount: 0
            debug: false
            timeout: 20s
            block-timeout: ''
            output-format: json
            sign-mode: direct
            extra-codecs: []
            coin-type: 118
            signing-algorithm: ''
            broadcast-mode: batch
            min-loop-duration: 0s
            extension-options: []
            feegrants: null
    mocha:
        type: cosmos
        value:
            key-directory: '$HOME/.relayer/keys/mocha-4'
            key: a
            chain-id: mocha-4
            rpc-addr: https://celestia-testnet-rpc.publicnode.com:443
            account-prefix: celestia
            keyring-backend: test
            gas-adjustment: 1.5
            gas-prices: 0.15utia
            min-gas-amount: 0
            max-gas-amount: 0
            debug: false
            timeout: 20s
            block-timeout: ''
            output-format: json
            sign-mode: direct
            extra-codecs: []
            coin-type: 118
            signing-algorithm: ''
            broadcast-mode: batch
            min-loop-duration: 0s
            extension-options: []
            feegrants: null
paths:
    gm_mocha-4:
        src:
            chain-id: gm
        dst:
            chain-id: mocha-4
        src-channel-filter:
            rule: ''
            channel-list: []
" > "$HOME/.relayer/config/config.yaml"

rly keys restore gm_chain a "regret resist either bid upon yellow leaf early symbol win market vital"
rly keys restore mocha     a "regret resist either bid upon yellow leaf early symbol win market vital"
```

Get the relayer accounts:

```bash
rly address gm_chain a # => gm1jqevcsld0dqpjp3csfg7alkv3lehvn8uswknrc
rly address mocha     a # => celestia1jqevcsld0dqpjp3csfg7alkv3lehvn8u04ymsu
```

Note: These accounts should always be the same because of the hardcoded mnemonics that we've loaded in the `rly keys restore` step.

Fund the relayer on our chain:

```bash
evolve tx bank send gm-key-2 gm1jqevcsld0dqpjp3csfg7alkv3lehvn8uswknrc 10000000stake --keyring-backend test --chain-id gm --fees 5000stake -y
```

Fund the relayer on the Celestia Mocha testnet:

[Mocha Testnet Faucet Instructions](https://docs.celestia.org/how-to-guides/mocha-testnet#mocha-testnet-faucet).

Verify the relayer is funded:

```bash
rly q balance mocha     a # => address {celestia1jqevcsld0dqpjp3csfg7alkv3lehvn8u04ymsu} balance {10000000utia}
rly q balance gm_chain a # => address {gm1jqevcsld0dqpjp3csfg7alkv3lehvn8uswknrc} balance {10000000stake}
```

Create IBC clients:

```bash
rly tx client gm_chain mocha gm_mocha-4 --override
rly tx client mocha gm_chain gm_mocha-4 --override
```

Create IBC connection:

```bash
rly tx connection gm_mocha-4
```

Create IBC channel:

```bash
rly tx channel gm_mocha-4 --src-port transfer --dst-port transfer --version ics20-1
```

Start the relayer:

```bash
rly start gm_mocha-4
```

Transfer TIA from Mocha to our chain:

```bash
ACCOUNT_ON_CHAIN="$(evolve keys show -a --keyring-backend test gm-key-2)"
CHANNEL_ID_ON_MOCHA="$(rly q channels mocha gm_chain | jq -r .channel_id | tail -1)"

rly tx transfer mocha gm_chain 1000000utia "$ACCOUNT_ON_CHAIN" "$CHANNEL_ID_ON_MOCHA" --path gm_mocha-4
```

Verify the account on our chain is funded with IBC TIA:

```bash
gmd query bank balances "$(evolve keys show -a --keyring-backend test gm-key-2)"
# =>
# balances:
# - amount: "1000000"
#   denom: ibc/C3E53D20BC7A4CC993B17C7971F8ECD06A433C10B6A96F4C4C3714F0624C56DA
# - amount: "9999999999999999989995000"
#   denom: stake
# pagination:
#   total: "2"
```

## 💸 Transactions {#transactions}

Finally, send a transaction on our chain using IBC TIA as the gas token:

```bash
ACCOUNT_ON_CHAIN="$(evolve keys show -a --keyring-backend test gm-key-2)"

# Send the transaction
TX_HASH=$(evolve tx bank send "$ACCOUNT_ON_CHAIN" "$ACCOUNT_ON_CHAIN" 1stake --keyring-backend test --chain-id gm --gas-prices 0.02ibc/C3E53D20BC7A4CC993B17C7971F8ECD06A433C10B6A96F4C4C3714F0624C56DA -y --output json | jq -r .txhash)

# Verify success
evolve q tx "$TX_HASH" --output json | jq .code # => 0
```

## 🎉 Next steps

Congratulations! You've built a local chain that posts to a local DA network and uses TIA as the gas token!
