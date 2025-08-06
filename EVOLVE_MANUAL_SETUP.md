# Evolve Stack Manual Setup Guide

This guide documents the complete process for running the Evolve stack (ev-reth, ev-node, local-da) locally in manual mode.

## Prerequisites

- Go installed
- Rust/Cargo installed  
- `jq` installed for JSON parsing
- OpenSSL for JWT generation

## Important Notes

⚠️ **CRITICAL**: Always reset state directories when restarting or encountering "validation failed" errors
⚠️ **NEVER use `--release` flag when building ev-reth during testing** - debug mode only

## Complete Setup Process

### Step 1: Stop All Services and Clear State

```bash
# Stop all services first
pkill -f local-da && pkill -f ev-reth && pkill -f evm-single && pkill -f spamoor-daemon

# Clear state directories (essential for fresh start)
rm -rf ~/.evm-single
rm -rf "/Users/manav/Library/Application Support/reth"
```

### Step 2: Build ev-reth (Debug Mode Only)

```bash
cd ev-reth && make clean && make build-dev
```

**Note**: NEVER use `--release` flag when testing - debug mode is faster for development.

### Step 3: Setup and Run local-da

```bash
cd ev-node/da/cmd/local-da && go build -o local-da . && ./local-da -listen-all &
```

### Step 4: Start ev-reth with Correct Configuration

```bash
cd ev-reth
./target/debug/ev-reth node \
  --chain /Users/manav/Documents/Celestia/ev-node/apps/evm/single/chain/genesis.json \
  --authrpc.addr 0.0.0.0 --authrpc.port 8551 --authrpc.jwtsecret /Users/manav/Documents/Celestia/ev-node/apps/evm/single/jwttoken/jwt.hex \
  --http --http.addr 0.0.0.0 --http.port 8545 --http.api eth,net,web3,txpool \
  --ws --ws.addr 0.0.0.0 --ws.port 8546 --ws.api eth,net,web3 \
  --engine.persistence-threshold 0 --engine.memory-block-buffer-target 0 \
  --disable-discovery \
  --txpool.queued-max-count 200000 \
  --txpool.max-account-slots 2048 \
  --txpool.additional-validation-tasks 16 \
  --ev-reth.enable
```

### Step 5: Get Genesis Hash and Setup evm-single

```bash
# Wait for ev-reth to start and get genesis hash
curl -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["0x0", false],"id":1}' http://localhost:8545 | jq -r '.result.hash'

# Build and initialize evm-single
cd ev-node/apps/evm/single && go build -o evm-single .
./evm-single init --rollkit.node.aggregator=true --rollkit.signer.passphrase secret
```

### Step 6: Start evm-single

```bash
./evm-single start \
  --evm.jwt-secret $(cat /Users/manav/Documents/Celestia/ev-node/apps/evm/single/jwttoken/jwt.hex) \
  --evm.genesis-hash 0x2b8bbb1ea1e04f9c9809b4b278a8687806edc061a356c7dbc491930d8e922503 \
  --rollkit.node.block_time 1s --rollkit.node.aggregator=true \
  --rollkit.signer.passphrase secret --rollkit.da.address=http://localhost:7980 \
  --evm.eth-url=http://localhost:8545 --evm.engine-url=http://localhost:8551
```

## Verification Commands

### Check Running Services
```bash
ps aux | grep -E "(local-da|ev-reth|evm-single)" | grep -v grep
```

### Check Block Production
```bash
# Get current block number
curl -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' http://localhost:8545

# Check again after a few seconds to verify blocks are being produced
sleep 5 && curl -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' http://localhost:8545
```

### Check Account Balance (for debugging)
```bash
# Check balance of a specific address
curl -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"eth_getBalance","params":["ADDRESS_HERE", "latest"],"id":1}' http://localhost:8545
```

## Running Spamoor (Optional)

### Start Spamoor Daemon
```bash
cd spamoor
./bin/spamoor-daemon -h "http://localhost:8545" -p "PRIVATE_KEY_HERE"
```

**Note**: Make sure to use a private key that corresponds to a pre-funded account in the genesis file.

### Access Spamoor Web Interface
- Navigate to: http://localhost:8080
- Enable spammers and check graphs for transaction activity

## Service Ports

- **ev-reth HTTP**: 8545
- **ev-reth WebSocket**: 8546  
- **ev-reth AuthRPC**: 8551
- **local-da**: 7980
- **evm-single**: 7331
- **spamoor**: 8080

## Key Success Factors

1. ✅ **Use debug mode only** for ev-reth (never --release during testing)
2. ✅ **Always reset state directories** when restarting or encountering errors
3. ✅ **Use correct genesis file**: `ev-node/apps/evm/single/chain/genesis.json`
4. ✅ **Include --ev-reth.enable flag** when starting ev-reth
5. ✅ **Initialize evm-single** with aggregator and signer passphrase flags
6. ✅ **Start services in correct order**: local-da → ev-reth → evm-single

## Troubleshooting

### "Validation failed" errors
- **Solution**: Reset state directories and restart all services

### No blocks being produced  
- **Check**: All three services are running
- **Check**: Correct genesis hash is being used
- **Check**: JWT secret is properly configured

### Spamoor "insufficient funds" errors
- **Check**: Private key corresponds to pre-funded genesis account
- **Check**: Account has sufficient balance for transactions

### Port conflicts
- **Check**: No other services are using the required ports
- **Solution**: Kill any conflicting processes

## Genesis File Pre-funded Accounts

The genesis file contains three pre-funded accounts:
- `0xd143C405751162d0F96bEE2eB5eb9C61882a736E`
- `0x944fDcD1c868E3cC566C78023CcB38A32cDA836E`  
- `0x4567BF59F76c18cEa2131BDA24A7b70744308f54`

Each has balance: `0x4a47e3c12448f4ad000000` (very large amount)

## Quick Restart Script

```bash
#!/bin/bash
# Quick restart script
pkill -f local-da && pkill -f ev-reth && pkill -f evm-single && pkill -f spamoor-daemon
rm -rf ~/.evm-single
rm -rf "/Users/manav/Library/Application Support/reth"

# Then follow steps 3-6 above
```

---

*This guide ensures consistent and reliable setup of the Evolve stack for local development and testing.*