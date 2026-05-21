package syncing

import "fmt"

// ValidationFault identifies which side of a (header, data) pair is responsible
// for a block validation failure. Callers use it to decide what to evict from
// caches so a legitimate counterpart is not discarded together with the bad one.
type ValidationFault int

const (
	// FaultHeader marks the header as invalid on its own (signature, proposer,
	// chain id, height, sequence). The data attached to the event may still be
	// legitimate and pair with a different valid header.
	FaultHeader ValidationFault = iota

	// FaultData marks the data as the suspect. It is used when the header
	// passed all header-only checks but the data does not match the header
	// (mismatched DataHash or metadata).
	FaultData
)

// BlockValidationError wraps the underlying validation error with a fault tag.
type BlockValidationError struct {
	Fault ValidationFault
	Err   error
}

func (e *BlockValidationError) Error() string {
	return fmt.Sprintf("block validation failed (%s): %v", e.Fault, e.Err)
}

func (e *BlockValidationError) Unwrap() error { return e.Err }

func (f ValidationFault) String() string {
	switch f {
	case FaultHeader:
		return "header"
	case FaultData:
		return "data"
	default:
		return "unknown"
	}
}
