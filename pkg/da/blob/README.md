# blob package

This package is a **trimmed copy** of code from `celestia-node` to stay JSON-compatible with the blob RPC without importing the full Cosmos/Celestia dependency set.

## Upstream source

- `blob.go` comes from `celestia-node/blob/blob.go` @ tag `v0.28.4` (release v0.28.4), with unused pieces removed (blob v1, proof helpers, share length calc, appconsts dependency, etc.).
- `submit_options.go` mirrors the exported JSON fields of `celestia-node/state/tx_config.go` @ the same tag, leaving out functional options, defaults, and Cosmos keyring helpers.

## Why copy instead of import?

- Avoids pulling Cosmos SDK / celestia-app dependencies into ev-node for the small surface we need (blob JSON and commitment for v0).
- Keeps binary size and module graph smaller while remaining wire-compatible with celestia-node's blob service.

## Keeping it in sync

- When celestia-node changes blob JSON or tx config fields, update this package manually:
  1. `diff -u pkg/da/blob/blob.go ../Celestia/celestia-node/blob/blob.go`
  2. `diff -u pkg/da/blob/submit_options.go ../Celestia/celestia-node/state/tx_config.go`
  3. Port only the fields/logic required for our RPC surface.
- Consider adding a CI check that diffs these files against upstream to detect drift.
