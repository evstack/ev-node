window.BENCHMARK_DATA = {
  "lastUpdate": 1776788039282,
  "repoUrl": "https://github.com/evstack/ev-node",
  "entries": {
    "EVM Contract Roundtrip": [
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
          "id": "4c7323fb87cee63dd9b3abfa5e4533d4c95cc5ff",
          "message": "refactor(pkg/p2p): swap GossipSub by FloodSub (#3263)\n\n* refactor(pkg/p2p): swap GossipSub by FloodSub\n\n* docs + cl",
          "timestamp": "2026-04-17T09:44:58Z",
          "tree_id": "2913f0e3c36409f9c190274761b6768dbdfd371c",
          "url": "https://github.com/evstack/ev-node/commit/4c7323fb87cee63dd9b3abfa5e4533d4c95cc5ff"
        },
        "date": 1776420441726,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 907931794,
            "unit": "ns/op\t32002580 B/op\t  176600 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 907931794,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 32002580,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 176600,
            "unit": "allocs/op",
            "extra": "2 times\n4 procs"
          }
        ]
      },
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
          "id": "d9046e6706be9c4b1008be08c30d611e16c4388c",
          "message": "fix: raft HA production hardening — leader fencing, log compaction, election timeout, audit log (#3230)\n\n* Fix raft leader handoff regression after SIGTERM\n\n* fix: follower crash on restart when EVM is ahead of stale raft snapshot\n\nBug A: RecoverFromRaft crashed with \"invalid block height\" when the node\nrestarted after SIGTERM and the EVM state (persisted before kill) was ahead\nof the raft FSM snapshot (which hadn't finished log replay yet). The function\nnow verifies the hash of the local block at raftState.Height — if it matches\nthe snapshot hash the EVM history is correct and recovery is safely skipped;\na mismatch returns an error indicating a genuine fork.\n\nBug B: waitForMsgsLanded used two repeating tickers with the same effective\nperiod (SendTimeout/2 poll, SendTimeout timeout), so both could fire\nsimultaneously in select and the timeout would win even when AppliedIndex >=\nCommitIndex. Replaced the deadline ticker with a one-shot time.NewTimer,\nadded a final check in the deadline branch, and reduced poll interval to\nmin(50ms, timeout/4) for more responsive detection.\n\nFixes the crash-restart Docker backoff loop observed in SIGTERM HA test\ncycle 7 (poc-ha-2 never rejoining within the 300s kill interval).\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* fix(raft): guard FSM apply callback with RWMutex to prevent data race\n\nSetApplyCallback(nil) called from raftRetriever.Stop() raced with\nFSM.Apply reading applyCh: wg.Wait() only ensures the consumer goroutine\nexited, but the raft library can still invoke Apply concurrently.\n\nAdd applyMu sync.RWMutex to FSM; take write lock in SetApplyCallback and\nread lock in Apply before reading the channel pointer.\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* feat(raft): add ResignLeader() public method on Node\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* feat(node): implement LeaderResigner interface on FullNode\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* fix(shutdown): resign raft leadership before cancelling context on SIGTERM\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* feat(config): add election_timeout, snapshot_threshold, trailing_logs to RaftConfig; fix SnapCount default 0→3\n\nAdd three new Raft config parameters:\n  - ElectionTimeout: timeout for candidate to wait for votes (defaults to 1s)\n  - SnapshotThreshold: outstanding log entries that trigger snapshot (defaults to 500)\n  - TrailingLogs: log entries to retain after snapshot (defaults to 200)\n\nFix critical default: SnapCount was 0 (broken, retains no snapshots) → 3\n\nThis enables control over Raft's snapshot frequency and recovery behavior to prevent\nresync debt from accumulating unbounded during normal operation.\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* fix(raft): wire snapshot_threshold, trailing_logs, election_timeout into hashicorp/raft config\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* feat(raft): annotate FSM apply log and RaftApplyMsg with raft term for block provenance audit\n\nAdd Term field to RaftApplyMsg struct to track the raft term in which each\nblock was committed. Update FSM.Apply() debug logging to include both\nraft_term and raft_index fields alongside block height and hash. This\nenables better audit trails and debugging of replication issues.\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* fix(ci): fix gci comment alignment in defaults.go; remove boltdb-triggering tests\n\nThe gci formatter requires single space before inline comments (not aligned\ndouble-space). Also removed TestNodeResignLeader_NotLeaderNoop and\nTestNewNode_SnapshotConfigApplied which create real boltdb-backed raft nodes:\nboltdb@v1.3.1 has an unsafe pointer alignment issue that panics under\nGo 1.25's -checkptr. The nil-receiver test (TestNodeResignLeader_NilNoop)\nis retained as it exercises the same guard without touching boltdb.\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* fix(raft): suppress boltdb 'Rollback failed: tx closed' log noise\n\nhashicorp/raft-boltdb uses defer tx.Rollback() as a safety net on every\ntransaction. When Commit() succeeds, the deferred Rollback() returns\nbolt.ErrTxClosed and raft-boltdb logs it as an error — even though it is\nthe expected outcome of every successful read or write. The message has no\nactionable meaning and floods logs at high block rates.\n\nAdd a one-time stdlib log filter (sync.Once in NewNode) that silently drops\nlines containing 'tx closed' and forwards everything else to stderr.\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* fix(raft): address PR review — shutdown wiring, error logging, snap docs, tests\n\n- Call raftRetriever.Stop() in Syncer.Stop() so SetApplyCallback(nil) is\n  actually reached and the goroutine is awaited before wg.Wait()\n- Log leadershipTransfer error at warn level in Node.Stop() instead of\n  discarding it silently\n- Fix SnapCount comments in config.go: it retains snapshot files on disk\n  (NewFileSnapshotStore retain param), not log-entry frequency\n- Extract buildRaftConfig helper from NewNode to enable config wiring tests\n- Add TestNodeResignLeader_NotLeaderNoop (non-nil node, nil raft → noop)\n- Add TestNewNode_SnapshotConfigApplied (table-driven, verifies\n  SnapshotThreshold and TrailingLogs wiring with custom and zero values)\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* fix(raft): address code review issues — ShutdownTimeout, resign fence, election validation\n\n- Add ShutdownTimeout field (default 5s) to raft Config so Stop() drains\n  committed logs with a meaningful timeout instead of the 200ms SendTimeout\n- Wrap ResignLeader() in a 3s goroutine+select fence in the SIGTERM handler\n  so a hung leadership transfer cannot block graceful shutdown indefinitely\n- Validate ElectionTimeout >= HeartbeatTimeout in RaftConfig.Validate() to\n  prevent hashicorp/raft panicking at startup with an invalid config\n- Fix stale \"NewNode creates a new raft node\" comment that had migrated onto\n  buildRaftConfig after the function was extracted\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* style(raft): fix gci struct field alignment in node_test.go\n\ngofmt/gci requires minimal alignment; excessive spaces in the\nTestNewNode_SnapshotConfigApplied struct literal caused a lint failure.\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* test: improve patch coverage for raft shutdown and resign paths\n\nAdd unit tests for lines flagged by Codecov:\n- boltTxClosedFilter.Write: filter drops \"tx closed\", forwards others\n- buildRaftConfig: ElectionTimeout > 0 applied, zero uses default\n- FullNode.ResignLeader: nil raftNode no-op; non-leader raftNode no-op\n- Syncer.Stop: raftRetriever.Stop is called when raftRetriever is set\n- Syncer.RecoverFromRaft: GetHeader failure when local state is ahead of\n  stale raft snapshot returns \"cannot verify hash\" error\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* fix(config): reject negative ElectionTimeout in RaftConfig.Validate\n\nA negative ElectionTimeout was silently ignored (buildRaftConfig only\napplies values > 0), allowing a misconfigured node to start with the\nlibrary default instead of failing fast.  Add an explicit < 0 check\nthat returns an error; 0 remains valid as the \"use library default\"\nsentinel.\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* fix(raft): preserve stdlib logger writer in bolt filter; propagate ctx through ResignLeader\n\n- suppressBoltNoise.Do now wraps log.Writer() instead of os.Stderr so any\n  existing stdlib logger redirection is preserved rather than clobbered\n- ResignLeader now accepts a context.Context: leadershipTransfer runs in a\n  goroutine and a select abandons the caller at ctx.Done(), returning\n  ctx.Err(); the goroutine itself exits once the inner raft transfer\n  completes (bounded by ElectionTimeout)\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* fix(node): propagate context through LeaderResigner.ResignLeader interface\n\n- LeaderResigner.ResignLeader() → ResignLeader(ctx context.Context) error\n- FullNode.ResignLeader passes ctx down to raft.Node.ResignLeader\n- run_node.go calls resigner.ResignLeader(resignCtx) directly — no wrapper\n  goroutine/select needed; context.DeadlineExceeded vs other errors are\n  logged distinctly\n- Merge TestFullNode_ResignLeader_NilRaftNode and\n  TestFullNode_ResignLeader_NonLeaderRaftNode into single table-driven test\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* fix(raft): abdicate leadership when store is significantly behind raft state\n\nWhen a node wins election but its local store is more than 1 block behind\nthe raft FSM state, RecoverFromRaft cannot replay the gap (it only handles\nthe single latest block in the raft snapshot). Previously the node would\nlog \"recovery successful\" and start leader operations anyway, then stall\nblock production while the executor repeatedly failed to sync the missing\nblocks from a store that did not have them.\n\nFix: in DynamicLeaderElection.Run, detect diff < -1 at the\nfollower→leader transition and immediately transfer leadership to a\nbetter-synced peer. diff == -1 is preserved: RecoverFromRaft can apply\nexactly one block from the raft snapshot, so that path is unchanged.\n\nCloses #3255\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* fix(raft): address julienrbrt review — logger, boltdb filter, ShutdownTimeout\n\n- Remove stdlib log filter (boltTxClosedFilter / suppressBoltNoise): it\n  redirected the global stdlib logger which is the wrong scope. raft-boltdb\n  uses log.Printf directly with no Logger option, so the \"tx closed\" noise\n  is now accepted as-is from stderr.\n\n- Wire hashicorp/raft's internal hclog output through zerolog: set\n  raft.Config.Logger to an hclog.Logger backed by the zerolog writer so\n  all raft-internal messages appear in the structured log stream under\n  component=raft-hashicorp.\n\n- Remove ShutdownTimeout from public config: it was a second \"how long to\n  wait\" knob that confused operators. ShutdownTimeout is now computed\n  internally as 5 × SendTimeout at the initRaftNode call site.\n\n- Delete TestRaftRetrieverStopClearsApplyCallback: tested an implementation\n  detail (Stop clears the apply callback pointer) rather than observable\n  behaviour. The stubRaftNode helper it defined is moved to syncer_test.go\n  where it is still needed.\n\n- Rename TestNewNode_SnapshotConfigApplied → TestBuildRaftConfig_SnapshotConfigApplied\n  to reflect that it tests buildRaftConfig, not NewNode.\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* fix(ci): promote go-hclog to direct dep; fix gci alignment in syncer_test\n\ngo mod tidy promotes github.com/hashicorp/go-hclog from indirect to\ndirect now that pkg/raft/node.go imports it explicitly.\n\ngci auto-formatted stubRaftNode method stubs in syncer_test.go.\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* fix(raft): address coderabbitai feedback — ShutdownTimeout clamp, transfer error propagation, deterministic test\n\nShutdownTimeout zero-value panic (critical):\nNewNode now clamps ShutdownTimeout to 5*SendTimeout when the caller passes\nzero, preventing a panic in time.NewTicker inside waitForMsgsLanded. The\nnormal path through initRaftNode already sets it explicitly; this guard\nprotects direct callers (e.g. tests) that omit the field.\n\nLeadership transfer error propagation (major):\nWhen store-lag abdication calls leadershipTransfer() and it fails, the\nerror is now returned instead of being logged and silently continuing.\nContinuing after a failed transfer left the node as leader-without-worker,\nstalling the cluster.\n\nDeterministic abdication test (major):\nReplace time.Sleep(10ms) + t.Fatal-in-goroutine with channel-based\nsynchronization: leader runFn signals leaderStarted; the test goroutine\nwaits up to 50ms for that signal and calls t.Error (safe from goroutines)\nif it arrives, then cancels the context either way.\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* docs(changelog): add unreleased entries for raft HA hardening (#3230)\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* fix(raft): wait for block-store sync before abdicating on leader election\n\nWhen all nodes restart simultaneously their block stores can lag behind\nthe raft FSM height (block data arrives via p2p, not raft). With the\nprevious code every elected node saw diff < -1 and immediately called\nleadershipTransfer(), creating an infinite hot-potato: no node ever\nstabilised as leader and block production stalled.\n\nInstead of abdicating immediately, the new waitForBlockStoreSync helper\npolls IsSynced for up to ShutdownTimeout (default ~1s). The fastest-\nsyncing peer proceeds as leader; nodes that cannot catch up in time still\nabdicate and yield to a better candidate. Leadership also checks mid-wait\nso a lost-leadership event aborts the wait early.\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* fix(raft): distinguish sync wait outcomes with syncResult enum\n\nwaitForBlockStoreSync previously returned bool, conflating three distinct\nfailure modes (ctx canceled, timeout, lost leadership). The caller in Run\nthen unconditionally called leadershipTransfer() on any false return, which\nis wrong when leadership was already lost.\n\nIntroduce a syncResult enum (syncResultSynced, syncResultTimeout,\nsyncResultLostLeadership, syncResultCanceled) and update Run to handle each\ncase correctly:\n- syncResultCanceled    → return ctx.Err()\n- syncResultLostLeadership → continue without calling leadershipTransfer()\n- syncResultTimeout     → leadershipTransfer() + continue as before\n- syncResultSynced      → refresh raftState/diff and proceed\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* fix(raft): fix gci alignment in syncResult const block\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n---------\n\nCo-authored-by: Claude Sonnet 4.6 <noreply@anthropic.com>",
          "timestamp": "2026-04-20T09:34:03Z",
          "tree_id": "77955a3ad0f421db5db82e84dbae67fb0a0ae959",
          "url": "https://github.com/evstack/ev-node/commit/d9046e6706be9c4b1008be08c30d611e16c4388c"
        },
        "date": 1776678832919,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 908787609,
            "unit": "ns/op\t32272204 B/op\t  180764 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 908787609,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 32272204,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 180764,
            "unit": "allocs/op",
            "extra": "2 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "raddole@ukr.net",
            "name": "Radovenchyk",
            "username": "Radovenchyk"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "2eb15bd5ecb85ddb548b10f944d70361d2ec4ae8",
          "message": "test: remove racy node startup from execution test (#3264)\n\nfix(test): remove racy node startup from execution test",
          "timestamp": "2026-04-20T11:19:34Z",
          "tree_id": "2efb7427eb4e3b35d45fd5d0b995ba0f9c9f64cf",
          "url": "https://github.com/evstack/ev-node/commit/2eb15bd5ecb85ddb548b10f944d70361d2ec4ae8"
        },
        "date": 1776685159771,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 910652046,
            "unit": "ns/op\t32446756 B/op\t  181062 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 910652046,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 32446756,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 181062,
            "unit": "allocs/op",
            "extra": "2 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "chuanshanjida@outlook.com",
            "name": "chuanshanjida",
            "username": "chuanshanjida"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "83edf6d2355cef0033d86f44bd7359d6e0775137",
          "message": "refactor: modernize sync/atomic usage with typed atomic apis (#3269)\n\nSigned-off-by: chuanshanjida <chuanshanjida@outlook.com>",
          "timestamp": "2026-04-21T07:33:42Z",
          "tree_id": "631f0fcdf2be59af0fe6650d9c4a8c0e41ae8e51",
          "url": "https://github.com/evstack/ev-node/commit/83edf6d2355cef0033d86f44bd7359d6e0775137"
        },
        "date": 1776758016111,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 857506350,
            "unit": "ns/op\t27563032 B/op\t  140006 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 857506350,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 27563032,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 140006,
            "unit": "allocs/op",
            "extra": "2 times\n4 procs"
          }
        ]
      },
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
          "distinct": false,
          "id": "c753c0bb2193faa5bd0b713bb7787757221c59de",
          "message": "fix(rpc): derive /raft/node leadership from raft leader ID (#3266)\n\n* fix(rpc): derive /raft/node leadership from raft leader ID\n\n* fix(raft): add LeaderID doc comment; consolidate raft-status tests\n\n- Add Go doc comment to Node.LeaderID() describing return value, nil-safety,\n  and staleness semantics, consistent with IsLeader/HasQuorum style.\n- Consolidate three near-duplicate TestRegisterCustomHTTPEndpoints_RaftNodeStatus*\n  tests into a single table-driven test covering: leaderID==nodeID (is_leader\n  true), leaderID!=nodeID (is_leader false), empty leaderID fallback, and\n  non-GET method (405). Clarifies that is_leader is derived from LeaderID(),\n  not the IsLeader() field on testRaftNodeSource.\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* fix(rpc): fix gci import grouping in http_test.go\n\nSeparate third-party imports from evstack/ev-node-prefixed imports\nper project gci custom-order config.\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n---------\n\nCo-authored-by: Claude Sonnet 4.6 <noreply@anthropic.com>",
          "timestamp": "2026-04-21T09:58:26Z",
          "tree_id": "12fe1920fc341aee1aa3071ced43d8f90ee95838",
          "url": "https://github.com/evstack/ev-node/commit/c753c0bb2193faa5bd0b713bb7787757221c59de"
        },
        "date": 1776766661769,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 913180694,
            "unit": "ns/op\t33180400 B/op\t  186529 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 913180694,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 33180400,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 186529,
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
          "id": "c1d4996ca8e76b459aa9a6eb5e5a955013303607",
          "message": "build(deps): Bump the all-go group across 5 directories with 14 updates (#3271)\n\n* build(deps): Bump the all-go group across 5 directories with 14 updates\n\nBumps the all-go group with 9 updates in the / directory:\n\n| Package | From | To |\n| --- | --- | --- |\n| [cloud.google.com/go/kms](https://github.com/googleapis/google-cloud-go) | `1.27.0` | `1.29.0` |\n| [connectrpc.com/connect](https://github.com/connectrpc/connect-go) | `1.19.1` | `1.19.2` |\n| [github.com/aws/aws-sdk-go-v2](https://github.com/aws/aws-sdk-go-v2) | `1.41.5` | `1.41.6` |\n| [github.com/aws/aws-sdk-go-v2/config](https://github.com/aws/aws-sdk-go-v2) | `1.32.14` | `1.32.16` |\n| [github.com/aws/aws-sdk-go-v2/service/kms](https://github.com/aws/aws-sdk-go-v2) | `1.50.4` | `1.50.5` |\n| [github.com/celestiaorg/nmt](https://github.com/celestiaorg/nmt) | `0.24.2` | `0.24.3` |\n| [github.com/libp2p/go-libp2p-kad-dht](https://github.com/libp2p/go-libp2p-kad-dht) | `0.39.0` | `0.39.1` |\n| [golang.org/x/crypto](https://github.com/golang/crypto) | `0.49.0` | `0.50.0` |\n| [golang.org/x/net](https://github.com/golang/net) | `0.52.0` | `0.53.0` |\n\nBumps the all-go group with 1 update in the /execution/evm directory: [github.com/evstack/ev-node](https://github.com/evstack/ev-node).\nBumps the all-go group with 3 updates in the /execution/grpc directory: [connectrpc.com/connect](https://github.com/connectrpc/connect-go), [golang.org/x/net](https://github.com/golang/net) and [github.com/evstack/ev-node](https://github.com/evstack/ev-node).\nBumps the all-go group with 2 updates in the /test/docker-e2e directory: [github.com/celestiaorg/tastora](https://github.com/celestiaorg/tastora) and [github.com/evstack/ev-node/execution/evm](https://github.com/evstack/ev-node).\nBumps the all-go group with 1 update in the /test/e2e directory: [github.com/celestiaorg/tastora](https://github.com/celestiaorg/tastora).\n\n\nUpdates `cloud.google.com/go/kms` from 1.27.0 to 1.29.0\n- [Release notes](https://github.com/googleapis/google-cloud-go/releases)\n- [Changelog](https://github.com/googleapis/google-cloud-go/blob/main/documentai/CHANGES.md)\n- [Commits](https://github.com/googleapis/google-cloud-go/compare/kms/v1.27.0...dlp/v1.29.0)\n\nUpdates `connectrpc.com/connect` from 1.19.1 to 1.19.2\n- [Release notes](https://github.com/connectrpc/connect-go/releases)\n- [Changelog](https://github.com/connectrpc/connect-go/blob/main/RELEASE.md)\n- [Commits](https://github.com/connectrpc/connect-go/compare/v1.19.1...v1.19.2)\n\nUpdates `github.com/aws/aws-sdk-go-v2` from 1.41.5 to 1.41.6\n- [Release notes](https://github.com/aws/aws-sdk-go-v2/releases)\n- [Commits](https://github.com/aws/aws-sdk-go-v2/compare/v1.41.5...v1.41.6)\n\nUpdates `github.com/aws/aws-sdk-go-v2/config` from 1.32.14 to 1.32.16\n- [Release notes](https://github.com/aws/aws-sdk-go-v2/releases)\n- [Commits](https://github.com/aws/aws-sdk-go-v2/compare/config/v1.32.14...config/v1.32.16)\n\nUpdates `github.com/aws/aws-sdk-go-v2/service/kms` from 1.50.4 to 1.50.5\n- [Release notes](https://github.com/aws/aws-sdk-go-v2/releases)\n- [Commits](https://github.com/aws/aws-sdk-go-v2/compare/service/ssm/v1.50.4...service/ssm/v1.50.5)\n\nUpdates `github.com/aws/smithy-go` from 1.24.3 to 1.25.0\n- [Release notes](https://github.com/aws/smithy-go/releases)\n- [Changelog](https://github.com/aws/smithy-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/aws/smithy-go/compare/v1.24.3...v1.25.0)\n\nUpdates `github.com/celestiaorg/nmt` from 0.24.2 to 0.24.3\n- [Release notes](https://github.com/celestiaorg/nmt/releases)\n- [Commits](https://github.com/celestiaorg/nmt/compare/v0.24.2...v0.24.3)\n\nUpdates `github.com/libp2p/go-libp2p-kad-dht` from 0.39.0 to 0.39.1\n- [Release notes](https://github.com/libp2p/go-libp2p-kad-dht/releases)\n- [Commits](https://github.com/libp2p/go-libp2p-kad-dht/compare/v0.39.0...v0.39.1)\n\nUpdates `golang.org/x/crypto` from 0.49.0 to 0.50.0\n- [Commits](https://github.com/golang/crypto/compare/v0.49.0...v0.50.0)\n\nUpdates `golang.org/x/net` from 0.52.0 to 0.53.0\n- [Commits](https://github.com/golang/net/compare/v0.52.0...v0.53.0)\n\nUpdates `google.golang.org/api` from 0.273.1 to 0.274.0\n- [Release notes](https://github.com/googleapis/google-api-go-client/releases)\n- [Changelog](https://github.com/googleapis/google-api-go-client/blob/main/CHANGES.md)\n- [Commits](https://github.com/googleapis/google-api-go-client/compare/v0.273.1...v0.274.0)\n\nUpdates `github.com/evstack/ev-node` from 1.0.0 to 1.1.0\n- [Release notes](https://github.com/evstack/ev-node/releases)\n- [Changelog](https://github.com/evstack/ev-node/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/evstack/ev-node/compare/v1.0.0...v1.1.0)\n\nUpdates `connectrpc.com/connect` from 1.19.1 to 1.19.2\n- [Release notes](https://github.com/connectrpc/connect-go/releases)\n- [Changelog](https://github.com/connectrpc/connect-go/blob/main/RELEASE.md)\n- [Commits](https://github.com/connectrpc/connect-go/compare/v1.19.1...v1.19.2)\n\nUpdates `golang.org/x/net` from 0.52.0 to 0.53.0\n- [Commits](https://github.com/golang/net/compare/v0.52.0...v0.53.0)\n\nUpdates `github.com/evstack/ev-node` from 1.0.0 to 1.1.0\n- [Release notes](https://github.com/evstack/ev-node/releases)\n- [Changelog](https://github.com/evstack/ev-node/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/evstack/ev-node/compare/v1.0.0...v1.1.0)\n\nUpdates `github.com/celestiaorg/tastora` from 0.17.0 to 0.19.0\n- [Release notes](https://github.com/celestiaorg/tastora/releases)\n- [Commits](https://github.com/celestiaorg/tastora/compare/v0.17.0...v0.19.0)\n\nUpdates `github.com/evstack/ev-node/execution/evm` from 1.0.0 to 1.0.1\n- [Release notes](https://github.com/evstack/ev-node/releases)\n- [Changelog](https://github.com/evstack/ev-node/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/evstack/ev-node/compare/v1.0.0...execution/evm/v1.0.1)\n\nUpdates `github.com/celestiaorg/tastora` from 0.16.1-0.20260312082036-2ee1b0a2ac4e to 0.19.0\n- [Release notes](https://github.com/celestiaorg/tastora/releases)\n- [Commits](https://github.com/celestiaorg/tastora/compare/v0.17.0...v0.19.0)\n\n---\nupdated-dependencies:\n- dependency-name: cloud.google.com/go/kms\n  dependency-version: 1.29.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: connectrpc.com/connect\n  dependency-version: 1.19.2\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/aws/aws-sdk-go-v2\n  dependency-version: 1.41.6\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/aws/aws-sdk-go-v2/config\n  dependency-version: 1.32.16\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/aws/aws-sdk-go-v2/service/kms\n  dependency-version: 1.50.5\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/aws/smithy-go\n  dependency-version: 1.25.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/celestiaorg/nmt\n  dependency-version: 0.24.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/libp2p/go-libp2p-kad-dht\n  dependency-version: 0.39.1\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: golang.org/x/crypto\n  dependency-version: 0.50.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: golang.org/x/net\n  dependency-version: 0.53.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: google.golang.org/api\n  dependency-version: 0.274.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/evstack/ev-node\n  dependency-version: 1.1.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: connectrpc.com/connect\n  dependency-version: 1.19.2\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: golang.org/x/net\n  dependency-version: 0.53.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/evstack/ev-node\n  dependency-version: 1.1.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/celestiaorg/tastora\n  dependency-version: 0.19.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/evstack/ev-node/execution/evm\n  dependency-version: 1.0.1\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/celestiaorg/tastora\n  dependency-version: 0.19.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\n\n* downgrade tastora\n\n---------\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>\nCo-authored-by: Julien Robert <julien@rbrt.fr>",
          "timestamp": "2026-04-21T12:45:24Z",
          "tree_id": "af1e4168064c00d22c4221b5c0afb5a0db4b55db",
          "url": "https://github.com/evstack/ev-node/commit/c1d4996ca8e76b459aa9a6eb5e5a955013303607"
        },
        "date": 1776776891788,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 919117987,
            "unit": "ns/op\t32057844 B/op\t  176476 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 919117987,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 32057844,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 176476,
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
          "id": "cd76eab72c0d432b031101253508b0b9549e95d0",
          "message": "build(deps): Bump rustls-webpki from 0.103.10 to 0.103.13 in the cargo group across 1 directory (#3272)\n\nbuild(deps): Bump rustls-webpki in the cargo group across 1 directory\n\nBumps the cargo group with 1 update in the / directory: [rustls-webpki](https://github.com/rustls/webpki).\n\n\nUpdates `rustls-webpki` from 0.103.10 to 0.103.13\n- [Release notes](https://github.com/rustls/webpki/releases)\n- [Commits](https://github.com/rustls/webpki/compare/v/0.103.10...v/0.103.13)\n\n---\nupdated-dependencies:\n- dependency-name: rustls-webpki\n  dependency-version: 0.103.13\n  dependency-type: indirect\n  dependency-group: cargo\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-04-21T18:09:48+02:00",
          "tree_id": "eae97f0f87e36c0e0b78e358fc3bae96d818dc55",
          "url": "https://github.com/evstack/ev-node/commit/cd76eab72c0d432b031101253508b0b9549e95d0"
        },
        "date": 1776788034116,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 921215948,
            "unit": "ns/op\t33446980 B/op\t  190880 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 921215948,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 33446980,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 190880,
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
          "id": "4c7323fb87cee63dd9b3abfa5e4533d4c95cc5ff",
          "message": "refactor(pkg/p2p): swap GossipSub by FloodSub (#3263)\n\n* refactor(pkg/p2p): swap GossipSub by FloodSub\n\n* docs + cl",
          "timestamp": "2026-04-17T09:44:58Z",
          "tree_id": "2913f0e3c36409f9c190274761b6768dbdfd371c",
          "url": "https://github.com/evstack/ev-node/commit/4c7323fb87cee63dd9b3abfa5e4533d4c95cc5ff"
        },
        "date": 1776420446711,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 36916,
            "unit": "ns/op\t    4754 B/op\t      50 allocs/op",
            "extra": "32746 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 36916,
            "unit": "ns/op",
            "extra": "32746 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4754,
            "unit": "B/op",
            "extra": "32746 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 50,
            "unit": "allocs/op",
            "extra": "32746 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 37383,
            "unit": "ns/op\t    4954 B/op\t      54 allocs/op",
            "extra": "32443 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 37383,
            "unit": "ns/op",
            "extra": "32443 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 4954,
            "unit": "B/op",
            "extra": "32443 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "32443 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 43889,
            "unit": "ns/op\t   10246 B/op\t      54 allocs/op",
            "extra": "27990 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 43889,
            "unit": "ns/op",
            "extra": "27990 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10246,
            "unit": "B/op",
            "extra": "27990 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "27990 times\n4 procs"
          }
        ]
      },
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
          "id": "d9046e6706be9c4b1008be08c30d611e16c4388c",
          "message": "fix: raft HA production hardening — leader fencing, log compaction, election timeout, audit log (#3230)\n\n* Fix raft leader handoff regression after SIGTERM\n\n* fix: follower crash on restart when EVM is ahead of stale raft snapshot\n\nBug A: RecoverFromRaft crashed with \"invalid block height\" when the node\nrestarted after SIGTERM and the EVM state (persisted before kill) was ahead\nof the raft FSM snapshot (which hadn't finished log replay yet). The function\nnow verifies the hash of the local block at raftState.Height — if it matches\nthe snapshot hash the EVM history is correct and recovery is safely skipped;\na mismatch returns an error indicating a genuine fork.\n\nBug B: waitForMsgsLanded used two repeating tickers with the same effective\nperiod (SendTimeout/2 poll, SendTimeout timeout), so both could fire\nsimultaneously in select and the timeout would win even when AppliedIndex >=\nCommitIndex. Replaced the deadline ticker with a one-shot time.NewTimer,\nadded a final check in the deadline branch, and reduced poll interval to\nmin(50ms, timeout/4) for more responsive detection.\n\nFixes the crash-restart Docker backoff loop observed in SIGTERM HA test\ncycle 7 (poc-ha-2 never rejoining within the 300s kill interval).\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* fix(raft): guard FSM apply callback with RWMutex to prevent data race\n\nSetApplyCallback(nil) called from raftRetriever.Stop() raced with\nFSM.Apply reading applyCh: wg.Wait() only ensures the consumer goroutine\nexited, but the raft library can still invoke Apply concurrently.\n\nAdd applyMu sync.RWMutex to FSM; take write lock in SetApplyCallback and\nread lock in Apply before reading the channel pointer.\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* feat(raft): add ResignLeader() public method on Node\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* feat(node): implement LeaderResigner interface on FullNode\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* fix(shutdown): resign raft leadership before cancelling context on SIGTERM\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* feat(config): add election_timeout, snapshot_threshold, trailing_logs to RaftConfig; fix SnapCount default 0→3\n\nAdd three new Raft config parameters:\n  - ElectionTimeout: timeout for candidate to wait for votes (defaults to 1s)\n  - SnapshotThreshold: outstanding log entries that trigger snapshot (defaults to 500)\n  - TrailingLogs: log entries to retain after snapshot (defaults to 200)\n\nFix critical default: SnapCount was 0 (broken, retains no snapshots) → 3\n\nThis enables control over Raft's snapshot frequency and recovery behavior to prevent\nresync debt from accumulating unbounded during normal operation.\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* fix(raft): wire snapshot_threshold, trailing_logs, election_timeout into hashicorp/raft config\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* feat(raft): annotate FSM apply log and RaftApplyMsg with raft term for block provenance audit\n\nAdd Term field to RaftApplyMsg struct to track the raft term in which each\nblock was committed. Update FSM.Apply() debug logging to include both\nraft_term and raft_index fields alongside block height and hash. This\nenables better audit trails and debugging of replication issues.\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* fix(ci): fix gci comment alignment in defaults.go; remove boltdb-triggering tests\n\nThe gci formatter requires single space before inline comments (not aligned\ndouble-space). Also removed TestNodeResignLeader_NotLeaderNoop and\nTestNewNode_SnapshotConfigApplied which create real boltdb-backed raft nodes:\nboltdb@v1.3.1 has an unsafe pointer alignment issue that panics under\nGo 1.25's -checkptr. The nil-receiver test (TestNodeResignLeader_NilNoop)\nis retained as it exercises the same guard without touching boltdb.\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* fix(raft): suppress boltdb 'Rollback failed: tx closed' log noise\n\nhashicorp/raft-boltdb uses defer tx.Rollback() as a safety net on every\ntransaction. When Commit() succeeds, the deferred Rollback() returns\nbolt.ErrTxClosed and raft-boltdb logs it as an error — even though it is\nthe expected outcome of every successful read or write. The message has no\nactionable meaning and floods logs at high block rates.\n\nAdd a one-time stdlib log filter (sync.Once in NewNode) that silently drops\nlines containing 'tx closed' and forwards everything else to stderr.\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* fix(raft): address PR review — shutdown wiring, error logging, snap docs, tests\n\n- Call raftRetriever.Stop() in Syncer.Stop() so SetApplyCallback(nil) is\n  actually reached and the goroutine is awaited before wg.Wait()\n- Log leadershipTransfer error at warn level in Node.Stop() instead of\n  discarding it silently\n- Fix SnapCount comments in config.go: it retains snapshot files on disk\n  (NewFileSnapshotStore retain param), not log-entry frequency\n- Extract buildRaftConfig helper from NewNode to enable config wiring tests\n- Add TestNodeResignLeader_NotLeaderNoop (non-nil node, nil raft → noop)\n- Add TestNewNode_SnapshotConfigApplied (table-driven, verifies\n  SnapshotThreshold and TrailingLogs wiring with custom and zero values)\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* fix(raft): address code review issues — ShutdownTimeout, resign fence, election validation\n\n- Add ShutdownTimeout field (default 5s) to raft Config so Stop() drains\n  committed logs with a meaningful timeout instead of the 200ms SendTimeout\n- Wrap ResignLeader() in a 3s goroutine+select fence in the SIGTERM handler\n  so a hung leadership transfer cannot block graceful shutdown indefinitely\n- Validate ElectionTimeout >= HeartbeatTimeout in RaftConfig.Validate() to\n  prevent hashicorp/raft panicking at startup with an invalid config\n- Fix stale \"NewNode creates a new raft node\" comment that had migrated onto\n  buildRaftConfig after the function was extracted\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* style(raft): fix gci struct field alignment in node_test.go\n\ngofmt/gci requires minimal alignment; excessive spaces in the\nTestNewNode_SnapshotConfigApplied struct literal caused a lint failure.\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* test: improve patch coverage for raft shutdown and resign paths\n\nAdd unit tests for lines flagged by Codecov:\n- boltTxClosedFilter.Write: filter drops \"tx closed\", forwards others\n- buildRaftConfig: ElectionTimeout > 0 applied, zero uses default\n- FullNode.ResignLeader: nil raftNode no-op; non-leader raftNode no-op\n- Syncer.Stop: raftRetriever.Stop is called when raftRetriever is set\n- Syncer.RecoverFromRaft: GetHeader failure when local state is ahead of\n  stale raft snapshot returns \"cannot verify hash\" error\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* fix(config): reject negative ElectionTimeout in RaftConfig.Validate\n\nA negative ElectionTimeout was silently ignored (buildRaftConfig only\napplies values > 0), allowing a misconfigured node to start with the\nlibrary default instead of failing fast.  Add an explicit < 0 check\nthat returns an error; 0 remains valid as the \"use library default\"\nsentinel.\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* fix(raft): preserve stdlib logger writer in bolt filter; propagate ctx through ResignLeader\n\n- suppressBoltNoise.Do now wraps log.Writer() instead of os.Stderr so any\n  existing stdlib logger redirection is preserved rather than clobbered\n- ResignLeader now accepts a context.Context: leadershipTransfer runs in a\n  goroutine and a select abandons the caller at ctx.Done(), returning\n  ctx.Err(); the goroutine itself exits once the inner raft transfer\n  completes (bounded by ElectionTimeout)\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* fix(node): propagate context through LeaderResigner.ResignLeader interface\n\n- LeaderResigner.ResignLeader() → ResignLeader(ctx context.Context) error\n- FullNode.ResignLeader passes ctx down to raft.Node.ResignLeader\n- run_node.go calls resigner.ResignLeader(resignCtx) directly — no wrapper\n  goroutine/select needed; context.DeadlineExceeded vs other errors are\n  logged distinctly\n- Merge TestFullNode_ResignLeader_NilRaftNode and\n  TestFullNode_ResignLeader_NonLeaderRaftNode into single table-driven test\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* fix(raft): abdicate leadership when store is significantly behind raft state\n\nWhen a node wins election but its local store is more than 1 block behind\nthe raft FSM state, RecoverFromRaft cannot replay the gap (it only handles\nthe single latest block in the raft snapshot). Previously the node would\nlog \"recovery successful\" and start leader operations anyway, then stall\nblock production while the executor repeatedly failed to sync the missing\nblocks from a store that did not have them.\n\nFix: in DynamicLeaderElection.Run, detect diff < -1 at the\nfollower→leader transition and immediately transfer leadership to a\nbetter-synced peer. diff == -1 is preserved: RecoverFromRaft can apply\nexactly one block from the raft snapshot, so that path is unchanged.\n\nCloses #3255\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* fix(raft): address julienrbrt review — logger, boltdb filter, ShutdownTimeout\n\n- Remove stdlib log filter (boltTxClosedFilter / suppressBoltNoise): it\n  redirected the global stdlib logger which is the wrong scope. raft-boltdb\n  uses log.Printf directly with no Logger option, so the \"tx closed\" noise\n  is now accepted as-is from stderr.\n\n- Wire hashicorp/raft's internal hclog output through zerolog: set\n  raft.Config.Logger to an hclog.Logger backed by the zerolog writer so\n  all raft-internal messages appear in the structured log stream under\n  component=raft-hashicorp.\n\n- Remove ShutdownTimeout from public config: it was a second \"how long to\n  wait\" knob that confused operators. ShutdownTimeout is now computed\n  internally as 5 × SendTimeout at the initRaftNode call site.\n\n- Delete TestRaftRetrieverStopClearsApplyCallback: tested an implementation\n  detail (Stop clears the apply callback pointer) rather than observable\n  behaviour. The stubRaftNode helper it defined is moved to syncer_test.go\n  where it is still needed.\n\n- Rename TestNewNode_SnapshotConfigApplied → TestBuildRaftConfig_SnapshotConfigApplied\n  to reflect that it tests buildRaftConfig, not NewNode.\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* fix(ci): promote go-hclog to direct dep; fix gci alignment in syncer_test\n\ngo mod tidy promotes github.com/hashicorp/go-hclog from indirect to\ndirect now that pkg/raft/node.go imports it explicitly.\n\ngci auto-formatted stubRaftNode method stubs in syncer_test.go.\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* fix(raft): address coderabbitai feedback — ShutdownTimeout clamp, transfer error propagation, deterministic test\n\nShutdownTimeout zero-value panic (critical):\nNewNode now clamps ShutdownTimeout to 5*SendTimeout when the caller passes\nzero, preventing a panic in time.NewTicker inside waitForMsgsLanded. The\nnormal path through initRaftNode already sets it explicitly; this guard\nprotects direct callers (e.g. tests) that omit the field.\n\nLeadership transfer error propagation (major):\nWhen store-lag abdication calls leadershipTransfer() and it fails, the\nerror is now returned instead of being logged and silently continuing.\nContinuing after a failed transfer left the node as leader-without-worker,\nstalling the cluster.\n\nDeterministic abdication test (major):\nReplace time.Sleep(10ms) + t.Fatal-in-goroutine with channel-based\nsynchronization: leader runFn signals leaderStarted; the test goroutine\nwaits up to 50ms for that signal and calls t.Error (safe from goroutines)\nif it arrives, then cancels the context either way.\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* docs(changelog): add unreleased entries for raft HA hardening (#3230)\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* fix(raft): wait for block-store sync before abdicating on leader election\n\nWhen all nodes restart simultaneously their block stores can lag behind\nthe raft FSM height (block data arrives via p2p, not raft). With the\nprevious code every elected node saw diff < -1 and immediately called\nleadershipTransfer(), creating an infinite hot-potato: no node ever\nstabilised as leader and block production stalled.\n\nInstead of abdicating immediately, the new waitForBlockStoreSync helper\npolls IsSynced for up to ShutdownTimeout (default ~1s). The fastest-\nsyncing peer proceeds as leader; nodes that cannot catch up in time still\nabdicate and yield to a better candidate. Leadership also checks mid-wait\nso a lost-leadership event aborts the wait early.\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* fix(raft): distinguish sync wait outcomes with syncResult enum\n\nwaitForBlockStoreSync previously returned bool, conflating three distinct\nfailure modes (ctx canceled, timeout, lost leadership). The caller in Run\nthen unconditionally called leadershipTransfer() on any false return, which\nis wrong when leadership was already lost.\n\nIntroduce a syncResult enum (syncResultSynced, syncResultTimeout,\nsyncResultLostLeadership, syncResultCanceled) and update Run to handle each\ncase correctly:\n- syncResultCanceled    → return ctx.Err()\n- syncResultLostLeadership → continue without calling leadershipTransfer()\n- syncResultTimeout     → leadershipTransfer() + continue as before\n- syncResultSynced      → refresh raftState/diff and proceed\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* fix(raft): fix gci alignment in syncResult const block\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n---------\n\nCo-authored-by: Claude Sonnet 4.6 <noreply@anthropic.com>",
          "timestamp": "2026-04-20T09:34:03Z",
          "tree_id": "77955a3ad0f421db5db82e84dbae67fb0a0ae959",
          "url": "https://github.com/evstack/ev-node/commit/d9046e6706be9c4b1008be08c30d611e16c4388c"
        },
        "date": 1776678839695,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 37565,
            "unit": "ns/op\t    4960 B/op\t      54 allocs/op",
            "extra": "32187 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 37565,
            "unit": "ns/op",
            "extra": "32187 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 4960,
            "unit": "B/op",
            "extra": "32187 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "32187 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 44039,
            "unit": "ns/op\t   10258 B/op\t      54 allocs/op",
            "extra": "27628 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 44039,
            "unit": "ns/op",
            "extra": "27628 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10258,
            "unit": "B/op",
            "extra": "27628 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "27628 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37221,
            "unit": "ns/op\t    4760 B/op\t      50 allocs/op",
            "extra": "32505 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37221,
            "unit": "ns/op",
            "extra": "32505 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4760,
            "unit": "B/op",
            "extra": "32505 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 50,
            "unit": "allocs/op",
            "extra": "32505 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "raddole@ukr.net",
            "name": "Radovenchyk",
            "username": "Radovenchyk"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "2eb15bd5ecb85ddb548b10f944d70361d2ec4ae8",
          "message": "test: remove racy node startup from execution test (#3264)\n\nfix(test): remove racy node startup from execution test",
          "timestamp": "2026-04-20T11:19:34Z",
          "tree_id": "2efb7427eb4e3b35d45fd5d0b995ba0f9c9f64cf",
          "url": "https://github.com/evstack/ev-node/commit/2eb15bd5ecb85ddb548b10f944d70361d2ec4ae8"
        },
        "date": 1776685165136,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 36535,
            "unit": "ns/op\t    4741 B/op\t      50 allocs/op",
            "extra": "33314 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 36535,
            "unit": "ns/op",
            "extra": "33314 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4741,
            "unit": "B/op",
            "extra": "33314 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 50,
            "unit": "allocs/op",
            "extra": "33314 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 37304,
            "unit": "ns/op\t    4950 B/op\t      54 allocs/op",
            "extra": "32607 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 37304,
            "unit": "ns/op",
            "extra": "32607 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 4950,
            "unit": "B/op",
            "extra": "32607 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "32607 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 42955,
            "unit": "ns/op\t   10242 B/op\t      54 allocs/op",
            "extra": "28118 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 42955,
            "unit": "ns/op",
            "extra": "28118 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10242,
            "unit": "B/op",
            "extra": "28118 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "28118 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "chuanshanjida@outlook.com",
            "name": "chuanshanjida",
            "username": "chuanshanjida"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "83edf6d2355cef0033d86f44bd7359d6e0775137",
          "message": "refactor: modernize sync/atomic usage with typed atomic apis (#3269)\n\nSigned-off-by: chuanshanjida <chuanshanjida@outlook.com>",
          "timestamp": "2026-04-21T07:33:42Z",
          "tree_id": "631f0fcdf2be59af0fe6650d9c4a8c0e41ae8e51",
          "url": "https://github.com/evstack/ev-node/commit/83edf6d2355cef0033d86f44bd7359d6e0775137"
        },
        "date": 1776758021968,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 44860,
            "unit": "ns/op\t   10273 B/op\t      54 allocs/op",
            "extra": "27180 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 44860,
            "unit": "ns/op",
            "extra": "27180 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10273,
            "unit": "B/op",
            "extra": "27180 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "27180 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 38432,
            "unit": "ns/op\t    4785 B/op\t      50 allocs/op",
            "extra": "31478 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 38432,
            "unit": "ns/op",
            "extra": "31478 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4785,
            "unit": "B/op",
            "extra": "31478 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 50,
            "unit": "allocs/op",
            "extra": "31478 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 38698,
            "unit": "ns/op\t    4991 B/op\t      54 allocs/op",
            "extra": "30948 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 38698,
            "unit": "ns/op",
            "extra": "30948 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 4991,
            "unit": "B/op",
            "extra": "30948 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "30948 times\n4 procs"
          }
        ]
      },
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
          "distinct": false,
          "id": "c753c0bb2193faa5bd0b713bb7787757221c59de",
          "message": "fix(rpc): derive /raft/node leadership from raft leader ID (#3266)\n\n* fix(rpc): derive /raft/node leadership from raft leader ID\n\n* fix(raft): add LeaderID doc comment; consolidate raft-status tests\n\n- Add Go doc comment to Node.LeaderID() describing return value, nil-safety,\n  and staleness semantics, consistent with IsLeader/HasQuorum style.\n- Consolidate three near-duplicate TestRegisterCustomHTTPEndpoints_RaftNodeStatus*\n  tests into a single table-driven test covering: leaderID==nodeID (is_leader\n  true), leaderID!=nodeID (is_leader false), empty leaderID fallback, and\n  non-GET method (405). Clarifies that is_leader is derived from LeaderID(),\n  not the IsLeader() field on testRaftNodeSource.\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* fix(rpc): fix gci import grouping in http_test.go\n\nSeparate third-party imports from evstack/ev-node-prefixed imports\nper project gci custom-order config.\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n---------\n\nCo-authored-by: Claude Sonnet 4.6 <noreply@anthropic.com>",
          "timestamp": "2026-04-21T09:58:26Z",
          "tree_id": "12fe1920fc341aee1aa3071ced43d8f90ee95838",
          "url": "https://github.com/evstack/ev-node/commit/c753c0bb2193faa5bd0b713bb7787757221c59de"
        },
        "date": 1776766667311,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 34675,
            "unit": "ns/op\t    4706 B/op\t      50 allocs/op",
            "extra": "34920 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 34675,
            "unit": "ns/op",
            "extra": "34920 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4706,
            "unit": "B/op",
            "extra": "34920 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 50,
            "unit": "allocs/op",
            "extra": "34920 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 35156,
            "unit": "ns/op\t    4910 B/op\t      54 allocs/op",
            "extra": "34400 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 35156,
            "unit": "ns/op",
            "extra": "34400 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 4910,
            "unit": "B/op",
            "extra": "34400 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "34400 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 40754,
            "unit": "ns/op\t   10199 B/op\t      54 allocs/op",
            "extra": "29521 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 40754,
            "unit": "ns/op",
            "extra": "29521 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10199,
            "unit": "B/op",
            "extra": "29521 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "29521 times\n4 procs"
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
          "id": "c1d4996ca8e76b459aa9a6eb5e5a955013303607",
          "message": "build(deps): Bump the all-go group across 5 directories with 14 updates (#3271)\n\n* build(deps): Bump the all-go group across 5 directories with 14 updates\n\nBumps the all-go group with 9 updates in the / directory:\n\n| Package | From | To |\n| --- | --- | --- |\n| [cloud.google.com/go/kms](https://github.com/googleapis/google-cloud-go) | `1.27.0` | `1.29.0` |\n| [connectrpc.com/connect](https://github.com/connectrpc/connect-go) | `1.19.1` | `1.19.2` |\n| [github.com/aws/aws-sdk-go-v2](https://github.com/aws/aws-sdk-go-v2) | `1.41.5` | `1.41.6` |\n| [github.com/aws/aws-sdk-go-v2/config](https://github.com/aws/aws-sdk-go-v2) | `1.32.14` | `1.32.16` |\n| [github.com/aws/aws-sdk-go-v2/service/kms](https://github.com/aws/aws-sdk-go-v2) | `1.50.4` | `1.50.5` |\n| [github.com/celestiaorg/nmt](https://github.com/celestiaorg/nmt) | `0.24.2` | `0.24.3` |\n| [github.com/libp2p/go-libp2p-kad-dht](https://github.com/libp2p/go-libp2p-kad-dht) | `0.39.0` | `0.39.1` |\n| [golang.org/x/crypto](https://github.com/golang/crypto) | `0.49.0` | `0.50.0` |\n| [golang.org/x/net](https://github.com/golang/net) | `0.52.0` | `0.53.0` |\n\nBumps the all-go group with 1 update in the /execution/evm directory: [github.com/evstack/ev-node](https://github.com/evstack/ev-node).\nBumps the all-go group with 3 updates in the /execution/grpc directory: [connectrpc.com/connect](https://github.com/connectrpc/connect-go), [golang.org/x/net](https://github.com/golang/net) and [github.com/evstack/ev-node](https://github.com/evstack/ev-node).\nBumps the all-go group with 2 updates in the /test/docker-e2e directory: [github.com/celestiaorg/tastora](https://github.com/celestiaorg/tastora) and [github.com/evstack/ev-node/execution/evm](https://github.com/evstack/ev-node).\nBumps the all-go group with 1 update in the /test/e2e directory: [github.com/celestiaorg/tastora](https://github.com/celestiaorg/tastora).\n\n\nUpdates `cloud.google.com/go/kms` from 1.27.0 to 1.29.0\n- [Release notes](https://github.com/googleapis/google-cloud-go/releases)\n- [Changelog](https://github.com/googleapis/google-cloud-go/blob/main/documentai/CHANGES.md)\n- [Commits](https://github.com/googleapis/google-cloud-go/compare/kms/v1.27.0...dlp/v1.29.0)\n\nUpdates `connectrpc.com/connect` from 1.19.1 to 1.19.2\n- [Release notes](https://github.com/connectrpc/connect-go/releases)\n- [Changelog](https://github.com/connectrpc/connect-go/blob/main/RELEASE.md)\n- [Commits](https://github.com/connectrpc/connect-go/compare/v1.19.1...v1.19.2)\n\nUpdates `github.com/aws/aws-sdk-go-v2` from 1.41.5 to 1.41.6\n- [Release notes](https://github.com/aws/aws-sdk-go-v2/releases)\n- [Commits](https://github.com/aws/aws-sdk-go-v2/compare/v1.41.5...v1.41.6)\n\nUpdates `github.com/aws/aws-sdk-go-v2/config` from 1.32.14 to 1.32.16\n- [Release notes](https://github.com/aws/aws-sdk-go-v2/releases)\n- [Commits](https://github.com/aws/aws-sdk-go-v2/compare/config/v1.32.14...config/v1.32.16)\n\nUpdates `github.com/aws/aws-sdk-go-v2/service/kms` from 1.50.4 to 1.50.5\n- [Release notes](https://github.com/aws/aws-sdk-go-v2/releases)\n- [Commits](https://github.com/aws/aws-sdk-go-v2/compare/service/ssm/v1.50.4...service/ssm/v1.50.5)\n\nUpdates `github.com/aws/smithy-go` from 1.24.3 to 1.25.0\n- [Release notes](https://github.com/aws/smithy-go/releases)\n- [Changelog](https://github.com/aws/smithy-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/aws/smithy-go/compare/v1.24.3...v1.25.0)\n\nUpdates `github.com/celestiaorg/nmt` from 0.24.2 to 0.24.3\n- [Release notes](https://github.com/celestiaorg/nmt/releases)\n- [Commits](https://github.com/celestiaorg/nmt/compare/v0.24.2...v0.24.3)\n\nUpdates `github.com/libp2p/go-libp2p-kad-dht` from 0.39.0 to 0.39.1\n- [Release notes](https://github.com/libp2p/go-libp2p-kad-dht/releases)\n- [Commits](https://github.com/libp2p/go-libp2p-kad-dht/compare/v0.39.0...v0.39.1)\n\nUpdates `golang.org/x/crypto` from 0.49.0 to 0.50.0\n- [Commits](https://github.com/golang/crypto/compare/v0.49.0...v0.50.0)\n\nUpdates `golang.org/x/net` from 0.52.0 to 0.53.0\n- [Commits](https://github.com/golang/net/compare/v0.52.0...v0.53.0)\n\nUpdates `google.golang.org/api` from 0.273.1 to 0.274.0\n- [Release notes](https://github.com/googleapis/google-api-go-client/releases)\n- [Changelog](https://github.com/googleapis/google-api-go-client/blob/main/CHANGES.md)\n- [Commits](https://github.com/googleapis/google-api-go-client/compare/v0.273.1...v0.274.0)\n\nUpdates `github.com/evstack/ev-node` from 1.0.0 to 1.1.0\n- [Release notes](https://github.com/evstack/ev-node/releases)\n- [Changelog](https://github.com/evstack/ev-node/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/evstack/ev-node/compare/v1.0.0...v1.1.0)\n\nUpdates `connectrpc.com/connect` from 1.19.1 to 1.19.2\n- [Release notes](https://github.com/connectrpc/connect-go/releases)\n- [Changelog](https://github.com/connectrpc/connect-go/blob/main/RELEASE.md)\n- [Commits](https://github.com/connectrpc/connect-go/compare/v1.19.1...v1.19.2)\n\nUpdates `golang.org/x/net` from 0.52.0 to 0.53.0\n- [Commits](https://github.com/golang/net/compare/v0.52.0...v0.53.0)\n\nUpdates `github.com/evstack/ev-node` from 1.0.0 to 1.1.0\n- [Release notes](https://github.com/evstack/ev-node/releases)\n- [Changelog](https://github.com/evstack/ev-node/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/evstack/ev-node/compare/v1.0.0...v1.1.0)\n\nUpdates `github.com/celestiaorg/tastora` from 0.17.0 to 0.19.0\n- [Release notes](https://github.com/celestiaorg/tastora/releases)\n- [Commits](https://github.com/celestiaorg/tastora/compare/v0.17.0...v0.19.0)\n\nUpdates `github.com/evstack/ev-node/execution/evm` from 1.0.0 to 1.0.1\n- [Release notes](https://github.com/evstack/ev-node/releases)\n- [Changelog](https://github.com/evstack/ev-node/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/evstack/ev-node/compare/v1.0.0...execution/evm/v1.0.1)\n\nUpdates `github.com/celestiaorg/tastora` from 0.16.1-0.20260312082036-2ee1b0a2ac4e to 0.19.0\n- [Release notes](https://github.com/celestiaorg/tastora/releases)\n- [Commits](https://github.com/celestiaorg/tastora/compare/v0.17.0...v0.19.0)\n\n---\nupdated-dependencies:\n- dependency-name: cloud.google.com/go/kms\n  dependency-version: 1.29.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: connectrpc.com/connect\n  dependency-version: 1.19.2\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/aws/aws-sdk-go-v2\n  dependency-version: 1.41.6\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/aws/aws-sdk-go-v2/config\n  dependency-version: 1.32.16\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/aws/aws-sdk-go-v2/service/kms\n  dependency-version: 1.50.5\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/aws/smithy-go\n  dependency-version: 1.25.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/celestiaorg/nmt\n  dependency-version: 0.24.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/libp2p/go-libp2p-kad-dht\n  dependency-version: 0.39.1\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: golang.org/x/crypto\n  dependency-version: 0.50.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: golang.org/x/net\n  dependency-version: 0.53.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: google.golang.org/api\n  dependency-version: 0.274.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/evstack/ev-node\n  dependency-version: 1.1.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: connectrpc.com/connect\n  dependency-version: 1.19.2\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: golang.org/x/net\n  dependency-version: 0.53.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/evstack/ev-node\n  dependency-version: 1.1.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/celestiaorg/tastora\n  dependency-version: 0.19.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/evstack/ev-node/execution/evm\n  dependency-version: 1.0.1\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/celestiaorg/tastora\n  dependency-version: 0.19.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\n\n* downgrade tastora\n\n---------\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>\nCo-authored-by: Julien Robert <julien@rbrt.fr>",
          "timestamp": "2026-04-21T12:45:24Z",
          "tree_id": "af1e4168064c00d22c4221b5c0afb5a0db4b55db",
          "url": "https://github.com/evstack/ev-node/commit/c1d4996ca8e76b459aa9a6eb5e5a955013303607"
        },
        "date": 1776776898516,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 39283,
            "unit": "ns/op\t    4798 B/op\t      50 allocs/op",
            "extra": "30992 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 39283,
            "unit": "ns/op",
            "extra": "30992 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4798,
            "unit": "B/op",
            "extra": "30992 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 50,
            "unit": "allocs/op",
            "extra": "30992 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 39704,
            "unit": "ns/op\t    5005 B/op\t      54 allocs/op",
            "extra": "30444 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 39704,
            "unit": "ns/op",
            "extra": "30444 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 5005,
            "unit": "B/op",
            "extra": "30444 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "30444 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 45841,
            "unit": "ns/op\t   10294 B/op\t      54 allocs/op",
            "extra": "26593 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 45841,
            "unit": "ns/op",
            "extra": "26593 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10294,
            "unit": "B/op",
            "extra": "26593 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "26593 times\n4 procs"
          }
        ]
      }
    ]
  }
}