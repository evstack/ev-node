window.BENCHMARK_DATA = {
  "lastUpdate": 1779277655946,
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
          "id": "27eeb486a21a44030226d5e31472171dd83209b1",
          "message": "build(deps): Bump uuid from 11.1.0 to 14.0.0 in /docs in the npm_and_yarn group across 1 directory (#3321)\n\nbuild(deps): Bump uuid\n\nBumps the npm_and_yarn group with 1 update in the /docs directory: [uuid](https://github.com/uuidjs/uuid).\n\n\nUpdates `uuid` from 11.1.0 to 14.0.0\n- [Release notes](https://github.com/uuidjs/uuid/releases)\n- [Changelog](https://github.com/uuidjs/uuid/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/uuidjs/uuid/compare/v11.1.0...v14.0.0)\n\n---\nupdated-dependencies:\n- dependency-name: uuid\n  dependency-version: 14.0.0\n  dependency-type: indirect\n  dependency-group: npm_and_yarn\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-05-12T10:37:43+02:00",
          "tree_id": "c2deadee909f25e0b4776e61e8c6b7319e79107e",
          "url": "https://github.com/evstack/ev-node/commit/27eeb486a21a44030226d5e31472171dd83209b1"
        },
        "date": 1778575567842,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 913727039,
            "unit": "ns/op\t32946924 B/op\t  186129 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 913727039,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 32946924,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 186129,
            "unit": "allocs/op",
            "extra": "2 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "cuoguojida@outlook.com",
            "name": "cuoguojida",
            "username": "cuoguojida"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "a2dae255e6252340f9676231ca0a52456808f612",
          "message": "chore: fix some comments to improve readability (#3325)\n\nSigned-off-by: cuoguojida <cuoguojida@outlook.com>",
          "timestamp": "2026-05-18T08:55:23+02:00",
          "tree_id": "ba5ae484f3b4608a6356b7da78639229b22fa104",
          "url": "https://github.com/evstack/ev-node/commit/a2dae255e6252340f9676231ca0a52456808f612"
        },
        "date": 1779087539255,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 910577446,
            "unit": "ns/op\t31848864 B/op\t  176240 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 910577446,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 31848864,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 176240,
            "unit": "allocs/op",
            "extra": "2 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "github.qpeyb@simplelogin.fr",
            "name": "Cian Hatton",
            "username": "chatton"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "890438743e3f5c74e4563fbd229759d8e449bdc9",
          "message": "chore: using v0.20.0 tag of tastora and removing temporary duplication (#3326)\n\n* refactor: use tastora spamoor and victoriatraces implementations\n\nReplace local spamoor_node.go and victoriatraces_node.go with tastora's\nspamoor.NewNodeBuilder and victoriatraces.New. Fixes macOS compatibility\nby replacing 0.0.0.0 with 127.0.0.1 for external endpoints and using\nUTC timestamps in trace query URLs to avoid broken timezone offsets.\n\n* build(deps): bump tastora to v0.20.0 and remove local replace\n\nRemove the local filesystem replace directive for tastora in\ntest/e2e/go.mod now that v0.20.0 is published.\n\n* build: bump go directive to 1.25.8\n\nRequired by tastora v0.20.0. Aligns root and execution/evm\nwith the other modules already at 1.25.8.",
          "timestamp": "2026-05-18T12:03:09+02:00",
          "tree_id": "0820ca693f7423abc20da38da593de2ba6cb9557",
          "url": "https://github.com/evstack/ev-node/commit/890438743e3f5c74e4563fbd229759d8e449bdc9"
        },
        "date": 1779098848878,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 904754234,
            "unit": "ns/op\t31828520 B/op\t  175940 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 904754234,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 31828520,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 175940,
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
          "id": "53c5669779cf534047380a73c91ef90c97fee142",
          "message": "build(deps): Bump the all-go group across 6 directories with 7 updates (#3327)\n\n* build(deps): Bump the all-go group across 6 directories with 7 updates\n\nBumps the all-go group with 6 updates in the / directory:\n\n| Package | From | To |\n| --- | --- | --- |\n| [cloud.google.com/go/kms](https://github.com/googleapis/google-cloud-go) | `1.30.0` | `1.31.0` |\n| [github.com/libp2p/go-libp2p-kad-dht](https://github.com/libp2p/go-libp2p-kad-dht) | `0.39.1` | `0.39.2` |\n| [golang.org/x/crypto](https://github.com/golang/crypto) | `0.50.0` | `0.51.0` |\n| [golang.org/x/net](https://github.com/golang/net) | `0.53.0` | `0.54.0` |\n| [google.golang.org/api](https://github.com/googleapis/google-api-go-client) | `0.277.0` | `0.279.0` |\n| [google.golang.org/grpc](https://github.com/grpc/grpc-go) | `1.81.0` | `1.81.1` |\n\nBumps the all-go group with 1 update in the /apps/evm directory: [github.com/ethereum/go-ethereum](https://github.com/ethereum/go-ethereum).\nBumps the all-go group with 1 update in the /execution/evm directory: [github.com/ethereum/go-ethereum](https://github.com/ethereum/go-ethereum).\nBumps the all-go group with 1 update in the /execution/grpc directory: [golang.org/x/net](https://github.com/golang/net).\nBumps the all-go group with 1 update in the /test/docker-e2e directory: [github.com/ethereum/go-ethereum](https://github.com/ethereum/go-ethereum).\nBumps the all-go group with 1 update in the /test/e2e directory: [github.com/ethereum/go-ethereum](https://github.com/ethereum/go-ethereum).\n\n\nUpdates `cloud.google.com/go/kms` from 1.30.0 to 1.31.0\n- [Release notes](https://github.com/googleapis/google-cloud-go/releases)\n- [Changelog](https://github.com/googleapis/google-cloud-go/blob/main/documentai/CHANGES.md)\n- [Commits](https://github.com/googleapis/google-cloud-go/compare/kms/v1.30.0...dlp/v1.31.0)\n\nUpdates `github.com/libp2p/go-libp2p-kad-dht` from 0.39.1 to 0.39.2\n- [Release notes](https://github.com/libp2p/go-libp2p-kad-dht/releases)\n- [Commits](https://github.com/libp2p/go-libp2p-kad-dht/compare/v0.39.1...v0.39.2)\n\nUpdates `golang.org/x/crypto` from 0.50.0 to 0.51.0\n- [Commits](https://github.com/golang/crypto/compare/v0.50.0...v0.51.0)\n\nUpdates `golang.org/x/net` from 0.53.0 to 0.54.0\n- [Commits](https://github.com/golang/net/compare/v0.53.0...v0.54.0)\n\nUpdates `google.golang.org/api` from 0.277.0 to 0.279.0\n- [Release notes](https://github.com/googleapis/google-api-go-client/releases)\n- [Changelog](https://github.com/googleapis/google-api-go-client/blob/main/CHANGES.md)\n- [Commits](https://github.com/googleapis/google-api-go-client/compare/v0.277.0...v0.279.0)\n\nUpdates `google.golang.org/grpc` from 1.81.0 to 1.81.1\n- [Release notes](https://github.com/grpc/grpc-go/releases)\n- [Commits](https://github.com/grpc/grpc-go/compare/v1.81.0...v1.81.1)\n\nUpdates `golang.org/x/net` from 0.53.0 to 0.54.0\n- [Commits](https://github.com/golang/net/compare/v0.53.0...v0.54.0)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.2 to 1.17.3\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.2...v1.17.3)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.2 to 1.17.3\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.2...v1.17.3)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.2 to 1.17.3\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.2...v1.17.3)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.2 to 1.17.3\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.2...v1.17.3)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.2 to 1.17.3\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.2...v1.17.3)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.2 to 1.17.3\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.2...v1.17.3)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.2 to 1.17.3\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.2...v1.17.3)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.2 to 1.17.3\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.2...v1.17.3)\n\nUpdates `golang.org/x/net` from 0.53.0 to 0.54.0\n- [Commits](https://github.com/golang/net/compare/v0.53.0...v0.54.0)\n\nUpdates `golang.org/x/net` from 0.53.0 to 0.54.0\n- [Commits](https://github.com/golang/net/compare/v0.53.0...v0.54.0)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.2 to 1.17.3\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.2...v1.17.3)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.2 to 1.17.3\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.2...v1.17.3)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.2 to 1.17.3\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.2...v1.17.3)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.2 to 1.17.3\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.2...v1.17.3)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.2 to 1.17.3\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.2...v1.17.3)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.2 to 1.17.3\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.2...v1.17.3)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.2 to 1.17.3\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.2...v1.17.3)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.2 to 1.17.3\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.2...v1.17.3)\n\n---\nupdated-dependencies:\n- dependency-name: cloud.google.com/go/kms\n  dependency-version: 1.31.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/libp2p/go-libp2p-kad-dht\n  dependency-version: 0.39.2\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: golang.org/x/crypto\n  dependency-version: 0.51.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: golang.org/x/net\n  dependency-version: 0.54.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: google.golang.org/api\n  dependency-version: 0.279.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: google.golang.org/grpc\n  dependency-version: 1.81.1\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: golang.org/x/net\n  dependency-version: 0.54.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: golang.org/x/net\n  dependency-version: 0.54.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: golang.org/x/net\n  dependency-version: 0.54.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\n\n* chore: run just deps after Dependabot update\n\n* trigger ci\n\n---------\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>\nCo-authored-by: github-actions[bot] <41898282+github-actions[bot]@users.noreply.github.com>\nCo-authored-by: Julien Robert <julien@rbrt.fr>",
          "timestamp": "2026-05-20T13:43:54+02:00",
          "tree_id": "b3733932d50ef16abfb06d38efbcc97250a36337",
          "url": "https://github.com/evstack/ev-node/commit/53c5669779cf534047380a73c91ef90c97fee142"
        },
        "date": 1779277650170,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEvmContractRoundtrip",
            "value": 913616166,
            "unit": "ns/op\t30530060 B/op\t  161658 allocs/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - ns/op",
            "value": 913616166,
            "unit": "ns/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - B/op",
            "value": 30530060,
            "unit": "B/op",
            "extra": "2 times\n4 procs"
          },
          {
            "name": "BenchmarkEvmContractRoundtrip - allocs/op",
            "value": 161658,
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
          "id": "27eeb486a21a44030226d5e31472171dd83209b1",
          "message": "build(deps): Bump uuid from 11.1.0 to 14.0.0 in /docs in the npm_and_yarn group across 1 directory (#3321)\n\nbuild(deps): Bump uuid\n\nBumps the npm_and_yarn group with 1 update in the /docs directory: [uuid](https://github.com/uuidjs/uuid).\n\n\nUpdates `uuid` from 11.1.0 to 14.0.0\n- [Release notes](https://github.com/uuidjs/uuid/releases)\n- [Changelog](https://github.com/uuidjs/uuid/blob/main/CHANGELOG.md)\n- [Commits](https://github.com/uuidjs/uuid/compare/v11.1.0...v14.0.0)\n\n---\nupdated-dependencies:\n- dependency-name: uuid\n  dependency-version: 14.0.0\n  dependency-type: indirect\n  dependency-group: npm_and_yarn\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>",
          "timestamp": "2026-05-12T10:37:43+02:00",
          "tree_id": "c2deadee909f25e0b4776e61e8c6b7319e79107e",
          "url": "https://github.com/evstack/ev-node/commit/27eeb486a21a44030226d5e31472171dd83209b1"
        },
        "date": 1778575574311,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 36158,
            "unit": "ns/op\t    4724 B/op\t      50 allocs/op",
            "extra": "34096 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 36158,
            "unit": "ns/op",
            "extra": "34096 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4724,
            "unit": "B/op",
            "extra": "34096 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 50,
            "unit": "allocs/op",
            "extra": "34096 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 36421,
            "unit": "ns/op\t    4927 B/op\t      54 allocs/op",
            "extra": "33602 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 36421,
            "unit": "ns/op",
            "extra": "33602 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 4927,
            "unit": "B/op",
            "extra": "33602 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "33602 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 42440,
            "unit": "ns/op\t   10222 B/op\t      54 allocs/op",
            "extra": "28754 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 42440,
            "unit": "ns/op",
            "extra": "28754 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10222,
            "unit": "B/op",
            "extra": "28754 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "28754 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "cuoguojida@outlook.com",
            "name": "cuoguojida",
            "username": "cuoguojida"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "a2dae255e6252340f9676231ca0a52456808f612",
          "message": "chore: fix some comments to improve readability (#3325)\n\nSigned-off-by: cuoguojida <cuoguojida@outlook.com>",
          "timestamp": "2026-05-18T08:55:23+02:00",
          "tree_id": "ba5ae484f3b4608a6356b7da78639229b22fa104",
          "url": "https://github.com/evstack/ev-node/commit/a2dae255e6252340f9676231ca0a52456808f612"
        },
        "date": 1779087544315,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 38435,
            "unit": "ns/op\t    4768 B/op\t      50 allocs/op",
            "extra": "32162 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 38435,
            "unit": "ns/op",
            "extra": "32162 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4768,
            "unit": "B/op",
            "extra": "32162 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 50,
            "unit": "allocs/op",
            "extra": "32162 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 38874,
            "unit": "ns/op\t    4978 B/op\t      54 allocs/op",
            "extra": "31465 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 38874,
            "unit": "ns/op",
            "extra": "31465 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 4978,
            "unit": "B/op",
            "extra": "31465 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "31465 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 44771,
            "unit": "ns/op\t   10268 B/op\t      54 allocs/op",
            "extra": "27326 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 44771,
            "unit": "ns/op",
            "extra": "27326 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10268,
            "unit": "B/op",
            "extra": "27326 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "27326 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "github.qpeyb@simplelogin.fr",
            "name": "Cian Hatton",
            "username": "chatton"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "890438743e3f5c74e4563fbd229759d8e449bdc9",
          "message": "chore: using v0.20.0 tag of tastora and removing temporary duplication (#3326)\n\n* refactor: use tastora spamoor and victoriatraces implementations\n\nReplace local spamoor_node.go and victoriatraces_node.go with tastora's\nspamoor.NewNodeBuilder and victoriatraces.New. Fixes macOS compatibility\nby replacing 0.0.0.0 with 127.0.0.1 for external endpoints and using\nUTC timestamps in trace query URLs to avoid broken timezone offsets.\n\n* build(deps): bump tastora to v0.20.0 and remove local replace\n\nRemove the local filesystem replace directive for tastora in\ntest/e2e/go.mod now that v0.20.0 is published.\n\n* build: bump go directive to 1.25.8\n\nRequired by tastora v0.20.0. Aligns root and execution/evm\nwith the other modules already at 1.25.8.",
          "timestamp": "2026-05-18T12:03:09+02:00",
          "tree_id": "0820ca693f7423abc20da38da593de2ba6cb9557",
          "url": "https://github.com/evstack/ev-node/commit/890438743e3f5c74e4563fbd229759d8e449bdc9"
        },
        "date": 1779098856300,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 39482,
            "unit": "ns/op\t    4802 B/op\t      50 allocs/op",
            "extra": "30817 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 39482,
            "unit": "ns/op",
            "extra": "30817 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4802,
            "unit": "B/op",
            "extra": "30817 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 50,
            "unit": "allocs/op",
            "extra": "30817 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 40255,
            "unit": "ns/op\t    5007 B/op\t      54 allocs/op",
            "extra": "30358 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 40255,
            "unit": "ns/op",
            "extra": "30358 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 5007,
            "unit": "B/op",
            "extra": "30358 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "30358 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 46400,
            "unit": "ns/op\t   10305 B/op\t      54 allocs/op",
            "extra": "26276 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 46400,
            "unit": "ns/op",
            "extra": "26276 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10305,
            "unit": "B/op",
            "extra": "26276 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "26276 times\n4 procs"
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
          "id": "53c5669779cf534047380a73c91ef90c97fee142",
          "message": "build(deps): Bump the all-go group across 6 directories with 7 updates (#3327)\n\n* build(deps): Bump the all-go group across 6 directories with 7 updates\n\nBumps the all-go group with 6 updates in the / directory:\n\n| Package | From | To |\n| --- | --- | --- |\n| [cloud.google.com/go/kms](https://github.com/googleapis/google-cloud-go) | `1.30.0` | `1.31.0` |\n| [github.com/libp2p/go-libp2p-kad-dht](https://github.com/libp2p/go-libp2p-kad-dht) | `0.39.1` | `0.39.2` |\n| [golang.org/x/crypto](https://github.com/golang/crypto) | `0.50.0` | `0.51.0` |\n| [golang.org/x/net](https://github.com/golang/net) | `0.53.0` | `0.54.0` |\n| [google.golang.org/api](https://github.com/googleapis/google-api-go-client) | `0.277.0` | `0.279.0` |\n| [google.golang.org/grpc](https://github.com/grpc/grpc-go) | `1.81.0` | `1.81.1` |\n\nBumps the all-go group with 1 update in the /apps/evm directory: [github.com/ethereum/go-ethereum](https://github.com/ethereum/go-ethereum).\nBumps the all-go group with 1 update in the /execution/evm directory: [github.com/ethereum/go-ethereum](https://github.com/ethereum/go-ethereum).\nBumps the all-go group with 1 update in the /execution/grpc directory: [golang.org/x/net](https://github.com/golang/net).\nBumps the all-go group with 1 update in the /test/docker-e2e directory: [github.com/ethereum/go-ethereum](https://github.com/ethereum/go-ethereum).\nBumps the all-go group with 1 update in the /test/e2e directory: [github.com/ethereum/go-ethereum](https://github.com/ethereum/go-ethereum).\n\n\nUpdates `cloud.google.com/go/kms` from 1.30.0 to 1.31.0\n- [Release notes](https://github.com/googleapis/google-cloud-go/releases)\n- [Changelog](https://github.com/googleapis/google-cloud-go/blob/main/documentai/CHANGES.md)\n- [Commits](https://github.com/googleapis/google-cloud-go/compare/kms/v1.30.0...dlp/v1.31.0)\n\nUpdates `github.com/libp2p/go-libp2p-kad-dht` from 0.39.1 to 0.39.2\n- [Release notes](https://github.com/libp2p/go-libp2p-kad-dht/releases)\n- [Commits](https://github.com/libp2p/go-libp2p-kad-dht/compare/v0.39.1...v0.39.2)\n\nUpdates `golang.org/x/crypto` from 0.50.0 to 0.51.0\n- [Commits](https://github.com/golang/crypto/compare/v0.50.0...v0.51.0)\n\nUpdates `golang.org/x/net` from 0.53.0 to 0.54.0\n- [Commits](https://github.com/golang/net/compare/v0.53.0...v0.54.0)\n\nUpdates `google.golang.org/api` from 0.277.0 to 0.279.0\n- [Release notes](https://github.com/googleapis/google-api-go-client/releases)\n- [Changelog](https://github.com/googleapis/google-api-go-client/blob/main/CHANGES.md)\n- [Commits](https://github.com/googleapis/google-api-go-client/compare/v0.277.0...v0.279.0)\n\nUpdates `google.golang.org/grpc` from 1.81.0 to 1.81.1\n- [Release notes](https://github.com/grpc/grpc-go/releases)\n- [Commits](https://github.com/grpc/grpc-go/compare/v1.81.0...v1.81.1)\n\nUpdates `golang.org/x/net` from 0.53.0 to 0.54.0\n- [Commits](https://github.com/golang/net/compare/v0.53.0...v0.54.0)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.2 to 1.17.3\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.2...v1.17.3)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.2 to 1.17.3\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.2...v1.17.3)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.2 to 1.17.3\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.2...v1.17.3)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.2 to 1.17.3\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.2...v1.17.3)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.2 to 1.17.3\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.2...v1.17.3)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.2 to 1.17.3\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.2...v1.17.3)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.2 to 1.17.3\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.2...v1.17.3)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.2 to 1.17.3\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.2...v1.17.3)\n\nUpdates `golang.org/x/net` from 0.53.0 to 0.54.0\n- [Commits](https://github.com/golang/net/compare/v0.53.0...v0.54.0)\n\nUpdates `golang.org/x/net` from 0.53.0 to 0.54.0\n- [Commits](https://github.com/golang/net/compare/v0.53.0...v0.54.0)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.2 to 1.17.3\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.2...v1.17.3)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.2 to 1.17.3\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.2...v1.17.3)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.2 to 1.17.3\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.2...v1.17.3)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.2 to 1.17.3\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.2...v1.17.3)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.2 to 1.17.3\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.2...v1.17.3)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.2 to 1.17.3\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.2...v1.17.3)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.2 to 1.17.3\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.2...v1.17.3)\n\nUpdates `github.com/ethereum/go-ethereum` from 1.17.2 to 1.17.3\n- [Release notes](https://github.com/ethereum/go-ethereum/releases)\n- [Commits](https://github.com/ethereum/go-ethereum/compare/v1.17.2...v1.17.3)\n\n---\nupdated-dependencies:\n- dependency-name: cloud.google.com/go/kms\n  dependency-version: 1.31.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/libp2p/go-libp2p-kad-dht\n  dependency-version: 0.39.2\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: golang.org/x/crypto\n  dependency-version: 0.51.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: golang.org/x/net\n  dependency-version: 0.54.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: google.golang.org/api\n  dependency-version: 0.279.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: google.golang.org/grpc\n  dependency-version: 1.81.1\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: golang.org/x/net\n  dependency-version: 0.54.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: golang.org/x/net\n  dependency-version: 0.54.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: golang.org/x/net\n  dependency-version: 0.54.0\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n- dependency-name: github.com/ethereum/go-ethereum\n  dependency-version: 1.17.3\n  dependency-type: direct:production\n  update-type: version-update:semver-patch\n  dependency-group: all-go\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\n\n* chore: run just deps after Dependabot update\n\n* trigger ci\n\n---------\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>\nCo-authored-by: github-actions[bot] <41898282+github-actions[bot]@users.noreply.github.com>\nCo-authored-by: Julien Robert <julien@rbrt.fr>",
          "timestamp": "2026-05-20T13:43:54+02:00",
          "tree_id": "b3733932d50ef16abfb06d38efbcc97250a36337",
          "url": "https://github.com/evstack/ev-node/commit/53c5669779cf534047380a73c91ef90c97fee142"
        },
        "date": 1779277655444,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkProduceBlock/empty_batch",
            "value": 39845,
            "unit": "ns/op\t    4803 B/op\t      50 allocs/op",
            "extra": "30795 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - ns/op",
            "value": 39845,
            "unit": "ns/op",
            "extra": "30795 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - B/op",
            "value": 4803,
            "unit": "B/op",
            "extra": "30795 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/empty_batch - allocs/op",
            "value": 50,
            "unit": "allocs/op",
            "extra": "30795 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx",
            "value": 40337,
            "unit": "ns/op\t    5018 B/op\t      54 allocs/op",
            "extra": "29964 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - ns/op",
            "value": 40337,
            "unit": "ns/op",
            "extra": "29964 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - B/op",
            "value": 5018,
            "unit": "B/op",
            "extra": "29964 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/single_tx - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "29964 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs",
            "value": 47364,
            "unit": "ns/op\t   10320 B/op\t      54 allocs/op",
            "extra": "25874 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - ns/op",
            "value": 47364,
            "unit": "ns/op",
            "extra": "25874 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - B/op",
            "value": 10320,
            "unit": "B/op",
            "extra": "25874 times\n4 procs"
          },
          {
            "name": "BenchmarkProduceBlock/100_txs - allocs/op",
            "value": 54,
            "unit": "allocs/op",
            "extra": "25874 times\n4 procs"
          }
        ]
      }
    ]
  }
}