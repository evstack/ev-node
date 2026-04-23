package fibremock

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/evstack/ev-node/block/internal/da/fiber"
)

var (
	// ErrBlobNotFound is returned when a blob ID is not in the store.
	ErrBlobNotFound = errors.New("blob not found")
	// ErrDataEmpty is returned when Upload is called with empty data.
	ErrDataEmpty = errors.New("data cannot be empty")
)

// MockDAConfig configures the mock DA implementation.
type MockDAConfig struct {
	// MaxBlobs is the maximum number of blobs stored in memory.
	// When exceeded, the oldest blob is evicted regardless of retention.
	// 0 means no limit (use with caution — large blobs will OOM).
	MaxBlobs int
	// Retention is how long blobs are kept before automatic pruning.
	// 0 means blobs are kept until evicted by MaxBlobs.
	Retention time.Duration
}

// DefaultMockDAConfig returns a config suitable for testing:
// 100 blobs max, 10 minute retention.
func DefaultMockDAConfig() MockDAConfig {
	return MockDAConfig{
		MaxBlobs:  100,
		Retention: 10 * time.Minute,
	}
}

// storedBlob holds a blob and its metadata in the mock store.
type storedBlob struct {
	namespace []byte
	data      []byte
	height    uint64
	expiresAt time.Time
	createdAt time.Time
}

// subscriber tracks a Listen subscription.
type subscriber struct {
	namespace []byte
	ch        chan fiber.BlobEvent
}

// MockDA is an in-memory mock implementation of the DA interface.
// It stores blobs in memory with configurable retention and max blob count.
// Safe for concurrent use.
type MockDA struct {
	cfg MockDAConfig

	mu          sync.RWMutex
	blobs       map[string]*storedBlob // keyed by hex(blobID)
	order       []string               // insertion order for LRU eviction
	height      uint64
	subscribers []subscriber
}

// NewMockDA creates a new mock DA with the given config.
func NewMockDA(cfg MockDAConfig) *MockDA {
	return &MockDA{
		cfg:   cfg,
		blobs: make(map[string]*storedBlob),
	}
}

// Upload stores the blob in memory and notifies listeners.
func (m *MockDA) Upload(ctx context.Context, namespace []byte, data []byte) (fiber.UploadResult, error) {
	if len(data) == 0 {
		return fiber.UploadResult{}, ErrDataEmpty
	}

	blobID := mockBlobID(data)
	key := fmt.Sprintf("%x", blobID)
	now := time.Now()

	var expiresAt time.Time
	if m.cfg.Retention > 0 {
		expiresAt = now.Add(m.cfg.Retention)
	}

	m.mu.Lock()

	// Evict oldest if at capacity
	if m.cfg.MaxBlobs > 0 && len(m.blobs) >= m.cfg.MaxBlobs {
		m.evictOldestLocked()
	}

	// Prune expired blobs opportunistically
	if m.cfg.Retention > 0 {
		m.pruneExpiredLocked(now)
	}

	m.height++
	height := m.height

	m.blobs[key] = &storedBlob{
		namespace: namespace,
		data:      data,
		height:    height,
		expiresAt: expiresAt,
		createdAt: now,
	}
	m.order = append(m.order, key)

	// Notify subscribers (non-blocking)
	event := fiber.BlobEvent{
		BlobID:   blobID,
		Height:   height,
		DataSize: uint64(len(data)),
	}
	for i := range m.subscribers {
		if namespaceMatch(m.subscribers[i].namespace, namespace) {
			select {
			case m.subscribers[i].ch <- event:
			default:
				// Channel full, drop event. Subscriber is too slow.
			}
		}
	}

	m.mu.Unlock()

	return fiber.UploadResult{
		BlobID:    blobID,
		ExpiresAt: expiresAt,
	}, nil
}

