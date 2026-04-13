window.BENCHMARK_DATA = {
  "lastUpdate": 1776074952290,
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
          "distinct": false,
          "id": "a9e08e8f1477aed0aafbd2c0735affad8a829a07",
          "message": "chore: docs restyle for new landing page (#3210)\n\ndocs restyle for new landing page",
          "timestamp": "2026-04-13T11:51:49+02:00",
          "tree_id": "ecf1aff3a8794a92e55f778917461ec2c2d82837",
          "url": "https://github.com/evstack/ev-node/commit/a9e08e8f1477aed0aafbd2c0735affad8a829a07"
        },
        "date": 1776074947230,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 916119372,
            "unit": "ns/op\t32017116 B/op\t  176321 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 916119372,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 32017116,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 176321,
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
          "distinct": false,
          "id": "a9e08e8f1477aed0aafbd2c0735affad8a829a07",
          "message": "chore: docs restyle for new landing page (#3210)\n\ndocs restyle for new landing page",
          "timestamp": "2026-04-13T11:51:49+02:00",
          "tree_id": "ecf1aff3a8794a92e55f778917461ec2c2d82837",
          "url": "https://github.com/evstack/ev-node/commit/a9e08e8f1477aed0aafbd2c0735affad8a829a07"
        },
        "date": 1776074951891,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 40199,
            "unit": "ns/op\t    7114 B/op\t      71 allocs/op",
            "extra": "30444 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 40199,
            "unit": "ns/op",
            "extra": "30444 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7114,
            "unit": "B/op",
            "extra": "30444 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "30444 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 40816,
            "unit": "ns/op\t    7570 B/op\t      81 allocs/op",
            "extra": "29901 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 40816,
            "unit": "ns/op",
            "extra": "29901 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7570,
            "unit": "B/op",
            "extra": "29901 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "29901 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 50257,
            "unit": "ns/op\t   26230 B/op\t      81 allocs/op",
            "extra": "24238 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 50257,
            "unit": "ns/op",
            "extra": "24238 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26230,
            "unit": "B/op",
            "extra": "24238 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "24238 times\n4 procs"
          }
        ]
      }
    ]
  }
}