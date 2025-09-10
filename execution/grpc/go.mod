module github.com/evstack/ev-node/execution/grpc

go 1.24.2

require (
	connectrpc.com/connect v1.18.1
	connectrpc.com/grpcreflect v1.3.0
	github.com/evstack/ev-node v0.0.0
	github.com/evstack/ev-node/core v0.0.0
	golang.org/x/net v0.43.0
	google.golang.org/protobuf v1.36.7
)

require golang.org/x/text v0.28.0 // indirect

replace (
	github.com/evstack/ev-node => ../../
	github.com/evstack/ev-node/core => ../../core
)
