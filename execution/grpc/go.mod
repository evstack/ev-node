module github.com/evstack/ev-node/execution/grpc

go 1.25.7

replace (
	github.com/evstack/ev-node => ../../
	github.com/evstack/ev-node/core => ../../core
)

require (
	connectrpc.com/connect v1.19.2
	connectrpc.com/grpcreflect v1.3.0
	github.com/evstack/ev-node v1.1.0
	github.com/evstack/ev-node/core v1.0.0
	golang.org/x/net v0.53.0
	google.golang.org/protobuf v1.36.11
)

require golang.org/x/text v0.36.0 // indirect
