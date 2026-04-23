package da

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/block/internal/da/fiber"
	datypes "github.com/evstack/ev-node/pkg/da/types"
)

type (
	FiberClient  = fiber.DA
	BlobID       = fiber.BlobID
	UploadResult = fiber.UploadResult
	BlobEvent    = fiber.BlobEvent
)

type FiberConfig struct {
	Client         FiberClient
	Logger         zerolog.Logger
	DefaultTimeout time.Duration
	Namespace      string
	DataNamespace  string
	UploadWorkers  int
}

type fiberDAClient struct {
	fiber           FiberClient
	logger          zerolog.Logger
	defaultTimeout  time.Duration
	namespaceBz     []byte
	dataNamespaceBz []byte
	uploadWorkers   int
	latestHeight    atomic.Uint64
}

var _ FullClient = (*fiberDAClient)(nil)

func NewFiberClient(cfg FiberConfig) (FullClient, error) {
	if cfg.Client == nil {
		return nil, fmt.Errorf("fiber client in config is nil")
	}

	if cfg.DefaultTimeout == 0 {
		cfg.DefaultTimeout = 60 * time.Second
	}

	if cfg.UploadWorkers == 0 {
		cfg.UploadWorkers = 8
	}

	return &fiberDAClient{
		fiber:           cfg.Client,
		logger:          cfg.Logger.With().Str("component", "fiber_da_client").Logger(),
		defaultTimeout:  cfg.DefaultTimeout,
		namespaceBz:     datypes.NamespaceFromString(cfg.Namespace).Bytes(),
		dataNamespaceBz: datypes.NamespaceFromString(cfg.DataNamespace).Bytes(),
		uploadWorkers:   cfg.UploadWorkers,
	}, nil
}

func makeFiberID(height uint64, blobID []byte) datypes.ID {
	id := make([]byte, 8+len(blobID))
	binary.LittleEndian.PutUint64(id, height)
	copy(id[8:], blobID)
	return id
}

func splitFiberID(id datypes.ID) (uint64, []byte) {
	if len(id) <= 8 {
		return 0, nil
	}
	return binary.LittleEndian.Uint64(id[:8]), id[8:]
}

func (c *fiberDAClient) Submit(ctx context.Context, data [][]byte, _ float64, namespace []byte, _ []byte) datypes.ResultSubmit {
	var blobSize uint64
	for _, b := range data {
		blobSize += uint64(len(b))
	}

	for i, raw := range data {
		if uint64(len(raw)) > common.DefaultMaxBlobSize {
			return datypes.ResultSubmit{
				BaseResult: datypes.BaseResult{
					Code:    datypes.StatusTooBig,
					Message: fmt.Sprintf("blob %d exceeds max size (%d > %d)", i, len(raw), common.DefaultMaxBlobSize),
				},
			}
		}
	}

	type uploadTask struct {
		index int
		data  []byte
	}

	type uploadResponse struct {
		index  int
		blobID []byte
		err    error
	}

	taskCh := make(chan uploadTask, len(data))
	respCh := make(chan uploadResponse, len(data))

	var wg sync.WaitGroup
	for range c.uploadWorkers {
		wg.Go(func() {
			for task := range taskCh {
				uploadCtx, cancel := context.WithTimeout(ctx, c.defaultTimeout)
				result, err := c.fiber.Upload(uploadCtx, namespace, task.data)
				cancel()
				respCh <- uploadResponse{
					index:  task.index,
					blobID: result.BlobID,
					err:    err,
				}
			}
		})
	}

	for i, raw := range data {
		taskCh <- uploadTask{index: i, data: raw}
	}
	close(taskCh)

	results := make([]uploadResponse, 0, len(data))
	for range len(data) {
		resp := <-respCh
		results = append(results, resp)
		if resp.err != nil {
			code := datypes.StatusError
			switch {
			case errors.Is(resp.err, context.Canceled):
				code = datypes.StatusContextCanceled
			case errors.Is(resp.err, context.DeadlineExceeded):
				code = datypes.StatusContextDeadline
			}

			c.logger.Error().Err(resp.err).Int("blob_index", resp.index).Msg("fiber upload failed")

			return datypes.ResultSubmit{
				BaseResult: datypes.BaseResult{
					Code:           code,
					Message:        fmt.Sprintf("fiber upload failed for blob %d: %v", resp.index, resp.err),
					SubmittedCount: uint64(len(results) - 1),
					BlobSize:       blobSize,
					Timestamp:      time.Now(),
				},
			}
		}
	}

	submitHeight := c.latestHeight.Add(1)

	ids := make([]datypes.ID, len(data))
	for _, r := range results {
		ids[r.index] = makeFiberID(submitHeight, r.blobID)
	}

	c.logger.Debug().Int("num_ids", len(data)).Uint64("height", submitHeight).Msg("fiber DA submission successful")

	return datypes.ResultSubmit{
		BaseResult: datypes.BaseResult{
			Code:           datypes.StatusSuccess,
			IDs:            ids,
			SubmittedCount: uint64(len(ids)),
			Height:         submitHeight,
			BlobSize:       blobSize,
			Timestamp:      time.Now(),
		},
	}
}

