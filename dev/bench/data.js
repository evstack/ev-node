window.BENCHMARK_DATA = {
  "lastUpdate": 1784211428368,
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
          "id": "19d3a86e7567d76c7a75194eac561ad7da9ffe0a",
          "message": "feat: support p2p-only followers without DA (#3386)\n\n* support p2p-only followers without DA\n\n* require DA endpoint for aggregators\n\n* add DA configuration changelog entry\n\n* normalize DA endpoint at startup\n\n* scope DA endpoint validation to node startup",
          "timestamp": "2026-07-16T12:27:42+02:00",
          "tree_id": "ac236361fd51b929f63b21cbf567e026e4587d69",
          "url": "https://github.com/evstack/ev-node/commit/19d3a86e7567d76c7a75194eac561ad7da9ffe0a"
        },
        "date": 1784197893195,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 912978348,
            "unit": "ns/op\t31955952 B/op\t  178075 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 912978348,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 31955952,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 178075,
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
          "id": "b4b895b47afa4795155c0782f529d49f197c0d55",
          "message": "fix(cache,da): drop finalized snapshot entries on restore and drain DA subscription channel (#3385)\n\nTwo memory fixes for long-running nodes:\n\nCache restore: the DA-inclusion snapshot is only written on graceful\nshutdown, so after a crash (e.g. OOM kill) the restored snapshot can\ncontain heights already below the persisted DA-included watermark. The\ninclusion loop never evicts below its watermark, so those placeholder\nentries leaked for the process lifetime and were re-persisted on every\nsubsequent save, growing the snapshot monotonically across crash/restart\ncycles. RestoreFromStore now skips entries at or below the persisted\nDAIncludedHeight; skipped entries still seed maxDAHeight so DaHeight()\nis unchanged.\n\nDA subscription: the Subscribe wrapper goroutine exited on ctx\ncancellation without draining the underlying jsonrpc channel, leaving\nthe go-jsonrpc delivery goroutine blocked on send — it never observed\nthe cancellation and never closed its channel, leaking one goroutine\nper watchdog reconnect. The wrapper now drains the raw channel on exit.\n\nCo-authored-by: Claude Fable 5 <noreply@anthropic.com>",
          "timestamp": "2026-07-16T12:52:54+02:00",
          "tree_id": "8b332b599d1cbc3e55726e5c45ccdbb116e2d612",
          "url": "https://github.com/evstack/ev-node/commit/b4b895b47afa4795155c0782f529d49f197c0d55"
        },
        "date": 1784199262994,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 898301478,
            "unit": "ns/op\t32316624 B/op\t  184691 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 898301478,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 32316624,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 184691,
            "unit": "allocs/op",
            "extra": "2 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "pengqima@outlook.com",
            "name": "pengqima",
            "username": "pengqima"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "974265e35904a5b159c683cd27f77a5d7ab8e269",
          "message": "chore: fix based sequencer terminology in state comments (#3387)\n\nSigned-off-by: pengqima <pengqima@outlook.com>",
          "timestamp": "2026-07-16T16:12:59+02:00",
          "tree_id": "c3d45ffd57a7adce814958d444106032112e317d",
          "url": "https://github.com/evstack/ev-node/commit/974265e35904a5b159c683cd27f77a5d7ab8e269"
        },
        "date": 1784211423089,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 854550832,
            "unit": "ns/op\t31466236 B/op\t  182577 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 854550832,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 31466236,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 182577,
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
          "id": "19d3a86e7567d76c7a75194eac561ad7da9ffe0a",
          "message": "feat: support p2p-only followers without DA (#3386)\n\n* support p2p-only followers without DA\n\n* require DA endpoint for aggregators\n\n* add DA configuration changelog entry\n\n* normalize DA endpoint at startup\n\n* scope DA endpoint validation to node startup",
          "timestamp": "2026-07-16T12:27:42+02:00",
          "tree_id": "ac236361fd51b929f63b21cbf567e026e4587d69",
          "url": "https://github.com/evstack/ev-node/commit/19d3a86e7567d76c7a75194eac561ad7da9ffe0a"
        },
        "date": 1784197898846,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37952,
            "unit": "ns/op\t    4928 B/op\t      51 allocs/op",
            "extra": "30235 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37952,
            "unit": "ns/op",
            "extra": "30235 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4928,
            "unit": "B/op",
            "extra": "30235 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 51,
            "unit": "allocs/op",
            "extra": "30235 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 38108,
            "unit": "ns/op\t    5084 B/op\t      55 allocs/op",
            "extra": "31599 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 38108,
            "unit": "ns/op",
            "extra": "31599 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 5084,
            "unit": "B/op",
            "extra": "31599 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "31599 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 44657,
            "unit": "ns/op\t   10385 B/op\t      55 allocs/op",
            "extra": "27115 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 44657,
            "unit": "ns/op",
            "extra": "27115 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10385,
            "unit": "B/op",
            "extra": "27115 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "27115 times\n4 procs"
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
          "id": "b4b895b47afa4795155c0782f529d49f197c0d55",
          "message": "fix(cache,da): drop finalized snapshot entries on restore and drain DA subscription channel (#3385)\n\nTwo memory fixes for long-running nodes:\n\nCache restore: the DA-inclusion snapshot is only written on graceful\nshutdown, so after a crash (e.g. OOM kill) the restored snapshot can\ncontain heights already below the persisted DA-included watermark. The\ninclusion loop never evicts below its watermark, so those placeholder\nentries leaked for the process lifetime and were re-persisted on every\nsubsequent save, growing the snapshot monotonically across crash/restart\ncycles. RestoreFromStore now skips entries at or below the persisted\nDAIncludedHeight; skipped entries still seed maxDAHeight so DaHeight()\nis unchanged.\n\nDA subscription: the Subscribe wrapper goroutine exited on ctx\ncancellation without draining the underlying jsonrpc channel, leaving\nthe go-jsonrpc delivery goroutine blocked on send — it never observed\nthe cancellation and never closed its channel, leaking one goroutine\nper watchdog reconnect. The wrapper now drains the raw channel on exit.\n\nCo-authored-by: Claude Fable 5 <noreply@anthropic.com>",
          "timestamp": "2026-07-16T12:52:54+02:00",
          "tree_id": "8b332b599d1cbc3e55726e5c45ccdbb116e2d612",
          "url": "https://github.com/evstack/ev-node/commit/b4b895b47afa4795155c0782f529d49f197c0d55"
        },
        "date": 1784199270599,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37483,
            "unit": "ns/op\t    4864 B/op\t      51 allocs/op",
            "extra": "32730 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37483,
            "unit": "ns/op",
            "extra": "32730 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4864,
            "unit": "B/op",
            "extra": "32730 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 51,
            "unit": "allocs/op",
            "extra": "32730 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 37747,
            "unit": "ns/op\t    5067 B/op\t      55 allocs/op",
            "extra": "32275 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 37747,
            "unit": "ns/op",
            "extra": "32275 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 5067,
            "unit": "B/op",
            "extra": "32275 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "32275 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 44233,
            "unit": "ns/op\t   10383 B/op\t      55 allocs/op",
            "extra": "27153 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 44233,
            "unit": "ns/op",
            "extra": "27153 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10383,
            "unit": "B/op",
            "extra": "27153 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "27153 times\n4 procs"
          }
        ]
      }
    ]
  }
}