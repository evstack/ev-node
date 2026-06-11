window.BENCHMARK_DATA = {
  "lastUpdate": 1781176182509,
  "repoUrl": "https://github.com/evstack/ev-node",
  "entries": {
    "EVM Contract Roundtrip": [
      {
        "commit": {
          "author": {
            "email": "nuke-web3@proton.me",
            "name": "Nuke 🌄",
            "username": "nuke-web3"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "74efee38260cd5072000a3a1ef349ae6a408fe1a",
          "message": "Add locad devnet guide (#3350)\n\n* oneshot local devnet guide w/ claude\n\n* tested to work with docker, but really upstream tooling needs some tweaks",
          "timestamp": "2026-06-11T13:01:13+02:00",
          "tree_id": "bb7035bd753c9e0fd8874d4a84d0c8025bebd379",
          "url": "https://github.com/evstack/ev-node/commit/74efee38260cd5072000a3a1ef349ae6a408fe1a"
        },
        "date": 1781176172377,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 895622446,
            "unit": "ns/op\t30641340 B/op\t  165486 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 895622446,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 30641340,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 165486,
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
          "id": "29e854269b75c3fb222efc7c8f49821485fa5d27",
          "message": "build(deps): Bump codecov/codecov-action from 6 to 7 (#3348)\n\nBumps [codecov/codecov-action](https://github.com/codecov/codecov-action) from 6 to 7.\n- [Release notes](https://github.com/codecov/codecov-action/releases)\n- [Changelog](https://github.com/codecov/codecov-action/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/codecov/codecov-action/compare/v6...v7)\n\n---\nupdated-dependencies:\n- dependency-name: codecov/codecov-action\n  dependency-version: '7'\n  dependency-type: direct:production\n  update-type: version-update:semver-major\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-06-11T13:04:00+02:00",
          "tree_id": "abcef3b79a2f5784521d5b8a885ddb4676ec27d3",
          "url": "https://github.com/evstack/ev-node/commit/29e854269b75c3fb222efc7c8f49821485fa5d27"
        },
        "date": 1781176177103,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 895332713,
            "unit": "ns/op\t30646116 B/op\t  162045 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 895332713,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 30646116,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 162045,
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
            "email": "nuke-web3@proton.me",
            "name": "Nuke 🌄",
            "username": "nuke-web3"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "74efee38260cd5072000a3a1ef349ae6a408fe1a",
          "message": "Add locad devnet guide (#3350)\n\n* oneshot local devnet guide w/ claude\n\n* tested to work with docker, but really upstream tooling needs some tweaks",
          "timestamp": "2026-06-11T13:01:13+02:00",
          "tree_id": "bb7035bd753c9e0fd8874d4a84d0c8025bebd379",
          "url": "https://github.com/evstack/ev-node/commit/74efee38260cd5072000a3a1ef349ae6a408fe1a"
        },
        "date": 1781176178291,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 39607,
            "unit": "ns/op\t    4803 B/op\t      50 allocs/op",
            "extra": "30816 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 39607,
            "unit": "ns/op",
            "extra": "30816 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4803,
            "unit": "B/op",
            "extra": "30816 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 50,
            "unit": "allocs/op",
            "extra": "30816 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 40052,
            "unit": "ns/op\t    5018 B/op\t      54 allocs/op",
            "extra": "29985 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 40052,
            "unit": "ns/op",
            "extra": "29985 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 5018,
            "unit": "B/op",
            "extra": "29985 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "29985 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 46709,
            "unit": "ns/op\t   10303 B/op\t      54 allocs/op",
            "extra": "26326 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 46709,
            "unit": "ns/op",
            "extra": "26326 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10303,
            "unit": "B/op",
            "extra": "26326 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "26326 times\n4 procs"
          }
        ]
      }
    ]
  }
}