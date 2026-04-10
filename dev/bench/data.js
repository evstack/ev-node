window.BENCHMARK_DATA = {
  "lastUpdate": 1775831665323,
  "repoUrl": "https://github.com/evstack/ev-node",
  "entries": {
    "EVM Contract Roundtrip": [
      {
        "commit": {
          "author": {
            "email": "julien@rbrt.fr",
            "name": "julienrbrt",
            "username": "julienrbrt"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "6f8e09383a150fa712cb659ceb571a57064009f9",
          "message": "refactor: reaper to drain mempool (#3236)\n\n* refactor: reaper to drain mempool\n\n* feedback\n\n* fix partial drain\n\n* cleanup old readme\n\n* Prevent multiple Start() calls across components\n\n* fix unwanted log\n\n* lock\n\n* updates\n\n* remove redundant\n\n* changelog\n\n* feedback\n\n* feedback\n\n* feedback\n\n* add bench",
          "timestamp": "2026-04-10T14:15:00Z",
          "tree_id": "b398e4dfaca8c3f6ff712b4434128de20d53ff47",
          "url": "https://github.com/evstack/ev-node/commit/6f8e09383a150fa712cb659ceb571a57064009f9"
        },
        "date": 1775831659220,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 919823078,
            "unit": "ns/op\t33552200 B/op\t  191049 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 919823078,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 33552200,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 191049,
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
            "email": "julien@rbrt.fr",
            "name": "julienrbrt",
            "username": "julienrbrt"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "6f8e09383a150fa712cb659ceb571a57064009f9",
          "message": "refactor: reaper to drain mempool (#3236)\n\n* refactor: reaper to drain mempool\n\n* feedback\n\n* fix partial drain\n\n* cleanup old readme\n\n* Prevent multiple Start() calls across components\n\n* fix unwanted log\n\n* lock\n\n* updates\n\n* remove redundant\n\n* changelog\n\n* feedback\n\n* feedback\n\n* feedback\n\n* add bench",
          "timestamp": "2026-04-10T14:15:00Z",
          "tree_id": "b398e4dfaca8c3f6ff712b4434128de20d53ff47",
          "url": "https://github.com/evstack/ev-node/commit/6f8e09383a150fa712cb659ceb571a57064009f9"
        },
        "date": 1775831664677,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 35609,
            "unit": "ns/op\t    7027 B/op\t      71 allocs/op",
            "extra": "34047 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 35609,
            "unit": "ns/op",
            "extra": "34047 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7027,
            "unit": "B/op",
            "extra": "34047 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "34047 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 36129,
            "unit": "ns/op\t    7486 B/op\t      81 allocs/op",
            "extra": "33210 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 36129,
            "unit": "ns/op",
            "extra": "33210 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7486,
            "unit": "B/op",
            "extra": "33210 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "33210 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 46474,
            "unit": "ns/op\t   26177 B/op\t      81 allocs/op",
            "extra": "26173 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 46474,
            "unit": "ns/op",
            "extra": "26173 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26177,
            "unit": "B/op",
            "extra": "26173 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "26173 times\n4 procs"
          }
        ]
      }
    ]
  }
}