window.BENCHMARK_DATA = {
  "lastUpdate": 1788270486580,
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
          "id": "cca63ca2a26cb19d24e9d81b113d3555fc6fbe3d",
          "message": "fix(node): fail closed during sequencer recovery (#3443)\n\n* fix(node): fail closed during sequencer recovery\n\n* fix(sync): retry P2P init throughout catchup recovery\n\nDo not abandon P2P initialization after the 30s Start timeout when\ncatchup recovery requires continuity. Keep retrying in the background\nso P2PInitialized can still flip during waitForCatchup, and include\nreadiness flags in the timeout error.",
          "timestamp": "2026-09-01T15:44:15+02:00",
          "tree_id": "a89aff0a7b0aa7649af35452f73427995b03f92a",
          "url": "https://github.com/evstack/ev-node/commit/cca63ca2a26cb19d24e9d81b113d3555fc6fbe3d"
        },
        "date": 1788270481708,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 895054383,
            "unit": "ns/op\t 4249828 B/op\t   35939 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 895054383,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 4249828,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 35939,
            "unit": "allocs/op",
            "extra": "2 times\n4 procs"
          }
        ]
      }
    ]
  }
}