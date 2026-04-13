window.BENCHMARK_DATA = {
  "lastUpdate": 1776077639533,
  "repoUrl": "https://github.com/evstack/ev-node",
  "entries": {
    "EVM Contract Roundtrip": [
      {
        "commit": {
          "author": {
            "email": "marko@baricevic.me",
            "name": "Marko",
            "username": "tac0turtle"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "cc540d53d9cab422ac069402f859a182d4f6bfd7",
          "message": "chore(docs): changes to url (#3242)\n\nchanges to url",
          "timestamp": "2026-04-13T10:34:40Z",
          "tree_id": "44d3b51fd58909a207e7d6061fb34d520099e0f2",
          "url": "https://github.com/evstack/ev-node/commit/cc540d53d9cab422ac069402f859a182d4f6bfd7"
        },
        "date": 1776077633457,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 901718222,
            "unit": "ns/op\t32213956 B/op\t  180700 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 901718222,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 32213956,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 180700,
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
            "email": "marko@baricevic.me",
            "name": "Marko",
            "username": "tac0turtle"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "cc540d53d9cab422ac069402f859a182d4f6bfd7",
          "message": "chore(docs): changes to url (#3242)\n\nchanges to url",
          "timestamp": "2026-04-13T10:34:40Z",
          "tree_id": "44d3b51fd58909a207e7d6061fb34d520099e0f2",
          "url": "https://github.com/evstack/ev-node/commit/cc540d53d9cab422ac069402f859a182d4f6bfd7"
        },
        "date": 1776077639005,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 38698,
            "unit": "ns/op\t    7055 B/op\t      71 allocs/op",
            "extra": "32791 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 38698,
            "unit": "ns/op",
            "extra": "32791 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7055,
            "unit": "B/op",
            "extra": "32791 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "32791 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 37857,
            "unit": "ns/op\t    7528 B/op\t      81 allocs/op",
            "extra": "31442 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 37857,
            "unit": "ns/op",
            "extra": "31442 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7528,
            "unit": "B/op",
            "extra": "31442 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "31442 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 47339,
            "unit": "ns/op\t   26187 B/op\t      81 allocs/op",
            "extra": "25900 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 47339,
            "unit": "ns/op",
            "extra": "25900 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26187,
            "unit": "B/op",
            "extra": "25900 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "25900 times\n4 procs"
          }
        ]
      }
    ]
  }
}