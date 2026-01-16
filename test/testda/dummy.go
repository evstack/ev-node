// Package testda provides test implementations of the DA client interface.
package testda

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	datypes "github.com/evstack/ev-node/pkg/da/types"
)

const (
	// DefaultMaxBlobSize is the default maximum blob size (7MB).
	DefaultMaxBlobSize = 7 * 1024 * 1024
)

// Header contains DA layer header information for a given height.
// This mirrors the structure used by real DA clients like Celestia.
type Header struct {
	Height    uint64
	Timestamp time.Time
}

// Time returns the block time from the header.
// This mirrors the jsonrpc.Header.Time() method.
func (h *Header) Time() time.Time {
	return h.Timestamp
}

// DummyDA is a test implementation of the DA client interface.
// It supports blob storage, height simulation, failure injection, and header retrieval.
type DummyDA struct {
	mu         sync.Mutex
	height     atomic.Uint64
	maxBlobSz  uint64
	blobs      map[uint64]map[string][][]byte // height -> namespace -> blobs
	headers    map[uint64]*Header             // height -> header (with timestamp)
	failSubmit atomic.Bool

	tickerMu   sync.Mutex
	tickerStop chan struct{}
}

// Option configures a DummyDA instance.
type Option func(*DummyDA)

// WithMaxBlobSize sets the maximum blob size.
func WithMaxBlobSize(size uint64) Option {
	return func(d *DummyDA) {
		d.maxBlobSz = size
	}
}

// WithStartHeight sets the initial DA height.
func WithStartHeight(height uint64) Option {
	return func(d *DummyDA) {
		d.height.Store(height)
	}
}

