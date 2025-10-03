package cache

func newIterator[T any](fetch func() (T, error)) *Iterator[T] {
	return &Iterator[T]{
		fetch: fetch,
	}
}

type Iterator[T any] struct {
	fetch func() (T, error)
}

func (it *Iterator[T]) Next() (T, bool, error) {
	var zero T
	if it.fetch == nil {
		return zero, false, nil
	}

	val, err := it.fetch()
	if err != nil {
		return zero, false, err
	}

	return val, true, nil
}
