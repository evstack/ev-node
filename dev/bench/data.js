window.BENCHMARK_DATA = {
  "lastUpdate": 1784122794719,
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
          "id": "e437a6243362bb8ce95d252bc1c361596cf770e2",
          "message": "chore: prep for Amsterdam fork (#3352)\n\n* update reth 2.3 and prep for amsterdam fork\n\n* add test\n\n* updates\n\n* updates\n\n* dep updates\n\n* dep updates\n\n* test with pr-266",
          "timestamp": "2026-07-15T15:35:38+02:00",
          "tree_id": "f9df3037f55e8c1faa8bd37f7aa390db2083e551",
          "url": "https://github.com/evstack/ev-node/commit/e437a6243362bb8ce95d252bc1c361596cf770e2"
        },
        "date": 1784122788275,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 903557416,
            "unit": "ns/op\t32048412 B/op\t  179710 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 903557416,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 32048412,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 179710,
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
          "id": "e437a6243362bb8ce95d252bc1c361596cf770e2",
          "message": "chore: prep for Amsterdam fork (#3352)\n\n* update reth 2.3 and prep for amsterdam fork\n\n* add test\n\n* updates\n\n* updates\n\n* dep updates\n\n* dep updates\n\n* test with pr-266",
          "timestamp": "2026-07-15T15:35:38+02:00",
          "tree_id": "f9df3037f55e8c1faa8bd37f7aa390db2083e551",
          "url": "https://github.com/evstack/ev-node/commit/e437a6243362bb8ce95d252bc1c361596cf770e2"
        },
        "date": 1784122794292,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37453,
            "unit": "ns/op\t    4879 B/op\t      51 allocs/op",
            "extra": "32125 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37453,
            "unit": "ns/op",
            "extra": "32125 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4879,
            "unit": "B/op",
            "extra": "32125 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 51,
            "unit": "allocs/op",
            "extra": "32125 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 38206,
            "unit": "ns/op\t    5066 B/op\t      55 allocs/op",
            "extra": "32329 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 38206,
            "unit": "ns/op",
            "extra": "32329 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 5066,
            "unit": "B/op",
            "extra": "32329 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "32329 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 44795,
            "unit": "ns/op\t   10374 B/op\t      55 allocs/op",
            "extra": "27427 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 44795,
            "unit": "ns/op",
            "extra": "27427 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10374,
            "unit": "B/op",
            "extra": "27427 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "27427 times\n4 procs"
          }
        ]
      }
    ]
  }
}