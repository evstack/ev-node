# fiber-bench

Single-sequencer ev-node throughput bench against a remote Fibre network.

## What it is

A self-contained binary that spins up an ev-node aggregator wired to a
remote Fibre network (no bridge node, no P2P, no syncer, no
state-machine cost) and pumps transactions into its mempool as fast as
the configured backpressure allows.

The intent is a fail-fast baseline so we can isolate ev-node's
batching + DA-submit pipeline from everything else when chasing the
1k tps regression.

### What's stripped out (and why)

| Stripped       | Why                                                        |
|----------------|------------------------------------------------------------|
| Bridge node    | Upload only needs consensus gRPC + FSPs.                   |
| Syncer         | Aggregator-only single-node setup.                         |
| P2P outbound   | ev-node already disables it when `da.fiber.enabled=true`.  |
| Forced incl.   | Solo sequencer.                                            |
| Real state machine | Constant state root — measure ev-node, not state cost. |
| HTTP tx ingress | Direct `InjectTx`. Removes HTTP from the hot path.        |

## Layout

```
tools/celestia-node-fiber/cmd/fiber-bench/
  main.go        cobra root
  keys.go        cosmos keyring management (test backend)
  escrow.go      Fibre escrow deposit/query
  run.go         the bench
  executor.go    in-mem core.Executor with constant state root
  loader.go      internal tx pump
  stats.go       periodic stats line + final baseline summary
  fibre.go       bridge-bypass cnfiber.Adapter constructor
  run-bench.sh   convenience wrapper
```

## Quick start

```sh
cd tools/celestia-node-fiber

# 1. Build — the `fibre` build tag is REQUIRED so celestia-app's
# x/fibre messages (MsgPayForFibre, MsgDepositToEscrow) are registered
# in the codec. Without it the async PFF settlement fails with
# "unable to resolve type URL /celestia.fibre.v1.MsgPayForFibre".
go build -tags fibre -o bin/fiber-bench ./cmd/fiber-bench/

# 2. Create the bench key (cosmos keyring, test backend = unencrypted on disk)
./bin/fiber-bench keys add bench
#   prints: address: celestia1...
#           mnemonic: ...

# 3. Top up the printed address with utia on the chain (out of band).

# 4. Deposit into the Fibre escrow
./bin/fiber-bench escrow deposit \
    --consensus-grpc 139.59.229.101:9091 \
    --key-name bench \
    --amount 50000000        # 50 TIA

# 5. Sanity check
./bin/fiber-bench escrow query \
    --consensus-grpc 139.59.229.101:9091 \
    --key-name bench

# 6. Run the bench
./bin/fiber-bench run \
    --consensus-grpc 139.59.229.101:9091 \
    --chain-id <chain-id-the-server-actually-reports> \
    --key-name bench \
    --duration 2m \
    --workers 32 \
    --tx-size 200 \
    --block-time 1s \
    --batching-strategy immediate
```

Or use the convenience wrapper:

```sh
CONSENSUS_GRPC=139.59.229.101:9091 \
CHAIN_ID=talis-slab-diag \
    ./cmd/fiber-bench/run-bench.sh 2m 32
```

## What the run prints

A header, then one line per `--stats-interval` (default 1s):

```
elapsed   injected     inj/s     exec/s    blocks/s  committed_h  txs/blk blob_bytes pending   drops
------------------------------------------------------------------------------------------------------
1s        1452609      1452212   0         0.00      0            0.0     0         0         293116
2s        1544094      91444     0         0.00      0            0.0     0         0         1007270
```

Columns:

- `injected` — total txs the load generator has called `InjectTx` for
- `inj/s` — injection rate over the last interval
- `exec/s` — txs included in produced blocks (rate)
- `blocks/s` — block production rate
- `committed_h` — last block height confirmed by DA (0 until first
  Upload settles)
- `txs/blk` — running average over all blocks
- `blob_bytes` — last block's data size in bytes
- `pending` — `evnode_da_submitter_pending_blobs` gauge
- `drops` — txs the load generator could not enqueue because the
  in-mem mempool channel was full (this is the backpressure signal)

At the end:

```
============================================================
                BASELINE SUMMARY
============================================================
Duration:               2m0s
Injected:               XXX (avg N tx/s, peak N tx/s)
Dropped (mempool full): XXX
Mempool high-water:     XXX
Blocks produced:        XXX (committed_h=YYY)
Txs executed:           XXX (avg N tx/s, peak N tx/s, T tx/blk)
============================================================
```

## Knobs worth flipping while debugging

| Flag                    | Default      | Why                                               |
|-------------------------|--------------|---------------------------------------------------|
| `--block-time`          | `1s`         | Drop to e.g. `100ms` to expose per-block overhead |
| `--batching-strategy`   | `immediate`  | Try `time` / `size` / `adaptive`                  |
| `--reaper-interval`     | `100ms`      | How often the mempool drain runs                  |
| `--max-pending`         | `0`          | Cap pending DA blobs to test backpressure         |
| `--workers`             | `32`         | Tx-injection concurrency                          |
| `--tx-size`             | `200`        | Bytes per tx (matches user-reported regression)   |
| `--mempool-size`        | `1_000_000`  | Bench's bounded backpressure boundary             |
| `--keep-home`           | `false`      | Resume from prior state (defaults to wipe)        |
| `--log-level`           | `info`       | `debug` to see ev-node block production logs      |

## ev-node Prometheus

When `--prometheus=true` (default), ev-node exposes metrics at
`http://127.0.0.1:26660/metrics`. The bench scrapes a handful of them
for its stats line, but you can hit the endpoint directly for the full
picture: `evnode_block_production_duration_seconds`,
`evnode_da_submitter_failures_total`, etc.

## Operational notes

- **Test-backend keyring**: keys live unencrypted on disk under
  `~/.fiber-bench/keyring`. Fine for a bench account funded with a
  small amount of utia. Don't use for anything else.
- **The bench wipes its ev-node home (`~/.fiber-bench/node`) on every
  run** unless `--keep-home` is passed. Block-signing key, store, and
  any in-flight pending blocks all reset. The cosmos keyring is
  separate and is preserved.
- **Bridge bypass**: the bench builds the `cnfiber.Adapter` via
  `cnfiber.FromModules` with a stub Blob module that errors on every
  call. The aggregator-only setup never invokes Listen/Subscribe, so
  this is safe; if the assumption breaks, you'll see a clear
  `fiber-bench: blob module not supported` error rather than a nil
  panic.
- **Chain ID** is what the consensus node reports; the bench logs it
  on startup. Pass the same value via `--chain-id` for config
  validation; mismatch is logged but tx submission proceeds against
  the chain's actual ID.
