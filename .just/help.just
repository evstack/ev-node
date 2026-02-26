# Help

# List available recipes when running `just` with no args
[group('help')]
default:
    @just --list --unsorted
