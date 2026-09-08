window.BENCHMARK_DATA = {
  "lastUpdate": 1788854899412,
  "repoUrl": "https://github.com/evstack/ev-node",
  "entries": {
    "EVM Contract Roundtrip": [
      {
        "commit": {
          "author": {
            "email": "marko@baricevic.me",
            "name": "Marko",
            "username": "tac0turtle"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "cca63ca2a26cb19d24e9d81b113d3555fc6fbe3d",
          "message": "fix(node): fail closed during sequencer recovery (#3443)\n\n* fix(node): fail closed during sequencer recovery\n\n* fix(sync): retry P2P init throughout catchup recovery\n\nDo not abandon P2P initialization after the 30s Start timeout when\ncatchup recovery requires continuity. Keep retrying in the background\nso P2PInitialized can still flip during waitForCatchup, and include\nreadiness flags in the timeout error.",
          "timestamp": "2026-09-01T15:44:15+02:00",
          "tree_id": "a89aff0a7b0aa7649af35452f73427995b03f92a",
          "url": "https://github.com/evstack/ev-node/commit/cca63ca2a26cb19d24e9d81b113d3555fc6fbe3d"
        },
        "date": 1788270481708,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 895054383,
            "unit": "ns/op\t 4249828 B/op\t   35939 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 895054383,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 4249828,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 35939,
            "unit": "allocs/op",
            "extra": "2 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "luangucun@outlook.com",
            "name": "luangucun",
            "username": "luangucun"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": false,
          "id": "8b13733cec7208a268192db65661781ae04eff1b",
          "message": "fix(store): remove canceled height waiters (#3445)\n\nSigned-off-by: luangucun <luangucun@outlook.com>",
          "timestamp": "2026-09-08T07:43:53Z",
          "tree_id": "9eb61585a73abae4924f9913ed10b0ef102ab6ae",
          "url": "https://github.com/evstack/ev-node/commit/8b13733cec7208a268192db65661781ae04eff1b"
        },
        "date": 1788854758640,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 915541246,
            "unit": "ns/op\t 4267464 B/op\t   36827 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 915541246,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 4267464,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 36827,
            "unit": "allocs/op",
            "extra": "2 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "49699333+dependabot[bot]@users.noreply.github.com",
            "name": "dependabot[bot]",
            "username": "dependabot[bot]"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": false,
          "id": "27b6a6ca2ffeea40824faa2ce8abb12512aa660e",
          "message": "build(deps): Bump the all-go group across 4 directories with 9 updates (#3447)\n\n* build(deps): Bump the all-go group across 4 directories with 9 updates\n\nBumps the all-go group with 7 updates in the / directory:\n\n| Package | From | To |\n| --- | --- | --- |\n| [github.com/aws/aws-sdk-go-v2](https://github.com/aws/aws-sdk-go-v2) | `1.45.1` | `1.46.0` |\n| [github.com/aws/aws-sdk-go-v2/config](https://github.com/aws/aws-sdk-go-v2) | `1.33.1` | `1.33.3` |\n| [github.com/aws/aws-sdk-go-v2/service/kms](https://github.com/aws/aws-sdk-go-v2) | `1.57.1` | `1.59.0` |\n| [github.com/celestiaorg/go-header](https://github.com/celestiaorg/go-header) | `0.8.6` | `0.8.7` |\n| [github.com/celestiaorg/nmt](https://github.com/celestiaorg/nmt) | `0.24.3` | `0.24.4` |\n| [golang.org/x/crypto](https://github.com/golang/crypto) | `0.55.0` | `0.56.0` |\n| [google.golang.org/api](https://github.com/googleapis/google-api-go-client) | `0.294.0` | `0.297.0` |\n\nBumps the all-go group with 1 update in the /apps/loadgen directory: [github.com/prometheus/client_model](https://github.com/prometheus/client_model).\nBumps the all-go group with 1 update in the /test/docker-e2e directory: [github.com/moby/moby/client](https://github.com/moby/moby).\nBumps the all-go group with 1 update in the /test/e2e directory: [github.com/prometheus/client_model](https://github.com/prometheus/client_model).\n\n\nUpdates `github.com/aws/aws-sdk-go-v2` from 1.45.1 to 1.46.0\n- [Release notes](https://github.com/aws/aws-sdk-go-v2/releases)\n- [Commits](https://github.com/aws/aws-sdk-go-v2/compare/v1.45.1...v1.46.0)\n\nUpdates `github.com/aws/aws-sdk-go-v2/config` from 1.33.1 to 1.33.3\n- [Release notes](https://github.com/aws/aws-sdk-go-v2/releases)\n- [Commits](https://github.com/aws/aws-sdk-go-v2/compare/config/v1.33.1...config/v1.33.3)\n\nUpdates `github.com/aws/aws-sdk-go-v2/service/kms` from 1.57.1 to 1.59.0\n- [Release notes](https://github.com/aws/aws-sdk-go-v2/releases)\n- [Commits](https://github.com/aws/aws-sdk-go-v2/compare/service/s3/v1.57.1...service/s3/v1.59.0)\n\nUpdates `github.com/celestiaorg/go-header` from 0.8.6 to 0.8.7\n- [Release notes](https://github.com/celestiaorg/go-header/releases)\n- [Commits](https://github.com/celestiaorg/go-header/compare/v0.8.6...v0.8.7)\n\nUpdates `github.com/celestiaorg/nmt` from 0.24.3 to 0.24.4\n- [Release notes](https://github.com/celestiaorg/nmt/releases)\n- [Commits](https://github.com/celestiaorg/nmt/compare/v0.24.3...v0.24.4)\n\nUpdates `golang.org/x/crypto` from 0.55.0 to 0.56.0\n- [Commits](https://github.com/golang/crypto/compare/v0.55.0...v0.56.0)\n\nUpdates `google.golang.org/api` from 0.294.0 to 0.297.0\n- [Release notes](https://github.com/googleapis/google-api-go-client/releases)\n- [Changelog](https://github.com/googleapis/google-api-go-client/blob/main/CHANGES.md)\n- [Commits](https://github.com/googleapis/google-api-go-client/compare/v0.294.0...v0.297.0)\n\nUpdates `github.com/prometheus/client_model` from 0.6.2 to 0.6.3\n- [Release notes](https://github.com/prometheus/client_model/releases)\n- [Commits](https://github.com/prometheus/client_model/compare/v0.6.2...v0.6.3)\n\nUpdates `github.com/moby/moby/client` from 0.5.1 to 0.6.0\n- [Release notes](https://github.com/moby/moby/releases)\n- [Changelog](https://github.com/moby/moby/blob/v0.6.0/CHANGELOG.md)\n- [Commits](https://github.com/moby/moby/compare/v0.5.1...v0.6.0)\n\nUpdates `github.com/prometheus/client_model` from 0.6.2 to 0.6.3\n- [Release notes](https://github.com/prometheus/client_model/releases)\n- [Commits](https://github.com/prometheus/client_model/compare/v0.6.2...v0.6.3)\n\n---\nupdated-dependencies:\n- dependency-name: github.com/aws/aws-sdk-go-v2\n  dependency-version: 1.46.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/aws/aws-sdk-go-v2/config\n  dependency-version: 1.33.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/aws/aws-sdk-go-v2/service/kms\n  dependency-version: 1.59.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/celestiaorg/go-header\n  dependency-version: 0.8.7\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/celestiaorg/nmt\n  dependency-version: 0.24.4\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: golang.org/x/crypto\n  dependency-version: 0.56.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: google.golang.org/api\n  dependency-version: 0.297.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/prometheus/client_model\n  dependency-version: 0.6.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/moby/moby/client\n  dependency-version: 0.6.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/prometheus/client_model\n  dependency-version: 0.6.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\n\n* chore: run just deps after Dependabot update\n\n---------\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>\nCo-authored-by: github-actions[bot] <41898282+github-actions[bot]@users.noreply.github.com>",
          "timestamp": "2026-09-08T07:45:57Z",
          "tree_id": "bf0206de474f22c41c14b3729f6abf0a5ae30f7d",
          "url": "https://github.com/evstack/ev-node/commit/27b6a6ca2ffeea40824faa2ce8abb12512aa660e"
        },
        "date": 1788854891286,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 908337761,
            "unit": "ns/op\t 4203616 B/op\t   36368 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 908337761,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 4203616,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 36368,
            "unit": "allocs/op",
            "extra": "2 times\n4 procs"
          }
        ]
      }
    ],
    "Block Executor Benchmark": [
      {
        "commit": {
          "author": {
            "email": "marko@baricevic.me",
            "name": "Marko",
            "username": "tac0turtle"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "cca63ca2a26cb19d24e9d81b113d3555fc6fbe3d",
          "message": "fix(node): fail closed during sequencer recovery (#3443)\n\n* fix(node): fail closed during sequencer recovery\n\n* fix(sync): retry P2P init throughout catchup recovery\n\nDo not abandon P2P initialization after the 30s Start timeout when\ncatchup recovery requires continuity. Keep retrying in the background\nso P2PInitialized can still flip during waitForCatchup, and include\nreadiness flags in the timeout error.",
          "timestamp": "2026-09-01T15:44:15+02:00",
          "tree_id": "a89aff0a7b0aa7649af35452f73427995b03f92a",
          "url": "https://github.com/evstack/ev-node/commit/cca63ca2a26cb19d24e9d81b113d3555fc6fbe3d"
        },
        "date": 1788270488677,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 35478,
            "unit": "ns/op\t    4841 B/op\t      51 allocs/op",
            "extra": "33741 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 35478,
            "unit": "ns/op",
            "extra": "33741 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4841,
            "unit": "B/op",
            "extra": "33741 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 51,
            "unit": "allocs/op",
            "extra": "33741 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 36101,
            "unit": "ns/op\t    5045 B/op\t      55 allocs/op",
            "extra": "33216 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 36101,
            "unit": "ns/op",
            "extra": "33216 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 5045,
            "unit": "B/op",
            "extra": "33216 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "33216 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 42687,
            "unit": "ns/op\t   10347 B/op\t      55 allocs/op",
            "extra": "28250 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 42687,
            "unit": "ns/op",
            "extra": "28250 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10347,
            "unit": "B/op",
            "extra": "28250 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "28250 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "luangucun@outlook.com",
            "name": "luangucun",
            "username": "luangucun"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": false,
          "id": "8b13733cec7208a268192db65661781ae04eff1b",
          "message": "fix(store): remove canceled height waiters (#3445)\n\nSigned-off-by: luangucun <luangucun@outlook.com>",
          "timestamp": "2026-09-08T07:43:53Z",
          "tree_id": "9eb61585a73abae4924f9913ed10b0ef102ab6ae",
          "url": "https://github.com/evstack/ev-node/commit/8b13733cec7208a268192db65661781ae04eff1b"
        },
        "date": 1788854764819,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 33476,
            "unit": "ns/op\t    4794 B/op\t      51 allocs/op",
            "extra": "36056 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 33476,
            "unit": "ns/op",
            "extra": "36056 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4794,
            "unit": "B/op",
            "extra": "36056 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 51,
            "unit": "allocs/op",
            "extra": "36056 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 33539,
            "unit": "ns/op\t    4991 B/op\t      55 allocs/op",
            "extra": "35811 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 33539,
            "unit": "ns/op",
            "extra": "35811 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 4991,
            "unit": "B/op",
            "extra": "35811 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "35811 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 40787,
            "unit": "ns/op\t   10305 B/op\t      55 allocs/op",
            "extra": "29676 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 40787,
            "unit": "ns/op",
            "extra": "29676 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10305,
            "unit": "B/op",
            "extra": "29676 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "29676 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "49699333+dependabot[bot]@users.noreply.github.com",
            "name": "dependabot[bot]",
            "username": "dependabot[bot]"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": false,
          "id": "27b6a6ca2ffeea40824faa2ce8abb12512aa660e",
          "message": "build(deps): Bump the all-go group across 4 directories with 9 updates (#3447)\n\n* build(deps): Bump the all-go group across 4 directories with 9 updates\n\nBumps the all-go group with 7 updates in the / directory:\n\n| Package | From | To |\n| --- | --- | --- |\n| [github.com/aws/aws-sdk-go-v2](https://github.com/aws/aws-sdk-go-v2) | `1.45.1` | `1.46.0` |\n| [github.com/aws/aws-sdk-go-v2/config](https://github.com/aws/aws-sdk-go-v2) | `1.33.1` | `1.33.3` |\n| [github.com/aws/aws-sdk-go-v2/service/kms](https://github.com/aws/aws-sdk-go-v2) | `1.57.1` | `1.59.0` |\n| [github.com/celestiaorg/go-header](https://github.com/celestiaorg/go-header) | `0.8.6` | `0.8.7` |\n| [github.com/celestiaorg/nmt](https://github.com/celestiaorg/nmt) | `0.24.3` | `0.24.4` |\n| [golang.org/x/crypto](https://github.com/golang/crypto) | `0.55.0` | `0.56.0` |\n| [google.golang.org/api](https://github.com/googleapis/google-api-go-client) | `0.294.0` | `0.297.0` |\n\nBumps the all-go group with 1 update in the /apps/loadgen directory: [github.com/prometheus/client_model](https://github.com/prometheus/client_model).\nBumps the all-go group with 1 update in the /test/docker-e2e directory: [github.com/moby/moby/client](https://github.com/moby/moby).\nBumps the all-go group with 1 update in the /test/e2e directory: [github.com/prometheus/client_model](https://github.com/prometheus/client_model).\n\n\nUpdates `github.com/aws/aws-sdk-go-v2` from 1.45.1 to 1.46.0\n- [Release notes](https://github.com/aws/aws-sdk-go-v2/releases)\n- [Commits](https://github.com/aws/aws-sdk-go-v2/compare/v1.45.1...v1.46.0)\n\nUpdates `github.com/aws/aws-sdk-go-v2/config` from 1.33.1 to 1.33.3\n- [Release notes](https://github.com/aws/aws-sdk-go-v2/releases)\n- [Commits](https://github.com/aws/aws-sdk-go-v2/compare/config/v1.33.1...config/v1.33.3)\n\nUpdates `github.com/aws/aws-sdk-go-v2/service/kms` from 1.57.1 to 1.59.0\n- [Release notes](https://github.com/aws/aws-sdk-go-v2/releases)\n- [Commits](https://github.com/aws/aws-sdk-go-v2/compare/service/s3/v1.57.1...service/s3/v1.59.0)\n\nUpdates `github.com/celestiaorg/go-header` from 0.8.6 to 0.8.7\n- [Release notes](https://github.com/celestiaorg/go-header/releases)\n- [Commits](https://github.com/celestiaorg/go-header/compare/v0.8.6...v0.8.7)\n\nUpdates `github.com/celestiaorg/nmt` from 0.24.3 to 0.24.4\n- [Release notes](https://github.com/celestiaorg/nmt/releases)\n- [Commits](https://github.com/celestiaorg/nmt/compare/v0.24.3...v0.24.4)\n\nUpdates `golang.org/x/crypto` from 0.55.0 to 0.56.0\n- [Commits](https://github.com/golang/crypto/compare/v0.55.0...v0.56.0)\n\nUpdates `google.golang.org/api` from 0.294.0 to 0.297.0\n- [Release notes](https://github.com/googleapis/google-api-go-client/releases)\n- [Changelog](https://github.com/googleapis/google-api-go-client/blob/main/CHANGES.md)\n- [Commits](https://github.com/googleapis/google-api-go-client/compare/v0.294.0...v0.297.0)\n\nUpdates `github.com/prometheus/client_model` from 0.6.2 to 0.6.3\n- [Release notes](https://github.com/prometheus/client_model/releases)\n- [Commits](https://github.com/prometheus/client_model/compare/v0.6.2...v0.6.3)\n\nUpdates `github.com/moby/moby/client` from 0.5.1 to 0.6.0\n- [Release notes](https://github.com/moby/moby/releases)\n- [Changelog](https://github.com/moby/moby/blob/v0.6.0/CHANGELOG.md)\n- [Commits](https://github.com/moby/moby/compare/v0.5.1...v0.6.0)\n\nUpdates `github.com/prometheus/client_model` from 0.6.2 to 0.6.3\n- [Release notes](https://github.com/prometheus/client_model/releases)\n- [Commits](https://github.com/prometheus/client_model/compare/v0.6.2...v0.6.3)\n\n---\nupdated-dependencies:\n- dependency-name: github.com/aws/aws-sdk-go-v2\n  dependency-version: 1.46.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/aws/aws-sdk-go-v2/config\n  dependency-version: 1.33.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/aws/aws-sdk-go-v2/service/kms\n  dependency-version: 1.59.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/celestiaorg/go-header\n  dependency-version: 0.8.7\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/celestiaorg/nmt\n  dependency-version: 0.24.4\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: golang.org/x/crypto\n  dependency-version: 0.56.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: google.golang.org/api\n  dependency-version: 0.297.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/prometheus/client_model\n  dependency-version: 0.6.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/moby/moby/client\n  dependency-version: 0.6.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/prometheus/client_model\n  dependency-version: 0.6.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\n\n* chore: run just deps after Dependabot update\n\n---------\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>\nCo-authored-by: github-actions[bot] <41898282+github-actions[bot]@users.noreply.github.com>",
          "timestamp": "2026-09-08T07:45:57Z",
          "tree_id": "bf0206de474f22c41c14b3729f6abf0a5ae30f7d",
          "url": "https://github.com/evstack/ev-node/commit/27b6a6ca2ffeea40824faa2ce8abb12512aa660e"
        },
        "date": 1788854898560,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 36107,
            "unit": "ns/op\t    4849 B/op\t      51 allocs/op",
            "extra": "33380 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 36107,
            "unit": "ns/op",
            "extra": "33380 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4849,
            "unit": "B/op",
            "extra": "33380 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 51,
            "unit": "allocs/op",
            "extra": "33380 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 36904,
            "unit": "ns/op\t    5057 B/op\t      55 allocs/op",
            "extra": "32707 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 36904,
            "unit": "ns/op",
            "extra": "32707 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 5057,
            "unit": "B/op",
            "extra": "32707 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "32707 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 43338,
            "unit": "ns/op\t   10361 B/op\t      55 allocs/op",
            "extra": "27805 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 43338,
            "unit": "ns/op",
            "extra": "27805 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10361,
            "unit": "B/op",
            "extra": "27805 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "27805 times\n4 procs"
          }
        ]
      }
    ]
  }
}