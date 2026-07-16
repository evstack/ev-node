window.BENCHMARK_DATA = {
  "lastUpdate": 1784197899368,
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
          "id": "19d3a86e7567d76c7a75194eac561ad7da9ffe0a",
          "message": "feat: support p2p-only followers without DA (#3386)\n\n* support p2p-only followers without DA\n\n* require DA endpoint for aggregators\n\n* add DA configuration changelog entry\n\n* normalize DA endpoint at startup\n\n* scope DA endpoint validation to node startup",
          "timestamp": "2026-07-16T12:27:42+02:00",
          "tree_id": "ac236361fd51b929f63b21cbf567e026e4587d69",
          "url": "https://github.com/evstack/ev-node/commit/19d3a86e7567d76c7a75194eac561ad7da9ffe0a"
        },
        "date": 1784197893195,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 912978348,
            "unit": "ns/op\t31955952 B/op\t  178075 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 912978348,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 31955952,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 178075,
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
          "id": "19d3a86e7567d76c7a75194eac561ad7da9ffe0a",
          "message": "feat: support p2p-only followers without DA (#3386)\n\n* support p2p-only followers without DA\n\n* require DA endpoint for aggregators\n\n* add DA configuration changelog entry\n\n* normalize DA endpoint at startup\n\n* scope DA endpoint validation to node startup",
          "timestamp": "2026-07-16T12:27:42+02:00",
          "tree_id": "ac236361fd51b929f63b21cbf567e026e4587d69",
          "url": "https://github.com/evstack/ev-node/commit/19d3a86e7567d76c7a75194eac561ad7da9ffe0a"
        },
        "date": 1784197898846,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37952,
            "unit": "ns/op\t    4928 B/op\t      51 allocs/op",
            "extra": "30235 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37952,
            "unit": "ns/op",
            "extra": "30235 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4928,
            "unit": "B/op",
            "extra": "30235 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 51,
            "unit": "allocs/op",
            "extra": "30235 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 38108,
            "unit": "ns/op\t    5084 B/op\t      55 allocs/op",
            "extra": "31599 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 38108,
            "unit": "ns/op",
            "extra": "31599 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 5084,
            "unit": "B/op",
            "extra": "31599 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "31599 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 44657,
            "unit": "ns/op\t   10385 B/op\t      55 allocs/op",
            "extra": "27115 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 44657,
            "unit": "ns/op",
            "extra": "27115 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10385,
            "unit": "B/op",
            "extra": "27115 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "27115 times\n4 procs"
          }
        ]
      }
    ]
  }
}