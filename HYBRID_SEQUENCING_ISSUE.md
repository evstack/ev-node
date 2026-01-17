# GitHub Issue: feat: Hybrid Sequencing Mode (Sequenced + Based Blocks)

> **To create this issue**: Go to https://github.com/evstack/ev-node/issues/new and paste the content below (starting from "## Motivation")

---

## Motivation

ev-node currently supports two mutually exclusive sequencing modes:

| Mode | Latency | Finality | Censorship Resistance | Permissionless |
|------|---------|----------|----------------------|----------------|
| **Single Sequencer** | ~100ms | Soft (until DA) | Forced inclusion (grace period) | No |
| **Based Sequencer** | ~6s (DA block) | Hard (immediate) | Full | Yes |

**The problem**: Users must choose between low latency OR strong guarantees. There's no middle ground.

**The hybrid model** combines both: fast sequenced blocks during normal operation with periodic "finality checkpoints" that permit permissionless based block construction.

## What This Unlocks

1. **Finality checkpoints**: Clear boundaries where soft confirmations become hard finality
2. **Graceful degradation**: If sequencer goes offline, chain continues via based blocks
3. **Sequencer accountability**: Based blocks can include transactions the sequencer censored
4. **User choice**: Users can opt for fast (sequenced) or secure (based) inclusion
5. **Cross-L2 foundation**: State roots at finality boundaries enable trust-minimized verification across rollups

## Background

