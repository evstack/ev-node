window.BENCHMARK_DATA = {
  "lastUpdate": 1784037050761,
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
          "id": "34b278a7936d893a6f298d851918e69ff1d0c371",
          "message": "chore: remove ev-grpc (#3380)",
          "timestamp": "2026-07-10T16:07:08+02:00",
          "tree_id": "8eb1217b0cb96398f190dd499e134fd1ccda22f7",
          "url": "https://github.com/evstack/ev-node/commit/34b278a7936d893a6f298d851918e69ff1d0c371"
        },
        "date": 1783692526624,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 919321162,
            "unit": "ns/op\t31393060 B/op\t  170407 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 919321162,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 31393060,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 170407,
            "unit": "allocs/op",
            "extra": "2 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "weifanglab@outlook.com",
            "name": "weifanglab",
            "username": "weifanglab"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "dd9903384fc9ae628a9bb86559c476329bd0de59",
          "message": "chore: fix some comments to improve readability (#3382)\n\nSigned-off-by: weifanglab <weifanglab@outlook.com>",
          "timestamp": "2026-07-14T15:46:55+02:00",
          "tree_id": "906acdcf81cbba7f37fd7c04c3830ea2d2b17d35",
          "url": "https://github.com/evstack/ev-node/commit/dd9903384fc9ae628a9bb86559c476329bd0de59"
        },
        "date": 1784037046008,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 915849594,
            "unit": "ns/op\t32126512 B/op\t  180101 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 915849594,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 32126512,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 180101,
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
          "id": "34b278a7936d893a6f298d851918e69ff1d0c371",
          "message": "chore: remove ev-grpc (#3380)",
          "timestamp": "2026-07-10T16:07:08+02:00",
          "tree_id": "8eb1217b0cb96398f190dd499e134fd1ccda22f7",
          "url": "https://github.com/evstack/ev-node/commit/34b278a7936d893a6f298d851918e69ff1d0c371"
        },
        "date": 1783692535358,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37617,
            "unit": "ns/op\t    4870 B/op\t      51 allocs/op",
            "extra": "32508 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37617,
            "unit": "ns/op",
            "extra": "32508 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4870,
            "unit": "B/op",
            "extra": "32508 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 51,
            "unit": "allocs/op",
            "extra": "32508 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 38738,
            "unit": "ns/op\t    5083 B/op\t      55 allocs/op",
            "extra": "31660 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 38738,
            "unit": "ns/op",
            "extra": "31660 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 5083,
            "unit": "B/op",
            "extra": "31660 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "31660 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 44528,
            "unit": "ns/op\t   10377 B/op\t      55 allocs/op",
            "extra": "27344 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 44528,
            "unit": "ns/op",
            "extra": "27344 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10377,
            "unit": "B/op",
            "extra": "27344 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "27344 times\n4 procs"
          }
        ]
      }
    ]
  }
}