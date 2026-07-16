window.BENCHMARK_DATA = {
  "lastUpdate": 1784211946273,
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
      }
    ]
  }
}