# Evolve EVM Chain Migration Guide

## Introduction

This guide covers how to perform a chain migration for Evolve EVM-based blockchains using reth's `dump-genesis` and `init` functions. Chain migration allows you to export the current state of your blockchain and initialize a new chain with that state, which is useful for major upgrades, network resets, or transitioning between different configurations.

## Prerequisites

Before starting a chain migration, ensure you have:

- A running Evolve EVM node with fully synchronized state
- reth CLI tool installed and configured
- Sufficient disk space for state export (at least 2x current state size)
- Access to all node operators in your network
- Coordination with external services (see [External Services Coordination](#external-services-coordination))
- Backup of critical data and configurations

## Understanding Chain Migration

Chain migration involves several critical steps:

1. **State Export**: Extracting the current blockchain state at a specific block height
2. **Genesis Generation**: Creating a new genesis file with the exported state
3. **Network Coordination**: Ensuring all validators and nodes migrate simultaneously
4. **Service Updates**: Updating external services to recognize the new chain

## Step-by-Step Migration Process

### 1. Pre-Migration Planning

Before initiating the migration, establish a clear plan:

```bash
# Document current chain parameters
curl -X POST -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}' \
  http://localhost:8545

# Record current block height for migration
curl -X POST -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
  http://localhost:8545
```

### 2. Coordinate Migration Block Height

All network participants must agree on:

- **Migration block height**: The specific block at which to export state
- **Migration timestamp**: When the migration will occur
- **New chain parameters**: Chain ID, network ID, and other configurations

### 3. Stop the Node at Target Height

Configure your node to stop at the agreed-upon migration height:

```bash
# Stop the Evolve node gracefully
docker stop evolve-node

# Or if running as a service
systemctl stop evolve-node
```

### 4. Export Current State Using reth dump-genesis

reth dump-genesis exports the current head state in the node’s local database into a genesis file.
It does not accept a --block flag, so to capture a specific migration height you must prepare the DB first.

⸻

Option 1 — Stop at the target height before dumping
- Sync your Evolve/reth node to the agreed migration height.
- Disable block production or stop the sequencer so no new blocks are added.
- Run dump-genesis while the DB is exactly at the migration height.

# Stop node at migration height
systemctl stop evolve-node

# Export state at current head (must be migration height)
reth dump-genesis \
  --datadir /path/to/reth/datadir \
  --output genesis-export.json


⸻

Option 2 — Rewind DB to migration height

If your node has already synced past the migration block:

reth db revert-to-block <MIGRATION_BLOCK_HEIGHT> --datadir /path/to/reth/datadir

# Then dump the state
reth dump-genesis \
  --datadir /path/to/reth/datadir \
  --output genesis-export.json


⸻

Verification after export:

# Number of accounts in alloc
jq '.alloc | length' genesis-export.json

# Review chain configuration
jq '.config' genesis-export.json

The exported genesis will contain:
- All account balances and nonces
- Contract bytecode and storage
- Current chain configuration


### 5. Prepare New Genesis Configuration

Once you have genesis-export.json, update it for the new chain parameters.

⸻

If staying in the same stack (only chain ID / forks change)

Use jq to adjust the config:

```bash
jq '.config.chainId = <NEW_CHAIN_ID> |
    .config.homesteadBlock = 0 |
    .config.eip150Block = 0 |
    .config.eip155Block = 0 |
    .config.eip158Block = 0 |
    .config.byzantiumBlock = 0 |
    .config.constantinopleBlock = 0 |
    .config.petersburgBlock = 0 |
    .config.istanbulBlock = 0 |
    .config.berlinBlock = 0 |
    .config.londonBlock = 0 |
    .timestamp = <NEW_GENESIS_TIMESTAMP>' \
    genesis-export.json > genesis-new.json
```


### 6. Initialize New Chain with reth init

Initialize the new chain using the modified genesis:

```bash
# Clean the old datadir or use a new one
rm -rf /path/to/new/reth/datadir

# Initialize with new genesis
reth init \
  --datadir /path/to/new/reth/datadir \
  --chain genesis-new.json

# Verify initialization
reth db stats --datadir /path/to/new/reth/datadir
```

### 7. Update Node Configuration

Update your Evolve node configuration for the new chain:

```yaml
# evolve.yml
chain_id: <NEW_CHAIN_ID>
genesis_file: /path/to/genesis-new.json
reth:
  datadir: /path/to/new/reth/datadir
  network_id: <NEW_NETWORK_ID>
```

### 8. Start the New Chain

Start your node with the new configuration:

```bash
# Start Evolve node with new configuration
evolve start --config evolve.yml

# Monitor logs for successful startup
tail -f /var/log/evolve/node.log
```

## External Services Coordination

### Bridges

Bridge operators must be notified well in advance to:

- **Pause deposits**: Stop accepting deposits before migration block
- **Process withdrawals**: Complete all pending withdrawals
- **Update contracts**: Deploy new bridge contracts on the migrated chain
- **Map balances**: Ensure locked tokens match bridged tokens on new chain

Coordination checklist:
```markdown
- [ ] Notify bridge operators 2 weeks before migration
- [ ] Pause bridge operations at block height - 100
- [ ] Export bridge state and locked balances
- [ ] Deploy new bridge contracts post-migration
- [ ] Verify balance mappings
- [ ] Resume operations after validation
```

### Block Explorers

Explorer operators need to:

- **Archive old chain data**: Maintain historical data accessibility
- **Index new chain**: Start indexing from new genesis
- **Update chain ID**: Configure explorer for new chain parameters
- **Redirect users**: Implement proper redirects and notifications

Explorer coordination:
```bash
# Provide explorer operators with:
- New genesis file
- New chain ID and network ID
- RPC endpoints for new chain
- Migration block height reference
- Old chain archive endpoints (if maintaining)
```

### Exchanges

Exchange integration requires careful coordination:

- **Trading halt**: Suspend trading pairs before migration
- **Deposit/withdrawal freeze**: Stop all chain interactions
- **Balance snapshot**: Record user balances at migration height
- **Node updates**: Update exchange nodes to new chain
- **Wallet updates**: Reconfigure hot/cold wallets
- **Resume operations**: Gradual reopening after validation

Exchange checklist:
```markdown
- [ ] 30-day advance notice to exchanges
- [ ] Provide technical migration guide
- [ ] Coordinate maintenance windows
- [ ] Share new chain parameters
- [ ] Provide test environment access
- [ ] Confirm balance reconciliation
- [ ] Coordinate reopening schedule
```

### DeFi Protocols

DeFi protocols and dApps require:

- **Pause operations**: Freeze smart contract interactions
- **State verification**: Ensure protocol state consistency
- **Contract redeployment**: May need new deployments on migrated chain
- **Liquidity migration**: Coordinate liquidity provider transitions
- **Oracle updates**: Reconfigure price feeds and data sources

### Wallet Providers

Wallet services need updates for:

- **Chain ID**: Update network configuration
- **RPC endpoints**: Point to new chain infrastructure
- **Transaction history**: Handle historical data appropriately
- **User notifications**: Inform users about the migration

## Important Considerations

### Data Integrity

- **Verify state export**: Check that all accounts and contracts are included
- **Validate balances**: Ensure total supply remains consistent
- **Test transactions**: Verify that transactions work on the new chain
- **Cross-reference**: Compare state roots between export and import

### Network Consensus

- **Validator coordination**: All validators must migrate simultaneously
- **Genesis agreement**: All nodes must use identical genesis files
- **Time synchronization**: Ensure all nodes have synchronized clocks
- **Peer discovery**: Update bootnodes and peer lists

### Rollback Planning

Prepare for potential issues:

```bash
# Backup original datadir before migration
tar -czf reth-backup-$(date +%Y%m%d).tar.gz /path/to/reth/datadir

# Keep original genesis and configuration
cp genesis-export.json genesis-backup.json
cp evolve.yml evolve-backup.yml
```

### Communication Plan

Establish clear communication channels:

1. **Technical team channel**: For real-time coordination
2. **Validator notifications**: Direct communication with validators
3. **Public announcements**: User-facing migration updates
4. **Service provider updates**: Direct line to exchanges/explorers
5. **Emergency contacts**: 24/7 availability during migration

## Post-Migration Validation

### 1. Chain Health Checks

```bash
# Verify chain is producing blocks
curl -X POST -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
  http://localhost:8545

# Check peer connectivity
curl -X POST -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"net_peerCount","params":[],"id":1}' \
  http://localhost:8545

# Verify chain ID
curl -X POST -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}' \
  http://localhost:8545
```

### 2. State Verification

```bash
# Sample account balance checks
curl -X POST -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_getBalance","params":["0xADDRESS","latest"],"id":1}' \
  http://localhost:8545

# Verify contract code
curl -X POST -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_getCode","params":["0xCONTRACT","latest"],"id":1}' \
  http://localhost:8545
```

### 3. Service Validation

- **Bridges**: Test small value transfers in both directions
- **Explorers**: Verify block and transaction indexing
- **Exchanges**: Confirm deposit/withdrawal processing
- **DeFi**: Check protocol functionality and TVL consistency

## Migration Timeline Template

Here's a recommended timeline for chain migration:

```markdown
### T-30 days
- [ ] Initial migration announcement
- [ ] Technical documentation published
- [ ] Begin coordination with service providers

### T-14 days
- [ ] Finalize migration block height
- [ ] Confirm participation from all validators
- [ ] Service providers confirm readiness

### T-7 days
- [ ] Final migration parameters locked
- [ ] Test migration on testnet
- [ ] Public reminder announcements

### T-24 hours
- [ ] Final coordination call
- [ ] Service suspension notifications
- [ ] Last-chance backups

### T-0 (Migration)
- [ ] Stop nodes at target height
- [ ] Export state
- [ ] Generate new genesis
- [ ] Initialize new chain
- [ ] Start new network
- [ ] Initial validation

### T+6 hours
- [ ] Service providers begin reconnection
- [ ] Initial functionality tests
- [ ] Public progress updates

### T+24 hours
- [ ] Full service restoration
- [ ] Post-migration report
- [ ] Address any issues
```

## Troubleshooting

### Common Issues and Solutions

**State Export Failures**:
```bash
# If export fails due to memory, increase heap size
export RETH_HEAP_SIZE=8g
reth dump-genesis --datadir /path/to/datadir
```

**Genesis Initialization Errors**:
```bash
# Verify genesis file format
jq empty genesis-new.json  # Should return nothing if valid

# Check for required fields
jq '.config.chainId, .alloc, .timestamp' genesis-new.json
```

**Node Sync Issues**:
```bash
# Clear peer database and reconnect
rm -rf /path/to/datadir/nodes
# Restart with explicit bootnodes
evolve start --bootnodes "enode://..."
```

## Security Considerations

- **Private key management**: Never expose validator keys during migration
- **Genesis distribution**: Use secure channels for genesis file distribution
- **Verification process**: Multiple parties should verify genesis independently
- **Access control**: Limit access to migration operations to authorized personnel
- **Audit trail**: Maintain detailed logs of all migration activities

## Conclusion

Chain migration is a complex process requiring careful planning and coordination. Success depends on:

1. Clear communication with all stakeholders
2. Thorough testing before migration
3. Precise coordination during migration
4. Comprehensive validation after migration
5. Ready rollback plans for contingencies

Always perform a test migration on a testnet environment before attempting mainnet migration. Document all steps and maintain communication channels throughout the process.

## Additional Resources

- [reth Documentation](https://paradigmxyz.github.io/reth/)
- [Evolve EVM Single Sequencer Guide](./single.md)
- [Evolve State Backup Guide](./reth-backup.md)
- [Ethereum JSON-RPC Specification](https://ethereum.github.io/execution-apis/api-documentation/)
