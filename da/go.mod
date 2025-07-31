module github.com/evstack/ev-node/da

go 1.24.1

replace github.com/evstack/ev-node/core => ../core

replace github.com/evstack/ev-node/pkg/logging => ../pkg/logging

require (
	github.com/celestiaorg/go-square/v2 v2.2.0
	github.com/evstack/ev-node/core v0.0.0-20250312114929-104787ba1a4c
	github.com/evstack/ev-node/pkg/logging v0.0.0-00010101000000-000000000000
	github.com/filecoin-project/go-jsonrpc v0.7.1
	github.com/ipfs/go-log/v2 v2.5.1
	github.com/stretchr/testify v1.10.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/golang/groupcache v0.0.0-20190702054246-869f871628b6 // indirect
	github.com/google/uuid v1.1.1 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	go.opencensus.io v0.22.3 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/exp v0.0.0-20231206192017-f3f8817b8deb // indirect
	golang.org/x/sys v0.0.0-20210630005230-0f9fa26af87c // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
