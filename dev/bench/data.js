window.BENCHMARK_DATA = {
  "lastUpdate": 1778512500153,
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
          "id": "7c1338825bf7196122bbc46c4e8e8af62f50e2ee",
          "message": "build(deps): Bump uuid from 11.1.0 to 14.0.0 in /docs in the npm_and_yarn group across 1 directory (#3320)\n\nbuild(deps): Bump uuid\n\nBumps the npm_and_yarn group with 1 update in the /docs directory: [uuid](https://github.com/uuidjs/uuid).\n\n\nUpdates `uuid` from 11.1.0 to 14.0.0\n- [Release notes](https://github.com/uuidjs/uuid/releases)\n- [Changelog](https://github.com/uuidjs/uuid/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/uuidjs/uuid/compare/v11.1.0...v14.0.0)\n\n---\nupdated-dependencies:\n- dependency-name: uuid\n  dependency-version: 14.0.0\n  dependency-type: indirect\n  dependency-group: npm_and_yarn\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-05-11T14:53:37Z",
          "tree_id": "9cdd44b8b0c7182e880afb5150627d8c07ddcf3c",
          "url": "https://github.com/evstack/ev-node/commit/7c1338825bf7196122bbc46c4e8e8af62f50e2ee"
        },
        "date": 1778512494071,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 885946721,
            "unit": "ns/op\t30102816 B/op\t  160837 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 885946721,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 30102816,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 160837,
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
          "id": "7c1338825bf7196122bbc46c4e8e8af62f50e2ee",
          "message": "build(deps): Bump uuid from 11.1.0 to 14.0.0 in /docs in the npm_and_yarn group across 1 directory (#3320)\n\nbuild(deps): Bump uuid\n\nBumps the npm_and_yarn group with 1 update in the /docs directory: [uuid](https://github.com/uuidjs/uuid).\n\n\nUpdates `uuid` from 11.1.0 to 14.0.0\n- [Release notes](https://github.com/uuidjs/uuid/releases)\n- [Changelog](https://github.com/uuidjs/uuid/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/uuidjs/uuid/compare/v11.1.0...v14.0.0)\n\n---\nupdated-dependencies:\n- dependency-name: uuid\n  dependency-version: 14.0.0\n  dependency-type: indirect\n  dependency-group: npm_and_yarn\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-05-11T14:53:37Z",
          "tree_id": "9cdd44b8b0c7182e880afb5150627d8c07ddcf3c",
          "url": "https://github.com/evstack/ev-node/commit/7c1338825bf7196122bbc46c4e8e8af62f50e2ee"
        },
        "date": 1778512499393,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37548,
            "unit": "ns/op\t    4762 B/op\t      50 allocs/op",
            "extra": "32418 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37548,
            "unit": "ns/op",
            "extra": "32418 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4762,
            "unit": "B/op",
            "extra": "32418 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 50,
            "unit": "allocs/op",
            "extra": "32418 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 38299,
            "unit": "ns/op\t    4970 B/op\t      54 allocs/op",
            "extra": "31786 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 38299,
            "unit": "ns/op",
            "extra": "31786 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 4970,
            "unit": "B/op",
            "extra": "31786 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "31786 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 44770,
            "unit": "ns/op\t   10269 B/op\t      54 allocs/op",
            "extra": "27313 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 44770,
            "unit": "ns/op",
            "extra": "27313 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10269,
            "unit": "B/op",
            "extra": "27313 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "27313 times\n4 procs"
          }
        ]
      }
    ]
  }
}