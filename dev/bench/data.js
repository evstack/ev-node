window.BENCHMARK_DATA = {
  "lastUpdate": 1784110150187,
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
          "id": "34b278a7936d893a6f298d851918e69ff1d0c371",
          "message": "chore: remove ev-grpc (#3380)",
          "timestamp": "2026-07-10T16:07:08+02:00",
          "tree_id": "8eb1217b0cb96398f190dd499e134fd1ccda22f7",
          "url": "https://github.com/evstack/ev-node/commit/34b278a7936d893a6f298d851918e69ff1d0c371"
        },
        "date": 1783692526624,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 919321162,
            "unit": "ns/op\t31393060 B/op\t  170407 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 919321162,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 31393060,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 170407,
            "unit": "allocs/op",
            "extra": "2 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "weifanglab@outlook.com",
            "name": "weifanglab",
            "username": "weifanglab"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "dd9903384fc9ae628a9bb86559c476329bd0de59",
          "message": "chore: fix some comments to improve readability (#3382)\n\nSigned-off-by: weifanglab <weifanglab@outlook.com>",
          "timestamp": "2026-07-14T15:46:55+02:00",
          "tree_id": "906acdcf81cbba7f37fd7c04c3830ea2d2b17d35",
          "url": "https://github.com/evstack/ev-node/commit/dd9903384fc9ae628a9bb86559c476329bd0de59"
        },
        "date": 1784037046008,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 915849594,
            "unit": "ns/op\t32126512 B/op\t  180101 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 915849594,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 32126512,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 180101,
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
          "id": "67b2dbc72fd1f06f5917b0ca5343a219edbabc25",
          "message": "chore: remove dead code (#3384)\n\nRemove code unreachable from any binary, test (including\nintegration-tagged tests), or known downstream consumer:\n\n- apps/evm/cmd/post_tx_cmd.go: PostTxCmd was added in #2888 but never\n  registered in the CLI; ev-abci carries its own copy\n- apps/evm/server: handleChainID was never routed; eth_chainId falls\n  through to the execution RPC proxy\n- apps/testapp/kv: HTTPServer.Stop is redundant, Start already shuts\n  down via context cancellation\n- node: MockTester has no users\n- pkg/da/types: SplitID duplicates pkg/da/jsonrpc.SplitID, which is\n  the copy in use\n- pkg/store: GetPrefixEntries has no callers\n\nCo-authored-by: Claude Fable 5 <noreply@anthropic.com>",
          "timestamp": "2026-07-15T12:06:39+02:00",
          "tree_id": "4424f46ce3916ebb62db2286a208395236552ba6",
          "url": "https://github.com/evstack/ev-node/commit/67b2dbc72fd1f06f5917b0ca5343a219edbabc25"
        },
        "date": 1784110142731,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 895552670,
            "unit": "ns/op\t30349140 B/op\t  164225 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 895552670,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 30349140,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 164225,
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
          "id": "34b278a7936d893a6f298d851918e69ff1d0c371",
          "message": "chore: remove ev-grpc (#3380)",
          "timestamp": "2026-07-10T16:07:08+02:00",
          "tree_id": "8eb1217b0cb96398f190dd499e134fd1ccda22f7",
          "url": "https://github.com/evstack/ev-node/commit/34b278a7936d893a6f298d851918e69ff1d0c371"
        },
        "date": 1783692535358,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37617,
            "unit": "ns/op\t    4870 B/op\t      51 allocs/op",
            "extra": "32508 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37617,
            "unit": "ns/op",
            "extra": "32508 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4870,
            "unit": "B/op",
            "extra": "32508 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 51,
            "unit": "allocs/op",
            "extra": "32508 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 38738,
            "unit": "ns/op\t    5083 B/op\t      55 allocs/op",
            "extra": "31660 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 38738,
            "unit": "ns/op",
            "extra": "31660 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 5083,
            "unit": "B/op",
            "extra": "31660 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "31660 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 44528,
            "unit": "ns/op\t   10377 B/op\t      55 allocs/op",
            "extra": "27344 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 44528,
            "unit": "ns/op",
            "extra": "27344 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10377,
            "unit": "B/op",
            "extra": "27344 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "27344 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "weifanglab@outlook.com",
            "name": "weifanglab",
            "username": "weifanglab"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "dd9903384fc9ae628a9bb86559c476329bd0de59",
          "message": "chore: fix some comments to improve readability (#3382)\n\nSigned-off-by: weifanglab <weifanglab@outlook.com>",
          "timestamp": "2026-07-14T15:46:55+02:00",
          "tree_id": "906acdcf81cbba7f37fd7c04c3830ea2d2b17d35",
          "url": "https://github.com/evstack/ev-node/commit/dd9903384fc9ae628a9bb86559c476329bd0de59"
        },
        "date": 1784037052439,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 40392,
            "unit": "ns/op\t    4934 B/op\t      51 allocs/op",
            "extra": "30004 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 40392,
            "unit": "ns/op",
            "extra": "30004 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4934,
            "unit": "B/op",
            "extra": "30004 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 51,
            "unit": "allocs/op",
            "extra": "30004 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 40909,
            "unit": "ns/op\t    5136 B/op\t      55 allocs/op",
            "extra": "29674 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 40909,
            "unit": "ns/op",
            "extra": "29674 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 5136,
            "unit": "B/op",
            "extra": "29674 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "29674 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 47419,
            "unit": "ns/op\t   10442 B/op\t      55 allocs/op",
            "extra": "25536 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 47419,
            "unit": "ns/op",
            "extra": "25536 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10442,
            "unit": "B/op",
            "extra": "25536 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "25536 times\n4 procs"
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
          "id": "67b2dbc72fd1f06f5917b0ca5343a219edbabc25",
          "message": "chore: remove dead code (#3384)\n\nRemove code unreachable from any binary, test (including\nintegration-tagged tests), or known downstream consumer:\n\n- apps/evm/cmd/post_tx_cmd.go: PostTxCmd was added in #2888 but never\n  registered in the CLI; ev-abci carries its own copy\n- apps/evm/server: handleChainID was never routed; eth_chainId falls\n  through to the execution RPC proxy\n- apps/testapp/kv: HTTPServer.Stop is redundant, Start already shuts\n  down via context cancellation\n- node: MockTester has no users\n- pkg/da/types: SplitID duplicates pkg/da/jsonrpc.SplitID, which is\n  the copy in use\n- pkg/store: GetPrefixEntries has no callers\n\nCo-authored-by: Claude Fable 5 <noreply@anthropic.com>",
          "timestamp": "2026-07-15T12:06:39+02:00",
          "tree_id": "4424f46ce3916ebb62db2286a208395236552ba6",
          "url": "https://github.com/evstack/ev-node/commit/67b2dbc72fd1f06f5917b0ca5343a219edbabc25"
        },
        "date": 1784110149534,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 36984,
            "unit": "ns/op\t    4862 B/op\t      51 allocs/op",
            "extra": "32851 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 36984,
            "unit": "ns/op",
            "extra": "32851 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4862,
            "unit": "B/op",
            "extra": "32851 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 51,
            "unit": "allocs/op",
            "extra": "32851 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 37718,
            "unit": "ns/op\t    5068 B/op\t      55 allocs/op",
            "extra": "32268 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 37718,
            "unit": "ns/op",
            "extra": "32268 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 5068,
            "unit": "B/op",
            "extra": "32268 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "32268 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 43988,
            "unit": "ns/op\t   10364 B/op\t      55 allocs/op",
            "extra": "27728 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 43988,
            "unit": "ns/op",
            "extra": "27728 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10364,
            "unit": "B/op",
            "extra": "27728 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "27728 times\n4 procs"
          }
        ]
      }
    ]
  }
}