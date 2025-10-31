package sync

import (
	"context"

	"github.com/celestiaorg/go-header"
)

type publishFn[H header.Header[H]] func([]H)

type instrumentedStore[H header.Header[H]] struct {
	header.Store[H]
	publish publishFn[H]
}

func newInstrumentedStore[H header.Header[H]](
	delegate header.Store[H],
	publish publishFn[H],
) header.Store[H] {
	if publish == nil {
		return delegate
	}
	return &instrumentedStore[H]{
		Store:   delegate,
		publish: publish,
	}
}

func (s *instrumentedStore[H]) Append(ctx context.Context, headers ...H) error {
	if err := s.Store.Append(ctx, headers...); err != nil {
		return err
	}

	if len(headers) > 0 {
		s.publish(headers)
	}

	return nil
}
