# db-bench

Local BadgerDB benchmark for ev-node write patterns (append + overwrite).

## Usage

Run the tuned defaults:

```sh
go run ./tools/db-bench -bytes 536870912 -value-size 4096 -batch-size 1000 -overwrite-ratio 0.1
```

Compare against Badger defaults:

```sh
go run ./tools/db-bench -profile all -bytes 1073741824 -value-size 4096 -batch-size 1000 -overwrite-ratio 0.1
```

Notes:

- `-bytes` is the total data volume; the tool rounds down to full `-value-size` writes.
- `-profile all` runs `evnode` and `default` in separate subdirectories under a temp base dir.
