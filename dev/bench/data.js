window.BENCHMARK_DATA = {
  "lastUpdate": 1771493142211,
  "repoUrl": "https://github.com/evstack/ev-node",
  "entries": {
    "EVM Contract Roundtrip": [
      {
        "commit": {
          "author": {
            "name": "evstack",
            "username": "evstack"
          },
          "committer": {
            "name": "evstack",
            "username": "evstack"
          },
          "id": "3b74695b3b8d389271a0c122f656a2a7199ef382",
          "message": "feat: introduce e2e benchmarking ",
          "timestamp": "2026-02-17T17:03:09Z",
          "url": "https://github.com/evstack/ev-node/pull/3078/commits/3b74695b3b8d389271a0c122f656a2a7199ef382"
        },
        "date": 1771408346118,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 877896807,
            "unit": "ns/op\t 1937448 B/op\t   11512 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 877896807,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 1937448,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 11512,
            "unit": "allocs/op",
            "extra": "2 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "evstack",
            "username": "evstack"
          },
          "committer": {
            "name": "evstack",
            "username": "evstack"
          },
          "id": "d2512dd0e01ccf461e59c94906aafd5397e92bb0",
          "message": "feat: introduce e2e benchmarking ",
          "timestamp": "2026-02-18T12:45:05Z",
          "url": "https://github.com/evstack/ev-node/pull/3078/commits/d2512dd0e01ccf461e59c94906aafd5397e92bb0"
        },
        "date": 1771421156072,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 878594158,
            "unit": "ns/op\t 1986828 B/op\t   11521 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 878594158,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 1986828,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 11521,
            "unit": "allocs/op",
            "extra": "2 times\n4 procs"
          }
        ]
      },
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
          "id": "dc473e56d221d42faac16e6db53a1319c255a7c8",
          "message": "feat: introduce e2e benchmarking  (#3078)\n\n* feat: introduce EVM contract benchmarking with new tests and a GitHub Actions workflow.\n\n* Capture otel\n\n* Go mod tidy\n\n* Enable benchmark on PRs - revert before merge\n\n* Push benchmark results on PR - revert later\n\n* Review feedback\n\n* Revert \"Push benchmark results on PR - revert later\"\n\nThis reverts commit 3b74695b3b8d389271a0c122f656a2a7199ef382.\n\n* Revert \"Enable benchmark on PRs - revert before merge\"\n\nThis reverts commit c7a5914df0b5400db5c26aa60bbb99437d745f9a.",
          "timestamp": "2026-02-19T09:06:22Z",
          "tree_id": "5cf2dbd500f1b330948f29327bbb66ec819f5dfe",
          "url": "https://github.com/evstack/ev-node/commit/dc473e56d221d42faac16e6db53a1319c255a7c8"
        },
        "date": 1771493138917,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 878664842,
            "unit": "ns/op\t 1972496 B/op\t   11531 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 878664842,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 1972496,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 11531,
            "unit": "allocs/op",
            "extra": "2 times\n4 procs"
          }
        ]
      }
    ]
  }
}