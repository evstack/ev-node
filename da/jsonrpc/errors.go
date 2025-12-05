package jsonrpc

import (
	"github.com/filecoin-project/go-jsonrpc"

	datypes "github.com/evstack/ev-node/pkg/da/types"
)

// getKnownErrorsMapping returns a mapping of known error codes to their corresponding error types.
func getKnownErrorsMapping() jsonrpc.Errors {
	errs := jsonrpc.NewErrors()
	errs.Register(jsonrpc.ErrorCode(datypes.StatusNotFound), &datypes.ErrBlobNotFound)
	errs.Register(jsonrpc.ErrorCode(datypes.StatusTooBig), &datypes.ErrBlobSizeOverLimit)
	errs.Register(jsonrpc.ErrorCode(datypes.StatusContextDeadline), &datypes.ErrTxTimedOut)
	errs.Register(jsonrpc.ErrorCode(datypes.StatusAlreadyInMempool), &datypes.ErrTxAlreadyInMempool)
	errs.Register(jsonrpc.ErrorCode(datypes.StatusIncorrectAccountSequence), &datypes.ErrTxIncorrectAccountSequence)
	errs.Register(jsonrpc.ErrorCode(datypes.StatusContextDeadline), &datypes.ErrContextDeadline)
	errs.Register(jsonrpc.ErrorCode(datypes.StatusContextCanceled), &datypes.ErrContextCanceled)
	errs.Register(jsonrpc.ErrorCode(datypes.StatusHeightFromFuture), &datypes.ErrHeightFromFuture)
	return errs
}
