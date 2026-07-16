window.BENCHMARK_DATA = {
  "lastUpdate": 1784214160329,
  "repoUrl": "https://github.com/evstack/ev-node",
  "entries": {
    "EVM Contract Roundtrip": [
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
          "id": "8e8cddedd78125d403ccec66245c203b2372d204",
          "message": "chore(deps): fix all open dependabot security alerts (#3388)\n\n* chore(deps): fix all open dependabot security alerts\n\n- tools/da-debug: bump golang.org/x/crypto to v0.52.0 and golang.org/x/net\n  to v0.55.0 (7 critical SSH advisories, plus high/medium DoS fixes)\n- root, apps/testapp, apps/evm, test/e2e: bump quic-go to v0.59.1\n  (HTTP/3 QPACK trailer expansion memory exhaustion)\n- apps/loadgen: bump filippo.io/edwards25519 to v1.1.1\n- docs: force patched vite ^6.4.3, esbuild ^0.25.0, lodash-es ^4.18.0 via\n  yarn resolutions (vitepress 1.x pins EOL vite 5; vitepress 2 is still alpha)\n- docs: remove stale package-lock.json — CI workflows only use yarn, the npm\n  lockfile just generated duplicate alerts\n\nCo-Authored-By: Claude Fable 5 <noreply@anthropic.com>\n\n* chore: remove accidentally committed da-debug binary\n\nCo-Authored-By: Claude Fable 5 <noreply@anthropic.com>\n\n* chore(deps): propagate quic-go v0.59.1 sums via just tidy-all\n\nCo-Authored-By: Claude Fable 5 <noreply@anthropic.com>\n\n---------\n\nCo-authored-by: Claude Fable 5 <noreply@anthropic.com>",
          "timestamp": "2026-07-16T16:19:01+02:00",
          "tree_id": "ac65b9a0278e10c67ddfbc92b94264c4c9ecd05e",
          "url": "https://github.com/evstack/ev-node/commit/8e8cddedd78125d403ccec66245c203b2372d204"
        },
        "date": 1784211941315,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 900574726,
            "unit": "ns/op\t32649216 B/op\t  184375 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 900574726,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 32649216,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 184375,
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
          "id": "f59b3319fb810251e15e9b48895a51fe07024a64",
          "message": "fix(evm): don't create single sequencer for non-aggregator nodes (#3389)\n\n* fix(evm): don't create single sequencer for non-aggregator nodes\n\nP2P-only followers (no DA address) crashed on startup with a nil\npointer dereference: the EVM app's createSequencer unconditionally\nbuilt a single sequencer, whose forced-inclusion retriever calls\nGetForcedInclusionNamespace on the nil DA client.\n\nMirror the testapp guard from #3386 by skipping sequencer creation\nfor non-aggregators, and fail fast in single.NewSequencer when the\nDA client is nil so future regressions surface as a clean error\ninstead of a SIGSEGV.\n\nCo-Authored-By: Claude Fable 5 <noreply@anthropic.com>\n\n* fix: keep sequencer for promotable followers and require DA for promotable mode\n\nPromotable nodes start as followers but can be promoted to proposer at\nruntime, which hands the sequencer to the aggregator components. The\nnon-aggregator guard (added to testapp in #3386 and mirrored here for\nthe EVM app) would leave promotable nodes with a nil sequencer and\nbreak promotion. Create the sequencer for promotable nodes too, and\nrequire a DA address for promotable mode at config validation, same as\naggregator mode.\n\nCo-Authored-By: Claude Fable 5 <noreply@anthropic.com>\n\n---------\n\nCo-authored-by: Claude Fable 5 <noreply@anthropic.com>",
          "timestamp": "2026-07-16T16:58:33+02:00",
          "tree_id": "e9d0191939e837d5f1aae26c5321368d8c0719e7",
          "url": "https://github.com/evstack/ev-node/commit/f59b3319fb810251e15e9b48895a51fe07024a64"
        },
        "date": 1784214151382,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 898275188,
            "unit": "ns/op\t32362640 B/op\t  184457 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 898275188,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 32362640,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 184457,
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
          "id": "8e8cddedd78125d403ccec66245c203b2372d204",
          "message": "chore(deps): fix all open dependabot security alerts (#3388)\n\n* chore(deps): fix all open dependabot security alerts\n\n- tools/da-debug: bump golang.org/x/crypto to v0.52.0 and golang.org/x/net\n  to v0.55.0 (7 critical SSH advisories, plus high/medium DoS fixes)\n- root, apps/testapp, apps/evm, test/e2e: bump quic-go to v0.59.1\n  (HTTP/3 QPACK trailer expansion memory exhaustion)\n- apps/loadgen: bump filippo.io/edwards25519 to v1.1.1\n- docs: force patched vite ^6.4.3, esbuild ^0.25.0, lodash-es ^4.18.0 via\n  yarn resolutions (vitepress 1.x pins EOL vite 5; vitepress 2 is still alpha)\n- docs: remove stale package-lock.json — CI workflows only use yarn, the npm\n  lockfile just generated duplicate alerts\n\nCo-Authored-By: Claude Fable 5 <noreply@anthropic.com>\n\n* chore: remove accidentally committed da-debug binary\n\nCo-Authored-By: Claude Fable 5 <noreply@anthropic.com>\n\n* chore(deps): propagate quic-go v0.59.1 sums via just tidy-all\n\nCo-Authored-By: Claude Fable 5 <noreply@anthropic.com>\n\n---------\n\nCo-authored-by: Claude Fable 5 <noreply@anthropic.com>",
          "timestamp": "2026-07-16T16:19:01+02:00",
          "tree_id": "ac65b9a0278e10c67ddfbc92b94264c4c9ecd05e",
          "url": "https://github.com/evstack/ev-node/commit/8e8cddedd78125d403ccec66245c203b2372d204"
        },
        "date": 1784211947950,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 38012,
            "unit": "ns/op\t    4877 B/op\t      51 allocs/op",
            "extra": "32211 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 38012,
            "unit": "ns/op",
            "extra": "32211 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4877,
            "unit": "B/op",
            "extra": "32211 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 51,
            "unit": "allocs/op",
            "extra": "32211 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 38416,
            "unit": "ns/op\t    5082 B/op\t      55 allocs/op",
            "extra": "31683 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 38416,
            "unit": "ns/op",
            "extra": "31683 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 5082,
            "unit": "B/op",
            "extra": "31683 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "31683 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 44851,
            "unit": "ns/op\t   10391 B/op\t      55 allocs/op",
            "extra": "26946 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 44851,
            "unit": "ns/op",
            "extra": "26946 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10391,
            "unit": "B/op",
            "extra": "26946 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "26946 times\n4 procs"
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
          "id": "f59b3319fb810251e15e9b48895a51fe07024a64",
          "message": "fix(evm): don't create single sequencer for non-aggregator nodes (#3389)\n\n* fix(evm): don't create single sequencer for non-aggregator nodes\n\nP2P-only followers (no DA address) crashed on startup with a nil\npointer dereference: the EVM app's createSequencer unconditionally\nbuilt a single sequencer, whose forced-inclusion retriever calls\nGetForcedInclusionNamespace on the nil DA client.\n\nMirror the testapp guard from #3386 by skipping sequencer creation\nfor non-aggregators, and fail fast in single.NewSequencer when the\nDA client is nil so future regressions surface as a clean error\ninstead of a SIGSEGV.\n\nCo-Authored-By: Claude Fable 5 <noreply@anthropic.com>\n\n* fix: keep sequencer for promotable followers and require DA for promotable mode\n\nPromotable nodes start as followers but can be promoted to proposer at\nruntime, which hands the sequencer to the aggregator components. The\nnon-aggregator guard (added to testapp in #3386 and mirrored here for\nthe EVM app) would leave promotable nodes with a nil sequencer and\nbreak promotion. Create the sequencer for promotable nodes too, and\nrequire a DA address for promotable mode at config validation, same as\naggregator mode.\n\nCo-Authored-By: Claude Fable 5 <noreply@anthropic.com>\n\n---------\n\nCo-authored-by: Claude Fable 5 <noreply@anthropic.com>",
          "timestamp": "2026-07-16T16:58:33+02:00",
          "tree_id": "e9d0191939e837d5f1aae26c5321368d8c0719e7",
          "url": "https://github.com/evstack/ev-node/commit/f59b3319fb810251e15e9b48895a51fe07024a64"
        },
        "date": 1784214159459,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 41481,
            "unit": "ns/op\t   10308 B/op\t      55 allocs/op",
            "extra": "29575 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 41481,
            "unit": "ns/op",
            "extra": "29575 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10308,
            "unit": "B/op",
            "extra": "29575 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "29575 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 35556,
            "unit": "ns/op\t    4833 B/op\t      51 allocs/op",
            "extra": "34122 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 35556,
            "unit": "ns/op",
            "extra": "34122 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4833,
            "unit": "B/op",
            "extra": "34122 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 51,
            "unit": "allocs/op",
            "extra": "34122 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 35547,
            "unit": "ns/op\t    5029 B/op\t      55 allocs/op",
            "extra": "33966 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 35547,
            "unit": "ns/op",
            "extra": "33966 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 5029,
            "unit": "B/op",
            "extra": "33966 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "33966 times\n4 procs"
          }
        ]
      }
    ]
  }
}