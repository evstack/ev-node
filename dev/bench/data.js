window.BENCHMARK_DATA = {
  "lastUpdate": 1775831663342,
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
    ]
  }
}