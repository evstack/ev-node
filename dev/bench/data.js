window.BENCHMARK_DATA = {
  "lastUpdate": 1774351045171,
  "repoUrl": "https://github.com/evstack/ev-node",
  "entries": {
    "EVM Contract Roundtrip": [
      {
        "commit": {
          "author": {
            "email": "github.qpeyb@simplelogin.fr",
            "name": "Cian Hatton",
            "username": "chatton"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": false,
          "id": "ad76f1c4c7375ec9268ace44c2dcb9126bac69bc",
          "message": "feat: add TestStatePressure benchmark for state trie stress test (#3188)\n\n* feat: add TestStatePressure benchmark for state trie stress test\n\n* refactor: use benchConfig for TestStatePressure spamoor parameters\n\n* ci: add state pressure benchmark to CI workflow\n\n* fix: address PR review feedback for TestStatePressure\n\n- use cfg.GasUnitsToBurn instead of re-reading env var with different default\n- add comment explaining time.Sleep before requireSpammersRunning\n- add post-run assertions for sent > 0 and failed == 0\n- add TODO comment for CI result publishing",
          "timestamp": "2026-03-24T10:29:02Z",
          "tree_id": "674cae654ad803ac72372d00511c57ba8d78f953",
          "url": "https://github.com/evstack/ev-node/commit/ad76f1c4c7375ec9268ace44c2dcb9126bac69bc"
        },
        "date": 1774349554197,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 905996692,
            "unit": "ns/op\t31071532 B/op\t  166742 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 905996692,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 31071532,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 166742,
            "unit": "allocs/op",
            "extra": "2 times\n4 procs"
          }
        ]
      },
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
          "id": "9d7b1c2c059b2d97691fbdb66a14dadd49ae5807",
          "message": "chore: remove cache analyzer (#3192)",
          "timestamp": "2026-03-24T10:39:16Z",
          "tree_id": "f052e9cc07e7387a68ee9da794564b9acf0bbb0a",
          "url": "https://github.com/evstack/ev-node/commit/9d7b1c2c059b2d97691fbdb66a14dadd49ae5807"
        },
        "date": 1774351041965,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 904073722,
            "unit": "ns/op\t31953604 B/op\t  175966 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 904073722,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 31953604,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 175966,
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
            "email": "github.qpeyb@simplelogin.fr",
            "name": "Cian Hatton",
            "username": "chatton"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": false,
          "id": "ad76f1c4c7375ec9268ace44c2dcb9126bac69bc",
          "message": "feat: add TestStatePressure benchmark for state trie stress test (#3188)\n\n* feat: add TestStatePressure benchmark for state trie stress test\n\n* refactor: use benchConfig for TestStatePressure spamoor parameters\n\n* ci: add state pressure benchmark to CI workflow\n\n* fix: address PR review feedback for TestStatePressure\n\n- use cfg.GasUnitsToBurn instead of re-reading env var with different default\n- add comment explaining time.Sleep before requireSpammersRunning\n- add post-run assertions for sent > 0 and failed == 0\n- add TODO comment for CI result publishing",
          "timestamp": "2026-03-24T10:29:02Z",
          "tree_id": "674cae654ad803ac72372d00511c57ba8d78f953",
          "url": "https://github.com/evstack/ev-node/commit/ad76f1c4c7375ec9268ace44c2dcb9126bac69bc"
        },
        "date": 1774349558172,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37606,
            "unit": "ns/op\t    6996 B/op\t      71 allocs/op",
            "extra": "32589 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37606,
            "unit": "ns/op",
            "extra": "32589 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 6996,
            "unit": "B/op",
            "extra": "32589 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "32589 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 38394,
            "unit": "ns/op\t    7464 B/op\t      81 allocs/op",
            "extra": "31450 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 38394,
            "unit": "ns/op",
            "extra": "31450 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7464,
            "unit": "B/op",
            "extra": "31450 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "31450 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 48988,
            "unit": "ns/op\t   26170 B/op\t      81 allocs/op",
            "extra": "24712 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 48988,
            "unit": "ns/op",
            "extra": "24712 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26170,
            "unit": "B/op",
            "extra": "24712 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "24712 times\n4 procs"
          }
        ]
      }
    ]
  }
}