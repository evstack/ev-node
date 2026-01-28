# Upgrades

Guide for upgrading Evolve nodes and handling version migrations.

## Upgrade Types

### Minor Upgrades (Patch/Minor Version)

Non-breaking changes, bug fixes, and minor improvements.

**Process:**

1. Stop the node
2. Replace binary
3. Restart

```bash
# Stop
systemctl stop evnode

# Upgrade (example with go install)
go install github.com/evstack/ev-node@v1.2.3

# Restart
systemctl start evnode
```

### Major Upgrades (Breaking Changes)

May require state migration or coordinated network upgrade.

**Process:**

1. Review changelog for breaking changes
2. Coordinate upgrade height with network
3. Stop at designated height
4. Upgrade binary
5. Run any migration scripts
6. Restart

## ev-node Upgrades

### Check Current Version

```bash
evnode version
```

### Upgrade Binary

**Using Go:**

```bash
go install github.com/evstack/ev-node@latest
```

**Using Docker:**

```bash
docker pull evstack/evnode:latest
```

**From Source:**

```bash
cd ev-node
git fetch --tags
git checkout v1.2.3
make build
```

### Configuration Changes

After upgrading, check for new or changed configuration options:

1. Review the [changelog](https://github.com/evstack/ev-node/releases)
2. Compare your config with the new defaults
3. Update configuration as needed

## ev-reth Upgrades

### Version Compatibility

ev-reth versions must be compatible with ev-node. Check the compatibility matrix:

| ev-node | ev-reth |
|---------|---------|
| v1.x | v0.x |

### Upgrade Process

```bash
# Stop both nodes
systemctl stop evnode ev-reth

# Upgrade ev-reth
cd ev-reth
git fetch --tags
git checkout v0.2.0
cargo build --release

# Verify chainspec compatibility
# (check for new required fields)

# Restart
systemctl start ev-reth evnode
```

### Database Migrations

Some ev-reth upgrades require database migration:

```bash
# Check if migration needed
ev-reth db version

# Run migration if needed
ev-reth db migrate
```

## ev-abci Upgrades

### Cosmos SDK Compatibility

ev-abci tracks Cosmos SDK versions. Ensure your app's SDK version is compatible:

| ev-abci | Cosmos SDK |
|---------|------------|
| v1.x | v0.50.x |

### Module Upgrades

For Cosmos SDK apps with custom modules:

1. Update module dependencies in `go.mod`
2. Run any module migration handlers
3. Update genesis if needed

```go
// In app.go upgrade handler
app.UpgradeKeeper.SetUpgradeHandler(
    "v2",
    func(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
        // Migration logic
        return app.ModuleManager.RunMigrations(ctx, app.Configurator(), fromVM)
    },
)
```

## Coordinated Network Upgrades

For networks with multiple node operators:

### Planning

1. Announce upgrade timeline (minimum 1 week notice)
2. Agree on upgrade block height
3. Share upgrade binary/instructions

### Execution

1. All nodes stop at designated height
2. Operators upgrade binaries
3. Coordinators verify readiness
4. Network restarts

### Handling Stragglers

If some nodes don't upgrade:

- They will reject new blocks (if consensus rules changed)
- They can sync from upgraded nodes after upgrading

## Rollback Procedures

If an upgrade causes issues:

### ev-node Rollback

```bash
# Stop
systemctl stop evnode

# Restore previous binary
cp /backup/evnode-v1.1.0 /usr/local/bin/evnode

# Optionally restore data
# (only if upgrade corrupted state)
rm -rf ~/.evnode/data
cp -r /backup/evnode-data ~/.evnode/data

# Restart
systemctl start evnode
```

### ev-reth Rollback

```bash
# Stop
systemctl stop ev-reth evnode

# Restore binary
cp /backup/ev-reth-v0.1.0 /usr/local/bin/ev-reth

# Restore database if needed
rm -rf ~/.ev-reth/db
cp -r /backup/ev-reth-db ~/.ev-reth/db

# Restart
systemctl start ev-reth evnode
```

## State Migration

### Export State

Before major upgrades, export state:

```bash
# ev-node
evnode export > state-export.json

# Cosmos SDK
appd export --height <height> > genesis-export.json
```

### Migrate State

If state format changes:

```bash
# Run migration tool
evnode migrate state-export.json --to-version v2 > state-migrated.json
```

### Import State

```bash
# Initialize with migrated state
evnode init --genesis state-migrated.json
```

## Best Practices

### Pre-Upgrade Checklist

- [ ] Review changelog for breaking changes
- [ ] Test upgrade on testnet first
- [ ] Backup current state
- [ ] Backup configuration files
- [ ] Notify dependent services
- [ ] Schedule maintenance window

### Post-Upgrade Verification

- [ ] Node starts successfully
- [ ] Blocks are being produced/synced
- [ ] RPC endpoints responding
- [ ] Metrics reporting correctly
- [ ] P2P connections established

### Automation

Consider automating upgrades with tools like:

- Ansible playbooks
- Kubernetes operators
- systemd timers for scheduled upgrades

## See Also

- [Troubleshooting Guide](/guides/operations/troubleshooting) - Handling upgrade issues
- [Deployment Guide](/guides/operations/deployment) - Infrastructure setup
