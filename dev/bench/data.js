window.BENCHMARK_DATA = {
  "lastUpdate": 1774349557183,
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
      }
    ]
  }
}