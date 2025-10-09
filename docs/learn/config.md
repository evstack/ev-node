# Config

This document provides a comprehensive reference for all configuration options available in Evolve. Understanding these configurations will help you tailor Evolve's behavior to your specific needs, whether you're running an aggregator, a full node, or a light client.

## Table of Contents

- [DA-Only Sync Mode](#da-only-sync-mode)
- [Introduction to Configurations](#configs)
- [Base Configuration](#base-configuration)
  - [Root Directory](#root-directory)
  - [Database Path](#database-path)
  - [Chain ID](#chain-id)
- [Node Configuration (`node`)](#node-configuration-node)
  - [Aggregator Mode](#aggregator-mode)
  - [Light Client Mode](#light-client-mode)
  - [Block Time](#block-time)
  - [Maximum Pending Blocks](#maximum-pending-blocks)
  - [Lazy Mode (Lazy Aggregator)](#lazy-mode-lazy-aggregator)
  - [Lazy Block Interval](#lazy-block-interval)
  - [Trusted Hash](#trusted-hash)
- [Data Availability Configuration (`da`)](#data-availability-configuration-da)
  - [DA Service Address](#da-service-address)
  - [DA Authentication Token](#da-authentication-token)
  - [DA Gas Price](#da-gas-price)
  - [DA Gas Multiplier](#da-gas-multiplier)
  - [DA Submit Options](#da-submit-options)
  - [DA Namespace](#da-namespace)
  - [DA Header Namespace](#da-namespace)
  - [DA Data Namespace](#da-data-namespace)
  - [DA Block Time](#da-block-time)
  - [DA Mempool TTL](#da-mempool-ttl)
- [P2P Configuration (`p2p`)](#p2p-configuration-p2p)
  - [P2P Listen Address](#p2p-listen-address)
  - [P2P Peers](#p2p-peers)
  - [P2P Blocked Peers](#p2p-blocked-peers)
  - [P2P Allowed Peers](#p2p-allowed-peers)
- [RPC Configuration (`rpc`)](#rpc-configuration-rpc)
  - [RPC Server Address](#rpc-server-address)
  - [Enable DA Visualization](#enable-da-visualization)
- [Instrumentation Configuration (`instrumentation`)](#instrumentation-configuration-instrumentation)
  - [Enable Prometheus Metrics](#enable-prometheus-metrics)
  - [Prometheus Listen Address](#prometheus-listen-address)
  - [Maximum Open Connections](#maximum-open-connections)
  - [Enable Pprof Profiling](#enable-pprof-profiling)
  - [Pprof Listen Address](#pprof-listen-address)
- [Logging Configuration (`log`)](#logging-configuration-log)
  - [Log Level](#log-level)
  - [Log Format](#log-format)
  - [Log Trace (Stack Traces)](#log-trace-stack-traces)
- [Signer Configuration (`signer`)](#signer-configuration-signer)
  - [Signer Type](#signer-type)
  - [Signer Path](#signer-path)
  - [Signer Passphrase](#signer-passphrase)

## DA-Only Sync Mode

Evolve supports running nodes that sync exclusively from the Data Availability (DA) layer without participating in P2P networking. This mode is useful for:

- **Pure DA followers**: Nodes that only need the canonical chain data from DA
- **Resource optimization**: Reducing network overhead by avoiding P2P gossip
- **Simplified deployment**: No need to configure or maintain P2P peer connections
- **Isolated environments**: Nodes that should not participate in P2P communication

**To enable DA-only sync mode:**

1. **Leave P2P peers empty** (default behavior):

   ```yaml
   p2p:
     peers: "" # Empty or omit this field entirely
   ```

2. **Configure DA connection** (required):

   ```yaml
   da:
     address: "your-da-service:port"
     namespace: "your-namespace"
     # ... other DA configuration
   ```

3. **Optional**: You can still configure P2P listen address for potential future connections, but without peers, no P2P networking will occur.

When running in DA-only mode, the node will:

- ✅ Sync blocks and headers from the DA layer
- ✅ Validate transactions and maintain state
- ✅ Serve RPC requests
- ❌ Not participate in P2P gossip or peer discovery
- ❌ Not share blocks with other nodes via P2P
- ❌ Not receive transactions via P2P (only from direct RPC submission)

## Configs

Evolve configurations can be managed through a YAML file (typically `evnode.yml` located in `~/.evolve/config/` or `<your_home_dir>/config/`) and command-line flags. The system prioritizes configurations in the following order (highest priority first):

1. **Command-line flags:** Override all other settings.
2. **YAML configuration file:** Values specified in the `config.yaml` file.
3. **Default values:** Predefined defaults within Evolve.

Environment variables can also be used, typically prefixed with your executable's name (e.g., `YOURAPP_CHAIN_ID="my-chain"`).

## Base Configuration

These are fundamental settings for your Evolve node.

### Root Directory

**Description:**
The root directory where Evolve stores its data, including the database and configuration files. This is a foundational setting that dictates where all other file paths are resolved from.

**YAML:**
This option is not set within the YAML configuration file itself, as it specifies the location _of_ the configuration file and other application data.

**Command-line Flag:**
`--home <path>`
_Example:_ `--home /mnt/data/evolve_node`
_Default:_ `~/.evolve` (or a directory derived from the application name if `defaultHome` is customized).
_Constant:_ `FlagRootDir`

### Database Path

**Description:**
The path, relative to the Root Directory, where the Evolve database will be stored. This database contains blockchain state, blocks, and other critical node data.

**YAML:**
Set this in your configuration file at the top level:

```yaml
db_path: "data"
```

**Command-line Flag:**
`--rollkit.db_path <path>`
_Example:_ `--rollkit.db_path "node_db"`
_Default:_ `"data"`
_Constant:_ `FlagDBPath`

### Chain ID

**Description:**
The unique identifier for your chain. This ID is used to differentiate your network from others and is crucial for network communication and transaction validation.

**YAML:**
Set this in your configuration file at the top level:

```yaml
chain_id: "my-evolve-chain"
```

**Command-line Flag:**
`--chain_id <string>`
_Example:_ `--chain_id "super_rollup_testnet_v1"`
_Default:_ `"evolve"`
_Constant:_ `FlagChainID`

## Node Configuration (`node`)

Settings related to the core behavior of the Evolve node, including its mode of operation and block production parameters.

**YAML Section:**

```yaml
node:
  # ... node configurations ...
```

### Aggregator Mode

**Description:**
If true, the node runs in aggregator mode. Aggregators are responsible for producing blocks by collecting transactions, ordering them, and proposing them to the network.

**YAML:**

```yaml
node:
  aggregator: true
```

**Command-line Flag:**
`--rollkit.node.aggregator` (boolean, presence enables it)
_Example:_ `--rollkit.node.aggregator`
_Default:_ `false`
_Constant:_ `FlagAggregator`

### Light Client Mode

**Description:**
If true, the node runs in light client mode. Light clients rely on full nodes for block headers and state information, offering a lightweight way to interact with the chain without storing all data.

**YAML:**

```yaml
node:
  light: true
```

**Command-line Flag:**
`--rollkit.node.light` (boolean, presence enables it)
_Example:_ `--rollkit.node.light`
_Default:_ `false`
_Constant:_ `FlagLight`

### Block Time

**Description:**
The target time interval between consecutive blocks produced by an aggregator. This duration (e.g., "500ms", "1s", "5s") dictates the pace of block production.

**YAML:**

```yaml
node:
  block_time: "1s"
```

**Command-line Flag:**
`--rollkit.node.block_time <duration>`
_Example:_ `--rollkit.node.block_time 2s`
_Default:_ `"1s"`
_Constant:_ `FlagBlockTime`

### Maximum Pending Blocks

**Description:**
The maximum number of blocks that can be pending Data Availability (DA) submission. When this limit is reached, the aggregator pauses block production until some blocks are confirmed on the DA layer. Use 0 for no limit. This helps manage resource usage and DA layer capacity.

**YAML:**

```yaml
node:
  max_pending_blocks: 100
```

**Command-line Flag:**
`--rollkit.node.max_pending_blocks <uint64>`
_Example:_ `--rollkit.node.max_pending_blocks 50`
_Default:_ `0` (no limit)
_Constant:_ `FlagMaxPendingBlocks`

### Lazy Mode (Lazy Aggregator)

**Description:**
Enables lazy aggregation mode. In this mode, blocks are produced only when new transactions are available in the mempool or after the `lazy_block_interval` has passed. This optimizes resource usage by avoiding the creation of empty blocks during periods of inactivity.

**YAML:**

```yaml
node:
  lazy_mode: true
```

**Command-line Flag:**
`--rollkit.node.lazy_mode` (boolean, presence enables it)
_Example:_ `--rollkit.node.lazy_mode`
_Default:_ `false`
_Constant:_ `FlagLazyAggregator`

### Lazy Block Interval

**Description:**
The maximum time interval between blocks when running in lazy aggregation mode (`lazy_mode`). This ensures that blocks are produced periodically even if there are no new transactions, keeping the chain active. This value is generally larger than `block_time`.

**YAML:**

```yaml
node:
  lazy_block_interval: "30s"
```

**Command-line Flag:**
`--rollkit.node.lazy_block_interval <duration>`
_Example:_ `--rollkit.node.lazy_block_interval 1m`
_Default:_ `"30s"`
_Constant:_ `FlagLazyBlockTime`

### Trusted Hash

**Description:**
The initial trusted hash used to bootstrap the header exchange service. This allows nodes to start synchronizing from a specific, trusted point in the chain history instead of from the genesis block. When provided, the node will fetch the corresponding header/block from peers using this hash. If not provided, the node attempts to sync from genesis.

**YAML:**

```yaml
node:
  trusted_hash: "YOUR_TRUSTED_HASH_HEX_STRING"
```

**Command-line Flag:**
`--rollkit.node.trusted_hash <string>`
_Example:_ `--rollkit.node.trusted_hash ABCDEF012345...`
_Default:_ `""` (empty, sync from genesis)
_Constant:_ `FlagTrustedHash`

## Data Availability Configuration (`da`)

Parameters for connecting and interacting with the Data Availability (DA) layer, which Evolve uses to publish block data.

**YAML Section:**

```yaml
da:
  # ... DA configurations ...
```

### DA Service Address

**Description:**
The network address (host:port) of the Data Availability layer service. Evolve connects to this endpoint to submit and retrieve block data.

**YAML:**

```yaml
da:
  address: "localhost:26659"
```

**Command-line Flag:**
`--rollkit.da.address <string>`
_Example:_ `--rollkit.da.address 192.168.1.100:26659`
_Default:_ `""` (empty, must be configured if DA is used)
_Constant:_ `FlagDAAddress`

### DA Authentication Token

**Description:**
The authentication token required to interact with the DA layer service, if the service mandates authentication.

**YAML:**

```yaml
da:
  auth_token: "YOUR_DA_AUTH_TOKEN"
```

**Command-line Flag:**
`--rollkit.da.auth_token <string>`
_Example:_ `--rollkit.da.auth_token mysecrettoken`
_Default:_ `""` (empty)
_Constant:_ `FlagDAAuthToken`

### DA Gas Price

**Description:**
The gas price to use for transactions submitted to the DA layer. A value of -1 indicates automatic gas price determination (if supported by the DA layer). Higher values may lead to faster inclusion of data.

**YAML:**

```yaml
da:
  gas_price: 0.025
```

**Command-line Flag:**
`--rollkit.da.gas_price <float64>`
_Example:_ `--rollkit.da.gas_price 0.05`
_Default:_ `-1` (automatic)
_Constant:_ `FlagDAGasPrice`

### DA Gas Multiplier

**Description:**
A multiplier applied to the gas price when retrying failed DA submissions. Values greater than 1 increase the gas price on retries, potentially improving the chances of successful inclusion.

**YAML:**

```yaml
da:
  gas_multiplier: 1.1
```

**Command-line Flag:**
`--rollkit.da.gas_multiplier <float64>`
_Example:_ `--rollkit.da.gas_multiplier 1.5`
_Default:_ `1.0` (no multiplication)
_Constant:_ `FlagDAGasMultiplier`

### DA Submit Options

**Description:**
Additional options passed to the DA layer when submitting data. The format and meaning of these options depend on the specific DA implementation being used.

**YAML:**

```yaml
da:
  submit_options: "{"key":"value"}" # Example, format depends on DA layer
```

**Command-line Flag:**
`--rollkit.da.submit_options <string>`
_Example:_ `--rollkit.da.submit_options '{"custom_param":true}'`
_Default:_ `""` (empty)
_Constant:_ `FlagDASubmitOptions`

### DA Namespace

**Description:**
The namespace ID used when submitting blobs (block data) to the DA layer. This helps segregate data from different chains or applications on a shared DA layer.

**Note:** If only `namespace` is provided, it will be used for both headers and data, otherwise the `data_namespace` will be used for data. Doing so allows speeding up light clients.

**YAML:**

```yaml
da:
  namespace: "MY_UNIQUE_NAMESPACE_ID"
```

**Command-line Flag:**
`--rollkit.da.namespace <string>`
_Example:_ `--rollkit.da.namespace 0x1234567890abcdef`
_Default:_ `""` (empty)
_Constant:_ `FlagDANamespace`

### DA Data Namespace

**Description:**
The namespace ID specifically for submitting transaction data to the DA layer. Transaction data is submitted separately from headers, enabling nodes to sync only the data they need. The namespace value is encoded by the node to ensure proper formatting and compatibility with the DA layer.

**YAML:**

```yaml
da:
  data_namespace: "DATA_NAMESPACE_ID"
```

**Command-line Flag:**
`--rollkit.da.data_namespace <string>`
_Example:_ `--rollkit.da.data_namespace my_data_namespace`
_Default:_ Falls back to `namespace` if not set
_Constant:_ `FlagDADataNamespace`

### DA Block Time

**Description:**
The average block time of the Data Availability chain (specified as a duration string, e.g., "15s", "1m"). This value influences:

- The frequency of DA layer syncing.
- The maximum backoff time for retrying DA submissions.
- Calculation of transaction expiration when multiplied by `mempool_ttl`.

**YAML:**

```yaml
da:
  block_time: "6s"
```

**Command-line Flag:**
`--rollkit.da.block_time <duration>`
_Example:_ `--rollkit.da.block_time 12s`
_Default:_ `"6s"`
_Constant:_ `FlagDABlockTime`

### DA Mempool TTL

**Description:**
The number of DA blocks after which a transaction submitted to the DA layer is considered expired and potentially dropped from the DA layer's mempool. This also controls the retry backoff timing for DA submissions.

**YAML:**

```yaml
da:
  mempool_ttl: 20
```

**Command-line Flag:**
`--rollkit.da.mempool_ttl <uint64>`
_Example:_ `--rollkit.da.mempool_ttl 30`
_Default:_ `20`
_Constant:_ `FlagDAMempoolTTL`

## P2P Configuration (`p2p`)

Settings for peer-to-peer networking, enabling nodes to discover each other, exchange blocks, and share transactions.

**YAML Section:**

```yaml
p2p:
  # ... P2P configurations ...
```

### P2P Listen Address

**Description:**
The network address (host:port) on which the Evolve node will listen for incoming P2P connections from other nodes.

**YAML:**

```yaml
p2p:
  listen_address: "0.0.0.0:7676"
```

**Command-line Flag:**
`--rollkit.p2p.listen_address <string>`
_Example:_ `--rollkit.p2p.listen_address /ip4/127.0.0.1/tcp/26656`
_Default:_ `"/ip4/0.0.0.0/tcp/7676"`
_Constant:_ `FlagP2PListenAddress`

### P2P Peers

**Description:**
A comma-separated list of peer addresses (e.g., multiaddresses) that the node will attempt to connect to for bootstrapping its P2P connections. These are often referred to as seed nodes.

**For DA-only sync mode:** Leave this field empty (default) to disable P2P networking entirely. When no peers are configured, the node will sync exclusively from the Data Availability layer without participating in P2P gossip, peer discovery, or block sharing. This is useful for nodes that only need to follow the canonical chain data from DA.

**YAML:**

```yaml
p2p:
  peers: "/ip4/some_peer_ip/tcp/7676/p2p/PEER_ID1,/ip4/another_peer_ip/tcp/7676/p2p/PEER_ID2"
  # For DA-only sync, leave peers empty:
  # peers: ""
```

**Command-line Flag:**
`--rollkit.p2p.peers <string>`
_Example:_ `--rollkit.p2p.peers /dns4/seed.example.com/tcp/26656/p2p/12D3KooW...`
_Default:_ `""` (empty - enables DA-only sync mode)
_Constant:_ `FlagP2PPeers`

### P2P Blocked Peers

**Description:**
A comma-separated list of peer IDs that the node should block from connecting. This can be used to prevent connections from known malicious or problematic peers.

**YAML:**

```yaml
p2p:
  blocked_peers: "PEER_ID_TO_BLOCK1,PEER_ID_TO_BLOCK2"
```

**Command-line Flag:**
`--rollkit.p2p.blocked_peers <string>`
_Example:_ `--rollkit.p2p.blocked_peers 12D3KooW...,12D3KooX...`
_Default:_ `""` (empty)
_Constant:_ `FlagP2PBlockedPeers`

### P2P Allowed Peers

**Description:**
A comma-separated list of peer IDs that the node should exclusively allow connections from. If this list is non-empty, only peers in this list will be able to connect.

**YAML:**

```yaml
p2p:
  allowed_peers: "PEER_ID_TO_ALLOW1,PEER_ID_TO_ALLOW2"
```

**Command-line Flag:**
`--rollkit.p2p.allowed_peers <string>`
_Example:_ `--rollkit.p2p.allowed_peers 12D3KooY...,12D3KooZ...`
_Default:_ `""` (empty, allow all unless blocked)
_Constant:_ `FlagP2PAllowedPeers`

## RPC Configuration (`rpc`)

Settings for the Remote Procedure Call (RPC) server, which allows clients and applications to interact with the Evolve node.

**YAML Section:**

```yaml
rpc:
  # ... RPC configurations ...
```

### RPC Server Address

**Description:**
The network address (host:port) to which the RPC server will bind and listen for incoming requests.

**YAML:**

```yaml
rpc:
  address: "127.0.0.1:7331"
```

**Command-line Flag:**
`--rollkit.rpc.address <string>`
_Example:_ `--rollkit.rpc.address 0.0.0.0:26657`
_Default:_ `"127.0.0.1:7331"`
_Constant:_ `FlagRPCAddress`

### Enable DA Visualization

**Description:**
If true, enables the Data Availability (DA) visualization endpoints that provide real-time monitoring of blob submissions to the DA layer. This includes a web-based dashboard and REST API endpoints for tracking submission statistics, monitoring DA health, and analyzing blob details. Only aggregator nodes submit data to the DA layer, so this feature is most useful when running in aggregator mode.

**YAML:**

```yaml
rpc:
  enable_da_visualization: true
```

**Command-line Flag:**
`--rollkit.rpc.enable_da_visualization` (boolean, presence enables it)
_Example:_ `--rollkit.rpc.enable_da_visualization`
_Default:_ `false`
_Constant:_ `FlagRPCEnableDAVisualization`

See the [DA Visualizer Guide](../guides/da/visualizer.md) for detailed information on using this feature.

## Instrumentation Configuration (`instrumentation`)

Settings for enabling and configuring metrics and profiling endpoints, useful for monitoring node performance and debugging.

**YAML Section:**

```yaml
instrumentation:
  # ... instrumentation configurations ...
```

### Enable Prometheus Metrics

**Description:**
If true, enables the Prometheus metrics endpoint, allowing Prometheus to scrape operational data from the Evolve node.

**YAML:**

```yaml
instrumentation:
  prometheus: true
```

**Command-line Flag:**
`--rollkit.instrumentation.prometheus` (boolean, presence enables it)
_Example:_ `--rollkit.instrumentation.prometheus`
_Default:_ `false`
_Constant:_ `FlagPrometheus`

### Prometheus Listen Address

**Description:**
The network address (host:port) where the Prometheus metrics server will listen for scraping requests.

See [Metrics](../guides/metrics.md) for more details on what metrics are exposed.

**YAML:**

```yaml
instrumentation:
  prometheus_listen_addr: ":2112"
```

**Command-line Flag:**
`--rollkit.instrumentation.prometheus_listen_addr <string>`
_Example:_ `--rollkit.instrumentation.prometheus_listen_addr 0.0.0.0:9090`
_Default:_ `":2112"`
_Constant:_ `FlagPrometheusListenAddr`

### Maximum Open Connections

**Description:**
The maximum number of simultaneous connections allowed for the metrics server (e.g., Prometheus endpoint).

**YAML:**

```yaml
instrumentation:
  max_open_connections: 100
```

**Command-line Flag:**
`--rollkit.instrumentation.max_open_connections <int>`
_Example:_ `--rollkit.instrumentation.max_open_connections 50`
_Default:_ (Refer to `DefaultInstrumentationConfig()` in code, typically a reasonable number like 100)
_Constant:_ `FlagMaxOpenConnections`

### Enable Pprof Profiling

**Description:**
If true, enables the pprof HTTP endpoint, which provides runtime profiling data for debugging performance issues. Accessing these endpoints can help diagnose CPU and memory usage.

**YAML:**

```yaml
instrumentation:
  pprof: true
```

**Command-line Flag:**
`--rollkit.instrumentation.pprof` (boolean, presence enables it)
_Example:_ `--rollkit.instrumentation.pprof`
_Default:_ `false`
_Constant:_ `FlagPprof`

### Pprof Listen Address

**Description:**
The network address (host:port) where the pprof HTTP server will listen for profiling requests.

**YAML:**

```yaml
instrumentation:
  pprof_listen_addr: "localhost:6060"
```

**Command-line Flag:**
`--rollkit.instrumentation.pprof_listen_addr <string>`
_Example:_ `--rollkit.instrumentation.pprof_listen_addr 0.0.0.0:6061`
_Default:_ `"localhost:6060"`
_Constant:_ `FlagPprofListenAddr`

## Logging Configuration (`log`)

Settings that control the verbosity and format of log output from the Evolve node. These are typically set via global flags.

**YAML Section:**

```yaml
log:
  # ... logging configurations ...
```

### Log Level

**Description:**
Sets the minimum severity level for log messages to be displayed. Common levels include `debug`, `info`, `warn`, `error`.

**YAML:**

```yaml
log:
  level: "info"
```

**Command-line Flag:**
`--log.level <string>` (Note: some applications might use a different flag name like `--log_level`)
_Example:_ `--log.level debug`
_Default:_ `"info"`
_Constant:_ `FlagLogLevel` (value: "evolve.log.level", but often overridden by global app flags)

### Log Format

**Description:**
Sets the format for log output. Common formats include `text` (human-readable) and `json` (structured, machine-readable).

**YAML:**

```yaml
log:
  format: "text"
```

**Command-line Flag:**
`--log.format <string>` (Note: some applications might use a different flag name like `--log_format`)
_Example:_ `--log.format json`
_Default:_ `"text"`
_Constant:_ `FlagLogFormat` (value: "evolve.log.format", but often overridden by global app flags)

### Log Trace (Stack Traces)

**Description:**
If true, enables the inclusion of stack traces in error logs. This can be very helpful for debugging issues by showing the call stack at the point of an error.

**YAML:**

```yaml
log:
  trace: false
```

**Command-line Flag:**
`--log.trace` (boolean, presence enables it; Note: some applications might use a different flag name like `--log_trace`)
_Example:_ `--log.trace`
_Default:_ `false`
_Constant:_ `FlagLogTrace` (value: "evolve.log.trace", but often overridden by global app flags)

## Signer Configuration (`signer`)

Settings related to the signing mechanism used by the node, particularly for aggregators that need to sign blocks.

**YAML Section:**

```yaml
signer:
  # ... signer configurations ...
```

### Signer Type

**Description:**
Specifies the type of remote signer to use. Common options might include `file` (for key files) or `grpc` (for connecting to a remote signing service).

**YAML:**

```yaml
signer:
  signer_type: "file"
```

**Command-line Flag:**
`--rollkit.signer.signer_type <string>`
_Example:_ `--rollkit.signer.signer_type grpc`
_Default:_ (Depends on application, often "file" or none if not an aggregator)
_Constant:_ `FlagSignerType`

### Signer Path

**Description:**
The path to the signer file (if `signer_type` is `file`) or the address of the remote signer service (if `signer_type` is `grpc` or similar).

**YAML:**

```yaml
signer:
  signer_path: "/path/to/priv_validator_key.json" # For file signer
  # signer_path: "localhost:9000" # For gRPC signer
```

**Command-line Flag:**
`--rollkit.signer.signer_path <string>`
_Example:_ `--rollkit.signer.signer_path ./config`
_Default:_ (Depends on application)
_Constant:_ `FlagSignerPath`

### Signer Passphrase

**Description:**
The passphrase required to decrypt or access the signer key, particularly if using a `file` signer and the key is encrypted, or if the aggregator mode is enabled and requires it. This flag is not directly a field in the `SignerConfig` struct but is used in conjunction with it.

**YAML:**
This is typically not stored in the YAML file for security reasons but provided via flag or environment variable.

**Command-line Flag:**
`--rollkit.signer.passphrase <string>`
_Example:_ `--rollkit.signer.passphrase "mysecretpassphrase"`
_Default:_ `""` (empty)
_Constant:_ `FlagSignerPassphrase`
_Note:_ Be cautious with providing passphrases directly on the command line in shared environments due to history logging. Environment variables or secure input methods are often preferred.

---

This reference should help you configure your Evolve node effectively. Always refer to the specific version of Evolve you are using, as options and defaults may change over time.
