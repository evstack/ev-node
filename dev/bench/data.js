window.BENCHMARK_DATA = {
  "lastUpdate": 1776154069851,
  "repoUrl": "https://github.com/evstack/ev-node",
  "entries": {
    "EVM Contract Roundtrip": [
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
          "distinct": true,
          "id": "2b1784a6bc869d6eec3cd1de04f0a8f9b78b06e1",
          "message": "build(deps): Bump actions/upload-artifact from 7.0.0 to 7.0.1 in the patch-updates group (#3246)\n\nbuild(deps): Bump actions/upload-artifact in the patch-updates group\n\nBumps the patch-updates group with 1 update: [actions/upload-artifact](https://github.com/actions/upload-artifact).\n\n\nUpdates `actions/upload-artifact` from 7.0.0 to 7.0.1\n- [Release notes](https://github.com/actions/upload-artifact/releases)\n- [Commits](https://github.com/actions/upload-artifact/compare/v7...v7.0.1)\n\n---\nupdated-dependencies:\n- dependency-name: actions/upload-artifact\n  dependency-version: 7.0.1\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: patch-updates\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-04-14T09:15:31+02:00",
          "tree_id": "2a53071d06a7e752aa2cd523a235eda138954133",
          "url": "https://github.com/evstack/ev-node/commit/2b1784a6bc869d6eec3cd1de04f0a8f9b78b06e1"
        },
        "date": 1776154064322,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 910376030,
            "unit": "ns/op\t32434800 B/op\t  180985 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 910376030,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 32434800,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 180985,
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
            "email": "49699333+dependabot[bot]@users.noreply.github.com",
            "name": "dependabot[bot]",
            "username": "dependabot[bot]"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "2b1784a6bc869d6eec3cd1de04f0a8f9b78b06e1",
          "message": "build(deps): Bump actions/upload-artifact from 7.0.0 to 7.0.1 in the patch-updates group (#3246)\n\nbuild(deps): Bump actions/upload-artifact in the patch-updates group\n\nBumps the patch-updates group with 1 update: [actions/upload-artifact](https://github.com/actions/upload-artifact).\n\n\nUpdates `actions/upload-artifact` from 7.0.0 to 7.0.1\n- [Release notes](https://github.com/actions/upload-artifact/releases)\n- [Commits](https://github.com/actions/upload-artifact/compare/v7...v7.0.1)\n\n---\nupdated-dependencies:\n- dependency-name: actions/upload-artifact\n  dependency-version: 7.0.1\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: patch-updates\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-04-14T09:15:31+02:00",
          "tree_id": "2a53071d06a7e752aa2cd523a235eda138954133",
          "url": "https://github.com/evstack/ev-node/commit/2b1784a6bc869d6eec3cd1de04f0a8f9b78b06e1"
        },
        "date": 1776154069270,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 48073,
            "unit": "ns/op\t   26193 B/op\t      81 allocs/op",
            "extra": "25753 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 48073,
            "unit": "ns/op",
            "extra": "25753 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26193,
            "unit": "B/op",
            "extra": "25753 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "25753 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37006,
            "unit": "ns/op\t    7051 B/op\t      71 allocs/op",
            "extra": "32982 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37006,
            "unit": "ns/op",
            "extra": "32982 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7051,
            "unit": "B/op",
            "extra": "32982 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "32982 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 38289,
            "unit": "ns/op\t    7512 B/op\t      81 allocs/op",
            "extra": "32108 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 38289,
            "unit": "ns/op",
            "extra": "32108 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7512,
            "unit": "B/op",
            "extra": "32108 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "32108 times\n4 procs"
          }
        ]
      }
    ]
  }
}