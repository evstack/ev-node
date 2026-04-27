# Fibre 4-validator + bridge docker showcase

A docker-compose stack that brings up four celestia-app validators
(each running a Fibre server), a celestia-node bridge, and a one-shot
init container that registers Fibre Storage Provider hosts and funds an
escrow account. A Go test driver (`docker_test.go`) connects from the
host and exercises the `celestia-node-fiber` adapter end-to-end against
the real 2/3-quorum network.

## Why

The in-process `testing/showcase_test.go` runs against a single
validator inside the test process. That proves the adapter wires
correctly, but it doesn't exercise:

- real consensus 2/3 quorum collection (single validator trivially
  satisfies it),
- inter-validator P2P,
- multiple Fibre servers contributing partial signatures,
- the dns:/// host registry resolution path,
- the bridge syncing real headers off a network it doesn't itself drive.

This stack does.

## Architecture

```
                 +---------- bootstrap (one-shot) ----------+
                 |  init-genesis.sh: 4-val genesis + keys   |
                 +-------+----------------------------------+
                         | shared volume
            +------------+------------+------------+
            v            v            v            v
          val0         val1         val2         val3
        (appd +      (appd +      (appd +      (appd +
         fibre)       fibre)       fibre)       fibre)
            ^
            | gRPC :9090, RPC :26657
            |
           bridge (celestia-node)
            ^
            | JSON-RPC/WebSocket :26658
            |
   +--------+--------+
   |    Go test      |
   | (docker_test.go) |
   +-----------------+
```

## Run

```bash
cd tools/celestia-node-fiber/testing/docker

# First boot: builds two images (~5–10 min on a cold cache).
docker compose up -d --build

# Watch the bootstrap + registration progress:
docker compose logs -f bootstrap register

# Once `register` exits 0 and writes /shared/setup.done, the bridge
# connects and the stack is ready.

# From the parent dir, run the Go-side driver:
cd ../..
go test -tags 'fibre fibre_docker' -count=1 -timeout 5m ./testing/docker/...

# Tear down (preserves volumes — add -v to wipe shared genesis state):
docker compose -f testing/docker/compose.yaml down
```

Override endpoints from the host with env vars if your ports collide:

```
FIBRE_BRIDGE_ADDR=ws://127.0.0.1:36658 \
FIBRE_CONSENSUS_ADDR=127.0.0.1:19090 \
go test -tags 'fibre fibre_docker' ...
```

## Build args

Both Dockerfiles accept refs:

| arg | Dockerfile | default | what it does |
|---|---|---|---|
| `CELESTIA_APP_REPO` | `Dockerfile.app` | celestia-app upstream | clone source |
| `CELESTIA_APP_REF` | `Dockerfile.app` | `main` | git ref to build with `-tags fibre,ledger` |
| `CELESTIA_NODE_REPO` | `Dockerfile.bridge` | celestia-node upstream | clone source |
| `CELESTIA_NODE_REF` | `Dockerfile.bridge` | `feature/fibre` | git ref to build with `-tags fibre` |

Example pinning to a specific commit:

```
docker compose build --build-arg CELESTIA_NODE_REF=194cc74c ...
```

## Known TODOs

The scaffold has been validated end-to-end on Apple Silicon
(Docker Desktop 4.70 / linux/arm64). A few rough edges remain that
are worth tightening for CI:

1. **`config.toml` / `app.toml` overrides** in `start-validator.sh`
   use `sed` against expected default lines. If the celestia-app
   defaults change verb/spacing, the substitutions silently no-op.
   Consider a `dasel`/`tomlq` rewrite if it bites.
2. **No healthchecks** on validators. `register` waits on
   `service_started`, which is only "container booted", not "RPC
   responding". The script polls `celestia-appd status` which
   handles that, but a proper healthcheck would let `bridge` start
   sooner without polling itself.
3. **No CI integration**. Adding `make docker-test` that wraps
   `docker compose up -d --wait`, runs the test, then tears down,
   is a sensible follow-up.
4. **Build cache** — every `docker compose up --build` re-clones
   celestia-app + celestia-node. To iterate faster, set up a
   docker volume cache for `/go/pkg/mod` and `/root/.cache/go-build`,
   or build the images once and re-use.
