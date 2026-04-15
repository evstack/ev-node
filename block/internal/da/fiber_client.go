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
	"github.com/evstack/ev-node/block/internal/da/fibremock"
	datypes "github.com/evstack/ev-node/pkg/da/types"
)

const (
	fiberIndexRetainHeights = 4096
	fiberSubscribePollFreq  = 1 * time.Second
	fiberSubscribeChanSize  = 16
)

type (
	FiberClient  = fibremock.DA
	BlobID       = fibremock.BlobID
	UploadResult = fibremock.UploadResult
	BlobEvent    = fibremock.BlobEvent
)

type FiberConfig struct {
	Client                   FiberClient
	Logger                   zerolog.Logger
	DefaultTimeout           time.Duration
	Namespace                string
	DataNamespace            string
	ForcedInclusionNamespace string
}

type fiberDAClient struct {
	fiber              FiberClient
	logger             zerolog.Logger
	defaultTimeout     time.Duration
	namespaceBz        []byte
	dataNamespaceBz    []byte
	forcedNamespaceBz  []byte
	hasForcedNamespace bool

	mu           sync.RWMutex
	index        map[uint64][]fiberIndexedBlob
	indexTail    uint64
	indexWindow  uint64
	latestHeight uint64
}

type fiberIndexedBlob struct {
	id        datypes.ID
	namespace []byte
	data      []byte
	blobID    []byte
}

var _ FullClient = (*fiberDAClient)(nil)

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
		indexWindow:        fiberIndexRetainHeights,
	}
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

func (c *fiberDAClient) pruneIndexLocked() {
	if c.indexWindow == 0 || c.indexTail == 0 || c.indexTail < c.indexWindow {
		return
	}
	cutoff := c.indexTail - c.indexWindow + 1
	for h := range c.index {
		if h < cutoff {
			delete(c.index, h)
		}
	}
}

func (c *fiberDAClient) Submit(ctx context.Context, data [][]byte, _ float64, namespace []byte, _ []byte) datypes.ResultSubmit {
	var blobSize uint64
	for _, b := range data {
		blobSize += uint64(len(b))
	}

	uploaded := make([]fiberUploadResult, 0, len(data))

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

			if len(uploaded) > 0 {
				c.indexUploaded(uploaded, namespace)
			}

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

		c.mu.Lock()
		c.latestHeight++
		h := c.latestHeight
		c.mu.Unlock()

		uploaded = append(uploaded, fiberUploadResult{
			blobID: result.BlobID,
			height: h,
			data:   raw,
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

	ids := c.indexUploaded(uploaded, namespace)

	c.logger.Debug().Int("num_ids", len(ids)).Uint64("height", uploaded[len(uploaded)-1].height).Msg("fiber DA submission successful")

	return datypes.ResultSubmit{
		BaseResult: datypes.BaseResult{
			Code:           datypes.StatusSuccess,
			IDs:            ids,
			SubmittedCount: uint64(len(ids)),
			Height:         uploaded[len(uploaded)-1].height,
			BlobSize:       blobSize,
			Timestamp:      time.Now(),
		},
	}
}

type fiberUploadResult struct {
	blobID []byte
	height uint64
	data   []byte
}

func (c *fiberDAClient) indexUploaded(uploaded []fiberUploadResult, namespace []byte) []datypes.ID {
	submitHeight := uploaded[len(uploaded)-1].height
	ids := make([]datypes.ID, len(uploaded))

	c.mu.Lock()
	for i, u := range uploaded {
		id := makeFiberID(submitHeight, u.blobID)
		ids[i] = id
		c.index[submitHeight] = append(c.index[submitHeight], fiberIndexedBlob{
			id:        id,
			namespace: namespace,
			data:      u.data,
			blobID:    u.blobID,
		})
	}
	if submitHeight > c.indexTail {
		c.indexTail = submitHeight
	}
	c.pruneIndexLocked()
	c.mu.Unlock()

	return ids
}

func (c *fiberDAClient) Retrieve(ctx context.Context, height uint64, namespace []byte) datypes.ResultRetrieve {
	return c.retrieve(ctx, height, namespace, true)
}

func (c *fiberDAClient) RetrieveBlobs(ctx context.Context, height uint64, namespace []byte) datypes.ResultRetrieve {
	return c.retrieve(ctx, height, namespace, false)
}

func (c *fiberDAClient) retrieve(_ context.Context, height uint64, namespace []byte, _ bool) datypes.ResultRetrieve {
	c.mu.RLock()
	allBlobs, ok := c.index[height]
	c.mu.RUnlock()

	if !ok || len(allBlobs) == 0 {
		return datypes.ResultRetrieve{
			BaseResult: datypes.BaseResult{
				Code:      datypes.StatusNotFound,
				Message:   "no blobs found at height in fiber index",
				Height:    height,
				Timestamp: time.Now(),
			},
		}
	}

	var matching []fiberIndexedBlob
	for _, b := range allBlobs {
		if nsEqual(b.namespace, namespace) {
			matching = append(matching, b)
		}
	}

	if len(matching) == 0 {
		return datypes.ResultRetrieve{
			BaseResult: datypes.BaseResult{
				Code:      datypes.StatusNotFound,
				Message:   "no blobs found at height for given namespace in fiber index",
				Height:    height,
				Timestamp: time.Now(),
			},
		}
	}

	ids := make([]datypes.ID, len(matching))
	data := make([]datypes.Blob, len(matching))
	for i, b := range matching {
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

func nsEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
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

func (c *fiberDAClient) Subscribe(ctx context.Context, _ []byte, _ bool) (<-chan datypes.SubscriptionEvent, error) {
	out := make(chan datypes.SubscriptionEvent, fiberSubscribeChanSize)

	go func() {
		defer close(out)

		ticker := time.NewTicker(fiberSubscribePollFreq)
		defer ticker.Stop()

		var lastHeight uint64

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.mu.RLock()
				height := c.latestHeight
				c.mu.RUnlock()

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

func (c *fiberDAClient) GetLatestDAHeight(context.Context) (uint64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.latestHeight, nil
}

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
				proofs[i] = b.blobID
				break
			}
		}
	}

	return proofs, nil
}

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
				results[i] = len(proofs[i]) > 0 && string(proofs[i]) == string(b.blobID)
				break
			}
		}
	}

	return results, nil
}

func (c *fiberDAClient) GetHeaderNamespace() []byte          { return c.namespaceBz }
func (c *fiberDAClient) GetDataNamespace() []byte            { return c.dataNamespaceBz }
func (c *fiberDAClient) GetForcedInclusionNamespace() []byte { return c.forcedNamespaceBz }
func (c *fiberDAClient) HasForcedInclusionNamespace() bool   { return c.hasForcedNamespace }
