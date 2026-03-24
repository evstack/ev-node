window.BENCHMARK_DATA = {
  "lastUpdate": 1774382499411,
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
          "id": "10231090ddc800aa11a186dfff3850184522f780",
          "message": "refactor: display block source in sync log (#3193)\n\n* refactor: display block source in sync log\n\n* add cl",
          "timestamp": "2026-03-24T11:56:03+01:00",
          "tree_id": "cdd49fb4b5a37b457de0f6029a7b082e24979149",
          "url": "https://github.com/evstack/ev-node/commit/10231090ddc800aa11a186dfff3850184522f780"
        },
        "date": 1774351238324,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 907814890,
            "unit": "ns/op\t32450512 B/op\t  181068 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 907814890,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 32450512,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 181068,
            "unit": "allocs/op",
            "extra": "2 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "49699333+dependabot[bot]@users.noreply.github.com",
            "name": "dependabot[bot]",
            "username": "dependabot[bot]"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "459f3c9ff350fc99b3f221d5c9bb870ea2ba4766",
          "message": "build(deps): Bump the all-go group across 1 directory with 2 updates (#3191)\n\n* build(deps): Bump the all-go group across 1 directory with 2 updates\n\nBumps the all-go group with 2 updates in the / directory: [github.com/go-viper/mapstructure/v2](https://github.com/go-viper/mapstructure) and [github.com/libp2p/go-libp2p-kad-dht](https://github.com/libp2p/go-libp2p-kad-dht).\n\n\nUpdates `github.com/go-viper/mapstructure/v2` from 2.4.0 to 2.5.0\n- [Release notes](https://github.com/go-viper/mapstructure/releases)\n- [Changelog](https://github.com/go-viper/mapstructure/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/go-viper/mapstructure/compare/v2.4.0...v2.5.0)\n\nUpdates `github.com/libp2p/go-libp2p-kad-dht` from 0.38.0 to 0.39.0\n- [Release notes](https://github.com/libp2p/go-libp2p-kad-dht/releases)\n- [Commits](https://github.com/libp2p/go-libp2p-kad-dht/compare/v0.38.0...v0.39.0)\n\n---\nupdated-dependencies:\n- dependency-name: github.com/go-viper/mapstructure/v2\n  dependency-version: 2.5.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/libp2p/go-libp2p-kad-dht\n  dependency-version: 0.39.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\n\n* chore: run just deps after Dependabot update\n\n* trigger build\n\n---------\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>\nCo-authored-by: github-actions[bot] <41898282+github-actions[bot]@users.noreply.github.com>\nCo-authored-by: Julien Robert <julien@rbrt.fr>",
          "timestamp": "2026-03-24T14:31:41+01:00",
          "tree_id": "510175252cb14972b0f92359b664535f46a27016",
          "url": "https://github.com/evstack/ev-node/commit/459f3c9ff350fc99b3f221d5c9bb870ea2ba4766"
        },
        "date": 1774359321689,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 911271086,
            "unit": "ns/op\t32457680 B/op\t  181055 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 911271086,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 32457680,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 181055,
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
          "id": "1b73b887050a376d10ab7a9437d0f6ba9a1f96c2",
          "message": "chore: demote warn log as debug (#3198)",
          "timestamp": "2026-03-24T17:16:02+01:00",
          "tree_id": "0b2c844c05b6d3104a8f0fcc27822fd6d901ffb9",
          "url": "https://github.com/evstack/ev-node/commit/1b73b887050a376d10ab7a9437d0f6ba9a1f96c2"
        },
        "date": 1774369511121,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 897224536,
            "unit": "ns/op\t32058088 B/op\t  180356 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 897224536,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 32058088,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 180356,
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
          "id": "9461b328df3873c32ee05cf364299dfb59ce5d56",
          "message": "fix(pkg/da): fix json rpc types (#3200)",
          "timestamp": "2026-03-24T20:58:06+01:00",
          "tree_id": "a3c797b781090b02c502881ad8e645833b9d8937",
          "url": "https://github.com/evstack/ev-node/commit/9461b328df3873c32ee05cf364299dfb59ce5d56"
        },
        "date": 1774382483495,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 908664887,
            "unit": "ns/op\t32376220 B/op\t  181031 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 908664887,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 32376220,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 181031,
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
        "date": 1774351046133,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 47825,
            "unit": "ns/op\t   26130 B/op\t      81 allocs/op",
            "extra": "25737 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 47825,
            "unit": "ns/op",
            "extra": "25737 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26130,
            "unit": "B/op",
            "extra": "25737 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "25737 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37797,
            "unit": "ns/op\t    7005 B/op\t      71 allocs/op",
            "extra": "32210 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37797,
            "unit": "ns/op",
            "extra": "32210 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7005,
            "unit": "B/op",
            "extra": "32210 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "32210 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 38289,
            "unit": "ns/op\t    7449 B/op\t      81 allocs/op",
            "extra": "32052 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 38289,
            "unit": "ns/op",
            "extra": "32052 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7449,
            "unit": "B/op",
            "extra": "32052 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "32052 times\n4 procs"
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
          "id": "10231090ddc800aa11a186dfff3850184522f780",
          "message": "refactor: display block source in sync log (#3193)\n\n* refactor: display block source in sync log\n\n* add cl",
          "timestamp": "2026-03-24T11:56:03+01:00",
          "tree_id": "cdd49fb4b5a37b457de0f6029a7b082e24979149",
          "url": "https://github.com/evstack/ev-node/commit/10231090ddc800aa11a186dfff3850184522f780"
        },
        "date": 1774351245276,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37460,
            "unit": "ns/op\t    6990 B/op\t      71 allocs/op",
            "extra": "32856 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37460,
            "unit": "ns/op",
            "extra": "32856 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 6990,
            "unit": "B/op",
            "extra": "32856 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "32856 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 38584,
            "unit": "ns/op\t    7452 B/op\t      81 allocs/op",
            "extra": "31935 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 38584,
            "unit": "ns/op",
            "extra": "31935 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7452,
            "unit": "B/op",
            "extra": "31935 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "31935 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 47871,
            "unit": "ns/op\t   26138 B/op\t      81 allocs/op",
            "extra": "25518 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 47871,
            "unit": "ns/op",
            "extra": "25518 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26138,
            "unit": "B/op",
            "extra": "25518 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "25518 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "49699333+dependabot[bot]@users.noreply.github.com",
            "name": "dependabot[bot]",
            "username": "dependabot[bot]"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "459f3c9ff350fc99b3f221d5c9bb870ea2ba4766",
          "message": "build(deps): Bump the all-go group across 1 directory with 2 updates (#3191)\n\n* build(deps): Bump the all-go group across 1 directory with 2 updates\n\nBumps the all-go group with 2 updates in the / directory: [github.com/go-viper/mapstructure/v2](https://github.com/go-viper/mapstructure) and [github.com/libp2p/go-libp2p-kad-dht](https://github.com/libp2p/go-libp2p-kad-dht).\n\n\nUpdates `github.com/go-viper/mapstructure/v2` from 2.4.0 to 2.5.0\n- [Release notes](https://github.com/go-viper/mapstructure/releases)\n- [Changelog](https://github.com/go-viper/mapstructure/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/go-viper/mapstructure/compare/v2.4.0...v2.5.0)\n\nUpdates `github.com/libp2p/go-libp2p-kad-dht` from 0.38.0 to 0.39.0\n- [Release notes](https://github.com/libp2p/go-libp2p-kad-dht/releases)\n- [Commits](https://github.com/libp2p/go-libp2p-kad-dht/compare/v0.38.0...v0.39.0)\n\n---\nupdated-dependencies:\n- dependency-name: github.com/go-viper/mapstructure/v2\n  dependency-version: 2.5.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/libp2p/go-libp2p-kad-dht\n  dependency-version: 0.39.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\n\n* chore: run just deps after Dependabot update\n\n* trigger build\n\n---------\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>\nCo-authored-by: github-actions[bot] <41898282+github-actions[bot]@users.noreply.github.com>\nCo-authored-by: Julien Robert <julien@rbrt.fr>",
          "timestamp": "2026-03-24T14:31:41+01:00",
          "tree_id": "510175252cb14972b0f92359b664535f46a27016",
          "url": "https://github.com/evstack/ev-node/commit/459f3c9ff350fc99b3f221d5c9bb870ea2ba4766"
        },
        "date": 1774359326428,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 49116,
            "unit": "ns/op\t   26162 B/op\t      81 allocs/op",
            "extra": "24724 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 49116,
            "unit": "ns/op",
            "extra": "24724 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26162,
            "unit": "B/op",
            "extra": "24724 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "24724 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 38545,
            "unit": "ns/op\t    7025 B/op\t      71 allocs/op",
            "extra": "31412 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 38545,
            "unit": "ns/op",
            "extra": "31412 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7025,
            "unit": "B/op",
            "extra": "31412 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "31412 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 39626,
            "unit": "ns/op\t    7483 B/op\t      81 allocs/op",
            "extra": "30726 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 39626,
            "unit": "ns/op",
            "extra": "30726 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7483,
            "unit": "B/op",
            "extra": "30726 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "30726 times\n4 procs"
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
          "id": "1b73b887050a376d10ab7a9437d0f6ba9a1f96c2",
          "message": "chore: demote warn log as debug (#3198)",
          "timestamp": "2026-03-24T17:16:02+01:00",
          "tree_id": "0b2c844c05b6d3104a8f0fcc27822fd6d901ffb9",
          "url": "https://github.com/evstack/ev-node/commit/1b73b887050a376d10ab7a9437d0f6ba9a1f96c2"
        },
        "date": 1774369516121,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 38419,
            "unit": "ns/op\t    7455 B/op\t      81 allocs/op",
            "extra": "31828 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 38419,
            "unit": "ns/op",
            "extra": "31828 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7455,
            "unit": "B/op",
            "extra": "31828 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "31828 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 48252,
            "unit": "ns/op\t   26158 B/op\t      81 allocs/op",
            "extra": "24903 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 48252,
            "unit": "ns/op",
            "extra": "24903 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26158,
            "unit": "B/op",
            "extra": "24903 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "24903 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 38675,
            "unit": "ns/op\t    7008 B/op\t      71 allocs/op",
            "extra": "32089 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 38675,
            "unit": "ns/op",
            "extra": "32089 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7008,
            "unit": "B/op",
            "extra": "32089 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "32089 times\n4 procs"
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
          "id": "9461b328df3873c32ee05cf364299dfb59ce5d56",
          "message": "fix(pkg/da): fix json rpc types (#3200)",
          "timestamp": "2026-03-24T20:58:06+01:00",
          "tree_id": "a3c797b781090b02c502881ad8e645833b9d8937",
          "url": "https://github.com/evstack/ev-node/commit/9461b328df3873c32ee05cf364299dfb59ce5d56"
        },
        "date": 1774382498912,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 36586,
            "unit": "ns/op\t    6985 B/op\t      71 allocs/op",
            "extra": "33039 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 36586,
            "unit": "ns/op",
            "extra": "33039 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 6985,
            "unit": "B/op",
            "extra": "33039 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "33039 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 37427,
            "unit": "ns/op\t    7442 B/op\t      81 allocs/op",
            "extra": "32341 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 37427,
            "unit": "ns/op",
            "extra": "32341 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7442,
            "unit": "B/op",
            "extra": "32341 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "32341 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 47398,
            "unit": "ns/op\t   26117 B/op\t      81 allocs/op",
            "extra": "26084 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 47398,
            "unit": "ns/op",
            "extra": "26084 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26117,
            "unit": "B/op",
            "extra": "26084 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "26084 times\n4 procs"
          }
        ]
      }
    ]
  }
}