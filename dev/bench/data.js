window.BENCHMARK_DATA = {
  "lastUpdate": 1775581762338,
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
      }
    ]
  }
}