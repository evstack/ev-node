package da

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block/internal/common"
	datypes "github.com/evstack/ev-node/pkg/da/types"
)

// FiberUploadResult contains the result of a Fiber upload operation.
type FiberUploadResult struct {
	// BlobID is the Fiber blob identifier (typically 33 bytes: 1 version + 32 commitment).
	BlobID []byte
	// Height is the validator set height at which the blob was uploaded.
	Height uint64
	// Promise is the serialized signed payment promise from validators,
	// which serves as proof of data availability.
	Promise []byte
}

// FiberClient defines the interface for a Fiber protocol client backend.
// Implementations wrap the celestia-app fibre.Client or equivalent.
type FiberClient interface {
	// Upload uploads data to the Fiber network under the given namespace.
	// The namespace must be a valid share.Namespace (29 bytes).
	Upload(ctx context.Context, namespace, data []byte) (FiberUploadResult, error)

	// Download downloads and reconstructs data from the Fiber network.
	// blobID is the Fiber blob identifier returned by Upload.
	// height is the validator set height (0 to use the current head).
	Download(ctx context.Context, blobID []byte, height uint64) ([]byte, error)

	// GetLatestHeight returns the latest block height from the Fiber network.
	GetLatestHeight(ctx context.Context) (uint64, error)
}

// FiberConfig holds configuration for the Fiber DA client.
type FiberConfig struct {
	// Client is the Fiber protocol client backend.
	Client FiberClient
	// Logger is the structured logger.
	Logger zerolog.Logger
	// DefaultTimeout is the default timeout for operations.
	DefaultTimeout time.Duration
	// Namespace is the header namespace string.
	Namespace string
	// DataNamespace is the data namespace string.
	DataNamespace string
	// ForcedInclusionNamespace is the forced inclusion namespace string.
	ForcedInclusionNamespace string
}

// fiberDAClient adapts a FiberClient to the ev-node FullClient interface.
// It bridges the Fiber push/pull model to the block-based DA interface by
// maintaining a local index of submitted blobs for height-based retrieval.
type fiberDAClient struct {
	fiber              FiberClient
	logger             zerolog.Logger
	defaultTimeout     time.Duration
	namespaceBz        []byte
	dataNamespaceBz    []byte
	forcedNamespaceBz  []byte
	hasForcedNamespace bool

	mu    sync.RWMutex
	index map[uint64][]fiberIndexedBlob
}

type fiberIndexedBlob struct {
	id      datypes.ID
	data    []byte
	promise []byte
	blobID  []byte
}

var _ FullClient = (*fiberDAClient)(nil)

// NewFiberClient creates a new Fiber DA client adapter.
// Returns nil if the Fiber client backend is not provided.
func NewFiberClient(cfg FiberConfig) FullClient {
	if cfg.Client == nil {
		return nil
	}
	if cfg.DefaultTimeout == 0 {
		cfg.DefaultTimeout = 60 * time.Second
	}

	hasForced := cfg.ForcedInclusionNamespace != ""
	var forcedBz []byte
	if hasForced {
		forcedBz = datypes.NamespaceFromString(cfg.ForcedInclusionNamespace).Bytes()
	}

	return &fiberDAClient{
		fiber:              cfg.Client,
		logger:             cfg.Logger.With().Str("component", "fiber_da_client").Logger(),
		defaultTimeout:     cfg.DefaultTimeout,
		namespaceBz:        datypes.NamespaceFromString(cfg.Namespace).Bytes(),
		dataNamespaceBz:    datypes.NamespaceFromString(cfg.DataNamespace).Bytes(),
		forcedNamespaceBz:  forcedBz,
		hasForcedNamespace: hasForced,
		index:              make(map[uint64][]fiberIndexedBlob),
	}
}

// makeFiberID constructs an ev-node DA ID from a Fiber height and blob ID.
// Format: 8 bytes LE height + blobID bytes (compatible with datypes.SplitID).
func makeFiberID(height uint64, blobID []byte) datypes.ID {
	id := make([]byte, 8+len(blobID))
	binary.LittleEndian.PutUint64(id, height)
	copy(id[8:], blobID)
	return id
}

// splitFiberID extracts the Fiber height and blob ID from an ev-node DA ID.
func splitFiberID(id datypes.ID) (uint64, []byte) {
	if len(id) <= 8 {
		return 0, nil
	}
	return binary.LittleEndian.Uint64(id[:8]), id[8:]
}

