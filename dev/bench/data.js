window.BENCHMARK_DATA = {
  "lastUpdate": 1778505185902,
  "repoUrl": "https://github.com/evstack/ev-node",
  "entries": {
    "EVM Contract Roundtrip": [
      {
        "commit": {
          "author": {
            "email": "27022259+auricom@users.noreply.github.com",
            "name": "auricom",
            "username": "auricom"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "0900dc504b67821d6703431ee8fbe0bdd6de8318",
          "message": "docs: high availability sequencer guide (#3293)\n\n* docs: ev-node high availability\n\n* docs: node placement\n\n* docs(ha): address PR review feedback\n\nCritical fixes:\n- Fix snapshot_threshold math: 5000 ÷ 10 = 500s ≈ 8.3 min (not 83s)\n- Fix trailing_logs math: 18000 ÷ 10 = 1800s = 30 min (not 5 min)\n\nMedium fixes:\n- Fix heartbeat_timeout description: it is a follower-side election trigger,\n  not the interval at which the leader sends heartbeats\n- Add explicit restart instruction after Step 5 data copy in single-to-ha.md\n  so the chain keeps producing blocks during preparation (Steps 6-8)\n- Replace priv_validator_key.json with signer.json in single-to-ha.md\n  to match cluster-setup.md and the E2E tests\n\nMinor fixes:\n- Exclude self from raft.peers in all examples (cluster-setup.md node-1\n  yaml/CLI/systemd, single-to-ha.md node-1 and node-2)\n- Add \"exclude local node\" note to raft.peers description in overview.md\n- Fix P2P port in overview.md Interaction with P2P section (7676 → 26656)\n- Add text language tag to all bare fenced blocks (MD040): multiaddr\n  example, RTT equations, and all log snippets\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* docs(ha): absorb raft_production.md into ha/overview.md\n\nraft_production.md had no sidebar entry and its content was fully\nsuperseded by the new ha/ guides. Extract the three pieces that were\nunique to it — bootstrap flag docs, auto-detection startup mode\nexplanation, and static-membership limitation note — into\nha/overview.md, then delete the file.\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* docs(ha): use EnvironmentFile for signer passphrase\n\nPassing --evnode.signer.passphrase inline exposes the secret in\nps aux, journalctl, and shell history.\n\n- Add EnvironmentFile=/etc/ev-node/env (chmod 600) to the systemd\n  unit in cluster-setup.md with setup instructions\n- Replace all inline <YOUR_PASSPHRASE> occurrences with\n  $EV_SIGNER_PASSPHRASE sourced from /etc/ev-node/env in every\n  evm start / evm init snippet across both guides\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* docs(ha): explicit node-2 peers and action-based rolling restart\n\n- Replace \"peers list is identical\" stub in node-2 config with an\n  explicit peers list that excludes node-2 itself, and add a note\n  that each node must omit itself from raft.peers\n- Replace \"Wait ~30 seconds\" in rolling restart with journalctl\n  one-liners that exit as soon as the node logs follower/leader state,\n  giving a deterministic signal instead of an arbitrary timeout\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* docs(ha): fix raft.peers self-inclusion startup bug\n\nThe abbreviated node-2 snippet with \"# peers list is identical\" caused\na startup failure: with raft_addr=0.0.0.0:5001 the bootstrap code's\nliteral address comparison does not recognise node-2@10.0.0.2:5001 as\nself, so node-2 is appended twice and deduplicateServers returns\n\"duplicate peers found in config\".\n\n- Fix intro text: \"only raft.node_id and raft_addr differ\" →\n  \"raft.node_id is unique; raft.peers and p2p.peers must exclude self\"\n- Expand node-2 snippet to a full evnode.yaml with the correct peers\n  list (node-1, node-3, node-4, node-5 — no node-2) and an inline\n  explanation of the wildcard address pitfall\n- Align overview.md trailing_logs example to 1 block/s (matching\n  block_time: \"1s\" used throughout) and note the 10 block/s rate too\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* docs(ha): fix passphrase flag and failover kill cardinality check\n\nReplace non-existent --evnode.signer.passphrase with the actual\n--evnode.signer.passphrase_file flag throughout cluster-setup and\nsingle-to-ha guides. Update passphrase setup to create a chmod 600\nfile at /etc/ev-node/passphrase referenced directly by the flag.\n\nAdd mapfile-based cardinality check in the failover test fallback\nkill command to guard against killing the wrong process.\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* docs(ha): fix RPC endpoints, init ordering, and snap_count CLI flag\n\nReplace incorrect CometBFT RPC calls (port 26657/status) with the\nactual ev-node HTTP API (port 7331 /health/ready, /raft/node) and\nEVM execution layer (cast block latest) throughout both guides.\n\nAlign single-to-ha Step 2 init ordering with cluster-setup: create\npassphrase file before evm init so the signer key is encrypted from\nthe start, and pass --evnode.node.aggregator and passphrase_file flags.\n\nFix Step 9a fallback kill in single-to-ha to use mapfile cardinality\ncheck, matching the pattern already applied in cluster-setup.\n\nAdd --evnode.raft.snap_count=3 to the CLI start example to match\nthe YAML config block.\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n---------\n\nCo-authored-by: Claude Sonnet 4.6 <noreply@anthropic.com>",
          "timestamp": "2026-04-29T08:26:25Z",
          "tree_id": "87ae53ee2fd23172f4d08ae2e82d790f017c6252",
          "url": "https://github.com/evstack/ev-node/commit/0900dc504b67821d6703431ee8fbe0bdd6de8318"
        },
        "date": 1777452407874,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 898871345,
            "unit": "ns/op\t32267336 B/op\t  180756 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 898871345,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 32267336,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 180756,
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
          "id": "fdc79adf064927335bfd57f2240c1e1d26467c04",
          "message": "perf(store): save metadata async (#3298)\n\n* perf(store): save metadata async\n\n* cl\n\n* Optimize metadata writes with batching\n\n* feedback\n\n* De-duplicate batched writes by key in cached store\n\n* fix\n\n* updates",
          "timestamp": "2026-04-29T11:43:07Z",
          "tree_id": "a8cb2add8c0dec60d2b3b9ea46c11b127f15a7e1",
          "url": "https://github.com/evstack/ev-node/commit/fdc79adf064927335bfd57f2240c1e1d26467c04"
        },
        "date": 1777465744168,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 908413220,
            "unit": "ns/op\t30651044 B/op\t  161685 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 908413220,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 30651044,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 161685,
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
          "id": "1f1f48e3398eed95b062f79c8a9bff2f71395b01",
          "message": "chore(deps): security  (#3296)\n\n* fix security deps\n\n* fix helpers",
          "timestamp": "2026-04-29T14:01:56+02:00",
          "tree_id": "382b1a3bfe25b61487699863ff31fd6f58ab0fee",
          "url": "https://github.com/evstack/ev-node/commit/1f1f48e3398eed95b062f79c8a9bff2f71395b01"
        },
        "date": 1777465785022,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 926411624,
            "unit": "ns/op\t33528616 B/op\t  191025 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 926411624,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 33528616,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 191025,
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
          "id": "406369874b2faf7cf0664752f9284dc94ece780a",
          "message": "feat: add grpc socket and flattn tx batches to allow for lower allocations (#3297)\n\n* add grpc socket and flattn tx batches to allow for lower allocations\n\n* redo proto\n\n* docs: update changelog for grpc execution transport\n\n* remove extra txs",
          "timestamp": "2026-04-29T14:04:12+02:00",
          "tree_id": "73c5304f4d8c2dc277cd754e8912062c9f188052",
          "url": "https://github.com/evstack/ev-node/commit/406369874b2faf7cf0664752f9284dc94ece780a"
        },
        "date": 1777465800421,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 914700600,
            "unit": "ns/op\t32156080 B/op\t  176285 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 914700600,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 32156080,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 176285,
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
          "id": "05979c178033c40fe969f8d842cbc7a3d572b8f5",
          "message": "refactor(execution/grpc): move execution service where it belongs (#3302)\n\n* refactor(execution/grpc): move execution service where it belongs\n\n* reduce diff\n\n* fix lint",
          "timestamp": "2026-04-29T17:13:17+02:00",
          "tree_id": "78cf2234b6b93b8149dd51bbe9428301015dc40b",
          "url": "https://github.com/evstack/ev-node/commit/05979c178033c40fe969f8d842cbc7a3d572b8f5"
        },
        "date": 1777476021735,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 910171035,
            "unit": "ns/op\t31291388 B/op\t  171208 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 910171035,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 31291388,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 171208,
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
          "id": "03d0d4d60960ffbdb25333e7e0c3826211765b18",
          "message": "feat(execution/grpc): adding support for grpc otlp (#3300)\n\n* feat: adding support for grpc oltp\n\n* chore: fix linting\n\n* cl\n\n---------\n\nCo-authored-by: Julien Robert <julien@rbrt.fr>",
          "timestamp": "2026-04-30T07:36:10Z",
          "tree_id": "be39434dcabf24176c31edb11905bb09c232f470",
          "url": "https://github.com/evstack/ev-node/commit/03d0d4d60960ffbdb25333e7e0c3826211765b18"
        },
        "date": 1777535703203,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 915916748,
            "unit": "ns/op\t29984804 B/op\t  156875 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 915916748,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 29984804,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 156875,
            "unit": "allocs/op",
            "extra": "2 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "cricis@msn.com",
            "name": "criciss",
            "username": "criciss"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "f36fbd2001535adcaca881adb285d76bf1cbbd2b",
          "message": "chore: fix some minor issues in comments (#3304)",
          "timestamp": "2026-05-03T00:51:25+02:00",
          "tree_id": "a2556f1a189335253b7c9d4c5405f1e203516f7f",
          "url": "https://github.com/evstack/ev-node/commit/f36fbd2001535adcaca881adb285d76bf1cbbd2b"
        },
        "date": 1777762361974,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 915925797,
            "unit": "ns/op\t33364068 B/op\t  190745 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 915925797,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 33364068,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 190745,
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
          "id": "264314f342dfaa8123a12b24ded8081df1197b63",
          "message": "build(deps): Bump dorny/paths-filter from 3 to 4 (#3308)\n\nBumps [dorny/paths-filter](https://github.com/dorny/paths-filter) from 3 to 4.\n- [Release notes](https://github.com/dorny/paths-filter/releases)\n- [Changelog](https://github.com/dorny/paths-filter/blob/master/CHANGELOG.md)\n- [Commits](https://github.com/dorny/paths-filter/compare/v3...v4)\n\n---\nupdated-dependencies:\n- dependency-name: dorny/paths-filter\n  dependency-version: '4'\n  dependency-type: direct:production\n  update-type: version-update:semver-major\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-05-06T09:56:35+02:00",
          "tree_id": "3df3233fc98f51d3a165b907d1bec30a94cf1415",
          "url": "https://github.com/evstack/ev-node/commit/264314f342dfaa8123a12b24ded8081df1197b63"
        },
        "date": 1778054415780,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 918451522,
            "unit": "ns/op\t31548148 B/op\t  171719 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 918451522,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 31548148,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 171719,
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
          "id": "495c6215793c3efc3512ba8bde9cd4c17ebbf572",
          "message": "feat(pkg/sequencers): add queue limit in solo sequencer (#3312)\n\n* feat(pkg/sequencers): add queue limit in solo sequencer\n\n* use option\n\n* cl\n\n* move test files",
          "timestamp": "2026-05-06T16:25:26Z",
          "tree_id": "45f2f3e5c7b73d4a69a5cb9bf67ea602444bebfe",
          "url": "https://github.com/evstack/ev-node/commit/495c6215793c3efc3512ba8bde9cd4c17ebbf572"
        },
        "date": 1778085862694,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 891726465,
            "unit": "ns/op\t31477080 B/op\t  175260 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 891726465,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 31477080,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 175260,
            "unit": "allocs/op",
            "extra": "2 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "caelrowley@pm.me",
            "name": "Cael Rowley",
            "username": "CaelRowley"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "01791ca1b4b528357b70896c296023fa5e621f5d",
          "message": "test(config): use `synctest` in TestAddFlags for avoiding nondeterminism  (#3311)\n\n* fix(config): de-flake TestAddFlags by stubbing time source\n\n* test(config): de-flake TestAddFlags via testing/synctest",
          "timestamp": "2026-05-07T11:57:50+02:00",
          "tree_id": "1a3f313873483be0c46eb038f1cb6b6a0f86199f",
          "url": "https://github.com/evstack/ev-node/commit/01791ca1b4b528357b70896c296023fa5e621f5d"
        },
        "date": 1778148151924,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 924315706,
            "unit": "ns/op\t28769124 B/op\t  142243 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 924315706,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 28769124,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 142243,
            "unit": "allocs/op",
            "extra": "2 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "box4wangjing@outlook.com",
            "name": "box4wangjing",
            "username": "box4wangjing"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "45b5f3e8270be5cf6662cfec830bca5536422780",
          "message": "chore: remove extra space in comment (#3319)\n\nSigned-off-by: box4wangjing <box4wangjing@outlook.com>",
          "timestamp": "2026-05-11T09:20:11+02:00",
          "tree_id": "10c085511e072d8ae3cc93c93201714e508f5a8e",
          "url": "https://github.com/evstack/ev-node/commit/45b5f3e8270be5cf6662cfec830bca5536422780"
        },
        "date": 1778484239821,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 879660364,
            "unit": "ns/op\t27989628 B/op\t  144858 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 879660364,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 27989628,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 144858,
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
          "id": "acdbdac1756399cbb1757c21b4789de6d937787c",
          "message": "build(deps): Bump the all-go group across 3 directories with 9 updates (#3309)\n\n* build(deps): Bump the all-go group across 3 directories with 9 updates\n\nBumps the all-go group with 6 updates in the / directory:\n\n| Package | From | To |\n| --- | --- | --- |\n| [cloud.google.com/go/kms](https://github.com/googleapis/google-cloud-go) | `1.29.0` | `1.30.0` |\n| [github.com/aws/aws-sdk-go-v2](https://github.com/aws/aws-sdk-go-v2) | `1.41.6` | `1.41.7` |\n| [github.com/aws/aws-sdk-go-v2/config](https://github.com/aws/aws-sdk-go-v2) | `1.32.16` | `1.32.17` |\n| [github.com/aws/aws-sdk-go-v2/service/kms](https://github.com/aws/aws-sdk-go-v2) | `1.51.0` | `1.51.1` |\n| [google.golang.org/api](https://github.com/googleapis/google-api-go-client) | `0.276.0` | `0.277.0` |\n| [google.golang.org/grpc](https://github.com/grpc/grpc-go) | `1.80.0` | `1.81.0` |\n\nBumps the all-go group with 1 update in the /test/docker-e2e directory: [github.com/moby/moby/client](https://github.com/moby/moby).\nBumps the all-go group with 2 updates in the /test/e2e directory: [github.com/moby/moby/client](https://github.com/moby/moby) and [go.uber.org/zap](https://github.com/uber-go/zap).\n\n\nUpdates `cloud.google.com/go/kms` from 1.29.0 to 1.30.0\n- [Release notes](https://github.com/googleapis/google-cloud-go/releases)\n- [Changelog](https://github.com/googleapis/google-cloud-go/blob/main/documentai/CHANGES.md)\n- [Commits](https://github.com/googleapis/google-cloud-go/compare/kms/v1.29.0...dlp/v1.30.0)\n\nUpdates `github.com/aws/aws-sdk-go-v2` from 1.41.6 to 1.41.7\n- [Release notes](https://github.com/aws/aws-sdk-go-v2/releases)\n- [Commits](https://github.com/aws/aws-sdk-go-v2/compare/v1.41.6...v1.41.7)\n\nUpdates `github.com/aws/aws-sdk-go-v2/config` from 1.32.16 to 1.32.17\n- [Release notes](https://github.com/aws/aws-sdk-go-v2/releases)\n- [Commits](https://github.com/aws/aws-sdk-go-v2/compare/config/v1.32.16...config/v1.32.17)\n\nUpdates `github.com/aws/aws-sdk-go-v2/service/kms` from 1.51.0 to 1.51.1\n- [Release notes](https://github.com/aws/aws-sdk-go-v2/releases)\n- [Commits](https://github.com/aws/aws-sdk-go-v2/compare/service/s3/v1.51.0...service/s3/v1.51.1)\n\nUpdates `google.golang.org/api` from 0.276.0 to 0.277.0\n- [Release notes](https://github.com/googleapis/google-api-go-client/releases)\n- [Changelog](https://github.com/googleapis/google-api-go-client/blob/main/CHANGES.md)\n- [Commits](https://github.com/googleapis/google-api-go-client/compare/v0.276.0...v0.277.0)\n\nUpdates `google.golang.org/grpc` from 1.80.0 to 1.81.0\n- [Release notes](https://github.com/grpc/grpc-go/releases)\n- [Commits](https://github.com/grpc/grpc-go/compare/v1.80.0...v1.81.0)\n\nUpdates `github.com/moby/moby/client` from 0.4.0 to 0.4.1\n- [Release notes](https://github.com/moby/moby/releases)\n- [Changelog](https://github.com/moby/moby/blob/v0.4.1/CHANGELOG.md)\n- [Commits](https://github.com/moby/moby/compare/v0.4.0...v0.4.1)\n\nUpdates `github.com/moby/moby/client` from 0.4.0 to 0.4.1\n- [Release notes](https://github.com/moby/moby/releases)\n- [Changelog](https://github.com/moby/moby/blob/v0.4.1/CHANGELOG.md)\n- [Commits](https://github.com/moby/moby/compare/v0.4.0...v0.4.1)\n\nUpdates `github.com/moby/moby/api` from 1.54.1 to 1.54.2\n- [Release notes](https://github.com/moby/moby/releases)\n- [Commits](https://github.com/moby/moby/compare/api/v1.54.1...api/v1.54.2)\n\nUpdates `go.uber.org/zap` from 1.27.1 to 1.28.0\n- [Release notes](https://github.com/uber-go/zap/releases)\n- [Changelog](https://github.com/uber-go/zap/blob/master/CHANGELOG.md)\n- [Commits](https://github.com/uber-go/zap/compare/v1.27.1...v1.28.0)\n\n---\nupdated-dependencies:\n- dependency-name: cloud.google.com/go/kms\n  dependency-version: 1.30.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/aws/aws-sdk-go-v2\n  dependency-version: 1.41.7\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/aws/aws-sdk-go-v2/config\n  dependency-version: 1.32.17\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/aws/aws-sdk-go-v2/service/kms\n  dependency-version: 1.51.1\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: google.golang.org/api\n  dependency-version: 0.277.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: google.golang.org/grpc\n  dependency-version: 1.81.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/moby/moby/client\n  dependency-version: 0.4.1\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/moby/moby/client\n  dependency-version: 0.4.1\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/moby/moby/api\n  dependency-version: 1.54.2\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: go.uber.org/zap\n  dependency-version: 1.28.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\n\n* chore: run just deps after Dependabot update\n\n---------\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>\nCo-authored-by: github-actions[bot] <41898282+github-actions[bot]@users.noreply.github.com>\nCo-authored-by: julienrbrt <julien@rbrt.fr>",
          "timestamp": "2026-05-11T12:53:31Z",
          "tree_id": "f35ca3b99108df7b4f003a833bc6f23d5d6836ca",
          "url": "https://github.com/evstack/ev-node/commit/acdbdac1756399cbb1757c21b4789de6d937787c"
        },
        "date": 1778505181201,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 915244148,
            "unit": "ns/op\t30986544 B/op\t  166293 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 915244148,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 30986544,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 166293,
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
            "email": "27022259+auricom@users.noreply.github.com",
            "name": "auricom",
            "username": "auricom"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "0900dc504b67821d6703431ee8fbe0bdd6de8318",
          "message": "docs: high availability sequencer guide (#3293)\n\n* docs: ev-node high availability\n\n* docs: node placement\n\n* docs(ha): address PR review feedback\n\nCritical fixes:\n- Fix snapshot_threshold math: 5000 ÷ 10 = 500s ≈ 8.3 min (not 83s)\n- Fix trailing_logs math: 18000 ÷ 10 = 1800s = 30 min (not 5 min)\n\nMedium fixes:\n- Fix heartbeat_timeout description: it is a follower-side election trigger,\n  not the interval at which the leader sends heartbeats\n- Add explicit restart instruction after Step 5 data copy in single-to-ha.md\n  so the chain keeps producing blocks during preparation (Steps 6-8)\n- Replace priv_validator_key.json with signer.json in single-to-ha.md\n  to match cluster-setup.md and the E2E tests\n\nMinor fixes:\n- Exclude self from raft.peers in all examples (cluster-setup.md node-1\n  yaml/CLI/systemd, single-to-ha.md node-1 and node-2)\n- Add \"exclude local node\" note to raft.peers description in overview.md\n- Fix P2P port in overview.md Interaction with P2P section (7676 → 26656)\n- Add text language tag to all bare fenced blocks (MD040): multiaddr\n  example, RTT equations, and all log snippets\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* docs(ha): absorb raft_production.md into ha/overview.md\n\nraft_production.md had no sidebar entry and its content was fully\nsuperseded by the new ha/ guides. Extract the three pieces that were\nunique to it — bootstrap flag docs, auto-detection startup mode\nexplanation, and static-membership limitation note — into\nha/overview.md, then delete the file.\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* docs(ha): use EnvironmentFile for signer passphrase\n\nPassing --evnode.signer.passphrase inline exposes the secret in\nps aux, journalctl, and shell history.\n\n- Add EnvironmentFile=/etc/ev-node/env (chmod 600) to the systemd\n  unit in cluster-setup.md with setup instructions\n- Replace all inline <YOUR_PASSPHRASE> occurrences with\n  $EV_SIGNER_PASSPHRASE sourced from /etc/ev-node/env in every\n  evm start / evm init snippet across both guides\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* docs(ha): explicit node-2 peers and action-based rolling restart\n\n- Replace \"peers list is identical\" stub in node-2 config with an\n  explicit peers list that excludes node-2 itself, and add a note\n  that each node must omit itself from raft.peers\n- Replace \"Wait ~30 seconds\" in rolling restart with journalctl\n  one-liners that exit as soon as the node logs follower/leader state,\n  giving a deterministic signal instead of an arbitrary timeout\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* docs(ha): fix raft.peers self-inclusion startup bug\n\nThe abbreviated node-2 snippet with \"# peers list is identical\" caused\na startup failure: with raft_addr=0.0.0.0:5001 the bootstrap code's\nliteral address comparison does not recognise node-2@10.0.0.2:5001 as\nself, so node-2 is appended twice and deduplicateServers returns\n\"duplicate peers found in config\".\n\n- Fix intro text: \"only raft.node_id and raft_addr differ\" →\n  \"raft.node_id is unique; raft.peers and p2p.peers must exclude self\"\n- Expand node-2 snippet to a full evnode.yaml with the correct peers\n  list (node-1, node-3, node-4, node-5 — no node-2) and an inline\n  explanation of the wildcard address pitfall\n- Align overview.md trailing_logs example to 1 block/s (matching\n  block_time: \"1s\" used throughout) and note the 10 block/s rate too\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* docs(ha): fix passphrase flag and failover kill cardinality check\n\nReplace non-existent --evnode.signer.passphrase with the actual\n--evnode.signer.passphrase_file flag throughout cluster-setup and\nsingle-to-ha guides. Update passphrase setup to create a chmod 600\nfile at /etc/ev-node/passphrase referenced directly by the flag.\n\nAdd mapfile-based cardinality check in the failover test fallback\nkill command to guard against killing the wrong process.\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* docs(ha): fix RPC endpoints, init ordering, and snap_count CLI flag\n\nReplace incorrect CometBFT RPC calls (port 26657/status) with the\nactual ev-node HTTP API (port 7331 /health/ready, /raft/node) and\nEVM execution layer (cast block latest) throughout both guides.\n\nAlign single-to-ha Step 2 init ordering with cluster-setup: create\npassphrase file before evm init so the signer key is encrypted from\nthe start, and pass --evnode.node.aggregator and passphrase_file flags.\n\nFix Step 9a fallback kill in single-to-ha to use mapfile cardinality\ncheck, matching the pattern already applied in cluster-setup.\n\nAdd --evnode.raft.snap_count=3 to the CLI start example to match\nthe YAML config block.\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n---------\n\nCo-authored-by: Claude Sonnet 4.6 <noreply@anthropic.com>",
          "timestamp": "2026-04-29T08:26:25Z",
          "tree_id": "87ae53ee2fd23172f4d08ae2e82d790f017c6252",
          "url": "https://github.com/evstack/ev-node/commit/0900dc504b67821d6703431ee8fbe0bdd6de8318"
        },
        "date": 1777452413202,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37274,
            "unit": "ns/op\t    4756 B/op\t      50 allocs/op",
            "extra": "32680 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37274,
            "unit": "ns/op",
            "extra": "32680 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4756,
            "unit": "B/op",
            "extra": "32680 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 50,
            "unit": "allocs/op",
            "extra": "32680 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 37866,
            "unit": "ns/op\t    4960 B/op\t      54 allocs/op",
            "extra": "32192 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 37866,
            "unit": "ns/op",
            "extra": "32192 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 4960,
            "unit": "B/op",
            "extra": "32192 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "32192 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 43911,
            "unit": "ns/op\t   10248 B/op\t      54 allocs/op",
            "extra": "27952 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 43911,
            "unit": "ns/op",
            "extra": "27952 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10248,
            "unit": "B/op",
            "extra": "27952 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "27952 times\n4 procs"
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
          "id": "fdc79adf064927335bfd57f2240c1e1d26467c04",
          "message": "perf(store): save metadata async (#3298)\n\n* perf(store): save metadata async\n\n* cl\n\n* Optimize metadata writes with batching\n\n* feedback\n\n* De-duplicate batched writes by key in cached store\n\n* fix\n\n* updates",
          "timestamp": "2026-04-29T11:43:07Z",
          "tree_id": "a8cb2add8c0dec60d2b3b9ea46c11b127f15a7e1",
          "url": "https://github.com/evstack/ev-node/commit/fdc79adf064927335bfd57f2240c1e1d26467c04"
        },
        "date": 1777465750050,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 38288,
            "unit": "ns/op\t    4758 B/op\t      50 allocs/op",
            "extra": "32592 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 38288,
            "unit": "ns/op",
            "extra": "32592 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4758,
            "unit": "B/op",
            "extra": "32592 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 50,
            "unit": "allocs/op",
            "extra": "32592 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 38821,
            "unit": "ns/op\t    4991 B/op\t      54 allocs/op",
            "extra": "30937 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 38821,
            "unit": "ns/op",
            "extra": "30937 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 4991,
            "unit": "B/op",
            "extra": "30937 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "30937 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 45234,
            "unit": "ns/op\t   10293 B/op\t      54 allocs/op",
            "extra": "26604 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 45234,
            "unit": "ns/op",
            "extra": "26604 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10293,
            "unit": "B/op",
            "extra": "26604 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "26604 times\n4 procs"
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
          "id": "1f1f48e3398eed95b062f79c8a9bff2f71395b01",
          "message": "chore(deps): security  (#3296)\n\n* fix security deps\n\n* fix helpers",
          "timestamp": "2026-04-29T14:01:56+02:00",
          "tree_id": "382b1a3bfe25b61487699863ff31fd6f58ab0fee",
          "url": "https://github.com/evstack/ev-node/commit/1f1f48e3398eed95b062f79c8a9bff2f71395b01"
        },
        "date": 1777465792010,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 35548,
            "unit": "ns/op\t    4719 B/op\t      50 allocs/op",
            "extra": "34321 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 35548,
            "unit": "ns/op",
            "extra": "34321 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4719,
            "unit": "B/op",
            "extra": "34321 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 50,
            "unit": "allocs/op",
            "extra": "34321 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 35540,
            "unit": "ns/op\t    4923 B/op\t      54 allocs/op",
            "extra": "33762 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 35540,
            "unit": "ns/op",
            "extra": "33762 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 4923,
            "unit": "B/op",
            "extra": "33762 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "33762 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 41470,
            "unit": "ns/op\t   10208 B/op\t      54 allocs/op",
            "extra": "29217 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 41470,
            "unit": "ns/op",
            "extra": "29217 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10208,
            "unit": "B/op",
            "extra": "29217 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "29217 times\n4 procs"
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
          "id": "406369874b2faf7cf0664752f9284dc94ece780a",
          "message": "feat: add grpc socket and flattn tx batches to allow for lower allocations (#3297)\n\n* add grpc socket and flattn tx batches to allow for lower allocations\n\n* redo proto\n\n* docs: update changelog for grpc execution transport\n\n* remove extra txs",
          "timestamp": "2026-04-29T14:04:12+02:00",
          "tree_id": "73c5304f4d8c2dc277cd754e8912062c9f188052",
          "url": "https://github.com/evstack/ev-node/commit/406369874b2faf7cf0664752f9284dc94ece780a"
        },
        "date": 1777465806328,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 37607,
            "unit": "ns/op\t    4975 B/op\t      54 allocs/op",
            "extra": "31592 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 37607,
            "unit": "ns/op",
            "extra": "31592 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 4975,
            "unit": "B/op",
            "extra": "31592 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "31592 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 44134,
            "unit": "ns/op\t   10240 B/op\t      54 allocs/op",
            "extra": "28201 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 44134,
            "unit": "ns/op",
            "extra": "28201 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10240,
            "unit": "B/op",
            "extra": "28201 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "28201 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37320,
            "unit": "ns/op\t    4780 B/op\t      50 allocs/op",
            "extra": "31692 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37320,
            "unit": "ns/op",
            "extra": "31692 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4780,
            "unit": "B/op",
            "extra": "31692 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 50,
            "unit": "allocs/op",
            "extra": "31692 times\n4 procs"
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
          "id": "05979c178033c40fe969f8d842cbc7a3d572b8f5",
          "message": "refactor(execution/grpc): move execution service where it belongs (#3302)\n\n* refactor(execution/grpc): move execution service where it belongs\n\n* reduce diff\n\n* fix lint",
          "timestamp": "2026-04-29T17:13:17+02:00",
          "tree_id": "78cf2234b6b93b8149dd51bbe9428301015dc40b",
          "url": "https://github.com/evstack/ev-node/commit/05979c178033c40fe969f8d842cbc7a3d572b8f5"
        },
        "date": 1777476028070,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 44829,
            "unit": "ns/op\t   10268 B/op\t      54 allocs/op",
            "extra": "27319 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 44829,
            "unit": "ns/op",
            "extra": "27319 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10268,
            "unit": "B/op",
            "extra": "27319 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "27319 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37697,
            "unit": "ns/op\t    4770 B/op\t      50 allocs/op",
            "extra": "32085 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37697,
            "unit": "ns/op",
            "extra": "32085 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4770,
            "unit": "B/op",
            "extra": "32085 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 50,
            "unit": "allocs/op",
            "extra": "32085 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 38327,
            "unit": "ns/op\t    4969 B/op\t      54 allocs/op",
            "extra": "31827 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 38327,
            "unit": "ns/op",
            "extra": "31827 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 4969,
            "unit": "B/op",
            "extra": "31827 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "31827 times\n4 procs"
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
          "id": "03d0d4d60960ffbdb25333e7e0c3826211765b18",
          "message": "feat(execution/grpc): adding support for grpc otlp (#3300)\n\n* feat: adding support for grpc oltp\n\n* chore: fix linting\n\n* cl\n\n---------\n\nCo-authored-by: Julien Robert <julien@rbrt.fr>",
          "timestamp": "2026-04-30T07:36:10Z",
          "tree_id": "be39434dcabf24176c31edb11905bb09c232f470",
          "url": "https://github.com/evstack/ev-node/commit/03d0d4d60960ffbdb25333e7e0c3826211765b18"
        },
        "date": 1777535708296,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 42959,
            "unit": "ns/op\t   10235 B/op\t      54 allocs/op",
            "extra": "28360 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 42959,
            "unit": "ns/op",
            "extra": "28360 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10235,
            "unit": "B/op",
            "extra": "28360 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "28360 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 36604,
            "unit": "ns/op\t    4751 B/op\t      50 allocs/op",
            "extra": "32893 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 36604,
            "unit": "ns/op",
            "extra": "32893 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4751,
            "unit": "B/op",
            "extra": "32893 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 50,
            "unit": "allocs/op",
            "extra": "32893 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 37183,
            "unit": "ns/op\t    4949 B/op\t      54 allocs/op",
            "extra": "32654 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 37183,
            "unit": "ns/op",
            "extra": "32654 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 4949,
            "unit": "B/op",
            "extra": "32654 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "32654 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "cricis@msn.com",
            "name": "criciss",
            "username": "criciss"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "f36fbd2001535adcaca881adb285d76bf1cbbd2b",
          "message": "chore: fix some minor issues in comments (#3304)",
          "timestamp": "2026-05-03T00:51:25+02:00",
          "tree_id": "a2556f1a189335253b7c9d4c5405f1e203516f7f",
          "url": "https://github.com/evstack/ev-node/commit/f36fbd2001535adcaca881adb285d76bf1cbbd2b"
        },
        "date": 1777762367377,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 36013,
            "unit": "ns/op\t    4730 B/op\t      50 allocs/op",
            "extra": "33825 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 36013,
            "unit": "ns/op",
            "extra": "33825 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4730,
            "unit": "B/op",
            "extra": "33825 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 50,
            "unit": "allocs/op",
            "extra": "33825 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 37359,
            "unit": "ns/op\t    4924 B/op\t      54 allocs/op",
            "extra": "33740 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 37359,
            "unit": "ns/op",
            "extra": "33740 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 4924,
            "unit": "B/op",
            "extra": "33740 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "33740 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 42580,
            "unit": "ns/op\t   10222 B/op\t      54 allocs/op",
            "extra": "28764 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 42580,
            "unit": "ns/op",
            "extra": "28764 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10222,
            "unit": "B/op",
            "extra": "28764 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "28764 times\n4 procs"
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
          "id": "264314f342dfaa8123a12b24ded8081df1197b63",
          "message": "build(deps): Bump dorny/paths-filter from 3 to 4 (#3308)\n\nBumps [dorny/paths-filter](https://github.com/dorny/paths-filter) from 3 to 4.\n- [Release notes](https://github.com/dorny/paths-filter/releases)\n- [Changelog](https://github.com/dorny/paths-filter/blob/master/CHANGELOG.md)\n- [Commits](https://github.com/dorny/paths-filter/compare/v3...v4)\n\n---\nupdated-dependencies:\n- dependency-name: dorny/paths-filter\n  dependency-version: '4'\n  dependency-type: direct:production\n  update-type: version-update:semver-major\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-05-06T09:56:35+02:00",
          "tree_id": "3df3233fc98f51d3a165b907d1bec30a94cf1415",
          "url": "https://github.com/evstack/ev-node/commit/264314f342dfaa8123a12b24ded8081df1197b63"
        },
        "date": 1778054421748,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 36663,
            "unit": "ns/op\t    4745 B/op\t      50 allocs/op",
            "extra": "33154 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 36663,
            "unit": "ns/op",
            "extra": "33154 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4745,
            "unit": "B/op",
            "extra": "33154 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 50,
            "unit": "allocs/op",
            "extra": "33154 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 37103,
            "unit": "ns/op\t    4949 B/op\t      54 allocs/op",
            "extra": "32644 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 37103,
            "unit": "ns/op",
            "extra": "32644 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 4949,
            "unit": "B/op",
            "extra": "32644 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "32644 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 43088,
            "unit": "ns/op\t   10244 B/op\t      54 allocs/op",
            "extra": "28062 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 43088,
            "unit": "ns/op",
            "extra": "28062 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10244,
            "unit": "B/op",
            "extra": "28062 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "28062 times\n4 procs"
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
          "id": "495c6215793c3efc3512ba8bde9cd4c17ebbf572",
          "message": "feat(pkg/sequencers): add queue limit in solo sequencer (#3312)\n\n* feat(pkg/sequencers): add queue limit in solo sequencer\n\n* use option\n\n* cl\n\n* move test files",
          "timestamp": "2026-05-06T16:25:26Z",
          "tree_id": "45f2f3e5c7b73d4a69a5cb9bf67ea602444bebfe",
          "url": "https://github.com/evstack/ev-node/commit/495c6215793c3efc3512ba8bde9cd4c17ebbf572"
        },
        "date": 1778085869331,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 38475,
            "unit": "ns/op\t    4787 B/op\t      50 allocs/op",
            "extra": "31422 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 38475,
            "unit": "ns/op",
            "extra": "31422 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4787,
            "unit": "B/op",
            "extra": "31422 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 50,
            "unit": "allocs/op",
            "extra": "31422 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 38804,
            "unit": "ns/op\t    4990 B/op\t      54 allocs/op",
            "extra": "31012 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 38804,
            "unit": "ns/op",
            "extra": "31012 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 4990,
            "unit": "B/op",
            "extra": "31012 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "31012 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 44836,
            "unit": "ns/op\t   10285 B/op\t      54 allocs/op",
            "extra": "26832 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 44836,
            "unit": "ns/op",
            "extra": "26832 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10285,
            "unit": "B/op",
            "extra": "26832 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "26832 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "caelrowley@pm.me",
            "name": "Cael Rowley",
            "username": "CaelRowley"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "01791ca1b4b528357b70896c296023fa5e621f5d",
          "message": "test(config): use `synctest` in TestAddFlags for avoiding nondeterminism  (#3311)\n\n* fix(config): de-flake TestAddFlags by stubbing time source\n\n* test(config): de-flake TestAddFlags via testing/synctest",
          "timestamp": "2026-05-07T11:57:50+02:00",
          "tree_id": "1a3f313873483be0c46eb038f1cb6b6a0f86199f",
          "url": "https://github.com/evstack/ev-node/commit/01791ca1b4b528357b70896c296023fa5e621f5d"
        },
        "date": 1778148158147,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37463,
            "unit": "ns/op\t    4756 B/op\t      50 allocs/op",
            "extra": "32679 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37463,
            "unit": "ns/op",
            "extra": "32679 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4756,
            "unit": "B/op",
            "extra": "32679 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 50,
            "unit": "allocs/op",
            "extra": "32679 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 37763,
            "unit": "ns/op\t    4957 B/op\t      54 allocs/op",
            "extra": "32304 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 37763,
            "unit": "ns/op",
            "extra": "32304 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 4957,
            "unit": "B/op",
            "extra": "32304 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "32304 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 44195,
            "unit": "ns/op\t   10257 B/op\t      54 allocs/op",
            "extra": "27669 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 44195,
            "unit": "ns/op",
            "extra": "27669 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10257,
            "unit": "B/op",
            "extra": "27669 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "27669 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "box4wangjing@outlook.com",
            "name": "box4wangjing",
            "username": "box4wangjing"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "45b5f3e8270be5cf6662cfec830bca5536422780",
          "message": "chore: remove extra space in comment (#3319)\n\nSigned-off-by: box4wangjing <box4wangjing@outlook.com>",
          "timestamp": "2026-05-11T09:20:11+02:00",
          "tree_id": "10c085511e072d8ae3cc93c93201714e508f5a8e",
          "url": "https://github.com/evstack/ev-node/commit/45b5f3e8270be5cf6662cfec830bca5536422780"
        },
        "date": 1778484245697,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 39191,
            "unit": "ns/op\t    4798 B/op\t      50 allocs/op",
            "extra": "30994 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 39191,
            "unit": "ns/op",
            "extra": "30994 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4798,
            "unit": "B/op",
            "extra": "30994 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 50,
            "unit": "allocs/op",
            "extra": "30994 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 39883,
            "unit": "ns/op\t    5005 B/op\t      54 allocs/op",
            "extra": "30435 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 39883,
            "unit": "ns/op",
            "extra": "30435 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 5005,
            "unit": "B/op",
            "extra": "30435 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "30435 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 45677,
            "unit": "ns/op\t   10298 B/op\t      54 allocs/op",
            "extra": "26484 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 45677,
            "unit": "ns/op",
            "extra": "26484 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10298,
            "unit": "B/op",
            "extra": "26484 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "26484 times\n4 procs"
          }
        ]
      }
    ]
  }
}