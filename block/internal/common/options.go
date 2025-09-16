package common

import (
    "context"
    "fmt"

    "github.com/evstack/ev-node/types"
)

// BlockOptions defines the options for creating block components
type BlockOptions struct {
    AggregatorNodeSignatureBytesProvider types.AggregatorNodeSignatureBytesProvider
    SyncNodeSignatureBytesProvider       types.SyncNodeSignatureBytesProvider
    ValidatorHasherProvider              types.ValidatorHasherProvider
    // SignedDataBytesProvider provides the bytes to verify a SignedData signature (DA blobs)
    // Defaults to using the binary-encoded Data payload.
    SignedDataBytesProvider              func(context.Context, *types.Data) ([]byte, error)
}

// DefaultBlockOptions returns the default block options
func DefaultBlockOptions() BlockOptions {
    return BlockOptions{
        AggregatorNodeSignatureBytesProvider: types.DefaultAggregatorNodeSignatureBytesProvider,
        SyncNodeSignatureBytesProvider:       types.DefaultSyncNodeSignatureBytesProvider,
        ValidatorHasherProvider:              types.DefaultValidatorHasherProvider,
        SignedDataBytesProvider: func(ctx context.Context, d *types.Data) ([]byte, error) {
            if d == nil {
                return nil, nil
            }
            return d.MarshalBinary()
        },
    }
}

// Validate validates the BlockOptions
func (opts *BlockOptions) Validate() error {
    if opts.AggregatorNodeSignatureBytesProvider == nil {
        return fmt.Errorf("aggregator node signature bytes provider cannot be nil")
    }

	if opts.SyncNodeSignatureBytesProvider == nil {
		return fmt.Errorf("sync node signature bytes provider cannot be nil")
	}

    if opts.ValidatorHasherProvider == nil {
        return fmt.Errorf("validator hasher provider cannot be nil")
    }

    if opts.SignedDataBytesProvider == nil {
        return fmt.Errorf("signed data bytes provider cannot be nil")
    }

    return nil
}
