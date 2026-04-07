window.BENCHMARK_DATA = {
  "lastUpdate": 1775559288395,
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
          "id": "920f0c9da38deb23fc70a0a717c2cc3f6beaaa35",
          "message": "build(deps): Bump extractions/setup-just from 3 to 4 (#3227)\n\nBumps [extractions/setup-just](https://github.com/extractions/setup-just) from 3 to 4.\n- [Release notes](https://github.com/extractions/setup-just/releases)\n- [Commits](https://github.com/extractions/setup-just/compare/v3...v4)\n\n---\nupdated-dependencies:\n- dependency-name: extractions/setup-just\n  dependency-version: '4'\n  dependency-type: direct:production\n  update-type: version-update:semver-major\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-04-07T12:45:43+02:00",
          "tree_id": "f79670819d05a09daecaee67220f1aa3df8fa261",
          "url": "https://github.com/evstack/ev-node/commit/920f0c9da38deb23fc70a0a717c2cc3f6beaaa35"
        },
        "date": 1775559283482,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 906978622,
            "unit": "ns/op\t32750984 B/op\t  185428 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 906978622,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 32750984,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 185428,
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
          "id": "920f0c9da38deb23fc70a0a717c2cc3f6beaaa35",
          "message": "build(deps): Bump extractions/setup-just from 3 to 4 (#3227)\n\nBumps [extractions/setup-just](https://github.com/extractions/setup-just) from 3 to 4.\n- [Release notes](https://github.com/extractions/setup-just/releases)\n- [Commits](https://github.com/extractions/setup-just/compare/v3...v4)\n\n---\nupdated-dependencies:\n- dependency-name: extractions/setup-just\n  dependency-version: '4'\n  dependency-type: direct:production\n  update-type: version-update:semver-major\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-04-07T12:45:43+02:00",
          "tree_id": "f79670819d05a09daecaee67220f1aa3df8fa261",
          "url": "https://github.com/evstack/ev-node/commit/920f0c9da38deb23fc70a0a717c2cc3f6beaaa35"
        },
        "date": 1775559287909,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 36874,
            "unit": "ns/op\t    7027 B/op\t      71 allocs/op",
            "extra": "34057 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 36874,
            "unit": "ns/op",
            "extra": "34057 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7027,
            "unit": "B/op",
            "extra": "34057 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "34057 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 36119,
            "unit": "ns/op\t    7479 B/op\t      81 allocs/op",
            "extra": "33492 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 36119,
            "unit": "ns/op",
            "extra": "33492 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7479,
            "unit": "B/op",
            "extra": "33492 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "33492 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 46370,
            "unit": "ns/op\t   26185 B/op\t      81 allocs/op",
            "extra": "25976 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 46370,
            "unit": "ns/op",
            "extra": "25976 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26185,
            "unit": "B/op",
            "extra": "25976 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "25976 times\n4 procs"
          }
        ]
      }
    ]
  }
}