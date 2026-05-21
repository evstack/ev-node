package da

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
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
	Client            FiberClient
	Logger            zerolog.Logger
	DefaultTimeout    time.Duration
	Namespace         string
	DataNamespace     string
	LastKnownDAHeight uint64
}

type fiberDAClient struct {
	fiber             FiberClient
	logger            zerolog.Logger
	defaultTimeout    time.Duration
	namespaceBz       []byte
	dataNamespaceBz   []byte
	lastKnownDAHeight uint64
}

var _ FullClient = (*fiberDAClient)(nil)

func NewFiberClient(cfg FiberConfig) (FullClient, error) {
	if cfg.Client == nil {
		return nil, fmt.Errorf("fiber client in config is nil")
	}

	if cfg.DefaultTimeout == 0 {
		cfg.DefaultTimeout = 60 * time.Second
	}

	return &fiberDAClient{
		fiber:             cfg.Client,
		logger:            cfg.Logger.With().Str("component", "fiber_da_client").Logger(),
		defaultTimeout:    cfg.DefaultTimeout,
		lastKnownDAHeight: cfg.LastKnownDAHeight,
		namespaceBz:       datypes.NamespaceFromString(cfg.Namespace).Bytes(),
		dataNamespaceBz:   datypes.NamespaceFromString(cfg.DataNamespace).Bytes(),
	}, nil
}

func (c *fiberDAClient) Submit(ctx context.Context, data [][]byte, _ float64, namespace []byte, _ []byte) datypes.ResultSubmit {
	if len(data) == 0 {
		return datypes.ResultSubmit{
			BaseResult: datypes.BaseResult{
				Code:           datypes.StatusSuccess,
				SubmittedCount: 0,
				Timestamp:      time.Now(),
			},
		}
	}

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

	// Fibre's per-upload cap is ~128 MiB (hard server-side reject:
	// "data size %d exceeds maximum 134217723"). flattenBlobs adds
	// 4 bytes per blob + 4 prefix, so we target 120 MiB per chunk
	// to leave overhead room and avoid borderline rejects.
	chunks := chunkBlobsForFibre(data, fibreUploadChunkBudget)
	nsID := namespace[len(namespace)-10:]

	ids := make([][]byte, 0, len(chunks))
	var submitted int
	for chunkIdx, chunk := range chunks {
		flat := flattenBlobs(chunk)
		uploadStart := time.Now()
		result, err := c.fiber.Upload(context.Background(), nsID, flat)
		uploadDuration := time.Since(uploadStart)
		if err != nil {
			c.logger.Warn().
				Dur("duration", uploadDuration).
				Int("flat_size", len(flat)).
				Int("blob_count", len(chunk)).
				Int("chunk_idx", chunkIdx).
				Int("chunk_total", len(chunks)).
				Err(err).
				Msg("fiber upload duration (failed)")
			code := datypes.StatusError
			switch {
			case errors.Is(err, context.Canceled):
				code = datypes.StatusContextCanceled
			case errors.Is(err, context.DeadlineExceeded):
				code = datypes.StatusContextDeadline
			}
			c.logger.Error().Err(err).Msg("fiber upload failed")
			return datypes.ResultSubmit{
				BaseResult: datypes.BaseResult{
					Code:           code,
					Message:        fmt.Sprintf("fiber upload failed for blob (chunk %d/%d): %v", chunkIdx+1, len(chunks), err),
					SubmittedCount: uint64(submitted),
					BlobSize:       blobSize,
					Timestamp:      time.Now(),
				},
			}
		}
		c.logger.Info().
			Dur("duration", uploadDuration).
			Int("flat_size", len(flat)).
			Int("blob_count", len(chunk)).
			Int("chunk_idx", chunkIdx).
			Int("chunk_total", len(chunks)).
			Msg("fiber upload duration (ok)")
		ids = append(ids, result.BlobID)
		submitted += len(chunk)
	}

	c.logger.Debug().Int("num_ids", len(data)).Int("chunks", len(chunks)).Uint64("height", 0 /* TODO */).Msg("fiber DA submission successful")

	return datypes.ResultSubmit{
		BaseResult: datypes.BaseResult{
			Code:           datypes.StatusSuccess,
			IDs:            ids,
			SubmittedCount: uint64(submitted),
			Height:         0, /* TODO */
			BlobSize:       blobSize,
			Timestamp:      time.Now(),
		},
	}
}

