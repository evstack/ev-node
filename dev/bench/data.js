window.BENCHMARK_DATA = {
  "lastUpdate": 1774456341849,
  "repoUrl": "https://github.com/evstack/ev-node",
  "entries": {
    "EVM Contract Roundtrip": [
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
          "id": "4c8757c63937dcf481b348de01689f9dbe9a3407",
          "message": "feat:  Add KMS signer backend (#3171)\n\n* Start ksm\n\n* Extend kms\n\n* Improve kms\n\n* Better doc, config and tests\n\n* Linting only\n\n* Changelog\n\n* Add remote signer GCO KMS (#3182)\n\n* Add remote signer GCO KMS\n\n* Review feedback\n\n* Minor updates\n\n* Review feedback\n\n* Minor udpates\n\n* Review feedback\n\n* Linter\n\n* Update changelog",
          "timestamp": "2026-03-25T16:11:37Z",
          "tree_id": "85d4923f57acc94725012d88b24c0413b7b622c2",
          "url": "https://github.com/evstack/ev-node/commit/4c8757c63937dcf481b348de01689f9dbe9a3407"
        },
        "date": 1774456337285,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 898816088,
            "unit": "ns/op\t29681864 B/op\t  155389 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 898816088,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 29681864,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 155389,
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
          "id": "4c8757c63937dcf481b348de01689f9dbe9a3407",
          "message": "feat:  Add KMS signer backend (#3171)\n\n* Start ksm\n\n* Extend kms\n\n* Improve kms\n\n* Better doc, config and tests\n\n* Linting only\n\n* Changelog\n\n* Add remote signer GCO KMS (#3182)\n\n* Add remote signer GCO KMS\n\n* Review feedback\n\n* Minor updates\n\n* Review feedback\n\n* Minor udpates\n\n* Review feedback\n\n* Linter\n\n* Update changelog",
          "timestamp": "2026-03-25T16:11:37Z",
          "tree_id": "85d4923f57acc94725012d88b24c0413b7b622c2",
          "url": "https://github.com/evstack/ev-node/commit/4c8757c63937dcf481b348de01689f9dbe9a3407"
        },
        "date": 1774456341468,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 38403,
            "unit": "ns/op\t    7006 B/op\t      71 allocs/op",
            "extra": "32178 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 38403,
            "unit": "ns/op",
            "extra": "32178 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7006,
            "unit": "B/op",
            "extra": "32178 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "32178 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 39072,
            "unit": "ns/op\t    7480 B/op\t      81 allocs/op",
            "extra": "30836 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 39072,
            "unit": "ns/op",
            "extra": "30836 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7480,
            "unit": "B/op",
            "extra": "30836 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "30836 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 52408,
            "unit": "ns/op\t   26160 B/op\t      81 allocs/op",
            "extra": "24858 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 52408,
            "unit": "ns/op",
            "extra": "24858 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26160,
            "unit": "B/op",
            "extra": "24858 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "24858 times\n4 procs"
          }
        ]
      }
    ]
  }
}