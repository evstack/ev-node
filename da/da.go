package da

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"

	goDA "github.com/rollkit/go-da"
	"github.com/rollkit/rollkit/third_party/log"
	"github.com/rollkit/rollkit/types"
	pb "github.com/rollkit/rollkit/types/pb/rollkit"
)

const (
	// defaultSubmitTimeout is the timeout for block submission
	defaultSubmitTimeout = 60 * time.Second

	// defaultRetrieveTimeout is the timeout for block retrieval
	defaultRetrieveTimeout = 60 * time.Second
)

// StatusCode is a type for DA layer return status.
// TODO: define an enum of different non-happy-path cases
// that might need to be handled by Rollkit independent of
// the underlying DA chain.
type StatusCode uint64

// Data Availability return codes.
const (
	StatusUnknown StatusCode = iota
	StatusSuccess
	StatusNotFound
	StatusNotIncludedInBlock
	StatusAlreadyInMempool
	StatusTooBig
	StatusContextDeadline
	StatusError
)

// BaseResult contains basic information returned by DA layer.
type BaseResult struct {
	// Code is to determine if the action succeeded.
	Code StatusCode
	// Message may contain DA layer specific information (like DA block height/hash, detailed error message, etc)
	Message string
	// DAHeight informs about a height on Data Availability Layer for given result.
	DAHeight uint64
	// SubmittedCount is the number of successfully submitted blocks.
	SubmittedCount uint64
}

// ResultSubmit contains information returned from DA layer after block headers/data submission.
type ResultSubmit struct {
	BaseResult
	// Not sure if this needs to be bubbled up to other
	// parts of Rollkit.
	// Hash hash.Hash
}

// ResultRetrieveHeaders contains batch of block headers returned from DA layer client.
type ResultRetrieveHeaders struct {
	BaseResult
	// Header is the block header retrieved from Data Availability Layer.
	// If Code is not equal to StatusSuccess, it has to be nil.
	Headers []*types.SignedHeader
}

// DAClient is a new DA implementation.
type DAClient struct {
	DA              goDA.DA
	GasPrice        float64
	GasMultiplier   float64
	Namespace       goDA.Namespace
	SubmitOptions   []byte
	SubmitTimeout   time.Duration
	RetrieveTimeout time.Duration
	Logger          log.Logger
}

// NewDAClient returns a new DA client.
func NewDAClient(da goDA.DA, gasPrice, gasMultiplier float64, ns goDA.Namespace, options []byte, logger log.Logger) *DAClient {
	return &DAClient{
		DA:              da,
		GasPrice:        gasPrice,
		GasMultiplier:   gasMultiplier,
		Namespace:       ns,
		SubmitOptions:   options,
		SubmitTimeout:   defaultSubmitTimeout,
		RetrieveTimeout: defaultRetrieveTimeout,
		Logger:          logger,
	}
}

