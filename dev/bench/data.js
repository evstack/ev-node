window.BENCHMARK_DATA = {
  "lastUpdate": 1779646994325,
  "repoUrl": "https://github.com/evstack/ev-node",
  "entries": {
    "EVM Contract Roundtrip": [
      {
        "commit": {
          "author": {
            "email": "caltechustc@outlook.com",
            "name": "caltechustc",
            "username": "caltechustc"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "5ee6483aa360fab4785bb5fe820a1b5a01e8ecf2",
          "message": "chore: remove duplicate package import (#3331)\n\nchore: remove duplicate package import and fix some CI issues\n\nSigned-off-by: caltechustc <caltechustc@outlook.com>",
          "timestamp": "2026-05-24T18:03:26Z",
          "tree_id": "2dc6474b4cd0dc53fa8ca1155e12691b9e7cbb40",
          "url": "https://github.com/evstack/ev-node/commit/5ee6483aa360fab4785bb5fe820a1b5a01e8ecf2"
        },
        "date": 1779646986805,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 916781524,
            "unit": "ns/op\t33067568 B/op\t  186084 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 916781524,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 33067568,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 186084,
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
            "email": "caltechustc@outlook.com",
            "name": "caltechustc",
            "username": "caltechustc"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "5ee6483aa360fab4785bb5fe820a1b5a01e8ecf2",
          "message": "chore: remove duplicate package import (#3331)\n\nchore: remove duplicate package import and fix some CI issues\n\nSigned-off-by: caltechustc <caltechustc@outlook.com>",
          "timestamp": "2026-05-24T18:03:26Z",
          "tree_id": "2dc6474b4cd0dc53fa8ca1155e12691b9e7cbb40",
          "url": "https://github.com/evstack/ev-node/commit/5ee6483aa360fab4785bb5fe820a1b5a01e8ecf2"
        },
        "date": 1779646993517,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 34523,
            "unit": "ns/op\t    4705 B/op\t      50 allocs/op",
            "extra": "34983 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 34523,
            "unit": "ns/op",
            "extra": "34983 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4705,
            "unit": "B/op",
            "extra": "34983 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 50,
            "unit": "allocs/op",
            "extra": "34983 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 35103,
            "unit": "ns/op\t    4907 B/op\t      54 allocs/op",
            "extra": "34531 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 35103,
            "unit": "ns/op",
            "extra": "34531 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 4907,
            "unit": "B/op",
            "extra": "34531 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "34531 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 40894,
            "unit": "ns/op\t   10202 B/op\t      54 allocs/op",
            "extra": "29430 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 40894,
            "unit": "ns/op",
            "extra": "29430 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10202,
            "unit": "B/op",
            "extra": "29430 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "29430 times\n4 procs"
          }
        ]
      }
    ]
  }
}