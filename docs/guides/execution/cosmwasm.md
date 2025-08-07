# 🗞️ CosmWasm chain

<!-- markdownlint-disable MD033 -->
<script setup>
import Callout from '../../.vitepress/components/callout.vue'
import constants from '../../.vitepress/constants/constants.js'
</script>
<!-- markdownlint-enable MD033 -->

:::tip
<Callout />
:::

CosmWasm is a smart contracting platform built for the Cosmos
ecosystem by making use of [WebAssembly](https://webassembly.org) (Wasm)
to build smart contracts for Cosmos-SDK. In this tutorial, we will be
exploring how to integrate CosmWasm with local DA layer using Evolve.

The smart contract we will use for this tutorial is one provided by
the CosmWasm team for Nameservice purchasing.

How to write the Rust smart contract for Nameservice is outside the scope of
this tutorial.

## 💻 CosmWasm dependency {#dependencies}

As with the [GM Chain](/guides/gm-world.md), we use [kurtosis](https://docs.kurtosis.com/) to help with managing all the services we need to run. You can [install kurtosis here](https://docs.kurtosis.com/install).

Once installed, you can verify the installation by running:

```bash
kurtosis version
```

```bash
CLI Version:   0.90.1

To see the engine version (provided it is running): kurtosis engine status
```

## 🚀 Starting your chain {#start-your-chain}

Now that we have kurtosis installed, we can launch our CosmWasm chain along with the local DA by running the following command:

```bash
kurtosis run github.com/evolve/cosmwasm@v0.2.0
```

You should see an output like this:

```bash
INFO[2024-07-02T11:15:43-04:00] Creating a new enclave for Starlark to run inside...
INFO[2024-07-11T11:53:13-04:00] Enclave 'forgotten-fen' created successfully 

Container images used in this run:
> ghcr.io/evolve/local-da:v0.2.1 - remotely downloaded
> ghcr.io/evolve/cosmwasm:v0.1.0 - remotely downloaded

Adding service with name 'local-da' and image 'ghcr.io/evolve/local-da:v0.2.1'
Service 'local-da' added with service UUID '96d04bc472c9455d88d046128fbdefa6'

Printing a message
connecting to da layer via http://172.16.0.5:7980

Printing a message
Adding CosmWasm service

Adding service with name 'wasm' and image 'ghcr.io/evolve/cosmwasm:3b5a25b'
Service 'wasm' added with service UUID 'c71b0308616d40ad919ad24c3d14f35b'

Printing a message
CosmWasm service is available at http://172.16.0.6:36657

Starlark code successfully run. No output was returned.

⭐ us on GitHub - https://github.com/kurtosis-tech/kurtosis
INFO[2024-07-11T11:53:27-04:00] ====================================================== 
INFO[2024-07-11T11:53:27-04:00] ||          Created enclave: forgotten-fen          || 
INFO[2024-07-11T11:53:27-04:00] ====================================================== 
Name:            forgotten-fen
UUID:            8cd936e91ada
Status:          RUNNING
Creation Time:   Thu, 11 Jul 2024 11:53:00 EDT
Flags:           

========================================= Files Artifacts =========================================
UUID   Name

========================================== User Services ==========================================
UUID           Name       Ports                                            Status
96d04bc472c9   local-da   jsonrpc: 7980/tcp -> http://127.0.0.1:7980       RUNNING
c71b0308616d   wasm       grpc-addr: 9290/tcp -> http://127.0.0.1:9290     RUNNING
                          p2p-laddr: 36656/tcp -> http://127.0.0.1:36656   
                          rpc-laddr: 36657/tcp -> http://127.0.0.1:36657   
```

Kurtosis has successfully launched the CosmWasm chain and the local DA network. You can see the services running in docker as well:

```bash
docker ps
```

```bash
CONTAINER ID   IMAGE                              COMMAND                  CREATED              STATUS              PORTS                                                                              NAMES
5bfeda0a871f   ghcr.io/evolve/cosmwasm:v0.1.0    "/bin/sh -c 'wasmd s…"   About a minute ago   Up About a minute   0.0.0.0:9290->9290/tcp, 0.0.0.0:36656-36657->36656-36657/tcp                       wasm--c71b0308616d40ad919ad24c3d14f35b
782dec73fcf8   ghcr.io/evolve/local-da:v0.2.1    "local-da -listen-all"   About a minute ago   Up About a minute   0.0.0.0:7980->7980/tcp                                                             local-da--96d04bc472c9455d88d046128fbdefa6
62da89015918   kurtosistech/core:0.90.1           "/bin/sh -c ./api-co…"   About a minute ago   Up About a minute   0.0.0.0:55500->7443/tcp                                                            kurtosis-api--8cd936e91ada45beab50f0d19be8c57f
1eb6366a5e16   fluent/fluent-bit:1.9.7            "/fluent-bit/bin/flu…"   About a minute ago   Up About a minute   2020/tcp                                                                           kurtosis-logs-collector--8cd936e91ada45beab50f0d19be8c57f
8bfee95b49ee   kurtosistech/engine:0.90.1         "/bin/sh -c ./kurtos…"   39 minutes ago       Up 39 minutes       0.0.0.0:8081->8081/tcp, 0.0.0.0:9710-9711->9710-9711/tcp, 0.0.0.0:9779->9779/tcp   kurtosis-engine--cee974a1c2b141478c9eb2a9b1e4f87f
d532fc82579f   traefik:2.10.6                     "/bin/sh -c 'mkdir -…"   39 minutes ago       Up 39 minutes       80/tcp, 0.0.0.0:9730-9731->9730-9731/tcp                                           kurtosis-reverse-proxy--cee974a1c2b141478c9eb2a9b1e4f87f
7700c0b72195   timberio/vector:0.31.0-debian      "/bin/sh -c 'printf …"   39 minutes ago       Up 39 minutes                                                                                          kurtosis-logs-aggregator
```

We can see the CosmWasm chain running in container `wasm--c71b0308616d40ad919ad24c3d14f35b` and the local DA network running in container `local-da--96d04bc472c9455d88d046128fbdefa6`.

Let's hold on to the container name for the CosmWasm chain, as we will need it later.

```bash
CW=$(docker ps --format '{{.Names}}' | grep wasm)
echo $CW
```

You can verify the chain is running by checking the logs:

```bash
docker logs $CW
```

```bash
...
3:55PM INF Creating and publishing block height=137 module=BlockManager
3:55PM INF finalized block block_app_hash=E71622A57B08D28613A34E3D7AD36BF294CF5A88F4CDD5DD18E6FB65C76F7209 height=137 module=BlockManager num_txs_res=0 num_val_updates=0
3:55PM INF executed block app_hash=E71622A57B08D28613A34E3D7AD36BF294CF5A88F4CDD5DD18E6FB65C76F7209 height=137 module=BlockManager
3:55PM INF indexed block events height=137 module=txindex
3:55PM INF Creating and publishing block height=138 module=BlockManager
3:55PM INF finalized block block_app_hash=E09F4A71E216D85F4CCB9FCBCEE53D82BCA597451C1D4B4FCE0E4081B5FA40E3 height=138 module=BlockManager num_txs_res=0 num_val_updates=0
3:55PM INF executed block app_hash=E09F4A71E216D85F4CCB9FCBCEE53D82BCA597451C1D4B4FCE0E4081B5FA40E3 height=138 module=BlockManager
...
```

Good work so far, we have a Chain node, DA network node, now we can move onto the contract deployment.

## 📒 Contract deployment on CosmWasm with Evolve {#contract-deployment-on-cosmwasm}

### 🤖 Compile the smart contract {#compile-smart-contract}

To compile the smart contract, you can use our docker image.

First download the image:

```bash
docker pull ghcr.io/evolve/contract:v0.2.0
```

Then run the container:

```bash
docker run --rm -d --name cw ghcr.io/evolve/contract:v0.2.0
```

The container is now running and has the pre-built nameservice contract for us. Let's copy it out of the container.

```bash
docker cp cw:/root/cw-contracts/contracts/nameservice .
```

We now have the nameservice contract in the `nameservice` directory.

### 🏎️ Optimized smart contract {#optimized-smart-contract}

Because we are deploying the compiled smart contract to `wasmd`,
we want it to be as small as possible.

<!-- markdownlint-disable MD051 -->
The CosmWasm team provides a tool called `rust-optimizer`, which requires
[Docker](#docker-installation) in order to compile.
<!-- markdownlint-enable MD051 -->

Run the following command in the `~/nameservice` directory you just copied:

```bash
sudo docker run --rm -v "$(pwd)":/code \
  --mount type=volume,source="$(basename "$(pwd)")_cache",target=/code/target \
  --mount type=volume,source=registry_cache,target=/usr/local/cargo/registry \
  cosmwasm/rust-optimizer:0.12.6
```

This will place the optimized Wasm bytecode at `artifacts/cw_nameservice.wasm`.

### 🚀 Contract deployment {#contract-deployment}

Let's now deploy our smart contract!

We will need to do this in the docker container that the CosmWasm chain is running. So first let's move the compiled contract to the container:

```bash
docker cp artifacts/cw_nameservice.wasm $CW:/root/cw_nameservice.wasm
```

Now let's jump into the container:

```bash
docker exec -it $CW sh
```

In order to deploy a contract, you can use the command line as described below.
For a better experience and to use Rust code instead of the command line to
deploy/script and test your contracts, you can use cw-orchestrator.

<!-- markdownlint-disable MD013 -->
```bash
TX_HASH=$(wasmd tx wasm store cw_nameservice.wasm --from localwasm-key --keyring-backend test --chain-id localwasm --gas-prices 0.025uwasm --gas auto --gas-adjustment 1.3 --node http://127.0.0.1:36657 --output json -y | jq -r '.txhash') && echo $TX_HASH
```
<!-- markdownlint-enable MD013 -->

This will get you the transaction hash for the smart contract deployment.

::: danger
If you run into errors with variables on the previous command,
or commands in the remainder of the tutorial, cross-reference
the variables in the command with the variables in the `init.sh` script.
:::

## 🌟 Contract interaction on CosmWasm {#contract-interaction-on-local-da}
<!-- markdownlint-disable MD013 -->

In the previous steps, we have stored out contract's tx hash in an
environment variable for later use.

The following guide will show you how to deploy and interact with a contract using CLI.
For scripting using Rust, you can use cw-orchestrator.

### 🔎 Contract querying {#contract-querying}

Now, let's query our transaction hash for its code ID:

```bash
CODE_ID=$(wasmd query tx --type=hash $TX_HASH --node http://127.0.0.1:36657 --output json | jq -r '.events[-1].attributes[1].value')
echo $CODE_ID
```

This will give us back the Code ID of the deployed contract.

In our case, since it's the first contract deployed on our local network,
the value is `1`.

Now, we can take a look at the contracts instantiated by this Code ID:

```bash
wasmd query wasm list-contract-by-code $CODE_ID --node http://127.0.0.1:36657 --output json
```

We get the following output:

```json
{"contracts":[],"pagination":{"next_key":null,"total":"0"}}
```

### 📃 Contract instantiation {#contract-instantiation}

We start instantiating the contract by writing up the following `INIT` message
for nameservice contract. Here, we are specifying that `purchase_price` of a name
is `100uwasm` and `transfer_price` is `999uwasm`.

```bash
INIT='{"purchase_price":{"amount":"100","denom":"uwasm"},"transfer_price":{"amount":"999","denom":"uwasm"}}'
wasmd tx wasm instantiate $CODE_ID "$INIT" --from localwasm-key --keyring-backend test --label "name service" --chain-id localwasm --gas-prices 0.025uwasm --gas auto --gas-adjustment 1.3 -y --no-admin --node http://127.0.0.1:36657
```

### 📄 Contract interaction {#contract-interaction}

Now that we instantiated it, we can interact further with the contract:

```bash
wasmd query wasm list-contract-by-code $CODE_ID --output json --node http://127.0.0.1:36657
CONTRACT=$(wasmd query wasm list-contract-by-code $CODE_ID --output json --node http://127.0.0.1:36657 | jq -r '.contracts[-1]')
echo $CONTRACT

wasmd query wasm contract --node http://127.0.0.1:36657 $CONTRACT
wasmd query bank balances --node http://127.0.0.1:36657 $CONTRACT
```

This allows us to see the contract address, contract details, and
bank balances.

Your output will look similar to below:

```bash
{"contracts":["wasm14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s0phg4d"],"pagination":{"next_key":null,"total":"0"}}
wasm14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s0phg4d
address: wasm14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s0phg4d
contract_info:
  admin: ""
  code_id: "1"
  created: null
  creator: wasm1y9ceqvnsnm9xtcdmhrjvv4rslgwfzmrzky2c5z
  extension: null
  ibc_port_id: ""
  label: name service
balances: []
pagination:
  next_key: null
  total: "0"
```

Now, let's register a name to the contract for our wallet address:

```bash
REGISTER='{"register":{"name":"fred"}}'
wasmd tx wasm execute $CONTRACT "$REGISTER" --amount 100uwasm --from localwasm-key --chain-id localwasm --gas-prices 0.025uwasm --gas auto --gas-adjustment 1.3 --node http://127.0.0.1:36657 --keyring-backend test -y
```

Your output will look similar to below:

```bash
gas estimate: 167533
code: 0
codespace: ""
data: ""
events: []
gas_used: "0"
gas_wanted: "0"
height: "0"
info: ""
logs: []
raw_log: '[]'
timestamp: ""
tx: null
txhash: C147257485B72E7FFA5FDB943C94CE951A37817554339586FFD645AD2AA397C3
```

If you try to register the same name again, you'll see an expected error:

```bash
Error: rpc error: code = Unknown desc = rpc error: code = Unknown desc = failed to execute message; message index: 0: Name has been taken (name fred): execute wasm contract failed [CosmWasm/wasmd/x/wasm/keeper/keeper.go:364] With gas wanted: '0' and gas used: '123809' : unknown request
```

Next, query the owner of the name record:

```bash
NAME_QUERY='{"resolve_record": {"name": "fred"}}'
wasmd query wasm contract-state smart $CONTRACT "$NAME_QUERY" --node http://127.0.0.1:36657 --output json
```

You'll see the owner's address in a JSON response:

```bash
{"data":{"address":"wasm1y9ceqvnsnm9xtcdmhrjvv4rslgwfzmrzky2c5z"}}
```

With that, we have instantiated and interacted with the CosmWasm nameservice
smart contract on our local DA network using Evolve!
