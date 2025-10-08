# Cache Analyzer

> The cache analyzer only works currently for sync nodes.

This small program is designed to analyze the cache of a sync node.
It is useful to debug when the sync node is downloading from DA but not advancing.
This usually means the `DA_START_HEIGHT` is too late. This tool allows clearly to identify the first height fetched from DA.

## Installation

Install using `go install`:

```bash
go install github.com/evstack/ev-node/tools/cache-analyzer@main
```

After installation, the `cache-analyzer` binary will be available in your `$GOPATH/bin` directory.

## Usage

```bash
cache-analyzer -data-dir ~/.appd/data/ -summary
cache-analyzer -data-dir ~/.appd/data/ -limit 50
```
