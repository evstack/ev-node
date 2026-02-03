# Header and Data Sync

## Abstract

The nodes in the P2P network sync headers and data using separate sync services that implement the [go-header][go-header] interface. Evolve uses a header/data separation architecture where headers and transaction data are synchronized independently through parallel services. Each sync service consists of several components as listed below.

| Component  | Description                                                                                                                                                    |
| ---------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| store      | a prefixed [datastore][datastore] where synced items are stored (`headerSync` prefix for headers, `dataSync` prefix for data)                                  |
| subscriber | a [libp2p][libp2p] node pubsub subscriber for the specific data type                                                                                           |
| P2P server | a server for handling requests between peers in the P2P network                                                                                                |
| exchange   | a client that enables sending in/out-bound requests from/to the P2P network                                                                                    |
| syncer     | a service for efficient synchronization. When a P2P node falls behind and wants to catch up to the latest network head via P2P network, it can use the syncer. |

## Details

Evolve implements two separate sync services:

### Header Sync Service

- Synchronizes `SignedHeader` structures containing block headers with signatures
- Used by all node types (sequencer, full, and light)
- Essential for maintaining the canonical view of the chain

### Data Sync Service

- Synchronizes `Data` structures containing transaction data
- Used only by full nodes and sequencers
- Light nodes do not run this service as they only need headers

Both services:

- Utilize the generic `SyncService[H header.Header[H]]` implementation
- Inherit the `ConnectionGater` from the node's P2P client for peer management
- Use `NodeConfig.BlockTime` to determine outdated items during sync
- Operate independently on separate P2P topics and datastores

### Consumption of Sync Services

#### Header Sync

- Sequencer nodes publish signed headers to the P2P network after block creation
- Full and light nodes receive and store headers for chain validation
- Headers contain commitments (DataHash) that link to the corresponding data

#### Data Sync

- Sequencer nodes publish transaction data separately from headers
- Only full nodes receive and store data (light nodes skip this)
- Data is linked to headers through the DataHash commitment

#### Parallel Broadcasting

The Executor component (in aggregator nodes) broadcasts headers and data in parallel when publishing blocks:

- Headers are sent through `headerBroadcaster`
- Data is sent through `dataBroadcaster`
- This enables efficient network propagation of both components

## Assumptions

- Separate datastores are created with different prefixes:
  - Headers: `headerSync` prefix on the main datastore
  - Data: `dataSync` prefix on the main datastore
- Network IDs are suffixed to distinguish services:
  - Header sync: `{network}-headerSync`
  - Data sync: `{network}-dataSync`
- Chain IDs for pubsub topics are also separated:
  - Headers: `{chainID}-headerSync` creates topic like `/gm-headerSync/header-sub/v0.0.1`
  - Data: `{chainID}-dataSync` creates topic like `/gm-dataSync/header-sub/v0.0.1`
- Both stores must contain at least one item before the syncer starts:
  - On first boot, the services fetch the configured genesis height from peers
  - On restart, each store reuses its latest item to derive the initial height requested from peers
- Sync services work only when connected to P2P network via `P2PConfig.Seeds`
- Node context is passed to all components for graceful shutdown
- Headers and data are linked through DataHash but synced independently

## Implementation

The sync service implementation can be found in [pkg/sync/sync_service.go][sync-service]. The generic `SyncService[H header.Header[H]]` is instantiated as:

- `HeaderSyncService` for syncing `*types.SignedHeader`
- `DataSyncService` for syncing `*types.Data`

Full nodes create and start both services, while light nodes only start the header sync service. The services are created in [full][fullnode] and [light][lightnode] node implementations.

The block components integrate with both services through:

- The Syncer component's P2PHandler retrieves headers and data from P2P
- The Executor component publishes headers and data through broadcast channels
- Separate stores and channels manage header and data synchronization

## DA Height Hints

DA Height Hints (DAHint) provide an optimization for P2P synchronization by indicating which DA layer height contains a block's header or data. This allows syncing nodes to fetch missing DA data directly instead of performing sequential DA scanning.

### Naming Considerations

The naming convention follows this pattern:

| Name              | Usage                                                      |
| ----------------- | ---------------------------------------------------------- |
| `DAHeightHint`    | Internal struct field storing the hint value               |
| `DAHint()`        | Getter method returning the DA height hint                 |
| `SetDAHint()`     | Setter method for the DA height hint                       |
| `P2PSignedHeader` | Wrapper around `SignedHeader` that includes `DAHeightHint` |
| `P2PData`         | Wrapper around `Data` that includes `DAHeightHint`         |

The term "hint" is used deliberately because:

1. **It's advisory, not authoritative**: The hint suggests where to find data on the DA layer, but the authoritative source is always the DA layer itself
2. **It may be absent**: Hints are only populated during certain sync scenarios (see below)
3. **It optimizes but doesn't replace**: Nodes can still function without hints by scanning the DA layer sequentially

### When DAHints Are Populated

DAHints are **only populated when a node catches up from P2P** and is not yet synced to the head. When a node is already synced to the head:

- The executor broadcasts headers/data immediately after block creation
- At this point, DA submission has not occurred yet (it happens later in the flow)
- Therefore, the broadcasted P2P messages do not contain DA hints

This means:

- **Syncing nodes** (catching up): Receive headers/data with DA hints populated
- **Synced nodes** (at head): Receive headers/data without DA hints

The DA hints are set by the DA submitter after successful inclusion on the DA layer and stored for later P2P propagation to syncing peers.

### Implementation Details

The P2P wrapper types (`P2PSignedHeader` and `P2PData`) extend the base types with an optional `DAHeightHint` field:

- Uses protobuf optional fields (`optional uint64 da_height_hint`) for backward compatibility
- Old nodes can still unmarshal new messages (the hint field is simply ignored)
- New nodes can unmarshal old messages (the hint field defaults to zero/absent)

The hint flow:

1. **Set by the DA Submitter** when headers/data are successfully included on the DA layer
2. **Stored in the P2P store** alongside the header/data
3. **Propagated via P2P** when syncing nodes request blocks
4. **Queued as priority** by the Syncer's DA retriever when received via P2P
5. **Fetched before sequential heights** - priority heights take precedence over normal DA scanning

### Priority Queue Mechanism

When a P2P event arrives with a DA height hint, the hint is queued as a priority height in the DA retriever. The `fetchDAUntilCaughtUp` loop checks for priority heights first:

1. If priority heights are queued, pop and fetch the lowest one first
2. If no priority heights, continue sequential DA fetching (form last known da height)
3. Priority heights are sorted ascending to process lower heights first
4. Already-processed priority heights are tracked to avoid duplicate fetches

This ensures that when syncing from P2P, the node can immediately fetch the DA data for blocks it receives, rather than waiting for sequential scanning to reach that height.

## References

[1] [Header Sync][sync-service]

[2] [Full Node][fullnode]

[3] [Light Node][lightnode]

[4] [go-header][go-header]

[sync-service]: https://github.com/evstack/ev-node/blob/main/pkg/sync/sync_service.go
[fullnode]: https://github.com/evstack/ev-node/blob/main/node/full.go
[lightnode]: https://github.com/evstack/ev-node/blob/main/node/light.go
[go-header]: https://github.com/celestiaorg/go-header
[libp2p]: https://github.com/libp2p/go-libp2p
[datastore]: https://github.com/ipfs/go-datastore
