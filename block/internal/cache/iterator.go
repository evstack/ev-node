package cache

import (
	"context"
	"io"
)

type Iterator[T any] struct {
	base      *pendingBase[T]
	ctx       context.Context
	cursor    uint64
	limit     uint64
	exhausted bool
}

func (it *Iterator[T]) Next() (T, bool, error) {
	var zero T

	if it == nil || it.base == nil {
		return zero, false, io.EOF
	}
	if it.exhausted {
		return zero, false, io.EOF
	}
	if it.cursor > it.limit {
		it.exhausted = true
		return zero, false, io.EOF
	}

	val, err := it.base.fetch(it.ctx, it.base.store, it.cursor)
	if err != nil {
		it.exhausted = true
		return zero, false, err
	}

	it.cursor++
	if it.cursor > it.limit {
		it.exhausted = true
	}

	return val, true, nil
}
