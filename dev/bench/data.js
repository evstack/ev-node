window.BENCHMARK_DATA = {
  "lastUpdate": 1779781542100,
  "repoUrl": "https://github.com/evstack/ev-node",
  "entries": {
    "EVM Contract Roundtrip": [
      {
        "commit": {
          "author": {
            "email": "caltechustc@outlook.com",
            "name": "caltechustc",
            "username": "caltechustc"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "5ee6483aa360fab4785bb5fe820a1b5a01e8ecf2",
          "message": "chore: remove duplicate package import (#3331)\n\nchore: remove duplicate package import and fix some CI issues\n\nSigned-off-by: caltechustc <caltechustc@outlook.com>",
          "timestamp": "2026-05-24T18:03:26Z",
          "tree_id": "2dc6474b4cd0dc53fa8ca1155e12691b9e7cbb40",
          "url": "https://github.com/evstack/ev-node/commit/5ee6483aa360fab4785bb5fe820a1b5a01e8ecf2"
        },
        "date": 1779646986805,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 916781524,
            "unit": "ns/op\t33067568 B/op\t  186084 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 916781524,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 33067568,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 186084,
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
          "id": "8edee19b6111669c7103460fd8377a5583be66aa",
          "message": "build(deps): Bump golangci/golangci-lint-action from 9.2.0 to 9.2.1 in the patch-updates group (#3332)\n\nbuild(deps): Bump golangci/golangci-lint-action\n\nBumps the patch-updates group with 1 update: [golangci/golangci-lint-action](https://github.com/golangci/golangci-lint-action).\n\n\nUpdates `golangci/golangci-lint-action` from 9.2.0 to 9.2.1\n- [Release notes](https://github.com/golangci/golangci-lint-action/releases)\n- [Commits](https://github.com/golangci/golangci-lint-action/compare/v9.2.0...v9.2.1)\n\n---\nupdated-dependencies:\n- dependency-name: golangci/golangci-lint-action\n  dependency-version: 9.2.1\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: patch-updates\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-05-26T09:41:47+02:00",
          "tree_id": "dabfb040f762f5e396b9fa901994dc7752825140",
          "url": "https://github.com/evstack/ev-node/commit/8edee19b6111669c7103460fd8377a5583be66aa"
        },
        "date": 1779781538140,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 910957676,
            "unit": "ns/op\t32408576 B/op\t  181141 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 910957676,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 32408576,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 181141,
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
            "email": "caltechustc@outlook.com",
            "name": "caltechustc",
            "username": "caltechustc"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "5ee6483aa360fab4785bb5fe820a1b5a01e8ecf2",
          "message": "chore: remove duplicate package import (#3331)\n\nchore: remove duplicate package import and fix some CI issues\n\nSigned-off-by: caltechustc <caltechustc@outlook.com>",
          "timestamp": "2026-05-24T18:03:26Z",
          "tree_id": "2dc6474b4cd0dc53fa8ca1155e12691b9e7cbb40",
          "url": "https://github.com/evstack/ev-node/commit/5ee6483aa360fab4785bb5fe820a1b5a01e8ecf2"
        },
        "date": 1779646993517,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 34523,
            "unit": "ns/op\t    4705 B/op\t      50 allocs/op",
            "extra": "34983 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 34523,
            "unit": "ns/op",
            "extra": "34983 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4705,
            "unit": "B/op",
            "extra": "34983 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 50,
            "unit": "allocs/op",
            "extra": "34983 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 35103,
            "unit": "ns/op\t    4907 B/op\t      54 allocs/op",
            "extra": "34531 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 35103,
            "unit": "ns/op",
            "extra": "34531 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 4907,
            "unit": "B/op",
            "extra": "34531 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "34531 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 40894,
            "unit": "ns/op\t   10202 B/op\t      54 allocs/op",
            "extra": "29430 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 40894,
            "unit": "ns/op",
            "extra": "29430 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10202,
            "unit": "B/op",
            "extra": "29430 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "29430 times\n4 procs"
          }
        ]
      }
    ]
  }
}