// Download retrieves a blob by ID.
func (m *MockDA) Download(ctx context.Context, blobID fiber.BlobID) ([]byte, error) {
	key := fmt.Sprintf("%x", blobID)

	m.mu.RLock()
	blob, ok := m.blobs[key]
	m.mu.RUnlock()

	if !ok {
		return nil, ErrBlobNotFound
	}

	if !blob.expiresAt.IsZero() && time.Now().After(blob.expiresAt) {
		return nil, ErrBlobNotFound
	}

	return blob.data, nil
}

// Listen returns a channel that receives events when blobs matching the
// namespace are uploaded, starting at fromHeight.
//
// fromHeight == 0 subscribes to future uploads only. fromHeight > 0 first
// replays every matching blob still in the store with height >= fromHeight,
// then attaches a live subscriber for subsequent uploads. The replay may
// interleave with live events emitted between the Listen call and the
// replay goroutine's drain; consumers should dedupe by BlobID.
//
// The channel is closed when ctx is cancelled.
func (m *MockDA) Listen(ctx context.Context, namespace []byte, fromHeight uint64) (<-chan fiber.BlobEvent, error) {
	ch := make(chan fiber.BlobEvent, 64)

	m.mu.Lock()
	// Snapshot matching historicals under the lock to avoid racing with
	// concurrent Upload calls; the replay goroutine emits them after.
	var replay []fiber.BlobEvent
	if fromHeight > 0 {
		for _, key := range m.order {
			b, ok := m.blobs[key]
			if !ok {
				continue
			}
			if !namespaceMatch(namespace, b.namespace) {
				continue
			}
			if b.height < fromHeight {
				continue
			}
			replay = append(replay, fiber.BlobEvent{
				BlobID:   mockBlobID(b.data),
				Height:   b.height,
				DataSize: uint64(len(b.data)),
			})
		}
	}
	idx := len(m.subscribers)
	m.subscribers = append(m.subscribers, subscriber{
		namespace: namespace,
		ch:        ch,
	})
	m.mu.Unlock()

	// Replay historical events in a goroutine so the caller isn't
	// blocked if the buffer fills. Live events may interleave.
	if len(replay) > 0 {
		go func() {
			for _, ev := range replay {
				select {
				case ch <- ev:
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	// Clean up when context is done.
	go func() {
		<-ctx.Done()
		m.mu.Lock()
		// Remove subscriber by swapping with last
		last := len(m.subscribers) - 1
		if idx <= last {
			m.subscribers[idx] = m.subscribers[last]
		}
		m.subscribers = m.subscribers[:last]
		m.mu.Unlock()
		close(ch)
	}()

	return ch, nil
}

// BlobCount returns the number of blobs currently stored.
func (m *MockDA) BlobCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.blobs)
}

// evictOldestLocked removes the oldest blob. Caller must hold m.mu.
func (m *MockDA) evictOldestLocked() {
	if len(m.order) == 0 {
		return
	}
	key := m.order[0]
	m.order = m.order[1:]
	delete(m.blobs, key)
}

// pruneExpiredLocked removes blobs past their retention. Caller must hold m.mu.
func (m *MockDA) pruneExpiredLocked(now time.Time) {
	surviving := m.order[:0]
	for _, key := range m.order {
		blob, ok := m.blobs[key]
		if !ok {
			continue
		}
		if !blob.expiresAt.IsZero() && now.After(blob.expiresAt) {
			delete(m.blobs, key)
		} else {
			surviving = append(surviving, key)
		}
	}
	m.order = surviving
}

// namespaceMatch returns true if the subscription namespace matches the blob namespace.
// An empty subscription namespace matches all namespaces (wildcard).
func namespaceMatch(subNS, blobNS []byte) bool {
	if len(subNS) == 0 {
		return true
	}
	if len(subNS) != len(blobNS) {
		return false
	}
	for i := range subNS {
		if subNS[i] != blobNS[i] {
			return false
		}
	}
	return true
}

// mockBlobID produces a deterministic blob ID from the data.
// Format: 1 byte version (0) + 32 bytes SHA256 hash.
func mockBlobID(data []byte) fiber.BlobID {
	hash := sha256.Sum256(data)
	id := make([]byte, 33)
	id[0] = 0 // version byte
	copy(id[1:], hash[:])
	return id
}
