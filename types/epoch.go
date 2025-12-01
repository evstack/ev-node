package types

// CalculateEpochNumber returns the deterministic epoch number for a given DA height.
// Epoch 1 starts at daStartHeight.
//
// Parameters:
//   - daHeight: The DA height to calculate the epoch for
//   - daStartHeight: The genesis DA start height
//   - daEpochSize: The number of DA blocks per epoch (0 means all blocks in epoch 1)
//
// Returns:
//   - Epoch number (0 if before daStartHeight, 1+ otherwise)
func CalculateEpochNumber(daHeight, daStartHeight, daEpochSize uint64) uint64 {
	if daHeight < daStartHeight {
		return 0
	}

	if daEpochSize == 0 {
		return 1
	}

	return ((daHeight - daStartHeight) / daEpochSize) + 1
}

// CalculateEpochBoundaries returns the start and end DA heights for the epoch
// containing the given DA height. The boundaries are inclusive.
//
// Parameters:
//   - daHeight: The DA height to calculate boundaries for
//   - daStartHeight: The genesis DA start height
//   - daEpochSize: The number of DA blocks per epoch (0 means single epoch)
//
// Returns:
//   - start: The first DA height in the epoch (inclusive)
//   - end: The last DA height in the epoch (inclusive)
func CalculateEpochBoundaries(daHeight, daStartHeight, daEpochSize uint64) (start, end, epochNum uint64) {
	epochNum = CalculateEpochNumber(daHeight, daStartHeight, daEpochSize)

	if daEpochSize == 0 {
		return daStartHeight, daStartHeight, epochNum
	}

	if daHeight < daStartHeight {
		return daStartHeight, daStartHeight + daEpochSize - 1, epochNum
	}

	start = daStartHeight + (epochNum-1)*daEpochSize
	end = daStartHeight + epochNum*daEpochSize - 1

	return start, end, epochNum
}
