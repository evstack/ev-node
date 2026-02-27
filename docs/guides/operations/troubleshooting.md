# Troubleshooting

Common issues and solutions when running Evolve nodes.

## Diagnostic Commands

### Check Node Status

```bash
# Health check
curl http://localhost:7331/health/live
curl http://localhost:7331/health/ready

# Node status
curl http://localhost:26657/status
```

### View Logs

Use the log command that matches your runtime:

```bash
# systemd service
journalctl -u evnode -f

# foreground process
./evnode start ... 2>&1 | tee evnode.log

# docker container
docker logs -f <container_name>
```

## Common Issues

### Node Won't Start

**Symptom:** Node exits immediately after starting.

**Solutions:**

1. Check for port conflicts:

```bash
lsof -i :26657
lsof -i :7676
```

1. Verify configuration file syntax:

```bash
cat ~/.evnode/config/evnode.yml
```

1. Check data directory permissions:

```bash
ls -la ~/.evnode/data
```

### DA Connection Failures

**Symptom:** Logs show `DA layer submission failed` errors.

**Error example:**

```text
ERR DA layer submission failed error="connection refused"
```

**Solutions:**

1. Verify DA endpoint is reachable:

```bash
curl http://localhost:26658/health
```

1. Check authentication token (Celestia):

```bash
celestia light auth write --p2p.network mocha
```

1. Verify DA node is fully synced:

```bash
celestia header sync-state
```

### Out of DA Funds

**Symptom:** `Code: 19` errors in logs.

**Error example:**

```text
ERR DA layer submission failed error="Codespace: 'sdk', Code: 19, Message: "
```

**Solutions:**

1. Check DA account balance
2. Fund the account with more tokens
3. Increase gas price to unstick pending transactions:

```bash
--evnode.da.gas_price 0.05
```

See [Restart Chain Guide](/guides/restart-chain) for detailed steps.

### P2P Connection Issues

**Symptom:** Node not syncing, no peers connected.

**Solutions:**

1. Verify peer address format:

```bash
# Correct format
/ip4/1.2.3.4/tcp/7676/p2p/12D3KooWABC...

# NOT just the peer ID
12D3KooWABC...
```

1. Check firewall allows P2P port:

```bash
sudo ufw status
# Ensure port 7676 (or your P2P port) is allowed
```

1. Try DA-only sync mode (no P2P):

```bash
evnode start --evnode.da.address http://localhost:26658
# Leave --evnode.p2p.peers empty
```

### Node Falling Behind

**Symptom:** `catching_up: true` in status, height increasing slowly.

**Solutions:**

1. Check system resources:

```bash
htop
df -h
```

1. Increase DA request timeout:

```bash
--evnode.da.request_timeout 60s
```

1. Verify DA layer is responding quickly:

```bash
time curl http://localhost:26658/header/sync_state
```

### Execution Layer Desync

**Symptom:** State root mismatches, execution errors.

**EVM (ev-reth):**

```bash
# Check ev-reth logs for errors
journalctl -u ev-reth -f

# Verify Engine API connectivity
curl -X POST -H "Content-Type: application/json" \
  -H "Authorization: Bearer $(cat jwt.hex)" \
  --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
  http://localhost:8551
```

**Cosmos SDK (ev-abci):**

```bash
# Check app hash consistency
curl http://localhost:26657/status | jq '.sync_info'
```

## Reset Procedures

### Soft Reset (Keep Genesis)

Reset state while keeping configuration:

```bash
# Stop the node
systemctl stop evnode

# Clear data directory
rm -rf ~/.evnode/data/*

# Restart
systemctl start evnode
```

### Hard Reset (Full Reinitialize)

Complete reset including configuration:

```bash
# Stop the node
systemctl stop evnode

# Remove everything
rm -rf ~/.evnode

# Reinitialize
evnode init
```

### Reset EVM State (ev-reth)

```bash
# Stop both nodes
systemctl stop evnode ev-reth

# Clear ev-reth data
rm -rf ~/.ev-reth/db

# Clear ev-node cache
rm -rf ~/.evnode/data/cache

# Restart
systemctl start ev-reth evnode
```

## Log Analysis

### Enable Debug Logging

```bash
evnode start --log.level debug
```

Or in configuration:

```yaml
log:
  level: debug
```

## Performance Issues

### High Memory Usage

1. Reduce cache size in configuration
2. Enable lazy aggregation mode
3. Limit max pending blocks:

```bash
--evnode.node.max_pending_blocks 50
```

### High CPU Usage

1. Increase block time:

```bash
--evnode.node.block_time 2s
```

1. Check for transaction spam
2. Monitor execution layer performance

### Disk Space

1. Check disk usage:

```bash
du -sh ~/.evnode/data/*
```

1. Prune old data (if supported by execution layer)
2. Consider moving data to larger disk

## Getting Help

1. Check logs for specific error messages
2. Search [GitHub Issues](https://github.com/evstack/ev-node/issues)
3. Join the community Discord for support

## See Also

- [Reset State Guide](/guides/reset-state) - Detailed reset procedures
- [Restart Chain Guide](/guides/restart-chain) - Restarting after issues
- [Monitoring Guide](/guides/operations/monitoring) - Proactive monitoring
