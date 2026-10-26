window.BENCHMARK_DATA = {
  "lastUpdate": 1793007570371,
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
          "id": "69edce1af8e3e1c111ef5787aed7cf40e757d671",
          "message": "chore: prepare v1.3.0 release (#3496)",
          "timestamp": "2026-10-26T10:33:16+01:00",
          "tree_id": "7941b2ae6d7b7004293d744d3e2d17dcf3dae2de",
          "url": "https://github.com/evstack/ev-node/commit/69edce1af8e3e1c111ef5787aed7cf40e757d671"
        },
        "date": 1793007561490,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 880486518,
            "unit": "ns/op\t 4109560 B/op\t   35339 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 880486518,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 4109560,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 35339,
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
          "id": "69edce1af8e3e1c111ef5787aed7cf40e757d671",
          "message": "chore: prepare v1.3.0 release (#3496)",
          "timestamp": "2026-10-26T10:33:16+01:00",
          "tree_id": "7941b2ae6d7b7004293d744d3e2d17dcf3dae2de",
          "url": "https://github.com/evstack/ev-node/commit/69edce1af8e3e1c111ef5787aed7cf40e757d671"
        },
        "date": 1793007569671,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 45504,
            "unit": "ns/op\t   10398 B/op\t      55 allocs/op",
            "extra": "26736 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 45504,
            "unit": "ns/op",
            "extra": "26736 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10398,
            "unit": "B/op",
            "extra": "26736 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "26736 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 41048,
            "unit": "ns/op\t    4905 B/op\t      51 allocs/op",
            "extra": "31096 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 41048,
            "unit": "ns/op",
            "extra": "31096 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4905,
            "unit": "B/op",
            "extra": "31096 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 51,
            "unit": "allocs/op",
            "extra": "31096 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 39586,
            "unit": "ns/op\t    5109 B/op\t      55 allocs/op",
            "extra": "30636 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 39586,
            "unit": "ns/op",
            "extra": "30636 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 5109,
            "unit": "B/op",
            "extra": "30636 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "30636 times\n4 procs"
          }
        ]
      }
    ]
  }
}