window.BENCHMARK_DATA = {
  "lastUpdate": 1782809034686,
  "repoUrl": "https://github.com/evstack/ev-node",
  "entries": {
    "EVM Contract Roundtrip": [
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
          "id": "69c655c80bab7f129fe6f3ad4cd27e48bdbcc603",
          "message": "build(deps): Bump dompurify from 3.4.0 to 3.4.11 in /docs in the npm_and_yarn group across 1 directory (#3366)\n\nbuild(deps): Bump dompurify\n\nBumps the npm_and_yarn group with 1 update in the /docs directory: [dompurify](https://github.com/cure53/DOMPurify).\n\n\nUpdates `dompurify` from 3.4.0 to 3.4.11\n- [Release notes](https://github.com/cure53/DOMPurify/releases)\n- [Commits](https://github.com/cure53/DOMPurify/compare/3.4.0...3.4.11)\n\n---\nupdated-dependencies:\n- dependency-name: dompurify\n  dependency-version: 3.4.11\n  dependency-type: indirect\n  dependency-group: npm_and_yarn\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-06-29T12:02:17+02:00",
          "tree_id": "8d16df760af045f500d1ac88648f9edae597b8d5",
          "url": "https://github.com/evstack/ev-node/commit/69c655c80bab7f129fe6f3ad4cd27e48bdbcc603"
        },
        "date": 1782727569653,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 899645124,
            "unit": "ns/op\t31028384 B/op\t  168446 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 899645124,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 31028384,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 168446,
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
          "id": "a5ad63cce26a5846c524edc98ef0a26238d4b6c8",
          "message": "build(deps): Bump the all-go group across 7 directories with 11 updates (#3364)\n\n* build(deps): Bump the all-go group across 7 directories with 11 updates\n\nBumps the all-go group with 6 updates in the / directory:\n\n| Package | From | To |\n| --- | --- | --- |\n| [github.com/aws/aws-sdk-go-v2](https://github.com/aws/aws-sdk-go-v2) | `1.41.11` | `1.42.0` |\n| [github.com/aws/aws-sdk-go-v2/config](https://github.com/aws/aws-sdk-go-v2) | `1.32.22` | `1.32.25` |\n| [github.com/aws/aws-sdk-go-v2/service/kms](https://github.com/aws/aws-sdk-go-v2) | `1.53.2` | `1.53.4` |\n| [golang.org/x/crypto](https://github.com/golang/crypto) | `0.52.0` | `0.53.0` |\n| [golang.org/x/net](https://github.com/golang/net) | `0.55.0` | `0.56.0` |\n| [google.golang.org/api](https://github.com/googleapis/google-api-go-client) | `0.283.0` | `0.286.0` |\n\nBumps the all-go group with 1 update in the /apps/evm directory: [github.com/ethereum/go-ethereum](https://github.com/ethereum/go-ethereum).\nBumps the all-go group with 1 update in the /apps/loadgen directory: [github.com/spf13/cobra](https://github.com/spf13/cobra).\nBumps the all-go group with 1 update in the /execution/evm directory: [github.com/ethereum/go-ethereum](https://github.com/ethereum/go-ethereum).\nBumps the all-go group with 1 update in the /execution/grpc directory: [golang.org/x/net](https://github.com/golang/net).\nBumps the all-go group with 2 updates in the /test/docker-e2e directory: [github.com/ethereum/go-ethereum](https://github.com/ethereum/go-ethereum) and [github.com/moby/moby/client](https://github.com/moby/moby).\nBumps the all-go group with 1 update in the /test/e2e directory: [github.com/ethereum/go-ethereum](https://github.com/ethereum/go-ethereum).\n\n\nUpdates `github.com/aws/aws-sdk-go-v2` from 1.41.11 to 1.42.0\n- [Release notes](https://github.com/aws/aws-sdk-go-v2/releases)\n- [Commits](https://github.com/aws/aws-sdk-go-v2/compare/v1.41.11...v1.42.0)\n\nUpdates `github.com/aws/aws-sdk-go-v2/config` from 1.32.22 to 1.32.25\n- [Release notes](https://github.com/aws/aws-sdk-go-v2/releases)\n- [Commits](https://github.com/aws/aws-sdk-go-v2/compare/config/v1.32.22...config/v1.32.25)\n\nUpdates `github.com/aws/aws-sdk-go-v2/service/kms` from 1.53.2 to 1.53.4\n- [Release notes](https://github.com/aws/aws-sdk-go-v2/releases)\n- [Commits](https://github.com/aws/aws-sdk-go-v2/compare/service/s3/v1.53.2...service/kms/v1.53.4)\n\nUpdates `github.com/aws/smithy-go` from 1.27.0 to 1.27.1\n- [Release notes](https://github.com/aws/smithy-go/releases)\n- [Changelog](https://github.com/aws/smithy-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/aws/smithy-go/compare/v1.27.0...v1.27.1)\n\nUpdates `golang.org/x/crypto` from 0.52.0 to 0.53.0\n- [Commits](https://github.com/golang/crypto/compare/v0.52.0...v0.53.0)\n\nUpdates `golang.org/x/net` from 0.55.0 to 0.56.0\n- [Commits](https://github.com/golang/net/compare/v0.55.0...v0.56.0)\n\nUpdates `golang.org/x/sync` from 0.20.0 to 0.21.0\n- [Commits](https://github.com/golang/sync/compare/v0.20.0...v0.21.0)\n\nUpdates `google.golang.org/api` from 0.283.0 to 0.286.0\n- [Release notes](https://github.com/googleapis/google-api-go-client/releases)\n- [Changelog](https://github.com/googleapis/google-api-go-client/blob/main/CHANGES.md)\n- [Commits](https://github.com/googleapis/google-api-go-client/compare/v0.283.0...v0.286.0)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.3 to 1.17.4\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.3...v1.17.4)\n\nUpdates `github.com/spf13/cobra` from 1.9.1 to 1.10.2\n- [Release notes](https://github.com/spf13/cobra/releases)\n- [Commits](https://github.com/spf13/cobra/compare/v1.9.1...v1.10.2)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.3 to 1.17.4\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.3...v1.17.4)\n\nUpdates `golang.org/x/net` from 0.55.0 to 0.56.0\n- [Commits](https://github.com/golang/net/compare/v0.55.0...v0.56.0)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.3 to 1.17.4\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.3...v1.17.4)\n\nUpdates `github.com/moby/moby/client` from 0.4.1 to 0.5.0\n- [Release notes](https://github.com/moby/moby/releases)\n- [Changelog](https://github.com/moby/moby/blob/v0.5.0/CHANGELOG.md)\n- [Commits](https://github.com/moby/moby/compare/v0.4.1...v0.5.0)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.3 to 1.17.4\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.3...v1.17.4)\n\n---\nupdated-dependencies:\n- dependency-name: github.com/aws/aws-sdk-go-v2\n  dependency-version: 1.42.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/aws/aws-sdk-go-v2/config\n  dependency-version: 1.32.25\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/aws/aws-sdk-go-v2/service/kms\n  dependency-version: 1.53.4\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/aws/smithy-go\n  dependency-version: 1.27.1\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: golang.org/x/crypto\n  dependency-version: 0.53.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: golang.org/x/net\n  dependency-version: 0.56.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: golang.org/x/sync\n  dependency-version: 0.21.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: google.golang.org/api\n  dependency-version: 0.286.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.4\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/spf13/cobra\n  dependency-version: 1.10.2\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.4\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: golang.org/x/net\n  dependency-version: 0.56.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.4\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/moby/moby/client\n  dependency-version: 0.5.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.4\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\n\n* chore: run just deps after Dependabot update\n\n---------\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>\nCo-authored-by: github-actions[bot] <41898282+github-actions[bot]@users.noreply.github.com>\nCo-authored-by: julienrbrt <julien@rbrt.fr>",
          "timestamp": "2026-06-29T10:51:54Z",
          "tree_id": "2d3e6cd8c802ecb054a36cc52f42343f76b99892",
          "url": "https://github.com/evstack/ev-node/commit/a5ad63cce26a5846c524edc98ef0a26238d4b6c8"
        },
        "date": 1782731586788,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 879082498,
            "unit": "ns/op\t31449880 B/op\t  178351 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 879082498,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 31449880,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 178351,
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
          "id": "daddf655ce327ec097d8d23f4fdc5228eaebeb2e",
          "message": "build(deps): Bump actions/setup-go from 6.4.0 to 6.5.0 (#3368)\n\nBumps [actions/setup-go](https://github.com/actions/setup-go) from 6.4.0 to 6.5.0.\n- [Release notes](https://github.com/actions/setup-go/releases)\n- [Commits](https://github.com/actions/setup-go/compare/v6.4.0...v6.5.0)\n\n---\nupdated-dependencies:\n- dependency-name: actions/setup-go\n  dependency-version: 6.5.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-06-30T10:36:27+02:00",
          "tree_id": "0db84599e6119d30a598a6fad704da22dae6cb96",
          "url": "https://github.com/evstack/ev-node/commit/daddf655ce327ec097d8d23f4fdc5228eaebeb2e"
        },
        "date": 1782809011959,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 908618120,
            "unit": "ns/op\t33034580 B/op\t  189693 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 908618120,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 33034580,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 189693,
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
          "id": "999c53e97c61dacad192b195f256a3e728e811b5",
          "message": "build(deps): Bump actions/cache from 5 to 6 (#3370)\n\nBumps [actions/cache](https://github.com/actions/cache) from 5 to 6.\n- [Release notes](https://github.com/actions/cache/releases)\n- [Changelog](https://github.com/actions/cache/blob/main/RELEASES.md)\n- [Commits](https://github.com/actions/cache/compare/v5...v6)\n\n---\nupdated-dependencies:\n- dependency-name: actions/cache\n  dependency-version: '6'\n  dependency-type: direct:production\n  update-type: version-update:semver-major\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-06-30T10:36:45+02:00",
          "tree_id": "21c496a4bac9220b92b761f53d48d2d53f213f3f",
          "url": "https://github.com/evstack/ev-node/commit/999c53e97c61dacad192b195f256a3e728e811b5"
        },
        "date": 1782809028627,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 912532066,
            "unit": "ns/op\t31839468 B/op\t  174946 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 912532066,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 31839468,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 174946,
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
          "id": "69c655c80bab7f129fe6f3ad4cd27e48bdbcc603",
          "message": "build(deps): Bump dompurify from 3.4.0 to 3.4.11 in /docs in the npm_and_yarn group across 1 directory (#3366)\n\nbuild(deps): Bump dompurify\n\nBumps the npm_and_yarn group with 1 update in the /docs directory: [dompurify](https://github.com/cure53/DOMPurify).\n\n\nUpdates `dompurify` from 3.4.0 to 3.4.11\n- [Release notes](https://github.com/cure53/DOMPurify/releases)\n- [Commits](https://github.com/cure53/DOMPurify/compare/3.4.0...3.4.11)\n\n---\nupdated-dependencies:\n- dependency-name: dompurify\n  dependency-version: 3.4.11\n  dependency-type: indirect\n  dependency-group: npm_and_yarn\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-06-29T12:02:17+02:00",
          "tree_id": "8d16df760af045f500d1ac88648f9edae597b8d5",
          "url": "https://github.com/evstack/ev-node/commit/69c655c80bab7f129fe6f3ad4cd27e48bdbcc603"
        },
        "date": 1782727576680,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 38274,
            "unit": "ns/op\t    5079 B/op\t      55 allocs/op",
            "extra": "31792 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 38274,
            "unit": "ns/op",
            "extra": "31792 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 5079,
            "unit": "B/op",
            "extra": "31792 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "31792 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 44718,
            "unit": "ns/op\t   10384 B/op\t      55 allocs/op",
            "extra": "27130 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 44718,
            "unit": "ns/op",
            "extra": "27130 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10384,
            "unit": "B/op",
            "extra": "27130 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "27130 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 38236,
            "unit": "ns/op\t    4879 B/op\t      51 allocs/op",
            "extra": "32136 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 38236,
            "unit": "ns/op",
            "extra": "32136 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4879,
            "unit": "B/op",
            "extra": "32136 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 51,
            "unit": "allocs/op",
            "extra": "32136 times\n4 procs"
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
          "id": "a5ad63cce26a5846c524edc98ef0a26238d4b6c8",
          "message": "build(deps): Bump the all-go group across 7 directories with 11 updates (#3364)\n\n* build(deps): Bump the all-go group across 7 directories with 11 updates\n\nBumps the all-go group with 6 updates in the / directory:\n\n| Package | From | To |\n| --- | --- | --- |\n| [github.com/aws/aws-sdk-go-v2](https://github.com/aws/aws-sdk-go-v2) | `1.41.11` | `1.42.0` |\n| [github.com/aws/aws-sdk-go-v2/config](https://github.com/aws/aws-sdk-go-v2) | `1.32.22` | `1.32.25` |\n| [github.com/aws/aws-sdk-go-v2/service/kms](https://github.com/aws/aws-sdk-go-v2) | `1.53.2` | `1.53.4` |\n| [golang.org/x/crypto](https://github.com/golang/crypto) | `0.52.0` | `0.53.0` |\n| [golang.org/x/net](https://github.com/golang/net) | `0.55.0` | `0.56.0` |\n| [google.golang.org/api](https://github.com/googleapis/google-api-go-client) | `0.283.0` | `0.286.0` |\n\nBumps the all-go group with 1 update in the /apps/evm directory: [github.com/ethereum/go-ethereum](https://github.com/ethereum/go-ethereum).\nBumps the all-go group with 1 update in the /apps/loadgen directory: [github.com/spf13/cobra](https://github.com/spf13/cobra).\nBumps the all-go group with 1 update in the /execution/evm directory: [github.com/ethereum/go-ethereum](https://github.com/ethereum/go-ethereum).\nBumps the all-go group with 1 update in the /execution/grpc directory: [golang.org/x/net](https://github.com/golang/net).\nBumps the all-go group with 2 updates in the /test/docker-e2e directory: [github.com/ethereum/go-ethereum](https://github.com/ethereum/go-ethereum) and [github.com/moby/moby/client](https://github.com/moby/moby).\nBumps the all-go group with 1 update in the /test/e2e directory: [github.com/ethereum/go-ethereum](https://github.com/ethereum/go-ethereum).\n\n\nUpdates `github.com/aws/aws-sdk-go-v2` from 1.41.11 to 1.42.0\n- [Release notes](https://github.com/aws/aws-sdk-go-v2/releases)\n- [Commits](https://github.com/aws/aws-sdk-go-v2/compare/v1.41.11...v1.42.0)\n\nUpdates `github.com/aws/aws-sdk-go-v2/config` from 1.32.22 to 1.32.25\n- [Release notes](https://github.com/aws/aws-sdk-go-v2/releases)\n- [Commits](https://github.com/aws/aws-sdk-go-v2/compare/config/v1.32.22...config/v1.32.25)\n\nUpdates `github.com/aws/aws-sdk-go-v2/service/kms` from 1.53.2 to 1.53.4\n- [Release notes](https://github.com/aws/aws-sdk-go-v2/releases)\n- [Commits](https://github.com/aws/aws-sdk-go-v2/compare/service/s3/v1.53.2...service/kms/v1.53.4)\n\nUpdates `github.com/aws/smithy-go` from 1.27.0 to 1.27.1\n- [Release notes](https://github.com/aws/smithy-go/releases)\n- [Changelog](https://github.com/aws/smithy-go/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/aws/smithy-go/compare/v1.27.0...v1.27.1)\n\nUpdates `golang.org/x/crypto` from 0.52.0 to 0.53.0\n- [Commits](https://github.com/golang/crypto/compare/v0.52.0...v0.53.0)\n\nUpdates `golang.org/x/net` from 0.55.0 to 0.56.0\n- [Commits](https://github.com/golang/net/compare/v0.55.0...v0.56.0)\n\nUpdates `golang.org/x/sync` from 0.20.0 to 0.21.0\n- [Commits](https://github.com/golang/sync/compare/v0.20.0...v0.21.0)\n\nUpdates `google.golang.org/api` from 0.283.0 to 0.286.0\n- [Release notes](https://github.com/googleapis/google-api-go-client/releases)\n- [Changelog](https://github.com/googleapis/google-api-go-client/blob/main/CHANGES.md)\n- [Commits](https://github.com/googleapis/google-api-go-client/compare/v0.283.0...v0.286.0)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.3 to 1.17.4\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.3...v1.17.4)\n\nUpdates `github.com/spf13/cobra` from 1.9.1 to 1.10.2\n- [Release notes](https://github.com/spf13/cobra/releases)\n- [Commits](https://github.com/spf13/cobra/compare/v1.9.1...v1.10.2)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.3 to 1.17.4\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.3...v1.17.4)\n\nUpdates `golang.org/x/net` from 0.55.0 to 0.56.0\n- [Commits](https://github.com/golang/net/compare/v0.55.0...v0.56.0)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.3 to 1.17.4\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.3...v1.17.4)\n\nUpdates `github.com/moby/moby/client` from 0.4.1 to 0.5.0\n- [Release notes](https://github.com/moby/moby/releases)\n- [Changelog](https://github.com/moby/moby/blob/v0.5.0/CHANGELOG.md)\n- [Commits](https://github.com/moby/moby/compare/v0.4.1...v0.5.0)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.3 to 1.17.4\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.3...v1.17.4)\n\n---\nupdated-dependencies:\n- dependency-name: github.com/aws/aws-sdk-go-v2\n  dependency-version: 1.42.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/aws/aws-sdk-go-v2/config\n  dependency-version: 1.32.25\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/aws/aws-sdk-go-v2/service/kms\n  dependency-version: 1.53.4\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/aws/smithy-go\n  dependency-version: 1.27.1\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: golang.org/x/crypto\n  dependency-version: 0.53.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: golang.org/x/net\n  dependency-version: 0.56.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: golang.org/x/sync\n  dependency-version: 0.21.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: google.golang.org/api\n  dependency-version: 0.286.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.4\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/spf13/cobra\n  dependency-version: 1.10.2\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.4\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: golang.org/x/net\n  dependency-version: 0.56.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.4\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/moby/moby/client\n  dependency-version: 0.5.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.4\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\n\n* chore: run just deps after Dependabot update\n\n---------\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>\nCo-authored-by: github-actions[bot] <41898282+github-actions[bot]@users.noreply.github.com>\nCo-authored-by: julienrbrt <julien@rbrt.fr>",
          "timestamp": "2026-06-29T10:51:54Z",
          "tree_id": "2d3e6cd8c802ecb054a36cc52f42343f76b99892",
          "url": "https://github.com/evstack/ev-node/commit/a5ad63cce26a5846c524edc98ef0a26238d4b6c8"
        },
        "date": 1782731593659,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37381,
            "unit": "ns/op\t    4867 B/op\t      51 allocs/op",
            "extra": "32638 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37381,
            "unit": "ns/op",
            "extra": "32638 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4867,
            "unit": "B/op",
            "extra": "32638 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 51,
            "unit": "allocs/op",
            "extra": "32638 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 38054,
            "unit": "ns/op\t    5069 B/op\t      55 allocs/op",
            "extra": "32208 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 38054,
            "unit": "ns/op",
            "extra": "32208 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 5069,
            "unit": "B/op",
            "extra": "32208 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "32208 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 44568,
            "unit": "ns/op\t   10381 B/op\t      55 allocs/op",
            "extra": "27223 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 44568,
            "unit": "ns/op",
            "extra": "27223 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10381,
            "unit": "B/op",
            "extra": "27223 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "27223 times\n4 procs"
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
          "id": "daddf655ce327ec097d8d23f4fdc5228eaebeb2e",
          "message": "build(deps): Bump actions/setup-go from 6.4.0 to 6.5.0 (#3368)\n\nBumps [actions/setup-go](https://github.com/actions/setup-go) from 6.4.0 to 6.5.0.\n- [Release notes](https://github.com/actions/setup-go/releases)\n- [Commits](https://github.com/actions/setup-go/compare/v6.4.0...v6.5.0)\n\n---\nupdated-dependencies:\n- dependency-name: actions/setup-go\n  dependency-version: 6.5.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-06-30T10:36:27+02:00",
          "tree_id": "0db84599e6119d30a598a6fad704da22dae6cb96",
          "url": "https://github.com/evstack/ev-node/commit/daddf655ce327ec097d8d23f4fdc5228eaebeb2e"
        },
        "date": 1782809018795,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 37134,
            "unit": "ns/op\t    4859 B/op\t      51 allocs/op",
            "extra": "32961 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 37134,
            "unit": "ns/op",
            "extra": "32961 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4859,
            "unit": "B/op",
            "extra": "32961 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 51,
            "unit": "allocs/op",
            "extra": "32961 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 37568,
            "unit": "ns/op\t    5058 B/op\t      55 allocs/op",
            "extra": "32667 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 37568,
            "unit": "ns/op",
            "extra": "32667 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 5058,
            "unit": "B/op",
            "extra": "32667 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "32667 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 43920,
            "unit": "ns/op\t   10354 B/op\t      55 allocs/op",
            "extra": "28064 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 43920,
            "unit": "ns/op",
            "extra": "28064 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10354,
            "unit": "B/op",
            "extra": "28064 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "28064 times\n4 procs"
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
          "id": "999c53e97c61dacad192b195f256a3e728e811b5",
          "message": "build(deps): Bump actions/cache from 5 to 6 (#3370)\n\nBumps [actions/cache](https://github.com/actions/cache) from 5 to 6.\n- [Release notes](https://github.com/actions/cache/releases)\n- [Changelog](https://github.com/actions/cache/blob/main/RELEASES.md)\n- [Commits](https://github.com/actions/cache/compare/v5...v6)\n\n---\nupdated-dependencies:\n- dependency-name: actions/cache\n  dependency-version: '6'\n  dependency-type: direct:production\n  update-type: version-update:semver-major\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-06-30T10:36:45+02:00",
          "tree_id": "21c496a4bac9220b92b761f53d48d2d53f213f3f",
          "url": "https://github.com/evstack/ev-node/commit/999c53e97c61dacad192b195f256a3e728e811b5"
        },
        "date": 1782809034252,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 48104,
            "unit": "ns/op\t   10437 B/op\t      55 allocs/op",
            "extra": "25663 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 48104,
            "unit": "ns/op",
            "extra": "25663 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10437,
            "unit": "B/op",
            "extra": "25663 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "25663 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 40500,
            "unit": "ns/op\t    4930 B/op\t      51 allocs/op",
            "extra": "30151 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 40500,
            "unit": "ns/op",
            "extra": "30151 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4930,
            "unit": "B/op",
            "extra": "30151 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 51,
            "unit": "allocs/op",
            "extra": "30151 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 41248,
            "unit": "ns/op\t    5146 B/op\t      55 allocs/op",
            "extra": "29323 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 41248,
            "unit": "ns/op",
            "extra": "29323 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 5146,
            "unit": "B/op",
            "extra": "29323 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 55,
            "unit": "allocs/op",
            "extra": "29323 times\n4 procs"
          }
        ]
      }
    ]
  }
}