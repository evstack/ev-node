window.BENCHMARK_DATA = {
  "lastUpdate": 1789548223928,
  "repoUrl": "https://github.com/evstack/ev-node",
  "entries": {
    "EVM Contract Roundtrip": [
      {
        "commit": {
          "author": {
            "email": "ginavalent@outlook.com",
            "name": "ginavalent",
            "username": "ginavalent"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "6b40f9768b8b480329ae375471334a535e3656ae",
          "message": "chore: minor improvement for docs (#3450)\n\nSigned-off-by: ginavalent <ginavalent@outlook.com>",
          "timestamp": "2026-09-15T15:41:43+02:00",
          "tree_id": "80337cdcee0da223f5c6517c7384d7db651ab043",
          "url": "https://github.com/evstack/ev-node/commit/6b40f9768b8b480329ae375471334a535e3656ae"
        },
        "date": 1789479873472,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 912338854,
            "unit": "ns/op\t 4313292 B/op\t   36731 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 912338854,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 4313292,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 36731,
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
          "distinct": true,
          "id": "f2a026e36ba4547494b143bcfb8fda3dfdcb828e",
          "message": "build(deps): Bump google.golang.org/grpc from 1.82.1 to 1.83.1 in /tools/da-debug in the go_modules group across 1 directory (#3448)\n\nbuild(deps): Bump google.golang.org/grpc\n\nBumps the go_modules group with 1 update in the /tools/da-debug directory: [google.golang.org/grpc](https://github.com/grpc/grpc-go).\n\n\nUpdates `google.golang.org/grpc` from 1.82.1 to 1.83.1\n- [Release notes](https://github.com/grpc/grpc-go/releases)\n- [Commits](https://github.com/grpc/grpc-go/compare/v1.82.1...v1.83.1)\n\n---\nupdated-dependencies:\n- dependency-name: google.golang.org/grpc\n  dependency-version: 1.83.1\n  dependency-type: indirect\n  dependency-group: go_modules\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-09-16T10:41:32+02:00",
          "tree_id": "56b234b9be22ebcabdb580d51d933818d280c472",
          "url": "https://github.com/evstack/ev-node/commit/f2a026e36ba4547494b143bcfb8fda3dfdcb828e"
        },
        "date": 1789548215508,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 903141285,
            "unit": "ns/op\t 4214548 B/op\t   36334 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 903141285,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 4214548,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 36334,
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
            "email": "ginavalent@outlook.com",
            "name": "ginavalent",
            "username": "ginavalent"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "6b40f9768b8b480329ae375471334a535e3656ae",
          "message": "chore: minor improvement for docs (#3450)\n\nSigned-off-by: ginavalent <ginavalent@outlook.com>",
          "timestamp": "2026-09-15T15:41:43+02:00",
          "tree_id": "80337cdcee0da223f5c6517c7384d7db651ab043",
          "url": "https://github.com/evstack/ev-node/commit/6b40f9768b8b480329ae375471334a535e3656ae"
        },
        "date": 1789479881190,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 22071,
            "unit": "ns/op\t    4996 B/op\t      51 allocs/op",
            "extra": "55904 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 22071,
            "unit": "ns/op",
            "extra": "55904 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4996,
            "unit": "B/op",
            "extra": "55904 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 51,
            "unit": "allocs/op",
            "extra": "55904 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 23183,
            "unit": "ns/op\t    5220 B/op\t      55 allocs/op",
            "extra": "54032 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 23183,
            "unit": "ns/op",
            "extra": "54032 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 5220,
            "unit": "B/op",
            "extra": "54032 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "54032 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 26747,
            "unit": "ns/op\t   10235 B/op\t      55 allocs/op",
            "extra": "45478 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 26747,
            "unit": "ns/op",
            "extra": "45478 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10235,
            "unit": "B/op",
            "extra": "45478 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "45478 times\n4 procs"
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
          "distinct": true,
          "id": "f2a026e36ba4547494b143bcfb8fda3dfdcb828e",
          "message": "build(deps): Bump google.golang.org/grpc from 1.82.1 to 1.83.1 in /tools/da-debug in the go_modules group across 1 directory (#3448)\n\nbuild(deps): Bump google.golang.org/grpc\n\nBumps the go_modules group with 1 update in the /tools/da-debug directory: [google.golang.org/grpc](https://github.com/grpc/grpc-go).\n\n\nUpdates `google.golang.org/grpc` from 1.82.1 to 1.83.1\n- [Release notes](https://github.com/grpc/grpc-go/releases)\n- [Commits](https://github.com/grpc/grpc-go/compare/v1.82.1...v1.83.1)\n\n---\nupdated-dependencies:\n- dependency-name: google.golang.org/grpc\n  dependency-version: 1.83.1\n  dependency-type: indirect\n  dependency-group: go_modules\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-09-16T10:41:32+02:00",
          "tree_id": "56b234b9be22ebcabdb580d51d933818d280c472",
          "url": "https://github.com/evstack/ev-node/commit/f2a026e36ba4547494b143bcfb8fda3dfdcb828e"
        },
        "date": 1789548222988,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 28609,
            "unit": "ns/op\t    4690 B/op\t      51 allocs/op",
            "extra": "42524 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 28609,
            "unit": "ns/op",
            "extra": "42524 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4690,
            "unit": "B/op",
            "extra": "42524 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 51,
            "unit": "allocs/op",
            "extra": "42524 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 29565,
            "unit": "ns/op\t    4892 B/op\t      55 allocs/op",
            "extra": "41634 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 29565,
            "unit": "ns/op",
            "extra": "41634 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 4892,
            "unit": "B/op",
            "extra": "41634 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "41634 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 35052,
            "unit": "ns/op\t   10186 B/op\t      55 allocs/op",
            "extra": "34504 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 35052,
            "unit": "ns/op",
            "extra": "34504 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10186,
            "unit": "B/op",
            "extra": "34504 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "34504 times\n4 procs"
          }
        ]
      }
    ]
  }
}