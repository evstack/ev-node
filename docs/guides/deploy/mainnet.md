---
description: Checklist and guide for launching an EVM mainnet using ev-node and ev-reth.
---

# EVM Mainnet Checklist

This guide covers launching a mainnet using **ev-reth** and **ev-node**.

## Ev-node

Ev-node is the sequencer that creates blocks, propagates them to other peers, and submits them to the DA layer.

### Chain ID

- Pick a unique EVM chain ID for your network
- Verify it does not collide with existing chains at [chainlist.org](https://chainlist.org)

### Block Time

- Pick a block time for your chain
- **Optional:** Decide if you would like lazy blocks

### Data Availability (DA)

| Configuration | Description |
|---|---|
| Header Namespace | Required. Namespace for block headers. |
| Data Namespace | Required. Namespace for block data. Two namespaces are recommended, but one can be used. |
| Forced Inclusion Namespace | Optional. For censorship resistance. |
| DA Block Time | Controls blob submission frequency. Should be **higher** than the underlying DA block time. |

#### DA Account Funding

The DA account needs tokens to submit blobs to the DA layer. If the account runs dry, blob submission stops and your chain halts.

- Fund the DA account with sufficient tokens before launch
- Set up balance monitoring with alerts at a threshold that gives you enough runway to top up (e.g. alert when balance drops below 48 hours of estimated submission costs)
- Establish a process for topping up the DA account

### Sequencer Key Management

The sequencer signing key is the most security-critical component of your chain. A compromised key allows an attacker to produce arbitrary blocks.

- Use an HSM or remote signer for the sequencer key in production -- do not store plaintext keys on disk
- Restrict access to the sequencer machine to a minimal set of operators
- Have a key rotation plan ready before launch

### P2P

- Configure p2p peers for a stable network
- The sequencer should be connected to **at least two full nodes you control**
- Third-party full nodes should connect to your full nodes, **not** directly to the sequencer

### Network Security

- Place the sequencer behind a firewall; only allow p2p connections from your own full nodes
- Apply rate limiting on public RPC endpoints to prevent abuse
- Consider DDoS mitigation (e.g. Cloudflare, HAProxy) in front of public-facing full nodes
- Restrict SSH and management ports to a VPN or bastion host

### Metrics and Monitoring

- Set up a metric gathering system (Prometheus + Grafana recommended)
- ev-metrics was created to help with basic metrics and alerting

Key metrics to monitor:

| Metric | Why |
|---|---|
| Block production rate | Detect if the sequencer has stalled |
| DA submission lag / failures | Catch blob submission issues before they become critical |
| Peer count | Ensure network connectivity is healthy |
| Mempool depth | Detect congestion or spam |
| Disk usage | Prevent nodes from running out of storage |
| Sequencer balance | Ensure the sequencer can pay for DA submissions |
| DA account balance | Chain halts if this runs dry |
| RPC latency / error rate | Catch degraded user experience |

### RPC

- Use the full nodes connected to the sequencer for public/application RPCs
- **Do not expose the sequencer directly**

## Ev-reth

Ev-reth is the execution engine. It uses reth as a library to make custom configurations for the Evolve use case. Changes are documented in the readme.

### Precompiles

Ev-reth comes with a set of optional precompiles:

| Precompile | Description |
|---|---|
| Basefee Redirect | Redirects the basefee (burned under EIP-1559) to a specified address |
| Native Mint & Burn | Allows minting and burning the native token. Can be used with a bridge like Hyperlane. |

### Checklist

- Decide which precompiles are needed
- Set admin accounts for precompiles / basefee redirect
- Decide if a proxy contract is needed (provided proxy contract)
- Decide on EIP-1559 configurations
- Configure basefee
- Optional: Feevault contract

### Backup and Recovery

Establish a backup and recovery strategy before launch. See the [Reth State Backup](../evm/reth-backup.md) guide for detailed instructions.

- Take periodic state snapshots (frequency depends on your RTO requirements)
- Test the recovery procedure on a staging environment before mainnet launch
- Keep at least one full node with archival state as a fallback
- Document the recovery runbook so any operator can execute it

### Upgrade Strategy

Plan how you will ship new versions of ev-node and ev-reth to mainnet.

- **Rolling restart order:** Upgrade full nodes first, then the sequencer. This ensures full nodes can handle the new version before the sequencer starts producing blocks with it.
- **Hard fork coordination:** If a release includes consensus-breaking changes, coordinate an activation height with all node operators in advance.
- **Rollback plan:** Know how to revert to the previous binary and state if an upgrade causes issues. Test this on a staging network.
- **Communication:** Establish a channel (Telegram, Discord, etc.) to notify node operators of upcoming upgrades and activation heights.

## Chain Startup Flow

### Genesis

Configuring the genesis is the first step to starting the chain.

If using the proxy admin contract alongside both native mint/burn and basefee redirect, set the admin of those to the proxy contract. This allows the chain to modify the admin to a multisig later.

#### Genesis Token Distribution

Define the initial token supply and allocation before generating the genesis file.

- Decide the total initial supply and how it is split (team, treasury, partners, bridge reserves, etc.)
- Configure genesis balances in the `alloc` section of the genesis file
- Ensure the sequencer EOA has enough balance to submit transactions
- If using a bridge (e.g. Hyperlane), reserve sufficient supply for the bridge contract

:::info
All flows below assume usage of the proxy admin contract.
:::

### Flow 1: Full Setup

**Basefee redirect + feevault + native mint/burn + bridge (Hyperlane)**

#### Genesis Setup

Embed the proxy contract with an EOA address as admin. The EOA must have at least one token to submit transactions. The proxy contract will have a predictable address, which is added to the `EvolveConfig` in the Chain Config as admin for feevault and native mint/burn.

**Steps:**

1. Pick an EOA as admin of the proxy contract
2. Set EOA and create alloc of proxy contract for the genesis file
3. Set the admin proxy contract as admin in Evolve config:

```json
{
  "evolve": {
    "baseFeeSink": "<EOA_to_receive_funds>",
    "baseFeeRedirectActivationHeight": 0,
    "mintAdmin": "<EOA_admin/proxyContract>",
    "mintPrecompileActivationHeight": 0,
    "contractSizeLimit": 131072,
    "contractSizeLimitActivationHeight": 0
  }
}
```

4. Pick a max contract size (24kb default, 128kb is a safe upgrade)
5. Pick EIP-1559 config:

```json
{
  "baseFeeMaxChangeDenominator": 8,
  "baseFeeElasticityMultiplier": 2,
  "initialBaseFeePerGas": 1000000000
}
```

#### Post Genesis

- Deploy Hyperlane native mint contract (if using Hyperlane)
- Provide funds to partner wallets deploying on your chain
- Connect full nodes via reth p2p on top of the Evolve system (consult reth documentation for key discovery and connection)

### Flow 2: Minimal Setup

**Basefee redirect only, with an EOA receiving all funds.**

#### Genesis Setup

1. Set EOA address as the sink in the Evolve config
2. Pick a max contract size (24kb default, 128kb is a safe upgrade)
3. Pick EIP-1559 config:

```json
{
  "baseFeeMaxChangeDenominator": 8,
  "baseFeeElasticityMultiplier": 2,
  "initialBaseFeePerGas": 1000000000
}
```

#### Post Genesis

- Provide funds to partner wallets deploying on your chain
- Connect full nodes via reth p2p on top of the Evolve system (consult reth documentation for key discovery and connection)
