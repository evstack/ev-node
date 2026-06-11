window.BENCHMARK_DATA = {
  "lastUpdate": 1781180759525,
  "repoUrl": "https://github.com/evstack/ev-node",
  "entries": {
    "EVM Contract Roundtrip": [
      {
        "commit": {
          "author": {
            "email": "nuke-web3@proton.me",
            "name": "Nuke 🌄",
            "username": "nuke-web3"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "74efee38260cd5072000a3a1ef349ae6a408fe1a",
          "message": "Add locad devnet guide (#3350)\n\n* oneshot local devnet guide w/ claude\n\n* tested to work with docker, but really upstream tooling needs some tweaks",
          "timestamp": "2026-06-11T13:01:13+02:00",
          "tree_id": "bb7035bd753c9e0fd8874d4a84d0c8025bebd379",
          "url": "https://github.com/evstack/ev-node/commit/74efee38260cd5072000a3a1ef349ae6a408fe1a"
        },
        "date": 1781176172377,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 895622446,
            "unit": "ns/op\t30641340 B/op\t  165486 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 895622446,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 30641340,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 165486,
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
          "id": "29e854269b75c3fb222efc7c8f49821485fa5d27",
          "message": "build(deps): Bump codecov/codecov-action from 6 to 7 (#3348)\n\nBumps [codecov/codecov-action](https://github.com/codecov/codecov-action) from 6 to 7.\n- [Release notes](https://github.com/codecov/codecov-action/releases)\n- [Changelog](https://github.com/codecov/codecov-action/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/codecov/codecov-action/compare/v6...v7)\n\n---\nupdated-dependencies:\n- dependency-name: codecov/codecov-action\n  dependency-version: '7'\n  dependency-type: direct:production\n  update-type: version-update:semver-major\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-06-11T13:04:00+02:00",
          "tree_id": "abcef3b79a2f5784521d5b8a885ddb4676ec27d3",
          "url": "https://github.com/evstack/ev-node/commit/29e854269b75c3fb222efc7c8f49821485fa5d27"
        },
        "date": 1781176177103,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 895332713,
            "unit": "ns/op\t30646116 B/op\t  162045 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 895332713,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 30646116,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 162045,
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
          "id": "3fed7009e42b834e76d04b8e8be928ac0987efa1",
          "message": "feat(loadgen): adding loadgen binary (#3335)\n\n* feat: benchmarks container image\n\n* fix: entrypoint\n\n* fix\n\n* Fix typo in README for benchmark runner command\n\n* Update Dockerfile to copy ev-benchmarks\n\n* Update crontab\n\n* feat(benchmarks): add ev-benchmarks cobra CLI for stress testing via spamoor\n\nStandalone Go binary that orchestrates spamoor-daemon via HTTP API to\ndrive sustained and burst transaction load against ev-reth. Structured\nas a cobra CLI with subcommands: check (connectivity), regular (~1M\ntx/day), burst (500K probabilistic), and run (custom matrix).\n\nIncludes Dockerfile, docker-compose, baseline/burst matrices, crontab\nfor supercronic scheduling, just targets for build and smoke testing.\n\n* feat(benchmarks): replace supercronic with in-process scheduler\n\nReplace separate regular/burst subcommands invoked by supercronic with\na single `start` command that runs an infinite scheduling loop. Regular\nworkloads fire on a ticker, bursts at random times within rolling 24h\nwindows. All timing controlled via CLI flags with env fallbacks.\n\n- remove supercronic from Dockerfile\n- delete regular.go, burst.go, crontab\n- add start.go with scheduler loop\n- plumb context for graceful shutdown\n- unify ExecuteMatrix/ExecuteMatrixWithOverrides via matrixOpts\n- bump fees (BASE_FEE 20→500, TIP_FEE 2→50)\n- remove healthcheck from docker-compose, rely on WaitForSync\n\n* feat(loadgen): rename benchmarks app and add tests\n\n* chore: simplify ci, use struct instead of raw json in tests\n\n* chore: rename bench -> loadgen\n\n* chore: use correct block height\n\n* fix(loadgen): address PR review feedback\n\n- run container as non-root user\n- use ParseUint for numeric env values in scenario config\n- add language identifiers to fenced code blocks (MD040)\n- wrap bare errors with context in client.go\n- remove redundant root cmd in flags_test.go\n- validate timeout duration during matrix validation\n- remove startup banner from init()\n\nCo-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>\n\n* chore: ensure possibility for concurrent spammers if time window is short\n\n* fix(loadgen): fix burst scheduling window and shutdown races\n\n- use remaining time until next reset for burst timer instead of full 24h window\n- track workload goroutines with WaitGroup, wait before cleanup on shutdown\n\nCo-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>\n\n* feat(loadgen): add burst subcommand and make scheduled bursts opt-in\n\nDefault burst-per-day changed from 2 to 0 so the start command\nno longer fires bursts unless explicitly configured. New burst\nsubcommand triggers a single burst workload immediately, reusing\nthe same matrix execution path as the scheduler.\n\n* fix(loadgen): filter spamoor counters by spammer name prefix\n\nCounter-based progress tracking summed all spammer metrics globally,\ncausing overshoot when concurrent workloads (e.g. baseline + burst)\nran on the same spamoor instance. Filter by spammer_name label prefix\nso each workload only tracks its own spammers.\n\n* refactor(loadgen): split internal package and simplify\n\n- Split flat internal/ into internal/{matrix,spamoor,runner} packages\n- Remove check subcommand (unused)\n- Remove intermediate exported functions (ExecuteMatrix, ExecuteMatrixWithOverrides)\n- Remove double validation in executeMatrix (already done by matrix.Load)\n- Extract metric name and env var constants\n- Store parsed timeout on Entry to avoid double time.ParseDuration\n- Replace time.NewTimer per-iteration with single ticker in waitForSync\n- Fix double time.Since call in poll loop\n- Update README: fix burst-per-day default (0 not 2), remove stale\n  serialization claim, document all subcommands clearly\n\n* chore: apply spammoor fix\n\n---------\n\nCo-authored-by: auricom <27022259+auricom@users.noreply.github.com>\nCo-authored-by: Claude Opus 4.6 <noreply@anthropic.com>",
          "timestamp": "2026-06-11T12:04:58Z",
          "tree_id": "0478428595db91bd3329d5fc309544ff100ea17a",
          "url": "https://github.com/evstack/ev-node/commit/3fed7009e42b834e76d04b8e8be928ac0987efa1"
        },
        "date": 1781180753356,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 890656928,
            "unit": "ns/op\t31995824 B/op\t  180207 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 890656928,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 31995824,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 180207,
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
            "email": "nuke-web3@proton.me",
            "name": "Nuke 🌄",
            "username": "nuke-web3"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "74efee38260cd5072000a3a1ef349ae6a408fe1a",
          "message": "Add locad devnet guide (#3350)\n\n* oneshot local devnet guide w/ claude\n\n* tested to work with docker, but really upstream tooling needs some tweaks",
          "timestamp": "2026-06-11T13:01:13+02:00",
          "tree_id": "bb7035bd753c9e0fd8874d4a84d0c8025bebd379",
          "url": "https://github.com/evstack/ev-node/commit/74efee38260cd5072000a3a1ef349ae6a408fe1a"
        },
        "date": 1781176178291,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 39607,
            "unit": "ns/op\t    4803 B/op\t      50 allocs/op",
            "extra": "30816 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 39607,
            "unit": "ns/op",
            "extra": "30816 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4803,
            "unit": "B/op",
            "extra": "30816 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 50,
            "unit": "allocs/op",
            "extra": "30816 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 40052,
            "unit": "ns/op\t    5018 B/op\t      54 allocs/op",
            "extra": "29985 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 40052,
            "unit": "ns/op",
            "extra": "29985 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 5018,
            "unit": "B/op",
            "extra": "29985 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "29985 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 46709,
            "unit": "ns/op\t   10303 B/op\t      54 allocs/op",
            "extra": "26326 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 46709,
            "unit": "ns/op",
            "extra": "26326 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10303,
            "unit": "B/op",
            "extra": "26326 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "26326 times\n4 procs"
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
          "id": "29e854269b75c3fb222efc7c8f49821485fa5d27",
          "message": "build(deps): Bump codecov/codecov-action from 6 to 7 (#3348)\n\nBumps [codecov/codecov-action](https://github.com/codecov/codecov-action) from 6 to 7.\n- [Release notes](https://github.com/codecov/codecov-action/releases)\n- [Changelog](https://github.com/codecov/codecov-action/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/codecov/codecov-action/compare/v6...v7)\n\n---\nupdated-dependencies:\n- dependency-name: codecov/codecov-action\n  dependency-version: '7'\n  dependency-type: direct:production\n  update-type: version-update:semver-major\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-06-11T13:04:00+02:00",
          "tree_id": "abcef3b79a2f5784521d5b8a885ddb4676ec27d3",
          "url": "https://github.com/evstack/ev-node/commit/29e854269b75c3fb222efc7c8f49821485fa5d27"
        },
        "date": 1781176184141,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37543,
            "unit": "ns/op\t    4772 B/op\t      50 allocs/op",
            "extra": "32006 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37543,
            "unit": "ns/op",
            "extra": "32006 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4772,
            "unit": "B/op",
            "extra": "32006 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 50,
            "unit": "allocs/op",
            "extra": "32006 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 37909,
            "unit": "ns/op\t    4961 B/op\t      54 allocs/op",
            "extra": "32126 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 37909,
            "unit": "ns/op",
            "extra": "32126 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 4961,
            "unit": "B/op",
            "extra": "32126 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "32126 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 44248,
            "unit": "ns/op\t   10265 B/op\t      54 allocs/op",
            "extra": "27428 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 44248,
            "unit": "ns/op",
            "extra": "27428 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10265,
            "unit": "B/op",
            "extra": "27428 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "27428 times\n4 procs"
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
          "id": "3fed7009e42b834e76d04b8e8be928ac0987efa1",
          "message": "feat(loadgen): adding loadgen binary (#3335)\n\n* feat: benchmarks container image\n\n* fix: entrypoint\n\n* fix\n\n* Fix typo in README for benchmark runner command\n\n* Update Dockerfile to copy ev-benchmarks\n\n* Update crontab\n\n* feat(benchmarks): add ev-benchmarks cobra CLI for stress testing via spamoor\n\nStandalone Go binary that orchestrates spamoor-daemon via HTTP API to\ndrive sustained and burst transaction load against ev-reth. Structured\nas a cobra CLI with subcommands: check (connectivity), regular (~1M\ntx/day), burst (500K probabilistic), and run (custom matrix).\n\nIncludes Dockerfile, docker-compose, baseline/burst matrices, crontab\nfor supercronic scheduling, just targets for build and smoke testing.\n\n* feat(benchmarks): replace supercronic with in-process scheduler\n\nReplace separate regular/burst subcommands invoked by supercronic with\na single `start` command that runs an infinite scheduling loop. Regular\nworkloads fire on a ticker, bursts at random times within rolling 24h\nwindows. All timing controlled via CLI flags with env fallbacks.\n\n- remove supercronic from Dockerfile\n- delete regular.go, burst.go, crontab\n- add start.go with scheduler loop\n- plumb context for graceful shutdown\n- unify ExecuteMatrix/ExecuteMatrixWithOverrides via matrixOpts\n- bump fees (BASE_FEE 20→500, TIP_FEE 2→50)\n- remove healthcheck from docker-compose, rely on WaitForSync\n\n* feat(loadgen): rename benchmarks app and add tests\n\n* chore: simplify ci, use struct instead of raw json in tests\n\n* chore: rename bench -> loadgen\n\n* chore: use correct block height\n\n* fix(loadgen): address PR review feedback\n\n- run container as non-root user\n- use ParseUint for numeric env values in scenario config\n- add language identifiers to fenced code blocks (MD040)\n- wrap bare errors with context in client.go\n- remove redundant root cmd in flags_test.go\n- validate timeout duration during matrix validation\n- remove startup banner from init()\n\nCo-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>\n\n* chore: ensure possibility for concurrent spammers if time window is short\n\n* fix(loadgen): fix burst scheduling window and shutdown races\n\n- use remaining time until next reset for burst timer instead of full 24h window\n- track workload goroutines with WaitGroup, wait before cleanup on shutdown\n\nCo-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>\n\n* feat(loadgen): add burst subcommand and make scheduled bursts opt-in\n\nDefault burst-per-day changed from 2 to 0 so the start command\nno longer fires bursts unless explicitly configured. New burst\nsubcommand triggers a single burst workload immediately, reusing\nthe same matrix execution path as the scheduler.\n\n* fix(loadgen): filter spamoor counters by spammer name prefix\n\nCounter-based progress tracking summed all spammer metrics globally,\ncausing overshoot when concurrent workloads (e.g. baseline + burst)\nran on the same spamoor instance. Filter by spammer_name label prefix\nso each workload only tracks its own spammers.\n\n* refactor(loadgen): split internal package and simplify\n\n- Split flat internal/ into internal/{matrix,spamoor,runner} packages\n- Remove check subcommand (unused)\n- Remove intermediate exported functions (ExecuteMatrix, ExecuteMatrixWithOverrides)\n- Remove double validation in executeMatrix (already done by matrix.Load)\n- Extract metric name and env var constants\n- Store parsed timeout on Entry to avoid double time.ParseDuration\n- Replace time.NewTimer per-iteration with single ticker in waitForSync\n- Fix double time.Since call in poll loop\n- Update README: fix burst-per-day default (0 not 2), remove stale\n  serialization claim, document all subcommands clearly\n\n* chore: apply spammoor fix\n\n---------\n\nCo-authored-by: auricom <27022259+auricom@users.noreply.github.com>\nCo-authored-by: Claude Opus 4.6 <noreply@anthropic.com>",
          "timestamp": "2026-06-11T12:04:58Z",
          "tree_id": "0478428595db91bd3329d5fc309544ff100ea17a",
          "url": "https://github.com/evstack/ev-node/commit/3fed7009e42b834e76d04b8e8be928ac0987efa1"
        },
        "date": 1781180758963,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37127,
            "unit": "ns/op\t    4762 B/op\t      50 allocs/op",
            "extra": "32421 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37127,
            "unit": "ns/op",
            "extra": "32421 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4762,
            "unit": "B/op",
            "extra": "32421 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 50,
            "unit": "allocs/op",
            "extra": "32421 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 37513,
            "unit": "ns/op\t    4956 B/op\t      54 allocs/op",
            "extra": "32349 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 37513,
            "unit": "ns/op",
            "extra": "32349 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 4956,
            "unit": "B/op",
            "extra": "32349 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "32349 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 43868,
            "unit": "ns/op\t   10247 B/op\t      54 allocs/op",
            "extra": "27974 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 43868,
            "unit": "ns/op",
            "extra": "27974 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10247,
            "unit": "B/op",
            "extra": "27974 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "27974 times\n4 procs"
          }
        ]
      }
    ]
  }
}