// fibreUploadChunkBudget is the target maximum flattened size of a single
// Fibre Upload call. Fibre rejects payloads above ~128 MiB
// ("data size N exceeds maximum 134217723"); 120 MiB leaves slack for
// flattenBlobs's per-blob length prefixes and for any future overhead.
const fibreUploadChunkBudget = 120 * 1024 * 1024

// chunkBlobsForFibre groups data into chunks whose flattened size stays
// below budget. Per-blob length-prefix overhead matches flattenBlobs.
// A single oversized blob (already validated against DefaultMaxBlobSize
// above) lands in its own chunk; the upload still fails server-side but
// at least we don't drag healthy peers down with it.
func chunkBlobsForFibre(data [][]byte, budget int) [][][]byte {
	if len(data) == 0 {
		return nil
	}
	chunks := make([][][]byte, 0, 1)
	cur := make([][]byte, 0, len(data))
	curSize := 4 // flattenBlobs's count prefix
	for _, b := range data {
		entry := 4 + len(b)
		if len(cur) > 0 && curSize+entry > budget {
			chunks = append(chunks, cur)
			cur = make([][]byte, 0, len(data))
			curSize = 4
		}
		cur = append(cur, b)
		curSize += entry
	}
	if len(cur) > 0 {
		chunks = append(chunks, cur)
	}
	return chunks
}

func (c *fiberDAClient) Retrieve(ctx context.Context, height uint64, namespace []byte) datypes.ResultRetrieve {
	return c.retrieve(ctx, height, namespace, true)
}

func (c *fiberDAClient) RetrieveBlobs(ctx context.Context, height uint64, namespace []byte) datypes.ResultRetrieve {
	return c.retrieve(ctx, height, namespace, false)
}

func (c *fiberDAClient) retrieve(ctx context.Context, height uint64, namespace []byte, _ bool) datypes.ResultRetrieve {
	listenCtx, listenCancel := context.WithTimeout(ctx, c.defaultTimeout)
	defer listenCancel()

	blobCh, err := c.fiber.Listen(listenCtx, namespace[len(namespace)-10:], height)
	if err != nil {
		return datypes.ResultRetrieve{
			BaseResult: datypes.BaseResult{
				Code:      datypes.StatusError,
				Message:   fmt.Sprintf("fiber listen failed: %v", err),
				Height:    height,
				Timestamp: time.Now(),
			},
		}
	}

	var blobIDs []BlobID
loop:
	for {
		select {
		case <-listenCtx.Done():
			break loop
		case event, ok := <-blobCh:
			if !ok {
				break loop
			}
			if event.Height > height {
				break loop
			}
			blobIDs = append(blobIDs, event.BlobID)
		}
	}

	if len(blobIDs) == 0 {
		return datypes.ResultRetrieve{
			BaseResult: datypes.BaseResult{
				Code:      datypes.StatusNotFound,
				Message:   "no blobs found at height for given namespace",
				Height:    height,
				Timestamp: time.Now(),
			},
		}
	}

	ids := make([]datypes.ID, 0, len(blobIDs))
	data := make([][]byte, 0, len(blobIDs))
	for _, blobID := range blobIDs {
		dlCtx, dlCancel := context.WithTimeout(ctx, c.defaultTimeout)
		blobData, dlErr := c.fiber.Download(dlCtx, blobID)
		dlCancel()
		if dlErr != nil {
			return datypes.ResultRetrieve{
				BaseResult: datypes.BaseResult{
					Code:      datypes.StatusError,
					Message:   fmt.Sprintf("fiber download failed for blob %x: %v", blobID, dlErr),
					Height:    height,
					Timestamp: time.Now(),
				},
			}
		}
		split, splitErr := splitBlobs(blobData)
		if splitErr != nil {
			return datypes.ResultRetrieve{
				BaseResult: datypes.BaseResult{
					Code:      datypes.StatusError,
					Message:   fmt.Sprintf("fiber decode failed for blob %x: %v", blobID, splitErr),
					Height:    height,
					Timestamp: time.Now(),
				},
			}
		}
		for _, b := range split {
			ids = append(ids, blobID)
			data = append(data, b)
		}
	}

	return datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{
			Code:      datypes.StatusSuccess,
			Height:    height,
			IDs:       ids,
			Timestamp: time.Now(),
		},
		Data: data,
	}
}

