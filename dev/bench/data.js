window.BENCHMARK_DATA = {
  "lastUpdate": 1777380129374,
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
          "id": "e83cf6c7447839067e22e1d444214dd645f7afb2",
          "message": "build(deps): Bump postcss from 8.5.8 to 8.5.12 in /docs in the npm_and_yarn group across 1 directory (#3292)\n\nbuild(deps): Bump postcss\n\nBumps the npm_and_yarn group with 1 update in the /docs directory: [postcss](https://github.com/postcss/postcss).\n\n\nUpdates `postcss` from 8.5.8 to 8.5.12\n- [Release notes](https://github.com/postcss/postcss/releases)\n- [Changelog](https://github.com/postcss/postcss/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/postcss/postcss/compare/8.5.8...8.5.12)\n\n---\nupdated-dependencies:\n- dependency-name: postcss\n  dependency-version: 8.5.12\n  dependency-type: indirect\n  dependency-group: npm_and_yarn\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-04-28T14:31:09+02:00",
          "tree_id": "b21bed04cbae419fb1620e0ef1662158cafe5cff",
          "url": "https://github.com/evstack/ev-node/commit/e83cf6c7447839067e22e1d444214dd645f7afb2"
        },
        "date": 1777380122207,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 904735388,
            "unit": "ns/op\t31698224 B/op\t  175848 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 904735388,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 31698224,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 175848,
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
          "id": "e83cf6c7447839067e22e1d444214dd645f7afb2",
          "message": "build(deps): Bump postcss from 8.5.8 to 8.5.12 in /docs in the npm_and_yarn group across 1 directory (#3292)\n\nbuild(deps): Bump postcss\n\nBumps the npm_and_yarn group with 1 update in the /docs directory: [postcss](https://github.com/postcss/postcss).\n\n\nUpdates `postcss` from 8.5.8 to 8.5.12\n- [Release notes](https://github.com/postcss/postcss/releases)\n- [Changelog](https://github.com/postcss/postcss/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/postcss/postcss/compare/8.5.8...8.5.12)\n\n---\nupdated-dependencies:\n- dependency-name: postcss\n  dependency-version: 8.5.12\n  dependency-type: indirect\n  dependency-group: npm_and_yarn\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-04-28T14:31:09+02:00",
          "tree_id": "b21bed04cbae419fb1620e0ef1662158cafe5cff",
          "url": "https://github.com/evstack/ev-node/commit/e83cf6c7447839067e22e1d444214dd645f7afb2"
        },
        "date": 1777380128791,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 36915,
            "unit": "ns/op\t    4755 B/op\t      50 allocs/op",
            "extra": "32728 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 36915,
            "unit": "ns/op",
            "extra": "32728 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4755,
            "unit": "B/op",
            "extra": "32728 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 50,
            "unit": "allocs/op",
            "extra": "32728 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 37551,
            "unit": "ns/op\t    4948 B/op\t      54 allocs/op",
            "extra": "32694 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 37551,
            "unit": "ns/op",
            "extra": "32694 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 4948,
            "unit": "B/op",
            "extra": "32694 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "32694 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 43629,
            "unit": "ns/op\t   10260 B/op\t      54 allocs/op",
            "extra": "27562 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 43629,
            "unit": "ns/op",
            "extra": "27562 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10260,
            "unit": "B/op",
            "extra": "27562 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "27562 times\n4 procs"
          }
        ]
      }
    ]
  }
}