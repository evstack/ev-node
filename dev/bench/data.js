window.BENCHMARK_DATA = {
  "lastUpdate": 1779098853434,
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
          "id": "27eeb486a21a44030226d5e31472171dd83209b1",
          "message": "build(deps): Bump uuid from 11.1.0 to 14.0.0 in /docs in the npm_and_yarn group across 1 directory (#3321)\n\nbuild(deps): Bump uuid\n\nBumps the npm_and_yarn group with 1 update in the /docs directory: [uuid](https://github.com/uuidjs/uuid).\n\n\nUpdates `uuid` from 11.1.0 to 14.0.0\n- [Release notes](https://github.com/uuidjs/uuid/releases)\n- [Changelog](https://github.com/uuidjs/uuid/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/uuidjs/uuid/compare/v11.1.0...v14.0.0)\n\n---\nupdated-dependencies:\n- dependency-name: uuid\n  dependency-version: 14.0.0\n  dependency-type: indirect\n  dependency-group: npm_and_yarn\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-05-12T10:37:43+02:00",
          "tree_id": "c2deadee909f25e0b4776e61e8c6b7319e79107e",
          "url": "https://github.com/evstack/ev-node/commit/27eeb486a21a44030226d5e31472171dd83209b1"
        },
        "date": 1778575567842,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 913727039,
            "unit": "ns/op\t32946924 B/op\t  186129 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 913727039,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 32946924,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 186129,
            "unit": "allocs/op",
            "extra": "2 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "cuoguojida@outlook.com",
            "name": "cuoguojida",
            "username": "cuoguojida"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "a2dae255e6252340f9676231ca0a52456808f612",
          "message": "chore: fix some comments to improve readability (#3325)\n\nSigned-off-by: cuoguojida <cuoguojida@outlook.com>",
          "timestamp": "2026-05-18T08:55:23+02:00",
          "tree_id": "ba5ae484f3b4608a6356b7da78639229b22fa104",
          "url": "https://github.com/evstack/ev-node/commit/a2dae255e6252340f9676231ca0a52456808f612"
        },
        "date": 1779087539255,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 910577446,
            "unit": "ns/op\t31848864 B/op\t  176240 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 910577446,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 31848864,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 176240,
            "unit": "allocs/op",
            "extra": "2 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "github.qpeyb@simplelogin.fr",
            "name": "Cian Hatton",
            "username": "chatton"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "890438743e3f5c74e4563fbd229759d8e449bdc9",
          "message": "chore: using v0.20.0 tag of tastora and removing temporary duplication (#3326)\n\n* refactor: use tastora spamoor and victoriatraces implementations\n\nReplace local spamoor_node.go and victoriatraces_node.go with tastora's\nspamoor.NewNodeBuilder and victoriatraces.New. Fixes macOS compatibility\nby replacing 0.0.0.0 with 127.0.0.1 for external endpoints and using\nUTC timestamps in trace query URLs to avoid broken timezone offsets.\n\n* build(deps): bump tastora to v0.20.0 and remove local replace\n\nRemove the local filesystem replace directive for tastora in\ntest/e2e/go.mod now that v0.20.0 is published.\n\n* build: bump go directive to 1.25.8\n\nRequired by tastora v0.20.0. Aligns root and execution/evm\nwith the other modules already at 1.25.8.",
          "timestamp": "2026-05-18T12:03:09+02:00",
          "tree_id": "0820ca693f7423abc20da38da593de2ba6cb9557",
          "url": "https://github.com/evstack/ev-node/commit/890438743e3f5c74e4563fbd229759d8e449bdc9"
        },
        "date": 1779098848878,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 904754234,
            "unit": "ns/op\t31828520 B/op\t  175940 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 904754234,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 31828520,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 175940,
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
          "id": "27eeb486a21a44030226d5e31472171dd83209b1",
          "message": "build(deps): Bump uuid from 11.1.0 to 14.0.0 in /docs in the npm_and_yarn group across 1 directory (#3321)\n\nbuild(deps): Bump uuid\n\nBumps the npm_and_yarn group with 1 update in the /docs directory: [uuid](https://github.com/uuidjs/uuid).\n\n\nUpdates `uuid` from 11.1.0 to 14.0.0\n- [Release notes](https://github.com/uuidjs/uuid/releases)\n- [Changelog](https://github.com/uuidjs/uuid/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/uuidjs/uuid/compare/v11.1.0...v14.0.0)\n\n---\nupdated-dependencies:\n- dependency-name: uuid\n  dependency-version: 14.0.0\n  dependency-type: indirect\n  dependency-group: npm_and_yarn\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-05-12T10:37:43+02:00",
          "tree_id": "c2deadee909f25e0b4776e61e8c6b7319e79107e",
          "url": "https://github.com/evstack/ev-node/commit/27eeb486a21a44030226d5e31472171dd83209b1"
        },
        "date": 1778575574311,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 36158,
            "unit": "ns/op\t    4724 B/op\t      50 allocs/op",
            "extra": "34096 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 36158,
            "unit": "ns/op",
            "extra": "34096 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4724,
            "unit": "B/op",
            "extra": "34096 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 50,
            "unit": "allocs/op",
            "extra": "34096 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 36421,
            "unit": "ns/op\t    4927 B/op\t      54 allocs/op",
            "extra": "33602 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 36421,
            "unit": "ns/op",
            "extra": "33602 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 4927,
            "unit": "B/op",
            "extra": "33602 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "33602 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 42440,
            "unit": "ns/op\t   10222 B/op\t      54 allocs/op",
            "extra": "28754 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 42440,
            "unit": "ns/op",
            "extra": "28754 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10222,
            "unit": "B/op",
            "extra": "28754 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "28754 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "cuoguojida@outlook.com",
            "name": "cuoguojida",
            "username": "cuoguojida"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "a2dae255e6252340f9676231ca0a52456808f612",
          "message": "chore: fix some comments to improve readability (#3325)\n\nSigned-off-by: cuoguojida <cuoguojida@outlook.com>",
          "timestamp": "2026-05-18T08:55:23+02:00",
          "tree_id": "ba5ae484f3b4608a6356b7da78639229b22fa104",
          "url": "https://github.com/evstack/ev-node/commit/a2dae255e6252340f9676231ca0a52456808f612"
        },
        "date": 1779087544315,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 38435,
            "unit": "ns/op\t    4768 B/op\t      50 allocs/op",
            "extra": "32162 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 38435,
            "unit": "ns/op",
            "extra": "32162 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4768,
            "unit": "B/op",
            "extra": "32162 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 50,
            "unit": "allocs/op",
            "extra": "32162 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 38874,
            "unit": "ns/op\t    4978 B/op\t      54 allocs/op",
            "extra": "31465 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 38874,
            "unit": "ns/op",
            "extra": "31465 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 4978,
            "unit": "B/op",
            "extra": "31465 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "31465 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 44771,
            "unit": "ns/op\t   10268 B/op\t      54 allocs/op",
            "extra": "27326 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 44771,
            "unit": "ns/op",
            "extra": "27326 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10268,
            "unit": "B/op",
            "extra": "27326 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "27326 times\n4 procs"
          }
        ]
      }
    ]
  }
}