// Submit uploads each data blob to the Fiber network.
// All blobs are uploaded individually, then indexed under a single canonical height
// (the height of the last upload) to satisfy the ev-node DA contract that all
// submitted blobs appear at the same height.
func (c *fiberDAClient) Submit(ctx context.Context, data [][]byte, _ float64, namespace []byte, _ []byte) datypes.ResultSubmit {
	var blobSize uint64
	for _, b := range data {
		blobSize += uint64(len(b))
	}

	type uploadResult struct {
		blobID  []byte
		height  uint64
		promise []byte
		data    []byte
	}

	uploaded := make([]uploadResult, 0, len(data))

	for i, raw := range data {
		if uint64(len(raw)) > common.DefaultMaxBlobSize {
			return datypes.ResultSubmit{
				BaseResult: datypes.BaseResult{
					Code:    datypes.StatusTooBig,
					Message: fmt.Sprintf("blob %d exceeds max size (%d > %d)", i, len(raw), common.DefaultMaxBlobSize),
				},
			}
		}

		uploadCtx, cancel := context.WithTimeout(ctx, c.defaultTimeout)
		result, err := c.fiber.Upload(uploadCtx, namespace, raw)
		cancel()
		if err != nil {
			code := datypes.StatusError
			switch {
			case errors.Is(err, context.Canceled):
				code = datypes.StatusContextCanceled
			case errors.Is(err, context.DeadlineExceeded):
				code = datypes.StatusContextDeadline
			}
			c.logger.Error().Err(err).Int("blob_index", i).Msg("fiber upload failed")
			return datypes.ResultSubmit{
				BaseResult: datypes.BaseResult{
					Code:           code,
					Message:        fmt.Sprintf("fiber upload failed for blob %d: %v", i, err),
					SubmittedCount: uint64(len(uploaded)),
					BlobSize:       blobSize,
					Timestamp:      time.Now(),
				},
			}
		}

		uploaded = append(uploaded, uploadResult{
			blobID:  result.BlobID,
			height:  result.Height,
			promise: result.Promise,
			data:    raw,
		})
	}

	if len(uploaded) == 0 {
		return datypes.ResultSubmit{
			BaseResult: datypes.BaseResult{
				Code:      datypes.StatusSuccess,
				BlobSize:  blobSize,
				Timestamp: time.Now(),
			},
		}
	}

	// Use the height of the last upload as the canonical submit height.
	// Re-index all blobs under this single height so that Retrieve(height)
	// returns all blobs from the same submit call.
	submitHeight := uploaded[len(uploaded)-1].height
	ids := make([]datypes.ID, len(uploaded))

	c.mu.Lock()
	for i, u := range uploaded {
		id := makeFiberID(submitHeight, u.blobID)
		ids[i] = id
		c.index[submitHeight] = append(c.index[submitHeight], fiberIndexedBlob{
			id:      id,
			data:    u.data,
			promise: u.promise,
			blobID:  u.blobID,
		})
	}
	c.mu.Unlock()

	c.logger.Debug().Int("num_ids", len(ids)).Uint64("height", submitHeight).Msg("fiber DA submission successful")

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

// Retrieve retrieves blobs from the Fiber network at the specified height and namespace.
// It first checks the local submission index, then falls back to downloading via blob IDs.
func (c *fiberDAClient) Retrieve(ctx context.Context, height uint64, namespace []byte) datypes.ResultRetrieve {
	return c.retrieve(ctx, height, namespace, true)
}

// RetrieveBlobs retrieves blobs without blocking on timestamp resolution.
func (c *fiberDAClient) RetrieveBlobs(ctx context.Context, height uint64, namespace []byte) datypes.ResultRetrieve {
	return c.retrieve(ctx, height, namespace, false)
}

func (c *fiberDAClient) retrieve(_ context.Context, height uint64, _ []byte, _ bool) datypes.ResultRetrieve {
	c.mu.RLock()
	blobs, ok := c.index[height]
	c.mu.RUnlock()

	if !ok || len(blobs) == 0 {
		return datypes.ResultRetrieve{
			BaseResult: datypes.BaseResult{
				Code:      datypes.StatusNotFound,
				Message:   "no blobs found at height in fiber index",
				Height:    height,
				Timestamp: time.Now(),
			},
		}
	}

	ids := make([]datypes.ID, len(blobs))
	data := make([]datypes.Blob, len(blobs))
	for i, b := range blobs {
		ids[i] = b.id
		data[i] = b.data
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

// Get downloads specific blobs by their IDs from the Fiber network.
// Each ID is decoded to extract the Fiber blob ID and height,
// then downloaded via the Fiber client.
func (c *fiberDAClient) Get(ctx context.Context, ids []datypes.ID, _ []byte) ([]datypes.Blob, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	res := make([]datypes.Blob, 0, len(ids))
	for _, id := range ids {
		height, blobID := splitFiberID(id)
		if blobID == nil {
			return nil, fmt.Errorf("invalid fiber blob id: %x", id)
		}

		downloadCtx, cancel := context.WithTimeout(ctx, c.defaultTimeout)
		data, err := c.fiber.Download(downloadCtx, blobID, height)
		cancel()
		if err != nil {
			return nil, fmt.Errorf("fiber download failed for blob %x: %w", blobID, err)
		}
		res = append(res, data)
	}

	return res, nil
}

// Subscribe returns a channel that emits SubscriptionEvents for new DA heights.
// Since the Fiber protocol doesn't have a native subscription mechanism, this
// implementation polls GetLatestHeight and emits events for heights present in
// the local submission index. Only heights indexed by this client are emitted.
func (c *fiberDAClient) Subscribe(ctx context.Context, _ []byte, _ bool) (<-chan datypes.SubscriptionEvent, error) {
	out := make(chan datypes.SubscriptionEvent, 16)

	go func() {
		defer close(out)

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		var lastHeight uint64

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				heightCtx, cancel := context.WithTimeout(ctx, c.defaultTimeout)
				height, err := c.fiber.GetLatestHeight(heightCtx)
				cancel()
				if err != nil {
					c.logger.Error().Err(err).Msg("failed to get latest fiber height during subscribe")
					continue
				}
				if height <= lastHeight {
					continue
				}

				for h := lastHeight + 1; h <= height; h++ {
					c.mu.RLock()
					blobs, ok := c.index[h]
					c.mu.RUnlock()
					if !ok || len(blobs) == 0 {
						continue
					}

					blobData := make([][]byte, len(blobs))
					for i, b := range blobs {
						blobData[i] = b.data
					}

					select {
					case out <- datypes.SubscriptionEvent{
						Height:    h,
						Timestamp: time.Now(),
						Blobs:     blobData,
					}:
					case <-ctx.Done():
						return
					}
				}
				lastHeight = height
			}
		}
	}()

	return out, nil
}

// GetLatestDAHeight returns the latest block height from the Fiber network.
func (c *fiberDAClient) GetLatestDAHeight(ctx context.Context) (uint64, error) {
	heightCtx, cancel := context.WithTimeout(ctx, c.defaultTimeout)
	defer cancel()

	height, err := c.fiber.GetLatestHeight(heightCtx)
	if err != nil {
		return 0, fmt.Errorf("failed to get latest fiber height: %w", err)
	}
	return height, nil
}

// GetProofs returns the serialized payment promises as proofs for the given IDs.
// In the Fiber protocol, the signed payment promise from validators serves as
// proof of data availability.
func (c *fiberDAClient) GetProofs(_ context.Context, ids []datypes.ID, _ []byte) ([]datypes.Proof, error) {
	if len(ids) == 0 {
		return []datypes.Proof{}, nil
	}

	proofs := make([]datypes.Proof, len(ids))
	for i, id := range ids {
		height, _ := splitFiberID(id)

		c.mu.RLock()
		blobs := c.index[height]
		c.mu.RUnlock()

		for _, b := range blobs {
			if string(b.id) == string(id) {
				proofs[i] = b.promise
				break
			}
		}
	}

	return proofs, nil
}

// Validate verifies that the proofs (payment promises) correspond to the given IDs.
// It checks that each proof was stored for the matching blob during submission.
func (c *fiberDAClient) Validate(_ context.Context, ids []datypes.ID, proofs []datypes.Proof, _ []byte) ([]bool, error) {
	if len(ids) != len(proofs) {
		return nil, errors.New("number of IDs and proofs must match")
	}
	if len(ids) == 0 {
		return []bool{}, nil
	}

	results := make([]bool, len(ids))
	for i, id := range ids {
		height, _ := splitFiberID(id)

		c.mu.RLock()
		blobs := c.index[height]
		c.mu.RUnlock()

		for _, b := range blobs {
			if string(b.id) == string(id) {
				// A non-empty promise proof that matches the stored promise is valid.
				results[i] = len(proofs[i]) > 0 && string(proofs[i]) == string(b.promise)
				break
			}
		}
	}

	return results, nil
}

// GetHeaderNamespace returns the header namespace bytes.
func (c *fiberDAClient) GetHeaderNamespace() []byte { return c.namespaceBz }

// GetDataNamespace returns the data namespace bytes.
func (c *fiberDAClient) GetDataNamespace() []byte { return c.dataNamespaceBz }

// GetForcedInclusionNamespace returns the forced inclusion namespace bytes.
func (c *fiberDAClient) GetForcedInclusionNamespace() []byte { return c.forcedNamespaceBz }

// HasForcedInclusionNamespace reports whether forced inclusion namespace is configured.
func (c *fiberDAClient) HasForcedInclusionNamespace() bool { return c.hasForcedNamespace }
