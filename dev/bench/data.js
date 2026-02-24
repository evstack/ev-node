window.BENCHMARK_DATA = {
  "lastUpdate": 1771926757399,
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
      }
    ]
  }
}