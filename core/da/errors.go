package da

import datypes "github.com/evstack/ev-node/pkg/da/types"

// Deprecated: use pkg/da/types equivalents.
var (
	ErrBlobNotFound               = datypes.ErrBlobNotFound
	ErrBlobSizeOverLimit          = datypes.ErrBlobSizeOverLimit
	ErrTxTimedOut                 = datypes.ErrTxTimedOut
	ErrTxAlreadyInMempool         = datypes.ErrTxAlreadyInMempool
	ErrTxIncorrectAccountSequence = datypes.ErrTxIncorrectAccountSequence
	ErrContextDeadline            = datypes.ErrContextDeadline
	ErrHeightFromFuture           = datypes.ErrHeightFromFuture
	ErrContextCanceled            = datypes.ErrContextCanceled
)
