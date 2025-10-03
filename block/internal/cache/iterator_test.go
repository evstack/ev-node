package cache

import (
	"errors"
	"io"
	"testing"
)

func TestIterator_Next(t *testing.T) {
	t.Run("sequence", func(t *testing.T) {
		start, total := 5, 3
		current := start
		produced := 0
		fetch := func() (int, error) {
			if produced >= total {
				return 0, io.EOF
			}
			v := current
			current++
			produced++
			return v, nil
		}
		it := newIterator[int](fetch)
		for i := 0; i < total; i++ {
			v, ok, err := it.Next()
			if err != nil {
				t.Fatalf("unexpected error at i=%d: %v", i, err)
			}
			if !ok {
				t.Fatalf("expected ok=true at i=%d", i)
			}
			expected := start + i
			if v != expected {
				t.Fatalf("expected value=%d, got %d", expected, v)
			}
		}
		v, ok, err := it.Next()
		if err == nil || err != io.EOF {
			t.Fatalf("expected io.EOF, got %v", err)
		}
		if ok {
			t.Fatalf("expected ok=false after exhaustion")
		}
		if v != 0 {
			t.Fatalf("expected zero value after exhaustion, got %v", v)
		}
	})

	t.Run("error propagation", func(t *testing.T) {
		calls := 0
		fetch := func() (int, error) {
			if calls == 0 {
				calls++
				return 42, nil
			}
			return 0, errors.New("boom")
		}
		it := newIterator[int](fetch)
		v, ok, err := it.Next()
		if err != nil || !ok || v != 42 {
			t.Fatalf("unexpected result on first call: v=%d ok=%v err=%v", v, ok, err)
		}
		v, ok, err = it.Next()
		if err == nil {
			t.Fatalf("expected error on second call")
		}
		if ok {
			t.Fatalf("expected ok=false on error")
		}
		if v != 0 {
			t.Fatalf("expected zero value on error, got %v", v)
		}
	})
}
