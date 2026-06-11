window.BENCHMARK_DATA = {
  "lastUpdate": 1781176176602,
  "repoUrl": "https://github.com/evstack/ev-node",
  "entries": {
    "EVM Contract Roundtrip": [
      {
        "commit": {
          "author": {
            "email": "nuke-web3@proton.me",
            "name": "Nuke 🌄",
            "username": "nuke-web3"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "74efee38260cd5072000a3a1ef349ae6a408fe1a",
          "message": "Add locad devnet guide (#3350)\n\n* oneshot local devnet guide w/ claude\n\n* tested to work with docker, but really upstream tooling needs some tweaks",
          "timestamp": "2026-06-11T13:01:13+02:00",
          "tree_id": "bb7035bd753c9e0fd8874d4a84d0c8025bebd379",
          "url": "https://github.com/evstack/ev-node/commit/74efee38260cd5072000a3a1ef349ae6a408fe1a"
        },
        "date": 1781176172377,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 895622446,
            "unit": "ns/op\t30641340 B/op\t  165486 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 895622446,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 30641340,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 165486,
            "unit": "allocs/op",
            "extra": "2 times\n4 procs"
          }
        ]
      }
    ]
  }
}