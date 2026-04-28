window.BENCHMARK_DATA = {
  "lastUpdate": 1777384900065,
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
          "id": "389e904d623f6b8cb60113aa19df836a2103f155",
          "message": "docs: brand-aligned syntax theme for code blocks (#3294)\n\n* docs: better code readability\n\n* chore: restore yarn.lock to main\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* docs(style): address PR review feedback\n\n- Add `\"type\": \"dark\"` to ev-dark.json theme manifest\n- Raise punctuation token contrast from #505050 to #767676 (WCAG AA)\n- Align --vp-code-block-color CSS var with ev-dark default text (#dbd7ca)\n- Use ThemeRegistration type instead of `as any` in config.ts\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n---------\n\nCo-authored-by: Claude Sonnet 4.6 <noreply@anthropic.com>",
          "timestamp": "2026-04-28T15:54:57+02:00",
          "tree_id": "61d3f8e85e149aa29b58597284df9ca243f1d49a",
          "url": "https://github.com/evstack/ev-node/commit/389e904d623f6b8cb60113aa19df836a2103f155"
        },
        "date": 1777384892764,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 882156127,
            "unit": "ns/op\t30411152 B/op\t  165250 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 882156127,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 30411152,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 165250,
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
          "id": "389e904d623f6b8cb60113aa19df836a2103f155",
          "message": "docs: brand-aligned syntax theme for code blocks (#3294)\n\n* docs: better code readability\n\n* chore: restore yarn.lock to main\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n* docs(style): address PR review feedback\n\n- Add `\"type\": \"dark\"` to ev-dark.json theme manifest\n- Raise punctuation token contrast from #505050 to #767676 (WCAG AA)\n- Align --vp-code-block-color CSS var with ev-dark default text (#dbd7ca)\n- Use ThemeRegistration type instead of `as any` in config.ts\n\nCo-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>\n\n---------\n\nCo-authored-by: Claude Sonnet 4.6 <noreply@anthropic.com>",
          "timestamp": "2026-04-28T15:54:57+02:00",
          "tree_id": "61d3f8e85e149aa29b58597284df9ca243f1d49a",
          "url": "https://github.com/evstack/ev-node/commit/389e904d623f6b8cb60113aa19df836a2103f155"
        },
        "date": 1777384899390,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 39850,
            "unit": "ns/op\t    4806 B/op\t      50 allocs/op",
            "extra": "30684 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 39850,
            "unit": "ns/op",
            "extra": "30684 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4806,
            "unit": "B/op",
            "extra": "30684 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 50,
            "unit": "allocs/op",
            "extra": "30684 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 40467,
            "unit": "ns/op\t    5027 B/op\t      54 allocs/op",
            "extra": "29653 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 40467,
            "unit": "ns/op",
            "extra": "29653 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 5027,
            "unit": "B/op",
            "extra": "29653 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "29653 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 47212,
            "unit": "ns/op\t   10310 B/op\t      54 allocs/op",
            "extra": "26125 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 47212,
            "unit": "ns/op",
            "extra": "26125 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10310,
            "unit": "B/op",
            "extra": "26125 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "26125 times\n4 procs"
          }
        ]
      }
    ]
  }
}