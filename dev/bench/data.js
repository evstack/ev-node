window.BENCHMARK_DATA = {
  "lastUpdate": 1785143035047,
  "repoUrl": "https://github.com/evstack/ev-node",
  "entries": {
    "EVM Contract Roundtrip": [
      {
        "commit": {
          "author": {
            "email": "jgimeno@gmail.com",
            "name": "Jonathan Gimeno",
            "username": "jgimeno"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "2a19748e8bad1d2422f4d54576b66096a002fe35",
          "message": "fix: skip DA cache restore for P2P-only nodes (#3408)",
          "timestamp": "2026-07-27T10:57:17+02:00",
          "tree_id": "e2f699b254b23684230d8c8e156675c2faf6408e",
          "url": "https://github.com/evstack/ev-node/commit/2a19748e8bad1d2422f4d54576b66096a002fe35"
        },
        "date": 1785143027718,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 919874624,
            "unit": "ns/op\t30378488 B/op\t  160390 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 919874624,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 30378488,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 160390,
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
            "email": "jgimeno@gmail.com",
            "name": "Jonathan Gimeno",
            "username": "jgimeno"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "2a19748e8bad1d2422f4d54576b66096a002fe35",
          "message": "fix: skip DA cache restore for P2P-only nodes (#3408)",
          "timestamp": "2026-07-27T10:57:17+02:00",
          "tree_id": "e2f699b254b23684230d8c8e156675c2faf6408e",
          "url": "https://github.com/evstack/ev-node/commit/2a19748e8bad1d2422f4d54576b66096a002fe35"
        },
        "date": 1785143034208,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 41053,
            "unit": "ns/op\t    5138 B/op\t      55 allocs/op",
            "extra": "29613 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 41053,
            "unit": "ns/op",
            "extra": "29613 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 5138,
            "unit": "B/op",
            "extra": "29613 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "29613 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 47416,
            "unit": "ns/op\t   10451 B/op\t      55 allocs/op",
            "extra": "25318 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 47416,
            "unit": "ns/op",
            "extra": "25318 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10451,
            "unit": "B/op",
            "extra": "25318 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "25318 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 40414,
            "unit": "ns/op\t    4942 B/op\t      51 allocs/op",
            "extra": "29733 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 40414,
            "unit": "ns/op",
            "extra": "29733 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4942,
            "unit": "B/op",
            "extra": "29733 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 51,
            "unit": "allocs/op",
            "extra": "29733 times\n4 procs"
          }
        ]
      }
    ]
  }
}