window.BENCHMARK_DATA = {
  "lastUpdate": 1774689358917,
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
          "id": "47ed4ed91f3b0bf6021a43ff9f6e096f977ad5f7",
          "message": "fix(syncer): refetch latest da height instead of da height +1 (#3201)\n\n* fix(syncer): refetch latest da height instead of da height +1\n\n* wip\n\n* fixes\n\n* fix changelog and underflow\n\n* fix nil\n\n* fix unit tests\n\n* arrange cl\n\n* wip",
          "timestamp": "2026-03-26T13:37:11+01:00",
          "tree_id": "12fcfef6184aa63994e80fc5b9d6b70afebbcc2c",
          "url": "https://github.com/evstack/ev-node/commit/47ed4ed91f3b0bf6021a43ff9f6e096f977ad5f7"
        },
        "date": 1774528843219,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 904788858,
            "unit": "ns/op\t29869408 B/op\t  156834 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 904788858,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 29869408,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 156834,
            "unit": "allocs/op",
            "extra": "2 times\n4 procs"
          }
        ]
      },
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
          "distinct": true,
          "id": "162cda6ced7c806257d87d34e9d3aaaec9599e3f",
          "message": "fix: state pressure benchmark test (#3203)\n\nfix: increase default CountPerSpammer to prevent empty measurement window\n\nWith 2 spammers at 200 tx/s each, the previous default of 2000 txs per\nspammer meant all 4000 txs could complete during the 3s init sleep +\nwarmup phase, leaving the measurement window with only empty blocks.\nIncreasing to 5000 (10000 total) ensures enough txs remain after warmup.",
          "timestamp": "2026-03-26T13:42:33Z",
          "tree_id": "53a7fdc3d6e0fae78b15a1e5fca6467cd0287cd4",
          "url": "https://github.com/evstack/ev-node/commit/162cda6ced7c806257d87d34e9d3aaaec9599e3f"
        },
        "date": 1774533746001,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 897074570,
            "unit": "ns/op\t31112656 B/op\t  170715 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 897074570,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 31112656,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 170715,
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
          "id": "1c2309a05de6ab2ceeb7f1489df73b6edc79de3a",
          "message": "test: Add benchmarks for KMS signer (#3205)\n\nAdd kms signer benchmarks",
          "timestamp": "2026-03-28T10:12:24+01:00",
          "tree_id": "570fd7bf3b1b500e9a2352967d0e153447366f19",
          "url": "https://github.com/evstack/ev-node/commit/1c2309a05de6ab2ceeb7f1489df73b6edc79de3a"
        },
        "date": 1774689355919,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 916137419,
            "unit": "ns/op\t31703292 B/op\t  171531 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 916137419,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 31703292,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 171531,
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
          "id": "47ed4ed91f3b0bf6021a43ff9f6e096f977ad5f7",
          "message": "fix(syncer): refetch latest da height instead of da height +1 (#3201)\n\n* fix(syncer): refetch latest da height instead of da height +1\n\n* wip\n\n* fixes\n\n* fix changelog and underflow\n\n* fix nil\n\n* fix unit tests\n\n* arrange cl\n\n* wip",
          "timestamp": "2026-03-26T13:37:11+01:00",
          "tree_id": "12fcfef6184aa63994e80fc5b9d6b70afebbcc2c",
          "url": "https://github.com/evstack/ev-node/commit/47ed4ed91f3b0bf6021a43ff9f6e096f977ad5f7"
        },
        "date": 1774528847655,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 38606,
            "unit": "ns/op\t    7018 B/op\t      71 allocs/op",
            "extra": "31694 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 38606,
            "unit": "ns/op",
            "extra": "31694 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7018,
            "unit": "B/op",
            "extra": "31694 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "31694 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 39567,
            "unit": "ns/op\t    7476 B/op\t      81 allocs/op",
            "extra": "31000 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 39567,
            "unit": "ns/op",
            "extra": "31000 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7476,
            "unit": "B/op",
            "extra": "31000 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "31000 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 49727,
            "unit": "ns/op\t   26166 B/op\t      81 allocs/op",
            "extra": "24706 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 49727,
            "unit": "ns/op",
            "extra": "24706 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26166,
            "unit": "B/op",
            "extra": "24706 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "24706 times\n4 procs"
          }
        ]
      },
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
          "distinct": true,
          "id": "162cda6ced7c806257d87d34e9d3aaaec9599e3f",
          "message": "fix: state pressure benchmark test (#3203)\n\nfix: increase default CountPerSpammer to prevent empty measurement window\n\nWith 2 spammers at 200 tx/s each, the previous default of 2000 txs per\nspammer meant all 4000 txs could complete during the 3s init sleep +\nwarmup phase, leaving the measurement window with only empty blocks.\nIncreasing to 5000 (10000 total) ensures enough txs remain after warmup.",
          "timestamp": "2026-03-26T13:42:33Z",
          "tree_id": "53a7fdc3d6e0fae78b15a1e5fca6467cd0287cd4",
          "url": "https://github.com/evstack/ev-node/commit/162cda6ced7c806257d87d34e9d3aaaec9599e3f"
        },
        "date": 1774533751046,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37826,
            "unit": "ns/op\t    7001 B/op\t      71 allocs/op",
            "extra": "32388 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37826,
            "unit": "ns/op",
            "extra": "32388 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7001,
            "unit": "B/op",
            "extra": "32388 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "32388 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 38929,
            "unit": "ns/op\t    7453 B/op\t      81 allocs/op",
            "extra": "31878 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 38929,
            "unit": "ns/op",
            "extra": "31878 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7453,
            "unit": "B/op",
            "extra": "31878 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "31878 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 50045,
            "unit": "ns/op\t   26163 B/op\t      81 allocs/op",
            "extra": "24892 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 50045,
            "unit": "ns/op",
            "extra": "24892 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26163,
            "unit": "B/op",
            "extra": "24892 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "24892 times\n4 procs"
          }
        ]
      }
    ]
  }
}