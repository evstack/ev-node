package sync

import (
	"context"

	"github.com/celestiaorg/go-header"

	syncnotifier "github.com/evstack/ev-node/pkg/sync/notifier"
)

type publishFn[H header.Header[H]] func([]H)

type instrumentedStore[H header.Header[H]] struct {
	delegate header.Store[H]
	publish  publishFn[H]
}

func newInstrumentedStore[H header.Header[H]](
	delegate header.Store[H],
	publish publishFn[H],
) header.Store[H] {
	if publish == nil {
		return delegate
	}
	return &instrumentedStore[H]{
		delegate: delegate,
		publish:  publish,
	}
}

func (s *instrumentedStore[H]) Head(ctx context.Context, opts ...header.HeadOption[H]) (H, error) {
	return s.delegate.Head(ctx, opts...)
}

func (s *instrumentedStore[H]) Tail(ctx context.Context) (H, error) {
	return s.delegate.Tail(ctx)
}

func (s *instrumentedStore[H]) Height() uint64 {
	return s.delegate.Height()
}

func (s *instrumentedStore[H]) Has(ctx context.Context, hash header.Hash) (bool, error) {
	return s.delegate.Has(ctx, hash)
}

func (s *instrumentedStore[H]) HasAt(ctx context.Context, height uint64) bool {
	return s.delegate.HasAt(ctx, height)
}

func (s *instrumentedStore[H]) Append(ctx context.Context, headers ...H) error {
	if err := s.delegate.Append(ctx, headers...); err != nil {
		return err
	}

	if len(headers) > 0 {
		s.publish(headers)
	}

	return nil
}

func (s *instrumentedStore[H]) Get(ctx context.Context, hash header.Hash) (H, error) {
	return s.delegate.Get(ctx, hash)
}

func (s *instrumentedStore[H]) GetByHeight(ctx context.Context, height uint64) (H, error) {
	return s.delegate.GetByHeight(ctx, height)
}

func (s *instrumentedStore[H]) GetRangeByHeight(ctx context.Context, from H, to uint64) ([]H, error) {
	return s.delegate.GetRangeByHeight(ctx, from, to)
}

func (s *instrumentedStore[H]) GetRange(ctx context.Context, from, to uint64) ([]H, error) {
	return s.delegate.GetRange(ctx, from, to)
}

// func (s *instrumentedStore[H]) DeleteTo(ctx context.Context, to uint64) error {
// 	return s.delegate.DeleteTo(ctx, to)
// }

func (s *instrumentedStore[H]) DeleteRange(ctx context.Context, from, to uint64) error {
	return s.delegate.DeleteRange(ctx, from, to)
}

func (s *instrumentedStore[H]) OnDelete(handler func(ctx context.Context, height uint64) error) {
	s.delegate.OnDelete(handler)
}

func eventTypeForSync(syncType syncType) syncnotifier.EventType {
	switch syncType {
	case headerSync:
		return syncnotifier.EventTypeHeader
	case dataSync:
		return syncnotifier.EventTypeData
	default:
		return syncnotifier.EventTypeHeader
	}
}
