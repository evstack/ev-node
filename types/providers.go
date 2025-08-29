package types

import "context"

// AggregatorNodeSignatureBytesProvider defines the function type for providing a signature payload to sign.
type AggregatorNodeSignatureBytesProvider func(*Header) ([]byte, error)

// DefaultAggregatorNodeSignatureBytesProvider is the default implementation of AggregatorNodeSignatureBytesProvider.
func DefaultAggregatorNodeSignatureBytesProvider(header *Header) ([]byte, error) {
	return header.MarshalBinary()
}

// SyncNodeSignatureBytesProvider defines the function type for providing a signature payload to be verified by syncing nodes.
type SyncNodeSignatureBytesProvider func(context.Context, *Header, *Data) ([]byte, error)

// DefaultSyncNodeSignatureBytesProvider is the default implementation of SyncNodeSignatureBytesProvider.
func DefaultSyncNodeSignatureBytesProvider(_ context.Context, header *Header, _ *Data) ([]byte, error) {
	return DefaultAggregatorNodeSignatureBytesProvider(header)
}