// SubmitHeaders submits block headers to DA.
func (dac *DAClient) SubmitHeaders(ctx context.Context, headers []*types.SignedHeader, maxBlobSize uint64, gasPrice float64) ResultSubmit {
	var (
		blobs    [][]byte
		blobSize uint64
		message  string
	)
	for i := range headers {
		blob, err := headers[i].MarshalBinary()
		if err != nil {
			message = fmt.Sprint("failed to serialize header", err)
			dac.Logger.Info(message)
			break
		}
		if blobSize+uint64(len(blob)) > maxBlobSize {
			message = fmt.Sprint((&goDA.ErrBlobSizeOverLimit{}).Error(), "blob size limit reached", "maxBlobSize", maxBlobSize, "index", i, "blobSize", blobSize, "len(blob)", len(blob))
			dac.Logger.Info(message)
			break
		}
		blobSize += uint64(len(blob))
		blobs = append(blobs, blob)
	}
	if len(blobs) == 0 {
		return ResultSubmit{
			BaseResult: BaseResult{
				Code:    StatusError,
				Message: "failed to submit headers: no blobs generated " + message,
			},
		}
	}

	ctx, cancel := context.WithTimeout(ctx, dac.SubmitTimeout)
	defer cancel()
	ids, err := dac.submit(ctx, blobs, gasPrice, dac.Namespace)
	if err != nil {
		status := StatusError
		switch {
		case errors.Is(err, &goDA.ErrTxTimedOut{}):
			status = StatusNotIncludedInBlock
		case errors.Is(err, &goDA.ErrTxAlreadyInMempool{}):
			status = StatusAlreadyInMempool
		case errors.Is(err, &goDA.ErrTxIncorrectAccountSequence{}):
			status = StatusAlreadyInMempool
		case errors.Is(err, &goDA.ErrTxTooLarge{}):
			status = StatusTooBig
		case errors.Is(err, &goDA.ErrContextDeadline{}):
			status = StatusContextDeadline
		}
		return ResultSubmit{
			BaseResult: BaseResult{
				Code:    status,
				Message: "failed to submit headers: " + err.Error(),
			},
		}
	}

	if len(ids) == 0 {
		return ResultSubmit{
			BaseResult: BaseResult{
				Code:    StatusError,
				Message: "failed to submit headers: unexpected len(ids): 0",
			},
		}
	}

	return ResultSubmit{
		BaseResult: BaseResult{
			Code:           StatusSuccess,
			DAHeight:       binary.LittleEndian.Uint64(ids[0]),
			SubmittedCount: uint64(len(ids)),
		},
	}
}

// RetrieveHeaders retrieves block headers from DA.
func (dac *DAClient) RetrieveHeaders(ctx context.Context, dataLayerHeight uint64) ResultRetrieveHeaders {
	result, err := dac.DA.GetIDs(ctx, dataLayerHeight, dac.Namespace)
	if err != nil {
		return ResultRetrieveHeaders{
			BaseResult: BaseResult{
				Code:     StatusError,
				Message:  fmt.Sprintf("failed to get IDs: %s", err.Error()),
				DAHeight: dataLayerHeight,
			},
		}
	}

	// If no blocks are found, return a non-blocking error.
	if len(result.IDs) == 0 {
		return ResultRetrieveHeaders{
			BaseResult: BaseResult{
				Code:     StatusNotFound,
				Message:  (&goDA.ErrBlobNotFound{}).Error(),
				DAHeight: dataLayerHeight,
			},
		}
	}

	ctx, cancel := context.WithTimeout(ctx, dac.RetrieveTimeout)
	defer cancel()
	blobs, err := dac.DA.Get(ctx, result.IDs, dac.Namespace)
	if err != nil {
		return ResultRetrieveHeaders{
			BaseResult: BaseResult{
				Code:     StatusError,
				Message:  fmt.Sprintf("failed to get blobs: %s", err.Error()),
				DAHeight: dataLayerHeight,
			},
		}
	}

	headers := make([]*types.SignedHeader, len(blobs))
	for i, blob := range blobs {
		var header pb.SignedHeader
		err = proto.Unmarshal(blob, &header)
		if err != nil {
			dac.Logger.Error("failed to unmarshal block", "daHeight", dataLayerHeight, "position", i, "error", err)
			continue
		}
		headers[i] = new(types.SignedHeader)
		err := headers[i].FromProto(&header)
		if err != nil {
			return ResultRetrieveHeaders{
				BaseResult: BaseResult{
					Code:    StatusError,
					Message: err.Error(),
				},
			}
		}
	}

	return ResultRetrieveHeaders{
		BaseResult: BaseResult{
			Code:     StatusSuccess,
			DAHeight: dataLayerHeight,
		},
		Headers: headers,
	}
}

func (dac *DAClient) submit(ctx context.Context, blobs []goDA.Blob, gasPrice float64, namespace goDA.Namespace) ([]goDA.ID, error) {
	if len(dac.SubmitOptions) == 0 {
		return dac.DA.Submit(ctx, blobs, gasPrice, namespace)
	}
	return dac.DA.SubmitWithOptions(ctx, blobs, gasPrice, namespace, dac.SubmitOptions)
}
