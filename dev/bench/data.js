window.BENCHMARK_DATA = {
  "lastUpdate": 1785143032478,
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
    ]
  }
}