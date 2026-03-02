# Run 'n' nodes (default: 1). Usage: just run-n 3
[group('run')]
run-n nodes="1": build build-da
    @go run -tags=run scripts/run.go --nodes={{ nodes }}

# Run EVM nodes (one sequencer and one full node)
[group('run')]
run-evm-nodes nodes="1": build-da build-evm
    @echo "Starting EVM nodes..."
    @go run -tags=run_evm scripts/run-evm-nodes.go --nodes={{ nodes }}
