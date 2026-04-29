window.BENCHMARK_DATA = {
  "lastUpdate": 1777465804459,
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
      }
    ]
  }
}