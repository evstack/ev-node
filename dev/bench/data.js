window.BENCHMARK_DATA = {
  "lastUpdate": 1772138129369,
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
      }
    ]
  }
}