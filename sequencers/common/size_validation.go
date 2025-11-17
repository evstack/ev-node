package common

// TODO: technically we need to check for block gas as well

// ValidateBlobSize checks if a single blob exceeds the maximum allowed size.
// Returns true if the blob is within the size limit, false otherwise.
func ValidateBlobSize(blob []byte, maxBytes uint64) bool {
	return uint64(len(blob)) <= maxBytes
}

// WouldExceedCumulativeSize checks if adding a blob would exceed the cumulative size limit.
// Returns true if adding the blob would exceed the limit, false otherwise.
func WouldExceedCumulativeSize(currentSize int, blobSize int, maxBytes uint64) bool {
	return uint64(currentSize)+uint64(blobSize) > maxBytes
}

// GetBlobSize returns the size of a blob in bytes.
func GetBlobSize(blob []byte) int {
	return len(blob)
}
