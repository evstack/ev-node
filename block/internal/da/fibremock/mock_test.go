package fibremock

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/evstack/ev-node/block/internal/da/fiber"
)

func TestMockDA_UploadDownload(t *testing.T) {
	m := NewMockDA(DefaultMockDAConfig())
	ctx := context.Background()

	ns := []byte("test-ns")
	data := []byte("hello fibre")

	result, err := m.Upload(ctx, ns, data)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.BlobID) != 33 {
		t.Fatalf("expected 33-byte blob ID, got %d", len(result.BlobID))
	}

	got, err := m.Download(ctx, result.BlobID)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("data mismatch: got %q, want %q", got, data)
	}
}

func TestMockDA_UploadEmpty(t *testing.T) {
	m := NewMockDA(DefaultMockDAConfig())
	_, err := m.Upload(context.Background(), []byte("ns"), nil)
	if err != ErrDataEmpty {
		t.Fatalf("expected ErrDataEmpty, got %v", err)
	}
}

func TestMockDA_DownloadNotFound(t *testing.T) {
	m := NewMockDA(DefaultMockDAConfig())
	_, err := m.Download(context.Background(), fiber.BlobID{0, 1, 2})
	if err != ErrBlobNotFound {
		t.Fatalf("expected ErrBlobNotFound, got %v", err)
	}
}

func TestMockDA_Listen(t *testing.T) {
	m := NewMockDA(DefaultMockDAConfig())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ns := []byte("test-ns")
	ch, err := m.Listen(ctx, ns, 0)
	if err != nil {
		t.Fatal(err)
	}

	// Upload a blob — should trigger the listener
	data := []byte("listened blob")
	result, err := m.Upload(ctx, ns, data)
	if err != nil {
		t.Fatal(err)
	}

	select {
	case event := <-ch:
		if !bytes.Equal(event.BlobID, result.BlobID) {
			t.Fatal("blob ID mismatch in event")
		}
		if event.Height == 0 {
			t.Fatal("expected non-zero height")
		}
		if event.DataSize != uint64(len(data)) {
			t.Fatalf("expected data size %d, got %d", len(data), event.DataSize)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestMockDA_ListenNamespaceFilter(t *testing.T) {
	m := NewMockDA(DefaultMockDAConfig())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := m.Listen(ctx, []byte("ns-A"), 0)
	if err != nil {
		t.Fatal(err)
	}

	// Upload to different namespace — should NOT trigger
	m.Upload(ctx, []byte("ns-B"), []byte("wrong namespace"))

	select {
	case <-ch:
		t.Fatal("should not receive event for different namespace")
	case <-time.After(50 * time.Millisecond):
		// good
	}
}

func TestMockDA_ListenWildcard(t *testing.T) {
	m := NewMockDA(DefaultMockDAConfig())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Empty namespace = wildcard
	ch, err := m.Listen(ctx, nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	m.Upload(ctx, []byte("any-ns"), []byte("wildcard test"))

	select {
	case event := <-ch:
		if event.Height == 0 {
			t.Fatal("expected event")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for wildcard event")
	}
}

func TestMockDA_MaxBlobsEviction(t *testing.T) {
	m := NewMockDA(MockDAConfig{MaxBlobs: 3})
	ctx := context.Background()

	var ids []fiber.BlobID
	for i := range 5 {
		r, err := m.Upload(ctx, nil, []byte{byte(i), 1, 2, 3})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, r.BlobID)
	}

	// First two should be evicted
	if _, err := m.Download(ctx, ids[0]); err != ErrBlobNotFound {
		t.Fatal("expected first blob to be evicted")
	}
	if _, err := m.Download(ctx, ids[1]); err != ErrBlobNotFound {
		t.Fatal("expected second blob to be evicted")
	}

	// Last three should still be there
	for i := 2; i < 5; i++ {
		if _, err := m.Download(ctx, ids[i]); err != nil {
			t.Fatalf("blob %d should exist: %v", i, err)
		}
	}

	if m.BlobCount() != 3 {
		t.Fatalf("expected 3 blobs, got %d", m.BlobCount())
	}
}

func TestMockDA_Retention(t *testing.T) {
	m := NewMockDA(MockDAConfig{Retention: 50 * time.Millisecond})
	ctx := context.Background()

	r, err := m.Upload(ctx, nil, []byte("ephemeral"))
	if err != nil {
		t.Fatal(err)
	}

	// Should exist immediately
	if _, err := m.Download(ctx, r.BlobID); err != nil {
		t.Fatal("blob should exist immediately")
	}

	// Wait for expiry
	time.Sleep(100 * time.Millisecond)

	if _, err := m.Download(ctx, r.BlobID); err != ErrBlobNotFound {
		t.Fatal("blob should have expired")
	}
}

func TestMockDA_DeterministicBlobID(t *testing.T) {
	m := NewMockDA(DefaultMockDAConfig())
	ctx := context.Background()

	data := []byte("deterministic")
	r1, _ := m.Upload(ctx, nil, data)
	r2, _ := m.Upload(ctx, nil, data)

	if !bytes.Equal(r1.BlobID, r2.BlobID) {
		t.Fatal("same data should produce same blob ID")
	}
}

// Verify MockDA satisfies the DA interface at compile time.
var _ fiber.DA = (*MockDA)(nil)
