window.BENCHMARK_DATA = {
  "lastUpdate": 1778575575081,
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
      }
    ]
  }
}