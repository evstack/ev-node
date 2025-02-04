# Rollkit

Rollkit is the first sovereign rollup framework. For more in-depth information about Rollkit, please visit our [website][docs].

<!-- markdownlint-disable MD013 -->
[![build-and-test](https://github.com/rollkit/rollkit/actions/workflows/test.yml/badge.svg)](https://github.com/rollkit/rollkit/actions/workflows/test.yml)
[![golangci-lint](https://github.com/rollkit/rollkit/actions/workflows/lint.yml/badge.svg)](https://github.com/rollkit/rollkit/actions/workflows/lint.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/rollkit/rollkit)](https://goreportcard.com/report/github.com/rollkit/rollkit)
[![codecov](https://codecov.io/gh/rollkit/rollkit/branch/main/graph/badge.svg?token=CWGA4RLDS9)](https://codecov.io/gh/rollkit/rollkit)
[![GoDoc](https://godoc.org/github.com/rollkit/rollkit?status.svg)](https://godoc.org/github.com/rollkit/rollkit)
<!-- markdownlint-enable MD013 -->

## Using Rollkit

### Rollkit CLI

The easiest way to use rollkit is via the rollkit CLI.

Requires Go version >= 1.22.

A cli tool that allows you to run different kinds of nodes for a rollkit network
while also helping you generate the required configuration files

#### Install

To install `rollkit`, simply run the following command at the root of the
rollkit repo

```bash
make install
```

The latest Rollkit is now installed. You can verify the installation by running:

```bash
rollkit version
```

#### Quick Start

You can spin up a local rollkit network with the following command:

```bash
rollkit start
```

Explore the CLI documentation [here](./cmd/rollkit/docs/rollkit.md)

## Building with Rollkit

Rollkit is the first sovereign rollup framework that allows you to launch
a sovereign, customizable blockchain as easily as a smart contract.

Check out our tutorials on our [website][docs].

## Contributing

We welcome your contributions! Everyone is welcome to contribute, whether it's
in the form of code, documentation, bug reports, feature
requests, or anything else.

If you're looking for issues to work on, try looking at the
[good first issue list](https://github.com/rollkit/rollkit/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22).
Issues with this tag are suitable for a new external contributor and is a great
way to find something you can help with!

See
[the contributing guide](https://github.com/rollkit/rollkit/blob/main/CONTRIBUTING.md)
for more details.

Please join our
[Community Discord](https://discord.com/invite/YsnTPcSfWQ)
to ask questions, discuss your ideas, and connect with other contributors.

### Helpful commands

```sh
# Run unit tests
make test

# Generate protobuf files (requires Docker)
make proto-gen

# Run linters (requires golangci-lint, markdownlint, hadolint, and yamllint)
make lint

# Lint protobuf files (requires Docker and buf)
make proto-lint

```

### Tools

1. Install [golangci-lint](https://golangci-lint.run/welcome/install/)
1. Install [markdownlint](https://github.com/DavidAnson/markdownlint)
1. Install [hadolint](https://github.com/hadolint/hadolint)
1. Install [yamllint](https://yamllint.readthedocs.io/en/stable/quickstart.html)

## Dependency graph

To see our progress and a possible future of Rollkit visit our [Dependency
Graph](https://github.com/rollkit/rollkit/blob/main/specs/src/specs/rollkit-dependency-graph.md).

## Audits

| Date | Auditor | Version | Report |
|---|---|---|---|
| 2024/01/12 | [Informal Systems](https://informal.systems/) | [eccdd...bcb9d](https://github.com/rollkit/rollkit/commit/eccdd0f1793a5ac532011ef4d896de9e0d8bcb9d) | [informal-systems.pdf](specs/audit/informal-systems.pdf) |
| 2024/01/10 | [Binary Builders](https://binary.builders/)   | [eccdd...bcb9d](https://github.com/rollkit/rollkit/commit/eccdd0f1793a5ac532011ef4d896de9e0d8bcb9d) | [binary-builders.pdf](specs/audit/binary-builders.pdf)   |

[docs]: https://rollkit.dev
