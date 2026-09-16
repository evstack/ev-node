window.BENCHMARK_DATA = {
  "lastUpdate": 1789549361443,
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
          "id": "6acb56dcf034a6c2a780b93fbd1b24b11d7eb39c",
          "message": "build(deps): Bump the npm_and_yarn group across 1 directory with 2 updates (#3449)\n\nBumps the npm_and_yarn group with 1 update in the /docs directory: [mermaid](https://github.com/mermaid-js/mermaid).\n\n\nUpdates `mermaid` from 11.15.0 to 11.16.1\n- [Release notes](https://github.com/mermaid-js/mermaid/releases)\n- [Commits](https://github.com/mermaid-js/mermaid/compare/mermaid@11.15.0...mermaid@11.16.1)\n\nUpdates `dompurify` from 3.4.12 to 3.4.15\n- [Release notes](https://github.com/cure53/DOMPurify/releases)\n- [Commits](https://github.com/cure53/DOMPurify/compare/3.4.12...3.4.15)\n\n---\nupdated-dependencies:\n- dependency-name: mermaid\n  dependency-version: 11.16.1\n  dependency-type: direct:development\n  dependency-group: npm_and_yarn\n- dependency-name: dompurify\n  dependency-version: 3.4.15\n  dependency-type: indirect\n  dependency-group: npm_and_yarn\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-09-16T08:41:46Z",
          "tree_id": "47663588df51a8ff203731382fbb4e8dc887a404",
          "url": "https://github.com/evstack/ev-node/commit/6acb56dcf034a6c2a780b93fbd1b24b11d7eb39c"
        },
        "date": 1789549352641,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 910454268,
            "unit": "ns/op\t 4256228 B/op\t   36402 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 910454268,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 4256228,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 36402,
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
          "id": "6acb56dcf034a6c2a780b93fbd1b24b11d7eb39c",
          "message": "build(deps): Bump the npm_and_yarn group across 1 directory with 2 updates (#3449)\n\nBumps the npm_and_yarn group with 1 update in the /docs directory: [mermaid](https://github.com/mermaid-js/mermaid).\n\n\nUpdates `mermaid` from 11.15.0 to 11.16.1\n- [Release notes](https://github.com/mermaid-js/mermaid/releases)\n- [Commits](https://github.com/mermaid-js/mermaid/compare/mermaid@11.15.0...mermaid@11.16.1)\n\nUpdates `dompurify` from 3.4.12 to 3.4.15\n- [Release notes](https://github.com/cure53/DOMPurify/releases)\n- [Commits](https://github.com/cure53/DOMPurify/compare/3.4.12...3.4.15)\n\n---\nupdated-dependencies:\n- dependency-name: mermaid\n  dependency-version: 11.16.1\n  dependency-type: direct:development\n  dependency-group: npm_and_yarn\n- dependency-name: dompurify\n  dependency-version: 3.4.15\n  dependency-type: indirect\n  dependency-group: npm_and_yarn\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-09-16T08:41:46Z",
          "tree_id": "47663588df51a8ff203731382fbb4e8dc887a404",
          "url": "https://github.com/evstack/ev-node/commit/6acb56dcf034a6c2a780b93fbd1b24b11d7eb39c"
        },
        "date": 1789549360535,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 46105,
            "unit": "ns/op\t   10417 B/op\t      55 allocs/op",
            "extra": "26206 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 46105,
            "unit": "ns/op",
            "extra": "26206 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10417,
            "unit": "B/op",
            "extra": "26206 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "26206 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 39814,
            "unit": "ns/op\t    4909 B/op\t      51 allocs/op",
            "extra": "30943 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 39814,
            "unit": "ns/op",
            "extra": "30943 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4909,
            "unit": "B/op",
            "extra": "30943 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 51,
            "unit": "allocs/op",
            "extra": "30943 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 40530,
            "unit": "ns/op\t    5131 B/op\t      55 allocs/op",
            "extra": "29858 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 40530,
            "unit": "ns/op",
            "extra": "29858 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 5131,
            "unit": "B/op",
            "extra": "29858 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "29858 times\n4 procs"
          }
        ]
      }
    ]
  }
}