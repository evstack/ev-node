package common

// TODO(@julienrbrt): technically we may need to check for block gas as well

const (
	// AbsoluteMaxBlobSize is the absolute maximum size for a single blob (DA layer limit).
	// Blobs exceeding this size are invalid and should be rejected permanently.
	AbsoluteMaxBlobSize = 2 * 1024 * 1024 // 2MB
)
