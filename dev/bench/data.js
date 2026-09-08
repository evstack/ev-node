window.BENCHMARK_DATA = {
  "lastUpdate": 1788854765461,
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
      },
      {
        "commit": {
          "author": {
            "email": "luangucun@outlook.com",
            "name": "luangucun",
            "username": "luangucun"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": false,
          "id": "8b13733cec7208a268192db65661781ae04eff1b",
          "message": "fix(store): remove canceled height waiters (#3445)\n\nSigned-off-by: luangucun <luangucun@outlook.com>",
          "timestamp": "2026-09-08T07:43:53Z",
          "tree_id": "9eb61585a73abae4924f9913ed10b0ef102ab6ae",
          "url": "https://github.com/evstack/ev-node/commit/8b13733cec7208a268192db65661781ae04eff1b"
        },
        "date": 1788854758640,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 915541246,
            "unit": "ns/op\t 4267464 B/op\t   36827 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 915541246,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 4267464,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 36827,
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
        "date": 1788270488677,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 35478,
            "unit": "ns/op\t    4841 B/op\t      51 allocs/op",
            "extra": "33741 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 35478,
            "unit": "ns/op",
            "extra": "33741 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4841,
            "unit": "B/op",
            "extra": "33741 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 51,
            "unit": "allocs/op",
            "extra": "33741 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 36101,
            "unit": "ns/op\t    5045 B/op\t      55 allocs/op",
            "extra": "33216 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 36101,
            "unit": "ns/op",
            "extra": "33216 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 5045,
            "unit": "B/op",
            "extra": "33216 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "33216 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 42687,
            "unit": "ns/op\t   10347 B/op\t      55 allocs/op",
            "extra": "28250 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 42687,
            "unit": "ns/op",
            "extra": "28250 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10347,
            "unit": "B/op",
            "extra": "28250 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "28250 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "luangucun@outlook.com",
            "name": "luangucun",
            "username": "luangucun"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": false,
          "id": "8b13733cec7208a268192db65661781ae04eff1b",
          "message": "fix(store): remove canceled height waiters (#3445)\n\nSigned-off-by: luangucun <luangucun@outlook.com>",
          "timestamp": "2026-09-08T07:43:53Z",
          "tree_id": "9eb61585a73abae4924f9913ed10b0ef102ab6ae",
          "url": "https://github.com/evstack/ev-node/commit/8b13733cec7208a268192db65661781ae04eff1b"
        },
        "date": 1788854764819,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 33476,
            "unit": "ns/op\t    4794 B/op\t      51 allocs/op",
            "extra": "36056 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 33476,
            "unit": "ns/op",
            "extra": "36056 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4794,
            "unit": "B/op",
            "extra": "36056 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 51,
            "unit": "allocs/op",
            "extra": "36056 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 33539,
            "unit": "ns/op\t    4991 B/op\t      55 allocs/op",
            "extra": "35811 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 33539,
            "unit": "ns/op",
            "extra": "35811 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 4991,
            "unit": "B/op",
            "extra": "35811 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "35811 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 40787,
            "unit": "ns/op\t   10305 B/op\t      55 allocs/op",
            "extra": "29676 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 40787,
            "unit": "ns/op",
            "extra": "29676 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10305,
            "unit": "B/op",
            "extra": "29676 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "29676 times\n4 procs"
          }
        ]
      }
    ]
  }
}