// New creates a new DummyDA with the given options.
func New(opts ...Option) *DummyDA {
	d := &DummyDA{
		maxBlobSz: DefaultMaxBlobSize,
		blobs:     make(map[uint64]map[string][][]byte),
		headers:   make(map[uint64]*Header),
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// Submit stores blobs and returns success or simulated failure.
func (d *DummyDA) Submit(_ context.Context, data [][]byte, _ float64, namespace []byte, _ []byte) datypes.ResultSubmit {
	if d.failSubmit.Load() {
		return datypes.ResultSubmit{
			BaseResult: datypes.BaseResult{
				Code:    datypes.StatusError,
				Message: "simulated DA failure",
			},
		}
	}

	var blobSz uint64
	for _, b := range data {
		if uint64(len(b)) > d.maxBlobSz {
			return datypes.ResultSubmit{
				BaseResult: datypes.BaseResult{
					Code:    datypes.StatusTooBig,
					Message: datypes.ErrBlobSizeOverLimit.Error(),
				},
			}
		}
		blobSz += uint64(len(b))
	}

	now := time.Now()
	d.mu.Lock()
	height := d.height.Add(1)
	if d.blobs[height] == nil {
		d.blobs[height] = make(map[string][][]byte)
	}
	nsKey := string(namespace)
	d.blobs[height][nsKey] = append(d.blobs[height][nsKey], data...)
	// Store header with timestamp for this height
	if d.headers[height] == nil {
		d.headers[height] = &Header{
			Height:    height,
			Timestamp: now,
		}
	}
	d.mu.Unlock()

	return datypes.ResultSubmit{
		BaseResult: datypes.BaseResult{
			Code:           datypes.StatusSuccess,
			Height:         height,
			BlobSize:       blobSz,
			SubmittedCount: uint64(len(data)),
			Timestamp:      now,
		},
	}
}

// Retrieve returns blobs stored at the given height and namespace.
func (d *DummyDA) Retrieve(_ context.Context, height uint64, namespace []byte) datypes.ResultRetrieve {
	d.mu.Lock()
	byHeight := d.blobs[height]
	var blobs [][]byte
	if byHeight != nil {
		blobs = byHeight[string(namespace)]
	}
	// Get timestamp from header if available, otherwise use current time
	var timestamp time.Time
	if header := d.headers[height]; header != nil {
		timestamp = header.Timestamp
	}
	d.mu.Unlock()

	if len(blobs) == 0 {
		return datypes.ResultRetrieve{
			BaseResult: datypes.BaseResult{
				Code:      datypes.StatusNotFound,
				Height:    height,
				Message:   datypes.ErrBlobNotFound.Error(),
				Timestamp: timestamp,
			},
		}
	}

	ids := make([][]byte, len(blobs))
	return datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{
			Code:      datypes.StatusSuccess,
			Height:    height,
			IDs:       ids,
			Timestamp: timestamp,
		},
		Data: blobs,
	}
}

// GetHeaderNamespace returns the header namespace.
func (d *DummyDA) GetHeaderNamespace() []byte { return []byte("hdr") }

// GetDataNamespace returns the data namespace.
func (d *DummyDA) GetDataNamespace() []byte { return []byte("data") }

// GetForcedInclusionNamespace returns the forced inclusion namespace.
func (d *DummyDA) GetForcedInclusionNamespace() []byte { return nil }

// HasForcedInclusionNamespace reports whether forced inclusion is configured.
func (d *DummyDA) HasForcedInclusionNamespace() bool { return false }

// Get retrieves blobs by ID (stub implementation).
func (d *DummyDA) Get(_ context.Context, _ []datypes.ID, _ []byte) ([]datypes.Blob, error) {
	return nil, nil
}

// GetProofs returns inclusion proofs (stub implementation).
func (d *DummyDA) GetProofs(_ context.Context, _ []datypes.ID, _ []byte) ([]datypes.Proof, error) {
	return nil, nil
}

// Validate validates inclusion proofs.
func (d *DummyDA) Validate(_ context.Context, ids []datypes.ID, _ []datypes.Proof, _ []byte) ([]bool, error) {
	res := make([]bool, len(ids))
	for i := range res {
		res[i] = true
	}
	return res, nil
}

// SetSubmitFailure controls whether Submit should return errors.
func (d *DummyDA) SetSubmitFailure(shouldFail bool) {
	d.failSubmit.Store(shouldFail)
}

// Height returns the current DA height.
func (d *DummyDA) Height() uint64 {
	return d.height.Load()
}

// SetHeight sets the DA height directly.
func (d *DummyDA) SetHeight(h uint64) {
	d.height.Store(h)
}

// StartHeightTicker starts a goroutine that increments height at the given interval.
// Returns a stop function that should be called to stop the ticker.
func (d *DummyDA) StartHeightTicker(interval time.Duration) func() {
	if interval == 0 {
		return func() {}
	}

	d.tickerMu.Lock()
	if d.tickerStop != nil {
		d.tickerMu.Unlock()
		return func() {} // already running
	}
	d.tickerStop = make(chan struct{})
	stopCh := d.tickerStop
	d.tickerMu.Unlock()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				now := time.Now()
				height := d.height.Add(1)
				d.mu.Lock()
				if d.headers[height] == nil {
					d.headers[height] = &Header{
						Height:    height,
						Timestamp: now,
					}
				}
				d.mu.Unlock()
			case <-stopCh:
				return
			}
		}
	}()

	return func() {
		d.tickerMu.Lock()
		defer d.tickerMu.Unlock()
		if d.tickerStop != nil {
			close(d.tickerStop)
			d.tickerStop = nil
		}
	}
}

// Reset clears all stored blobs, headers, and resets the height.
func (d *DummyDA) Reset() {
	d.mu.Lock()
	d.height.Store(0)
	d.blobs = make(map[uint64]map[string][][]byte)
	d.headers = make(map[uint64]*Header)
	d.failSubmit.Store(false)
	d.mu.Unlock()

	d.tickerMu.Lock()
	if d.tickerStop != nil {
		close(d.tickerStop)
		d.tickerStop = nil
	}
	d.tickerMu.Unlock()
}

// GetHeaderByHeight retrieves the header for the given DA height.
// This mirrors the HeaderAPI.GetByHeight method from the real DA client.
// Returns nil if no header exists for the given height.
func (d *DummyDA) GetHeaderByHeight(_ context.Context, height uint64) (*Header, error) {
	d.mu.Lock()
	header := d.headers[height]
	d.mu.Unlock()

	if header == nil {
		currentHeight := d.height.Load()
		if height > currentHeight {
			return nil, datypes.ErrHeightFromFuture
		}
		return nil, datypes.ErrBlobNotFound
	}
	return header, nil
}
