window.BENCHMARK_DATA = {
  "lastUpdate": 1782727577590,
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
          "id": "69c655c80bab7f129fe6f3ad4cd27e48bdbcc603",
          "message": "build(deps): Bump dompurify from 3.4.0 to 3.4.11 in /docs in the npm_and_yarn group across 1 directory (#3366)\n\nbuild(deps): Bump dompurify\n\nBumps the npm_and_yarn group with 1 update in the /docs directory: [dompurify](https://github.com/cure53/DOMPurify).\n\n\nUpdates `dompurify` from 3.4.0 to 3.4.11\n- [Release notes](https://github.com/cure53/DOMPurify/releases)\n- [Commits](https://github.com/cure53/DOMPurify/compare/3.4.0...3.4.11)\n\n---\nupdated-dependencies:\n- dependency-name: dompurify\n  dependency-version: 3.4.11\n  dependency-type: indirect\n  dependency-group: npm_and_yarn\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-06-29T12:02:17+02:00",
          "tree_id": "8d16df760af045f500d1ac88648f9edae597b8d5",
          "url": "https://github.com/evstack/ev-node/commit/69c655c80bab7f129fe6f3ad4cd27e48bdbcc603"
        },
        "date": 1782727569653,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 899645124,
            "unit": "ns/op\t31028384 B/op\t  168446 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 899645124,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 31028384,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 168446,
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
          "id": "69c655c80bab7f129fe6f3ad4cd27e48bdbcc603",
          "message": "build(deps): Bump dompurify from 3.4.0 to 3.4.11 in /docs in the npm_and_yarn group across 1 directory (#3366)\n\nbuild(deps): Bump dompurify\n\nBumps the npm_and_yarn group with 1 update in the /docs directory: [dompurify](https://github.com/cure53/DOMPurify).\n\n\nUpdates `dompurify` from 3.4.0 to 3.4.11\n- [Release notes](https://github.com/cure53/DOMPurify/releases)\n- [Commits](https://github.com/cure53/DOMPurify/compare/3.4.0...3.4.11)\n\n---\nupdated-dependencies:\n- dependency-name: dompurify\n  dependency-version: 3.4.11\n  dependency-type: indirect\n  dependency-group: npm_and_yarn\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-06-29T12:02:17+02:00",
          "tree_id": "8d16df760af045f500d1ac88648f9edae597b8d5",
          "url": "https://github.com/evstack/ev-node/commit/69c655c80bab7f129fe6f3ad4cd27e48bdbcc603"
        },
        "date": 1782727576680,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 38274,
            "unit": "ns/op\t    5079 B/op\t      55 allocs/op",
            "extra": "31792 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 38274,
            "unit": "ns/op",
            "extra": "31792 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 5079,
            "unit": "B/op",
            "extra": "31792 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "31792 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 44718,
            "unit": "ns/op\t   10384 B/op\t      55 allocs/op",
            "extra": "27130 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 44718,
            "unit": "ns/op",
            "extra": "27130 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10384,
            "unit": "B/op",
            "extra": "27130 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "27130 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 38236,
            "unit": "ns/op\t    4879 B/op\t      51 allocs/op",
            "extra": "32136 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 38236,
            "unit": "ns/op",
            "extra": "32136 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4879,
            "unit": "B/op",
            "extra": "32136 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 51,
            "unit": "allocs/op",
            "extra": "32136 times\n4 procs"
          }
        ]
      }
    ]
  }
}