window.BENCHMARK_DATA = {
  "lastUpdate": 1772527675919,
  "repoUrl": "https://github.com/evstack/ev-node",
  "entries": {
    "EVM Contract Roundtrip": [
      {
        "commit": {
          "author": {
            "name": "evstack",
            "username": "evstack"
          },
          "committer": {
            "name": "evstack",
            "username": "evstack"
          },
          "id": "3b74695b3b8d389271a0c122f656a2a7199ef382",
          "message": "feat: introduce e2e benchmarking ",
          "timestamp": "2026-02-17T17:03:09Z",
          "url": "https://github.com/evstack/ev-node/pull/3078/commits/3b74695b3b8d389271a0c122f656a2a7199ef382"
        },
        "date": 1771408346118,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 877896807,
            "unit": "ns/op\t 1937448 B/op\t   11512 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 877896807,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 1937448,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 11512,
            "unit": "allocs/op",
            "extra": "2 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "evstack",
            "username": "evstack"
          },
          "committer": {
            "name": "evstack",
            "username": "evstack"
          },
          "id": "d2512dd0e01ccf461e59c94906aafd5397e92bb0",
          "message": "feat: introduce e2e benchmarking ",
          "timestamp": "2026-02-18T12:45:05Z",
          "url": "https://github.com/evstack/ev-node/pull/3078/commits/d2512dd0e01ccf461e59c94906aafd5397e92bb0"
        },
        "date": 1771421156072,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 878594158,
            "unit": "ns/op\t 1986828 B/op\t   11521 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 878594158,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 1986828,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 11521,
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
          "id": "dc473e56d221d42faac16e6db53a1319c255a7c8",
          "message": "feat: introduce e2e benchmarking  (#3078)\n\n* feat: introduce EVM contract benchmarking with new tests and a GitHub Actions workflow.\n\n* Capture otel\n\n* Go mod tidy\n\n* Enable benchmark on PRs - revert before merge\n\n* Push benchmark results on PR - revert later\n\n* Review feedback\n\n* Revert \"Push benchmark results on PR - revert later\"\n\nThis reverts commit 3b74695b3b8d389271a0c122f656a2a7199ef382.\n\n* Revert \"Enable benchmark on PRs - revert before merge\"\n\nThis reverts commit c7a5914df0b5400db5c26aa60bbb99437d745f9a.",
          "timestamp": "2026-02-19T09:06:22Z",
          "tree_id": "5cf2dbd500f1b330948f29327bbb66ec819f5dfe",
          "url": "https://github.com/evstack/ev-node/commit/dc473e56d221d42faac16e6db53a1319c255a7c8"
        },
        "date": 1771493138917,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 878664842,
            "unit": "ns/op\t 1972496 B/op\t   11531 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 878664842,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 1972496,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 11531,
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
          "id": "057c388b6be66a0082175d6b2a384f1ae6a6373e",
          "message": "feat: restore sequencer (#3061)\n\n* Recover sequencer\n\n* Review feedback\n\n* feat: Implement aggregator catchup phase to sync from DA and P2P before block production.\n\n* Linter\n\n* Review feedback",
          "timestamp": "2026-02-19T12:23:47+01:00",
          "tree_id": "2706c4cd8e969dd2cc0d03818a642b2b9616a911",
          "url": "https://github.com/evstack/ev-node/commit/057c388b6be66a0082175d6b2a384f1ae6a6373e"
        },
        "date": 1771500421718,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 879117590,
            "unit": "ns/op\t 1986400 B/op\t   11525 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 879117590,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 1986400,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 11525,
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
          "id": "a9c50d2e6905ec305a112e7960096e822ce162cc",
          "message": "build(deps): Bump the go_modules group across 1 directory with 2 updates (#3084)\n\n* build(deps): Bump the go_modules group across 1 directory with 2 updates\n\nBumps the go_modules group with 2 updates in the /execution/evm/test directory: [github.com/ethereum/go-ethereum](https://github.com/ethereum/go-ethereum) and [filippo.io/edwards25519](https://github.com/FiloSottile/edwards25519).\n\n\nUpdates `github.com/ethereum/go-ethereum` from 1.16.8 to 1.17.0\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.16.8...v1.17.0)\n\nUpdates `filippo.io/edwards25519` from 1.1.0 to 1.1.1\n- [Commits](https://github.com/FiloSottile/edwards25519/compare/v1.1.0...v1.1.1)\n\n---\nupdated-dependencies:\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.0\n  dependency-type: direct:production\n  dependency-group: go_modules\n- dependency-name: filippo.io/edwards25519\n  dependency-version: 1.1.1\n  dependency-type: indirect\n  dependency-group: go_modules\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\n\n* go mod tidy\n\n---------\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>\nCo-authored-by: Julien Robert <julien@rbrt.fr>",
          "timestamp": "2026-02-19T12:34:18+01:00",
          "tree_id": "bc58928ed70a574b66d686ca522f33ae246d4809",
          "url": "https://github.com/evstack/ev-node/commit/a9c50d2e6905ec305a112e7960096e822ce162cc"
        },
        "date": 1771501151706,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 878732112,
            "unit": "ns/op\t 1938764 B/op\t   11528 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 878732112,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 1938764,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 11528,
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
          "id": "ef3469a5ae53ef41b58f741a17343d1ef4a900be",
          "message": "feat: adding spamoor test (#3091)\n\n* feat: introduce EVM contract benchmarking with new tests and a GitHub Actions workflow.\n\n* Capture otel\n\n* wip: basic spamoor test running and reporting metrics\n\n* chore: adding trance benchmark e2e\n\n* wip: experimenting with gas burner tx\n\n* wip\n\n* chore: adding basic spamoor test\n\n* chore: remove local pin\n\n* chore: adding basic assertion\n\n* fix linter\n\n* chore: fix indentation\n\n---------\n\nCo-authored-by: Alex Peters <alpe@users.noreply.github.com>",
          "timestamp": "2026-02-19T12:07:32Z",
          "tree_id": "4f2196dc31f4d710af0f5c5579b893ad6c135ca4",
          "url": "https://github.com/evstack/ev-node/commit/ef3469a5ae53ef41b58f741a17343d1ef4a900be"
        },
        "date": 1771504100914,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 878058140,
            "unit": "ns/op\t 1974596 B/op\t   11527 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 878058140,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 1974596,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 11527,
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
          "id": "05ce69eed18a39e6851f362f27797e1e188e91af",
          "message": "test: phase1 benchmarks (#3081)\n\n* Introduce phase1 benchmark\n\n* Bench refactor\n\n* bench: fix monotonically-increasing timestamp + add 100-tx case",
          "timestamp": "2026-02-19T17:20:33+01:00",
          "tree_id": "3d5412d9ac6ef97e584e4dc4bdea730e21f02545",
          "url": "https://github.com/evstack/ev-node/commit/05ce69eed18a39e6851f362f27797e1e188e91af"
        },
        "date": 1771518313149,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 878348924,
            "unit": "ns/op\t 1937040 B/op\t   11510 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 878348924,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 1937040,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 11510,
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
          "id": "ce184847d2e3a610967f70b27ddae3e075b77e97",
          "message": "perf: optimize block creation (#3093)\n\n* Introduce phase1 benchmark\n\n* Bench refactor\n\n* bench: fix monotonically-increasing timestamp + add 100-tx case\n\n* shot1\n\n* x\n\n* y\n\n* cgo\n\n* z\n\n* Revert proto changes and interface\n\n* Remove unecessay caches\n\n* Save pending\n\n* test: add 100-transaction benchmark for block production and simplify executor comments.\n\n* refactor: remove `ValidateBlock` method and simplify state validation logic.\n\n* Extract last block info",
          "timestamp": "2026-02-20T15:47:35+01:00",
          "tree_id": "d590f22cec34019392dd991cf9c67b2c8d67b1f7",
          "url": "https://github.com/evstack/ev-node/commit/ce184847d2e3a610967f70b27ddae3e075b77e97"
        },
        "date": 1771599038724,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 878010518,
            "unit": "ns/op\t 1967664 B/op\t   11720 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 878010518,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 1967664,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 11720,
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
          "id": "aa1af66cb8796af95c95e9d6f8461fc2f096dbdc",
          "message": "chore: enable goimports (#3096)\n\nfix goimports",
          "timestamp": "2026-02-23T09:27:47+01:00",
          "tree_id": "369f5d672564e1d143fdedabebcf49aa7eab052d",
          "url": "https://github.com/evstack/ev-node/commit/aa1af66cb8796af95c95e9d6f8461fc2f096dbdc"
        },
        "date": 1771835451876,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 878085212,
            "unit": "ns/op\t 2051368 B/op\t   11747 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 878085212,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 2051368,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 11747,
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
          "id": "6d6cc1749533eafb89020c4d3162d6ce49bcc9ba",
          "message": "docs: add deployment guide (#3097)\n\n* add mainnet docs closes #2597\n\n* amend",
          "timestamp": "2026-02-23T14:35:02+01:00",
          "tree_id": "dc137609c78d961d84a054b52eb880e20a0bd8ad",
          "url": "https://github.com/evstack/ev-node/commit/6d6cc1749533eafb89020c4d3162d6ce49bcc9ba"
        },
        "date": 1771854023530,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 877878650,
            "unit": "ns/op\t 1947688 B/op\t   11723 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 877878650,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 1947688,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 11723,
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
          "id": "a5ef7718dda652d127f2dab2e063b07d2c86bd2d",
          "message": "chore: refactor tests to allow dynamic injection of docker client (#3098)\n\n* chore: refactor test setup to allow for injection of docker client\n\n* chore: fix test compilation errors, removed useless comments\n\n* chore: addresing PR feedback",
          "timestamp": "2026-02-23T15:25:16Z",
          "tree_id": "fa7406b76c87f46f5a3497cf54e31ca7d93ea6fb",
          "url": "https://github.com/evstack/ev-node/commit/a5ef7718dda652d127f2dab2e063b07d2c86bd2d"
        },
        "date": 1771861505165,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 877835574,
            "unit": "ns/op\t 1946868 B/op\t   11716 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 877835574,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 1946868,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 11716,
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
          "id": "81d3558db594974fdbe9226d62e6f6ab997d4836",
          "message": "build(deps): Bump actions/checkout from 4.2.2 to 6.0.2 (#3103)\n\n* build(deps): Bump actions/checkout from 4.2.2 to 6.0.2\n\nBumps [actions/checkout](https://github.com/actions/checkout) from 4.2.2 to 6.0.2.\n- [Release notes](https://github.com/actions/checkout/releases)\n- [Commits](https://github.com/actions/checkout/compare/v4.2.2...v6.0.2)\n\n---\nupdated-dependencies:\n- dependency-name: actions/checkout\n  dependency-version: 6.0.2\n  dependency-type: direct:production\n  update-type: version-update:semver-major\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\n\n* Update dependabot-auto-fix.yml\n\n---------\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>\nCo-authored-by: Julien Robert <julien@rbrt.fr>",
          "timestamp": "2026-02-24T10:49:01+01:00",
          "tree_id": "48ba30fe437a1ebf10b115b0e967bfadcd017397",
          "url": "https://github.com/evstack/ev-node/commit/81d3558db594974fdbe9226d62e6f6ab997d4836"
        },
        "date": 1771926744985,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 877915415,
            "unit": "ns/op\t 1947844 B/op\t   11725 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 877915415,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 1947844,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 11725,
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
          "id": "fce49c0cdedfab67fc359846dcecf27e76793d98",
          "message": "build(deps): Bump goreleaser/goreleaser-action from 6 to 7 (#3101)\n\nBumps [goreleaser/goreleaser-action](https://github.com/goreleaser/goreleaser-action) from 6 to 7.\n- [Release notes](https://github.com/goreleaser/goreleaser-action/releases)\n- [Commits](https://github.com/goreleaser/goreleaser-action/compare/v6...v7)\n\n---\nupdated-dependencies:\n- dependency-name: goreleaser/goreleaser-action\n  dependency-version: '7'\n  dependency-type: direct:production\n  update-type: version-update:semver-major\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-02-24T10:49:51+01:00",
          "tree_id": "c36646a47791d7a593806c673cb79b4e379d9fa3",
          "url": "https://github.com/evstack/ev-node/commit/fce49c0cdedfab67fc359846dcecf27e76793d98"
        },
        "date": 1771926810013,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 853094756,
            "unit": "ns/op\t 1919504 B/op\t   11660 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 853094756,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 1919504,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 11660,
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
          "id": "42f7321644819c590cf3189b6751fdf714448ba5",
          "message": "build(deps): Bump actions/setup-go from 5.5.0 to 6.2.0 (#3102)\n\nBumps [actions/setup-go](https://github.com/actions/setup-go) from 5.5.0 to 6.2.0.\n- [Release notes](https://github.com/actions/setup-go/releases)\n- [Commits](https://github.com/actions/setup-go/compare/v5.5.0...v6.2.0)\n\n---\nupdated-dependencies:\n- dependency-name: actions/setup-go\n  dependency-version: 6.2.0\n  dependency-type: direct:production\n  update-type: version-update:semver-major\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-02-24T10:52:05+01:00",
          "tree_id": "405d57eab0429dc42d2049417e35844cbac686e3",
          "url": "https://github.com/evstack/ev-node/commit/42f7321644819c590cf3189b6751fdf714448ba5"
        },
        "date": 1771926994093,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 878453114,
            "unit": "ns/op\t 1944692 B/op\t   11695 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 878453114,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 1944692,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 11695,
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
          "id": "6a6c2a08bcd1bb7bfad7ea4b72785987d4f6e3f3",
          "message": "build(deps): Bump the all-go group across 4 directories with 2 updates (#3100)\n\nBumps the all-go group with 1 update in the /apps/evm directory: [github.com/ethereum/go-ethereum](https://github.com/ethereum/go-ethereum).\nBumps the all-go group with 1 update in the /execution/evm directory: [github.com/ethereum/go-ethereum](https://github.com/ethereum/go-ethereum).\nBumps the all-go group with 2 updates in the /test/docker-e2e directory: [github.com/ethereum/go-ethereum](https://github.com/ethereum/go-ethereum) and [github.com/celestiaorg/tastora](https://github.com/celestiaorg/tastora).\nBumps the all-go group with 1 update in the /test/e2e directory: [github.com/celestiaorg/tastora](https://github.com/celestiaorg/tastora).\n\n\nUpdates `github.com/ethereum/go-ethereum` from 1.16.8 to 1.17.0\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.16.8...v1.17.0)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.16.8 to 1.17.0\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.16.8...v1.17.0)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.16.8 to 1.17.0\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.16.8...v1.17.0)\n\nUpdates `github.com/celestiaorg/tastora` from 0.12.0 to 0.15.0\n- [Release notes](https://github.com/celestiaorg/tastora/releases)\n- [Commits](https://github.com/celestiaorg/tastora/compare/v0.12.0...v0.15.0)\n\nUpdates `github.com/celestiaorg/tastora` from 0.14.0 to 0.15.0\n- [Release notes](https://github.com/celestiaorg/tastora/releases)\n- [Commits](https://github.com/celestiaorg/tastora/compare/v0.12.0...v0.15.0)\n\n---\nupdated-dependencies:\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/celestiaorg/tastora\n  dependency-version: 0.15.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/celestiaorg/tastora\n  dependency-version: 0.15.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>\nCo-authored-by: julienrbrt <julien@rbrt.fr>",
          "timestamp": "2026-02-24T10:12:50Z",
          "tree_id": "cc70f192234ee688064f4e8075b707e13b645057",
          "url": "https://github.com/evstack/ev-node/commit/6a6c2a08bcd1bb7bfad7ea4b72785987d4f6e3f3"
        },
        "date": 1771929777227,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 877408776,
            "unit": "ns/op\t 1950632 B/op\t   11710 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 877408776,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 1950632,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 11710,
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
          "id": "67e18bdae65f2b2e919d34f51e4a58c08459c386",
          "message": "chore: gather spans from both ev-node and ev-reth in spamoor test (#3099)\n\n* wip: passing test, but hacky\n\n* chore: refactor test setup to allow for injection of docker client\n\n* chore: fix test compilation errors, removed useless comments\n\n* chore: addresing PR feedback\n\n* chore: create common trace printing code\n\n* chore: adding assertions on addtional spans\n\n* deps: tidy all\n\n* chore: address PR feedback\n\n* chore: removed unnessesdary test\n\n* chore: removed unused imports\n\n* chore: removed non-existant trace",
          "timestamp": "2026-02-24T11:17:08Z",
          "tree_id": "6b532ff738799a952fe7418571ac0dd81cb24e33",
          "url": "https://github.com/evstack/ev-node/commit/67e18bdae65f2b2e919d34f51e4a58c08459c386"
        },
        "date": 1771933018956,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 879465332,
            "unit": "ns/op\t 1944308 B/op\t   11702 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 879465332,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 1944308,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 11702,
            "unit": "allocs/op",
            "extra": "2 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "chatton",
            "username": "chatton",
            "email": "github.qpeyb@simplelogin.fr"
          },
          "committer": {
            "name": "chatton",
            "username": "chatton",
            "email": "github.qpeyb@simplelogin.fr"
          },
          "id": "f4a09a2eae5966e209b287614b4ea4c0cf51858b",
          "message": "ci: run benchmarks on PRs without updating baseline",
          "timestamp": "2026-02-24T13:35:00Z",
          "url": "https://github.com/evstack/ev-node/commit/f4a09a2eae5966e209b287614b4ea4c0cf51858b"
        },
        "date": 1771940578795,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 877725048,
            "unit": "ns/op\t 1944456 B/op\t   11694 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 877725048,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 1944456,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 11694,
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
          "id": "212ac0881c0c32481afa69d10a6ac571ddfea168",
          "message": "feat: adding spammoor test to benchmark (#3105)\n\n* feat: adding spammoor test to benchmark\n\n* fix: use microseconds in benchmark JSON to preserve sub-millisecond precision\n\n* ci: run benchmarks on PRs without updating baseline\n\n* ci: fan-out/fan-in benchmark jobs to avoid gh-pages race condition\n\n* ci: reset local gh-pages between benchmark steps to avoid fetch conflict\n\n* ci: fix spamoor benchmark artifact download path\n\n* ci: isolate benchmark publish steps so failures don't cascade\n\n* ci: only emit avg in benchmark JSON to reduce alert noise",
          "timestamp": "2026-02-25T11:06:23Z",
          "tree_id": "3fb93a845ff6b98b0a4d9dafdcf27ce575de9257",
          "url": "https://github.com/evstack/ev-node/commit/212ac0881c0c32481afa69d10a6ac571ddfea168"
        },
        "date": 1772018864606,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 878015404,
            "unit": "ns/op\t 1976228 B/op\t   11713 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 878015404,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 1976228,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 11713,
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
          "id": "52080e92e4b810b47fb23d567470e17f198c1ea6",
          "message": "build: migrate from Make to just as command runner (#3110)\n\n* build: migrate from Make to just as command runner\n\nReplace Makefile + 6 .mk include files with a single justfile.\nUpdate all CI workflows (setup-just action) and docs references.\n\nCo-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>\n\n* build: make `just` list recipes by default\n\nCo-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>\n\n* build: add recipe groupings to justfile\n\nGroups: build, test, proto, lint, codegen, run, tools.\nUses --unsorted to preserve logical ordering within groups.\n\nCo-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>\n\n* build: split justfile into per-group files under .just/\n\nRoot justfile now holds variables and imports.\nRecipe files: build, test, proto, lint, codegen, run, tools.\n\nCo-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>\n\n* fix: use extractions/setup-just@v3 (v4 does not exist)\n\nCo-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>\n\n* fix: remove make dependency from Dockerfiles\n\nInline go build/install commands directly instead of depending on\nmake/just inside the container.\n\nCo-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>\n\n---------\n\nCo-authored-by: Claude Opus 4.6 <noreply@anthropic.com>",
          "timestamp": "2026-02-26T21:30:46+01:00",
          "tree_id": "5f17998a73c0d6103492cbc52e124969dfb6f21a",
          "url": "https://github.com/evstack/ev-node/commit/52080e92e4b810b47fb23d567470e17f198c1ea6"
        },
        "date": 1772138123980,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 878340082,
            "unit": "ns/op\t 1946772 B/op\t   11714 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 878340082,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 1946772,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 11714,
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
          "id": "c449847baaad8c3e6378c5bc50818765093de3d2",
          "message": "test: faster block time (#3094)\n\n* test: faster block time\n\n* Faster block times",
          "timestamp": "2026-02-27T08:42:54Z",
          "tree_id": "a9dce02f0dfc6387c181244728fea120a1a09bc6",
          "url": "https://github.com/evstack/ev-node/commit/c449847baaad8c3e6378c5bc50818765093de3d2"
        },
        "date": 1772183105662,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 922144886,
            "unit": "ns/op\t27213812 B/op\t  123038 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 922144886,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 27213812,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 123038,
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
          "id": "f73a1241bb0b7926f9cbf3d26ffd35501d0e04b8",
          "message": "docs: rewrite and restructure docs (#3026)\n\n* rewrite and restructure docs\n\n* refernce and change of sequencing concept\n\n* Refactor documentation for data availability layers and node operations\n\n- Updated Celestia guide to clarify prerequisites, installation, and configuration for connecting Evolve chains to Celestia.\n- Enhanced Local DA documentation with installation instructions, configuration options, and use cases for development and testing.\n- Expanded troubleshooting guide with detailed diagnostic commands, common issues, and solutions for node operations.\n- Created comprehensive upgrades guide covering minor and major upgrades, version compatibility, and rollback procedures.\n- Added aggregator node documentation detailing configuration, block production settings, and monitoring options.\n- Introduced attester node overview with configuration and use cases for low-latency applications.\n- Removed outdated light node documentation.\n- Improved formatting and clarity in ev-reth chainspec reference for better readability.\n\n* format\n\n* claenup and comments\n\n* Update block-lifecycle.md\n\n* adjustments\n\n* docs: fix broken links, stale flag prefixes, and formatting issues (#3112)\n\n* docs: fix broken links, stale --rollkit.* flag prefixes, and escaped backticks\n\n- Fix forced-inclusion.md: based.md → based-sequencing.md (2 links)\n- Fix block-lifecycle.md: da.md → data-availability.md (2 links)\n- Fix reference/specs/overview.md: p2p.md → learn/specs/p2p.md\n- Fix running-nodes/full-node.md: relative paths to guides/\n- Fix what-is-evolve.md: remove escaped backticks around ev-node\n- Replace all --rollkit.* flags with --evnode.* in ev-node-config.md\n- Fix visualizer.md: --rollkit.rpc → --evnode.rpc\n\n* docs: fix markdownlint errors (MD040 missing code block languages, MD012 extra blank lines)\n\n- Add 'text' language specifier to 25 fenced code blocks containing ASCII\n  diagrams, log output, and formulas\n- Remove extra blank line in migration-from-cometbft.md\n- Fix non-descriptive link text in deployment.md\n\n* fix and push\n\n---------\n\nCo-authored-by: Alexander Peters <alpe@users.noreply.github.com>",
          "timestamp": "2026-03-02T11:16:10+01:00",
          "tree_id": "a963782a287c23f898a8cf0cf5b35097986d39dd",
          "url": "https://github.com/evstack/ev-node/commit/f73a1241bb0b7926f9cbf3d26ffd35501d0e04b8"
        },
        "date": 1772446875860,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 918901684,
            "unit": "ns/op\t26683124 B/op\t  117859 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 918901684,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 26683124,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 117859,
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
          "distinct": false,
          "id": "fe99227d249ed6a5f341fa4124c51e8ab747cbed",
          "message": "refactor: move spamoor benchmark into testify suite (#3107)\n\n* refactor: move spamoor benchmark into testify suite in test/e2e/benchmark\n\n- Create test/e2e/benchmark/ subpackage with SpamoorSuite (testify/suite)\n- Move spamoor smoke test into suite as TestSpamoorSmoke\n- Split helpers into focused files: traces.go, output.go, metrics.go\n- Introduce resultWriter for defer-based benchmark JSON output\n- Export shared symbols from evm_test_common.go for cross-package use\n- Restructure CI to fan-out benchmark jobs and fan-in publishing\n- Run benchmarks on PRs only when benchmark-related files change\n\n* fix: correct BENCH_JSON_OUTPUT path for spamoor benchmark\n\ngo test sets the working directory to the package under test, so the\nenv var should be relative to test/e2e/benchmark/, not test/e2e/.\n\n* fix: place package pattern before test binary flags in benchmark CI\n\ngo test treats all arguments after an unknown flag (--evm-binary) as\ntest binary args, so ./benchmark/ was never recognized as a package\npattern.\n\n* fix: adjust evm-binary path for benchmark subpackage working directory\n\ngo test sets the cwd to the package directory (test/e2e/benchmark/),\nso the binary path needs an extra parent traversal.\n\n* fix: exclude benchmark subpackage from make test-e2e\n\nThe benchmark package doesn't define the --binary flag that test-e2e\npasses. It has its own CI workflow so it doesn't need to run here.\n\n* fix: address PR review feedback for benchmark suite\n\n- make reth tag configurable via EV_RETH_TAG env var (default pr-140)\n- fix OTLP config: remove duplicate env vars, use http/protobuf protocol\n- use require.Eventually for host readiness polling\n- rename requireHTTP to requireHostUp\n- use non-fatal logging in resultWriter.flush deferred context\n- fix stale doc comment (setupCommonEVMEnv -> SetupCommonEVMEnv)\n- rename loop variable to avoid shadowing testing.TB convention\n- add block/internal/executing/** to CI path trigger\n- remove unused require import from output.go\n\n* chore: specify http\n\n* chore: filter out benchmark tests from test-e2e",
          "timestamp": "2026-03-02T11:29:50Z",
          "tree_id": "5287f1bf224b065c9a1ee5efcbe78c1e7eebce47",
          "url": "https://github.com/evstack/ev-node/commit/fe99227d249ed6a5f341fa4124c51e8ab747cbed"
        },
        "date": 1772452495375,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 914549018,
            "unit": "ns/op\t26764136 B/op\t  117962 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 914549018,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 26764136,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 117962,
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
          "id": "805f927798c111fcae8a9f2edeb9ab37ee20a353",
          "message": "feat: auto-detect Engine API GetPayload version for Osaka fork (#3113)\n\n* feat: auto-detect Engine API GetPayload version for Osaka fork\n\nGetPayload now automatically selects between engine_getPayloadV4 (Prague)\nand engine_getPayloadV5 (Osaka) by caching the last successful version and\nretrying with the alternative on \"Unsupported fork\" errors (code -38005).\n\nThis handles Prague chains, Osaka-at-genesis chains, and time-based\nPrague-to-Osaka upgrades with zero configuration. At most one extra\nRPC call occurs at the fork transition point.\n\nCo-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>\n\n* fix comments\n\n* rename\n\n---------\n\nCo-authored-by: Claude Opus 4.6 <noreply@anthropic.com>",
          "timestamp": "2026-03-02T11:47:28Z",
          "tree_id": "3b891e4d6b22ff25318d8eff8a63852bc52680ef",
          "url": "https://github.com/evstack/ev-node/commit/805f927798c111fcae8a9f2edeb9ab37ee20a353"
        },
        "date": 1772453260658,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 911846044,
            "unit": "ns/op\t26736288 B/op\t  117500 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 911846044,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 26736288,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 117500,
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
          "id": "9af0f905c61d12fbfae49fa79b770e884dbff5e7",
          "message": "feat(sequencer): catchup from base (#3057)\n\n* feat(sequencer): catchup from base\n\n* catch up fixes\n\n* improvements\n\n* update catchup test\n\n* code cleanups\n\n* imp\n\n* only produce 1 base block per da epoch (unless more needed)\n\n* fixes\n\n* updates\n\n* cleanup comments\n\n* updates\n\n* fix failover with local-da change",
          "timestamp": "2026-03-02T22:17:52+01:00",
          "tree_id": "d0f758c41d610b32872577b7545fc908c283d8be",
          "url": "https://github.com/evstack/ev-node/commit/9af0f905c61d12fbfae49fa79b770e884dbff5e7"
        },
        "date": 1772486561192,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 902693379,
            "unit": "ns/op\t26330816 B/op\t  117139 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 902693379,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 26330816,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 117139,
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
          "id": "036633646c7fea603bdf7f407715a82cea3701dc",
          "message": "build(deps): Bump actions/setup-go from 6.2.0 to 6.3.0 (#3118)\n\nBumps [actions/setup-go](https://github.com/actions/setup-go) from 6.2.0 to 6.3.0.\n- [Release notes](https://github.com/actions/setup-go/releases)\n- [Commits](https://github.com/actions/setup-go/compare/v6.2.0...v6.3.0)\n\n---\nupdated-dependencies:\n- dependency-name: actions/setup-go\n  dependency-version: 6.3.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-03-03T09:15:40+01:00",
          "tree_id": "039d153e6f1b42c23d381d919ef8f2a7bb34636c",
          "url": "https://github.com/evstack/ev-node/commit/036633646c7fea603bdf7f407715a82cea3701dc"
        },
        "date": 1772526021812,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 899526720,
            "unit": "ns/op\t26279812 B/op\t  117198 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 899526720,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 26279812,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 117198,
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
          "id": "e4a68aa598e4ed69c7033bea171c654e63950c2a",
          "message": "build(deps): Bump benchmark-action/github-action-benchmark from 1.20.7 to 1.21.0 (#3120)\n\nbuild(deps): Bump benchmark-action/github-action-benchmark\n\nBumps [benchmark-action/github-action-benchmark](https://github.com/benchmark-action/github-action-benchmark) from 1.20.7 to 1.21.0.\n- [Release notes](https://github.com/benchmark-action/github-action-benchmark/releases)\n- [Changelog](https://github.com/benchmark-action/github-action-benchmark/blob/master/CHANGELOG.md)\n- [Commits](https://github.com/benchmark-action/github-action-benchmark/compare/4bdcce38c94cec68da58d012ac24b7b1155efe8b...a7bc2366eda11037936ea57d811a43b3418d3073)\n\n---\nupdated-dependencies:\n- dependency-name: benchmark-action/github-action-benchmark\n  dependency-version: 1.21.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-03-03T09:18:54+01:00",
          "tree_id": "ad6f70791082c4158ecddf3385d98beccc529371",
          "url": "https://github.com/evstack/ev-node/commit/e4a68aa598e4ed69c7033bea171c654e63950c2a"
        },
        "date": 1772526518980,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 914755166,
            "unit": "ns/op\t27091812 B/op\t  122507 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 914755166,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 27091812,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 122507,
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
          "id": "cfb479ad6e6d1cf0cca57dba72ba2234c8c8f713",
          "message": "build(deps): Bump actions/download-artifact from 4.3.0 to 8.0.0 (#3119)\n\nBumps [actions/download-artifact](https://github.com/actions/download-artifact) from 4.3.0 to 8.0.0.\n- [Release notes](https://github.com/actions/download-artifact/releases)\n- [Commits](https://github.com/actions/download-artifact/compare/v4.3.0...v8)\n\n---\nupdated-dependencies:\n- dependency-name: actions/download-artifact\n  dependency-version: 8.0.0\n  dependency-type: direct:production\n  update-type: version-update:semver-major\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-03-03T09:21:04+01:00",
          "tree_id": "cc79b8b73e321429ff0c2cee550491eee84f9285",
          "url": "https://github.com/evstack/ev-node/commit/cfb479ad6e6d1cf0cca57dba72ba2234c8c8f713"
        },
        "date": 1772526547467,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 909354604,
            "unit": "ns/op\t26456288 B/op\t  117657 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 909354604,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 26456288,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 117657,
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
          "id": "a698fcd493f15e7833eede2472c52a30a305bc3b",
          "message": "build(deps): Bump actions/upload-artifact from 4.6.2 to 7.0.0 (#3117)\n\nBumps [actions/upload-artifact](https://github.com/actions/upload-artifact) from 4.6.2 to 7.0.0.\n- [Release notes](https://github.com/actions/upload-artifact/releases)\n- [Commits](https://github.com/actions/upload-artifact/compare/v4.6.2...v7)\n\n---\nupdated-dependencies:\n- dependency-name: actions/upload-artifact\n  dependency-version: 7.0.0\n  dependency-type: direct:production\n  update-type: version-update:semver-major\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>\nCo-authored-by: julienrbrt <julien@rbrt.fr>",
          "timestamp": "2026-03-03T09:39:50+01:00",
          "tree_id": "aec82ed2d9c373e0f79c50e7e4d2db499320a2c6",
          "url": "https://github.com/evstack/ev-node/commit/a698fcd493f15e7833eede2472c52a30a305bc3b"
        },
        "date": 1772527668509,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 913476988,
            "unit": "ns/op\t26581868 B/op\t  117646 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 913476988,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 26581868,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 117646,
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
          "id": "05ce69eed18a39e6851f362f27797e1e188e91af",
          "message": "test: phase1 benchmarks (#3081)\n\n* Introduce phase1 benchmark\n\n* Bench refactor\n\n* bench: fix monotonically-increasing timestamp + add 100-tx case",
          "timestamp": "2026-02-19T17:20:33+01:00",
          "tree_id": "3d5412d9ac6ef97e584e4dc4bdea730e21f02545",
          "url": "https://github.com/evstack/ev-node/commit/05ce69eed18a39e6851f362f27797e1e188e91af"
        },
        "date": 1771518327858,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 122134,
            "unit": "ns/op\t   10857 B/op\t     154 allocs/op",
            "extra": "9375 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 122134,
            "unit": "ns/op",
            "extra": "9375 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 10857,
            "unit": "B/op",
            "extra": "9375 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 154,
            "unit": "allocs/op",
            "extra": "9375 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 125225,
            "unit": "ns/op\t   11387 B/op\t     170 allocs/op",
            "extra": "9832 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 125225,
            "unit": "ns/op",
            "extra": "9832 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 11387,
            "unit": "B/op",
            "extra": "9832 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 170,
            "unit": "allocs/op",
            "extra": "9832 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 150525,
            "unit": "ns/op\t   47018 B/op\t     277 allocs/op",
            "extra": "7332 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 150525,
            "unit": "ns/op",
            "extra": "7332 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 47018,
            "unit": "B/op",
            "extra": "7332 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 277,
            "unit": "allocs/op",
            "extra": "7332 times\n4 procs"
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
          "id": "ce184847d2e3a610967f70b27ddae3e075b77e97",
          "message": "perf: optimize block creation (#3093)\n\n* Introduce phase1 benchmark\n\n* Bench refactor\n\n* bench: fix monotonically-increasing timestamp + add 100-tx case\n\n* shot1\n\n* x\n\n* y\n\n* cgo\n\n* z\n\n* Revert proto changes and interface\n\n* Remove unecessay caches\n\n* Save pending\n\n* test: add 100-transaction benchmark for block production and simplify executor comments.\n\n* refactor: remove `ValidateBlock` method and simplify state validation logic.\n\n* Extract last block info",
          "timestamp": "2026-02-20T15:47:35+01:00",
          "tree_id": "d590f22cec34019392dd991cf9c67b2c8d67b1f7",
          "url": "https://github.com/evstack/ev-node/commit/ce184847d2e3a610967f70b27ddae3e075b77e97"
        },
        "date": 1771599050884,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 39971,
            "unit": "ns/op\t    7047 B/op\t      71 allocs/op",
            "extra": "30560 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 39971,
            "unit": "ns/op",
            "extra": "30560 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7047,
            "unit": "B/op",
            "extra": "30560 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "30560 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 42309,
            "unit": "ns/op\t    7514 B/op\t      81 allocs/op",
            "extra": "29611 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 42309,
            "unit": "ns/op",
            "extra": "29611 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7514,
            "unit": "B/op",
            "extra": "29611 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "29611 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 50862,
            "unit": "ns/op\t   26170 B/op\t      81 allocs/op",
            "extra": "24241 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 50862,
            "unit": "ns/op",
            "extra": "24241 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26170,
            "unit": "B/op",
            "extra": "24241 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "24241 times\n4 procs"
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
          "id": "aa1af66cb8796af95c95e9d6f8461fc2f096dbdc",
          "message": "chore: enable goimports (#3096)\n\nfix goimports",
          "timestamp": "2026-02-23T09:27:47+01:00",
          "tree_id": "369f5d672564e1d143fdedabebcf49aa7eab052d",
          "url": "https://github.com/evstack/ev-node/commit/aa1af66cb8796af95c95e9d6f8461fc2f096dbdc"
        },
        "date": 1771835466177,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 35830,
            "unit": "ns/op\t    6960 B/op\t      71 allocs/op",
            "extra": "34179 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 35830,
            "unit": "ns/op",
            "extra": "34179 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 6960,
            "unit": "B/op",
            "extra": "34179 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "34179 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 36164,
            "unit": "ns/op\t    7424 B/op\t      81 allocs/op",
            "extra": "33114 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 36164,
            "unit": "ns/op",
            "extra": "33114 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7424,
            "unit": "B/op",
            "extra": "33114 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "33114 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 47490,
            "unit": "ns/op\t   26123 B/op\t      81 allocs/op",
            "extra": "25924 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 47490,
            "unit": "ns/op",
            "extra": "25924 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26123,
            "unit": "B/op",
            "extra": "25924 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "25924 times\n4 procs"
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
          "id": "6d6cc1749533eafb89020c4d3162d6ce49bcc9ba",
          "message": "docs: add deployment guide (#3097)\n\n* add mainnet docs closes #2597\n\n* amend",
          "timestamp": "2026-02-23T14:35:02+01:00",
          "tree_id": "dc137609c78d961d84a054b52eb880e20a0bd8ad",
          "url": "https://github.com/evstack/ev-node/commit/6d6cc1749533eafb89020c4d3162d6ce49bcc9ba"
        },
        "date": 1771854035895,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 38254,
            "unit": "ns/op\t    7448 B/op\t      81 allocs/op",
            "extra": "32090 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 38254,
            "unit": "ns/op",
            "extra": "32090 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7448,
            "unit": "B/op",
            "extra": "32090 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "32090 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 48850,
            "unit": "ns/op\t   26152 B/op\t      81 allocs/op",
            "extra": "25060 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 48850,
            "unit": "ns/op",
            "extra": "25060 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26152,
            "unit": "B/op",
            "extra": "25060 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "25060 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 38322,
            "unit": "ns/op\t    7017 B/op\t      71 allocs/op",
            "extra": "31741 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 38322,
            "unit": "ns/op",
            "extra": "31741 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7017,
            "unit": "B/op",
            "extra": "31741 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "31741 times\n4 procs"
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
          "id": "a5ef7718dda652d127f2dab2e063b07d2c86bd2d",
          "message": "chore: refactor tests to allow dynamic injection of docker client (#3098)\n\n* chore: refactor test setup to allow for injection of docker client\n\n* chore: fix test compilation errors, removed useless comments\n\n* chore: addresing PR feedback",
          "timestamp": "2026-02-23T15:25:16Z",
          "tree_id": "fa7406b76c87f46f5a3497cf54e31ca7d93ea6fb",
          "url": "https://github.com/evstack/ev-node/commit/a5ef7718dda652d127f2dab2e063b07d2c86bd2d"
        },
        "date": 1771861517003,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 38518,
            "unit": "ns/op\t    7029 B/op\t      71 allocs/op",
            "extra": "31255 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 38518,
            "unit": "ns/op",
            "extra": "31255 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7029,
            "unit": "B/op",
            "extra": "31255 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "31255 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 40864,
            "unit": "ns/op\t    7545 B/op\t      81 allocs/op",
            "extra": "28550 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 40864,
            "unit": "ns/op",
            "extra": "28550 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7545,
            "unit": "B/op",
            "extra": "28550 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "28550 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 50434,
            "unit": "ns/op\t   26162 B/op\t      81 allocs/op",
            "extra": "24328 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 50434,
            "unit": "ns/op",
            "extra": "24328 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26162,
            "unit": "B/op",
            "extra": "24328 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "24328 times\n4 procs"
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
          "id": "81d3558db594974fdbe9226d62e6f6ab997d4836",
          "message": "build(deps): Bump actions/checkout from 4.2.2 to 6.0.2 (#3103)\n\n* build(deps): Bump actions/checkout from 4.2.2 to 6.0.2\n\nBumps [actions/checkout](https://github.com/actions/checkout) from 4.2.2 to 6.0.2.\n- [Release notes](https://github.com/actions/checkout/releases)\n- [Commits](https://github.com/actions/checkout/compare/v4.2.2...v6.0.2)\n\n---\nupdated-dependencies:\n- dependency-name: actions/checkout\n  dependency-version: 6.0.2\n  dependency-type: direct:production\n  update-type: version-update:semver-major\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\n\n* Update dependabot-auto-fix.yml\n\n---------\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>\nCo-authored-by: Julien Robert <julien@rbrt.fr>",
          "timestamp": "2026-02-24T10:49:01+01:00",
          "tree_id": "48ba30fe437a1ebf10b115b0e967bfadcd017397",
          "url": "https://github.com/evstack/ev-node/commit/81d3558db594974fdbe9226d62e6f6ab997d4836"
        },
        "date": 1771926756951,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 38020,
            "unit": "ns/op\t    7006 B/op\t      71 allocs/op",
            "extra": "32173 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 38020,
            "unit": "ns/op",
            "extra": "32173 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7006,
            "unit": "B/op",
            "extra": "32173 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "32173 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 40425,
            "unit": "ns/op\t    7475 B/op\t      81 allocs/op",
            "extra": "31036 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 40425,
            "unit": "ns/op",
            "extra": "31036 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7475,
            "unit": "B/op",
            "extra": "31036 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "31036 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 49195,
            "unit": "ns/op\t   26169 B/op\t      81 allocs/op",
            "extra": "24652 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 49195,
            "unit": "ns/op",
            "extra": "24652 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26169,
            "unit": "B/op",
            "extra": "24652 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "24652 times\n4 procs"
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
          "id": "fce49c0cdedfab67fc359846dcecf27e76793d98",
          "message": "build(deps): Bump goreleaser/goreleaser-action from 6 to 7 (#3101)\n\nBumps [goreleaser/goreleaser-action](https://github.com/goreleaser/goreleaser-action) from 6 to 7.\n- [Release notes](https://github.com/goreleaser/goreleaser-action/releases)\n- [Commits](https://github.com/goreleaser/goreleaser-action/compare/v6...v7)\n\n---\nupdated-dependencies:\n- dependency-name: goreleaser/goreleaser-action\n  dependency-version: '7'\n  dependency-type: direct:production\n  update-type: version-update:semver-major\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-02-24T10:49:51+01:00",
          "tree_id": "c36646a47791d7a593806c673cb79b4e379d9fa3",
          "url": "https://github.com/evstack/ev-node/commit/fce49c0cdedfab67fc359846dcecf27e76793d98"
        },
        "date": 1771926825574,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37027,
            "unit": "ns/op\t    6986 B/op\t      71 allocs/op",
            "extra": "33033 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37027,
            "unit": "ns/op",
            "extra": "33033 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 6986,
            "unit": "B/op",
            "extra": "33033 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "33033 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 37748,
            "unit": "ns/op\t    7451 B/op\t      81 allocs/op",
            "extra": "31990 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 37748,
            "unit": "ns/op",
            "extra": "31990 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7451,
            "unit": "B/op",
            "extra": "31990 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "31990 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 47630,
            "unit": "ns/op\t   26138 B/op\t      81 allocs/op",
            "extra": "25524 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 47630,
            "unit": "ns/op",
            "extra": "25524 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26138,
            "unit": "B/op",
            "extra": "25524 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "25524 times\n4 procs"
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
          "id": "42f7321644819c590cf3189b6751fdf714448ba5",
          "message": "build(deps): Bump actions/setup-go from 5.5.0 to 6.2.0 (#3102)\n\nBumps [actions/setup-go](https://github.com/actions/setup-go) from 5.5.0 to 6.2.0.\n- [Release notes](https://github.com/actions/setup-go/releases)\n- [Commits](https://github.com/actions/setup-go/compare/v5.5.0...v6.2.0)\n\n---\nupdated-dependencies:\n- dependency-name: actions/setup-go\n  dependency-version: 6.2.0\n  dependency-type: direct:production\n  update-type: version-update:semver-major\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-02-24T10:52:05+01:00",
          "tree_id": "405d57eab0429dc42d2049417e35844cbac686e3",
          "url": "https://github.com/evstack/ev-node/commit/42f7321644819c590cf3189b6751fdf714448ba5"
        },
        "date": 1771927007272,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 38098,
            "unit": "ns/op\t    7448 B/op\t      81 allocs/op",
            "extra": "32076 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 38098,
            "unit": "ns/op",
            "extra": "32076 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7448,
            "unit": "B/op",
            "extra": "32076 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "32076 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 48377,
            "unit": "ns/op\t   26150 B/op\t      81 allocs/op",
            "extra": "25118 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 48377,
            "unit": "ns/op",
            "extra": "25118 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26150,
            "unit": "B/op",
            "extra": "25118 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "25118 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 38241,
            "unit": "ns/op\t    7008 B/op\t      71 allocs/op",
            "extra": "32108 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 38241,
            "unit": "ns/op",
            "extra": "32108 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7008,
            "unit": "B/op",
            "extra": "32108 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "32108 times\n4 procs"
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
          "id": "6a6c2a08bcd1bb7bfad7ea4b72785987d4f6e3f3",
          "message": "build(deps): Bump the all-go group across 4 directories with 2 updates (#3100)\n\nBumps the all-go group with 1 update in the /apps/evm directory: [github.com/ethereum/go-ethereum](https://github.com/ethereum/go-ethereum).\nBumps the all-go group with 1 update in the /execution/evm directory: [github.com/ethereum/go-ethereum](https://github.com/ethereum/go-ethereum).\nBumps the all-go group with 2 updates in the /test/docker-e2e directory: [github.com/ethereum/go-ethereum](https://github.com/ethereum/go-ethereum) and [github.com/celestiaorg/tastora](https://github.com/celestiaorg/tastora).\nBumps the all-go group with 1 update in the /test/e2e directory: [github.com/celestiaorg/tastora](https://github.com/celestiaorg/tastora).\n\n\nUpdates `github.com/ethereum/go-ethereum` from 1.16.8 to 1.17.0\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.16.8...v1.17.0)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.16.8 to 1.17.0\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.16.8...v1.17.0)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.16.8 to 1.17.0\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.16.8...v1.17.0)\n\nUpdates `github.com/celestiaorg/tastora` from 0.12.0 to 0.15.0\n- [Release notes](https://github.com/celestiaorg/tastora/releases)\n- [Commits](https://github.com/celestiaorg/tastora/compare/v0.12.0...v0.15.0)\n\nUpdates `github.com/celestiaorg/tastora` from 0.14.0 to 0.15.0\n- [Release notes](https://github.com/celestiaorg/tastora/releases)\n- [Commits](https://github.com/celestiaorg/tastora/compare/v0.12.0...v0.15.0)\n\n---\nupdated-dependencies:\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/celestiaorg/tastora\n  dependency-version: 0.15.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/celestiaorg/tastora\n  dependency-version: 0.15.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>\nCo-authored-by: julienrbrt <julien@rbrt.fr>",
          "timestamp": "2026-02-24T10:12:50Z",
          "tree_id": "cc70f192234ee688064f4e8075b707e13b645057",
          "url": "https://github.com/evstack/ev-node/commit/6a6c2a08bcd1bb7bfad7ea4b72785987d4f6e3f3"
        },
        "date": 1771929789003,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37678,
            "unit": "ns/op\t    7010 B/op\t      71 allocs/op",
            "extra": "31995 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37678,
            "unit": "ns/op",
            "extra": "31995 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7010,
            "unit": "B/op",
            "extra": "31995 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "31995 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 38757,
            "unit": "ns/op\t    7462 B/op\t      81 allocs/op",
            "extra": "31545 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 38757,
            "unit": "ns/op",
            "extra": "31545 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7462,
            "unit": "B/op",
            "extra": "31545 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "31545 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 49028,
            "unit": "ns/op\t   26159 B/op\t      81 allocs/op",
            "extra": "24892 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 49028,
            "unit": "ns/op",
            "extra": "24892 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26159,
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
          "id": "67e18bdae65f2b2e919d34f51e4a58c08459c386",
          "message": "chore: gather spans from both ev-node and ev-reth in spamoor test (#3099)\n\n* wip: passing test, but hacky\n\n* chore: refactor test setup to allow for injection of docker client\n\n* chore: fix test compilation errors, removed useless comments\n\n* chore: addresing PR feedback\n\n* chore: create common trace printing code\n\n* chore: adding assertions on addtional spans\n\n* deps: tidy all\n\n* chore: address PR feedback\n\n* chore: removed unnessesdary test\n\n* chore: removed unused imports\n\n* chore: removed non-existant trace",
          "timestamp": "2026-02-24T11:17:08Z",
          "tree_id": "6b532ff738799a952fe7418571ac0dd81cb24e33",
          "url": "https://github.com/evstack/ev-node/commit/67e18bdae65f2b2e919d34f51e4a58c08459c386"
        },
        "date": 1771933031467,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37144,
            "unit": "ns/op\t    7006 B/op\t      71 allocs/op",
            "extra": "32186 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37144,
            "unit": "ns/op",
            "extra": "32186 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7006,
            "unit": "B/op",
            "extra": "32186 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "32186 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 37917,
            "unit": "ns/op\t    7462 B/op\t      81 allocs/op",
            "extra": "31525 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 37917,
            "unit": "ns/op",
            "extra": "31525 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7462,
            "unit": "B/op",
            "extra": "31525 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "31525 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 48823,
            "unit": "ns/op\t   26147 B/op\t      81 allocs/op",
            "extra": "25281 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 48823,
            "unit": "ns/op",
            "extra": "25281 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26147,
            "unit": "B/op",
            "extra": "25281 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "25281 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "chatton",
            "username": "chatton",
            "email": "github.qpeyb@simplelogin.fr"
          },
          "committer": {
            "name": "chatton",
            "username": "chatton",
            "email": "github.qpeyb@simplelogin.fr"
          },
          "id": "f4a09a2eae5966e209b287614b4ea4c0cf51858b",
          "message": "ci: run benchmarks on PRs without updating baseline",
          "timestamp": "2026-02-24T13:35:00Z",
          "url": "https://github.com/evstack/ev-node/commit/f4a09a2eae5966e209b287614b4ea4c0cf51858b"
        },
        "date": 1771940590760,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37930,
            "unit": "ns/op\t    7007 B/op\t      71 allocs/op",
            "extra": "32122 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37930,
            "unit": "ns/op",
            "extra": "32122 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7007,
            "unit": "B/op",
            "extra": "32122 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "32122 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 38930,
            "unit": "ns/op\t    7488 B/op\t      81 allocs/op",
            "extra": "30552 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 38930,
            "unit": "ns/op",
            "extra": "30552 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7488,
            "unit": "B/op",
            "extra": "30552 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "30552 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 49784,
            "unit": "ns/op\t   26164 B/op\t      81 allocs/op",
            "extra": "24774 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 49784,
            "unit": "ns/op",
            "extra": "24774 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26164,
            "unit": "B/op",
            "extra": "24774 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "24774 times\n4 procs"
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
          "id": "212ac0881c0c32481afa69d10a6ac571ddfea168",
          "message": "feat: adding spammoor test to benchmark (#3105)\n\n* feat: adding spammoor test to benchmark\n\n* fix: use microseconds in benchmark JSON to preserve sub-millisecond precision\n\n* ci: run benchmarks on PRs without updating baseline\n\n* ci: fan-out/fan-in benchmark jobs to avoid gh-pages race condition\n\n* ci: reset local gh-pages between benchmark steps to avoid fetch conflict\n\n* ci: fix spamoor benchmark artifact download path\n\n* ci: isolate benchmark publish steps so failures don't cascade\n\n* ci: only emit avg in benchmark JSON to reduce alert noise",
          "timestamp": "2026-02-25T11:06:23Z",
          "tree_id": "3fb93a845ff6b98b0a4d9dafdcf27ce575de9257",
          "url": "https://github.com/evstack/ev-node/commit/212ac0881c0c32481afa69d10a6ac571ddfea168"
        },
        "date": 1772018869777,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37497,
            "unit": "ns/op\t    7001 B/op\t      71 allocs/op",
            "extra": "32364 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37497,
            "unit": "ns/op",
            "extra": "32364 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7001,
            "unit": "B/op",
            "extra": "32364 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "32364 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 38213,
            "unit": "ns/op\t    7459 B/op\t      81 allocs/op",
            "extra": "31659 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 38213,
            "unit": "ns/op",
            "extra": "31659 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7459,
            "unit": "B/op",
            "extra": "31659 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "31659 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 49000,
            "unit": "ns/op\t   26151 B/op\t      81 allocs/op",
            "extra": "24790 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 49000,
            "unit": "ns/op",
            "extra": "24790 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26151,
            "unit": "B/op",
            "extra": "24790 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "24790 times\n4 procs"
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
          "id": "52080e92e4b810b47fb23d567470e17f198c1ea6",
          "message": "build: migrate from Make to just as command runner (#3110)\n\n* build: migrate from Make to just as command runner\n\nReplace Makefile + 6 .mk include files with a single justfile.\nUpdate all CI workflows (setup-just action) and docs references.\n\nCo-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>\n\n* build: make `just` list recipes by default\n\nCo-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>\n\n* build: add recipe groupings to justfile\n\nGroups: build, test, proto, lint, codegen, run, tools.\nUses --unsorted to preserve logical ordering within groups.\n\nCo-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>\n\n* build: split justfile into per-group files under .just/\n\nRoot justfile now holds variables and imports.\nRecipe files: build, test, proto, lint, codegen, run, tools.\n\nCo-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>\n\n* fix: use extractions/setup-just@v3 (v4 does not exist)\n\nCo-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>\n\n* fix: remove make dependency from Dockerfiles\n\nInline go build/install commands directly instead of depending on\nmake/just inside the container.\n\nCo-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>\n\n---------\n\nCo-authored-by: Claude Opus 4.6 <noreply@anthropic.com>",
          "timestamp": "2026-02-26T21:30:46+01:00",
          "tree_id": "5f17998a73c0d6103492cbc52e124969dfb6f21a",
          "url": "https://github.com/evstack/ev-node/commit/52080e92e4b810b47fb23d567470e17f198c1ea6"
        },
        "date": 1772138128488,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 48505,
            "unit": "ns/op\t   26140 B/op\t      81 allocs/op",
            "extra": "25459 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 48505,
            "unit": "ns/op",
            "extra": "25459 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26140,
            "unit": "B/op",
            "extra": "25459 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "25459 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37792,
            "unit": "ns/op\t    7007 B/op\t      71 allocs/op",
            "extra": "32139 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37792,
            "unit": "ns/op",
            "extra": "32139 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7007,
            "unit": "B/op",
            "extra": "32139 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "32139 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 38873,
            "unit": "ns/op\t    7526 B/op\t      81 allocs/op",
            "extra": "29185 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 38873,
            "unit": "ns/op",
            "extra": "29185 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7526,
            "unit": "B/op",
            "extra": "29185 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "29185 times\n4 procs"
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
          "id": "c449847baaad8c3e6378c5bc50818765093de3d2",
          "message": "test: faster block time (#3094)\n\n* test: faster block time\n\n* Faster block times",
          "timestamp": "2026-02-27T08:42:54Z",
          "tree_id": "a9dce02f0dfc6387c181244728fea120a1a09bc6",
          "url": "https://github.com/evstack/ev-node/commit/c449847baaad8c3e6378c5bc50818765093de3d2"
        },
        "date": 1772183110485,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 36015,
            "unit": "ns/op\t    6972 B/op\t      71 allocs/op",
            "extra": "33649 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 36015,
            "unit": "ns/op",
            "extra": "33649 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 6972,
            "unit": "B/op",
            "extra": "33649 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "33649 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 36686,
            "unit": "ns/op\t    7438 B/op\t      81 allocs/op",
            "extra": "32502 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 36686,
            "unit": "ns/op",
            "extra": "32502 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7438,
            "unit": "B/op",
            "extra": "32502 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "32502 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 46837,
            "unit": "ns/op\t   26153 B/op\t      81 allocs/op",
            "extra": "25136 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 46837,
            "unit": "ns/op",
            "extra": "25136 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26153,
            "unit": "B/op",
            "extra": "25136 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "25136 times\n4 procs"
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
          "id": "f73a1241bb0b7926f9cbf3d26ffd35501d0e04b8",
          "message": "docs: rewrite and restructure docs (#3026)\n\n* rewrite and restructure docs\n\n* refernce and change of sequencing concept\n\n* Refactor documentation for data availability layers and node operations\n\n- Updated Celestia guide to clarify prerequisites, installation, and configuration for connecting Evolve chains to Celestia.\n- Enhanced Local DA documentation with installation instructions, configuration options, and use cases for development and testing.\n- Expanded troubleshooting guide with detailed diagnostic commands, common issues, and solutions for node operations.\n- Created comprehensive upgrades guide covering minor and major upgrades, version compatibility, and rollback procedures.\n- Added aggregator node documentation detailing configuration, block production settings, and monitoring options.\n- Introduced attester node overview with configuration and use cases for low-latency applications.\n- Removed outdated light node documentation.\n- Improved formatting and clarity in ev-reth chainspec reference for better readability.\n\n* format\n\n* claenup and comments\n\n* Update block-lifecycle.md\n\n* adjustments\n\n* docs: fix broken links, stale flag prefixes, and formatting issues (#3112)\n\n* docs: fix broken links, stale --rollkit.* flag prefixes, and escaped backticks\n\n- Fix forced-inclusion.md: based.md → based-sequencing.md (2 links)\n- Fix block-lifecycle.md: da.md → data-availability.md (2 links)\n- Fix reference/specs/overview.md: p2p.md → learn/specs/p2p.md\n- Fix running-nodes/full-node.md: relative paths to guides/\n- Fix what-is-evolve.md: remove escaped backticks around ev-node\n- Replace all --rollkit.* flags with --evnode.* in ev-node-config.md\n- Fix visualizer.md: --rollkit.rpc → --evnode.rpc\n\n* docs: fix markdownlint errors (MD040 missing code block languages, MD012 extra blank lines)\n\n- Add 'text' language specifier to 25 fenced code blocks containing ASCII\n  diagrams, log output, and formulas\n- Remove extra blank line in migration-from-cometbft.md\n- Fix non-descriptive link text in deployment.md\n\n* fix and push\n\n---------\n\nCo-authored-by: Alexander Peters <alpe@users.noreply.github.com>",
          "timestamp": "2026-03-02T11:16:10+01:00",
          "tree_id": "a963782a287c23f898a8cf0cf5b35097986d39dd",
          "url": "https://github.com/evstack/ev-node/commit/f73a1241bb0b7926f9cbf3d26ffd35501d0e04b8"
        },
        "date": 1772446880333,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37819,
            "unit": "ns/op\t    6992 B/op\t      71 allocs/op",
            "extra": "32744 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37819,
            "unit": "ns/op",
            "extra": "32744 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 6992,
            "unit": "B/op",
            "extra": "32744 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "32744 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 39080,
            "unit": "ns/op\t    7453 B/op\t      81 allocs/op",
            "extra": "31887 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 39080,
            "unit": "ns/op",
            "extra": "31887 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7453,
            "unit": "B/op",
            "extra": "31887 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "31887 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 48705,
            "unit": "ns/op\t   26149 B/op\t      81 allocs/op",
            "extra": "25148 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 48705,
            "unit": "ns/op",
            "extra": "25148 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26149,
            "unit": "B/op",
            "extra": "25148 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "25148 times\n4 procs"
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
          "distinct": false,
          "id": "fe99227d249ed6a5f341fa4124c51e8ab747cbed",
          "message": "refactor: move spamoor benchmark into testify suite (#3107)\n\n* refactor: move spamoor benchmark into testify suite in test/e2e/benchmark\n\n- Create test/e2e/benchmark/ subpackage with SpamoorSuite (testify/suite)\n- Move spamoor smoke test into suite as TestSpamoorSmoke\n- Split helpers into focused files: traces.go, output.go, metrics.go\n- Introduce resultWriter for defer-based benchmark JSON output\n- Export shared symbols from evm_test_common.go for cross-package use\n- Restructure CI to fan-out benchmark jobs and fan-in publishing\n- Run benchmarks on PRs only when benchmark-related files change\n\n* fix: correct BENCH_JSON_OUTPUT path for spamoor benchmark\n\ngo test sets the working directory to the package under test, so the\nenv var should be relative to test/e2e/benchmark/, not test/e2e/.\n\n* fix: place package pattern before test binary flags in benchmark CI\n\ngo test treats all arguments after an unknown flag (--evm-binary) as\ntest binary args, so ./benchmark/ was never recognized as a package\npattern.\n\n* fix: adjust evm-binary path for benchmark subpackage working directory\n\ngo test sets the cwd to the package directory (test/e2e/benchmark/),\nso the binary path needs an extra parent traversal.\n\n* fix: exclude benchmark subpackage from make test-e2e\n\nThe benchmark package doesn't define the --binary flag that test-e2e\npasses. It has its own CI workflow so it doesn't need to run here.\n\n* fix: address PR review feedback for benchmark suite\n\n- make reth tag configurable via EV_RETH_TAG env var (default pr-140)\n- fix OTLP config: remove duplicate env vars, use http/protobuf protocol\n- use require.Eventually for host readiness polling\n- rename requireHTTP to requireHostUp\n- use non-fatal logging in resultWriter.flush deferred context\n- fix stale doc comment (setupCommonEVMEnv -> SetupCommonEVMEnv)\n- rename loop variable to avoid shadowing testing.TB convention\n- add block/internal/executing/** to CI path trigger\n- remove unused require import from output.go\n\n* chore: specify http\n\n* chore: filter out benchmark tests from test-e2e",
          "timestamp": "2026-03-02T11:29:50Z",
          "tree_id": "5287f1bf224b065c9a1ee5efcbe78c1e7eebce47",
          "url": "https://github.com/evstack/ev-node/commit/fe99227d249ed6a5f341fa4124c51e8ab747cbed"
        },
        "date": 1772452499481,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 38576,
            "unit": "ns/op\t    7024 B/op\t      71 allocs/op",
            "extra": "31442 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 38576,
            "unit": "ns/op",
            "extra": "31442 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7024,
            "unit": "B/op",
            "extra": "31442 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "31442 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 40025,
            "unit": "ns/op\t    7485 B/op\t      81 allocs/op",
            "extra": "30639 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 40025,
            "unit": "ns/op",
            "extra": "30639 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7485,
            "unit": "B/op",
            "extra": "30639 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "30639 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 48846,
            "unit": "ns/op\t   25760 B/op\t      81 allocs/op",
            "extra": "21974 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 48846,
            "unit": "ns/op",
            "extra": "21974 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 25760,
            "unit": "B/op",
            "extra": "21974 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "21974 times\n4 procs"
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
          "id": "805f927798c111fcae8a9f2edeb9ab37ee20a353",
          "message": "feat: auto-detect Engine API GetPayload version for Osaka fork (#3113)\n\n* feat: auto-detect Engine API GetPayload version for Osaka fork\n\nGetPayload now automatically selects between engine_getPayloadV4 (Prague)\nand engine_getPayloadV5 (Osaka) by caching the last successful version and\nretrying with the alternative on \"Unsupported fork\" errors (code -38005).\n\nThis handles Prague chains, Osaka-at-genesis chains, and time-based\nPrague-to-Osaka upgrades with zero configuration. At most one extra\nRPC call occurs at the fork transition point.\n\nCo-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>\n\n* fix comments\n\n* rename\n\n---------\n\nCo-authored-by: Claude Opus 4.6 <noreply@anthropic.com>",
          "timestamp": "2026-03-02T11:47:28Z",
          "tree_id": "3b891e4d6b22ff25318d8eff8a63852bc52680ef",
          "url": "https://github.com/evstack/ev-node/commit/805f927798c111fcae8a9f2edeb9ab37ee20a353"
        },
        "date": 1772453263982,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 38899,
            "unit": "ns/op\t    7028 B/op\t      71 allocs/op",
            "extra": "31290 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 38899,
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
            "value": 38876,
            "unit": "ns/op\t    7468 B/op\t      81 allocs/op",
            "extra": "31292 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 38876,
            "unit": "ns/op",
            "extra": "31292 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7468,
            "unit": "B/op",
            "extra": "31292 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "31292 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 50055,
            "unit": "ns/op\t   26162 B/op\t      81 allocs/op",
            "extra": "24042 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 50055,
            "unit": "ns/op",
            "extra": "24042 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26162,
            "unit": "B/op",
            "extra": "24042 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "24042 times\n4 procs"
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
          "id": "9af0f905c61d12fbfae49fa79b770e884dbff5e7",
          "message": "feat(sequencer): catchup from base (#3057)\n\n* feat(sequencer): catchup from base\n\n* catch up fixes\n\n* improvements\n\n* update catchup test\n\n* code cleanups\n\n* imp\n\n* only produce 1 base block per da epoch (unless more needed)\n\n* fixes\n\n* updates\n\n* cleanup comments\n\n* updates\n\n* fix failover with local-da change",
          "timestamp": "2026-03-02T22:17:52+01:00",
          "tree_id": "d0f758c41d610b32872577b7545fc908c283d8be",
          "url": "https://github.com/evstack/ev-node/commit/9af0f905c61d12fbfae49fa79b770e884dbff5e7"
        },
        "date": 1772486565720,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 38501,
            "unit": "ns/op\t    7006 B/op\t      71 allocs/op",
            "extra": "32181 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 38501,
            "unit": "ns/op",
            "extra": "32181 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7006,
            "unit": "B/op",
            "extra": "32181 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "32181 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 39107,
            "unit": "ns/op\t    7477 B/op\t      81 allocs/op",
            "extra": "30960 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 39107,
            "unit": "ns/op",
            "extra": "30960 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7477,
            "unit": "B/op",
            "extra": "30960 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "30960 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 49981,
            "unit": "ns/op\t   26149 B/op\t      81 allocs/op",
            "extra": "25028 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 49981,
            "unit": "ns/op",
            "extra": "25028 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26149,
            "unit": "B/op",
            "extra": "25028 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "25028 times\n4 procs"
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
          "id": "036633646c7fea603bdf7f407715a82cea3701dc",
          "message": "build(deps): Bump actions/setup-go from 6.2.0 to 6.3.0 (#3118)\n\nBumps [actions/setup-go](https://github.com/actions/setup-go) from 6.2.0 to 6.3.0.\n- [Release notes](https://github.com/actions/setup-go/releases)\n- [Commits](https://github.com/actions/setup-go/compare/v6.2.0...v6.3.0)\n\n---\nupdated-dependencies:\n- dependency-name: actions/setup-go\n  dependency-version: 6.3.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-03-03T09:15:40+01:00",
          "tree_id": "039d153e6f1b42c23d381d919ef8f2a7bb34636c",
          "url": "https://github.com/evstack/ev-node/commit/036633646c7fea603bdf7f407715a82cea3701dc"
        },
        "date": 1772526026117,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 38397,
            "unit": "ns/op\t    6998 B/op\t      71 allocs/op",
            "extra": "32491 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 38397,
            "unit": "ns/op",
            "extra": "32491 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 6998,
            "unit": "B/op",
            "extra": "32491 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "32491 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 38503,
            "unit": "ns/op\t    7461 B/op\t      81 allocs/op",
            "extra": "31586 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 38503,
            "unit": "ns/op",
            "extra": "31586 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7461,
            "unit": "B/op",
            "extra": "31586 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "31586 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 48631,
            "unit": "ns/op\t   26133 B/op\t      81 allocs/op",
            "extra": "25644 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 48631,
            "unit": "ns/op",
            "extra": "25644 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26133,
            "unit": "B/op",
            "extra": "25644 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "25644 times\n4 procs"
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
          "id": "e4a68aa598e4ed69c7033bea171c654e63950c2a",
          "message": "build(deps): Bump benchmark-action/github-action-benchmark from 1.20.7 to 1.21.0 (#3120)\n\nbuild(deps): Bump benchmark-action/github-action-benchmark\n\nBumps [benchmark-action/github-action-benchmark](https://github.com/benchmark-action/github-action-benchmark) from 1.20.7 to 1.21.0.\n- [Release notes](https://github.com/benchmark-action/github-action-benchmark/releases)\n- [Changelog](https://github.com/benchmark-action/github-action-benchmark/blob/master/CHANGELOG.md)\n- [Commits](https://github.com/benchmark-action/github-action-benchmark/compare/4bdcce38c94cec68da58d012ac24b7b1155efe8b...a7bc2366eda11037936ea57d811a43b3418d3073)\n\n---\nupdated-dependencies:\n- dependency-name: benchmark-action/github-action-benchmark\n  dependency-version: 1.21.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-03-03T09:18:54+01:00",
          "tree_id": "ad6f70791082c4158ecddf3385d98beccc529371",
          "url": "https://github.com/evstack/ev-node/commit/e4a68aa598e4ed69c7033bea171c654e63950c2a"
        },
        "date": 1772526523419,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 48719,
            "unit": "ns/op\t   26147 B/op\t      81 allocs/op",
            "extra": "25296 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 48719,
            "unit": "ns/op",
            "extra": "25296 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26147,
            "unit": "B/op",
            "extra": "25296 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "25296 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37574,
            "unit": "ns/op\t    7012 B/op\t      71 allocs/op",
            "extra": "31912 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37574,
            "unit": "ns/op",
            "extra": "31912 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7012,
            "unit": "B/op",
            "extra": "31912 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "31912 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 37780,
            "unit": "ns/op\t    7442 B/op\t      81 allocs/op",
            "extra": "32334 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 37780,
            "unit": "ns/op",
            "extra": "32334 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7442,
            "unit": "B/op",
            "extra": "32334 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "32334 times\n4 procs"
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
          "id": "cfb479ad6e6d1cf0cca57dba72ba2234c8c8f713",
          "message": "build(deps): Bump actions/download-artifact from 4.3.0 to 8.0.0 (#3119)\n\nBumps [actions/download-artifact](https://github.com/actions/download-artifact) from 4.3.0 to 8.0.0.\n- [Release notes](https://github.com/actions/download-artifact/releases)\n- [Commits](https://github.com/actions/download-artifact/compare/v4.3.0...v8)\n\n---\nupdated-dependencies:\n- dependency-name: actions/download-artifact\n  dependency-version: 8.0.0\n  dependency-type: direct:production\n  update-type: version-update:semver-major\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-03-03T09:21:04+01:00",
          "tree_id": "cc79b8b73e321429ff0c2cee550491eee84f9285",
          "url": "https://github.com/evstack/ev-node/commit/cfb479ad6e6d1cf0cca57dba72ba2234c8c8f713"
        },
        "date": 1772526551945,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 36547,
            "unit": "ns/op\t    6983 B/op\t      71 allocs/op",
            "extra": "33150 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 36547,
            "unit": "ns/op",
            "extra": "33150 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 6983,
            "unit": "B/op",
            "extra": "33150 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "33150 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 37352,
            "unit": "ns/op\t    7439 B/op\t      81 allocs/op",
            "extra": "32457 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 37352,
            "unit": "ns/op",
            "extra": "32457 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7439,
            "unit": "B/op",
            "extra": "32457 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "32457 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 47009,
            "unit": "ns/op\t   26123 B/op\t      81 allocs/op",
            "extra": "25899 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 47009,
            "unit": "ns/op",
            "extra": "25899 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26123,
            "unit": "B/op",
            "extra": "25899 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "25899 times\n4 procs"
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
          "id": "a698fcd493f15e7833eede2472c52a30a305bc3b",
          "message": "build(deps): Bump actions/upload-artifact from 4.6.2 to 7.0.0 (#3117)\n\nBumps [actions/upload-artifact](https://github.com/actions/upload-artifact) from 4.6.2 to 7.0.0.\n- [Release notes](https://github.com/actions/upload-artifact/releases)\n- [Commits](https://github.com/actions/upload-artifact/compare/v4.6.2...v7)\n\n---\nupdated-dependencies:\n- dependency-name: actions/upload-artifact\n  dependency-version: 7.0.0\n  dependency-type: direct:production\n  update-type: version-update:semver-major\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>\nCo-authored-by: julienrbrt <julien@rbrt.fr>",
          "timestamp": "2026-03-03T09:39:50+01:00",
          "tree_id": "aec82ed2d9c373e0f79c50e7e4d2db499320a2c6",
          "url": "https://github.com/evstack/ev-node/commit/a698fcd493f15e7833eede2472c52a30a305bc3b"
        },
        "date": 1772527673094,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 40294,
            "unit": "ns/op\t    7049 B/op\t      71 allocs/op",
            "extra": "30508 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 40294,
            "unit": "ns/op",
            "extra": "30508 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7049,
            "unit": "B/op",
            "extra": "30508 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "30508 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 41117,
            "unit": "ns/op\t    7509 B/op\t      81 allocs/op",
            "extra": "29769 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 41117,
            "unit": "ns/op",
            "extra": "29769 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7509,
            "unit": "B/op",
            "extra": "29769 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "29769 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 51473,
            "unit": "ns/op\t   26129 B/op\t      81 allocs/op",
            "extra": "23743 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 51473,
            "unit": "ns/op",
            "extra": "23743 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26129,
            "unit": "B/op",
            "extra": "23743 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "23743 times\n4 procs"
          }
        ]
      }
    ],
    "Spamoor Trace Benchmarks": [
      {
        "commit": {
          "author": {
            "name": "chatton",
            "username": "chatton",
            "email": "github.qpeyb@simplelogin.fr"
          },
          "committer": {
            "name": "chatton",
            "username": "chatton",
            "email": "github.qpeyb@simplelogin.fr"
          },
          "id": "f4a09a2eae5966e209b287614b4ea4c0cf51858b",
          "message": "ci: run benchmarks on PRs without updating baseline",
          "timestamp": "2026-02-24T13:35:00Z",
          "url": "https://github.com/evstack/ev-node/commit/f4a09a2eae5966e209b287614b4ea4c0cf51858b"
        },
        "date": 1771940453407,
        "tool": "customSmallerIsBetter",
        "benches": [
          {
            "name": "SpamoorSmoke - Batch.Commit (avg)",
            "value": 48.32185628742515,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.Commit (min)",
            "value": 22,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.Commit (max)",
            "value": 346,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.Put (avg)",
            "value": 2.7173447537473234,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.Put (min)",
            "value": 1,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.Put (max)",
            "value": 41,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SaveBlockData (avg)",
            "value": 11.889221556886227,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SaveBlockData (min)",
            "value": 6,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SaveBlockData (max)",
            "value": 45,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SetHeight (avg)",
            "value": 9.535928143712574,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SetHeight (min)",
            "value": 4,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SetHeight (max)",
            "value": 36,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.UpdateState (avg)",
            "value": 4.332335329341317,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.UpdateState (min)",
            "value": 2,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.UpdateState (max)",
            "value": 22,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ApplyBlock (avg)",
            "value": 6096.185628742515,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ApplyBlock (min)",
            "value": 3020,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ApplyBlock (max)",
            "value": 75572,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.CreateBlock (avg)",
            "value": 3.3293413173652695,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.CreateBlock (min)",
            "value": 1,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.CreateBlock (max)",
            "value": 26,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ProduceBlock (avg)",
            "value": 6856.468468468468,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ProduceBlock (min)",
            "value": 3679,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ProduceBlock (max)",
            "value": 78275,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.RetrieveBatch (avg)",
            "value": 32.775449101796404,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.RetrieveBatch (min)",
            "value": 12,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.RetrieveBatch (max)",
            "value": 109,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DA.Submit (avg)",
            "value": 654.0113636363636,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DA.Submit (min)",
            "value": 481,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DA.Submit (max)",
            "value": 2652,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitData (avg)",
            "value": 1004.7575757575758,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitData (min)",
            "value": 781,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitData (max)",
            "value": 3018,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitHeaders (avg)",
            "value": 970.5594405594405,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitHeaders (min)",
            "value": 754,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitHeaders (max)",
            "value": 2035,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.ForkchoiceUpdated (avg)",
            "value": 590.9287863590772,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.ForkchoiceUpdated (min)",
            "value": 344,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.ForkchoiceUpdated (max)",
            "value": 21684,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.GetPayload (avg)",
            "value": 1485.7305389221558,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.GetPayload (min)",
            "value": 359,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.GetPayload (max)",
            "value": 28945,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.NewPayload (avg)",
            "value": 1980.2365269461077,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.NewPayload (min)",
            "value": 655,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.NewPayload (max)",
            "value": 20741,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Eth.GetBlockByNumber (avg)",
            "value": 713.8996990972919,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Eth.GetBlockByNumber (min)",
            "value": 346,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Eth.GetBlockByNumber (max)",
            "value": 16904,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.ExecuteTxs (avg)",
            "value": 6085.254491017964,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.ExecuteTxs (min)",
            "value": 3009,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.ExecuteTxs (max)",
            "value": 75559,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.GetTxs (avg)",
            "value": 1134.2,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.GetTxs (min)",
            "value": 551,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.GetTxs (max)",
            "value": 6415,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.SetFinal (avg)",
            "value": 1468.8966565349544,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.SetFinal (min)",
            "value": 900,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.SetFinal (max)",
            "value": 36164,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - ForcedInclusionRetriever.RetrieveForcedIncludedTxs (avg)",
            "value": 4.961077844311378,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - ForcedInclusionRetriever.RetrieveForcedIncludedTxs (min)",
            "value": 2,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - ForcedInclusionRetriever.RetrieveForcedIncludedTxs (max)",
            "value": 20,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.GetNextBatch (avg)",
            "value": 24.64071856287425,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.GetNextBatch (min)",
            "value": 8,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.GetNextBatch (max)",
            "value": 101,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.SubmitBatchTxs (avg)",
            "value": 52.84848484848485,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.SubmitBatchTxs (min)",
            "value": 28,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.SubmitBatchTxs (max)",
            "value": 132,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.DeleteMetadata (avg)",
            "value": 21.62049861495845,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.DeleteMetadata (min)",
            "value": 8,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.DeleteMetadata (max)",
            "value": 109,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetBlockData (avg)",
            "value": 9.355760368663594,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetBlockData (min)",
            "value": 1,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetBlockData (max)",
            "value": 90,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetHeader (avg)",
            "value": 8.402537485582469,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetHeader (min)",
            "value": 1,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetHeader (max)",
            "value": 62,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetMetadata (avg)",
            "value": 6.571637426900585,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetMetadata (min)",
            "value": 1,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetMetadata (max)",
            "value": 246,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.Height (avg)",
            "value": 9.20261031696706,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.Height (min)",
            "value": 1,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.Height (max)",
            "value": 186,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.NewBatch (avg)",
            "value": 8.260479041916168,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.NewBatch (min)",
            "value": 2,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.NewBatch (max)",
            "value": 70,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.SetMetadata (avg)",
            "value": 25.51448879168945,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.SetMetadata (min)",
            "value": 6,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.SetMetadata (max)",
            "value": 1538,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - TxPool.GetTxs (avg)",
            "value": 1117.6285714285714,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - TxPool.GetTxs (min)",
            "value": 535,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - TxPool.GetTxs (max)",
            "value": 6407,
            "unit": "us"
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
          "id": "212ac0881c0c32481afa69d10a6ac571ddfea168",
          "message": "feat: adding spammoor test to benchmark (#3105)\n\n* feat: adding spammoor test to benchmark\n\n* fix: use microseconds in benchmark JSON to preserve sub-millisecond precision\n\n* ci: run benchmarks on PRs without updating baseline\n\n* ci: fan-out/fan-in benchmark jobs to avoid gh-pages race condition\n\n* ci: reset local gh-pages between benchmark steps to avoid fetch conflict\n\n* ci: fix spamoor benchmark artifact download path\n\n* ci: isolate benchmark publish steps so failures don't cascade\n\n* ci: only emit avg in benchmark JSON to reduce alert noise",
          "timestamp": "2026-02-25T11:06:23Z",
          "tree_id": "3fb93a845ff6b98b0a4d9dafdcf27ce575de9257",
          "url": "https://github.com/evstack/ev-node/commit/212ac0881c0c32481afa69d10a6ac571ddfea168"
        },
        "date": 1772018871759,
        "tool": "customSmallerIsBetter",
        "benches": [
          {
            "name": "SpamoorSmoke - Batch.Commit (avg)",
            "value": 48.975903614457835,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.Put (avg)",
            "value": 2.540772532188841,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SaveBlockData (avg)",
            "value": 11.811377245508982,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SetHeight (avg)",
            "value": 9.33933933933934,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.UpdateState (avg)",
            "value": 4.620481927710843,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ApplyBlock (avg)",
            "value": 5910.753753753754,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.CreateBlock (avg)",
            "value": 3.210843373493976,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ProduceBlock (avg)",
            "value": 6611.737160120846,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.RetrieveBatch (avg)",
            "value": 30.260479041916167,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DA.Submit (avg)",
            "value": 636.5232558139535,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitData (avg)",
            "value": 931.4242424242424,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitHeaders (avg)",
            "value": 947.9136690647482,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.ForkchoiceUpdated (avg)",
            "value": 528.5097435897436,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.GetPayload (avg)",
            "value": 1419.4676923076922,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.NewPayload (avg)",
            "value": 1964.2615384615385,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Eth.GetBlockByNumber (avg)",
            "value": 607.3958974358974,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.ExecuteTxs (avg)",
            "value": 5864.043076923077,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.GetTxs (avg)",
            "value": 915,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.SetFinal (avg)",
            "value": 1093.5907692307692,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - ForcedInclusionRetriever.RetrieveForcedIncludedTxs (avg)",
            "value": 4.516516516516517,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.GetNextBatch (avg)",
            "value": 22.94478527607362,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.SubmitBatchTxs (avg)",
            "value": 42.45454545454545,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.DeleteMetadata (avg)",
            "value": 23.02445652173913,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetBlockData (avg)",
            "value": 9.416276894293732,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetHeader (avg)",
            "value": 8.771069182389937,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetMetadata (avg)",
            "value": 6.306730415593968,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.Height (avg)",
            "value": 8.473258706467663,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.NewBatch (avg)",
            "value": 7.635542168674699,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.SetMetadata (avg)",
            "value": 25.514673913043477,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - TxPool.GetTxs (avg)",
            "value": 899.5588235294117,
            "unit": "us"
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
          "id": "52080e92e4b810b47fb23d567470e17f198c1ea6",
          "message": "build: migrate from Make to just as command runner (#3110)\n\n* build: migrate from Make to just as command runner\n\nReplace Makefile + 6 .mk include files with a single justfile.\nUpdate all CI workflows (setup-just action) and docs references.\n\nCo-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>\n\n* build: make `just` list recipes by default\n\nCo-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>\n\n* build: add recipe groupings to justfile\n\nGroups: build, test, proto, lint, codegen, run, tools.\nUses --unsorted to preserve logical ordering within groups.\n\nCo-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>\n\n* build: split justfile into per-group files under .just/\n\nRoot justfile now holds variables and imports.\nRecipe files: build, test, proto, lint, codegen, run, tools.\n\nCo-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>\n\n* fix: use extractions/setup-just@v3 (v4 does not exist)\n\nCo-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>\n\n* fix: remove make dependency from Dockerfiles\n\nInline go build/install commands directly instead of depending on\nmake/just inside the container.\n\nCo-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>\n\n---------\n\nCo-authored-by: Claude Opus 4.6 <noreply@anthropic.com>",
          "timestamp": "2026-02-26T21:30:46+01:00",
          "tree_id": "5f17998a73c0d6103492cbc52e124969dfb6f21a",
          "url": "https://github.com/evstack/ev-node/commit/52080e92e4b810b47fb23d567470e17f198c1ea6"
        },
        "date": 1772138130490,
        "tool": "customSmallerIsBetter",
        "benches": [
          {
            "name": "SpamoorSmoke - Batch.Commit (avg)",
            "value": 46.368263473053894,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.Put (avg)",
            "value": 2.4565217391304346,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SaveBlockData (avg)",
            "value": 11.395209580838323,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SetHeight (avg)",
            "value": 8.847305389221557,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.UpdateState (avg)",
            "value": 4.2844311377245505,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ApplyBlock (avg)",
            "value": 6154.769461077844,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.CreateBlock (avg)",
            "value": 2.7981927710843375,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ProduceBlock (avg)",
            "value": 6906.925149700599,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.RetrieveBatch (avg)",
            "value": 30.847305389221557,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DA.Submit (avg)",
            "value": 657.542372881356,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitData (avg)",
            "value": 968.8181818181819,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitHeaders (avg)",
            "value": 959.6993006993007,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.ForkchoiceUpdated (avg)",
            "value": 570.595166163142,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.GetPayload (avg)",
            "value": 1415.745508982036,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.NewPayload (avg)",
            "value": 1995.1497005988024,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Eth.GetBlockByNumber (avg)",
            "value": 681.8751258811682,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.ExecuteTxs (avg)",
            "value": 6144.377245508982,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.GetTxs (avg)",
            "value": 1052.25,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.SetFinal (avg)",
            "value": 1196.2092307692308,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - ForcedInclusionRetriever.RetrieveForcedIncludedTxs (avg)",
            "value": 4.320359281437126,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.GetNextBatch (avg)",
            "value": 23.434131736526947,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.SubmitBatchTxs (avg)",
            "value": 47.911764705882355,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.DeleteMetadata (avg)",
            "value": 22.022408963585434,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetBlockData (avg)",
            "value": 9.854085603112841,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetHeader (avg)",
            "value": 8.533734939759036,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetMetadata (avg)",
            "value": 6.871944545786209,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.Height (avg)",
            "value": 7.965753424657534,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.NewBatch (avg)",
            "value": 7.748502994011976,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.SetMetadata (avg)",
            "value": 24.562260010970927,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - TxPool.GetTxs (avg)",
            "value": 1030.5555555555557,
            "unit": "us"
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
          "id": "c449847baaad8c3e6378c5bc50818765093de3d2",
          "message": "test: faster block time (#3094)\n\n* test: faster block time\n\n* Faster block times",
          "timestamp": "2026-02-27T08:42:54Z",
          "tree_id": "a9dce02f0dfc6387c181244728fea120a1a09bc6",
          "url": "https://github.com/evstack/ev-node/commit/c449847baaad8c3e6378c5bc50818765093de3d2"
        },
        "date": 1772183112329,
        "tool": "customSmallerIsBetter",
        "benches": [
          {
            "name": "SpamoorSmoke - Batch.Commit (avg)",
            "value": 45.43263473053892,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.Put (avg)",
            "value": 2.495515695067265,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SaveBlockData (avg)",
            "value": 11.892215568862275,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SetHeight (avg)",
            "value": 8.335329341317365,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.UpdateState (avg)",
            "value": 4.793413173652695,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ApplyBlock (avg)",
            "value": 6153.311377245509,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.CreateBlock (avg)",
            "value": 2.4564564564564564,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ProduceBlock (avg)",
            "value": 6907.440119760479,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.RetrieveBatch (avg)",
            "value": 28.236526946107784,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DA.Submit (avg)",
            "value": 659.3011363636364,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitData (avg)",
            "value": 976.7575757575758,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitHeaders (avg)",
            "value": 967.5734265734266,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.ForkchoiceUpdated (avg)",
            "value": 599.3778894472362,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.GetPayload (avg)",
            "value": 1446.131736526946,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.NewPayload (avg)",
            "value": 2046.991017964072,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Eth.GetBlockByNumber (avg)",
            "value": 718.332663316583,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.ExecuteTxs (avg)",
            "value": 6143.0329341317365,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.GetTxs (avg)",
            "value": 1562.4722222222222,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.SetFinal (avg)",
            "value": 1480.8073394495414,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - ForcedInclusionRetriever.RetrieveForcedIncludedTxs (avg)",
            "value": 4.023952095808383,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.GetNextBatch (avg)",
            "value": 21.377245508982035,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.SubmitBatchTxs (avg)",
            "value": 51.05882352941177,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.DeleteMetadata (avg)",
            "value": 23.233983286908078,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetBlockData (avg)",
            "value": 10.0187265917603,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetHeader (avg)",
            "value": 9.463659147869674,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetMetadata (avg)",
            "value": 7.150694952450622,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.Height (avg)",
            "value": 7.906774394033561,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.NewBatch (avg)",
            "value": 7.696107784431137,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.SetMetadata (avg)",
            "value": 24.557260273972602,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - TxPool.GetTxs (avg)",
            "value": 1546.611111111111,
            "unit": "us"
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
          "id": "f73a1241bb0b7926f9cbf3d26ffd35501d0e04b8",
          "message": "docs: rewrite and restructure docs (#3026)\n\n* rewrite and restructure docs\n\n* refernce and change of sequencing concept\n\n* Refactor documentation for data availability layers and node operations\n\n- Updated Celestia guide to clarify prerequisites, installation, and configuration for connecting Evolve chains to Celestia.\n- Enhanced Local DA documentation with installation instructions, configuration options, and use cases for development and testing.\n- Expanded troubleshooting guide with detailed diagnostic commands, common issues, and solutions for node operations.\n- Created comprehensive upgrades guide covering minor and major upgrades, version compatibility, and rollback procedures.\n- Added aggregator node documentation detailing configuration, block production settings, and monitoring options.\n- Introduced attester node overview with configuration and use cases for low-latency applications.\n- Removed outdated light node documentation.\n- Improved formatting and clarity in ev-reth chainspec reference for better readability.\n\n* format\n\n* claenup and comments\n\n* Update block-lifecycle.md\n\n* adjustments\n\n* docs: fix broken links, stale flag prefixes, and formatting issues (#3112)\n\n* docs: fix broken links, stale --rollkit.* flag prefixes, and escaped backticks\n\n- Fix forced-inclusion.md: based.md → based-sequencing.md (2 links)\n- Fix block-lifecycle.md: da.md → data-availability.md (2 links)\n- Fix reference/specs/overview.md: p2p.md → learn/specs/p2p.md\n- Fix running-nodes/full-node.md: relative paths to guides/\n- Fix what-is-evolve.md: remove escaped backticks around ev-node\n- Replace all --rollkit.* flags with --evnode.* in ev-node-config.md\n- Fix visualizer.md: --rollkit.rpc → --evnode.rpc\n\n* docs: fix markdownlint errors (MD040 missing code block languages, MD012 extra blank lines)\n\n- Add 'text' language specifier to 25 fenced code blocks containing ASCII\n  diagrams, log output, and formulas\n- Remove extra blank line in migration-from-cometbft.md\n- Fix non-descriptive link text in deployment.md\n\n* fix and push\n\n---------\n\nCo-authored-by: Alexander Peters <alpe@users.noreply.github.com>",
          "timestamp": "2026-03-02T11:16:10+01:00",
          "tree_id": "a963782a287c23f898a8cf0cf5b35097986d39dd",
          "url": "https://github.com/evstack/ev-node/commit/f73a1241bb0b7926f9cbf3d26ffd35501d0e04b8"
        },
        "date": 1772446882235,
        "tool": "customSmallerIsBetter",
        "benches": [
          {
            "name": "SpamoorSmoke - Batch.Commit (avg)",
            "value": 62.45645645645646,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.Put (avg)",
            "value": 3.2956685499058382,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SaveBlockData (avg)",
            "value": 17.027027027027028,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SetHeight (avg)",
            "value": 13.732732732732734,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.UpdateState (avg)",
            "value": 5.633633633633633,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ApplyBlock (avg)",
            "value": 6836.073619631902,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.CreateBlock (avg)",
            "value": 4.695384615384615,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ProduceBlock (avg)",
            "value": 7776.676923076923,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.RetrieveBatch (avg)",
            "value": 44.18098159509202,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DA.Submit (avg)",
            "value": 723.245810055866,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitData (avg)",
            "value": 1128.5,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitHeaders (avg)",
            "value": 1080.8827586206896,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.ForkchoiceUpdated (avg)",
            "value": 619.7179226069246,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.GetPayload (avg)",
            "value": 1550.4420731707316,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.NewPayload (avg)",
            "value": 2166.8662613981764,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Eth.GetBlockByNumber (avg)",
            "value": 757.5727181544634,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.ExecuteTxs (avg)",
            "value": 6831.740740740741,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.GetTxs (avg)",
            "value": 1265.6857142857143,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.SetFinal (avg)",
            "value": 1257,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - ForcedInclusionRetriever.RetrieveForcedIncludedTxs (avg)",
            "value": 6.746223564954683,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.GetNextBatch (avg)",
            "value": 33.84355828220859,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.SubmitBatchTxs (avg)",
            "value": 57.35294117647059,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.DeleteMetadata (avg)",
            "value": 25.263013698630136,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetBlockData (avg)",
            "value": 9.728465955701395,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetHeader (avg)",
            "value": 9.200985221674877,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetMetadata (avg)",
            "value": 8.188644688644688,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.Height (avg)",
            "value": 11.056627255756068,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.NewBatch (avg)",
            "value": 10.942942942942944,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.SetMetadata (avg)",
            "value": 27.63264192139738,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - TxPool.GetTxs (avg)",
            "value": 1242,
            "unit": "us"
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
          "distinct": false,
          "id": "fe99227d249ed6a5f341fa4124c51e8ab747cbed",
          "message": "refactor: move spamoor benchmark into testify suite (#3107)\n\n* refactor: move spamoor benchmark into testify suite in test/e2e/benchmark\n\n- Create test/e2e/benchmark/ subpackage with SpamoorSuite (testify/suite)\n- Move spamoor smoke test into suite as TestSpamoorSmoke\n- Split helpers into focused files: traces.go, output.go, metrics.go\n- Introduce resultWriter for defer-based benchmark JSON output\n- Export shared symbols from evm_test_common.go for cross-package use\n- Restructure CI to fan-out benchmark jobs and fan-in publishing\n- Run benchmarks on PRs only when benchmark-related files change\n\n* fix: correct BENCH_JSON_OUTPUT path for spamoor benchmark\n\ngo test sets the working directory to the package under test, so the\nenv var should be relative to test/e2e/benchmark/, not test/e2e/.\n\n* fix: place package pattern before test binary flags in benchmark CI\n\ngo test treats all arguments after an unknown flag (--evm-binary) as\ntest binary args, so ./benchmark/ was never recognized as a package\npattern.\n\n* fix: adjust evm-binary path for benchmark subpackage working directory\n\ngo test sets the cwd to the package directory (test/e2e/benchmark/),\nso the binary path needs an extra parent traversal.\n\n* fix: exclude benchmark subpackage from make test-e2e\n\nThe benchmark package doesn't define the --binary flag that test-e2e\npasses. It has its own CI workflow so it doesn't need to run here.\n\n* fix: address PR review feedback for benchmark suite\n\n- make reth tag configurable via EV_RETH_TAG env var (default pr-140)\n- fix OTLP config: remove duplicate env vars, use http/protobuf protocol\n- use require.Eventually for host readiness polling\n- rename requireHTTP to requireHostUp\n- use non-fatal logging in resultWriter.flush deferred context\n- fix stale doc comment (setupCommonEVMEnv -> SetupCommonEVMEnv)\n- rename loop variable to avoid shadowing testing.TB convention\n- add block/internal/executing/** to CI path trigger\n- remove unused require import from output.go\n\n* chore: specify http\n\n* chore: filter out benchmark tests from test-e2e",
          "timestamp": "2026-03-02T11:29:50Z",
          "tree_id": "5287f1bf224b065c9a1ee5efcbe78c1e7eebce47",
          "url": "https://github.com/evstack/ev-node/commit/fe99227d249ed6a5f341fa4124c51e8ab747cbed"
        },
        "date": 1772452500966,
        "tool": "customSmallerIsBetter",
        "benches": [
          {
            "name": "SpamoorSmoke - Batch.Commit (avg)",
            "value": 67.42121212121212,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.Put (avg)",
            "value": 3.2254697286012526,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SaveBlockData (avg)",
            "value": 19.675757575757576,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SetHeight (avg)",
            "value": 15.978787878787879,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.UpdateState (avg)",
            "value": 10.433333333333334,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ApplyBlock (avg)",
            "value": 7830.615151515151,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.CreateBlock (avg)",
            "value": 4.2727272727272725,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ProduceBlock (avg)",
            "value": 8818.139393939395,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.RetrieveBatch (avg)",
            "value": 39.981818181818184,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Creating db provider (avg)",
            "value": 5.74468085106383,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DA.Submit (avg)",
            "value": 769.3465909090909,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitData (avg)",
            "value": 1133.1612903225807,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitHeaders (avg)",
            "value": 1114.3862068965518,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DatabaseProvider::commit (avg)",
            "value": 4662.593639575972,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.ForkchoiceUpdated (avg)",
            "value": 729.716148445336,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.GetPayload (avg)",
            "value": 1660.821212121212,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.NewPayload (avg)",
            "value": 2649.478787878788,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Eth.GetBlockByNumber (avg)",
            "value": 803.8946840521564,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.ExecuteTxs (avg)",
            "value": 7813.063636363637,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.GetTxs (avg)",
            "value": 1382.3513513513512,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.SetFinal (avg)",
            "value": 1322.685459940653,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - ForcedInclusionRetriever.RetrieveForcedIncludedTxs (avg)",
            "value": 6.151515151515151,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Pruner::run_with_provider (avg)",
            "value": 94,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.GetNextBatch (avg)",
            "value": 30.112121212121213,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.SubmitBatchTxs (avg)",
            "value": 49.65625,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - StaticFileProviderRW::finalize (avg)",
            "value": 1312.673733804476,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - StaticFileWriters::finalize (avg)",
            "value": 3983.4734982332157,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Storage trie (avg)",
            "value": 2.0450181629475868,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.DeleteMetadata (avg)",
            "value": 20.418478260869566,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetBlockData (avg)",
            "value": 10.889194139194139,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetHeader (avg)",
            "value": 10.640625,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetMetadata (avg)",
            "value": 8.411373707533235,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.Height (avg)",
            "value": 10.679455445544555,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.NewBatch (avg)",
            "value": 11.97121212121212,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.SetMetadata (avg)",
            "value": 25.91833423472147,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Tx::commit (avg)",
            "value": 666.6219081272085,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - TxPool.GetTxs (avg)",
            "value": 1362.5675675675675,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - accounts (avg)",
            "value": 1.7634408602150538,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - begin (avg)",
            "value": 18.657243816254418,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - build_payload (avg)",
            "value": 1698.7570422535211,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - cache_for (avg)",
            "value": 48.57597173144876,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - calculate_overlay (avg)",
            "value": 5.167844522968198,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - commit (avg)",
            "value": 569.2332155477031,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - compute_state_root_parallel (avg)",
            "value": 621.9363957597174,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - compute_trie_input_task (avg)",
            "value": 388.7985865724382,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - connection (avg)",
            "value": 21.666666666666668,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - contracts (avg)",
            "value": 2.3568904593639575,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - convert_to_block (avg)",
            "value": 16.233215547703182,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - database_provider_ro (avg)",
            "value": 27.011063829787233,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - evm env (avg)",
            "value": 2.904593639575972,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - evm_for_ctx (avg)",
            "value": 90.53225806451613,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execute tx (avg)",
            "value": 731.70625,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execute_block (avg)",
            "value": 981.6855123674911,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execute_tx (avg)",
            "value": 683.15625,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execution (avg)",
            "value": 843.2720848056537,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - finish (avg)",
            "value": 9.519434628975265,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - get_cache_for (avg)",
            "value": 27.872791519434628,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - get_overlay (avg)",
            "value": 10.556595744680852,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - hashed_post_state (avg)",
            "value": 12.614840989399294,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_block_mdbx_only (avg)",
            "value": 86.57597173144876,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_block_or_payload (avg)",
            "value": 2100.6607773851592,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_state (avg)",
            "value": 17.13427561837456,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - merge transitions (avg)",
            "value": 2.6,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_forkchoice_updated (avg)",
            "value": 64.56014150943396,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_new_payload (avg)",
            "value": 2142.3038869257953,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_save_blocks (avg)",
            "value": 6586.713780918728,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - poll_next_event (avg)",
            "value": 7.768563357546409,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - pre execution (avg)",
            "value": 21.042402826855124,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prewarm and caching (avg)",
            "value": 1792.4558303886927,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prewarm tx (avg)",
            "value": 911.971875,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prewarm worker (avg)",
            "value": 3134.048387096774,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prune_segments (avg)",
            "value": 1.9649122807017543,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - save_blocks (avg)",
            "value": 1670.5724381625441,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - save_cache (avg)",
            "value": 762.791519434629,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_all (avg)",
            "value": 5174.258064516129,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_cache_exclusive (avg)",
            "value": 100.53356890459364,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_payload_processor (avg)",
            "value": 104.63250883392226,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - state provider (avg)",
            "value": 38.81625441696113,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - transact_batch (avg)",
            "value": 2830.8709677419356,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - trie_data (avg)",
            "value": 5.019434628975265,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_build (avg)",
            "value": 1751.281690140845,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_connect_buffered_blocks (avg)",
            "value": 6.070671378091872,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_insert_payload (avg)",
            "value": 2118.8904593639577,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - update_history_indices (avg)",
            "value": 21.82685512367491,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - update_pipeline_stages (avg)",
            "value": 27.752650176678443,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_inner (avg)",
            "value": 7.084805653710247,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_post_execution (avg)",
            "value": 1.8480565371024735,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_post_execution_with_hashed_state (avg)",
            "value": 1.2,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_with_state (avg)",
            "value": 1953.494699646643,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_header_against_parent (avg)",
            "value": 1.3215547703180213,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_post_execution (avg)",
            "value": 37.04593639575972,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_transaction (avg)",
            "value": 36.78660049627791,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - wait_cloned (avg)",
            "value": 110.72627737226277,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_blocks_data (avg)",
            "value": 1457.1802120141342,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_hashed_state (avg)",
            "value": 92.41935483870968,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_headers (avg)",
            "value": 27.042402826855124,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_receipts (avg)",
            "value": 28.20141342756184,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_state (avg)",
            "value": 33.74911660777385,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_transactions (avg)",
            "value": 12.063604240282686,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_trie_updates_sorted (avg)",
            "value": 36.44169611307421,
            "unit": "us"
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
          "id": "805f927798c111fcae8a9f2edeb9ab37ee20a353",
          "message": "feat: auto-detect Engine API GetPayload version for Osaka fork (#3113)\n\n* feat: auto-detect Engine API GetPayload version for Osaka fork\n\nGetPayload now automatically selects between engine_getPayloadV4 (Prague)\nand engine_getPayloadV5 (Osaka) by caching the last successful version and\nretrying with the alternative on \"Unsupported fork\" errors (code -38005).\n\nThis handles Prague chains, Osaka-at-genesis chains, and time-based\nPrague-to-Osaka upgrades with zero configuration. At most one extra\nRPC call occurs at the fork transition point.\n\nCo-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>\n\n* fix comments\n\n* rename\n\n---------\n\nCo-authored-by: Claude Opus 4.6 <noreply@anthropic.com>",
          "timestamp": "2026-03-02T11:47:28Z",
          "tree_id": "3b891e4d6b22ff25318d8eff8a63852bc52680ef",
          "url": "https://github.com/evstack/ev-node/commit/805f927798c111fcae8a9f2edeb9ab37ee20a353"
        },
        "date": 1772453265322,
        "tool": "customSmallerIsBetter",
        "benches": [
          {
            "name": "SpamoorSmoke - Batch.Commit (avg)",
            "value": 69.75151515151515,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.Put (avg)",
            "value": 3.5906183368869935,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SaveBlockData (avg)",
            "value": 18.215151515151515,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SetHeight (avg)",
            "value": 18.054545454545455,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.UpdateState (avg)",
            "value": 7.1030303030303035,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ApplyBlock (avg)",
            "value": 8038.281818181818,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.CreateBlock (avg)",
            "value": 4.1030303030303035,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ProduceBlock (avg)",
            "value": 9074.624242424243,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.RetrieveBatch (avg)",
            "value": 42.00606060606061,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Creating db provider (avg)",
            "value": 7.9761702127659575,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DA.Submit (avg)",
            "value": 754.2272727272727,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitData (avg)",
            "value": 1117.8387096774193,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitHeaders (avg)",
            "value": 1115.393103448276,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DatabaseProvider::commit (avg)",
            "value": 3898.705673758865,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.ForkchoiceUpdated (avg)",
            "value": 748.3420260782347,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.GetPayload (avg)",
            "value": 1799.769696969697,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.NewPayload (avg)",
            "value": 2629.2606060606063,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Eth.GetBlockByNumber (avg)",
            "value": 820.3811434302909,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.ExecuteTxs (avg)",
            "value": 8014.890909090909,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.GetTxs (avg)",
            "value": 1316.6486486486488,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.SetFinal (avg)",
            "value": 1355.0771513353116,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - ForcedInclusionRetriever.RetrieveForcedIncludedTxs (avg)",
            "value": 6.281818181818182,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Pruner::run_with_provider (avg)",
            "value": 84.73214285714286,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.GetNextBatch (avg)",
            "value": 31.78787878787879,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.SubmitBatchTxs (avg)",
            "value": 50.625,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - StaticFileProviderRW::finalize (avg)",
            "value": 1084,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - StaticFileWriters::finalize (avg)",
            "value": 3298.4028268551237,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Storage trie (avg)",
            "value": 2.1129729441272214,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.DeleteMetadata (avg)",
            "value": 21.98913043478261,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetBlockData (avg)",
            "value": 11.322553191489362,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetHeader (avg)",
            "value": 11.834517766497463,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetMetadata (avg)",
            "value": 8.677991137370753,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.Height (avg)",
            "value": 10.993800371977681,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.NewBatch (avg)",
            "value": 15.101515151515152,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.SetMetadata (avg)",
            "value": 26.246078961600865,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Tx::commit (avg)",
            "value": 588.3617021276596,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - TxPool.GetTxs (avg)",
            "value": 1293.8918918918919,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - accounts (avg)",
            "value": 1.7374100719424461,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - begin (avg)",
            "value": 18.78014184397163,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - build_payload (avg)",
            "value": 1737.854609929078,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - cache_for (avg)",
            "value": 52.4468085106383,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - calculate_overlay (avg)",
            "value": 5.9787234042553195,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - commit (avg)",
            "value": 486.35460992907804,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - compute_state_root_parallel (avg)",
            "value": 620.3758865248227,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - compute_trie_input_task (avg)",
            "value": 380.822695035461,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - connection (avg)",
            "value": 8.307692307692308,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - contracts (avg)",
            "value": 3.25177304964539,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - convert_to_block (avg)",
            "value": 17.138297872340427,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - database_provider_ro (avg)",
            "value": 31.919148936170213,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - evm env (avg)",
            "value": 3.478723404255319,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - evm_for_ctx (avg)",
            "value": 38.41129032258065,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execute tx (avg)",
            "value": 723.6884735202492,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execute_block (avg)",
            "value": 978.6276595744681,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execute_tx (avg)",
            "value": 704.7102803738318,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execution (avg)",
            "value": 835.4397163120567,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - finish (avg)",
            "value": 9.73049645390071,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - get_cache_for (avg)",
            "value": 30.71276595744681,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - get_overlay (avg)",
            "value": 14.148936170212766,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - hashed_post_state (avg)",
            "value": 10.131205673758865,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_block_mdbx_only (avg)",
            "value": 82.37102473498233,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_block_or_payload (avg)",
            "value": 2094.54609929078,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_state (avg)",
            "value": 18.01063829787234,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - merge transitions (avg)",
            "value": 2.6244541484716155,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_forkchoice_updated (avg)",
            "value": 59.964580873671785,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_new_payload (avg)",
            "value": 2133.0106382978724,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_save_blocks (avg)",
            "value": 5635.4609929078015,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - poll_next_event (avg)",
            "value": 7.372333942717855,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - pre execution (avg)",
            "value": 22.620567375886523,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prewarm and caching (avg)",
            "value": 1798.599290780142,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prewarm tx (avg)",
            "value": 888.0249221183801,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prewarm worker (avg)",
            "value": 2960.9193548387098,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prune_segments (avg)",
            "value": 2.375,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - save_blocks (avg)",
            "value": 1487.4204946996467,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - save_cache (avg)",
            "value": 768.0780141843971,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_all (avg)",
            "value": 5148.193548387097,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_cache_exclusive (avg)",
            "value": 96.87588652482269,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_payload_processor (avg)",
            "value": 101.30496453900709,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - state provider (avg)",
            "value": 39.41489361702128,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - transact_batch (avg)",
            "value": 2639.6290322580644,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - trie_data (avg)",
            "value": 5.273851590106007,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_build (avg)",
            "value": 1792.822695035461,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_connect_buffered_blocks (avg)",
            "value": 5.031914893617022,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_insert_payload (avg)",
            "value": 2111.5141843971633,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - update_history_indices (avg)",
            "value": 25.83745583038869,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - update_pipeline_stages (avg)",
            "value": 31.49469964664311,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_inner (avg)",
            "value": 6.833333333333333,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_post_execution (avg)",
            "value": 1.8865248226950355,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_post_execution_with_hashed_state (avg)",
            "value": 1.3357400722021662,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_with_state (avg)",
            "value": 1942.5425531914893,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_header_against_parent (avg)",
            "value": 1.4113475177304964,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_post_execution (avg)",
            "value": 34.98936170212766,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_transaction (avg)",
            "value": 34.24671916010499,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - wait_cloned (avg)",
            "value": 101.8551724137931,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_blocks_data (avg)",
            "value": 1278.6489361702127,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_hashed_state (avg)",
            "value": 101.3225806451613,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_headers (avg)",
            "value": 33.57243816254417,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_receipts (avg)",
            "value": 36.787234042553195,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_state (avg)",
            "value": 41.332155477031804,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_transactions (avg)",
            "value": 13.067375886524824,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_trie_updates_sorted (avg)",
            "value": 41.639575971731446,
            "unit": "us"
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
          "id": "9af0f905c61d12fbfae49fa79b770e884dbff5e7",
          "message": "feat(sequencer): catchup from base (#3057)\n\n* feat(sequencer): catchup from base\n\n* catch up fixes\n\n* improvements\n\n* update catchup test\n\n* code cleanups\n\n* imp\n\n* only produce 1 base block per da epoch (unless more needed)\n\n* fixes\n\n* updates\n\n* cleanup comments\n\n* updates\n\n* fix failover with local-da change",
          "timestamp": "2026-03-02T22:17:52+01:00",
          "tree_id": "d0f758c41d610b32872577b7545fc908c283d8be",
          "url": "https://github.com/evstack/ev-node/commit/9af0f905c61d12fbfae49fa79b770e884dbff5e7"
        },
        "date": 1772486567706,
        "tool": "customSmallerIsBetter",
        "benches": [
          {
            "name": "SpamoorSmoke - Batch.Commit (avg)",
            "value": 64.77912254160363,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.Put (avg)",
            "value": 3.549889135254989,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SaveBlockData (avg)",
            "value": 17.418181818181818,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SetHeight (avg)",
            "value": 15.306060606060607,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.UpdateState (avg)",
            "value": 8.018181818181818,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ApplyBlock (avg)",
            "value": 8025.90606060606,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.CreateBlock (avg)",
            "value": 3.8277945619335347,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ProduceBlock (avg)",
            "value": 9009.248484848486,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.RetrieveBatch (avg)",
            "value": 41.25377643504532,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Creating db provider (avg)",
            "value": 6.287542662116041,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DA.Submit (avg)",
            "value": 737.1468926553672,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitData (avg)",
            "value": 1070.967741935484,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitHeaders (avg)",
            "value": 1089.5684931506848,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DatabaseProvider::commit (avg)",
            "value": 3747.7580071174375,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.ForkchoiceUpdated (avg)",
            "value": 754.0712851405623,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.GetPayload (avg)",
            "value": 1786.8696969696969,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.NewPayload (avg)",
            "value": 2678.724242424242,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Eth.GetBlockByNumber (avg)",
            "value": 804.8846539618856,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.ExecuteTxs (avg)",
            "value": 8009.1,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.GetTxs (avg)",
            "value": 1298.7297297297298,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.SetFinal (avg)",
            "value": 1357.7432835820896,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - ForcedInclusionRetriever.RetrieveForcedIncludedTxs (avg)",
            "value": 6.099697885196375,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Pruner::run_with_provider (avg)",
            "value": 109.48214285714286,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.GetNextBatch (avg)",
            "value": 31.643504531722055,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.SubmitBatchTxs (avg)",
            "value": 58.4375,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - StaticFileProviderRW::finalize (avg)",
            "value": 1037.3985765124555,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - StaticFileWriters::finalize (avg)",
            "value": 3161.76512455516,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Storage trie (avg)",
            "value": 2.1455912850111187,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.DeleteMetadata (avg)",
            "value": 19.87704918032787,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetBlockData (avg)",
            "value": 11.353974121996304,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetHeader (avg)",
            "value": 10.705632306057385,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetMetadata (avg)",
            "value": 8.285029498525073,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.Height (avg)",
            "value": 10.857585139318886,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.NewBatch (avg)",
            "value": 11.84266263237519,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.SetMetadata (avg)",
            "value": 25.061621621621622,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Tx::commit (avg)",
            "value": 573.4768683274021,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - TxPool.GetTxs (avg)",
            "value": 1277.5405405405406,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - accounts (avg)",
            "value": 1.641025641025641,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - begin (avg)",
            "value": 32.72597864768683,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - build_payload (avg)",
            "value": 1824.8727915194347,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - cache_for (avg)",
            "value": 48.765957446808514,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - calculate_overlay (avg)",
            "value": 4.662522202486678,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - commit (avg)",
            "value": 463.57801418439715,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - compute_state_root_parallel (avg)",
            "value": 670.5195729537367,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - compute_trie_input_task (avg)",
            "value": 396.7722419928826,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - connection (avg)",
            "value": 9.757575757575758,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - contracts (avg)",
            "value": 2.478723404255319,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - convert_to_block (avg)",
            "value": 16.77304964539007,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - database_provider_ro (avg)",
            "value": 27.42419080068143,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - evm env (avg)",
            "value": 3.6595744680851063,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - evm_for_ctx (avg)",
            "value": 34.20161290322581,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execute tx (avg)",
            "value": 762.4205607476636,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execute_block (avg)",
            "value": 1037.177304964539,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execute_tx (avg)",
            "value": 732.8706624605678,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execution (avg)",
            "value": 895.1312056737588,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - finish (avg)",
            "value": 9.322695035460994,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - get_cache_for (avg)",
            "value": 28.30141843971631,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - get_overlay (avg)",
            "value": 10.843989769820972,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - hashed_post_state (avg)",
            "value": 10.121428571428572,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_block_mdbx_only (avg)",
            "value": 84.3950177935943,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_block_or_payload (avg)",
            "value": 2190.9572953736656,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_state (avg)",
            "value": 17.46099290780142,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - merge transitions (avg)",
            "value": 2.6973684210526314,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_forkchoice_updated (avg)",
            "value": 61.80166270783848,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_new_payload (avg)",
            "value": 2229.3950177935944,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_save_blocks (avg)",
            "value": 5435.672597864768,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - poll_next_event (avg)",
            "value": 7.954462289083147,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - pre execution (avg)",
            "value": 21.351063829787233,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prewarm and caching (avg)",
            "value": 1867.7651245551601,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prewarm tx (avg)",
            "value": 1030.745341614907,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prewarm worker (avg)",
            "value": 3378.081300813008,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prune_segments (avg)",
            "value": 4.053571428571429,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - save_blocks (avg)",
            "value": 1405.6049822064058,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - save_cache (avg)",
            "value": 791.7615658362989,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_all (avg)",
            "value": 5843.129032258064,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_cache_exclusive (avg)",
            "value": 92.74733096085409,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_payload_processor (avg)",
            "value": 97.27402135231317,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - state provider (avg)",
            "value": 41.63120567375886,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - transact_batch (avg)",
            "value": 3122.0564516129034,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - trie_data (avg)",
            "value": 5.3113879003558715,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_build (avg)",
            "value": 1877.636042402827,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_connect_buffered_blocks (avg)",
            "value": 5.1886120996441285,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_insert_payload (avg)",
            "value": 2208,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - update_history_indices (avg)",
            "value": 21.558718861209965,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - update_pipeline_stages (avg)",
            "value": 27.16370106761566,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_inner (avg)",
            "value": 7.159574468085107,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_post_execution (avg)",
            "value": 1.8971631205673758,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_post_execution_with_hashed_state (avg)",
            "value": 1.3563636363636364,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_with_state (avg)",
            "value": 2039.715302491103,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_header_against_parent (avg)",
            "value": 1.8617021276595744,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_post_execution (avg)",
            "value": 35.833333333333336,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_transaction (avg)",
            "value": 34.90394088669951,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - wait_cloned (avg)",
            "value": 114.59154929577464,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_blocks_data (avg)",
            "value": 1194.404255319149,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_hashed_state (avg)",
            "value": 85.1,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_headers (avg)",
            "value": 33.43262411347518,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_receipts (avg)",
            "value": 31.21631205673759,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_state (avg)",
            "value": 35.654804270462634,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_transactions (avg)",
            "value": 13.638297872340425,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_trie_updates_sorted (avg)",
            "value": 37.29181494661922,
            "unit": "us"
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
          "id": "036633646c7fea603bdf7f407715a82cea3701dc",
          "message": "build(deps): Bump actions/setup-go from 6.2.0 to 6.3.0 (#3118)\n\nBumps [actions/setup-go](https://github.com/actions/setup-go) from 6.2.0 to 6.3.0.\n- [Release notes](https://github.com/actions/setup-go/releases)\n- [Commits](https://github.com/actions/setup-go/compare/v6.2.0...v6.3.0)\n\n---\nupdated-dependencies:\n- dependency-name: actions/setup-go\n  dependency-version: 6.3.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-03-03T09:15:40+01:00",
          "tree_id": "039d153e6f1b42c23d381d919ef8f2a7bb34636c",
          "url": "https://github.com/evstack/ev-node/commit/036633646c7fea603bdf7f407715a82cea3701dc"
        },
        "date": 1772526028127,
        "tool": "customSmallerIsBetter",
        "benches": [
          {
            "name": "SpamoorSmoke - Batch.Commit (avg)",
            "value": 72.37821482602118,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.Put (avg)",
            "value": 3.4989384288747347,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SaveBlockData (avg)",
            "value": 21.784848484848485,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SetHeight (avg)",
            "value": 21.312121212121212,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.UpdateState (avg)",
            "value": 7.890909090909091,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ApplyBlock (avg)",
            "value": 7926.336363636364,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.CreateBlock (avg)",
            "value": 4.6797583081570995,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ProduceBlock (avg)",
            "value": 8977.70606060606,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.RetrieveBatch (avg)",
            "value": 46.79154078549849,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Creating db provider (avg)",
            "value": 11.665243381725022,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DA.Submit (avg)",
            "value": 789.6285714285714,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitData (avg)",
            "value": 1248.4333333333334,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitHeaders (avg)",
            "value": 1145.1655172413793,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DatabaseProvider::commit (avg)",
            "value": 3902.3936170212764,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.ForkchoiceUpdated (avg)",
            "value": 721.751,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.GetPayload (avg)",
            "value": 1722.0151057401813,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.NewPayload (avg)",
            "value": 2675.7492447129907,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Eth.GetBlockByNumber (avg)",
            "value": 819.554,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.ExecuteTxs (avg)",
            "value": 7902.6,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.GetTxs (avg)",
            "value": 1435.4594594594594,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.SetFinal (avg)",
            "value": 1401.4497041420118,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - ForcedInclusionRetriever.RetrieveForcedIncludedTxs (avg)",
            "value": 6.758308157099698,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Pruner::run_with_provider (avg)",
            "value": 108.58928571428571,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.GetNextBatch (avg)",
            "value": 35.37160120845922,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.SubmitBatchTxs (avg)",
            "value": 71.61290322580645,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - StaticFileProviderRW::finalize (avg)",
            "value": 1086.6560283687943,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - StaticFileWriters::finalize (avg)",
            "value": 3307.946808510638,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Storage trie (avg)",
            "value": 2.0536368246098373,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.DeleteMetadata (avg)",
            "value": 21.41576086956522,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetBlockData (avg)",
            "value": 11.023510971786834,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetHeader (avg)",
            "value": 10.533644859813084,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetMetadata (avg)",
            "value": 9.321138211382113,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.Height (avg)",
            "value": 12.411145510835913,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.NewBatch (avg)",
            "value": 16.081694402420574,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.SetMetadata (avg)",
            "value": 26.92760669908158,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Tx::commit (avg)",
            "value": 575.3439716312057,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - TxPool.GetTxs (avg)",
            "value": 1412.1351351351352,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - accounts (avg)",
            "value": 2.44043321299639,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - begin (avg)",
            "value": 22.43971631205674,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - build_payload (avg)",
            "value": 1722.2570422535211,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - cache_for (avg)",
            "value": 52.0354609929078,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - calculate_overlay (avg)",
            "value": 5.518650088809947,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - commit (avg)",
            "value": 452.65248226950354,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - compute_state_root_parallel (avg)",
            "value": 631.1245551601423,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - compute_trie_input_task (avg)",
            "value": 384.7402135231317,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - connection (avg)",
            "value": 12.8,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - contracts (avg)",
            "value": 2.814946619217082,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - convert_to_block (avg)",
            "value": 17.588652482269502,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - database_provider_ro (avg)",
            "value": 35.56874466268147,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - evm env (avg)",
            "value": 3.97864768683274,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - evm_for_ctx (avg)",
            "value": 33.08064516129032,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execute tx (avg)",
            "value": 712.790625,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execute_block (avg)",
            "value": 983.9113475177305,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execute_tx (avg)",
            "value": 698.575,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execution (avg)",
            "value": 820.1063829787234,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - finish (avg)",
            "value": 10.836879432624114,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - get_cache_for (avg)",
            "value": 29.939501779359432,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - get_overlay (avg)",
            "value": 13.488054607508532,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - hashed_post_state (avg)",
            "value": 12.71985815602837,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_block_mdbx_only (avg)",
            "value": 88.90425531914893,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_block_or_payload (avg)",
            "value": 2147.359430604982,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_state (avg)",
            "value": 20.03191489361702,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - merge transitions (avg)",
            "value": 2.6988847583643123,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_forkchoice_updated (avg)",
            "value": 67.66232227488152,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_new_payload (avg)",
            "value": 2191.0604982206405,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_save_blocks (avg)",
            "value": 5597.078014184397,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - poll_next_event (avg)",
            "value": 8.616020984665052,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - pre execution (avg)",
            "value": 24.95390070921986,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prewarm and caching (avg)",
            "value": 1821.8967971530249,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prewarm tx (avg)",
            "value": 863.46875,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prewarm worker (avg)",
            "value": 2725.8951612903224,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prune_segments (avg)",
            "value": 2.5357142857142856,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - save_blocks (avg)",
            "value": 1417.9751773049645,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - save_cache (avg)",
            "value": 774.9572953736655,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_all (avg)",
            "value": 4988.548387096775,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_cache_exclusive (avg)",
            "value": 96.45551601423487,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_payload_processor (avg)",
            "value": 100.93594306049822,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - state provider (avg)",
            "value": 45.57801418439716,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - transact_batch (avg)",
            "value": 2485.766129032258,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - trie_data (avg)",
            "value": 6.294326241134752,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_build (avg)",
            "value": 1781.5704225352113,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_connect_buffered_blocks (avg)",
            "value": 7.241992882562277,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_insert_payload (avg)",
            "value": 2167.391459074733,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - update_history_indices (avg)",
            "value": 25.00354609929078,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - update_pipeline_stages (avg)",
            "value": 37.840425531914896,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_inner (avg)",
            "value": 7.081560283687943,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_post_execution (avg)",
            "value": 2.1773049645390072,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_post_execution_with_hashed_state (avg)",
            "value": 1.523465703971119,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_with_state (avg)",
            "value": 1984.679715302491,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_header_against_parent (avg)",
            "value": 1.599290780141844,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_post_execution (avg)",
            "value": 39.301418439716315,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_transaction (avg)",
            "value": 43.9009900990099,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - wait_cloned (avg)",
            "value": 102.25084745762712,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_blocks_data (avg)",
            "value": 1193.9716312056737,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_hashed_state (avg)",
            "value": 101.64516129032258,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_headers (avg)",
            "value": 36.769503546099294,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_receipts (avg)",
            "value": 21.51063829787234,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_state (avg)",
            "value": 37.843971631205676,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_transactions (avg)",
            "value": 10.666666666666666,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_trie_updates_sorted (avg)",
            "value": 42.48398576512456,
            "unit": "us"
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
          "id": "e4a68aa598e4ed69c7033bea171c654e63950c2a",
          "message": "build(deps): Bump benchmark-action/github-action-benchmark from 1.20.7 to 1.21.0 (#3120)\n\nbuild(deps): Bump benchmark-action/github-action-benchmark\n\nBumps [benchmark-action/github-action-benchmark](https://github.com/benchmark-action/github-action-benchmark) from 1.20.7 to 1.21.0.\n- [Release notes](https://github.com/benchmark-action/github-action-benchmark/releases)\n- [Changelog](https://github.com/benchmark-action/github-action-benchmark/blob/master/CHANGELOG.md)\n- [Commits](https://github.com/benchmark-action/github-action-benchmark/compare/4bdcce38c94cec68da58d012ac24b7b1155efe8b...a7bc2366eda11037936ea57d811a43b3418d3073)\n\n---\nupdated-dependencies:\n- dependency-name: benchmark-action/github-action-benchmark\n  dependency-version: 1.21.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-03-03T09:18:54+01:00",
          "tree_id": "ad6f70791082c4158ecddf3385d98beccc529371",
          "url": "https://github.com/evstack/ev-node/commit/e4a68aa598e4ed69c7033bea171c654e63950c2a"
        },
        "date": 1772526525480,
        "tool": "customSmallerIsBetter",
        "benches": [
          {
            "name": "SpamoorSmoke - Batch.Commit (avg)",
            "value": 58.5105421686747,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.Put (avg)",
            "value": 2.6244343891402715,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SaveBlockData (avg)",
            "value": 16.210843373493976,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SetHeight (avg)",
            "value": 13.159638554216867,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.UpdateState (avg)",
            "value": 7.240963855421687,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ApplyBlock (avg)",
            "value": 7672.731927710844,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.CreateBlock (avg)",
            "value": 2.891566265060241,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ProduceBlock (avg)",
            "value": 8582.885542168675,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.RetrieveBatch (avg)",
            "value": 33.623493975903614,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Creating db provider (avg)",
            "value": 9.005106382978724,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DA.Submit (avg)",
            "value": 701.7627118644068,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitData (avg)",
            "value": 1078.1935483870968,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitHeaders (avg)",
            "value": 1028.7739726027398,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DatabaseProvider::commit (avg)",
            "value": 3591.452296819788,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.ForkchoiceUpdated (avg)",
            "value": 720.5636910732196,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.GetPayload (avg)",
            "value": 1649.4427710843374,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.NewPayload (avg)",
            "value": 2649.605421686747,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Eth.GetBlockByNumber (avg)",
            "value": 758.1484453360081,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.ExecuteTxs (avg)",
            "value": 7656.921686746988,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.GetTxs (avg)",
            "value": 1183.4864864864865,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.SetFinal (avg)",
            "value": 1279.563253012048,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - ForcedInclusionRetriever.RetrieveForcedIncludedTxs (avg)",
            "value": 4.951807228915663,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Pruner::run_with_provider (avg)",
            "value": 100.875,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.GetNextBatch (avg)",
            "value": 25.545180722891565,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.SubmitBatchTxs (avg)",
            "value": 48.9375,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - StaticFileProviderRW::finalize (avg)",
            "value": 997.4452296819788,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - StaticFileWriters::finalize (avg)",
            "value": 3037.190812720848,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Storage trie (avg)",
            "value": 2.059996313590563,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.DeleteMetadata (avg)",
            "value": 20.286501377410467,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetBlockData (avg)",
            "value": 10.006439742410304,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetHeader (avg)",
            "value": 11.326241134751774,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetMetadata (avg)",
            "value": 7.5793534166054375,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.Height (avg)",
            "value": 9.726256983240223,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.NewBatch (avg)",
            "value": 12.215361445783133,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.SetMetadata (avg)",
            "value": 25.216069489685125,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Tx::commit (avg)",
            "value": 541.5406360424029,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - TxPool.GetTxs (avg)",
            "value": 1159.2972972972973,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - accounts (avg)",
            "value": 1.3992673992673992,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - begin (avg)",
            "value": 17.22695035460993,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - build_payload (avg)",
            "value": 1646.625441696113,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - cache_for (avg)",
            "value": 47.729537366548044,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - calculate_overlay (avg)",
            "value": 5.23758865248227,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - commit (avg)",
            "value": 449.60070671378094,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - compute_state_root_parallel (avg)",
            "value": 681.6205673758865,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - compute_trie_input_task (avg)",
            "value": 348.822695035461,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - connection (avg)",
            "value": 8.555555555555555,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - contracts (avg)",
            "value": 2.234042553191489,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - convert_to_block (avg)",
            "value": 16.00354609929078,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - database_provider_ro (avg)",
            "value": 27.729982964224874,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - evm env (avg)",
            "value": 3.017730496453901,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - evm_for_ctx (avg)",
            "value": 49.33870967741935,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execute tx (avg)",
            "value": 762.0403726708074,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execute_block (avg)",
            "value": 1008.0248226950355,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execute_tx (avg)",
            "value": 703.9584664536741,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execution (avg)",
            "value": 897.8804347826087,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - finish (avg)",
            "value": 9.00709219858156,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - get_cache_for (avg)",
            "value": 28.02127659574468,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - get_overlay (avg)",
            "value": 9.838297872340426,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - hashed_post_state (avg)",
            "value": 10.067615658362989,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_block_mdbx_only (avg)",
            "value": 86.34628975265018,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_block_or_payload (avg)",
            "value": 2161.354609929078,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_state (avg)",
            "value": 16.21276595744681,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - merge transitions (avg)",
            "value": 2.482300884955752,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_forkchoice_updated (avg)",
            "value": 56.3886925795053,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_new_payload (avg)",
            "value": 2205.758865248227,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_save_blocks (avg)",
            "value": 5146.3533568904595,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - poll_next_event (avg)",
            "value": 6.711308921707465,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - pre execution (avg)",
            "value": 19.358156028368793,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prewarm and caching (avg)",
            "value": 1874.514184397163,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prewarm tx (avg)",
            "value": 942.9037267080745,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prewarm worker (avg)",
            "value": 3421.3467741935483,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prune_segments (avg)",
            "value": 2.0892857142857144,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - save_blocks (avg)",
            "value": 1324.886925795053,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - save_cache (avg)",
            "value": 816.8191489361702,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_all (avg)",
            "value": 5433.354838709677,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_cache_exclusive (avg)",
            "value": 82.00709219858156,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_payload_processor (avg)",
            "value": 86.03191489361703,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - state provider (avg)",
            "value": 37.87234042553192,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - transact_batch (avg)",
            "value": 3048.314516129032,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - trie_data (avg)",
            "value": 4.901060070671378,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_build (avg)",
            "value": 1697.86925795053,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_connect_buffered_blocks (avg)",
            "value": 5.294326241134752,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_insert_payload (avg)",
            "value": 2185.0425531914893,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - update_history_indices (avg)",
            "value": 26.166077738515902,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - update_pipeline_stages (avg)",
            "value": 32.34628975265018,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_inner (avg)",
            "value": 6.4787234042553195,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_post_execution (avg)",
            "value": 1.6631205673758864,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_post_execution_with_hashed_state (avg)",
            "value": 1.2815884476534296,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_with_state (avg)",
            "value": 2018.8262411347519,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_header_against_parent (avg)",
            "value": 1.3829787234042554,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_post_execution (avg)",
            "value": 34.195035460992905,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_transaction (avg)",
            "value": 27.172661870503596,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - wait_cloned (avg)",
            "value": 93.86524822695036,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_blocks_data (avg)",
            "value": 1139.7773851590107,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_hashed_state (avg)",
            "value": 69,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_headers (avg)",
            "value": 39.03180212014134,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_receipts (avg)",
            "value": 27.886925795053003,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_state (avg)",
            "value": 34.33922261484099,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_transactions (avg)",
            "value": 11.968197879858657,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_trie_updates_sorted (avg)",
            "value": 36.561837455830386,
            "unit": "us"
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
          "id": "cfb479ad6e6d1cf0cca57dba72ba2234c8c8f713",
          "message": "build(deps): Bump actions/download-artifact from 4.3.0 to 8.0.0 (#3119)\n\nBumps [actions/download-artifact](https://github.com/actions/download-artifact) from 4.3.0 to 8.0.0.\n- [Release notes](https://github.com/actions/download-artifact/releases)\n- [Commits](https://github.com/actions/download-artifact/compare/v4.3.0...v8)\n\n---\nupdated-dependencies:\n- dependency-name: actions/download-artifact\n  dependency-version: 8.0.0\n  dependency-type: direct:production\n  update-type: version-update:semver-major\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-03-03T09:21:04+01:00",
          "tree_id": "cc79b8b73e321429ff0c2cee550491eee84f9285",
          "url": "https://github.com/evstack/ev-node/commit/cfb479ad6e6d1cf0cca57dba72ba2234c8c8f713"
        },
        "date": 1772526554336,
        "tool": "customSmallerIsBetter",
        "benches": [
          {
            "name": "SpamoorSmoke - Batch.Commit (avg)",
            "value": 63.206015037593986,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.Put (avg)",
            "value": 2.8520971302428255,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SaveBlockData (avg)",
            "value": 21.490963855421686,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SetHeight (avg)",
            "value": 12.153614457831326,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.UpdateState (avg)",
            "value": 7.572289156626506,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ApplyBlock (avg)",
            "value": 7504.057057057057,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.CreateBlock (avg)",
            "value": 4.675675675675675,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ProduceBlock (avg)",
            "value": 8443.75,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.RetrieveBatch (avg)",
            "value": 33.912912912912915,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Creating db provider (avg)",
            "value": 7.774576271186441,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DA.Submit (avg)",
            "value": 709.9488636363636,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitData (avg)",
            "value": 1096.4193548387098,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitHeaders (avg)",
            "value": 1035.1241379310345,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DatabaseProvider::commit (avg)",
            "value": 4469.704225352113,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.ForkchoiceUpdated (avg)",
            "value": 820.189378757515,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.GetPayload (avg)",
            "value": 1652.7447447447448,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.NewPayload (avg)",
            "value": 2508.6516516516517,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Eth.GetBlockByNumber (avg)",
            "value": 822.9308617234469,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.ExecuteTxs (avg)",
            "value": 7487.438438438438,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.GetTxs (avg)",
            "value": 1825.4444444444443,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.SetFinal (avg)",
            "value": 1802.816265060241,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - ForcedInclusionRetriever.RetrieveForcedIncludedTxs (avg)",
            "value": 4.903903903903904,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Pruner::run_with_provider (avg)",
            "value": 85.10714285714286,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.GetNextBatch (avg)",
            "value": 25.543543543543542,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.SubmitBatchTxs (avg)",
            "value": 60.41935483870968,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - StaticFileProviderRW::finalize (avg)",
            "value": 1259.588028169014,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - StaticFileWriters::finalize (avg)",
            "value": 3820.1091549295775,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Storage trie (avg)",
            "value": 1.9280728861420426,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.DeleteMetadata (avg)",
            "value": 18.73756906077348,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetBlockData (avg)",
            "value": 10.538022813688213,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetHeader (avg)",
            "value": 10.997621878715815,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetMetadata (avg)",
            "value": 7.4331864904552125,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.Height (avg)",
            "value": 9.417596034696407,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.NewBatch (avg)",
            "value": 11.153153153153154,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.SetMetadata (avg)",
            "value": 24.56917211328976,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Tx::commit (avg)",
            "value": 637.8485915492957,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - TxPool.GetTxs (avg)",
            "value": 1805.9166666666667,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - accounts (avg)",
            "value": 1.5035714285714286,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - begin (avg)",
            "value": 18.052816901408452,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - build_payload (avg)",
            "value": 1651.6947368421052,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - cache_for (avg)",
            "value": 46.86971830985915,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - calculate_overlay (avg)",
            "value": 4.38556338028169,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - commit (avg)",
            "value": 549.1866197183099,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - compute_state_root_parallel (avg)",
            "value": 599.0246478873239,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - compute_trie_input_task (avg)",
            "value": 342.13028169014086,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - connection (avg)",
            "value": 14.5,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - contracts (avg)",
            "value": 2.602112676056338,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - convert_to_block (avg)",
            "value": 15.045936395759718,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - database_provider_ro (avg)",
            "value": 29.165394402035624,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - evm env (avg)",
            "value": 2.8028169014084505,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - evm_for_ctx (avg)",
            "value": 37.524193548387096,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execute tx (avg)",
            "value": 748.6149068322982,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execute_block (avg)",
            "value": 996.5915492957746,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execute_tx (avg)",
            "value": 698.4223602484471,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execution (avg)",
            "value": 865.7605633802817,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - finish (avg)",
            "value": 8.471830985915492,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - get_cache_for (avg)",
            "value": 26.6056338028169,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - get_overlay (avg)",
            "value": 12.03728813559322,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - hashed_post_state (avg)",
            "value": 9.637992831541219,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_block_mdbx_only (avg)",
            "value": 80.2394366197183,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_block_or_payload (avg)",
            "value": 2042.6866197183099,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_state (avg)",
            "value": 17.570422535211268,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - merge transitions (avg)",
            "value": 2.3947368421052633,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_forkchoice_updated (avg)",
            "value": 54.57931844888367,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_new_payload (avg)",
            "value": 2078.211267605634,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_save_blocks (avg)",
            "value": 6229.644366197183,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - poll_next_event (avg)",
            "value": 7.072533654812136,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - pre execution (avg)",
            "value": 19.788732394366196,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prewarm and caching (avg)",
            "value": 1764.8978873239437,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prewarm tx (avg)",
            "value": 923.527950310559,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prewarm worker (avg)",
            "value": 3042.8387096774195,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prune_segments (avg)",
            "value": 1.9649122807017543,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - save_blocks (avg)",
            "value": 1534.2711267605634,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - save_cache (avg)",
            "value": 724.0739436619718,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_all (avg)",
            "value": 5216.290322580645,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_cache_exclusive (avg)",
            "value": 85.78169014084507,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_payload_processor (avg)",
            "value": 89.55985915492958,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - state provider (avg)",
            "value": 37.71478873239437,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - transact_batch (avg)",
            "value": 2786.782258064516,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - trie_data (avg)",
            "value": 4.651408450704225,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_build (avg)",
            "value": 1704.6140350877192,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_connect_buffered_blocks (avg)",
            "value": 4.848591549295775,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_insert_payload (avg)",
            "value": 2058.4154929577467,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - update_history_indices (avg)",
            "value": 22.316901408450704,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - update_pipeline_stages (avg)",
            "value": 30.545774647887324,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_inner (avg)",
            "value": 6.630281690140845,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_post_execution (avg)",
            "value": 1.9647887323943662,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_post_execution_with_hashed_state (avg)",
            "value": 1.0698529411764706,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_with_state (avg)",
            "value": 1914.9084507042253,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_header_against_parent (avg)",
            "value": 1.3274647887323943,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_post_execution (avg)",
            "value": 33.933098591549296,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_transaction (avg)",
            "value": 27.766756032171582,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - wait_cloned (avg)",
            "value": 108.8063872255489,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_blocks_data (avg)",
            "value": 1347.1795774647887,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_hashed_state (avg)",
            "value": 75.12903225806451,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_headers (avg)",
            "value": 41.45422535211268,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_receipts (avg)",
            "value": 26.845070422535212,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_state (avg)",
            "value": 34.735915492957744,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_transactions (avg)",
            "value": 11.950704225352112,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_trie_updates_sorted (avg)",
            "value": 30.781690140845072,
            "unit": "us"
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
          "id": "a698fcd493f15e7833eede2472c52a30a305bc3b",
          "message": "build(deps): Bump actions/upload-artifact from 4.6.2 to 7.0.0 (#3117)\n\nBumps [actions/upload-artifact](https://github.com/actions/upload-artifact) from 4.6.2 to 7.0.0.\n- [Release notes](https://github.com/actions/upload-artifact/releases)\n- [Commits](https://github.com/actions/upload-artifact/compare/v4.6.2...v7)\n\n---\nupdated-dependencies:\n- dependency-name: actions/upload-artifact\n  dependency-version: 7.0.0\n  dependency-type: direct:production\n  update-type: version-update:semver-major\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>\nCo-authored-by: julienrbrt <julien@rbrt.fr>",
          "timestamp": "2026-03-03T09:39:50+01:00",
          "tree_id": "aec82ed2d9c373e0f79c50e7e4d2db499320a2c6",
          "url": "https://github.com/evstack/ev-node/commit/a698fcd493f15e7833eede2472c52a30a305bc3b"
        },
        "date": 1772527675137,
        "tool": "customSmallerIsBetter",
        "benches": [
          {
            "name": "SpamoorSmoke - Batch.Commit (avg)",
            "value": 62.57878787878788,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.Put (avg)",
            "value": 2.868008948545861,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SaveBlockData (avg)",
            "value": 18.384848484848487,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SetHeight (avg)",
            "value": 15.55151515151515,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.UpdateState (avg)",
            "value": 6.721212121212122,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ApplyBlock (avg)",
            "value": 7744.278787878788,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.CreateBlock (avg)",
            "value": 3.4606060606060605,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ProduceBlock (avg)",
            "value": 8703.5,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.RetrieveBatch (avg)",
            "value": 35.35454545454545,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Creating db provider (avg)",
            "value": 5.876805437553101,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DA.Submit (avg)",
            "value": 729.4488636363636,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitData (avg)",
            "value": 1055.2903225806451,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitHeaders (avg)",
            "value": 1071.5379310344827,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DatabaseProvider::commit (avg)",
            "value": 3879.56338028169,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.ForkchoiceUpdated (avg)",
            "value": 704.0501002004008,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.GetPayload (avg)",
            "value": 1816.4636363636364,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.NewPayload (avg)",
            "value": 2662.2545454545457,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Eth.GetBlockByNumber (avg)",
            "value": 743.4889779559119,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.ExecuteTxs (avg)",
            "value": 7727.539393939394,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.GetTxs (avg)",
            "value": 1235.7297297297298,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.SetFinal (avg)",
            "value": 1304.0680473372781,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - ForcedInclusionRetriever.RetrieveForcedIncludedTxs (avg)",
            "value": 5.066666666666666,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Pruner::run_with_provider (avg)",
            "value": 89.73684210526316,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.GetNextBatch (avg)",
            "value": 27.312121212121212,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.SubmitBatchTxs (avg)",
            "value": 49.25,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - StaticFileProviderRW::finalize (avg)",
            "value": 1091.530516431925,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - StaticFileWriters::finalize (avg)",
            "value": 3323.0880281690143,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Storage trie (avg)",
            "value": 1.979787050925812,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.DeleteMetadata (avg)",
            "value": 21.222826086956523,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetBlockData (avg)",
            "value": 11.203935599284437,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetHeader (avg)",
            "value": 11.010845986984815,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetMetadata (avg)",
            "value": 7.83302548947174,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.Height (avg)",
            "value": 10.385856079404466,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.NewBatch (avg)",
            "value": 13.475757575757576,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.SetMetadata (avg)",
            "value": 25.514023732470335,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Tx::commit (avg)",
            "value": 543.75,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - TxPool.GetTxs (avg)",
            "value": 1216.054054054054,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - accounts (avg)",
            "value": 1.7014388489208634,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - begin (avg)",
            "value": 22.440140845070424,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - build_payload (avg)",
            "value": 1714.0669014084508,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - cache_for (avg)",
            "value": 48.71024734982332,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - calculate_overlay (avg)",
            "value": 4.671378091872792,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - commit (avg)",
            "value": 452.61267605633805,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - compute_state_root_parallel (avg)",
            "value": 649.3710247349824,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - compute_trie_input_task (avg)",
            "value": 362.9257950530035,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - connection (avg)",
            "value": 11,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - contracts (avg)",
            "value": 3.0247349823321557,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - convert_to_block (avg)",
            "value": 16.11660777385159,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - database_provider_ro (avg)",
            "value": 28.097706032285473,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - evm env (avg)",
            "value": 2.8586572438162543,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - evm_for_ctx (avg)",
            "value": 40.33870967741935,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execute tx (avg)",
            "value": 694.2554517133956,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execute_block (avg)",
            "value": 935.0918727915194,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execute_tx (avg)",
            "value": 710.6915887850467,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execution (avg)",
            "value": 798.2685512367491,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - finish (avg)",
            "value": 8.692579505300353,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - get_cache_for (avg)",
            "value": 27.96113074204947,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - get_overlay (avg)",
            "value": 11.86321155480034,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - hashed_post_state (avg)",
            "value": 10.060070671378092,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_block_mdbx_only (avg)",
            "value": 93.17957746478874,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_block_or_payload (avg)",
            "value": 2065.0035335689045,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_state (avg)",
            "value": 18.26501766784452,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - merge transitions (avg)",
            "value": 3.1645021645021645,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_forkchoice_updated (avg)",
            "value": 58.54171562867215,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_new_payload (avg)",
            "value": 2102.236749116608,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_save_blocks (avg)",
            "value": 5546.080985915493,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - poll_next_event (avg)",
            "value": 7.490909090909091,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - pre execution (avg)",
            "value": 19.85512367491166,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prewarm and caching (avg)",
            "value": 1770.9116607773851,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prewarm tx (avg)",
            "value": 903.5482866043614,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prewarm worker (avg)",
            "value": 2988.766129032258,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prune_segments (avg)",
            "value": 3.210526315789474,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - save_blocks (avg)",
            "value": 1419.0809859154929,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - save_cache (avg)",
            "value": 786.2014134275619,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_all (avg)",
            "value": 5025.193548387097,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_cache_exclusive (avg)",
            "value": 95.9434628975265,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_payload_processor (avg)",
            "value": 100.31095406360424,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - state provider (avg)",
            "value": 38.22968197879859,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - transact_batch (avg)",
            "value": 2712.4596774193546,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - trie_data (avg)",
            "value": 8.484154929577464,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_build (avg)",
            "value": 1762.8978873239437,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_connect_buffered_blocks (avg)",
            "value": 4.869257950530035,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_insert_payload (avg)",
            "value": 2081.5088339222616,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - update_history_indices (avg)",
            "value": 25.316901408450704,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - update_pipeline_stages (avg)",
            "value": 31.264084507042252,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_inner (avg)",
            "value": 6.282685512367491,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_post_execution (avg)",
            "value": 1.8445229681978799,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_post_execution_with_hashed_state (avg)",
            "value": 1.2454873646209386,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_with_state (avg)",
            "value": 1919.8197879858658,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_header_against_parent (avg)",
            "value": 1.3710247349823321,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_post_execution (avg)",
            "value": 33.69964664310954,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_transaction (avg)",
            "value": 28.179551122194514,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - wait_cloned (avg)",
            "value": 104.52720450281426,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_blocks_data (avg)",
            "value": 1214.3204225352113,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_hashed_state (avg)",
            "value": 86.45161290322581,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_headers (avg)",
            "value": 28.070422535211268,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_receipts (avg)",
            "value": 20.862676056338028,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_state (avg)",
            "value": 32.190140845070424,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_transactions (avg)",
            "value": 8.630281690140846,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_trie_updates_sorted (avg)",
            "value": 45.20422535211268,
            "unit": "us"
          }
        ]
      }
    ]
  }
}