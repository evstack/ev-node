window.BENCHMARK_DATA = {
  "lastUpdate": 1775582486206,
  "repoUrl": "https://github.com/evstack/ev-node",
  "entries": {
    "EVM Contract Roundtrip": [
      {
        "commit": {
          "author": {
            "email": "alpe@users.noreply.github.com",
            "name": "Alexander Peters",
            "username": "alpe"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "d163059fa575d9397eadb087ace737f828b84407",
          "message": "fix: Publisher-mode synchronization option for failover scenario (#3222)\n\n* Publisher-mode synchronization option for failover scenario\n\n* Changelog\n\n* Review feedback\n\n* Doc update\n\n* just tidy all\n\n---------\n\nCo-authored-by: julienrbrt <julien@rbrt.fr>",
          "timestamp": "2026-04-07T19:06:56+02:00",
          "tree_id": "f3b3558c56af0d01f8253e67e92dcea222cb7369",
          "url": "https://github.com/evstack/ev-node/commit/d163059fa575d9397eadb087ace737f828b84407"
        },
        "date": 1775581758189,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 905175094,
            "unit": "ns/op\t29919716 B/op\t  156557 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 905175094,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 29919716,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 156557,
            "unit": "allocs/op",
            "extra": "2 times\n4 procs"
          }
        ]
      },
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
          "id": "d2a29e86d5930d2f9d8ff5198e6ebea6255bc2fd",
          "message": "chore: prep rc.2 (#3231)\n\n* chore: prep rc.2\n\n* tidy",
          "timestamp": "2026-04-07T19:18:50+02:00",
          "tree_id": "5beadac939cc03bccb9a8319d8817c0838736605",
          "url": "https://github.com/evstack/ev-node/commit/d2a29e86d5930d2f9d8ff5198e6ebea6255bc2fd"
        },
        "date": 1775582482131,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 907965593,
            "unit": "ns/op\t31336016 B/op\t  171007 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 907965593,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 31336016,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 171007,
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
            "email": "alpe@users.noreply.github.com",
            "name": "Alexander Peters",
            "username": "alpe"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "d163059fa575d9397eadb087ace737f828b84407",
          "message": "fix: Publisher-mode synchronization option for failover scenario (#3222)\n\n* Publisher-mode synchronization option for failover scenario\n\n* Changelog\n\n* Review feedback\n\n* Doc update\n\n* just tidy all\n\n---------\n\nCo-authored-by: julienrbrt <julien@rbrt.fr>",
          "timestamp": "2026-04-07T19:06:56+02:00",
          "tree_id": "f3b3558c56af0d01f8253e67e92dcea222cb7369",
          "url": "https://github.com/evstack/ev-node/commit/d163059fa575d9397eadb087ace737f828b84407"
        },
        "date": 1775581763643,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 46983,
            "unit": "ns/op\t   26188 B/op\t      81 allocs/op",
            "extra": "25876 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 46983,
            "unit": "ns/op",
            "extra": "25876 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26188,
            "unit": "B/op",
            "extra": "25876 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "25876 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37110,
            "unit": "ns/op\t    7063 B/op\t      71 allocs/op",
            "extra": "32460 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37110,
            "unit": "ns/op",
            "extra": "32460 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7063,
            "unit": "B/op",
            "extra": "32460 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "32460 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 37547,
            "unit": "ns/op\t    7510 B/op\t      81 allocs/op",
            "extra": "32161 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 37547,
            "unit": "ns/op",
            "extra": "32161 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7510,
            "unit": "B/op",
            "extra": "32161 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "32161 times\n4 procs"
          }
        ]
      }
    ]
  }
}