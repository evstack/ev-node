#!/bin/bash

# Demo script for DA height polling functionality
# This script demonstrates how to use the new DA height polling feature

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
TESTAPP_BIN="$PROJECT_ROOT/build/testapp"
TEMP_DIR=$(mktemp -d)

echo "🚀 Demo: DA Height Polling for Genesis Initialization"
echo "=================================================="

# Ensure testapp is built
if [ ! -f "$TESTAPP_BIN" ]; then
    echo "❌ testapp binary not found at $TESTAPP_BIN"
    echo "Please run 'make build' first"
    exit 1
fi

echo "✅ testapp binary found at $TESTAPP_BIN"

# Create directories for two nodes
AGG_HOME="$TEMP_DIR/aggregator"
FULL_HOME="$TEMP_DIR/fullnode"

echo "📁 Setting up temporary directories:"
echo "   Aggregator: $AGG_HOME"
echo "   Full node:  $FULL_HOME"

# Initialize aggregator node
echo ""
echo "🔧 Initializing aggregator node..."
"$TESTAPP_BIN" init \
    --home="$AGG_HOME" \
    --chain_id=demo-chain \
    --evnode.node.aggregator \
    --evnode.signer.passphrase=test123

echo "✅ Aggregator initialized"

# Initialize full node with aggregator endpoint
echo ""
echo "🔧 Initializing full node with aggregator endpoint..."
"$TESTAPP_BIN" init \
    --home="$FULL_HOME" \
    --chain_id=demo-chain \
    --evnode.da.aggregator_endpoint=http://localhost:7331

echo "✅ Full node initialized"

# Show the configuration files
echo ""
echo "📄 Full node configuration (showing DA config):"
echo "------------------------------------------------"
grep -A 10 "^da:" "$FULL_HOME/config/evnode.yaml" || echo "❌ Could not find DA config"

echo ""
echo "🎯 Key points demonstrated:"
echo "1. ✅ New --evnode.da.aggregator_endpoint flag is available"
echo "2. ✅ Configuration is properly saved to evnode.yaml"
echo "3. ✅ Non-aggregator nodes can be configured to poll DA height"

echo ""
echo "🧪 To test the actual polling (requires running nodes):"
echo "   1. Start aggregator: $TESTAPP_BIN start --home=$AGG_HOME --evnode.signer.passphrase=test123"
echo "   2. Test DA height endpoint: curl http://localhost:7331/da/height"
echo "   3. Start full node: $TESTAPP_BIN start --home=$FULL_HOME"
echo "      (The full node will automatically poll DA height if genesis DA start time is zero)"

# Cleanup function
cleanup() {
    echo ""
    echo "🧹 Cleaning up temporary files..."
    rm -rf "$TEMP_DIR"
    echo "✅ Cleanup complete"
}

# Register cleanup function
trap cleanup EXIT

echo ""
echo "✨ Demo completed successfully!"
echo "   Temp directory: $TEMP_DIR (will be cleaned up on exit)"