window.BENCHMARK_DATA = {
  "lastUpdate": 1795614992959,
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
      }
    ]
  }
}