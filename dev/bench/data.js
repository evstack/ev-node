window.BENCHMARK_DATA = {
  "lastUpdate": 1776420445507,
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
          "id": "4c7323fb87cee63dd9b3abfa5e4533d4c95cc5ff",
          "message": "refactor(pkg/p2p): swap GossipSub by FloodSub (#3263)\n\n* refactor(pkg/p2p): swap GossipSub by FloodSub\n\n* docs + cl",
          "timestamp": "2026-04-17T09:44:58Z",
          "tree_id": "2913f0e3c36409f9c190274761b6768dbdfd371c",
          "url": "https://github.com/evstack/ev-node/commit/4c7323fb87cee63dd9b3abfa5e4533d4c95cc5ff"
        },
        "date": 1776420441726,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 907931794,
            "unit": "ns/op\t32002580 B/op\t  176600 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 907931794,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 32002580,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 176600,
            "unit": "allocs/op",
            "extra": "2 times\n4 procs"
          }
        ]
      }
    ]
  }
}