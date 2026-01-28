# Attester Node

Attester nodes participate in the validator network to provide faster soft finality through attestations. This is an advanced feature for chains requiring sub-DA-finality confirmation times.

## Overview

Attesters:

- Validate blocks produced by the aggregator
- Sign attestations confirming block validity
- Participate in a soft consensus protocol
- Enable faster finality than DA-only confirmation

## Status

The attester network feature is under active development. This documentation will be updated as the feature matures.

For technical details on the validator network design, see [ADR-022: Validator Network](https://github.com/evstack/ev-node/blob/main/specs/src/adr/adr-022-validator-network.md).

## How It Works

### Soft Finality

Without attesters, finality depends on DA confirmation (~6-12 seconds for Celestia). With an attester network:

1. Aggregator produces block
2. Attesters validate and sign attestations
3. When threshold of attestations collected, block has soft finality
4. DA finality provides hard finality later

### Trust Model

- Soft finality requires trusting the attester set (configurable threshold)
- Hard finality (DA) remains trustless
- Applications can choose which finality level to wait for

## Configuration (Preview)

```bash
# Run as attester (preview configuration)
evnode start \
  --evnode.node.attester \
  --evnode.da.address http://localhost:26658 \
  --evnode.p2p.peers /ip4/sequencer.example.com/tcp/7676/p2p/12D3KooW...
```

## Use Cases

### Low-Latency Applications

Applications requiring confirmation faster than DA finality:

- Trading platforms
- Gaming
- Real-time settlement

### Enhanced Security

Additional validation layer before DA confirmation:

- Multi-party validation
- Early fraud detection

## See Also

- [Finality Concepts](/concepts/finality) - Understanding finality in Evolve
- [Full Node Guide](/guides/running-nodes/full-node) - Running a full node
