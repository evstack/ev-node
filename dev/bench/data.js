window.BENCHMARK_DATA = {
  "lastUpdate": 1780238696899,
  "repoUrl": "https://github.com/evstack/ev-node",
  "entries": {
    "EVM Contract Roundtrip": [
      {
        "commit": {
          "author": {
            "email": "marko@baricevic.me",
            "name": "tac0turtle",
            "username": "tac0turtle"
          },
          "committer": {
            "email": "marko@baricevic.me",
            "name": "tac0turtle",
            "username": "tac0turtle"
          },
          "distinct": true,
          "id": "59ed284dba3c87bdd0626d262676ca775b9d6e7f",
          "message": "add docs for new ev-reth changes",
          "timestamp": "2026-05-27T13:56:11+02:00",
          "tree_id": "6c4b8a285c527d1f86e885e344583c456c7e17e7",
          "url": "https://github.com/evstack/ev-node/commit/59ed284dba3c87bdd0626d262676ca775b9d6e7f"
        },
        "date": 1779883332971,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 914189630,
            "unit": "ns/op\t31550164 B/op\t  171388 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 914189630,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 31550164,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 171388,
            "unit": "allocs/op",
            "extra": "2 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "cricis@msn.com",
            "name": "criciss",
            "username": "criciss"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": false,
          "id": "6b2629ec76095dce9740520b1d98c23b0c15e4f5",
          "message": "chore: fix typo in comment (#3334)\n\nSigned-off-by: criciss <cricis@msn.com>",
          "timestamp": "2026-05-31T14:26:25Z",
          "tree_id": "2f7207c05bfa22bffac3325f8d337ba362fc0970",
          "url": "https://github.com/evstack/ev-node/commit/6b2629ec76095dce9740520b1d98c23b0c15e4f5"
        },
        "date": 1780238690642,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 926783202,
            "unit": "ns/op\t33938956 B/op\t  196079 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 926783202,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 33938956,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 196079,
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
            "email": "cricis@msn.com",
            "name": "criciss",
            "username": "criciss"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": false,
          "id": "6b2629ec76095dce9740520b1d98c23b0c15e4f5",
          "message": "chore: fix typo in comment (#3334)\n\nSigned-off-by: criciss <cricis@msn.com>",
          "timestamp": "2026-05-31T14:26:25Z",
          "tree_id": "2f7207c05bfa22bffac3325f8d337ba362fc0970",
          "url": "https://github.com/evstack/ev-node/commit/6b2629ec76095dce9740520b1d98c23b0c15e4f5"
        },
        "date": 1780238696290,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 31153,
            "unit": "ns/op\t    4622 B/op\t      50 allocs/op",
            "extra": "39552 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 31153,
            "unit": "ns/op",
            "extra": "39552 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4622,
            "unit": "B/op",
            "extra": "39552 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 50,
            "unit": "allocs/op",
            "extra": "39552 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 31242,
            "unit": "ns/op\t    4820 B/op\t      54 allocs/op",
            "extra": "39180 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 31242,
            "unit": "ns/op",
            "extra": "39180 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 4820,
            "unit": "B/op",
            "extra": "39180 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "39180 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 36336,
            "unit": "ns/op\t   10089 B/op\t      54 allocs/op",
            "extra": "33915 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 36336,
            "unit": "ns/op",
            "extra": "33915 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10089,
            "unit": "B/op",
            "extra": "33915 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "33915 times\n4 procs"
          }
        ]
      }
    ]
  }
}