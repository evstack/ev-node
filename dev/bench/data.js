window.BENCHMARK_DATA = {
  "lastUpdate": 1776420447150,
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
          "id": "4c7323fb87cee63dd9b3abfa5e4533d4c95cc5ff",
          "message": "refactor(pkg/p2p): swap GossipSub by FloodSub (#3263)\n\n* refactor(pkg/p2p): swap GossipSub by FloodSub\n\n* docs + cl",
          "timestamp": "2026-04-17T09:44:58Z",
          "tree_id": "2913f0e3c36409f9c190274761b6768dbdfd371c",
          "url": "https://github.com/evstack/ev-node/commit/4c7323fb87cee63dd9b3abfa5e4533d4c95cc5ff"
        },
        "date": 1776420446711,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 36916,
            "unit": "ns/op\t    4754 B/op\t      50 allocs/op",
            "extra": "32746 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 36916,
            "unit": "ns/op",
            "extra": "32746 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4754,
            "unit": "B/op",
            "extra": "32746 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 50,
            "unit": "allocs/op",
            "extra": "32746 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 37383,
            "unit": "ns/op\t    4954 B/op\t      54 allocs/op",
            "extra": "32443 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 37383,
            "unit": "ns/op",
            "extra": "32443 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 4954,
            "unit": "B/op",
            "extra": "32443 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "32443 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 43889,
            "unit": "ns/op\t   10246 B/op\t      54 allocs/op",
            "extra": "27990 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 43889,
            "unit": "ns/op",
            "extra": "27990 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10246,
            "unit": "B/op",
            "extra": "27990 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "27990 times\n4 procs"
          }
        ]
      }
    ]
  }
}