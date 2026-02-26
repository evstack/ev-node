# Finality

Finality determines when a transaction is irreversible. Evolve has a multi-stage finality model.

## Finality Stages

```
Transaction Submitted
        │
        ▼
┌───────────────────┐
│  Soft Confirmed   │  ← Block produced, gossiped via P2P
└─────────┬─────────┘
          │
          ▼
┌───────────────────┐
│   DA Finalized    │  ← DA layer confirms inclusion
└───────────────────┘
```

### Soft Confirmation

When a block is produced and gossiped via P2P:

- **Latency**: Milliseconds (block time)
- **Guarantee**: Sequencer has committed to this ordering
- **Risk**: Sequencer could equivocate (produce conflicting blocks)

### DA Finalized

When the DA layer confirms the block is included:

- **Latency**: ~6 seconds (Celestia)
- **Guarantee**: Block data is permanently available and ordered
- **Risk**: None (assuming DA layer security)

## Choosing Finality Thresholds

| Use Case | Recommended Finality |
|----------|---------------------|
| Display balance | Soft confirmation |
| Accept payment | Soft confirmation |
| Process withdrawal | DA finalized |
| Bridge transfer | DA finalized |

## Configuration

Block time affects soft confirmation latency:

```yaml
node:
  block-time: 100ms
```

DA finality depends on the DA layer. Celestia provides ~6 second finality.