func (c *fiberDAClient) Retrieve(ctx context.Context, height uint64, namespace []byte) datypes.ResultRetrieve {
	return c.retrieve(ctx, height, namespace, true)
}

func (c *fiberDAClient) RetrieveBlobs(ctx context.Context, height uint64, namespace []byte) datypes.ResultRetrieve {
	return c.retrieve(ctx, height, namespace, false)
}

func (c *fiberDAClient) retrieve(_ context.Context, height uint64, namespace []byte, _ bool) datypes.ResultRetrieve {
	// not implemented, we cannot get block from on specific fiber height.
	return datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{
			Code:      datypes.StatusNotFound,
			Message:   "no blobs found at height for given namespace in fiber index",
			Height:    height,
			Timestamp: time.Now(),
		},
		Data: nil,
	}
}

func (c *fiberDAClient) Get(ctx context.Context, ids []datypes.ID, _ []byte) ([]datypes.Blob, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	res := make([]datypes.Blob, 0, len(ids))
	for _, id := range ids {
		_, blobID := splitFiberID(id)
		if blobID == nil {
			return nil, fmt.Errorf("invalid fiber blob id: %x", id)
		}

		downloadCtx, cancel := context.WithTimeout(ctx, c.defaultTimeout)
		data, err := c.fiber.Download(downloadCtx, blobID)
		cancel()
		if err != nil {
			return nil, fmt.Errorf("fiber download failed for blob %x: %w", blobID, err)
		}
		res = append(res, data)
	}

	return res, nil
}

const fiberSubscribeChanSize = 42

func (c *fiberDAClient) Subscribe(ctx context.Context, namespace []byte, _ bool) (<-chan datypes.SubscriptionEvent, error) {
	out := make(chan datypes.SubscriptionEvent, fiberSubscribeChanSize)

	go func() {
		defer close(out)

		// The outer DA Subscribe entry point does not expose a starting
		// height, so start from the live tip (fromHeight=0). A future
		// refactor that plumbs resume-from-height through datypes.DA can
		// thread the value here.
		blobCh, err := c.fiber.Listen(ctx, namespace, 0)
		if err != nil {
			c.logger.Error().Err(err).Msg("fiber listen failed")
			return
		}

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-blobCh:
				if !ok {
					return
				}

				blobData, err := c.fiber.Download(ctx, event.BlobID)
				if err != nil {
					c.logger.Error().Err(err).Bytes("blob_id", event.BlobID).Msg("failed to retrier blob id")
				}

				select {
				case out <- datypes.SubscriptionEvent{
					Height:    event.Height,
					Timestamp: time.Now(),
					Blobs:     [][]byte{blobData},
				}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return out, nil
}

func (c *fiberDAClient) GetLatestDAHeight(context.Context) (uint64, error) {
	return c.latestHeight.Load(), nil
}

func (c *fiberDAClient) GetProofs(_ context.Context, ids []datypes.ID, _ []byte) ([]datypes.Proof, error) {
	// not implemented.
	return []datypes.Proof{}, nil
}

func (c *fiberDAClient) Validate(_ context.Context, ids []datypes.ID, proofs []datypes.Proof, _ []byte) ([]bool, error) {
	if len(ids) != len(proofs) {
		return nil, errors.New("number of IDs and proofs must match")
	}
	if len(ids) == 0 {
		return []bool{}, nil
	}

	results := make([]bool, len(ids))

	// not implemented.
	for i := range results {
		results[i] = true
	}

	return results, nil
}

func (c *fiberDAClient) GetHeaderNamespace() []byte { return c.namespaceBz }
func (c *fiberDAClient) GetDataNamespace() []byte   { return c.dataNamespaceBz }

// Force Inclusion is disabled for Fiber PoC.
func (c *fiberDAClient) HasForcedInclusionNamespace() bool   { return false }
func (c *fiberDAClient) GetForcedInclusionNamespace() []byte { return nil }