func (c *fiberDAClient) Get(ctx context.Context, ids []datypes.ID, _ []byte) ([]datypes.Blob, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	res := make([]datypes.Blob, 0, len(ids))
	for _, id := range ids {
		downloadCtx, cancel := context.WithTimeout(ctx, c.defaultTimeout)
		data, err := c.fiber.Download(downloadCtx, id)
		cancel()
		if err != nil {
			return nil, fmt.Errorf("fiber download failed for blob %x: %w", id, err)
		}
		split, splitErr := splitBlobs(data)
		if splitErr != nil {
			return nil, fmt.Errorf("fiber decode failed for blob %x: %w", id, splitErr)
		}
		res = append(res, split...)
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
		blobCh, err := c.fiber.Listen(ctx, namespace[len(namespace)-10:], c.lastKnownDAHeight)
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
					c.logger.Error().Err(err).Bytes("blob_id", event.BlobID).Msg("failed to retrieve blob")
					continue
				}

				split, splitErr := splitBlobs(blobData)
				if splitErr != nil {
					c.logger.Error().Err(splitErr).Bytes("blob_id", event.BlobID).Msg("failed to decode blob")
					continue
				}

				select {
				case out <- datypes.SubscriptionEvent{
					Height:    event.Height,
					Timestamp: time.Now(),
					Blobs:     split,
				}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return out, nil
}

// GetLatestDAHeight returns the latest height available on the DA layer by
// querying the network head.
func (c *fiberDAClient) GetLatestDAHeight(ctx context.Context) (uint64, error) {
	headCtx, cancel := context.WithTimeout(ctx, c.defaultTimeout)
	defer cancel()

	heigth, err := c.fiber.Head(headCtx)
	if err != nil {
		return 0, fmt.Errorf("failed to get DA network head: %w", err)
	}

	return heigth, nil
}

func (c *fiberDAClient) GetProofs(_ context.Context, ids []datypes.ID, _ []byte) ([]datypes.Proof, error) {
	return []datypes.Proof{}, fmt.Errorf("not implemented")
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

func flattenBlobs(blobs [][]byte) []byte {
	if len(blobs) == 0 {
		return nil
	}

	var total int
	for _, b := range blobs {
		total += 4 + len(b)
	}
	total += 4

	buf := make([]byte, total)
	binary.BigEndian.PutUint32(buf, uint32(len(blobs)))
	off := 4
	for _, b := range blobs {
		binary.BigEndian.PutUint32(buf[off:], uint32(len(b)))
		off += 4
		copy(buf[off:], b)
		off += len(b)
	}
	return buf
}

func splitBlobs(data []byte) ([][]byte, error) {
	if len(data) == 0 {
		return nil, nil
	}
	if len(data) < 4 {
		return nil, fmt.Errorf("invalid blob encoding: header too short")
	}

	count := int(binary.BigEndian.Uint32(data))
	off := 4
	blobs := make([][]byte, 0, count)
	for i := range count {
		if off+4 > len(data) {
			return nil, fmt.Errorf("invalid blob encoding: truncated length at index %d", i)
		}
		size := int(binary.BigEndian.Uint32(data[off:]))
		off += 4
		end := off + size
		if end < off || end > len(data) {
			return nil, fmt.Errorf("invalid blob encoding: truncated data at index %d", i)
		}
		blob := make([]byte, size)
		copy(blob, data[off:end])
		off = end
		blobs = append(blobs, blob)
	}
	return blobs, nil
}

// Force Inclusion is disabled for Fiber PoC.
func (c *fiberDAClient) HasForcedInclusionNamespace() bool   { return false }
func (c *fiberDAClient) GetForcedInclusionNamespace() []byte { return nil }
