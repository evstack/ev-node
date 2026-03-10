window.BENCHMARK_DATA = {
  "lastUpdate": 1773150148737,
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
          "id": "a02e90bf99ee919746a42051d053c23cd92e8d06",
          "message": "build(deps): Bump the all-go group across 4 directories with 6 updates (#3116)\n\n* build(deps): Bump the all-go group across 4 directories with 6 updates\n\nBumps the all-go group with 3 updates in the / directory: [go.opentelemetry.io/otel](https://github.com/open-telemetry/opentelemetry-go), [go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp](https://github.com/open-telemetry/opentelemetry-go) and [golang.org/x/net](https://github.com/golang/net).\nBumps the all-go group with 1 update in the /execution/evm directory: [go.opentelemetry.io/otel](https://github.com/open-telemetry/opentelemetry-go).\nBumps the all-go group with 1 update in the /execution/grpc directory: [golang.org/x/net](https://github.com/golang/net).\nBumps the all-go group with 1 update in the /test/docker-e2e directory: [github.com/celestiaorg/tastora](https://github.com/celestiaorg/tastora).\n\n\nUpdates `go.opentelemetry.io/otel` from 1.40.0 to 1.41.0\n- [Release notes](https://github.com/open-telemetry/opentelemetry-go/releases)\n- [Changelog](https://github.com/open-telemetry/opentelemetry-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/open-telemetry/opentelemetry-go/compare/v1.40.0...v1.41.0)\n\nUpdates `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp` from 1.40.0 to 1.41.0\n- [Release notes](https://github.com/open-telemetry/opentelemetry-go/releases)\n- [Changelog](https://github.com/open-telemetry/opentelemetry-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/open-telemetry/opentelemetry-go/compare/v1.40.0...v1.41.0)\n\nUpdates `go.opentelemetry.io/otel/sdk` from 1.40.0 to 1.41.0\n- [Release notes](https://github.com/open-telemetry/opentelemetry-go/releases)\n- [Changelog](https://github.com/open-telemetry/opentelemetry-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/open-telemetry/opentelemetry-go/compare/v1.40.0...v1.41.0)\n\nUpdates `go.opentelemetry.io/otel/trace` from 1.40.0 to 1.41.0\n- [Release notes](https://github.com/open-telemetry/opentelemetry-go/releases)\n- [Changelog](https://github.com/open-telemetry/opentelemetry-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/open-telemetry/opentelemetry-go/compare/v1.40.0...v1.41.0)\n\nUpdates `golang.org/x/net` from 0.50.0 to 0.51.0\n- [Commits](https://github.com/golang/net/compare/v0.50.0...v0.51.0)\n\nUpdates `go.opentelemetry.io/otel` from 1.40.0 to 1.41.0\n- [Release notes](https://github.com/open-telemetry/opentelemetry-go/releases)\n- [Changelog](https://github.com/open-telemetry/opentelemetry-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/open-telemetry/opentelemetry-go/compare/v1.40.0...v1.41.0)\n\nUpdates `go.opentelemetry.io/otel/sdk` from 1.40.0 to 1.41.0\n- [Release notes](https://github.com/open-telemetry/opentelemetry-go/releases)\n- [Changelog](https://github.com/open-telemetry/opentelemetry-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/open-telemetry/opentelemetry-go/compare/v1.40.0...v1.41.0)\n\nUpdates `go.opentelemetry.io/otel/trace` from 1.40.0 to 1.41.0\n- [Release notes](https://github.com/open-telemetry/opentelemetry-go/releases)\n- [Changelog](https://github.com/open-telemetry/opentelemetry-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/open-telemetry/opentelemetry-go/compare/v1.40.0...v1.41.0)\n\nUpdates `golang.org/x/net` from 0.50.0 to 0.51.0\n- [Commits](https://github.com/golang/net/compare/v0.50.0...v0.51.0)\n\nUpdates `github.com/celestiaorg/tastora` from 0.15.0 to 0.16.0\n- [Release notes](https://github.com/celestiaorg/tastora/releases)\n- [Commits](https://github.com/celestiaorg/tastora/compare/v0.15.0...v0.16.0)\n\n---\nupdated-dependencies:\n- dependency-name: go.opentelemetry.io/otel\n  dependency-version: 1.41.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp\n  dependency-version: 1.41.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: go.opentelemetry.io/otel/sdk\n  dependency-version: 1.41.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: go.opentelemetry.io/otel/trace\n  dependency-version: 1.41.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: golang.org/x/net\n  dependency-version: 0.51.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: go.opentelemetry.io/otel\n  dependency-version: 1.41.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: go.opentelemetry.io/otel/sdk\n  dependency-version: 1.41.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: go.opentelemetry.io/otel/trace\n  dependency-version: 1.41.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: golang.org/x/net\n  dependency-version: 0.51.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/celestiaorg/tastora\n  dependency-version: 0.16.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\n\n* chore: run just deps after Dependabot update\n\n---------\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>\nCo-authored-by: github-actions[bot] <41898282+github-actions[bot]@users.noreply.github.com>\nCo-authored-by: julienrbrt <julien@rbrt.fr>",
          "timestamp": "2026-03-03T09:40:06+01:00",
          "tree_id": "859cd3366eb40e2ed6583d6e95efe07cdedf87de",
          "url": "https://github.com/evstack/ev-node/commit/a02e90bf99ee919746a42051d053c23cd92e8d06"
        },
        "date": 1772527785189,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 908939746,
            "unit": "ns/op\t26504228 B/op\t  117338 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 908939746,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 26504228,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 117338,
            "unit": "allocs/op",
            "extra": "2 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "eroderust@outlook.com",
            "name": "eroderust",
            "username": "eroderust"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "e877782df0afe0addc8243aa740e6dfb8e1becdb",
          "message": "refactor: use the built-in max/min to simplify the code (#3121)\n\nSigned-off-by: eroderust <eroderust@outlook.com>",
          "timestamp": "2026-03-03T09:41:20+01:00",
          "tree_id": "483c77832c31f15735c302743bbe65ea8ab03a5f",
          "url": "https://github.com/evstack/ev-node/commit/e877782df0afe0addc8243aa740e6dfb8e1becdb"
        },
        "date": 1772527893916,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 905055774,
            "unit": "ns/op\t26872376 B/op\t  122021 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 905055774,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 26872376,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 122021,
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
          "id": "042b75a30eb6427a15c0a98989360b52361ec9cc",
          "message": "feat: ensure p2p DAHint within limits (#3128)\n\n* feat: validate P2P DA height hints in the syncer against the latest DA height.\n\n* Review feedback\n\n* Changelog",
          "timestamp": "2026-03-03T12:25:04+01:00",
          "tree_id": "d250855acfce39ef0287a5840ed9d9db6f8c6329",
          "url": "https://github.com/evstack/ev-node/commit/042b75a30eb6427a15c0a98989360b52361ec9cc"
        },
        "date": 1772537394955,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 904753708,
            "unit": "ns/op\t25970140 B/op\t  112310 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 904753708,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 25970140,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 112310,
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
          "id": "f8fa22ed52b39b68b042b9c08ae0709cf27abdad",
          "message": "feat(benchmarking): adding ERC20 benchmarking test (#3114)\n\n* refactor: move spamoor benchmark into testify suite in test/e2e/benchmark\n\n- Create test/e2e/benchmark/ subpackage with SpamoorSuite (testify/suite)\n- Move spamoor smoke test into suite as TestSpamoorSmoke\n- Split helpers into focused files: traces.go, output.go, metrics.go\n- Introduce resultWriter for defer-based benchmark JSON output\n- Export shared symbols from evm_test_common.go for cross-package use\n- Restructure CI to fan-out benchmark jobs and fan-in publishing\n- Run benchmarks on PRs only when benchmark-related files change\n\n* fix: correct BENCH_JSON_OUTPUT path for spamoor benchmark\n\ngo test sets the working directory to the package under test, so the\nenv var should be relative to test/e2e/benchmark/, not test/e2e/.\n\n* fix: place package pattern before test binary flags in benchmark CI\n\ngo test treats all arguments after an unknown flag (--evm-binary) as\ntest binary args, so ./benchmark/ was never recognized as a package\npattern.\n\n* fix: adjust evm-binary path for benchmark subpackage working directory\n\ngo test sets the cwd to the package directory (test/e2e/benchmark/),\nso the binary path needs an extra parent traversal.\n\n* wip: erc20 benchmark test\n\n* fix: exclude benchmark subpackage from make test-e2e\n\nThe benchmark package doesn't define the --binary flag that test-e2e\npasses. It has its own CI workflow so it doesn't need to run here.\n\n* fix: replace FilterLogs with header iteration and optimize spamoor config\n\ncollectBlockMetrics hit reth's 20K FilterLogs limit at high tx volumes.\nReplace with direct header iteration over [startBlock, endBlock] and add\nPhase 1 metrics: non-empty ratio, block interval p50/p99, gas/block and\ntx/block p50/p99.\n\nOptimize spamoor configuration for 100ms block time:\n- --slot-duration 100ms, --startup-delay 0 on daemon\n- throughput=50 per 100ms slot (500 tx/s per spammer)\n- max_pending=50000 to avoid 3s block poll backpressure\n- 5 staggered spammers with 50K txs each\n\nResults: 55 MGas/s, 1414 TPS, 19.8% non-empty blocks (up from 6%).\n\n* fix: improve benchmark measurement window and reliability\n\n- Move startBlock capture after spammer creation to exclude warm-up\n- Replace 20s drain sleep with smart poll (waitForDrain)\n- Add deleteAllSpammers cleanup to handle stale spamoor DB entries\n- Lower trace sample rate to 10% to prevent Jaeger OOM\n\n* fix: address PR review feedback for benchmark suite\n\n- make reth tag configurable via EV_RETH_TAG env var (default pr-140)\n- fix OTLP config: remove duplicate env vars, use http/protobuf protocol\n- use require.Eventually for host readiness polling\n- rename requireHTTP to requireHostUp\n- use non-fatal logging in resultWriter.flush deferred context\n- fix stale doc comment (setupCommonEVMEnv -> SetupCommonEVMEnv)\n- rename loop variable to avoid shadowing testing.TB convention\n- add block/internal/executing/** to CI path trigger\n- remove unused require import from output.go\n\n* chore: specify http\n\n* chore: filter out benchmark tests from test-e2e\n\n* refactor: centralize reth config and lower ERC20 spammer count\n\nmove EV_RETH_TAG resolution and rpc connection limits into setupEnv\nso all benchmark tests share the same reth configuration. lower ERC20\nspammer count from 5 to 2 to reduce resource contention on local\nhardware while keeping the loop for easy scaling on dedicated infra.\n\n* chore: collect all traces at once\n\n* chore: self review\n\n* refactor: extract benchmark helpers to slim down ERC20 test body\n\n- add blockMetricsSummary with summarize(), log(), and entries() methods\n- add evNodeOverhead() for computing ProduceBlock vs ExecuteTxs overhead\n- add collectTraces() suite method to deduplicate trace collection pattern\n- add addEntries() convenience method on resultWriter\n- slim TestERC20Throughput from ~217 to ~119 lines\n- reuse collectTraces in TestSpamoorSmoke\n\n* docs: add detailed documentation to benchmark helper methods\n\n* ci: add ERC20 throughput benchmark job\n\n* chore: remove span assertions\n\n* fix: guard against drain timeout and zero-duration TPS division\n\n- waitForDrain returns an error on timeout instead of silently logging\n- guard AchievedTPS computation when steady-state duration is zero",
          "timestamp": "2026-03-04T10:52:03Z",
          "tree_id": "2bc9d57d990adf28d62ad86a162eb94c872fb433",
          "url": "https://github.com/evstack/ev-node/commit/f8fa22ed52b39b68b042b9c08ae0709cf27abdad"
        },
        "date": 1772622763442,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 912025088,
            "unit": "ns/op\t26501356 B/op\t  117380 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 912025088,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 26501356,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 117380,
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
          "id": "2c75e9ecfaefd3d5e94cdb978607151cfdae5264",
          "message": "chore: add stricter linting (#3132)\n\n* add stricterlinting\n\n* revert some changes and enable prealloc and predeclared\n\n* ci\n\n* fix lint\n\n* fix comments\n\n* fix\n\n* remove spec\n\n* test stability",
          "timestamp": "2026-03-04T14:05:07+01:00",
          "tree_id": "b07cba63c6dbca067217d5a3aaf172c0310617b9",
          "url": "https://github.com/evstack/ev-node/commit/2c75e9ecfaefd3d5e94cdb978607151cfdae5264"
        },
        "date": 1772629782296,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 909475974,
            "unit": "ns/op\t26737088 B/op\t  117934 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 909475974,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 26737088,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 117934,
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
          "id": "c0bc14150bd7cbc2353136b202269881fbfd0142",
          "message": "fix(node): race on caught up (#3133)\n\nfeat: add `Syncer.PendingCount` and use it to ensure the sync pipeline is drained before marking the node as caught up in failover.",
          "timestamp": "2026-03-04T17:51:20Z",
          "tree_id": "e2de7975904545942b3a234d86d0162e50422142",
          "url": "https://github.com/evstack/ev-node/commit/c0bc14150bd7cbc2353136b202269881fbfd0142"
        },
        "date": 1772647963520,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 915432190,
            "unit": "ns/op\t26235736 B/op\t  112879 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 915432190,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 26235736,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 112879,
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
          "id": "5a07bc3f93f70f4bac64c8ed7a3a3791d563b78f",
          "message": "build(deps): bump core v1.0.0 (#3135)",
          "timestamp": "2026-03-05T07:40:26Z",
          "tree_id": "6a5024806183e9888e830d7b913ae82fc0ed211e",
          "url": "https://github.com/evstack/ev-node/commit/5a07bc3f93f70f4bac64c8ed7a3a3791d563b78f"
        },
        "date": 1772697677068,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 907767640,
            "unit": "ns/op\t26020812 B/op\t  112319 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 907767640,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 26020812,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 112319,
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
          "distinct": false,
          "id": "b1e3010e428a0d1bad3a0ef2fcd8df821a88cc6c",
          "message": "build(deps): Bump rollup from 4.22.4 to 4.59.0 in /docs in the npm_and_yarn group across 1 directory (#3136)\n\nbuild(deps): Bump rollup\n\nBumps the npm_and_yarn group with 1 update in the /docs directory: [rollup](https://github.com/rollup/rollup).\n\n\nUpdates `rollup` from 4.22.4 to 4.59.0\n- [Release notes](https://github.com/rollup/rollup/releases)\n- [Changelog](https://github.com/rollup/rollup/blob/master/CHANGELOG.md)\n- [Commits](https://github.com/rollup/rollup/compare/v4.22.4...v4.59.0)\n\n---\nupdated-dependencies:\n- dependency-name: rollup\n  dependency-version: 4.59.0\n  dependency-type: indirect\n  dependency-group: npm_and_yarn\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-03-05T09:26:29Z",
          "tree_id": "b1eee9dfeaba9650a397d7a0e7052541d2c80540",
          "url": "https://github.com/evstack/ev-node/commit/b1e3010e428a0d1bad3a0ef2fcd8df821a88cc6c"
        },
        "date": 1772704113866,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 914845206,
            "unit": "ns/op\t33987008 B/op\t  198974 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 914845206,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 33987008,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 198974,
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
          "id": "f0eba0d1aed9b775db26d56db07236470786edd7",
          "message": "ci: remove spamoor results from benchmark results per PR (#3138)\n\nchore: remove spamoor results from benchmark results per PR",
          "timestamp": "2026-03-05T11:00:35+01:00",
          "tree_id": "d416d2db76edfd2050ed6e9e643127a91392341c",
          "url": "https://github.com/evstack/ev-node/commit/f0eba0d1aed9b775db26d56db07236470786edd7"
        },
        "date": 1772705054390,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 902601813,
            "unit": "ns/op\t31951576 B/op\t  179632 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 902601813,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 31951576,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 179632,
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
          "distinct": false,
          "id": "a96974bbdfcda99d6f9ac42f998400937eb07774",
          "message": "refactor(store,cache)!: optimize cache restore as O(1) (#3134)\n\n* refactor(store,cache)!: optimize cache restore as O(n)\n\n* lint\n\n* updates\n\n* updates\n\n* updates\n\n* feedback\n\n* assert error\n\n* feedback\n\n* feedback\n\n* feedback + fix in syncer",
          "timestamp": "2026-03-06T10:12:42Z",
          "tree_id": "1fd30667b250611cd2e2425f1163654dc6f6ec8a",
          "url": "https://github.com/evstack/ev-node/commit/a96974bbdfcda99d6f9ac42f998400937eb07774"
        },
        "date": 1772793238364,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 898506882,
            "unit": "ns/op\t32301128 B/op\t  184760 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 898506882,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 32301128,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 184760,
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
          "id": "3b3d5e71a31c2d9fb9f0ae082a847f43ce2325f8",
          "message": "chore: minor deduplication (#3139)\n\n* simplify\n\n* add grpc\n\n* lint\n\n* fix: distinguish store not-found from errors, persist before advancing state\n\nAddress PR review feedback:\n- getMetadataUint64 returns (uint64, bool, error) to distinguish missing\n  keys from backend failures\n- processDAInclusionLoop persists DAIncludedHeightKey before advancing\n  in-memory state to prevent cache deletion on persist failure\n- Expose store.ErrNotFound and store.IsNotFound for clean sentinel checks\n\nCo-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>\n\n---------\n\nCo-authored-by: Claude Opus 4.6 <noreply@anthropic.com>",
          "timestamp": "2026-03-09T07:43:02+01:00",
          "tree_id": "e6def28cea5aa967b065104f0a16daec90bfc4d6",
          "url": "https://github.com/evstack/ev-node/commit/3b3d5e71a31c2d9fb9f0ae082a847f43ce2325f8"
        },
        "date": 1773038777098,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 903692384,
            "unit": "ns/op\t32476084 B/op\t  184787 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 903692384,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 32476084,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 184787,
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
          "id": "3823efd4be1367171fcbf3c614d8c1aebe986e1c",
          "message": "feat(benchmarking): adding gas burner test (#3115)\n\n* refactor: move spamoor benchmark into testify suite in test/e2e/benchmark\n\n- Create test/e2e/benchmark/ subpackage with SpamoorSuite (testify/suite)\n- Move spamoor smoke test into suite as TestSpamoorSmoke\n- Split helpers into focused files: traces.go, output.go, metrics.go\n- Introduce resultWriter for defer-based benchmark JSON output\n- Export shared symbols from evm_test_common.go for cross-package use\n- Restructure CI to fan-out benchmark jobs and fan-in publishing\n- Run benchmarks on PRs only when benchmark-related files change\n\n* fix: correct BENCH_JSON_OUTPUT path for spamoor benchmark\n\ngo test sets the working directory to the package under test, so the\nenv var should be relative to test/e2e/benchmark/, not test/e2e/.\n\n* fix: place package pattern before test binary flags in benchmark CI\n\ngo test treats all arguments after an unknown flag (--evm-binary) as\ntest binary args, so ./benchmark/ was never recognized as a package\npattern.\n\n* fix: adjust evm-binary path for benchmark subpackage working directory\n\ngo test sets the cwd to the package directory (test/e2e/benchmark/),\nso the binary path needs an extra parent traversal.\n\n* wip: erc20 benchmark test\n\n* fix: exclude benchmark subpackage from make test-e2e\n\nThe benchmark package doesn't define the --binary flag that test-e2e\npasses. It has its own CI workflow so it doesn't need to run here.\n\n* fix: replace FilterLogs with header iteration and optimize spamoor config\n\ncollectBlockMetrics hit reth's 20K FilterLogs limit at high tx volumes.\nReplace with direct header iteration over [startBlock, endBlock] and add\nPhase 1 metrics: non-empty ratio, block interval p50/p99, gas/block and\ntx/block p50/p99.\n\nOptimize spamoor configuration for 100ms block time:\n- --slot-duration 100ms, --startup-delay 0 on daemon\n- throughput=50 per 100ms slot (500 tx/s per spammer)\n- max_pending=50000 to avoid 3s block poll backpressure\n- 5 staggered spammers with 50K txs each\n\nResults: 55 MGas/s, 1414 TPS, 19.8% non-empty blocks (up from 6%).\n\n* fix: improve benchmark measurement window and reliability\n\n- Move startBlock capture after spammer creation to exclude warm-up\n- Replace 20s drain sleep with smart poll (waitForDrain)\n- Add deleteAllSpammers cleanup to handle stale spamoor DB entries\n- Lower trace sample rate to 10% to prevent Jaeger OOM\n\n* fix: address PR review feedback for benchmark suite\n\n- make reth tag configurable via EV_RETH_TAG env var (default pr-140)\n- fix OTLP config: remove duplicate env vars, use http/protobuf protocol\n- use require.Eventually for host readiness polling\n- rename requireHTTP to requireHostUp\n- use non-fatal logging in resultWriter.flush deferred context\n- fix stale doc comment (setupCommonEVMEnv -> SetupCommonEVMEnv)\n- rename loop variable to avoid shadowing testing.TB convention\n- add block/internal/executing/** to CI path trigger\n- remove unused require import from output.go\n\n* chore: specify http\n\n* chore: filter out benchmark tests from test-e2e\n\n* refactor: centralize reth config and lower ERC20 spammer count\n\nmove EV_RETH_TAG resolution and rpc connection limits into setupEnv\nso all benchmark tests share the same reth configuration. lower ERC20\nspammer count from 5 to 2 to reduce resource contention on local\nhardware while keeping the loop for easy scaling on dedicated infra.\n\n* chore: collect all traces at once\n\n* chore: self review\n\n* refactor: extract benchmark helpers to slim down ERC20 test body\n\n- add blockMetricsSummary with summarize(), log(), and entries() methods\n- add evNodeOverhead() for computing ProduceBlock vs ExecuteTxs overhead\n- add collectTraces() suite method to deduplicate trace collection pattern\n- add addEntries() convenience method on resultWriter\n- slim TestERC20Throughput from ~217 to ~119 lines\n- reuse collectTraces in TestSpamoorSmoke\n\n* docs: add detailed documentation to benchmark helper methods\n\n* ci: add ERC20 throughput benchmark job\n\n* chore: remove span assertions\n\n* chore: adding gas burner test",
          "timestamp": "2026-03-09T09:30:15Z",
          "tree_id": "e9be3a3536a80303ddcc969e250c656e1d089cba",
          "url": "https://github.com/evstack/ev-node/commit/3823efd4be1367171fcbf3c614d8c1aebe986e1c"
        },
        "date": 1773049904508,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 895881635,
            "unit": "ns/op\t32192768 B/op\t  184459 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 895881635,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 32192768,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 184459,
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
          "id": "067718dd453313220e35a503f9927d3d7dc8f40e",
          "message": "build(deps): Bump dompurify from 3.2.6 to 3.3.2 in /docs in the npm_and_yarn group across 1 directory (#3140)\n\nbuild(deps): Bump dompurify\n\nBumps the npm_and_yarn group with 1 update in the /docs directory: [dompurify](https://github.com/cure53/DOMPurify).\n\n\nUpdates `dompurify` from 3.2.6 to 3.3.2\n- [Release notes](https://github.com/cure53/DOMPurify/releases)\n- [Commits](https://github.com/cure53/DOMPurify/compare/3.2.6...3.3.2)\n\n---\nupdated-dependencies:\n- dependency-name: dompurify\n  dependency-version: 3.3.2\n  dependency-type: indirect\n  dependency-group: npm_and_yarn\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-03-09T10:24:43Z",
          "tree_id": "e200ed5775a4ac2c6294520c4598d9be76602454",
          "url": "https://github.com/evstack/ev-node/commit/067718dd453313220e35a503f9927d3d7dc8f40e"
        },
        "date": 1773053265034,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 904344194,
            "unit": "ns/op\t32146448 B/op\t  179703 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 904344194,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 32146448,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 179703,
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
          "id": "69d2ba8ca6e22000962e18123887c878fa6ea2f1",
          "message": "feat(block): Event-Driven DA Follower with WebSocket Subscriptions (#3131)\n\n* feat: Replace the Syncer's polling DA worker with an event-driven DAFollower and introduce DA client subscription.\n\n* feat: add inline blob processing to DAFollower for zero-latency follow mode\n\nWhen the DA subscription delivers blobs at the current local DA height,\nthe followLoop now processes them inline via ProcessBlobs — avoiding\na round-trip re-fetch from the DA layer.\n\nArchitecture:\n- followLoop: processes subscription blobs inline when caught up (fast path),\n  falls through to catchupLoop when behind (slow path).\n- catchupLoop: unchanged — sequential RetrieveFromDA() for bulk sync.\n\nChanges:\n- Add Blobs field to SubscriptionEvent for carrying raw blob data\n- Add extractBlobData() to DA client Subscribe adapter\n- Export ProcessBlobs on DARetriever interface\n- Add handleSubscriptionEvent() to DAFollower with inline fast path\n- Add TestDAFollower_InlineProcessing with 3 sub-tests\n\n* feat: subscribe to both header and data namespaces for inline processing\n\nWhen header and data use different DA namespaces, the DAFollower now\nsubscribes to both and merges events via a fan-in goroutine. This ensures\ninline blob processing works correctly for split-namespace configurations.\n\nChanges:\n- Add DataNamespace to DAFollowerConfig and daFollower\n- Subscribe to both namespaces in runSubscription with mergeSubscriptions fan-in\n- Guard handleSubscriptionEvent to only advance localDAHeight when\n  ProcessBlobs returns at least one complete event (header+data matched)\n- Pass DataNamespace from syncer.go\n- Implement Subscribe on DummyDA test helper with subscriber notification\n\n* feat: add subscription watchdog to detect stalled DA subscriptions\n\nIf no subscription events arrive within 3× the DA block time (default\n30s), the watchdog triggers and returns an error. The followLoop then\nreconnects the subscription with the standard backoff. This prevents\nthe node from silently stopping sync when the DA subscription stalls\n(e.g., network partition, DA node freeze).\n\n* fix: security hardening for DA subscription path\n\n* feat: Implement blob subscription for local DA and update JSON-RPC client to use WebSockets, along with E2E test updates for new `evnode` flags and P2P address retrieval.\n\n* WS client constructor\n\n* Merge\n\n* Linter\n\n* Review feedback\n\n* Review feedback\n\n* Review feedbac\n\n* Linter",
          "timestamp": "2026-03-09T16:26:47+01:00",
          "tree_id": "c9bde05e4ef642fca7e08d64a012830d83e06a24",
          "url": "https://github.com/evstack/ev-node/commit/69d2ba8ca6e22000962e18123887c878fa6ea2f1"
        },
        "date": 1773070246713,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 914342866,
            "unit": "ns/op\t31243380 B/op\t  170624 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 914342866,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 31243380,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 170624,
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
          "id": "e599cae367b24dc3e290a258cccae8880a8dd5d4",
          "message": "build(deps): bump ev-node (#3144)",
          "timestamp": "2026-03-09T16:37:00+01:00",
          "tree_id": "c2840b2c78f266b5556ecde2fe6cb25e8f5bf57b",
          "url": "https://github.com/evstack/ev-node/commit/e599cae367b24dc3e290a258cccae8880a8dd5d4"
        },
        "date": 1773070962312,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 909878534,
            "unit": "ns/op\t29263032 B/op\t  150936 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 909878534,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 29263032,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 150936,
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
          "id": "e599cae367b24dc3e290a258cccae8880a8dd5d4",
          "message": "build(deps): bump ev-node (#3144)",
          "timestamp": "2026-03-09T16:37:00+01:00",
          "tree_id": "c2840b2c78f266b5556ecde2fe6cb25e8f5bf57b",
          "url": "https://github.com/evstack/ev-node/commit/e599cae367b24dc3e290a258cccae8880a8dd5d4"
        },
        "date": 1773071735350,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 902804264,
            "unit": "ns/op\t32495844 B/op\t  185738 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 902804264,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 32495844,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 185738,
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
          "id": "87d8ba97f048678b3ea06ef9c6f114798d0e33ae",
          "message": "chore: prep evm rc.5 (#3145)",
          "timestamp": "2026-03-09T16:56:02+01:00",
          "tree_id": "af276d6ba1967d3326dc6cbadf29f10cc176d260",
          "url": "https://github.com/evstack/ev-node/commit/87d8ba97f048678b3ea06ef9c6f114798d0e33ae"
        },
        "date": 1773072576628,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 909883440,
            "unit": "ns/op\t33271872 B/op\t  194493 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 909883440,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 33271872,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 194493,
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
          "id": "34db9464a59b6373ba0a8214c86e3d1bac3f7bce",
          "message": "build(deps): Bump actions/setup-go from 6.2.0 to 6.3.0 (#3150)\n\nBumps [actions/setup-go](https://github.com/actions/setup-go) from 6.2.0 to 6.3.0.\n- [Release notes](https://github.com/actions/setup-go/releases)\n- [Commits](https://github.com/actions/setup-go/compare/v6.2.0...v6.3.0)\n\n---\nupdated-dependencies:\n- dependency-name: actions/setup-go\n  dependency-version: 6.3.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-03-10T08:58:57+01:00",
          "tree_id": "63e1ae8fbe7377205ed17dd4d1b616ee75075841",
          "url": "https://github.com/evstack/ev-node/commit/34db9464a59b6373ba0a8214c86e3d1bac3f7bce"
        },
        "date": 1773130380190,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 912876972,
            "unit": "ns/op\t29920340 B/op\t  157269 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 912876972,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 29920340,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 157269,
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
          "id": "5fd82361cdf409a551e5837737911d15275989e7",
          "message": "build(deps): Bump docker/build-push-action from 6 to 7 (#3151)\n\n* build(deps): Bump docker/build-push-action from 6 to 7\n\nBumps [docker/build-push-action](https://github.com/docker/build-push-action) from 6 to 7.\n- [Release notes](https://github.com/docker/build-push-action/releases)\n- [Commits](https://github.com/docker/build-push-action/compare/v6...v7)\n\n---\nupdated-dependencies:\n- dependency-name: docker/build-push-action\n  dependency-version: '7'\n  dependency-type: direct:production\n  update-type: version-update:semver-major\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\n\n* fix flaky test\n\n---------\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>\nCo-authored-by: Julien Robert <julien@rbrt.fr>",
          "timestamp": "2026-03-10T09:01:53+01:00",
          "tree_id": "a8efc6d6486e479c32acaa8718dc824469e1b29f",
          "url": "https://github.com/evstack/ev-node/commit/5fd82361cdf409a551e5837737911d15275989e7"
        },
        "date": 1773130637434,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 905955163,
            "unit": "ns/op\t32897992 B/op\t  189563 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 905955163,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 32897992,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 189563,
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
          "id": "f74b456d8be50afc28432b68199359aeabfee201",
          "message": "build(deps): Bump docker/login-action from 3 to 4 (#3149)\n\nBumps [docker/login-action](https://github.com/docker/login-action) from 3 to 4.\n- [Release notes](https://github.com/docker/login-action/releases)\n- [Commits](https://github.com/docker/login-action/compare/v3...v4)\n\n---\nupdated-dependencies:\n- dependency-name: docker/login-action\n  dependency-version: '4'\n  dependency-type: direct:production\n  update-type: version-update:semver-major\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-03-10T09:02:36+01:00",
          "tree_id": "496fc8fc6c34f1976682d103174068f6746ee509",
          "url": "https://github.com/evstack/ev-node/commit/f74b456d8be50afc28432b68199359aeabfee201"
        },
        "date": 1773130955607,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 906874712,
            "unit": "ns/op\t31872680 B/op\t  179934 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 906874712,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 31872680,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 179934,
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
          "id": "c588547ec5200e454eb8b0fb59091cd92aa969c5",
          "message": "build(deps): Bump the all-go group across 5 directories with 8 updates (#3147)\n\n* build(deps): Bump the all-go group across 5 directories with 8 updates\n\nBumps the all-go group with 3 updates in the / directory: [go.opentelemetry.io/otel](https://github.com/open-telemetry/opentelemetry-go), [go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp](https://github.com/open-telemetry/opentelemetry-go) and [golang.org/x/sync](https://github.com/golang/sync).\nBumps the all-go group with 1 update in the /apps/evm directory: [github.com/ethereum/go-ethereum](https://github.com/ethereum/go-ethereum).\nBumps the all-go group with 2 updates in the /execution/evm directory: [go.opentelemetry.io/otel](https://github.com/open-telemetry/opentelemetry-go) and [github.com/ethereum/go-ethereum](https://github.com/ethereum/go-ethereum).\nBumps the all-go group with 2 updates in the /test/docker-e2e directory: [github.com/ethereum/go-ethereum](https://github.com/ethereum/go-ethereum) and [github.com/evstack/ev-node/execution/evm](https://github.com/evstack/ev-node).\nBumps the all-go group with 2 updates in the /test/e2e directory: [github.com/ethereum/go-ethereum](https://github.com/ethereum/go-ethereum) and [go.opentelemetry.io/proto/otlp](https://github.com/open-telemetry/opentelemetry-proto-go).\n\n\nUpdates `go.opentelemetry.io/otel` from 1.41.0 to 1.42.0\n- [Release notes](https://github.com/open-telemetry/opentelemetry-go/releases)\n- [Changelog](https://github.com/open-telemetry/opentelemetry-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/open-telemetry/opentelemetry-go/compare/v1.41.0...v1.42.0)\n\nUpdates `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp` from 1.41.0 to 1.42.0\n- [Release notes](https://github.com/open-telemetry/opentelemetry-go/releases)\n- [Changelog](https://github.com/open-telemetry/opentelemetry-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/open-telemetry/opentelemetry-go/compare/v1.41.0...v1.42.0)\n\nUpdates `go.opentelemetry.io/otel/sdk` from 1.41.0 to 1.42.0\n- [Release notes](https://github.com/open-telemetry/opentelemetry-go/releases)\n- [Changelog](https://github.com/open-telemetry/opentelemetry-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/open-telemetry/opentelemetry-go/compare/v1.41.0...v1.42.0)\n\nUpdates `go.opentelemetry.io/otel/trace` from 1.41.0 to 1.42.0\n- [Release notes](https://github.com/open-telemetry/opentelemetry-go/releases)\n- [Changelog](https://github.com/open-telemetry/opentelemetry-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/open-telemetry/opentelemetry-go/compare/v1.41.0...v1.42.0)\n\nUpdates `golang.org/x/sync` from 0.19.0 to 0.20.0\n- [Commits](https://github.com/golang/sync/compare/v0.19.0...v0.20.0)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.0 to 1.17.1\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.0...v1.17.1)\n\nUpdates `go.opentelemetry.io/otel` from 1.41.0 to 1.42.0\n- [Release notes](https://github.com/open-telemetry/opentelemetry-go/releases)\n- [Changelog](https://github.com/open-telemetry/opentelemetry-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/open-telemetry/opentelemetry-go/compare/v1.41.0...v1.42.0)\n\nUpdates `go.opentelemetry.io/otel/sdk` from 1.41.0 to 1.42.0\n- [Release notes](https://github.com/open-telemetry/opentelemetry-go/releases)\n- [Changelog](https://github.com/open-telemetry/opentelemetry-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/open-telemetry/opentelemetry-go/compare/v1.41.0...v1.42.0)\n\nUpdates `go.opentelemetry.io/otel/trace` from 1.41.0 to 1.42.0\n- [Release notes](https://github.com/open-telemetry/opentelemetry-go/releases)\n- [Changelog](https://github.com/open-telemetry/opentelemetry-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/open-telemetry/opentelemetry-go/compare/v1.41.0...v1.42.0)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.0 to 1.17.1\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.0...v1.17.1)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.0 to 1.17.1\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.0...v1.17.1)\n\nUpdates `github.com/evstack/ev-node/execution/evm` from 1.0.0-rc.3 to 1.0.0-rc.4\n- [Release notes](https://github.com/evstack/ev-node/releases)\n- [Changelog](https://github.com/evstack/ev-node/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/evstack/ev-node/compare/v1.0.0-rc.3...v1.0.0-rc.4)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.0 to 1.17.1\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.0...v1.17.1)\n\nUpdates `go.opentelemetry.io/proto/otlp` from 1.9.0 to 1.10.0\n- [Release notes](https://github.com/open-telemetry/opentelemetry-proto-go/releases)\n- [Commits](https://github.com/open-telemetry/opentelemetry-proto-go/compare/v1.9.0...v1.10.0)\n\n---\nupdated-dependencies:\n- dependency-name: go.opentelemetry.io/otel\n  dependency-version: 1.42.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp\n  dependency-version: 1.42.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: go.opentelemetry.io/otel/sdk\n  dependency-version: 1.42.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: go.opentelemetry.io/otel/trace\n  dependency-version: 1.42.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: golang.org/x/sync\n  dependency-version: 0.20.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.1\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: go.opentelemetry.io/otel\n  dependency-version: 1.42.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: go.opentelemetry.io/otel/sdk\n  dependency-version: 1.42.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: go.opentelemetry.io/otel/trace\n  dependency-version: 1.42.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.1\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.1\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/evstack/ev-node/execution/evm\n  dependency-version: 1.0.0-rc.4\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.1\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: go.opentelemetry.io/proto/otlp\n  dependency-version: 1.10.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\n\n* chore: run just deps after Dependabot update\n\n---------\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>\nCo-authored-by: github-actions[bot] <41898282+github-actions[bot]@users.noreply.github.com>\nCo-authored-by: julienrbrt <julien@rbrt.fr>",
          "timestamp": "2026-03-10T14:38:12+01:00",
          "tree_id": "d9ededb810577958f7f4b09f1b8569501957b2de",
          "url": "https://github.com/evstack/ev-node/commit/c588547ec5200e454eb8b0fb59091cd92aa969c5"
        },
        "date": 1773150144248,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 912940182,
            "unit": "ns/op\t33212672 B/op\t  189852 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 912940182,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 33212672,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 189852,
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
          "id": "a02e90bf99ee919746a42051d053c23cd92e8d06",
          "message": "build(deps): Bump the all-go group across 4 directories with 6 updates (#3116)\n\n* build(deps): Bump the all-go group across 4 directories with 6 updates\n\nBumps the all-go group with 3 updates in the / directory: [go.opentelemetry.io/otel](https://github.com/open-telemetry/opentelemetry-go), [go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp](https://github.com/open-telemetry/opentelemetry-go) and [golang.org/x/net](https://github.com/golang/net).\nBumps the all-go group with 1 update in the /execution/evm directory: [go.opentelemetry.io/otel](https://github.com/open-telemetry/opentelemetry-go).\nBumps the all-go group with 1 update in the /execution/grpc directory: [golang.org/x/net](https://github.com/golang/net).\nBumps the all-go group with 1 update in the /test/docker-e2e directory: [github.com/celestiaorg/tastora](https://github.com/celestiaorg/tastora).\n\n\nUpdates `go.opentelemetry.io/otel` from 1.40.0 to 1.41.0\n- [Release notes](https://github.com/open-telemetry/opentelemetry-go/releases)\n- [Changelog](https://github.com/open-telemetry/opentelemetry-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/open-telemetry/opentelemetry-go/compare/v1.40.0...v1.41.0)\n\nUpdates `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp` from 1.40.0 to 1.41.0\n- [Release notes](https://github.com/open-telemetry/opentelemetry-go/releases)\n- [Changelog](https://github.com/open-telemetry/opentelemetry-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/open-telemetry/opentelemetry-go/compare/v1.40.0...v1.41.0)\n\nUpdates `go.opentelemetry.io/otel/sdk` from 1.40.0 to 1.41.0\n- [Release notes](https://github.com/open-telemetry/opentelemetry-go/releases)\n- [Changelog](https://github.com/open-telemetry/opentelemetry-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/open-telemetry/opentelemetry-go/compare/v1.40.0...v1.41.0)\n\nUpdates `go.opentelemetry.io/otel/trace` from 1.40.0 to 1.41.0\n- [Release notes](https://github.com/open-telemetry/opentelemetry-go/releases)\n- [Changelog](https://github.com/open-telemetry/opentelemetry-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/open-telemetry/opentelemetry-go/compare/v1.40.0...v1.41.0)\n\nUpdates `golang.org/x/net` from 0.50.0 to 0.51.0\n- [Commits](https://github.com/golang/net/compare/v0.50.0...v0.51.0)\n\nUpdates `go.opentelemetry.io/otel` from 1.40.0 to 1.41.0\n- [Release notes](https://github.com/open-telemetry/opentelemetry-go/releases)\n- [Changelog](https://github.com/open-telemetry/opentelemetry-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/open-telemetry/opentelemetry-go/compare/v1.40.0...v1.41.0)\n\nUpdates `go.opentelemetry.io/otel/sdk` from 1.40.0 to 1.41.0\n- [Release notes](https://github.com/open-telemetry/opentelemetry-go/releases)\n- [Changelog](https://github.com/open-telemetry/opentelemetry-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/open-telemetry/opentelemetry-go/compare/v1.40.0...v1.41.0)\n\nUpdates `go.opentelemetry.io/otel/trace` from 1.40.0 to 1.41.0\n- [Release notes](https://github.com/open-telemetry/opentelemetry-go/releases)\n- [Changelog](https://github.com/open-telemetry/opentelemetry-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/open-telemetry/opentelemetry-go/compare/v1.40.0...v1.41.0)\n\nUpdates `golang.org/x/net` from 0.50.0 to 0.51.0\n- [Commits](https://github.com/golang/net/compare/v0.50.0...v0.51.0)\n\nUpdates `github.com/celestiaorg/tastora` from 0.15.0 to 0.16.0\n- [Release notes](https://github.com/celestiaorg/tastora/releases)\n- [Commits](https://github.com/celestiaorg/tastora/compare/v0.15.0...v0.16.0)\n\n---\nupdated-dependencies:\n- dependency-name: go.opentelemetry.io/otel\n  dependency-version: 1.41.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp\n  dependency-version: 1.41.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: go.opentelemetry.io/otel/sdk\n  dependency-version: 1.41.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: go.opentelemetry.io/otel/trace\n  dependency-version: 1.41.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: golang.org/x/net\n  dependency-version: 0.51.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: go.opentelemetry.io/otel\n  dependency-version: 1.41.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: go.opentelemetry.io/otel/sdk\n  dependency-version: 1.41.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: go.opentelemetry.io/otel/trace\n  dependency-version: 1.41.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: golang.org/x/net\n  dependency-version: 0.51.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/celestiaorg/tastora\n  dependency-version: 0.16.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\n\n* chore: run just deps after Dependabot update\n\n---------\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>\nCo-authored-by: github-actions[bot] <41898282+github-actions[bot]@users.noreply.github.com>\nCo-authored-by: julienrbrt <julien@rbrt.fr>",
          "timestamp": "2026-03-03T09:40:06+01:00",
          "tree_id": "859cd3366eb40e2ed6583d6e95efe07cdedf87de",
          "url": "https://github.com/evstack/ev-node/commit/a02e90bf99ee919746a42051d053c23cd92e8d06"
        },
        "date": 1772527788949,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 38502,
            "unit": "ns/op\t    7022 B/op\t      71 allocs/op",
            "extra": "31536 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 38502,
            "unit": "ns/op",
            "extra": "31536 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7022,
            "unit": "B/op",
            "extra": "31536 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "31536 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 39219,
            "unit": "ns/op\t    7461 B/op\t      81 allocs/op",
            "extra": "31585 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 39219,
            "unit": "ns/op",
            "extra": "31585 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7461,
            "unit": "B/op",
            "extra": "31585 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "31585 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 48766,
            "unit": "ns/op\t   26174 B/op\t      81 allocs/op",
            "extra": "24522 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 48766,
            "unit": "ns/op",
            "extra": "24522 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26174,
            "unit": "B/op",
            "extra": "24522 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "24522 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "eroderust@outlook.com",
            "name": "eroderust",
            "username": "eroderust"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "e877782df0afe0addc8243aa740e6dfb8e1becdb",
          "message": "refactor: use the built-in max/min to simplify the code (#3121)\n\nSigned-off-by: eroderust <eroderust@outlook.com>",
          "timestamp": "2026-03-03T09:41:20+01:00",
          "tree_id": "483c77832c31f15735c302743bbe65ea8ab03a5f",
          "url": "https://github.com/evstack/ev-node/commit/e877782df0afe0addc8243aa740e6dfb8e1becdb"
        },
        "date": 1772527898422,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37099,
            "unit": "ns/op\t    6987 B/op\t      71 allocs/op",
            "extra": "32991 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37099,
            "unit": "ns/op",
            "extra": "32991 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 6987,
            "unit": "B/op",
            "extra": "32991 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "32991 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 37933,
            "unit": "ns/op\t    7446 B/op\t      81 allocs/op",
            "extra": "32182 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 37933,
            "unit": "ns/op",
            "extra": "32182 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7446,
            "unit": "B/op",
            "extra": "32182 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "32182 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 47657,
            "unit": "ns/op\t   26143 B/op\t      81 allocs/op",
            "extra": "25398 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 47657,
            "unit": "ns/op",
            "extra": "25398 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26143,
            "unit": "B/op",
            "extra": "25398 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "25398 times\n4 procs"
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
          "id": "042b75a30eb6427a15c0a98989360b52361ec9cc",
          "message": "feat: ensure p2p DAHint within limits (#3128)\n\n* feat: validate P2P DA height hints in the syncer against the latest DA height.\n\n* Review feedback\n\n* Changelog",
          "timestamp": "2026-03-03T12:25:04+01:00",
          "tree_id": "d250855acfce39ef0287a5840ed9d9db6f8c6329",
          "url": "https://github.com/evstack/ev-node/commit/042b75a30eb6427a15c0a98989360b52361ec9cc"
        },
        "date": 1772537399353,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 38927,
            "unit": "ns/op\t    7024 B/op\t      71 allocs/op",
            "extra": "31458 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 38927,
            "unit": "ns/op",
            "extra": "31458 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7024,
            "unit": "B/op",
            "extra": "31458 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "31458 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 39636,
            "unit": "ns/op\t    7499 B/op\t      81 allocs/op",
            "extra": "30138 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 39636,
            "unit": "ns/op",
            "extra": "30138 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7499,
            "unit": "B/op",
            "extra": "30138 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "30138 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 52450,
            "unit": "ns/op\t   26177 B/op\t      81 allocs/op",
            "extra": "24447 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 52450,
            "unit": "ns/op",
            "extra": "24447 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26177,
            "unit": "B/op",
            "extra": "24447 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "24447 times\n4 procs"
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
          "id": "f8fa22ed52b39b68b042b9c08ae0709cf27abdad",
          "message": "feat(benchmarking): adding ERC20 benchmarking test (#3114)\n\n* refactor: move spamoor benchmark into testify suite in test/e2e/benchmark\n\n- Create test/e2e/benchmark/ subpackage with SpamoorSuite (testify/suite)\n- Move spamoor smoke test into suite as TestSpamoorSmoke\n- Split helpers into focused files: traces.go, output.go, metrics.go\n- Introduce resultWriter for defer-based benchmark JSON output\n- Export shared symbols from evm_test_common.go for cross-package use\n- Restructure CI to fan-out benchmark jobs and fan-in publishing\n- Run benchmarks on PRs only when benchmark-related files change\n\n* fix: correct BENCH_JSON_OUTPUT path for spamoor benchmark\n\ngo test sets the working directory to the package under test, so the\nenv var should be relative to test/e2e/benchmark/, not test/e2e/.\n\n* fix: place package pattern before test binary flags in benchmark CI\n\ngo test treats all arguments after an unknown flag (--evm-binary) as\ntest binary args, so ./benchmark/ was never recognized as a package\npattern.\n\n* fix: adjust evm-binary path for benchmark subpackage working directory\n\ngo test sets the cwd to the package directory (test/e2e/benchmark/),\nso the binary path needs an extra parent traversal.\n\n* wip: erc20 benchmark test\n\n* fix: exclude benchmark subpackage from make test-e2e\n\nThe benchmark package doesn't define the --binary flag that test-e2e\npasses. It has its own CI workflow so it doesn't need to run here.\n\n* fix: replace FilterLogs with header iteration and optimize spamoor config\n\ncollectBlockMetrics hit reth's 20K FilterLogs limit at high tx volumes.\nReplace with direct header iteration over [startBlock, endBlock] and add\nPhase 1 metrics: non-empty ratio, block interval p50/p99, gas/block and\ntx/block p50/p99.\n\nOptimize spamoor configuration for 100ms block time:\n- --slot-duration 100ms, --startup-delay 0 on daemon\n- throughput=50 per 100ms slot (500 tx/s per spammer)\n- max_pending=50000 to avoid 3s block poll backpressure\n- 5 staggered spammers with 50K txs each\n\nResults: 55 MGas/s, 1414 TPS, 19.8% non-empty blocks (up from 6%).\n\n* fix: improve benchmark measurement window and reliability\n\n- Move startBlock capture after spammer creation to exclude warm-up\n- Replace 20s drain sleep with smart poll (waitForDrain)\n- Add deleteAllSpammers cleanup to handle stale spamoor DB entries\n- Lower trace sample rate to 10% to prevent Jaeger OOM\n\n* fix: address PR review feedback for benchmark suite\n\n- make reth tag configurable via EV_RETH_TAG env var (default pr-140)\n- fix OTLP config: remove duplicate env vars, use http/protobuf protocol\n- use require.Eventually for host readiness polling\n- rename requireHTTP to requireHostUp\n- use non-fatal logging in resultWriter.flush deferred context\n- fix stale doc comment (setupCommonEVMEnv -> SetupCommonEVMEnv)\n- rename loop variable to avoid shadowing testing.TB convention\n- add block/internal/executing/** to CI path trigger\n- remove unused require import from output.go\n\n* chore: specify http\n\n* chore: filter out benchmark tests from test-e2e\n\n* refactor: centralize reth config and lower ERC20 spammer count\n\nmove EV_RETH_TAG resolution and rpc connection limits into setupEnv\nso all benchmark tests share the same reth configuration. lower ERC20\nspammer count from 5 to 2 to reduce resource contention on local\nhardware while keeping the loop for easy scaling on dedicated infra.\n\n* chore: collect all traces at once\n\n* chore: self review\n\n* refactor: extract benchmark helpers to slim down ERC20 test body\n\n- add blockMetricsSummary with summarize(), log(), and entries() methods\n- add evNodeOverhead() for computing ProduceBlock vs ExecuteTxs overhead\n- add collectTraces() suite method to deduplicate trace collection pattern\n- add addEntries() convenience method on resultWriter\n- slim TestERC20Throughput from ~217 to ~119 lines\n- reuse collectTraces in TestSpamoorSmoke\n\n* docs: add detailed documentation to benchmark helper methods\n\n* ci: add ERC20 throughput benchmark job\n\n* chore: remove span assertions\n\n* fix: guard against drain timeout and zero-duration TPS division\n\n- waitForDrain returns an error on timeout instead of silently logging\n- guard AchievedTPS computation when steady-state duration is zero",
          "timestamp": "2026-03-04T10:52:03Z",
          "tree_id": "2bc9d57d990adf28d62ad86a162eb94c872fb433",
          "url": "https://github.com/evstack/ev-node/commit/f8fa22ed52b39b68b042b9c08ae0709cf27abdad"
        },
        "date": 1772622767772,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37727,
            "unit": "ns/op\t    6997 B/op\t      71 allocs/op",
            "extra": "32562 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37727,
            "unit": "ns/op",
            "extra": "32562 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 6997,
            "unit": "B/op",
            "extra": "32562 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "32562 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 38039,
            "unit": "ns/op\t    7442 B/op\t      81 allocs/op",
            "extra": "32348 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 38039,
            "unit": "ns/op",
            "extra": "32348 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7442,
            "unit": "B/op",
            "extra": "32348 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "32348 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 52713,
            "unit": "ns/op\t   26153 B/op\t      81 allocs/op",
            "extra": "24056 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 52713,
            "unit": "ns/op",
            "extra": "24056 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26153,
            "unit": "B/op",
            "extra": "24056 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "24056 times\n4 procs"
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
          "id": "2c75e9ecfaefd3d5e94cdb978607151cfdae5264",
          "message": "chore: add stricter linting (#3132)\n\n* add stricterlinting\n\n* revert some changes and enable prealloc and predeclared\n\n* ci\n\n* fix lint\n\n* fix comments\n\n* fix\n\n* remove spec\n\n* test stability",
          "timestamp": "2026-03-04T14:05:07+01:00",
          "tree_id": "b07cba63c6dbca067217d5a3aaf172c0310617b9",
          "url": "https://github.com/evstack/ev-node/commit/2c75e9ecfaefd3d5e94cdb978607151cfdae5264"
        },
        "date": 1772629785783,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37247,
            "unit": "ns/op\t    6991 B/op\t      71 allocs/op",
            "extra": "32814 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37247,
            "unit": "ns/op",
            "extra": "32814 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 6991,
            "unit": "B/op",
            "extra": "32814 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "32814 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 37965,
            "unit": "ns/op\t    7441 B/op\t      81 allocs/op",
            "extra": "32404 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 37965,
            "unit": "ns/op",
            "extra": "32404 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7441,
            "unit": "B/op",
            "extra": "32404 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "32404 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 48071,
            "unit": "ns/op\t   26131 B/op\t      81 allocs/op",
            "extra": "25712 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 48071,
            "unit": "ns/op",
            "extra": "25712 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26131,
            "unit": "B/op",
            "extra": "25712 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "25712 times\n4 procs"
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
          "id": "c0bc14150bd7cbc2353136b202269881fbfd0142",
          "message": "fix(node): race on caught up (#3133)\n\nfeat: add `Syncer.PendingCount` and use it to ensure the sync pipeline is drained before marking the node as caught up in failover.",
          "timestamp": "2026-03-04T17:51:20Z",
          "tree_id": "e2de7975904545942b3a234d86d0162e50422142",
          "url": "https://github.com/evstack/ev-node/commit/c0bc14150bd7cbc2353136b202269881fbfd0142"
        },
        "date": 1772647967153,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 38377,
            "unit": "ns/op\t    7015 B/op\t      71 allocs/op",
            "extra": "31790 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 38377,
            "unit": "ns/op",
            "extra": "31790 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7015,
            "unit": "B/op",
            "extra": "31790 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "31790 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 39374,
            "unit": "ns/op\t    7475 B/op\t      81 allocs/op",
            "extra": "31027 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 39374,
            "unit": "ns/op",
            "extra": "31027 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7475,
            "unit": "B/op",
            "extra": "31027 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "31027 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 49761,
            "unit": "ns/op\t   26173 B/op\t      81 allocs/op",
            "extra": "24266 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 49761,
            "unit": "ns/op",
            "extra": "24266 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26173,
            "unit": "B/op",
            "extra": "24266 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "24266 times\n4 procs"
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
          "id": "5a07bc3f93f70f4bac64c8ed7a3a3791d563b78f",
          "message": "build(deps): bump core v1.0.0 (#3135)",
          "timestamp": "2026-03-05T07:40:26Z",
          "tree_id": "6a5024806183e9888e830d7b913ae82fc0ed211e",
          "url": "https://github.com/evstack/ev-node/commit/5a07bc3f93f70f4bac64c8ed7a3a3791d563b78f"
        },
        "date": 1772697680931,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 38770,
            "unit": "ns/op\t    7013 B/op\t      71 allocs/op",
            "extra": "31868 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 38770,
            "unit": "ns/op",
            "extra": "31868 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7013,
            "unit": "B/op",
            "extra": "31868 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "31868 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 39228,
            "unit": "ns/op\t    7479 B/op\t      81 allocs/op",
            "extra": "30862 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 39228,
            "unit": "ns/op",
            "extra": "30862 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7479,
            "unit": "B/op",
            "extra": "30862 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "30862 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 49339,
            "unit": "ns/op\t   26164 B/op\t      81 allocs/op",
            "extra": "24478 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 49339,
            "unit": "ns/op",
            "extra": "24478 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26164,
            "unit": "B/op",
            "extra": "24478 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "24478 times\n4 procs"
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
          "distinct": false,
          "id": "b1e3010e428a0d1bad3a0ef2fcd8df821a88cc6c",
          "message": "build(deps): Bump rollup from 4.22.4 to 4.59.0 in /docs in the npm_and_yarn group across 1 directory (#3136)\n\nbuild(deps): Bump rollup\n\nBumps the npm_and_yarn group with 1 update in the /docs directory: [rollup](https://github.com/rollup/rollup).\n\n\nUpdates `rollup` from 4.22.4 to 4.59.0\n- [Release notes](https://github.com/rollup/rollup/releases)\n- [Changelog](https://github.com/rollup/rollup/blob/master/CHANGELOG.md)\n- [Commits](https://github.com/rollup/rollup/compare/v4.22.4...v4.59.0)\n\n---\nupdated-dependencies:\n- dependency-name: rollup\n  dependency-version: 4.59.0\n  dependency-type: indirect\n  dependency-group: npm_and_yarn\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-03-05T09:26:29Z",
          "tree_id": "b1eee9dfeaba9650a397d7a0e7052541d2c80540",
          "url": "https://github.com/evstack/ev-node/commit/b1e3010e428a0d1bad3a0ef2fcd8df821a88cc6c"
        },
        "date": 1772704126818,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 35341,
            "unit": "ns/op\t    6970 B/op\t      71 allocs/op",
            "extra": "33715 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 35341,
            "unit": "ns/op",
            "extra": "33715 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 6970,
            "unit": "B/op",
            "extra": "33715 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "33715 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 36546,
            "unit": "ns/op\t    7420 B/op\t      81 allocs/op",
            "extra": "33280 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 36546,
            "unit": "ns/op",
            "extra": "33280 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7420,
            "unit": "B/op",
            "extra": "33280 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "33280 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 45957,
            "unit": "ns/op\t   26112 B/op\t      81 allocs/op",
            "extra": "26208 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 45957,
            "unit": "ns/op",
            "extra": "26208 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26112,
            "unit": "B/op",
            "extra": "26208 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "26208 times\n4 procs"
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
          "id": "f0eba0d1aed9b775db26d56db07236470786edd7",
          "message": "ci: remove spamoor results from benchmark results per PR (#3138)\n\nchore: remove spamoor results from benchmark results per PR",
          "timestamp": "2026-03-05T11:00:35+01:00",
          "tree_id": "d416d2db76edfd2050ed6e9e643127a91392341c",
          "url": "https://github.com/evstack/ev-node/commit/f0eba0d1aed9b775db26d56db07236470786edd7"
        },
        "date": 1772705058288,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 38916,
            "unit": "ns/op\t    7468 B/op\t      81 allocs/op",
            "extra": "31312 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 38916,
            "unit": "ns/op",
            "extra": "31312 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7468,
            "unit": "B/op",
            "extra": "31312 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "31312 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 48941,
            "unit": "ns/op\t   26157 B/op\t      81 allocs/op",
            "extra": "24939 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 48941,
            "unit": "ns/op",
            "extra": "24939 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26157,
            "unit": "B/op",
            "extra": "24939 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "24939 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 38422,
            "unit": "ns/op\t    7008 B/op\t      71 allocs/op",
            "extra": "32103 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 38422,
            "unit": "ns/op",
            "extra": "32103 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7008,
            "unit": "B/op",
            "extra": "32103 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "32103 times\n4 procs"
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
          "distinct": false,
          "id": "a96974bbdfcda99d6f9ac42f998400937eb07774",
          "message": "refactor(store,cache)!: optimize cache restore as O(1) (#3134)\n\n* refactor(store,cache)!: optimize cache restore as O(n)\n\n* lint\n\n* updates\n\n* updates\n\n* updates\n\n* feedback\n\n* assert error\n\n* feedback\n\n* feedback\n\n* feedback + fix in syncer",
          "timestamp": "2026-03-06T10:12:42Z",
          "tree_id": "1fd30667b250611cd2e2425f1163654dc6f6ec8a",
          "url": "https://github.com/evstack/ev-node/commit/a96974bbdfcda99d6f9ac42f998400937eb07774"
        },
        "date": 1772793242009,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 37737,
            "unit": "ns/op\t    7442 B/op\t      81 allocs/op",
            "extra": "32353 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 37737,
            "unit": "ns/op",
            "extra": "32353 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7442,
            "unit": "B/op",
            "extra": "32353 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "32353 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 47828,
            "unit": "ns/op\t   26129 B/op\t      81 allocs/op",
            "extra": "25760 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 47828,
            "unit": "ns/op",
            "extra": "25760 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26129,
            "unit": "B/op",
            "extra": "25760 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "25760 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37736,
            "unit": "ns/op\t    6999 B/op\t      71 allocs/op",
            "extra": "32460 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37736,
            "unit": "ns/op",
            "extra": "32460 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 6999,
            "unit": "B/op",
            "extra": "32460 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "32460 times\n4 procs"
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
          "id": "3b3d5e71a31c2d9fb9f0ae082a847f43ce2325f8",
          "message": "chore: minor deduplication (#3139)\n\n* simplify\n\n* add grpc\n\n* lint\n\n* fix: distinguish store not-found from errors, persist before advancing state\n\nAddress PR review feedback:\n- getMetadataUint64 returns (uint64, bool, error) to distinguish missing\n  keys from backend failures\n- processDAInclusionLoop persists DAIncludedHeightKey before advancing\n  in-memory state to prevent cache deletion on persist failure\n- Expose store.ErrNotFound and store.IsNotFound for clean sentinel checks\n\nCo-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>\n\n---------\n\nCo-authored-by: Claude Opus 4.6 <noreply@anthropic.com>",
          "timestamp": "2026-03-09T07:43:02+01:00",
          "tree_id": "e6def28cea5aa967b065104f0a16daec90bfc4d6",
          "url": "https://github.com/evstack/ev-node/commit/3b3d5e71a31c2d9fb9f0ae082a847f43ce2325f8"
        },
        "date": 1773038781314,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37476,
            "unit": "ns/op\t    6986 B/op\t      71 allocs/op",
            "extra": "33027 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37476,
            "unit": "ns/op",
            "extra": "33027 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 6986,
            "unit": "B/op",
            "extra": "33027 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "33027 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 38354,
            "unit": "ns/op\t    7471 B/op\t      81 allocs/op",
            "extra": "31165 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 38354,
            "unit": "ns/op",
            "extra": "31165 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7471,
            "unit": "B/op",
            "extra": "31165 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "31165 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 47405,
            "unit": "ns/op\t   26157 B/op\t      81 allocs/op",
            "extra": "24951 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 47405,
            "unit": "ns/op",
            "extra": "24951 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26157,
            "unit": "B/op",
            "extra": "24951 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "24951 times\n4 procs"
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
          "id": "3823efd4be1367171fcbf3c614d8c1aebe986e1c",
          "message": "feat(benchmarking): adding gas burner test (#3115)\n\n* refactor: move spamoor benchmark into testify suite in test/e2e/benchmark\n\n- Create test/e2e/benchmark/ subpackage with SpamoorSuite (testify/suite)\n- Move spamoor smoke test into suite as TestSpamoorSmoke\n- Split helpers into focused files: traces.go, output.go, metrics.go\n- Introduce resultWriter for defer-based benchmark JSON output\n- Export shared symbols from evm_test_common.go for cross-package use\n- Restructure CI to fan-out benchmark jobs and fan-in publishing\n- Run benchmarks on PRs only when benchmark-related files change\n\n* fix: correct BENCH_JSON_OUTPUT path for spamoor benchmark\n\ngo test sets the working directory to the package under test, so the\nenv var should be relative to test/e2e/benchmark/, not test/e2e/.\n\n* fix: place package pattern before test binary flags in benchmark CI\n\ngo test treats all arguments after an unknown flag (--evm-binary) as\ntest binary args, so ./benchmark/ was never recognized as a package\npattern.\n\n* fix: adjust evm-binary path for benchmark subpackage working directory\n\ngo test sets the cwd to the package directory (test/e2e/benchmark/),\nso the binary path needs an extra parent traversal.\n\n* wip: erc20 benchmark test\n\n* fix: exclude benchmark subpackage from make test-e2e\n\nThe benchmark package doesn't define the --binary flag that test-e2e\npasses. It has its own CI workflow so it doesn't need to run here.\n\n* fix: replace FilterLogs with header iteration and optimize spamoor config\n\ncollectBlockMetrics hit reth's 20K FilterLogs limit at high tx volumes.\nReplace with direct header iteration over [startBlock, endBlock] and add\nPhase 1 metrics: non-empty ratio, block interval p50/p99, gas/block and\ntx/block p50/p99.\n\nOptimize spamoor configuration for 100ms block time:\n- --slot-duration 100ms, --startup-delay 0 on daemon\n- throughput=50 per 100ms slot (500 tx/s per spammer)\n- max_pending=50000 to avoid 3s block poll backpressure\n- 5 staggered spammers with 50K txs each\n\nResults: 55 MGas/s, 1414 TPS, 19.8% non-empty blocks (up from 6%).\n\n* fix: improve benchmark measurement window and reliability\n\n- Move startBlock capture after spammer creation to exclude warm-up\n- Replace 20s drain sleep with smart poll (waitForDrain)\n- Add deleteAllSpammers cleanup to handle stale spamoor DB entries\n- Lower trace sample rate to 10% to prevent Jaeger OOM\n\n* fix: address PR review feedback for benchmark suite\n\n- make reth tag configurable via EV_RETH_TAG env var (default pr-140)\n- fix OTLP config: remove duplicate env vars, use http/protobuf protocol\n- use require.Eventually for host readiness polling\n- rename requireHTTP to requireHostUp\n- use non-fatal logging in resultWriter.flush deferred context\n- fix stale doc comment (setupCommonEVMEnv -> SetupCommonEVMEnv)\n- rename loop variable to avoid shadowing testing.TB convention\n- add block/internal/executing/** to CI path trigger\n- remove unused require import from output.go\n\n* chore: specify http\n\n* chore: filter out benchmark tests from test-e2e\n\n* refactor: centralize reth config and lower ERC20 spammer count\n\nmove EV_RETH_TAG resolution and rpc connection limits into setupEnv\nso all benchmark tests share the same reth configuration. lower ERC20\nspammer count from 5 to 2 to reduce resource contention on local\nhardware while keeping the loop for easy scaling on dedicated infra.\n\n* chore: collect all traces at once\n\n* chore: self review\n\n* refactor: extract benchmark helpers to slim down ERC20 test body\n\n- add blockMetricsSummary with summarize(), log(), and entries() methods\n- add evNodeOverhead() for computing ProduceBlock vs ExecuteTxs overhead\n- add collectTraces() suite method to deduplicate trace collection pattern\n- add addEntries() convenience method on resultWriter\n- slim TestERC20Throughput from ~217 to ~119 lines\n- reuse collectTraces in TestSpamoorSmoke\n\n* docs: add detailed documentation to benchmark helper methods\n\n* ci: add ERC20 throughput benchmark job\n\n* chore: remove span assertions\n\n* chore: adding gas burner test",
          "timestamp": "2026-03-09T09:30:15Z",
          "tree_id": "e9be3a3536a80303ddcc969e250c656e1d089cba",
          "url": "https://github.com/evstack/ev-node/commit/3823efd4be1367171fcbf3c614d8c1aebe986e1c"
        },
        "date": 1773049909140,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37444,
            "unit": "ns/op\t    6991 B/op\t      71 allocs/op",
            "extra": "32814 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37444,
            "unit": "ns/op",
            "extra": "32814 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 6991,
            "unit": "B/op",
            "extra": "32814 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "32814 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 38129,
            "unit": "ns/op\t    7440 B/op\t      81 allocs/op",
            "extra": "32418 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 38129,
            "unit": "ns/op",
            "extra": "32418 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7440,
            "unit": "B/op",
            "extra": "32418 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "32418 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 47615,
            "unit": "ns/op\t   26132 B/op\t      81 allocs/op",
            "extra": "25676 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 47615,
            "unit": "ns/op",
            "extra": "25676 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26132,
            "unit": "B/op",
            "extra": "25676 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "25676 times\n4 procs"
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
          "id": "067718dd453313220e35a503f9927d3d7dc8f40e",
          "message": "build(deps): Bump dompurify from 3.2.6 to 3.3.2 in /docs in the npm_and_yarn group across 1 directory (#3140)\n\nbuild(deps): Bump dompurify\n\nBumps the npm_and_yarn group with 1 update in the /docs directory: [dompurify](https://github.com/cure53/DOMPurify).\n\n\nUpdates `dompurify` from 3.2.6 to 3.3.2\n- [Release notes](https://github.com/cure53/DOMPurify/releases)\n- [Commits](https://github.com/cure53/DOMPurify/compare/3.2.6...3.3.2)\n\n---\nupdated-dependencies:\n- dependency-name: dompurify\n  dependency-version: 3.3.2\n  dependency-type: indirect\n  dependency-group: npm_and_yarn\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-03-09T10:24:43Z",
          "tree_id": "e200ed5775a4ac2c6294520c4598d9be76602454",
          "url": "https://github.com/evstack/ev-node/commit/067718dd453313220e35a503f9927d3d7dc8f40e"
        },
        "date": 1773053269229,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 40956,
            "unit": "ns/op\t    7062 B/op\t      71 allocs/op",
            "extra": "30022 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 40956,
            "unit": "ns/op",
            "extra": "30022 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7062,
            "unit": "B/op",
            "extra": "30022 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "30022 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 41932,
            "unit": "ns/op\t    7525 B/op\t      81 allocs/op",
            "extra": "29229 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 41932,
            "unit": "ns/op",
            "extra": "29229 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7525,
            "unit": "B/op",
            "extra": "29229 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "29229 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 52623,
            "unit": "ns/op\t   26086 B/op\t      81 allocs/op",
            "extra": "23464 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 52623,
            "unit": "ns/op",
            "extra": "23464 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26086,
            "unit": "B/op",
            "extra": "23464 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "23464 times\n4 procs"
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
          "id": "69d2ba8ca6e22000962e18123887c878fa6ea2f1",
          "message": "feat(block): Event-Driven DA Follower with WebSocket Subscriptions (#3131)\n\n* feat: Replace the Syncer's polling DA worker with an event-driven DAFollower and introduce DA client subscription.\n\n* feat: add inline blob processing to DAFollower for zero-latency follow mode\n\nWhen the DA subscription delivers blobs at the current local DA height,\nthe followLoop now processes them inline via ProcessBlobs — avoiding\na round-trip re-fetch from the DA layer.\n\nArchitecture:\n- followLoop: processes subscription blobs inline when caught up (fast path),\n  falls through to catchupLoop when behind (slow path).\n- catchupLoop: unchanged — sequential RetrieveFromDA() for bulk sync.\n\nChanges:\n- Add Blobs field to SubscriptionEvent for carrying raw blob data\n- Add extractBlobData() to DA client Subscribe adapter\n- Export ProcessBlobs on DARetriever interface\n- Add handleSubscriptionEvent() to DAFollower with inline fast path\n- Add TestDAFollower_InlineProcessing with 3 sub-tests\n\n* feat: subscribe to both header and data namespaces for inline processing\n\nWhen header and data use different DA namespaces, the DAFollower now\nsubscribes to both and merges events via a fan-in goroutine. This ensures\ninline blob processing works correctly for split-namespace configurations.\n\nChanges:\n- Add DataNamespace to DAFollowerConfig and daFollower\n- Subscribe to both namespaces in runSubscription with mergeSubscriptions fan-in\n- Guard handleSubscriptionEvent to only advance localDAHeight when\n  ProcessBlobs returns at least one complete event (header+data matched)\n- Pass DataNamespace from syncer.go\n- Implement Subscribe on DummyDA test helper with subscriber notification\n\n* feat: add subscription watchdog to detect stalled DA subscriptions\n\nIf no subscription events arrive within 3× the DA block time (default\n30s), the watchdog triggers and returns an error. The followLoop then\nreconnects the subscription with the standard backoff. This prevents\nthe node from silently stopping sync when the DA subscription stalls\n(e.g., network partition, DA node freeze).\n\n* fix: security hardening for DA subscription path\n\n* feat: Implement blob subscription for local DA and update JSON-RPC client to use WebSockets, along with E2E test updates for new `evnode` flags and P2P address retrieval.\n\n* WS client constructor\n\n* Merge\n\n* Linter\n\n* Review feedback\n\n* Review feedback\n\n* Review feedbac\n\n* Linter",
          "timestamp": "2026-03-09T16:26:47+01:00",
          "tree_id": "c9bde05e4ef642fca7e08d64a012830d83e06a24",
          "url": "https://github.com/evstack/ev-node/commit/69d2ba8ca6e22000962e18123887c878fa6ea2f1"
        },
        "date": 1773070251333,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 47108,
            "unit": "ns/op\t   26116 B/op\t      81 allocs/op",
            "extra": "26104 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 47108,
            "unit": "ns/op",
            "extra": "26104 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26116,
            "unit": "B/op",
            "extra": "26104 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "26104 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37181,
            "unit": "ns/op\t    6997 B/op\t      71 allocs/op",
            "extra": "32535 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37181,
            "unit": "ns/op",
            "extra": "32535 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 6997,
            "unit": "B/op",
            "extra": "32535 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "32535 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 38628,
            "unit": "ns/op\t    7447 B/op\t      81 allocs/op",
            "extra": "32132 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 38628,
            "unit": "ns/op",
            "extra": "32132 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7447,
            "unit": "B/op",
            "extra": "32132 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "32132 times\n4 procs"
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
          "id": "e599cae367b24dc3e290a258cccae8880a8dd5d4",
          "message": "build(deps): bump ev-node (#3144)",
          "timestamp": "2026-03-09T16:37:00+01:00",
          "tree_id": "c2840b2c78f266b5556ecde2fe6cb25e8f5bf57b",
          "url": "https://github.com/evstack/ev-node/commit/e599cae367b24dc3e290a258cccae8880a8dd5d4"
        },
        "date": 1773070965896,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 40497,
            "unit": "ns/op\t    7055 B/op\t      71 allocs/op",
            "extra": "30288 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 40497,
            "unit": "ns/op",
            "extra": "30288 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7055,
            "unit": "B/op",
            "extra": "30288 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "30288 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 41131,
            "unit": "ns/op\t    7502 B/op\t      81 allocs/op",
            "extra": "30044 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 41131,
            "unit": "ns/op",
            "extra": "30044 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7502,
            "unit": "B/op",
            "extra": "30044 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "30044 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 51520,
            "unit": "ns/op\t   26160 B/op\t      81 allocs/op",
            "extra": "24093 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 51520,
            "unit": "ns/op",
            "extra": "24093 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26160,
            "unit": "B/op",
            "extra": "24093 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "24093 times\n4 procs"
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
          "id": "e599cae367b24dc3e290a258cccae8880a8dd5d4",
          "message": "build(deps): bump ev-node (#3144)",
          "timestamp": "2026-03-09T16:37:00+01:00",
          "tree_id": "c2840b2c78f266b5556ecde2fe6cb25e8f5bf57b",
          "url": "https://github.com/evstack/ev-node/commit/e599cae367b24dc3e290a258cccae8880a8dd5d4"
        },
        "date": 1773071738877,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 36757,
            "unit": "ns/op\t    6987 B/op\t      71 allocs/op",
            "extra": "32980 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 36757,
            "unit": "ns/op",
            "extra": "32980 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 6987,
            "unit": "B/op",
            "extra": "32980 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "32980 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 37439,
            "unit": "ns/op\t    7444 B/op\t      81 allocs/op",
            "extra": "32245 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 37439,
            "unit": "ns/op",
            "extra": "32245 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7444,
            "unit": "B/op",
            "extra": "32245 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "32245 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 47197,
            "unit": "ns/op\t   26129 B/op\t      81 allocs/op",
            "extra": "25742 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 47197,
            "unit": "ns/op",
            "extra": "25742 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26129,
            "unit": "B/op",
            "extra": "25742 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "25742 times\n4 procs"
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
          "id": "87d8ba97f048678b3ea06ef9c6f114798d0e33ae",
          "message": "chore: prep evm rc.5 (#3145)",
          "timestamp": "2026-03-09T16:56:02+01:00",
          "tree_id": "af276d6ba1967d3326dc6cbadf29f10cc176d260",
          "url": "https://github.com/evstack/ev-node/commit/87d8ba97f048678b3ea06ef9c6f114798d0e33ae"
        },
        "date": 1773072581213,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 47720,
            "unit": "ns/op\t   26131 B/op\t      81 allocs/op",
            "extra": "25704 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 47720,
            "unit": "ns/op",
            "extra": "25704 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26131,
            "unit": "B/op",
            "extra": "25704 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "25704 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37760,
            "unit": "ns/op\t    7004 B/op\t      71 allocs/op",
            "extra": "32264 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37760,
            "unit": "ns/op",
            "extra": "32264 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7004,
            "unit": "B/op",
            "extra": "32264 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "32264 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 38240,
            "unit": "ns/op\t    7455 B/op\t      81 allocs/op",
            "extra": "31821 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 38240,
            "unit": "ns/op",
            "extra": "31821 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7455,
            "unit": "B/op",
            "extra": "31821 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "31821 times\n4 procs"
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
          "id": "34db9464a59b6373ba0a8214c86e3d1bac3f7bce",
          "message": "build(deps): Bump actions/setup-go from 6.2.0 to 6.3.0 (#3150)\n\nBumps [actions/setup-go](https://github.com/actions/setup-go) from 6.2.0 to 6.3.0.\n- [Release notes](https://github.com/actions/setup-go/releases)\n- [Commits](https://github.com/actions/setup-go/compare/v6.2.0...v6.3.0)\n\n---\nupdated-dependencies:\n- dependency-name: actions/setup-go\n  dependency-version: 6.3.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-03-10T08:58:57+01:00",
          "tree_id": "63e1ae8fbe7377205ed17dd4d1b616ee75075841",
          "url": "https://github.com/evstack/ev-node/commit/34db9464a59b6373ba0a8214c86e3d1bac3f7bce"
        },
        "date": 1773130383990,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 39001,
            "unit": "ns/op\t    7025 B/op\t      71 allocs/op",
            "extra": "31390 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 39001,
            "unit": "ns/op",
            "extra": "31390 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7025,
            "unit": "B/op",
            "extra": "31390 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "31390 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 39523,
            "unit": "ns/op\t    7506 B/op\t      81 allocs/op",
            "extra": "29882 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 39523,
            "unit": "ns/op",
            "extra": "29882 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7506,
            "unit": "B/op",
            "extra": "29882 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "29882 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 52608,
            "unit": "ns/op\t   26136 B/op\t      81 allocs/op",
            "extra": "23869 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 52608,
            "unit": "ns/op",
            "extra": "23869 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26136,
            "unit": "B/op",
            "extra": "23869 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "23869 times\n4 procs"
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
          "id": "5fd82361cdf409a551e5837737911d15275989e7",
          "message": "build(deps): Bump docker/build-push-action from 6 to 7 (#3151)\n\n* build(deps): Bump docker/build-push-action from 6 to 7\n\nBumps [docker/build-push-action](https://github.com/docker/build-push-action) from 6 to 7.\n- [Release notes](https://github.com/docker/build-push-action/releases)\n- [Commits](https://github.com/docker/build-push-action/compare/v6...v7)\n\n---\nupdated-dependencies:\n- dependency-name: docker/build-push-action\n  dependency-version: '7'\n  dependency-type: direct:production\n  update-type: version-update:semver-major\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\n\n* fix flaky test\n\n---------\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>\nCo-authored-by: Julien Robert <julien@rbrt.fr>",
          "timestamp": "2026-03-10T09:01:53+01:00",
          "tree_id": "a8efc6d6486e479c32acaa8718dc824469e1b29f",
          "url": "https://github.com/evstack/ev-node/commit/5fd82361cdf409a551e5837737911d15275989e7"
        },
        "date": 1773130642352,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 38592,
            "unit": "ns/op\t    7022 B/op\t      71 allocs/op",
            "extra": "31533 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 38592,
            "unit": "ns/op",
            "extra": "31533 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 7022,
            "unit": "B/op",
            "extra": "31533 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "31533 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 39000,
            "unit": "ns/op\t    7471 B/op\t      81 allocs/op",
            "extra": "31172 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 39000,
            "unit": "ns/op",
            "extra": "31172 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7471,
            "unit": "B/op",
            "extra": "31172 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "31172 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 50193,
            "unit": "ns/op\t   26172 B/op\t      81 allocs/op",
            "extra": "24465 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 50193,
            "unit": "ns/op",
            "extra": "24465 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26172,
            "unit": "B/op",
            "extra": "24465 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "24465 times\n4 procs"
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
          "id": "f74b456d8be50afc28432b68199359aeabfee201",
          "message": "build(deps): Bump docker/login-action from 3 to 4 (#3149)\n\nBumps [docker/login-action](https://github.com/docker/login-action) from 3 to 4.\n- [Release notes](https://github.com/docker/login-action/releases)\n- [Commits](https://github.com/docker/login-action/compare/v3...v4)\n\n---\nupdated-dependencies:\n- dependency-name: docker/login-action\n  dependency-version: '4'\n  dependency-type: direct:production\n  update-type: version-update:semver-major\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-03-10T09:02:36+01:00",
          "tree_id": "496fc8fc6c34f1976682d103174068f6746ee509",
          "url": "https://github.com/evstack/ev-node/commit/f74b456d8be50afc28432b68199359aeabfee201"
        },
        "date": 1773130959548,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 36758,
            "unit": "ns/op\t    6986 B/op\t      71 allocs/op",
            "extra": "33000 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 36758,
            "unit": "ns/op",
            "extra": "33000 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 6986,
            "unit": "B/op",
            "extra": "33000 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "33000 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 37694,
            "unit": "ns/op\t    7446 B/op\t      81 allocs/op",
            "extra": "32164 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 37694,
            "unit": "ns/op",
            "extra": "32164 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7446,
            "unit": "B/op",
            "extra": "32164 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "32164 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 48990,
            "unit": "ns/op\t   26170 B/op\t      81 allocs/op",
            "extra": "24626 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 48990,
            "unit": "ns/op",
            "extra": "24626 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26170,
            "unit": "B/op",
            "extra": "24626 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "24626 times\n4 procs"
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
          "id": "c588547ec5200e454eb8b0fb59091cd92aa969c5",
          "message": "build(deps): Bump the all-go group across 5 directories with 8 updates (#3147)\n\n* build(deps): Bump the all-go group across 5 directories with 8 updates\n\nBumps the all-go group with 3 updates in the / directory: [go.opentelemetry.io/otel](https://github.com/open-telemetry/opentelemetry-go), [go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp](https://github.com/open-telemetry/opentelemetry-go) and [golang.org/x/sync](https://github.com/golang/sync).\nBumps the all-go group with 1 update in the /apps/evm directory: [github.com/ethereum/go-ethereum](https://github.com/ethereum/go-ethereum).\nBumps the all-go group with 2 updates in the /execution/evm directory: [go.opentelemetry.io/otel](https://github.com/open-telemetry/opentelemetry-go) and [github.com/ethereum/go-ethereum](https://github.com/ethereum/go-ethereum).\nBumps the all-go group with 2 updates in the /test/docker-e2e directory: [github.com/ethereum/go-ethereum](https://github.com/ethereum/go-ethereum) and [github.com/evstack/ev-node/execution/evm](https://github.com/evstack/ev-node).\nBumps the all-go group with 2 updates in the /test/e2e directory: [github.com/ethereum/go-ethereum](https://github.com/ethereum/go-ethereum) and [go.opentelemetry.io/proto/otlp](https://github.com/open-telemetry/opentelemetry-proto-go).\n\n\nUpdates `go.opentelemetry.io/otel` from 1.41.0 to 1.42.0\n- [Release notes](https://github.com/open-telemetry/opentelemetry-go/releases)\n- [Changelog](https://github.com/open-telemetry/opentelemetry-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/open-telemetry/opentelemetry-go/compare/v1.41.0...v1.42.0)\n\nUpdates `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp` from 1.41.0 to 1.42.0\n- [Release notes](https://github.com/open-telemetry/opentelemetry-go/releases)\n- [Changelog](https://github.com/open-telemetry/opentelemetry-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/open-telemetry/opentelemetry-go/compare/v1.41.0...v1.42.0)\n\nUpdates `go.opentelemetry.io/otel/sdk` from 1.41.0 to 1.42.0\n- [Release notes](https://github.com/open-telemetry/opentelemetry-go/releases)\n- [Changelog](https://github.com/open-telemetry/opentelemetry-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/open-telemetry/opentelemetry-go/compare/v1.41.0...v1.42.0)\n\nUpdates `go.opentelemetry.io/otel/trace` from 1.41.0 to 1.42.0\n- [Release notes](https://github.com/open-telemetry/opentelemetry-go/releases)\n- [Changelog](https://github.com/open-telemetry/opentelemetry-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/open-telemetry/opentelemetry-go/compare/v1.41.0...v1.42.0)\n\nUpdates `golang.org/x/sync` from 0.19.0 to 0.20.0\n- [Commits](https://github.com/golang/sync/compare/v0.19.0...v0.20.0)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.0 to 1.17.1\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.0...v1.17.1)\n\nUpdates `go.opentelemetry.io/otel` from 1.41.0 to 1.42.0\n- [Release notes](https://github.com/open-telemetry/opentelemetry-go/releases)\n- [Changelog](https://github.com/open-telemetry/opentelemetry-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/open-telemetry/opentelemetry-go/compare/v1.41.0...v1.42.0)\n\nUpdates `go.opentelemetry.io/otel/sdk` from 1.41.0 to 1.42.0\n- [Release notes](https://github.com/open-telemetry/opentelemetry-go/releases)\n- [Changelog](https://github.com/open-telemetry/opentelemetry-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/open-telemetry/opentelemetry-go/compare/v1.41.0...v1.42.0)\n\nUpdates `go.opentelemetry.io/otel/trace` from 1.41.0 to 1.42.0\n- [Release notes](https://github.com/open-telemetry/opentelemetry-go/releases)\n- [Changelog](https://github.com/open-telemetry/opentelemetry-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/open-telemetry/opentelemetry-go/compare/v1.41.0...v1.42.0)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.0 to 1.17.1\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.0...v1.17.1)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.0 to 1.17.1\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.0...v1.17.1)\n\nUpdates `github.com/evstack/ev-node/execution/evm` from 1.0.0-rc.3 to 1.0.0-rc.4\n- [Release notes](https://github.com/evstack/ev-node/releases)\n- [Changelog](https://github.com/evstack/ev-node/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/evstack/ev-node/compare/v1.0.0-rc.3...v1.0.0-rc.4)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.0 to 1.17.1\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.0...v1.17.1)\n\nUpdates `go.opentelemetry.io/proto/otlp` from 1.9.0 to 1.10.0\n- [Release notes](https://github.com/open-telemetry/opentelemetry-proto-go/releases)\n- [Commits](https://github.com/open-telemetry/opentelemetry-proto-go/compare/v1.9.0...v1.10.0)\n\n---\nupdated-dependencies:\n- dependency-name: go.opentelemetry.io/otel\n  dependency-version: 1.42.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp\n  dependency-version: 1.42.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: go.opentelemetry.io/otel/sdk\n  dependency-version: 1.42.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: go.opentelemetry.io/otel/trace\n  dependency-version: 1.42.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: golang.org/x/sync\n  dependency-version: 0.20.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.1\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: go.opentelemetry.io/otel\n  dependency-version: 1.42.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: go.opentelemetry.io/otel/sdk\n  dependency-version: 1.42.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: go.opentelemetry.io/otel/trace\n  dependency-version: 1.42.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.1\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.1\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/evstack/ev-node/execution/evm\n  dependency-version: 1.0.0-rc.4\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.1\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: go.opentelemetry.io/proto/otlp\n  dependency-version: 1.10.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\n\n* chore: run just deps after Dependabot update\n\n---------\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>\nCo-authored-by: github-actions[bot] <41898282+github-actions[bot]@users.noreply.github.com>\nCo-authored-by: julienrbrt <julien@rbrt.fr>",
          "timestamp": "2026-03-10T14:38:12+01:00",
          "tree_id": "d9ededb810577958f7f4b09f1b8569501957b2de",
          "url": "https://github.com/evstack/ev-node/commit/c588547ec5200e454eb8b0fb59091cd92aa969c5"
        },
        "date": 1773150148187,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 36491,
            "unit": "ns/op\t    6985 B/op\t      71 allocs/op",
            "extra": "33038 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 36491,
            "unit": "ns/op",
            "extra": "33038 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 6985,
            "unit": "B/op",
            "extra": "33038 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "33038 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 37365,
            "unit": "ns/op\t    7444 B/op\t      81 allocs/op",
            "extra": "32248 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 37365,
            "unit": "ns/op",
            "extra": "32248 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 7444,
            "unit": "B/op",
            "extra": "32248 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "32248 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 46862,
            "unit": "ns/op\t   26125 B/op\t      81 allocs/op",
            "extra": "25852 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 46862,
            "unit": "ns/op",
            "extra": "25852 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 26125,
            "unit": "B/op",
            "extra": "25852 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 81,
            "unit": "allocs/op",
            "extra": "25852 times\n4 procs"
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
          "id": "a02e90bf99ee919746a42051d053c23cd92e8d06",
          "message": "build(deps): Bump the all-go group across 4 directories with 6 updates (#3116)\n\n* build(deps): Bump the all-go group across 4 directories with 6 updates\n\nBumps the all-go group with 3 updates in the / directory: [go.opentelemetry.io/otel](https://github.com/open-telemetry/opentelemetry-go), [go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp](https://github.com/open-telemetry/opentelemetry-go) and [golang.org/x/net](https://github.com/golang/net).\nBumps the all-go group with 1 update in the /execution/evm directory: [go.opentelemetry.io/otel](https://github.com/open-telemetry/opentelemetry-go).\nBumps the all-go group with 1 update in the /execution/grpc directory: [golang.org/x/net](https://github.com/golang/net).\nBumps the all-go group with 1 update in the /test/docker-e2e directory: [github.com/celestiaorg/tastora](https://github.com/celestiaorg/tastora).\n\n\nUpdates `go.opentelemetry.io/otel` from 1.40.0 to 1.41.0\n- [Release notes](https://github.com/open-telemetry/opentelemetry-go/releases)\n- [Changelog](https://github.com/open-telemetry/opentelemetry-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/open-telemetry/opentelemetry-go/compare/v1.40.0...v1.41.0)\n\nUpdates `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp` from 1.40.0 to 1.41.0\n- [Release notes](https://github.com/open-telemetry/opentelemetry-go/releases)\n- [Changelog](https://github.com/open-telemetry/opentelemetry-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/open-telemetry/opentelemetry-go/compare/v1.40.0...v1.41.0)\n\nUpdates `go.opentelemetry.io/otel/sdk` from 1.40.0 to 1.41.0\n- [Release notes](https://github.com/open-telemetry/opentelemetry-go/releases)\n- [Changelog](https://github.com/open-telemetry/opentelemetry-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/open-telemetry/opentelemetry-go/compare/v1.40.0...v1.41.0)\n\nUpdates `go.opentelemetry.io/otel/trace` from 1.40.0 to 1.41.0\n- [Release notes](https://github.com/open-telemetry/opentelemetry-go/releases)\n- [Changelog](https://github.com/open-telemetry/opentelemetry-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/open-telemetry/opentelemetry-go/compare/v1.40.0...v1.41.0)\n\nUpdates `golang.org/x/net` from 0.50.0 to 0.51.0\n- [Commits](https://github.com/golang/net/compare/v0.50.0...v0.51.0)\n\nUpdates `go.opentelemetry.io/otel` from 1.40.0 to 1.41.0\n- [Release notes](https://github.com/open-telemetry/opentelemetry-go/releases)\n- [Changelog](https://github.com/open-telemetry/opentelemetry-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/open-telemetry/opentelemetry-go/compare/v1.40.0...v1.41.0)\n\nUpdates `go.opentelemetry.io/otel/sdk` from 1.40.0 to 1.41.0\n- [Release notes](https://github.com/open-telemetry/opentelemetry-go/releases)\n- [Changelog](https://github.com/open-telemetry/opentelemetry-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/open-telemetry/opentelemetry-go/compare/v1.40.0...v1.41.0)\n\nUpdates `go.opentelemetry.io/otel/trace` from 1.40.0 to 1.41.0\n- [Release notes](https://github.com/open-telemetry/opentelemetry-go/releases)\n- [Changelog](https://github.com/open-telemetry/opentelemetry-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/open-telemetry/opentelemetry-go/compare/v1.40.0...v1.41.0)\n\nUpdates `golang.org/x/net` from 0.50.0 to 0.51.0\n- [Commits](https://github.com/golang/net/compare/v0.50.0...v0.51.0)\n\nUpdates `github.com/celestiaorg/tastora` from 0.15.0 to 0.16.0\n- [Release notes](https://github.com/celestiaorg/tastora/releases)\n- [Commits](https://github.com/celestiaorg/tastora/compare/v0.15.0...v0.16.0)\n\n---\nupdated-dependencies:\n- dependency-name: go.opentelemetry.io/otel\n  dependency-version: 1.41.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp\n  dependency-version: 1.41.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: go.opentelemetry.io/otel/sdk\n  dependency-version: 1.41.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: go.opentelemetry.io/otel/trace\n  dependency-version: 1.41.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: golang.org/x/net\n  dependency-version: 0.51.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: go.opentelemetry.io/otel\n  dependency-version: 1.41.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: go.opentelemetry.io/otel/sdk\n  dependency-version: 1.41.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: go.opentelemetry.io/otel/trace\n  dependency-version: 1.41.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: golang.org/x/net\n  dependency-version: 0.51.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/celestiaorg/tastora\n  dependency-version: 0.16.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\n\n* chore: run just deps after Dependabot update\n\n---------\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>\nCo-authored-by: github-actions[bot] <41898282+github-actions[bot]@users.noreply.github.com>\nCo-authored-by: julienrbrt <julien@rbrt.fr>",
          "timestamp": "2026-03-03T09:40:06+01:00",
          "tree_id": "859cd3366eb40e2ed6583d6e95efe07cdedf87de",
          "url": "https://github.com/evstack/ev-node/commit/a02e90bf99ee919746a42051d053c23cd92e8d06"
        },
        "date": 1772527790417,
        "tool": "customSmallerIsBetter",
        "benches": [
          {
            "name": "SpamoorSmoke - Batch.Commit (avg)",
            "value": 73.81515151515151,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.Put (avg)",
            "value": 3.578021978021978,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SaveBlockData (avg)",
            "value": 18.86060606060606,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SetHeight (avg)",
            "value": 15.033333333333333,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.UpdateState (avg)",
            "value": 7.581818181818182,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ApplyBlock (avg)",
            "value": 7754.087878787879,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.CreateBlock (avg)",
            "value": 4.115151515151515,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ProduceBlock (avg)",
            "value": 8789.58787878788,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.RetrieveBatch (avg)",
            "value": 42.093939393939394,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Creating db provider (avg)",
            "value": 6.471137521222411,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DA.Submit (avg)",
            "value": 786.3714285714286,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitData (avg)",
            "value": 1142.2,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitHeaders (avg)",
            "value": 1184.0206896551724,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DatabaseProvider::commit (avg)",
            "value": 3799.844522968198,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.ForkchoiceUpdated (avg)",
            "value": 836.4814814814815,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.GetPayload (avg)",
            "value": 1644.6878787878788,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.NewPayload (avg)",
            "value": 2649.742424242424,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Eth.GetBlockByNumber (avg)",
            "value": 950.8858858858858,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.ExecuteTxs (avg)",
            "value": 7734.478787878787,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.GetTxs (avg)",
            "value": 1484.2162162162163,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.SetFinal (avg)",
            "value": 2141.8230088495575,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - ForcedInclusionRetriever.RetrieveForcedIncludedTxs (avg)",
            "value": 6.381818181818182,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Pruner::run_with_provider (avg)",
            "value": 86.45614035087719,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.GetNextBatch (avg)",
            "value": 32.2030303030303,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.SubmitBatchTxs (avg)",
            "value": 56,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - StaticFileProviderRW::finalize (avg)",
            "value": 1058.6173708920187,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - StaticFileWriters::finalize (avg)",
            "value": 3224.711267605634,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Storage trie (avg)",
            "value": 2.173298572996707,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.DeleteMetadata (avg)",
            "value": 21.266304347826086,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetBlockData (avg)",
            "value": 10.55694098088113,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetHeader (avg)",
            "value": 10.411483253588516,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetMetadata (avg)",
            "value": 8.927911275415896,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.Height (avg)",
            "value": 11.620347394540943,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.NewBatch (avg)",
            "value": 13.513636363636364,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.SetMetadata (avg)",
            "value": 27.826508620689655,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Tx::commit (avg)",
            "value": 564.5070422535211,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - TxPool.GetTxs (avg)",
            "value": 1457.4594594594594,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - accounts (avg)",
            "value": 1.924731182795699,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - begin (avg)",
            "value": 20.461267605633804,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - build_payload (avg)",
            "value": 1736.6982456140352,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - cache_for (avg)",
            "value": 50.55438596491228,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - calculate_overlay (avg)",
            "value": 6,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - commit (avg)",
            "value": 463.3719298245614,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - compute_state_root_parallel (avg)",
            "value": 626.612676056338,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - compute_trie_input_task (avg)",
            "value": 406.05281690140845,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - connection (avg)",
            "value": 11.333333333333334,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - contracts (avg)",
            "value": 2.4119718309859155,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - convert_to_block (avg)",
            "value": 16.814035087719297,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - database_provider_ro (avg)",
            "value": 29.22005097706032,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - evm env (avg)",
            "value": 3.23859649122807,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - evm_for_ctx (avg)",
            "value": 34.604838709677416,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execute tx (avg)",
            "value": 733.340625,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execute_block (avg)",
            "value": 976.6842105263158,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execute_tx (avg)",
            "value": 709.746875,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execution (avg)",
            "value": 831.8035087719298,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - finish (avg)",
            "value": 9.656140350877193,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - get_cache_for (avg)",
            "value": 28.736842105263158,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - get_overlay (avg)",
            "value": 12.830926083262533,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - hashed_post_state (avg)",
            "value": 11.866197183098592,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_block_mdbx_only (avg)",
            "value": 87.08450704225352,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_block_or_payload (avg)",
            "value": 2116.8838028169016,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_state (avg)",
            "value": 18.489436619718308,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - merge transitions (avg)",
            "value": 2.6929460580912865,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_forkchoice_updated (avg)",
            "value": 63.551764705882356,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_new_payload (avg)",
            "value": 2158.218309859155,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_save_blocks (avg)",
            "value": 5470.102112676056,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - poll_next_event (avg)",
            "value": 8.08227465214761,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - pre execution (avg)",
            "value": 22.796491228070174,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prewarm and caching (avg)",
            "value": 1819.088028169014,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prewarm tx (avg)",
            "value": 874.690625,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prewarm worker (avg)",
            "value": 2965.3548387096776,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prune_segments (avg)",
            "value": 1.9298245614035088,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - save_blocks (avg)",
            "value": 1406.5140845070423,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - save_cache (avg)",
            "value": 787.2429577464789,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_all (avg)",
            "value": 5084.903225806452,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_cache_exclusive (avg)",
            "value": 91.56338028169014,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_payload_processor (avg)",
            "value": 96.20070422535211,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - state provider (avg)",
            "value": 41.835087719298244,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - transact_batch (avg)",
            "value": 2667.435483870968,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - trie_data (avg)",
            "value": 5.348591549295775,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_build (avg)",
            "value": 1794.1649122807019,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_connect_buffered_blocks (avg)",
            "value": 5.73943661971831,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_insert_payload (avg)",
            "value": 2136.8661971830984,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - update_history_indices (avg)",
            "value": 28.697183098591548,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - update_pipeline_stages (avg)",
            "value": 30.654929577464788,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_inner (avg)",
            "value": 7.368421052631579,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_post_execution (avg)",
            "value": 1.9929824561403509,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_post_execution_with_hashed_state (avg)",
            "value": 1.6446886446886446,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_with_state (avg)",
            "value": 1964.1549295774648,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_header_against_parent (avg)",
            "value": 1.6385964912280702,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_post_execution (avg)",
            "value": 38.06315789473684,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_transaction (avg)",
            "value": 39.13350785340314,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - wait_cloned (avg)",
            "value": 110.79204339963833,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_blocks_data (avg)",
            "value": 1186.4526315789474,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_hashed_state (avg)",
            "value": 94.38709677419355,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_headers (avg)",
            "value": 29.5859649122807,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_receipts (avg)",
            "value": 38.92280701754386,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_state (avg)",
            "value": 35.36395759717315,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_transactions (avg)",
            "value": 11.115789473684211,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_trie_updates_sorted (avg)",
            "value": 38.017605633802816,
            "unit": "us"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "eroderust@outlook.com",
            "name": "eroderust",
            "username": "eroderust"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "e877782df0afe0addc8243aa740e6dfb8e1becdb",
          "message": "refactor: use the built-in max/min to simplify the code (#3121)\n\nSigned-off-by: eroderust <eroderust@outlook.com>",
          "timestamp": "2026-03-03T09:41:20+01:00",
          "tree_id": "483c77832c31f15735c302743bbe65ea8ab03a5f",
          "url": "https://github.com/evstack/ev-node/commit/e877782df0afe0addc8243aa740e6dfb8e1becdb"
        },
        "date": 1772527900249,
        "tool": "customSmallerIsBetter",
        "benches": [
          {
            "name": "SpamoorSmoke - Batch.Commit (avg)",
            "value": 52.5,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.Put (avg)",
            "value": 2.492134831460674,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SaveBlockData (avg)",
            "value": 17.454819277108435,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SetHeight (avg)",
            "value": 11.177710843373495,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.UpdateState (avg)",
            "value": 6.219879518072289,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ApplyBlock (avg)",
            "value": 7513.846385542169,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.CreateBlock (avg)",
            "value": 2.5167173252279635,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ProduceBlock (avg)",
            "value": 8378.484939759037,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.RetrieveBatch (avg)",
            "value": 27.768072289156628,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Creating db provider (avg)",
            "value": 5.873728813559322,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DA.Submit (avg)",
            "value": 678.9887005649717,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitData (avg)",
            "value": 1011.6774193548387,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitHeaders (avg)",
            "value": 990.5931034482759,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DatabaseProvider::commit (avg)",
            "value": 4356.68661971831,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.ForkchoiceUpdated (avg)",
            "value": 804.0190763052209,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.GetPayload (avg)",
            "value": 1659.436746987952,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.NewPayload (avg)",
            "value": 2473.5873493975905,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Eth.GetBlockByNumber (avg)",
            "value": 832.960843373494,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.ExecuteTxs (avg)",
            "value": 7501.469879518072,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.GetTxs (avg)",
            "value": 1766.111111111111,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.SetFinal (avg)",
            "value": 1731.0903614457832,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - ForcedInclusionRetriever.RetrieveForcedIncludedTxs (avg)",
            "value": 3.9216867469879517,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Pruner::run_with_provider (avg)",
            "value": 100.625,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.GetNextBatch (avg)",
            "value": 20.912650602409638,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.SubmitBatchTxs (avg)",
            "value": 49.38709677419355,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - StaticFileProviderRW::finalize (avg)",
            "value": 1230.4342723004695,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - StaticFileWriters::finalize (avg)",
            "value": 3743.4859154929577,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Storage trie (avg)",
            "value": 2.0353831201239854,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.DeleteMetadata (avg)",
            "value": 20.743093922651934,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetBlockData (avg)",
            "value": 9.851104707012489,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetHeader (avg)",
            "value": 9.723095525997582,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetMetadata (avg)",
            "value": 7.238619676945668,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.Height (avg)",
            "value": 8.726934984520124,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.NewBatch (avg)",
            "value": 11.066265060240964,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.SetMetadata (avg)",
            "value": 23.04671374253123,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Tx::commit (avg)",
            "value": 602.1549295774648,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - TxPool.GetTxs (avg)",
            "value": 1746.75,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - accounts (avg)",
            "value": 2.0072463768115942,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - begin (avg)",
            "value": 16.788732394366196,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - build_payload (avg)",
            "value": 1638.688811188811,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - cache_for (avg)",
            "value": 44.524647887323944,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - calculate_overlay (avg)",
            "value": 5.089788732394366,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - commit (avg)",
            "value": 513.5246478873239,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - compute_state_root_parallel (avg)",
            "value": 615.7394366197183,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - compute_trie_input_task (avg)",
            "value": 350.7957746478873,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - connection (avg)",
            "value": 24.166666666666668,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - contracts (avg)",
            "value": 2.7816901408450705,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - convert_to_block (avg)",
            "value": 13.669014084507042,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - database_provider_ro (avg)",
            "value": 25.877966101694916,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - evm env (avg)",
            "value": 2.482394366197183,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - evm_for_ctx (avg)",
            "value": 25.637096774193548,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execute tx (avg)",
            "value": 732.4906832298136,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execute_block (avg)",
            "value": 967.1795774647887,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execute_tx (avg)",
            "value": 683.4316770186335,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execution (avg)",
            "value": 851.1352313167259,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - finish (avg)",
            "value": 8.077464788732394,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - get_cache_for (avg)",
            "value": 25.496478873239436,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - get_overlay (avg)",
            "value": 11.289830508474576,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - hashed_post_state (avg)",
            "value": 10.088652482269504,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_block_mdbx_only (avg)",
            "value": 92.13380281690141,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_block_or_payload (avg)",
            "value": 2022.7394366197184,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_state (avg)",
            "value": 17.683098591549296,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - merge transitions (avg)",
            "value": 2.310924369747899,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_forkchoice_updated (avg)",
            "value": 51.90727699530517,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_new_payload (avg)",
            "value": 2059.218309859155,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_save_blocks (avg)",
            "value": 6128.429577464789,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - poll_next_event (avg)",
            "value": 7.610026172740085,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - pre execution (avg)",
            "value": 18.450704225352112,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prewarm and caching (avg)",
            "value": 1763.0528169014085,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prewarm tx (avg)",
            "value": 874.7111801242236,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prewarm worker (avg)",
            "value": 2875.4596774193546,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prune_segments (avg)",
            "value": 1.9464285714285714,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - save_blocks (avg)",
            "value": 1541.5211267605634,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - save_cache (avg)",
            "value": 747.1514084507043,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_all (avg)",
            "value": 5186.322580645161,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_cache_exclusive (avg)",
            "value": 83.97887323943662,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_payload_processor (avg)",
            "value": 89.72183098591549,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - state provider (avg)",
            "value": 34.90492957746479,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - transact_batch (avg)",
            "value": 2656.5967741935483,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - trie_data (avg)",
            "value": 5.322183098591549,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_build (avg)",
            "value": 1690.0104895104896,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_connect_buffered_blocks (avg)",
            "value": 6.714788732394366,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_insert_payload (avg)",
            "value": 2039.7535211267605,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - update_history_indices (avg)",
            "value": 21.119718309859156,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - update_pipeline_stages (avg)",
            "value": 29.052816901408452,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_inner (avg)",
            "value": 6.408450704225352,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_post_execution (avg)",
            "value": 1.630281690140845,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_post_execution_with_hashed_state (avg)",
            "value": 1.0823970037453183,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_with_state (avg)",
            "value": 1891.5704225352113,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_header_against_parent (avg)",
            "value": 1.1079136690647482,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_post_execution (avg)",
            "value": 32.63380281690141,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_transaction (avg)",
            "value": 25.44109589041096,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - wait_cloned (avg)",
            "value": 108.74898785425101,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_blocks_data (avg)",
            "value": 1339.1971830985915,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_hashed_state (avg)",
            "value": 93.64516129032258,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_headers (avg)",
            "value": 25.41549295774648,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_receipts (avg)",
            "value": 24.345070422535212,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_state (avg)",
            "value": 33.00704225352113,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_transactions (avg)",
            "value": 10.419014084507042,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_trie_updates_sorted (avg)",
            "value": 38.79929577464789,
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
          "id": "042b75a30eb6427a15c0a98989360b52361ec9cc",
          "message": "feat: ensure p2p DAHint within limits (#3128)\n\n* feat: validate P2P DA height hints in the syncer against the latest DA height.\n\n* Review feedback\n\n* Changelog",
          "timestamp": "2026-03-03T12:25:04+01:00",
          "tree_id": "d250855acfce39ef0287a5840ed9d9db6f8c6329",
          "url": "https://github.com/evstack/ev-node/commit/042b75a30eb6427a15c0a98989360b52361ec9cc"
        },
        "date": 1772537401473,
        "tool": "customSmallerIsBetter",
        "benches": [
          {
            "name": "SpamoorSmoke - Batch.Commit (avg)",
            "value": 59.21921921921922,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.Put (avg)",
            "value": 2.6130434782608694,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SaveBlockData (avg)",
            "value": 16.885885885885887,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SetHeight (avg)",
            "value": 13.564564564564565,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.UpdateState (avg)",
            "value": 7.36036036036036,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ApplyBlock (avg)",
            "value": 7392.792792792793,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.CreateBlock (avg)",
            "value": 3.24024024024024,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ProduceBlock (avg)",
            "value": 8343.466867469879,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.RetrieveBatch (avg)",
            "value": 34.85585585585586,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Creating db provider (avg)",
            "value": 5.310696095076401,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DA.Submit (avg)",
            "value": 707.7727272727273,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitData (avg)",
            "value": 1043.258064516129,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitHeaders (avg)",
            "value": 1040.606896551724,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DatabaseProvider::commit (avg)",
            "value": 4054.919014084507,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.ForkchoiceUpdated (avg)",
            "value": 693.8284854563691,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.GetPayload (avg)",
            "value": 1667.3753753753754,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.NewPayload (avg)",
            "value": 2610.774774774775,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Eth.GetBlockByNumber (avg)",
            "value": 776.1414242728184,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.ExecuteTxs (avg)",
            "value": 7375.666666666667,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.GetTxs (avg)",
            "value": 1272.2777777777778,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.SetFinal (avg)",
            "value": 1516.5317220543807,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - ForcedInclusionRetriever.RetrieveForcedIncludedTxs (avg)",
            "value": 5.1441441441441444,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Pruner::run_with_provider (avg)",
            "value": 90.40350877192982,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.GetNextBatch (avg)",
            "value": 26.62162162162162,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.SubmitBatchTxs (avg)",
            "value": 53.12903225806452,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - StaticFileProviderRW::finalize (avg)",
            "value": 1133.4941314553992,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - StaticFileWriters::finalize (avg)",
            "value": 3443.644366197183,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Storage trie (avg)",
            "value": 2.11914881872378,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.DeleteMetadata (avg)",
            "value": 26.321329639889196,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetBlockData (avg)",
            "value": 10.81232492997199,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetHeader (avg)",
            "value": 10.321001088139282,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetMetadata (avg)",
            "value": 7.7895122845617895,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.Height (avg)",
            "value": 9.986352357320099,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.NewBatch (avg)",
            "value": 12.004504504504505,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.SetMetadata (avg)",
            "value": 24.741689373297003,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Tx::commit (avg)",
            "value": 598.1413427561837,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - TxPool.GetTxs (avg)",
            "value": 1250.4444444444443,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - accounts (avg)",
            "value": 1.8021582733812949,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - begin (avg)",
            "value": 17.85211267605634,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - build_payload (avg)",
            "value": 1744.475352112676,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - cache_for (avg)",
            "value": 49.716312056737586,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - calculate_overlay (avg)",
            "value": 5.496466431095406,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - commit (avg)",
            "value": 495.6232394366197,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - compute_state_root_parallel (avg)",
            "value": 638.3710247349824,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - compute_trie_input_task (avg)",
            "value": 388.4381625441696,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - connection (avg)",
            "value": 53.5,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - contracts (avg)",
            "value": 2.2190812720848054,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - convert_to_block (avg)",
            "value": 15.985865724381625,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - database_provider_ro (avg)",
            "value": 29.527612574341546,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - evm env (avg)",
            "value": 2.88339222614841,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - evm_for_ctx (avg)",
            "value": 44.806451612903224,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execute tx (avg)",
            "value": 818.3416149068323,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execute_block (avg)",
            "value": 1073.6572438162543,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execute_tx (avg)",
            "value": 753.8726708074535,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execution (avg)",
            "value": 938.4063604240283,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - finish (avg)",
            "value": 9.042402826855124,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - get_cache_for (avg)",
            "value": 28.459363957597173,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - get_overlay (avg)",
            "value": 14.197792869269948,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - hashed_post_state (avg)",
            "value": 25.62190812720848,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_block_mdbx_only (avg)",
            "value": 79.97535211267606,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_block_or_payload (avg)",
            "value": 2206.6961130742047,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_state (avg)",
            "value": 16.469964664310954,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - merge transitions (avg)",
            "value": 2.7927927927927927,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_forkchoice_updated (avg)",
            "value": 60.04711425206125,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_new_payload (avg)",
            "value": 2243.08480565371,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_save_blocks (avg)",
            "value": 5712.588028169014,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - poll_next_event (avg)",
            "value": 7.71744817870799,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - pre execution (avg)",
            "value": 20.322695035460992,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prewarm and caching (avg)",
            "value": 1924.2473498233217,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prewarm tx (avg)",
            "value": 955.1832298136646,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prewarm worker (avg)",
            "value": 3080.5403225806454,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prune_segments (avg)",
            "value": 6.333333333333333,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - save_blocks (avg)",
            "value": 1402,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - save_cache (avg)",
            "value": 781.2730496453901,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_all (avg)",
            "value": 5900.193548387097,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_cache_exclusive (avg)",
            "value": 96.85159010600707,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_payload_processor (avg)",
            "value": 100.86925795053004,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - state provider (avg)",
            "value": 39.41696113074205,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - transact_batch (avg)",
            "value": 2798.0806451612902,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - trie_data (avg)",
            "value": 6.866197183098592,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_build (avg)",
            "value": 1795.5669014084508,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_connect_buffered_blocks (avg)",
            "value": 4.69964664310954,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_insert_payload (avg)",
            "value": 2222.5512367491165,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - update_history_indices (avg)",
            "value": 22.559859154929576,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - update_pipeline_stages (avg)",
            "value": 30.017605633802816,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_inner (avg)",
            "value": 7.565371024734982,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_post_execution (avg)",
            "value": 1.7773851590106007,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_post_execution_with_hashed_state (avg)",
            "value": 1.570921985815603,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_with_state (avg)",
            "value": 2067.2120141342757,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_header_against_parent (avg)",
            "value": 1.4381625441696113,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_post_execution (avg)",
            "value": 51.48056537102474,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_transaction (avg)",
            "value": 27.83739837398374,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - wait_cloned (avg)",
            "value": 107.60070671378092,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_blocks_data (avg)",
            "value": 1191.8309859154929,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_hashed_state (avg)",
            "value": 89.45161290322581,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_headers (avg)",
            "value": 30.35211267605634,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_receipts (avg)",
            "value": 31.18374558303887,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_state (avg)",
            "value": 27.841549295774648,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_transactions (avg)",
            "value": 12.112676056338028,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_trie_updates_sorted (avg)",
            "value": 39.00352112676056,
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
          "id": "f8fa22ed52b39b68b042b9c08ae0709cf27abdad",
          "message": "feat(benchmarking): adding ERC20 benchmarking test (#3114)\n\n* refactor: move spamoor benchmark into testify suite in test/e2e/benchmark\n\n- Create test/e2e/benchmark/ subpackage with SpamoorSuite (testify/suite)\n- Move spamoor smoke test into suite as TestSpamoorSmoke\n- Split helpers into focused files: traces.go, output.go, metrics.go\n- Introduce resultWriter for defer-based benchmark JSON output\n- Export shared symbols from evm_test_common.go for cross-package use\n- Restructure CI to fan-out benchmark jobs and fan-in publishing\n- Run benchmarks on PRs only when benchmark-related files change\n\n* fix: correct BENCH_JSON_OUTPUT path for spamoor benchmark\n\ngo test sets the working directory to the package under test, so the\nenv var should be relative to test/e2e/benchmark/, not test/e2e/.\n\n* fix: place package pattern before test binary flags in benchmark CI\n\ngo test treats all arguments after an unknown flag (--evm-binary) as\ntest binary args, so ./benchmark/ was never recognized as a package\npattern.\n\n* fix: adjust evm-binary path for benchmark subpackage working directory\n\ngo test sets the cwd to the package directory (test/e2e/benchmark/),\nso the binary path needs an extra parent traversal.\n\n* wip: erc20 benchmark test\n\n* fix: exclude benchmark subpackage from make test-e2e\n\nThe benchmark package doesn't define the --binary flag that test-e2e\npasses. It has its own CI workflow so it doesn't need to run here.\n\n* fix: replace FilterLogs with header iteration and optimize spamoor config\n\ncollectBlockMetrics hit reth's 20K FilterLogs limit at high tx volumes.\nReplace with direct header iteration over [startBlock, endBlock] and add\nPhase 1 metrics: non-empty ratio, block interval p50/p99, gas/block and\ntx/block p50/p99.\n\nOptimize spamoor configuration for 100ms block time:\n- --slot-duration 100ms, --startup-delay 0 on daemon\n- throughput=50 per 100ms slot (500 tx/s per spammer)\n- max_pending=50000 to avoid 3s block poll backpressure\n- 5 staggered spammers with 50K txs each\n\nResults: 55 MGas/s, 1414 TPS, 19.8% non-empty blocks (up from 6%).\n\n* fix: improve benchmark measurement window and reliability\n\n- Move startBlock capture after spammer creation to exclude warm-up\n- Replace 20s drain sleep with smart poll (waitForDrain)\n- Add deleteAllSpammers cleanup to handle stale spamoor DB entries\n- Lower trace sample rate to 10% to prevent Jaeger OOM\n\n* fix: address PR review feedback for benchmark suite\n\n- make reth tag configurable via EV_RETH_TAG env var (default pr-140)\n- fix OTLP config: remove duplicate env vars, use http/protobuf protocol\n- use require.Eventually for host readiness polling\n- rename requireHTTP to requireHostUp\n- use non-fatal logging in resultWriter.flush deferred context\n- fix stale doc comment (setupCommonEVMEnv -> SetupCommonEVMEnv)\n- rename loop variable to avoid shadowing testing.TB convention\n- add block/internal/executing/** to CI path trigger\n- remove unused require import from output.go\n\n* chore: specify http\n\n* chore: filter out benchmark tests from test-e2e\n\n* refactor: centralize reth config and lower ERC20 spammer count\n\nmove EV_RETH_TAG resolution and rpc connection limits into setupEnv\nso all benchmark tests share the same reth configuration. lower ERC20\nspammer count from 5 to 2 to reduce resource contention on local\nhardware while keeping the loop for easy scaling on dedicated infra.\n\n* chore: collect all traces at once\n\n* chore: self review\n\n* refactor: extract benchmark helpers to slim down ERC20 test body\n\n- add blockMetricsSummary with summarize(), log(), and entries() methods\n- add evNodeOverhead() for computing ProduceBlock vs ExecuteTxs overhead\n- add collectTraces() suite method to deduplicate trace collection pattern\n- add addEntries() convenience method on resultWriter\n- slim TestERC20Throughput from ~217 to ~119 lines\n- reuse collectTraces in TestSpamoorSmoke\n\n* docs: add detailed documentation to benchmark helper methods\n\n* ci: add ERC20 throughput benchmark job\n\n* chore: remove span assertions\n\n* fix: guard against drain timeout and zero-duration TPS division\n\n- waitForDrain returns an error on timeout instead of silently logging\n- guard AchievedTPS computation when steady-state duration is zero",
          "timestamp": "2026-03-04T10:52:03Z",
          "tree_id": "2bc9d57d990adf28d62ad86a162eb94c872fb433",
          "url": "https://github.com/evstack/ev-node/commit/f8fa22ed52b39b68b042b9c08ae0709cf27abdad"
        },
        "date": 1772622769788,
        "tool": "customSmallerIsBetter",
        "benches": [
          {
            "name": "SpamoorSmoke - Batch.Commit (avg)",
            "value": 59.37692307692308,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.Put (avg)",
            "value": 2.9397590361445785,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SaveBlockData (avg)",
            "value": 23.4,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SetHeight (avg)",
            "value": 15.784615384615385,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.UpdateState (avg)",
            "value": 5.584615384615384,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ApplyBlock (avg)",
            "value": 5063.507692307692,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.CreateBlock (avg)",
            "value": 2.646153846153846,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ProduceBlock (avg)",
            "value": 5962.0615384615385,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.RetrieveBatch (avg)",
            "value": 33.07692307692308,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Creating db provider (avg)",
            "value": 6.664688427299703,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DA.Submit (avg)",
            "value": 1005.4444444444445,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitData (avg)",
            "value": 5189,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitHeaders (avg)",
            "value": 1094.5294117647059,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DatabaseProvider::commit (avg)",
            "value": 3374.967261904762,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.ForkchoiceUpdated (avg)",
            "value": 661.0532544378698,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.GetPayload (avg)",
            "value": 609.3692307692307,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.NewPayload (avg)",
            "value": 1468.2307692307693,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Eth.GetBlockByNumber (avg)",
            "value": 730.1597633136095,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.ExecuteTxs (avg)",
            "value": 5050.523076923077,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.GetTxs (avg)",
            "value": 5084.875,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.SetFinal (avg)",
            "value": 1393.6153846153845,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - ForcedInclusionRetriever.RetrieveForcedIncludedTxs (avg)",
            "value": 5.671875,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Pruner::run_with_provider (avg)",
            "value": 82.97014925373135,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.GetNextBatch (avg)",
            "value": 24.323076923076922,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.SubmitBatchTxs (avg)",
            "value": 165,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - StaticFileProviderRW::finalize (avg)",
            "value": 952.3026706231454,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - StaticFileWriters::finalize (avg)",
            "value": 2902.685459940653,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.DeleteMetadata (avg)",
            "value": 19.39622641509434,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetBlockData (avg)",
            "value": 10.4688995215311,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetHeader (avg)",
            "value": 9.120218579234972,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetMetadata (avg)",
            "value": 8.690537084398978,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.Height (avg)",
            "value": 11.003759398496241,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.NewBatch (avg)",
            "value": 14.684615384615384,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.SetMetadata (avg)",
            "value": 27.16600790513834,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Tx::commit (avg)",
            "value": 459.43620178041544,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - TxPool.GetTxs (avg)",
            "value": 4996.625,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - accounts (avg)",
            "value": 1.4969879518072289,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - begin (avg)",
            "value": 18.201183431952664,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - build_payload (avg)",
            "value": 419.18911174785103,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - cache_for (avg)",
            "value": 49.21661721068249,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - calculate_overlay (avg)",
            "value": 5.318047337278107,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - commit (avg)",
            "value": 368.3958333333333,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - compute_state_root_parallel (avg)",
            "value": 216.67655786350147,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - compute_trie_input_task (avg)",
            "value": 304.0443786982249,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - contracts (avg)",
            "value": 2.408284023668639,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - convert_to_block (avg)",
            "value": 12.130563798219585,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - database_provider_ro (avg)",
            "value": 29.971851851851852,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - evm env (avg)",
            "value": 2.744047619047619,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execute_block (avg)",
            "value": 134.10979228486647,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execution (avg)",
            "value": 1.2997032640949555,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - finish (avg)",
            "value": 8.623145400593472,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - get_cache_for (avg)",
            "value": 29.676557863501483,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - get_overlay (avg)",
            "value": 14.285502958579881,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - hashed_post_state (avg)",
            "value": 1.056379821958457,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_block_mdbx_only (avg)",
            "value": 77.35798816568047,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_block_or_payload (avg)",
            "value": 788.7159763313609,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_state (avg)",
            "value": 17.66765578635015,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - merge transitions (avg)",
            "value": 1.0155642023346303,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_forkchoice_updated (avg)",
            "value": 56.753709198813056,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_new_payload (avg)",
            "value": 828.0029673590504,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_save_blocks (avg)",
            "value": 4897.421364985164,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - poll_next_event (avg)",
            "value": 6.830881109419922,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - pre execution (avg)",
            "value": 22.080118694362017,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prewarm and caching (avg)",
            "value": 500.1750741839763,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prune_segments (avg)",
            "value": 2.5074626865671643,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - save_blocks (avg)",
            "value": 1293.7507418397627,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - save_cache (avg)",
            "value": 319.566765578635,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_cache_exclusive (avg)",
            "value": 80.20710059171597,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_payload_processor (avg)",
            "value": 85.36607142857143,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - state provider (avg)",
            "value": 38.62017804154303,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - trie_data (avg)",
            "value": 6.96301775147929,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_build (avg)",
            "value": 473.56733524355303,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_connect_buffered_blocks (avg)",
            "value": 5.287833827893175,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_insert_payload (avg)",
            "value": 807.387573964497,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - update_history_indices (avg)",
            "value": 10.801775147928995,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - update_pipeline_stages (avg)",
            "value": 33.866863905325445,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_inner (avg)",
            "value": 3.8694362017804154,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_post_execution (avg)",
            "value": 1.8372781065088757,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_post_execution_with_hashed_state (avg)",
            "value": 1.6059701492537313,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_with_state (avg)",
            "value": 648.6795252225519,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_header_against_parent (avg)",
            "value": 1.180473372781065,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_post_execution (avg)",
            "value": 23.71513353115727,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - wait_cloned (avg)",
            "value": 81.33535660091047,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_blocks_data (avg)",
            "value": 1099.5103857566767,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_headers (avg)",
            "value": 30.985207100591715,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_receipts (avg)",
            "value": 6.233727810650888,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_state (avg)",
            "value": 18.55325443786982,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_transactions (avg)",
            "value": 5.632047477744807,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_trie_updates_sorted (avg)",
            "value": 18.875370919881306,
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
          "id": "2c75e9ecfaefd3d5e94cdb978607151cfdae5264",
          "message": "chore: add stricter linting (#3132)\n\n* add stricterlinting\n\n* revert some changes and enable prealloc and predeclared\n\n* ci\n\n* fix lint\n\n* fix comments\n\n* fix\n\n* remove spec\n\n* test stability",
          "timestamp": "2026-03-04T14:05:07+01:00",
          "tree_id": "b07cba63c6dbca067217d5a3aaf172c0310617b9",
          "url": "https://github.com/evstack/ev-node/commit/2c75e9ecfaefd3d5e94cdb978607151cfdae5264"
        },
        "date": 1772629787056,
        "tool": "customSmallerIsBetter",
        "benches": [
          {
            "name": "SpamoorSmoke - Batch.Commit (avg)",
            "value": 57.424778761061944,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.Put (avg)",
            "value": 2.1142857142857143,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SaveBlockData (avg)",
            "value": 14.321428571428571,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SetHeight (avg)",
            "value": 8.824561403508772,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.UpdateState (avg)",
            "value": 5.684210526315789,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ApplyBlock (avg)",
            "value": 70962.07142857143,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.CreateBlock (avg)",
            "value": 3.5892857142857144,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ProduceBlock (avg)",
            "value": 71864.85714285714,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.RetrieveBatch (avg)",
            "value": 43,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Creating db provider (avg)",
            "value": 5.812684365781711,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DA.GetLatestDAHeight (avg)",
            "value": 582,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DA.Submit (avg)",
            "value": 1387.375,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitData (avg)",
            "value": 7429.5,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitHeaders (avg)",
            "value": 919.3571428571429,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DatabaseProvider::commit (avg)",
            "value": 3342.377581120944,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.ForkchoiceUpdated (avg)",
            "value": 5078.619631901841,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.GetPayload (avg)",
            "value": 31057.785714285714,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.NewPayload (avg)",
            "value": 37060.42857142857,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Eth.GetBlockByNumber (avg)",
            "value": 940.7546012269938,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.ExecuteTxs (avg)",
            "value": 70947.26785714286,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.GetTxs (avg)",
            "value": 6638.222222222223,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.SetFinal (avg)",
            "value": 16351.43137254902,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - ForcedInclusionRetriever.RetrieveForcedIncludedTxs (avg)",
            "value": 4.7407407407407405,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Pruner::run_with_provider (avg)",
            "value": 86.92647058823529,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.GetNextBatch (avg)",
            "value": 35.017857142857146,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.SubmitBatchTxs (avg)",
            "value": 302,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - StaticFileProviderRW::finalize (avg)",
            "value": 945.5300492610837,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - StaticFileWriters::finalize (avg)",
            "value": 2879.622418879056,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.DeleteMetadata (avg)",
            "value": 25.7,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetBlockData (avg)",
            "value": 7.780701754385965,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetHeader (avg)",
            "value": 8.56020942408377,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetMetadata (avg)",
            "value": 7.337278106508876,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.Height (avg)",
            "value": 9.76923076923077,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.NewBatch (avg)",
            "value": 13.769911504424778,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.SetMetadata (avg)",
            "value": 22.8300395256917,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Tx::commit (avg)",
            "value": 450.6401179941003,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - TxPool.GetTxs (avg)",
            "value": 6546.888888888889,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - accounts (avg)",
            "value": 1.290909090909091,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - begin (avg)",
            "value": 16.35693215339233,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - build_payload (avg)",
            "value": 355.90358126721765,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - cache_for (avg)",
            "value": 45.69911504424779,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - calculate_overlay (avg)",
            "value": 5.030973451327434,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - commit (avg)",
            "value": 365.16272189349115,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - compute_state_root_parallel (avg)",
            "value": 208.18047337278105,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - compute_trie_input_task (avg)",
            "value": 273.88495575221236,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - connection (avg)",
            "value": 25,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - contracts (avg)",
            "value": 2.5132743362831858,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - convert_to_block (avg)",
            "value": 11.482248520710058,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - database_provider_ro (avg)",
            "value": 27.398818316100442,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - evm env (avg)",
            "value": 2.4631268436578173,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execute_block (avg)",
            "value": 118.46902654867256,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execution (avg)",
            "value": 1.2592592592592593,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - finish (avg)",
            "value": 7.743362831858407,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - get_cache_for (avg)",
            "value": 26.766272189349113,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - get_overlay (avg)",
            "value": 13.338257016248154,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - hashed_post_state (avg)",
            "value": 1.2189349112426036,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_block_mdbx_only (avg)",
            "value": 84.7905604719764,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_block_or_payload (avg)",
            "value": 718.47197640118,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_state (avg)",
            "value": 16.778761061946902,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - merge transitions (avg)",
            "value": 1.0656934306569343,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_forkchoice_updated (avg)",
            "value": 50.79543197616683,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_new_payload (avg)",
            "value": 752.4513274336283,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_save_blocks (avg)",
            "value": 4796.569321533923,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - poll_next_event (avg)",
            "value": 6.48909551986475,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - pre execution (avg)",
            "value": 18.48967551622419,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prewarm and caching (avg)",
            "value": 463.81710914454277,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prune_segments (avg)",
            "value": 2.0441176470588234,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - save_blocks (avg)",
            "value": 1248.8219584569733,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - save_cache (avg)",
            "value": 297.81710914454277,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_cache_exclusive (avg)",
            "value": 73.25073746312684,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_payload_processor (avg)",
            "value": 76.98525073746313,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - state provider (avg)",
            "value": 34.4070796460177,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - trie_data (avg)",
            "value": 5.635693215339233,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_build (avg)",
            "value": 404.94214876033055,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_connect_buffered_blocks (avg)",
            "value": 4.928994082840236,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_insert_payload (avg)",
            "value": 733.929203539823,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - update_history_indices (avg)",
            "value": 7.766272189349112,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - update_pipeline_stages (avg)",
            "value": 30.548672566371682,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_inner (avg)",
            "value": 3.2595870206489677,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_post_execution (avg)",
            "value": 1.6342182890855457,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_post_execution_with_hashed_state (avg)",
            "value": 1.4335443037974684,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_with_state (avg)",
            "value": 600.1331360946746,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_header_against_parent (avg)",
            "value": 1.146268656716418,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_post_execution (avg)",
            "value": 21.209439528023598,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - wait_cloned (avg)",
            "value": 73.33541341653667,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_blocks_data (avg)",
            "value": 1083.0619469026549,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_headers (avg)",
            "value": 26.43952802359882,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_receipts (avg)",
            "value": 5.15929203539823,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_state (avg)",
            "value": 12.887905604719764,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_transactions (avg)",
            "value": 7.455621301775148,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_trie_updates_sorted (avg)",
            "value": 18.469026548672566,
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
          "id": "c0bc14150bd7cbc2353136b202269881fbfd0142",
          "message": "fix(node): race on caught up (#3133)\n\nfeat: add `Syncer.PendingCount` and use it to ensure the sync pipeline is drained before marking the node as caught up in failover.",
          "timestamp": "2026-03-04T17:51:20Z",
          "tree_id": "e2de7975904545942b3a234d86d0162e50422142",
          "url": "https://github.com/evstack/ev-node/commit/c0bc14150bd7cbc2353136b202269881fbfd0142"
        },
        "date": 1772647968537,
        "tool": "customSmallerIsBetter",
        "benches": [
          {
            "name": "SpamoorSmoke - Batch.Commit (avg)",
            "value": 51.98809523809524,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.Put (avg)",
            "value": 2.4727272727272727,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SaveBlockData (avg)",
            "value": 19.023809523809526,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SetHeight (avg)",
            "value": 10.642857142857142,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.UpdateState (avg)",
            "value": 5.714285714285714,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ApplyBlock (avg)",
            "value": 49699.57142857143,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.CreateBlock (avg)",
            "value": 2.5238095238095237,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ProduceBlock (avg)",
            "value": 50511.57142857143,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.RetrieveBatch (avg)",
            "value": 26.738095238095237,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Creating db provider (avg)",
            "value": 5.517138599105812,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DA.Submit (avg)",
            "value": 1307.857142857143,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitData (avg)",
            "value": 11784.5,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitHeaders (avg)",
            "value": 920.2307692307693,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DatabaseProvider::commit (avg)",
            "value": 3216.1101190476193,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.ForkchoiceUpdated (avg)",
            "value": 689.015037593985,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.GetPayload (avg)",
            "value": 20279.428571428572,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.NewPayload (avg)",
            "value": 26598.904761904763,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Eth.GetBlockByNumber (avg)",
            "value": 825.6541353383459,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.ExecuteTxs (avg)",
            "value": 49683.80952380953,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.GetTxs (avg)",
            "value": 18856,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.InitChain (avg)",
            "value": 2853,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.SetFinal (avg)",
            "value": 1854.6666666666667,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - ForcedInclusionRetriever.RetrieveForcedIncludedTxs (avg)",
            "value": 4.925,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - IpfsDHT.Provide (avg)",
            "value": 44,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - KademliaDHT.IpfsDHT.GetClosestPeers (avg)",
            "value": 11,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - KademliaDHT.IpfsDHT.RunLookupWithFollowup (avg)",
            "value": 5,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - KademliaDHT.IpfsDHT.RunQuery (avg)",
            "value": 2,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - KademliaDHT.ProviderManager.AddProvider (avg)",
            "value": 10,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Pruner::run_with_provider (avg)",
            "value": 78.2089552238806,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.GetNextBatch (avg)",
            "value": 19.738095238095237,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - StaticFileProviderRW::finalize (avg)",
            "value": 912.1110009910802,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - StaticFileWriters::finalize (avg)",
            "value": 2778.7351190476193,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.DeleteMetadata (avg)",
            "value": 23.148936170212767,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetBlockData (avg)",
            "value": 7.207373271889401,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetHeader (avg)",
            "value": 10.840425531914894,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetMetadata (avg)",
            "value": 8.348039215686274,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetMetadataByPrefix (avg)",
            "value": 27,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.Height (avg)",
            "value": 11.16597510373444,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.NewBatch (avg)",
            "value": 8.857142857142858,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.SetMetadata (avg)",
            "value": 28.756457564575644,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Tx::commit (avg)",
            "value": 425.53731343283584,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - TxPool.GetTxs (avg)",
            "value": 18687.5,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - accounts (avg)",
            "value": 1.3689024390243902,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - begin (avg)",
            "value": 17.84179104477612,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - build_payload (avg)",
            "value": 345.9116883116883,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - cache_for (avg)",
            "value": 46.12835820895523,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - calculate_overlay (avg)",
            "value": 5.071641791044776,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - commit (avg)",
            "value": 344.6973293768546,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - compute_state_root_parallel (avg)",
            "value": 199.43880597014925,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - compute_trie_input_task (avg)",
            "value": 272.9940476190476,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - contracts (avg)",
            "value": 2.1552238805970148,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - convert_to_block (avg)",
            "value": 11.577844311377245,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - database_provider_ro (avg)",
            "value": 27.116244411326377,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - evm env (avg)",
            "value": 2.426865671641791,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execute_block (avg)",
            "value": 113.90746268656716,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execution (avg)",
            "value": 1.2307692307692308,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - finish (avg)",
            "value": 8.119047619047619,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - get_cache_for (avg)",
            "value": 27.095238095238095,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - get_overlay (avg)",
            "value": 13.98206278026906,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - hashed_post_state (avg)",
            "value": 1.0746268656716418,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_block_mdbx_only (avg)",
            "value": 78.53115727002967,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_block_or_payload (avg)",
            "value": 721.6517857142857,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_state (avg)",
            "value": 15.839285714285714,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - merge transitions (avg)",
            "value": 1.2526690391459074,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_forkchoice_updated (avg)",
            "value": 52.59108910891089,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_new_payload (avg)",
            "value": 754.1880597014925,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_save_blocks (avg)",
            "value": 4607.803571428572,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - poll_next_event (avg)",
            "value": 6.138227848101266,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - pre execution (avg)",
            "value": 17.52537313432836,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prewarm and caching (avg)",
            "value": 452.55223880597015,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prune_segments (avg)",
            "value": 2.373134328358209,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - save_blocks (avg)",
            "value": 1191.8961424332344,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - save_cache (avg)",
            "value": 293.86309523809524,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_cache_exclusive (avg)",
            "value": 73.52083333333333,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_payload_processor (avg)",
            "value": 77.2797619047619,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - state provider (avg)",
            "value": 33.668656716417914,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - trie_data (avg)",
            "value": 5.24962852897474,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_build (avg)",
            "value": 395.35064935064935,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_connect_buffered_blocks (avg)",
            "value": 5.029850746268656,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_insert_payload (avg)",
            "value": 736.4567164179105,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - update_history_indices (avg)",
            "value": 8.473214285714286,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - update_pipeline_stages (avg)",
            "value": 30.679525222551927,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_inner (avg)",
            "value": 2.9613095238095237,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_post_execution (avg)",
            "value": 1.3982035928143712,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_post_execution_with_hashed_state (avg)",
            "value": 1.206451612903226,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_with_state (avg)",
            "value": 588.6428571428571,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_header_against_parent (avg)",
            "value": 1.108761329305136,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_post_execution (avg)",
            "value": 19.803571428571427,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - wait_cloned (avg)",
            "value": 78.8951048951049,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_blocks_data (avg)",
            "value": 1019.1483679525222,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_headers (avg)",
            "value": 23.07418397626113,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_receipts (avg)",
            "value": 5.456973293768546,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_state (avg)",
            "value": 14.17507418397626,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_transactions (avg)",
            "value": 5.427299703264095,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_trie_updates_sorted (avg)",
            "value": 15.985163204747774,
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
          "id": "5a07bc3f93f70f4bac64c8ed7a3a3791d563b78f",
          "message": "build(deps): bump core v1.0.0 (#3135)",
          "timestamp": "2026-03-05T07:40:26Z",
          "tree_id": "6a5024806183e9888e830d7b913ae82fc0ed211e",
          "url": "https://github.com/evstack/ev-node/commit/5a07bc3f93f70f4bac64c8ed7a3a3791d563b78f"
        },
        "date": 1772697682397,
        "tool": "customSmallerIsBetter",
        "benches": [
          {
            "name": "SpamoorSmoke - Batch.Commit (avg)",
            "value": 73.91304347826087,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.Put (avg)",
            "value": 3.338235294117647,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SaveBlockData (avg)",
            "value": 15.456521739130435,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SetHeight (avg)",
            "value": 15.782608695652174,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.UpdateState (avg)",
            "value": 6.1521739130434785,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ApplyBlock (avg)",
            "value": 89421.84782608696,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.CreateBlock (avg)",
            "value": 5.608695652173913,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ProduceBlock (avg)",
            "value": 90540.43478260869,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.RetrieveBatch (avg)",
            "value": 48.97826086956522,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Creating db provider (avg)",
            "value": 6.402077151335312,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DA.Submit (avg)",
            "value": 1011.9090909090909,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitData (avg)",
            "value": 4829,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitHeaders (avg)",
            "value": 1014.35,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DatabaseProvider::commit (avg)",
            "value": 3757.8575667655787,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.ForkchoiceUpdated (avg)",
            "value": 990.2518518518518,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.GetPayload (avg)",
            "value": 37608.391304347824,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.NewPayload (avg)",
            "value": 46875.04347826087,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Eth.GetBlockByNumber (avg)",
            "value": 1129.9333333333334,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.ExecuteTxs (avg)",
            "value": 89398.30434782608,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.GetTxs (avg)",
            "value": 993,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.SetFinal (avg)",
            "value": 1666.860465116279,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - ForcedInclusionRetriever.RetrieveForcedIncludedTxs (avg)",
            "value": 10.021739130434783,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Pruner::run_with_provider (avg)",
            "value": 94.38235294117646,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.GetNextBatch (avg)",
            "value": 38.28260869565217,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.SubmitBatchTxs (avg)",
            "value": 126,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - StaticFileProviderRW::finalize (avg)",
            "value": 1059.0316518298714,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - StaticFileWriters::finalize (avg)",
            "value": 3224.4421364985164,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.DeleteMetadata (avg)",
            "value": 21.90909090909091,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetBlockData (avg)",
            "value": 10.278026905829597,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetHeader (avg)",
            "value": 10.744897959183673,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetMetadata (avg)",
            "value": 9.968668407310705,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.Height (avg)",
            "value": 16.486692015209126,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.NewBatch (avg)",
            "value": 15.565217391304348,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.SetMetadata (avg)",
            "value": 28.565217391304348,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Tx::commit (avg)",
            "value": 519.106824925816,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - TxPool.GetTxs (avg)",
            "value": 977.75,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - accounts (avg)",
            "value": 1.2522522522522523,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - begin (avg)",
            "value": 20.61242603550296,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - build_payload (avg)",
            "value": 435.04481792717087,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - cache_for (avg)",
            "value": 50.00890207715133,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - calculate_overlay (avg)",
            "value": 7.114413075780089,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - commit (avg)",
            "value": 425.3620178041543,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - compute_state_root_parallel (avg)",
            "value": 221.99107142857142,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - compute_trie_input_task (avg)",
            "value": 322.20414201183434,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - connection (avg)",
            "value": 22,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - contracts (avg)",
            "value": 2.744807121661721,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - convert_to_block (avg)",
            "value": 12.679525222551929,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - database_provider_ro (avg)",
            "value": 33.102222222222224,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - evm env (avg)",
            "value": 2.887240356083086,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execute_block (avg)",
            "value": 145.46884272997033,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execution (avg)",
            "value": 1.2751479289940828,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - finish (avg)",
            "value": 9.3946587537092,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - get_cache_for (avg)",
            "value": 29.338278931750743,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - get_overlay (avg)",
            "value": 17.08160237388724,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - hashed_post_state (avg)",
            "value": 1.1253731343283582,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_block_mdbx_only (avg)",
            "value": 91.25443786982248,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_block_or_payload (avg)",
            "value": 816.6379821958457,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_state (avg)",
            "value": 18.01780415430267,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - merge transitions (avg)",
            "value": 1.0851063829787233,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_forkchoice_updated (avg)",
            "value": 60.52390438247012,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_new_payload (avg)",
            "value": 858.0949554896142,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_save_blocks (avg)",
            "value": 5427.323442136499,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - poll_next_event (avg)",
            "value": 7.4208924949290065,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - pre execution (avg)",
            "value": 23.587537091988132,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prewarm and caching (avg)",
            "value": 519.8278931750742,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prune_segments (avg)",
            "value": 3.323529411764706,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - save_blocks (avg)",
            "value": 1426.610119047619,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - save_cache (avg)",
            "value": 328.35311572700294,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_cache_exclusive (avg)",
            "value": 82.31157270029674,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_payload_processor (avg)",
            "value": 86.93471810089021,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - state provider (avg)",
            "value": 39.62611275964392,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - trie_data (avg)",
            "value": 5.29037037037037,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_build (avg)",
            "value": 493.26610644257704,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_connect_buffered_blocks (avg)",
            "value": 5.279761904761905,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_insert_payload (avg)",
            "value": 835.8249258160238,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - update_history_indices (avg)",
            "value": 11.065088757396449,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - update_pipeline_stages (avg)",
            "value": 30.22189349112426,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_inner (avg)",
            "value": 4.1335311572700295,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_post_execution (avg)",
            "value": 1.804154302670623,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_post_execution_with_hashed_state (avg)",
            "value": 1.8353293413173652,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_with_state (avg)",
            "value": 671.6547619047619,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_header_against_parent (avg)",
            "value": 1.319402985074627,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_post_execution (avg)",
            "value": 23.37091988130564,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - wait_cloned (avg)",
            "value": 86.25198098256736,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_blocks_data (avg)",
            "value": 1218.2307692307693,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_headers (avg)",
            "value": 34.48816568047337,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_receipts (avg)",
            "value": 7.455621301775148,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_state (avg)",
            "value": 23.74852071005917,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_transactions (avg)",
            "value": 7.050295857988166,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_trie_updates_sorted (avg)",
            "value": 17.236686390532544,
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
          "distinct": false,
          "id": "b1e3010e428a0d1bad3a0ef2fcd8df821a88cc6c",
          "message": "build(deps): Bump rollup from 4.22.4 to 4.59.0 in /docs in the npm_and_yarn group across 1 directory (#3136)\n\nbuild(deps): Bump rollup\n\nBumps the npm_and_yarn group with 1 update in the /docs directory: [rollup](https://github.com/rollup/rollup).\n\n\nUpdates `rollup` from 4.22.4 to 4.59.0\n- [Release notes](https://github.com/rollup/rollup/releases)\n- [Changelog](https://github.com/rollup/rollup/blob/master/CHANGELOG.md)\n- [Commits](https://github.com/rollup/rollup/compare/v4.22.4...v4.59.0)\n\n---\nupdated-dependencies:\n- dependency-name: rollup\n  dependency-version: 4.59.0\n  dependency-type: indirect\n  dependency-group: npm_and_yarn\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-03-05T09:26:29Z",
          "tree_id": "b1eee9dfeaba9650a397d7a0e7052541d2c80540",
          "url": "https://github.com/evstack/ev-node/commit/b1e3010e428a0d1bad3a0ef2fcd8df821a88cc6c"
        },
        "date": 1772704128882,
        "tool": "customSmallerIsBetter",
        "benches": [
          {
            "name": "SpamoorSmoke - Batch.Commit (avg)",
            "value": 89.47368421052632,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.Put (avg)",
            "value": 2.971830985915493,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SaveBlockData (avg)",
            "value": 16.425531914893618,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.SetHeight (avg)",
            "value": 12.446808510638299,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Batch.UpdateState (avg)",
            "value": 12.27659574468085,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ApplyBlock (avg)",
            "value": 68850.72340425532,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.CreateBlock (avg)",
            "value": 7.916666666666667,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.ProduceBlock (avg)",
            "value": 69965.51063829787,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - BlockExecutor.RetrieveBatch (avg)",
            "value": 40.083333333333336,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Creating db provider (avg)",
            "value": 6.974814814814815,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DA.Submit (avg)",
            "value": 1014.2352941176471,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitData (avg)",
            "value": 3956.5,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DASubmitter.SubmitHeaders (avg)",
            "value": 984.1333333333333,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - DatabaseProvider::commit (avg)",
            "value": 3432.46884272997,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.ForkchoiceUpdated (avg)",
            "value": 807.8943661971831,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.GetPayload (avg)",
            "value": 30242.854166666668,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Engine.NewPayload (avg)",
            "value": 33365.458333333336,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Eth.GetBlockByNumber (avg)",
            "value": 972.4265734265734,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.ExecuteTxs (avg)",
            "value": 68836.74468085106,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.GetTxs (avg)",
            "value": 5909.428571428572,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Executor.SetFinal (avg)",
            "value": 1675.2340425531916,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - ForcedInclusionRetriever.RetrieveForcedIncludedTxs (avg)",
            "value": 6.282608695652174,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - IpfsDHT.Provide (avg)",
            "value": 188,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - KademliaDHT.IpfsDHT.GetClosestPeers (avg)",
            "value": 9,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - KademliaDHT.IpfsDHT.RunLookupWithFollowup (avg)",
            "value": 3,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - KademliaDHT.IpfsDHT.RunQuery (avg)",
            "value": 1,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - KademliaDHT.ProviderManager.AddProvider (avg)",
            "value": 147,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Pruner::run_with_provider (avg)",
            "value": 97.94029850746269,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.GetNextBatch (avg)",
            "value": 30.291666666666668,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Sequencer.SubmitBatchTxs (avg)",
            "value": 108,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - StaticFileProviderRW::finalize (avg)",
            "value": 972.1633663366337,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - StaticFileWriters::finalize (avg)",
            "value": 2958.246290801187,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.DeleteMetadata (avg)",
            "value": 25.274509803921568,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetBlockData (avg)",
            "value": 11.4149377593361,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetHeader (avg)",
            "value": 8.888324873096447,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.GetMetadata (avg)",
            "value": 10.693827160493827,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.Height (avg)",
            "value": 12.227906976744187,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.NewBatch (avg)",
            "value": 14.631578947368421,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Store.SetMetadata (avg)",
            "value": 25.127340823970037,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - Tx::commit (avg)",
            "value": 462.43026706231456,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - TxPool.GetTxs (avg)",
            "value": 5782.571428571428,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - accounts (avg)",
            "value": 1.13595166163142,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - begin (avg)",
            "value": 17.367952522255194,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - build_payload (avg)",
            "value": 386.4103260869565,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - cache_for (avg)",
            "value": 47.585798816568044,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - calculate_overlay (avg)",
            "value": 5.187221396731055,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - commit (avg)",
            "value": 373.0207715133531,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - compute_state_root_parallel (avg)",
            "value": 214.86904761904762,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - compute_trie_input_task (avg)",
            "value": 304.946587537092,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - connection (avg)",
            "value": 150,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - contracts (avg)",
            "value": 2.0474777448071215,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - convert_to_block (avg)",
            "value": 12.596439169139465,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - database_provider_ro (avg)",
            "value": 29.444444444444443,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - evm env (avg)",
            "value": 2.798816568047337,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execute_block (avg)",
            "value": 133.15727002967358,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - execution (avg)",
            "value": 1.2746268656716417,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - finish (avg)",
            "value": 9.005917159763314,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - get_cache_for (avg)",
            "value": 27.807692307692307,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - get_overlay (avg)",
            "value": 13.91715976331361,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - hashed_post_state (avg)",
            "value": 1.063063063063063,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_block_mdbx_only (avg)",
            "value": 85.50445103857567,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_block_or_payload (avg)",
            "value": 793.9821958456973,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - insert_state (avg)",
            "value": 15.359050445103858,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - merge transitions (avg)",
            "value": 1.0267175572519085,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_forkchoice_updated (avg)",
            "value": 58.481701285855586,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_new_payload (avg)",
            "value": 829.540059347181,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - on_save_blocks (avg)",
            "value": 4971.050445103858,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - poll_next_event (avg)",
            "value": 6.792957508041307,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - pre execution (avg)",
            "value": 20.801186943620177,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prewarm and caching (avg)",
            "value": 507.4213649851632,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - prune_segments (avg)",
            "value": 2.1194029850746268,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - save_blocks (avg)",
            "value": 1309.4629080118693,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - save_cache (avg)",
            "value": 320.1513353115727,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_cache_exclusive (avg)",
            "value": 78.3620178041543,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - spawn_payload_processor (avg)",
            "value": 82.45994065281899,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - state provider (avg)",
            "value": 39.446745562130175,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - trie_data (avg)",
            "value": 4.524517087667162,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_build (avg)",
            "value": 442.35967302452315,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_connect_buffered_blocks (avg)",
            "value": 4.667655786350148,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - try_insert_payload (avg)",
            "value": 809.6409495548961,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - update_history_indices (avg)",
            "value": 10.761904761904763,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - update_pipeline_stages (avg)",
            "value": 32.89910979228487,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_inner (avg)",
            "value": 3.464497041420118,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_post_execution (avg)",
            "value": 1.8254437869822486,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_post_execution_with_hashed_state (avg)",
            "value": 1.2774390243902438,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_block_with_state (avg)",
            "value": 645.8011869436202,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_header_against_parent (avg)",
            "value": 1.272189349112426,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - validate_post_execution (avg)",
            "value": 21.504451038575667,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - wait_cloned (avg)",
            "value": 77.39670658682634,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_blocks_data (avg)",
            "value": 1123.652818991098,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_headers (avg)",
            "value": 25.327380952380953,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_receipts (avg)",
            "value": 5.465875370919881,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_state (avg)",
            "value": 17.50741839762611,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_transactions (avg)",
            "value": 5.724035608308605,
            "unit": "us"
          },
          {
            "name": "SpamoorSmoke - write_trie_updates_sorted (avg)",
            "value": 17.4540059347181,
            "unit": "us"
          }
        ]
      }
    ]
  }
}