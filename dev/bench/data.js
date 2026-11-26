window.BENCHMARK_DATA = {
  "lastUpdate": 1795709504039,
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
          "id": "010ab7b725131066a7ff2f8e4d587f5ee946fc3a",
          "message": "fix(p2p): add p2p.announce_addresses to override advertised addresses (#3510)\n\n* feat(p2p): add p2p.announce_addresses to override advertised addresses\n\nNodes running in containers without host networking (or behind NAT)\nadvertised only their loopback/Docker bridge addresses, so remote peers\ncould not dial them. The new option installs a libp2p AddrsFactory that\nreplaces the auto-detected addresses with the configured multiaddrs for\nidentify, the DHT and namespace discovery.\n\nCo-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>\n\n* fix(p2p): reject announce addresses that parse to an empty list\n\nA value made only of separators (e.g. \",\") passed the empty-string guard\nand installed an AddrsFactory returning no addresses, so the node\nadvertised nothing. Also make the test order-independent (libp2p sorts\nthe factory output), align the p2p README table and link the PR in the\nchangelog.\n\nCo-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>\n\n---------\n\nCo-authored-by: Claude Opus 5.5 <noreply@anthropic.com>",
          "timestamp": "2026-11-25T13:27:14Z",
          "tree_id": "96b8a91e956c5eca9343594e59811b7e864ec898",
          "url": "https://github.com/evstack/ev-node/commit/010ab7b725131066a7ff2f8e4d587f5ee946fc3a"
        },
        "date": 1795614988408,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 924299566,
            "unit": "ns/op\t 4278304 B/op\t   37209 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 924299566,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 4278304,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 37209,
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
          "id": "e8626b1094d670805387d258534fb5113cd0c006",
          "message": "ci: automate releases on semver tags (#3518)\n\n* Froze CHANGELOG.md: replaced the [Unreleased] block (#3510) with a note\n\n* Added optional boolean input `latest` (default false) to docker-build-pu\n\n* Added cliff.toml at the repo root as specified; verified with git-cliff\n\n* Removed the orphaned [Unreleased] link reference definition (and precedi\n\n* Added .github/workflows/release.yml: a release workflow triggered by v*.\n\n* Deleted .github/workflows/release-github.yml and .github/workflows/relea\n\n* Rewrote RELEASE.md around a tag-driven 'Cutting a Release' flow with PR-\n\n* RELEASE.md: keep the apps/* replace directives so images built from the\n\n* Verified release flow (actionlint, git-cliff v1.2.3..v1.3.0 and v1.0.0..\n\n* Removed the stray NOTES.md dry-run artifact from the repo root.\n\n* Fixed all 5 findings: added v1.4.0/v1.4.1 CHANGELOG sections and moved t",
          "timestamp": "2026-11-26T15:40:19Z",
          "tree_id": "530db3fac7925fec1363b2cdb47303c35f758580",
          "url": "https://github.com/evstack/ev-node/commit/e8626b1094d670805387d258534fb5113cd0c006"
        },
        "date": 1795709498879,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 910311228,
            "unit": "ns/op\t 4239972 B/op\t   36574 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 910311228,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 4239972,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 36574,
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
          "id": "010ab7b725131066a7ff2f8e4d587f5ee946fc3a",
          "message": "fix(p2p): add p2p.announce_addresses to override advertised addresses (#3510)\n\n* feat(p2p): add p2p.announce_addresses to override advertised addresses\n\nNodes running in containers without host networking (or behind NAT)\nadvertised only their loopback/Docker bridge addresses, so remote peers\ncould not dial them. The new option installs a libp2p AddrsFactory that\nreplaces the auto-detected addresses with the configured multiaddrs for\nidentify, the DHT and namespace discovery.\n\nCo-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>\n\n* fix(p2p): reject announce addresses that parse to an empty list\n\nA value made only of separators (e.g. \",\") passed the empty-string guard\nand installed an AddrsFactory returning no addresses, so the node\nadvertised nothing. Also make the test order-independent (libp2p sorts\nthe factory output), align the p2p README table and link the PR in the\nchangelog.\n\nCo-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>\n\n---------\n\nCo-authored-by: Claude Opus 5.5 <noreply@anthropic.com>",
          "timestamp": "2026-11-25T13:27:14Z",
          "tree_id": "96b8a91e956c5eca9343594e59811b7e864ec898",
          "url": "https://github.com/evstack/ev-node/commit/010ab7b725131066a7ff2f8e4d587f5ee946fc3a"
        },
        "date": 1795614994570,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 25606,
            "unit": "ns/op\t    5093 B/op\t      51 allocs/op",
            "extra": "47730 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 25606,
            "unit": "ns/op",
            "extra": "47730 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 5093,
            "unit": "B/op",
            "extra": "47730 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 51,
            "unit": "allocs/op",
            "extra": "47730 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 26076,
            "unit": "ns/op\t    5280 B/op\t      55 allocs/op",
            "extra": "47671 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 26076,
            "unit": "ns/op",
            "extra": "47671 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 5280,
            "unit": "B/op",
            "extra": "47671 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "47671 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 30879,
            "unit": "ns/op\t   10124 B/op\t      55 allocs/op",
            "extra": "37707 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 30879,
            "unit": "ns/op",
            "extra": "37707 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10124,
            "unit": "B/op",
            "extra": "37707 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "37707 times\n4 procs"
          }
        ]
      }
    ]
  }
}