This design is inspired by [Vitalik's proposal on combining preconfirmations with based rollups](https://ethresear.ch/t/combining-preconfirmations-with-based-rollups-for-synchronous-composability/23863), adapted for Celestia's architecture.

**Key simplification**: Celestia has single-slot finality with no reorgs. This eliminates the complex "L2 reverts when L1 reverts" handling required on Ethereum, making the implementation significantly simpler.

## Trade-offs

| Aspect | Gain | Cost |
|--------|------|------|
| **Flexibility** | Users choose latency vs finality | More complex mental model |
| **Liveness** | Chain survives sequencer downtime | Based blocks have higher latency |
| **Permissionlessness** | Based blocks are permissionless | Sequenced windows still require sequencer |
| **Complexity** | Better guarantees | ~1500 lines of new code, new block type |
| **Backwards compatibility** | Old nodes can sync | Need header versioning |

**What we explicitly don't get** (compared to Ethereum version):
- Synchronous L1-L2 composability (Celestia is DA-only, no shared execution)
- MEV redistribution via L1 proposers

---

## Design: Phase 1 — Slot-Ending Blocks (MVP)

**Goal**: Minimal changes to introduce finality checkpoints.

### 1.1 Block Type Field

Add a block type to headers:

```go
// types/header.go
type BlockType uint8

const (
    BlockTypeSequenced   BlockType = 0 // Default, backwards compatible
    BlockTypeSlotEnding  BlockType = 1 // Permits based block construction
    BlockTypeBased       BlockType = 2 // Permissionlessly constructed
)

type Header struct {
    // ... existing fields
    BlockType BlockType // New field
}
```

**Backwards compatibility**: `BlockType = 0` is the default, so existing blocks are valid sequenced blocks.

### 1.2 Slot-Ending Block Emission

The sequencer emits a slot-ending block at configurable intervals (default: every DA epoch):

```go
// pkg/sequencers/hybrid/sequencer.go
type HybridSequencer struct {
    single        *single.Sequencer
    slotInterval  uint64 // DA blocks between slot-ending blocks
    lastSlotEnd   uint64 // Last DA height with slot-ending block
}

func (h *HybridSequencer) GetNextBatch(ctx context.Context, req Request) (*Response, error) {
    currentDA := h.GetDAHeight()

    // Check if we should emit slot-ending block
    if currentDA >= h.lastSlotEnd + h.slotInterval {
        h.lastSlotEnd = currentDA
        return h.createSlotEndingBatch(ctx, req)
    }

    // Normal sequenced block
    return h.single.GetNextBatch(ctx, req)
}
```

### 1.3 Configuration

```yaml
# config.yaml
node:
  aggregator: true
  hybrid_sequencer: true  # New flag
  slot_interval: 1        # Slot-ending block every N DA epochs
```

### 1.4 Files Changed (Phase 1)

| File | Change |
|------|--------|
| `types/header.go` | Add `BlockType` field |
| `types/pb/types.proto` | Add block_type to Header proto |
| `pkg/config/config.go` | Add `HybridSequencer`, `SlotInterval` flags |
| `pkg/sequencers/hybrid/` | New package (~400 lines) |
| `node/node.go` | Wire up hybrid sequencer |

**Estimated effort**: ~2 weeks

---

## Design: Phase 2 — Based Block Construction

**Goal**: Allow permissionless based blocks after slot-ending blocks.

### 2.1 Slot-Ending Certificate

Slot-ending blocks include a certificate that authorizes based block construction:

```go
// types/certificate.go
type SlotEndingCertificate struct {
    SlotEndHeight    uint64    // Rollup height of slot-ending block
    DAEpochEnd       uint64    // DA height this slot covers
    NextBasedAllowed bool      // Whether based block can follow
    SequencerSig     []byte    // Sequencer signature over above
}
```

### 2.2 Based Block Validation

Syncer validates based blocks reference a valid certificate:

```go
// block/internal/syncing/syncer.go
func (s *Syncer) validateBasedBlock(header *types.SignedHeader) error {
    if header.BlockType != types.BlockTypeBased {
        return nil // Not a based block
    }

    // Previous block must be slot-ending
    prevHeader, err := s.store.GetHeader(header.Height() - 1)
    if err != nil {
        return err
    }

    if prevHeader.BlockType != types.BlockTypeSlotEnding {
        return ErrBasedBlockRequiresSlotEnding
    }

    // Verify based block data comes from DA forced inclusion namespace
    return s.verifyForcedInclusionSource(header)
}
```

### 2.3 Permissionless Based Block Production

Any node can produce a based block after seeing a slot-ending block:

```go
// pkg/sequencers/hybrid/based_producer.go
func (h *HybridSequencer) TryProduceBasedBlock(ctx context.Context) (*types.SignedHeader, error) {
    lastBlock := h.getLastBlock()

    if lastBlock.BlockType != types.BlockTypeSlotEnding {
        return nil, ErrNotAfterSlotEnding
    }

    // Fetch forced inclusion txs from DA
    forcedTxs, err := h.fiRetriever.RetrieveForcedIncludedTxs(ctx, lastBlock.DAHeight)
    if err != nil {
        return nil, err
    }

    if len(forcedTxs) == 0 {
        return nil, nil // No based block needed
    }

    return h.produceBasedBlock(ctx, forcedTxs)
}
```

### 2.4 Files Changed (Phase 2)

| File | Change |
|------|--------|
| `types/certificate.go` | New file (~100 lines) |
| `pkg/sequencers/hybrid/based_producer.go` | New file (~200 lines) |
| `block/internal/syncing/syncer.go` | Add based block validation (~100 lines) |
| `block/internal/executing/executor.go` | Handle based block production (~50 lines) |

**Estimated effort**: ~3 weeks

---

## Design: Phase 3 — Future Evolution

### 3.1 Dynamic Slot Intervals

Adjust slot interval based on chain activity:

```go
type AdaptiveSlotConfig struct {
    MinInterval     uint64 // Minimum DA epochs between slots
    MaxInterval     uint64 // Maximum DA epochs between slots
    TargetTxPerSlot uint64 // Target transactions per slot window
}
```

- High activity → shorter slots (more frequent finality checkpoints)
- Low activity → longer slots (reduced overhead)

### 3.2 Cross-L2 Asynchronous Composability

With multiple rollups posting state roots to Celestia at finality boundaries, we can enable trust-minimized cross-rollup verification.

**How it works**:

```
Celestia Block N:
┌─────────────────────────────────────────────────────────┐
│  Chain A: SlotEnding block → StateRoot 0xabc...         │
│  Chain B: SlotEnding block → StateRoot 0xdef...         │
│  Chain C: SlotEnding block → StateRoot 0x123...         │
└─────────────────────────────────────────────────────────┘

Chain A can now verify Chain B's state at a known finality boundary
```

**Use cases enabled**:

| Use Case | Description |
|----------|-------------|
| **Trust-minimized bridges** | Verify token balances on source chain before minting on destination |
| **Cross-rollup messaging** | Prove message was included in source chain's finalized state |
| **Conditional execution** | Execute on Chain A only if condition is met on Chain B's finalized state |
| **Unified liquidity views** | Aggregate balances across rollups with cryptographic proofs |

**Example: Cross-rollup token verification**

```go
// On Chain A, verify user's balance on Chain B
func VerifyCrossChainBalance(
    chainBStateRoot []byte,      // From Celestia at finality boundary
    userAddress     []byte,
    claimedBalance  uint64,
    merkleProof     [][]byte,
) error {
    // Verify the state root is from a finalized slot-ending block
    if !isValidFinalizedStateRoot(chainBStateRoot) {
        return ErrStateRootNotFinalized
    }

    // Verify Merkle proof of balance against state root
    balanceKey := accountBalanceKey(userAddress)
    if !verifyMerkleProof(chainBStateRoot, balanceKey, claimedBalance, merkleProof) {
        return ErrInvalidBalanceProof
    }

    return nil
}
```

**Key properties**:
- **Asynchronous**: Verification is against past finalized state (at least 1 DA block old)
- **Trust-minimized**: Only requires trusting Celestia DA, not the other rollup's sequencer
- **Finality-aligned**: Slot-ending blocks provide clear points where state roots are final

**Limitations** (not synchronous composability):
- Cannot atomically execute across chains in same transaction
- State proof is always slightly stale (previous finality boundary)
- Each chain must independently verify proofs

### 3.3 Based Block Builders

Allow specialized builders to construct optimized based blocks:

```go
type BasedBlockBuilder interface {
    BuildBasedBlock(ctx context.Context, forcedTxs [][]byte, prevHeader *Header) (*SignedHeader, *Data, error)
}
```

Builders can optimize transaction ordering within based blocks for better execution.

### 3.4 Observability

New metrics for hybrid mode:

```
hybrid_sequenced_blocks_total
hybrid_slot_ending_blocks_total
hybrid_based_blocks_total
hybrid_slot_interval_current
hybrid_forced_txs_included_via_based
hybrid_time_to_finality_seconds
```

---

## Implementation Checklist

### Phase 1: MVP
- [ ] Add `BlockType` to `types/header.go`
- [ ] Update protobuf definitions in `types/pb/`
- [ ] Add config flags in `pkg/config/config.go`
- [ ] Create `pkg/sequencers/hybrid/sequencer.go`
- [ ] Wire up in `node/node.go`
- [ ] Add unit tests
- [ ] Add integration test with slot-ending blocks
- [ ] Update documentation

### Phase 2: Based Blocks
- [ ] Create `types/certificate.go`
- [ ] Add based block validation to syncer
- [ ] Implement `based_producer.go`
- [ ] Add based block execution path
- [ ] Integration tests for based block production
- [ ] Test sequencer failure → based block fallback

### Phase 3: Future
- [ ] Adaptive slot intervals
- [ ] Cross-L2 state proof verification helpers
- [ ] Builder interface
- [ ] Metrics and dashboards

---

## Open Questions

1. **Slot interval default**: Should we default to 1 DA epoch (every ~6s) or allow longer windows?
2. **Based block proposer selection**: First valid submission wins, or some selection mechanism?
3. **Certificate expiry**: Should slot-ending certificates expire if no based block is produced?
4. **Forced inclusion interaction**: How do existing forced inclusion grace periods interact with slot boundaries?
5. **State root format**: Should we standardize state root format across rollups to enable easier cross-L2 verification?

---

## References

- [Combining preconfirmations with based rollups](https://ethresear.ch/t/combining-preconfirmations-with-based-rollups-for-synchronous-composability/23863)
- [ev-node sequencer interface](https://github.com/evstack/ev-node/blob/main/core/sequencer/sequencing.go)
- [Based sequencer implementation](https://github.com/evstack/ev-node/blob/main/pkg/sequencers/based/sequencer.go)

---

## Slack Context

Discussion: https://celestia-team.slack.com/archives/C08HMT9K6UA/p1768680079881889?thread_ts=1768679430.013729&cid=C08HMT9K6UA
