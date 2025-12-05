package common

// TODO(@julienrbrt): technically we may need to check for block gas as well

const (
	// AbsoluteMaxBlobSize is the absolute maximum size for a single blob (DA layer limit).
	// Blobs exceeding this size are invalid and should be rejected permanently.
	AbsoluteMaxBlobSize = 2 * 1024 * 1024 // 2MB
)

// ValidateBlobSize checks if a single blob exceeds the absolute maximum allowed size.
// This checks against the DA layer limit, not the per-batch limit.
// Returns true if the blob is within the absolute size limit, false otherwise.
func ValidateBlobSize(blob []byte) bool {
	return uint64(len(blob)) <= AbsoluteMaxBlobSize
}
