window.BENCHMARK_DATA = {
  "lastUpdate": 1774884518051,
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
          "id": "763e8c6a802460241b68edda723185dcd14bbaff",
          "message": "refactor: remove unnecessary usage of lru (#3204)\n\n* fix(syncer): refetch latest da height instead of da height +1\n\n* wip\n\n* fixes\n\n* fix changelog and underflow\n\n* fix nil\n\n* fix unit tests\n\n* arrange cl\n\n* updates\n\n* fixes\n\n* fix\n\n* remove lru from generic cache as slow cleanup already happens\n\n* simplify\n\n* Update CHANGELOG.md",
          "timestamp": "2026-03-28T10:07:28Z",
          "tree_id": "0c7921ad92c9f2e4bddb33f8cba54eafb4a18b08",
          "url": "https://github.com/evstack/ev-node/commit/763e8c6a802460241b68edda723185dcd14bbaff"
        },
        "date": 1774693608857,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 911615133,
            "unit": "ns/op\t32571492 B/op\t  181570 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 911615133,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 32571492,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 181570,
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
          "id": "889da9a884f29812201635ea89de6d69350a986e",
          "message": "chore: missed feedback from #3204 (#3206)",
          "timestamp": "2026-03-30T10:08:17+02:00",
          "tree_id": "9fd3ba56baa2beb737075664da1b1b14190780e2",
          "url": "https://github.com/evstack/ev-node/commit/889da9a884f29812201635ea89de6d69350a986e"
        },
        "date": 1774858302914,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 889121134,
            "unit": "ns/op\t30653244 B/op\t  165447 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 889121134,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 30653244,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 165447,
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
          "id": "a4484f5a3d803a1cfd2597e65272b0cef427ad26",
          "message": "chore(syncing): avoid sending duplicate events to channel (#3207)\n\n* chore(syncing): avoid sending duplicate events to channel\n\n* add comment\n\n* improve comment",
          "timestamp": "2026-03-30T12:08:02Z",
          "tree_id": "61052d7c4b3ab80d29fd866424b0d4bfc2d72a27",
          "url": "https://github.com/evstack/ev-node/commit/a4484f5a3d803a1cfd2597e65272b0cef427ad26"
        },
        "date": 1774873769935,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 919033660,
            "unit": "ns/op\t33468204 B/op\t  191357 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 919033660,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 33468204,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 191357,
            "unit": "allocs/op",
            "extra": "2 times\n4 procs"
          }
        ]
      },
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
          "id": "146e6e125d7eb093e35eaa8cbc0254f272f18f15",
          "message": "chore: add badger constraints on index and block cache (#3209)\n\n* add sustainable badger configs\n\n* remove blockcache size\n\n* changelog",
          "timestamp": "2026-03-30T16:10:43+02:00",
          "tree_id": "e79c677b0c8ebee40dc4dd039cc3c4dfd4ccce41",
          "url": "https://github.com/evstack/ev-node/commit/146e6e125d7eb093e35eaa8cbc0254f272f18f15"
        },
        "date": 1774880574179,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 920444454,
            "unit": "ns/op\t32316792 B/op\t  176897 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 920444454,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 32316792,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 176897,
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
          "id": "5e3db23fb2fdc090312bbc39abd831c07a1a2ceb",
          "message": "feat: structured benchmark result output via BENCH_RESULT_OUTPUT (#3195)\n\n* feat(benchmarking): add structured result output via BENCH_RESULT_OUTPUT\n\nemit full benchmark run metadata (config, tags, metrics, block range,\nspamoor stats) as JSON when BENCH_RESULT_OUTPUT is set. consumed by\nexternal matrix runner for table generation.\n\n* fix: address PR review feedback for structured benchmark output\n\n- Deduplicate overhead/reth-rate computation: move stats-based helpers\n  to helpers.go, make span-based wrappers delegate to them\n- Fix sub-millisecond precision loss in engine span timings by using\n  microsecond-based float division instead of integer truncation\n- Add spamoor stats to TestGasBurner for consistency with other tests\n\n* refactor: make spamoor config fully configurable via BENCH_* env vars\n\n- Add MaxPending, Rebroadcast, BaseFee, TipFee to benchConfig\n- Fix ERC20 test hardcoding max_wallets=200 instead of using cfg\n- Replace all hardcoded spamoor params with cfg fields across tests\n\n* feat: extract host metadata from OTEL resource attributes in trace spans\n\n- Add resourceAttrs struct with host, OS, and service fields\n- Extract attributes from VictoriaTraces LogsQL span data via\n  resourceAttrCollector interface\n- Include host metadata in structured benchmark result JSON\n\n* fix: defer emitRunResult so results are written even on test failure\n\nMove emitRunResult into a deferred closure in all three test functions.\nIf the test fails after metrics are collected, the structured JSON is\nstill written. If it fails before result data exists, the defer is a\nno-op.\n\n* fix: state pressure benchmark CI failure and align with other tests\n\nRemove the 3-second sleep before requireSpammersRunning that caused all\ntransactions to be mined before the measurement window started, leaving\nSteadyState at 0s. Also add deferred emitRunResult, configurable spamoor\nparams, and spamoorStats collection to match the other benchmark tests.\n\n* fix: use deployment-level service names for trace queries in external mode\n\nIn external mode the sequencer reports spans as \"ev-node\" (not the\ntest-specific name like \"ev-node-erc20\"), so trace queries returned\nzero spans. Store service names on env: local mode uses the\ntest-specific name, external mode defaults to \"ev-node\"/\"ev-reth\"\nwith BENCH_EVNODE_SERVICE_NAME/BENCH_EVRETH_SERVICE_NAME overrides.\n\n* perf: use limit=1 for resource attribute trace queries\n\nfetchResourceAttrs only needs one span but was streaming the full\nresult set from VictoriaTraces. Add limit=1 to the LogsQL query to\navoid wasting bandwidth on long-lived instances with many spans.\n\n* docs: add missing doc comments to run_result.go functions",
          "timestamp": "2026-03-30T14:54:13Z",
          "tree_id": "55b09c16e7fe1a5e662b08495fdda99c8122bd5b",
          "url": "https://github.com/evstack/ev-node/commit/5e3db23fb2fdc090312bbc39abd831c07a1a2ceb"
        },
        "date": 1774883650981,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 898740693,
            "unit": "ns/op\t31855596 B/op\t  175757 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 898740693,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 31855596,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 175757,
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
          "id": "8d68f9d5c5f683bdfff1b1e0958182da70e9a371",
          "message": "fix(pkg/da): fallback to polling when ws cannot connect (#3211)\n\n* fix(pkg/da): fallback to polling when ws cannot connect\n\n* log err and fix typos\n\n* rewording",
          "timestamp": "2026-03-30T17:21:23+02:00",
          "tree_id": "cad9732a133fbed7b4725341d5a8f09fa5b6c048",
          "url": "https://github.com/evstack/ev-node/commit/8d68f9d5c5f683bdfff1b1e0958182da70e9a371"
        },
        "date": 1774884514880,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 915174771,
            "unit": "ns/op\t31627820 B/op\t  171843 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 915174771,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 31627820,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 171843,
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
        "date": 1774689360018,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 47422,
            "unit": "ns/op\t   26119 B/op\t      81 allocs/op",
            "extra": "26018 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 47422,
            "unit": "ns/op",
            "extra": "26018 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26119,
            "unit": "B/op",
            "extra": "26018 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "26018 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37312,
            "unit": "ns/op\t    6994 B/op\t      71 allocs/op",
            "extra": "32685 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37312,
            "unit": "ns/op",
            "extra": "32685 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 6994,
            "unit": "B/op",
            "extra": "32685 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "32685 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 37801,
            "unit": "ns/op\t    7446 B/op\t      81 allocs/op",
            "extra": "32175 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 37801,
            "unit": "ns/op",
            "extra": "32175 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7446,
            "unit": "B/op",
            "extra": "32175 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "32175 times\n4 procs"
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
          "id": "763e8c6a802460241b68edda723185dcd14bbaff",
          "message": "refactor: remove unnecessary usage of lru (#3204)\n\n* fix(syncer): refetch latest da height instead of da height +1\n\n* wip\n\n* fixes\n\n* fix changelog and underflow\n\n* fix nil\n\n* fix unit tests\n\n* arrange cl\n\n* updates\n\n* fixes\n\n* fix\n\n* remove lru from generic cache as slow cleanup already happens\n\n* simplify\n\n* Update CHANGELOG.md",
          "timestamp": "2026-03-28T10:07:28Z",
          "tree_id": "0c7921ad92c9f2e4bddb33f8cba54eafb4a18b08",
          "url": "https://github.com/evstack/ev-node/commit/763e8c6a802460241b68edda723185dcd14bbaff"
        },
        "date": 1774693613491,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 36838,
            "unit": "ns/op\t    6983 B/op\t      71 allocs/op",
            "extra": "33146 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 36838,
            "unit": "ns/op",
            "extra": "33146 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 6983,
            "unit": "B/op",
            "extra": "33146 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "33146 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 37454,
            "unit": "ns/op\t    7443 B/op\t      81 allocs/op",
            "extra": "32314 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 37454,
            "unit": "ns/op",
            "extra": "32314 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7443,
            "unit": "B/op",
            "extra": "32314 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "32314 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 46880,
            "unit": "ns/op\t   26124 B/op\t      81 allocs/op",
            "extra": "25885 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 46880,
            "unit": "ns/op",
            "extra": "25885 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26124,
            "unit": "B/op",
            "extra": "25885 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "25885 times\n4 procs"
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
          "id": "889da9a884f29812201635ea89de6d69350a986e",
          "message": "chore: missed feedback from #3204 (#3206)",
          "timestamp": "2026-03-30T10:08:17+02:00",
          "tree_id": "9fd3ba56baa2beb737075664da1b1b14190780e2",
          "url": "https://github.com/evstack/ev-node/commit/889da9a884f29812201635ea89de6d69350a986e"
        },
        "date": 1774858306826,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 39846,
            "unit": "ns/op\t    7045 B/op\t      71 allocs/op",
            "extra": "30639 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 39846,
            "unit": "ns/op",
            "extra": "30639 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7045,
            "unit": "B/op",
            "extra": "30639 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "30639 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 40491,
            "unit": "ns/op\t    7495 B/op\t      81 allocs/op",
            "extra": "30265 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 40491,
            "unit": "ns/op",
            "extra": "30265 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7495,
            "unit": "B/op",
            "extra": "30265 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "30265 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 49384,
            "unit": "ns/op\t   26158 B/op\t      81 allocs/op",
            "extra": "24910 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 49384,
            "unit": "ns/op",
            "extra": "24910 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26158,
            "unit": "B/op",
            "extra": "24910 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "24910 times\n4 procs"
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
          "id": "a4484f5a3d803a1cfd2597e65272b0cef427ad26",
          "message": "chore(syncing): avoid sending duplicate events to channel (#3207)\n\n* chore(syncing): avoid sending duplicate events to channel\n\n* add comment\n\n* improve comment",
          "timestamp": "2026-03-30T12:08:02Z",
          "tree_id": "61052d7c4b3ab80d29fd866424b0d4bfc2d72a27",
          "url": "https://github.com/evstack/ev-node/commit/a4484f5a3d803a1cfd2597e65272b0cef427ad26"
        },
        "date": 1774873775086,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 47155,
            "unit": "ns/op\t   26127 B/op\t      81 allocs/op",
            "extra": "25813 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 47155,
            "unit": "ns/op",
            "extra": "25813 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26127,
            "unit": "B/op",
            "extra": "25813 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "25813 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 36097,
            "unit": "ns/op\t    6981 B/op\t      71 allocs/op",
            "extra": "33250 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 36097,
            "unit": "ns/op",
            "extra": "33250 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 6981,
            "unit": "B/op",
            "extra": "33250 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "33250 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 36625,
            "unit": "ns/op\t    7433 B/op\t      81 allocs/op",
            "extra": "32731 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 36625,
            "unit": "ns/op",
            "extra": "32731 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7433,
            "unit": "B/op",
            "extra": "32731 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "32731 times\n4 procs"
          }
        ]
      },
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
          "id": "146e6e125d7eb093e35eaa8cbc0254f272f18f15",
          "message": "chore: add badger constraints on index and block cache (#3209)\n\n* add sustainable badger configs\n\n* remove blockcache size\n\n* changelog",
          "timestamp": "2026-03-30T16:10:43+02:00",
          "tree_id": "e79c677b0c8ebee40dc4dd039cc3c4dfd4ccce41",
          "url": "https://github.com/evstack/ev-node/commit/146e6e125d7eb093e35eaa8cbc0254f272f18f15"
        },
        "date": 1774880579068,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 38914,
            "unit": "ns/op\t    7030 B/op\t      71 allocs/op",
            "extra": "31194 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 38914,
            "unit": "ns/op",
            "extra": "31194 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7030,
            "unit": "B/op",
            "extra": "31194 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "31194 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 39244,
            "unit": "ns/op\t    7495 B/op\t      81 allocs/op",
            "extra": "30298 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 39244,
            "unit": "ns/op",
            "extra": "30298 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7495,
            "unit": "B/op",
            "extra": "30298 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "30298 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 49833,
            "unit": "ns/op\t   26159 B/op\t      81 allocs/op",
            "extra": "24888 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 49833,
            "unit": "ns/op",
            "extra": "24888 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26159,
            "unit": "B/op",
            "extra": "24888 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "24888 times\n4 procs"
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
          "id": "5e3db23fb2fdc090312bbc39abd831c07a1a2ceb",
          "message": "feat: structured benchmark result output via BENCH_RESULT_OUTPUT (#3195)\n\n* feat(benchmarking): add structured result output via BENCH_RESULT_OUTPUT\n\nemit full benchmark run metadata (config, tags, metrics, block range,\nspamoor stats) as JSON when BENCH_RESULT_OUTPUT is set. consumed by\nexternal matrix runner for table generation.\n\n* fix: address PR review feedback for structured benchmark output\n\n- Deduplicate overhead/reth-rate computation: move stats-based helpers\n  to helpers.go, make span-based wrappers delegate to them\n- Fix sub-millisecond precision loss in engine span timings by using\n  microsecond-based float division instead of integer truncation\n- Add spamoor stats to TestGasBurner for consistency with other tests\n\n* refactor: make spamoor config fully configurable via BENCH_* env vars\n\n- Add MaxPending, Rebroadcast, BaseFee, TipFee to benchConfig\n- Fix ERC20 test hardcoding max_wallets=200 instead of using cfg\n- Replace all hardcoded spamoor params with cfg fields across tests\n\n* feat: extract host metadata from OTEL resource attributes in trace spans\n\n- Add resourceAttrs struct with host, OS, and service fields\n- Extract attributes from VictoriaTraces LogsQL span data via\n  resourceAttrCollector interface\n- Include host metadata in structured benchmark result JSON\n\n* fix: defer emitRunResult so results are written even on test failure\n\nMove emitRunResult into a deferred closure in all three test functions.\nIf the test fails after metrics are collected, the structured JSON is\nstill written. If it fails before result data exists, the defer is a\nno-op.\n\n* fix: state pressure benchmark CI failure and align with other tests\n\nRemove the 3-second sleep before requireSpammersRunning that caused all\ntransactions to be mined before the measurement window started, leaving\nSteadyState at 0s. Also add deferred emitRunResult, configurable spamoor\nparams, and spamoorStats collection to match the other benchmark tests.\n\n* fix: use deployment-level service names for trace queries in external mode\n\nIn external mode the sequencer reports spans as \"ev-node\" (not the\ntest-specific name like \"ev-node-erc20\"), so trace queries returned\nzero spans. Store service names on env: local mode uses the\ntest-specific name, external mode defaults to \"ev-node\"/\"ev-reth\"\nwith BENCH_EVNODE_SERVICE_NAME/BENCH_EVRETH_SERVICE_NAME overrides.\n\n* perf: use limit=1 for resource attribute trace queries\n\nfetchResourceAttrs only needs one span but was streaming the full\nresult set from VictoriaTraces. Add limit=1 to the LogsQL query to\navoid wasting bandwidth on long-lived instances with many spans.\n\n* docs: add missing doc comments to run_result.go functions",
          "timestamp": "2026-03-30T14:54:13Z",
          "tree_id": "55b09c16e7fe1a5e662b08495fdda99c8122bd5b",
          "url": "https://github.com/evstack/ev-node/commit/5e3db23fb2fdc090312bbc39abd831c07a1a2ceb"
        },
        "date": 1774883655770,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 38767,
            "unit": "ns/op\t    7028 B/op\t      71 allocs/op",
            "extra": "31290 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 38767,
            "unit": "ns/op",
            "extra": "31290 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7028,
            "unit": "B/op",
            "extra": "31290 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "31290 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 39805,
            "unit": "ns/op\t    7489 B/op\t      81 allocs/op",
            "extra": "30513 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 39805,
            "unit": "ns/op",
            "extra": "30513 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7489,
            "unit": "B/op",
            "extra": "30513 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "30513 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 50438,
            "unit": "ns/op\t   26168 B/op\t      81 allocs/op",
            "extra": "24180 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 50438,
            "unit": "ns/op",
            "extra": "24180 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26168,
            "unit": "B/op",
            "extra": "24180 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "24180 times\n4 procs"
          }
        ]
      }
    ]
  }
}